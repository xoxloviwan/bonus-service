package polling

import (
	"context"
	"errors"
	"fmt"
	"gophermart/internal/api"
	"gophermart/internal/types"
	"sync"
	"time"
)

// Нужно опрашивать внешний сервис Acrual с каким-то интервалом до тех пор, пока он не вернет нужный статус по заказу PROCESSED или INVALID

type Pollster struct {
	incoming    chan int
	orders      []int
	stopCh      chan struct{}
	accrualAddr string
	store       Store
}

func NewPollster(accrualAddr string, store Store) *Pollster {
	incoming := make(chan int)
	orders := make([]int, 0)
	stopCh := make(chan struct{})
	return &Pollster{
		incoming,
		orders,
		stopCh,
		accrualAddr,
		store,
	}
}

func (p *Pollster) Push(OrderID int) {
	go func() {
		p.incoming <- OrderID
	}()
}

func (p *Pollster) Run() {
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			taskNumber := len(p.orders)
			api.Log.Info(fmt.Sprintf("Pollster ticker. Tasks in background: %d", taskNumber))
			var wg sync.WaitGroup
			wg.Add(taskNumber)
			for _, orderID := range p.orders {
				go p.poll(orderID, &wg)
			}
			wg.Wait()
			p.orders = make([]int, 0)
		case <-p.stopCh:
			api.Log.Info("Pollster stopped")
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

func (p *Pollster) poll(orderID int, wg *sync.WaitGroup) {
	defer wg.Done()
	err := polling(context.TODO(), p.store, p.accrualAddr, orderID)
	if err != nil {
		if errors.Is(err, types.ErrOrderNotFound) || errors.Is(err, types.ErrOrderInProcess) {
			p.Push(orderID)
		}
	}
}
