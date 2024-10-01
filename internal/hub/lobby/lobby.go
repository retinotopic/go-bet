package lobby

import (
	"time"

	"github.com/chehsunliu/poker"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

func NewLobby(queue Queue) *Lobby {
	l := &Lobby{PlayerCh: make(chan *PlayUnit)}
	return l
}

const (
	flop = iota
	turn
	river
	postriver
)

type top struct {
	place  int
	rating int32
}
type Ctrl struct {
	IsExposed bool      `json:"-"` // means whether the cards should be shown to everyone
	Place     int       `json:"Place"`
	CtrlBet   int       `json:"CtrlBet"`
	Message   string    `json:"Message"`
	Plr       *PlayUnit `json:"-"`
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
	Validate     func(Ctrl)
}

func (l *Lobby) LobbyStart(yield func(PlayUnit) bool) {
	if l.Validate == nil {
		return
	}
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
