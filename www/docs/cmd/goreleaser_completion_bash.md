# goreleaser completion bash

Generate the autocompletion script for bash

## Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(goreleaser completion bash)

To load completions for every new session, execute once:

### Linux:

	goreleaser completion bash > /etc/bash_completion.d/goreleaser

### macOS:

	goreleaser completion bash > /usr/local/etc/bash_completion.d/goreleaser

You will need to start a new shell for this setup to take effect.


```
goreleaser completion bash
```

## Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

## Options inherited from parent commands

```
      --debug   Enable debug mode
```

## See also

* [goreleaser completion](/cmd/goreleaser_completion/)	 - Generate the autocompletion script for the specified shell

