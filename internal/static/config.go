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

// BunExampleConfig is the config used within goreleaser init --lang bun.
//
//go:embed config.bun.yaml
var BunExampleConfig []byte

// RustExampleConfig is the config used within goreleaser init --lang rust.
//
//go:embed config.rust.yaml
var RustExampleConfig []byte
