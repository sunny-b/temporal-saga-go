package saga

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func (s *UnitTestSuite) Test_BasicSagaWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()
	w := &testWorkflow{
		Count:             0,
		CompensationOrder: []int{},
	}
	env.RegisterActivity(w.add)
	env.RegisterActivity(w.minus)

	env.ExecuteWorkflow(w.basicSagaWorkflow, testSagaOptions{
		ParallelCompensation: false,
		ContinueOnError:      false,
	})

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var results testResults
	s.NoError(env.GetWorkflowResult(&results))
	s.Equal(60, results.Count)
	s.Equal(0, len(results.CompensationOrder))

	env.AssertExpectations(s.T())
}

func (s *UnitTestSuite) Test_BasicSagaWorkflow_Failure() {
	env := s.NewTestWorkflowEnvironment()
	w := &testWorkflow{
		Count:             0,
		CompensationOrder: []int{},
	}
	env.RegisterActivity(w.add)
	env.RegisterActivity(w.minus)

	env.OnActivity(w.add, mock.Anything, 10).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 20).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 30).Return(errors.New("Test Compensation"))

	env.ExecuteWorkflow(w.basicSagaWorkflow, testSagaOptions{
		ParallelCompensation: false,
		ContinueOnError:      false,
	})

	result, err := env.QueryWorkflow("count")
	s.NoError(err)
	var count int
	err = result.Get(&count)
	s.NoError(err)
	s.Equal(15, count)

	result, err = env.QueryWorkflow("compensationOrder")
	s.NoError(err)
	var compensationOrder []int
	err = result.Get(&compensationOrder)
	s.NoError(err)
	s.Equal([]int{10, 5}, compensationOrder)

	s.True(env.IsWorkflowCompleted())
	s.EqualError(env.GetWorkflowError(), "workflow execution error (type: basicSagaWorkflow, workflowID: default-test-workflow-id, runID: default-test-run-id): activity error (type: add, scheduledEventID: 0, startedEventID: 0, identity: ): Test Compensation")

	env.AssertExpectations(s.T())
}

func (s *UnitTestSuite) Test_BasicSagaWorkflow_Cancelled() {
	env := s.NewTestWorkflowEnvironment()
	w := &testWorkflow{
		Count:             0,
		CompensationOrder: []int{},
	}
	env.RegisterActivity(w.add)
	env.RegisterActivity(w.minus)

	env.OnActivity(w.add, mock.Anything, 10).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 20).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 30).Return(func(ctx context.Context, n int) error {
		env.CancelWorkflow()
		return nil
	})

	env.ExecuteWorkflow(w.basicSagaWorkflow, testSagaOptions{
		ParallelCompensation: false,
		ContinueOnError:      false,
	})

	s.True(env.IsWorkflowCompleted())
	s.EqualError(env.GetWorkflowError(), "workflow execution error (type: basicSagaWorkflow, workflowID: default-test-workflow-id, runID: default-test-run-id): canceled")

	result, err := env.QueryWorkflow("count")
	s.NoError(err)
	var count int
	err = result.Get(&count)
	s.NoError(err)
	s.Equal(15, count)

	result, err = env.QueryWorkflow("compensationOrder")
	s.NoError(err)
	var compensationOrder []int
	err = result.Get(&compensationOrder)
	s.NoError(err)
	s.Equal([]int{10, 5}, compensationOrder)

	env.AssertExpectations(s.T())
}

func (s *UnitTestSuite) Test_BasicSagaWorkflow_ContinueOnError() {
	env := s.NewTestWorkflowEnvironment()
	w := &testWorkflow{
		Count:             0,
		CompensationOrder: []int{},
	}
	env.RegisterActivity(w.add)
	env.RegisterActivity(w.minus)

	env.OnActivity(w.add, mock.Anything, 10).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 20).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 30).Return(errors.New("Test Compensation"))

	env.OnActivity(w.minus, mock.Anything, 10).Return(errors.New("Test ContinueOnError"))
	env.OnActivity(w.minus, mock.Anything, 5).Return(w.minus)

	env.ExecuteWorkflow(w.basicSagaWorkflow, testSagaOptions{
		ParallelCompensation: false,
		ContinueOnError:      true,
	})

	s.True(env.IsWorkflowCompleted())
	s.EqualError(env.GetWorkflowError(), "workflow execution error (type: basicSagaWorkflow, workflowID: default-test-workflow-id, runID: default-test-run-id): activity error (type: add, scheduledEventID: 0, startedEventID: 0, identity: ): Test Compensation")

	result, err := env.QueryWorkflow("count")
	s.NoError(err)
	var count int
	err = result.Get(&count)
	s.NoError(err)
	s.Equal(25, count)

	result, err = env.QueryWorkflow("compensationOrder")
	s.NoError(err)
	var compensationOrder []int
	err = result.Get(&compensationOrder)
	s.NoError(err)
	s.Equal([]int{5}, compensationOrder)

	env.AssertExpectations(s.T())
}

