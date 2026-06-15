package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"GoFaux/internal/assistant"
	"GoFaux/internal/config"
	"GoFaux/internal/generator"
	"GoFaux/internal/modelhub"
)

type evaluationTarget struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type evaluationCase struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Intent          string                     `json:"intent"`
	Method          string                     `json:"method,omitempty"`
	Endpoint        string                     `json:"endpoint,omitempty"`
	Status          int                        `json:"status,omitempty"`
	FieldsText      string                     `json:"fields_text,omitempty"`
	ConstraintsText string                     `json:"constraints_text,omitempty"`
	ResponseSchema  string                     `json:"response_schema,omitempty"`
	Constraints     []assistant.ConstraintSpec `json:"constraints,omitempty"`
}

type evaluationRequest struct {
	Targets []evaluationTarget `json:"targets,omitempty"`
	Cases   []evaluationCase   `json:"cases,omitempty"`
}

type evaluationJob struct {
	ID          string              `json:"id"`
	Status      string              `json:"status"`
	StartedAt   time.Time           `json:"started_at"`
	FinishedAt  time.Time           `json:"finished_at,omitempty"`
	Total       int                 `json:"total"`
	Completed   int                 `json:"completed"`
	Results     []evaluationResult  `json:"results"`
	Summary     []evaluationSummary `json:"summary"`
	Error       string              `json:"error,omitempty"`
	LastMessage string              `json:"last_message,omitempty"`
}

type evaluationResult struct {
	Target        evaluationTarget `json:"target"`
	Case          evaluationCase   `json:"case"`
	Provider      string           `json:"provider,omitempty"`
	Model         string           `json:"model,omitempty"`
	LatencyMS     int64            `json:"latency_ms"`
	Attempts      int              `json:"attempts,omitempty"`
	ValidJSON     bool             `json:"valid_json"`
	QualityPassed bool             `json:"quality_passed"`
	ModelPassed   bool             `json:"model_passed"`
	UsedFallback  bool             `json:"used_fallback"`
	Issues        []string         `json:"issues,omitempty"`
	Messages      []string         `json:"messages,omitempty"`
	Error         string           `json:"error,omitempty"`
	Body          json.RawMessage  `json:"body,omitempty"`
}

type evaluationSummary struct {
	TargetID       string  `json:"target_id"`
	Name           string  `json:"name"`
	Provider       string  `json:"provider"`
	Total          int     `json:"total"`
	ValidJSON      int     `json:"valid_json"`
	QualityPassed  int     `json:"quality_passed"`
	ModelPassed    int     `json:"model_passed"`
	Fallbacks      int     `json:"fallbacks"`
	AverageLatency float64 `json:"average_latency_ms"`
}

func (s *Server) handleAPIEvaluations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"jobs":    s.evaluationSnapshots(),
			"targets": s.evaluationTargets(),
			"cases":   defaultEvaluationCases(),
		})
	case http.MethodPost:
		var input evaluationRequest
		if err := readJSON(r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid evaluation JSON: " + err.Error()})
			return
		}
		job, err := s.startEvaluation(input)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, job)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleAPIEvaluationJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/_gofaux/api/evaluations/")
	job := s.evaluationSnapshot(id)
	if job.ID == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "evaluation job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) startEvaluation(input evaluationRequest) (evaluationJob, error) {
	targets := input.Targets
	if len(targets) == 0 {
		targets = []evaluationTarget{{ID: "template", Name: "Template fallback", Provider: "template"}}
	}
	cases := input.Cases
	if len(cases) == 0 {
		cases = defaultEvaluationCases()
	}
	if len(targets) == 0 || len(cases) == 0 {
		return evaluationJob{}, fmt.Errorf("at least one target and one case are required")
	}
	if len(targets)*len(cases) > 100 {
		return evaluationJob{}, fmt.Errorf("evaluation is limited to 100 target/case runs")
	}

	job := evaluationJob{
		ID:        fmt.Sprintf("eval_%d", time.Now().UnixNano()),
		Status:    "running",
		StartedAt: time.Now().UTC(),
		Total:     len(targets) * len(cases),
	}
	s.storeEvaluationJob(&job)
	go s.runEvaluation(job.ID, targets, cases)
	return job, nil
}

func (s *Server) runEvaluation(id string, targets []evaluationTarget, cases []evaluationCase) {
	baseCfg := s.configSnapshot()
	for _, target := range targets {
		for _, evalCase := range cases {
			s.updateEvaluationJob(id, func(job *evaluationJob) {
				job.LastMessage = "Running " + target.Name + " on " + evalCase.Name
			})
			result := runEvaluationCase(baseCfg, target, evalCase)
			s.updateEvaluationJob(id, func(job *evaluationJob) {
				job.Results = append(job.Results, result)
				job.Completed++
				job.Summary = summarizeEvaluation(job.Results)
			})
		}
	}
	s.updateEvaluationJob(id, func(job *evaluationJob) {
		job.Status = "completed"
		job.FinishedAt = time.Now().UTC()
		job.LastMessage = "Evaluation completed"
		job.Summary = summarizeEvaluation(job.Results)
	})
}

