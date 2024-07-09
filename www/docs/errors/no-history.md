# Changelog has only the latest commit

If your changelog has only a single commit, its likely GoReleaser ran against
a shallow clone, so the history isn't really there - you need a clone with
the full depth for it to work.

To fix it, please refer to the [CI section](../ci/index.md) of our docs.
