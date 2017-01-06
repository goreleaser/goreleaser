package tar

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
)

type Archive struct {
	gw *gzip.Writer
	tw *tar.Writer
}

func (a Archive) Close() error {
	if err := a.gw.Close(); err != nil {
		return err
	}
	if err := a.tw.Close(); err != nil {
		return err
	}
	return nil
}

func New(target *os.File) Archive {
	gw := gzip.NewWriter(target)
	tw := tar.NewWriter(gw)
	return Archive{
		gw: gw,
		tw: tw,
	}
}

func (a Archive) Add(name, path string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()
	stat, err := file.Stat()
	if err != nil {
		return
	}
	header := new(tar.Header)
	header.Name = name
	header.Size = stat.Size()
	header.Mode = int64(stat.Mode())
	header.ModTime = stat.ModTime()
	if err := a.tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := io.Copy(a.tw, file); err != nil {
		return err
	}
	return
}
