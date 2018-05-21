package httpupload

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/stretchr/testify/assert"
)

var (
	// mux is the HTTP request multiplexer used with the test server.
	mux *http.ServeMux

	// server is a test HTTP server used to provide mock API responses.
	server *httptest.Server
)

func setup() {
	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
}

// teardown closes the test HTTP server.
func teardown() {
	server.Close()
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func testHeader(t *testing.T, r *http.Request, header string, want string) {
	if got := r.Header.Get(header); got != want {
		t.Errorf("Header.Get(%q) returned %q, want %q", header, got, want)
	}
}

// TODO: improve all tests bellow by checking wether the mocked handlers
// were called or not.

func TestRunPipe_ModeBinary(t *testing.T) {
	setup()
	defer teardown()

	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	err = ioutil.WriteFile(binPath, d1, 0666)
	assert.NoError(t, err)

	// Dummy http server
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/example-repo-local/mybin/linux/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/production-repo-remote/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "productionuser" with secret "productionuser-apikey"
		testHeader(t, r, "Authorization", "Basic cHJvZHVjdGlvbnVzZXI6cHJvZHVjdGlvbnVzZXItYXBpa2V5")

		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/production-repo-remote/mybin/linux/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "productionuser" with secret "productionuser-apikey"
		testHeader(t, r, "Authorization", "Basic cHJvZHVjdGlvbnVzZXI6cHJvZHVjdGlvbnVzZXItYXBpa2V5")

		w.WriteHeader(http.StatusCreated)
	})

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		HTTPUploads: []config.HTTPUpload{
			{
				Name:     "production-us",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "deployuser",
			},
			{
				Name:     "production-eu",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/production-repo-remote/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "productionuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"HTTP_UPLOAD_PRODUCTION-US_SECRET": "deployuser-secret",
		"HTTP_UPLOAD_PRODUCTION-EU_SECRET": "productionuser-apikey",
	}
	for _, goos := range []string{"linux", "darwin"} {
		ctx.Artifacts.Add(artifact.Artifact{
			Name:   "mybin",
			Path:   binPath,
			Goarch: "amd64",
			Goos:   goos,
			Type:   artifact.UploadableBinary,
		})
	}

	assert.NoError(t, Pipe{}.Run(ctx))
}

func TestRunPipe_ModeArchive(t *testing.T) {
	setup()
	defer teardown()

	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(t, err)
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	assert.NoError(t, err)

	var ctx = context.New(config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		HTTPUploads: []config.HTTPUpload{
			{
				Name:     "production",
				Mode:     "archive",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Version }}/", server.URL),
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Version = "1.0.0"
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debfile.Name(),
	})

	var uploads sync.Map

	// Dummy http server
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		uploads.Store("targz", true)
	})
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.deb", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		uploads.Store("deb", true)
	})

	assert.NoError(t, Pipe{}.Run(ctx))
	_, ok := uploads.Load("targz")
	assert.True(t, ok, "tar.gz file was not uploaded")
	_, ok = uploads.Load("deb")
	assert.True(t, ok, "deb file was not uploaded")
}

func TestRunPipe_ArtifactoryDown(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(t, err)

	var ctx = context.New(config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		HTTPUploads: []config.HTTPUpload{
			{
				Name:     "production",
				Mode:     "archive",
				Target:   "http://localhost:1234/example-repo-local/{{ .ProjectName }}/{{ .Version }}/",
				Username: "deployuser",
			},
		},
	})
	ctx.Version = "2.0.0"
	ctx.Env = map[string]string{
		"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestRunPipe_TargetTemplateError(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	var binPath = filepath.Join(dist, "mybin", "mybin")

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		HTTPUploads: []config.HTTPUpload{
			{
				Name: "production",
				Mode: "binary",
				// This template is not correct and should fail
				Target:   "http://storage.company.com/example-repo-local/{{ .ProjectName /{{ .Version }}/",
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})
	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `httpupload: error while building the target url: template: mybin:1: unexpected "/" in operand`)
}

