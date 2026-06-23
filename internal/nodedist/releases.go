package nodedist

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed releases.json
var rawReleases []byte

// Releases returns the embedded Node.js release index, decoded once
// and cached for the lifetime of the process. Each entry maps an
// archive name (matching what Target.ArchiveName returns) to its
// SHA-256, hex-encoded.
var Releases = sync.OnceValue(func() map[string]map[string]string {
	var m map[string]map[string]string
	if err := json.Unmarshal(rawReleases, &m); err != nil {
		panic(fmt.Sprintf("nodedist: decode embedded releases.json: %v", err))
	}
	return m
})

// lookupSHA returns the embedded SHA-256 for the given (version,
// archiveName). Both arguments must match the upstream layout exactly
// (version with leading `v`, archiveName as published in
// SHASUMS256.txt). Returns an actionable error when missing.
func lookupSHA(version, archiveName string) (string, error) {
	files, ok := Releases()[version]
	if !ok {
		return "", fmt.Errorf("nodedist: no embedded entry for %s; run `task nodedist:releases:generate` (or update goreleaser)", version)
	}
	sha, ok := files[archiveName]
	if !ok {
		return "", fmt.Errorf("nodedist: no embedded SHA for %s/%s; run `task nodedist:releases:generate` (or update goreleaser)", version, archiveName)
	}
	return sha, nil
}
