package hub

import (
	"context"
	"hash/maphash"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
	csmap "github.com/mhmtszr/concurrent-swiss-map"

	"github.com/retinotopic/go-bet/internal/lobby"
)

func NewPump(lenBuffer int, queue lobby.Queue) *HubPump {
	lm := csmap.Create[uint64, *lobby.Lobby]( // URL to room
		csmap.WithShardCount[uint64, *lobby.Lobby](128),
		csmap.WithSize[uint64, *lobby.Lobby](1000),
	)
	plm := csmap.Create[string, string]( // user_id to room URL
		csmap.WithShardCount[string, string](128),
		csmap.WithSize[string, string](1000),
	)
	hub := &HubPump{reqPlayers: make(chan *lobby.PlayUnit, lenBuffer), lobby: lm, players: plm}
	hub.requests()
	return hub
}

type HubPump struct {
	lobby      *csmap.CsMap[uint64, *lobby.Lobby]
	players    *csmap.CsMap[string, string]
	reqPlayers chan *lobby.PlayUnit
	wg         sync.WaitGroup
}

func (h *HubPump) requests() {
	plrs := make([]*lobby.PlayUnit, 0, 8)
	timer := time.NewTimer(time.Minute * 5)
	for {
		select {
		case cl := <-h.reqPlayers:
			plrs = append(plrs, cl)
			if len(plrs) == 8 {
				h.startRatingGame(plrs)
				plrs = make([]*lobby.PlayUnit, 0, 8)
				timer.Reset(time.Minute * 5)
			}
		case <-timer.C:
			if len(plrs) >= 4 {
				h.startRatingGame(plrs)
				plrs = make([]*lobby.PlayUnit, 0, 8)
			} else {
				for cl := range h.reqPlayers {
					plrs = append(plrs, cl)
					if len(plrs) >= 4 {
						h.startRatingGame(plrs)
						plrs = make([]*lobby.PlayUnit, 0, 8)
						break
					}
				}
			}
			timer.Reset(time.Minute * 5)
		}

	}
}

func (h *HubPump) startRatingGame(plrs []*lobby.PlayUnit) {
	lb := &lobby.Lobby{}
	gm := &lobby.Game{Lobby: lb}
	rtng := &lobby.RatingImpl{Game: gm}
	gm.Impl = rtng
	isSet := false
	for !isSet {
		hash := new(maphash.Hash).Sum64()
		h.lobby.SetIf(hash, func(previousVale *lobby.Lobby, previousFound bool) (value *lobby.Lobby, set bool) {
			if !previousFound {
				lb.AllUsers.M = make(map[string]*lobby.PlayUnit)
				url := strconv.FormatUint(hash, 10)
				urlb := []byte(`{"URL":"` + url + `"}`)
				for _, plr := range plrs {
					lb.AllUsers.M[plr.User_id] = plr
					url := strconv.FormatUint(hash, 10)
					h.players.Store(plr.User_id, url)
					plr.Place = 0
					WriteTimeout(time.Second*5, plr.Conn, urlb)
				}
				previousVale = lb
				previousFound = true
				isSet = true
				go lb.LobbyStart(gm)
			} else {
				previousFound = false
			}
			return previousVale, previousFound
		})
	}

}
func WriteTimeout(timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}
