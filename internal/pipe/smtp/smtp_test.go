package smtp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
	gomail "gopkg.in/mail.v2"
)

func TestStringer(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.Wrap(t.Context()))
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Announce: config.Announce{
				SMTP: config.SMTP{
					Enabled: "true",
				},
			},
		})

		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, skip)
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Announce: config.Announce{
			SMTP: config.SMTP{
				Enabled: "true",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultBodyTemplate, ctx.Config.Announce.SMTP.BodyTemplate)
	require.Equal(t, defaultSubjectTemplate, ctx.Config.Announce.SMTP.SubjectTemplate)
}

func TestAnnounceTLSConfigUsesServerName(t *testing.T) {
	t.Run("starttls", func(t *testing.T) {
		requireTLSValidationError(t, 587, serveSMTPStartTLS)
	})

	t.Run("implicit tls", func(t *testing.T) {
		requireTLSValidationError(t, 465, serveImplicitSMTPTLS)
	})
}

func requireTLSValidationError(t *testing.T, port int, serve func(net.Listener, tls.Certificate, chan<- error)) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = ln.Close()
	})

	previousDial := gomail.NetDialTimeout
	gomail.NetDialTimeout = func(network, _ string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout(network, ln.Addr().String(), timeout)
	}
	t.Cleanup(func() {
		gomail.NetDialTimeout = previousDial
	})

	serverErr := make(chan error, 1)
	go serve(ln, smtpTestCertificate(t), serverErr)

	t.Setenv("SMTP_PASSWORD", "secret")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Announce: config.Announce{
			SMTP: config.SMTP{
				Enabled:         "true",
				Host:            "smtp.example.test",
				Port:            port,
				Username:        "user",
				From:            "from@example.com",
				To:              []string{"to@example.com"},
				SubjectTemplate: "subject",
				BodyTemplate:    "body",
			},
		},
	})

	err = Pipe{}.Announce(ctx)
	require.Error(t, err)

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP TLS handshake")
	}

	require.NotContains(t, err.Error(), "either ServerName or InsecureSkipVerify")
	require.Contains(t, err.Error(), "certificate")
}

func smtpTestCertificate(t *testing.T) tls.Certificate {
	t.Helper()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(srv.Close)
	require.NotEmpty(t, srv.TLS.Certificates)
	return srv.TLS.Certificates[0]
}

func serveImplicitSMTPTLS(ln net.Listener, cert tls.Certificate, errCh chan<- error) {
	conn, err := ln.Accept()
	if err != nil {
		errCh <- err
		return
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		errCh <- err
		return
	}

	errCh <- tls.Server(conn, &tls.Config{
		Certificates: []tls.Certificate{cert},
	}).Handshake()
}

func serveSMTPStartTLS(ln net.Listener, cert tls.Certificate, errCh chan<- error) {
	conn, err := ln.Accept()
	if err != nil {
		errCh <- err
		return
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		errCh <- err
		return
	}

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	if err := writeSMTP(rw, "220 smtp.example.test ESMTP"); err != nil {
		errCh <- err
		return
	}
	line, err := rw.ReadString('\n')
	if err != nil {
		errCh <- err
		return
	}
	if !strings.HasPrefix(line, "EHLO ") && !strings.HasPrefix(line, "HELO ") {
		errCh <- fmt.Errorf("expected EHLO or HELO, got %q", line)
		return
	}
	if err := writeSMTP(rw, "250-smtp.example.test\r\n250-STARTTLS\r\n250 AUTH PLAIN"); err != nil {
		errCh <- err
		return
	}
	line, err = rw.ReadString('\n')
	if err != nil {
		errCh <- err
		return
	}
	if !strings.HasPrefix(line, "STARTTLS") {
		errCh <- fmt.Errorf("expected STARTTLS, got %q", line)
		return
	}
	if err := writeSMTP(rw, "220 ready"); err != nil {
		errCh <- err
		return
	}

	errCh <- tls.Server(conn, &tls.Config{
		Certificates: []tls.Certificate{cert},
	}).Handshake()
}

func writeSMTP(rw *bufio.ReadWriter, msg string) error {
	if _, err := rw.WriteString(msg + "\r\n"); err != nil {
		return err
	}
	return rw.Flush()
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
