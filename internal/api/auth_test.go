package api

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"gophermart/internal/mock"

	"net/http/httptest"

	gomock "github.com/golang/mock/gomock"
)

func setupHandler(t *testing.T) *Handler {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)

	return &Handler{store: mockStore}
}

type want struct {
	contentType string
	statusCode  int
}

type authTescases []struct {
	name       string
	method     string
	url        string
	reqBody    string
	mockuserID int
	mockHash   []byte
	mockErr    error
	want       want
}

func TestHandler_Register(t *testing.T) {
	tests := authTescases{
		{
			name:   "register_status_code_200",
			url:    "/api/user/register",
			method: http.MethodPost,
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
			},
			reqBody: `{"login": "user", "password": "123456"}`,
		},
		{
			name:   "register_status_code_400",
			url:    "/api/user/register",
			method: http.MethodPost,
			want: want{
				statusCode: http.StatusBadRequest,
			},
			reqBody: `{"login": "user", "password": "123456"`,
		},
		{
			name:   "register_status_code_400",
			url:    "/api/user/register",
			method: http.MethodPost,
			want: want{
				statusCode: http.StatusBadRequest,
			},
			reqBody: `{"login": "user", "password": ""}`,
		},
		{
			name:   "register_status_code_400",
			url:    "/api/user/register",
			method: http.MethodPost,
			want: want{
				statusCode: http.StatusBadRequest,
			},
			reqBody: `{"login": "", "password": "123456"}`,
		},
		{
			name:   "register_status_code_409",
			url:    "/api/user/register",
			method: http.MethodPost,
			want: want{
				statusCode: http.StatusConflict,
			},
			mockuserID: 0,
			mockErr:    errors.New("failed to add user"),
			reqBody:    `{"login": "user", "password": "123456"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupHandler(t)
			reqBody := bytes.NewBuffer([]byte(tt.reqBody))
			req := httptest.NewRequest(tt.method, tt.url, reqBody)
			req.Header = map[string][]string{
				"Content-Type": {"application/json"},
			}
			w := httptest.NewRecorder()

			m := h.store.(*mock.MockStore)

			m.EXPECT().AddUser(gomock.Any(), gomock.Any()).Return(tt.mockuserID, tt.mockErr).Times(1)

			h.Register(w, req)

			result := w.Result()
			defer result.Body.Close()

			if tt.want.statusCode != result.StatusCode {
				t.Errorf("got status %v, want %v", result.StatusCode, tt.want.statusCode)
			}
		})
	}
}

func TestHandler_Login(t *testing.T) {
	tests := authTescases{
		{
			name:   "login_status_code_200",
			url:    "/api/user/login",
			method: http.MethodPost,
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
			},
			mockHash: []byte("$2a$10$35jb2VUM8yhqH/NtLh.r7ujcLFJScQmu6XwRcTEuSENFbFxFn6eL2"),
			reqBody: `{
					"login": "user",
					"password": "123456"
				}`,
		},
		{
			name:   "login_status_code_400",
			url:    "/api/user/login",
			method: http.MethodPost,
			want: want{
				statusCode: http.StatusBadRequest,
			},
			reqBody: `{"login": "user", "password": "123456"`,
		},
		{
			name:   "login_status_code_400",
			url:    "/api/user/login",
			method: http.MethodPost,
			want: want{
				statusCode: http.StatusBadRequest,
			},
			reqBody: `{"login": "user", "password": ""}`,
		},
		{
			name:   "login_status_code_400",
			url:    "/api/user/login",
			method: http.MethodPost,
			want: want{
				statusCode: http.StatusBadRequest,
			},
			reqBody: `{"login": "", "password": "123456"}`,
		},
		{
			name:   "login_status_code_401",
			url:    "/api/user/login",
			method: http.MethodPost,
			want: want{
				statusCode: http.StatusUnauthorized,
			},
			mockuserID: 0,
			reqBody:    `{"login": "user", "password": "123456"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupHandler(t)
			reqBody := bytes.NewBuffer([]byte(tt.reqBody))
			req := httptest.NewRequest(tt.method, tt.url, reqBody)
			req.Header = map[string][]string{
				"Content-Type": {"application/json"},
			}
			w := httptest.NewRecorder()

			m := h.store.(*mock.MockStore)

			m.EXPECT().GetUser(gomock.Any()).Return(tt.mockHash, tt.mockuserID, tt.mockErr).Times(1)

			h.Login(w, req)

			result := w.Result()
			defer result.Body.Close()

			if tt.want.statusCode != result.StatusCode {
				t.Errorf("got status %v, want %v", result.StatusCode, tt.want.statusCode)
			}
		})
	}
}
