package gnucash

import (
	"errors"
	"fmt"
)

type Book struct {
	ID                 GUID               `xml:"id"`
	Accounts           Accounts           `xml:"account"`
	AccountsLookup     *AccountsLookup    `xml:"-"`
	RootAccount        *Account           `xml:"-"`
	Transactions       Transactions       `xml:"transaction"`
	TransactionsLookup TransactionsLookup `xml:"-"`
	Scheduled          Schedules          `xml:"schedxaction"`
}

func (b *Book) String() string {
	return fmt.Sprintf(
		"BOOK\nID: %s\nAccounts:\n%s\nTransactions:\n%s\nScheduled:\n%s",
		b.ID,
		b.Accounts.RootString(),
		b.Transactions.String(),
		b.Scheduled.String(),
	)
}

func (b *Book) validate() error {
	if b.ID == "" {
		return errors.New("Empty book id")
	}

	b.RootAccount = b.Accounts.root()
	b.AccountsLookup = b.Accounts.lookup()
	b.TransactionsLookup = b.Transactions.lookup()

	if err := b.Accounts.validate(b.AccountsLookup, b.TransactionsLookup); err != nil {
		return err
	}

	if err := b.Transactions.validate(b.AccountsLookup); err != nil {
		return err
	}

	if err := b.Scheduled.validate(b.AccountsLookup); err != nil {
		return err
	}

	return nil
}
