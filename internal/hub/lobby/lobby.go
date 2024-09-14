package lobby

import (
	"sync"
	"time"

	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

func NewLobby(queue Queue) *Lobby {
	l := &Lobby{PlayerCh: make(chan PlayUnit), StartGame: make(chan bool, 1)}
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

type Lobby struct {
	PlayersRing                                 //
	m            *csmap.CsMap[string, PlayUnit] // id to place mapping
	GameMtx      sync.RWMutex
	PlayerCh     chan PlayUnit
	checkTimeout *time.Ticker
	lastResponse time.Time
	StartGame    chan bool
	BroadcastCh  chan PlayUnit
	Shutdown     chan bool
	Url          uint64
	PlayerOut    func([]PlayUnit)
}

func (l *Lobby) LobbyStart(yield func(PlayUnit) bool) {
	l.checkTimeout = time.NewTicker(time.Minute * 5)
	l.Shutdown = make(chan bool, 2)
	l.BroadcastCh = make(chan PlayUnit)
	l.PlayerCh = make(chan PlayUnit)
	timeout := time.Minute * 4
	for {
		select {
		case <-l.checkTimeout.C:
			if time.Since(l.lastResponse) > timeout {
				for range 2 { //shutdowns,one for game and one for lobby
					l.Shutdown <- true
				}
			}
		case <-l.StartGame:
			ok := l.GameMtx.TryLock() // trying to check if there already a game in progress
			if ok {
				go func() {
					g := Game{Lobby: l}
					g.Game()
					defer l.GameMtx.Unlock()
				}()
			}
		case <-l.Shutdown:
			return
		}

	}
}

// player broadcast method
func (l *Lobby) Broadcast() {
	for pb := range l.BroadcastCh {
		l.lastResponse = time.Now()
		for v := range l.PlayerIter {
			if pb.Place != v.Place || !pb.Exposed {
				pb = v.PrivateSend()
			}
			v.Conn.WriteJSON(pb)
			pb.Exposed = false
		}
	}
}
func (l *Lobby) PlayerIter(yield func(PlayUnit) bool) {
	for _, pl := range l.Players { /// players
		yield(pl)
	}
	l.m.Range(func(key string, val PlayUnit) (stop bool) { // spectators
		yield(val)
		return
	})
}
