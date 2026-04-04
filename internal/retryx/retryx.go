// Package retryx provides shared retry configuration for goreleaser.
package retryx

import (
	"errors"
	"net/http"
	"strings"

	retry "github.com/avast/retry-go/v4"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

// HTTPError carries an HTTP status code alongside the original error.
type HTTPError struct {
	Err    error
	Status int
}

func (e HTTPError) Error() string { return e.Err.Error() }
func (e HTTPError) Unwrap() error { return e.Err }

// HTTP wraps err with the status code from resp.
// A nil resp yields Status 0 (network-level failure).
func HTTP(err error, resp *http.Response) error {
	if err == nil {
		return nil
	}
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	return HTTPError{Err: err, Status: status}
}

type retriableError struct{ error }

func (e retriableError) Unwrap() error { return e.error }

// Retriable wraps err so IsRetriable returns true unconditionally.
func Retriable(err error) error {
	if err == nil {
		return nil
	}
	return retriableError{err}
}

// IsRetriable returns true if the error represents a transient failure worth
// retrying: network errors, 5xx, 429, or explicitly marked retriable.
func IsRetriable(err error) bool {
	if IsNetworkError(err) {
		return true
	}
	var re retriableError
	if errors.As(err, &re) {
		return true
	}
	var he HTTPError
	if errors.As(err, &he) {
		return he.Status >= 500 || he.Status == http.StatusTooManyRequests
	}
	return false
}

// DoWithData retries the given retryableFunc with the given configuration,
// following retryIf, and returns the data from retryableFunc.
func DoWithData[T any](c config.Retry, retryableFunc func() (T, error), retryIf func(error) bool) (T, error) {
	return retry.DoWithData(retryableFunc, opts(c, retryIf)...)
}

// Do retries the given retryableFunc with the given configuration, following retryIf.
func Do(c config.Retry, retryableFunc func() error, retryIf func(error) bool) error {
	return retry.Do(retryableFunc, opts(c, retryIf)...)
}

func opts(c config.Retry, retryIf func(error) bool) []retry.Option {
	attempts := c.Attempts
	if attempts == 0 {
		attempts = 1
	}
	opts := []retry.Option{
		retry.Attempts(attempts),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(c.Delay),
		retry.MaxDelay(c.MaxDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			log.WithError(err).WithField("try", n+1).Warn("retrying")
		}),
	}
	if retryIf != nil {
		opts = append(opts, retry.RetryIf(retryIf))
	}
	return opts
}

// IsNetworkError returns true if the error looks like a transient network error.
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection reset") ||
		strings.Contains(s, "network is unreachable") ||
		strings.Contains(s, "connection closed") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "tls handshake timeout") ||
		strings.Contains(s, "i/o timeout")
}

// Unrecoverable wraps an error so that the retry loop stops immediately.
func Unrecoverable(err error) error {
	return retry.Unrecoverable(err)
}
