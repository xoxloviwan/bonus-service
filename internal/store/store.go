package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	*pgxpool.Pool
}

func NewStore(ctx context.Context, connString string) (Store, error) {
	dbpool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return Store{}, err
	}
	return Store{dbpool}, nil
}

func (db Store) CreateUsersTable(ctx context.Context) error {
	_, err := db.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS users (
			id bigint GENERATED ALWAYS AS IDENTITY,
			login text NOT NULL UNIQUE,
			password text NOT NULL)`)
	return err
}

func (db Store) AddUser(login string, hash []byte) (int, error) {
	row := db.QueryRow(context.Background(), "INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id", login, hash)
	var userId int
	err := row.Scan(&userId)
	if err != nil {
		return 0, err
	}
	return userId, nil
}

func (db Store) GetUser(login string) (hash []byte, userId int, err error) {
	row := db.QueryRow(context.Background(), "SELECT id, password FROM users WHERE login = $1", login)
	err = row.Scan(&userId, &hash)
	if err != nil {
		return nil, 0, err
	}
	return hash, userId, nil
}
