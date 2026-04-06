package handler

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"yaitracker.com/loweryaustin/internal/model"
)

// ensure render.go imports are available for funcMap
var _ = contains

type projectAnalyticsData struct {
	Project    *model.Project
	Health     *model.ProjectHealth
	Velocity   *model.VelocityReport
	CycleTime  []model.CycleTimeStats
	Estimation *model.EstimationReport
	TimeByType []model.TimeByType
}

var projectAnalyticsTpl = template.Must(template.New("project-analytics").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
{{$p := .Content.Project}}
{{$h := .Content.Health}}
<div class="mb-4">
    <span class="font-mono text-xs text-slate-500">{{$p.Key}}</span>
    <span class="mx-1 text-slate-300">&gt;</span>
    <span>Analytics</span>
</div>
<div class="flex gap-2 border-b border-slate-200 mb-6">
    <a href="/projects/{{$p.Key}}/board" class="px-4 py-2 text-sm hover:text-blue-600">Board</a>
    <a href="/projects/{{$p.Key}}/issues" class="px-4 py-2 text-sm hover:text-blue-600">Issues</a>
    <a href="/projects/{{$p.Key}}/analytics" class="px-4 py-2 text-sm border-b-2 border-blue-600 text-blue-600 font-medium">Analytics</a>
    <a href="/projects/{{$p.Key}}/settings" class="px-4 py-2 text-sm hover:text-blue-600">Settings</a>
</div>

<!-- Health Banner -->
<div class="rounded-lg p-4 mb-6 {{if eq $h.Status "on_track"}}bg-emerald-50 border border-emerald-200{{else if eq $h.Status "at_risk"}}bg-amber-50 border border-amber-200{{else}}bg-red-50 border border-red-200{{end}}">
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-4">
            <span class="text-lg font-semibold {{if eq $h.Status "on_track"}}text-emerald-700{{else if eq $h.Status "at_risk"}}text-amber-700{{else}}text-red-700{{end}}">
                {{if eq $h.Status "on_track"}}On Track{{else if eq $h.Status "at_risk"}}At Risk{{else}}Behind{{end}}
            </span>
            <span class="text-sm">{{printf "%.0f" $h.ProgressPercent}}% done</span>
        </div>
        <div class="flex items-center gap-6 text-sm">
            {{if gtF $h.BudgetTotal 0.0}}
            <span>Budget: {{printf "%.0f" $h.BudgetUsed}} / {{printf "%.0f" $h.BudgetTotal}}h</span>
            {{end}}
            <span>Velocity: {{printf "%.1f" $h.AvgVelocity}} pts/wk
                {{if eq $h.VelocityTrend "up"}}↑{{else if eq $h.VelocityTrend "down"}}↓{{else}}→{{end}}
            </span>
            {{if gtI $h.DaysRemaining 0}}
            <span>{{$h.DaysRemaining}} days left</span>
            {{end}}
        </div>
    </div>
</div>

<div class="grid grid-cols-2 gap-6">
    <!-- Velocity -->
    <div class="bg-white rounded-lg border border-slate-200 p-4">
        <h3 class="text-sm font-medium text-slate-500 uppercase tracking-wide mb-3">Velocity</h3>
        {{if .Content.Velocity.Points}}
        <div class="space-y-2">
            {{range .Content.Velocity.Points}}
            <div class="flex items-center gap-3 text-sm">
                <span class="text-xs text-slate-400 w-20">{{.WeekStart.Format "Jan 2"}}</span>
                <div class="flex-1 bg-slate-100 rounded-full h-4">
                    <div class="bg-blue-500 h-4 rounded-full text-xs text-white flex items-center justify-center"
                         style="width: {{if gtI .Points 0}}{{min (mul .Points 10) 100}}{{else}}0{{end}}%">
                        {{.Points}}pts
                    </div>
                </div>
                <span class="text-xs text-slate-400">{{.IssueCount}} issues</span>
            </div>
            {{end}}
        </div>
        <p class="mt-3 text-sm text-slate-600">Avg: {{printf "%.1f" .Content.Velocity.AvgPoints}} pts/week, {{printf "%.1f" .Content.Velocity.AvgThroughput}} issues/week</p>
        {{else}}
        <p class="text-sm text-slate-400">Not enough data. Complete a few issues to see velocity.</p>
        {{end}}
    </div>

    <!-- Cycle Time -->
    <div class="bg-white rounded-lg border border-slate-200 p-4">
        <h3 class="text-sm font-medium text-slate-500 uppercase tracking-wide mb-3">Cycle Time by Type</h3>
        {{if .Content.CycleTime}}
        <div class="space-y-3">
            {{range .Content.CycleTime}}
            <div class="flex items-center justify-between text-sm">
                <span class="capitalize">{{.IssueType}}</span>
                <div class="flex items-center gap-3">
                    <span class="font-medium">{{printf "%.1f" .AvgDays}} days</span>
                    <span class="text-xs text-slate-400">({{.Count}} issues)</span>
                </div>
            </div>
            {{end}}
        </div>
        {{else}}
        <p class="text-sm text-slate-400">Complete some issues to see cycle time data.</p>
        {{end}}
    </div>

    <!-- Estimation Accuracy -->
    <div class="bg-white rounded-lg border border-slate-200 p-4">
        <h3 class="text-sm font-medium text-slate-500 uppercase tracking-wide mb-3">Estimation Accuracy</h3>
        {{$e := .Content.Estimation}}
        {{if gtI $e.SampleSize 0}}
        <div class="space-y-2 text-sm">
            <div class="flex justify-between"><span>Avg ratio (actual/estimated):</span><span class="font-medium">{{printf "%.1fx" $e.AvgRatio}}</span></div>
            <div class="flex justify-between"><span>Hours per point:</span><span class="font-medium">{{printf "%.1f" $e.HoursPerPoint}}h</span></div>
            <div class="flex justify-between"><span>Sample size:</span><span>{{$e.SampleSize}} issues</span></div>
        </div>
        {{else}}
        <p class="text-sm text-slate-400">Add estimates and track time to see accuracy data.</p>
        {{end}}
    </div>

    <!-- Time by Type -->
    <div class="bg-white rounded-lg border border-slate-200 p-4">
        <h3 class="text-sm font-medium text-slate-500 uppercase tracking-wide mb-3">Time by Type</h3>
        {{if .Content.TimeByType}}
        <div class="space-y-2">
            {{range .Content.TimeByType}}
            <div class="flex items-center gap-3 text-sm">
                <span class="capitalize w-24">{{.IssueType}}</span>
                <div class="flex-1 bg-slate-100 rounded-full h-4">
                    <div class="bg-purple-500 h-4 rounded-full" style="width: {{printf "%.0f" .Percentage}}%"></div>
                </div>
                <span class="text-xs text-slate-400 w-16 text-right">{{printf "%.0f" .Percentage}}%</span>
            </div>
            {{end}}
        </div>
        {{else}}
        <p class="text-sm text-slate-400">Track time to see distribution.</p>
        {{end}}
    </div>
</div>
{{end}}`))

func (h *Handler) GetProjectAnalytics(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	health, err := h.Store.GetProjectHealth(r.Context(), project.ID)
	if err != nil {
		log.Printf("get project health: %v", err)
	}
	velocity, err := h.Store.GetVelocity(r.Context(), project.ID, 8)
	if err != nil {
		log.Printf("get velocity: %v", err)
	}
	cycleTime, err := h.Store.GetCycleTimeStats(r.Context(), project.ID)
	if err != nil {
		log.Printf("get cycle time stats: %v", err)
	}
	estimation, err := h.Store.GetEstimationReport(r.Context(), project.ID)
	if err != nil {
		log.Printf("get estimation report: %v", err)
	}
	timeByType, err := h.Store.GetTimeByType(r.Context(), project.ID)
	if err != nil {
		log.Printf("get time by type: %v", err)
	}

	if health == nil {
		health = &model.ProjectHealth{ProjectID: project.ID, Status: "on_track"}
	}
	if velocity == nil {
		velocity = &model.VelocityReport{ProjectID: project.ID}
	}
	if estimation == nil {
		estimation = &model.EstimationReport{ProjectID: project.ID}
	}

	pd := h.newPageData(r, project.Name+" - Analytics", projectAnalyticsData{
		Project:    project,
		Health:     health,
		Velocity:   velocity,
		CycleTime:  cycleTime,
		Estimation: estimation,
		TimeByType: timeByType,
	})
	pd.ProjectKey = key
	pd.ActiveTab = "analytics"
	h.renderApp(w, "project-analytics", projectAnalyticsTpl, pd)
}

// Cross-project comparison

type compareData struct {
	GroupBy     string
	Groups      []string
	Comparisons []model.TagComparison
}

var compareTpl = template.Must(template.New("compare").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
<h1 class="text-xl font-semibold text-slate-900 mb-6">Cross-Project Comparison</h1>
<div class="flex items-center gap-4 mb-6">
    <label class="text-sm font-medium text-slate-700">Group by:</label>
    <select hx-get="/analytics/compare" hx-target="main" hx-swap="innerHTML" hx-push-url="true" name="group_by"
            class="px-3 py-1.5 border border-slate-300 rounded-md text-sm">
        <option value="">Select group...</option>
        {{range .Content.Groups}}
        <option value="{{.}}" {{if eq . $.Content.GroupBy}}selected{{end}}>{{.}}</option>
        {{end}}
    </select>
</div>
{{if .Content.Comparisons}}
<div class="bg-white rounded-lg border border-slate-200 overflow-hidden">
    <table class="w-full text-sm">
        <thead class="bg-slate-50 border-b border-slate-200">
            <tr>
                <th class="text-left px-4 py-2 font-medium text-slate-500">Group</th>
                <th class="text-center px-4 py-2 font-medium text-slate-500">Projects</th>
                <th class="text-center px-4 py-2 font-medium text-slate-500">Bugs/100h</th>
                <th class="text-center px-4 py-2 font-medium text-slate-500">Hrs/Point</th>
                <th class="text-center px-4 py-2 font-medium text-slate-500">Est. Ratio</th>
                <th class="text-center px-4 py-2 font-medium text-slate-500">Avg Cycle</th>
            </tr>
        </thead>
        <tbody>
            {{range .Content.Comparisons}}
            <tr class="border-b border-slate-100">
                <td class="px-4 py-2 font-medium">{{.Tag}}</td>
                <td class="text-center px-4 py-2">{{.ProjectCount}}</td>
                <td class="text-center px-4 py-2">{{printf "%.1f" .BugsPer100Hours}}</td>
                <td class="text-center px-4 py-2">{{printf "%.1f" .HoursPerPoint}}</td>
                <td class="text-center px-4 py-2">{{printf "%.1fx" .EstimationRatio}}</td>
                <td class="text-center px-4 py-2">{{printf "%.1f" .AvgCycleTimeDays}} days</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{else if .Content.GroupBy}}
<p class="text-slate-400">No data for this grouping yet.</p>
{{else}}
<p class="text-slate-400">Select a tag group to compare projects.</p>
{{end}}
{{end}}`))

