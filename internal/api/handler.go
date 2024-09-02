package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"crypto/sha256"

	"github.com/golang-jwt/jwt/v4"
)

type Handler struct{}

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
	userId, err = newUser(creds)
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

func newUser(creds Creds) (int, error) {
	h := sha256.New()
	// передаём байты для хеширования
	h.Write([]byte(creds.Pwd))
	// вычисляем хеш
	hash := h.Sum(nil)
	fmt.Println(hash)
	// TODO записать пару логин/хеш в БД
	return 0, nil
}

func authUser(creds Creds) (int, error) {
	// TODO
	return 0, nil
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
	userId, err = authUser(creds)
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
