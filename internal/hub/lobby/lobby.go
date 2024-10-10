package lobby

import (
	"bytes"
	"context"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/coder/websocket"
	"golang.org/x/sync/errgroup"
)

const (
	flop = iota
	turn
	river
	postriver
)

type Ctrl struct { //control union
	IsExposed  bool      `json:"-"` // means whether the cards should be shown to everyone
	Place      int       `json:"Place"`
	CtrlInt    int       `json:"CtrlInt"`
	CtrlString string    `json:"CtrlString"`
	Plr        *PlayUnit `json:"-"`
}

type Lobby struct {
	PlayersRing
	HasBegun     bool
	PlayerCh     chan Ctrl
	checkTimeout *time.Ticker
	lastResponse time.Time
	ValidateCh   chan Ctrl
	Shutdown     chan bool
	Url          uint64
}

func (l *Lobby) LobbyStart(gm *Game) {
	go gm.Game()
	l.checkTimeout = time.NewTicker(time.Minute * 3)
	l.Shutdown = make(chan bool, 10)
	l.PlayerCh = make(chan Ctrl)
	timeout := time.Minute * 4
	for {
		select {
		case <-l.checkTimeout.C:
			if time.Since(l.lastResponse) > timeout {
				for range 3 { //shutdowns,two for game and one for lobby
					l.Shutdown <- true
				}
			}
		case <-l.Shutdown:
			return
		}

	}
}

func (c *Lobby) HandleConn(plr *PlayUnit) {
	defer func() {
		defer c.AllUsers.Mtx.Unlock()
		c.AllUsers.Mtx.Lock()
		if plr.Place == -2 {
			delete(c.AllUsers.M, plr.User_id)
		}
		plr.Conn.CloseNow()
	}()
	g := errgroup.Group{}
	g.Go(func() error {
		return WriteTimeout(time.Second*5, plr.Conn, c.Board.GetCache())
	})
	isEmpty := true
	c.AllUsers.Mtx.RLock()
	for _, v := range c.AllUsers.M { // load current state of the game
		if v.Place >= 0 {
			if v == plr {
				isEmpty = false
			}
			g.Go(func() error {
				return WriteTimeout(time.Second*5, plr.Conn, EmptyCards(v.GetCache(), isEmpty))
			})
			isEmpty = true
		}
	}
	c.AllUsers.Mtx.RUnlock()
	if err := g.Wait(); err != nil {
		return
	}
	ctrl := Ctrl{}
	ctrl.Plr = plr
	for { // listening for actions
		_, data, err := plr.Conn.Read(context.TODO())
		if err != nil {
			break
		}
		err = json.Unmarshal(data, &ctrl)
		if err != nil {
			break
		}
		plr.IsAway = false
		c.PlayerCh <- ctrl
		time.Sleep(time.Millisecond * 500)
	}
}
func (l *Lobby) BroadcastPlayer(plr *PlayUnit, IsExposed bool) {
	defer l.AllUsers.Mtx.RUnlock()
	l.AllUsers.Mtx.RLock()
	var isEmpty bool
	for _, val := range l.AllUsers.M {

		if val == plr || IsExposed {
			isEmpty = false
		} else {
			isEmpty = true
		}
		go WriteTimeout(time.Second*5, val.Conn, EmptyCards(plr.GetCache(), isEmpty))

	}

}

func (l *Lobby) BroadcastBoard(board *GameBoard) {
	defer l.AllUsers.Mtx.RUnlock()
	l.AllUsers.Mtx.RLock()
	for _, val := range l.AllUsers.M {
		go WriteTimeout(time.Second*5, val.Conn, board.GetCache())
	}
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
