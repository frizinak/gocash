package gnucash

import "fmt"

type Slot struct {
	Key      string    `xml:"key"`
	RawValue SlotValue `xml:"value"`
}

func mkValueError(key, exp, act string) error {
	if exp != act {
		return fmt.Errorf("%s is a %s not a %s", key, act, exp)
	}
	return nil
}

func (s Slot) StringValue() (string, error) {
	if err := mkValueError(s.Key, "string", s.RawValue.Type); err != nil {
		return "", err
	}
	return s.RawValue.Value, nil
}

func (s Slot) String() string {
	return fmt.Sprintf("%s: %s", s.Key, s.RawValue.String())
}

type SlotValue struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

func (s SlotValue) String() string {
	return fmt.Sprintf("[%s] %s", s.Type, s.Value)
}
