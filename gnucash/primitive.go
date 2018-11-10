package gnucash

import (
	nxml "encoding/xml"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	AccountTypeRoot       AccountType = "ROOT"
	AccountTypeAsset                  = "ASSET"
	AccountTypeCash                   = "CASH"
	AccountTypeBank                   = "BANK"
	AccountTypeCredit                 = "CREDIT"
	AccountTypeEquity                 = "EQUITY"
	AccountTypeExpense                = "EXPENSE"
	AccountTypeIncome                 = "INCOME"
	AccountTypeLiability              = "LIABILITY"
	AccountTypePayable                = "PAYABLE"
	AccountTypeReceivable             = "RECEIVABLE"
)

const (
	ReconciledStateNew        ReconciledState = 'n'
	ReconciledStateCleared                    = 'c'
	ReconciledStateReconciled                 = 'y'
	ReconciledStateFrozen                     = 'f'
	ReconciledStateVoid                       = 'v'
)

type AccountType string
type ReconciledState rune
type GUID string
type Enabled bool
type Value float64
type Date struct {
	parsed bool
	d      time.Time
}

func (r *ReconciledState) UnmarshalXML(d *nxml.Decoder, start nxml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}

	*r = ReconciledState(content[0])
	return nil
}

func (r ReconciledState) String() string {
	return string(r)
}

func (r ReconciledState) Reconciled() bool {
	return r == ReconciledStateReconciled
}

func (r ReconciledState) Cleared() bool {
	return r == ReconciledStateReconciled || r == ReconciledStateCleared
}

func (e *Enabled) UnmarshalXML(d *nxml.Decoder, start nxml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}

	*e = content == "y"
	return nil
}

func (dt *Date) UnmarshalXML(d *nxml.Decoder, start nxml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}

	if content == "" {
		return nil
	}

	formats := []string{
		"2006-01-02 15:04:05 -0700",
		"2006-01-02",
	}

	var err error
	var parsed time.Time
	for _, format := range formats {
		parsed, err = time.Parse(format, content)
		if err == nil {
			dt.parsed = true
			dt.d = parsed
			return nil
		}
	}

	return err
}

func (dt *Date) Empty() bool {
	return !dt.parsed
}

func (dt *Date) Get() time.Time {
	return dt.d
}

func (v *Value) UnmarshalXML(d *nxml.Decoder, start nxml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}

	p := strings.SplitN(string(content), "/", 2)
	if len(p) != 2 {
		return errors.New("Unexpected Value: " + content)
	}

	val, err := strconv.Atoi(p[0])
	if err != nil {
		return err
	}

	div, err := strconv.Atoi(p[1])
	if err != nil {
		return err
	}

	if div > 0 {
		*v = Value(float64(val) / float64(div))
	}

	return nil
}
