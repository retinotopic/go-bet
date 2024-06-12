package lobby

import (
	"fmt"
	"sync"
	"time"

	"github.com/chehsunliu/poker"
)

func NewLobby() *Lobby {
	l := &Lobby{PlayerCh: make(chan PlayUnit), StartGame: make(chan struct{}, 1)}
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
	Deck       *poker.Deck
	Players    []*PlayUnit
	Admin      PlayUnit
	Board      PlayUnit
	SmallBlind int
	MaxBet     int
	TurnTicker *time.Ticker
	sync.Mutex
	Order           int
	AdminOnce       sync.Once
	PlayerCh        chan PlayUnit
	StartGame       chan struct{}
	PlayerBroadcast chan PlayUnit
	isRating        bool
}

func (l *Lobby) LobbyWork() {
	for {
		select {
		case x := <-l.PlayerCh: // broadcoasting one seat to everyone
			for _, v := range l.Players {
				v.Conn.WriteJSON(x)
			}
		case <-l.StartGame:
			l.Game()
		}
	}
}

func (l *Lobby) ConnHandle(plr *PlayUnit) {
	fmt.Println("im in2")
	l.AdminOnce.Do(func() {
		if l.isRating {
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
			l.PlayerCh <- *plr
		}
	}
}
func (l *Lobby) tickerTillGame() {
	ticker := time.NewTicker(time.Second * 1)
	i := 0
	for range ticker.C {
		if i >= 15 {
			l.StartGame <- struct{}{}
			return
		}
		i++
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
