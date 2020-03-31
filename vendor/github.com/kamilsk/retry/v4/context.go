package retry

import "context"

// TryContext takes an interruptable action and performs it, repetitively, until successful.
// It uses the Context as a Breaker to prevent unnecessary action execution.
//
// Optionally, strategies may be passed that assess whether or not an attempt
// should be made.
//
// Deprecated: will be replaced by Do function (current Try).
// TODO:v5 will be removed
func TryContext(
	ctx context.Context,
	action func(ctx context.Context, attempt uint) error,
	strategies ...func(attempt uint, err error) bool,
) error {
	return retry(ctx, func(attempt uint) error { return action(ctx, attempt) }, strategies...)
}
