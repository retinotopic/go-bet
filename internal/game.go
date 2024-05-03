package hub

import (
	"fmt"
	"sync"
	"time"

	"github.com/retinotopic/go-bet/internal/player"
	"github.com/retinotopic/go-bet/pkg/deck"
	"github.com/retinotopic/go-bet/pkg/randfuncs"
)

func NewLobby() *Lobby {
	l := &Lobby{Players: make([]*player.Player, 0), Occupied: make(map[int]bool), PlayerCh: make(chan player.Player), StartGame: make(chan struct{}, 1)}
	return l
}

type ControlJSON struct {
	Bet      int          `json:"Bet"`
	ValueSec int          `json:"Time,omitempty"`
	Place    int          `json:"Place"`
	Cards    [5]deck.Card `json:"Hand,omitempty"`
}

type Lobby struct {
	Players  []*player.Player
	Admin    *player.Player
	Occupied map[int]bool
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
			l.Admin = plr
			plr.Admin = true
		}
	})

	defer func() {
		fmt.Println("rip connection")

	}()
	for _, v := range l.Players { // load current state of the game
		vs := *v
		if vs != *plr {
			vs = v.PrivateSend()
		}
		err := plr.Conn.WriteJSON(vs)
		if err != nil {
			fmt.Println(err, "WriteJSON start")
		}
		fmt.Println("start")
	}
	for {
		player := player.Player{}
		err := plr.Conn.ReadJSON(player)
		if err != nil {
			fmt.Println(err, "conn read error")
			plr.Conn = nil
			break
		}
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
	timer := time.NewTicker(time.Second * 1)
	timevalue := 0
	for range timer.C {
		timevalue++
		l.PlayerBroadcast <- l.Players[l.Order].SendTimeValue(timevalue)
		if timevalue >= 30 {
			timer.Stop()
		}
	}
}
func (l *Lobby) Game() {
	for i, v := range l.Players {
		v.Place = i
	}
	l.PlayerBroadcast = make(chan player.Player)
	l.Order = randfuncs.NewSource().Intn(len(l.Players))
	go l.tickerTillNextTurn()
	for {
		select {
		case pb := <-l.PlayerBroadcast: // broadcoasting one to everyone
			for _, v := range l.Players {
				if pb != *v {
					pb = v.PrivateSend()
				}
				v.Conn.WriteJSON(pb)
				fmt.Println(v, "checkerv1")
			}
		case pl := <-l.PlayerCh: // evaluating hand
			if pl.Place == l.Order {
				// evaluate hand
				l.Order++
			}
			for _, v := range l.Players {
				pls := pl
				if pls != *v {
					pls = v.PrivateSend()
				}
				v.Conn.WriteJSON(pls)
				fmt.Println(v, "checkerv2")
			}
		}
	}
}
