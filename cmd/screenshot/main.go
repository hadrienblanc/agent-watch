package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"claude_monitor/internal/data"
	"claude_monitor/internal/ui"
)

// ── Terminal cell ──

type cell struct {
	ch   rune
	fg   color.RGBA
	bg   color.RGBA
	bold bool
}

// ── Default terminal colors ──

var (
	defaultFG = color.RGBA{250, 250, 250, 255} // #FAFAFA
	defaultBG = color.RGBA{30, 30, 46, 255}    // dark background
)

// ── 4-bit ANSI palette ──

var ansi4 = [16]color.RGBA{
	{0, 0, 0, 255},       // 0 black
	{205, 49, 49, 255},   // 1 red
	{13, 188, 121, 255},  // 2 green
	{229, 229, 16, 255},  // 3 yellow
	{36, 114, 200, 255},  // 4 blue
	{188, 63, 188, 255},  // 5 magenta
	{17, 168, 205, 255},  // 6 cyan
	{229, 229, 229, 255}, // 7 white
	{102, 102, 102, 255}, // 8 bright black
	{241, 76, 76, 255},   // 9 bright red
	{35, 209, 139, 255},  // 10 bright green
	{245, 245, 67, 255},  // 11 bright yellow
	{59, 142, 234, 255},  // 12 bright blue
	{214, 112, 214, 255}, // 13 bright magenta
	{41, 184, 219, 255},  // 14 bright cyan
	{255, 255, 255, 255}, // 15 bright white
}

func ansi256(n int) color.RGBA {
	if n < 16 {
		return ansi4[n]
	}
	if n < 232 {
		n -= 16
		r := (n / 36) * 51
		g := ((n % 36) / 6) * 51
		b := (n % 6) * 51
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}
	v := uint8(8 + (n-232)*10)
	return color.RGBA{v, v, v, 255}
}

// ── ANSI parser ──

type ansiState struct {
	fg, bg  color.RGBA
	bold    bool
	reverse bool
}

func parseANSI(input string) (grid [][]cell, maxWidth int) {
	state := ansiState{fg: defaultFG, bg: defaultBG}
	runes := []rune(input)
	var row []cell
	i := 0

	for i < len(runes) {
		ch := runes[i]

		if ch == '\n' {
			grid = append(grid, row)
			row = nil
			i++
			continue
		}
		if ch == '\r' {
			i++
			continue
		}
		if ch == '\t' {
			for s := 0; s < 8; s++ {
				row = append(row, cell{' ', state.fg, state.bg, state.bold})
			}
			i++
			continue
		}

		// CSI sequence: ESC [
		if ch == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			i += 2
			var params []int
			cur, has := 0, false
			for i < len(runes) && ((runes[i] >= '0' && runes[i] <= '9') || runes[i] == ';') {
				if runes[i] == ';' {
					params = append(params, cur)
					cur, has = 0, false
				} else {
					cur = cur*10 + int(runes[i]-'0')
					has = true
				}
				i++
			}
			if has {
				params = append(params, cur)
			}
			if i < len(runes) {
				if runes[i] == 'm' {
					applySGR(&state, params)
				}
				i++
			}
			continue
		}

		// Other ESC sequences (OSC, etc.) – skip
		if ch == '\x1b' {
			i++
			// Skip until ST or BEL
			for i < len(runes) && runes[i] != '\x1b' && runes[i] != '\a' && runes[i] != '\n' {
				i++
			}
			if i < len(runes) && (runes[i] == '\a' || runes[i] == '\x1b') {
				i++
				if i < len(runes) && runes[i] == '\\' {
					i++
				}
			}
			continue
		}

		fg, bg := state.fg, state.bg
		if state.reverse {
			fg, bg = bg, fg
		}
		row = append(row, cell{ch, fg, bg, state.bold})
		i++
	}
	if row != nil {
		grid = append(grid, row)
	}

	for _, r := range grid {
		if len(r) > maxWidth {
			maxWidth = len(r)
		}
	}
	return grid, maxWidth
}

