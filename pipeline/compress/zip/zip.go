package zip

import (
	"archive/zip"
	"io"
	"os"
)

type Archive struct {
	z *zip.Writer
}

func (a Archive) Close() error {
	return a.z.Close()
}

func New(target *os.File) Archive {
	return Archive{
		z: zip.NewWriter(target),
	}
}

func (a Archive) Add(name, path string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	f, err := a.z.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, file)
	return err
}
