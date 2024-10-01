package lobby

import (
	"time"

	"github.com/chehsunliu/poker"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

const (
	flop = iota
	turn
	river
	postriver
)

type Ctrl struct { // broadcast control union
	IsExposed bool      `json:"-"` // means whether the cards should be shown to everyone
	Place     int       `json:"Place"`
	CtrlBet   int       `json:"CtrlBet"`
	Message   string    `json:"-"`
	Plr       *PlayUnit `json:"-"`
	Brd       GameBoard `json:"-"`
}

type Lobby struct {
	PlayersRing
	m            *csmap.CsMap[string, *PlayUnit] // id to place mapping
	HasBegun     bool
	PlayerCh     chan Ctrl
	checkTimeout *time.Ticker
	lastResponse time.Time
	ValidateCh   chan Ctrl
	BroadcastCh  chan Ctrl
	Shutdown     chan bool
	Url          uint64
}

func (l *Lobby) LobbyStart() {
	gm := &Game{Lobby: l}
	go gm.Game()
	l.checkTimeout = time.NewTicker(time.Minute * 3)
	l.Shutdown = make(chan bool, 3)
	l.BroadcastCh = make(chan Ctrl, 10)
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
		case ctrl, ok := <-l.BroadcastCh:
			if ok {
				l.lastResponse = time.Now()
				if len(ctrl.Message) != 0 && ctrl.Place > 0 { //
					for v := range l.PlayerIter {
						go v.Conn.WriteJSON(ctrl.Message)
					}
				}
				go func(pb PlayUnit, exposed bool) {
					if !exposed {
						pb.Cards = []poker.Card{}
					}
					for v := range l.PlayerIter {
						if v.Place != pb.Place {
							go v.Conn.WriteJSON(&pb)
						} else {

						}
					}
				}(*ctrl.Plr, ctrl.IsExposed) //copy to prevent modification from the rest of the goroutines
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
		if c.m.DeleteIf(plr.User_id, func(value *PlayUnit) bool { return value.Place == -2 }) { //if the seat is unoccupied, delete from map
			c.BroadcastCh <- Ctrl{Plr: plr}
		}
		plr.Conn.Close()
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
		err := plr.Conn.ReadJSON(&ctrl)
		if err != nil {
			break
		}
		ctrl.Plr = plr
		plr.IsAway = false
		c.PlayerCh <- ctrl
	}
}
