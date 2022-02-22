package saga

import (
	"sync"

	"go.temporal.io/sdk/workflow"
)

// SerialCompensator implements the Saga interface
// it executes compensations serially, in FI-LO (stack) order
type SerialCompensator struct {
	ctx               workflow.Context
	compensationStack []*compensationOp
	cancelFunc        workflow.CancelFunc
	once              sync.Once
	continueOnError   bool
}

// NewSerialCompensator returns a new instance of a SerialCompensator
//
// if continueOnError is true, it will ignore errors that occur when running Compensate
func NewSerialCompensator(ctx workflow.Context, continueOnError bool) *SerialCompensator {
	sagaCtx, cancelFunc := workflow.NewDisconnectedContext(ctx)
	return &SerialCompensator{
		ctx:             sagaCtx,
		continueOnError: continueOnError,
		cancelFunc:      cancelFunc,
	}
}

// AddCompensation adds a rollback compensation step
func (s *SerialCompensator) AddCompensation(activity interface{}, args ...interface{}) {
	s.compensationStack = append(s.compensationStack, &compensationOp{
		activity: activity,
		args:     args,
	})
}

// Compensate serially executes all the compensation operations that have been defined
// After the first call, subsequent calls to a Compensate do nothing.
func (s *SerialCompensator) Compensate() error {
	var err error

	s.once.Do(func() {
		err = s.compensate()
	})

	return err
}

func (s *SerialCompensator) compensate() error {
	compensationError := new(CompensationError)

	// iterate through compensations in reverse order and execute them
	for i := len(s.compensationStack) - 1; i >= 0; i-- {
		op := s.compensationStack[i]

		future := workflow.ExecuteActivity(s.ctx, op.activity, op.args...)
		if err := future.Get(s.ctx, nil); err != nil {
			// if ContinueOnError is not set, return first err that occurs
			// otherwise, append err to compensation err
			if !s.continueOnError {
				return err
			}

			compensationError.addError(err)
		}
	}

	if compensationError.hasErrors() {
		return compensationError
	}

	return nil
}

// Cancel cancels the saga execution
func (s *SerialCompensator) Cancel() {
	s.cancelFunc()
}
