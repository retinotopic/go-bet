package middleware

import (
	"context"
	"log"
)

type IdentScope struct {
	User_id string
	Name    string
}
type userCtxKeyType string

const userCtxKey userCtxKeyType = "user"

func WithUser(ctx context.Context, ident IdentScope) context.Context {
	return context.WithValue(ctx, userCtxKey, ident)
}

func GetUser(ctx context.Context) (string, string) {
	user, ok := ctx.Value(userCtxKey).(IdentScope)
	if !ok {
		log.Println("retrieve user id from context error")
		return "", ""
	}
	return user.User_id, user.Name
}
