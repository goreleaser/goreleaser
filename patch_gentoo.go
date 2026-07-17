package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("internal/pipe/gentoo/gentoo.go")
	s := string(b)

	// Add encoding/xml
	if !strings.Contains(s, "\"encoding/xml\"") {
		s = strings.Replace(s, "\"crypto/sha512\"", "\"crypto/sha512\"\n\t\"encoding/xml\"", 1)
	}

	searchXML := `
	if len(cfg.Maintainers) > 0 || cfg.BugsTo != "" || cfg.Homepage != "" {
		var buf bytes.Buffer
		buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
		buf.WriteString("<!DOCTYPE pkgmetadata SYSTEM \"https://www.gentoo.org/dtd/metadata.dtd\">\n")
		buf.WriteString("<pkgmetadata>\n")
		for _, m := range cfg.Maintainers {
			fmt.Fprintf(&buf, "\t<maintainer type=\"person\">\n")
			if m.Email != "" {
				fmt.Fprintf(&buf, "\t\t<email>%s</email>\n", m.Email)
			}
			if m.Name != "" {
				fmt.Fprintf(&buf, "\t\t<name>%s</name>\n", m.Name)
			}
			fmt.Fprintf(&buf, "\t</maintainer>\n")
		}
		if cfg.BugsTo != "" || cfg.Homepage != "" {
			fmt.Fprintf(&buf, "\t<upstream>\n")
			if cfg.BugsTo != "" {
				fmt.Fprintf(&buf, "\t\t<bugs-to>%s</bugs-to>\n", cfg.BugsTo)
			}
			if cfg.Homepage != "" {
				fmt.Fprintf(&buf, "\t\t<doc>%s</doc>\n", cfg.Homepage)
			}
			fmt.Fprintf(&buf, "\t</upstream>\n")
		}
		buf.WriteString("</pkgmetadata>\n")
		*files = append(*files, client.RepoFile{
			Content: buf.Bytes(),
			Path:    filepath.ToSlash(filepath.Join(dir, "metadata.xml")),
		})
	}`

	replaceXML := `
	if len(cfg.Maintainers) > 0 || cfg.BugsTo != "" || cfg.Homepage != "" {
		type gentooMaintainer struct {
			Type  string "xml:\"type,attr\""
			Email string "xml:\"email,omitempty\""
			Name  string "xml:\"name,omitempty\""
		}
		type gentooUpstream struct {
			BugsTo string "xml:\"bugs-to,omitempty\""
			Doc    string "xml:\"doc,omitempty\""
		}
		type gentooMetadata struct {
			XMLName     xml.Name           "xml:\"pkgmetadata\""
			Maintainers []gentooMaintainer "xml:\"maintainer\""
			Upstream    *gentooUpstream    "xml:\"upstream,omitempty\""
		}

		meta := gentooMetadata{}
		for _, m := range cfg.Maintainers {
			meta.Maintainers = append(meta.Maintainers, gentooMaintainer{
				Type:  "person",
				Email: m.Email,
				Name:  m.Name,
			})
		}
		if cfg.BugsTo != "" || cfg.Homepage != "" {
			meta.Upstream = &gentooUpstream{
				BugsTo: cfg.BugsTo,
				Doc:    cfg.Homepage,
			}
		}

		marshaled, err := xml.MarshalIndent(meta, "", "\t")
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
		buf.WriteString("<!DOCTYPE pkgmetadata SYSTEM \"https://www.gentoo.org/dtd/metadata.dtd\">\n")
		buf.Write(marshaled)
		buf.WriteString("\n")

		*files = append(*files, client.RepoFile{
			Content: buf.Bytes(),
			Path:    filepath.ToSlash(filepath.Join(dir, "metadata.xml")),
		})
	}`

	s = strings.Replace(s, searchXML, replaceXML, 1)
	os.WriteFile("internal/pipe/gentoo/gentoo.go", []byte(s), 0644)
	fmt.Println("Applied XML fix")
}
