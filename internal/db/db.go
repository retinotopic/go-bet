package db

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgClient struct {
	Sub    string
	Name   string
	UserID uint32
	Mutex  sync.Mutex
}

var pool *pgxpool.Pool

func init() {
	var err error
	pool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func NewUser(ctx context.Context, sub, username string) error {
	if strings.ContainsAny(username, " \t\n") {
		return errors.New("contains spaces")
	}
	_, err := pool.Exec(ctx, "INSERT INTO users (subject,username) VALUES ($1,$2)", sub, username)
	return err
}
func NewClient(ctx context.Context, sub string) (*PgClient, error) {
	// check if user exists
	row := pool.QueryRow(ctx, "SELECT user_id,username FROM users WHERE subject=$1", sub)
	var name string
	var userid uint32
	err := row.Scan(&userid, &name)
	if err != nil {
		return nil, err
	}
	pc := &PgClient{
		Sub:    sub,
		UserID: userid,
		Name:   name,
	}

	return pc, nil
}

func ChangeRating(ctx context.Context, user_id string, rating int) error {
	_, err := pool.Exec(ctx, `INSERT INTO ratings (user_id, rating) VALUES ($1,$2) 
	ON CONFLICT (user_id) DO UPDATE SET rating = ratings.rating + $2`, user_id, rating)
	return err
}
