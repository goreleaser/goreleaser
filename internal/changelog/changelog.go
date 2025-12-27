package changelog

import (
	"regexp"
	"strings"
)

// Item represents a changelog item, basically, a commit and its authors.
type Item struct {
	SHA       string
	Message   string
	Author    Author
	CoAuthors []Author

	// Deprecated: use [ChangelogItem.Author].
	AuthorName string

	// Deprecated: use [ChangelogItem.Author].
	AuthorEmail string

	// Deprecated: use [ChangelogItem.Author].
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
		matches := coauthorRe.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 || len(matches[0]) != 3 {
			continue
		}
		match := matches[0]
		authors = append(authors, Author{
			Name:  strings.TrimSpace(match[1]),
			Email: strings.TrimSpace(match[2]),
		})
	}
	return authors
}
