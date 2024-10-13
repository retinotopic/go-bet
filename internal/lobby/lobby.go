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
		if plr.Place == -2 {
			c.AllUsers.Mtx.Lock()
			delete(c.AllUsers.M, plr.User_id)
			c.AllUsers.Mtx.Unlock()
		}
		plr.Conn.CloseNow()
	}()

	isEmpty := true
	c.AllUsers.Mtx.RLock()

	g := errgroup.Group{}
	g.Go(func() error {
		return WriteTimeout(time.Second*5, plr.Conn, c.Board.cache)
	})
	for _, v := range c.AllUsers.M { // load current state of the game
		if v.Place >= 0 {
			if v == plr {
				isEmpty = false
			}
			g.Go(func() error {
				return WriteTimeout(time.Second*5, plr.Conn, EmptyCards(v.cache, isEmpty))
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
		c.lastResponse = time.Now()
		plr.IsAway = false
		if ctrl.CtrlInt == 0 && len(ctrl.CtrlString) != 0 {
			c.BroadcastBytes([]byte(`{"Message":"` + ctrl.CtrlString + `","Name":"` + plr.Name + `"}`))
			continue
		}
		c.PlayerCh <- ctrl
		time.Sleep(time.Millisecond * 500)
	}
}
func (l *Lobby) BroadcastPlayer(plr *PlayUnit, IsExposed bool) {

	l.AllUsers.Mtx.Lock()
	plr.StoreCache()
	l.AllUsers.Mtx.Unlock()

	l.AllUsers.Mtx.RLock()
	var isEmpty bool
	for _, val := range l.AllUsers.M {

		if val == plr || IsExposed {
			isEmpty = false
		} else {
			isEmpty = true
		}
		go WriteTimeout(time.Second*5, val.Conn, EmptyCards(plr.cache, isEmpty))

	}
	l.AllUsers.Mtx.RUnlock()

}
func (l *Lobby) BroadcastBoard(brd *GameBoard) {

	l.AllUsers.Mtx.Lock()
	brd.StoreCache()
	l.AllUsers.Mtx.Unlock()

	l.AllUsers.Mtx.RLock()
	for _, val := range l.AllUsers.M {
		go WriteTimeout(time.Second*5, val.Conn, brd.cache)
	}
	l.AllUsers.Mtx.RUnlock()

}
func (l *Lobby) BroadcastBytes(b []byte) {
	l.AllUsers.Mtx.RLock()
	for _, val := range l.AllUsers.M {
		go WriteTimeout(time.Second*5, val.Conn, b)
	}
	l.AllUsers.Mtx.RUnlock()
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
