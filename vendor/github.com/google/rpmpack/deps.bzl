load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

def rpmpack_dependencies():
    go_rules_dependencies()
    go_register_toolchains()
    gazelle_dependencies()

    go_repository(
        name = "com_github_pkg_errors",
        importpath = "github.com/pkg/errors",
        tag = "v0.8.1",
    )

    go_repository(
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        tag = "v0.2.0",
    )

    go_repository(
        name = "com_github_cavaliercoder_go_cpio",
        commit = "925f9528c45e",
        importpath = "github.com/cavaliercoder/go-cpio",
    )

    go_repository(
        name = "com_github_ulikunitz_xz",
        importpath = "github.com/ulikunitz/xz",
        tag = "v0.5.6",
    )
