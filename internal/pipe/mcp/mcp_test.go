package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/modelcontextprotocol/registry/cmd/publisher/auth"
	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"github.com/modelcontextprotocol/registry/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "mcp", Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			MCP: config.MCP{
				Name: "foo",
			},
		})
		skips.Set(ctx, skips.MCP)
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("skip no mcp name", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		skips.Set(ctx, skips.MCP)
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			MCP: config.MCP{
				Name: "foo",
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestDefault(t *testing.T) {
	t.Run("empty auth type", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			MCP: config.MCP{
				Name: "test-server",
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, "none", ctx.Config.MCP.Auth.Type)
	})

	t.Run("none auth", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			MCP: config.MCP{
				Name: "test-server",
				Auth: config.MCPAuth{
					Type: "none",
				},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.Empty(t, ctx.Config.MCP.Auth.Token)
	})
}

func TestPublishSuccess(t *testing.T) {
	var receivedRequest apiv0.ServerJSON
	var receivedToken string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v0/publish", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		receivedToken = r.Header.Get("Authorization")
		assert.Contains(t, receivedToken, "Bearer ")

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		err = json.Unmarshal(body, &receivedRequest)
		assert.NoError(t, err)

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "pending",
				},
			},
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "test-project",
		MCP: config.MCP{
			Name:        "test-server",
			Title:       "Test Server",
			Description: "A test MCP server",
			Homepage:    "https://example.com",
			Repository: config.MCPRepository{
				URL:    "https://github.com/test/repo",
				Source: "github",
				ID:     "test/repo",
			},
			Packages: []config.MCPPackage{
				{
					RegistryType: "npm",
					Identifier:   "@test/server",
					Transport: config.MCPTransport{
						Type: "stdio",
					},
				},
			},
			Auth: config.MCPAuth{
				Type:  "none",
				Token: "",
			},
		},
	})
	ctx.Version = "1.0.0"

	pipe := &Pipe{registry: srv.URL}
	pipe.authProviderFn = func(_, _, token string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	require.NoError(t, pipe.Publish(ctx))

	expected := apiv0.ServerJSON{
		Schema:      "https://static.modelcontextprotocol.io/schemas/2025-10-17/server.schema.json",
		Name:        "test-server",
		Title:       "Test Server",
		Description: "A test MCP server",
		WebsiteURL:  "https://example.com",
		Version:     "1.0.0",
		Repository: &model.Repository{
			URL:    "https://github.com/test/repo",
			Source: "github",
			ID:     "test/repo",
		},
		Packages: []model.Package{
			{
				RegistryType: "npm",
				Identifier:   "@test/server",
				Version:      "1.0.0",
				Transport: model.Transport{
					Type: "stdio",
				},
			},
		},
	}
	require.Equal(t, expected, receivedRequest)
}

func TestPublishWithTemplates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req apiv0.ServerJSON
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(body, &req))

		expected := apiv0.ServerJSON{
			Schema:      "https://static.modelcontextprotocol.io/schemas/2025-10-17/server.schema.json",
			Name:        "my-test-project",
			Title:       "My-Test-Project v1.2.3",
			Description: "Server for my-test-project",
			Version:     "1.2.3",
			Repository: &model.Repository{
				URL: "https://github.com/user/my-test-project",
				ID:  "user/my-test-project",
			},
			Packages: []model.Package{
				{
					RegistryType: "npm",
					Identifier:   "@my-org/my-test-project",
					Version:      "1.2.3",
					Transport: model.Transport{
						Type: "stdio",
					},
				},
			},
		}
		assert.Equal(t, expected, req)

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "approved",
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "my-test-project",
		MCP: config.MCP{
			Name:        "{{ .ProjectName }}",
			Title:       "{{ .ProjectName | title }} v{{ .Version }}",
			Description: "Server for {{ .ProjectName }}",
			Repository: config.MCPRepository{
				URL: "https://github.com/user/{{ .ProjectName }}",
				ID:  "user/{{ .ProjectName }}",
			},
			Packages: []config.MCPPackage{
				{
					RegistryType: "npm",
					Identifier:   "@my-org/{{ .ProjectName }}",
					Transport: config.MCPTransport{
						Type: "stdio",
					},
				},
			},
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})
	ctx.Version = "1.2.3"

	pipe := &Pipe{registry: srv.URL}
	pipe.authProviderFn = func(_, _, token string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	require.NoError(t, pipe.Publish(ctx))
}

