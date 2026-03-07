package testlib

import (
	"runtime"
	"strings"
)

// GoVersion is the current Go version, without the "go" prefix.
var GoVersion = strings.TrimPrefix(runtime.Version(), "go")
