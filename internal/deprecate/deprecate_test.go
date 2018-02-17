package deprecate

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
)

func TestNotice(t *testing.T) {
	var out bytes.Buffer
	cli.Default.Writer = &out
	log.SetHandler(cli.Default)
	log.Info("first")
	Notice("foo.bar.whatever")
	log.Info("last")

	assert.Contains(t, out.String(), "   • first")
	assert.Contains(t, out.String(), "      • DEPRECATED: `foo.bar.whatever` should not be used anymore, check https://goreleaser.com/#deprecation_notices.foo_bar_whatever for more info.")
	assert.Contains(t, out.String(), "   • last")
}
