# goreleaser completion fish

Generate the autocompletion script for fish

## Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	goreleaser completion fish | source

To load completions for every new session, execute once:

	goreleaser completion fish > ~/.config/fish/completions/goreleaser.fish

You will need to start a new shell for this setup to take effect.


```
goreleaser completion fish [flags]
```

## Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

## Options inherited from parent commands

```
      --debug   Enable debug mode
```

## See also

* [goreleaser completion](/cmd/goreleaser_completion/)	 - Generate the autocompletion script for the specified shell

