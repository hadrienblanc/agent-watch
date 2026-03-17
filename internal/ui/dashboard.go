package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"claude_monitor/internal/data"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type statsMsg *data.Stats
type refreshMsg struct{}

const totalTabs = 5

type sortOrder struct {
	col string
	asc bool
}

type Dashboard struct {
	stats  *data.Stats
	width  int
	height int
	tab    int // 0=overview, 1=sessions, 2=tools, 3=projects
	scroll int
	loading bool
	sortSessions sortOrder
	sortTools    sortOrder
	sortProjects sortOrder
	sortCosts    sortOrder
	costView     string // "g" = graph, "t" = table
}

func NewDashboard() Dashboard {
	return Dashboard{
		loading:      true,
		sortSessions: sortOrder{col: "m", asc: false},
		sortTools:    sortOrder{col: "a", asc: false},
		sortProjects: sortOrder{col: "m", asc: false},
		sortCosts:    sortOrder{col: "date", asc: false},
		costView:     "g",
	}
}

func loadStats() tea.Msg {
	stats, err := data.LoadStats()
	if err != nil {
		return statsMsg(nil)
	}
	return statsMsg(stats)
}

func refreshCmd() tea.Cmd {
	return tea.Tick(60*time.Second, func(t time.Time) tea.Msg {
		return refreshMsg{}
	})
}

func (d Dashboard) Init() tea.Cmd {
	return tea.Batch(loadStats, refreshCmd())
}

func (d Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case statsMsg:
		d.stats = msg
		d.loading = false

	case refreshMsg:
		d.loading = true
		return d, tea.Batch(loadStats, refreshCmd())

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "escape":
			return d, tea.Quit
		case "1":
			d.tab = 0
			d.scroll = 0
		case "2":
			d.tab = 1
			d.scroll = 0
		case "3":
			d.tab = 2
			d.scroll = 0
		case "4":
			d.tab = 3
			d.scroll = 0
		case "5":
			d.tab = 4
			d.scroll = 0
		case "tab", "right":
			d.tab = (d.tab + 1) % totalTabs
			d.scroll = 0
		case "shift+tab", "left":
			d.tab = (d.tab - 1 + totalTabs) % totalTabs
			d.scroll = 0
		case "r":
			d.loading = true
			return d, loadStats
		case "j", "down":
			d.scroll++
		case "k", "up":
			if d.scroll > 0 {
				d.scroll--
			}
		default:
			d.handleSortKey(msg.String())
		}
	}

	return d, nil
}

