package model

import "time"

type VelocityPoint struct {
	WeekStart  time.Time `json:"week_start"`
	Points     int       `json:"points"`
	IssueCount int       `json:"issue_count"`
}

type VelocityReport struct {
	ProjectID     string          `json:"project_id"`
	Points        []VelocityPoint `json:"points"`
	AvgPoints     float64         `json:"avg_points_per_week"`
	AvgThroughput float64         `json:"avg_throughput_per_week"`
}

type CycleTimeStats struct {
	IssueType    string  `json:"issue_type"`
	AvgDays      float64 `json:"avg_days"`
	MedianDays   float64 `json:"median_days"`
	Count        int     `json:"count"`
}

type EstimationAccuracy struct {
	IssueID        string  `json:"issue_id,omitempty"`
	EstimatedHours float64 `json:"estimated_hours"`
	ActualHours    float64 `json:"actual_hours"`
	Ratio          float64 `json:"ratio"`
}

type EstimationReport struct {
	ProjectID    string  `json:"project_id"`
	AvgRatio     float64 `json:"avg_ratio"`
	HoursPerPoint float64 `json:"hours_per_point"`
	SampleSize   int     `json:"sample_size"`
}

type TimeByType struct {
	IssueType    string  `json:"issue_type"`
	TotalSeconds int64   `json:"total_seconds"`
	Percentage   float64 `json:"percentage"`
}

type ProjectHealth struct {
	ProjectID       string  `json:"project_id"`
	Status          string  `json:"status"` // on_track, at_risk, behind
	ProgressPercent float64 `json:"progress_percent"`
	BudgetUsed      float64 `json:"budget_used_hours"`
	BudgetTotal     float64 `json:"budget_total_hours"`
	VelocityTrend   string  `json:"velocity_trend"` // up, down, stable
	AvgVelocity     float64 `json:"avg_velocity"`
	DaysRemaining   int     `json:"days_remaining"`
}

type TagComparison struct {
	Tag              string  `json:"tag"`
	GroupName        string  `json:"group_name"`
	ProjectCount     int     `json:"project_count"`
	BugsPer100Hours  float64 `json:"bugs_per_100_hours"`
	HoursPerPoint    float64 `json:"hours_per_point"`
	EstimationRatio  float64 `json:"estimation_ratio"`
	AvgCycleTimeDays float64 `json:"avg_cycle_time_days"`
}

type ProjectPrediction struct {
	Tags             []string `json:"tags"`
	MatchingProjects int      `json:"matching_projects"`
	HoursPerPoint    float64  `json:"hours_per_point"`
	EstimationRatio  float64  `json:"estimation_ratio"`
	RawHours         float64  `json:"raw_hours"`
	AdjustedHours    float64  `json:"adjusted_hours"`
	BugsPer100Hours  float64  `json:"bugs_per_100_hours"`
	ExpectedBugs     int      `json:"expected_bugs"`
	Confidence       string   `json:"confidence"` // low, medium, high
}
