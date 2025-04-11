{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
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
      }
    );
}
