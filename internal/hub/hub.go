package hub

import (
	"context"
	"hash/maphash"
	"strconv"
	// "sync"
	"time"

	"github.com/Nerdmaster/poker"
	"github.com/coder/websocket"
	csmap "github.com/mhmtszr/concurrent-swiss-map"

	"github.com/retinotopic/go-bet/internal/lobby"
)

func NewPump(lenBuffer int, queue lobby.Queue) *Hub {
	lm := csmap.Create( // URL to room
		csmap.WithShardCount[uint64, *lobby.Lobby](208),
		csmap.WithSize[uint64, *lobby.Lobby](624),
	)
	plm := csmap.Create( // user_id to room URL
		csmap.WithShardCount[string, *awaitingPlayer](1664),
		csmap.WithSize[string, *awaitingPlayer](4992),
	)
	hub := &Hub{reqPlayers: make(chan *awaitingPlayer, lenBuffer), lobby: lm, players: plm}
	hub.requests()
	return hub
}

type Databaser interface {
	GetRatings(user_id string) ([]byte, error)
}
type awaitingPlayer struct {
	User_id string `json:"-"`
	Name    string `json:"-"`
	Counter int    `json:"counter"` // represents ActivityCounter
	Queued  bool   `json:"-"`
	QueueId int    `json:"-"`
	URL     string `json:"URL"`
	Conn    *websocket.Conn
}
type Hub struct {
	db                Databaser
	lobby             *csmap.CsMap[uint64, *lobby.Lobby]    // URL to room
	players           *csmap.CsMap[string, *awaitingPlayer] // user_id to waiting player OR room URL
	reqPlayers        chan *awaitingPlayer
	ActivityCounter   int
	stackids          []int
	ratingsCache      []byte
	lastRatingsUpdate time.Time
}

func (h *Hub) requests() {
	plrs := make([]*awaitingPlayer, 8)
	h.stackids = []int{8, 7, 6, 5, 4, 3, 2, 1}
	timer := time.NewTimer(time.Minute * 5)
	required := 8 //required players to start the game
	for {
		select {
		case plr := <-h.reqPlayers:
			last := len(h.stackids) - 1
			plr.QueueId = h.stackids[last]
			h.stackids = h.stackids[:last]
			plr.Counter = h.ActivityCounter
			plrs[plr.QueueId] = plr
		case <-timer.C:
			required = 4
		}

		c := 0
		for i := range plrs {
			if plrs[i] != nil {
				if !plrs[i].Queued { //if the player has canceled the game search
					WriteTimeout(time.Second*5, plrs[i].Conn, []byte(`{"counter":"`+"-1"+`"}`))
					h.stackids = append(h.stackids, plrs[i].QueueId) // back to the pool of free stack indices
					plrs[i] = nil
					h.ActivityCounter--
					c++
				} else if len(plrs[i].URL) != 0 { //if the game is found
					WriteTimeout(time.Second*5, plrs[i].Conn, []byte(`{"URL":"`+plrs[i].URL+`"}`))
				} else if plrs[i].Counter != h.ActivityCounter { // update current players who are queued up
					plrs[i].Counter = h.ActivityCounter
					WriteTimeout(time.Second*5, plrs[i].Conn, []byte(`{"counter":"`+strconv.Itoa(plrs[i].Counter)+`"}`))
				}
			}
		}
		if len(plrs)-c == required {
			h.startRatingGame(plrs)
			clear(plrs)
			required = 8
			timer.Reset(time.Minute * 5)
		}

	}
}

func (h *Hub) startRatingGame(plrs []*awaitingPlayer) {
	lb := &lobby.Lobby{}
	gm := &lobby.Game{Lb: lb}
	rtng := &lobby.RatingImpl{Game: gm}
	gm.Impl = rtng
	isSet := false
	for !isSet {
		hash := new(maphash.Hash).Sum64()
		h.lobby.SetIf(hash, func(previousVale *lobby.Lobby, previousFound bool) (value *lobby.Lobby, set bool) {
			if !previousFound {
				lb.Pr.AllUsers.M = make(map[string]*lobby.PlayUnit)
				url := strconv.FormatUint(hash, 10)
				//urlb := []byte(`{"URL":"` + url + `"}`)
				for _, plr := range plrs {
					if plr != nil {
						p := &lobby.PlayUnit{User_id: plr.User_id, Name: plr.Name, Cards: make([]string, 0, 3), CardsEval: make([]poker.Card, 0, 2)}
						lb.Pr.AllUsers.M[plr.User_id] = p
						plr.URL = url
						p.Place = 0
					}
				}
				previousVale = lb
				previousFound = true
				isSet = true
				go func() {
					for _, plr := range plrs {
						if plr != nil {
							defer h.players.Delete(plr.User_id)
						}
					}
					rtng.Validate(lobby.Ctrl{Ctrl: 3})
					rtng.Lb.LobbyStart(gm)
				}()
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
func ReadTimeout(timeout time.Duration, c *websocket.Conn) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, b, err := c.Read(ctx)
	return b, err
}
