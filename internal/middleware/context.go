package middleware

import (
	"context"
	"log"
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
