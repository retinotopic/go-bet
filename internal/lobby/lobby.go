package lobby

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/retinotopic/go-bet/internal/queue"
	"github.com/retinotopic/go-bet/pkg/wsutils"
)

func NewLobby(queue *queue.TaskQueue) *Lobby {
	l := &Lobby{PlayerCh: make(chan PlayUnit), StartGame: make(chan bool, 1), Queue: queue}
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
type Lobby struct { //
	PlayersRing //
	Admin       PlayUnit
	sync.Mutex
	AdminOnce       sync.Once
	PlayerCh        chan PlayUnit
	shutdown        chan bool
	timeout         time.Duration
	StartGame       chan bool
	GameOver        atomic.Bool
	PlayerBroadcast chan PlayUnit
	IsRating        bool
	LenPlayers      int //
	HasBegun        atomic.Bool
	Url             string
	Queue           *queue.TaskQueue
}

func (l *Lobby) LobbyWork() {
	l.LenPlayers = len(l.Players)
	for {
		select {
		case <-l.shutdown:
			return
		case <-l.StartGame:
			game := Game{Lobby: l}
			l.HasBegun.Store(true)
			game.Game()
			l.GameOver.Store(true)
			return
		}

	}
}

func (l *Lobby) ConnHandle(plr *PlayUnit) {
	go wsutils.KeepAlive(plr.Conn, time.Second*15)
	l.AdminOnce.Do(func() {
		if l.IsRating {
			go l.tickerTillGame()
		} else {
			l.Admin = *plr
			plr.Admin = true
		}
	})

	defer func() {
		if plr.Conn != nil {
			err := plr.Conn.Close()
			if err != nil {
				log.Println(err, "error closing connection")
			}
		}
	}()
	for _, v := range l.Players { // load current state of the game
		if v.Place != plr.Place {
			*v = v.PrivateSend()
		}
		err := plr.Conn.WriteJSON(v)
		if err != nil {
			fmt.Println(err, "WriteJSON start")
		}
		fmt.Println("start")
	}
	for {
		ctrl := PlayUnit{}
		err := plr.Conn.ReadJSON(&ctrl)
		if err != nil {
			fmt.Println(err, "conn read error")
			break
		}
		if ctrl.CtrlBet <= plr.Bankroll && ctrl.CtrlBet >= 0 {
			plr.CtrlBet = ctrl.CtrlBet
			plr.ExpirySec = 30
			l.PlayerCh <- *plr
		}
	}
}
func (l *Lobby) tickerTillGame() {
	timer := time.NewTimer(time.Second * 45)
	for range timer.C {
		l.StartGame <- true
		close(l.StartGame)
		return
	}
}

// player broadcast method
func (l *Lobby) Broadcast() {
	lastResponse := time.Now()
	for pb := range l.PlayerBroadcast {
		if time.Since(lastResponse) > l.timeout {
			l.shutdown <- true
		} else {
			lastResponse = time.Now()
		}
		for _, v := range l.Players {
			if pb.Place != v.Place {
				pb = v.PrivateSend()
			}
			v.Conn.WriteJSON(pb)
		}
	}
}
