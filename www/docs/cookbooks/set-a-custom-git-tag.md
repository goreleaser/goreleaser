# Setting a custom git tag

You can force the [build tag](/customization/build/#define-build-tag) and [previous changelog tag](/customization/release/#define-previous-tag) using environment variables.
This can be useful in cases where one git commit is referenced by multiple git tags.

Example usage:

```sh
export GORELEASER_CURRENT_TAG=v1.2.3
export GORELEASER_PREVIOUS_TAG=v1.1.0
goreleaser release
```
