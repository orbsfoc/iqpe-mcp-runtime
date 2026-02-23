package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type unitMetadataFile struct {
	Units []map[string]any `yaml:"units"`
}

type humanOverviewFile struct {
	SystemOverview       string `yaml:"system_overview"`
	ProductOverview      string `yaml:"product_overview"`
	ArchitectureOverview string `yaml:"architecture_overview"`
	OperationsOverview   string `yaml:"operations_overview"`
}

type metadataOverviewLintReport struct {
	Summary metadataOverviewLintSummary `yaml:"summary"`
	Issues  []string                    `yaml:"issues,omitempty"`
}

type metadataOverviewLintSummary struct {
	MetadataPath     string `yaml:"metadata_path"`
	OverviewPath     string `yaml:"overview_path"`
	Units            int    `yaml:"units"`
	ValidUnits       int    `yaml:"valid_units"`
	OverviewSections int    `yaml:"overview_sections"`
	Issues           int    `yaml:"issues"`
	Status           string `yaml:"status"`
}

func MetadataOverviewLint(metadataPath, overviewPath, reportPath string) error {
	metadataContent, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("metadata-overview-lint read metadata: %w", err)
	}

	overviewContent, err := os.ReadFile(overviewPath)
	if err != nil {
		return fmt.Errorf("metadata-overview-lint read overview: %w", err)
	}

	var metadata unitMetadataFile
	if err := yaml.Unmarshal(metadataContent, &metadata); err != nil {
		return fmt.Errorf("metadata-overview-lint parse metadata: %w", err)
	}

	var overview humanOverviewFile
	if err := yaml.Unmarshal(overviewContent, &overview); err != nil {
		return fmt.Errorf("metadata-overview-lint parse overview: %w", err)
	}

	issues := make([]string, 0)
	if len(metadata.Units) == 0 {
		issues = append(issues, "missing:units")
	}

	allowedStatus := map[string]bool{
		"ACTIVE":     true,
		"DEPRECATED": true,
		"MIGRATING":  true,
	}

	validUnits := 0
	for idx, unit := range metadata.Units {
		unitID := strings.TrimSpace(stringField(unit, "unit_id"))
		unitType := strings.TrimSpace(stringField(unit, "unit_type"))
		if unitType == "" {
			unitType = strings.TrimSpace(stringField(unit, "kind"))
		}
		repoName := strings.TrimSpace(stringField(unit, "repo_name"))
		if repoName == "" {
			repoName = strings.TrimSpace(stringField(unit, "repo"))
		}
		repoPath := strings.TrimSpace(stringField(unit, "repo_path"))
		if repoPath == "" {
			repoPath = strings.TrimSpace(stringField(unit, "path"))
		}
		ownerTeam := strings.TrimSpace(stringField(unit, "owner_team"))
		if ownerTeam == "" {
			ownerTeam = strings.TrimSpace(stringField(unit, "owner"))
		}
		domain := strings.TrimSpace(stringField(unit, "domain"))
		buildCommand := strings.TrimSpace(stringField(unit, "build_command"))
		testCommand := strings.TrimSpace(stringField(unit, "test_command"))
		deployReference := strings.TrimSpace(stringField(unit, "deploy_reference"))
		status := strings.ToUpper(strings.TrimSpace(stringField(unit, "status")))
		lastReviewedAt := strings.TrimSpace(stringField(unit, "last_reviewed_at"))

		entryIssues := 0
		if unitID == "" {
			issues = append(issues, fmt.Sprintf("missing:unit_id:index:%d", idx))
			entryIssues++
		}
		if unitType == "" {
			issues = append(issues, fmt.Sprintf("missing:unit_type:%s", unitID))
			entryIssues++
		}
		if repoName == "" {
			issues = append(issues, fmt.Sprintf("missing:repo_name:%s", unitID))
			entryIssues++
		}
		if repoPath == "" {
			issues = append(issues, fmt.Sprintf("missing:repo_path:%s", unitID))
			entryIssues++
		}
		if ownerTeam == "" {
			issues = append(issues, fmt.Sprintf("missing:owner_team:%s", unitID))
			entryIssues++
		}
		if domain == "" {
			issues = append(issues, fmt.Sprintf("missing:domain:%s", unitID))
			entryIssues++
		}
		if !hasNonEmptyField(unit, "linked_req_ids") {
			issues = append(issues, fmt.Sprintf("missing:linked_req_ids:%s", unitID))
			entryIssues++
		}
		if !hasNonEmptyField(unit, "linked_plan_ids") {
			issues = append(issues, fmt.Sprintf("missing:linked_plan_ids:%s", unitID))
			entryIssues++
		}
		if !hasNonEmptyField(unit, "linked_adr_ids") {
			issues = append(issues, fmt.Sprintf("missing:linked_adr_ids:%s", unitID))
			entryIssues++
		}
		if !hasNonEmptyField(unit, "linked_tc_ids") {
			issues = append(issues, fmt.Sprintf("missing:linked_tc_ids:%s", unitID))
			entryIssues++
		}
		if !hasNonEmptyField(unit, "linked_test_ids") {
			issues = append(issues, fmt.Sprintf("missing:linked_test_ids:%s", unitID))
			entryIssues++
		}
		if buildCommand == "" {
			issues = append(issues, fmt.Sprintf("missing:build_command:%s", unitID))
			entryIssues++
		}
		if testCommand == "" {
			issues = append(issues, fmt.Sprintf("missing:test_command:%s", unitID))
			entryIssues++
		}
		if deployReference == "" {
			issues = append(issues, fmt.Sprintf("missing:deploy_reference:%s", unitID))
			entryIssues++
		}
		if status == "" {
			issues = append(issues, fmt.Sprintf("missing:status:%s", unitID))
			entryIssues++
		} else if !allowedStatus[status] {
			issues = append(issues, fmt.Sprintf("invalid:status:%s:%s", unitID, status))
			entryIssues++
		}
		if lastReviewedAt == "" {
			issues = append(issues, fmt.Sprintf("missing:last_reviewed_at:%s", unitID))
			entryIssues++
		}

		if entryIssues == 0 {
			validUnits++
		}
	}

	overviewSections := 0
	if strings.TrimSpace(overview.SystemOverview) == "" {
		issues = append(issues, "missing:system_overview")
	} else {
		overviewSections++
	}
	if strings.TrimSpace(overview.ProductOverview) == "" {
		issues = append(issues, "missing:product_overview")
	} else {
		overviewSections++
	}
	if strings.TrimSpace(overview.ArchitectureOverview) == "" {
		issues = append(issues, "missing:architecture_overview")
	} else {
		overviewSections++
	}
	if strings.TrimSpace(overview.OperationsOverview) == "" {
		issues = append(issues, "missing:operations_overview")
	} else {
		overviewSections++
	}

	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}

	report := metadataOverviewLintReport{
		Summary: metadataOverviewLintSummary{
			MetadataPath:     metadataPath,
			OverviewPath:     overviewPath,
			Units:            len(metadata.Units),
			ValidUnits:       validUnits,
			OverviewSections: overviewSections,
			Issues:           len(issues),
			Status:           status,
		},
		Issues: issues,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("metadata-overview-lint write report: %w", err)
	}

	if status != "pass" {
		return fmt.Errorf("metadata-overview-lint failed: %d issue(s), see %s", len(issues), reportPath)
	}

	return nil
}
