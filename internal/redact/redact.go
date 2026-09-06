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
func Writer(w io.Writer, env []string) io.WriteCloser {
	return &redactWriter{
		re: redact(env),
		w:  w,
	}
}

type redactWriter struct {
	re      *redacter
	w       io.Writer
	pending string
}

// Write implements [io.Writer].
func (w *redactWriter) Write(p []byte) (int, error) {
	s := w.pending + string(p)
	redacted, pending := w.re.replacePartial(s)
	_, err := io.WriteString(w.w, redacted)
	if err != nil {
		return 0, err
	}
	w.pending = pending
	return len(p), nil
}

// Close flushes any pending data that may have been a secret prefix.
func (w *redactWriter) Close() error {
	if w.pending == "" {
		return nil
	}
	_, err := io.WriteString(w.w, w.re.Replace(w.pending))
	w.pending = ""
	return err
}

// redact returns a redacter that replaces all occurrences of
// secret-looking environment variable values in s with their "$NAME"
// counterparts.
//
// Each entry in env should be in "KEY=VALUE" format.
func redact(env []string) *redacter {
	var secrets []secret
	for _, e := range env {
		k, v, ok := strings.Cut(e, "=")
		if !ok || v == "" {
			continue
		}
		if looksSecret(k, v) {
			secrets = append(secrets, secret{k, v})
		}
	}
	slices.SortFunc(secrets, func(a, b secret) int {
		if c := cmp.Compare(len(b.v), len(a.v)); c != 0 {
			return c
		}
		return cmp.Compare(a.k, b.k)
	})
	oldnew := make([]string, 0, len(secrets)*2)
	for _, e := range secrets {
		oldnew = append(oldnew, e.v, "$"+e.k)
	}
	return &redacter{
		secrets:  secrets,
		maxLen:   maxSecretLen(secrets),
		replacer: strings.NewReplacer(oldnew...),
	}
}

type redacter struct {
	secrets  []secret
	maxLen   int
	replacer *strings.Replacer
}

type secret struct{ k, v string }

func (r *redacter) Replace(s string) string {
	return r.replacer.Replace(s)
}

func (r *redacter) replacePartial(s string) (redacted, pending string) {
	if len(r.secrets) == 0 {
		return s, ""
	}

	var b strings.Builder
	for i := 0; i < len(s); {
		rest := s[i:]
		if r.isIncompleteSecret(rest) {
			return b.String(), rest
		}
		if secret, ok := r.match(rest); ok {
			b.WriteString("$" + secret.k)
			i += len(secret.v)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String(), ""
}

func (r *redacter) isIncompleteSecret(s string) bool {
	if len(s) >= r.maxLen {
		return false
	}
	for _, secret := range r.secrets {
		if len(secret.v) > len(s) && strings.HasPrefix(secret.v, s) {
			return true
		}
	}
	return false
}

func (r *redacter) match(s string) (secret, bool) {
	for _, secret := range r.secrets {
		if strings.HasPrefix(s, secret.v) {
			return secret, true
		}
	}
	return secret{}, false
}

func maxSecretLen(secrets []secret) int {
	var maxLen int
	for _, secret := range secrets {
		maxLen = max(maxLen, len(secret.v))
	}
	return maxLen
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

// String redacts secret-looking environment variable values in s
// with their "$NAME" counterparts.
//
// Each entry in env should be in "KEY=VALUE" format.
func String(s string, env []string) string {
	return redact(env).Replace(s)
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
