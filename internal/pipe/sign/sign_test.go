package sign

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

var (
	originKeyring = "testdata/gnupg"
	keyring       string
)

const (
	user             = "nopass"
	passwordUser     = "password"
	passwordUserTmpl = "{{ .Env.GPG_PASSWORD }}"
	fakeGPGKeyID     = "23E7505E"
)

func TestMain(m *testing.M) {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	keyring = filepath.Join(os.TempDir(), fmt.Sprintf("gorel_gpg_test.%d", rand.Int()))
	fmt.Println("copying", originKeyring, "to", keyring)
	if err := gio.Copy(originKeyring, keyring); err != nil {
		fmt.Printf("failed to copy %s to %s: %s", originKeyring, keyring, err)
		os.Exit(1)
	}

	m.Run()
	_ = os.RemoveAll(keyring)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSignDefault(t *testing.T) {
	_ = testlib.Mktmp(t)
	testlib.GitInit(t)

	ctx := testctx.NewWithCfg(config.Project{
		Signs: []config.Sign{{}},
	})
	setGpg(t, ctx, "") // force empty gpg.program

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "gpg", ctx.Config.Signs[0].Cmd)
	require.Equal(t, "${artifact}.sig", ctx.Config.Signs[0].Signature)
	require.Equal(t, []string{"--output", "$signature", "--detach-sig", "$artifact"}, ctx.Config.Signs[0].Args)
	require.Equal(t, "none", ctx.Config.Signs[0].Artifacts)
}

func TestDefaultGpgFromGitConfig(t *testing.T) {
	_ = testlib.Mktmp(t)
	testlib.GitInit(t)

	ctx := testctx.NewWithCfg(config.Project{
		Signs: []config.Sign{{}},
	})
	setGpg(t, ctx, "not-really-gpg")

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "not-really-gpg", ctx.Config.Signs[0].Cmd)
}

func TestSignDisabled(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{Signs: []config.Sign{{Artifacts: "none"}}})
	err := Pipe{}.Run(ctx)
	require.EqualError(t, err, "artifact signing is disabled")
}

func TestSignInvalidArtifacts(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{Signs: []config.Sign{{Artifacts: "foo"}}})
	err := Pipe{}.Run(ctx)
	require.EqualError(t, err, "invalid list of artifacts to sign: foo")
}

