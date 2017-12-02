package artifactory

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
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

func TestRunPipe(t *testing.T) {
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

	// Dummy artifactories
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "example-repo-local",
			"path" : "/mybin/darwin/amd64/mybin",
			"created" : "2017-12-02T19:30:45.436Z",
			"createdBy" : "deployuser",
			"downloadUri" : "http://127.0.0.1:56563/example-repo-local/mybin/darwin/amd64/mybin",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/example-repo-local/mybin/darwin/amd64/mybin"
		  }`)
	})
	mux.HandleFunc("/example-repo-local/mybin/linux/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "example-repo-local",
			"path" : "mybin/linux/amd64/mybin",
			"created" : "2017-12-02T19:30:46.436Z",
			"createdBy" : "deployuser",
			"downloadUri" : "http://127.0.0.1:56563/example-repo-local/mybin/linux/amd64/mybin",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/example-repo-local/mybin/linux/amd64/mybin"
		  }`)
	})
	mux.HandleFunc("/production-repo-remote/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "productionuser" with secret "productionuser-apikey"
		testHeader(t, r, "Authorization", "Basic cHJvZHVjdGlvbnVzZXI6cHJvZHVjdGlvbnVzZXItYXBpa2V5")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "production-repo-remote",
			"path" : "mybin/darwin/amd64/mybin",
			"created" : "2017-12-02T19:30:46.436Z",
			"createdBy" : "productionuser",
			"downloadUri" : "http://127.0.0.1:56563/production-repo-remote/mybin/darwin/amd64/mybin",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/production-repo-remote/mybin/darwin/amd64/mybin"
		  }`)
	})
	mux.HandleFunc("/production-repo-remote/mybin/linux/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "productionuser" with secret "productionuser-apikey"
		testHeader(t, r, "Authorization", "Basic cHJvZHVjdGlvbnVzZXI6cHJvZHVjdGlvbnVzZXItYXBpa2V5")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "production-repo-remote",
			"path" : "mybin/linux/amd64/mybin",
			"created" : "2017-12-02T19:30:46.436Z",
			"createdBy" : "productionuser",
			"downloadUri" : "http://127.0.0.1:56563/production-repo-remote/mybin/linux/amd64/mybin",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/production-repo-remote/mybin/linux/amd64/mybin"
		  }`)
	})

	// Set secrets for artifactory instances
	os.Setenv("ARTIFACTORY_0_SECRET", "deployuser-secret")
	defer os.Unsetenv("ARTIFACTORY_0_SECRET")
	os.Setenv("ARTIFACTORY_1_SECRET", "productionuser-apikey")
	defer os.Unsetenv("ARTIFACTORY_1_SECRET")

	var ctx = &context.Context{
		Version: "1.0.0",
		Publish: true,
		Config: config.Project{
			ProjectName: "mybin",
			Dist:        dist,
			Builds: []config.Build{
				{
					Env:    []string{"CGO_ENABLED=0"},
					Goos:   []string{"linux", "darwin"},
					Goarch: []string{"amd64"},
				},
			},
			Artifactories: []config.Artifactory{
				{
					Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
					Username: "deployuser",
				},
				{
					Target:   fmt.Sprintf("%s/production-repo-remote/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
					Username: "productionuser",
				},
			},
		},
	}
	for _, plat := range []string{"linuxamd64", "linux386", "darwinamd64"} {
		ctx.AddBinary(plat, "mybin", "mybin", binPath)
	}
	ctx.Parallelism = 4

	assert.NoError(t, Pipe{}.Run(ctx))
}

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestNoArtifactories(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{}))))
}

func TestNoArtifactoriesWithoutTarget(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		Artifactories: []config.Artifactory{
			{
				Username: "deployuser",
			},
		},
	}))))
}

func TestNoArtifactoriesWithoutUsername(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		Artifactories: []config.Artifactory{
			{
				Target: "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
			},
		},
	}))))
}

func TestNoArtifactoriesWithoutSecret(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		Artifactories: []config.Artifactory{
			{
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	}))))
}
