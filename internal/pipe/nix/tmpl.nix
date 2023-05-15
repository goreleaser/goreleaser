{ stdenv, fetchurl, installShellFiles }:
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
in stdenv.mkDerivation {
  pname = "{{ .Name }}";
  version = "{{ .Version }}";
  src = fetchurl {
    url = urlMap.${builtins.currentSystem};
    sha256 = shaMap.${builtins.currentSystem};
  };

  sourceRoot = "{{ .SourceRoot }}";

  nativeBuildInputs = [ installShellFiles ];

  installPhase = ''
    {{ .Install }}
  '';

  system = builtins.currentSystem;
}
