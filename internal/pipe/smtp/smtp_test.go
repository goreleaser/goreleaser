package smtp

import (
	"strconv"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Announce: config.Announce{
				SMTP: config.SMTP{
					Enabled: true,
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			SMTP: config.SMTP{
				Enabled: true,
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultBodyTemplate, ctx.Config.Announce.SMTP.BodyTemplate)
	require.Equal(t, defaultSubjectTemplate, ctx.Config.Announce.SMTP.SubjectTemplate)
}

func TestGetConfig(t *testing.T) {
	t.Run("from env", func(t *testing.T) {
		expect := Config{
			Host:     "hostname",
			Port:     123,
			Username: "user",
			Password: "secret",
		}
		t.Setenv("SMTP_HOST", expect.Host)
		t.Setenv("SMTP_USERNAME", expect.Username)
		t.Setenv("SMTP_PASSWORD", expect.Password)
		t.Setenv("SMTP_PORT", strconv.Itoa(expect.Port))
		cfg, err := getConfig(config.SMTP{})
		require.NoError(t, err)
		require.Equal(t, expect, cfg)
	})

	t.Run("mixed", func(t *testing.T) {
		expect := Config{
			Host:     "hostname",
			Port:     123,
			Username: "user",
			Password: "secret",
		}
		t.Setenv("SMTP_HOST", expect.Host)
		t.Setenv("SMTP_PASSWORD", expect.Password)
		cfg, err := getConfig(config.SMTP{
			Port:     expect.Port,
			Username: expect.Username,
		})
		require.NoError(t, err)
		require.Equal(t, expect, cfg)
	})

	t.Run("from conf", func(t *testing.T) {
		expect := Config{
			Host:     "hostname",
			Port:     123,
			Username: "user",
			Password: "secret",
		}
		t.Setenv("SMTP_PASSWORD", expect.Password)
		cfg, err := getConfig(config.SMTP{
			Host:     expect.Host,
			Port:     expect.Port,
			Username: expect.Username,
		})
		require.NoError(t, err)
		require.Equal(t, expect, cfg)
	})

	t.Run("no port", func(t *testing.T) {
		t.Setenv("SMTP_HOST", "host")
		t.Setenv("SMTP_PASSWORD", "pwd")
		_, err := getConfig(config.SMTP{
			Username: "user",
		})
		require.ErrorIs(t, err, errNoPort)
	})

	t.Run("no username", func(t *testing.T) {
		t.Setenv("SMTP_HOST", "host")
		t.Setenv("SMTP_PASSWORD", "pwd")
		_, err := getConfig(config.SMTP{
			Port: 10,
		})
		require.ErrorIs(t, err, errNoUsername)
	})

	t.Run("no host", func(t *testing.T) {
		t.Setenv("SMTP_PASSWORD", "pwd")
		_, err := getConfig(config.SMTP{
			Port:     10,
			Username: "user",
		})
		require.ErrorIs(t, err, errNoHost)
	})

	t.Run("no password", func(t *testing.T) {
		_, err := getConfig(config.SMTP{
			Port:     10,
			Username: "user",
			Host:     "host",
		})
		require.EqualError(t, err, "SMTP: env: environment variable \"SMTP_PASSWORD\" should not be empty")
	})
}
