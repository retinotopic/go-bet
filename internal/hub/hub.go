package hub

import (
	"sync"
	"time"

	"github.com/retinotopic/go-bet/internal/lobby"
	"github.com/retinotopic/go-bet/internal/player"
)

type Hub struct {
	Lobby      map[string]*lobby.Lobby
	ReqPlayers chan player.PlayUnit
	wg         sync.WaitGroup
}

func NewHub() *Hub {
	return &Hub{ReqPlayers: make(chan player.PlayUnit, 1250)}
}

func (h *Hub) GreenReceive() {
	plrs := make([]player.PlayUnit, 0, 8)
	for i := range h.ReqPlayers {
		plrs = append(plrs, i)
		if len(plrs) == 8 {
			go lobby.NewLobby(plrs).LobbyWork()
			h.wg.Done()
			return
		}
	}
}
func (h *Hub) Requests() {
	t := time.NewTicker(time.Millisecond * 25)
	for range t.C {
		s := len(h.ReqPlayers) / 8
		for range s {
			h.wg.Add(1)
			go h.GreenReceive()
		}
		h.wg.Wait()
		t.Reset(time.Millisecond * 25)
	}
}
