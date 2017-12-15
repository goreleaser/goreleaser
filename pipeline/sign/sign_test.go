package sign

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"

	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestSignDefault(t *testing.T) {
	ctx := &context.Context{}
	Pipe{}.Default(ctx)
	assert.Equal(t, ctx.Config.Sign.Cmd, "gpg")
	assert.Equal(t, ctx.Config.Sign.Signature, "${artifact}.sig")
	assert.Equal(t, ctx.Config.Sign.Args, []string{"--output", "$signature", "--detach-sig", "$artifact"})
	assert.Equal(t, ctx.Config.Sign.Artifacts, "none")
}

func TestSignDisabled(t *testing.T) {
	ctx := &context.Context{}
	ctx.Config.Sign.Artifacts = "none"
	err := Pipe{}.Run(ctx)
	assert.EqualError(t, err, "artifact signing disabled")
}

func TestSignInvalidArtifacts(t *testing.T) {
	ctx := &context.Context{}
	ctx.Config.Sign.Artifacts = "foo"
	err := Pipe{}.Run(ctx)
	assert.EqualError(t, err, "invalid list of artifacts to sign: foo")
}

func TestSignArtifacts(t *testing.T) {
	// fix permission on keyring dir to suppress warning about insecure permissions
	if err := os.Chmod(keyring, 0700); err != nil {
		t.Fatal("Chmod: ", err)
	}

	tests := []struct {
		desc       string
		ctx        *context.Context
		signatures []string
	}{
		{
			desc: "sign all artifacts",
			ctx: &context.Context{
				Config: config.Project{
					Sign: config.Sign{Artifacts: "all"},
				},
				Artifacts: []string{"artifact1", "artifact2", "checksum"},
				Checksums: []string{"checksum"},
			},
			signatures: []string{"artifact1.sig", "artifact2.sig", "checksum.sig"},
		},
		{
			desc: "sign only checksums",
			ctx: &context.Context{
				Config: config.Project{
					Sign: config.Sign{Artifacts: "checksum"},
				},
				Artifacts: []string{"artifact1", "artifact2", "checksum"},
				Checksums: []string{"checksum"},
			},
			signatures: []string{"checksum.sig"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testSign(t, tt.ctx, tt.signatures)
		})
	}
}

const keyring = "testdata/gnupg"
const user = "nopass"

func testSign(t *testing.T, ctx *context.Context, signatures []string) {
	// create temp dir for file and signature
	tmpdir, err := ioutil.TempDir("", "goreleaser")
	if err != nil {
		t.Fatal("TempDir: ", err)
	}
	defer os.RemoveAll(tmpdir)

	ctx.Config.Dist = tmpdir

	// create some fake artifacts
	artifacts := ctx.Artifacts
	for _, f := range artifacts {
		file := filepath.Join(tmpdir, f)
		if err2 := ioutil.WriteFile(file, []byte("foo"), 0644); err2 != nil {
			t.Fatal("WriteFile: ", err2)
		}
	}

	// configure the pipeline
	// make sure we are using the test keyring
	err = Pipe{}.Default(ctx)
	if err != nil {
		t.Fatal("Default: ", err)
	}
	ctx.Config.Sign.Args = append([]string{"--homedir", keyring}, ctx.Config.Sign.Args...)

	// run the pipeline
	err = Pipe{}.Run(ctx)
	if err != nil {
		t.Fatal("Run: ", err)
	}

	// verify that only the artifacts and the signatures are in the dist dir
	files, err := ioutil.ReadDir(tmpdir)
	if err != nil {
		t.Fatal("ReadDir: ", err)
	}
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

	// check signature is an artifact
	assert.Equal(t, ctx.Artifacts, append(artifacts, signatures...))
}

func verifySignature(t *testing.T, ctx *context.Context, sig string) {
	artifact := sig[:len(sig)-len(".sig")]

	// verify signature was made with key for usesr 'nopass'
	cmd := exec.Command("gpg", "--homedir", keyring, "--verify", filepath.Join(ctx.Config.Dist, sig), filepath.Join(ctx.Config.Dist, artifact))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatal("verify: ", err)
	}

	// check if the signature matches the user we expect to do this properly we
	// might need to have either separate keyrings or export the key from the
	// keyring before we do the verification. For now we punt and look in the
	// output.
	if !bytes.Contains(out, []byte(user)) {
		t.Fatalf("signature is not from %s", user)
	}
}
