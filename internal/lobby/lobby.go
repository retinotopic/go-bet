package lobby

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/retinotopic/go-bet/internal/queue"
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
		case x := <-l.PlayerCh: // broadcoasting one seat to everyone
			for _, v := range l.Players {
				v.Conn.WriteJSON(x)
			}
		case <-l.StartGame:
			close(l.StartGame)
			game := Game{Lobby: l}
			l.HasBegun.Store(true)
			game.Game()
			l.GameOver.Store(true)
			return
		}
	}
}

func (l *Lobby) ConnHandle(plr *PlayUnit) {
	fmt.Println("im in2")
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
				fmt.Println(err, "error closing connection")
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
		if ctrl.Bet <= plr.Bankroll && ctrl.Bet >= 0 {
			plr.Bet = plr.Bet + ctrl.Bet
			plr.Bankroll -= ctrl.Bet
			plr.ExpirySec = 30
			l.PlayerCh <- *plr
		}
	}
}
func (l *Lobby) tickerTillGame() {
	timer := time.NewTimer(time.Second * 45)
	for range timer.C {
		l.StartGame <- true
		return
	}
}

// player broadcast method
func (l *Lobby) Broadcast() {
	for pb := range l.PlayerBroadcast {
		for _, v := range l.Players {
			if pb.Place != v.Place {
				pb = v.PrivateSend()
			}
			v.Conn.WriteJSON(pb)
		}
	}
}
