// Package static contains static "text" files.
package static

import _ "embed"

// GoExampleConfig is the config used within goreleaser init.
//
//go:embed config.yaml
var GoExampleConfig []byte

// ZigExampleConfig is the config used within goreleaser init --language zig.
//
//go:embed config.zig.yaml
var ZigExampleConfig []byte

// BunExampleConfig is the config used within goreleaser init --language bun.
//
//go:embed config.bun.yaml
var BunExampleConfig []byte

// DenoExampleConfig is the config used within goreleaser init --language deno.
//
//go:embed config.deno.yaml
var DenoExampleConfig []byte

// NodeExampleConfig is the config used within goreleaser init --language node.
//
//go:embed config.node.yaml
var NodeExampleConfig []byte

// RustExampleConfig is the config used within goreleaser init --language rust.
//
//go:embed config.rust.yaml
var RustExampleConfig []byte

// UVExampleConfig is the config used within goreleaser init --language uv.
//
//go:embed config.uv.yaml
var UVExampleConfig []byte

// PoetryExampleConfig is the config used within goreleaser init --language poetry.
//
//go:embed config.poetry.yaml
var PoetryExampleConfig []byte
