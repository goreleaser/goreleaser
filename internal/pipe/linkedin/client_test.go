package linkedin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestCreateLinkedInClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     oauthClientConfig
		wantErr error
	}{
		{
			"non-empty context and access token",
			oauthClientConfig{
				Context:     context.New(config.Project{}),
				AccessToken: "foo",
			},
			nil,
		},
		{
			"empty context",
			oauthClientConfig{
				Context:     nil,
				AccessToken: "foo",
			},
			fmt.Errorf("context is nil"),
		},
		{
			"empty access token",
			oauthClientConfig{
				Context:     context.New(config.Project{}),
				AccessToken: "",
			},
			fmt.Errorf("empty access token"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createLinkedInClient(tt.cfg)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_Share(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(`
{
	"id": "foo",
	"activity": "123456789"
}
`))
	}))
	defer server.Close()

	c, err := createLinkedInClient(oauthClientConfig{
		Context:     context.New(config.Project{}),
		AccessToken: "foo",
	})
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}

	c.baseURL = server.URL

	link, err := c.Share("test")
	if err != nil {
		t.Fatalf("could not share: %v", err)
	}

	wantLink := "https://www.linkedin.com/feed/update/123456789"
	require.Equal(t, wantLink, link)
}
