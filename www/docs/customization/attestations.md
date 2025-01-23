# Attestations

If you're using GitHub Actions and want to attest your build artifacts, you can
do add the following to your release workflow:

```yaml title=".github/workflows/release.yml"
# ...
permissions:
  # ...
  # Give the workflow permission to write attestations.
  attestations: write

jobs:
  release:
    # ...
    steps:
      # ...
      - uses: goreleaser/goreleaser-action@v6
        with:
          # ...
      # After GoReleaser runs, attest all the files in ./dist/checksums.txt:
      - uses: actions/attest-build-provenance@v2
        with:
          subject-checksums: ./dist/checksums.txt
```

Users can then verify it with:

```bash
gh attestation verify --owner <user-or-org> <filename>
```

Refer to [this repository](https://github.com/goreleaser/example-supply-chain)
for an example, as well as signing, SBOMs, and more.
