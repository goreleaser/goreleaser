// Package nexus_raw provides a Pipe that push artifacts to a Sonatype Nexus repository of type 'raw'
package nexus_raw

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string {
	return "Sonatype Nexus Raw Repositories"
}

func (Pipe) Publish(ctx *context.Context) error {
	if len(ctx.Config.NexusRaws) == 0 {
		return pipe.Skip("sonatype nexus section is not configured")
	}
	for _, instance := range ctx.Config.NexusRaws {
		instance := instance
		if err := checkConfig(&instance); err != nil {
			return err
		}
	}

	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}

	for _, nexus := range ctx.Config.NexusRaws {
		if err := upload(ctx, &nexus); err != nil {
			return err
		}
	}
	return nil
}

func upload(ctx *context.Context, nexus *config.NexusRaw) error {
	g := semerrgroup.New(ctx.Parallelism)
	for _, art := range ctx.Artifacts.Filter(
		artifact.Or(
			artifact.ByType(artifact.UploadableArchive),
			artifact.ByType(artifact.UploadableBinary),
			artifact.ByType(artifact.Checksum),
			artifact.ByType(artifact.Signature),
			artifact.ByType(artifact.LinuxPackage),
		),
	).List() {
		art := art
		g.Go(func() error {
			return uploadAsset(nexus, art)
		})
	}
	return g.Wait()
}

func uploadAsset(nexus *config.NexusRaw, artif *artifact.Artifact) error {
	var b bytes.Buffer
	log.WithField("file", artif.Path).WithField("name", artif.Name).Info("uploading to nexus")
	w := multipart.NewWriter(&b)
	if err := createForm(w, artif.Path, artif.Name, nexus.Directory); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/service/rest/v1/components?repository=%s", nexus.URL, nexus.Repository)
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if nexus.Username != "" {
		req.SetBasicAuth(nexus.Username, nexus.Password)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		var bodyContents string
		body, err := ioutil.ReadAll(res.Body)
		if err == nil {
			bodyContents = string(body)
		}
		return pipe.Skip(fmt.Sprintf("bad status: %s, body: %s", res.Status, bodyContents))
	}
	return nil
}

func createForm(w *multipart.Writer, artifPath, artifName, nexusDir string) error {
	asset, err := os.Open(artifPath)
	if err != nil {
		return fmt.Errorf("cannot open artifact file: '%s'", err.Error())
	}
	defer asset.Close()
	artifactField, err := w.CreateFormFile("raw.asset1", asset.Name())
	if err != nil {
		return fmt.Errorf("cannot create post field for artifact: '%s'", err.Error())
	}
	if _, err = io.Copy(artifactField, asset); err != nil {
		return fmt.Errorf("cannot copy artifact contents to post field: '%s'", err.Error())
	}
	directoryField, err := w.CreateFormField("raw.directory")
	if err != nil {
		return fmt.Errorf("cannot create directory field: '%s'", err.Error())
	}
	if _, err = io.Copy(directoryField, strings.NewReader(nexusDir)); err != nil {
		return fmt.Errorf("cannot fill directory field contents: '%s'", err.Error())
	}
	filenameField, err := w.CreateFormField("raw.asset1.filename")
	if err != nil {
		return fmt.Errorf("cannot create filename field: '%s'", err.Error())
	}
	if _, err = io.Copy(filenameField, strings.NewReader(artifName)); err != nil {
		return fmt.Errorf("cannot fill filename field contents: '%s'", err.Error())
	}
	return w.Close()
}

func checkConfig(nexus *config.NexusRaw) error {
	if nexus.URL == "" {
		return pipe.Skip("nexus_raws section 'url' is not configured properly (missing url)")
	}
	if nexus.Repository == "" {
		return pipe.Skip("nexus_raws section 'repository' is not configured properly (missing repository name)")
	}
	if nexus.Directory == "" {
		return pipe.Skip("nexus_raws section 'directory' is not configured properly (missing directory)")
	}
	if (nexus.Username == "" || nexus.Password == "") && (nexus.Username != "" || nexus.Password != "") {
		return pipe.Skip("nexus_raws sections 'username' and 'password' are not configured properly (they must be either both empty or both filled)")
	}
	return nil
}
