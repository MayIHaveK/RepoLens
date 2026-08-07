package filter

import (
	"regexp"
	"strings"
)

type Matcher struct {
	patterns []*regexp.Regexp
}

func NewMatcher(patterns []string) Matcher {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(strings.ReplaceAll(pattern, "\\", "/"))
		if pattern == "" {
			continue
		}
		if expression, err := regexp.Compile(globExpression(pattern)); err == nil {
			compiled = append(compiled, expression)
		}
	}
	return Matcher{patterns: compiled}
}

func (m Matcher) Match(path string) bool {
	path = strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
	for _, pattern := range m.patterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

func globExpression(pattern string) string {
	var out strings.Builder
	out.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++
					out.WriteString("(?:.*/)?")
				} else {
					out.WriteString(".*")
				}
			} else {
				out.WriteString("[^/]*")
			}
		case '?':
			out.WriteString("[^/]")
		default:
			out.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	out.WriteString("$")
	return out.String()
}
