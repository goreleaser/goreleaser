package tarzst

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

func TestTarZstConfig(t *testing.T) {
	data, err := os.ReadFile("tarzst.go")
	require.NoError(t, err)
	data = bytes.Repeat(data, 128)
	source := filepath.Join(t.TempDir(), "binary")
	require.NoError(t, os.WriteFile(source, data, 0o755))

	for _, tc := range []struct {
		name    string
		level   string
		encoder zstd.EncoderLevel
		window  int
	}{
		{name: "unset"},
		{name: "fastest", level: "fastest", encoder: zstd.SpeedFastest},
		{name: "default", level: "default", encoder: zstd.SpeedDefault},
		{name: "better", level: "better", encoder: zstd.SpeedBetterCompression},
		{name: "best", level: "best", encoder: zstd.SpeedBestCompression},
		{name: "window only", window: 10},
		{name: "level and window", level: "better", encoder: zstd.SpeedBetterCompression, window: 20},
		{name: "large window", level: "fastest", encoder: zstd.SpeedFastest, window: 27},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Zstd{Level: tc.level}
			var opts []zstd.EOption
			if tc.level != "" {
				opts = append(opts, zstd.WithEncoderLevel(tc.encoder))
			}
			if tc.window != 0 {
				cfg.WindowSize = &tc.window
				opts = append(opts, zstd.WithWindowSize(1<<tc.window))
			}
			var compressed bytes.Buffer
			a, err := New(&compressed, cfg)
			require.NoError(t, err)
			require.NoError(t, a.Add(config.File{Source: source, Destination: "binary"}))
			require.NoError(t, a.Close())

			decoder, err := zstd.NewReader(nil)
			require.NoError(t, err)
			defer decoder.Close()
			uncompressed, err := decoder.DecodeAll(compressed.Bytes(), nil)
			require.NoError(t, err)
			reader := tar.NewReader(bytes.NewReader(uncompressed))
			header, err := reader.Next()
			require.NoError(t, err)
			require.Equal(t, "binary", header.Name)
			actual, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.Equal(t, data, actual)
			_, err = reader.Next()
			require.ErrorIs(t, err, io.EOF)

			var expected bytes.Buffer
			encoder, err := zstd.NewWriter(&expected, opts...)
			require.NoError(t, err)
			_, err = encoder.Write(uncompressed)
			require.NoError(t, err)
			require.NoError(t, encoder.Close())
			require.Equal(t, expected.Len(), compressed.Len(), "compressed size must match the configured encoder")
			require.True(t, bytes.Equal(expected.Bytes(), compressed.Bytes()), "archive must match the configured encoder output")
		})
	}
}

func TestTarZstConfigInvalid(t *testing.T) {
	t.Run("level", func(t *testing.T) {
		_, err := New(io.Discard, config.Zstd{Level: "invalid"})
		require.EqualError(t, err, `invalid zstd level "invalid": must be fastest, default, better, or best`)
	})
	for _, window := range []int{-1, 0, 9, 30, 64} {
		t.Run(fmt.Sprintf("window/%d", window), func(t *testing.T) {
			_, err := New(io.Discard, config.Zstd{WindowSize: &window})
			require.EqualError(t, err, fmt.Sprintf("invalid zstd window_size %d: must be between 10 and 29 (log2 bytes)", window))
		})
	}
}

func TestTarZstFile(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "test.tar.zst"))
	require.NoError(t, err)
	defer f.Close()
	archive, err := New(f, config.Zstd{})
	require.NoError(t, err)
	defer archive.Close()

	require.Error(t, archive.Add(config.File{
		Source:      "../testdata/nope.txt",
		Destination: "nope.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1",
		Destination: "sub1",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1/bar.txt",
		Destination: "sub1/bar.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1/executable",
		Destination: "sub1/executable",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1/sub2",
		Destination: "sub1/sub2",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1/sub2/subfoo.txt",
		Destination: "sub1/sub2/subfoo.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/regular.txt",
		Destination: "regular.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/link.txt",
		Destination: "link.txt",
	}))

	require.NoError(t, archive.Close())
	require.Error(t, archive.Add(config.File{
		Source:      "tar.go",
		Destination: "tar.go",
	}))
	require.NoError(t, f.Close())

	f, err = os.Open(f.Name())
	require.NoError(t, err)
	defer f.Close()

	info, err := f.Stat()
	require.NoError(t, err)
	require.Lessf(t, info.Size(), int64(500), "archived file should be smaller than %d", info.Size())

	zstf, err := zstd.NewReader(f)
	require.NoError(t, err)

	var paths []string
	r := tar.NewReader(zstf)
	for {
		next, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		paths = append(paths, next.Name)
		if testlib.IsWindows() {
			// both of the following checks don't work on windows.
			continue
		}
		if next.Name == "sub1/executable" {
			require.NotEqualf(
				t,
				0,
				next.FileInfo().Mode()&0o111,
				"expected executable perms, got %s",
				next.FileInfo().Mode().String(),
			)
		}
		if next.Name == "link.txt" {
			require.Equal(t, "regular.txt", next.Linkname)
		}
	}
	require.Equal(t, []string{
		"foo.txt",
		"sub1",
		"sub1/bar.txt",
		"sub1/executable",
		"sub1/sub2",
		"sub1/sub2/subfoo.txt",
		"regular.txt",
		"link.txt",
	}, paths)
}

func TestTarZstFileInfo(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	f, err := os.Create(filepath.Join(t.TempDir(), "test.tar.gz"))
	require.NoError(t, err)
	defer f.Close()
	archive, err := New(f, config.Zstd{})
	require.NoError(t, err)
	defer archive.Close()

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "nope.txt",
		Info: config.FileInfo{
			Mode:        0o755,
			Owner:       "carlos",
			Group:       "root",
			ParsedMTime: now,
		},
	}))

	require.NoError(t, archive.Close())
	require.NoError(t, f.Close())

	f, err = os.Open(f.Name())
	require.NoError(t, err)
	defer f.Close()

	zstf, err := zstd.NewReader(f)
	require.NoError(t, err)

	var found int
	r := tar.NewReader(zstf)
	for {
		next, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		found++
		require.Equal(t, "nope.txt", next.Name)
		require.Equal(t, now, next.ModTime)
		require.Equal(t, fs.FileMode(0o755), next.FileInfo().Mode())
		require.Equal(t, "carlos", next.Uname)
		require.Equal(t, 0, next.Uid)
		require.Equal(t, "root", next.Gname)
		require.Equal(t, 0, next.Gid)
	}
	require.Equal(t, 1, found)
}
