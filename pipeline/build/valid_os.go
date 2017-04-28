package build

// list originally from https://golang.org/doc/install/source#environment
// later tweaked.
// line suffix comments explain why
var validBuildTargets = []buildTarget{
	buildTarget{"android", "arm", ""},
	buildTarget{"darwin", "386", ""},
	buildTarget{"darwin", "amd64", ""},
	// buildTarget{"darwin", "arm", ""}, requires admin rights and other ios stuff
	// buildTarget{"darwin", "arm64", ""}, requires admin rights and other ios stuff
	buildTarget{"dragonfly", "amd64", ""},
	buildTarget{"freebsd", "386", ""},
	buildTarget{"freebsd", "amd64", ""},
	// buildTarget{"freebsd", "arm", ""}, the default would be armv6, but that would cause double build for this target
	buildTarget{"freebsd", "arm", "5"},
	buildTarget{"freebsd", "arm", "6"},
	buildTarget{"freebsd", "arm", "7"},
	buildTarget{"linux", "386", ""},
	buildTarget{"linux", "amd64", ""},
	// buildTarget{"linux", "arm", ""}, the default would be armv6, but that would cause double build for this target
	buildTarget{"linux", "arm", "5"},
	buildTarget{"linux", "arm", "6"},
	buildTarget{"linux", "arm", "7"},
	buildTarget{"linux", "arm64", ""},
	// buildTarget{ "linux", "ppc64", ""}, https://github.com/golang/go/issues/10087
	// buildTarget{ "linux", "ppc64le", ""}, https://github.com/golang/go/issues/10087
	buildTarget{"linux", "mips", ""},
	buildTarget{"linux", "mipsle", ""},
	buildTarget{"linux", "mips64", ""},
	buildTarget{"linux", "mips64le", ""},
	buildTarget{"netbsd", "386", ""},
	buildTarget{"netbsd", "amd64", ""},
	// buildTarget{"netbsd", "arm", ""}, , the default would be armv6, but that would cause double build for this target
	buildTarget{"netbsd", "arm", "5"},
	buildTarget{"netbsd", "arm", "6"},
	buildTarget{"netbsd", "arm", "7"},
	buildTarget{"openbsd", "386", ""},
	buildTarget{"openbsd", "amd64", ""},
	// buildTarget{"openbsd", "arm", ""}", https://github.com/golang/go/issues/10087
	buildTarget{"plan9", "386", ""},
	buildTarget{"plan9", "amd64", ""},
	buildTarget{"solaris", "amd64", ""},
	buildTarget{"windows", "386", ""},
	buildTarget{"windows", "amd64", ""},
}
