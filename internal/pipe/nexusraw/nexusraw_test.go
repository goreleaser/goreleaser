package nexusraw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type components struct {
	Items []item `json:"items"`
}

type item struct {
	Name string `json:"name"`
}

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNoNexusRaw(t *testing.T) {
	testlib.AssertSkipped(t, Pipe{}.Publish(context.New(config.Project{})))
}

func TestUpload(t *testing.T) {
	artifactsFolder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	artifactPath := filepath.Join(artifactsFolder, "artifact.tar.gz")
	artifact2Path := filepath.Join(artifactsFolder, "artifact2.tar.gz")
	directoryPath := filepath.Join(artifactsFolder, "directory")
	artifact3Path := filepath.Join(directoryPath, "binaryArtifact")
	assert.NoError(t, ioutil.WriteFile(artifactPath, []byte("fake\nartifact"), 0744))
	assert.NoError(t, ioutil.WriteFile(artifact2Path, []byte("fake\nartifact\t2"), 0744))
	assert.NoError(t, os.Mkdir(directoryPath, 0744))
	assert.NoError(t, ioutil.WriteFile(artifact3Path, []byte("fake\nartifact\t3"), 0744))
	listen := randomListen(t)
	nexusPassword := startNexus(t, "nexusTestContainer", listen)
	defer stopNexus(t, "nexusTestContainer")
	repositoryName := "goreleaserrawrepo"
	prepareEnv(t, listen, nexusPassword, repositoryName)
	ctx := context.New(config.Project{
		Dist:        artifactsFolder,
		ProjectName: "testupload",
		NexusRaws: []config.NexusRaw{
			{
				URL:        "http://" + listen,
				Repository: repositoryName,
				Directory:  "testDirectory",
				Username:   "admin",
				Password:   nexusPassword,
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact.tar.gz",
		Path: artifactPath,
		Type: artifact.UploadableArchive,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact2.tar.gz",
		Path: artifact2Path,
		Type: artifact.UploadableArchive,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "binaryArtifact",
		Path: artifact3Path,
		Type: artifact.Binary,
	})
	assert.NoError(t, Pipe{}.Publish(ctx))

	resp, err := http.Get(fmt.Sprintf("http://%s/service/rest/v1/components?repository=%s", listen, repositoryName))
	assert.NoError(t, err)

	var cs components
	err = json.NewDecoder(resp.Body).Decode(&cs)
	assert.NoError(t, err)
	assert.Len(t, cs.Items, 2)
	assert.ElementsMatch(t, cs.Items, []item{
		{"testDirectory/artifact.tar.gz"},
		{"testDirectory/artifact2.tar.gz"}},
	)
}

func randomListen(t *testing.T) string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	listener.Close()
	return listener.Addr().String()
}

func startNexus(t *testing.T, containerName, listen string) string {
	volumeFolder, err := ioutil.TempDir("", "nexustestvolumefolder")
	if err != nil {
		t.Fatalf("cannot create folder for nexus volume: %s", err.Error())
	}
	err = os.Chmod(volumeFolder, 0777)
	if err != nil {
		t.Fatalf("cannot modify privileges of the nexus volume folder: %s", err.Error())
	}
	if out, err := exec.Command(
		"docker", "run", "-d", "--rm", "--name", containerName,
		"-p", listen+":8081",
		"-v", fmt.Sprintf("%s:/nexus-data", volumeFolder),
		"sonatype/nexus3",
	).CombinedOutput(); err != nil {
		t.Fatalf("failed to start nexus: %s", string(out))
	}
	var adminPass []byte
	for range time.Tick(time.Second) {
		adminPass, err = ioutil.ReadFile(filepath.Join(volumeFolder, "admin.password"))
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			t.Fatalf("cannot read nexus admin password file: %s", err.Error())
		}
	}
	i := 0
	var resp *http.Response
	for range time.Tick(time.Second) {
		url := fmt.Sprintf("http://%s/service/rest/v1/repositories", listen)
		resp, err = http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			break
		}

		if i > 20 {
			t.Fatalf("stopping: cannot list nexus repositories (url) '%s': %s", url, err.Error())
		}
		i++
	}
	j := 0
	for range time.Tick(time.Second) {
		if resp.StatusCode == http.StatusOK {
			break
		}
		if j > 5 {
			t.Fatalf("stopping: wrong status code: %s", resp.Status)
		}
		j++
	}
	return string(adminPass)
}

func prepareEnv(t *testing.T, listen, adminPass, repository string) {
	scriptName := "addRepo"
	uploadScriptToNexus(t, listen, adminPass, repository, scriptName)
	executeScript(t, listen, adminPass, scriptName)
}

func uploadScriptToNexus(t *testing.T, listen, adminPass, repository, scriptName string) {
	url := fmt.Sprintf("http://%s/service/rest/v1/script", listen)
	script := "def rawStore = blobStore.createFileBlobStore('raw', 'raw')\\nrepository.createRawHosted('" + repository + "', rawStore.name)"
	postData := []byte(`{"name":"` + scriptName + `","content":"` + script + `","type":"groovy"}"`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postData))
	if err != nil {
		t.Fatalf("cannot create post request to add script: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", adminPass)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("cannot upload script to nexus: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("wrong status after uploading script to nexus: %s", resp.Status)
	}
}

func executeScript(t *testing.T, listen, adminPass, scriptName string) {
	url := fmt.Sprintf("http://%s/service/rest/v1/script/%s/run", listen, scriptName)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		t.Fatalf("cannot create post request to execute script: %s", err.Error())
	}
	req.Header.Set("Content-Type", "text/plain")
	req.SetBasicAuth("admin", adminPass)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("cannot execute script: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("wrong status after executing script: %s", resp.Status)
	}
}

func stopNexus(t *testing.T, containerName string) {
	if out, err := exec.Command("docker", "stop", containerName).CombinedOutput(); err != nil {
		t.Fatalf("failed to stop nexus: %s", string(out))
	}
}
