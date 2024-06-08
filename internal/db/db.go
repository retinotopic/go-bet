package db

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/retinotopic/go-bet/internal/lobby"
	"github.com/retinotopic/go-bet/internal/player"
)

type PgClient struct {
	Sub              string
	Name             string
	UserID           uint32
	Conn             *pgxpool.Conn
	Mutex            sync.Mutex
	RoomsPagination  []uint32
	RoomsCount       uint8 // no more than 250
	PaginationOffset uint8
	CurrentLobby     *lobby.Lobby
	Player           *player.PlayUnit
}

func ConnectToDB(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	return db, ctx.Err()
}
func NewClient(ctx context.Context, sub string, pool *pgxpool.Pool) (*PgClient, error) {
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
	pc := &PgClient{
		Sub:    sub,
		Conn:   conn,
		UserID: userid,
		Name:   name,
	}

	return pc, nil
}
