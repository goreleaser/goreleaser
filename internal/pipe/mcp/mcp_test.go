package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/modelcontextprotocol/registry/cmd/publisher/auth"
	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"github.com/modelcontextprotocol/registry/pkg/model"
	"github.com/stretchr/testify/require"
)

type mockAuthProvider struct {
	token       string
	loginErr    error
	getTokenErr error
}

func (m *mockAuthProvider) GetToken(ctx context.Context) (string, error) {
	return m.token, m.getTokenErr
}

func (m *mockAuthProvider) NeedsLogin() bool {
	return false
}

func (m *mockAuthProvider) Login(ctx context.Context) error {
	return m.loginErr
}

func (m *mockAuthProvider) Name() string {
	return "mock"
}

func TestStringer(t *testing.T) {
	require.Equal(t, "mcp", Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		skips.Set(ctx, skips.MCP)
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.False(t, Pipe{}.Skip(ctx))
	})
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

	t.Run("github auth without token", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			MCP: config.MCP{
				Name: "test-server",
				Auth: config.MCPAuth{
					Type: "github",
				},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, "{{ .Env.GITHUB_TOKEN }}", ctx.Config.MCP.Auth.Token)
	})

	t.Run("github auth with token", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			MCP: config.MCP{
				Name: "test-server",
				Auth: config.MCPAuth{
					Type:  "github",
					Token: "custom-token",
				},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, "custom-token", ctx.Config.MCP.Auth.Token)
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
		require.Equal(t, "", ctx.Config.MCP.Auth.Token)
	})
}

func TestPublishSuccess(t *testing.T) {
	var receivedRequest apiv0.ServerJSON
	var receivedToken string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v0/publish", r.URL.Path)
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		receivedToken = r.Header.Get("Authorization")
		require.Contains(t, receivedToken, "Bearer ")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		err = json.Unmarshal(body, &receivedRequest)
		require.NoError(t, err)

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "pending",
				},
			},
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
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

	pipe := &Pipe{registry: srv.URL + "/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
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
		require.NoError(t, err)
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

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
		require.Equal(t, expected, req)

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "approved",
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
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

	pipe := &Pipe{registry: srv.URL + "/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
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

	pipe := &Pipe{registry: "http://localhost/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	testlib.RequireTemplateError(t, pipe.Publish(ctx))
}

func TestPublishServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
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

	pipe := &Pipe{registry: srv.URL + "/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	err := pipe.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "got status code 500")
	require.Contains(t, err.Error(), "internal server error")
}

func TestPublishBadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid server name"}`))
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

	pipe := &Pipe{registry: srv.URL + "/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
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
		require.NoError(t, err)
		err = json.Unmarshal(body, &receivedRequest)
		require.NoError(t, err)

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "pending",
				},
			},
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
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

	pipe := &Pipe{registry: srv.URL + "/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	require.NoError(t, pipe.Publish(ctx))

	expected := apiv0.ServerJSON{
		Schema:     "https://static.modelcontextprotocol.io/schemas/2025-10-17/server.schema.json",
		Name:       "multi-package-server",
		Title:      "Multi Package Server",
		Version:    "2.0.0",
		Repository: &model.Repository{},
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
				Version:      "2.0.0",
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
		require.NoError(t, err)
		err = json.Unmarshal(body, &receivedRequest)
		require.NoError(t, err)

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "pending",
				},
			},
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
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

	pipe := &Pipe{registry: srv.URL + "/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
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
	require.EqualValues(t, expected, receivedRequest)
}

func TestAuthProvider(t *testing.T) {
	t.Run("none auth", func(t *testing.T) {
		provider, err := authProvider("none", "", "http://registry.test")
		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("github auth", func(t *testing.T) {
		provider, err := authProvider("github", "test-token", "http://registry.test")
		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("github-oidc auth", func(t *testing.T) {
		provider, err := authProvider("github-oidc", "", "http://registry.test")
		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("unknown auth method", func(t *testing.T) {
		provider, err := authProvider("unknown", "", "http://registry.test")
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

	pipe := &Pipe{registry: "http://localhost/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
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

	pipe := &Pipe{registry: "http://localhost/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
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
		require.NoError(t, err)
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		require.Len(t, req.Packages, 0)

		response := apiv0.ServerResponse{
			Meta: apiv0.ResponseMeta{
				Official: &apiv0.RegistryExtensions{
					Status: "pending",
				},
			},
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
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

	pipe := &Pipe{registry: srv.URL + "/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
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

	pipe := &Pipe{registry: "http://invalid-url-that-does-not-exist.local/"}
	pipe.authProvider = func(method, token, registryURL string) (auth.Provider, error) {
		return &mockAuthProvider{token: "test-token"}, nil
	}
	err := pipe.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not send request")
}