func TestSignArtifacts(t *testing.T) {
	testlib.SkipIfWindows(t, "tries to use /usr/bin/gpg-agent")
	stdin := passwordUser
	tmplStdin := passwordUserTmpl
	tests := []struct {
		desc             string
		ctx              *context.Context
		signaturePaths   []string
		signatureNames   []string
		certificateNames []string
		expectedErrMsg   string
		expectedErrIs    error
		expectedErrAs    any
		user             string
	}{
		{
			desc:          "sign cmd not found",
			expectedErrIs: exec.ErrNotFound,
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Cmd:       "not-a-valid-cmd",
					},
				},
			}),
		},
		{
			desc:           "sign errors",
			expectedErrMsg: "sign: exit failed",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Cmd:       "exit",
						Args:      []string{"1"},
					},
				},
			}),
		},
		{
			desc:          "invalid certificate template",
			expectedErrAs: &tmpl.Error{},
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts:   "all",
						Cmd:         "exit",
						Certificate: "{{ .blah }}",
					},
				},
			}),
		},
		{
			desc:          "invalid signature template",
			expectedErrAs: &tmpl.Error{},
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Cmd:       "exit",
						Signature: "{{ .blah }}",
					},
				},
			}),
		},
		{
			desc:          "invalid args template",
			expectedErrAs: &tmpl.Error{},
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Cmd:       "exit",
						Args:      []string{"${FOO}-{{ .foo }{{}}{"},
					},
				},
				Env: []string{
					"FOO=BAR",
				},
			}),
		},
		{
			desc:          "invalid env template",
			expectedErrAs: &tmpl.Error{},
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Cmd:       "exit",
						Env:       []string{"A={{ .blah }}"},
					},
				},
			}),
		},
		{
			desc: "sign all artifacts",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
					},
				},
			}),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
		},
		{
			desc: "sign archives",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "archive",
					},
				},
			}),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig"},
		},
		{
			desc: "sign packages",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "package",
					},
				},
			}),
			signaturePaths: []string{"package1.deb.sig"},
			signatureNames: []string{"package1.deb.sig"},
		},
		{
			desc: "sign binaries",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "binary",
					},
				},
			}),
			signaturePaths: []string{"artifact3.sig", "linux_amd64/artifact4.sig"},
			signatureNames: []string{"artifact3_1.0.0_linux_amd64.sig", "artifact4_1.0.0_linux_amd64.sig"},
		},
		{
			desc: "multiple sign configs",
			ctx: testctx.NewWithCfg(config.Project{
				Env: []string{
					"GPG_KEY_ID=" + fakeGPGKeyID,
				},
				Signs: []config.Sign{
					{
						ID:        "s1",
						Artifacts: "checksum",
					},
					{
						ID:        "s2",
						Artifacts: "archive",
						Signature: "${artifact}.{{ .Env.GPG_KEY_ID }}.sig",
					},
				},
			}),
			signaturePaths: []string{
				"artifact1." + fakeGPGKeyID + ".sig",
				"artifact2." + fakeGPGKeyID + ".sig",
				"checksum.sig",
				"checksum2.sig",
			},
			signatureNames: []string{
				"artifact1." + fakeGPGKeyID + ".sig",
				"artifact2." + fakeGPGKeyID + ".sig",
				"checksum.sig",
				"checksum2.sig",
			},
		},
		{
			desc: "sign filtered artifacts",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						IDs:       []string{"foo"},
					},
				},
			}),
			signaturePaths: []string{"artifact1.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
		},
		{
			desc: "sign only checksums",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "checksum",
					},
				},
			}),
			signaturePaths: []string{"checksum.sig", "checksum2.sig"},
			signatureNames: []string{"checksum.sig", "checksum2.sig"},
		},
		{
			desc: "sign only filtered checksums",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "checksum",
						IDs:       []string{"foo"},
					},
				},
			}),
			signaturePaths: []string{"checksum.sig", "checksum2.sig"},
			signatureNames: []string{"checksum.sig", "checksum2.sig"},
		},
		{
			desc: "sign only source",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "source",
					},
				},
			}),
			signaturePaths: []string{"artifact5.tar.gz.sig"},
			signatureNames: []string{"artifact5.tar.gz.sig"},
		},
		{
			desc: "sign only source filter by id",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "source",
						IDs:       []string{"should-not-be-used"},
					},
				},
			}),
			signaturePaths: []string{"artifact5.tar.gz.sig"},
			signatureNames: []string{"artifact5.tar.gz.sig"},
		},
		{
			desc: "sign only sbom",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "sbom",
					},
				},
			}),
			signaturePaths: []string{"artifact5.tar.gz.sbom.sig"},
			signatureNames: []string{"artifact5.tar.gz.sbom.sig"},
		},
		{
			desc: "sign all artifacts with env",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
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
				},
				Env: []string{
					fmt.Sprintf("TEST_USER=%s", user),
				},
			}),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
		},
		{
			desc: "sign all artifacts with template",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Args: []string{
							"-u",
							"{{ .Env.SOME_TEST_USER }}",
							"--output",
							"${signature}",
							"--detach-sign",
							"${artifact}",
						},
					},
				},
				Env: []string{
					fmt.Sprintf("SOME_TEST_USER=%s", user),
				},
			}),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
		},
		{
			desc: "sign single with password from stdin",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Args: []string{
							"-u",
							passwordUser,
							"--batch",
							"--pinentry-mode",
							"loopback",
							"--passphrase-fd",
							"0",
							"--output",
							"${signature}",
							"--detach-sign",
							"${artifact}",
						},
						Stdin: &stdin,
					},
				},
			}),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			user:           passwordUser,
		},
		{
			desc: "sign single with password from templated stdin",
			ctx: testctx.NewWithCfg(config.Project{
				Env: []string{"GPG_PASSWORD=" + stdin},
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Args: []string{
							"-u",
							passwordUser,
							"--batch",
							"--pinentry-mode",
							"loopback",
							"--passphrase-fd",
							"0",
							"--output",
							"${signature}",
							"--detach-sign",
							"${artifact}",
						},
						Stdin: &tmplStdin,
					},
				},
			}),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			user:           passwordUser,
		},
		{
			desc: "sign single with password from stdin_file",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Args: []string{
							"-u",
							passwordUser,
							"--batch",
							"--pinentry-mode",
							"loopback",
							"--passphrase-fd",
							"0",
							"--output",
							"${signature}",
							"--detach-sign",
							"${artifact}",
						},
						StdinFile: filepath.Join(keyring, passwordUser),
					},
				},
			}),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			user:           passwordUser,
		},
		{
			desc: "missing stdin_file",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Artifacts: "all",
						Args: []string{
							"--batch",
							"--pinentry-mode",
							"loopback",
							"--passphrase-fd",
							"0",
						},
						StdinFile: "/tmp/non-existing-file",
					},
				},
			}),
			expectedErrIs: os.ErrNotExist,
		},
		{
			desc: "sign creating certificate",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Certificate: "${artifact}.pem",
						Artifacts:   "checksum",
					},
				},
			}),
			signaturePaths:   []string{"checksum.sig", "checksum2.sig"},
			signatureNames:   []string{"checksum.sig", "checksum2.sig"},
			certificateNames: []string{"checksum.pem", "checksum2.pem"},
		},
		{
			desc: "sign all artifacts with env and certificate",
			ctx: testctx.NewWithCfg(config.Project{
				Signs: []config.Sign{
					{
						Env:         []string{"NOT_HONK=honk", "HONK={{ .Env.NOT_HONK }}"},
						Certificate: `{{ trimsuffix (trimsuffix .Env.artifact ".tar.gz") ".deb" }}_${HONK}.pem`,
						Artifacts:   "all",
					},
				},
			}),
			signaturePaths:   []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			signatureNames:   []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "artifact5.tar.gz.sbom.sig", "package1.deb.sig"},
			certificateNames: []string{"artifact1_honk.pem", "artifact2_honk.pem", "artifact3_1.0.0_linux_amd64_honk.pem", "checksum_honk.pem", "checksum2_honk.pem", "artifact4_1.0.0_linux_amd64_honk.pem", "artifact5_honk.pem", "artifact5.tar.gz.sbom_honk.pem", "package1_honk.pem"},
		},
	}

	for _, test := range tests {
		if test.user == "" {
			test.user = user
		}

		t.Run(test.desc, func(t *testing.T) {
			testlib.CheckPath(t, "gpg")
			testSign(
				t,
				test.ctx,
				test.certificateNames,
				test.signaturePaths,
				test.signatureNames,
				test.user,
				test.expectedErrMsg,
				test.expectedErrIs,
				test.expectedErrAs,
			)
		})
	}
}

