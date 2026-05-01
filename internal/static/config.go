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

// NodeExampleConfig is the config used within goreleaser init --lang node.
//
//go:embed config.node.yaml
var NodeExampleConfig []byte

// NodeSEAConfig is the starter sea-config.json dropped by
// goreleaser init --lang node when the project does not already have
// one. GoReleaser overwrites main / output / executable / useCodeCache
// / useSnapshot at build time, so the user-facing surface is just the
// few options that are theirs to customize.
//
//go:embed config.node.sea.json
var NodeSEAConfig []byte

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
