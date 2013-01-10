package pie

import (
	"regexp"
)

type rePathMatcher struct {
	*regexp.Regexp
}

func (r rePathMatcher) PathMatch(p string) bool {
	return r.MatchString(p)
}

func RegExpPathMatcher(re string) (PathMatcher, error) {
	c, err := regexp.Compile(re)
	if err != nil {
		return nil, err
	}
	return rePathMatcher{c}, nil
}
