package hub

import (
	"log"
	"net/http"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/retinotopic/go-bet/internal/auth"
	"github.com/retinotopic/go-bet/internal/db"
	"github.com/retinotopic/go-bet/internal/lobby"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *hubPump) FindGame(w http.ResponseWriter, r *http.Request) {
	prvdr, err := auth.Mproviders.GetProvider(w, r)
	if err != nil {
		http.Error(w, "error retrieving user", 500)
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sub, err := prvdr.FetchUser(w, r)
	if err != nil {
		http.Error(w, "error retrieving user", 500)
	}
	user_id, name, err := db.GetUser(sub)
	if err != nil {
		http.Error(w, "error retrieving user", 500)
		return
	}
	h.PlrMutex.RLock()
	pl, ok := h.Players[user_id]
	h.PlrMutex.RUnlock()

	if !ok || ok && len(pl.URLlobby) == 0 {
		pl = &lobby.PlayUnit{User_id: user_id, Name: name, Conn: conn}
		h.ReqPlayers <- pl
		h.keepAlive(conn, time.Second*15)

		pl = &lobby.PlayUnit{}
		err := conn.ReadJSON(pl)
		if err != nil {
			log.Println(err)
		}
		conn.Close()
		if err != nil {
			log.Println(err)
		}

	} else if ok && len(pl.URLlobby) != 0 {
		pl.Conn.WriteJSON(pl)
	}
}
func (h *hubPump) ConnectLobby(w http.ResponseWriter, r *http.Request) {
	prvdr, err := auth.Mproviders.GetProvider(w, r)
	if err != nil {
		http.Error(w, "error retrieving user", 500)
	}
	sub, err := prvdr.FetchUser(w, r)
	if err != nil {
		http.Error(w, "error retrieving user", 500)
	}
	user_id, name, err := db.GetUser(sub)
	if err != nil {
		http.Error(w, "error retrieving user", 500)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	h.PlrMutex.RLock()
	pl, ok := h.Players[user_id]
	h.PlrMutex.RUnlock()

	wsurl := r.URL.Path[7:]
	if wsurl == pl.URLlobby && ok {
		pl.Conn = conn
		pl.Name = name
		pl.Conn.WriteJSON(pl)
		h.LMutex.RLock()
		h.Lobby[pl.URLlobby].ConnHandle(pl)
		h.LMutex.Unlock()
	} else {
		conn.Close()
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
