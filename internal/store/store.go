package store

import (
	"context"
	"time"

	"gophermart/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	*pgxpool.Pool
}

type User = model.User
type Order = model.Order
type OrderStatus = model.OrderStatus
type Balance = model.Balance

func NewStore(ctx context.Context, connString string) (*Store, error) {
	dbpool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return &Store{}, err
	}
	st := Store{dbpool}
	err = st.CreateUsersTable(ctx)
	if err != nil {
		return &Store{}, err
	}
	err = st.CreateOrdersTable(ctx)
	if err != nil {
		return &Store{}, err
	}

	return &st, nil
}

func (db *Store) CreateUsersTable(ctx context.Context) error {
	_, err := db.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS users (
			id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			login text NOT NULL UNIQUE,
			password text NOT NULL,
			sum double precision,
			writeoff double precision)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx,
		`CREATE INDEX IF NOT EXISTS users_login_idx ON users (login)`)
	return err
}

func (db *Store) AddUser(ctx context.Context, u User) (int, error) {
	row := db.QueryRow(ctx, "INSERT INTO users (login, password, sum, writeoff) VALUES ($1, $2, $3, $4) RETURNING id", u.Login, u.Hash, 0, 0)
	err := row.Scan(&u.ID)
	if err != nil {
		return 0, err
	}
	return u.ID, nil
}

func (db *Store) GetUser(ctx context.Context, login string) (*User, error) {
	u := &User{Login: login}
	row := db.QueryRow(ctx, "SELECT id, password FROM users WHERE login = $1", u.Login)
	err := row.Scan(&u.ID, &u.Hash)
	if err != nil {
		return u, err
	}
	return u, nil
}

func (db *Store) CreateOrdersTable(ctx context.Context) error {
	_, err := db.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS orders (
			id bigint NOT NULL PRIMARY KEY,
			status integer NOT NULL,
			user_id bigint NOT NULL,
			uploaded_at timestamp with time zone NOT NULL,
			processed_at timestamp with time zone,
			accrual double precision)`) // accrual - сумма начисленных баллов за заказ, получаем из внешней системы
	return err
}

func (db *Store) CreateWriteOffTable(ctx context.Context) error {
	_, err := db.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS writeoff (
			id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id bigint NOT NULL,
			order_id bigint NOT NULL,
			processed_at timestamp with time zone,			
			sum double precision)`)
	return err
}

func (db *Store) AddOrder(ctx context.Context, orderID int, userID int) (model.OrderStatus, error) {
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
			"status":      model.OrderStatusNew,
		})
	if err != nil {
		return -1, err
	}
	if ct.RowsAffected() == 0 {
		row := db.QueryRow(ctx, "SELECT status, user_id, uploaded_at FROM orders WHERE id = @id", pgx.NamedArgs{"id": orderID})
		var uploadedAt time.Time
		var status OrderStatus
		var userIDFromOrder int
		err = row.Scan(&status, &userIDFromOrder, &uploadedAt)
		if err != nil {
			return -1, err
		}
		if userID == userIDFromOrder {
			return status, model.ErrOldOrder
		}
		return -1, model.ErrOrderExists
	}
	return model.OrderStatusNew, nil
}

func (db *Store) UpdateOrderInfo(ctx context.Context, info model.AccrualResp) error {
	u := &User{}
	row := db.QueryRow(ctx, "UPDATE orders SET status = @status, processed_at = @processed_at, accrual = @accrual WHERE id = @id RETURNING user_id",
		pgx.NamedArgs{
			"id":           info.Order,
			"status":       info.Status,
			"processed_at": time.Now(),
			"accrual":      info.Accrual,
		})
	err := row.Scan(&u.ID)
	if err != nil {
		return err
	}
	if info.Status == model.OrderStatusProcessed {
		_, err = db.Exec(ctx, "UPDATE users SET sum = sum + @sum WHERE id = @id", pgx.NamedArgs{"sum": info.Accrual, "id": u.ID})
	}
	return err
}

func (db *Store) ListOrders(ctx context.Context, userID int) ([]Order, error) {
	orders := []Order{}
	rows, err := db.Query(ctx, `SELECT
				id,
				status,
				uploaded_at,
				accrual
			FROM orders
			WHERE user_id = @user_id ORDER BY uploaded_at DESC`,
		pgx.NamedArgs{"user_id": userID})
	if err != nil {
		return orders, err
	}
	defer rows.Close()

	for rows.Next() {
		order := Order{}
		err = rows.Scan(&order.ID, &order.Status, &order.UploadedAt, &order.Accrual)
		if err != nil {
			return orders, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (db *Store) GetBalance(ctx context.Context, userID int) (*Balance, error) {
	balance := Balance{}
	row := db.QueryRow(ctx, "SELECT sum, writeoff FROM users WHERE id = @id", pgx.NamedArgs{"id": userID})
	err := row.Scan(&balance.Sum, &balance.WriteOff)
	if err != nil {
		return &balance, err
	}
	return &balance, nil
}
