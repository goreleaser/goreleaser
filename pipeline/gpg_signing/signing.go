// Package checksums provides a Pipe that creates .checksums files for
// each artifact.
package gpg_signing

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/apex/log"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
)

// Pipe for checksums
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Signing release artifacts"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	if ctx.Config.GPGSigning.KeyID == "" {
		return pipeline.Skip("gpg_signing.key_id is not configured")
	}

	// read key id from config
	if len(ctx.Config.GPGSigning.KeyID) != 16 {
		return fmt.Errorf("invalid key_id '%s', needs to be a 8 byte long hex key id", ctx.Config.GPGSigning.KeyID)
	}
	keyID, err := strconv.ParseUint(ctx.Config.GPGSigning.KeyID, 16, 64)
	if err != nil {
		return fmt.Errorf("invalid key_id '%s': %s", ctx.Config.GPGSigning.KeyID, err)
	}

	// read gpg key ring
	keyRingPath, err := homedir.Expand("~/.gnupg/secring.gpg")
	if err != nil {
		return err
	}
	keyRingFile, err := os.OpenFile(
		keyRingPath,
		os.O_RDONLY,
		0644,
	)
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
		return fmt.Errorf("no key with id '%s' found", ctx.Config.GPGSigning.KeyID)
	}

	key := keys[0]
	artifacts := ctx.Artifacts
	signatures := make([]string, len(ctx.Artifacts))

	defer func() {
		for _, signature := range signatures {
			if signature != "" {
				ctx.AddArtifact(signature)
			}
		}
	}()
	var g errgroup.Group
	for i, _ := range artifacts {
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
	return g.Wait()
}

func signArtifact(ctx *context.Context, signer *openpgp.Entity, name string) (signaturePath string, err error) {
	signatureFilename := fmt.Sprintf("%s.asc", name)
	signatureFile, err := os.OpenFile(
		filepath.Join(ctx.Config.Dist, signatureFilename),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0644,
	)
	if err != nil {
		return "", err
	}
	defer signatureFile.Close()

	artifactFile, err := os.OpenFile(
		filepath.Join(ctx.Config.Dist, name),
		os.O_RDONLY,
		0644,
	)
	if err != nil {
		return "", err
	}
	defer artifactFile.Close()

	log.WithField("file", name).WithField("signature", signatureFilename).Info("signing")

	err = openpgp.ArmoredDetachSign(
		signatureFile,
		signer,
		artifactFile,
		nil,
	)

	if err != nil {
		return "", err
	}

	return signatureFile.Name(), nil
}
