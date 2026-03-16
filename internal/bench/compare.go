package bench

// Compare produces a CompareReport showing per-task differences between two
// benchmark runs. Tasks are matched by TaskID.
func Compare(base, comp *BenchReport) *CompareReport {
	baseMap := make(map[string]BenchStatus, len(base.Results))
	for _, r := range base.Results {
		baseMap[r.TaskID] = r.Status
	}
	compMap := make(map[string]BenchStatus, len(comp.Results))
	for _, r := range comp.Results {
		compMap[r.TaskID] = r.Status
	}

	report := &CompareReport{
		Base: ReportRef{
			Pipeline: base.Pipeline,
			Mode:     base.Mode,
			RunLabel: base.RunLabel,
			Total:    base.Total,
			Passed:   base.Passed,
			PassRate: base.PassRate,
		},
		Compare: ReportRef{
			Pipeline: comp.Pipeline,
			Mode:     comp.Mode,
			RunLabel: comp.RunLabel,
			Total:    comp.Total,
			Passed:   comp.Passed,
			PassRate: comp.PassRate,
		},
	}

	// Collect all task IDs in order (base first, then compare-only).
	seen := make(map[string]bool)
	var allIDs []string
	for _, r := range base.Results {
		if !seen[r.TaskID] {
			allIDs = append(allIDs, r.TaskID)
			seen[r.TaskID] = true
		}
	}
	for _, r := range comp.Results {
		if !seen[r.TaskID] {
			allIDs = append(allIDs, r.TaskID)
			seen[r.TaskID] = true
		}
	}

	for _, id := range allIDs {
		bs, inBase := baseMap[id]
		cs, inComp := compMap[id]

		diff := TaskDiff{TaskID: id}

		switch {
		case inBase && !inComp:
			diff.Change = "only_base"
			diff.BaseStatus = bs
			report.Summary.OnlyInBase++
		case !inBase && inComp:
			diff.Change = "only_compare"
			diff.CompStatus = cs
			report.Summary.OnlyInComp++
		case bs == cs:
			diff.Change = "unchanged"
			diff.BaseStatus = bs
			diff.CompStatus = cs
			report.Summary.Unchanged++
		case cs == StatusPass && bs != StatusPass:
			diff.Change = "improved"
			diff.BaseStatus = bs
			diff.CompStatus = cs
			report.Summary.Improved++
		case bs == StatusPass && cs != StatusPass:
			diff.Change = "regressed"
			diff.BaseStatus = bs
			diff.CompStatus = cs
			report.Summary.Regressed++
		default:
			// Both non-pass but different (e.g. fail→error) — treat as unchanged.
			diff.Change = "unchanged"
			diff.BaseStatus = bs
			diff.CompStatus = cs
			report.Summary.Unchanged++
		}

		report.Diffs = append(report.Diffs, diff)
	}

	// Delta pass rate.
	report.Summary.DeltaRate = comp.PassRate - base.PassRate

	return report
}
