{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    staging.url = "github:caarlos0/nixpkgs/wip";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { nixpkgs, staging, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        staging-pkgs = staging.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "goreleaser";
          version = "unversioned";
          src = ./.;
          ldflags = [ "-s" "-w" "-X main.version=dev" "-X main.builtBy=flake" ];
          doCheck = false;
          vendorHash = "sha256-2CoQuiv8lVjdNJwwuX/rezoHRaMph0AsptLAudztqF8=";
        };

        devShells.default = pkgs.mkShellNoCC {
          packages = with pkgs; [
            go_1_22
            go-task
            gofumpt
            syft
            upx
            cosign
            gnupg
            nix-prefetch
          ];
          shellHook = "go mod tidy";
        };

        devShells.docs = pkgs.mkShellNoCC {
          packages = with pkgs; with pkgs.python311Packages; [
            go-task
            htmltest
            mkdocs-material
            mkdocs-redirects
            mkdocs-minify
            staging-pkgs.pkgs.python311Packages.mkdocs-rss-plugin # https://github.com/NixOS/nixpkgs/pull/277350
            staging-pkgs.pkgs.python311Packages.mkdocs-include-markdown-plugin # https://github.com/NixOS/nixpkgs/pull/277351
          ] ++ mkdocs-material.passthru.optional-dependencies.git;
        };
      }
    );
}

