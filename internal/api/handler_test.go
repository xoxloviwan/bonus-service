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
	mockOrders []model.Order
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

			m.EXPECT().AddOrder(ctx, orderID, userID).Return("NEW", tt.mockErr).Times(1)

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
			mockOrders: []model.Order{
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
			mockOrders: []model.Order{},
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

			req, err := http.NewRequest("GET", "/api/user/orders", nil)
			if err != nil {
				t.Fatal(err)
			}
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

			var gotBody []model.Order
			var wantBody []model.Order

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
