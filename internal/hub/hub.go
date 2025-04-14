package hub

import (
	"bytes"
	crand "crypto/rand"
	"hash/maphash"
	"io"
	mrand "math/rand/v2"
	"strconv"
	"sync"
	"time"

	// "golang.org/x/sync/errgroup"

	"github.com/Nerdmaster/poker"
	// json "github.com/bytedance/sonic"

	"github.com/retinotopic/go-bet/internal/lobby"
)

// const
var strs = [4]string{`{"counter":"-1`, `{"URL":"`, `{"counter":"`, `"}`}

func NewHub(lenBuffer int) *Hub {
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
	qpool := &sync.Pool{
		New: func() any {
			plr := &awaitingPlayer{ReadB: make([]byte, 8)}
			return plr
		},
	}
	ap := make([]*awaitingPlayer, 8)
	hub := &Hub{reqPlayers: make(chan *awaitingPlayer, lenBuffer),
		lobby: sync.Map{}, players: sync.Map{}, lobbyPool: pool, queuedPool: qpool, plrs: ap,
	}
	hub.requests()
	return hub
}

type awaitingPlayer struct {
	UserId  string             `json:"-"`
	Name    string             `json:"-"`
	Counter int                `json:"counter"` // represents ActivityCounter
	Queued  bool               `json:"-"`
	QueueId int                `json:"-"`
	URL     string             `json:"URL"`
	Conn    io.ReadWriteCloser `json:"-"`
	ReadB   []byte             `json:"-"`
}

type Hub struct {
	lobby           sync.Map // URL to *Lobby map[uint64]*Lobby
	players         sync.Map // user_id to room URL map[string]uint64
	lobbyPool       *sync.Pool
	queuedPool      *sync.Pool
	reqPlayers      chan *awaitingPlayer
	buf             *bytes.Buffer
	ActivityCounter int
	stackids        []int
	plrs            []*awaitingPlayer
}

func (h *Hub) requests() {
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
			h.plrs[plr.QueueId] = plr
		case <-timer.C:
			required = 4
		}

		c := 0
		for i := range h.plrs {
			if h.plrs[i] != nil {
				if !h.plrs[i].Queued { //if the player has canceled the game search
					h.WriteNotification(h.plrs[i].Conn, 0, "")
					h.stackids = append(h.stackids, h.plrs[i].QueueId) // back to the pool of free stack indices
					h.queuedPool.Put(h.plrs[i])
					h.plrs[i] = nil
					h.ActivityCounter--
					c++
				} else if len(h.plrs[i].URL) != 0 { //if the game is found
					h.WriteNotification(h.plrs[i].Conn, 1, h.plrs[i].URL)
				} else if h.plrs[i].Counter != h.ActivityCounter { // update current players who are queued up
					h.plrs[i].Counter = h.ActivityCounter
					h.WriteNotification(h.plrs[i].Conn, 2, strconv.Itoa(h.plrs[i].Counter))
				}
			}
		}
		if len(h.plrs)-c == required {
			h.startRatingGame(h.plrs)
			clear(h.plrs)
			required = 8
			timer.Reset(time.Minute * 5)
		}

	}
}

func (h *Hub) CreateLobby(lb *lobby.Lobby) uint64 {
	loaded := true
	var hash uint64
	for loaded {
		hash = new(maphash.Hash).Sum64()
		_, loaded = h.lobby.LoadOrStore(hash, lb)
	}
	return hash
}

func (h *Hub) startRatingGame(plrs []*awaitingPlayer) {
	lb := h.lobbyPool.Get().(*lobby.Lobby)
	lb.Validate = lb.ValidateRating
	hash := h.CreateLobby(lb)

	for i, plr := range plrs {
		if plr != nil {
			lb.MapUsers.M[plr.UserId] = i
			h.players.Store(plr.UserId, hash)
			lb.LoadPlayer(plr.UserId, plr.Name)
		}
	}
	go func() {
		defer func() {
			for _, plr := range plrs {
				if plr != nil {
					h.players.Delete(plr.UserId)
				}
			}
		}()
		lb.Validate(lobby.Ctrl{Ctrl: 3})
		lb.LobbyStart()
		h.lobby.Delete(hash)
		h.lobbyPool.Put(lb)
	}()
}

func (h *Hub) FindGame(userid string, username string, wrc io.ReadWriteCloser) {
	urlAny, ok := h.players.Load(userid)
	if ok {
		url, ok := urlAny.(uint64)
		if ok {
			h.WriteNotification(wrc, 1, strconv.FormatUint(url, 10))
			wrc.Close()
		} else {
			plrAny := h.queuedPool.Get()
			plr, ok := plrAny.(*awaitingPlayer)
			if ok {
				plr.UserId, plr.Name, plr.Conn = userid, username, wrc
				defer func() {
					plr.Queued = false
				}()
				for {
					n, err := wrc.Read(plr.ReadB)
					if err != nil {
						wrc.Close()
					}
					if string(plr.ReadB[:n]) == "active" {
						plr.Queued = true
						plr.Counter = h.ActivityCounter
						h.reqPlayers <- plr
					} else if string(plr.ReadB[:n]) == "inactive" {
						plr.Queued = false
					}

				}
			}
		}
	}
}

func (h *Hub) WriteNotification(wrc io.ReadWriteCloser, id int, payload string) {
	h.buf.Reset()
	h.buf.WriteString(strs[id])
	if len(payload) != 0 {
		h.buf.WriteString(payload)
	}
	h.buf.WriteString(strs[3])
	wrc.Write(h.buf.Bytes())
}
