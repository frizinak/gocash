package gnucash

import (
	"io"

	nxml "encoding/xml"
)

type XML struct {
	Books Books `xml:"book"`
}

func (x *XML) String() string {
	return x.Books.String()
}

func (x *XML) validate() error {
	if err := x.Books.validate(); err != nil {
		return err
	}

	return nil
}

func Read(r io.Reader) (*XML, error) {
	dec := nxml.NewDecoder(r)
	xml := &XML{}
	if err := dec.Decode(xml); err != nil {
		return nil, err
	}

	return xml, xml.validate()
}

type AccountsXML struct {
	Accounts Accounts `xml:"account"`
}

func (a *AccountsXML) validate() error {
	return a.Accounts.validate(a.Accounts.lookup(), make(Transactions, 0).lookup())
}

func ReadAccounts(r io.Reader) (*AccountsXML, error) {
	dec := nxml.NewDecoder(r)
	xml := &AccountsXML{}
	if err := dec.Decode(xml); err != nil {
		return nil, err
	}

	return xml, xml.validate()
}
