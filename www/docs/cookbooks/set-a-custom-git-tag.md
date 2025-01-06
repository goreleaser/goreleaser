# Setting a custom git tag

You can override the current and previous tags by setting some environment
variables. This can be useful in cases where one git commit is referenced by
multiple git tags, for example.

Example usage:

```sh
export GORELEASER_CURRENT_TAG=v1.2.3
export GORELEASER_PREVIOUS_TAG=v1.1.0
goreleaser release
```
