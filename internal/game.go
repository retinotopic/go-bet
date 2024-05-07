package hub

import (
	"fmt"
	"sync"
	"time"

	"github.com/chehsunliu/poker"
	"github.com/retinotopic/go-bet/internal/player"
	"github.com/retinotopic/go-bet/pkg/randfuncs"
)

func NewLobby() *Lobby {
	l := &Lobby{Players: make([]player.PlayUnit, 0), Occupied: make(map[int]bool), PlayerCh: make(chan player.PlayUnit), StartGame: make(chan struct{}, 1)}
	return l
}

type Lobby struct {
	Players    []player.PlayUnit
	Admin      player.PlayUnit
	Board      player.PlayUnit
	MaxBet     int
	Occupied   map[int]bool
	TurnTicker *time.Ticker
	sync.Mutex
	Order           int
	AdminOnce       sync.Once
	PlayerCh        chan player.PlayUnit
	StartGame       chan struct{}
	PlayerBroadcast chan player.PlayUnit
	isRating        bool
}

func (l *Lobby) LobbyWork() {
	fmt.Println("im in")
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

func (l *Lobby) ConnHandle(plr *player.PlayUnit) {
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
		fmt.Println("rip connection")

	}()
	for _, v := range l.Players { // load current state of the game
		if v.Place != plr.Place {
			v = v.PrivateSend()
		}
		err := plr.Conn.WriteJSON(v)
		if err != nil {
			fmt.Println(err, "WriteJSON start")
		}
		fmt.Println("start")
	}
	for {
		ctrl := player.PlayUnit{}
		err := plr.Conn.ReadJSON(&ctrl)
		if err != nil {
			fmt.Println(err, "conn read error")
			plr.Conn = nil
			break
		}
		plr.Bet = plr.Bet + ctrl.Bet
		l.PlayerCh <- *plr

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
func (l *Lobby) tickerTillNextTurn() {
	l.TurnTicker = time.NewTicker(time.Second * 1)
	timevalue := 0
	for range l.TurnTicker.C {
		timevalue++
		l.PlayerBroadcast <- l.Players[l.Order].SendTimeValue(timevalue)
		if timevalue >= 30 {
			l.TurnTicker.Stop()
			l.PlayerCh <- l.Players[l.Order]
		}
	}
}
func (l *Lobby) Game() {
	var flop bool
	deck := poker.NewDeck()
	deck.Shuffle()
	for i, v := range l.Players {
		v.Cards = deck.Draw(2)
		v.Place = i
		if i+1 == len(l.Players) {
			v.NextPlayer = &l.Players[0]
		} else {
			v.NextPlayer = &l.Players[i+1]
		}
	}
	l.Board = player.PlayUnit{}
	l.PlayerBroadcast = make(chan player.PlayUnit)
	button := randfuncs.NewSource().Intn(len(l.Players))
	l.Order = button
	go l.tickerTillNextTurn()
	for pl := range l.PlayerCh { // evaluating hand
		if pl.Place == l.Order {
			l.TurnTicker.Stop()
			if pl.Bet >= l.MaxBet {
				l.MaxBet = pl.Bet
			} else {
				pl.IsFold = true
			}
			pl.HasActed = true
			for pl.IsFold {
				pl = *pl.NextPlayer
			}
			if pl.Bet == l.MaxBet && pl.HasActed {
				if flop {
					l.Board.Cards = append(l.Board.Cards, pl.Cards...)
				} else {
					l.Board.Cards = pl.Cards
				}
			}
			l.Order = pl.Place
			go l.tickerTillNextTurn()
		}
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
			fmt.Println(v, "checkerv1")
		}
	}
}
