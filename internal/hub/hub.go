package hub

import (
	"sync"
	"time"

	"github.com/retinotopic/go-bet/internal/db"
	"github.com/retinotopic/go-bet/internal/lobby"
)

type Hub struct {
	Lobby      map[string]*lobby.Lobby
	Player     map[string]*lobby.PlayUnit
	ReqPlayers chan *db.PgClient
	wg         sync.WaitGroup
}

func NewHub() *Hub {
	return &Hub{ReqPlayers: make(chan *db.PgClient, 1250)}
}

func (h *Hub) GreenReceive() {
	plrs := make([]*lobby.PlayUnit, 0, 8)
	lb := lobby.NewLobby()
	for cl := range h.ReqPlayers {
		cl.CurrentLobby = lb
		cl.Player = &lobby.PlayUnit{}
		plrs = append(plrs, cl.Player)
		if len(plrs) == 8 {
			h.wg.Done()
			return
		}
	}
}
func (h *Hub) Requests() {
	t := time.NewTicker(time.Millisecond * 25)
	t1 := time.NewTimer(time.Second * 30)
	for {
		select {
		case <-t.C:
			s := len(h.ReqPlayers) / 8
			for range s {
				h.wg.Add(1)
				go h.GreenReceive()
			}
			h.wg.Wait()
			t.Reset(time.Millisecond * 25)
			t1.Reset(time.Second * 30)
		case <-t1.C:
			s := len(h.ReqPlayers)
			if s > 1 {
				for range s {
					h.wg.Add(1)
					go h.GreenReceive()
				}
				h.wg.Wait()
				t.Reset(time.Millisecond * 25)
				t1.Reset(time.Second * 30)
			}
		}
	}
}
