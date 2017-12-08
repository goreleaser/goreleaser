// Package sign provides a Pipe that signs .checksums files.
package sign

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/apex/log"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
)

type Pipe struct{}

func (Pipe) String() string {
	return "Signing release artifacts"
}

func (Pipe) Run(ctx *context.Context) (err error) {
	if ctx.Config.Sign.GPGKeyID == "" {
		return pipeline.Skip("sign.gpg_key_id is not configured")
	}
	return gpgSign(ctx)
}

func gpgSign(ctx *context.Context) error {
	rawKeyID := ctx.Config.Sign.GPGKeyID

	if len(rawKeyID) != 16 {
		return fmt.Errorf("invalid gpg_key_id %q, needs to be a 8 byte long hex key id", rawKeyID)
	}

	keyID, err := strconv.ParseUint(rawKeyID, 16, 64)
	if err != nil {
		return fmt.Errorf("invalid gpg_key_id '%s': %s", rawKeyID, err)
	}

	// todo(fs): is this always in that location?
	keyRingPath, err := homedir.Expand("~/.gnupg/secring.gpg")
	if err != nil {
		return err
	}

	keyRingFile, err := os.OpenFile(keyRingPath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer keyRingFile.Close()

	keyList, err := openpgp.ReadKeyRing(keyRingFile)
	if err != nil {
		return err
	}

	keys := keyList.KeysById(keyID)
	if len(keys) == 0 {
		return fmt.Errorf("no key found with id %q", rawKeyID)
	}

	key := keys[0]
	if key.Entity.PrivateKey.Encrypted {
		err := decrypt(key.Entity, readPassword)
		if err != nil {
			return fmt.Errorf("cannot decrypt private key: %s", err)
		}
	}

	artifacts := ctx.Artifacts
	if ctx.Config.Sign.ChecksumOnly {
		artifacts = ctx.Checksums
	}
	signatures := make([]string, len(artifacts))

	var g errgroup.Group
	for i := range artifacts {
		pos := i
		g.Go(func() error {
			signature, err := signArtifact(ctx, key.Entity, artifacts[pos])
			if err != nil {
				return err
			}
			signatures[pos] = signature
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	for _, sig := range signatures {
		if sig != "" {
			ctx.AddArtifact(sig)
		}
	}
	return nil
}

// decrypt reads the passphrase from the readPassword function and attempts
// to decrypt the key with it.
func decrypt(e *openpgp.Entity, passwdFn func(string) (string, error)) error {
	// number of retries for password entry.
	const attempts = 3

	// time between retries.
	const delay = time.Second

	if !e.PrivateKey.Encrypted {
		return nil
	}

	var err error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			time.Sleep(delay)
		}

		prompt := fmt.Sprintf("Enter passphrase (%d/%d): ", i+1, attempts)
		text, err := passwdFn(prompt)
		if err != nil {
			continue
		}

		err = e.PrivateKey.Decrypt([]byte(text))
		if err != nil {
			fmt.Println("Bad passphrase")
			continue
		}

		// passphrase ok
		return nil
	}
	return err
}

// readPassword reads a password from stdin without echoing it.
func readPassword(prompt string) (string, error) {
	// switch terminal to raw mode to disable echoing
	// of passphrase and restore old state on return
	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer terminal.Restore(fd, state)

	t := terminal.NewTerminal(os.Stdin, "")
	return t.ReadPassword(prompt)
}

func signArtifact(ctx *context.Context, signer *openpgp.Entity, name string) (signaturePath string, err error) {
	sigExt := ctx.Config.Sign.SignatureExt
	if sigExt == "" {
		sigExt = ".asc"
	}

	sigFilename := name + sigExt
	sigPath := filepath.Join(ctx.Config.Dist, sigFilename)
	sigFile, err := os.OpenFile(sigPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer sigFile.Close()

	artifactPath := filepath.Join(ctx.Config.Dist, name)
	artifactFile, err := os.OpenFile(artifactPath, os.O_RDONLY, 0644)
	if err != nil {
		return "", err
	}
	defer artifactFile.Close()

	log.WithField("file", name).WithField("signature", sigFilename).Info("signing")

	err = openpgp.ArmoredDetachSign(sigFile, signer, artifactFile, nil)
	if err != nil {
		return "", err
	}
	return sigFile.Name(), nil
}
