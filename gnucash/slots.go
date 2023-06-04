package gnucash

import "strings"

type Slots []Slot

func (s Slots) KeyValue() KeyValue {
	m := make(KeyValue, len(s))
	for _, i := range s {
		m[i.Key] = i
	}
	return m
}

func (s Slots) String() string {
	t := make([]string, len(s))
	for i := range t {
		t[i] = s[i].String()
	}
	return strings.Join(t, "\n")
}

type KeyValue map[string]Slot