func (d Dashboard) View() tea.View {
	if d.width < 40 {
		v := tea.NewView("Terminal trop petit")
		v.AltScreen = true
		return v
	}

	if d.loading {
		s := lipgloss.NewStyle().Padding(2, 4).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				headerStyle.Render(" Claude Monitor "),
				"",
				labelStyle.Render("Chargement des conversations..."),
			),
		)
		v := tea.NewView(s)
		v.AltScreen = true
		return v
	}

	if d.stats == nil {
		s := lipgloss.NewStyle().Padding(2, 4).Render(
			errValStyle.Render("Impossible de charger les données Claude"),
		)
		v := tea.NewView(s)
		v.AltScreen = true
		return v
	}

	w := d.width - 4

	header := lipgloss.PlaceHorizontal(w, lipgloss.Center, headerStyle.Render(" Claude Monitor "))
	tabs := d.viewTabs()

	var content string
	switch d.tab {
	case 0:
		content = d.viewOverview(w)
	case 1:
		content = d.viewSessions(w)
	case 2:
		content = d.viewTools(w)
	case 3:
		content = d.viewProjects(w)
	case 4:
		content = d.viewCosts(w)
	}

	status := d.viewStatus(w)

	page := lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabs,
		"",
		content,
		"",
		status,
	)

	s := lipgloss.NewStyle().Padding(1, 2).Render(page)
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func (d Dashboard) viewTabs() string {
	tabs := []string{"1:Overview", "2:Sessions", "3:Outils", "4:Projets", "5:Coûts"}
	var parts []string
	for i, t := range tabs {
		if i == d.tab {
			parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(purple).Render("["+t+"]"))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(dimWhite).Render(" "+t+" "))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// --- Tab 0: Overview ---

func (d Dashboard) viewOverview(w int) string {
	s := d.stats
	thirdW := (w - 6) / 3
	if thirdW < 25 {
		thirdW = 25
	}
	halfW := (w - 5) / 2
	if halfW < 30 {
		halfW = 30
	}

	// Activité
	activeLabel := bigNumStyle.Render(fmt.Sprintf("%d", s.ActiveSessions))
	if s.ActiveSessions == 0 {
		activeLabel = labelStyle.Render("0")
	}

	activity := d.panel("Activité", thirdW,
		kv{"Sessions actives", activeLabel},
		kv{"", ""},
		kv{"Aujourd'hui", ""},
		kv{"  Sessions", bigNumStyle.Render(fmt.Sprintf("%d", s.TodaySessions))},
		kv{"  Messages", valueStyle.Render(fmtNum(s.TodayMessages))},
		kv{"  Tokens", valueStyle.Render(fmtNum(s.TodayTokens))},
	)

	thisWeek := d.panel("Cette semaine", thirdW,
		kv{"Sessions", bigNumStyle.Render(fmt.Sprintf("%d", s.WeekSessions))},
		kv{"Messages", valueStyle.Render(fmtNum(s.WeekMessages))},
		kv{"Tokens", valueStyle.Render(fmtNum(s.WeekTokens))},
		kv{"", ""},
		kv{"Tout temps", ""},
		kv{"  Sessions", labelStyle.Render(fmt.Sprintf("%d", s.TotalSessions))},
	)

	// Tokens
	totalTokens := s.TotalInputTokens + s.TotalOutputTokens
	cacheRate := 0.0
	if s.TotalInputTokens > 0 {
		cacheRate = float64(s.TotalCacheRead) / float64(s.TotalInputTokens) * 100
	}

	tokens := d.panel("Tokens (tout temps)", thirdW,
		kv{"Total", bigNumStyle.Render(fmtNum(totalTokens))},
		kv{"Input", valueStyle.Render(fmtNum(s.TotalInputTokens))},
		kv{"Output", valueStyle.Render(fmtNum(s.TotalOutputTokens))},
		kv{"Cache lu", cyanStyle.Render(fmtNum(s.TotalCacheRead))},
		kv{"Taux cache", d.colorCacheRate(cacheRate)},
		kv{"Coût estimé", orangeStyle.Render(fmt.Sprintf("~$%.2f", s.TotalCost))},
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, activity, " ", thisWeek, " ", tokens)

	// Métriques secondaires
	errRate := 0.0
	if s.TotalToolUses > 0 {
		errRate = float64(s.TotalToolErrors) / float64(s.TotalToolUses) * 100
	}

	secondary := d.panel("Santé", halfW,
		kv{"Modèle principal", cyanStyle.Render(s.ActiveModel)},
		kv{"Appels outils", valueStyle.Render(fmtNum(s.TotalToolUses))},
		kv{"Erreurs outils", d.colorErrors(s.TotalToolErrors)},
		kv{"Taux erreur", d.colorErrorRate(errRate)},
	)

	// Modèles
	var modelRows []kv
	type modelEntry struct {
		name  string
		count int
	}
	var models []modelEntry
	for m, c := range s.Models {
		models = append(models, modelEntry{m, c})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].count > models[j].count })
	for _, m := range models {
		pct := float64(m.count) / float64(max(s.TotalMessages, 1)) * 100
		bar := d.miniBar(pct, 20)
		modelRows = append(modelRows, kv{m.name, fmt.Sprintf("%s %s %s",
			valueStyle.Render(fmtNum(m.count)),
			labelStyle.Render(fmt.Sprintf("(%.0f%%)", pct)),
			bar,
		)})
	}
	modelsPanel := d.panel("Modèles", halfW, modelRows...)

	midRow := lipgloss.JoinHorizontal(lipgloss.Top, secondary, " ", modelsPanel)

	// Sources
	type sourceEntry struct {
		name     string
		sessions int
		messages int
		cost     float64
	}
	srcMap := make(map[string]*sourceEntry)
	for _, sess := range s.Sessions {
		src := sess.Source
		if src == "" {
			src = "claude"
		}
		se := srcMap[src]
		if se == nil {
			se = &sourceEntry{name: src}
			srcMap[src] = se
		}
		se.sessions++
		se.messages += sess.UserMessages + sess.AssistantMessages
		se.cost += sess.Cost
	}
	var srcs []sourceEntry
	for _, se := range srcMap {
		srcs = append(srcs, *se)
	}
	sort.Slice(srcs, func(i, j int) bool { return srcs[i].sessions > srcs[j].sessions })

	var srcRows []kv
	for _, se := range srcs {
		pct := float64(se.sessions) / float64(max(s.TotalSessions, 1)) * 100
		bar := d.miniBar(pct, 15)
		srcRows = append(srcRows, kv{se.name, fmt.Sprintf("%s %s  %s msgs  %s %s",
			valueStyle.Render(fmt.Sprintf("%d", se.sessions)),
			labelStyle.Render("sess"),
			valueStyle.Render(fmtNum(se.messages)),
			orangeStyle.Render(fmt.Sprintf("$%.0f", se.cost)),
			bar,
		)})
	}
	sourcesPanel := d.panel("Sources", w, srcRows...)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, "", midRow, "", sourcesPanel)
}

