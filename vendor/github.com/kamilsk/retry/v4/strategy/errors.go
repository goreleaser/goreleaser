package strategy

import "net"

const (
	Skip   = true
	Strict = false
)

// An Error represents a retriable error.
type Error interface {
	error
	Retriable() bool // Is the error retriable?
}

// CheckError creates a Strategy that checks an error and returns
// if an error is retriable or not. Otherwise, it returns the defaults.
func CheckError(defaults bool) Strategy {
	return func(_ uint, err error) bool {
		if err == nil {
			return true
		}
		if err, is := err.(Error); is {
			return err.Retriable()
		}
		return defaults
	}
}

// CheckNetworkError creates a Strategy that checks an error and returns true
// if an error is the temporary network error.
// The Strategy returns the defaults if an error is not a network error.
func CheckNetworkError(defaults bool) Strategy {
	return func(_ uint, err error) bool {
		if err == nil {
			return true
		}
		if err, is := err.(net.Error); is {
			return err.Temporary() || err.Timeout()
		}
		return defaults
	}
}
