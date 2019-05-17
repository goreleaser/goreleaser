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
		desc           string
		ctx            *context.Context
		signaturePaths []string
		signatureNames []string
	}{
		{
			desc: "sign all artifacts",
			ctx: context.New(
				config.Project{
					Sign: config.Sign{Artifacts: "all"},
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "linux_amd64/artifact4.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "artifact4_1.0.0_linux_amd64.sig"},
		},
		{
			desc: "sign only checksums",
			ctx: context.New(
				config.Project{
					Sign: config.Sign{Artifacts: "checksum"},
				},
			),
			signaturePaths: []string{"checksum.sig"},
			signatureNames: []string{"checksum.sig"},
		},
		{
			desc: "sign all artifacts with env",
			ctx: context.New(
				config.Project{
					Sign: config.Sign{
						Artifacts: "all",
						Args: []string{
							"-u",
							"${TEST_USER}",
							"--output",
							"${signature}",
							"--detach-sign",
							"${artifact}",
						},
					},
					Env: []string{
						fmt.Sprintf("TEST_USER=%s", user),
					},
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "linux_amd64/artifact4.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "artifact4_1.0.0_linux_amd64.sig"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(tt *testing.T) {
			testSign(tt, test.ctx, test.signaturePaths, test.signatureNames)
		})
	}
}

const user = "nopass"

func testSign(t *testing.T, ctx *context.Context, signaturePaths []string, signatureNames []string) {
	// create temp dir for file and signature
	tmpdir, err := ioutil.TempDir("", "goreleaser")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	ctx.Config.Dist = tmpdir

	// create some fake artifacts
	var artifacts = []string{"artifact1", "artifact2", "artifact3", "checksum"}
	os.Mkdir(filepath.Join(tmpdir, "linux_amd64"), os.ModePerm)
	for _, f := range artifacts {
		file := filepath.Join(tmpdir, f)
		assert.NoError(t, ioutil.WriteFile(file, []byte("foo"), 0644))
	}
	assert.NoError(t, ioutil.WriteFile(filepath.Join(tmpdir, "linux_amd64", "artifact4"), []byte("foo"), 0644))
	artifacts = append(artifacts, "linux_amd64/artifact4")
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
		Name: "artifact3_1.0.0_linux_amd64",
		Path: filepath.Join(tmpdir, "artifact3"),
		Type: artifact.UploadableBinary,
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Name: "checksum",
		Path: filepath.Join(tmpdir, "checksum"),
		Type: artifact.Checksum,
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Name: "artifact4_1.0.0_linux_amd64",
		Path: filepath.Join(tmpdir, "linux_amd64", "artifact4"),
		Type: artifact.UploadableBinary,
	})

	// configure the pipeline
	// make sure we are using the test keyring
	assert.NoError(t, Pipe{}.Default(ctx))
	ctx.Config.Sign.Args = append([]string{"--homedir", keyring}, ctx.Config.Sign.Args...)

	// run the pipeline
	assert.NoError(t, Pipe{}.Run(ctx))

	// verify that only the artifacts and the signatures are in the dist dir
	gotFiles := []string{}

	err = filepath.Walk(tmpdir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(tmpdir, path)
			if err != nil {
				return err
			}
			gotFiles = append(gotFiles, relPath)
			return nil
		})
	assert.NoError(t, err)

	wantFiles := append(artifacts, signaturePaths...)
	sort.Strings(wantFiles)
	assert.Equal(t, wantFiles, gotFiles)

	// verify the signatures
	for _, sig := range signaturePaths {
		verifySignature(t, ctx, sig)
	}

	var signArtifacts []string
	for _, sig := range ctx.Artifacts.Filter(artifact.ByType(artifact.Signature)).List() {
		signArtifacts = append(signArtifacts, sig.Name)
	}
	// check signature is an artifact
	assert.Equal(t, signArtifacts, signatureNames)
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
