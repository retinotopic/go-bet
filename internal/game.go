package hub

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/retinotopic/go-bet/internal/player"
	"github.com/retinotopic/pokerGO/pkg/randfuncs"
)

func NewLobby() *Lobby {
	l := &Lobby{Players: make([]*player.Player, 0), Occupied: make(map[int]bool), PlayerCh: make(chan player.Player), StartGame: make(chan struct{}, 1)}
	return l
}

type Lobby struct {
	Players  []*player.Player
	Admin    *player.Player
	Occupied map[int]bool
	sync.Mutex
	GameState int
	AdminOnce sync.Once
	PlayerCh  chan player.Player
	StartGame chan struct{}
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

func (l *Lobby) Connhandle(plr *player.Player, conn *websocket.Conn) {
	fmt.Println("im in2")
	l.AdminOnce.Do(func() {
		l.Admin = plr
		plr.Admin = true
	})

	defer func() {
		fmt.Println("rip connection")

	}()
	plr.Conn = conn
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
func (l *Lobby) Game() {
	plorder := []player.Player{}
	for i, v := range l.Players {
		plorder = append(plorder, *v)
		v.Place = i
	}
	timer := time.NewTicker(time.Second * 1)
	PlayerBroadcast := make(chan player.Player)
	timevalue := 0
	k := randfuncs.NewSource().Intn(len(l.Occupied))
	for {
		select {
		case pb := <-PlayerBroadcast: // broadcoasting one to everyone
			for _, v := range l.Players {
				if pb != *v {
					pb = v.PrivateSend()
				}
				v.Conn.WriteJSON(pb)
				fmt.Println(v, "checkerv1")
			}
		case <-timer.C:
			timevalue++
			PlayerBroadcast <- plorder[k].SendTimeValue(timevalue)
			if timevalue >= 30 {
				timer.Stop()
			}
		case pl := <-l.PlayerCh: // evaluating hand
			if pl.Place == k {
				// evaluate hand
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