func runEvaluationCase(baseCfg config.Config, target evaluationTarget, evalCase evaluationCase) evaluationResult {
	cfg := baseCfg
	cfg.Assistant.Provider = target.Provider
	cfg.Assistant.Model = target.Model
	cfg.Assistant.Endpoint = target.Endpoint
	input := generator.Request{
		Intent:          evalCase.Intent,
		Method:          evalCase.Method,
		Endpoint:        evalCase.Endpoint,
		Status:          evalCase.Status,
		FieldsText:      evalCase.FieldsText,
		ConstraintsText: evalCase.ConstraintsText,
		ResponseSchema:  evalCase.ResponseSchema,
		Constraints:     evalCase.Constraints,
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	result, err := generator.Generate(ctx, cfg, input)
	latency := time.Since(started).Milliseconds()
	out := evaluationResult{Target: target, Case: evalCase, LatencyMS: latency}
	if err != nil {
		out.Error = err.Error()
		out.Issues = []string{err.Error()}
		return out
	}
	out.Provider = result.Provider
	out.Model = result.Model
	out.Attempts = result.Attempts
	out.UsedFallback = result.UsedFallback
	out.Messages = result.Messages
	out.Body = result.Body
	out.ValidJSON = json.Valid(result.Body)
	issues := assistant.ValidateMockQuality(result.Body, result.Request)
	out.Issues = assistant.QualityIssueMessages(issues)
	out.QualityPassed = out.ValidJSON && len(issues) == 0
	out.ModelPassed = out.QualityPassed && !result.UsedFallback
	return out
}

func (s *Server) evaluationTargets() []evaluationTarget {
	cfg := s.configSnapshot()
	targets := []evaluationTarget{{ID: "template", Name: "Template fallback", Provider: "template"}}
	if strings.TrimSpace(cfg.Assistant.Provider) != "" && !strings.EqualFold(cfg.Assistant.Provider, "template") {
		targets = append(targets, evaluationTarget{
			ID:       "current",
			Name:     "Current provider",
			Provider: cfg.Assistant.Provider,
			Model:    cfg.Assistant.Model,
			Endpoint: cfg.Assistant.Endpoint,
		})
	}
	manager := modelhub.NewManager(cfg.ModelDir)
	downloaded, err := manager.ListDownloaded()
	if err == nil {
		for _, model := range downloaded {
			targets = append(targets, evaluationTarget{
				ID:       "managed:" + model.Path,
				Name:     model.Spec.Name,
				Provider: "managed",
				Model:    model.Path,
				Size:     model.SizeBytes,
			})
		}
	}
	return targets
}

func defaultEvaluationCases() []evaluationCase {
	return []evaluationCase{
		{
			ID:              "people-passports",
			Name:            "People passport pattern",
			Intent:          "list 5 people with name, age, nationality, hair color, and passport number with WS-xxxxxxx pattern",
			Method:          "GET",
			Endpoint:        "/people",
			Status:          http.StatusOK,
			ConstraintsText: "passportNumber: pattern WS-xxxxxxx\nage: integer 18-70",
		},
		{
			ID:              "products-range",
			Name:            "Products price and stock",
			Intent:          "list 4 products with id, name, price from 10 to 500, category, and inStock boolean",
			Method:          "GET",
			Endpoint:        "/products",
			Status:          http.StatusOK,
			ConstraintsText: "price: number 10-500\ninStock: boolean",
		},
		{
			ID:       "single-order",
			Name:     "Single order object",
			Intent:   "single order with orderNumber, status, total, createdAt ISO date, and customerName",
			Method:   "GET",
			Endpoint: "/orders/{id}",
			Status:   http.StatusOK,
		},
		{
			ID:       "error-response",
			Name:     "Unauthorized error response",
			Intent:   "unauthorized error response with code, message, traceId, and timestamp",
			Method:   "POST",
			Endpoint: "/login",
			Status:   http.StatusUnauthorized,
		},
	}
}

func summarizeEvaluation(results []evaluationResult) []evaluationSummary {
	byTarget := map[string]*evaluationSummary{}
	latencyTotals := map[string]int64{}
	for _, result := range results {
		key := result.Target.ID
		summary := byTarget[key]
		if summary == nil {
			summary = &evaluationSummary{TargetID: key, Name: result.Target.Name, Provider: result.Target.Provider}
			byTarget[key] = summary
		}
		summary.Total++
		if result.ValidJSON {
			summary.ValidJSON++
		}
		if result.QualityPassed {
			summary.QualityPassed++
		}
		if result.ModelPassed {
			summary.ModelPassed++
		}
		if result.UsedFallback {
			summary.Fallbacks++
		}
		latencyTotals[key] += result.LatencyMS
	}
	var summaries []evaluationSummary
	for key, summary := range byTarget {
		if summary.Total > 0 {
			summary.AverageLatency = float64(latencyTotals[key]) / float64(summary.Total)
		}
		summaries = append(summaries, *summary)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Name < summaries[j].Name })
	return summaries
}

func (s *Server) storeEvaluationJob(job *evaluationJob) {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()
	s.evaluations[job.ID] = job
}

func (s *Server) updateEvaluationJob(id string, update func(*evaluationJob)) {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()
	if job := s.evaluations[id]; job != nil {
		update(job)
	}
}

func (s *Server) evaluationSnapshot(id string) evaluationJob {
	s.evalMu.RLock()
	defer s.evalMu.RUnlock()
	if job := s.evaluations[id]; job != nil {
		return cloneEvaluationJob(*job)
	}
	return evaluationJob{}
}

func (s *Server) evaluationSnapshots() []evaluationJob {
	s.evalMu.RLock()
	defer s.evalMu.RUnlock()
	jobs := make([]evaluationJob, 0, len(s.evaluations))
	for _, job := range s.evaluations {
		jobs = append(jobs, cloneEvaluationJob(*job))
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].StartedAt.After(jobs[j].StartedAt) })
	return jobs
}

func cloneEvaluationJob(job evaluationJob) evaluationJob {
	job.Results = append([]evaluationResult(nil), job.Results...)
	job.Summary = append([]evaluationSummary(nil), job.Summary...)
	return job
}
