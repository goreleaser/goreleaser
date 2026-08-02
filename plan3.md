Let's see what happens if `thin-manifests = true`. Then the manifest only contains `DIST`.
If `thin-manifests = false` or not present, it contains `EBUILD`, `AUX`, `MISC`, and `DIST`.
Wait, usually `thin-manifests = false` requires the extra entries. We must look for `thin-manifests` in `layout.conf`.

1. Add variable `thinManifests := false` (or `true`? wait, default in Gentoo overlays used to be thick unless specified `thin-manifests = true`. Currently `masters = gentoo` inherits `thin-manifests = true` if `gentoo` has it? No, wait, if `thin-manifests` is not specified, what is the default? The issue says: "overlays that do not enable thin-manifests also require records for the new ebuild... The code reads layout.conf solely for hash names and never checks thin-manifests, so publishing to a default or explicitly non-thin overlay produces an incomplete Manifest")
We should parse `thin-manifests` from `layout.conf`.
Wait, is default false or true? If they say "overlays that do not enable thin-manifests", it implies default is false (thick). If `thin-manifests = true` is found, then it's thin.
