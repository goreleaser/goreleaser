package build

// list from https://golang.org/doc/install/source#environment
var valids = []string{
	"androidarm",
	"darwin386",
	"darwinamd64",
	"darwinarm",
	"darwinarm64",
	"dragonflyamd64",
	"freebsd386",
	"freebsdamd64",
	"freebsdarm",
	"linux386",
	"linuxamd64",
	"linuxarm",
	"linuxarm64",
	"linuxppc64",
	"linuxppc64le",
	"linuxmips",
	"linuxmipsle",
	"linuxmips64",
	"linuxmips64le",
	"netbsd386",
	"netbsdamd64",
	"netbsdarm",
	"openbsd386",
	"openbsdamd64",
	"openbsdarm",
	"plan9386",
	"plan9amd64",
	"solarisamd64",
	"windows386",
	"windowsamd64",
}

func valid(goos, goarch string) bool {
	var s = goos + goarch
	for _, a := range valids {
		if a == s {
			return true
		}
	}
	return false
}
