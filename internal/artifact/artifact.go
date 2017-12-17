// Package artifact provides the core artifact storage for goreleaser
package artifact

import "sync"

// Type defines the type of an artifact
type Type int

const (
	// UploadableArchive a tar.gz/zip archive to be uploaded
	UploadableArchive Type = iota
	// UploadableBinary is a binary file to be uploaded
	UploadableBinary
	// Binary is a binary (output of a gobuild)
	Binary
	// DockerImage is a docker image
	DockerImage
	// Checksum is a checksum file
	Checksum
)

// Artifact represents an artifact and its relevant info
type Artifact struct {
	Name   string
	Path   string
	Goos   string
	Goarch string
	Goarm  string
	Type   Type
}

// Artifacts is a list of artifacts
type Artifacts struct {
	items []Artifact
	lock  *sync.Mutex
}

// New return a new list of artifacts
func New() Artifacts {
	return Artifacts{
		items: []Artifact{},
		lock:  &sync.Mutex{},
	}
}

// List return the actual list of artifacts
func (artifacts Artifacts) List() []Artifact {
	return artifacts.items
}

// Platform returns the platform string of a given artifact
func (a Artifact) Platform() string {
	return a.Goos + a.Goarch + a.Goarm
}

// GroupByPlatform groups the artifacts by their platform
func (artifacts *Artifacts) GroupByPlatform() map[string][]Artifact {
	var result = map[string][]Artifact{}
	for _, a := range artifacts.items {
		result[a.Platform()] = append(result[a.Platform()], a)
	}
	return result
}

// Add safely adds a new artifact to an artifact list
func (artifacts *Artifacts) Add(a Artifact) {
	artifacts.lock.Lock()
	defer artifacts.lock.Unlock()
	artifacts.items = append(artifacts.items, a)
}

// Filter defines an artifact filter which can be used within the Filter
// function
type Filter func(a Artifact) bool

// ByGoos is a predefined filter that filters by the given goos
func ByGoos(s string) Filter {
	return func(a Artifact) bool {
		return a.Goos == s
	}
}

// ByGoarch is a predefined filter that filters by the given goarch
func ByGoarch(s string) Filter {
	return func(a Artifact) bool {
		return a.Goarch == s
	}
}

// ByGoarm is a predefined filter that filters by the given goarm
func ByGoarm(s string) Filter {
	return func(a Artifact) bool {
		return a.Goarm == s
	}
}

// ByType is a predefined filter that filters by the given type
func ByType(t Type) Filter {
	return func(a Artifact) bool {
		return a.Type == t
	}
}

// Filter filters the artifact list, returning a new instance.
// There are some pre-defined filters but anything of the Type Filter
// is accepted.
func (artifacts *Artifacts) Filter(filter Filter) Artifacts {
	var result = New()
	for _, a := range artifacts.items {
		if filter(a) {
			result.Add(a)
		}
	}
	return result
}
