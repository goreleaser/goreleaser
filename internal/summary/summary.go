// Package summary appends one-line entries describing notable release actions
// (published releases, opened pull requests, pushed packages, uploaded
// artifacts) to the GitHub Actions job summary. Outside GitHub Actions it does
// nothing — the regular logs already cover the same ground.
package summary

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/caarlos0/log"
)

// envSummary is the file GitHub Actions renders the job summary from.
const envSummary = "GITHUB_STEP_SUMMARY"

var mu sync.Mutex

// Append writes a markdown bullet to the GitHub Actions job summary. It is a
// no-op when not running in GitHub Actions, and is safe for concurrent use.
func Append(line string) {
	path := os.Getenv(envSummary)
	if path == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		_, werr := fmt.Fprintln(f, "- "+line)
		err = errors.Join(werr, f.Close())
	}
	if err != nil {
		log.WithError(err).Warn("failed to write github actions summary")
	}
}

// Appendf is [Append] with formatting.
func Appendf(format string, a ...any) {
	Append(fmt.Sprintf(format, a...))
}
