package linkedin

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
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
				Context:     testctx.New(),
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
				Context:     testctx.New(),
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
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(rw, `
{
	"sub": "foo",
	"activity": "123456789"
}
`)
	}))
	defer server.Close()

	c, err := createLinkedInClient(oauthClientConfig{
		Context:     testctx.New(),
		AccessToken: "foo",
	})
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}

	c.baseURL = server.URL

	link, err := c.Share(t.Context(), "test")
	if err != nil {
		t.Fatalf("could not share: %v", err)
	}

	wantLink := "https://www.linkedin.com/feed/update/123456789"
	require.Equal(t, wantLink, link)
}

func TestClientLegacyProfile_Share(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/v2/userinfo" {
			rw.WriteHeader(http.StatusForbidden)
			return
		}
		// this is the response from /v2/me (legacy as a fallback)
		_, _ = io.WriteString(rw, `
		{
			"id": "foo",
			"activity": "123456789"
		}
		`)
	}))
	defer server.Close()

	c, err := createLinkedInClient(oauthClientConfig{
		Context:     testctx.New(),
		AccessToken: "foo",
	})
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}

	c.baseURL = server.URL

	link, err := c.Share(t.Context(), "test")
	if err != nil {
		t.Fatalf("could not share: %v", err)
	}

	wantLink := "https://www.linkedin.com/feed/update/123456789"
	require.Equal(t, wantLink, link)
}
