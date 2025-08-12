package retry

import (
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	r := New[string](
		"test operation",
		3,
		10*time.Millisecond,
		100*time.Millisecond,
		func(error) bool { return true },
	)

	callCount := 0
	result, err := r.Do(func() (string, error) {
		callCount++
		if callCount < 2 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result != "success" {
		t.Fatalf("expected 'success', got %q", result)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount)
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	r := New[string](
		"test operation",
		3,
		10*time.Millisecond,
		100*time.Millisecond,
		func(err error) bool { return err.Error() != "non-retryable" },
	)

	callCount := 0
	_, err := r.Do(func() (string, error) {
		callCount++
		return "", errors.New("non-retryable")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}
}

func TestRetry_MaxAttemptsReached(t *testing.T) {
	r := New[string](
		"test operation",
		2,
		1*time.Millisecond,
		10*time.Millisecond,
		func(error) bool { return true },
	)

	callCount := 0
	_, err := r.Do(func() (string, error) {
		callCount++
		return "", errors.New("always fails")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount)
	}
}
