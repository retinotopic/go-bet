package hub

import (
	"hash/maphash"
	"strconv"
	"sync"
	"time"

	csmap "github.com/mhmtszr/concurrent-swiss-map"

	"github.com/retinotopic/go-bet/internal/hub/lobby"
)

func NewPump(lenBuffer int, queue lobby.Queue) *HubPump {
	lm := csmap.Create[uint64, LobbyImpler](
		csmap.WithShardCount[uint64, LobbyImpler](128),
		csmap.WithSize[uint64, LobbyImpler](1000),
	)
	plm := csmap.Create[string, lobby.PlayUnit](
		csmap.WithShardCount[string, lobby.PlayUnit](128),
		csmap.WithSize[string, lobby.PlayUnit](1000),
	)
	hub := &HubPump{reqPlayers: make(chan lobby.PlayUnit, lenBuffer), lobby: lm, players: plm}
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
	lMutex     sync.RWMutex
	players    *csmap.CsMap[string, lobby.PlayUnit] // id to player
	plrMutex   sync.RWMutex
	reqPlayers chan lobby.PlayUnit
	wg         sync.WaitGroup
}

func (h *HubPump) greenReceive() {
	plrs := make([]lobby.PlayUnit, 0, 8)
	lb := lobby.NewLobby()
	for cl := range h.reqPlayers {
		plrs = append(plrs, cl)
		if len(plrs) == 8 {
			lb.Players = plrs
			url := new(maphash.Hash).Sum64()
			for _, v := range plrs {
				v.URLlobby = url
				h.players.Store(strconv.Itoa(v.User_id), v)
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
func (h *HubPump) requests() {
	t := time.NewTicker(time.Millisecond * 25)
	t1 := time.NewTimer(time.Second * 30)
	for {
		select {
		case <-t.C:
			s := len(h.reqPlayers) / 8
			for range s {
				h.wg.Add(1)
				go h.greenReceive()
			}
			h.wg.Wait()
			t.Reset(time.Millisecond * 25)
			t1.Reset(time.Second * 30)
		case <-t1.C: //if there is no lobby after 30 seconds, the game will start if there are more than 1 player
			s := len(h.reqPlayers)
			if s > 1 {
				h.wg.Add(1)
				go h.greenReceive()
				h.wg.Wait()
				t.Reset(time.Millisecond * 25)
				t1.Reset(time.Second * 30)
			}
		}
	}
}
