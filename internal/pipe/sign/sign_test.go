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

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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
	require.NotEmpty(t, Pipe{}.String())
}

func TestSignDefault(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Signs: []config.Sign{{}},
		},
	}
	err := Pipe{}.Default(ctx)
	require.NoError(t, err)
	require.Equal(t, ctx.Config.Signs[0].Cmd, "gpg")
	require.Equal(t, ctx.Config.Signs[0].Signature, "${artifact}.sig")
	require.Equal(t, ctx.Config.Signs[0].Args, []string{"--output", "$signature", "--detach-sig", "$artifact"})
	require.Equal(t, ctx.Config.Signs[0].Artifacts, "none")
}

func TestSignDisabled(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.Config.Signs = []config.Sign{
		{Artifacts: "none"},
	}
	err := Pipe{}.Run(ctx)
	require.EqualError(t, err, "artifact signing is disabled")
}

func TestSignInvalidArtifacts(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.Config.Signs = []config.Sign{
		{Artifacts: "foo"},
	}
	err := Pipe{}.Run(ctx)
	require.EqualError(t, err, "invalid list of artifacts to sign: foo")
}

func TestSignArtifacts(t *testing.T) {
	stdin := passwordUser
	tmplStdin := passwordUserTmpl
	tests := []struct {
		desc             string
		ctx              *context.Context
		signaturePaths   []string
		signatureNames   []string
		certificateNames []string
		expectedErrMsg   string
		user             string
	}{
		{
			desc:           "sign cmd not found",
			expectedErrMsg: `sign: not-a-valid-cmd failed: exec: "not-a-valid-cmd": executable file not found in $PATH: `,
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "all",
							Cmd:       "not-a-valid-cmd",
						},
					},
				},
			),
		},
		{
			desc:           "sign errors",
			expectedErrMsg: "sign: exit failed",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "all",
							Cmd:       "exit",
							Args:      []string{"1"},
						},
					},
				},
			),
		},
		{
			desc:           "invalid certificate template",
			expectedErrMsg: `sign failed: artifact1: template: tmpl:1:3: executing "tmpl" at <.blah>: map has no entry for key "blah"`,
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts:   "all",
							Cmd:         "exit",
							Certificate: "{{ .blah }}",
						},
					},
				},
			),
		},
		{
			desc:           "invalid signature template",
			expectedErrMsg: `sign failed: artifact1: template: tmpl:1:3: executing "tmpl" at <.blah>: map has no entry for key "blah"`,
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "all",
							Cmd:       "exit",
							Signature: "{{ .blah }}",
						},
					},
				},
			),
		},
		{
			desc:           "invalid args template",
			expectedErrMsg: `sign failed: artifact1: template: tmpl:1: unexpected "}" in operand`,
			ctx: context.New(
				config.Project{
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
				},
			),
		},
		{
			desc:           "invalid env template",
			expectedErrMsg: `sign failed: artifact1: template: tmpl:1:5: executing "tmpl" at <.blah>: map has no entry for key "blah"`,
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "all",
							Cmd:       "exit",
							Env:       []string{"A={{ .blah }}"},
						},
					},
				},
			),
		},
		{
			desc: "sign single",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{Artifacts: "all"},
					},
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
		},
		{
			desc: "sign all artifacts",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "all",
						},
					},
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
		},
		{
			desc: "sign archives",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "archive",
						},
					},
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig"},
		},
		{
			desc: "sign packages",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "package",
						},
					},
				},
			),
			signaturePaths: []string{"package1.deb.sig"},
			signatureNames: []string{"package1.deb.sig"},
		},
		{
			desc: "sign binaries",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "binary",
						},
					},
				},
			),
			signaturePaths: []string{"artifact3.sig", "linux_amd64/artifact4.sig"},
			signatureNames: []string{"artifact3_1.0.0_linux_amd64.sig", "artifact4_1.0.0_linux_amd64.sig"},
		},
		{
			desc: "multiple sign configs",
			ctx: context.New(
				config.Project{
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
				},
			),
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
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "all",
							IDs:       []string{"foo"},
						},
					},
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
		},
		{
			desc: "sign only checksums",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "checksum",
						},
					},
				},
			),
			signaturePaths: []string{"checksum.sig", "checksum2.sig"},
			signatureNames: []string{"checksum.sig", "checksum2.sig"},
		},
		{
			desc: "sign only filtered checksums",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "checksum",
							IDs:       []string{"foo"},
						},
					},
				},
			),
			signaturePaths: []string{"checksum.sig", "checksum2.sig"},
			signatureNames: []string{"checksum.sig", "checksum2.sig"},
		},
		{
			desc: "sign only source",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "source",
						},
					},
				},
			),
			signaturePaths: []string{"artifact5.tar.gz.sig"},
			signatureNames: []string{"artifact5.tar.gz.sig"},
		},
		{
			desc: "sign only source filter by id",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Artifacts: "source",
							IDs:       []string{"should-not-be-used"},
						},
					},
				},
			),
			signaturePaths: []string{"artifact5.tar.gz.sig"},
			signatureNames: []string{"artifact5.tar.gz.sig"},
		},
		{
			desc: "sign all artifacts with env",
			ctx: context.New(
				config.Project{
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
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
		},
		{
			desc: "sign all artifacts with template",
			ctx: context.New(
				config.Project{
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
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
		},
		{
			desc: "sign single with password from stdin",
			ctx: context.New(
				config.Project{
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
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			user:           passwordUser,
		},
		{
			desc: "sign single with password from templated stdin",
			ctx: context.New(
				config.Project{
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
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			user:           passwordUser,
		},
		{
			desc: "sign single with password from stdin_file",
			ctx: context.New(
				config.Project{
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
				},
			),
			signaturePaths: []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames: []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			user:           passwordUser,
		},
		{
			desc: "missing stdin_file",
			ctx: context.New(
				config.Project{
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
				},
			),
			expectedErrMsg: `sign failed: cannot open file /tmp/non-existing-file: open /tmp/non-existing-file: no such file or directory`,
		},
		{
			desc: "sign creating certificate",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Certificate: "${artifact}.pem",
							Artifacts:   "checksum",
						},
					},
				},
			),
			signaturePaths:   []string{"checksum.sig", "checksum2.sig"},
			signatureNames:   []string{"checksum.sig", "checksum2.sig"},
			certificateNames: []string{"checksum.pem", "checksum2.pem"},
		},
		{
			desc: "sign all artifacts with env and certificate",
			ctx: context.New(
				config.Project{
					Signs: []config.Sign{
						{
							Env:         []string{"NOT_HONK=honk", "HONK={{ .Env.NOT_HONK }}"},
							Certificate: `{{ trimsuffix (trimsuffix .Env.artifact ".tar.gz") ".deb" }}_${HONK}.pem`,
							Artifacts:   "all",
						},
					},
				},
			),
			signaturePaths:   []string{"artifact1.sig", "artifact2.sig", "artifact3.sig", "checksum.sig", "checksum2.sig", "linux_amd64/artifact4.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			signatureNames:   []string{"artifact1.sig", "artifact2.sig", "artifact3_1.0.0_linux_amd64.sig", "checksum.sig", "checksum2.sig", "artifact4_1.0.0_linux_amd64.sig", "artifact5.tar.gz.sig", "package1.deb.sig"},
			certificateNames: []string{"artifact1_honk.pem", "artifact2_honk.pem", "artifact3_1.0.0_linux_amd64_honk.pem", "checksum_honk.pem", "checksum2_honk.pem", "artifact4_1.0.0_linux_amd64_honk.pem", "artifact5_honk.pem", "package1_honk.pem"},
		},
	}

	for _, test := range tests {
		if test.user == "" {
			test.user = user
		}

		t.Run(test.desc, func(t *testing.T) {
			testSign(t, test.ctx, test.certificateNames, test.signaturePaths, test.signatureNames, test.user, test.expectedErrMsg)
		})
	}
}

func testSign(tb testing.TB, ctx *context.Context, certificateNames, signaturePaths, signatureNames []string, user, expectedErrMsg string) {
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

	// run the pipeline
	if expectedErrMsg != "" {
		err := Pipe{}.Run(ctx)
		require.Error(tb, err)
		require.Contains(tb, err.Error(), expectedErrMsg)
		return
	}

	require.NoError(tb, Pipe{}.Run(ctx))

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
	sort.Strings(certificateNames)
	sort.Strings(certNames)
	require.Equal(tb, len(certificateNames), len(certificates))
	if len(certificateNames) > 0 {
		require.Equal(tb, certificateNames, certNames)
	}

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

	// verify signature was made with key for usesr 'nopass'
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
	ctx := &context.Context{
		Config: config.Project{
			Signs: []config.Sign{
				{
					ID: "a",
				},
				{
					ID: "a",
				},
			},
		},
	}
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 signs with the ID 'a', please fix your config")
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("skip sign", func(t *testing.T) {
		ctx := context.New(config.Project{})
		ctx.SkipSign = true
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Signs: []config.Sign{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}
