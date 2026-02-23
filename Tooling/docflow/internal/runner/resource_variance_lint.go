package runner

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type planResourceEstimationFile struct {
	PlanEstimations []map[string]any `yaml:"plan_estimations"`
}

type developerUsageReportFile struct {
	AgentReports []map[string]any `yaml:"agent_reports"`
	Summary      map[string]any   `yaml:"summary"`
}

type resourceVarianceLintReport struct {
	Summary   resourceVarianceLintSummary   `yaml:"summary"`
	Threshold resourceVarianceLintThreshold `yaml:"thresholds"`
	Items     []resourceVarianceItem        `yaml:"items"`
	Issues    []string                      `yaml:"issues,omitempty"`
}

type resourceVarianceLintSummary struct {
	EstimationPath    string `yaml:"estimation_path"`
	UsagePath         string `yaml:"usage_path"`
	EstimatedItems    int    `yaml:"estimated_items"`
	UsageItems        int    `yaml:"usage_items"`
	ComparedItems     int    `yaml:"compared_items"`
	CoverageGaps      int    `yaml:"coverage_gaps"`
	ThresholdBreaches int    `yaml:"threshold_breaches"`
	Issues            int    `yaml:"issues"`
	Status            string `yaml:"status"`
}

type resourceVarianceLintThreshold struct {
	RequestRatio float64 `yaml:"request_ratio"`
	TokenRatio   float64 `yaml:"token_ratio"`
}

type resourceVarianceItem struct {
	PlanID               string  `yaml:"plan_id"`
	SliceID              string  `yaml:"slice_id"`
	EstimatedRequests    int64   `yaml:"estimated_requests"`
	ActualRequests       int64   `yaml:"actual_requests"`
	RequestVariance      int64   `yaml:"request_variance"`
	RequestVarianceRatio float64 `yaml:"request_variance_ratio"`
	EstimatedTotalTokens int64   `yaml:"estimated_total_tokens"`
	ActualTotalTokens    int64   `yaml:"actual_total_tokens"`
	TokenVariance        int64   `yaml:"token_variance"`
	TokenVarianceRatio   float64 `yaml:"token_variance_ratio"`
	ThresholdBreached    bool    `yaml:"threshold_breached"`
}

