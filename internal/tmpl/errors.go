package tmpl

import (
	"fmt"
	"regexp"
)

var res = []*regexp.Regexp{
	regexp.MustCompile(`^template: tmpl:\d+:\d+: executing ".+" at .+: `),
	regexp.MustCompile(`^template: tmpl:\d+:\d+: `),
	regexp.MustCompile(`^template: tmpl:\d+: `),
}

func newTmplError(str string, err error) error {
	if err == nil {
		return nil
	}
	details := err.Error()
	for _, re := range res {
		if re.MatchString(details) {
			details = re.ReplaceAllString(details, "")
			break
		}
	}
	return Error{str, details}
}

// Error is returned on any template error.
type Error struct {
	str     string
	details string
}

func (e Error) Error() string {
	return fmt.Sprintf(
		"template: failed to apply %q: %s",
		e.str,
		e.details,
	)
}
