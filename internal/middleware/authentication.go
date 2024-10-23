package middleware

import (
	"net/http"
	"strconv"
)

type UserMiddleware struct {
	GetUser    func(string) (int, string, error)
	GetSubject func(w http.ResponseWriter, r *http.Request) (string, error)

	WriteCookie func(http.ResponseWriter) *http.Cookie
	ReadCookie  func(*http.Request, *http.Cookie) (string, error)
}

func (um *UserMiddleware) FetchUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user_id, name string

		sub, err := um.GetSubject(w, r)
		if err != nil {
			user_id, err = um.ReadCookie(r, nil)
			if err != nil {
				user_id, err = um.ReadCookie(r, um.WriteCookie(w))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			name = "guest"
		} else {
			var ident int
			ident, name, err = um.GetUser(sub)
			user_id = strconv.Itoa(ident)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		idents := IdentScope{Name: name, User_id: user_id}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), idents)))
	})
}
