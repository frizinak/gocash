package gnucash

import (
	"errors"
	"fmt"
)

type Account struct {
	ID           GUID         `xml:"id"`
	FQN          string       `xml:"-"`
	Type         AccountType  `xml:"type"`
	Name         string       `xml:"name"`
	Description  string       `xml:"description"`
	ParentID     GUID         `xml:"parent"`
	Parent       *Account     `xml:"-"`
	Children     Accounts     `xml:"-"`
	Transactions Transactions `xml:"-"`
	Commodity    CommodityRef `xml:"commodity"`
}

func (a *Account) String() string {
	return fmt.Sprintf(
		"%s\nID: %s\nType: %s\nCurrency: %s\nDescription: %s\nChildren:\n%s\nTransactions:\n%s",
		a.FQN,
		a.ID,
		a.Type,
		a.Commodity.ID,
		a.Description,
		a.Children,
		a.Transactions,
	)
}

func (a *Account) fqn() string {
	if a.FQN != "" {
		return a.FQN
	}

	fqn := []string{a.Name}
	parent := a.Parent
	for {
		if parent == nil || parent.Type == AccountTypeRoot {
			break
		}

		fqn = append(fqn, parent.Name)
		parent = parent.Parent
	}

	for i := len(fqn) - 1; i >= 0; i-- {
		a.FQN += fqn[i]
		if i != 0 {
			a.FQN += "."
		}
	}

	return a.FQN
}

func (a *Account) validate(lookup *AccountsLookup, txLookup TransactionsLookup) error {
	if a.ID == "" {
		return errors.New("Empty account id")
	}

	if a.Type != AccountTypeRoot &&
		a.Type != AccountTypeAsset &&
		a.Type != AccountTypeCash &&
		a.Type != AccountTypeBank &&
		a.Type != AccountTypeStock &&
		a.Type != AccountTypeMutual &&
		a.Type != AccountTypeCredit &&
		a.Type != AccountTypeEquity &&
		a.Type != AccountTypeExpense &&
		a.Type != AccountTypeIncome &&
		a.Type != AccountTypePayable &&
		a.Type != AccountTypeReceivable &&
		a.Type != AccountTypeLiability {
		return fmt.Errorf("Invalid account type '%s'", a.Type)
	}

	a.Transactions, _ = txLookup.Find(a.ID)

	return nil
}
