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
        devShells. default = pkgs.mkShell {
          packages = with pkgs; with staging-pkgs.python311Packages;[
            go
            go-task
            gofumpt

            mkdocs-material
            mkdocs-redirects
            mkdocs-minify
            mkdocs-rss-plugin
            mkdocs-include-markdown-plugin
          ] ++ mkdocs-material.passthru.optional-dependencies.git;
        };
      }
    );
}

