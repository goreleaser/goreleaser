# Semantic Release

GoReleaser does not create any tags, it just runs on what is already there.

You can, though, leverage other tools to do the work for you, like for example
[svu](https://github.com/caarlos0/svu):

```bash
git tag "$(svu next)"
git push --tags
goreleaser --rm-dist
```
