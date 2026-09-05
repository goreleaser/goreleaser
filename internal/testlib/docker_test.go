package testlib

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKillContainer(t *testing.T) {
	for _, tc := range []struct {
		name        string
		neighbor    string
		exists      bool
		deleteError bool
		want        []string
	}{
		{name: "alt_registry", neighbor: "alt_registry-v2", exists: true, want: []string{"owned"}},
		{name: "registry.test", neighbor: "registryXtest", exists: true, want: []string{"owned"}},
		{name: "registry", neighbor: "registry-v2"},
		{name: "delete-error", neighbor: "delete-error-v2", exists: true, deleteError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			containers := []docker.Container{{ID: "neighbor", Name: "/" + tc.neighbor}}
			if tc.exists {
				containers = append(containers, docker.Container{ID: "owned", Name: "/" + tc.name})
			}
			deleted := make(chan string, len(containers))
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/containers/json") {
					var filters map[string][]string
					if !assert.NoError(t, json.Unmarshal([]byte(r.URL.Query().Get("filters")), &filters)) ||
						!assert.Len(t, filters["name"], 1) {
						http.Error(w, "invalid name filter", http.StatusBadRequest)
						return
					}
					filter, err := regexp.Compile(filters["name"][0])
					if !assert.NoError(t, err) {
						http.Error(w, "invalid name filter", http.StatusBadRequest)
						return
					}
					var matches []docker.APIContainers
					for _, container := range containers {
						if filter.MatchString(container.Name) {
							matches = append(matches, docker.APIContainers{
								ID: container.ID, Names: []string{container.Name},
							})
						}
					}
					assert.NoError(t, json.NewEncoder(w).Encode(matches))
					return
				}
				for _, container := range containers {
					if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/containers/"+container.ID+"/json") {
						assert.NoError(t, json.NewEncoder(w).Encode(container))
						return
					}
					if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/containers/"+container.ID) {
						if tc.deleteError {
							http.Error(w, "could not remove fixture", http.StatusInternalServerError)
							return
						}
						deleted <- container.ID
						w.WriteHeader(http.StatusNoContent)
						return
					}
				}
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
			}))
			t.Cleanup(srv.Close)
			pool, err := dockertest.NewPool(srv.URL)
			require.NoError(t, err)

			var fatalMessages []string
			killContainer(fatalFunc(func(args ...any) {
				fatalMessages = append(fatalMessages, fmt.Sprint(args...))
			}), pool, tc.name)
			if tc.deleteError {
				require.Len(t, fatalMessages, 1)
				require.Contains(t, fatalMessages[0], "could not remove fixture")
			} else {
				require.Empty(t, fatalMessages)
			}
			srv.Close()
			close(deleted)
			var removed []string
			for id := range deleted {
				removed = append(removed, id)
			}
			require.Equal(t, tc.want, removed)
		})
	}
}

type fatalFunc func(...any)

func (f fatalFunc) Fatal(args ...any) {
	f(args...)
}
