package db

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/retinotopic/go-bet/internal/lobby"
	"github.com/retinotopic/go-bet/internal/player"
)

type PostgresClient struct {
	Sub          string
	Name         string
	UserID       uint32
	Conn         *pgxpool.Conn
	Mutex        sync.Mutex
	CurrentLobby *lobby.Lobby
	Player       *player.PlayUnit
}

func NewUser(ctx context.Context, sub, username string, pool *pgxpool.Pool) error {
	if strings.ContainsAny(username, " \t\n") {
		return errors.New("contains spaces")
	}
	_, err := pool.Exec(ctx, "INSERT INTO users (subject,username) VALUES ($1,$2)", sub, username)
	return err
}

func ConnectToDB(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	return db, ctx.Err()
}
func NewClient(ctx context.Context, sub string, pool *pgxpool.Pool) (*PostgresClient, error) {
	// check if user exists
	row := pool.QueryRow(ctx, "SELECT user_id,username FROM users WHERE subject=$1", sub)
	var name string
	var userid uint32
	err := row.Scan(&userid, &name)
	if err != nil {
		return nil, err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	pc := &PostgresClient{
		Sub:    sub,
		Conn:   conn,
		UserID: userid,
		Name:   name,
	}

	return pc, nil
}
