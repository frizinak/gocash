package gnucash

import (
	"regexp"
	"strings"
)

type Splits []*Split

func (ss Splits) String() string {
	str := make([]string, len(ss))
	for i, s := range ss {
		str[i] = s.String()
	}

	return strings.Join(str, "\n")
}

func (ss Splits) ValueForAccount(accountID GUID, includeChildren bool) Value {
	var v Value
	for _, s := range ss {
		p := s.Account
		for {
			if p == nil {
				break
			}

			if p.ID == accountID {
				v += s.Value
			}

			if !includeChildren {
				break
			}

			p = p.Parent
		}
	}

	return v
}

func (ss Splits) Filter(
	accountFQN,
	accountExcludeFQN *regexp.Regexp,
) Splits {
	f := make(Splits, 0)
	for _, s := range ss {
		if matchFQN(accountFQN, accountExcludeFQN, s.Account.FQN) {
			f = append(f, s)
		}
	}

	return f
}

func (ss Splits) Sum() Value {
	var v Value
	for _, s := range ss {
		v += s.Value
	}

	return v
}

func (ss Splits) validate(lookup *AccountsLookup) error {
	for _, s := range ss {
		if err := s.validate(lookup); err != nil {
			return err
		}
	}

	return nil
}
