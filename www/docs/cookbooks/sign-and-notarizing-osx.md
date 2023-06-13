# Signing and Notarizing Releases with Gon

OSX artifacts must be both signed and notarized to prevent users from receiving security prompts.

Both hon and goreleaser have somewhat different ideas of how to go about this. It is possible to leverage them together to create a pipeline.

Major concepts:

1. We want gon to sign and notarize.
2. We want to be able to support/use goreleaser's template naming of our artifacts.
3. OSX requires binaries to be signed before the archive may be notarized.
4. Each OSX binary must be indiviually signed. (offline activity)
5. Each OSX archive must be individually notarized. (online activity)

**Note**: There are two cert keypairs one for signing and one for notarization.

## Configuring Signing

The `env` parameters below will need to be customized as appropriate of one's keychain details. See [gon configuration][gon configuration] documentation for what these values should be set to. For simplicity they will be re-used by the notarization step later.

We separate out the mac builds because we are required (by OSX) to sign the binaries themselves. Since signing is an offline operation it's nice and speedy.

Hon is configuration driven, doesn't have a CLI interface, and the configuration doesn't support much in the way of templating. Instead we generate the appropriate configuration on the fly, leverage goreleaser's templating tools to hard-code the correct values, and then run hon to perform the signing.

```yaml
# .goreleaser.yml
env:
  - BUNDLE_ID=com.example.myapp
  - APPLE_ID_USERNAME=user@example.com
  - APPLE_ID_PASSWORD=@keychain:gon
  - "APPLE_APPLICATION_IDENTITY=Developer ID Application: My Name"

builds:
  - id: default
    goos: [linux, windows]
    goarch: [arm64, amd64]
    ignore:
      - goarch: arm
        goos: windows
      - goarch: arm64
        goos: windows
  - id: macos
    goos: [darwin]
    goarch: [arm64, amd64]
    hooks:
      post:
        - |
          sh -c '
          fn=dist/macos_{{.Target}}/gon.hcl
          cat >"$fn" <<EOF
          bundle_id = "{{.Env.BUNDLE_ID}}"
          apple_id {
            username = "{{.Env.APPLE_ID_USERNAME}}"
            password = "{{.Env.APPLE_ID_PASSWORD}}"
          }
          source = ["dist/macos_{{.Target}}/{{.Name}}"]
          sign {
            application_identity = "{{.Env.APPLE_APPLICATION_IDENTITY}}"
          }
          EOF
          '
        - "gon 'dist/macos_{{.Target}}/gon.hcl'"
```

Result: the `dist/macos_*/myapp` will be signed.

```
$ codesign --verify --deep --verbose dist/macos_darwin_arm64/myapp
dist/macos_darwin_arm64/myapp: valid on disk
dist/macos_darwin_arm64/myapp: satisfies its Designated Requirement
```


## Configuring Notarization

Notarizing takes our archived (with a signed binary inside) and submits it to Apple. This is an online activity than can take meaningful time but it necessary for production releases to avoid security prompts.

Here we are doing a very similar behavior as we did during the Build step, where we define a hon config block on the fly and then execute.

We must be sure to define an archive ID, so we when we go to Release later, Goreleaser will pick the correct files. Here we do a specific notarize step _replacing_ the original artifact in-place. One could generate a separate archive filename, just be sure not to Release it.

**Important**:
  - zip, pkg, dmg, app format is required for notarization.

```yaml
# .goreleaser.yml

archives:
  - format: zip
    builds: [default]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: linux
        format: tar.gz
  - id: macos
    builds: [macos]
    format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

signs:
  - id: mac-notarize
    artifacts: archive
    ids: [macos]
    signature: "${artifact}"
    output: true
    cmd: sh
    args:
      - "-c"
      - |-
        cat >"dist/gon.notarize.hcl" <<EOF
        apple_id {
          username = "{{.Env.APPLE_ID_USERNAME}}"
          password = "{{.Env.APPLE_ID_PASSWORD}}"
        }
        notarize {
          path = "${artifact}"
          bundle_id = "{{.Env.BUNDLE_ID}}"
        }
        EOF
        gon "dist/gon.notarize.hcl"
```

Result: notarized artifacts.
FIXME: We know it notarized correctly, why is our archive being rejected by spctl?

```
$ spctl --assess -v dist/myapp_1.0.0_darwin_arm64.zip
dist/myapp_1.0.0_darwin_arm64.zip: rejected
source=no usable signature
```

## Releasing

