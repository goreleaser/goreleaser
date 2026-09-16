package tmpl

import (
	"fmt"
	"strings"
)

// Toolchain defaults. A variant that matches its default is left out of the
// suffix, so that names stay stable for the common case of not setting it.
const (
	defaultAmd64   = "v1"
	defaultArm64   = "v8.0"
	default386     = "sse2"
	defaultPpc64   = "power8"
	defaultRiscv64 = "rva20u64"
)

// variant renders the suffix that tells apart artifacts which share an
// os/arch but differ by target variant, e.g. "v9.0" for
// `goarm64: [v8.0, v9.0]`, or "_musl" for a musl target.
//
// Without it such artifacts resolve to the same name, and one of them is
// silently lost: overwritten on disk, or dropped when uploading two assets
// with the same name.
func variant(v any) (string, error) {
	fields, ok := v.(Fields)
	if !ok {
		return "", fmt.Errorf("variant: expected the template context, got %T: use it as '{{ variant . }}'", v)
	}
	get := func(key string) string {
		s, _ := fields[key].(string)
		return s
	}

	var sb strings.Builder
	// Arm and Mips have no default, as they always rendered, including at
	// the toolchain default, long before this function existed.
	for _, v := range []struct{ prefix, value, def string }{
		{"v", get(KeyArm), ""},
		{"_", get(KeyMips), ""},
		{"", get(KeyAmd64), defaultAmd64},
		{"", get(KeyArm64), defaultArm64},
		{"", get(Key386), default386},
		{"", get(KeyPpc64), defaultPpc64},
		{"", get(KeyRiscv64), defaultRiscv64},
		{"_", get(KeyAbi), ""},
	} {
		if v.value != "" && v.value != v.def {
			sb.WriteString(v.prefix + v.value)
		}
	}

	// Goarm64 features are comma-separated, e.g. "v9.0,lse". Keep them, as
	// they make up a distinct target, but do not put a comma in a file or
	// release asset name.
	return strings.ReplaceAll(sb.String(), ",", "-"), nil
}
