package hub

import (
	"hash/maphash"
	"net/http"
	"strconv"
	"time"
	"unicode"

	"github.com/Nerdmaster/poker"
	"github.com/coder/websocket"
	"github.com/retinotopic/go-bet/internal/lobby"
	"github.com/retinotopic/go-bet/internal/middleware"
)

func (h *Hub) GetInitialData(user_id string, conn *websocket.Conn) (err error) {

	if time.Since(h.lastRatingsUpdate) > time.Duration(time.Minute*30) { // update top 100 players every 30 minutes
		var err error
		h.ratingsCache, err = h.db.GetRatings(user_id)
		if err != nil {
			conn.Close(websocket.StatusInternalError, "db fetch ratings error")
			return err
		}
		h.lastRatingsUpdate = time.Now()
		err = WriteTimeout(time.Second*5, conn, h.ratingsCache)
		if err != nil {
			conn.CloseNow()
			return err
		}
	}
	return err
}
func (h *Hub) FindGame(w http.ResponseWriter, r *http.Request) {
	user_id, username := middleware.GetUser(r.Context())
	if !isNumeric(user_id) {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.GetInitialData(user_id, conn)
	if err != nil {
		return
	}
	plr, ok := h.players.Load(user_id)
	if ok && len(plr.URL) != 0 {
		err = WriteTimeout(time.Second*5, conn, []byte(`{"URL":"`+plr.URL+`"}`))
		if err != nil {
			conn.CloseNow()
		}
	} else {
		plr := &awaitingPlayer{User_id: user_id, Name: username, Conn: conn}
		defer func() {
			plr.Queued = false
		}()
		for {
			b, err := ReadTimeout(time.Minute*5, conn)
			if err != nil {
				conn.CloseNow()
			}
			if string(b) == "active" {
				plr.Queued = true
				plr.Counter = h.ActivityCounter
				h.reqPlayers <- plr
			} else if string(b) == "inactive" {
				plr.Queued = false
			}

		}
	}
}
func (h *Hub) ConnectLobby(w http.ResponseWriter, r *http.Request) {
	//check for player presence in map
	user_id, name := middleware.GetUser(r.Context())

	wsurl, err := strconv.ParseUint(r.PathValue("roomId"), 10, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	lb, ok := h.lobby.Load(wsurl)

	if ok {
		var plr *lobby.PlayUnit

		lb.AllUsers.Mtx.Lock()
		pl, ok := lb.AllUsers.M[user_id]
		if ok {
			plr = pl
		} else {
			if len(lb.AllUsers.M) >= 15 {
				http.Error(w, "room is full", http.StatusBadRequest)
				return
			}
			plr = &lobby.PlayUnit{User_id: user_id, Name: name, Place: -2, Cards: make([]string, 0, 3), CardsEval: make([]poker.Card, 0, 2)}
			lb.AllUsers.M[user_id] = plr
		}
		lb.AllUsers.Mtx.Unlock()

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		plr.Conn = conn
		go lb.HandleConn(plr)
	}
}
func (h *Hub) CreateLobby(w http.ResponseWriter, r *http.Request) {
	user_id, name := middleware.GetUser(r.Context())

	lb := &lobby.Lobby{}
	gm := &lobby.Game{Lobby: lb}
	cstm := &lobby.CustomImpl{Game: gm}
	gm.Impl = cstm
	isSet := false
	var plr *lobby.PlayUnit
	for !isSet {
		hash := new(maphash.Hash).Sum64()
		h.lobby.SetIf(hash, func(previousVale *lobby.Lobby, previousFound bool) (value *lobby.Lobby, set bool) {
			if !previousFound {
				lb.AllUsers.M = make(map[string]*lobby.PlayUnit)
				plr = &lobby.PlayUnit{User_id: user_id, Name: name, Place: -2, Cards: make([]string, 0, 3), CardsEval: make([]poker.Card, 0, 2)}
				lb.AllUsers.M[user_id] = plr
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
	var err error
	plr.Conn, err = websocket.Accept(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go lb.HandleConn(plr)

}
func isNumeric(s string) bool {
	for _, char := range s {
		if !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}