func testSign(
	tb testing.TB,
	ctx *context.Context,
	certificateNames, signaturePaths, signatureNames []string,
	user, expectedErrMsg string,
	expectedErrIs error,
	expectedErrAs any,
) {
	tb.Helper()
	tmpdir := tb.TempDir()

	ctx.Config.Dist = tmpdir

	// create some fake artifacts
	artifacts := []string{"artifact1", "artifact2", "artifact3", "checksum", "checksum2", "package1.deb"}
	require.NoError(tb, os.Mkdir(filepath.Join(tmpdir, "linux_amd64"), os.ModePerm))
	for _, f := range artifacts {
		file := filepath.Join(tmpdir, f)
		require.NoError(tb, os.WriteFile(file, []byte("foo"), 0o644))
	}
	require.NoError(tb, os.WriteFile(filepath.Join(tmpdir, "linux_amd64", "artifact4"), []byte("foo"), 0o644))
	artifacts = append(artifacts, "linux_amd64/artifact4")
	require.NoError(tb, os.WriteFile(filepath.Join(tmpdir, "artifact5.tar.gz"), []byte("foo"), 0o644))
	artifacts = append(artifacts, "artifact5.tar.gz")
	require.NoError(tb, os.WriteFile(filepath.Join(tmpdir, "artifact5.tar.gz.sbom"), []byte("sbom(foo)"), 0o644))
	artifacts = append(artifacts, "artifact5.tar.gz.sbom")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact1",
		Path: filepath.Join(tmpdir, "artifact1"),
		Type: artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact2",
		Path: filepath.Join(tmpdir, "artifact2"),
		Type: artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo3",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact3_1.0.0_linux_amd64",
		Path: filepath.Join(tmpdir, "artifact3"),
		Type: artifact.UploadableBinary,
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "checksum",
		Path: filepath.Join(tmpdir, "checksum"),
		Type: artifact.Checksum,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "checksum2",
		Path: filepath.Join(tmpdir, "checksum2"),
		Type: artifact.Checksum,
		Extra: map[string]interface{}{
			"Refresh": func() error {
				file := filepath.Join(tmpdir, "checksum2")
				return os.WriteFile(file, []byte("foo"), 0o644)
			},
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact4_1.0.0_linux_amd64",
		Path: filepath.Join(tmpdir, "linux_amd64", "artifact4"),
		Type: artifact.UploadableBinary,
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo3",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact5.tar.gz",
		Path: filepath.Join(tmpdir, "artifact5.tar.gz"),
		Type: artifact.UploadableSourceArchive,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact5.tar.gz.sbom",
		Path: filepath.Join(tmpdir, "artifact5.tar.gz.sbom"),
		Type: artifact.SBOM,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "package1.deb",
		Path: filepath.Join(tmpdir, "package1.deb"),
		Type: artifact.LinuxPackage,
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo",
		},
	})

	// configure the pipeline
	// make sure we are using the test keyring
	require.NoError(tb, Pipe{}.Default(ctx))
	for i := range ctx.Config.Signs {
		ctx.Config.Signs[i].Args = append(
			[]string{"--homedir", keyring},
			ctx.Config.Signs[i].Args...,
		)
	}

	err := Pipe{}.Run(ctx)

	// run the pipeline
	if expectedErrMsg != "" {
		require.ErrorContains(tb, err, expectedErrMsg)
		return
	}

	if expectedErrIs != nil {
		require.ErrorIs(tb, err, expectedErrIs)
		return
	}

	if expectedErrAs != nil {
		require.ErrorAs(tb, err, expectedErrAs)
		return
	}

	require.NoError(tb, err)

	// ensure all artifacts have an ID
	for _, arti := range ctx.Artifacts.Filter(
		artifact.Or(
			artifact.ByType(artifact.Signature),
			artifact.ByType(artifact.Certificate),
		),
	).List() {
		require.NotEmptyf(tb, arti.ID(), ".Extra.ID on %s", arti.Path)
	}

	certificates := ctx.Artifacts.Filter(artifact.ByType(artifact.Certificate)).List()
	certNames := []string{}
	for _, cert := range certificates {
		certNames = append(certNames, cert.Name)
		require.True(tb, strings.HasPrefix(cert.Path, ctx.Config.Dist))
	}

	assert.ElementsMatch(tb, certificateNames, certNames)

	// verify that only the artifacts and the signatures are in the dist dir
	gotFiles := []string{}

	require.NoError(tb, filepath.Walk(tmpdir,
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
		}),
	)

	wantFiles := append(artifacts, signaturePaths...)
	sort.Strings(wantFiles)
	require.ElementsMatch(tb, wantFiles, gotFiles)

	// verify the signatures
	for _, sig := range signaturePaths {
		verifySignature(tb, ctx, sig, user)
	}

	var signArtifacts []string
	for _, sig := range ctx.Artifacts.Filter(artifact.ByType(artifact.Signature)).List() {
		signArtifacts = append(signArtifacts, sig.Name)
	}
	// check signature is an artifact
	require.ElementsMatch(tb, signArtifacts, signatureNames)
}

