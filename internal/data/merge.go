package data

// Merge combines another Stats into this one (in-place).
// Used for aggregating stats from multiple machines.
func (s *Stats) Merge(other *Stats) {
	if other == nil {
		return
	}

	// Merge sessions (mark as remote)
	for _, sess := range other.Sessions {
		sessCopy := sess
		s.Sessions = append(s.Sessions, sessCopy)
	}

	// Merge projects
	for _, proj := range other.Projects {
		projCopy := proj
		s.Projects = append(s.Projects, projCopy)
	}

	// Merge tool usage
	for tool, count := range other.ToolUsage {
		s.ToolUsage[tool] += count
	}

	// Merge totals
	s.TotalInputTokens += other.TotalInputTokens
	s.TotalOutputTokens += other.TotalOutputTokens
	s.TotalCacheRead += other.TotalCacheRead
	s.TotalMessages += other.TotalMessages
	s.TotalToolUses += other.TotalToolUses
	s.TotalToolErrors += other.TotalToolErrors
	s.TotalSessions += other.TotalSessions

	// Merge temporals (add to today/week counts)
	s.ActiveSessions += other.ActiveSessions
	s.TodaySessions += other.TodaySessions
	s.TodayMessages += other.TodayMessages
	s.TodayTokens += other.TodayTokens
	s.WeekSessions += other.WeekSessions
	s.WeekMessages += other.WeekMessages
	s.WeekTokens += other.WeekTokens

	// Merge models
	for model, count := range other.Models {
		s.Models[model] += count
	}

	// Merge total cost
	s.TotalCost += other.TotalCost

	// Merge daily costs
	dayMap := make(map[string]*DayCost)
	for i := range s.DailyCosts {
		dc := &s.DailyCosts[i]
		key := dc.Date.Format("2006-01-02")
		dayMap[key] = dc
	}
	for _, dc := range other.DailyCosts {
		key := dc.Date.Format("2006-01-02")
		if existing, ok := dayMap[key]; ok {
			existing.InputTokens += dc.InputTokens
			existing.OutputTokens += dc.OutputTokens
			existing.CacheRead += dc.CacheRead
			existing.CacheWrite += dc.CacheWrite
			existing.Sessions += dc.Sessions
			existing.Messages += dc.Messages
			existing.Cost += dc.Cost
		} else {
			s.DailyCosts = append(s.DailyCosts, dc)
		}
	}
}
