package ui

import "claude_monitor/internal/data"

// RenderScreenshot returns the ANSI content for a given tab with the provided stats.
func RenderScreenshot(tab int, stats *data.Stats, width, height int) string {
	d := NewDashboard()
	d.stats = stats
	d.localStats = stats
	d.loading = false
	d.width = width
	d.height = height
	d.tab = tab
	d.costView = "g"
	return d.View().Content
}
