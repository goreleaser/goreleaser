// Package redact provides functions to redact secrets from strings.
package redact

import (
	"cmp"
	"io"
	"slices"
	"strings"
)

// Writer wraps the writer and redacts secret-looking environment
// variable values in the written data with their "$NAME" counterparts.
//
// Each entry in env should be in "KEY=VALUE" format.
func Writer(w io.Writer, env []string) io.Writer {
	return &redactWriter{
		re: redact(env),
		w:  w,
	}
}

type redactWriter struct {
	re *strings.Replacer
	w  io.Writer
}

// Write implements [io.Writer].
func (w *redactWriter) Write(p []byte) (int, error) {
	_, err := io.WriteString(w.w, w.re.Replace(string(p)))
	return len(p), err
}

// redact returns a strings.Replacer that replaces all occurrences of
// secret-looking environment variable values in s with their "$NAME"
// counterparts.
//
// Each entry in env should be in "KEY=VALUE" format.
func redact(env []string) *strings.Replacer {
	type kv struct{ k, v string }
	var secrets []kv
	for _, e := range env {
		k, v, ok := strings.Cut(e, "=")
		if !ok || len(v) < 10 {
			continue
		}
		if looksSecret(k, v) {
			secrets = append(secrets, kv{k, v})
		}
	}
	slices.SortFunc(secrets, func(a, b kv) int {
		if c := cmp.Compare(len(b.v), len(a.v)); c != 0 {
			return c
		}
		return cmp.Compare(a.k, b.k)
	})
	oldnew := make([]string, 0, len(secrets)*2)
	for _, e := range secrets {
		oldnew = append(oldnew, e.v, "$"+e.k)
	}
	return strings.NewReplacer(oldnew...)
}

var keySuffixes = []string{
	"_KEY",
	"_SECRET",
	"_PASSWORD",
	"_TOKEN",
}

var valuePrefixes = []string{
	"sk-",
	"ghp_",
	"ghs_",
	"gho_",
	"ghu_",
	"dckr_pat_",
	"glpat-",
	"AIZA",
	"xox",
}

func looksSecret(k, v string) bool {
	for _, s := range keySuffixes {
		if strings.HasSuffix(k, s) {
			return true
		}
	}
	for _, p := range valuePrefixes {
		if strings.HasPrefix(v, p) {
			return true
		}
	}
	return false
}
