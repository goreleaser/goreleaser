package retryx

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func fastRetry(attempts uint) config.Retry {
	return config.Retry{
		Attempts: attempts,
		Delay:    time.Millisecond,
		MaxDelay: 10 * time.Millisecond,
	}
}

func TestIsNetworkError(t *testing.T) {
	for _, s := range []string{
		"connection reset by peer",
		"network is unreachable",
		"connection closed unexpectedly",
		"connection refused",
		"tls handshake timeout",
		"i/o timeout",
		"CONNECTION RESET",
		"TLS Handshake Timeout",
	} {
		t.Run(s, func(t *testing.T) {
			require.True(t, IsNetworkError(errors.New(s)))
		})
	}
}

func TestIsNetworkErrorNil(t *testing.T) {
	require.False(t, IsNetworkError(nil))
}

func TestIsNetworkErrorNotNetwork(t *testing.T) {
	for _, s := range []string{
		"file not found",
		"permission denied",
		"",
	} {
		t.Run(s, func(t *testing.T) {
			require.False(t, IsNetworkError(errors.New(s)))
		})
	}
}

func TestDoSuccess(t *testing.T) {
	err := Do(fastRetry(3), func() error {
		return nil
	}, nil)
	require.NoError(t, err)
}

func TestDoRetries(t *testing.T) {
	var calls atomic.Int32
	err := Do(fastRetry(3), func() error {
		if calls.Add(1) < 3 {
			return errors.New("transient")
		}
		return nil
	}, nil)
	require.NoError(t, err)
	require.Equal(t, int32(3), calls.Load())
}

func TestDoExhausted(t *testing.T) {
	var calls atomic.Int32
	err := Do(fastRetry(3), func() error {
		calls.Add(1)
		return errors.New("always fails")
	}, nil)
	require.ErrorContains(t, err, "always fails")
	require.Equal(t, int32(3), calls.Load())
}

func TestDoRetryIf(t *testing.T) {
	retryable := errors.New("retryable")
	fatal := errors.New("fatal")

	var calls atomic.Int32
	err := Do(fastRetry(5), func() error {
		if calls.Add(1) == 1 {
			return retryable
		}
		return fatal
	}, func(err error) bool {
		return errors.Is(err, retryable)
	})
	require.ErrorIs(t, err, fatal)
	require.Equal(t, int32(2), calls.Load())
}

func TestDoWithDataSuccess(t *testing.T) {
	val, err := DoWithData(fastRetry(3), func() (string, error) {
		return "hello", nil
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "hello", val)
}

func TestDoWithDataRetries(t *testing.T) {
	var calls atomic.Int32
	val, err := DoWithData(fastRetry(3), func() (int, error) {
		if calls.Add(1) < 3 {
			return 0, errors.New("transient")
		}
		return 42, nil
	}, nil)
	require.NoError(t, err)
	require.Equal(t, 42, val)
	require.Equal(t, int32(3), calls.Load())
}

func TestDoWithDataExhausted(t *testing.T) {
	val, err := DoWithData(fastRetry(2), func() (string, error) {
		return "", errors.New("always fails")
	}, nil)
	require.ErrorContains(t, err, "always fails")
	require.Empty(t, val)
}

func TestDoWithDataRetryIf(t *testing.T) {
	retryable := errors.New("retryable")
	fatal := errors.New("fatal")

	var calls atomic.Int32
	val, err := DoWithData(fastRetry(5), func() (string, error) {
		if calls.Add(1) == 1 {
			return "", retryable
		}
		return "", fatal
	}, func(err error) bool {
		return errors.Is(err, retryable)
	})
	require.ErrorIs(t, err, fatal)
	require.Empty(t, val)
	require.Equal(t, int32(2), calls.Load())
}
