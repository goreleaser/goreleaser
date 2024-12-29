package golang

import (
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

func formatTarget(o config.BuildDetailsOverride) string {
	target := o.Goos + "_" + o.Goarch
	if extra := o.Goamd64 + o.Go386 + o.Goarm + o.Goarm64 + o.Gomips + o.Goppc64 + o.Goriscv64; extra != "" {
		target += "_" + extra
	}
	return target
}

// Target is a Go build target.
type Target struct {
	Target    string
	Goos      string
	Goarch    string
	Goamd64   string
	Go386     string
	Goarm     string
	Goarm64   string
	Gomips    string
	Goppc64   string
	Goriscv64 string
}

// Fields implements build.Target.
func (t Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:      t.Goos,
		tmpl.KeyArch:    t.Goarch,
		tmpl.KeyAmd64:   t.Goamd64,
		tmpl.Key386:     t.Go386,
		tmpl.KeyArm:     t.Goarm,
		tmpl.KeyArm64:   t.Goarm64,
		tmpl.KeyMips:    t.Gomips,
		tmpl.KeyPpc64:   t.Goppc64,
		tmpl.KeyRiscv64: t.Goriscv64,
	}
}

// String implements fmt.Stringer.
func (t Target) String() string {
	return t.Target
}

func (t Target) env() []string {
	return []string{
		"GOOS=" + t.Goos,
		"GOARCH=" + t.Goarch,
		"GOAMD64=" + t.Goamd64,
		"GO386=" + t.Go386,
		"GOARM=" + t.Goarm,
		"GOARM64=" + t.Goarm64,
		"GOMIPS=" + t.Gomips,
		"GOMIPS64=" + t.Gomips,
		"GOPPC64=" + t.Goppc64,
		"GORISCV64=" + t.Goriscv64,
	}
}
