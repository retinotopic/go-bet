package middleware

import (
	"net/http"
	"strconv"

	"github.com/retinotopic/tinyauth/provider"
)

type CookieReadWriter interface {
	WriteCookie(http.ResponseWriter, string) *http.Cookie
	ReadCookie(*http.Request) (string, error)
}
type UserMiddleware struct {
	GetUser     func(string) (int, string, error)
	GetProvider func(http.ResponseWriter, *http.Request) (provider.Provider, error)
	CookieReadWriter
}

func (um *UserMiddleware) FetchUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user_id, name string
		prvdr, err := um.GetProvider(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		sub, err := prvdr.FetchUser(w, r)
		if err != nil {
			user_id, err = um.ReadCookie(r)
			if err != nil {
				um.WriteCookie(w, name)
			}
		} else {
			var ident int
			ident, name, err = um.GetUser(sub)
			user_id = strconv.Itoa(ident)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		idents := IdentScope{Name: name, User_id: user_id}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), idents)))
	})
}
