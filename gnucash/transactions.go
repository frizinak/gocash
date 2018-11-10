package gnucash

import (
	"regexp"
	"strings"
	"time"
)

type Transactions []*Transaction
type TransactionsLookup map[GUID]Transactions

func (ts Transactions) String() string {
	str := make([]string, len(ts))
	for i, t := range ts {
		str[i] = t.String()
	}

	return strings.Join(str, "\n")
}

func (ts Transactions) Between(start, end time.Time) Transactions {
	l := make(Transactions, 0, len(ts))
	for _, t := range ts {
		d := t.DateEntered.Get()
		if d.Before(start) || d.After(end) {
			continue
		}

		l = append(l, t)
	}

	return l
}

func (ts Transactions) ValueForAccount(
	accountID GUID,
	includeChildren bool,
) Value {
	var v Value
	for _, t := range ts {
		v += t.Splits.ValueForAccount(accountID, includeChildren)
	}

	return v
}

func (ts Transactions) Filter(
	accountFQN,
	accountExcludeFQN *regexp.Regexp,
) Transactions {
	txs := make(Transactions, 0)
	for _, t := range ts {
		for _, s := range t.Splits {
			if matchFQN(accountFQN, accountExcludeFQN, s.Account.FQN) {
				txs = append(txs, t)
				break
			}
		}
	}

	return txs
}

func (ts Transactions) FilterSplits(
	accountFQN,
	accountExcludeFQN *regexp.Regexp,
) Splits {
	ss := make(Splits, 0)
	for _, t := range ts {
		ss = append(ss, t.Splits.Filter(accountFQN, accountExcludeFQN)...)
	}

	return ss
}

func (ts Transactions) Simplified() FlatTransactions {
	return ts.mapTransactions().flatten().flattxs()
}

func (ts Transactions) lookup() TransactionsLookup {
	lookup := make(TransactionsLookup)
	for _, t := range ts {
		for _, s := range t.Splits {
			lookup[s.AccountID] = append(lookup[s.AccountID], t)
		}
	}

	return lookup
}

func (ts Transactions) validate(lookup *AccountsLookup) error {
	for _, t := range ts {
		if err := t.validate(lookup); err != nil {
			return err
		}
	}

	return nil
}

func (ts TransactionsLookup) Find(accountID GUID) (Transactions, bool) {
	t, ok := ts[accountID]
	return t, ok
}
