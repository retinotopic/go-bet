package lobby

import (
	"bytes"
	"context"
	json "github.com/bytedance/sonic"
	"github.com/coder/websocket"
	"sync"
	"time"
	// "github.com/panjf2000/ants/v2"
)

const (
	flop = iota
	turn
	river
	postriver
)

type Lobby struct {
	// Lobby info
	HasBegun     bool
	PlayerCh     chan Ctrl
	CheckTimeout *time.Ticker
	lastResponse time.Time
	ValidateCh   chan Ctrl
	Shutdown     chan bool
	Url          uint64
	MapUsers     mapUsers
	AllUsers     []PlayUnit

	// In process game info
	InitialPlayerBank int
	SidePots          [][]int
	Seats             [8]Seats
	TopPlaces         []Top

	// indices of AllUsers slice elements
	Winners []int
	Losers  []int
	Deck    Deck

	// current player
	Pl          int
	TurnTimer   *time.Timer
	BlindTimer  *time.Timer
	StartGameCh chan bool
	Stop        chan bool
	BlindRaise  bool

	Players  []int
	Idx      int
	Board    GameBoard
	Blindlvl int

	// method for handling board in no-ingame time
	Validate func(Ctrl)

	// method for handling kicked players
	PlayerOut func([]int, int)

	//for custom lobbies
	AdminOnce sync.Once
	Admin     int

	/*For rating lobbies, basically a queue,
	for some minimal persistence writes on disk if
	update not succeded */
	UpdateRating func(int, int) // user id, rating
}

type mapUsers struct {
	M   map[string]int // userId to index in AllUsers slice
	Mtx sync.RWMutex
}

func (l *Lobby) LobbyStart() {
	go l.Game()
	l.CheckTimeout.Reset(time.Minute * 3)
	timeout := time.Minute * 4
	for {
		select {
		case <-l.CheckTimeout.C:
			if time.Since(l.lastResponse) > timeout {
				for range 3 {
					l.Shutdown <- true
				}
			}
		case <-l.Shutdown:
			return
		}

	}
}

func (c *Lobby) HandleConn(idx int) {
	plr := &c.AllUsers[idx]
	defer c.Exit(idx)

	err := c.LoadCurrentState(idx)
	if err != nil {
		return
	}
	for { // listening for actions
		ctrl := Ctrl{}
		ctrl.Plr = idx
		_, data, err := plr.Conn.Read(context.TODO())
		if err != nil {
			break
		}
		err = json.Unmarshal(data, &ctrl)
		if err != nil {
			break
		}
		c.lastResponse = time.Now()
		plr.IsAway = false
		if ctrl.Ctrl == 0 && len(ctrl.Text) != 0 {
			go c.BroadcastBytes([]byte(`{"Message":"` + ctrl.Text + `","Name":"` + plr.Name + `"}`))
			continue
		}
		c.PlayerCh <- ctrl
		time.Sleep(time.Millisecond * 500)
	}
}
func (l *Lobby) BroadcastPlayer(idx int, IsExposed bool) {
	plr := &l.AllUsers[idx]
	l.MapUsers.Mtx.Lock()
	plr.StoreCache()
	l.MapUsers.Mtx.Unlock()

	l.MapUsers.Mtx.RLock()
	var isEmpty bool
	for _, v := range l.MapUsers.M {
		pl := &l.AllUsers[v]
		if pl == plr || IsExposed {
			isEmpty = false
		} else {
			isEmpty = true
		}
		go WriteTimeout(time.Second*5, pl.Conn, EmptyCards(plr.cache, isEmpty))

	}
	l.MapUsers.Mtx.RUnlock()

}
func (l *Lobby) BroadcastBoard() {

	l.MapUsers.Mtx.Lock()
	l.Board.StoreCache()
	l.MapUsers.Mtx.Unlock()

	l.MapUsers.Mtx.RLock()
	for _, v := range l.MapUsers.M {
		pl := l.AllUsers[v]
		go WriteTimeout(time.Second*5, pl.Conn, l.Board.cache)
	}
	l.MapUsers.Mtx.RUnlock()

}
func (l *Lobby) BroadcastBytes(b []byte) {
	l.MapUsers.Mtx.RLock()
	for _, v := range l.MapUsers.M {
		pl := l.AllUsers[v]
		go WriteTimeout(time.Second*5, pl.Conn, b)
	}
	l.MapUsers.Mtx.RUnlock()
}

func WriteTimeout(timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}
func EmptyCards(data []byte, isEmpty bool) (data2 []byte) { // in order to not marshaling twice, but for the cards to be empty
	if !isEmpty {
		return data
	}
	start := bytes.IndexByte(data, '[')
	if start == -1 {
		return data
	}
	start++
	end := bytes.IndexByte(data, ']')
	if end == -1 {
		return data
	}
	for i := start; i < end; i++ {
		data[i] = ' '
	}
	copy(data, data2)
	return data2
}
func (l *Lobby) Exit(idx int) {
	p := &l.AllUsers[idx]
	if p.Place == -2 {
		l.MapUsers.Mtx.Lock()
		delete(l.MapUsers.M, p.User_id)
		p.Place = -3
		l.MapUsers.Mtx.Unlock()

	}
	p.Conn.CloseNow()
}

func (l *Lobby) LoadPlayer(userid, name string) (int, bool) {
	l.MapUsers.Mtx.RLock()
	pl, ok := l.MapUsers.M[userid]
	l.MapUsers.Mtx.RUnlock()
	if ok {
		return pl, true
	} else {
		l.MapUsers.Mtx.Lock()
		defer l.MapUsers.Mtx.Unlock()

		if len(l.MapUsers.M) < 50 {
			for i := range l.AllUsers {
				if l.AllUsers[i].Place == -3 {
					l.MapUsers.M[userid] = i
					l.AllUsers[i].Place = -2
					return i, true
				}
			}
		}
	}
	return -1, false
}
func (l *Lobby) LoadCurrentState(idx int) (err error) {
	plr := &l.AllUsers[idx]
	isEmpty := true
	l.MapUsers.Mtx.RLock()

	for i := range l.Players { // load current state of the game
		plrs := &l.AllUsers[l.Players[i]]
		if plrs.Place >= 0 {
			if plrs == plr {
				isEmpty = false
			}

			go WriteTimeout(time.Second*5, plrs.Conn, EmptyCards(plrs.cache, isEmpty))

			isEmpty = true
		}
	}
	go WriteTimeout(time.Second*5, plr.Conn, l.Board.cache)
	l.MapUsers.Mtx.RUnlock()

	return
}
