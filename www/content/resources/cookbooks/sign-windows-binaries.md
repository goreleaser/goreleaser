---
title: "Signing Windows binaries with a post hook"
weight: 140
---

Windows binaries that are not Authenticode-signed trigger a SmartScreen warning
on first run. GoReleaser's `signs` section produces detached signature files,
which is not what Windows checks. Windows needs the signature embedded in the
`.exe` itself, and the place to do that is a build `post` hook, which runs
after the binary is compiled and before it is archived.

Because hooks run for every build target, keep the Windows targets in their own
build so the hook only runs on `.exe` files:

```yaml
# .goreleaser.yaml
builds:
  - id: unix
    goos: [linux, darwin]
    goarch: [amd64, arm64]

  - id: windows
    goos: [windows]
    goarch: [amd64, arm64]
    hooks:
      post:
        - osslsigncode sign -pkcs12 cert.pfx -pass "{{ .Env.CERT_PASSWORD }}" -n "{{ .ProjectName }}" -t http://timestamp.digicert.com -in "{{ .Path }}" -out "{{ .Path }}.signed"
        - mv "{{ .Path }}.signed" "{{ .Path }}"
```

The example above uses [osslsigncode](https://github.com/mtrojnar/osslsigncode)
with a certificate file, which works on Linux runners. Since 2023 most
certificate authorities only issue code signing keys on hardware tokens or in a
cloud HSM, so a plain `.pfx` is often not an option; in that case the hook
calls whatever tool fronts the key instead. A hosted signing service that
signs the file in place is one line:

```yaml
    hooks:
      post:
        - npx -y github:bamboodeploy/cli#v1.1.0 sign "{{ .Path }}"
```

Two things to keep in mind:

- Hooks are executed without a shell, so pipes and redirects do not work; put
  anything more complex in a script and call the script.
- The hook must leave the signed file at `{{ .Path }}`, because that is what the
  archive step picks up afterwards.
