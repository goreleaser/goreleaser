{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    staging.url = "github:caarlos0/nixpkgs/wip";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { self, nixpkgs, staging, flake-utils, ... }:
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
          vendorHash = "sha256-wY3kIhNIqTaK9MT1VeePERNhqvbtf6bsyRTjG8nrqxU=";
        };
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

          shellHook = ''
            		  go mod tidy
            		  '';
        };
      }
    );
}

