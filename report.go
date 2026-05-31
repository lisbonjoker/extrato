package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"
)

var ptMonthsShort = []string{"Jan", "Fev", "Mar", "Abr", "Mai", "Jun", "Jul", "Ago", "Set", "Out", "Nov", "Dez"}

type reportData struct {
	GeneratedAt     string
	TotalIncome     string
	TotalExpenses   string
	NetBalance      string
	NetPositive     bool
	TxnCount        int
	MonthLabels     []string
	MonthIncome     []float64
	MonthExpenses   []float64
	CategoryLabels  []string
	CategoryAmounts []float64
	BalanceDates    []string
	BalanceValues   []float64
	TopMerchants    []merchantStat
	RecentTxns      []txnRow
}

type merchantStat struct {
	Name       string
	Amount     string
	Count      int
	Proportion float64
}

type txnRow struct {
	Date        string
	Description string
	Category    string
	Amount      string
	Balance     string
	IsDebit     bool
}

func generateReport(txns []Transaction, outputPath string) error {
	sort.Slice(txns, func(i, j int) bool {
		return txns[i].Date.Before(txns[j].Date)
	})

	rd := computeReportData(txns)

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"inc": func(i int) int { return i + 1 },
		"pct": func(f float64) string { return fmt.Sprintf("%.1f", f) },
		"truncate": func(s string, n int) string {
			if len([]rune(s)) <= n {
				return s
			}
			return string([]rune(s)[:n]) + "…"
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, rd)
}

func computeReportData(txns []Transaction) reportData {
	var totalIncome, totalExpenses float64

	type monthBucket struct{ income, expense float64 }
	monthMap := map[string]*monthBucket{}
	catMap := map[string]float64{}
	merchantMap := map[string]*struct {
		amount float64
		count  int
	}{}

	for _, t := range txns {
		if t.Amount > 0 {
			totalIncome += t.Amount
		} else {
			totalExpenses += -t.Amount
		}
		k := t.Date.Format("2006-01")
		if monthMap[k] == nil {
			monthMap[k] = &monthBucket{}
		}
		if t.Amount > 0 {
			monthMap[k].income += t.Amount
		} else {
			monthMap[k].expense += -t.Amount
		}
		if t.Amount < 0 {
			catMap[t.Category] += -t.Amount
			name := shortMerchantName(t.Description)
			if merchantMap[name] == nil {
				merchantMap[name] = &struct{ amount float64; count int }{}
			}
			merchantMap[name].amount += -t.Amount
			merchantMap[name].count++
		}
	}

	// Monthly
	var monthKeys []string
	for k := range monthMap {
		monthKeys = append(monthKeys, k)
	}
	sort.Strings(monthKeys)
	var monthLabels []string
	var monthIncome, monthExpenses []float64
	for _, k := range monthKeys {
		t, _ := time.Parse("2006-01", k)
		monthLabels = append(monthLabels, ptMonthsShort[t.Month()-1]+" '"+t.Format("06"))
		monthIncome = append(monthIncome, round2(monthMap[k].income))
		monthExpenses = append(monthExpenses, round2(monthMap[k].expense))
	}

	// Categories
	type catEntry struct{ name string; amount float64 }
	var cats []catEntry
	for k, v := range catMap {
		cats = append(cats, catEntry{k, v})
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].amount > cats[j].amount })
	var catLabels []string
	var catAmounts []float64
	for _, c := range cats {
		catLabels = append(catLabels, c.name)
		catAmounts = append(catAmounts, round2(c.amount))
	}

	// Balance over time
	var balanceDates []string
	var balanceValues []float64
	hasBalance := false
	for _, t := range txns {
		if t.Balance != 0 {
			hasBalance = true
			break
		}
	}
	if hasBalance {
		for _, t := range txns {
			balanceDates = append(balanceDates, t.Date.Format("02/01"))
			balanceValues = append(balanceValues, t.Balance)
		}
	} else {
		running := 0.0
		for _, t := range txns {
			running = round2(running + t.Amount)
			balanceDates = append(balanceDates, t.Date.Format("02/01"))
			balanceValues = append(balanceValues, running)
		}
	}

	// Top merchants
	type mEntry struct {
		name   string
		amount float64
		count  int
	}
	var merchants []mEntry
	for k, v := range merchantMap {
		merchants = append(merchants, mEntry{k, v.amount, v.count})
	}
	sort.Slice(merchants, func(i, j int) bool { return merchants[i].amount > merchants[j].amount })
	if len(merchants) > 10 {
		merchants = merchants[:10]
	}
	var topMerchants []merchantStat
	for _, m := range merchants {
		prop := 0.0
		if totalExpenses > 0 {
			prop = m.amount / totalExpenses * 100
		}
		topMerchants = append(topMerchants, merchantStat{
			Name:       m.name,
			Amount:     formatPT(m.amount),
			Count:      m.count,
			Proportion: math.Round(prop*10) / 10,
		})
	}

	// Recent transactions (most recent first, max 25)
	var recentTxns []txnRow
	start := len(txns) - 25
	if start < 0 {
		start = 0
	}
	for i := len(txns) - 1; i >= start; i-- {
		t := txns[i]
		balStr := ""
		if t.Balance != 0 {
			balStr = formatPT(t.Balance)
		}
		recentTxns = append(recentTxns, txnRow{
			Date:        t.Date.Format("02/01/2006"),
			Description: t.Description,
			Category:    t.Category,
			Amount:      formatPT(math.Abs(t.Amount)),
			Balance:     balStr,
			IsDebit:     t.Amount < 0,
		})
	}

	net := totalIncome - totalExpenses
	return reportData{
		GeneratedAt:     time.Now().Format("02/01/2006 15:04"),
		TotalIncome:     formatPT(totalIncome),
		TotalExpenses:   formatPT(totalExpenses),
		NetBalance:      formatPT(math.Abs(net)),
		NetPositive:     net >= 0,
		TxnCount:        len(txns),
		MonthLabels:     monthLabels,
		MonthIncome:     monthIncome,
		MonthExpenses:   monthExpenses,
		CategoryLabels:  catLabels,
		CategoryAmounts: catAmounts,
		BalanceDates:    balanceDates,
		BalanceValues:   balanceValues,
		TopMerchants:    topMerchants,
		RecentTxns:      recentTxns,
	}
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

