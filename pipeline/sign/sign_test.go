package sign

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"

	"github.com/stretchr/testify/assert"
)

const keyring = "testdata/gnupg"

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestSign(t *testing.T) {
	// create temp dir for file and signature
	tmpdir, err := ioutil.TempDir("", "goreleaser")
	if err != nil {
		t.Fatal("TempDir: ", err)
	}
	defer os.RemoveAll(tmpdir)

	artifact := "foo.txt"
	signature := artifact + ".sig"

	// create fake artifact
	file := filepath.Join(tmpdir, artifact)
	if err = ioutil.WriteFile(file, []byte("foo"), 0644); err != nil {
		t.Fatal("WriteFile: ", err)
	}

	// fix permission on keyring dir to suppress warning about insecure permissions
	if err = os.Chmod(keyring, 0700); err != nil {
		t.Fatal("Chmod: ", err)
	}

	// sign artifact
	ctx := &context.Context{
		Config: config.Project{
			Dist: tmpdir,
			Sign: config.Sign{
				Artifacts: "all",
			},
		},
	}
	ctx.AddArtifact(artifact)

	err = Pipe{}.Default(ctx)
	if err != nil {
		t.Fatal("Default: ", err)
	}

	// make sure we are using the test keyring
	ctx.Config.Sign.Args = append([]string{"--homedir", keyring}, ctx.Config.Sign.Args...)

	err = Pipe{}.Run(ctx)
	if err != nil {
		t.Fatal("Run: ", err)
	}

	// verify signature was made with key for usesr 'nopass'
	if err := verifySig(t, keyring, file, filepath.Join(tmpdir, signature), "nopass"); err != nil {
		t.Fatal("verify: ", err)
	}

	// check signature is an artifact
	assert.Equal(t, ctx.Artifacts, []string{artifact, signature})
}

func verifySig(t *testing.T, keyring, file, sig, user string) error {
	cmd := exec.Command("gpg", "--homedir", keyring, "--verify", sig, file)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		return err
	}

	// check if the signature matches the user we expect to do this properly we
	// might need to have either separate keyrings or export the key from the
	// keyring before we do the verification. For now we punt and look in the
	// output.
	if !bytes.Contains(out, []byte(user)) {
		return fmt.Errorf("signature is not from %s", user)
	}

	return nil
}
