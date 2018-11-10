package gnucash

import "strings"

type Accounts []*Account
type AccountsLookup struct {
	byGUID map[GUID]*Account
	byFQN  map[string]*Account
}

func (as Accounts) lookup() *AccountsLookup {
	lookup := &AccountsLookup{
		make(map[GUID]*Account, len(as)),
		make(map[string]*Account, len(as)),
	}

	for _, a := range as {
		lookup.byGUID[a.ID] = a
		a.Children = make(Accounts, 0)
	}

	for _, a := range as {
		if a.ParentID == "" {
			continue
		}
		a.Parent = lookup.byGUID[a.ParentID]
		lookup.byGUID[a.ParentID].Children = append(
			lookup.byGUID[a.ParentID].Children,
			a,
		)
	}

	for _, a := range as {
		lookup.byFQN[a.fqn()] = a
	}

	return lookup
}

func (as Accounts) root() *Account {
	for _, a := range as {
		if a.Type == AccountTypeRoot {
			return a
		}
	}

	return nil
}

func (as Accounts) RootString() string {
	return as.root().String()
}
func (as Accounts) String() string {
	str := make([]string, len(as))
	for i, a := range as {
		str[i] = a.String()
	}

	return strings.Join(str, "\n")
}

func (as Accounts) validate(lookup *AccountsLookup, txLookup TransactionsLookup) error {
	for _, a := range as {
		if err := a.validate(lookup, txLookup); err != nil {
			return err
		}
	}

	return nil
}

func (as *AccountsLookup) ByGUID(id GUID) (*Account, bool) {
	a, ok := as.byGUID[id]
	return a, ok
}

func (as *AccountsLookup) ByFQN(fqn string) (*Account, bool) {
	a, ok := as.byFQN[fqn]
	return a, ok
}
