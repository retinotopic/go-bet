package hub

import (
	"fmt"
	"sync"
	"time"

	"github.com/retinotopic/go-bet/internal/player"
	"github.com/retinotopic/go-bet/pkg/randfuncs"
)

func NewLobby() *Lobby {
	l := &Lobby{Players: make([]player.Player, 0), Occupied: make(map[int]bool), PlayerCh: make(chan player.Player), StartGame: make(chan struct{}, 1)}
	return l
}

type Lobby struct {
	Players    []player.Player
	Admin      player.Player
	Board      player.Player
	MaxBet     int
	Occupied   map[int]bool
	TurnTicker *time.Ticker
	sync.Mutex
	Order           int
	AdminOnce       sync.Once
	PlayerCh        chan player.Player
	StartGame       chan struct{}
	PlayerBroadcast chan player.Player
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

func (l *Lobby) ConnHandle(plr *player.Player) {
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
		ctrl := player.Player{}
		err := plr.Conn.ReadJSON(&ctrl)
		if err != nil {
			fmt.Println(err, "conn read error")
			plr.Conn = nil
			break
		}
		plr.Bet = ctrl.Bet
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
	for i, v := range l.Players {
		v.Place = i
	}
	l.Board = player.Player{}
	l.PlayerBroadcast = make(chan player.Player)
	button := randfuncs.NewSource().Intn(len(l.Players))
	l.Order = button
	go l.tickerTillNextTurn()
	for pl := range l.PlayerCh { // evaluating hand
		if pl.Place == l.Order {
			l.TurnTicker.Stop()
			if l.Order+1 == len(l.Players) {
				l.Order = 0
			}
			if pl.Bet >= l.MaxBet {
				l.Board = pl
				l.MaxBet = pl.Bet
			}
			l.Order++
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
