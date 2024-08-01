package router

import (
	"net/http"

	"github.com/retinotopic/go-bet/internal/auth"
	"github.com/retinotopic/go-bet/internal/hub"
)

type Router struct {
	Addr string
}

func NewRouter(addr string) *Router {

	return &Router{Addr: addr}
}
func (r *Router) Run() error {

	mux := http.NewServeMux()
	mux.HandleFunc("/lobby", hub.Hub.ConnectLobby)
	mux.HandleFunc("/findgame", hub.Hub.FindGame)
	mux.HandleFunc("/beginauth", auth.Mproviders.BeginAuth)
	mux.HandleFunc("/completeauth", auth.Mproviders.CompleteAuth)
	return http.ListenAndServe(r.Addr, mux)
}
