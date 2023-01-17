package testlib

import (
	"sync"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
)

var (
	dockerPoolOnce sync.Once
	dockerPool     *dockertest.Pool
)

// MustDockerPool gets a single dockertet.Pool.
func MustDockerPool(f Fataler) *dockertest.Pool {
	dockerPoolOnce.Do(func() {
		pool, err := dockertest.NewPool("")
		if err != nil {
			f.Fatal(err)
		}
		if err := pool.Client.Ping(); err != nil {
			f.Fatal(err)
		}
		dockerPool = pool
	})
	return dockerPool
}

// MustKillContainer kills the given container by name if it exists in the
// current dockertest.Pool.
func MustKillContainer(f Fataler, name string) {
	pool := MustDockerPool(f)
	if trash, ok := pool.ContainerByName(name); ok {
		if err := pool.Purge(trash); err != nil {
			f.Fatal(err)
		}
	}
}

// Fataler interface, can be a log.Default() or testing.TB, for example.
type Fataler interface {
	Fatal(args ...any)
}

// StartRegistry starts a registry with the given name in the given port, and
// sets up its deletion on test.Cleanup.
func StartRegistry(tb testing.TB, name, port string) {
	tb.Helper()

	pool := MustDockerPool(tb)
	MustKillContainer(tb, name)
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:       name,
		Repository: "registry",
		Tag:        "2",
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port("5000/tcp"): {{HostPort: port}},
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	require.NoError(tb, err)

	tb.Cleanup(func() {
		require.NoError(tb, pool.Purge(resource))
	})
}
