package redact

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  []string
		in   string
		want string
	}{
		{
			name: "key suffix TOKEN",
			env:  []string{"GITHUB_TOKEN=abc123secret"},
			in:   "using abc123secret to auth",
			want: "using $GITHUB_TOKEN to auth",
		},
		{
			name: "key suffix KEY",
			env:  []string{"API_KEY=myapikeyval"},
			in:   "key=myapikeyval",
			want: "key=$API_KEY",
		},
		{
			name: "key suffix SECRET",
			env:  []string{"AWS_SECRET=s3cr3tvalue"},
			in:   "secret: s3cr3tvalue",
			want: "secret: $AWS_SECRET",
		},
		{
			name: "key suffix PASSWORD",
			env:  []string{"DB_PASSWORD=hunter2hunter2"},
			in:   "pass=hunter2hunter2",
			want: "pass=$DB_PASSWORD",
		},
		{
			name: "value prefix sk-",
			env:  []string{"OPENAI=sk-abcdef123456"},
			in:   "token sk-abcdef123456 used",
			want: "token $OPENAI used",
		},
		{
			name: "value prefix ghp_",
			env:  []string{"GH=ghp_xxxxxxxxxxxx"},
			in:   "ghp_xxxxxxxxxxxx",
			want: "$GH",
		},
		{
			name: "value prefix ghs_",
			env:  []string{"GH_APP=ghs_xxxxxxxxxxxx"},
			in:   "ghs_xxxxxxxxxxxx",
			want: "$GH_APP",
		},
		{
			name: "value prefix dckr_pat_",
			env:  []string{"DOCKER=dckr_pat_abcdefgh"},
			in:   "dckr_pat_abcdefgh",
			want: "$DOCKER",
		},
		{
			name: "value prefix glpat-",
			env:  []string{"GITLAB=glpat-xxxxxxxxxxxx"},
			in:   "glpat-xxxxxxxxxxxx",
			want: "$GITLAB",
		},
		{
			name: "no match leaves string unchanged",
			env:  []string{"HOME=/home/user", "GOPATH=/go"},
			in:   "home is /home/user",
			want: "home is /home/user",
		},
		{
			name: "empty value is not redacted",
			env:  []string{"MY_TOKEN="},
			in:   "nothing here",
			want: "nothing here",
		},
		{
			name: "short value is redacted when key looks secret",
			env:  []string{"MY_TOKEN=short"},
			in:   "short",
			want: "$MY_TOKEN",
		},
		{
			name: "minimum length value is redacted",
			env:  []string{"MY_TOKEN=longenough"},
			in:   "longenough",
			want: "$MY_TOKEN",
		},
		{
			name: "KEY mid-word does not match",
			env:  []string{"KEYBOARD_LAYOUT=us"},
			in:   "layout is us",
			want: "layout is us",
		},
		{
			name: "multiple secrets",
			env:  []string{"API_KEY=key123key123", "DB_SECRET=pass456pass456"},
			in:   "key123key123 and pass456pass456",
			want: "$API_KEY and $DB_SECRET",
		},
		{
			name: "longer secret replaced first",
			env:  []string{"SHORT_TOKEN=token12345", "LONG_TOKEN=token12345-extended"},
			in:   "value: token12345-extended",
			want: "value: $LONG_TOKEN",
		},
		{
			name: "multiple occurrences of same secret",
			env:  []string{"API_KEY=secretvalue"},
			in:   "secretvalue and secretvalue again",
			want: "$API_KEY and $API_KEY again",
		},
		{
			name: "empty input string",
			env:  []string{"API_KEY=secretvalue"},
			in:   "",
			want: "",
		},
		{
			name: "nil env",
			env:  nil,
			in:   "nothing",
			want: "nothing",
		},
		{
			name: "entry without equals sign is skipped",
			env:  []string{"NOTAVALIDENTRY"},
			in:   "NOTAVALIDENTRY",
			want: "NOTAVALIDENTRY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := redact(tt.env).Replace(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRedactWriter(t *testing.T) {
	t.Parallel()

	env := []string{
		"API_KEY=key123key123",
		"DB_SECRET=pass456pass456",
	}

	t.Run("redacts secrets", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := Writer(&buf, env)
		_, err := io.WriteString(w, "using key123key123 and pass456pass456\n")
		require.NoError(t, err)
		require.Equal(t, "using $API_KEY and $DB_SECRET\n", buf.String())
	})

	t.Run("returns original byte count", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := Writer(&buf, env)
		input := "key123key123\n"
		n, err := io.WriteString(w, input)
		require.NoError(t, err)
		require.Equal(t, len(input), n)
	})

	t.Run("no secrets passes through", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := Writer(&buf, env)
		_, err := io.WriteString(w, "nothing secret here\n")
		require.NoError(t, err)
		require.Equal(t, "nothing secret here\n", buf.String())
	})

	t.Run("redacts secrets split across every write boundary", func(t *testing.T) {
		t.Parallel()
		const secret = "key123key123"
		for i := 1; i < len(secret); i++ {
			t.Run(fmt.Sprintf("split at %d", i), func(t *testing.T) {
				t.Parallel()
				var buf bytes.Buffer
				w := Writer(&buf, env)
				n, err := io.WriteString(w, secret[:i])
				require.NoError(t, err)
				require.Equal(t, i, n)
				n, err = io.WriteString(w, secret[i:])
				require.NoError(t, err)
				require.Equal(t, len(secret)-i, n)
				require.Equal(t, "$API_KEY", buf.String())
			})
		}
	})

	t.Run("prefers longer overlapping secrets across writes", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := Writer(&buf, []string{
			"SHORT_TOKEN=redacted-test",
			"API_KEY=redacted-test-value",
		})
		_, err := io.WriteString(w, "redacted-test")
		require.NoError(t, err)
		_, err = io.WriteString(w, "-value")
		require.NoError(t, err)
		require.Equal(t, "$API_KEY", buf.String())
	})

	t.Run("flushes incomplete secret prefix on close", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := Writer(&buf, env)
		_, err := io.WriteString(w, "key123")
		require.NoError(t, err)
		require.Empty(t, buf.String())
		require.NoError(t, w.Close())
		require.Equal(t, "key123", buf.String())
	})

	t.Run("returns zero bytes on write error", func(t *testing.T) {
		t.Parallel()
		w := Writer(&errWriter{err: io.ErrShortWrite}, env)
		n, err := io.WriteString(w, "key123key123\n")
		require.ErrorIs(t, err, io.ErrShortWrite)
		require.Equal(t, 0, n)
	})
}

