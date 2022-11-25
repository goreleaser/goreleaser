// Package static contains static "text" files.
package static

import _ "embed"

// ExampleConfig is the config used within goreleaser init.
//
//go:embed config.yaml
var ExampleConfig []byte