func TestPublishInvalidTemplate(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:  "{{ .InvalidField }",
			Title: "Test",
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})

	pipe := &Pipe{registry: "http://localhost"}
	pipe.authProviderFn = func(_, _, token string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	testlib.RequireTemplateError(t, pipe.Publish(ctx))
}

func TestPublishServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "internal server error")
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:  "test-server",
			Title: "Test Server",
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})
	ctx.Version = "1.0.0"

	pipe := &Pipe{registry: srv.URL}
	pipe.authProviderFn = func(_, _, token string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	err := pipe.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "got status code 500")
	require.Contains(t, err.Error(), "internal server error")
}

func TestPublishBadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error": "invalid server name"}`)
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:  "test-server",
			Title: "Test Server",
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})
	ctx.Version = "1.0.0"

	pipe := &Pipe{registry: srv.URL}
	pipe.authProviderFn = func(_, _, token string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	err := pipe.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "got status code 400")
}

func TestPublishMultiplePackages(t *testing.T) {
	var receivedRequest apiv0.ServerJSON

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(body, &receivedRequest))

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "pending",
				},
			},
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:  "multi-package-server",
			Title: "Multi Package Server",
			Packages: []config.MCPPackage{
				{
					RegistryType: "npm",
					Identifier:   "@test/server-npm",
					Transport: config.MCPTransport{
						Type: "stdio",
					},
				},
				{
					RegistryType: "pypi",
					Identifier:   "test-server-pypi",
					Transport: config.MCPTransport{
						Type: "sse",
					},
				},
				{
					RegistryType: "oci",
					Identifier:   "ghcr.io/test/server",
					Transport: config.MCPTransport{
						Type: "streamable-http",
					},
				},
			},
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})
	ctx.Version = "2.0.0"

	pipe := &Pipe{registry: srv.URL}
	pipe.authProviderFn = func(_, _, token string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	require.NoError(t, pipe.Publish(ctx))

	expected := apiv0.ServerJSON{
		Schema:  "https://static.modelcontextprotocol.io/schemas/2025-10-17/server.schema.json",
		Name:    "multi-package-server",
		Title:   "Multi Package Server",
		Version: "2.0.0",
		Packages: []model.Package{
			{
				RegistryType: "npm",
				Identifier:   "@test/server-npm",
				Version:      "2.0.0",
				Transport: model.Transport{
					Type: "stdio",
				},
			},
			{
				RegistryType: "pypi",
				Identifier:   "test-server-pypi",
				Version:      "2.0.0",
				Transport: model.Transport{
					Type: "sse",
				},
			},
			{
				RegistryType: "oci",
				Identifier:   "ghcr.io/test/server",
				Transport: model.Transport{
					Type: "streamable-http",
				},
			},
		},
	}
	require.Equal(t, expected, receivedRequest)
}

func TestPublishWithRepository(t *testing.T) {
	var receivedRequest apiv0.ServerJSON

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(body, &receivedRequest))

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "pending",
				},
			},
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:  "repo-server",
			Title: "Repo Server",
			Repository: config.MCPRepository{
				URL:       "https://gitlab.com/group/project",
				Source:    "gitlab",
				ID:        "group/project",
				Subfolder: "servers/mcp",
			},
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})
	ctx.Version = "1.5.0"

	pipe := &Pipe{registry: srv.URL}
	pipe.authProviderFn = func(_, _, token string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	require.NoError(t, pipe.Publish(ctx))

	expected := apiv0.ServerJSON{
		Schema:  "https://static.modelcontextprotocol.io/schemas/2025-10-17/server.schema.json",
		Name:    "repo-server",
		Title:   "Repo Server",
		Version: "1.5.0",
		Repository: &model.Repository{
			URL:       "https://gitlab.com/group/project",
			Source:    "gitlab",
			ID:        "group/project",
			Subfolder: "servers/mcp",
		},
		Packages: nil,
	}
	require.Equal(t, expected, receivedRequest)
}

