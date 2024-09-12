package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/golang-jwt/jwt/v4"
)

type Creds struct {
	User string `json:"login"`
	Pwd  string `json:"password"`
}

type commonAuth func(creds Creds) (userID int, httpCode int, err error)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	AuthCmnHandler(w, r,
		func(creds Creds) (userID int, httpCode int, err error) {
			userID, err = newUser(r.Context(), h.store, creds)
			httpCode = http.StatusConflict
			return userID, httpCode, err
		},
	)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	AuthCmnHandler(w, r,
		func(creds Creds) (userID int, httpCode int, err error) {
			userID, err = authUser(r.Context(), h.store, creds)
			httpCode = http.StatusUnauthorized
			return userID, httpCode, err
		},
	)
}

func AuthCmnHandler(w http.ResponseWriter, r *http.Request, auth commonAuth) {
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
	var userID, httpErrCode int
	userID, httpErrCode, err = auth(creds)
	if err != nil {
		http.Error(w, err.Error(), httpErrCode)
		return
	}

	var tkn string
	tkn, err = BuildJWT(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", "Bearer "+tkn)
	w.WriteHeader(http.StatusOK)
}

func newUser(ctx context.Context, store Store, creds Creds) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Pwd), 0)
	if err != nil {
		return 0, err
	}
	return store.AddUser(ctx, creds.User, hash)
}

func authUser(ctx context.Context, store Store, creds Creds) (int, error) {
	hash, userID, err := store.GetUser(ctx, creds.User)
	if err != nil {
		return 0, errors.New("auth failed")
	}
	err = bcrypt.CompareHashAndPassword(hash, []byte(creds.Pwd))
	if err != nil {
		return 0, errors.New("auth failed")
	}
	return userID, nil
}

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское userID
type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TokenExp = time.Hour * 24
const SecretKey = "supersecretkey"

// BuildJWT создаёт токен и возвращает его в виде строки.
func BuildJWT(user int) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		// собственное утверждение
		UserID: user,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

func GetUserID(tokenString string) (int, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SecretKey), nil
		})
	if err != nil {
		return -1, err
	}

	if !token.Valid {
		return -1, errors.New("invalid token")
	}
	return claims.UserID, nil
}