func (h *Handler) GetCompare(w http.ResponseWriter, r *http.Request) {
	groupBy := h.queryParam(r, "group_by")
	groups, err := h.Store.ListTagGroups(r.Context())
	if err != nil {
		log.Printf("list tag groups: %v", err)
	}

	var comparisons []model.TagComparison
	if groupBy != "" {
		comparisons, err = h.Store.CompareByTag(r.Context(), groupBy)
		if err != nil {
			log.Printf("compare by tag: %v", err)
		}
	}

	pd := h.newPageData(r, "Compare", compareData{
		GroupBy:     groupBy,
		Groups:      groups,
		Comparisons: comparisons,
	})
	pd.ActiveNav = "compare"
	h.renderApp(w, "compare", compareTpl, pd)
}

// Predict

type predictData struct {
	Tags       string
	Points     int
	Prediction *model.ProjectPrediction
}

var predictTpl = template.Must(template.New("predict").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
<h1 class="text-xl font-semibold text-slate-900 mb-6">Estimate New Project</h1>
<form method="GET" action="/analytics/predict" class="bg-white rounded-lg border border-slate-200 p-6 max-w-xl space-y-4 mb-6">
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Tags (describe your new project)</label>
        <input type="text" name="tags" value="{{.Content.Tags}}" placeholder="go, web, htmx (comma separated)"
               class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
    </div>
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Estimated Story Points</label>
        <input type="number" name="points" value="{{if gtI .Content.Points 0}}{{.Content.Points}}{{end}}" min="1" placeholder="120"
               class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
    </div>
    <button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium text-sm">Calculate Prediction</button>
</form>

{{if .Content.Prediction}}
{{$pred := .Content.Prediction}}
<div class="bg-white rounded-lg border border-slate-200 p-6 max-w-xl">
    <h2 class="font-medium text-slate-900 mb-4">Prediction</h2>
    <p class="text-sm text-slate-600 mb-4">Based on <strong>{{$pred.MatchingProjects}}</strong> similar projects.</p>
    <div class="space-y-2 text-sm">
        <div class="flex justify-between"><span>Avg velocity:</span><span class="font-medium">{{printf "%.1f" $pred.HoursPerPoint}} hrs/point</span></div>
        <div class="flex justify-between"><span>Expected hours (raw):</span><span class="font-medium">{{printf "%.0f" $pred.RawHours}}h</span></div>
        <div class="flex justify-between"><span>Adjusted hours (x{{printf "%.1f" $pred.EstimationRatio}}):</span><span class="font-medium">{{printf "%.0f" $pred.AdjustedHours}}h</span></div>
        <div class="flex justify-between"><span>Bug rate:</span><span class="font-medium">{{printf "%.1f" $pred.BugsPer100Hours}} per 100h</span></div>
        <div class="flex justify-between"><span>Expected bugs:</span><span class="font-medium">~{{$pred.ExpectedBugs}}</span></div>
        <div class="flex justify-between mt-4 pt-4 border-t border-slate-100">
            <span>Confidence:</span>
            <span class="px-2 py-0.5 rounded text-xs font-medium
                {{if eq $pred.Confidence "high"}}bg-emerald-100 text-emerald-700
                {{else if eq $pred.Confidence "medium"}}bg-amber-100 text-amber-700
                {{else}}bg-slate-100 text-slate-600{{end}}">
                {{$pred.Confidence}}
            </span>
        </div>
    </div>
</div>
{{end}}
{{end}}`))

func (h *Handler) GetPredict(w http.ResponseWriter, r *http.Request) {
	tagsStr := h.queryParam(r, "tags")
	pointsStr := h.queryParam(r, "points")
	points, _ := strconv.Atoi(pointsStr) //nolint:errcheck // returns 0 on invalid input

	var prediction *model.ProjectPrediction
	if tagsStr != "" && points > 0 {
		var tags []string
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(strings.ToLower(t))
			if t != "" {
				tags = append(tags, t)
			}
		}
		var err error
		prediction, err = h.Store.PredictNewProject(r.Context(), tags, points)
		if err != nil {
			log.Printf("predict new project: %v", err)
		}
	}

	pd := h.newPageData(r, "Predict", predictData{
		Tags:       tagsStr,
		Points:     points,
		Prediction: prediction,
	})
	pd.ActiveNav = "compare"
	h.renderApp(w, "predict", predictTpl, pd)
}
