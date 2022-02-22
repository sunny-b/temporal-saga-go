package saga

import (
	"fmt"
	"strings"
)

// CompensationError implements the error interface and aggregates
// the compensation errors that occur during execution
type CompensationError struct {
	Errors []error
}

func (e *CompensationError) addError(err error) {
	e.Errors = append(e.Errors, err)
}

func (e *CompensationError) hasErrors() bool {
	return len(e.Errors) > 0
}

func (e *CompensationError) Error() string {
	if !e.hasErrors() {
		return ""
	}

	var errors []string
	for _, err := range e.Errors {
		errors = append(errors, err.Error())
	}
	return fmt.Sprintf("error(s) in saga compensation: %s", strings.Join(errors, "; "))
}
