package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type dataBoundaryEvidenceFile struct {
	AgentLocalData []map[string]any `yaml:"agent_local_data"`
	MCPData        []map[string]any `yaml:"mcp_data"`
	EvidenceLinks  []map[string]any `yaml:"evidence_links"`
}

type dataBoundaryLintReport struct {
	Summary dataBoundaryLintSummary `yaml:"summary"`
	Issues  []string                `yaml:"issues,omitempty"`
}

type dataBoundaryLintSummary struct {
	EvidencePath    string `yaml:"evidence_path"`
	AgentLocalData  int    `yaml:"agent_local_data"`
	MCPData         int    `yaml:"mcp_data"`
	EvidenceLinks   int    `yaml:"evidence_links"`
	UniqueSources   int    `yaml:"unique_sources"`
	CoverageClasses int    `yaml:"coverage_classes"`
	Issues          int    `yaml:"issues"`
	Status          string `yaml:"status"`
}

func DataBoundaryLint(evidencePath, reportPath string) error {
	content, err := os.ReadFile(evidencePath)
	if err != nil {
		return fmt.Errorf("data-boundary-lint read evidence: %w", err)
	}

	var evidence dataBoundaryEvidenceFile
	if err := yaml.Unmarshal(content, &evidence); err != nil {
		return fmt.Errorf("data-boundary-lint parse evidence: %w", err)
	}

	issues := make([]string, 0)
	if len(evidence.AgentLocalData) == 0 {
		issues = append(issues, "missing:agent_local_data")
	}
	if len(evidence.MCPData) == 0 {
		issues = append(issues, "missing:mcp_data")
	}
	if len(evidence.EvidenceLinks) == 0 {
		issues = append(issues, "missing:evidence_links")
	}

	sourceSet := map[string]bool{}
	coverageClasses := 0

	for idx, item := range evidence.AgentLocalData {
		source := strings.TrimSpace(stringField(item, "source"))
		class := strings.TrimSpace(stringField(item, "class"))
		if source == "" {
			issues = append(issues, fmt.Sprintf("missing:agent_local_data.source:index:%d", idx))
		} else {
			sourceSet[source] = true
		}
		if class == "" {
			issues = append(issues, fmt.Sprintf("missing:agent_local_data.class:index:%d", idx))
		} else {
			coverageClasses++
		}
	}

	for idx, item := range evidence.MCPData {
		source := strings.TrimSpace(stringField(item, "source"))
		class := strings.TrimSpace(stringField(item, "class"))
		if source == "" {
			issues = append(issues, fmt.Sprintf("missing:mcp_data.source:index:%d", idx))
		} else {
			sourceSet[source] = true
		}
		if class == "" {
			issues = append(issues, fmt.Sprintf("missing:mcp_data.class:index:%d", idx))
		} else {
			coverageClasses++
		}
	}

	for idx, item := range evidence.EvidenceLinks {
		artifact := strings.TrimSpace(stringField(item, "artifact"))
		reason := strings.TrimSpace(stringField(item, "reason"))
		if artifact == "" {
			issues = append(issues, fmt.Sprintf("missing:evidence_links.artifact:index:%d", idx))
		}
		if reason == "" {
			issues = append(issues, fmt.Sprintf("missing:evidence_links.reason:index:%d", idx))
		}
	}

	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}

	report := dataBoundaryLintReport{
		Summary: dataBoundaryLintSummary{
			EvidencePath:    evidencePath,
			AgentLocalData:  len(evidence.AgentLocalData),
			MCPData:         len(evidence.MCPData),
			EvidenceLinks:   len(evidence.EvidenceLinks),
			UniqueSources:   len(sourceSet),
			CoverageClasses: coverageClasses,
			Issues:          len(issues),
			Status:          status,
		},
		Issues: issues,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("data-boundary-lint write report: %w", err)
	}

	if status != "pass" {
		return fmt.Errorf("data-boundary-lint failed: %d issue(s), see %s", len(issues), reportPath)
	}

	return nil
}
