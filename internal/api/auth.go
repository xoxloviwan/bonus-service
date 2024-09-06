package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/golang-jwt/jwt/v4"
)

type Creds struct {
	User string `json:"login"`
	Pwd  string `json:"password"`
}

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TOKEN_EXP = time.Hour * 3
const SECRET_KEY = "supersecretkey"

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" ||
		r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	creds := Creds{}
	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = json.Unmarshal(buf.Bytes(), &creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if creds.User == "" || creds.Pwd == "" {
		http.Error(w, "empty login or password", http.StatusBadRequest)
		return
	}
	var userId int
	userId, err = newUser(h.store, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	var tkn string
	tkn, err = BuildJWT(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", "Bearer "+tkn)
	w.WriteHeader(http.StatusOK)
}

func CreateUsersTable(ctx context.Context, db Store) error {
	_, err := db.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS users (
			id bigint GENERATED ALWAYS AS IDENTITY,
			login text NOT NULL UNIQUE,
			password text NOT NULL)`)
	return err
}

func newUser(store Store, creds Creds) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Pwd), 0)
	if err != nil {
		return 0, err
	}
	row := store.QueryRow(context.Background(), "INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id", creds.User, hash)
	var userId int
	err = row.Scan(&userId)
	if err != nil {
		return 0, err
	}
	return userId, nil
}

func authUser(store Store, creds Creds) (int, error) {
	var err error
	row := store.QueryRow(context.Background(), "SELECT id, password FROM users WHERE login = $1", creds.User)
	var userId int
	var hash []byte
	err = row.Scan(&userId, &hash)
	if err != nil {
		return 0, err
	}
	err = bcrypt.CompareHashAndPassword(hash, []byte(creds.Pwd))
	if err != nil {
		return 0, errors.New("auth failed")
	}
	return userId, nil
}

// BuildJWT создаёт токен и возвращает его в виде строки.
func BuildJWT(user int) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		// собственное утверждение
		UserID: user,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" ||
		r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	creds := Creds{}
	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = json.Unmarshal(buf.Bytes(), &creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var userId int
	userId, err = authUser(h.store, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var tkn string
	tkn, err = BuildJWT(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", "Bearer "+tkn)
	w.WriteHeader(http.StatusOK)
}
