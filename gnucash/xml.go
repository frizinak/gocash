package gnucash

import "io"

import (
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
