package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/frizinak/gocash/flags"
	"github.com/frizinak/gocash/fuzzy"
	"github.com/frizinak/gocash/gnucash"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type transaction struct {
	state  string
	uid    string
	date   string
	from   string
	to     string
	amount string
	descr  string
}

func (tx *transaction) GenID() error {
	w := sha256.New()
	fmt.Fprintf(
		w,
		"4tezIbTWgCAlaBWFACBNYu55X2rVxrHYRq3ziDIeHCeaX2DjG/oj/88jbTPy5gq+zIk=%s:%s",
		tx.date,
		tx.descr,
	)

	s := w.Sum(nil)
	h := hex.EncodeToString(s)
	if tx.uid != "" && h != tx.uid {
		return fmt.Errorf("transaction has changed since uid was generated")
	}
	tx.uid = h
	return nil
}

func (tx *transaction) Fields(n int) [][]string {
	rows := make([][]string, 2)

	num := fmt.Sprintf("hash%d-%s", n, tx.uid)
	rows[0], rows[1] = make([]string, 6), make([]string, 6)
	rows[0][0] = num
	rows[0][1] = tx.date
	rows[0][2] = tx.to
	rows[0][3] = tx.amount
	rows[0][4] = "1"
	rows[0][5] = tx.descr

	rows[1][0] = num
	rows[1][1] = ""
	rows[1][2] = tx.from
	rows[1][3] = "-" + tx.amount
	rows[1][4] = "1"
	rows[1][5] = ""

	return rows
}

func num2uid(num string) (string, error) {
	v := strings.SplitN(num, "-", 2)
	if len(v) != 2 {
		return v[0], fmt.Errorf("NUM '%s' can't be converted to a UID", num)
	}
	return v[1], nil

}

func start(msg string) func() {
	msg = msg + "…"
	fmt.Fprintf(os.Stderr, "%-60s", msg)
	s := time.Now()
	return func() {
		fmt.Fprintf(os.Stderr, "done [%s]\n", time.Since(s).Round(time.Millisecond))
	}
}

const dFormat = "2006-01-02"

type Conf struct {
	o  []ConfKey
	kv map[ConfKey]string
}

func (c Conf) Get(k ConfKey) string { return c.kv[k] }

type ConfKey string

const (
	KDataFile                      = "datafile"
	KSheetID                       = "google-sheet-id"
	KServiceAccountCredentialsFile = "service-account-credentials"
	KAlias                         = "account.alias."
	KIgnore                        = "account.ignore"
	KReport                        = "report.profit.account"
	KReportIgnore                  = "report.profit.ignore"
)

var eg = map[ConfKey]string{
	KDataFile:                      fmt.Sprintf("e.g.: '%s=/home/user/Documents/db.gnucash'", KDataFile),
	KSheetID:                       fmt.Sprintf("e.g.: '%s=iUilufHz6OrPHnWEEXFkhbxkuf6WAlaPh8sQvC8ejUO7'", KSheetID),
	KServiceAccountCredentialsFile: fmt.Sprintf("e.g.: '%s=/home/user/Private/service-account-credentials.json'", KServiceAccountCredentialsFile),
}

var _c Conf

func readconf(conf string, req []ConfKey) (Conf, error) {
	if _c.kv == nil {
		c := Conf{make([]ConfKey, 0), make(map[ConfKey]string)}
		if conf == "" {
			uconfdir, err := os.UserConfigDir()
			if err != nil {
				return c, err
			}

			conf = filepath.Join(uconfdir, "gocash", "config")
		}
		f, err := os.Open(conf)
		if err != nil {
			return c, fmt.Errorf("could not read config file '%s': %w", conf, err)
		}
		s := bufio.NewScanner(f)
		s.Split(bufio.ScanLines)
		pushes := make(map[string]int, 0)
		for s.Scan() {
			t := strings.SplitN(s.Text(), "=", 2)
			if len(t) != 2 {
				t = append(t, "")
			}
			k, v := strings.TrimSpace(t[0]), strings.TrimSpace(t[1])

			if strings.HasSuffix(k, "[]") {
				k = k[:len(k)-2]
				pushes[k]++
				k = fmt.Sprintf("%s.%d", k, pushes[k])
			}
			key := ConfKey(k)

			if _, ok := c.kv[key]; !ok {
				c.o = append(c.o, key)
			}
			c.kv[key] = v
		}

		if err := s.Err(); err != nil {
			return c, err
		}

		_c = c
	}

	for _, k := range req {
		if _c.kv[k] == "" {
			if v := eg[k]; v != "" {
				return _c, fmt.Errorf("missing '%s' entry. %s", k, v)
			}

			return _c, fmt.Errorf("missing '%s' entry.", k)
		}
	}

	return _c, nil
}

