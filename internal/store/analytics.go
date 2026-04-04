package store

import (
	"context"
	"database/sql"
	"math"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) GetVelocity(ctx context.Context, projectID string, weeks int) (*model.VelocityReport, error) {
	if weeks <= 0 {
		weeks = 8
	}

	since := time.Now().UTC().AddDate(0, 0, -weeks*7).Format("2006-01-02 15:04:05")
	rows, err := s.db.QueryContext(ctx,
		`SELECT DATE(SUBSTR(completed_at, 1, 19), 'weekday 0', '-6 days') as week_start,
		        COALESCE(SUM(story_points), 0), COUNT(*)
		 FROM issues
		 WHERE project_id = ? AND status = 'done' AND SUBSTR(completed_at, 1, 19) >= ?
		 GROUP BY week_start ORDER BY week_start`,
		projectID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report := &model.VelocityReport{ProjectID: projectID}
	var totalPoints, totalIssues int
	for rows.Next() {
		var vp model.VelocityPoint
		var weekStr string
		if err := rows.Scan(&weekStr, &vp.Points, &vp.IssueCount); err != nil {
			return nil, err
		}
		vp.WeekStart, _ = time.Parse("2006-01-02", weekStr)
		report.Points = append(report.Points, vp)
		totalPoints += vp.Points
		totalIssues += vp.IssueCount
	}

	n := len(report.Points)
	if n > 0 {
		report.AvgPoints = float64(totalPoints) / float64(n)
		report.AvgThroughput = float64(totalIssues) / float64(n)
	}
	return report, nil
}

func (s *Store) GetCycleTimeStats(ctx context.Context, projectID string) ([]model.CycleTimeStats, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT type,
		        AVG(JULIANDAY(SUBSTR(completed_at, 1, 19)) - JULIANDAY(SUBSTR(started_at, 1, 19))),
		        COUNT(*)
		 FROM issues
		 WHERE project_id = ? AND status = 'done' AND started_at IS NOT NULL AND completed_at IS NOT NULL
		 GROUP BY type`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []model.CycleTimeStats
	for rows.Next() {
		var ct model.CycleTimeStats
		if err := rows.Scan(&ct.IssueType, &ct.AvgDays, &ct.Count); err != nil {
			return nil, err
		}
		ct.AvgDays = math.Round(ct.AvgDays*10) / 10
		stats = append(stats, ct)
	}
	return stats, rows.Err()
}

func (s *Store) GetEstimationReport(ctx context.Context, projectID string) (*model.EstimationReport, error) {
	report := &model.EstimationReport{ProjectID: projectID}

	// Estimation accuracy: actual hours vs estimated hours
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), AVG(actual_hrs / estimated_hours)
		 FROM (
		     SELECT i.estimated_hours,
		            CAST(COALESCE(SUM(te.duration), 0) AS REAL) / 3600.0 as actual_hrs
		     FROM issues i
		     LEFT JOIN time_entries te ON te.issue_id = i.id AND te.duration IS NOT NULL
		     WHERE i.project_id = ? AND i.status = 'done' AND i.estimated_hours > 0
		     GROUP BY i.id, i.estimated_hours
		 )`,
		projectID,
	).Scan(&report.SampleSize, &report.AvgRatio)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Hours per point
	s.db.QueryRowContext(ctx,
		`SELECT AVG(actual_hrs / story_points)
		 FROM (
		     SELECT i.story_points,
		            CAST(COALESCE(SUM(te.duration), 0) AS REAL) / 3600.0 as actual_hrs
		     FROM issues i
		     LEFT JOIN time_entries te ON te.issue_id = i.id AND te.duration IS NOT NULL
		     WHERE i.project_id = ? AND i.status = 'done' AND i.story_points > 0
		     GROUP BY i.id, i.story_points
		 )`,
		projectID,
	).Scan(&report.HoursPerPoint)

	return report, nil
}

