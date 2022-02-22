package saga

type compensationOp struct {
	activity interface{}
	args     []interface{}
}

// Saga defines the interface for Saga compensations
type Saga interface {
	// AddCompensation adds a rollback compensation to the Saga
	AddCompensation(activity interface{}, args ...interface{})
	// Compensate executes all the compensation operations that have been defined
	Compensate() error
	// Cancel cancels the Saga context
	Cancel()
}
