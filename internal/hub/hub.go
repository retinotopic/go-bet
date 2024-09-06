package hub

import (
	"hash/maphash"
	"strconv"
	"sync"
	"time"

	"github.com/retinotopic/go-bet/internal/hub/lobby"
)

func NewPump(lenBuffer int, queue lobby.Queue) *HubPump {
	hub := &HubPump{reqPlayers: make(chan lobby.PlayUnit, lenBuffer), queue: queue}
	hub.requests()
	return hub
}

type HubPump struct {
	lobby      map[uint64]*lobby.Lobby // url lobby
	lMutex     sync.RWMutex
	players    map[string]lobby.PlayUnit // id to player
	plrMutex   sync.RWMutex
	reqPlayers chan lobby.PlayUnit
	wg         sync.WaitGroup
	queue      lobby.Queue
}

func (h *HubPump) greenReceive() {
	plrs := make([]lobby.PlayUnit, 0, 8)
	lb := lobby.NewLobby(h.queue)
	for cl := range h.reqPlayers {
		plrs = append(plrs, cl)
		if len(plrs) == 8 {
			lb.Players = plrs
			url := new(maphash.Hash).Sum64()
			for _, v := range plrs {
				v.URLlobby = url

				h.plrMutex.Lock()
				h.players[strconv.Itoa(v.User_id)] = v
				h.plrMutex.Unlock()

				err := v.Conn.WriteJSON(v)
				if err != nil {
					v.Conn.WriteJSON(v)
				}
			}

			h.lMutex.Lock()
			h.lobby[url] = lb
			h.lMutex.Unlock()

			go func() {
				lb.LobbyWork()
				h.lMutex.Lock()
				delete(h.lobby, url)
				h.lMutex.Unlock()
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
