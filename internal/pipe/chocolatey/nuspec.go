package chocolatey

import (
	"bytes"
	"encoding/xml"
	"strings"
)

const schema = "http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd"

// Nuspec represents a Nuget/Chocolatey Nuspec.
// More info: https://learn.microsoft.com/en-us/nuget/reference/nuspec
// https://docs.chocolatey.org/en-us/create/create-packages
type Nuspec struct {
	XMLName  xml.Name `xml:"package"`
	Xmlns    string   `xml:"xmlns,attr,omitempty"`
	Metadata Metadata `xml:"metadata"`
	Files    Files    `xml:"files,omitempty"`
}

// Metadata contains information about a single package.
type Metadata struct {
	ID                       string        `xml:"id"`
	Version                  string        `xml:"version"`
	PackageSourceURL         string        `xml:"packageSourceUrl,omitempty"`
	Owners                   string        `xml:"owners,omitempty"`
	Title                    string        `xml:"title,omitempty"`
	Authors                  string        `xml:"authors"`
	ProjectURL               string        `xml:"projectUrl,omitempty"`
	IconURL                  string        `xml:"iconUrl,omitempty"`
	Copyright                string        `xml:"copyright,omitempty"`
	LicenseURL               string        `xml:"licenseUrl,omitempty"`
	RequireLicenseAcceptance bool          `xml:"requireLicenseAcceptance"`
	ProjectSourceURL         string        `xml:"projectSourceUrl,omitempty"`
	DocsURL                  string        `xml:"docsUrl,omitempty"`
	BugTrackerURL            string        `xml:"bugTrackerUrl,omitempty"`
	Tags                     string        `xml:"tags,omitempty"`
	Summary                  string        `xml:"summary,omitempty"`
	Description              string        `xml:"description"`
	ReleaseNotes             string        `xml:"releaseNotes,omitempty"`
	Dependencies             *Dependencies `xml:"dependencies,omitempty"`
}

// Dependency represents a dependency element.
type Dependency struct {
	ID      string `xml:"id,attr"`
	Version string `xml:"version,attr,omitempty"`
}

// Dependencies represents a collection zero or more dependency elements.
type Dependencies struct {
	Dependency []Dependency `xml:"dependency"`
}

// File represents a file to be copied.
type File struct {
	Source string `xml:"src,attr"`
	Target string `xml:"target,attr,omitempty"`
}

// Files represents files that will be copied during packaging.
type Files struct {
	File []File `xml:"file"`
}

// Bytes marshals the Nuspec into XML format and return as []byte.
func (m *Nuspec) Bytes() ([]byte, error) {
	b := &bytes.Buffer{}
	b.WriteString(strings.ToLower(xml.Header))

	enc := xml.NewEncoder(b)
	enc.Indent("", "  ")

	if err := enc.Encode(m); err != nil {
		return nil, err
	}

	out := b.Bytes()

	// Follows the nuget specification of self-closing xml tags.
	tags := []string{"dependency", "file"}
	for _, tag := range tags {
		out = bytes.ReplaceAll(out, []byte("></"+tag+">"), []byte(" />"))
	}

	return out, nil
}
