package testlib

import (
	"sync"

	"github.com/ory/dockertest/v3"
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
