{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    carlos.url = "github:caarlos0/nur";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      nixpkgs,
      carlos,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        cpkgs = carlos.packages.${system};
      in
      {
        devShells.default = pkgs.mkShellNoCC {
          shellHook = "go mod tidy";
        };

        # nix develop .#dev
        devShells.dev = pkgs.mkShellNoCC {
          packages =
            with pkgs;
            [
              go-task
              gofumpt
              syft
              upx
              cosign
              gnupg
              rustup
              zig
              bun
              deno
            ]
            ++ (lib.optionals pkgs.stdenv.isLinux [
              snapcraft
            ]);
        };

        # nix develop .#docs
        devShells.docs = pkgs.mkShellNoCC {
          packages =
            with pkgs;
            [
              go-task
              htmltest
            ]
            ++ (with cpkgs; [
              mkdocs-git-revision-date-localized-plugin
              mkdocs-include-markdown-plugin # https://github.com/NixOS/nixpkgs/pull/277351
            ])
            ++ (with pkgs.python312Packages; [
              regex
              mkdocs-material
              mkdocs-redirects
              mkdocs-minify
              mkdocs-rss-plugin
              filelock
            ]);
        };
      }
    );
}
