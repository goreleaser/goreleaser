# Build does not contain a main function

This usually happens if you're trying to build a library or if you didn't setup
the `builds.main` section in your `.goreleaser.yaml` and your `main.go` is not
in the root directory.

Here's an example error:

```sh
   ⨯ build failed after 0.11s error=build for foo does not contain a main function

Learn more at https://goreleaser.com/errors/no-main
```

## If you are building a library

Add something like this to your config:

```yaml
# .goreleaser.yaml
builds:
  - skip: true
```

## If your `main.go` is not in the root directory

Add something like this to your config:

```yaml
# .goreleaser.yaml
builds:
  - main: ./path/to/your/main/pkg/
```

For more info, check the [builds documentation](../customization/builds.md).

## If you ran goreleaser outside the root of the project

Run goreleaser in the root of the project.

## If you are building in `plugin`, `c-shared` or `c-archive` build modes

You can set `no_main_check` to `true`:

```yaml
# .goreleaser.yaml
builds:
  - main: ./path/...
    buildmode: plugin
    no_main_check: true
```

For more info, check the [builds documentation](../customization/builds.md).
