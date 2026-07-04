// Package gitlabregistry provides a Pipe that push to a GitLab generic
// package registry.
package gitlabregistry

import (
	"encoding/json"
	"fmt"
	"io"
	h "net/http"

	"github.com/goreleaser/goreleaser/v2/internal/http"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const kind = "gitlab_registry"

// Pipe for GitLab package registries.
type Pipe struct{}

func (Pipe) String() string                 { return "gitlab package registries" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.GitLabRegistries) == 0 }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.GitLabRegistries {
		ctx.Config.GitLabRegistries[i].Method = h.MethodPut
	}
	return http.Defaults(ctx.Config.GitLabRegistries)
}

// Publish artifacts to a GitLab generic package registry.
//
// Docs: https://docs.gitlab.com/user/packages/generic_packages/
func (Pipe) Publish(ctx *context.Context) error {
	// Check requirements for every instance we have configured.
	// If not fulfilled, we can skip this pipeline
	for _, instance := range ctx.Config.GitLabRegistries {
		if skip := http.CheckConfig(ctx, &instance, kind); skip != nil {
			return pipe.Skip(skip.Error())
		}
	}

	return http.Upload(ctx, ctx.Config.GitLabRegistries, kind, checkResponse)
}

// An errorResponse reports the error caused by an API request.
// GitLab API errors have either a "message" or an "error" field.
type errorResponse struct {
	Response *h.Response // HTTP response that caused this error
	Message  string      `json:"message"`
	Err      string      `json:"error"`
}

func (r *errorResponse) Error() string {
	msg := r.Message
	if msg == "" {
		msg = r.Err
	}
	return fmt.Sprintf("%v %v: %d %s",
		r.Response.Request.Method, r.Response.Request.URL,
		r.Response.StatusCode, msg)
}

// checkResponse checks the API response for errors, and returns them if
// present. A response is considered an error if it has a status code outside
// the 200 range.
// API error responses are expected to have either no response
// body, or a JSON response body. Any other response body will be silently
// ignored.
func checkResponse(r *h.Response) error {
	defer r.Body.Close()
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := &errorResponse{Response: r}
	data, err := io.ReadAll(r.Body)
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, errorResponse); err != nil {
			return fmt.Errorf("unexpected error: %w: %s", err, string(data))
		}
	}
	return errorResponse
}
