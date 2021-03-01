package deprecate

import (
	"bytes"
	"flag"
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update .golden files")

func TestNotice(t *testing.T) {
	var w bytes.Buffer

	color.NoColor = true
	log.SetHandler(cli.New(&w))

	log.Info("first")
	ctx := context.New(config.Project{})
	Notice(ctx, "foo.bar.whatever")
	log.Info("last")
	require.True(t, ctx.Deprecated)

	const golden = "testdata/output.txt.golden"
	if *update {
		require.NoError(t, os.WriteFile(golden, w.Bytes(), 0o655))
	}

	gbts, err := os.ReadFile(golden)
	require.NoError(t, err)

	require.Equal(t, string(gbts), w.String())
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

	const golden = "testdata/output_custom.txt.golden"
	if *update {
		require.NoError(t, os.WriteFile(golden, w.Bytes(), 0o655))
	}

	gbts, err := os.ReadFile(golden)
	require.NoError(t, err)

	require.Equal(t, string(gbts), w.String())
}
