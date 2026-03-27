package hooks

import "regexp"

// Matcher evaluates regex patterns against step IDs.
type Matcher struct {
	re *regexp.Regexp
}

// NewMatcher creates a Matcher from a regex pattern string.
// An empty pattern matches everything.
func NewMatcher(pattern string) (*Matcher, error) {
	if pattern == "" {
		return &Matcher{re: nil}, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &Matcher{re: re}, nil
}

// Match returns true if the given step ID matches the pattern.
// An empty pattern (nil regex) matches all step IDs.
func (m *Matcher) Match(stepID string) bool {
	if m.re == nil {
		return true
	}
	return m.re.MatchString(stepID)
}
