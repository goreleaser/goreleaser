package deprecate

import (
	"bytes"
	"testing"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/stretchr/testify/require"
)

func TestNotice(t *testing.T) {
	t.Setenv("CI", "")
	var w bytes.Buffer
	log.Log = log.New(&w)

	log.Info("first")
	ctx := testctx.Wrap(t.Context())
	Notice(ctx, "foo.bar_whatever: foobar")
	Notice(ctx, "foo.bar_whatever: foobar")
	Notice(ctx, "foo.bar_whatever: foobar")
	log.Info("last")
	require.True(t, ctx.Deprecated)

	golden.RequireEqualTxt(t, w.Bytes())
}

func TestNoticeCustom(t *testing.T) {
	t.Setenv("CI", "")
	var w bytes.Buffer
	log.Log = log.New(&w)

	log.Info("first")
	ctx := testctx.Wrap(t.Context())
	NoticeCustom(ctx, "something-else", "some custom template with a url {{ .URL }}")
	NoticeCustom(ctx, "something-else", "ignored")
	NoticeCustom(ctx, "something-else", "ignored")
	log.Info("last")
	require.True(t, ctx.Deprecated)

	golden.RequireEqualTxt(t, w.Bytes())
}
