package saga

import (
	"sync"

	"go.temporal.io/sdk/workflow"
)

// ParallelCompensator implements the Saga interface
// it executes compensations in parallel
type ParallelCompensator struct {
	ctx             workflow.Context
	compensationOps []*compensationOp
	cancelFunc      workflow.CancelFunc
	once            sync.Once
}

// NewParallelCompensator returns a new instance of a ParallelCompensator
func NewParallelCompensator(ctx workflow.Context) *ParallelCompensator {
	sagaCtx, cancelFunc := workflow.NewDisconnectedContext(ctx)
	return &ParallelCompensator{
		ctx:        sagaCtx,
		cancelFunc: cancelFunc,
	}
}

// AddCompensation adds a rollback compensation step
func (p *ParallelCompensator) AddCompensation(activity interface{}, args ...interface{}) {
	p.compensationOps = append(p.compensationOps, &compensationOp{
		activity: activity,
		args:     args,
	})
}

// Compensate executes all the compensation operations that have been defined in parallel
// After the first call, subsequent calls to a Compensate do nothing.
func (p *ParallelCompensator) Compensate() error {
	var err error

	p.once.Do(func() {
		err = p.compensate()
	})

	return err
}

func (p *ParallelCompensator) compensate() error {
	var (
		futures           []workflow.Future
		compensationError = new(CompensationError)
	)

	for _, op := range p.compensationOps {
		futures = append(futures, workflow.ExecuteActivity(p.ctx, op.activity, op.args...))
	}

	for _, future := range futures {
		err := future.Get(p.ctx, nil)
		if err != nil {
			compensationError.addError(err)
		}
	}
	if compensationError.hasErrors() {
		return compensationError
	}

	return nil
}

// Cancel cancels the Saga context
func (p *ParallelCompensator) Cancel() {
	p.cancelFunc()
}
