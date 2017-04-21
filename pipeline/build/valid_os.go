package build

// list from https://golang.org/doc/install/source#environment
var valids = []string{
	"androidarm",
	"darwin386",
	"darwinamd64",
	// "darwinarm", - requires admin rights and other ios stuff
	// "darwinarm64", - requires admin rights and other ios stuff
	"dragonflyamd64",
	"freebsd386",
	"freebsdamd64",
	"freebsdarm",
	"linux386",
	"linuxamd64",
	"linuxarm",
	"linuxarm64",
	// "linuxppc64", - https://github.com/golang/go/issues/10087
	// "linuxppc64le", - https://github.com/golang/go/issues/10087
	"linuxmips",
	"linuxmipsle",
	"linuxmips64",
	"linuxmips64le",
	"netbsd386",
	"netbsdamd64",
	"netbsdarm",
	"openbsd386",
	"openbsdamd64",
	// "openbsdarm", - https://github.com/golang/go/issues/10087
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