// --- Tab 1: Sessions ---

func (d Dashboard) viewSessions(w int) string {
	s := d.stats

	const (
		colProjet  = 28
		colMsgs    = 8
		colTools   = 8
		colTokens  = 10
		colDuree   = 8
	)

	// Tri
	sessions := make([]data.Session, len(s.Sessions))
	copy(sessions, s.Sessions)
	so := d.sortSessions
	sort.Slice(sessions, func(i, j int) bool {
		a, b := sessions[i], sessions[j]
		less := false
		switch so.col {
		case "projet":
			less = a.Project < b.Project
		case "msgs":
			less = (a.UserMessages + a.AssistantMessages) < (b.UserMessages + b.AssistantMessages)
		case "tools":
			ta, tb := 0, 0
			for _, c := range a.ToolUses { ta += c }
			for _, c := range b.ToolUses { tb += c }
			less = ta < tb
		case "tokens":
			less = (a.InputTokens + a.OutputTokens) < (b.InputTokens + b.OutputTokens)
		case "duree":
			less = a.TotalDuration < b.TotalDuration
		}
		if so.asc {
			return less
		}
		return !less
	})

	var rows []string

	si := sortIndicator
	header := fmt.Sprintf("  %-*s %*s %*s %*s %*s",
		colProjet, "(p)rojet"+si(so, "projet"),
		colMsgs, "(m)sgs"+si(so, "msgs"),
		colTools, "(t)ools"+si(so, "tools"),
		colTokens, "t(o)kens"+si(so, "tokens"),
		colDuree, "(d)urée"+si(so, "duree"),
	)
	rows = append(rows, tableHeaderStyle.Render(header))
	rows = append(rows, labelStyle.Render("  "+strings.Repeat("─", colProjet+colMsgs+colTools+colTokens+colDuree+6)))

	start := d.scroll
	end := min(start+15, len(sessions))
	if start >= len(sessions) {
		start = max(0, len(sessions)-1)
		end = len(sessions)
	}

	for _, sess := range sessions[start:end] {
		proj := sess.Project
		if len(proj) > colProjet-2 {
			proj = proj[:colProjet-2]
		}

		totalMsgs := sess.UserMessages + sess.AssistantMessages
		totalTools := 0
		for _, c := range sess.ToolUses {
			totalTools += c
		}
		totalTokens := sess.InputTokens + sess.OutputTokens

		row := fmt.Sprintf("  %-*s %*d %*d %*s %*s",
			colProjet, proj,
			colMsgs, totalMsgs,
			colTools, totalTools,
			colTokens, fmtNum(totalTokens),
			colDuree, fmtDuration(sess.TotalDuration),
		)
		rows = append(rows, row)
	}

	if len(sessions) > 15 {
		rows = append(rows, "")
		rows = append(rows, labelStyle.Render(fmt.Sprintf("  %d/%d sessions (j/k pour défiler)", min(end, len(sessions)), len(sessions))))
	}

	content := strings.Join(rows, "\n")
	return panelStyle.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			panelTitleStyle.Render("Sessions récentes"),
			content,
		),
	)
}

// --- Tab 2: Tools ---

