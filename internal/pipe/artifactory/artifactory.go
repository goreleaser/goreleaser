// Package artifactory provides a Pipe that push to artifactory
package artifactory

import (
	"encoding/json"
	"fmt"
	"io"
	h "net/http"

	"github.com/goreleaser/goreleaser/v2/internal/http"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe for Artifactory.
type Pipe struct{}

func (Pipe) String() string                 { return "artifactory" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Artifactories) == 0 }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Artifactories {
		if ctx.Config.Artifactories[i].ChecksumHeader == "" {
			ctx.Config.Artifactories[i].ChecksumHeader = "X-Checksum-SHA256"
		}
		ctx.Config.Artifactories[i].Method = h.MethodPut
	}
	return http.Defaults(ctx.Config.Artifactories)
}

// Publish artifacts to artifactory.
//
// Docs: https://www.jfrog.com/confluence/display/RTF/Artifactory+REST+API#ArtifactoryRESTAPI-Example-DeployinganArtifact
func (Pipe) Publish(ctx *context.Context) error {
	return http.Upload(ctx, ctx.Config.Artifactories, "artifactory", checkResponse)
}

// An ErrorResponse reports one or more errors caused by an API request.
type errorResponse struct {
	Response *h.Response // HTTP response that caused this error
	Errors   []Error     `json:"errors"` // more detail on individual errors
}

func (r *errorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %+v",
		r.Response.Request.Method, r.Response.Request.URL,
		r.Response.StatusCode, r.Errors)
}

// An Error reports more details on an individual error in an ErrorResponse.
type Error struct {
	Status  int    `json:"status"`  // Error code
	Message string `json:"message"` // Message describing the error.
}

// checkResponse treats status codes outside the 200 range as errors.
// It decodes the response body into an errorResponse. An empty body or invalid
// JSON returns a decoding error.
func checkResponse(r *h.Response) error {
	defer r.Body.Close()
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := &errorResponse{Response: r}
	data, err := io.ReadAll(r.Body)
	if err == nil && data != nil {
		err := json.Unmarshal(data, errorResponse)
		if err != nil {
			return fmt.Errorf("unexpected error: %w: %s", err, string(data))
		}
	}
	return errorResponse
}
