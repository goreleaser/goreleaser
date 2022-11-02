package ko

import (
	stdcontext "context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/auth/htpasswd"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

var (
	dockerReg  string
	_, b, _, _ = runtime.Caller(0)
	RootDir    = filepath.Join(filepath.Dir(b), "../../..")
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipe(t *testing.T) {
	startRegistryServer(t)

	ctx := &context.Context{
		Parallelism: 1,
		Config: config.Project{
			Builds: []config.Build{
				{
					ID: "foo",
					BuildDetails: config.BuildDetails{
						Ldflags: []string{"-s", "-w"},
						Flags:   []string{"-tags", "netgo"},
						Env:     []string{"GOCACHE=/workspace/.gocache"},
					},
				},
			},
			Kos: []config.Ko{
				{
					ID:         "default",
					Build:      "foo",
					WorkingDir: RootDir,
					BaseImage:  "cgr.dev/chainguard/static",
					Repository: fmt.Sprintf("%s/goreleaser", dockerReg),
					Platforms:  []string{"linux/amd64"},
					Tags:       []string{"latest"},
					Push:       true,
				},
			},
		},
	}

	err := Pipe{}.Run(ctx)
	require.NoError(t, err)
}

func startRegistryServer(t *testing.T) {
	t.Helper()
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 10*time.Second)
	// Registry config
	config := &configuration.Configuration{}
	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %s", err)
	}
	dockerReg = fmt.Sprintf("localhost:%d", port)
	config.HTTP.Addr = fmt.Sprintf("127.0.0.1:%d", port)
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}
	registry, err := registry.NewRegistry(ctx, config)
	if err != nil {
		t.Fatalf("failed to create docker registry: %v", err)
	}

	c := setupSignalHandler()

	// Start Docker registry
	eg, ctx := errgroup.WithContext(ctx)
	var errchan chan error
	eg.Go(func() error {
		// run registry server
		go func() {
			errchan <- registry.ListenAndServe()
		}()

		select {
		case err := <-errchan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})

	// Wait for all HTTP fetches to complete.
	if err := eg.Wait(); err == nil {
		fmt.Println("registry server started at:", dockerReg)
	}

	go func() {
		<-c
		cancel()
		fmt.Println("shutting down the registry server")
		<-c // wait for the second signal, then exit immediately
		os.Exit(1)
	}()
}

func setupSignalHandler() chan os.Signal {
	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	return signalChan
}