func (d Dashboard) viewTools(w int) string {
	s := d.stats

	type toolEntry struct {
		name  string
		count int
	}
	var tools []toolEntry
	for t, c := range s.ToolUsage {
		tools = append(tools, toolEntry{t, c})
	}
	so := d.sortTools
	sort.Slice(tools, func(i, j int) bool {
		less := false
		switch so.col {
		case "outil":
			less = tools[i].name < tools[j].name
		case "appels", "pct":
			less = tools[i].count < tools[j].count
		}
		if so.asc {
			return less
		}
		return !less
	})

	const (
		colTool = 20
		colCall = 8
		colPct  = 8
	)

	var rows []string

	si := sortIndicator
	header := fmt.Sprintf("  %-*s %*s %*s  %s",
		colTool, "(o)util"+si(so, "outil"),
		colCall, "(a)ppels"+si(so, "appels"),
		colPct, "(%)"+si(so, "pct"),
		"Distribution",
	)
	rows = append(rows, tableHeaderStyle.Render(header))
	rows = append(rows, labelStyle.Render("  "+strings.Repeat("─", colTool+colCall+colPct+35)))

	for _, t := range tools {
		pct := float64(t.count) / float64(max(s.TotalToolUses, 1)) * 100
		bar := d.miniBar(pct, 30)
		row := fmt.Sprintf("  %-*s %*s %*s  %s",
			colTool, t.name,
			colCall, fmtNum(t.count),
			colPct, fmt.Sprintf("%.1f%%", pct),
			bar,
		)
		rows = append(rows, row)
	}

	rows = append(rows, "")
	rows = append(rows, fmt.Sprintf("  Total: %s appels  •  Erreurs: %d",
		fmtNum(s.TotalToolUses),
		s.TotalToolErrors,
	))

	content := strings.Join(rows, "\n")
	return panelStyle.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			panelTitleStyle.Render("Utilisation des outils"),
			content,
		),
	)
}

// --- Tab 3: Projects ---

func (d Dashboard) viewProjects(w int) string {
	s := d.stats

	const (
		colName     = 30
		colSessions = 10
		colMessages = 10
		colTokens   = 10
		colCost     = 10
		tableMin    = colName + colSessions + colMessages + colTokens + colCost + 14
	)

	// Tri
	projects := make([]data.ProjectSummary, len(s.Projects))
	copy(projects, s.Projects)
	so := d.sortProjects
	sort.Slice(projects, func(i, j int) bool {
		a, b := projects[i], projects[j]
		less := false
		switch so.col {
		case "projet":
			less = a.Name < b.Name
		case "sessions":
			less = a.Sessions < b.Sessions
		case "messages":
			less = a.Messages < b.Messages
		case "tokens":
			less = a.Tokens < b.Tokens
		case "cout":
			less = a.Cost < b.Cost
		}
		if so.asc {
			return less
		}
		return !less
	})

	var rows []string

	si := sortIndicator
	header := fmt.Sprintf("  %-*s %*s %*s %*s %*s",
		colName, "(p)rojet"+si(so, "projet"),
		colSessions, "(s)essions"+si(so, "sessions"),
		colMessages, "(m)essages"+si(so, "messages"),
		colTokens, "(t)okens"+si(so, "tokens"),
		colCost, "(c)oût"+si(so, "cout"),
	)
	rows = append(rows, tableHeaderStyle.Render(header))
	rows = append(rows, labelStyle.Render("  "+strings.Repeat("─", tableMin)))

	showBar := w > tableMin+20
	maxMsgs := 0
	if showBar {
		for _, p := range projects {
			if p.Messages > maxMsgs {
				maxMsgs = p.Messages
			}
		}
	}

	for _, p := range projects {
		name := p.Name
		if len(name) > colName-2 {
			name = name[:colName-2]
		}

		bar := ""
		if showBar && maxMsgs > 0 {
			pct := float64(p.Messages) / float64(maxMsgs) * 100
			bar = "  " + d.miniBar(pct, 15)
		}

		row := fmt.Sprintf("  %-*s %*d %*s %*s %*s%s",
			colName, name,
			colSessions, p.Sessions,
			colMessages, fmtNum(p.Messages),
			colTokens, fmtNum(p.Tokens),
			colCost, fmt.Sprintf("$%.2f", p.Cost),
			bar,
		)
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")
	return panelStyle.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			panelTitleStyle.Render("Projets"),
			content,
		),
	)
}

// --- Tab 4: Coûts ---