Since during the Sign step we generated a new set of notarized artifacts, GoReleaser now thinks there are a double-set of OSX related artifacts (ones from the Artifacts step, ones from Sign step). When attempting to Release, it will try to release both sets.

If we've used the same name like in this example our publishing services may complain about overwriting existing artifacts. Instead we explictly inform goreleaser which artifacts to upload. Excluding the `macos` artifacts ID to prevent the duplication error.

```yaml
release:
  ids: [default, mac-notarize]
```

## Limitations

1. Build step (artifact signing) must be executed on an OSX machine.
2. Sign step (artifact notarization) must be executed on an OSX machine. While `alttool` is now deprecated, [gon doesn't yet support][gon notarize api] the new API.
3. Every build (including snapshots!) are signed. It's possible we could skip this but it'd require a bit more [scripting][^1] during sign.
4. Notarized artifacts don't seem to validate via `spctl --assess`, it is unclear why. **Improvements welcome!**
5. Notariziation is sequential, instead of running in parallel which gon can perform. Due to the way signing is triggered in this step, there's no real way to address this.
6. Loading and referencing the keychain in GitHub CI has an issue because the keychain password is unavailable. Create an ephemeral keychain and reference that instead, see [anchore blog entry][anchore blog entry] on this issue.
7. 
## Real Example

Including:

- go module support
- reproducible build flags
- signing
- notarization
- releasing

```yaml
# .goreleaser.yml
project_name: myapp
before:
  hooks:
    - go mod tidy
    - go mod download

env:
  - CGO_ENABLED=0
  # Signing parameters. Check gon's documentation on what the appropriate values should be.
  - BUNDLE_ID=com.example.myapp
  - APPLE_ID_USERNAME=dekimsey@example.com
  - APPLE_ID_PASSWORD=@keychain:m1-gon
  - "APPLE_APPLICATION_IDENTITY=Developer ID Application: Daniel Kimsey"

builds:
  - id: default
    goos: [linux, windows]
    goarch: [arm64, amd64]
    flags:
      - -trimpath
    ldflags:
      - -s -w -X main.version={{.Version}}
    ignore:
      - goarch: arm
        goos: windows
      - goarch: arm64
        goos: windows
    mod_timestamp: "{{ .CommitTimestamp }}"
  - id: macos
    goos: [darwin]
    goarch: [arm64, amd64]
    flags:
      - -trimpath
    ldflags:
      - -s -w -X main.version={{.Version}}
    mod_timestamp: "{{ .CommitTimestamp }}"
    hooks:
      post:
        - |
          sh -c '
          fn=dist/macos_{{.Target}}/gon.hcl
          cat >"$fn" <<EOF
          bundle_id = "{{.Env.BUNDLE_ID}}"
          apple_id {
            username = "{{.Env.APPLE_ID_USERNAME}}"
            password = "{{.Env.APPLE_ID_PASSWORD}}"
          }
          source = ["dist/macos_{{.Target}}/{{.Name}}"]
          sign {
            application_identity = "{{.Env.APPLE_APPLICATION_IDENTITY}}"
          }
          EOF
          '
        - "gon 'dist/macos_{{.Target}}/gon.hcl'"

archives:
  - format: zip
    builds: [default]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: linux
        format: tar.gz
  - id: macos
    builds: [macos]
    format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

signs:
  - id: mac-notarize
    artifacts: archive
    ids: [macos]
    signature: "${artifact}"
    output: true
    cmd: sh
    args:
      - "-c"
      - |-
        cat >"dist/gon.notarize.hcl" <<EOF
        apple_id {
          username = "{{.Env.APPLE_ID_USERNAME}}"
          password = "{{.Env.APPLE_ID_PASSWORD}}"
        }
        notarize {
          path = "${artifact}"
          bundle_id = "{{.Env.BUNDLE_ID}}"
        }
        EOF
        gon "dist/gon.notarize.hcl"

release:
  ids:
    - default
    - mac-notarize

```


## More information

You can find more information about this in the [discussion][discussion] that originated it.


[discussion]: https://github.com/goreleaser/goreleaser/discussions/3350
[gon configuration]: https://github.com/mitchellh/gon#configuration-file
[gon notarize api]: https://github.com/mitchellh/gon/issues/45
[anchore blog entry]: https://medium.com/anchore-engineering/developers-need-to-handle-macos-binary-signing-how-we-automated-the-solution-part-2-ad1e08caff0f
[^1]: One would need to extract the binary from goreleaser's Artifact stage, sign it with gon, put the signed binary back in the archive, and run gon notarize against the new archive. This is quite messy as it's highly dependent on the archive format tooling and the naming, but likely possible.