func TestAuthProvider(t *testing.T) {
	t.Run("none auth", func(t *testing.T) {
		provider, err := authProvider("http://registry.test", "none", "")
		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("github auth", func(t *testing.T) {
		provider, err := authProvider("http://registry.test", "github", "test-token")
		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("github-oidc auth", func(t *testing.T) {
		provider, err := authProvider("http://registry.test", "github-oidc", "")
		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("unknown auth method", func(t *testing.T) {
		provider, err := authProvider("http://registry.test", "unknown", "")
		require.Error(t, err)
		require.Nil(t, provider)
		require.Contains(t, err.Error(), "unknown auth method: unknown")
	})
}

func TestNew(t *testing.T) {
	pipe := New()
	require.NotEmpty(t, pipe.registry)
}

func TestPublishAuthLoginError(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:  "test-server",
			Title: "Test Server",
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})
	ctx.Version = "1.0.0"

	pipe := &Pipe{registry: "http://localhost"}
	pipe.authProviderFn = func(_, _, _ string) (auth.Provider, error) {
		return &mockAuthProvider{
			token:    "test-token",
			loginErr: fmt.Errorf("login failed"),
		}, nil
	}
	err := pipe.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not login")
	require.Contains(t, err.Error(), "login failed")
}

func TestPublishAuthProviderError(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:  "test-server",
			Title: "Test Server",
			Auth: config.MCPAuth{
				Type: "invalid",
			},
		},
	})
	ctx.Version = "1.0.0"

	pipe := New()
	err := pipe.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not login")
}

func TestPublishGetTokenError(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:  "test-server",
			Title: "Test Server",
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})
	ctx.Version = "1.0.0"

	pipe := &Pipe{registry: "http://localhost"}
	pipe.authProviderFn = func(_, _, _ string) (auth.Provider, error) {
		return &mockAuthProvider{
			token:       "test-token",
			getTokenErr: fmt.Errorf("token retrieval failed"),
		}, nil
	}
	err := pipe.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not get token")
	require.Contains(t, err.Error(), "token retrieval failed")
}

func TestPublishNoPackages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req apiv0.ServerJSON
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(body, &req))

		assert.Empty(t, req.Packages)

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "pending",
				},
			},
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:     "no-packages-server",
			Title:    "No Packages Server",
			Packages: []config.MCPPackage{},
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})
	ctx.Version = "1.0.0"

	pipe := &Pipe{registry: srv.URL}
	pipe.authProviderFn = func(_, _, token string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	require.NoError(t, pipe.Publish(ctx))
}

func TestPublishInvalidJSON(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		MCP: config.MCP{
			Name:  "test-server",
			Title: "Test Server",
			Auth: config.MCPAuth{
				Type: "none",
			},
		},
	})
	ctx.Version = "1.0.0"

	pipe := &Pipe{registry: "http://invalid-url-that-does-not-exist.local"}
	pipe.authProviderFn = func(_, _, token string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	err := pipe.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not send request")
}

type mockAuthProvider struct {
	token       string
	loginErr    error
	getTokenErr error
}

func (m *mockAuthProvider) GetToken(context.Context) (string, error) {
	return m.token, m.getTokenErr
}

func (m *mockAuthProvider) NeedsLogin() bool {
	return false
}

func (m *mockAuthProvider) Login(context.Context) error {
	return m.loginErr
}

func (m *mockAuthProvider) Name() string {
	return "mock"
}

func TestPublishIntegration(t *testing.T) {
	t.Skip("integration test")

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "goreleaser-mcp",
		MCP: config.MCP{
			Name:        "io.github.goreleaser/mcp",
			Description: "GoReleaser MCP server for build automation",
			Repository: config.MCPRepository{
				Source: "github",
				URL:    "https://github.com/goreleaser/mcp",
			},
			Packages: []config.MCPPackage{
				{
					RegistryType: "oci",
					Identifier:   "ghcr.io/goreleaser/mcp:{{.Version}}",
					Transport: config.MCPTransport{
						Type: "stdio",
					},
				},
				{
					RegistryType: "npm",
					Identifier:   "@goreleaser/mcp",
					Transport: config.MCPTransport{
						Type: "stdio",
					},
				},
			},
			Auth: config.MCPAuth{
				Type:  "github",
				Token: os.Getenv("GITHUB_TOKEN"),
			},
		},
	})
	ctx.Version = "0.1.4"

	pipe := Pipe{
		registry:       "https://staging.registry.modelcontextprotocol.io",
		authProviderFn: authProvider,
	}

	require.NoError(t, pipe.Default(ctx))
	require.NoError(t, pipe.Publish(ctx))
}
