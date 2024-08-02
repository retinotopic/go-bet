package router

import (
	"context"
	"net/http"
	"os"

	"github.com/retinotopic/go-bet/internal/auth"
	"github.com/retinotopic/go-bet/internal/db"
	"github.com/retinotopic/go-bet/internal/hub"
	"github.com/retinotopic/go-bet/internal/middleware"
	"github.com/retinotopic/go-bet/internal/queue"
)

type Router struct {
	Addr        string
	AddrQueue   string
	ConfigQueue queue.Config
	Queue       *queue.TaskQueue
	Auth        auth.ProviderMap
}

func NewRouter(addr string, addrQueue string, config queue.Config, mp auth.ProviderMap) *Router {
	return &Router{Addr: addr, AddrQueue: addrQueue, ConfigQueue: config, Auth: mp}
}
func (r *Router) Run() error {

	db, err := db.NewPool(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	r.Queue, err = queue.DeclareAndRun(r.AddrQueue, r.ConfigQueue.Consume, r.ConfigQueue.QueueDeclare, db.ChangeRating)
	if err != nil {
		return err
	}
	hub := hub.NewPump(1250)

	middleware := middleware.UserMiddleware{GetUser: db.GetUser, GetProvider: r.Auth.GetProvider, WriteCookie: auth.WriteCookie, ReadCookie: auth.ReadCookie}
	hConnectLobby := http.HandlerFunc(hub.ConnectLobby)
	hFindGame := http.HandlerFunc(hub.FindGame)

	mux := http.NewServeMux()
	mux.Handle("/lobby", middleware.FetchUser(hConnectLobby))
	mux.Handle("/findgame", middleware.FetchUser(hFindGame))
	mux.HandleFunc("/beginauth", r.Auth.BeginAuth)
	mux.HandleFunc("/completeauth", r.Auth.CompleteAuth)

	err = http.ListenAndServe(r.Addr, mux)
	if err != nil {
		return err
	}
	return nil
}
