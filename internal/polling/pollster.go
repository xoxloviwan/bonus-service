package polling

type OrderTask struct {
	OrderID int
}
type Status struct {
	OrderTask
	Err error
}

type Pollster struct {
	tasks       chan OrderTask
	results     chan Status
	accrualAddr string
	store       Store
}

func NewPollster(accrualAddr string, store Store) *Pollster {
	tasks := make(chan OrderTask)
	results := make(chan Status)
	return &Pollster{
		tasks,
		results,
		accrualAddr,
		store,
	}
}

func (p *Pollster) Push(OrderID int) {
	p.tasks <- OrderTask{OrderID}
}

func (p *Pollster) Poll() {

}
