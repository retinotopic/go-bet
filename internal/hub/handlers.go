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
		lb.MapTable.SetIf(user_id, func(previousVale *lobby.PlayUnit, previousFound bool) (value *lobby.PlayUnit, set bool) {
			if !previousFound {
				previousVale = &lobby.PlayUnit{User_id: user_id, Name: name}
			}
			return previousVale, !previousFound
		})
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
