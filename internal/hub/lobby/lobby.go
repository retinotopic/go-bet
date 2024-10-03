package lobby

import (
	"context"
	"time"

	"github.com/chehsunliu/poker"
	"github.com/coder/websocket"
	"github.com/goccy/go-json"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

const (
	flop = iota
	turn
	river
	postriver
)

type Ctrl struct { //control union
	IsExposed bool       `json:"-"` // means whether the cards should be shown to everyone
	Place     int        `json:"Place"`
	CtrlBet   int        `json:"CtrlBet"`
	Message   string     `json:"Message"`
	Plr       *PlayUnit  `json:"-"`
	Brd       *GameBoard `json:"GameBoard"`
}

type Lobby struct {
	PlayersRing
	m            *csmap.CsMap[string, *PlayUnit] // id to place mapping
	HasBegun     bool
	PlayerCh     chan Ctrl
	checkTimeout *time.Ticker
	lastResponse time.Time
	ValidateCh   chan Ctrl
	Shutdown     chan bool
	Url          uint64
}

func (l *Lobby) LobbyStart() {
	gm := &Game{Lobby: l}
	go gm.Game()
	l.checkTimeout = time.NewTicker(time.Minute * 3)
	l.Shutdown = make(chan bool, 3)
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

func (l *Lobby) PlayerIter(yield func(*PlayUnit) bool) {
	for _, pl := range l.Players { /// players
		yield(pl)
	}
	l.m.Range(func(key string, val *PlayUnit) (stop bool) { // spectators
		yield(val)
		return
	})
}

func (c *Lobby) HandleConn(plr *PlayUnit) {
	c.m.SetIfAbsent(plr.User_id, plr)

	defer func() {
		c.m.DeleteIf(plr.User_id, func(value *PlayUnit) bool { return value.Place == -2 }) //if the seat is unoccupied, delete from map
		plr.Conn.CloseNow()
	}()
	plr.Conn.WriteJSON(plr)
	plr.Conn.WriteJSON(c.Board)
	for _, v := range c.Players { // load current state of the game
		pb := *v
		if v != plr {
			pb.Cards = []poker.Card{}
			plr.Conn.WriteJSON(pb)
		}

	}
	for { // listening for actions
		ctrl := Ctrl{}
		_, data, err := plr.Conn.Read(context.TODO())
		if err != nil {
			break
		}
		err = json.Unmarshal(data, &ctrl)
		if err != nil {
			break
		}
		ctrl.Plr = plr
		plr.IsAway = false
		c.PlayerCh <- ctrl
	}
}
func (c *Lobby) BroadcastPlayer(plr PlayUnit, isExposed bool) (err error) {
	b, err := json.Marshal(plr)
	if err != nil {
		return err
	}
	go writeTimeout(time.Second*5, plr.Conn, b)
	if !isExposed {
		plr.Cards = []poker.Card{}
	}
	for pl := range c.PlayerIter {
		go writeTimeout(time.Second*5, pl.Conn, b)
	}
	return
}

func (c *Lobby) BroadcastBoard(board *GameBoard) (err error) {
	b, err := json.Marshal(board)
	if err != nil {
		return err
	}
	for pl := range c.PlayerIter {
		go writeTimeout(time.Second*5, pl.Conn, b)
	}
	return
}
func writeTimeout(timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}
