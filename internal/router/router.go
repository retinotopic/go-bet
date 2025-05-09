package router

import (
	"net/http"

	"github.com/retinotopic/go-bet/internal/auth"
	"github.com/retinotopic/go-bet/internal/hub"
	"github.com/retinotopic/go-bet/internal/middleware"
)

type Router struct {
	Addr       string
	RoutingKey string
}

func NewRouter(addr string) *Router {
	return &Router{Addr: addr}
}
func (r *Router) Run() error {

	hub := hub.NewHub(100)
	middleware := middleware.UserMiddleware{WriteCookie: auth.WriteCookie, ReadCookie: auth.ReadCookie}
	hConnectLobby := http.HandlerFunc(hub.ConnectLobby)
	hFindGame := http.HandlerFunc(hub.FindGameHandler)

	mux := http.NewServeMux()
	mux.Handle("/lobby/{roomId}", middleware.FetchUser(hConnectLobby))
	mux.Handle("/findgame", middleware.FetchUser(hFindGame))

	err := http.ListenAndServe(r.Addr, mux)
	if err != nil {
		return err
	}
	return nil
}
