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

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Artifactory{
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
		"ARTIFACTORY_PRODUCTION-US_SECRET": "deployuser-secret",
		"ARTIFACTORY_PRODUCTION-EU_SECRET": "productionuser-apikey",
	}
	ctx.Publish = true
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
		Artifactories: []config.Artifactory{
			{
				Name:     "production",
				Mode:     "archive",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Version }}/", server.URL),
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Publish = true
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

	// Dummy artifactories
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "example-repo-local",
			"path" : "/goreleaser/bin.tar.gz",
			"created" : "2017-12-02T19:30:45.436Z",
			"createdBy" : "deployuser",
			"downloadUri" : "http://127.0.0.1:56563/example-repo-local/goreleaser/bin.tar.gz",
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
			"uri" : "http://127.0.0.1:56563/example-repo-local/goreleaser/bin.tar.gz"
		  }`)
	})
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.deb", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "example-repo-local",
			"path" : "goreleaser/bin.deb",
			"created" : "2017-12-02T19:30:46.436Z",
			"createdBy" : "deployuser",
			"downloadUri" : "http://127.0.0.1:56563/example-repo-local/goreleaser/bin.deb",
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
			"uri" : "http://127.0.0.1:56563/example-repo-local/goreleaser/bin.deb"
		  }`)
	})

	assert.NoError(t, Pipe{}.Run(ctx))
}

func TestRunPipe_ArtifactoryDown(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(t, err)

	var ctx = context.New(config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		Artifactories: []config.Artifactory{
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
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Publish = true
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})

	assert.EqualError(t, Pipe{}.Run(ctx), `artifactory: upload failed: Put http://localhost:1234/example-repo-local/goreleaser/2.0.0/bin.tar.gz: dial tcp 127.0.0.1:1234: getsockopt: connection refused`)
}

func TestRunPipe_TargetTemplateError(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	var binPath = filepath.Join(dist, "mybin", "mybin")

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Artifactory{
			{
				Name: "production",
				Mode: "binary",
				// This template is not correct and should fail
				Target:   "http://storage.company.com/example-repo-local/{{ .ProjectName /{{ .Version }}/",
				Username: "deployuser",
			},
		},
	})
	ctx.Publish = true
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	assert.EqualError(t, Pipe{}.Run(ctx), `artifactory: error while building the target url: template: mybin:1: unexpected "/" in operand`)
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

	// Dummy artifactories
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{
			"errors" : [ {
			  "status" : 401,
			  "message" : "Bad credentials"
			} ]
		  }`)
	})

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Artifactory{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "deployuser",
			},
		},
	})
	ctx.Publish = true
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
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
	assert.Contains(t, err.Error(), "Bad credentials")
}

func TestRunPipe_UnparsableErrorResponse(t *testing.T) {
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

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `...{
			"errors" : [ {
			 ...
			} ]
		  }`)
	})

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Artifactory{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "deployuser",
			},
		},
	})
	ctx.Publish = true
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	assert.EqualError(t, Pipe{}.Run(ctx), `artifactory: upload failed: invalid character '.' looking for beginning of value`)
}

func TestRunPipe_UnparsableResponse(t *testing.T) {
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

	// Dummy artifactory with invalid JSON
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `invalid-json{
			"repo" : "example-repo-local",
			...
		  }`)
	})

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Artifactory{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "deployuser",
			},
		},
	})
	ctx.Publish = true
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	assert.EqualError(t, Pipe{}.Run(ctx), `artifactory: upload failed: invalid character 'i' looking for beginning of value`)
}

func TestRunPipe_FileNotFound(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        "archivetest/dist",
		Artifactories: []config.Artifactory{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	})
	ctx.Publish = true
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
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
		Artifactories: []config.Artifactory{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	})
	ctx.Publish = true
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	assert.EqualError(t, Pipe{}.Run(ctx), `artifactory: upload failed: parse ://artifacts.company.com/example-repo-local/mybin/darwin/amd64/mybin: missing protocol scheme`)
}

func TestRunPipe_SkipWhenPublishFalse(t *testing.T) {
	var ctx = context.New(config.Project{
		Artifactories: []config.Artifactory{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}

	err := Pipe{}.Run(ctx)
	assert.True(t, pipeline.IsSkip(err))
	assert.Equal(t, err.Error(), "--skip-publish is set")
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
		Artifactories: []config.Artifactory{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Publish = true
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Dir(binPath),
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	assert.EqualError(t, Pipe{}.Run(ctx), `artifactory: upload failed: the asset to upload can't be a directory`)
}

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNoArtifactories(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{}))))
}

func TestArtifactoriesWithoutTarget(t *testing.T) {
	var ctx = &context.Context{
		Env: map[string]string{
			"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			Artifactories: []config.Artifactory{
				{
					Name:     "production",
					Username: "deployuser",
				},
			},
		},
	}

	assert.True(t, pipeline.IsSkip(Pipe{}.Run(ctx)))
}

func TestArtifactoriesWithoutUsername(t *testing.T) {
	var ctx = &context.Context{
		Env: map[string]string{
			"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			Artifactories: []config.Artifactory{
				{
					Name:   "production",
					Target: "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				},
			},
		},
	}

	assert.True(t, pipeline.IsSkip(Pipe{}.Run(ctx)))
}

func TestArtifactoriesWithoutName(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		Artifactories: []config.Artifactory{
			{
				Username: "deployuser",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
			},
		},
	}))))
}

func TestArtifactoriesWithoutSecret(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		Artifactories: []config.Artifactory{
			{
				Name:     "production",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	}))))
}

func TestArtifactoriesWithInvalidMode(t *testing.T) {
	var ctx = &context.Context{
		Publish: true,
		Env: map[string]string{
			"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			Artifactories: []config.Artifactory{
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
			Artifactories: []config.Artifactory{
				{
					Name:     "production",
					Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
					Username: "deployuser",
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.Artifactories, 1)
	var artifactory = ctx.Config.Artifactories[0]
	assert.Equal(t, "archive", artifactory.Mode)
}

func TestDefaultNoArtifactories(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Artifactories: []config.Artifactory{},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Empty(t, ctx.Config.Artifactories)
}

func TestDefaultSet(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Artifactories: []config.Artifactory{
				{
					Mode: "custom",
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.Artifactories, 1)
	var artifactory = ctx.Config.Artifactories[0]
	assert.Equal(t, "custom", artifactory.Mode)
}
