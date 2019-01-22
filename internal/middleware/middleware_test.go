package middleware

import "github.com/goreleaser/goreleaser/pkg/context"

var ctx = &context.Context{}

func mockAction(err error) Action {
	return func(ctx *context.Context) error {
		return err
	}
}
