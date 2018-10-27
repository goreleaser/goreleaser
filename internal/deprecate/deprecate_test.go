package deprecate

import (
	"bytes"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/stretchr/testify/require"
)

func init() {
	color.NoColor = true
}

func TestNotice(t *testing.T) {
	var out bytes.Buffer
	cli.Default.Writer = &out
	log.SetHandler(cli.Default)

	log.Info("first")
	Notice("foo.bar.whatever")
	log.Info("last")

	require.Contains(t, out.String(), "   • first")
	require.Contains(t, out.String(), "      • DEPRECATED: `foo.bar.whatever` should not be used anymore, check https://goreleaser.com/deprecations#foo-bar-whatever for more info.")
	require.Contains(t, out.String(), "   • last")
}