func verifySignature(tb testing.TB, ctx *context.Context, sig string, user string) {
	tb.Helper()
	artifact := strings.TrimSuffix(sig, filepath.Ext(sig))
	artifact = strings.TrimSuffix(artifact, "."+fakeGPGKeyID)

	// verify signature was made with key for user 'nopass'
	cmd := exec.Command("gpg", "--homedir", keyring, "--verify", filepath.Join(ctx.Config.Dist, sig), filepath.Join(ctx.Config.Dist, artifact))
	out, err := cmd.CombinedOutput()
	require.NoError(tb, err, string(out))

	// check if the signature matches the user we expect to do this properly we
	// might need to have either separate keyrings or export the key from the
	// keyring before we do the verification. For now we punt and look in the
	// output.
	if !bytes.Contains(out, []byte(user)) {
		tb.Fatalf("%s: signature is not from %s: %s", sig, user, string(out))
	}
}

func TestSeveralSignsWithTheSameID(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Signs: []config.Sign{
			{
				ID: "a",
			},
			{
				ID: "a",
			},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 signs with the ID 'a', please fix your config")
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("skip sign", func(t *testing.T) {
		ctx := testctx.New(testctx.Skip(skips.Sign))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Signs: []config.Sign{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDependencies(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Signs: []config.Sign{
			{Cmd: "cosign"},
			{Cmd: "gpg2"},
		},
	})
	require.Equal(t, []string{"cosign", "gpg2"}, Pipe{}.Dependencies(ctx))
}

func setGpg(tb testing.TB, ctx *context.Context, p string) {
	tb.Helper()
	_, err := git.Run(ctx, "config", "--local", "--add", "gpg.program", p)
	require.NoError(tb, err)
}
