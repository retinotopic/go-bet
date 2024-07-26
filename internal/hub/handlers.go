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

	} else {
		http.Error(w, "user not found", http.StatusUnauthorized)
	}
}
func (h *hubPump) ConnectLobby(w http.ResponseWriter, r *http.Request) {
	var user_id int
	var name, ident string
	prvdr, err := auth.Mproviders.GetProvider(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	sub, err := prvdr.FetchUser(w, r)
	if err != nil {
		name, err = auth.ReadCookie(r)
		ident = name
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	} else {
		user_id, name, err = db.GetUser(sub)
		ident = strconv.Itoa(user_id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	//check for player presence in map
	h.PlrMutex.RLock()
	pl, ok := h.Players[ident]
	h.PlrMutex.RUnlock()

	wsurl := r.URL.Path[7:]
	if !ok {
		pl = &lobby.PlayUnit{User_id: user_id, Name: name}
	}

	if pl.URLlobby == wsurl || len(pl.URLlobby) == 0 {
		h.LMutex.RLock()
		lb, ok := h.Lobby[wsurl]
		h.LMutex.RUnlock()

		if len(pl.URLlobby) == 0 && (lb.IsRating || lb.HasBegun.Load()) {
			http.Error(w, "custom game has already started", http.StatusNotFound)
			return
		}
		if ok {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Println(err)
				return
			}
			pl.Conn = conn
			go h.keepAlive(pl.Conn, time.Second*20)
			lb.ConnHandle(pl)
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
