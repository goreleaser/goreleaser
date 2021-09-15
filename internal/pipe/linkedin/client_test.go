package linkedin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

func TestCreateLinkedInClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     OAuthClientConfig
		wantErr error
	}{
		{
			"non-empty context and access token",
			OAuthClientConfig{
				Context:     context.New(config.Project{}),
				AccessToken: "foo",
			},
			nil,
		},
		{
			"empty context",
			OAuthClientConfig{
				Context:     nil,
				AccessToken: "foo",
			},
			fmt.Errorf("context is nil"),
		},
		{
			"empty access token",
			OAuthClientConfig{
				Context:     context.New(config.Project{}),
				AccessToken: "",
			},
			fmt.Errorf("empty access token"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateLinkedInClient(tt.cfg)

			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("CreateLinkedInClient() error = %v, wantErr %v", err, tt.wantErr)
				return
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

	c, err := CreateLinkedInClient(OAuthClientConfig{
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

	if link != wantLink {
		t.Fatalf("link got: %s want: %s", link, wantLink)
	}
}
