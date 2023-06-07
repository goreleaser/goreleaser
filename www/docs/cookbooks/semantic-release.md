# Semantic Release

GoReleaser does not create any tags, it just runs on what is already there.

You can, though, leverage other tools to do the work for you, like for example
[svu](https://github.com/caarlos0/svu) or [semantic-release](https://github.com/semantic-release/semantic-release).

## Example: svu

```bash
git tag "$(svu next)"
git push --tags
goreleaser release --clean
```

## Example: semantic-release

.releaserc.yml

```yaml
preset: angular
plugins:
  - "@semantic-release/commit-analyzer"
  - "@semantic-release/release-notes-generator"
  - "@semantic-release/changelog"
  - "@semantic-release/git"
  - "@semantic-release/exec"
    - publishCmd: |
        echo "${nextRelease.notes}" > /tmp/release-notes.md
        goreleaser release --release-notes /tmp/release-notes.md --clean
```

```bash
npx -p @semantic-release/changelog -p @semantic-release/exec -p @semantic-release/git semantic-release
```
