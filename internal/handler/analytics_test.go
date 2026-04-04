package handler

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func TestProjectAnalyticsTemplateRendersWithFloatBudget(t *testing.T) {
	t.Parallel()

	week := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	pd := pageData{
		Title:     "Test",
		CSRFToken: "tok",
		Nonce:     "nonce",
		Content: projectAnalyticsData{
			Project: &model.Project{Key: "YAIT", Name: "Test"},
			Health: &model.ProjectHealth{
				Status:            "on_track",
				ProgressPercent:   94,
				BudgetUsed:        10.5,
				BudgetTotal:       40.0,
				AvgVelocity:       3.0,
				VelocityTrend:     "stable",
				DaysRemaining:     5,
			},
			Velocity: &model.VelocityReport{
				Points: []model.VelocityPoint{
					{WeekStart: week, Points: 5, IssueCount: 3},
				},
				AvgPoints:     5,
				AvgThroughput: 3,
			},
			CycleTime: []model.CycleTimeStats{
				{IssueType: "bug", AvgDays: 2.0, Count: 2},
			},
			Estimation: &model.EstimationReport{
				AvgRatio:      1.1,
				HoursPerPoint: 2.0,
				SampleSize:    4,
			},
			TimeByType: []model.TimeByType{
				{IssueType: "feature", TotalSeconds: 3600, Percentage: 50},
			},
		},
	}

	var buf bytes.Buffer
	if err := projectAnalyticsTpl.Execute(&buf, pd); err != nil {
		t.Fatalf("execute template: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "On Track") {
		t.Fatalf("expected health banner content, got snippet: %s", truncate(out, 200))
	}
	if !strings.Contains(out, "5pts") {
		t.Fatalf("expected velocity points in output")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
