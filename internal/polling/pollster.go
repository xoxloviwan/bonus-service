package polling

import (
	"context"
	"errors"
	"fmt"
	"gophermart/internal/model"
	"log/slog"
	"sync"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

// Нужно опрашивать внешний сервис Acrual с каким-то интервалом до тех пор, пока он не вернет нужный статус по заказу PROCESSED или INVALID

type Pollster struct {
	incoming    chan int
	orders      []int
	stopCh      chan struct{}
	accrualAddr string
	store       Store
	limiter     *rate.Limiter
}

func NewPollster(accrualAddr string, store Store) *Pollster {
	incoming := make(chan int)
	orders := make([]int, 0)
	stopCh := make(chan struct{})
	limiter := rate.NewLimiter(rate.Inf, 1_000_000)
	return &Pollster{
		incoming,
		orders,
		stopCh,
		accrualAddr,
		store,
		limiter,
	}
}

func (p *Pollster) Push(OrderID int) {
	p.incoming <- OrderID
}

func (p *Pollster) Run(ctx context.Context, polInterval time.Duration) {
	ticker := time.NewTicker(polInterval)
	for {
		select {
		case <-ticker.C:
			tasksCount := len(p.orders)
			slog.Debug(fmt.Sprintf("Pollster ticker. Tasks in background: %d", tasksCount))
			var wg sync.WaitGroup
			wg.Add(tasksCount)
			for _, orderID := range p.orders {
				if err := p.limiter.Wait(ctx); err == nil {
					go p.poll(ctx, orderID, &wg)
				} else {
					wg.Done()
					go p.Push(orderID)
				}
			}
			wg.Wait()
			p.orders = make([]int, 0)
		case <-ctx.Done():
			slog.Info(ctx.Err().Error())
			return
		case <-p.stopCh:
			slog.Info("Pollster stopped")
			return
		case orderID := <-p.incoming:
			p.orders = append(p.orders, orderID)
		default:
			continue
		}
	}
}

func (p *Pollster) Stop() {
	close(p.stopCh)
}

func (p *Pollster) poll(ctx context.Context, orderID int, wg *sync.WaitGroup) {
	defer wg.Done()
	err := polling(ctx, p.store, p.accrualAddr, orderID)
	var emr *errorManyRequests
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) ||
			errors.Is(err, model.ErrOrderInProcess) ||
			errors.Is(err, syscall.ECONNREFUSED) ||
			errors.Is(err, syscall.Errno(10061)) { // golang.org/x/sys/windows WSAECONNREFUSED
			go p.Push(orderID)
		} else if errors.As(err, &emr) {
			slog.Debug(fmt.Sprintf("%s downtime=%v rps=%f", emr.Error(), emr.downtime, emr.rps))

			p.limiter.SetLimit(rate.Limit(emr.rps))
			p.limiter.SetBurst(0)
			slog.Debug(fmt.Sprintf("New limit: %f burst: %d", p.limiter.Limit(), p.limiter.Burst()))

			time.AfterFunc(emr.downtime, func() {
				p.limiter.SetBurst(int(emr.rps))
				slog.Debug(fmt.Sprintf("New burst: %d", p.limiter.Burst()))
			})

			go p.Push(orderID)
		} else {
			slog.Error(fmt.Errorf("polling error: %w", err).Error())
		}
	}
}
