package hub

import (
	"net/http"
	"strconv"
	"unicode"

	"github.com/coder/websocket"
	"github.com/retinotopic/go-bet/internal/hub/lobby"
	"github.com/retinotopic/go-bet/internal/middleware"
)

func (h *HubPump) FindGame(w http.ResponseWriter, r *http.Request) {
	user_id, name := middleware.GetUser(r.Context())
	if !isNumeric(user_id) {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	url, ok := h.players.Load(user_id)

	if ok && len(url) != 0 {
		w.Write([]byte(url))
	} else {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.reqPlayers <- &lobby.PlayUnit{User_id: user_id, Name: name, Conn: conn}
	}

}
func (h *HubPump) ConnectLobby(w http.ResponseWriter, r *http.Request) {
	//check for player presence in map
	user_id, name := middleware.GetUser(r.Context())

	wsurl, err := strconv.ParseUint(r.URL.Path[7:], 10, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	lb, ok := h.lobby.Load(wsurl)

	if ok {
		var plr *lobby.PlayUnit

		lb.MapUsers.Mtx.Lock()
		pl, ok := lb.MapUsers.M[user_id]
		if ok {
			plr = pl
		} else {
			plr = &lobby.PlayUnit{User_id: user_id, Name: name}
			lb.MapUsers.M[user_id] = plr
		}
		lb.MapUsers.Mtx.Unlock()

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		plr.Conn = conn
		go lb.HandleConn(plr)
	}
}
func isNumeric(s string) bool {
	for _, char := range s {
		if !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}
