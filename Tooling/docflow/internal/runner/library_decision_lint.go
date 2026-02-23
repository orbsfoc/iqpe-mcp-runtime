package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type libraryDecisionRegister struct {
	Decisions []map[string]any `yaml:"decisions"`
}

type libraryDecisionLintReport struct {
	Summary   libraryDecisionLintSummary    `yaml:"summary"`
	Decisions []libraryDecisionLintDecision `yaml:"decisions"`
}

type libraryDecisionLintSummary struct {
	RegisterPath string `yaml:"register_path"`
	Decisions    int    `yaml:"decisions"`
	Issues       int    `yaml:"issues"`
	Blocked      int    `yaml:"blocked"`
	Proposed     int    `yaml:"proposed"`
	Status       string `yaml:"status"`
}

type libraryDecisionLintDecision struct {
	DecisionID string   `yaml:"decision_id"`
	Status     string   `yaml:"approval_status"`
	Result     string   `yaml:"result"`
	Issues     []string `yaml:"issues,omitempty"`
}

func LibraryDecisionLint(registerPath, reportPath string) error {
	content, err := os.ReadFile(registerPath)
	if err != nil {
		return fmt.Errorf("library-decision-lint read register: %w", err)
	}

	var register libraryDecisionRegister
	if err := yaml.Unmarshal(content, &register); err != nil {
		return fmt.Errorf("library-decision-lint parse register: %w", err)
	}

	allowedStatuses := map[string]bool{
		"PROPOSED": true,
		"APPROVED": true,
		"REJECTED": true,
		"BLOCKED":  true,
	}

	requiredFields := []string{
		"decision_id",
		"related_plan_ids",
		"related_req_ids",
		"proposed_library",
		"use_case",
		"alternatives_considered",
		"reuse_options_evaluated",
		"risk_assessment",
		"operational_impact",
		"approval_owner",
		"approval_status",
		"authoritative_source",
	}

	reportDecisions := make([]libraryDecisionLintDecision, 0, len(register.Decisions))
	issues := 0
	blocked := 0
	proposed := 0

	if len(register.Decisions) == 0 {
		reportDecisions = append(reportDecisions, libraryDecisionLintDecision{
			DecisionID: "",
			Status:     "",
			Result:     "fail",
			Issues:     []string{"missing:decisions"},
		})
		issues++
	}

	for idx, decision := range register.Decisions {
		decisionID := strings.TrimSpace(stringField(decision, "decision_id"))
		status := strings.ToUpper(strings.TrimSpace(stringField(decision, "approval_status")))
		if status == "" {
			status = strings.ToUpper(strings.TrimSpace(stringField(decision, "outcome")))
		}

		entry := libraryDecisionLintDecision{
			DecisionID: decisionID,
			Status:     status,
			Result:     "pass",
			Issues:     make([]string, 0),
		}

		if decisionID == "" {
			entry.Issues = append(entry.Issues, fmt.Sprintf("missing:decision_id:index:%d", idx))
		}

		for _, field := range requiredFields {
			if !hasNonEmptyField(decision, field) {
				entry.Issues = append(entry.Issues, fmt.Sprintf("missing:%s", field))
			}
		}

		if status == "" {
			entry.Issues = append(entry.Issues, "missing:approval_status")
		} else if !allowedStatuses[status] {
			entry.Issues = append(entry.Issues, fmt.Sprintf("invalid:approval_status:%s", status))
		}

		if status == "BLOCKED" {
			blocked++
		}
		if status == "PROPOSED" {
			proposed++
		}

		if len(entry.Issues) > 0 {
			entry.Result = "fail"
			issues += len(entry.Issues)
		}

		reportDecisions = append(reportDecisions, entry)
	}

	overallStatus := "pass"
	if issues > 0 || blocked > 0 || proposed > 0 {
		overallStatus = "fail"
	}

	report := libraryDecisionLintReport{
		Summary: libraryDecisionLintSummary{
			RegisterPath: registerPath,
			Decisions:    len(register.Decisions),
			Issues:       issues,
			Blocked:      blocked,
			Proposed:     proposed,
			Status:       overallStatus,
		},
		Decisions: reportDecisions,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("library-decision-lint write report: %w", err)
	}

	if overallStatus != "pass" {
		return fmt.Errorf("library-decision-lint failed: issues=%d blocked=%d proposed=%d, see %s", issues, blocked, proposed, reportPath)
	}

	return nil
}

func hasNonEmptyField(payload map[string]any, key string) bool {
	value, ok := payload[key]
	if !ok || value == nil {
		return false
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case []any:
		return len(v) > 0
	case []string:
		return len(v) > 0
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", v))
		return text != "" && text != "<nil>"
	}
}
