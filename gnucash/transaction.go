package gnucash

import (
	"errors"
	"fmt"
	"time"
)

type Transaction struct {
	ID          GUID   `xml:"id"`
	DatePosted  Date   `xml:"date-posted>date"`
	DateEntered Date   `xml:"date-entered>date"`
	Description string `xml:"description"`
	Splits      Splits `xml:"splits>split"`
}

func (t *Transaction) String() string {
	return fmt.Sprintf(
		"TRANSACTION\nID: %s\nDate: %s\nDescription: %s\nSplits:\n%s",
		t.ID,
		t.DatePosted.Get().Format(time.RFC822),
		t.Description,
		t.Splits.String(),
	)
}

func (t *Transaction) validate(lookup *AccountsLookup) error {
	if t.ID == "" {
		return errors.New("Empty transaction id")
	}

	if err := t.Splits.validate(lookup); err != nil {
		return err
	}

	return nil
}
