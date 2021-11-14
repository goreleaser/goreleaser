package deprecate

import (
	"bytes"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestNotice(t *testing.T) {
	var w bytes.Buffer

	color.NoColor = true
	log.SetHandler(cli.New(&w))

	log.Info("first")
	ctx := context.New(config.Project{})
	Notice(ctx, "foo.bar.whatever")
	log.Info("last")
	require.True(t, ctx.Deprecated)

	golden.RequireEqualTxt(t, w.Bytes())
}

func TestNoticeCustom(t *testing.T) {
	var w bytes.Buffer

	color.NoColor = true
	log.SetHandler(cli.New(&w))

	log.Info("first")
	ctx := context.New(config.Project{})
	NoticeCustom(ctx, "something-else", "some custom template with a url {{ .URL }}")
	log.Info("last")
	require.True(t, ctx.Deprecated)

	golden.RequireEqualTxt(t, w.Bytes())
}

func TestWriter(t *testing.T) {
	var w bytes.Buffer

	color.NoColor = true
	log.SetHandler(cli.New(&w))

	log.Info("first")
	ctx := context.New(config.Project{})
	ww := NewWriter(ctx)
	_, err := ww.Write([]byte("foo bar\n"))
	require.NoError(t, err)
	require.True(t, ctx.Deprecated)

	golden.RequireEqualTxt(t, w.Bytes())
}
