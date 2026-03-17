package ui

import (
	"claude_monitor/internal/data"
	"claude_monitor/internal/peer"
)

// RenderScreenshot returns the ANSI content for a given tab with the provided stats.
func RenderScreenshot(tab int, stats *data.Stats, width, height int) string {
	return RenderScreenshotOpts(tab, stats, width, height, "g", nil)
}

// RenderScreenshotOpts returns the ANSI content with extra options.
func RenderScreenshotOpts(tab int, stats *data.Stats, width, height int, costView string, peerStatuses []peer.PeerStatus) string {
	d := NewDashboard()
	d.stats = stats
	d.localStats = stats
	d.loading = false
	d.width = width
	d.height = height
	d.tab = tab
	d.costView = costView
	d.myIP = "192.168.1.42"
	if peerStatuses != nil {
		d.peerStatuses = peerStatuses
	}
	return d.View().Content
}
