package gnucash

import "fmt"

type Price struct {
	ID       GUID         `xml:"id"`
	Comodity CommodityRef `xml:"commodity"`
	Currency CommodityRef `xml:"currency"`
	Time     Date         `xml:"time>date"`
	Type     string       `xml:"type"`
	Value    Value        `xml:"value"`
}

func (p Price) String() string {
	return fmt.Sprintf(
		"[%s] %s > %s: %.2f",
		p.Time.Get().Format("2006-01-02 15:04"),
		p.Comodity.FQN(),
		p.Currency.FQN(),
		p.Value,
	)
}