func TestRunPipe_BadCredentials(t *testing.T) {
	setup()
	defer teardown()

	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	err = ioutil.WriteFile(binPath, d1, 0666)
	assert.NoError(t, err)

	// Dummy http server
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusUnauthorized)
	})

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		HTTPUploads: []config.HTTPUpload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unauthorized")
}

func TestRunPipe_FileNotFound(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        "archivetest/dist",
		HTTPUploads: []config.HTTPUpload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   "archivetest/dist/mybin/mybin",
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	assert.EqualError(t, Pipe{}.Run(ctx), `open archivetest/dist/mybin/mybin: no such file or directory`)
}

func TestRunPipe_UnparsableTarget(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	err = ioutil.WriteFile(binPath, d1, 0666)
	assert.NoError(t, err)

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		HTTPUploads: []config.HTTPUpload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	assert.EqualError(t, Pipe{}.Run(ctx), `httpupload: upload failed: parse ://artifacts.company.com/example-repo-local/mybin/darwin/amd64/mybin: missing protocol scheme`)
}

func TestRunPipe_SkipWhenPublishFalse(t *testing.T) {
	var ctx = context.New(config.Project{
		HTTPUploads: []config.HTTPUpload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.SkipPublish = true

	err := Pipe{}.Run(ctx)
	assert.True(t, pipeline.IsSkip(err))
	assert.EqualError(t, err, pipeline.ErrSkipPublishEnabled.Error())
}

func TestRunPipe_DirUpload(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin")

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		HTTPUploads: []config.HTTPUpload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Dir(binPath),
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	assert.EqualError(t, Pipe{}.Run(ctx), `httpupload: upload failed: the asset to upload can't be a directory`)
}

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNoHTTPUploads(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{}))))
}

func TestHTTPUploadsWithoutTarget(t *testing.T) {
	var ctx = &context.Context{
		Env: map[string]string{
			"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			HTTPUploads: []config.HTTPUpload{
				{
					Name:     "production",
					Username: "deployuser",
				},
			},
		},
	}

	assert.True(t, pipeline.IsSkip(Pipe{}.Run(ctx)))
}

func TestHTTPUploadsWithoutUsername(t *testing.T) {
	var ctx = &context.Context{
		Env: map[string]string{
			"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			HTTPUploads: []config.HTTPUpload{
				{
					Name:   "production",
					Target: "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				},
			},
		},
	}

	assert.True(t, pipeline.IsSkip(Pipe{}.Run(ctx)))
}

func TestHTTPUploadsWithoutName(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		HTTPUploads: []config.HTTPUpload{
			{
				Username: "deployuser",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
			},
		},
	}))))
}

func TestHTTPUploadsWithoutSecret(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		HTTPUploads: []config.HTTPUpload{
			{
				Name:     "production",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	}))))
}

func TestHTTPUploadsWithInvalidMode(t *testing.T) {
	var ctx = &context.Context{
		Env: map[string]string{
			"HTTP_UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			HTTPUploads: []config.HTTPUpload{
				{
					Name:     "production",
					Mode:     "does-not-exists",
					Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
					Username: "deployuser",
				},
			},
		},
	}
	assert.Error(t, Pipe{}.Run(ctx))
}

func TestDefault(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			HTTPUploads: []config.HTTPUpload{
				{
					Name:     "production",
					Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
					Username: "deployuser",
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.HTTPUploads, 1)
	var httpupload = ctx.Config.HTTPUploads[0]
	assert.Equal(t, "archive", httpupload.Mode)
}

func TestDefaultNoHTTPUploads(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			HTTPUploads: []config.HTTPUpload{},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Empty(t, ctx.Config.HTTPUploads)
}

func TestDefaultSet(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			HTTPUploads: []config.HTTPUpload{
				{
					Mode: "custom",
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.HTTPUploads, 1)
	var httpupload = ctx.Config.HTTPUploads[0]
	assert.Equal(t, "custom", httpupload.Mode)
}
