package polling

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"gophermart/internal/model"

	"github.com/go-resty/resty/v2"
)

type errorManyRequests struct {
	downtime time.Duration
	rps      float64
	error
}

func newErrorManyRequests(downtime time.Duration, rps float64) *errorManyRequests {
	return &errorManyRequests{downtime, rps, model.ErrManyRequests}
}

//go:generate mockgen -destination ./store_mock.go -package polling gophermart/internal/polling Store
type Store interface {
	UpdateOrderInfo(ctx context.Context, orderInfo model.AccrualResp) error
}

func polling(ctx context.Context, store Store, accrualAddr string, orderID int) error {
	url := fmt.Sprintf("%s/api/orders/%d", accrualAddr, orderID)

	orderInfo := model.AccrualResp{}

	client := resty.New()

	slog.Debug("Polling", slog.String("url", url))

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusNoContent {
		return model.ErrOrderNotFound
	}

	if resp.StatusCode() == http.StatusTooManyRequests {
		retryAfter, err := strconv.Atoi(resp.Header().Get("Retry-After"))
		if err != nil {
			return fmt.Errorf("unexpected header Retry-After: %s", resp.Header().Get("Retry-After"))
		}
		expr := regexp.MustCompile(`No more than (\d+) requests per minute allowed`)
		matches := expr.FindStringSubmatch(resp.String())
		var rps float64 = 0
		if len(matches) >= 1 {
			rpm, err := strconv.Atoi(matches[1])
			if err != nil {
				return err
			}
			rps = float64(rpm) / 60
		}
		downtime := time.Duration(retryAfter) * time.Second
		return newErrorManyRequests(downtime, rps)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	if err := json.Unmarshal(resp.Body(), &orderInfo); err != nil {
		return err
	}

	if err := store.UpdateOrderInfo(ctx, orderInfo); err != nil {
		return err
	}
	if orderInfo.Status == model.OrderStatusRegistered || orderInfo.Status == model.OrderStatusProcessing {
		return model.ErrOrderInProcess
	}
	return nil
}
