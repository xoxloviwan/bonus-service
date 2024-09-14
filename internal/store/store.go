package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	*pgxpool.Pool
}

func NewStore(ctx context.Context, connString string) (*Store, error) {
	dbpool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return &Store{}, err
	}
	return &Store{dbpool}, nil
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
	var userID int
	err := row.Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (db Store) GetUser(login string) (hash []byte, userID int, err error) {
	row := db.QueryRow(context.Background(), "SELECT id, password FROM users WHERE login = $1", login)
	err = row.Scan(&userID, &hash)
	if err != nil {
		return nil, 0, err
	}
	return hash, userID, nil
}
