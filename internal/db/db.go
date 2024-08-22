package db

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	Pool *pgxpool.Pool
}

func NewPool(ctx context.Context, addr string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &Pool{
		Pool: pool,
	}, nil
}

func (p *Pool) NewUser(ctx context.Context, sub, username string) error {
	if strings.ContainsAny(username, " \t\n") {
		return errors.New("contains spaces")
	}
	_, err := p.Pool.Exec(ctx, "INSERT INTO users (subject,username) VALUES ($1,$2)", sub, username)
	return err
}

func (p *Pool) GetRatings(user_id string) (pgx.Rows, error) {

	rows, err := p.Pool.Query(context.Background(), `WITH common_query AS (
    	SELECT users.name, ratings.rating 
    	FROM ratings 
    	JOIN users ON ratings.user_id = users.user_id )
		SELECT * FROM common_query WHERE ratings.user_id = $1
		UNION ALL
		SELECT * FROM common_query ORDER BY rating DESC LIMIT 100;`, user_id)
	return rows, err
}
func (p *Pool) ChangeRating(user_id int, rating int) error {
	_, err := p.Pool.Exec(context.Background(), `INSERT INTO ratings (user_id, rating) VALUES ($1,$2) 
	ON CONFLICT (user_id) DO UPDATE SET rating = ratings.rating + $2`, user_id, rating)
	return err
}
func (p *Pool) GetUser(sub string) (int, string, error) {
	var user_id int
	var name string
	err := p.Pool.QueryRow(context.Background(), `SELECT user_id,name FROM users WHERE subject = $1`, sub).Scan(&user_id, &name)
	if err != nil {
		return 0, "", err
	}
	return user_id, name, err
}
