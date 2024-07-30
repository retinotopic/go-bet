package hub

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/retinotopic/go-bet/internal/lobby"
	"github.com/retinotopic/go-bet/internal/middleware"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *hubPump) FindGame(w http.ResponseWriter, r *http.Request) {
	idents := middleware.GetUser(r.Context())
	if idents.User_id == 0 {
		http.Error(w, "user not found", http.StatusUnauthorized)
	}
	h.PlrMutex.RLock()
	pl, ok := h.Players[idents.Ident]
	h.PlrMutex.RUnlock()

	if !ok || ok && pl.URLlobby == 0 {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		pl = &lobby.PlayUnit{User_id: idents.User_id, Name: idents.Name, Conn: conn}
		h.ReqPlayers <- pl
		h.keepAlive(conn, time.Second*15)

		pl = &lobby.PlayUnit{}
		err = conn.ReadJSON(pl)
		if err != nil {
			log.Println(err)
		}
		err = conn.Close()
		if err != nil {
			log.Println(err)
		}

	} else {
		http.Error(w, "user not found", http.StatusUnauthorized)
	}
}
func (h *hubPump) ConnectLobby(w http.ResponseWriter, r *http.Request) {
	//check for player presence in map
	idents := middleware.GetUser(r.Context())
	h.PlrMutex.RLock()
	pl, ok := h.Players[idents.Ident]
	h.PlrMutex.RUnlock()

	wsurl, err := strconv.ParseUint(r.URL.Path[7:], 10, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if !ok {
		pl = &lobby.PlayUnit{User_id: idents.User_id, Name: idents.Name}
	}

	if pl.URLlobby == wsurl || pl.URLlobby == 0 {
		h.LMutex.RLock()
		lb, ok := h.Lobby[wsurl]
		h.LMutex.RUnlock()

		if pl.URLlobby == 0 && (lb.IsRating || lb.HasBegun.Load()) {
			http.Error(w, "custom game has already started", http.StatusNotFound)
			return
		}
		if pl.URLlobby == 0 && !ok {
			lb = lobby.NewLobby()
			h.LMutex.Lock()
			h.Lobby[wsurl] = lb
			h.LMutex.Unlock()
			go func() {
				lb.LobbyWork()
				h.LMutex.Lock()
				delete(h.Lobby, wsurl)
				h.LMutex.Unlock()
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
			go h.keepAlive(pl.Conn, time.Second*20)
			go lb.ConnHandle(pl)
		}

	} else {
		http.Error(w, "lobby not found", http.StatusNotFound)
	}
}

func (h *hubPump) keepAlive(c *websocket.Conn, timeout time.Duration) {
	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		for {
			err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			if err != nil {
				return
			}
			time.Sleep(timeout / 2)
			if time.Since(lastResponse) > timeout {
				c.Close()
				return
			}
		}
	}()
}
