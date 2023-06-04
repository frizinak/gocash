package gnucash

import "strings"

type Commodities []Commodity

func (c Commodities) Len() int           { return len(c) }
func (c Commodities) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c Commodities) Less(i, j int) bool { return c[i].FQN() < c[j].FQN() }

func (c Commodities) Lookup() CommoditiesLookup {
	m := make(CommoditiesLookup, len(c))
	for _, v := range c {
		m[v.FQN()] = v
	}

	return m
}

func (c Commodities) String() string {
	s := make([]string, len(c))
	for i := range c {
		s[i] = c[i].String()
	}

	return strings.Join(s, "\n")
}

type CommoditiesLookup map[CommodityFQN]Commodity
