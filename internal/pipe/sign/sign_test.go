package sign

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
)

var originKeyring = "testdata/gnupg"
var keyring string

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	keyring = fmt.Sprintf("/tmp/gorel_gpg_test.%d", rand.Int())
	fmt.Println("copying", originKeyring, "to", keyring)
	if err := exec.Command("cp", "-Rf", originKeyring, keyring).Run(); err != nil {
		fmt.Printf("failed to copy %s to %s: %s", originKeyring, keyring, err)
		os.Exit(1)
	}
	defer os.RemoveAll(keyring)
	os.Exit(m.Run())
}

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestSignDefault(t *testing.T) {
	ctx := &context.Context{}
	err := Pipe{}.Default(ctx)
	assert.NoError(t, err)
	assert.Equal(t, ctx.Config.Sign.Cmd, "gpg")
	assert.Equal(t, ctx.Config.Sign.Signature, "${artifact}.sig")
	assert.Equal(t, ctx.Config.Sign.Args, []string{"--output", "$signature", "--detach-sig", "$artifact"})
	assert.Equal(t, ctx.Config.Sign.Artifacts, "none")
}

func TestSignDisabled(t *testing.T) {
	ctx := &context.Context{}
	ctx.Config.Sign.Artifacts = "none"
	err := Pipe{}.Run(ctx)
	assert.EqualError(t, err, "artifact signing is disabled")
}

func TestSignSkipped(t *testing.T) {
	ctx := &context.Context{}
	ctx.SkipSign = true
	err := Pipe{}.Run(ctx)
	assert.EqualError(t, err, "artifact signing is disabled")
}

func TestSignInvalidArtifacts(t *testing.T) {
	ctx := &context.Context{}
	ctx.Config.Sign.Artifacts = "foo"
	err := Pipe{}.Run(ctx)
	assert.EqualError(t, err, "invalid list of artifacts to sign: foo")
}

func TestSignArtifacts(t *testing.T) {
	tests := []struct {
		desc       string
		ctx        *context.Context
		signatures []string
	}{
		{
			desc: "sign all artifacts",
			ctx: context.New(
				config.Project{
					Sign: config.Sign{Artifacts: "all"},
				},
			),
			signatures: []string{"artifact1.sig", "artifact2.sig", "checksum.sig"},
		},
		{
			desc: "sign only checksums",
			ctx: context.New(
				config.Project{
					Sign: config.Sign{Artifacts: "checksum"},
				},
			),
			signatures: []string{"checksum.sig"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(tt *testing.T) {
			testSign(tt, test.ctx, test.signatures)
		})
	}
}

const user = "nopass"

func testSign(t *testing.T, ctx *context.Context, signatures []string) {
	// create temp dir for file and signature
	tmpdir, err := ioutil.TempDir("", "goreleaser")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	ctx.Config.Dist = tmpdir

	// create some fake artifacts
	var artifacts = []string{"artifact1", "artifact2", "checksum"}
	for _, f := range artifacts {
		file := filepath.Join(tmpdir, f)
		assert.NoError(t, ioutil.WriteFile(file, []byte("foo"), 0644))
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name: "artifact1",
		Path: filepath.Join(tmpdir, "artifact1"),
		Type: artifact.UploadableArchive,
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Name: "artifact2",
		Path: filepath.Join(tmpdir, "artifact2"),
		Type: artifact.UploadableArchive,
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Name: "checksum",
		Path: filepath.Join(tmpdir, "checksum"),
		Type: artifact.Checksum,
	})

	// configure the pipeline
	// make sure we are using the test keyring
	assert.NoError(t, Pipe{}.Default(ctx))
	ctx.Config.Sign.Args = append([]string{"--homedir", keyring}, ctx.Config.Sign.Args...)

	// run the pipeline
	assert.NoError(t, Pipe{}.Run(ctx))

	// verify that only the artifacts and the signatures are in the dist dir
	files, err := ioutil.ReadDir(tmpdir)
	assert.NoError(t, err)
	gotFiles := []string{}
	for _, f := range files {
		gotFiles = append(gotFiles, f.Name())
	}
	wantFiles := append(artifacts, signatures...)
	sort.Strings(wantFiles)
	assert.Equal(t, wantFiles, gotFiles)

	// verify the signatures
	for _, sig := range signatures {
		verifySignature(t, ctx, sig)
	}

	var signArtifacts []string
	for _, sig := range ctx.Artifacts.Filter(artifact.ByType(artifact.Signature)).List() {
		signArtifacts = append(signArtifacts, sig.Name)
	}
	// check signature is an artifact
	assert.Equal(t, signArtifacts, signatures)
}

func verifySignature(t *testing.T, ctx *context.Context, sig string) {
	artifact := sig[:len(sig)-len(".sig")]

	// verify signature was made with key for usesr 'nopass'
	cmd := exec.Command("gpg", "--homedir", keyring, "--verify", filepath.Join(ctx.Config.Dist, sig), filepath.Join(ctx.Config.Dist, artifact))
	out, err := cmd.CombinedOutput()
	assert.NoError(t, err)

	// check if the signature matches the user we expect to do this properly we
	// might need to have either separate keyrings or export the key from the
	// keyring before we do the verification. For now we punt and look in the
	// output.
	if !bytes.Contains(out, []byte(user)) {
		t.Fatalf("signature is not from %s", user)
	}
}
