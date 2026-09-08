package effectiveconfig

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	slackpipe "github.com/goreleaser/goreleaser/v2/internal/pipe/slack"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestPipeDescription(t *testing.T) {
	require.Empty(t, Pipe{}.String())
}

func TestRun(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: dist,
	})

	require.NoError(t, Pipe{}.Run(ctx))
	bts, err := os.ReadFile(filepath.Join(dist, "config.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, string(bts))
}

func TestRunPreservesSlackBlocksAndAttachments(t *testing.T) {
	const conf = `
version: 2
project_name: schema-audit
announce:
  slack:
    enabled: true
    message_template: fallback
    blocks:
      - type: divider
    attachments:
      - text: hello
`
	project, err := config.LoadReader(strings.NewReader(conf))
	require.NoError(t, err)

	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	project.Dist = dist

	var payloads []slackWebhookPayload
	var payloadsMu sync.Mutex
	decodeErrs := make(chan error, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload slackWebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		decodeErrs <- err
		payloadsMu.Lock()
		payloads = append(payloads, payload)
		payloadsMu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("SLACK_WEBHOOK", srv.URL)

	require.NoError(t, slackpipe.Pipe{}.Announce(testctx.WrapWithCfg(t.Context(), project)))
	require.NoError(t, Pipe{}.Run(testctx.WrapWithCfg(t.Context(), project)))

	bts, err := os.ReadFile(filepath.Join(dist, "config.yaml"))
	require.NoError(t, err)
	require.NotContains(t, string(bts), "internal:")

	reloaded, err := config.Load(filepath.Join(dist, "config.yaml"))
	require.NoError(t, err)
	require.NoError(t, slackpipe.Pipe{}.Announce(testctx.WrapWithCfg(t.Context(), reloaded)))

	require.NoError(t, <-decodeErrs)
	require.NoError(t, <-decodeErrs)

	payloadsMu.Lock()
	got := append([]slackWebhookPayload(nil), payloads...)
	payloadsMu.Unlock()
	require.Len(t, got, 2)
	for _, payload := range got {
		require.Len(t, payload.Blocks, 1)
		require.Equal(t, "divider", payload.Blocks[0]["type"])
		require.NotContains(t, payload.Blocks[0], "internal")
		require.Len(t, payload.Attachments, 1)
		require.Equal(t, "hello", payload.Attachments[0]["text"])
		require.NotContains(t, payload.Attachments[0], "internal")
	}
}

type slackWebhookPayload struct {
	Blocks      []map[string]any `json:"blocks"`
	Attachments []map[string]any `json:"attachments"`
}