func confPrefix(conf string, prefix string) ([]string, map[string]string, error) {
	c, err := readconf(conf, nil)
	if err != nil {
		return nil, nil, err
	}

	kv := make(map[string]string)
	pl := len(prefix)
	o := make([]string, 0, len(c.o))
	for _, k := range c.o {
		if strings.HasPrefix(string(k), prefix) {
			v := c.kv[k]
			alias := string(k)[pl:]
			if _, ok := kv[alias]; !ok {
				o = append(o, alias)
			}
			kv[alias] = v
		}
	}

	return o, kv, nil
}

func confPrefixArray(conf string, prefix string) ([]string, error) {
	o, m, err := confPrefix(conf, prefix)
	l := make([]string, 0, len(o))
	for _, k := range o {
		l = append(l, m[k])
	}
	return l, err
}

func accountsWithAliases(accounts gnucash.Accounts, conf string) (
	order []string,
	list map[string]string,
	placeholder map[string]struct{},
	err error,
) {

	o, as, e := confPrefix(conf, KAlias)
	if e != nil {
		err = e
		return
	}

	placeholder = make(map[string]struct{}, len(accounts))
	order = make([]string, 0, len(accounts))
	for _, alias := range o {
		order = append(order, alias)
	}

	list = make(map[string]string)
	for _, a := range accounts {
		order = append(order, a.FQN)
		list[a.FQN] = a.FQN
		if a.Placeholder() {
			placeholder[a.FQN] = struct{}{}
		}
	}

	for alias, fqn := range as {
		if _, ok := list[fqn]; !ok {
			err = fmt.Errorf("can't create alias '%s' for non-existent account '%s'", alias, fqn)
			return
		}
		if _, ok := list[alias]; ok {
			err = fmt.Errorf("duplicate alias '%s' for account '%s'", alias, fqn)
			return
		}
		list[alias] = fqn
	}

	return
}

func readbook(conf string) (*gnucash.Book, error) {
	c, err := readconf(conf, []ConfKey{KDataFile})
	if err != nil {
		return nil, err
	}

	f, err := os.Open(c.Get(KDataFile))
	if err != nil {
		return nil, fmt.Errorf("could not read datafile '%s': %w", c.Get(KDataFile), err)
	}

	data, err := gnucash.Read(f)
	if err != nil {
		return nil, err
	}
	if len(data.Books) == 0 {
		// not a gnucash datafile xml
		return nil, fmt.Errorf("no book found in '%s'", c.Get(KDataFile))
	}

	return data.Books[0], nil
}

func accounts(conf string) (gnucash.Accounts, error) {
	c, err := readconf(conf, []ConfKey{KDataFile})
	if err != nil {
		return nil, err
	}
	f, err := os.Open(c.Get(KDataFile))
	if err != nil {
		return nil, fmt.Errorf("could not read datafile '%s': %w", c.Get(KDataFile), err)
	}

	data, err := gnucash.ReadAccounts(f)
	if err != nil {
		return nil, err
	}

	return data.Accounts, nil
}

func accountsFromAny(conf string) (gnucash.Accounts, error) {
	if book, err := readbook(conf); err == nil {
		return book.Accounts, nil
	}
	return accounts(conf)
}

func accountFuzzy(accounts gnucash.Accounts) ([]string, *fuzzy.Index) {
	accountNames := make([]string, len(accounts))
	for i, a := range accounts {
		accountNames[i] = a.FQN
	}

	fuzz := fuzzy.NewIndex(2, accountNames)

	return accountNames, fuzz
}

func sheetPad(vals [][]interface{}, innerSize int) [][]interface{} {
	n := len(vals)
	if n < cap(vals) {
		vals = vals[:cap(vals)]
	}
	for i := range vals {
		if len(vals[i]) < innerSize {
			x := make([]interface{}, innerSize-len(vals[i]))
			for j := range x {
				x[j] = ""
			}
			vals[i] = append(vals[i], x...)
		}
	}
	for i := n; i < len(vals); i++ {
		vals[i] = make([]interface{}, innerSize)
		for j := range vals[i] {
			vals[i][j] = ""
		}
	}

	return vals
}