func applySGR(s *ansiState, params []int) {
	if len(params) == 0 {
		params = []int{0}
	}
	i := 0
	for i < len(params) {
		p := params[i]
		switch {
		case p == 0:
			s.fg, s.bg = defaultFG, defaultBG
			s.bold, s.reverse = false, false
		case p == 1:
			s.bold = true
		case p == 7:
			s.reverse = true
		case p == 22:
			s.bold = false
		case p == 27:
			s.reverse = false
		case p == 39:
			s.fg = defaultFG
		case p == 49:
			s.bg = defaultBG
		case p >= 30 && p <= 37:
			s.fg = ansi4[p-30]
		case p >= 40 && p <= 47:
			s.bg = ansi4[p-40]
		case p >= 90 && p <= 97:
			s.fg = ansi4[p-90+8]
		case p >= 100 && p <= 107:
			s.bg = ansi4[p-100+8]
		case p == 38: // extended foreground
			if i+1 < len(params) {
				if params[i+1] == 2 && i+4 < len(params) {
					s.fg = color.RGBA{uint8(params[i+2]), uint8(params[i+3]), uint8(params[i+4]), 255}
					i += 4
				} else if params[i+1] == 5 && i+2 < len(params) {
					s.fg = ansi256(params[i+2])
					i += 2
				}
			}
		case p == 48: // extended background
			if i+1 < len(params) {
				if params[i+1] == 2 && i+4 < len(params) {
					s.bg = color.RGBA{uint8(params[i+2]), uint8(params[i+3]), uint8(params[i+4]), 255}
					i += 4
				} else if params[i+1] == 5 && i+2 < len(params) {
					s.bg = ansi256(params[i+2])
					i += 2
				}
			}
		}
		i++
	}
}

// ── PNG renderer ──

func renderPNG(grid [][]cell, maxWidth int, regFace, boldFace font.Face) *image.RGBA {
	metrics := regFace.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	cellH := ascent + descent + 2

	adv, _ := regFace.GlyphAdvance('M')
	cellW := adv.Ceil()

	// Terminal chrome dimensions
	titleBarH := 38
	padX, padY := 20, 14
	cornerR := 12

	contentW := maxWidth * cellW
	contentH := len(grid) * cellH

	imgW := contentW + 2*padX
	imgH := titleBarH + contentH + 2*padY

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))

	// Fill with transparent (for rounded corners)
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{0, 0, 0, 0}), image.Point{}, draw.Src)

	// Draw rounded rectangle background
	titleBG := color.RGBA{40, 40, 52, 255}
	drawRoundedRect(img, 0, 0, imgW, titleBarH, cornerR, titleBG, true, false)
	drawRoundedRect(img, 0, titleBarH, imgW, imgH, cornerR, defaultBG, false, true)

	// Draw window control dots
	dots := []color.RGBA{
		{255, 95, 86, 255},  // close
		{255, 189, 46, 255}, // minimize
		{39, 201, 63, 255},  // maximize
	}
	dotR := 7
	dotCY := titleBarH / 2
	for di, dc := range dots {
		dotCX := padX + 8 + di*24
		fillCircle(img, dotCX, dotCY, dotR, dc)
	}

	// Draw title text centered
	titleDrawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{190, 190, 200, 255}),
		Face: boldFace,
		Dot:  fixed.P(imgW/2-55, dotCY+ascent/2),
	}
	titleDrawer.DrawString("Claude Monitor")

	// Draw terminal content
	offX := padX
	offY := titleBarH + padY

	for row, line := range grid {
		for col, c := range line {
			x := offX + col*cellW
			y := offY + row*cellH

			// Draw cell background if not default
			if c.bg != defaultBG {
				for dy := 0; dy < cellH; dy++ {
					for dx := 0; dx < cellW; dx++ {
						px, py := x+dx, y+dy
						if px >= 0 && px < imgW && py >= 0 && py < imgH {
							img.SetRGBA(px, py, c.bg)
						}
					}
				}
			}

			if c.ch == ' ' {
				continue
			}

			face := regFace
			if c.bold {
				face = boldFace
			}

			d := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(c.fg),
				Face: face,
				Dot:  fixed.P(x, y+ascent+1),
			}
			d.DrawString(string(c.ch))
		}
	}

	// Clip rounded corners to transparent
	clipRoundedCorners(img, imgW, imgH, cornerR)

	return img
}

