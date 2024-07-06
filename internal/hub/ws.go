package hub

import (
	"log"
	"net/http"

	"github.com/fasthttp/websocket"
)

type wsurl struct {
	Url string `json:"url"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *Hub) FindGame(w http.ResponseWriter, r *http.Request) {
	_, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

}
