package hub

import (
	"log"
	"net/http"

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
	pl, ok := h.Players[user_id]
	if !ok || ok && len(pl.URLlobby) == 0 {
		pl = &lobby.PlayUnit{User_id: user_id, Name: name, Conn: conn}
		h.ReqPlayers <- pl
		// ping pong implementation here
	} else if ok && len(pl.URLlobby) != 0 {
		pl.Conn.WriteJSON(pl)
	}

}