func drawRoundedRect(img *image.RGBA, x0, y0, x1, y1, r int, c color.RGBA, topRound, bottomRound bool) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x < 0 || x >= img.Bounds().Dx() || y < 0 || y >= img.Bounds().Dy() {
				continue
			}

			// Top corners
			if topRound && y < y0+r {
				dy := float64(y0 + r - y)
				if x < x0+r {
					dx := float64(x0 + r - x)
					if math.Sqrt(dx*dx+dy*dy) > float64(r) {
						continue
					}
				}
				if x >= x1-r {
					dx := float64(x - (x1 - r - 1))
					if math.Sqrt(dx*dx+dy*dy) > float64(r) {
						continue
					}
				}
			}

			// Bottom corners
			if bottomRound && y >= y1-r {
				dy := float64(y - (y1 - r - 1))
				if x < x0+r {
					dx := float64(x0 + r - x)
					if math.Sqrt(dx*dx+dy*dy) > float64(r) {
						continue
					}
				}
				if x >= x1-r {
					dx := float64(x - (x1 - r - 1))
					if math.Sqrt(dx*dx+dy*dy) > float64(r) {
						continue
					}
				}
			}

			img.SetRGBA(x, y, c)
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				img.SetRGBA(cx+dx, cy+dy, c)
			}
		}
	}
}

func clipRoundedCorners(img *image.RGBA, w, h, r int) {
	transparent := color.RGBA{0, 0, 0, 0}

	for y := 0; y < r; y++ {
		for x := 0; x < r; x++ {
			dy := float64(r - y)
			dx := float64(r - x)
			if math.Sqrt(dx*dx+dy*dy) > float64(r) {
				img.SetRGBA(x, y, transparent)
			}
		}
		for x := w - r; x < w; x++ {
			dy := float64(r - y)
			dx := float64(x - (w - r - 1))
			if math.Sqrt(dx*dx+dy*dy) > float64(r) {
				img.SetRGBA(x, y, transparent)
			}
		}
	}

	for y := h - r; y < h; y++ {
		for x := 0; x < r; x++ {
			dy := float64(y - (h - r - 1))
			dx := float64(r - x)
			if math.Sqrt(dx*dx+dy*dy) > float64(r) {
				img.SetRGBA(x, y, transparent)
			}
		}
		for x := w - r; x < w; x++ {
			dy := float64(y - (h - r - 1))
			dx := float64(x - (w - r - 1))
			if math.Sqrt(dx*dx+dy*dy) > float64(r) {
				img.SetRGBA(x, y, transparent)
			}
		}
	}
}

// ── Fake data ──

