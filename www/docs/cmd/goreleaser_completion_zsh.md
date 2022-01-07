# goreleaser completion zsh

Generate the autocompletion script for zsh

## Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for every new session, execute once:

### Linux:

	goreleaser completion zsh > "${fpath[1]}/_goreleaser"

### macOS:

	goreleaser completion zsh > /usr/local/share/zsh/site-functions/_goreleaser

You will need to start a new shell for this setup to take effect.


```
goreleaser completion zsh [flags]
```

## Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

## Options inherited from parent commands

```
      --debug   Enable debug mode
```

## See also

* [goreleaser completion](/cmd/goreleaser_completion/)	 - Generate the autocompletion script for the specified shell

