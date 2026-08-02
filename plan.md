1. Modify `handleGentooManifestAndMetadata` in `internal/pipe/gentoo/gentoo.go` to parse `thin-manifests` from `layout.conf`.
2. If `thin-manifests = false` (or explicitly checking that it is NOT `thin-manifests = true`), generate Manifest entries for `EBUILD`, `AUX` (files in the `files/` directory), and `MISC` (like `metadata.xml`).
3. For these types, the format of the Manifest entry is: `<TYPE> <filename> <size> <HASH1> <HASHVAL1>...`
4. The file names should just be the base name of the file (e.g. `foo-1.0.ebuild`, `foo.patch`, `metadata.xml`).
5. When filtering out deleted ebuilds from the Manifest, we also need to remove their `EBUILD` lines in the Manifest if we are updating it. Actually, `handleGentooManifestAndMetadata` already parses the existing Manifest and removes `DIST` entries for deleted ebuilds. We should extend that logic to remove `EBUILD` entries for deleted ebuilds too. Wait, if an ebuild is deleted, its `EBUILD` entry should be removed. We should simply re-hash ALL currently existing EBUILD, AUX, and MISC files that are part of this commit, and combine them with the untouched `DIST` entries (since `DIST` relies on downloading files, while we only want to keep the valid `DIST` entries). Wait, what if there are other ebuilds in the directory? We don't have their content in memory to rehash them!
   Wait, if we are NOT on thin-manifests, and there are old ebuilds, how can we re-hash them if we didn't download them?
   If `thin-manifests` is false, and there are old ebuilds, we might need to keep their `EBUILD` entries from the old Manifest unless they are in the `deletedEbuilds` list.
   Yes! So the process for Manifest lines:
   - For old lines:
     - Keep `DIST` lines, unless they are associated with a deleted ebuild (this logic already exists).
     - Keep `EBUILD` lines, UNLESS the filename (field 2) matches a deleted ebuild OR matches a newly generated ebuild (we will replace it).
     - Keep `AUX` lines, UNLESS the filename matches a deleted file OR a newly generated file. (We can just check if it's being updated).
     - Keep `MISC` lines, UNLESS the filename matches a newly generated file (e.g. `metadata.xml`).
   - For new files (ebuilds, metadata.xml, files in `files/`):
     - Calculate their hashes.
     - Emit `EBUILD <filename>`, `MISC <filename>`, `AUX <filename>` lines.
     - Sort all lines (DIST, EBUILD, AUX, MISC) and put them in the Manifest.
