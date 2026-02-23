package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type planningLintReport struct {
	Summary planningLintSummary `yaml:"summary"`
	Plans   []planningLintPlan  `yaml:"plans"`
}

type planningLintSummary struct {
	RegisterPath string `yaml:"register_path"`
	PlansChecked int    `yaml:"plans_checked"`
	Violations   int    `yaml:"violations"`
	Status       string `yaml:"status"`
}

type planningLintPlan struct {
	PlanID     string   `yaml:"plan_id"`
	Violations []string `yaml:"violations"`
	Phase      string   `yaml:"phase,omitempty"`
	PlanStatus string   `yaml:"status,omitempty"`
}

func PlanningLint(registerPath, reportPath string) error {
	content, err := os.ReadFile(registerPath)
	if err != nil {
		return fmt.Errorf("planning-lint read register: %w", err)
	}

	var register planningRegisterFile
	if err := yaml.Unmarshal(content, &register); err != nil {
		return fmt.Errorf("planning-lint parse register: %w", err)
	}

	plans := register.PlanningRegister.Plans
	lintPlans := make([]planningLintPlan, 0)
	seenPlanIDs := map[string]bool{}
	violationsCount := 0

	for _, plan := range plans {
		planID := strings.TrimSpace(plan.PlanID)
		item := planningLintPlan{PlanID: planID, Phase: plan.Phase, PlanStatus: plan.Status, Violations: make([]string, 0)}

		if planID == "" {
			item.Violations = append(item.Violations, "missing:plan_id")
		} else if seenPlanIDs[planID] {
			item.Violations = append(item.Violations, "duplicate:plan_id")
		} else {
			seenPlanIDs[planID] = true
		}

		if len(plan.FeatureIDs) == 0 {
			item.Violations = append(item.Violations, "missing:feature_ids")
		}
		if len(plan.ADRIDs) == 0 {
			item.Violations = append(item.Violations, "missing:adr_ids")
		}
		if len(plan.ServiceIDs) == 0 {
			item.Violations = append(item.Violations, "missing:service_ids")
		}
		if len(plan.ComponentIDs) == 0 {
			item.Violations = append(item.Violations, "missing:component_ids")
		}
		if len(plan.ImplementationUnits) == 0 {
			item.Violations = append(item.Violations, "missing:implementation_units")
		}
		if strings.TrimSpace(plan.Phase) == "" {
			item.Violations = append(item.Violations, "missing:phase")
		}
		if strings.TrimSpace(plan.ProductReview) == "" {
			item.Violations = append(item.Violations, "missing:product_review")
		}
		if strings.TrimSpace(plan.ArchitectureReview) == "" {
			item.Violations = append(item.Violations, "missing:architecture_review")
		}
		if strings.TrimSpace(plan.EngineeringReview) == "" {
			item.Violations = append(item.Violations, "missing:engineering_review")
		}
		if strings.TrimSpace(plan.Status) == "" {
			item.Violations = append(item.Violations, "missing:status")
		}

		if plan.Status == "ready-for-build" {
			if plan.ProductReview != "approved" {
				item.Violations = append(item.Violations, "gate:product_review_not_approved")
			}
			if plan.ArchitectureReview != "approved" {
				item.Violations = append(item.Violations, "gate:architecture_review_not_approved")
			}
			if plan.EngineeringReview != "approved" {
				item.Violations = append(item.Violations, "gate:engineering_review_not_approved")
			}
		}

		violationsCount += len(item.Violations)
		lintPlans = append(lintPlans, item)
	}

	status := "pass"
	if len(plans) == 0 || violationsCount > 0 {
		status = "fail"
	}

	report := planningLintReport{
		Summary: planningLintSummary{
			RegisterPath: registerPath,
			PlansChecked: len(plans),
			Violations:   violationsCount,
			Status:       status,
		},
		Plans: lintPlans,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("planning-lint write report: %w", err)
	}

	if len(plans) == 0 {
		return fmt.Errorf("planning-lint failed: planning register has no plans")
	}
	if violationsCount > 0 {
		return fmt.Errorf("planning-lint failed: %d violations, see %s", violationsCount, reportPath)
	}

	return nil
}
