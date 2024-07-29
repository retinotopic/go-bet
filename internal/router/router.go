package router

import (
	"net/http"

	"github.com/retinotopic/go-bet/internal/auth"
	"github.com/retinotopic/go-bet/internal/hub"
	"github.com/retinotopic/go-bet/internal/queue"
)

type Router struct {
	addr string
}

func NewRouter(addr string) *Router {
	return &Router{addr: addr}
}
func (r *Router) Run() error {
	queue.Queue.ProcessConsume()
	mux := http.NewServeMux()
	mux.HandleFunc("/lobby", hub.Hub.ConnectLobby)
	mux.HandleFunc("/findgame", hub.Hub.FindGame)
	mux.HandleFunc("/beginauth", auth.Mproviders.BeginAuth)
	mux.HandleFunc("/completeauth", auth.Mproviders.CompleteAuth)
	return http.ListenAndServe(r.addr, mux)
}
