package hub

import (
	"net/http"
	"strconv"
	"time"
	"unicode"

	"github.com/fasthttp/websocket"
	"github.com/retinotopic/go-bet/internal/hub/lobby"
	"github.com/retinotopic/go-bet/internal/middleware"
	"github.com/retinotopic/go-bet/pkg/wsutils"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

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
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		go wsutils.KeepAlive(conn, time.Second*15)
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
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		go lb.HandleConn(&lobby.PlayUnit{User_id: user_id, Name: name, Conn: conn})
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