func main() {
	var conf string
	fr := flags.NewRoot(os.Stdout)
	fr.Define(func(set *flag.FlagSet) flags.HelpCB {
		set.StringVar(&conf, "c", "", "configfile")
		return func(h *flags.Help) {
			h.Add("Commands:")
			h.Add("  - account: fuzzy find an account fqn")
			h.Add("  - config:  print an example config on stdout")
			h.Add("  - tx:      interactively create an importable transaction")
			h.Add("  - sheet:   parse a google sheet and export as csv")
			h.Add("             (will alter your google sheet!)")
		}
	}).Handler(func(set *flags.Set, args []string) error {
		set.Usage(1)
		return nil
	})

	fr.Add("account").Define(func(set *flag.FlagSet) flags.HelpCB {
		return func(h *flags.Help) {
			h.Add("fuzzy find an account")
		}
	}).Handler(func(set *flags.Set, args []string) error {
		if len(args) == 0 {
			return errors.New("please provide a query")
		}

		accounts, err := accountsFromAny(conf)
		if err != nil {
			return err
		}

		accountNames, fuzz := accountFuzzy(accounts)
		res := make([]string, 0)
		query := strings.Join(args, " ")
		fuzz.Search(query, func(i int, score, low, high uint8) {
			if score == high {
				res = append(res, accountNames[i])
			}
		})

		fmt.Println(strings.Join(res, "\n"))

		return nil
	})

	fr.Add("tx").Define(func(set *flag.FlagSet) flags.HelpCB {
		return func(h *flags.Help) {
			h.Add("interactively create an importable transaction")
		}
	}).Handler(func(set *flags.Set, args []string) error {
		accounts, err := accountsFromAny(conf)
		if err != nil {
			return err
		}

		accountNames, fuzz := accountFuzzy(accounts)

		s := bufio.NewScanner(os.Stdin)
		s.Split(bufio.ScanLines)

		ask := func(lbl string, validate func(str string) (string, error)) (string, error) {
			for {
				fmt.Print(lbl)
				fmt.Print(": ")
				if !s.Scan() {
					return "", s.Err()
				}
				t := strings.TrimSpace(s.Text())
				if nv, err := validate(t); err == nil {
					return nv, nil
				}
			}
		}

		float := func(str string) (string, error) {
			_, err := strconv.ParseFloat(str, 32)
			return str, err
		}

		account := func(str string) (string, error) {
			var match string
			var err error
			fuzz.Search(str, func(i int, score, min, max uint8) {
				if score == max && match == "" {
					match = accountNames[i]
				}
			})
			return match, err
		}

		noop := func(str string) (string, error) { return str, nil }

		// todo validation / completion
		tx := &transaction{}
		tx.date = time.Now().Format(dFormat)
		tx.amount, err = ask("Amount", float)
		if err != nil {
			return err
		}

		tx.descr, err = ask("Description", noop)
		if err != nil {
			return err
		}

		tx.from, err = ask("From", account)
		if err != nil {
			return err
		}

		tx.to, err = ask("To", account)
		if err != nil {
			return err
		}

		if err := tx.GenID(); err != nil {
			return err
		}

		w := csv.NewWriter(os.Stdout)
		for _, row := range tx.Fields(0) {
			if err := w.Write(row); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	})

	fr.Add("config").Define(func(set *flag.FlagSet) flags.HelpCB {
		return func(h *flags.Help) {
			h.Add("print an example config on stdout")
		}
	}).Handler(func(set *flags.Set, args []string) error {
		fmt.Printf("%s                    = /home/user/Documents/db.gnucash\n", KDataFile)
		fmt.Printf("%s = /home/user/Private/service-account-credentials.json\n", KServiceAccountCredentialsFile)
		fmt.Printf("%s             = iUilufHz6OrPHnWEEXFkhbxkuf6WAlaPh8sQvC8ejUO7\n", KSheetID)
		fmt.Println()
		fmt.Printf("%sfood      = expenses.groceries.food\n", KAlias)
		fmt.Printf("%ssnacks    = expenses.groceries.snacks\n", KAlias)
		fmt.Printf("%shousehold = expenses.groceries.household\n", KAlias)
		fmt.Printf("%sdining    = expenses.dining\n", KAlias)
		fmt.Println()
		fmt.Printf("%sme.bank      = assets.current.some bank.me.checking\n", KAlias)
		fmt.Printf("%spartner.bank = assets.current.some bank.partner.checking\n", KAlias)
		fmt.Println()
		fmt.Printf("%sme.wallet       = assets.current.wallets.me\n", KAlias)
		fmt.Printf("%spartner.wallet  = assets.current.wallets.partner\n", KAlias)
		fmt.Println()
		fmt.Printf("%s[] = Root Account\n", KIgnore)
		fmt.Printf("%s[] = Opening Balances\n", KIgnore)
		fmt.Printf("%s[] = Orphan-EUR\n", KIgnore)
		fmt.Printf("%s[] = Imbalance-EUR\n", KIgnore)
		fmt.Println()
		fmt.Printf("%s[] = ^assets\\.current\\.wallets\\.me$\n", KReport)
		fmt.Printf("%s[] = ^assets\\.current\\..*bank\n", KReport)
		fmt.Printf("%s[]  = ^equity\\.opening balances$\n", KReportIgnore)
		return nil
	})

	fr.Add("sheet").Define(func(set *flag.FlagSet) flags.HelpCB {
		return func(h *flags.Help) {
			h.Add("parse a google sheet and export as csv.")
			h.Add("you will need to create a google project and link a service account")
			h.Add("(will alter your google sheet!)")
		}
	}).Handler(func(set *flags.Set, args []string) error {
		var book *gnucash.Book
		var sid string
		var srv *sheets.Service
		var aliases map[string]string
		var aliasesOrder []string
		var placeholder map[string]struct{}
		var accountsLookup *gnucash.AccountsLookup

		err := func() error {
			end := start("Parsing config and books")
			defer end()
			c, err := readconf(
				conf,
				[]ConfKey{
					KDataFile,
					KSheetID,
					KServiceAccountCredentialsFile,
				},
			)
			if err != nil {
				return err
			}

			bookf := c.Get(KDataFile)
			sid = c.Get(KSheetID)
			credsf := c.Get(KServiceAccountCredentialsFile)

			book, err = readbook(bookf)
			if err != nil {
				return err
			}
			_accounts := book.Accounts
			accounts := make(gnucash.Accounts, 0, len(_accounts))
			_ignore, err := confPrefixArray(conf, KIgnore)
			if err != nil {
				return err
			}

			ignore := make(map[string]struct{}, len(_ignore))
			for _, v := range _ignore {
				ignore[v] = struct{}{}
			}

			for _, a := range _accounts {
				if _, ok := ignore[a.FQN]; ok {
					continue
				}
				accounts = append(accounts, a)
			}

			aliasesOrder, aliases, placeholder, err = accountsWithAliases(accounts, conf)
			if err != nil {
				return err
			}

			accountsLookup = book.AccountsLookup

			ctx := context.Background()
			srv, err = sheets.NewService(
				ctx,
				option.WithCredentialsFile(credsf),
			)
			if err != nil {
				return fmt.Errorf("unable to create sheets service: %w", err)
			}
			return nil
		}()
		if err != nil {
			return err
		}

		err = func() error {
			end := start("Updating accounts sheets")
			defer end()
			vals := make([][]interface{}, 0, len(aliasesOrder)*5)
			for _, v := range aliasesOrder {
				item := make([]interface{}, 4)
				item[0] = ""
				item[1] = v
				item[2] = ""
				item[3] = ""
				fqn := aliases[v]
				_, ph := placeholder[fqn]
				if ph {
					item[0] = v
					item[1] = ""
				}

				acc, ok := book.AccountsLookup.ByFQN(fqn)
				if ok {
					accval := book.Transactions.ValueForAccount(acc.ID, false)
					accvalgross := book.Transactions.ValueForAccount(acc.ID, true)
					item[2] = accvalgross
					item[3] = accval
					if ph || accvalgross == accval {
						item[3] = ""
					}
				}

				vals = append(vals, item)
			}

			vals = sheetPad(vals, 3)

			values := &sheets.ValueRange{
				MajorDimension: "ROWS",
				Values:         vals,
			}
			ur := srv.Spreadsheets.Values.Update(sid, "Accounts!A1", values)
			ur.ValueInputOption("RAW")
			_, err := ur.Do()
			return err
		}()
		if err != nil {
			return err
		}

		txs := make([]*transaction, 0)
		txsByUIDs := make(map[string][]*transaction)

		errbuf := bytes.NewBuffer(nil)
		resultsBuf := bytes.NewBuffer(nil)
		err = func() error {
			end := start("Fetching transactions")
			defer end()
			readRange := "Tx!A2:G"
			resp, err := srv.Spreadsheets.Values.Get(sid, readRange).Do()
			if err != nil {
				return err
			}

			val := func(vals []interface{}, ix int, req bool, def interface{}) (interface{}, error) {
				if ix > len(vals)-1 {
					if !req {
						return def, nil
					}
					return nil, fmt.Errorf("no such column: %d", ix)
				}
				return vals[ix], nil
			}

			strval := func(vals []interface{}, ix int, req bool) (string, error) {
				val, err := val(vals, ix, req, "")
				if err != nil {
					return "", err
				}
				if v, ok := val.(string); ok {
					return v, nil
				}
				return "", fmt.Errorf("field %d is not a string: %T", ix, val)
			}

			formats := []string{
				"2006-01-02",
				"02-01-2006",
			}

			dateRepl := regexp.MustCompile(`[\-/ \.:]+`)

			bad, all, old := 0, 0, 0
			for y, row := range resp.Values {
				tx := &transaction{}
				err = func() error {
					var err error
					tx.state, err = strval(row, 0, true)
					if err != nil {
						return err
					}

					tx.state = strings.ToLower(strings.TrimSpace(tx.state))

					tx.uid, err = strval(row, 1, true)
					if err != nil {
						return err
					}

					var dt time.Time
					tx.date, err = strval(row, 2, true)
					if err != nil {
						return err
					}
					tx.date = dateRepl.ReplaceAllString(tx.date, "-")
					for _, f := range formats {
						t, err := time.Parse(f, tx.date)
						if err == nil {
							dt = t
							break
						}
					}

					if dt == (time.Time{}) {
						return fmt.Errorf("failed to parse date: %s", tx.date)
					}
					tx.date = dt.Format(dFormat)

					tx.from, err = strval(row, 3, true)
					if err != nil {
						return err
					}
					rfrom := aliases[tx.from]
					if _, ok := accountsLookup.ByFQN(rfrom); !ok {
						return fmt.Errorf("no such account: '%s'", tx.from)
					}
					tx.from = rfrom

					tx.to, err = strval(row, 4, true)
					if err != nil {
						return err
					}
					rto := aliases[tx.to]
					if _, ok := accountsLookup.ByFQN(rto); !ok {
						return fmt.Errorf("no such account: '%s'", tx.to)
					}
					tx.to = rto

					tx.amount, err = strval(row, 5, true)
					if err != nil {
						return err
					}
					_, err = strconv.ParseFloat(tx.amount, 32)
					if err != nil {
						return err
					}

					tx.descr, err = strval(row, 6, true)
					if err != nil {
						return err
					}

					err = tx.GenID()
					if err != nil {
						return err
					}

					return nil
				}()

				if tx.state == "e" {
					tx.state = ""
				}
				if tx.uid != "" {
					tx.state = "c"
				}

				if err != nil {
					fmt.Fprintf(errbuf, "\033[1;31mrow %d: %s\033[0m\n", y+2, err)
					tx.state = "e"
				}

				all++
				switch tx.state {
				case "x":
					old++
				case "e":
					bad++
				}

				if tx.uid != "" {
					if txsByUIDs[tx.uid] == nil {
						txsByUIDs[tx.uid] = make([]*transaction, 0, 1)
					}
					txsByUIDs[tx.uid] = append(txsByUIDs[tx.uid], tx)
				}
				txs = append(txs, tx)
			}

			fmt.Fprintf(resultsBuf, "  %d rows\n", all)
			fmt.Fprintf(resultsBuf, "  %d old\n", old)
			fmt.Fprintf(resultsBuf, "  %d new\n", all-old-bad)
			fmt.Fprintf(resultsBuf, "  %d bad\n", bad)

			return nil
		}()

		resultsBuf.WriteTo(os.Stderr)
		fmt.Fprint(os.Stderr, errbuf.String())
		if err != nil {
			return err
		}

		err = func() error {
			found := 0
			end := start("Search for existing transactions in book")
			defer end()
			for _, tx := range txs {
				if tx.state == "x" {
					tx.state = ""
				}
			}

			for _, tx := range book.Transactions {
				if tx.Num == "" {
					continue
				}
				uid, _ := num2uid(tx.Num)
				if uid == "" {
					continue
				}

				txs := txsByUIDs[uid]
				for _, t := range txs {
					t.state = "x"
					found++
				}
			}

			fmt.Fprintf(resultsBuf, "  matched %d transactions\n", found)
			return nil
		}()
		resultsBuf.WriteTo(os.Stderr)
		if err != nil {
			return err
		}

		err = func() error {
			end := start("Updating transactions' state fields")
			defer end()
			upd := make([][]interface{}, 3)
			upd[0] = make([]interface{}, 0)
			upd[1] = make([]interface{}, 0)
			upd[2] = make([]interface{}, 0)
			for _, tx := range txs {
				upd[0] = append(upd[0], tx.state)
				upd[1] = append(upd[1], tx.uid)
				upd[2] = append(upd[2], tx.date)
			}
			values := &sheets.ValueRange{MajorDimension: "COLUMNS", Values: upd}
			ur := srv.Spreadsheets.Values.Update(sid, "Tx!A2:C", values)
			ur.ValueInputOption("RAW")
			_, err := ur.Do()
			return err
		}()
		if err != nil {
			return err
		}

		csvbuf := bytes.NewBuffer(nil)
		toImport := 0
		err = func() error {
			end := start("Generating CSV")
			defer end()
			w := csv.NewWriter(csvbuf)
			row := []string{
				"num",
				"date",
				"account",
				"amount",
				"price",
				"description",
			}
			if err := w.Write(row); err != nil {
				return err
			}
			n := 0
			lastUID := ""
			for _, tx := range txs {
				if lastUID == "" || lastUID != tx.uid {
					n++
					lastUID = tx.uid
				}
				if tx.state != "c" {
					continue
				}

				rows := tx.Fields(n)
				for _, row := range rows {
					toImport++
					if err := w.Write(row); err != nil {
						return err
					}
				}
			}
			w.Flush()
			return w.Error()
		}()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Generated %d rows\n", toImport)
		fmt.Print(csvbuf.String())

		err = func() error {
			end := start("Updating report")
			defer end()
			accs, err := confPrefixArray(conf, KReport)
			if err != nil {
				return err
			}
			_ignores, err := confPrefixArray(conf, KReportIgnore)
			if err != nil {
				return err
			}
			ignores := make([]*regexp.Regexp, len(_ignores))
			for i, ig := range _ignores {
				ignores[i], err = regexp.Compile(ig)
				if err != nil {
					return err
				}
			}

			regexes := make([]*regexp.Regexp, len(accs))
			for i, acc := range accs {
				regexes[i], err = regexp.Compile(acc)
				if err != nil {
					return err
				}
			}

			now := time.Now()
			cur := now.AddDate(-1, 0, 0)
			cur = cur.AddDate(0, 0, -cur.Day()+1)
			all := book.Transactions.Between(cur, now)

			vals := make([][]interface{}, 1, 100)
			vals[0] = make([]interface{}, 1+len(regexes))
			vals[0][0] = "Date"
			for i, acc := range accs {
				vals[0][1+i] = acc
			}

			for ; cur.Before(now); cur = cur.AddDate(0, 1, 0) {
				monthly := all.Between(cur, cur.AddDate(0, 1, 0))

				entry := make([]interface{}, 1+len(regexes))
				entry[0] = cur.Format(dFormat)
				for i, re := range regexes {
					l := monthly.Simplified().RelativeFrom(re).Filter(nil, nil, nil, re)
					for _, ignore := range ignores {
						l = l.Filter(nil, nil, nil, ignore)
					}
					entry[i+1] = -l.Sum()
				}
				vals = append(vals, entry)
			}

			vals = sheetPad(vals, len(accs)+2)

			values := &sheets.ValueRange{
				MajorDimension: "COLUMNS",
				Values:         vals,
			}
			ur := srv.Spreadsheets.Values.Update(sid, "Report!A1", values)
			ur.ValueInputOption("RAW")
			_, err = ur.Do()
			return err
		}()
		if err != nil {
			return err
		}

		return nil
	})

	set, _ := fr.ParseCommandline()
	if err := set.Do(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
