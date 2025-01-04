# Builds: using complex template environment variables

Builds' environment variables accept templates.

You can leverage that to have a single build configuration with different
environment variables for each platform, for example.

A common example of this is the variables `CC` and `CXX`.

Here are two different examples:

## Using multiple envs

This example creates once `CC_` and `CXX_` variable for each platform, and then
set `CC` and `CXX` to the right one:

```yaml title=".goreleaser.yaml"
builds:
  - id: mybin
    binary: mybin
    main: .
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    env:
      - CGO_ENABLED=0
      - CC_darwin_amd64=o64-clang
      - CXX_darwin_amd64=o64-clang+
      - CC_darwin_arm64=aarch64-apple-darwin20.2-clang
      - CXX_darwin_arm64=aarch64-apple-darwin20.2-clang++
      - CC_windows_amd64=x86_64-w64-mingw32-gc
      - CXX_windows_amd64=x86_64-w64-mingw32-g++
      - 'CC={{ index .Env (print "CC_" .Os "_" .Arch) }}'
      - 'CXX={{ index .Env (print "CXX_" .Os "_" .Arch) }}'
```

## Using `if` statements

This example uses `if` statements to set `CC` and `CXX`:

```yaml title=".goreleaser.yaml"
builds:
  - id: mybin
    binary: mybin
    main: .
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    env:
      - CGO_ENABLED=0
      - >-
        {{- if eq .Os "darwin" }}
          {{- if eq .Arch "amd64"}}CC=o64-clang{{- end }}
          {{- if eq .Arch "arm64"}}CC=aarch64-apple-darwin20.2-clang{{- end }}
        {{- end }}
        {{- if eq .Os "windows" }}
          {{- if eq .Arch "amd64" }}CC=x86_64-w64-mingw32-gcc{{- end }}
        {{- end }}
      - >-
        {{- if eq .Os "darwin" }}
          {{- if eq .Arch "amd64"}}CXX=o64-clang+{{- end }}
          {{- if eq .Arch "arm64"}}CXX=aarch64-apple-darwin20.2-clang++{{- end }}
        {{- end }}
        {{- if eq .Os "windows" }}
          {{- if eq .Arch "amd64" }}CXX=x86_64-w64-mingw32-g++{{- end }}
        {{- end }}
```
