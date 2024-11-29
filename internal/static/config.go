// Package static contains static "text" files.
package static

import _ "embed"

// GoExampleConfig is the config used within goreleaser init.
//
//go:embed config.yaml
var GoExampleConfig []byte

// ZigExampleConfig is the config used within goreleaser init --lang zig.
//
//go:embed config.zig.yaml
var ZigExampleConfig []byte
