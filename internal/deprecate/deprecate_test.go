package deprecate

import (
	"bytes"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestNotice(t *testing.T) {
	var out bytes.Buffer
	cli.Default.Writer = &out
	log.SetHandler(cli.Default)
	log.Info("first")
	Notice("foo.bar.whatever")
	log.Info("last")
	color.NoColor = true

	assert.Contains(t, out.String(), "   • first")
	assert.Contains(t, out.String(), "      • DEPRECATED: `foo.bar.whatever` should not be used anymore, check https://goreleaser.com/deprecations#foo-bar-whatever for more info.")
	assert.Contains(t, out.String(), "   • last")
}
