package lobby

import (
	"bytes"
	"io"
	"sync"
	"time"

	json "github.com/bytedance/sonic"
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
	Shutdown     bool
	Url          uint64
	MapUsers     mapUsers
	AllUsers     []PlayUnit

	// In process game info
	InitialPlayerBank int
	SidePots          [][]int
	Seats             [8]Seats
	TopPlaces         []Top
	msgBuf            *bytes.Buffer
	MsgCh             chan Ctrl

	// indices of AllUsers slice elements
	Winners []int
	Losers  []int
	Deck    *Deck

	// current player
	Pl          int
	TurnTimer   *time.Timer
	BlindTimer  *time.Timer
	StartGameCh chan bool
	Stop        bool
	BlindRaise  bool

	wg       sync.WaitGroup
	Players  []int
	Idx      int
	Board    GameBoard
	Blindlvl int

	// method for handling board in no-ingame time
	Validate func(Ctrl)

	//for custom lobbies
	AdminOnce sync.Once
	Admin     int
}

type mapUsers struct {
	M   map[string]int // userId to index in AllUsers slice
	Mtx sync.RWMutex
}

func (l *Lobby) LobbyStart() {
	go l.Game()
	l.Shutdown = false
	l.CheckTimeout.Reset(time.Minute * 3)
	timeout := time.Minute * 4
	for {
		select {
		case <-l.CheckTimeout.C:
			if time.Since(l.lastResponse) > timeout {
				l.Shutdown = true
				return
			}
		case ctrl, ok := <-l.MsgCh:
			if ok {
				l.BroadcastMsg(ctrl)
			}
		}
	}
}

func (c *Lobby) HandlePlayer(idx int, wrc io.ReadWriteCloser) {
	plr := &c.AllUsers[idx]
	plr.Conn = wrc
	defer c.Exit(idx)

	err := c.LoadCurrentState(idx)
	if err != nil {
		return
	}
	ctrl := Ctrl{Plr: idx, Place: plr.Place}
	readb := make([]byte, 0, 40)
	for { // listening for actions
		ctrl.Plr = idx
		n, err := plr.Conn.Read(readb)
		if err != nil {
			break
		}
		err = json.Unmarshal(readb[:n], &ctrl)
		if err != nil {
			break
		}
		c.lastResponse = time.Now()
		plr.IsAway = false
		if ctrl.Ctrl == 0 && len(ctrl.Text) != 0 {
			c.MsgCh <- ctrl
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
	for _, v := range l.MapUsers.M {
		pl := &l.AllUsers[v]
		if pl == plr || IsExposed {
			go pl.Conn.Write(plr.cache.Bytes())
			continue
		} else {
			go pl.Conn.Write(plr.HiddenCardsCache)
		}

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
		go pl.Conn.Write(l.Board.cache.Bytes())

	}
	l.MapUsers.Mtx.RUnlock()

}

func (l *Lobby) BroadcastMsg(ctrl Ctrl) {
	if len(ctrl.Text) > 100 {
		return
	}

	l.msgBuf.Reset()
	l.msgBuf.WriteString(`{"Message":"`)
	l.msgBuf.WriteString(ctrl.Text)
	l.msgBuf.WriteString(`"Name":"`)
	l.msgBuf.WriteString(l.AllUsers[ctrl.Plr].Name)
	l.msgBuf.WriteString(`"}`)

	l.MapUsers.Mtx.RLock()
	for _, v := range l.MapUsers.M {
		pl := l.AllUsers[v]

		l.wg.Add(1)
		go func() {
			pl.Conn.Write(l.msgBuf.Bytes())
			l.wg.Done()
		}()

	}
	l.MapUsers.Mtx.RUnlock()
	l.wg.Wait()
}

func HideCards(dst, src []byte) { // in order to not marshaling twice, but for the cards to be empty
	copy(dst, src)
	start := bytes.IndexByte(dst, '[')
	if start == -1 {
		return
	}
	start++
	end := bytes.IndexByte(dst, ']')
	if end == -1 {
		return
	}
	for i := start; i < end; i++ {
		dst[i] = ' '
	}
}

func (l *Lobby) Exit(idx int) {
	p := &l.AllUsers[idx]
	if p.Place == -2 {
		l.MapUsers.Mtx.Lock()
		delete(l.MapUsers.M, p.User_id)
		p.Place = -3
		l.MapUsers.Mtx.Unlock()

	}
	p.Conn.Close()
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
	l.MapUsers.Mtx.RLock()

	for i := range l.Players { // load current state of the game
		plrs := &l.AllUsers[l.Players[i]]
		if plrs.Place >= 0 {
			if plrs == plr {
				go plr.Conn.Write(plrs.cache.Bytes())
				continue
			}
			go plr.Conn.Write(plrs.HiddenCardsCache)
		}
	}
	go plr.Conn.Write(l.Board.cache.Bytes())
	l.MapUsers.Mtx.RUnlock()

	return
}
