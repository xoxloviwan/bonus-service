package api

import (
	"bytes"
	"context"
	"errors"
	"gophermart/internal/mock"
	"gophermart/internal/model"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

type orderCase struct {
	name    string
	method  string
	url     string
	reqBody string
	mockErr error
	want    want
}

func TestHandler_NewOrder(t *testing.T) {
	tests := []orderCase{
		{
			name:    "new_order_status_code_202",
			method:  http.MethodPost,
			reqBody: "7992723465",
			url:     "/api/user/orders",
			want: want{
				statusCode: http.StatusAccepted,
			},
		},
		{
			name:    "new_order_status_code_200",
			method:  http.MethodPost,
			reqBody: "7992723465",
			url:     "/api/user/orders",
			want: want{
				statusCode: http.StatusOK,
			},
			mockErr: model.ErrOldOrder,
		},
		{
			name:    "new_order_status_code_400",
			method:  http.MethodPost,
			reqBody: "",
			url:     "/api/user/orders",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:    "new_order_status_code_422",
			method:  http.MethodPost,
			reqBody: "12121",
			url:     "/api/user/orders",
			want: want{
				statusCode: http.StatusUnprocessableEntity,
			},
		},
		{
			name:    "new_order_status_code_409",
			method:  http.MethodPost,
			reqBody: "7992723465",
			url:     "/api/user/orders",
			want: want{
				statusCode: http.StatusConflict,
			},
			mockErr: model.ErrOrderExists,
		},
		{
			name:    "new_order_status_code_500",
			method:  http.MethodPost,
			reqBody: "7992723465",
			url:     "/api/user/orders",
			want: want{
				statusCode: http.StatusInternalServerError,
			},
			mockErr: errors.New("any unexpected error"),
		},
	}
	userID := 77

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupHandler(t)
			reqBody := bytes.NewBuffer([]byte(tt.reqBody))
			req := httptest.NewRequest(tt.method, tt.url, reqBody)
			req.Header = map[string][]string{
				"Content-Type": {"text/plain"},
			}
			w := httptest.NewRecorder()

			ctx := context.WithValue(req.Context(), userIDCtxKey{}, userID)
			req = req.WithContext(ctx)

			m := h.store.(*mock.MockStore)

			orderID, err := strconv.Atoi(tt.reqBody)
			if err != nil {
				orderID = -1
			}

			m.EXPECT().AddOrder(ctx, orderID, userID).Return(tt.mockErr).Times(1)

			h.poller.(*MockPoller).EXPECT().Push(orderID).Times(1)

			h.NewOrder(w, req)

			result := w.Result()
			defer result.Body.Close()

			if tt.want.statusCode != result.StatusCode {
				t.Errorf("got status %v, want %v", result.StatusCode, tt.want.statusCode)
			}
		})
	}
}
