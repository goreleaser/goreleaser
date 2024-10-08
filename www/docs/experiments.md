# Experimental features

Much like Go, GoReleaser can be told to use an experimental feature via
environment variables.

Currently, the following experiments are available.

## Default `GOARM` change to `7`

<!-- md:version v2.4 -->

Historically, GoReleaser sets `GOARM` to `6` by default.
You can make it use `7` instead by setting:

```sh
export GORELEASER_EXPERIMENTAL="defaultgoarm"
```

This will be default behavior in GoReleaser v3.

!!! tip

    You can also set the `GORELEASER_EXPERIMENTAL` variable in `env` array in
    your `.goreleaser.yml`.
