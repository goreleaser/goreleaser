## goreleaser completion

Prints shell autocompletion scripts for GoReleaser

### Synopsis

Prints shell autocompletion scripts for GoReleaser.

To load completions:

#### Bash:

	$ source <(goreleaser completion bash)

To load completions for each session, execute once:

##### Linux:

	$ goreleaser completion bash > /etc/bash_completion.d/goreleaser

##### MacOS:

	$ goreleaser completion bash > /usr/local/etc/bash_completion.d/goreleaser

#### Zsh:

If shell completion is not already enabled in your environment you will need to enable it.
You can execute the following once:

	$ echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for each session, execute once:

	$ goreleaser completion zsh > "${fpath[1]}/_goreleaser"

You will need to start a new shell for this setup to take effect.

#### Fish:

	$ goreleaser completion fish | source

To load completions for each session, execute once:

	$ goreleaser completion fish > ~/.config/fish/completions/goreleaser.fish

If you are using an official GoReleaser package, it should do this for you automatically.

---


```
goreleaser completion [bash|zsh|fish]
```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --debug   Enable debug mode
```

### SEE ALSO

* [goreleaser](goreleaser.md)	 - Deliver Go binaries as fast and easily as possible

