package middleware

import (
	"net/http"
	"strconv"

	"github.com/retinotopic/tinyauth/provider"
)

type UserMiddleware struct {
	GetUser     func(string) (int, string, error)
	GetProvider func(http.ResponseWriter, *http.Request) (provider.Provider, error)
	WriteCookie func(http.ResponseWriter) *http.Cookie
	ReadCookie  func(*http.Request, *http.Cookie) (string, error)
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
			user_id, err = um.ReadCookie(r, nil)
			if err != nil {
				user_id, err = um.ReadCookie(r, um.WriteCookie(w))
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			name = "guest"
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