func ResourceVarianceLint(estimationPath, usagePath, reportPath string, requestRatioThreshold, tokenRatioThreshold float64) error {
	estimationContent, err := os.ReadFile(estimationPath)
	if err != nil {
		return fmt.Errorf("resource-variance-lint read estimation: %w", err)
	}

	usageContent, err := os.ReadFile(usagePath)
	if err != nil {
		return fmt.Errorf("resource-variance-lint read usage: %w", err)
	}

	var estimations planResourceEstimationFile
	if err := yaml.Unmarshal(estimationContent, &estimations); err != nil {
		return fmt.Errorf("resource-variance-lint parse estimation: %w", err)
	}

	var usage developerUsageReportFile
	if err := yaml.Unmarshal(usageContent, &usage); err != nil {
		return fmt.Errorf("resource-variance-lint parse usage: %w", err)
	}

	issues := make([]string, 0)
	if len(estimations.PlanEstimations) == 0 {
		issues = append(issues, "missing:plan_estimations")
	}
	if len(usage.AgentReports) == 0 {
		issues = append(issues, "missing:agent_reports")
	}

	estimationByKey := map[string]map[string]any{}
	for idx, item := range estimations.PlanEstimations {
		planID := strings.TrimSpace(stringField(item, "plan_id"))
		sliceID := strings.TrimSpace(stringField(item, "slice_id"))
		if sliceID == "" {
			sliceID = "FULL_PLAN"
		}
		if planID == "" {
			issues = append(issues, fmt.Sprintf("missing:plan_id:index:%d", idx))
			continue
		}
		key := planID + "::" + sliceID
		if _, exists := estimationByKey[key]; exists {
			issues = append(issues, fmt.Sprintf("duplicate:plan_estimation:%s", key))
			continue
		}
		estimationByKey[key] = item
	}

	usageByKey := map[string]map[string]any{}
	for idx, item := range usage.AgentReports {
		planID := strings.TrimSpace(stringField(item, "plan_id"))
		sliceID := strings.TrimSpace(stringField(item, "slice_id"))
		if sliceID == "" {
			sliceID = "FULL_PLAN"
		}
		if planID == "" {
			issues = append(issues, fmt.Sprintf("missing:usage.plan_id:index:%d", idx))
			continue
		}
		key := planID + "::" + sliceID
		if _, exists := usageByKey[key]; exists {
			issues = append(issues, fmt.Sprintf("duplicate:usage_report:%s", key))
			continue
		}
		usageByKey[key] = item
	}

	items := make([]resourceVarianceItem, 0)
	coverageGaps := 0
	thresholdBreaches := 0

	for key, estimation := range estimationByKey {
		usageItem, ok := usageByKey[key]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing:usage_coverage:%s", key))
			coverageGaps++
			continue
		}

		planID := strings.TrimSpace(stringField(estimation, "plan_id"))
		sliceID := strings.TrimSpace(stringField(estimation, "slice_id"))
		if sliceID == "" {
			sliceID = "FULL_PLAN"
		}

		estimatedRequests := intField(estimation, "estimated_requests")
		estimatedPromptTokens := intField(estimation, "estimated_prompt_tokens")
		estimatedCompletionTokens := intField(estimation, "estimated_completion_tokens")
		estimatedTotalTokens := estimatedPromptTokens + estimatedCompletionTokens

		actualRequests := intField(usageItem, "actual_requests")
		actualTotalTokens := intField(usageItem, "actual_total_tokens")
		if actualTotalTokens == 0 {
			actualTotalTokens = intField(usageItem, "actual_prompt_tokens") + intField(usageItem, "actual_completion_tokens")
		}

		requestVariance := actualRequests - estimatedRequests
		tokenVariance := actualTotalTokens - estimatedTotalTokens
		requestVarianceRatio := safeRatioFloat(requestVariance, estimatedRequests)
		tokenVarianceRatio := safeRatioFloat(tokenVariance, estimatedTotalTokens)

		breached := math.Abs(requestVarianceRatio) > requestRatioThreshold || math.Abs(tokenVarianceRatio) > tokenRatioThreshold
		if breached {
			thresholdBreaches++
		}

		items = append(items, resourceVarianceItem{
			PlanID:               planID,
			SliceID:              sliceID,
			EstimatedRequests:    estimatedRequests,
			ActualRequests:       actualRequests,
			RequestVariance:      requestVariance,
			RequestVarianceRatio: round4(requestVarianceRatio),
			EstimatedTotalTokens: estimatedTotalTokens,
			ActualTotalTokens:    actualTotalTokens,
			TokenVariance:        tokenVariance,
			TokenVarianceRatio:   round4(tokenVarianceRatio),
			ThresholdBreached:    breached,
		})
	}

	for key := range usageByKey {
		if _, ok := estimationByKey[key]; !ok {
			issues = append(issues, fmt.Sprintf("missing:estimation_coverage:%s", key))
			coverageGaps++
		}
	}

	status := "pass"
	if len(issues) > 0 || thresholdBreaches > 0 {
		status = "fail"
	}

	report := resourceVarianceLintReport{
		Summary: resourceVarianceLintSummary{
			EstimationPath:    estimationPath,
			UsagePath:         usagePath,
			EstimatedItems:    len(estimationByKey),
			UsageItems:        len(usageByKey),
			ComparedItems:     len(items),
			CoverageGaps:      coverageGaps,
			ThresholdBreaches: thresholdBreaches,
			Issues:            len(issues),
			Status:            status,
		},
		Threshold: resourceVarianceLintThreshold{
			RequestRatio: requestRatioThreshold,
			TokenRatio:   tokenRatioThreshold,
		},
		Items:  items,
		Issues: issues,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("resource-variance-lint write report: %w", err)
	}

	if status != "pass" {
		return fmt.Errorf("resource-variance-lint failed: issues=%d threshold_breaches=%d, see %s", len(issues), thresholdBreaches, reportPath)
	}

	return nil
}

func intField(payload map[string]any, key string) int64 {
	value, ok := payload[key]
	if !ok || value == nil {
		return 0
	}

	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err == nil {
			return parsed
		}
	}

	parsed, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprintf("%v", value)), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func safeRatioFloat(delta int64, baseline int64) float64 {
	if baseline == 0 {
		if delta == 0 {
			return 0
		}
		if delta > 0 {
			return 1
		}
		return -1
	}
	return float64(delta) / float64(baseline)
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