func richFakeStats() *data.Stats {
	now := time.Now()

	sessions := []data.Session{
		{
			ID: "abc123", Source: "claude", Slug: "refactor-auth-module", Project: "webapp-api",
			UserMessages: 45, AssistantMessages: 52,
			InputTokens: 185000, OutputTokens: 92000, CacheReadTokens: 45000,
			ToolUses:  map[string]int{"Read": 18, "Edit": 12, "Bash": 8, "Grep": 5},
			ToolErrors: 1,
			Models:    map[string]int{"claude-opus-4-6": 52},
			StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-15 * time.Minute),
			Cost: 4.82,
		},
		{
			ID: "def456", Source: "claude", Slug: "fix-payment-webhook", Project: "payment-service",
			UserMessages: 23, AssistantMessages: 31,
			InputTokens: 95000, OutputTokens: 48000, CacheReadTokens: 28000,
			ToolUses:  map[string]int{"Read": 10, "Bash": 6, "Edit": 4, "Grep": 3, "Write": 2},
			Models:    map[string]int{"claude-sonnet-4-6": 31},
			StartTime: now.Add(-90 * time.Minute), EndTime: now.Add(-30 * time.Minute),
			Cost: 1.45,
		},
		{
			ID: "ghi789", Source: "opencode", Slug: "migrate-database-v3", Project: "webapp-api",
			UserMessages: 18, AssistantMessages: 22,
			InputTokens: 72000, OutputTokens: 35000,
			ToolUses:  map[string]int{"read_file": 8, "write_file": 5, "bash": 4},
			Models:    map[string]int{"glm-5": 22},
			StartTime: now.Add(-3 * time.Hour), EndTime: now.Add(-2 * time.Hour),
			Cost: 0.38,
		},
		{
			ID: "jkl012", Source: "claude", Slug: "add-monitoring-dash", Project: "infra-tools",
			UserMessages: 35, AssistantMessages: 41,
			InputTokens: 142000, OutputTokens: 78000, CacheReadTokens: 52000,
			ToolUses:  map[string]int{"Read": 15, "Edit": 10, "Write": 8, "Bash": 6, "Glob": 4},
			ToolErrors: 2,
			Models:    map[string]int{"claude-opus-4-6": 41},
			StartTime: now.Add(-5 * time.Hour), EndTime: now.Add(-3 * time.Hour),
			Cost: 3.91,
		},
		{
			ID: "mno345", Source: "gemini", Slug: "review-pr-187", Project: "webapp-api",
			UserMessages: 8, AssistantMessages: 10,
			InputTokens: 45000, OutputTokens: 12000,
			ToolUses:  map[string]int{"read_file": 6},
			Models:    map[string]int{"gemini-3-flash-preview": 10},
			StartTime: now.Add(-45 * time.Minute), EndTime: now.Add(-20 * time.Minute),
			Cost: 0.08,
		},
		{
			ID: "pqr678", Source: "claude", Slug: "update-api-docs", Project: "webapp-api",
			UserMessages: 12, AssistantMessages: 15,
			InputTokens: 55000, OutputTokens: 32000, CacheReadTokens: 18000,
			ToolUses:  map[string]int{"Read": 8, "Edit": 5, "Grep": 3},
			Models:    map[string]int{"claude-sonnet-4-6": 15},
			StartTime: now.Add(-6 * time.Hour), EndTime: now.Add(-5 * time.Hour),
			Cost: 0.87,
		},
		{
			ID: "stu901", Source: "opencode", Slug: "optimize-queries", Project: "payment-service",
			UserMessages: 28, AssistantMessages: 33,
			InputTokens: 98000, OutputTokens: 52000,
			ToolUses:  map[string]int{"read_file": 12, "bash": 8, "write_file": 6},
			Models:    map[string]int{"glm-5": 33},
			StartTime: now.Add(-4 * time.Hour), EndTime: now.Add(-150 * time.Minute),
			Cost: 0.62,
		},
		{
			ID: "vwx234", Source: "claude", Slug: "implement-rate-limit", Project: "webapp-api",
			UserMessages: 42, AssistantMessages: 48,
			InputTokens: 168000, OutputTokens: 88000, CacheReadTokens: 62000,
			ToolUses:  map[string]int{"Read": 20, "Edit": 14, "Bash": 10, "Grep": 6, "Write": 3, "Glob": 2},
			Models:    map[string]int{"claude-opus-4-6": 48},
			StartTime: now.Add(-24 * time.Hour), EndTime: now.Add(-22 * time.Hour),
			Cost: 4.15,
		},
	}

	// Aggregate totals
	var totalInput, totalOutput, totalCache, totalMsgs, totalTools, totalToolErrors int
	toolUsage := make(map[string]int)
	models := make(map[string]int)
	projectMap := make(map[string]*data.ProjectSummary)

	for _, s := range sessions {
		totalInput += s.InputTokens
		totalOutput += s.OutputTokens
		totalCache += s.CacheReadTokens
		totalMsgs += s.UserMessages + s.AssistantMessages
		totalToolErrors += s.ToolErrors
		for tool, count := range s.ToolUses {
			toolUsage[tool] += count
			totalTools += count
		}
		for model, count := range s.Models {
			models[model] += count
		}
		proj, ok := projectMap[s.Project]
		if !ok {
			proj = &data.ProjectSummary{Name: s.Project}
			projectMap[s.Project] = proj
		}
		proj.Sessions++
		proj.Messages += s.UserMessages + s.AssistantMessages
		proj.Tokens += s.InputTokens + s.OutputTokens
		proj.Cost += s.Cost
	}

	var projects []data.ProjectSummary
	for _, p := range projectMap {
		projects = append(projects, *p)
	}

	// Daily costs for 14 days
	var dailyCosts []data.DayCost
	totalCost := 0.0
	for i := 13; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		cost := 0.50 + float64(13-i)*0.42
		dc := data.DayCost{
			Date:         d,
			Sessions:     2 + (13-i)%4,
			Messages:     20 + (13-i)*8,
			InputTokens:  30000 + (13-i)*12000,
			OutputTokens: 15000 + (13-i)*6000,
			CacheRead:    10000 + (13-i)*4000,
			Cost:         cost,
		}
		dailyCosts = append(dailyCosts, dc)
		totalCost += cost
	}

	return &data.Stats{
		TotalSessions:     len(sessions),
		TotalMessages:     totalMsgs,
		TotalInputTokens:  totalInput,
		TotalOutputTokens: totalOutput,
		TotalCacheRead:    totalCache,
		TotalToolUses:     totalTools,
		TotalToolErrors:   totalToolErrors,
		ActiveSessions:    2,
		TodaySessions:     5,
		TodayMessages:     220,
		TodayTokens:       520000,
		WeekSessions:      len(sessions),
		WeekMessages:      totalMsgs,
		WeekTokens:        totalInput + totalOutput,
		ActiveModel:       "claude-opus-4-6",
		Models:            models,
		ToolUsage:         toolUsage,
		Sessions:          sessions,
		Projects:          projects,
		DailyCosts:        dailyCosts,
		TotalCost:         totalCost,
		LastUpdated:       now,
	}
}