func (d Dashboard) viewCosts(w int) string {
	s := d.stats
	days := s.DailyCosts

	// Résumé en haut
	todayCost := 0.0
	weekCost := 0.0
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()-time.Monday))
	if todayStart.Weekday() == time.Sunday {
		weekStart = todayStart.AddDate(0, 0, -6)
	}

	for _, dc := range days {
		if !dc.Date.Before(todayStart) {
			todayCost += dc.Cost
		}
		if !dc.Date.Before(weekStart) {
			weekCost += dc.Cost
		}
	}

	thirdW := (w - 6) / 3
	if thirdW < 20 {
		thirdW = 20
	}

	summaryToday := d.panel("Aujourd'hui", thirdW,
		kv{"Coût", orangeStyle.Render(fmt.Sprintf("$%.2f", todayCost))},
	)
	summaryWeek := d.panel("Semaine", thirdW,
		kv{"Coût", orangeStyle.Render(fmt.Sprintf("$%.2f", weekCost))},
	)
	summaryTotal := d.panel("Total (60j)", thirdW,
		kv{"Coût", orangeStyle.Render(fmt.Sprintf("$%.2f", s.TotalCost))},
	)
	summaryRow := lipgloss.JoinHorizontal(lipgloss.Top, summaryToday, " ", summaryWeek, " ", summaryTotal)

	var detail string
	if d.costView == "g" {
		detail = d.viewCostGraph(w, days)
	} else {
		detail = d.viewCostTable(w, days)
	}

	toggle := labelStyle.Render("  (g): graphique  •  (t): tableau  •  actif: ") +
		bigNumStyle.Render(map[string]string{"g": "graphique", "t": "tableau"}[d.costView])

	return lipgloss.JoinVertical(lipgloss.Left, summaryRow, "", detail, "", toggle)
}

func (d Dashboard) viewCostGraph(w int, days []data.DayCost) string {
	// Trouver le max pour le scaling
	maxCost := 0.0
	for _, dc := range days {
		if dc.Cost > maxCost {
			maxCost = dc.Cost
		}
	}
	if maxCost == 0 {
		maxCost = 1
	}

	graphH := 15
	graphW := w - 8 // marges du panel

	// Regrouper les jours pour tenir dans la largeur
	// Chaque barre = 1 caractère + 0 espace
	barCount := len(days)
	if barCount > graphW {
		barCount = graphW
	}

	// Si on a plus de jours que de colonnes, agréger
	type bar struct {
		cost  float64
		label string
	}
	bars := make([]bar, barCount)
	daysPerBar := max(1, len(days)/barCount)

	for i := range barCount {
		startIdx := i * daysPerBar
		endIdx := min(startIdx+daysPerBar, len(days))
		totalCost := 0.0
		var lastDate time.Time
		for j := startIdx; j < endIdx; j++ {
			totalCost += days[j].Cost
			lastDate = days[j].Date
		}
		bars[i] = bar{cost: totalCost, label: lastDate.Format("02")}
	}

	// Recalculer le max après agrégation
	maxBar := 0.0
	for _, b := range bars {
		if b.cost > maxBar {
			maxBar = b.cost
		}
	}
	if maxBar == 0 {
		maxBar = 1
	}

	blocks := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	var rows []string

	// Axe Y (légende max)
	yLabel := fmt.Sprintf("$%.1f", maxBar)
	rows = append(rows, labelStyle.Render(fmt.Sprintf("%7s ┤", yLabel)))

	// Graphique ligne par ligne (du haut vers le bas)
	for row := graphH - 1; row >= 0; row-- {
		threshold := float64(row) / float64(graphH) * maxBar
		var sb strings.Builder
		sb.WriteString("        │")
		for _, b := range bars {
			if b.cost <= 0 {
				sb.WriteString(" ")
			} else if b.cost > threshold+maxBar/float64(graphH) {
				sb.WriteString(sparkStyle.Render("█"))
			} else if b.cost > threshold {
				// Barre partielle
				frac := (b.cost - threshold) / (maxBar / float64(graphH))
				idx := int(frac * float64(len(blocks)-1))
				if idx >= len(blocks) {
					idx = len(blocks) - 1
				}
				sb.WriteString(sparkStyle.Render(blocks[idx]))
			} else {
				sb.WriteString(" ")
			}
		}
		rows = append(rows, sb.String())
	}

	// Axe X
	rows = append(rows, labelStyle.Render("   $0.0 ┤")+labelStyle.Render(strings.Repeat("─", barCount)))

	// Labels dates (début, milieu, fin)
	if len(days) > 0 {
		first := days[0].Date.Format("02 Jan")
		last := days[len(days)-1].Date.Format("02 Jan")
		mid := ""
		if len(days) > 1 {
			mid = days[len(days)/2].Date.Format("02 Jan")
		}
		labelLine := fmt.Sprintf("         %-*s", barCount, first)
		if len(mid) > 0 && barCount > 30 {
			midPos := barCount/2 - len(mid)/2
			endPos := barCount - len(last)
			labelLine = "         " + first
			if midPos > len(first)+2 {
				labelLine += strings.Repeat(" ", midPos-len(first)) + mid
				if endPos > midPos+len(mid)+2 {
					labelLine += strings.Repeat(" ", endPos-midPos-len(mid)) + last
				}
			} else if endPos > len(first)+2 {
				labelLine += strings.Repeat(" ", endPos-len(first)) + last
			}
		}
		rows = append(rows, labelStyle.Render(labelLine))
	}

	content := strings.Join(rows, "\n")
	return panelStyle.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			panelTitleStyle.Render("Coût par jour (60 derniers jours)"),
			content,
		),
	)
}

