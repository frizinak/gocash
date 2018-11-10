package gnucash

import "strings"

type Books []*Book

func (bs Books) String() string {
	str := make([]string, len(bs))
	for i, b := range bs {
		str[i] = b.String()
	}

	return strings.Join(str, "\n")
}

func (bs Books) validate() error {
	for _, b := range bs {
		if err := b.validate(); err != nil {
			return err
		}
	}

	return nil
}