func (s *UnitTestSuite) Test_BasicSagaWorkflow_CompensationFail() {
	env := s.NewTestWorkflowEnvironment()
	w := &testWorkflow{
		Count:             0,
		CompensationOrder: []int{},
	}
	env.RegisterActivity(w.add)
	env.RegisterActivity(w.minus)

	env.OnActivity(w.add, mock.Anything, 10).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 20).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 30).Return(errors.New("Test Compensation"))

	env.OnActivity(w.minus, mock.Anything, 10).Return(errors.New("Test ContinueOnError"))

	env.ExecuteWorkflow(w.basicSagaWorkflow, testSagaOptions{
		ParallelCompensation: false,
		ContinueOnError:      false,
	})

	s.True(env.IsWorkflowCompleted())
	s.EqualError(env.GetWorkflowError(), "workflow execution error (type: basicSagaWorkflow, workflowID: default-test-workflow-id, runID: default-test-run-id): activity error (type: add, scheduledEventID: 0, startedEventID: 0, identity: ): Test Compensation")

	result, err := env.QueryWorkflow("count")
	s.NoError(err)
	var count int
	err = result.Get(&count)
	s.NoError(err)
	s.Equal(30, count)

	result, err = env.QueryWorkflow("compensationOrder")
	s.NoError(err)
	var compensationOrder []int
	err = result.Get(&compensationOrder)
	s.NoError(err)
	s.Equal([]int{}, compensationOrder)

	env.AssertExpectations(s.T())
}

func (s *UnitTestSuite) Test_BasicSagaWorkflow_Parallel() {
	env := s.NewTestWorkflowEnvironment()
	w := &testWorkflow{
		Count:             0,
		CompensationOrder: []int{},
	}
	env.RegisterActivity(w.add)
	env.RegisterActivity(w.minus)

	env.OnActivity(w.add, mock.Anything, 10).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 20).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 30).Return(errors.New("Test Compensation"))

	env.ExecuteWorkflow(w.basicSagaWorkflow, testSagaOptions{
		ParallelCompensation: true,
		ContinueOnError:      false,
	})

	result, err := env.QueryWorkflow("count")
	s.NoError(err)
	var count int
	err = result.Get(&count)
	s.NoError(err)
	s.Equal(15, count)

	s.True(env.IsWorkflowCompleted())
	s.EqualError(env.GetWorkflowError(), "workflow execution error (type: basicSagaWorkflow, workflowID: default-test-workflow-id, runID: default-test-run-id): activity error (type: add, scheduledEventID: 0, startedEventID: 0, identity: ): Test Compensation")

	env.AssertExpectations(s.T())
}

func (s *UnitTestSuite) Test_BasicSagaWorkflow_ParallelFailure() {
	env := s.NewTestWorkflowEnvironment()
	w := &testWorkflow{
		Count:             0,
		CompensationOrder: []int{},
	}
	env.RegisterActivity(w.add)
	env.RegisterActivity(w.minus)

	env.OnActivity(w.add, mock.Anything, 10).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 20).Return(w.add)
	env.OnActivity(w.add, mock.Anything, 30).Return(errors.New("Test Compensation"))

	env.OnActivity(w.minus, mock.Anything, 10).Return(w.minus)
	env.OnActivity(w.minus, mock.Anything, 5).Return(errors.New("Test Parallel Failure"))

	env.ExecuteWorkflow(w.basicSagaWorkflow, testSagaOptions{
		ParallelCompensation: true,
		ContinueOnError:      false,
	})

	result, err := env.QueryWorkflow("count")
	s.NoError(err)
	var count int
	err = result.Get(&count)
	s.NoError(err)
	s.Equal(20, count)

	s.True(env.IsWorkflowCompleted())
	s.EqualError(env.GetWorkflowError(), "workflow execution error (type: basicSagaWorkflow, workflowID: default-test-workflow-id, runID: default-test-run-id): activity error (type: add, scheduledEventID: 0, startedEventID: 0, identity: ): Test Compensation")

	env.AssertExpectations(s.T())
}
