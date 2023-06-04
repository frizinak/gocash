package gnucash

import "strings"

type Prices []Price

func (p Prices) String() string {
	l := make([]string, len(p))
	for i := range p {
		l[i] = p[i].String()
	}
	return strings.Join(l, "\n")
}

func (ps Prices) LastFor(com CommodityFQN) Price {
	var b Price
	for _, p := range ps {
		comFQN := p.Comodity.FQN()
		if comFQN != com {
			continue
		}
		if b.ID == "" || p.Time.Get().After(b.Time.Get()) {
			b = p
		}
	}

	return b
}
