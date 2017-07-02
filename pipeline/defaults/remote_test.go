package defaults

import (
	"reflect"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/stretchr/testify/assert"
)

func TestRepoName(t *testing.T) {
	var assert = assert.New(t)
	repo, err := remoteRepo()
	assert.NoError(err)
	assert.Equal("goreleaser/goreleaser", repo.String())
}

func TestExtractReporFromGitURL(t *testing.T) {
	var assert = assert.New(t)
	repo := extractRepoFromURL("git@github.com:goreleaser/goreleaser.git")
	assert.Equal("goreleaser/goreleaser", repo.String())
}

func TestExtractReporFromHttpsURL(t *testing.T) {
	var assert = assert.New(t)
	repo := extractRepoFromURL("https://github.com/goreleaser/goreleaser.git")
	assert.Equal("goreleaser/goreleaser", repo.String())
}

func Test_remoteRepo(t *testing.T) {
	tests := []struct {
		name       string
		wantResult config.Repo
		wantErr    bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := remoteRepo()
			if (err != nil) != tt.wantErr {
				t.Errorf("remoteRepo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("remoteRepo() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func Test_extractRepoFromURL(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want config.Repo
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractRepoFromURL(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractRepoFromURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toRepo(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want config.Repo
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toRepo(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}
