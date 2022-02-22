package saga

// This file is just a test workflow that is used in saga_test.go
// No tests are actually run in this file

import (
	"context"
	"sync"
	"time"

	"go.temporal.io/sdk/workflow"
)

type testResults struct {
	// used for basic workflow
	Count             int
	CompensationOrder []int
}

type testWorkflow struct {
	Count             int
	CompensationOrder []int
	mu                sync.Mutex
}

type testSagaOptions struct {
	ParallelCompensation bool
	ContinueOnError      bool
}

func (w *testWorkflow) basicSagaWorkflow(ctx workflow.Context, sagaOpts testSagaOptions) (testResults, error) {
	// logger := temporallog.WorkflowLogger(ctx)
	var emptyResults testResults

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var saga Saga
	if sagaOpts.ParallelCompensation {
		saga = NewParallelCompensator(ctx)
	} else {
		saga = NewSerialCompensator(ctx, sagaOpts.ContinueOnError)
	}

	err := workflow.SetQueryHandler(ctx, "count", func(input []byte) (int, error) {
		return w.Count, nil
	})
	if err != nil {
		return emptyResults, err
	}

	err = workflow.SetQueryHandler(ctx, "compensationOrder", func(input []byte) ([]int, error) {
		return w.CompensationOrder, nil
	})
	if err != nil {
		return emptyResults, err
	}

	err = workflow.ExecuteActivity(ctx, w.add, 10).Get(ctx, nil)
	if err != nil {
		// logger.Err(err).Error("activity failed")
		handleSagaErr(ctx, saga.Compensate())
		return emptyResults, err
	}
	saga.AddCompensation(w.minus, 5)

	err = workflow.ExecuteActivity(ctx, w.add, 20).Get(ctx, nil)
	if err != nil {
		// logger.Err(err).Error("activity failed")
		handleSagaErr(ctx, saga.Compensate())
		return emptyResults, err
	}
	saga.AddCompensation(w.minus, 10)

	err = workflow.ExecuteActivity(ctx, w.add, 30).Get(ctx, nil)
	if err != nil {
		// logger.Err(err).Error("activity failed")
		handleSagaErr(ctx, saga.Compensate())
		return emptyResults, err
	}

	return testResults{Count: w.Count, CompensationOrder: w.CompensationOrder}, nil
}

func (w *testWorkflow) add(ctx context.Context, n int) error {
	w.Count += n
	return nil
}

func (w *testWorkflow) minus(ctx context.Context, n int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Count -= n
	w.CompensationOrder = append(w.CompensationOrder, n)
	return nil
}

func handleSagaErr(ctx workflow.Context, err error) {
	// logger := temporallog.WorkflowLogger(ctx)
	// if err != nil {
	// 	logger.Err(err).Warn("Error(s) in saga compensation")
	// }
}
