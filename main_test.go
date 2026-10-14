package main

import (
"encoding/json"
"testing"

"github.com/goreleaser/goreleaser/v2/internal/golden"
"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
const (
goVersion = "go1.20.3"
compiler = "gc"
platform = "linux/amd64"
)

tests := []struct {
name string
version string
commit string
date string
builtBy string
treeState string
}{
{
name: "all empty",
},
{
name: "complete",
version: "1.2.3",
commit: "aaaa",
date: "12/12/12",
builtBy: "me",
treeState: "clean",
},
{
name: "only version",
version: "1.2.3",
},
{
name: "version and date",
version: "1.2.3",
date: "12/12/12",
},
{
name: "version, date, built by",
version: "1.2.3",
date: "12/12/12",
builtBy: "me",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
v := buildVersion(
tt.version,
tt.commit,
tt.date,
tt.builtBy,
tt.treeState,
)

v.GoVersion = goVersion
v.Compiler = compiler
v.Platform = platform

out, err := v.JSONString()
require.NoError(
t,
err,
"a geração do JSON não deve falhar",
)

require.True(
t,
json.Valid([]byte(out)),
"a saída deve conter JSON válido: %s",
out,
)

golden.RequireEqualJSON(t, []byte(out))
})
}
} 
