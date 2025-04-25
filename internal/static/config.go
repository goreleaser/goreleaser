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

// DenoExampleConfig is the config used within goreleaser init --lang deno.
//
//go:embed config.deno.yaml
var DenoExampleConfig []byte

// RustExampleConfig is the config used within goreleaser init --lang rust.
//
//go:embed config.rust.yaml
var RustExampleConfig []byte

// UVExampleConfig is the config used within goreleaser init --lang uv.
//
//go:embed config.uv.yaml
var UVExampleConfig []byte

// PoetryExampleConfig is the config used within goreleaser init --lang poetry.
//
//go:embed config.poetry.yaml
var PoetryExampleConfig []byte
