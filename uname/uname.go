package uname

var mapping = map[string]string{
	"darwin":  "Darwin",
	"linux":   "Linux",
	"freebsd": "FreeBSD",
	"openbsd": "OpenBSD",
	"netbsd":  "NetBSD",
	"386":     "i386",
	"amd64":   "x86_64",
}

func FromGo(s string) string {
	result := mapping[s]
	if result == "" {
		result = s
	}
	return result
}
