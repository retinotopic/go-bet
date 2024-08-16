package middleware

import (
	"context"
	"log"
)

type IdentScope struct {
	User_id int
	Ident   string
	Name    string
}
type userCtxKeyType string

const userCtxKey userCtxKeyType = "user"

func WithUser(ctx context.Context, ident IdentScope) context.Context {
	return context.WithValue(ctx, userCtxKey, ident)
}

func GetUser(ctx context.Context) (int, string, string) {
	user, ok := ctx.Value(userCtxKey).(IdentScope)
	if !ok {
		log.Println("retrieve user id from context error")
		return 0, "", ""
	}
	return user.User_id, user.Ident, user.Name
}
