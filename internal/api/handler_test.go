package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"gophermart/internal/mock"
	"gophermart/internal/model"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type orderCase struct {
	name       string
	reqBody    string
	mockErr    error
	want       want
	mockOrders []Order
}

func TestHandler_NewOrder(t *testing.T) {
	tests := []orderCase{
		{
			name:    "new_order_status_code_202",
			reqBody: "7992723465",
			want: want{
				statusCode: http.StatusAccepted,
			},
		},
		{
			name:    "new_order_status_code_200",
			reqBody: "7992723465",
			want: want{
				statusCode: http.StatusOK,
			},
			mockErr: model.ErrOldOrder,
		},
		{
			name:    "new_order_with_invalid_empty_body_should_repond_status_code_400",
			reqBody: "",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:    "new_order_status_code_422",
			reqBody: "12121",
			want: want{
				statusCode: http.StatusUnprocessableEntity,
			},
		},
		{
			name:    "new_order_status_code_409",
			reqBody: "7992723465",
			want: want{
				statusCode: http.StatusConflict,
			},
			mockErr: model.ErrOrderExists,
		},
		{
			name:    "new_order_status_code_500",
			reqBody: "7992723465",
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
			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", reqBody)
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

			m.EXPECT().AddOrder(ctx, orderID, userID).Return(model.OrderStatusNew, tt.mockErr).Times(1)

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

func TestHandler_OrderList(t *testing.T) {
	accrual := 500.0
	tests := []orderCase{
		{
			name: "order_list_status_code_200",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body: `[
					{
							"number": "9278923470",
							"status": "PROCESSED",
							"accrual": 500,
							"uploaded_at": "2020-12-10T15:15:45+03:00"
					},
					{
							"number": "12345678903",
							"status": "PROCESSING",
							"uploaded_at": "2020-12-10T15:12:01+03:00"
					},
					{
							"number": "346436439",
							"status": "INVALID",
							"uploaded_at": "2020-12-09T16:09:53+03:00"
					}
    		]`,
			},
			mockOrders: []Order{
				{
					ID:         9278923470,
					Status:     model.OrderStatusProcessed,
					Accrual:    &accrual,
					UploadedAt: time.Date(2020, 12, 10, 15, 15, 45, 0, time.FixedZone("UTC+3", 3*60*60)),
				}, {
					ID:         12345678903,
					Status:     model.OrderStatusProcessing,
					UploadedAt: time.Date(2020, 12, 10, 15, 12, 1, 0, time.FixedZone("UTC+3", 3*60*60)),
				}, {
					ID:         346436439,
					Status:     model.OrderStatusInvalid,
					UploadedAt: time.Date(2020, 12, 9, 16, 9, 53, 0, time.FixedZone("UTC+3", 3*60*60)),
				}},
		},
		{
			name: "order_list_no_orders_status_code_204",
			want: want{
				statusCode: http.StatusNoContent,
			},
			mockOrders: []Order{},
		},
		{
			name: "order_list_status_code_500",
			want: want{
				statusCode:  http.StatusInternalServerError,
				contentType: "text/plain; charset=utf-8",
			},
			mockErr: errors.New("any unexpected error"),
		},
	}

	userID := 77

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupHandler(t)

			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			ctx := context.WithValue(req.Context(), userIDCtxKey{}, userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			m := h.store.(*mock.MockStore)
			m.EXPECT().ListOrders(ctx, userID).Return(tt.mockOrders, tt.mockErr).Times(1)

			h.OrderList(w, req)

			result := w.Result()
			defer result.Body.Close()

			if tt.want.statusCode != result.StatusCode {
				t.Errorf("got status %v, want %v", result.StatusCode, tt.want.statusCode)
			}

			if result.StatusCode == http.StatusNoContent {
				return
			}

			resBody, err := io.ReadAll(result.Body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.want.contentType != result.Header.Get("Content-Type") {
				t.Errorf("got content type %v, want %v", result.Header.Get("Content-Type"), tt.want.contentType)
			}

			if result.StatusCode == http.StatusInternalServerError {
				return
			}

			var gotBody []Order
			var wantBody []Order

			err = json.Unmarshal(resBody, &gotBody)
			if err != nil {
				t.Fatal(err)
			}
			err = json.Unmarshal([]byte(tt.want.body), &wantBody)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantBody, gotBody); diff != "" {
				t.Errorf("Body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHandler_Balance(t *testing.T) {
	tests := []struct {
		name        string
		want        want
		mockErr     error
		mockBalance *Balance
	}{
		{
			name: "success",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"current":500.5,"withdrawn":42}`,
			},
			mockBalance: &Balance{
				Sum:      500.5,
				WriteOff: 42,
			},
		},
		{
			name: "internal_server_error",
			want: want{
				statusCode: http.StatusInternalServerError,
			},
			mockErr: errors.New("internal server error"),
		},
	}

	userID := 77

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := setupHandler(t)

			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			ctx := context.WithValue(req.Context(), userIDCtxKey{}, userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			m := h.store.(*mock.MockStore)

			m.EXPECT().GetBalance(ctx, userID).Return(tt.mockBalance, tt.mockErr).Times(1)
			h.Balance(w, req)

			result := w.Result()
			defer result.Body.Close()

			if tt.want.statusCode != result.StatusCode {
				t.Errorf("got status %v, want %v", result.StatusCode, tt.want.statusCode)
			}

			if result.StatusCode == http.StatusInternalServerError {
				return
			}

			resBody, err := io.ReadAll(result.Body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.want.contentType != result.Header.Get("Content-Type") {
				t.Errorf("got content type %v, want %v", result.Header.Get("Content-Type"), tt.want.contentType)
			}

			var gotBody *Balance
			var wantBody *Balance

			err = json.Unmarshal(resBody, &gotBody)
			if err != nil {
				t.Fatal(err)
			}
			err = json.Unmarshal([]byte(tt.want.body), &wantBody)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantBody, gotBody); diff != "" {
				t.Errorf("Body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHandler_PaymentList(t *testing.T) {

	tests := []struct {
		name         string
		want         want
		mockErr      error
		mockPayments []PaymentFact
	}{
		{
			name: "success",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body: `[
								{
									"order": "2377225624",
									"sum": 500,
									"processed_at": "2020-12-09T16:09:57+03:00"
								}
							]`,
			},
			mockPayments: []PaymentFact{
				{
					Payment: Payment{
						OrderID: 2377225624,
						Sum:     500,
					},
					ProcessedAt: time.Date(2020, 12, 9, 16, 9, 57, 0, time.FixedZone("UTC+3", 3*60*60)),
				},
			},
		},
		{
			name: "internal_server_error",
			want: want{
				statusCode: http.StatusInternalServerError,
			},
			mockErr: errors.New("internal server error"),
		},
		{
			name: "no_content",
			want: want{
				statusCode: http.StatusNoContent,
			},
			mockPayments: []PaymentFact{},
		},
	}
	userID := 77

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := setupHandler(t)

			req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
			ctx := context.WithValue(req.Context(), userIDCtxKey{}, userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			m := h.store.(*mock.MockStore)

			m.EXPECT().SpentBonusList(ctx, userID).Return(tt.mockPayments, tt.mockErr).Times(1)
			h.PaymentList(w, req)

			result := w.Result()
			defer result.Body.Close()

			if tt.want.statusCode != result.StatusCode {
				t.Errorf("got status %v, want %v", result.StatusCode, tt.want.statusCode)
			}

			if result.StatusCode == http.StatusInternalServerError || result.StatusCode == http.StatusNoContent {
				return
			}

			resBody, err := io.ReadAll(result.Body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.want.contentType != result.Header.Get("Content-Type") {
				t.Errorf("got content type %v, want %v", result.Header.Get("Content-Type"), tt.want.contentType)
			}

			var gotBody []PaymentFact
			var wantBody []PaymentFact

			err = json.Unmarshal(resBody, &gotBody)
			if err != nil {
				t.Fatal(err)
			}
			err = json.Unmarshal([]byte(tt.want.body), &wantBody)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantBody, gotBody); diff != "" {
				t.Errorf("Body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHandler_Pay(t *testing.T) {
	tests := []struct {
		name        string
		want        want
		reqBody     string
		mockErr     error
		mockPayment Payment
	}{
		{
			name: "success",
			want: want{
				statusCode: http.StatusOK,
			},
			reqBody: `{
				"order": "2377225624",
				"sum": 751
			}`,
			mockPayment: Payment{
				OrderID: 2377225624,
				Sum:     751,
			},
		},
		{
			name: "internal_server_error",
			want: want{
				statusCode: http.StatusInternalServerError,
			},
			mockErr: errors.New("internal server error"),
		},
		{
			name: "not_enough_funds",
			reqBody: `{
				"order": "2377225624",
				"sum": 751
			}`,
			mockPayment: Payment{
				OrderID: 2377225624,
				Sum:     751,
			},
			want: want{
				statusCode: http.StatusPaymentRequired,
			},
			mockErr: model.ErrNotEnough,
		},
		{
			name: "wrong_order_id",
			reqBody: `{
				"order": "11111",
				"sum": 751
			}`,
			mockPayment: Payment{
				OrderID: 11111,
				Sum:     751,
			},
			want: want{
				statusCode: http.StatusUnprocessableEntity,
			},
		},
	}
	userID := 77

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := setupHandler(t)

			reqBody := bytes.NewBuffer([]byte(tt.reqBody))
			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", reqBody)
			ctx := context.WithValue(req.Context(), userIDCtxKey{}, userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			m := h.store.(*mock.MockStore)

			m.EXPECT().SpendBonus(ctx, userID, tt.mockPayment).Return(tt.mockErr).Times(1)
			h.Pay(w, req)

			result := w.Result()
			defer result.Body.Close()

			if tt.want.statusCode != result.StatusCode {
				t.Errorf("got status %v, want %v", result.StatusCode, tt.want.statusCode)
			}
		})
	}
}