func (s *Store) GetTimeByType(ctx context.Context, projectID string) ([]model.TimeByType, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT i.type, COALESCE(SUM(te.duration), 0)
		 FROM issues i
		 LEFT JOIN time_entries te ON te.issue_id = i.id AND te.duration IS NOT NULL
		 WHERE i.project_id = ?
		 GROUP BY i.type`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.TimeByType
	var totalSecs int64
	for rows.Next() {
		var t model.TimeByType
		if err := rows.Scan(&t.IssueType, &t.TotalSeconds); err != nil {
			return nil, err
		}
		totalSecs += t.TotalSeconds
		results = append(results, t)
	}

	for i := range results {
		if totalSecs > 0 {
			results[i].Percentage = math.Round(float64(results[i].TotalSeconds)/float64(totalSecs)*1000) / 10
		}
	}
	return results, nil
}

func (s *Store) GetProjectHealth(ctx context.Context, projectID string) (*model.ProjectHealth, error) {
	p, err := s.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	health := &model.ProjectHealth{ProjectID: projectID}

	// Progress
	var total, done int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END)
		 FROM issues WHERE project_id = ?`, projectID,
	).Scan(&total, &done)
	if total > 0 {
		health.ProgressPercent = float64(done) / float64(total) * 100
	}

	// Budget
	var totalHours sql.NullFloat64
	s.db.QueryRowContext(ctx,
		`SELECT CAST(COALESCE(SUM(te.duration), 0) AS REAL) / 3600.0
		 FROM time_entries te
		 WHERE te.issue_id IN (SELECT id FROM issues WHERE project_id = ?)`, projectID,
	).Scan(&totalHours)
	if totalHours.Valid {
		health.BudgetUsed = math.Round(totalHours.Float64*10) / 10
	}
	if p.BudgetHours != nil {
		health.BudgetTotal = *p.BudgetHours
	}

	// Velocity trend
	velocity, _ := s.GetVelocity(ctx, projectID, 8)
	if velocity != nil && len(velocity.Points) >= 2 {
		health.AvgVelocity = velocity.AvgPoints
		recent := velocity.Points[len(velocity.Points)-1].Points
		prev := velocity.Points[len(velocity.Points)-2].Points
		if recent > prev {
			health.VelocityTrend = "up"
		} else if recent < prev {
			health.VelocityTrend = "down"
		} else {
			health.VelocityTrend = "stable"
		}
	}

	// Days remaining
	if p.TargetDate != nil {
		health.DaysRemaining = int(time.Until(*p.TargetDate).Hours() / 24)
	}

	// Status
	health.Status = "on_track"
	if health.BudgetTotal > 0 && health.BudgetUsed/health.BudgetTotal > 0.9 && health.ProgressPercent < 80 {
		health.Status = "behind"
	} else if health.VelocityTrend == "down" || (health.DaysRemaining < 14 && health.ProgressPercent < 70) {
		health.Status = "at_risk"
	}

	return health, nil
}

