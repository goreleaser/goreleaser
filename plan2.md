Wait, if an old ebuild is deleted, we also need to know if any of the `AUX` files (like patches) were ONLY used by the deleted ebuilds, to remove them? The issue says:
"also make sure that the ebuild is trimmed from deleted ebuild src that isn't referred to any of the remaining, when we are on prune mode."
Oh! "ebuild is trimmed from deleted ebuild src that isn't referred to any of the remaining" -> This likely means the *Manifest* entry for deleted ebuilds (i.e. the EBUILD line). Wait, the issue says:
"also make sure that the ebuild is trimmed from deleted ebuild src that isn't referred to any of the remaining, when we are on prune mode."
It probably means: remove `DIST` entries that are only associated with deleted ebuilds, but the current code already tries to do that! Let's check what the current code does for `DIST`. It removes a `DIST` line if its filename contains a substring matching the deleted version.
And for `EBUILD` entries, we should definitely remove the `EBUILD` line for a deleted ebuild.
