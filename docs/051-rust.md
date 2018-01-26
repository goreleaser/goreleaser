---
title: Rust
---

To compile Rust applications
Rust builds can be customized in multiple ways.
If your setup supports cross compilation, multiple targets can be entered under the `target` field.

Prerequisites:

* A running Rust environment with `cargo` installed

Here is a commented `rust` section with all fields specified:

```yml
# .goreleaser.yml
rust:
  # You can have multiple builds defined as a yaml list
  -
    # Name of the binary.
    # Default is the name of the project directory.
    binary: program

    # Custom environment variables to be set during the builds.
    # Default is empty.
    # For Cargo useful environment variables can be found at
    # https://doc.rust-lang.org/cargo/reference/environment-variables.html
    env:
      - KEY=val

    # target list to build for.
    # For more info refer to: https://forge.rust-lang.org/platform-support.html
    # No defaults are set.
    # Without targets, the Rust pipe is not able to operate.
    # To get all targets, execute `rustc --print target-list`
    # To get all supported targets for your computer, run `rustup show`
    target:
      - x86_64-apple-darwin

    # Hooks can be used to customize the final binary,
    # for example, to run some scripts.
    # Default is both hooks empty.
    hooks:
      pre: cargo clean
      post: ./post-script.sh
```
