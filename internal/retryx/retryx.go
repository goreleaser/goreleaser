// Package retryx provides shared retry configuration for goreleaser.
package retryx

import (
	"net/http"
	"strings"

	retry "github.com/avast/retry-go/v4"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

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
			log.IncreasePadding()
			log.WithError(err).WithField("try", n+1).Warn("retrying")
			log.DecreasePadding()
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

// IsRetriableHTTPError returns true if the status code or error indicates a
// transient HTTP failure worth retrying.
func IsRetriableHTTPError(statusCode int, err error) bool {
	if IsNetworkError(err) {
		return true
	}
	return statusCode >= 500 || statusCode == http.StatusTooManyRequests
}

// Unrecoverable wraps an error so that the retry loop stops immediately.
func Unrecoverable(err error) error {
	return retry.Unrecoverable(err)
}
