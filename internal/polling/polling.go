package polling

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"sync/atomic"

	"gophermart/internal/types"

	"github.com/go-resty/resty/v2"
)

var Downtime atomic.Uint64
var RPM atomic.Uint64

type accrualResp struct {
	Order   int    `json:"order,string"`
	Status  string `json:"status"`
	Accrual *int   `json:"accrual,omitempty"`
}

type Store interface {
	UpdateOrderInfo(ctx context.Context, orderID int, status string, accrual *int) error
}

func polling(ctx context.Context, store Store, accrualAddr string, orderID int) error {
	url := fmt.Sprintf("%s/api/orders/%d", accrualAddr, orderID)

	orderInfo := accrualResp{}

	client := resty.New()
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusNoContent {
		return types.ErrOrderNotFound
	}

	if resp.StatusCode() == http.StatusTooManyRequests {
		retryAfter, err := strconv.Atoi(resp.Header().Get("Retry-After"))
		if err != nil {
			return fmt.Errorf("unexpected header Retry-After: %s", resp.Header().Get("Retry-After"))
		}
		Downtime.Store(uint64(retryAfter))
		expr := regexp.MustCompile(`No more than (\d+) requests per minute allowed`)
		matches := expr.FindStringSubmatch(resp.String())
		if len(matches) >= 1 {
			rpm, err := strconv.Atoi(matches[1])
			if err != nil {
				return err
			}
			RPM.Store(uint64(rpm))
		}
		return types.ErrManyRequests
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	if err := json.Unmarshal(resp.Body(), &orderInfo); err != nil {
		return err
	}

	if err := store.UpdateOrderInfo(ctx, orderID, orderInfo.Status, orderInfo.Accrual); err != nil {
		return err
	}
	if orderInfo.Status == "REGISTERED" || orderInfo.Status == "PROCESSING" {
		return types.ErrOrderInProcess
	}
	return nil
}