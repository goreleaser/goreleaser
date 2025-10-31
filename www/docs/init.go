// Package docs contains the embedded documentation files.
package docs

import "embed"

// FS contains the FS with the markdown of the documentation.
//
//go:embed customization *deprecations.md
var FS embed.FS
