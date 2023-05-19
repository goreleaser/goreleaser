{ system ? builtins.currentSystem, pkgs, lib, fetchurl, installShellFiles }:
let
  shaMap = {
    x86_64-linux = "{{ .Archives.linuxamd64.Sha }}";
    aarch64-linux = "{{ .Archives.linuxarm64.Sha }}";
    x86_64-darwin = "{{ .Archives.darwinamd64.Sha }}";
    aarch64-darwin = "{{ .Archives.darwinarm64.Sha }}";
  };

  urlMap = {
    x86_64-linux = "{{ .Archives.linuxamd64.URL }}";
    aarch64-linux = "{{ .Archives.linuxarm64.URL }}";
    x86_64-darwin = "{{ .Archives.darwinamd64.URL }}";
    aarch64-darwin = "{{ .Archives.darwinarm64.URL }}";
  };
in pkgs.stdenv.mkDerivation {
  pname = "{{ .Name }}";
  version = "{{ .Version }}";
  src = fetchurl {
    url = urlMap.${system};
    sha256 = shaMap.${system};
  };

  sourceRoot = "{{ .SourceRoot }}";

  nativeBuildInputs = [ installShellFiles ];

  installPhase = ''
    {{- range $index, $element := .Install }}
    {{ . -}}
    {{- end }}
  '';

  system = system;

  meta = with lib; {
    description = "{{ .Description }}";
    homepage = "{{ .Homepage }}";
    license = licenses.{{.License}};

    platforms = [
      {{- range $index, $plat := .Platforms }}
      "{{ . }}"
      {{- end }}
    ];
  };
}
