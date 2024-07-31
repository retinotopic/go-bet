package middleware

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/retinotopic/tinyauth/provider"
)

type UserMiddleware struct {
	GetUser     func(string) (int, string, error)
	GetProvider func(http.ResponseWriter, *http.Request) (provider.Provider, error)
	WriteCookie func(http.ResponseWriter, string) *http.Cookie
	ReadCookie  func(*http.Request) (string, error)
}

func (um *UserMiddleware) FetchUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user_id int
		var name, ident string
		prvdr, err := um.GetProvider(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		sub, err := prvdr.FetchUser(w, r)
		if err != nil {
			name, err = um.ReadCookie(r)
			if err != nil {
				name = uuid.New().String()
				um.WriteCookie(w, name)
			}
			ident = name

		} else {
			user_id, name, err = um.GetUser(sub)
			ident = strconv.Itoa(user_id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		idents := IdentScope{Ident: ident, Name: name, User_id: user_id}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), idents)))
	})
}
