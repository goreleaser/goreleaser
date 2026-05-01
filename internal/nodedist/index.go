package nodedist

import (
	"context"
	"encoding/json"
	"fmt"
)

// Release mirrors one record in https://nodejs.org/dist/index.json,
// which lists every published Node.js release. Only the fields
// goreleaser actually consults are decoded; the upstream document is
// substantially larger.
type Release struct {
	Version string `json:"version"`
	LTS     any    `json:"lts"`
}

// IndexFetcher returns the parsed nodejs.org release index. It is a
// package-level variable so tests can stub it without hitting the
// network.
//
//nolint:gochecknoglobals
var IndexFetcher = fetchIndex

// Index returns the release index by delegating to IndexFetcher,
// allowing callers a single, replaceable entry-point.
func Index(ctx context.Context) ([]Release, error) {
	return IndexFetcher(ctx)
}

func fetchIndex(ctx context.Context) ([]Release, error) {
	u := distBaseURL + "/index.json"
	body, err := getBody(ctx, u)
	if err != nil {
		return nil, err
	}
	var entries []Release
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("nodedist: decode index.json: %w", err)
	}
	return entries, nil
}
