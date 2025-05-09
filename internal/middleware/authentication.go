package middleware

import (
	"net/http"
)

type UserMiddleware struct {
	WriteCookie func(http.ResponseWriter) *http.Cookie
	ReadCookie  func(*http.Request, *http.Cookie) (string, error)
}

func (um *UserMiddleware) FetchUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user_id, name string

		user_id, err := um.ReadCookie(r, nil)
		if err != nil {
			user_id, err = um.ReadCookie(r, um.WriteCookie(w))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		name = "guest"
		idents := IdentScope{Name: name, User_id: user_id}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), idents)))
	})
}
