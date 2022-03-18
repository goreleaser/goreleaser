package config

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalSlackBlocks(t *testing.T) {
	t.Parallel()

	t.Run("valid blocks", func(t *testing.T) {
		t.Parallel()

		prop, err := LoadReader(goodBlocksSlackConf())
		require.NoError(t, err)

		expectedBlocks := []SlackBlock{
			{
				Internal: map[string]interface{}{
					"type": "header",
					"text": map[string]interface{}{
						"type": "plain_text",
						"text": "{{ .Version }}",
					},
				},
			},
			{
				Internal: map[string]interface{}{
					"text": map[string]interface{}{
						"type": "mrkdwn",
						"text": "Heading\n=======\n\n**Bold**\n",
					},
					"type": "section",
				},
			},
		}
		// assert Unmarshal from YAML
		require.Equal(t, expectedBlocks, prop.Announce.Slack.Blocks)

		jazon, err := json.Marshal(prop.Announce.Slack.Blocks)
		require.NoError(t, err)

		var untyped []SlackBlock
		require.NoError(t, json.Unmarshal(jazon, &untyped))

		// assert that JSON Marshal didn't alter the struct
		require.Equal(t, expectedBlocks, prop.Announce.Slack.Blocks)
	})

	t.Run("invalid blocks", func(t *testing.T) {
		t.Parallel()

		_, err := LoadReader(badBlocksSlackConf())
		require.Error(t, err)
	})
}

func TestUnmarshalSlackAttachments(t *testing.T) {
	t.Parallel()

	t.Run("valid attachments", func(t *testing.T) {
		t.Parallel()

		prop, err := LoadReader(goodAttachmentsSlackConf())
		require.NoError(t, err)

		expectedAttachments := []SlackAttachment{
			{
				Internal: map[string]interface{}{
					"color": "#46a64f",
					"fields": []interface{}{
						map[string]interface{}{
							"short": false,
							"title": "field 1",
							"value": "value 1",
						},
					},
					"footer": "a footer",
					"mrkdwn_in": []interface{}{
						"text",
					},
					"pretext": "optional",
					"text":    "another",
					"title":   "my_title",
				},
			},
		}
		// assert Unmarshal from YAML
		require.Equal(t, expectedAttachments, prop.Announce.Slack.Attachments)

		jazon, err := json.Marshal(prop.Announce.Slack.Attachments)
		require.NoError(t, err)

		var untyped []SlackAttachment
		require.NoError(t, json.Unmarshal(jazon, &untyped))

		// assert that JSON Marshal didn't alter the struct
		require.Equal(t, expectedAttachments, prop.Announce.Slack.Attachments)
	})

	t.Run("invalid attachments", func(t *testing.T) {
		t.Parallel()

		_, err := LoadReader(badAttachmentsSlackConf())
		require.Error(t, err)
	})
}

func goodBlocksSlackConf() io.Reader {
	const conf = `
announce:
  slack:
    enabled: true
    username: my_user
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

            **Bold**
`

	buf := bytes.NewBufferString(conf)

	return bytes.NewReader(bytes.ReplaceAll(buf.Bytes(), []byte("\t"), []byte("    ")))
}

func badBlocksSlackConf() io.Reader {
	const conf = `
announce:
  slack:
    enabled: true
    username: my_user
    message_template: fallback
    channel: my_channel
    blocks:
      type: header
        text:
          type: plain_text
          text: '{{ .Version }}'
      type: section
        text:
          type: mrkdwn
          text: |
            **Bold**
`

	buf := bytes.NewBufferString(conf)

	return bytes.NewReader(bytes.ReplaceAll(buf.Bytes(), []byte("\t"), []byte("    ")))
}

func goodAttachmentsSlackConf() io.Reader {
	const conf = `
announce:
  slack:
    enabled: true
    username: my_user
    message_template: fallback
    channel: my_channel
    attachments:
      - mrkdwn_in: ["text"]
        color: '#46a64f'
        pretext: optional
        title: my_title
        text: another
        fields:
          - title: field 1
            value: value 1
            short: false
        footer: a footer
`

	buf := bytes.NewBufferString(conf)

	return bytes.NewReader(bytes.ReplaceAll(buf.Bytes(), []byte("\t"), []byte("    ")))
}

func badAttachmentsSlackConf() io.Reader {
	const conf = `
announce:
  slack:
    enabled: true
    username: my_user
    message_template: fallback
    channel: my_channel
    attachments:
      key:
        mrkdwn_in: ["text"]
        color: #46a64f
        pretext: optional
        title: my_title
        text: another
        fields:
          - title: field 1
            value: value 1
            short: false
        footer: a footer
`

	buf := bytes.NewBufferString(conf)

	return bytes.NewReader(bytes.ReplaceAll(buf.Bytes(), []byte("\t"), []byte("    ")))
}