func (d Dashboard) viewCostTable(w int, days []data.DayCost) string {
	const (
		colDate     = 12
		colSessions = 10
		colMessages = 10
		colInput    = 10
		colOutput   = 10
		colCache    = 10
		colCost     = 10
	)

	// Tri
	sorted := make([]data.DayCost, len(days))
	copy(sorted, days)
	so := d.sortCosts
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		less := false
		switch so.col {
		case "date":
			less = a.Date.Before(b.Date)
		case "sessions":
			less = a.Sessions < b.Sessions
		case "messages":
			less = a.Messages < b.Messages
		case "input":
			less = a.InputTokens < b.InputTokens
		case "output":
			less = a.OutputTokens < b.OutputTokens
		case "cout":
			less = a.Cost < b.Cost
		}
		if so.asc {
			return less
		}
		return !less
	})

	var rows []string

	si := sortIndicator
	header := fmt.Sprintf("  %-*s %*s %*s %*s %*s %*s %*s",
		colDate, "(d)ate"+si(so, "date"),
		colSessions, "(s)essions"+si(so, "sessions"),
		colMessages, "(m)essages"+si(so, "messages"),
		colInput, "(i)nput"+si(so, "input"),
		colOutput, "(o)utput"+si(so, "output"),
		colCache, "Cache R",
		colCost, "(c)oût"+si(so, "cout"),
	)
	rows = append(rows, tableHeaderStyle.Render(header))
	rows = append(rows, labelStyle.Render("  "+strings.Repeat("─", colDate+colSessions+colMessages+colInput+colOutput+colCache+colCost+14)))

	totalCost := 0.0
	for _, dc := range days {
		totalCost += dc.Cost
	}

	start := d.scroll
	end := min(start+20, len(sorted))
	if start >= len(sorted) {
		start = max(0, len(sorted)-1)
		end = len(sorted)
	}

	for _, dc := range sorted[start:end] {
		costStr := fmt.Sprintf("$%.2f", dc.Cost)
		row := fmt.Sprintf("  %-*s %*d %*s %*s %*s %*s %*s",
			colDate, dc.Date.Format("02 Jan 2006"),
			colSessions, dc.Sessions,
			colMessages, fmtNum(dc.Messages),
			colInput, fmtNum(dc.InputTokens),
			colOutput, fmtNum(dc.OutputTokens),
			colCache, fmtNum(dc.CacheRead),
			colCost, costStr,
		)
		rows = append(rows, row)
	}

	if len(sorted) > 20 {
		rows = append(rows, "")
		rows = append(rows, labelStyle.Render(fmt.Sprintf("  %d/%d jours (j/k pour défiler)  •  Total: $%.2f", min(end, len(sorted)), len(sorted), totalCost)))
	} else {
		rows = append(rows, "")
		rows = append(rows, labelStyle.Render(fmt.Sprintf("  Total: $%.2f", totalCost)))
	}

	content := strings.Join(rows, "\n")
	return panelStyle.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			panelTitleStyle.Render("Coûts journaliers"),
			content,
		),
	)
}

