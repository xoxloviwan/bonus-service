package polling

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"gophermart/internal/model"

	"github.com/golang/mock/gomock"
)

type accrualResp = model.AccrualResp

func setupMock(t *testing.T) *MockStore {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	return NewMockStore(ctrl)
}

func TestPolling(t *testing.T) {
	m := setupMock(t)

	accrual := 500.0
	orderID := 7992723465

	tests := []struct {
		name         string
		ac           accrualResp
		acStatusCode int
		wantErr      error
	}{
		{
			name: "polling_accrual_status_code_200_PROCESSED",
			ac: accrualResp{
				Order:   orderID,
				Status:  model.OrderStatusProcessed,
				Accrual: &accrual,
			},
			acStatusCode: http.StatusOK,
		},
		{
			name: "polling_accrual_status_code_204",
			ac: accrualResp{
				Order:  orderID,
				Status: model.OrderStatusNew,
			},
			acStatusCode: http.StatusNoContent,
			wantErr:      model.ErrOrderNotFound,
		},
		{
			name: "polling_accrual_status_code_429",
			ac: accrualResp{
				Order:  orderID,
				Status: model.OrderStatusNew,
			},
			acStatusCode: http.StatusTooManyRequests,
			wantErr:      model.ErrManyRequests,
		},
		{
			name: "polling_accrual_status_code_200_REGISTERED",
			ac: accrualResp{
				Order:  orderID,
				Status: model.OrderStatusRegistered,
			},
			acStatusCode: http.StatusOK,
			wantErr:      model.ErrOrderInProcess,
		},
		{
			name: "polling_accrual_status_code_200_PROCESSING",
			ac: accrualResp{
				Order:  orderID,
				Status: model.OrderStatusProcessing,
			},
			acStatusCode: http.StatusOK,
			wantErr:      model.ErrOrderInProcess,
		},
		{
			name: "polling_accrual_status_code_200_INVALID",
			ac: accrualResp{
				Order:  orderID,
				Status: model.OrderStatusInvalid,
			},
			acStatusCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			m.EXPECT().UpdateOrderInfo(context.Background(), tt.ac).Return(tt.wantErr).Times(1)

			// Start a local HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

				orderStr := strconv.Itoa(tt.ac.Order)
				// Test request parameters
				if req.URL.String() != "/api/orders/"+orderStr {
					t.Errorf("polling() got url = %v, want url %v", req.URL.String(), "/api/orders/"+orderStr)
				}

				if tt.acStatusCode == http.StatusNoContent {
					w.WriteHeader(http.StatusNoContent)
					return
				}

				if tt.acStatusCode == http.StatusTooManyRequests {
					w.Header().Set("Retry-After", "60")
					w.WriteHeader(http.StatusTooManyRequests)
					w.Write([]byte("No more than 100 requests per minute allowed"))
					return
				}

				respData := tt.ac
				respBytes, err := json.Marshal(respData)
				if err != nil {
					t.Errorf("Polling() error = %v", err)
				}
				w.Write(respBytes)
			}))
			// Close the server when test finishes
			defer server.Close()

			err := polling(context.Background(), m, server.URL, tt.ac.Order)
			if err != nil {
				var e *errorManyRequests
				if errors.As(err, &e) {
					t.Logf("%s downtime=%v rps=%f\n", e.Error(), e.downtime, e.rps)
					return
				}
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Polling() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
