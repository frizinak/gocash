package gnucash

import (
	"errors"
	"fmt"
)

type Scheduled struct {
	ID      GUID    `xml:"id"`
	Name    string  `xml:"name"`
	Enabled Enabled `xml:"enabled"`
}

func (s *Scheduled) String() string {
	enab := "y"
	if !s.Enabled {
		enab = "n"
	}
	return fmt.Sprintf(
		"SCHEDULED\n[%s] %s\nID: %s",
		enab,
		s.Name,
		s.ID,
	)
}

func (s *Scheduled) validate(lookup *AccountsLookup) error {
	if s.ID == "" {
		return errors.New("Empty schedule id")
	}

	return nil
}