func TestRedactString(t *testing.T) {
	t.Parallel()

	t.Run("redacts secrets", func(t *testing.T) {
		t.Parallel()
		env := []string{"API_KEY=key123key123"}
		got := String("using key123key123 to auth", env)
		require.Equal(t, "using $API_KEY to auth", got)
	})

	t.Run("no secrets passes through", func(t *testing.T) {
		t.Parallel()
		env := []string{"API_KEY=key123key123"}
		got := String("nothing secret here", env)
		require.Equal(t, "nothing secret here", got)
	})
}

func TestWriterMatchesStringAcrossChunks(t *testing.T) {
	t.Parallel()
	env := []string{
		"SHORT_TOKEN=aba",
		"LONG_TOKEN=abacus",
		"OTHER_TOKEN=bac",
		"UNICODE_TOKEN=\xc3\xa9_secret",
	}
	for _, input := range []string{
		"ordinary output\n",
		"abacus aba bac\n",
		"prefix abacus and a trailing ab",
		"aba",
		"abacus",
		"\xc3\xa9_secret and \xc3\xa9",
	} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			for size := 1; size <= len(input); size++ {
				var out bytes.Buffer
				w := Writer(&out, env)
				for i := 0; i < len(input); i += size {
					_, err := io.WriteString(w, input[i:min(i+size, len(input))])
					require.NoError(t, err)
				}
				require.NoError(t, w.Close())
				require.Equal(t, String(input, env), out.String(), "chunk size %d", size)
			}
		})
	}
}

type errWriter struct{ err error }

func (e *errWriter) Write([]byte) (int, error) { return 0, e.err }

func BenchmarkWriter(b *testing.B) {
	data := []byte(strings.Repeat("building package example.com/project/internal/component\n", 600))
	for _, count := range []int{0, 1, 16, 64} {
		b.Run(fmt.Sprintf("secrets=%d", count), func(b *testing.B) {
			env := make([]string, count)
			for i := range env {
				env[i] = fmt.Sprintf("SERVICE_%d_TOKEN=secret-value-%032d", i, i)
			}
			w := Writer(io.Discard, env)
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				if _, err := w.Write(data); err != nil {
					b.Fatal(err)
				}
			}
			require.NoError(b, w.Close())
		})
	}
}
