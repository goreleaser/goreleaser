package slack

import (
	"bytes"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/yaml"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, Pipe{}.String(), "slack")
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.Announce.Slack.MessageTemplate, defaultMessageTemplate)
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Slack: config.Slack{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	require.EqualError(t, Pipe{}.Announce(ctx), `slack: template: tmpl:1: unexpected "}" in operand`)
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Slack: config.Slack{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `slack: env: environment variable "SLACK_WEBHOOK" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Announce: config.Announce{
				Slack: config.Slack{
					Enabled: true,
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

const testVersion = "v1.2.3"

func TestParseRichText(t *testing.T) {
	t.Parallel()

	t.Run("parse only - full slack config with blocks and attachments", func(t *testing.T) {
		t.Parallel()

		var project config.Project
		require.NoError(t, yaml.Unmarshal(goodRichSlackConf(), &project))

		ctx := context.New(project)
		ctx.Version = testVersion

		blocks, attachments, err := parseAdvancedFormatting(ctx)
		require.NoError(t, err)

		require.Len(t, blocks.BlockSet, 4)
		require.Len(t, attachments, 2)
	})

	t.Run("parse only - slack config with bad blocks", func(t *testing.T) {
		t.Parallel()

		var project config.Project
		require.NoError(t, yaml.Unmarshal(badBlocksSlackConf(), &project))

		ctx := context.New(project)
		ctx.Version = testVersion

		_, _, err := parseAdvancedFormatting(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "json")
	})

	t.Run("parse only - slack config with bad attachments", func(t *testing.T) {
		t.Parallel()

		var project config.Project
		require.NoError(t, yaml.Unmarshal(badAttachmentsSlackConf(), &project))

		ctx := context.New(project)
		ctx.Version = testVersion

		_, _, err := parseAdvancedFormatting(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "json")
	})
}

func TestRichText(t *testing.T) {
	t.Parallel()
	os.Setenv("SLACK_WEBHOOK", slackTestHook())

	t.Run("e2e - full slack config with blocks and attachments", func(t *testing.T) {
		t.SkipNow() // requires a valid webhook for integration testing
		t.Parallel()

		var project config.Project
		require.NoError(t, yaml.Unmarshal(goodRichSlackConf(), &project))

		ctx := context.New(project)
		ctx.Version = testVersion

		require.NoError(t, Pipe{}.Announce(ctx))
	})

	t.Run("slack config with bad blocks", func(t *testing.T) {
		t.Parallel()

		var project config.Project
		require.NoError(t, yaml.Unmarshal(badBlocksSlackConf(), &project))

		ctx := context.New(project)
		ctx.Version = testVersion

		err := Pipe{}.Announce(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "json")
	})
}

func TestUnmarshall(t *testing.T) {
	t.Parallel()

	t.Run("happy unmarshal", func(t *testing.T) {
		t.Parallel()
		ctx := testctx.New(testctx.WithVersion(testVersion))
		var blocks slack.Blocks
		require.NoError(t, unmarshal(ctx, []interface{}{map[string]interface{}{"type": "divider"}}, &blocks))
	})

	t.Run("unmarshal fails on MarshalJSON", func(t *testing.T) {
		t.Parallel()
		ctx := testctx.New(testctx.WithVersion(testVersion))
		var blocks slack.Blocks
		require.Error(t, unmarshal(ctx, []interface{}{map[string]interface{}{"type": func() {}}}, &blocks))
	})

	t.Run("unmarshal happy to resolve template", func(t *testing.T) {
		t.Parallel()

		var project config.Project
		require.NoError(t, yaml.Unmarshal(goodTemplateSlackConf(), &project))
		ctx := context.New(project)
		ctx.Version = testVersion

		var blocks slack.Blocks
		require.NoError(t, unmarshal(ctx, ctx.Config.Announce.Slack.Blocks, &blocks))

		require.Len(t, blocks.BlockSet, 1)
		header, ok := blocks.BlockSet[0].(*slack.HeaderBlock)
		require.True(t, ok)
		require.Contains(t, header.Text.Text, testVersion)
	})

	t.Run("unmarshal fails on resolve template", func(t *testing.T) {
		t.Parallel()

		var project config.Project
		require.NoError(t, yaml.Unmarshal(badTemplateSlackConf(), &project))
		ctx := context.New(project)
		ctx.Version = testVersion

		var blocks slack.Blocks
		require.Error(t, unmarshal(ctx, ctx.Config.Announce.Slack.Blocks, &blocks))
	})
}

func slackTestHook() string {
	// redacted: replace this by a real Slack Web Incoming Hook to test the featue end to end.
	const hook = "https://hooks.slack.com/services/*********/***********/************************"

	return hook
}

func goodRichSlackConf() []byte {
	const conf = `
project_name: test
announce:
  slack:
    enabled: true
    message_template: fallback
    channel: my_channel
    blocks:
      - type: header
        text:
          type: plain_text
          text: '{{ .Version }}'
      - type: section
        text:
          type: mrkdwn
          text: |
            Heading
            =======

			# Other Heading

            *Bold*
			_italic_
            ~Strikethrough~

            ## Heading 2
            ### Heading 3
			* List item 1
			* List item 2

			- List item 3
			- List item 4

			[link](https://example.com)
			<https://example.com|link>

			:)

			:star:

	  - type: divider
	  - type: section
        text:
          type: mrkdwn
          text: |
            my release
    attachments:
        -
          title: Release artifacts
          color: '#2eb886'
		  text: |
            *Helm chart packages*
        - fallback: full changelog
          color: '#2eb886'
          title: Full Change Log
          text: |
            * this link
            * that link
`

	buf := bytes.NewBufferString(conf)

	return bytes.ReplaceAll(buf.Bytes(), []byte("\t"), []byte("    "))
}

func badBlocksSlackConf() []byte {
	const conf = `
project_name: test
announce:
  slack:
    enabled: true
    message_template: fallback
    channel: my_channel
    blocks:
      - type: header
		text: invalid  # <- wrong type for Slack API
`

	buf := bytes.NewBufferString(conf)

	return bytes.ReplaceAll(buf.Bytes(), []byte("\t"), []byte("    "))
}

func badAttachmentsSlackConf() []byte {
	const conf = `
project_name: test
announce:
  slack:
    enabled: true
    message_template: fallback
    channel: my_channel
    attachments:
        -
          title:
		   - Release artifacts
		   - wrong # <- title is not an array
          color: '#2eb886'
		  text: |
            *Helm chart packages*
`

	buf := bytes.NewBufferString(conf)

	return bytes.ReplaceAll(buf.Bytes(), []byte("\t"), []byte("    "))
}

func goodTemplateSlackConf() []byte {
	const conf = `
project_name: test
announce:
  slack:
    enabled: true
	message_template: '{{ .Version }}'
    channel: my_channel
    blocks:
      - type: header
        text:
          type: plain_text
          text: '{{ .Version }}'
`

	buf := bytes.NewBufferString(conf)

	return bytes.ReplaceAll(buf.Bytes(), []byte("\t"), []byte("    "))
}

func badTemplateSlackConf() []byte {
	const conf = `
project_name: test
announce:
  slack:
    enabled: true
	message_template: fallback
    channel: my_channel
    blocks:
      - type: header
        text:
          type: plain_text
		  text: '{{ .Wrong }}'
`

	buf := bytes.NewBufferString(conf)

	return bytes.ReplaceAll(buf.Bytes(), []byte("\t"), []byte("    "))
}
