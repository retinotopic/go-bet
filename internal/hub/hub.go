package hub

import (
	"hash/maphash"
	"sync"

	csmap "github.com/mhmtszr/concurrent-swiss-map"

	"github.com/retinotopic/go-bet/internal/hub/lobby"
)

func NewPump(lenBuffer int, queue lobby.Queue) *HubPump {
	lm := csmap.Create[uint64, LobbyImpler]( // URL to lobby
		csmap.WithShardCount[uint64, LobbyImpler](128),
		csmap.WithSize[uint64, LobbyImpler](1000),
	)
	plm := csmap.Create[string, string]( // user_id to lobby URL
		csmap.WithShardCount[string, string](128),
		csmap.WithSize[string, string](1000),
	)
	hub := &HubPump{reqPlayers: make(chan *lobby.PlayUnit, lenBuffer), lobby: lm, players: plm}
	hub.requests()
	return hub
}

type LobbyImpler interface {
	HandleConn(lobby.PlayUnit)
	Broadcast()
	LobbyStart()
}
type HubPump struct {
	lobby      *csmap.CsMap[uint64, LobbyImpler]
	players    *csmap.CsMap[string, string]
	reqPlayers chan *lobby.PlayUnit
	wg         sync.WaitGroup
}

func (h *HubPump) requests() {
	plrs := make([]*lobby.PlayUnit, 0, 8)
	lb := lobby.NewLobby()
	for cl := range h.reqPlayers {
		plrs = append(plrs, cl)
		if len(plrs) == 8 {
			lb.Players = plrs
			url := new(maphash.Hash).Sum64()
			for _, v := range plrs {
				v.URLlobby = url
				v.Conn.WriteJSON(v)
			}

			h.lobby.Store(url, lb)

			go func() {
				lb.LobbyWork()
				h.lobby.Delete(url)

			}()
			h.wg.Done()
			return
		}
	}
}
