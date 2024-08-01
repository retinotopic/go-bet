package router

import (
	"net/http"

	"github.com/retinotopic/go-bet/internal/auth"
	"github.com/retinotopic/go-bet/internal/hub"
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
	var err error
	r.Queue, err = queue.DeclareAndRun(r.AddrQueue, r.ConfigQueue.Consume, r.ConfigQueue.QueueDeclare)
	if err != nil {
		return err
	}
	hub := hub.NewPump(1250)
	mux := http.NewServeMux()

	mux.HandleFunc("/lobby", hub.ConnectLobby)
	mux.HandleFunc("/findgame", hub.FindGame)
	mux.HandleFunc("/beginauth", r.Auth.BeginAuth)
	mux.HandleFunc("/completeauth", r.Auth.CompleteAuth)
	err = http.ListenAndServe(r.Addr, mux)
	if err != nil {
		return err
	}
	return nil
}
