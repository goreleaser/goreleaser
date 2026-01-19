package changelog

import (
	"regexp"
	"strings"
)

// Item represents a changelog item, basically, a commit and its authors.
type Item struct {
	SHA     string
	Message string
	Authors []Author

	// Deprecated: use [Item.Authors].
	AuthorName string

	// Deprecated: use [Item.Authors].
	AuthorEmail string

	// Deprecated: use [Item.Authors].
	AuthorUsername string
}

// Author is somebody who authored or co-authored a commit.
type Author struct {
	Name     string
	Email    string
	Username string
}

var coauthorRe = regexp.MustCompile(`(?i)^co-authored-by:\s*([^<]+)\s*<([^>]+)>`)

// ExtractCoAuthors extracts co-authors from a commit message.
//
// It'll parse the 'co-authored-by' trailers according to the convention.
func ExtractCoAuthors(msg string) []Author {
	var authors []Author
	for line := range strings.SplitSeq(msg, "\n") {
		matches := coauthorRe.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		authors = append(authors, Author{
			Name:  strings.TrimSpace(matches[1]),
			Email: strings.TrimSpace(matches[2]),
		})
	}
	return authors
}
