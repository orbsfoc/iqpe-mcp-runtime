package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type techPolicyFile struct {
	PhasePolicy map[string]any `yaml:"phase_policy"`
}

type techPolicyEvalReport struct {
	Summary techPolicyEvalSummary `yaml:"summary"`
	Input   techPolicyEvalInput   `yaml:"input"`
	Result  techPolicyEvalResult  `yaml:"result"`
}

type techPolicyEvalSummary struct {
	PolicyPath string `yaml:"policy_path"`
	Status     string `yaml:"status"`
}

type techPolicyEvalInput struct {
	ProjectPhase string   `yaml:"project_phase"`
	Environment  string   `yaml:"environment"`
	DecisionType string   `yaml:"decision_type"`
	Technology   string   `yaml:"technology"`
	ContextTags  []string `yaml:"context_tags,omitempty"`
}

type techPolicyEvalResult struct {
	PolicyResult       string   `yaml:"policy_result"`
	ConditionsRequired []string `yaml:"conditions_required,omitempty"`
	ExceptionRequired  bool     `yaml:"exception_required"`
	ApprovalStatus     string   `yaml:"approval_status"`
	Issues             []string `yaml:"issues,omitempty"`
}

func TechPolicyEval(policyPath, reportPath, projectPhase, environment, decisionType, technology, contextTags, approvalStatus string) error {
	content, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("tech-policy-eval read policy: %w", err)
	}

	var policy techPolicyFile
	if err := yaml.Unmarshal(content, &policy); err != nil {
		return fmt.Errorf("tech-policy-eval parse policy: %w", err)
	}
	if len(policy.PhasePolicy) == 0 {
		return fmt.Errorf("tech-policy-eval invalid policy: missing phase_policy entries")
	}

	phase := strings.ToUpper(strings.TrimSpace(projectPhase))
	env := strings.ToUpper(strings.TrimSpace(environment))
	dType := strings.ToUpper(strings.TrimSpace(decisionType))
	tech := strings.ToLower(strings.TrimSpace(technology))
	approval := strings.ToUpper(strings.TrimSpace(approvalStatus))
	if approval == "" {
		approval = "PROPOSED"
	}

	tags := normalizeCSV(contextTags)
	tagSet := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagSet[tag] = true
	}

	result := techPolicyEvalResult{
		PolicyResult:       "DISALLOWED",
		ConditionsRequired: make([]string, 0),
		ExceptionRequired:  false,
		ApprovalStatus:     approval,
		Issues:             make([]string, 0),
	}

	switch {
	case dType == "RUNTIME" && (tech == "java17" || tech == "java-17"):
		if phase == "LEGACY_SUPPORT" && tagSet["legacy-java17-estate"] {
			result.PolicyResult = "ALLOWED"
		} else {
			result.PolicyResult = "DISALLOWED"
			result.ExceptionRequired = true
			result.ConditionsRequired = append(result.ConditionsRequired, "approved-java17-exception")
		}
	case dType == "RUNTIME" && (tech == "java21" || tech == "java-21"):
		if phase == "LEGACY_SUPPORT" {
			result.PolicyResult = "ALLOWED_WITH_CONDITION"
			result.ConditionsRequired = append(result.ConditionsRequired, "legacy-migration-plan-recorded")
		} else {
			result.PolicyResult = "ALLOWED"
		}
	case dType == "ORCHESTRATION" && (tech == "docker-compose" || tech == "docker compose"):
		if (phase == "POC" || phase == "MVP") && (env == "LOCAL" || env == "DEMO") {
			result.PolicyResult = "ALLOWED_WITH_CONDITION"
			result.ConditionsRequired = append(result.ConditionsRequired, "non-production-use-only")
		} else {
			result.PolicyResult = "DISALLOWED"
			result.ExceptionRequired = true
			result.ConditionsRequired = append(result.ConditionsRequired, "production-or-scale-orchestration-exception")
		}
	case (dType == "RUNTIME" || dType == "FRAMEWORK" || dType == "INFRA") && tech == "golang":
		result.PolicyResult = "ALLOWED"
	default:
		result.PolicyResult = "ALLOWED_WITH_CONDITION"
		result.ConditionsRequired = append(result.ConditionsRequired, "manual-policy-review")
	}

	if result.PolicyResult == "DISALLOWED" && approval != "APPROVED" {
		result.Issues = append(result.Issues, "disallowed-without-approved-exception")
	}

	status := "pass"
	if len(result.Issues) > 0 {
		status = "fail"
	}

	report := techPolicyEvalReport{
		Summary: techPolicyEvalSummary{
			PolicyPath: policyPath,
			Status:     status,
		},
		Input: techPolicyEvalInput{
			ProjectPhase: phase,
			Environment:  env,
			DecisionType: dType,
			Technology:   tech,
			ContextTags:  tags,
		},
		Result: result,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("tech-policy-eval write report: %w", err)
	}

	if status != "pass" {
		return fmt.Errorf("tech-policy-eval failed: %d issue(s), see %s", len(result.Issues), reportPath)
	}

	return nil
}

func normalizeCSV(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return []string{}
	}

	parts := strings.Split(trimmed, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.ToLower(strings.TrimSpace(part))
		if token != "" {
			result = append(result, token)
		}
	}
	return result
}
