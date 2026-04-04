package retryx

import (
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func retryConfig(attempts uint) config.Retry {
	return config.Retry{
		Attempts: attempts,
		Delay:    10 * time.Second,
		MaxDelay: 5 * time.Minute,
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
	synctest.Test(t, func(t *testing.T) {
		err := Do(retryConfig(3), func() error {
			return nil
		}, nil)
		require.NoError(t, err)
	})
}

func TestDoRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		err := Do(retryConfig(3), func() error {
			if calls.Add(1) < 3 {
				return errors.New("transient")
			}
			return nil
		}, nil)
		require.NoError(t, err)
		require.Equal(t, int32(3), calls.Load())
	})
}

func TestDoExhausted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		err := Do(retryConfig(3), func() error {
			calls.Add(1)
			return errors.New("always fails")
		}, nil)
		require.ErrorContains(t, err, "always fails")
		require.Equal(t, int32(3), calls.Load())
	})
}

func TestDoRetryIf(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		retryable := errors.New("retryable")
		fatal := errors.New("fatal")

		var calls atomic.Int32
		err := Do(retryConfig(5), func() error {
			if calls.Add(1) == 1 {
				return retryable
			}
			return fatal
		}, func(err error) bool {
			return errors.Is(err, retryable)
		})
		require.ErrorIs(t, err, fatal)
		require.Equal(t, int32(2), calls.Load())
	})
}

func TestDoWithDataSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		val, err := DoWithData(retryConfig(3), func() (string, error) {
			return "hello", nil
		}, nil)
		require.NoError(t, err)
		require.Equal(t, "hello", val)
	})
}

func TestDoWithDataRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		val, err := DoWithData(retryConfig(3), func() (int, error) {
			if calls.Add(1) < 3 {
				return 0, errors.New("transient")
			}
			return 42, nil
		}, nil)
		require.NoError(t, err)
		require.Equal(t, 42, val)
		require.Equal(t, int32(3), calls.Load())
	})
}

func TestDoWithDataExhausted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		val, err := DoWithData(retryConfig(2), func() (string, error) {
			return "", errors.New("always fails")
		}, nil)
		require.ErrorContains(t, err, "always fails")
		require.Empty(t, val)
	})
}

func TestDoWithDataRetryIf(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		retryable := errors.New("retryable")
		fatal := errors.New("fatal")

		var calls atomic.Int32
		val, err := DoWithData(retryConfig(5), func() (string, error) {
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
	})
}

func TestIsRetriableHTTPError(t *testing.T) {
	t.Run("network error", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(0, errors.New("connection reset")))
	})
	t.Run("500", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(500, errors.New("internal server error")))
	})
	t.Run("502", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(502, errors.New("bad gateway")))
	})
	t.Run("503", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(503, errors.New("service unavailable")))
	})
	t.Run("429", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(429, errors.New("rate limited")))
	})
	t.Run("404 not retriable", func(t *testing.T) {
		require.False(t, IsRetriableHTTPError(404, errors.New("not found")))
	})
	t.Run("422 not retriable", func(t *testing.T) {
		require.False(t, IsRetriableHTTPError(422, errors.New("unprocessable")))
	})
	t.Run("200 no error", func(t *testing.T) {
		require.False(t, IsRetriableHTTPError(200, nil))
	})
	t.Run("0 no error", func(t *testing.T) {
		require.False(t, IsRetriableHTTPError(0, nil))
	})
}

func TestUnrecoverable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		err := Unrecoverable(errors.New("permanent"))
		var calls atomic.Int32
		result := Do(retryConfig(5), func() error {
			calls.Add(1)
			return err
		}, nil)
		require.ErrorContains(t, result, "permanent")
		require.Equal(t, int32(1), calls.Load())
	})
}
