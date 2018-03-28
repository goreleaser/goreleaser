package client

import (
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestFuncMap(t *testing.T) {
	timeNow = func() time.Time {
		return time.Date(2018, 12, 11, 10, 9, 8, 7, time.UTC)
	}
	var ctx = context.New(config.Project{
		ProjectName: "proj",
	})
	for _, tc := range []struct {
		Template string
		Name     string
		Output   string
	}{
		{
			Template: `{{ time "2006-01-02" }}`,
			Name:     "YYYY-MM-DD",
			Output:   "2018-12-11",
		},
		{
			Template: `{{ time "01/02/2006" }}`,
			Name:     "MM/DD/YYYY",
			Output:   "12/11/2018",
		},
	} {
		ctx.Config.Release.NameTemplate = tc.Template
		out, err := releaseTitle(ctx)
		assert.NoError(t, err)
		assert.Equal(t, tc.Output, out)
	}
}