// ── Main ──

func main() {
	// Force true color output for lipgloss
	os.Setenv("CLICOLOR_FORCE", "1")
	os.Setenv("COLORTERM", "truecolor")

	// Load fonts
	regTT, err := opentype.Parse(gomono.TTF)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse regular font: %v\n", err)
		os.Exit(1)
	}
	boldTT, err := opentype.Parse(gomonobold.TTF)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse bold font: %v\n", err)
		os.Exit(1)
	}

	faceOpts := &opentype.FaceOptions{Size: 14, DPI: 72, Hinting: font.HintingFull}
	regFace, err := opentype.NewFace(regTT, faceOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create regular face: %v\n", err)
		os.Exit(1)
	}
	boldFace, err := opentype.NewFace(boldTT, faceOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create bold face: %v\n", err)
		os.Exit(1)
	}

	stats := richFakeStats()

	tabs := []struct {
		tab  int
		name string
	}{
		{0, "screenshot_overview.png"},
		{1, "screenshot_sessions.png"},
	}

	for _, cfg := range tabs {
		ansiStr := ui.RenderScreenshot(cfg.tab, stats, 140, 45)
		grid, maxW := parseANSI(ansiStr)
		img := renderPNG(grid, maxW, regFace, boldFace)

		f, err := os.Create(cfg.name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create %s: %v\n", cfg.name, err)
			os.Exit(1)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "encode %s: %v\n", cfg.name, err)
			os.Exit(1)
		}
		f.Close()
		fmt.Printf("Generated %s (%dx%d)\n", cfg.name, img.Bounds().Dx(), img.Bounds().Dy())
	}
}
