package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type architectureOpsBoundaryFile struct {
	ArchitectureArtifacts []string            `yaml:"architecture_artifacts"`
	OperationsArtifacts   []string            `yaml:"operations_artifacts"`
	LinkMap               map[string][]string `yaml:"architecture_to_operations"`
}

type architectureOpsLintReport struct {
	Summary architectureOpsLintSummary `yaml:"summary"`
	Issues  []string                   `yaml:"issues,omitempty"`
}

type architectureOpsLintSummary struct {
	BoundaryPath          string `yaml:"boundary_path"`
	ArchitectureArtifacts int    `yaml:"architecture_artifacts"`
	OperationsArtifacts   int    `yaml:"operations_artifacts"`
	LinkedArchitecture    int    `yaml:"linked_architecture"`
	UnlinkedArchitecture  int    `yaml:"unlinked_architecture"`
	Issues                int    `yaml:"issues"`
	Status                string `yaml:"status"`
}

func ArchitectureOpsLint(boundaryPath, reportPath string) error {
	content, err := os.ReadFile(boundaryPath)
	if err != nil {
		return fmt.Errorf("architecture-ops-lint read boundary: %w", err)
	}

	var boundary architectureOpsBoundaryFile
	if err := yaml.Unmarshal(content, &boundary); err != nil {
		return fmt.Errorf("architecture-ops-lint parse boundary: %w", err)
	}

	issues := make([]string, 0)
	if len(boundary.ArchitectureArtifacts) == 0 {
		issues = append(issues, "missing:architecture_artifacts")
	}
	if len(boundary.OperationsArtifacts) == 0 {
		issues = append(issues, "missing:operations_artifacts")
	}

	architectureSet := map[string]bool{}
	operationsSet := map[string]bool{}
	linkedArchitecture := 0

	for _, item := range boundary.ArchitectureArtifacts {
		value := strings.TrimSpace(item)
		if value == "" {
			issues = append(issues, "missing:architecture_artifact_id")
			continue
		}
		if architectureSet[value] {
			issues = append(issues, fmt.Sprintf("duplicate:architecture_artifact:%s", value))
			continue
		}
		architectureSet[value] = true
	}

	for _, item := range boundary.OperationsArtifacts {
		value := strings.TrimSpace(item)
		if value == "" {
			issues = append(issues, "missing:operations_artifact_id")
			continue
		}
		if operationsSet[value] {
			issues = append(issues, fmt.Sprintf("duplicate:operations_artifact:%s", value))
			continue
		}
		operationsSet[value] = true
	}

	for artifactID := range architectureSet {
		if operationsSet[artifactID] {
			issues = append(issues, fmt.Sprintf("invalid:artifact_in_both_lanes:%s", artifactID))
		}
	}

	for architectureID := range architectureSet {
		links := boundary.LinkMap[architectureID]
		if len(links) == 0 {
			issues = append(issues, fmt.Sprintf("missing:architecture_link:%s", architectureID))
			continue
		}

		hasValidLink := false
		for _, link := range links {
			linkedID := strings.TrimSpace(link)
			if linkedID == "" {
				issues = append(issues, fmt.Sprintf("missing:linked_operations_artifact:%s", architectureID))
				continue
			}
			if !operationsSet[linkedID] {
				issues = append(issues, fmt.Sprintf("unknown:operations_artifact:%s->%s", architectureID, linkedID))
				continue
			}
			hasValidLink = true
		}

		if hasValidLink {
			linkedArchitecture++
		}
	}

	for architectureID := range boundary.LinkMap {
		if !architectureSet[architectureID] {
			issues = append(issues, fmt.Sprintf("unknown:architecture_artifact:%s", architectureID))
		}
	}

	unlinkedArchitecture := 0
	if len(architectureSet) >= linkedArchitecture {
		unlinkedArchitecture = len(architectureSet) - linkedArchitecture
	}

	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}

	report := architectureOpsLintReport{
		Summary: architectureOpsLintSummary{
			BoundaryPath:          boundaryPath,
			ArchitectureArtifacts: len(boundary.ArchitectureArtifacts),
			OperationsArtifacts:   len(boundary.OperationsArtifacts),
			LinkedArchitecture:    linkedArchitecture,
			UnlinkedArchitecture:  unlinkedArchitecture,
			Issues:                len(issues),
			Status:                status,
		},
		Issues: issues,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("architecture-ops-lint write report: %w", err)
	}

	if status != "pass" {
		return fmt.Errorf("architecture-ops-lint failed: %d issue(s), see %s", len(issues), reportPath)
	}

	return nil
}
