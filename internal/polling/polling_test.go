package polling

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"gophermart/internal/types"

	"github.com/golang/mock/gomock"
)

func setupMock(t *testing.T) *MockStore {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	return NewMockStore(ctrl)
}

func TestPolling(t *testing.T) {
	m := setupMock(t)

	accrual := 500

	tests := []struct {
		name         string
		ac           accrualResp
		acStatusCode int
		wantErr      error
	}{
		{
			name: "polling_accrual_status_code_200_PROCESSED",
			ac: accrualResp{
				Order:   7992723465,
				Status:  "PROCESSED",
				Accrual: &accrual,
			},
			acStatusCode: http.StatusOK,
		},
		{
			name: "polling_accrual_status_code_204",
			ac: accrualResp{
				Order:  7992723465,
				Status: "NEW",
			},
			acStatusCode: http.StatusNoContent,
			wantErr:      types.ErrOrderNotFound,
		},
		{
			name: "polling_accrual_status_code_429",
			ac: accrualResp{
				Order:  7992723465,
				Status: "NEW",
			},
			acStatusCode: http.StatusTooManyRequests,
			wantErr:      types.ErrManyRequests,
		},
		{
			name: "polling_accrual_status_code_200_REGISTERED",
			ac: accrualResp{
				Order:  7992723465,
				Status: "REGISTERED",
			},
			acStatusCode: http.StatusOK,
			wantErr:      types.ErrOrderInProcess,
		},
		{
			name: "polling_accrual_status_code_200_PROCESSING",
			ac: accrualResp{
				Order:  7992723465,
				Status: "PROCESSING",
			},
			acStatusCode: http.StatusOK,
			wantErr:      types.ErrOrderInProcess,
		},
		{
			name: "polling_accrual_status_code_200_INVALID",
			ac: accrualResp{
				Order:  7992723465,
				Status: "INVALID",
			},
			acStatusCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			m.EXPECT().UpdateOrderInfo(context.Background(), tt.ac.Order, tt.ac.Status, gomock.Any()).Return(tt.wantErr).Times(1)

			// Start a local HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

				orderStr := strconv.Itoa(tt.ac.Order)
				// Test request parameters
				if req.URL.String() != "/api/orders/"+orderStr {
					t.Errorf("Polling() got url = %v, want url %v", req.URL.String(), "/api/orders/"+orderStr)
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

			err := Polling(context.Background(), m, server.URL, tt.ac.Order)
			if err != tt.wantErr {
				t.Errorf("Polling() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
