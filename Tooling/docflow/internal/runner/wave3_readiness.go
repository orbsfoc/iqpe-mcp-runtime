package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type wave3ReadinessReport struct {
	Summary wave3ReadinessSummary `yaml:"summary"`
	Entries []wave3ReadinessEntry `yaml:"entries"`
}

type wave3ReadinessSummary struct {
	CutoverProgressPath string `yaml:"cutover_progress_path"`
	ShadowLintPath      string `yaml:"shadow_lint_path"`
	PlanningLintPath    string `yaml:"planning_lint_path"`
	TotalEntries        int    `yaml:"total_entries"`
	ReadyForRemoval     int    `yaml:"ready_for_removal"`
	NotReadyEntries     int    `yaml:"not_ready_entries"`
	ShadowViolations    int    `yaml:"shadow_violations"`
	PlanningViolations  int    `yaml:"planning_violations"`
	GoNoGo              string `yaml:"go_no_go"`
	Status              string `yaml:"status"`
}

type wave3ReadinessEntry struct {
	EntryID      string   `yaml:"entry_id"`
	Source       string   `yaml:"source"`
	Target       string   `yaml:"target"`
	RemovalReady bool     `yaml:"removal_ready"`
	Blockers     []string `yaml:"blockers"`
}

type wave3ShadowLintReport struct {
	Summary struct {
		Violations int    `yaml:"violations"`
		Status     string `yaml:"status"`
	} `yaml:"summary"`
}

type wave3PlanningLintReport struct {
	Summary struct {
		Violations int    `yaml:"violations"`
		Status     string `yaml:"status"`
	} `yaml:"summary"`
}

func Wave3Readiness(cutoverProgressPath, shadowLintPath, planningLintPath, reportPath string) error {
	cutoverContent, err := os.ReadFile(cutoverProgressPath)
	if err != nil {
		return fmt.Errorf("wave3-readiness read cutover progress: %w", err)
	}

	var cutoverReport cutoverProgressReport
	if err := yaml.Unmarshal(cutoverContent, &cutoverReport); err != nil {
		return fmt.Errorf("wave3-readiness parse cutover progress: %w", err)
	}

	shadowContent, err := os.ReadFile(shadowLintPath)
	if err != nil {
		return fmt.Errorf("wave3-readiness read shadow lint: %w", err)
	}

	var shadowReport wave3ShadowLintReport
	if err := yaml.Unmarshal(shadowContent, &shadowReport); err != nil {
		return fmt.Errorf("wave3-readiness parse shadow lint: %w", err)
	}

	planningContent, err := os.ReadFile(planningLintPath)
	if err != nil {
		return fmt.Errorf("wave3-readiness read planning lint: %w", err)
	}

	var planningReport wave3PlanningLintReport
	if err := yaml.Unmarshal(planningContent, &planningReport); err != nil {
		return fmt.Errorf("wave3-readiness parse planning lint: %w", err)
	}

	readinessEntries := make([]wave3ReadinessEntry, 0, len(cutoverReport.Entries))
	notReady := 0

	for _, entry := range cutoverReport.Entries {
		blockers := make([]string, 0)
		if !entry.TechnicalPrimary {
			blockers = append(blockers, "technical_not_primary")
		}
		if !entry.TargetExists {
			blockers = append(blockers, "technical_target_missing")
		}
		shadowState := strings.TrimSpace(entry.ProductShadowState)
		if shadowState != "deprecated-ready" && shadowState != "removed" {
			blockers = append(blockers, "product_shadow_not_removal_ready")
		}
		if !entry.RemovalReady {
			blockers = append(blockers, "cutover_progress_not_removal_ready")
		}

		if len(blockers) > 0 {
			notReady++
		}

		readinessEntries = append(readinessEntries, wave3ReadinessEntry{
			EntryID:      entry.EntryID,
			Source:       entry.Source,
			Target:       entry.Target,
			RemovalReady: len(blockers) == 0,
			Blockers:     blockers,
		})
	}

	goNoGo := "go"
	status := "pass"
	if notReady > 0 || shadowReport.Summary.Violations > 0 || planningReport.Summary.Violations > 0 ||
		shadowReport.Summary.Status != "pass" || planningReport.Summary.Status != "pass" {
		goNoGo = "no-go"
		status = "fail"
	}

	report := wave3ReadinessReport{
		Summary: wave3ReadinessSummary{
			CutoverProgressPath: cutoverProgressPath,
			ShadowLintPath:      shadowLintPath,
			PlanningLintPath:    planningLintPath,
			TotalEntries:        len(cutoverReport.Entries),
			ReadyForRemoval:     cutoverReport.Summary.ReadyForRemoval,
			NotReadyEntries:     notReady,
			ShadowViolations:    shadowReport.Summary.Violations,
			PlanningViolations:  planningReport.Summary.Violations,
			GoNoGo:              goNoGo,
			Status:              status,
		},
		Entries: readinessEntries,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("wave3-readiness write report: %w", err)
	}

	if goNoGo == "no-go" {
		return fmt.Errorf("wave3-readiness no-go: %d entries not ready, %d shadow violations, %d planning violations (see %s)", notReady, shadowReport.Summary.Violations, planningReport.Summary.Violations, reportPath)
	}

	return nil
}
