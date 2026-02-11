package redact

import (
	"bytes"
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
			name: "short value is not redacted",
			env:  []string{"MY_TOKEN=short"},
			in:   "short",
			want: "short",
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
		_, err := w.Write([]byte("using key123key123 and pass456pass456\n"))
		require.NoError(t, err)
		require.Equal(t, "using $API_KEY and $DB_SECRET\n", buf.String())
	})

	t.Run("returns original byte count", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := Writer(&buf, env)
		input := []byte("key123key123\n")
		n, err := w.Write(input)
		require.NoError(t, err)
		require.Equal(t, len(input), n)
	})

	t.Run("no secrets passes through", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := Writer(&buf, env)
		_, err := w.Write([]byte("nothing secret here\n"))
		require.NoError(t, err)
		require.Equal(t, "nothing secret here\n", buf.String())
	})
}
