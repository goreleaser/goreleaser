// Package sign provides a Pipe that signs .checksums files.
package sign

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/apex/log"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
)

// Pipe for checksums
type Pipe struct{}

// Description of the pipe
func (Pipe) String() string {
	return "Signing release artifacts"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	if ctx.Config.Sign.GPGKeyID == "" {
		return pipeline.Skip("sign.gpg_key_id is not configured")
	}
	return gpgSign(ctx)
}

func gpgSign(ctx *context.Context) error {
	rawKeyID := ctx.Config.Sign.GPGKeyID

	if len(rawKeyID) != 16 {
		return fmt.Errorf("invalid key_id '%s', needs to be a 8 byte long hex key id", rawKeyID)
	}

	keyID, err := strconv.ParseUint(rawKeyID, 16, 64)
	if err != nil {
		return fmt.Errorf("invalid key_id '%s': %s", rawKeyID, err)
	}

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
	if len(keys) < 1 {
		return fmt.Errorf("no key with id %q found", rawKeyID)
	}

	key := keys[0]
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

func signArtifact(ctx *context.Context, signer *openpgp.Entity, name string) (signaturePath string, err error) {
	if signer.PrivateKey.Encrypted {
		fd := int(os.Stdin.Fd())
		state, err := terminal.MakeRaw(fd)
		if err != nil {
			return "", err
		}
		defer terminal.Restore(fd, state)

		t := terminal.NewTerminal(os.Stdin, "")
		text, err := t.ReadPassword("Enter passphrase: ")
		if err != nil {
			return "", err
		}
		if err := signer.PrivateKey.Decrypt([]byte(text)); err != nil {
			return "", err
		}
	}

	sigExt := ctx.Config.Sign.SignatureExt
	if sigExt == "" {
		sigExt = ".asc"
	}
	sigFilename := name + sigExt
	sigFile, err := os.OpenFile(
		filepath.Join(ctx.Config.Dist, sigFilename),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0644,
	)
	if err != nil {
		return "", err
	}
	defer sigFile.Close()

	artifactFile, err := os.OpenFile(
		filepath.Join(ctx.Config.Dist, name),
		os.O_RDONLY,
		0644,
	)
	if err != nil {
		return "", err
	}
	defer artifactFile.Close()

	log.WithField("file", name).WithField("signature", sigFilename).Info("signing")

	err = openpgp.ArmoredDetachSign(
		sigFile,
		signer,
		artifactFile,
		nil,
	)

	if err != nil {
		return "", err
	}

	return sigFile.Name(), nil
}
