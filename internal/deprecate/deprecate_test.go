package deprecate

import (
	"bytes"
	"testing"

	"github.com/caarlos0/log"
	"github.com/charmbracelet/lipgloss"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

func TestNotice(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)

	var w bytes.Buffer
	log.Log = log.New(&w)

	log.Info("first")
	ctx := testctx.New()
	Notice(ctx, "foo.bar.whatever: foobar")
	log.Info("last")
	require.True(t, ctx.Deprecated)

	golden.RequireEqualTxt(t, w.Bytes())
}

func TestNoticeCustom(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)

	var w bytes.Buffer
	log.Log = log.New(&w)

	log.Info("first")
	ctx := testctx.New()
	NoticeCustom(ctx, "something-else", "some custom template with a url {{ .URL }}")
	log.Info("last")
	require.True(t, ctx.Deprecated)

	golden.RequireEqualTxt(t, w.Bytes())
}
