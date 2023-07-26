# This file was generated by GoReleaser. DO NOT EDIT.
# vim: set ft=nix ts=2 sw=2 sts=2 et sta
{
system ? builtins.currentSystem
, pkgs
, lib
, fetchurl
, installShellFiles
{{- if .Dependencies }}
, makeWrapper
, stdenv
{{- end -}}
{{- range $index, $element := .Dependencies }}
, {{ . -}}
{{- end }}
}:
let
  shaMap = {
    {{- with  .Archives.linux386.Sha }}
    i686-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.linuxamd64.Sha }}
    x86_64-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.linuxarm6.Sha }}
    armv6l-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.linuxarm7.Sha }}
    armv7l-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.linuxarm64.Sha }}
    aarch64-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.darwinamd64.Sha }}
    x86_64-darwin = "{{ . }}";
    {{- end }}
    {{- with  .Archives.darwinarm64.Sha }}
    aarch64-darwin = "{{ . }}";
    {{- end }}
  };

  urlMap = {
    {{- with  .Archives.linux386.URL }}
    i686-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.linuxamd64.URL }}
    x86_64-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.linuxarm6.URL }}
    armv6l-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.linuxarm7.URL }}
    armv7l-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.linuxarm64.URL }}
    aarch64-linux = "{{ . }}";
    {{- end }}
    {{- with  .Archives.darwinamd64.URL }}
    x86_64-darwin = "{{ . }}";
    {{- end }}
    {{- with  .Archives.darwinarm64.URL }}
    aarch64-darwin = "{{ . }}";
    {{- end }}
  };
in
pkgs.stdenv.mkDerivation {
  pname = "{{ .Name }}";
  version = "{{ .Version }}";
  src = fetchurl {
    url = urlMap.${system};
    sha256 = shaMap.${system};
  };

  sourceRoot = "{{ .SourceRoot }}";

  nativeBuildInputs = [ {{ range $input, $plat := .Inputs }}{{ . }} {{ end }}];

  installPhase = ''
    {{- range $index, $element := .Install }}
    {{ . -}}
    {{- end }}
  '';

  {{- with .PostInstall }}
  postInstall = ''
    {{- range $index, $element := . }}
    {{ . -}}
    {{- end }}
  '';
  {{- end }}

  system = system;

  meta = {
    {{- with .Description }}
    description = "{{ . }}";
    {{- end }}
    {{- with .Homepage }}
    homepage = "{{ . }}";
    {{- end }}
    {{- with .License }}
    license = lib.licenses.{{ . }};
    {{- end }}

    platforms = [
      {{- range $index, $plat := .Platforms }}
      "{{ . }}"
      {{- end }}
    ];
  };
}
