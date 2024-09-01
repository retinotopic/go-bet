package hub

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/retinotopic/go-bet/internal/lobby"
	"github.com/retinotopic/go-bet/internal/middleware"
	"github.com/retinotopic/go-bet/pkg/wsutils"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *HubPump) FindGame(w http.ResponseWriter, r *http.Request) {
	user_id, ident, name := middleware.GetUser(r.Context())
	if user_id == 0 {
		http.Error(w, "user not found", http.StatusUnauthorized)
	}
	h.plrMutex.RLock()
	pl, ok := h.players[ident]
	h.plrMutex.RUnlock()

	if !ok || ok && pl.URLlobby == 0 {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		go wsutils.KeepAlive(conn, time.Second*15)
		h.reqPlayers <- lobby.PlayUnit{User_id: user_id, Name: name, Conn: conn}

		wsutils.KeepAlive(conn, time.Second*15)

	} else {
		http.Error(w, "user not found", http.StatusUnauthorized)
	}
}
func (h *HubPump) ConnectLobby(w http.ResponseWriter, r *http.Request) {
	//check for player presence in map
	user_id, ident, name := middleware.GetUser(r.Context())
	h.plrMutex.RLock()
	pl, ok := h.players[ident]
	h.plrMutex.RUnlock()

	wsurl, err := strconv.ParseUint(r.URL.Path[7:], 10, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if !ok {
		pl = lobby.PlayUnit{User_id: user_id, Name: name}
	}

	if pl.URLlobby == wsurl || pl.URLlobby == 0 {
		h.lMutex.RLock()
		lb, ok := h.lobby[wsurl]
		h.lMutex.RUnlock()

		if pl.URLlobby == 0 && (lb.IsRating || lb.HasBegun.Load()) {
			http.Error(w, "custom game has already started", http.StatusNotFound)
			return
		}
		if pl.URLlobby == 0 && !ok {
			lb = lobby.NewLobby(nil)
			h.lMutex.Lock()
			h.lobby[wsurl] = lb
			h.lMutex.Unlock()
			go func() {
				defer h.lMutex.Unlock()
				lb.LobbyWork()
				h.lMutex.Lock()
				delete(h.lobby, wsurl)
			}()
			ok = !ok
		}
		if ok {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Println(err)
				return
			}
			pl.Conn = conn
			go lb.ConnHandle(pl)
		}

	} else {
		http.Error(w, "lobby not found", http.StatusNotFound)
	}
}
