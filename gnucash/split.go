package gnucash

import (
	"errors"
	"fmt"
)

type Split struct {
	ID              GUID            `xml:"id"`
	ReconciledState ReconciledState `xml:"reconciled-state"`
	Value           Value           `xml:"value"`
	Quantity        Value           `xml:"quantity"`
	AccountID       GUID            `xml:"account"`
	Memo            string          `xml:"memo"`
	Account         *Account        `xml:"-"`
}

func (s *Split) String() string {
	dir := ">"
	if s.Value < 0 {
		dir = "<"
	}

	return fmt.Sprintf(
		"[%s] %8.2f %s %s",
		s.ReconciledState,
		s.Value,
		dir,
		s.Account.FQN,
	)
}

func (s *Split) validate(lookup *AccountsLookup) error {
	if s.ID == "" {
		return errors.New("Empty split id")
	}

	s.Account, _ = lookup.ByGUID(s.AccountID)

	if s.ReconciledState != ReconciledStateNew &&
		s.ReconciledState != ReconciledStateCleared &&
		s.ReconciledState != ReconciledStateReconciled &&
		s.ReconciledState != ReconciledStateFrozen &&
		s.ReconciledState != ReconciledStateVoid {
		return fmt.Errorf("Invalid reconciled state '%s'", s.ReconciledState)
	}

	return nil
}
