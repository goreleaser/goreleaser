#!/bin/sh
NEW_HASH="$(nix-prefetch \
	--option extra-experimental-features flakes \
	'{ sha256 }: (builtins.getFlake (toString ./.)).packages.x86_64-linux.default.goModules.overrideAttrs (_: { vendorSha256 = sha256; })')"

sed -i "s|vendorHash = \".*\"|vendorHash = \"${NEW_HASH}\"|" ./flake.nix
