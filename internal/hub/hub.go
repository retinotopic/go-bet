package hub

import (
	"hash/maphash"
	"strconv"
	"sync"
	"time"

	"github.com/retinotopic/go-bet/internal/lobby"
	"github.com/retinotopic/go-bet/internal/queue"
)

func NewPump(lenBuffer int) *HubPump {
	hub := &HubPump{ReqPlayers: make(chan *lobby.PlayUnit, lenBuffer)}
	hub.requests()
	return hub
}

type HubPump struct {
	Lobby      map[uint64]*lobby.Lobby // url lobby
	LMutex     sync.RWMutex
	Players    map[string]*lobby.PlayUnit // id to player
	PlrMutex   sync.RWMutex
	ReqPlayers chan *lobby.PlayUnit
	wg         sync.WaitGroup
	Queue      *queue.TaskQueue
}

func (h *HubPump) greenReceive() {
	plrs := make([]*lobby.PlayUnit, 0, 8)
	lb := lobby.NewLobby()
	for cl := range h.ReqPlayers {
		plrs = append(plrs, cl)
		if len(plrs) == 8 {
			lb.Players = plrs
			url := new(maphash.Hash).Sum64()
			for _, v := range plrs {
				v.URLlobby = url

				h.PlrMutex.Lock()
				h.Players[strconv.Itoa(v.User_id)] = v
				h.PlrMutex.Unlock()

				err := v.Conn.WriteJSON(v)
				if err != nil {
					v.Conn.WriteJSON(v)
				}
			}

			h.LMutex.Lock()
			h.Lobby[url] = lb
			h.LMutex.Unlock()

			go func() {
				lb.LobbyWork()
				h.LMutex.Lock()
				delete(h.Lobby, url)
				h.LMutex.Unlock()
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
			s := len(h.ReqPlayers) / 8
			for range s {
				h.wg.Add(1)
				go h.greenReceive()
			}
			h.wg.Wait()
			t.Reset(time.Millisecond * 25)
			t1.Reset(time.Second * 30)
		case <-t1.C: //if there is no lobby after 30 seconds, the game will start if there are more than 1 player
			s := len(h.ReqPlayers)
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
