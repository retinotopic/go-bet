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
}

func NewRouter(addr string, addrQueue string, config queue.Config) *Router {
	return &Router{Addr: addr, AddrQueue: addrQueue, ConfigQueue: config}
}
func (r *Router) Run() error {
	var err error
	r.Queue, err = queue.DeclareAndRun(r.AddrQueue, r.ConfigQueue.Consume, r.ConfigQueue.QueueDeclare)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/lobby", hub.Hub.ConnectLobby)
	mux.HandleFunc("/findgame", hub.Hub.FindGame)
	mux.HandleFunc("/beginauth", auth.Mproviders.BeginAuth)
	mux.HandleFunc("/completeauth", auth.Mproviders.CompleteAuth)
	err = http.ListenAndServe(r.Addr, mux)
	if err != nil {
		return err
	}
	return nil
}
