// Package nodedist is a small client for the official Node.js
// distribution at https://nodejs.org/dist. It exposes the embedded
// release index and a SHA-256 verifying downloader.
//
// The package is intentionally infrastructure-only: it knows nothing
// about Single Executable Applications, version resolution from
// project files, target/GOOS mapping, or build orchestration. Callers
// (typically the nodesea package) layer those concerns on top.
package nodedist

import (
	"net/http"
	"time"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

// distBaseURL is the prefix every nodejs.org/dist URL uses. It is a
// package-level variable so tests can point it at a stub server.
//
//nolint:gochecknoglobals
var distBaseURL = "https://nodejs.org/dist"

// httpClient is the HTTP client used for nodejs.org downloads. Tests
// may override it; production uses http.DefaultClient.
//
//nolint:gochecknoglobals
var httpClient = http.DefaultClient

// defaultRetry is the retry policy applied to every nodejs.org HTTP
// fetch. It mirrors what the rest of goreleaser uses for transient
// server errors but is fixed at package scope rather than threaded
// from ctx.Config.Retry to avoid leaking a goreleaser context
// dependency into nodedist.
//
//nolint:gochecknoglobals
var defaultRetry = config.Retry{
	Attempts: 4,
	Delay:    time.Second,
	MaxDelay: 30 * time.Second,
}
