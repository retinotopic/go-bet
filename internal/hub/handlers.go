package hub

import (
	"log"
	"net/http"
	"strconv"
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
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	sub, err := prvdr.FetchUser(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	user_id, name, err := db.GetUser(sub)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.PlrMutex.RLock()
	pl, ok := h.Players[strconv.Itoa(user_id)]
	h.PlrMutex.RUnlock()

	if !ok || ok && len(pl.URLlobby) == 0 {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		pl = &lobby.PlayUnit{User_id: user_id, Name: name, Conn: conn}
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

	} else if ok && len(pl.URLlobby) != 0 {
		pl.Conn.WriteJSON(pl)
	}
}
func (h *hubPump) ConnectLobby(w http.ResponseWriter, r *http.Request) {
	var user_id int
	var name string
	prvdr, err := auth.Mproviders.GetProvider(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	sub, err := prvdr.FetchUser(w, r)
	if err != nil {
		name, err = auth.ReadCookie(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	} else {
		user_id, name, err = db.GetUser(sub)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	//check for player presence in map
	h.PlrMutex.RLock()
	pl := h.Players[strconv.Itoa(user_id)]
	h.PlrMutex.RUnlock()

	wsurl := r.URL.Path[7:]

	//check for lobby presence in map
	h.LMutex.RLock()
	lb, ok := h.Lobby[wsurl]
	h.LMutex.RUnlock()

	if ok && ((lb.IsRating && pl.IsRating && wsurl == pl.URLlobby) || (!pl.IsRating && !lb.IsRating && !lb.HasBegun.Load())) {
		pl.Conn = conn
		pl.Name = name
		pl.Conn.WriteJSON(pl)
		h.LMutex.RLock()
		if h.Lobby[pl.URLlobby] != nil {
			h.Lobby[pl.URLlobby].ConnHandle(pl)
		}
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
