package api

import (
	"bytes"
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

type commonAuth func(creds Creds) (userId int, httpCode int, err error)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	AuthCmnHandler(w, r,
		func(creds Creds) (userId int, httpCode int, err error) {
			userId, err = newUser(h.store, creds)
			httpCode = http.StatusConflict
			return userId, httpCode, err
		},
	)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	AuthCmnHandler(w, r,
		func(creds Creds) (userId int, httpCode int, err error) {
			userId, err = authUser(h.store, creds)
			httpCode = http.StatusUnauthorized
			return userId, httpCode, err
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
	var userId, httpErrCode int
	userId, httpErrCode, err = auth(creds)
	if err != nil {
		http.Error(w, err.Error(), httpErrCode)
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

func newUser(store Store, creds Creds) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Pwd), 0)
	if err != nil {
		return 0, err
	}
	return store.AddUser(creds.User, hash)
}

func authUser(store Store, creds Creds) (int, error) {
	hash, userId, err := store.GetUser(creds.User)
	if err != nil {
		return 0, errors.New("auth failed")
	}
	err = bcrypt.CompareHashAndPassword(hash, []byte(creds.Pwd))
	if err != nil {
		return 0, errors.New("auth failed")
	}
	return userId, nil
}

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TOKEN_EXP = time.Hour * 3
const SECRET_KEY = "supersecretkey"

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
