package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Transaction represents a single bank statement entry.
type Transaction struct {
	Date        time.Time
	Description string
	Amount      float64 // positive = credit/income, negative = debit/expense
	Balance     float64
	Category    string
}

// parseCSV parses Santander Portugal CSV exports.
// Handles semicolon and comma separators, pt-PT number formats, BOM.
func parseCSV(data []byte) ([]Transaction, error) {
	// Strip UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	sep := detectSeparator(data)

	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = rune(sep)
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV inválido: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("ficheiro CSV sem dados")
	}

	header := records[0]
	cols := make(map[string]int, len(header))
	for i, h := range header {
		cols[normalizeHeader(h)] = i
	}

	dateCol := findCol(cols, "data", "date", "data mov", "data movimento", "data valor", "data transacao", "transaction date", "dt")
	descCol := findCol(cols, "descricao", "description", "memo", "historico", "detalhe", "descricao do movimento", "designacao", "nome")
	debitCol := findCol(cols, "debito", "saidas", "debit", "valor debito", "montante debito", "debitos")
	creditCol := findCol(cols, "credito", "entradas", "credit", "valor credito", "montante credito", "creditos")
	amountCol := findCol(cols, "valor", "amount", "montante", "importancia", "quantia", "movimento")
	balanceCol := findCol(cols, "saldo", "balance", "saldo contabilistico", "saldo disponivel", "saldo apos movimento")

	if dateCol < 0 {
		return nil, fmt.Errorf("coluna de data não encontrada; cabeçalho: %v", header)
	}
	if descCol < 0 {
		return nil, fmt.Errorf("coluna de descrição não encontrada; cabeçalho: %v", header)
	}

	var txns []Transaction
	for _, row := range records[1:] {
		if allEmpty(row) {
			continue
		}
		dateStr := safeGet(row, dateCol)
		if dateStr == "" {
			continue
		}
		t, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		desc := strings.TrimSpace(safeGet(row, descCol))

		var amount float64
		if debitCol >= 0 && creditCol >= 0 {
			debit := parsePTAmount(safeGet(row, debitCol))
			credit := parsePTAmount(safeGet(row, creditCol))
			amount = credit - debit
		} else if amountCol >= 0 {
			amount = parsePTAmount(safeGet(row, amountCol))
		}

		balance := 0.0
		if balanceCol >= 0 {
			balance = parsePTAmount(safeGet(row, balanceCol))
		}

		txns = append(txns, Transaction{
			Date:        t,
			Description: desc,
			Amount:      math.Round(amount*100) / 100,
			Balance:     math.Round(balance*100) / 100,
		})
	}
	return txns, nil
}

// parseOFX parses OFX/SGML bank exports (Santander Portugal format).
func parseOFX(data []byte) ([]Transaction, error) {
	// Strip BOM
	content := strings.TrimPrefix(string(data), "\xef\xbb\xbf")
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	upper := strings.ToUpper(content)

	var txns []Transaction
	offset := 0
	for {
		idx := strings.Index(upper[offset:], "<STMTTRN>")
		if idx < 0 {
			break
		}
		abs := offset + idx
		endTag := strings.Index(upper[abs:], "</STMTTRN>")
		if endTag < 0 {
			break
		}
		block := content[abs : abs+endTag+len("</STMTTRN>")]
		offset = abs + endTag + len("</STMTTRN>")

		if txn := parseOFXBlock(block); txn != nil {
			txns = append(txns, *txn)
		}
	}
	return txns, nil
}

func parseOFXBlock(block string) *Transaction {
	dtposted := ofxField(block, "DTPOSTED")
	trnamt := ofxField(block, "TRNAMT")
	name := ofxField(block, "NAME")
	memo := ofxField(block, "MEMO")

	if dtposted == "" || trnamt == "" {
		return nil
	}
	t, err := parseOFXDate(dtposted)
	if err != nil {
		return nil
	}
	amtStr := strings.TrimSpace(trnamt)
	amtStr = strings.ReplaceAll(amtStr, ",", ".")
	amount, err := strconv.ParseFloat(amtStr, 64)
	if err != nil {
		return nil
	}

	desc := strings.TrimSpace(name)
	memoClean := strings.TrimSpace(memo)
	if memoClean != "" && memoClean != desc {
		if desc == "" {
			desc = memoClean
		} else {
			desc = desc + " — " + memoClean
		}
	}

	return &Transaction{
		Date:        t,
		Description: desc,
		Amount:      math.Round(amount*100) / 100,
	}
}

func ofxField(block, field string) string {
	tag := "<" + field + ">"
	idx := strings.Index(strings.ToUpper(block), tag)
	if idx < 0 {
		return ""
	}
	rest := block[idx+len(tag):]
	end := strings.IndexAny(rest, "\n\r<")
	if end < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}

func parseOFXDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	// Strip timezone suffix e.g. [+1:CET]
	if i := strings.IndexByte(s, '['); i >= 0 {
		s = s[:i]
	}
	for _, layout := range []string{"20060102150405", "20060102"} {
		if len(s) >= len(layout) {
			if t, err := time.Parse(layout, s[:len(layout)]); err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("data OFX inválida: %s", s)
}

func detectSeparator(data []byte) byte {
	counts := map[byte]int{';': 0, ',': 0, '\t': 0}
	limit := 1000
	if len(data) < limit {
		limit = len(data)
	}
	for _, b := range data[:limit] {
		if _, ok := counts[b]; ok {
			counts[b]++
		}
	}
	best := byte(';')
	max := -1
	for b, c := range counts {
		if c > max {
			max = c
			best = b
		}
	}
	return best
}

func normalizeHeader(s string) string {
	s = strings.ToLower(strings.Trim(strings.TrimSpace(s), `"'`))
	s = strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a",
		"é", "e", "ê", "e", "è", "e",
		"í", "i", "ï", "i",
		"ó", "o", "ô", "o", "õ", "o",
		"ú", "u", "ü", "u",
		"ç", "c",
	).Replace(s)
	return s
}

func findCol(cols map[string]int, names ...string) int {
	for _, name := range names {
		if i, ok := cols[normalizeHeader(name)]; ok {
			return i
		}
	}
	// Partial match
	for _, name := range names {
		norm := normalizeHeader(name)
		for k, i := range cols {
			if strings.Contains(k, norm) || strings.Contains(norm, k) {
				return i
			}
		}
	}
	return -1
}

func safeGet(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.Trim(strings.TrimSpace(row[i]), `"'`)
}

func allEmpty(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

// parsePTAmount parses pt-PT formatted numbers like "1.234,56" or "-50,00".
func parsePTAmount(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "—" {
		return 0
	}
	// Remove currency symbol, non-breaking spaces
	s = strings.NewReplacer("€", "", " ", "", " ", "").Replace(s)
	// Handle trailing minus: "50,00-" → "-50.00"
	neg := false
	if strings.HasSuffix(s, "-") {
		neg = true
		s = s[:len(s)-1]
	}
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	// Remove thousands separator (.) only when followed by 3 digits not at end
	// Strategy: if comma exists use it as decimal, then dots are thousands
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	if neg {
		f = -f
	}
	return f
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		"02-01-2006", "02/01/2006",
		"2006-01-02", "2006/01/02",
		"01-02-2006", "01/02/2006",
		"02-01-06", "02/01/06",
		"2006-01-02 15:04:05",
		"02-01-2006 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("data inválida: %s", s)
}
