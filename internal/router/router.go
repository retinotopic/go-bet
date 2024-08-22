package router

import (
	"context"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
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
	sqldb := stdlib.OpenDBFromPool(db.Pool)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.Up(sqldb, "migrations"); err != nil {
		return err
	}
	if err := sqldb.Close(); err != nil {
		return err
	}
	queue := queue.NewQueue(r.AddrQueue, r.ConfigQueue.Consume, r.ConfigQueue.QueueDeclare, db.ChangeRating)
	queue.TryConnect()

	hub := hub.NewPump(1250, queue)

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
