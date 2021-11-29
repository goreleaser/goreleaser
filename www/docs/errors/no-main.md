# Build does not contain a main function

This usually happens if you're trying to build a library or if you didn't setup the `builds.main` section in your `.goreleaser.yml` and you `main.go` is not in the root folder.

## If you are building a library

Add something like this to your config:

```yaml
# .goreleaser.yml
builds:
- skip: true
```

## If your `main.go` is not in the root folder

Add something like this to your config:

```yaml
# .goreleaser.yml
builds:
- main: ./path/to/your/main/pkg/
```

For more info, check the [builds documentation](/customization/build/).
