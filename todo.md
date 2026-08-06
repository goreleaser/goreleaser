# Refactoring Gentoo Pipeline (v2)

- [x] Task 1: Extract `ebuildData` struct, receiver methods (`Validate`, `RenderEbuild`, `RenderMetaCache`, `SortedUseFlags`, `FormattedSrcURIs`), and `TestEbuildData` unit tests. Commit, push, and create Draft PR against `feature/gentoo-ebuild`.
- [x] Task 2: Extract `extraFilesProcessor.InstallExtraFiles` method and `TestInstallExtraFiles` unit test. Commit & push.
- [x] Task 3: Decompose `Publish` method into `publishGroup` struct and receiver methods (`collectPublishGroups`, `applyVersionRetention`, `publish`). Commit & push.
- [x] Task 4: Replace `metadata.xml.tmpl` text template with native Go `gentooMetadata` XML struct marshaling, methods (`AddMaintainers`, `AddUseFlags`, `SetUpstream`, `Marshal`), and `TestGentooMetadata` unit test. Commit & push.
- [ ] Task 5: Split `gentoo.go` into domain files (`ebuild.go`, `metadata.go`, `retention.go`, `gentoo.go`). Commit & push.
- [ ] Task 6: Finalize PR (ready for review).
