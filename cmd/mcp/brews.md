Follow all of these, in order:

- the `install` block should be removed if present, using the following syntax instead:

```yaml
binary: binary_name
completions:
  bash: filepath.bash
  zsh: filepath.zsh
  fish: filepath.fish
manpages:
  - filepath.1
```

- rename the root from `brews` to `homebrew_casks`
- remove the `license` field
- remove the `goarm` and `goamd64` fields
- move `url_headers` and `url_template` into a new `url` block:

```yaml
url:
  template: "the URL template value"
  headers:
    - header 1
    - header 2
```

- remove the `plist` field
- comment out the `service` field, informing that it should now point to a `.service` file that will be placed in the `~/Library/Services/` folder
- remove the `directory` option
- change the `dependencies` field, it should now follow this format:

```yaml
dependencies:
  - cask: some-cask
  - formula: some-formula
```

Assume all previously listed dependencies are of type 'formula'.

- change the `conflicts` field, it should now follow this format:

```yaml
conflicts:
  - cask: some-cask
  - formula: some-formula
```

Assume all previously listed dependencies are of type 'formula'.

- if the configuration don't have notarization enabled, add the following hook to the `homebrew_casks`:

```yaml
hooks:
  post:
    install: |
      on_macos do
        if system_command("/usr/bin/xattr", args: ["-h"]).exit_status == 0 # replace 'foo' with the actual binary name
          system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/foo"]
        end
      end
```

---

See https://goreleaser.com/customization/homebrew_casks/ for more the complete reference.
