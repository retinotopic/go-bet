package router

import (
	"net/http"
)

type Router struct {
	addr string
}

func NewRouter(addr string) *Router {
	return &Router{addr: addr}
}
func (r *Router) Run() error {
	mux := http.NewServeMux()
	return http.ListenAndServe(r.addr, mux)
}
