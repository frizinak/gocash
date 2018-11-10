package gnucash

import "strings"

type Schedules []*Scheduled

func (ss Schedules) String() string {
	str := make([]string, len(ss))
	for i, s := range ss {
		str[i] = s.String()
	}

	return strings.Join(str, "\n")
}

func (ss Schedules) validate(lookup *AccountsLookup) error {
	for _, s := range ss {
		if err := s.validate(lookup); err != nil {
			return err
		}
	}

	return nil
}
