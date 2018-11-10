package gnucash

import "regexp"

var matchesNothing *regexp.Regexp

func init() {
	matchesNothing = regexp.MustCompile(`$^`)
}

func CompileRegexp(regex string) *regexp.Regexp {
	r, err := regexp.Compile(regex)
	if err != nil {
		return matchesNothing
	}

	return r
}

func matchFQN(include, exclude *regexp.Regexp, accountFQN string) bool {
	return (include == nil || include.MatchString(accountFQN)) &&
		(exclude == nil || !exclude.MatchString(accountFQN))
}
