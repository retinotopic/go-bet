package hub

import (
	"context"
	crand "crypto/rand"
	"hash/maphash"
	mrand "math/rand/v2"
	"strconv"
	"sync"
	"time"

	// "golang.org/x/sync/errgroup"

	"github.com/Nerdmaster/poker"
	// json "github.com/bytedance/sonic"
	"github.com/coder/websocket"

	"github.com/retinotopic/go-bet/internal/lobby"
)

func NewPump(lenBuffer int) *Hub {
	pool := &sync.Pool{
		New: func() any {
			lb := &lobby.Lobby{}

			var buf [32]byte
			bufs := make([]byte, 32)
			crand.Read(bufs) // generating cryptographically random slice of bytes
			copy(buf[:], bufs)
			rand := mrand.New(mrand.NewChaCha8(buf)) //  and using it as a ChaCha8 seed
			lb.Deck = lobby.NewDeck(rand)            // and using it as a source for mrand

			lb.TopPlaces = make([]lobby.Top, 0, 8) //---\
			lb.Winners = make([]int, 0, 8)         //	 >----- reusable memory for post-river evaluation
			lb.Losers = make([]int, 0, 8)          //---/

			lb.Board.CardsEval = make(poker.CardList, 7) // poker.CardList for evaluating hand
			lb.Board.Cards = make(poker.CardList, 0, 5)  // current cards on table for json sending

			lb.MsgCh = make(chan lobby.Ctrl, 8)
			lb.PlayerCh = make(chan lobby.Ctrl)
			lb.StartGameCh = make(chan bool, 1)
			lb.ValidateCh = make(chan lobby.Ctrl)
			lb.PlayerCh = make(chan lobby.Ctrl)

			lb.CheckTimeout = time.NewTicker(time.Minute * 3)
			lb.CheckTimeout.Stop()
			lb.TurnTimer = time.NewTimer(time.Second * 10)
			lb.TurnTimer.Stop()
			lb.BlindTimer = time.NewTimer(time.Minute * 3)
			lb.BlindTimer.Stop()

			lb.MapUsers.M = make(map[string]int)
			lb.AllUsers = make([]lobby.PlayUnit, 50)
			for i := range lb.AllUsers {
				lb.AllUsers[i].Cards = make(poker.CardList, 0, 2)
			}

			return lb
		},
	}

	hub := &Hub{reqPlayers: make(chan *awaitingPlayer, lenBuffer),
		lobby: sync.Map{}, players: sync.Map{}, lobbyPool: pool,
	}
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

	lastTryToConnect time.Time
	Conn             *websocket.Conn
}
type Hub struct {
	db                Databaser
	lobby             sync.Map // URL to *Lobby map[uint64]*Lobby
	players           sync.Map // user_id to room URL map[string]uint64
	lobbyPool         *sync.Pool
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
	lb := h.lobbyPool.Get().(*lobby.Lobby)
	lb.Validate = lb.ValidateRating
	loaded := true
	var hash uint64
	for loaded {
		hash = new(maphash.Hash).Sum64()
		_, loaded = h.lobby.LoadOrStore(hash, &lb)
	}

	for i, plr := range plrs {
		if plr != nil {
			lb.MapUsers.M[plr.User_id] = i
			h.players.Store(plr.User_id, hash)
			lb.LoadPlayer(plr.User_id, plr.Name)
		}
	}
	go func() {
		defer func() {
			for _, plr := range plrs {
				if plr != nil {
					h.players.Delete(plr.User_id)
				}
			}
		}()
		lb.Validate(lobby.Ctrl{Ctrl: 3})
		lb.LobbyStart()
		h.lobby.Delete(hash)
		h.lobbyPool.Put(lb)
	}()
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