// --- Status ---

func (d Dashboard) viewStatus(w int) string {
	left := statusStyle.Render(fmt.Sprintf("Chargé: %s", d.stats.LastUpdated.Format("15:04:05")))
	helpText := "◀ ▶/tab: onglets  •  j/k: défiler  •  r: recharger  •  auto: 60s  •  q: quitter"
	if d.tab == 4 {
		helpText = "◀ ▶/tab: onglets  •  g: graphique  •  t: tableau  •  j/k: défiler  •  q: quitter"
	}
	right := helpStyle.Render(helpText)
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}
	return left + strings.Repeat(" ", gap) + right
}

// --- Sort ---

func (d *Dashboard) handleSortKey(key string) {
	switch d.tab {
	case 1: // Sessions: p=projet, m=msgs, t=tools, o=tokens, d=durée
		if col, ok := map[string]string{
			"p": "projet", "m": "msgs", "t": "tools", "o": "tokens", "d": "duree",
		}[key]; ok {
			d.toggleSort(&d.sortSessions, col)
		}
	case 2: // Tools: o=outil, a=appels, %=pct
		if col, ok := map[string]string{
			"o": "outil", "a": "appels", "%": "pct",
		}[key]; ok {
			d.toggleSort(&d.sortTools, col)
		}
	case 3: // Projects: p=projet, s=sessions, m=messages, t=tokens, c=coût
		if col, ok := map[string]string{
			"p": "projet", "s": "sessions", "m": "messages", "t": "tokens", "c": "cout",
		}[key]; ok {
			d.toggleSort(&d.sortProjects, col)
		}
	case 4: // Coûts: g=graph, t=table + sort d=date, s=sessions, m=messages, i=input, o=output, c=coût
		if key == "g" {
			d.costView = "g"
		} else if key == "t" {
			d.costView = "t"
		} else if col, ok := map[string]string{
			"d": "date", "s": "sessions", "m": "messages", "i": "input", "o": "output", "c": "cout",
		}[key]; ok {
			d.toggleSort(&d.sortCosts, col)
		}
	}
}

func (d *Dashboard) toggleSort(so *sortOrder, col string) {
	if so.col == col {
		so.asc = !so.asc
	} else {
		so.col = col
		so.asc = false
	}
	d.scroll = 0
}

func sortIndicator(so sortOrder, col string) string {
	if so.col != col {
		return ""
	}
	if so.asc {
		return " ▲"
	}
	return " ▼"
}

// --- Helpers ---

type kv struct {
	k string
	v string
}

func (d Dashboard) panel(title string, width int, items ...kv) string {
	var rows []string
	for _, item := range items {
		switch {
		case item.k == "" && item.v == "":
			rows = append(rows, "")
		case item.v == "":
			rows = append(rows, boldValueStyle.Render(item.k))
		default:
			rows = append(rows, fmt.Sprintf("%s  %s", labelStyle.Render(item.k+":"), item.v))
		}
	}
	content := strings.Join(rows, "\n")
	return panelStyle.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			panelTitleStyle.Render(title),
			content,
		),
	)
}

func (d Dashboard) miniBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}

	var sb strings.Builder
	for i := range width {
		if i < filled {
			sb.WriteString(sparkStyle.Render("█"))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(darkGray).Render("░"))
		}
	}
	return sb.String()
}

func (d Dashboard) colorErrors(count int) string {
	s := fmt.Sprintf("%d", count)
	if count > 50 {
		return errValStyle.Render(s)
	}
	if count > 10 {
		return warnValStyle.Render(s)
	}
	return bigNumStyle.Render(s)
}

func (d Dashboard) colorErrorRate(pct float64) string {
	s := fmt.Sprintf("%.1f%%", pct)
	if pct > 5 {
		return errValStyle.Render(s)
	}
	if pct > 2 {
		return warnValStyle.Render(s)
	}
	return bigNumStyle.Render(s)
}

func (d Dashboard) colorCacheRate(pct float64) string {
	s := fmt.Sprintf("%.0f%%", pct)
	if pct > 50 {
		return bigNumStyle.Render(s)
	}
	if pct > 20 {
		return warnValStyle.Render(s)
	}
	return errValStyle.Render(s)
}

func fmtNum(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func fmtDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
