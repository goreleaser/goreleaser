// Package static contains static "text" files, just because embedding real
// static files would be kind of an overengineering right now, so it's just
// strings in go code really.
package static

import _ "embed"

// ExampleConfig is the config used within goreleaser init.
//
//go:embed config.yaml
var ExampleConfig []byte
