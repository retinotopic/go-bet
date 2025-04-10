package hub

import (
	"io"
	"net/http"
	"strconv"

	"github.com/coder/websocket"
	"github.com/retinotopic/go-bet/internal/lobby"
	"github.com/retinotopic/go-bet/internal/middleware"
)

// github:coder/websocket DOESNT SUPPORT getting ReadWriteCloser FROM CONN!!!!!!!!!!!!!!!!
type ReadWriteCloserStruct struct {
	io.Reader
	io.WriteCloser
}

func (h *Hub) FindGameHandler(w http.ResponseWriter, r *http.Request) {
	userid, username := middleware.GetUser(r.Context())

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	wr, err := conn.Writer(r.Context(), websocket.MessageText)
	if err != nil {
		conn.Close(3008, "timeout")
	}
	_, rd, err := conn.Reader(r.Context())
	if err != nil {
		conn.Close(3008, "timeout")
	}
	h.FindGame(userid, username, ReadWriteCloserStruct{rd, wr})

}
func (h *Hub) ConnectLobby(w http.ResponseWriter, r *http.Request) {
	user_id, name := middleware.GetUser(r.Context())

	wsurl, err := strconv.ParseUint(r.PathValue("roomId"), 10, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	lb, ok := h.lobby.Load(wsurl)
	if ok {
		l, ok := lb.(*lobby.Lobby)
		if ok {
			idx, ok := l.LoadPlayer(user_id, name)
			if !ok {
				http.Error(w, "room is full", http.StatusBadRequest)
				return
			}
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				l.Exit(idx)
				return
			}
			wr, err := conn.Writer(r.Context(), websocket.MessageText)
			if err != nil {
				l.Exit(idx)
				conn.Close(3008, "timeout")
			}
			_, rd, err := conn.Reader(r.Context())
			if err != nil {
				l.Exit(idx)
				conn.Close(3008, "timeout")
			}
			l.HandlePlayer(idx, ReadWriteCloserStruct{rd, wr})
		}
	}
}
func (h *Hub) CreateLobbyHandler(w http.ResponseWriter, r *http.Request) {
	lb := h.lobbyPool.Get().(*lobby.Lobby)
	lb.Validate = lb.ValidateCustom
	url := h.CreateLobby(lb)
	r.SetPathValue("roomId", strconv.FormatUint(url, 10))
	http.Redirect(w, r, r.URL.Path, 301)
}
