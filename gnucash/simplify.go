package gnucash

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type FlatTransaction struct {
	From        *Account
	To          *Account
	Description string
	Value       Value
}

func (f *FlatTransaction) String() string {
	return fmt.Sprintf("%5.2f %s => %s", f.Value, f.From.FQN, f.To.FQN)
}

func (f *FlatTransaction) Inverse() *FlatTransaction {
	return &FlatTransaction{
		f.To,
		f.From,
		f.Description,
		-f.Value,
	}
}

func (f *FlatTransaction) InverseValue() *FlatTransaction {
	return &FlatTransaction{
		f.From,
		f.To,
		f.Description,
		-f.Value,
	}
}

type FlatTransactions []*FlatTransaction

func (f FlatTransactions) Inverse() FlatTransactions {
	n := make(FlatTransactions, len(f))
	for i := range f {
		n[i] = f[i].Inverse()
	}
	return n
}

func (f FlatTransactions) InverseValue() FlatTransactions {
	n := make(FlatTransactions, len(f))
	for i := range f {
		n[i] = f[i].InverseValue()
	}
	return n
}

func (f FlatTransactions) SortNormal(fromFirst bool) FlatTransactions {
	sort.SliceStable(
		f,
		func(i, j int) bool {
			a, b := f[i], f[j]
			if a.Value < 0 && b.Value >= 0 {
				return true
			} else if a.Value >= 0 && b.Value < 0 {
				return false
			}

			fromA, fromB, toA, toB := a.From.FQN, b.From.FQN, a.To.FQN, b.To.FQN
			if !fromFirst {
				fromA, fromB, toA, toB = toA, toB, fromA, fromB
			}

			return fromA < fromB || (fromA == fromB && toA < toB)
		},
	)

	return f
}

func (f FlatTransactions) SortValue() FlatTransactions {
	sort.SliceStable(
		f,
		func(i, j int) bool {
			return f[i].Value < f[j].Value
		},
	)

	return f
}

func (f FlatTransactions) String() string {
	str := make([]string, len(f))
	for i, t := range f {
		str[i] = t.String()
	}

	return strings.Join(str, "\n")
}

func (f FlatTransactions) Sum() Value {
	var v Value
	for _, t := range f {
		v += t.Value
	}
	return v
}

func (f FlatTransactions) Filter(
	fromFQN,
	fromExcludeFQN,
	toFQN,
	toExcludeFQN *regexp.Regexp,
) FlatTransactions {
	flattxs := make(FlatTransactions, 0)
	for _, t := range f {
		if matchFQN(fromFQN, fromExcludeFQN, t.From.FQN) &&
			matchFQN(toFQN, toExcludeFQN, t.To.FQN) {
			flattxs = append(flattxs, t)
		}
	}

	return flattxs
}

func (f FlatTransactions) Relative(
	fromFQN,
	toFQN *regexp.Regexp,
) FlatTransactions {
	flattxs := make(FlatTransactions, 0, len(f))
	for _, t := range f {
		if matchFQN(fromFQN, nil, t.From.FQN) &&
			matchFQN(toFQN, nil, t.To.FQN) {
			flattxs = append(flattxs, t)
			continue
		}

		if matchFQN(fromFQN, nil, t.To.FQN) &&
			matchFQN(fromFQN, nil, t.From.FQN) {
			continue
		}

		if matchFQN(toFQN, nil, t.To.FQN) &&
			matchFQN(toFQN, nil, t.From.FQN) {
			continue
		}

		if matchFQN(fromFQN, nil, t.To.FQN) ||
			matchFQN(toFQN, nil, t.From.FQN) {
			flattxs = append(flattxs, t.Inverse())
		}
	}

	return flattxs
}

func (f FlatTransactions) RelativeFrom(fromFQN *regexp.Regexp) FlatTransactions {
	flattxs := make(FlatTransactions, 0, len(f))
	for _, t := range f {
		if matchFQN(fromFQN, nil, t.From.FQN) {
			flattxs = append(flattxs, t)
		} else if matchFQN(fromFQN, nil, t.To.FQN) {
			flattxs = append(flattxs, t.Inverse())
		}
	}

	return flattxs
}

func (f FlatTransactions) RelativeTo(toFQN *regexp.Regexp) FlatTransactions {
	flattxs := make(FlatTransactions, 0, len(f))
	for _, t := range f {
		if matchFQN(toFQN, nil, t.To.FQN) {
			flattxs = append(flattxs, t)
		} else if matchFQN(toFQN, nil, t.From.FQN) {
			flattxs = append(flattxs, t.Inverse())
		}
	}

	return flattxs
}

func (f FlatTransactions) GroupBy(fromFQN *regexp.Regexp) FlatTransactions {
	m := make(map[string]int)
	amount := 0
	for i, t := range f {
		if j, ok := m[t.From.FQN]; ok {
			f[j].To = nil
			f[j].Value += t.Value
			t.Value = 0
			amount++
			continue
		}

		m[t.From.FQN] = i
	}

	n := make(FlatTransactions, 0, len(f)-amount)
	for _, t := range f {
		if t.Value != 0 {
			n = append(n, t)
		}
	}

	return n
}

type txMeta struct {
	from        *Account
	to          *Account
	value       Value
	description string
}
type simpleTXMap map[GUID]map[GUID]*txMeta

func (m simpleTXMap) flatten() simpleTXMap {
	for fid := range m {
		for tid := range m[fid] {
			if _, ok := m[tid][fid]; ok {
				fromI, toI := fid, tid
				if m[tid][fid].value < m[fid][tid].value {
					fromI, toI = toI, fromI
				}
				m[fromI][toI].value -= m[toI][fromI].value
				delete(m[toI], fromI)
			}
		}
	}

	return m
}

func (m simpleTXMap) flattxs() FlatTransactions {
	flattxs := make(FlatTransactions, 0, len(m))
	for fid := range m {
		for tid := range m[fid] {
			f, t := m[fid][tid].from, m[fid][tid].to
			var sign Value = -1
			if m[fid][tid].value > 0 {
				f, t = t, f
				sign = 1
			}
			tx := &FlatTransaction{
				From:        f,
				To:          t,
				Description: m[fid][tid].description,
				Value:       sign * m[fid][tid].value,
			}

			flattxs = append(flattxs, tx)
		}
	}

	return flattxs
}

func (ts Transactions) mapTransactions() simpleTXMap {
	m := make(simpleTXMap, len(ts))
	from := make([]*Split, 1)
	to := make([]*Split, 1)
	for _, tx := range ts {
		var diff Value = 0
		from = from[0:0]
		to = to[0:0]
		for _, s := range tx.Splits {
			if s.Value >= 0 {
				diff += s.Value
				to = append(to, s)
				continue
			}
			from = append(from, s)
		}

		for _, f := range from {
			if _, ok := m[f.AccountID]; !ok {
				m[f.AccountID] = make(map[GUID]*txMeta, len(to))
			}

			for _, t := range to {
				if m[f.AccountID][t.AccountID] == nil {
					m[f.AccountID][t.AccountID] = &txMeta{
						f.Account,
						t.Account,
						0,
						tx.Description,
					}
				}
				m[f.AccountID][t.AccountID].value += t.Value * f.Value / diff
			}
		}
	}

	return m
}
