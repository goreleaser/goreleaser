package scoop

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func Test_getDownloadURL(t *testing.T) {
	type args struct {
		githubURL string
		owner     string
		name      string
		version   string
		file      string
	}
	tests := []struct {
		name    string
		args    args
		wantURL string
	}{
		{"1", args{"https://github.com", "Southclaws", "sampctl", "1.4.0-RC13", "sampctl_1.4.0-RC13_darwin_386.tar.gz"},
			"https://github.com/Southclaws/sampctl/releases/download/1.4.0-RC13/sampctl_1.4.0-RC13_darwin_386.tar.gz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL := getDownloadURL(tt.args.githubURL, tt.args.owner, tt.args.name, tt.args.version, tt.args.file)
			assert.Equal(t, tt.wantURL, gotURL)
		})
	}
}

type DummyClient struct {
	CreatedFile bool
	Content     string
}

func (client *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID int, err error) {
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, content bytes.Buffer, path string) (err error) {
	client.CreatedFile = true
	bts, _ := ioutil.ReadAll(&content)
	client.Content = string(bts)
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int, name string, file *os.File) (err error) {
	return
}
