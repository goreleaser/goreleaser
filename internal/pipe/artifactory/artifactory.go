// Package artifactory provides a Pipe that push to artifactory
package artifactory

import (
	"encoding/json"
	"fmt"
	"io"
	h "net/http"

	"github.com/goreleaser/goreleaser/internal/http"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
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
	// Check requirements for every instance we have configured.
	// If not fulfilled, we can skip this pipeline
	for _, instance := range ctx.Config.Artifactories {
		instance := instance
		if skip := http.CheckConfig(ctx, &instance, "artifactory"); skip != nil {
			return pipe.Skip(skip.Error())
		}
	}

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

// checkResponse checks the API response for errors, and returns them if
// present. A response is considered an error if it has a status code outside
// the 200 range.
// API error responses are expected to have either no response
// body, or a JSON response body that maps to ErrorResponse. Any other
// response body will be silently ignored.
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
			return err
		}
	}
	return errorResponse
}
