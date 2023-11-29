{ pkgs ? import <nixpkgs> { } }:
let
  mkdocs-material = pkgs.python311Packages.buildPythonPackage
    {
      pname = "mkdocs-material";
      version = "9.4.14";
      src = pkgs.fetchFromGitHub {
        owner = "squidfunk";
        repo = "mkdocs-material";
        rev = "9.4.14";
        sha256 = "sha256-oP0DeSRgoLx6boEOa3if5BitGXmJ11DoUVZK16Sjlwg=";
      };
      doCheck = false;
      pythonImportsCheck = [
        "mkdocs"
      ];
      propagatedBuildInputs = [
        pkgs.python311Packages.mkdocs
        pkgs.python311Packages.hatchling
      ];
    };
  git-python = pkgs.python311Packages.buildPythonPackage
    {
      pname = "git-python";
      version = "3.1.40";
      src = pkgs.fetchFromGitHub {
        owner = "gitpython-developers";
        repo = "GitPython";
        rev = "3.1.40";
        sha256 = "sha256-a5Ez6SuSqrJE306FrFjEnSoVhALVvubF1pLW4awK4gM=";
      };
      doCheck = false;
      propagatedBuildInputs = [
        pkgs.python311Packages.gitdb
        pkgs.python311Packages.ddt
        pkgs.python311Packages.pytest
      ];
    };
  mkdocs-rss = pkgs.python311Packages.buildPythonPackage
    {
      pname = "mkdocs-rss";
      version = "1.8.0";
      src = pkgs.fetchFromGitHub {
        owner = "Guts";
        repo = "mkdocs-rss-plugin";
        rev = "1.8.0";
        sha256 = "sha256-rCz1Uk5uqIsnIWw0b1oBsjAO6aK/tpVgqAX/8dVnAGw=";
      };
      doCheck = false;
      propagatedBuildInputs = [
        pkgs.python311Packages.mkdocs
        git-python
      ];
    };
  mkdocs-include-markdown = pkgs.python311Packages.buildPythonPackage {
    pname = "mkdocs-include-markdown-plugin";
    version = "6.0.4";
    src = pkgs.fetchFromGitHub {
      owner = "mondeja";
      repo = "mkdocs-include-markdown-plugin";
      rev = "v6.0.4";
      sha256 = "sha256-wHaDvF+QsEa3G5+q1ZUQQpVmwy+oRsSEq2qeJIJjFeY=";
    };
    format = "pyproject";
    doCheck = false;
    propagatedBuildInputs = [
      pkgs.python311Packages.mkdocs
      pkgs.python311Packages.hatchling
      pkgs.python311Packages.wcmatch
    ];
  };
in
pkgs.mkShell
{
  packages = with pkgs;
    [
      go
      go-task
      gofumpt

      pkgs.python311Packages.mkdocs
      python311Packages.mkdocs-minify
      python311Packages.mkdocs-redirects
      mkdocs-material
      mkdocs-rss
      mkdocs-include-markdown
    ];
}
