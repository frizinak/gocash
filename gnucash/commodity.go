package gnucash

import (
	"fmt"
)

type CommodityID string
type CommodityNS string
type CommodityFQN string

const (
	CommodityCurrency CommodityNS = "CURRENCY"
)

type CommodityRef struct {
	ID CommodityID `xml:"id"`
	NS CommodityNS `xml:"space"`
}

type Commodity struct {
	CommodityRef
	Fraction int   `xml:"fraction"`
	Slots    Slots `xml:"slots>slot"`
}

func (c CommodityRef) FQN() CommodityFQN {
	return CommodityFQN(fmt.Sprintf("%s.%s", c.NS, c.ID))
}

func (c CommodityRef) IsCurrency() bool { return c.NS == CommodityCurrency }

func (c Commodity) String() string {
	kv := c.Slots.KeyValue()
	sym, err := kv["user_symbol"].StringValue()
	if err != nil || sym == "" {
		return fmt.Sprintf("%s", c.FQN())
	}
	return fmt.Sprintf("%s [%s]", c.FQN(), sym)
}
