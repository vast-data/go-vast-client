package core

import (
	"errors"
	"fmt"
)

type NotFoundError struct {
	Resource string
	Query    string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("resource '%s' not found for params '%s'", e.Resource, e.Query)
}

type TooManyRecordsError struct {
	ResourcePath string
	Params       Params
}

// Implement the Error method to satisfy the error interface
func (e *TooManyRecordsError) Error() string {
	return fmt.Sprintf("too many records found for resource '%s' with params '%v'", e.ResourcePath, e.Params)
}

func IsNotFoundErr(err error) bool {
	var nfErr *NotFoundError
	return errors.As(err, &nfErr)
}

func IgnoreNotFound(val Record, err error) (Record, error) {
	if IsNotFoundErr(err) {
		return val, nil
	}
	return val, err
}

func IsTooManyRecordsErr(err error) bool {
	var tooManyRecordsErr *TooManyRecordsError
	return errors.As(err, &tooManyRecordsErr)
}
