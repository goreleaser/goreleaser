package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/invopop/jsonschema"
	validator "github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/require"
)

func TestSlackJSONSchema(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(jsonschema.Reflect(&Project{}))
	require.NoError(t, err)
	schema, err := validator.CompileString("schema.json", string(data))
	require.NoError(t, err)

	for _, tc := range []struct {
		name   string
		config io.Reader
	}{
		{
			name: "blocks and attachments",
			config: strings.NewReader(`
announce:
  slack:
    blocks:
      - type: divider
    attachments:
      - text: hello
`),
		},
		{name: "advanced blocks", config: goodBlocksSlackConf()},
		{name: "advanced attachments", config: goodAttachmentsSlackConf()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			project, err := LoadReader(tc.config)
			require.NoError(t, err)
			data, err := json.Marshal(map[string]any{
				"announce": map[string]any{"slack": project.Announce.Slack},
			})
			require.NoError(t, err)
			var document any
			require.NoError(t, json.Unmarshal(data, &document))
			require.NoError(t, schema.Validate(document))
		})
	}

	for _, field := range []string{"blocks", "attachments"} {
		t.Run("invalid "+field, func(t *testing.T) {
			t.Parallel()

			require.Error(t, schema.Validate(map[string]any{
				"announce": map[string]any{
					"slack": map[string]any{field: []any{"not an object"}},
				},
			}))
		})
	}
}

func TestSignJSONSchema(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(jsonschema.Reflect(&Project{}))
	require.NoError(t, err)
	schema, err := validator.CompileString("schema.json", string(data))
	require.NoError(t, err)

	for _, field := range []string{"signs", "docker_signs"} {
		for _, selector := range []string{"none", "checksum", "invalid"} {
			t.Run(field+"/"+selector, func(t *testing.T) {
				t.Parallel()

				err := schema.Validate(map[string]any{
					field: []any{map[string]any{"artifacts": selector}},
				})
				if selector == "invalid" {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestMCPTransportStrictConfig(t *testing.T) {
	t.Parallel()

	project, err := LoadReader(strings.NewReader(`
mcp:
  name: io.github.test/server
  title: Test Server
  auth:
    type: none
  packages:
    - registry_type: npm
      identifier: '@test/server'
      transport:
        type: streamable-http
        url: https://example.com/mcp
`))
	require.NoError(t, err)
	require.Equal(t, "https://example.com/mcp", project.MCP.Packages[0].Transport.URL)
}

func TestAPKBuildsProJSONSchema(t *testing.T) {
	t.Parallel()

	for _, file := range []string{"schema.json", "schema-pro.json"} {
		t.Run(file, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(filepath.Join("..", "..", "www", "static", file))
			require.NoError(t, err)
			var document struct {
				Defs map[string]map[string]any `json:"$defs"`
			}
			require.NoError(t, json.Unmarshal(data, &document))
			// the full generated schemas do not compile with this validator
			// (repeated nested $id values), so only the parts apkbuilds needs
			// are compiled, keeping the project's additionalProperties rule.
			project := document.Defs["Project"]
			properties := project["properties"].(map[string]any)
			require.Contains(t, properties, "apkbuilds")
			data, err = json.Marshal(map[string]any{
				"type":                 "object",
				"additionalProperties": project["additionalProperties"],
				"properties":           map[string]any{"apkbuilds": properties["apkbuilds"]},
				"$defs":                referencedDefs(document.Defs, "APKBuild"),
			})
			require.NoError(t, err)
			schema, err := validator.CompileString(file, string(data))
			require.NoError(t, err)

			apkbuild := map[string]any{
				"name":                "foo",
				"ids":                 []any{"foo"},
				"commit_author":       map[string]any{"name": "bot", "email": "bot@example.com"},
				"commit_msg_template": "update {{ .ProjectName }}",
				"description":         "Foo",
				"homepage":            "https://example.com",
				"license":             "MIT",
				"skip_upload":         "auto",
				"url_template":        "https://example.com/{{ .ArtifactName }}",
				"maintainers":         []any{"Foo <foo@example.com>"},
				"contributors":        []any{"Bar <bar@example.com>"},
				"provides":            []any{"foo-bin"},
				"depends":             []any{"musl"},
				"makedepends":         []any{"go"},
				"replaces":            []any{"foo-old"},
				"options":             []any{"!strip"},
				"rel":                 "1",
				"package":             `install -Dm755 "$srcdir/foo" "$pkgdir/usr/bin/foo"`,
				"git_url":             "git@example.com:foo/aports.git",
				"git_ssh_command":     "ssh -i {{ .KeyPath }}",
				"private_key":         "{{ .Env.KEY }}",
				"goamd64":             "v1",
				"directory":           "testing/foo",
				"disable":             false,
			}
			require.NoError(t, schema.Validate(map[string]any{"apkbuilds": []any{apkbuild}}))

			apkbuild["not_a_field"] = true
			require.Error(t, schema.Validate(map[string]any{"apkbuilds": []any{apkbuild}}))
		})
	}
}

func referencedDefs(defs map[string]map[string]any, name string) map[string]any {
	result := map[string]any{}
	pending := []string{name}
	for len(pending) > 0 {
		name, pending = pending[0], pending[1:]
		if _, ok := result[name]; ok {
			continue
		}
		result[name] = defs[name]
		data, _ := json.Marshal(defs[name])
		for _, ref := range regexp.MustCompile(`#/\$defs/(\w+)`).FindAllStringSubmatch(string(data), -1) {
			pending = append(pending, ref[1])
		}
	}
	return result
}