func (s *Store) CompareByTag(ctx context.Context, groupName string) ([]model.TagComparison, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT pt.tag, pt.group_name, COUNT(DISTINCT pt.project_id)
		 FROM project_tags pt
		 WHERE pt.group_name = ?
		 GROUP BY pt.tag, pt.group_name
		 ORDER BY COUNT(DISTINCT pt.project_id) DESC`,
		groupName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comparisons []model.TagComparison
	for rows.Next() {
		var tc model.TagComparison
		var gn sql.NullString
		if err := rows.Scan(&tc.Tag, &gn, &tc.ProjectCount); err != nil {
			return nil, err
		}
		if gn.Valid {
			tc.GroupName = gn.String
		}

		// Compute metrics for projects with this tag
		s.db.QueryRowContext(ctx,
			`SELECT
			    COALESCE(
			        CAST(SUM(CASE WHEN i.type = 'bug' THEN 1 ELSE 0 END) AS REAL) /
			        NULLIF(CAST(SUM(te_dur.total_duration) AS REAL) / 360000.0, 0),
			    0),
			    COALESCE(
			        AVG(CAST(te_dur.total_duration AS REAL) / 3600.0 / NULLIF(i.story_points, 0)),
			    0)
			 FROM issues i
			 LEFT JOIN (
			     SELECT issue_id, SUM(duration) as total_duration
			     FROM time_entries WHERE duration IS NOT NULL GROUP BY issue_id
			 ) te_dur ON te_dur.issue_id = i.id
			 WHERE i.project_id IN (SELECT project_id FROM project_tags WHERE tag = ? AND group_name = ?)
			   AND i.status = 'done'`,
			tc.Tag, groupName,
		).Scan(&tc.BugsPer100Hours, &tc.HoursPerPoint)

		// Avg cycle time
		s.db.QueryRowContext(ctx,
			`SELECT COALESCE(AVG(JULIANDAY(SUBSTR(completed_at, 1, 19)) - JULIANDAY(SUBSTR(started_at, 1, 19))), 0)
			 FROM issues
			 WHERE project_id IN (SELECT project_id FROM project_tags WHERE tag = ? AND group_name = ?)
			   AND status = 'done' AND started_at IS NOT NULL AND completed_at IS NOT NULL`,
			tc.Tag, groupName,
		).Scan(&tc.AvgCycleTimeDays)

		tc.BugsPer100Hours = math.Round(tc.BugsPer100Hours*10) / 10
		tc.HoursPerPoint = math.Round(tc.HoursPerPoint*10) / 10
		tc.AvgCycleTimeDays = math.Round(tc.AvgCycleTimeDays*10) / 10

		comparisons = append(comparisons, tc)
	}
	return comparisons, rows.Err()
}

func (s *Store) PredictNewProject(ctx context.Context, tags []string, points int) (*model.ProjectPrediction, error) {
	pred := &model.ProjectPrediction{Tags: tags}

	if len(tags) == 0 || points <= 0 {
		return pred, nil
	}

	// Find projects matching any of the given tags
	placeholders := ""
	args := make([]interface{}, len(tags))
	for i, t := range tags {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = t
	}

	var matchCount int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT project_id) FROM project_tags WHERE tag IN (`+placeholders+`)`,
		args...,
	).Scan(&matchCount)
	pred.MatchingProjects = matchCount

	if matchCount == 0 {
		pred.Confidence = "low"
		return pred, nil
	}

	// Hours per point from matching projects
	s.db.QueryRowContext(ctx,
		`SELECT COALESCE(AVG(CAST(te_dur.total_duration AS REAL) / 3600.0 / NULLIF(i.story_points, 0)), 0)
		 FROM issues i
		 LEFT JOIN (
		     SELECT issue_id, SUM(duration) as total_duration
		     FROM time_entries WHERE duration IS NOT NULL GROUP BY issue_id
		 ) te_dur ON te_dur.issue_id = i.id
		 WHERE i.project_id IN (SELECT DISTINCT project_id FROM project_tags WHERE tag IN (`+placeholders+`))
		   AND i.status = 'done' AND i.story_points > 0`,
		args...,
	).Scan(&pred.HoursPerPoint)

	// Estimation ratio
	estArgs := make([]interface{}, len(tags))
	copy(estArgs, args)
	s.db.QueryRowContext(ctx,
		`SELECT COALESCE(AVG(actual_hrs / estimated_hours), 1.0)
		 FROM (
		     SELECT i.estimated_hours,
		            CAST(COALESCE(SUM(te.duration), 0) AS REAL) / 3600.0 as actual_hrs
		     FROM issues i
		     LEFT JOIN time_entries te ON te.issue_id = i.id AND te.duration IS NOT NULL
		     WHERE i.project_id IN (SELECT DISTINCT project_id FROM project_tags WHERE tag IN (`+placeholders+`))
		       AND i.status = 'done' AND i.estimated_hours > 0
		     GROUP BY i.id, i.estimated_hours
		 )`,
		estArgs...,
	).Scan(&pred.EstimationRatio)

	// Bug rate
	bugArgs := make([]interface{}, len(tags))
	copy(bugArgs, args)
	s.db.QueryRowContext(ctx,
		`SELECT COALESCE(
		     CAST(SUM(CASE WHEN i.type = 'bug' THEN 1 ELSE 0 END) AS REAL) /
		     NULLIF(CAST(SUM(te_dur.total_duration) AS REAL) / 360000.0, 0),
		 0)
		 FROM issues i
		 LEFT JOIN (
		     SELECT issue_id, SUM(duration) as total_duration
		     FROM time_entries WHERE duration IS NOT NULL GROUP BY issue_id
		 ) te_dur ON te_dur.issue_id = i.id
		 WHERE i.project_id IN (SELECT DISTINCT project_id FROM project_tags WHERE tag IN (`+placeholders+`))
		   AND i.status = 'done'`,
		bugArgs...,
	).Scan(&pred.BugsPer100Hours)

	if pred.HoursPerPoint > 0 {
		pred.RawHours = float64(points) * pred.HoursPerPoint
	}
	if pred.EstimationRatio > 0 {
		pred.AdjustedHours = pred.RawHours * pred.EstimationRatio
	} else {
		pred.AdjustedHours = pred.RawHours
	}
	if pred.BugsPer100Hours > 0 {
		pred.ExpectedBugs = int(math.Round(pred.AdjustedHours / 100 * pred.BugsPer100Hours))
	}

	pred.HoursPerPoint = math.Round(pred.HoursPerPoint*10) / 10
	pred.RawHours = math.Round(pred.RawHours)
	pred.AdjustedHours = math.Round(pred.AdjustedHours)
	pred.EstimationRatio = math.Round(pred.EstimationRatio*10) / 10
	pred.BugsPer100Hours = math.Round(pred.BugsPer100Hours*10) / 10

	switch {
	case matchCount >= 6:
		pred.Confidence = "high"
	case matchCount >= 3:
		pred.Confidence = "medium"
	default:
		pred.Confidence = "low"
	}

	return pred, nil
}
