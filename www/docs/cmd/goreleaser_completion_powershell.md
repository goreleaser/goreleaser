# goreleaser completion powershell

Generate the autocompletion script for powershell

## Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	goreleaser completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
goreleaser completion powershell [flags]
```

## Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

## Options inherited from parent commands

```
      --debug   Enable debug mode
```

## See also

* [goreleaser completion](/cmd/goreleaser_completion/)	 - Generate the autocompletion script for the specified shell