func shortMerchantName(desc string) string {
	desc = strings.TrimSpace(desc)
	parts := strings.Fields(desc)
	if len(parts) > 4 {
		parts = parts[:4]
	}
	return strings.Join(parts, " ")
}

// formatPT formats a float as pt-PT currency, e.g. 1.234,56 €
func formatPT(f float64) string {
	neg := f < 0
	if neg {
		f = -f
	}
	f = math.Round(f*100) / 100
	intPart := int64(f)
	decPart := int64(math.Round((f-float64(intPart))*100))
	if decPart >= 100 {
		intPart++
		decPart -= 100
	}
	intStr := fmt.Sprintf("%d", intPart)
	var buf strings.Builder
	for i, ch := range intStr {
		if i > 0 && (len(intStr)-i)%3 == 0 {
			buf.WriteByte('.')
		}
		buf.WriteRune(ch)
	}
	result := fmt.Sprintf("%s,%02d €", buf.String(), decPart)
	if neg {
		return "-" + result
	}
	return result
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="pt-PT">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Extrato Bancário — Relatório</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
<style>
:root {
  --red: #EC0000;
  --red-dim: rgba(236,0,0,0.12);
  --bg: #0d0d0d;
  --surface: #181818;
  --surface2: #222;
  --border: #2e2e2e;
  --text: #e8e8e8;
  --muted: #777;
  --green: #22c55e;
  --red-loss: #f87171;
}
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
html{font-size:15px}
body{background:var(--bg);color:var(--text);font-family:system-ui,-apple-system,sans-serif;line-height:1.5}
a{color:inherit}

header{
  background:var(--surface);
  border-bottom:3px solid var(--red);
  padding:1.25rem 2rem;
  display:flex;align-items:center;gap:1rem;
}
.logo-bar{width:8px;height:44px;background:var(--red);border-radius:3px;flex-shrink:0}
header h1{font-size:1.4rem;font-weight:700;letter-spacing:-0.02em}
header p{color:var(--muted);font-size:0.8rem;margin-top:2px}

main{max-width:1440px;margin:0 auto;padding:1.75rem 2rem 3rem}

/* KPIs */
.kpi-grid{
  display:grid;
  grid-template-columns:repeat(4,1fr);
  gap:1rem;
  margin-bottom:1.5rem;
}
.kpi{
  background:var(--surface);
  border:1px solid var(--border);
  border-radius:14px;
  padding:1.4rem 1.6rem;
}
.kpi-label{
  font-size:0.7rem;
  text-transform:uppercase;
  letter-spacing:0.1em;
  color:var(--muted);
  margin-bottom:0.5rem;
}
.kpi-value{
  font-size:1.65rem;
  font-weight:800;
  letter-spacing:-0.03em;
  line-height:1.1;
}
.green{color:var(--green)}
.redval{color:var(--red-loss)}
.accent{color:var(--red)}

/* Chart grid */
.charts-top{
  display:grid;
  grid-template-columns:1.6fr 1fr;
  gap:1rem;
  margin-bottom:1rem;
}
.chart-card{
  background:var(--surface);
  border:1px solid var(--border);
  border-radius:14px;
  padding:1.4rem 1.6rem;
}
.chart-card h2{
  font-size:0.75rem;
  text-transform:uppercase;
  letter-spacing:0.08em;
  color:var(--muted);
  margin-bottom:1rem;
}
.chart-card canvas{max-height:280px}
.balance-wrap{
  background:var(--surface);
  border:1px solid var(--border);
  border-radius:14px;
  padding:1.4rem 1.6rem;
  margin-bottom:1rem;
}
.balance-wrap h2{
  font-size:0.75rem;
  text-transform:uppercase;
  letter-spacing:0.08em;
  color:var(--muted);
  margin-bottom:1rem;
}
.balance-wrap canvas{max-height:200px}

/* Bottom grid */
.bottom-grid{
  display:grid;
  grid-template-columns:1fr 1.5fr;
  gap:1rem;
}
.panel{
  background:var(--surface);
  border:1px solid var(--border);
  border-radius:14px;
  padding:1.4rem 1.6rem;
  overflow:hidden;
}
.panel h2{
  font-size:0.75rem;
  text-transform:uppercase;
  letter-spacing:0.08em;
  color:var(--muted);
  margin-bottom:1rem;
}

/* Merchants */
.merchant-row{
  display:flex;
  align-items:center;
  gap:0.6rem;
  padding:0.45rem 0;
  border-bottom:1px solid var(--border);
}
.merchant-row:last-child{border-bottom:none}
.m-rank{width:1.4rem;font-size:0.7rem;color:var(--muted);text-align:center;flex-shrink:0}
.m-name{flex:1;font-size:0.85rem;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.m-bar-bg{width:80px;height:5px;background:var(--border);border-radius:3px;flex-shrink:0;overflow:hidden}
.m-bar{height:100%;background:var(--red);border-radius:3px}
.m-amount{width:7rem;text-align:right;font-size:0.8rem;font-weight:600;flex-shrink:0;color:var(--red-loss)}
.m-count{width:3.5rem;text-align:right;font-size:0.72rem;color:var(--muted);flex-shrink:0}

/* Table */
.tbl-wrap{overflow-x:auto;}
table{width:100%;border-collapse:collapse;font-size:0.82rem}
th{
  text-align:left;
  font-size:0.68rem;
  text-transform:uppercase;
  letter-spacing:0.06em;
  color:var(--muted);
  padding:0.5rem 0.75rem;
  border-bottom:1px solid var(--border);
  white-space:nowrap;
}
td{
  padding:0.55rem 0.75rem;
  border-bottom:1px solid var(--border);
  vertical-align:middle;
}
tr:last-child td{border-bottom:none}
.td-date{white-space:nowrap;color:var(--muted);font-size:0.78rem}
.td-desc{max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.badge{
  display:inline-block;
  padding:0.15rem 0.55rem;
  border-radius:999px;
  font-size:0.65rem;
  background:var(--surface2);
  color:var(--muted);
  white-space:nowrap;
}
.debit{color:var(--red-loss);font-weight:700;text-align:right}
.credit{color:var(--green);font-weight:700;text-align:right}
.td-bal{color:var(--muted);text-align:right;font-size:0.78rem}

footer{
  text-align:center;
  padding:2rem;
  color:var(--muted);
  font-size:0.72rem;
}

@media(max-width:960px){
  .kpi-grid{grid-template-columns:1fr 1fr}
  .charts-top{grid-template-columns:1fr}
  .bottom-grid{grid-template-columns:1fr}
}
@media(max-width:480px){
  .kpi-grid{grid-template-columns:1fr}
  main{padding:1rem}
}
</style>
</head>
<body>
<header>
  <div class="logo-bar"></div>
  <div>
    <h1>Extrato Bancário</h1>
    <p>Relatório gerado em {{.GeneratedAt}}</p>
  </div>
</header>

<main>
  <!-- KPIs -->
  <div class="kpi-grid">
    <div class="kpi">
      <div class="kpi-label">Entradas</div>
      <div class="kpi-value green">{{.TotalIncome}}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Saídas</div>
      <div class="kpi-value redval">{{.TotalExpenses}}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Saldo Líquido</div>
      <div class="kpi-value {{if .NetPositive}}green{{else}}redval{{end}}">{{if not .NetPositive}}-{{end}}{{.NetBalance}}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Nº Transações</div>
      <div class="kpi-value accent">{{.TxnCount}}</div>
    </div>
  </div>

  <!-- Monthly + Category charts -->
  <div class="charts-top">
    <div class="chart-card">
      <h2>Receitas vs Despesas Mensais</h2>
      <canvas id="chartMonthly"></canvas>
    </div>
    <div class="chart-card">
      <h2>Despesas por Categoria</h2>
      <canvas id="chartCategory"></canvas>
    </div>
  </div>

  <!-- Balance area chart -->
  <div class="balance-wrap">
    <h2>Evolução do Saldo</h2>
    <canvas id="chartBalance"></canvas>
  </div>

  <!-- Merchants + Transactions -->
  <div class="bottom-grid">
    <div class="panel">
      <h2>Top 10 Comerciantes</h2>
      {{range $i, $m := .TopMerchants}}
      <div class="merchant-row">
        <div class="m-rank">{{inc $i}}</div>
        <div class="m-name" title="{{$m.Name}}">{{truncate $m.Name 28}}</div>
        <div class="m-bar-bg"><div class="m-bar" style="width:{{pct $m.Proportion}}%"></div></div>
        <div class="m-amount">{{$m.Amount}}</div>
        <div class="m-count">{{$m.Count}}x</div>
      </div>
      {{end}}
    </div>

    <div class="panel">
      <h2>Transações Recentes</h2>
      <div class="tbl-wrap">
      <table>
        <thead>
          <tr>
            <th>Data</th>
            <th>Descrição</th>
            <th>Categoria</th>
            <th style="text-align:right">Valor</th>
            <th style="text-align:right">Saldo</th>
          </tr>
        </thead>
        <tbody>
          {{range .RecentTxns}}
          <tr>
            <td class="td-date">{{.Date}}</td>
            <td class="td-desc" title="{{.Description}}">{{truncate .Description 32}}</td>
            <td><span class="badge">{{.Category}}</span></td>
            <td class="{{if .IsDebit}}debit{{else}}credit{{end}}">{{if .IsDebit}}-{{end}}{{.Amount}}</td>
            <td class="td-bal">{{.Balance}}</td>
          </tr>
          {{end}}
        </tbody>
      </table>
      </div>
    </div>
  </div>
</main>

<footer>Gerado por <strong>extrato</strong> &mdash; Processamento 100% local, sem envio de dados</footer>

<script>
const monthLabels = {{json .MonthLabels}};
const monthIncome = {{json .MonthIncome}};
const monthExpenses = {{json .MonthExpenses}};
const catLabels = {{json .CategoryLabels}};
const catAmounts = {{json .CategoryAmounts}};
const balDates = {{json .BalanceDates}};
const balValues = {{json .BalanceValues}};

Chart.defaults.color = '#777';
Chart.defaults.borderColor = '#2e2e2e';
Chart.defaults.font.family = "system-ui, -apple-system, sans-serif";

// Monthly bar chart
new Chart(document.getElementById('chartMonthly'), {
  type: 'bar',
  data: {
    labels: monthLabels,
    datasets: [
      {
        label: 'Entradas',
        data: monthIncome,
        backgroundColor: 'rgba(34,197,94,0.75)',
        borderColor: '#22c55e',
        borderWidth: 1,
        borderRadius: 5,
      },
      {
        label: 'Saídas',
        data: monthExpenses,
        backgroundColor: 'rgba(248,113,113,0.75)',
        borderColor: '#f87171',
        borderWidth: 1,
        borderRadius: 5,
      }
    ]
  },
  options: {
    responsive: true,
    maintainAspectRatio: true,
    plugins: {
      legend: { labels: { boxWidth: 12, padding: 16, color: '#aaa' } },
      tooltip: { callbacks: { label: ctx => ' ' + ctx.dataset.label + ': ' + ctx.parsed.y.toLocaleString('pt-PT', {style:'currency',currency:'EUR'}) } }
    },
    scales: {
      x: { grid: { color: '#222' }, ticks: { color: '#666' } },
      y: { grid: { color: '#222' }, ticks: { color: '#666', callback: v => v.toLocaleString('pt-PT', {style:'currency',currency:'EUR',maximumFractionDigits:0}) }, beginAtZero: true }
    }
  }
});

// Category doughnut
const palette = [
  '#EC0000','#f87171','#fb923c','#fbbf24','#a3e635',
  '#34d399','#22d3ee','#818cf8','#c084fc','#f472b6',
  '#94a3b8','#64748b'
];
new Chart(document.getElementById('chartCategory'), {
  type: 'doughnut',
  data: {
    labels: catLabels,
    datasets: [{
      data: catAmounts,
      backgroundColor: palette,
      borderWidth: 2,
      borderColor: '#181818',
      hoverOffset: 6,
    }]
  },
  options: {
    responsive: true,
    maintainAspectRatio: true,
    cutout: '62%',
    plugins: {
      legend: {
        position: 'right',
        labels: { boxWidth: 11, padding: 8, font: { size: 11 }, color: '#aaa' }
      },
      tooltip: { callbacks: { label: ctx => ' ' + ctx.label + ': ' + ctx.parsed.toLocaleString('pt-PT', {style:'currency',currency:'EUR'}) } }
    }
  }
});

// Balance area chart
const sparse = balDates.length > 150;
new Chart(document.getElementById('chartBalance'), {
  type: 'line',
  data: {
    labels: balDates,
    datasets: [{
      label: 'Saldo',
      data: balValues,
      borderColor: '#EC0000',
      backgroundColor: 'rgba(236,0,0,0.07)',
      fill: true,
      tension: 0.35,
      pointRadius: sparse ? 0 : 2,
      pointHoverRadius: 4,
      borderWidth: 2,
    }]
  },
  options: {
    responsive: true,
    maintainAspectRatio: true,
    plugins: {
      legend: { display: false },
      tooltip: { callbacks: { label: ctx => ' Saldo: ' + ctx.parsed.y.toLocaleString('pt-PT', {style:'currency',currency:'EUR'}) } }
    },
    scales: {
      x: { grid: { color: '#1e1e1e' }, ticks: { color: '#555', maxTicksLimit: 14 } },
      y: { grid: { color: '#1e1e1e' }, ticks: { color: '#555', callback: v => v.toLocaleString('pt-PT', {style:'currency',currency:'EUR',maximumFractionDigits:0}) } }
    }
  }
});
</script>
</body>
</html>`
