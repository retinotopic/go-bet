package lobby

import (
	"time"

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
	Place   int    `json:"Place"`
	CtrlBet int    `json:"CtrlBet,omitempty"`
	User_id string `json:"UserId,omitempty"`
	plr     *PlayUnit
}

type Lobby struct {
	PlayersRing
	m            *csmap.CsMap[string, *PlayUnit] // id to place mapping
	HasBegun     bool
	PlayerCh     chan *PlayUnit
	checkTimeout *time.Ticker
	lastResponse time.Time
	ValidateCh   chan Ctrl
	BroadcastCh  chan *PlayUnit
	Shutdown     chan bool
	Url          uint64
	Validate     func(Ctrl)
}

func (l *Lobby) LobbyStart(yield func(PlayUnit) bool) {
	if l.Validate == nil {
		return
	}
	l.checkTimeout = time.NewTicker(time.Minute * 5)
	l.Shutdown = make(chan bool, 3)
	l.BroadcastCh = make(chan *PlayUnit, 10)
	l.PlayerCh = make(chan *PlayUnit)
	timeout := time.Minute * 4
	for {
		select {
		case <-l.checkTimeout.C:
			if time.Since(l.lastResponse) > timeout {
				for range 2 { //shutdowns,one for game and one for lobby
					l.Shutdown <- true
				}
			}
		case pb, ok := <-l.BroadcastCh:
			if ok {
				l.lastResponse = time.Now()
				go func() {
					for v := range l.PlayerIter {
						protect := false
						if pb.Place != v.Place || !pb.Exposed {
							protect = true
						}
						v.Send(pb, protect)
						pb.Exposed = false
					}
				}()
			}
		case ctrl := <-l.ValidateCh:
			l.Validate(ctrl)
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
