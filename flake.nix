{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    carlos.url = "github:caarlos0/nur";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { nixpkgs, carlos, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        cpkgs = carlos.packages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "goreleaser";
          version = "unversioned";
          src = ./.;
          ldflags = [ "-s" "-w" "-X main.version=dev" "-X main.builtBy=flake" ];
          doCheck = false;
          vendorHash = "";
        };

        devShells.default = pkgs.mkShellNoCC {
          shellHook = "go mod tidy";
        };

        # nix develop .#dev
        devShells.dev = pkgs.mkShellNoCC {
          packages = with pkgs; [
            go-task
            gofumpt
            syft
            upx
            cosign
            gnupg
            nix-prefetch
          ] ++ (lib.optionals pkgs.stdenv.isLinux [
            snapcraft
          ]);
        };

        # nix develop .#docs
        devShells.docs = pkgs.mkShellNoCC {
          packages = with pkgs; with pkgs.python311Packages; [
            go-task
            htmltest
            mkdocs-material
            mkdocs-redirects
            mkdocs-minify
            mkdocs-rss-plugin
            filelock
            cpkgs.mkdocs-include-markdown-plugin # https://github.com/NixOS/nixpkgs/pull/277351
          ] ++ mkdocs-material.passthru.optional-dependencies.git;
        };
      }
    );
}

