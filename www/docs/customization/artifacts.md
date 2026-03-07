# Artifacts

GoReleaser generates an `artifacts.json` file in the `dist` folder containing
information about all artifacts produced during the release.

This file is useful for integrating with other tools, such as `jq`, to query
information about the release artifacts.

## Structure

Each artifact in the `artifacts.json` file has the following fields:

| Field       | Description                                                      |
| ----------- | ---------------------------------------------------------------- |
| `name`      | The artifact filename                                            |
| `path`      | The relative path to the artifact                                |
| `goos`      | The target operating system (e.g., `linux`, `darwin`, `windows`) |
| `goarch`    | The target architecture (e.g., `amd64`, `arm64`, `386`)          |
| `goamd64`   | The amd64 microarchitecture level (e.g., `v1`, `v2`, `v3`)       |
| `go386`     | The 386 floating point instruction set                           |
| `goarm`     | The ARM version (e.g., `6`, `7`)                                 |
| `goarm64`   | The ARM64 version                                                |
| `gomips`    | The MIPS floating point instruction set                          |
| `goppc64`   | The PPC64 version                                                |
| `goriscv64` | The RISC-V 64 version                                            |
| `target`    | The full build target (e.g., `linux_amd64_v1`)                   |
| `type`      | The artifact type (see below)                                    |
| `extra`     | Additional metadata (see below)                                  |

## Artifact types

The `type` field indicates what kind of artifact it is:

| Type                     | Description                                |
| ------------------------ | ------------------------------------------ |
| `Archive`                | A compressed archive (tar.gz, zip, etc.)   |
| `Binary`                 | A compiled binary                          |
| `File`                   | A generic uploadable file                  |
| `Linux Package`          | A package created by nfpm (deb, rpm, etc.) |
| `Snap`                   | A Snapcraft package                        |
| `Docker Image`           | A Docker image                             |
| `Published Docker Image` | A published Docker image                   |
| `Docker Manifest`        | A Docker manifest                          |
| `Checksum`               | A checksums file                           |
| `Signature`              | A signature file                           |
| `Certificate`            | A signing certificate                      |
| `Source`                 | A source archive                           |
| `Homebrew Formula`       | A Homebrew formula file                    |
| `Homebrew Cask`          | A Homebrew cask file                       |
| `Krew Plugin Manifest`   | A Krew plugin manifest                     |
| `Scoop Manifest`         | A Scoop manifest file                      |
| `SBOM`                   | A Software Bill of Materials               |
| `PKGBUILD`               | An Arch Linux PKGBUILD file                |
| `SRCINFO`                | An Arch Linux .SRCINFO file                |
| `Chocolatey`             | A Chocolatey package                       |
| `C Header`               | A C header file                            |
| `C Archive Library`      | A C static library                         |
| `C Shared Library`       | A C shared library                         |
| `Winget Manifest`        | A Winget manifest file                     |
| `Nixpkg`                 | A Nix package                              |
| `Wheel`                  | A Python wheel package                     |
| `Source Dist`            | A Python source distribution               |
| `Makeself Package`       | A Makeself self-extracting archive         |
| `App Bundle`             | A macOS .app bundle                        |
| `DMG`                    | A macOS disk image                         |
| `MacOS Package`          | A macOS installer package                  |
| `MSI`                    | A Windows MSI installer                    |
| `NPM Package`            | An NPM package                             |

## Extra fields

The `extra` field contains additional metadata that varies by artifact type.
The most common fields are:

| Field               | Type       | Description                                                |
| ------------------- | ---------- | ---------------------------------------------------------- |
| `ID`                | `string`   | The artifact ID from the configuration                     |
| `Binary`            | `string`   | The binary name (for archives with a single binary)        |
| `Binaries`          | `[]string` | List of binary names (for archives with multiple binaries) |
| `Ext`               | `string`   | The file extension (including the leading `.`)             |
| `Format`            | `string`   | The archive format (e.g., `tar.gz`, `zip`)                 |
| `WrappedIn`         | `string`   | The directory name the files are wrapped in                |
| `Checksum`          | `string`   | The checksum in `algorithm:hash` format                    |
| `Size`              | `int`      | The file size in bytes (when `report_sizes` is enabled)    |
| `Digest`            | `string`   | The Docker image digest                                    |
| `Replaces`          | `bool`     | Whether a universal binary replaces single-arch ones       |
| `Files`             | `[]string` | Any extra files an archive might have                      |
| `DynamicallyLinked` | `bool`     | Whether or not the binary is dynamically linked            |

!!! note

    There might be other fields in `extra` depending on the artifact type and
    configuration. The fields listed above are the most commonly used ones
    across multiple artifact types.

## Example

Here's an example of what an artifact entry looks like:

```json
{
  "name": "myapp_1.0.0_linux_amd64.tar.gz",
  "path": "dist/myapp_1.0.0_linux_amd64.tar.gz",
  "goos": "linux",
  "goarch": "amd64",
  "goamd64": "v1",
  "type": "Archive",
  "extra": {
    "Binaries": ["myapp"],
    "Checksum": "sha256:abc123...",
    "Format": "tar.gz",
    "ID": "default"
  }
}
```
