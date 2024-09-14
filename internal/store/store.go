package store

import (
	"context"
	"time"

	"gophermart/internal/types"

	"github.com/jackc/pgx/v5"
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
			id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			login text NOT NULL UNIQUE,
			password text NOT NULL,
			sum double precision)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx,
		`CREATE INDEX IF NOT EXISTS users_login_idx ON users (login)`)
	return err
}

func (db Store) AddUser(ctx context.Context, login string, hash []byte) (int, error) {
	row := db.QueryRow(ctx, "INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id", login, hash)
	var userID int
	err := row.Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (db Store) GetUser(ctx context.Context, login string) (hash []byte, userID int, err error) {
	row := db.QueryRow(ctx, "SELECT id, password FROM users WHERE login = $1", login)
	err = row.Scan(&userID, &hash)
	if err != nil {
		return nil, 0, err
	}
	return hash, userID, nil
}

func (db Store) CreateOrdersTable(ctx context.Context) error {
	_, err := db.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS orders (
			id bigint NOT NULL PRIMARY KEY,
			status text NOT NULL,
			user_id bigint NOT NULL,
			uploaded_at timestamp with time zone NOT NULL,
			processed_at timestamp with time zone,
			accrual double precision)`) // accrual - сумма начисленных баллов за заказ, получаем из внешней системы
	return err
}

func (db Store) AddOrder(ctx context.Context, orderID int, userID int) error {
	t := time.Now()
	ct, err := db.Exec(ctx,
		`INSERT INTO orders (
			id,
			status,
			user_id,
			uploaded_at
		) VALUES (
			@id,
			@status,
			@user_id,
			@uploaded_at
		) ON CONFLICT (id) DO NOTHING`,
		pgx.NamedArgs{
			"id":          orderID,
			"user_id":     userID,
			"uploaded_at": t,
			"status":      "NEW",
		})
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		row := db.QueryRow(ctx, "SELECT status, user_id, uploaded_at FROM orders WHERE id = @id", pgx.NamedArgs{"id": orderID})
		var uploadedAt time.Time
		var status string
		var userIDFromOrder int
		err = row.Scan(&status, &userIDFromOrder, &uploadedAt)
		if err != nil {
			return err
		}
		if userID == userIDFromOrder {
			return types.ErrOldOrder
		}
		return types.ErrOrderExists
	}
	return nil
}

func (db Store) UpdateOrderInfo(ctx context.Context, orderID int, status string, accrual *int) error {
	_, err := db.Exec(ctx, "UPDATE orders SET status = @status, processed_at = @processed_at, accrual = @accrual WHERE id = @id",
		pgx.NamedArgs{
			"id":           orderID,
			"status":       status,
			"processed_at": time.Now(),
			"accrual":      *accrual,
		})
	return err
}
