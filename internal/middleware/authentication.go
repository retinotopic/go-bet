package middleware

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/retinotopic/go-bet/internal/auth"
	"github.com/retinotopic/go-bet/internal/db"
)

type IdentScope struct {
	Ident   string
	User_id int
	Name    string
}
type userCtxKeyType string

const userCtxKey userCtxKeyType = "user"

func WithUser(ctx context.Context, ident IdentScope) context.Context {
	return context.WithValue(ctx, userCtxKey, ident)
}

func GetUser(ctx context.Context) IdentScope {
	user, ok := ctx.Value(userCtxKey).(IdentScope)
	if !ok {
		log.Println()
		return IdentScope{}
	}
	return user
}
func FetchUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user_id int
		var name, ident string
		prvdr, err := auth.Mproviders.GetProvider(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		sub, err := prvdr.FetchUser(w, r)
		if err != nil {
			name, err = auth.ReadCookie(r)
			if err != nil {
				name = uuid.New().String()
				auth.WriteCookie(w, name)
			}
			ident = name

		} else {
			user_id, name, err = db.GetUser(sub)
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
