package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type wave4TopologyFile struct {
	MultiRepoTopology wave4Topology `yaml:"multi_repo_topology"`
}

type wave4Topology struct {
	Repositories []wave4Repository `yaml:"repositories"`
}

type wave4Repository struct {
	RepoID string `yaml:"repo_id"`
	Domain string `yaml:"domain"`
	Role   string `yaml:"role"`
}

type wave4BootstrapReport struct {
	Summary wave4BootstrapSummary `yaml:"summary"`
	Checks  wave4BootstrapChecks  `yaml:"checks"`
	Actions []string              `yaml:"actions"`
	Notes   []string              `yaml:"notes"`
}

type wave4BootstrapSummary struct {
	CheckpointID      string `yaml:"checkpoint_id"`
	CheckpointDate    string `yaml:"checkpoint_date"`
	ClosureReportPath string `yaml:"closure_report_path"`
	TopologyPath      string `yaml:"topology_path"`
	BootstrapStatus   string `yaml:"bootstrap_status"`
	Repositories      int    `yaml:"repositories"`
	TechnicalDocPaths int    `yaml:"technical_doc_paths"`
}

type wave4BootstrapChecks struct {
	Wave3ClosureClosed        bool `yaml:"wave3_closure_closed"`
	RequiredRepoDomainsFound  bool `yaml:"required_repo_domains_found"`
	RequiredRepoIDsFound      bool `yaml:"required_repo_ids_found"`
	TechnicalDocsRootPresent  bool `yaml:"technical_docs_root_present"`
	ArchitecturePathPresent   bool `yaml:"architecture_path_present"`
	ImplementationPathPresent bool `yaml:"implementation_path_present"`
	OperationsPathPresent     bool `yaml:"operations_path_present"`
}

func Wave4Bootstrap(closurePath, topologyPath, reportPath, docPath, checkpointDate string) error {
	closureContent, err := os.ReadFile(closurePath)
	if err != nil {
		return fmt.Errorf("wave4-bootstrap read closure report: %w", err)
	}

	var closure wave3ClosureReport
	if err := yaml.Unmarshal(closureContent, &closure); err != nil {
		return fmt.Errorf("wave4-bootstrap parse closure report: %w", err)
	}

	topologyContent, err := os.ReadFile(topologyPath)
	if err != nil {
		return fmt.Errorf("wave4-bootstrap read topology: %w", err)
	}

	var topologyFile wave4TopologyFile
	if err := yaml.Unmarshal(topologyContent, &topologyFile); err != nil {
		return fmt.Errorf("wave4-bootstrap parse topology: %w", err)
	}

	repoIDs := map[string]bool{}
	domains := map[string]bool{}
	for _, repository := range topologyFile.MultiRepoTopology.Repositories {
		repoIDs[strings.TrimSpace(repository.RepoID)] = true
		domains[strings.TrimSpace(repository.Domain)] = true
	}

	requiredRepoIDs := []string{
		"REPO-PRODUCT-DOCS",
		"REPO-ARCH-DOCS",
		"REPO-PLANNING",
		"REPO-SERVICE-CODE",
	}
	requiredDomains := []string{"product", "architecture", "planning", "implementation"}

	hasRepoIDs := true
	for _, repoID := range requiredRepoIDs {
		if !repoIDs[repoID] {
			hasRepoIDs = false
			break
		}
	}

	hasDomains := true
	for _, domain := range requiredDomains {
		if !domains[domain] {
			hasDomains = false
			break
		}
	}

	technicalRoot := filepath.Clean(filepath.Join("../../", filepath.FromSlash("Docs/RefactoredTechnicalDocs")))
	architecturePath := filepath.Join(technicalRoot, "00-architecture")
	implementationPath := filepath.Join(technicalRoot, "01-implementation")
	operationsPath := filepath.Join(technicalRoot, "02-operations")

	checks := wave4BootstrapChecks{
		Wave3ClosureClosed:        strings.TrimSpace(closure.Summary.ClosureStatus) == "closed",
		RequiredRepoDomainsFound:  hasDomains,
		RequiredRepoIDsFound:      hasRepoIDs,
		TechnicalDocsRootPresent:  exists(technicalRoot),
		ArchitecturePathPresent:   exists(architecturePath),
		ImplementationPathPresent: exists(implementationPath),
		OperationsPathPresent:     exists(operationsPath),
	}

	status := "ready"
	if !checks.Wave3ClosureClosed || !checks.RequiredRepoDomainsFound || !checks.RequiredRepoIDsFound ||
		!checks.TechnicalDocsRootPresent || !checks.ArchitecturePathPresent || !checks.ImplementationPathPresent || !checks.OperationsPathPresent {
		status = "blocked"
	}

	actions := []string{
		"establish owner mappings for architecture, implementation, and operations paths in technical docs repo",
		"wire weekly ownership evidence publication into CI for technical docs repo",
		"enforce cross-repo link validation using existing topology contract",
	}

	notes := []string{"wave4 bootstrap evaluates separation readiness after wave3 closure"}
	if status == "ready" {
		notes = append(notes, "wave4 bootstrap gates satisfied")
	} else {
		notes = append(notes, "wave4 bootstrap gates unresolved")
	}

	report := wave4BootstrapReport{
		Summary: wave4BootstrapSummary{
			CheckpointID:      "WAVE4-BOOTSTRAP-CHECKPOINT-001",
			CheckpointDate:    checkpointDate,
			ClosureReportPath: closurePath,
			TopologyPath:      topologyPath,
			BootstrapStatus:   status,
			Repositories:      len(topologyFile.MultiRepoTopology.Repositories),
			TechnicalDocPaths: boolCount(checks.ArchitecturePathPresent, checks.ImplementationPathPresent, checks.OperationsPathPresent),
		},
		Checks:  checks,
		Actions: actions,
		Notes:   notes,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("wave4-bootstrap write report: %w", err)
	}
	if err := writeWave4BootstrapDoc(docPath, report); err != nil {
		return fmt.Errorf("wave4-bootstrap write markdown: %w", err)
	}

	if status != "ready" {
		return fmt.Errorf("wave4-bootstrap blocked (see %s)", reportPath)
	}

	return nil
}

func boolCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func writeWave4BootstrapDoc(path string, report wave4BootstrapReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	doc := fmt.Sprintf(`---
doc_id: SUM-WAVE4-BOOTSTRAP-2026-02-21
title: Wave 4 Bootstrap Checkpoint (2026-02-21)
doc_type: technical-note
concern: process
status: accepted
phase_scope:
- Scale
owner_role: principal_architect
accountable_role: platform_lead
reviewers:
- tech_lead
- service_owner
tags:
- ai-development
- wave4
- bootstrap
- ownership
linked_ids:
- GOV-MULTIREPO-TRACE-001
review_cadence: weekly
source_of_truth: true
version: '1.0'
specificity: platform-shared
---

# Wave 4 Bootstrap Checkpoint (2026-02-21)

## Decision
- checkpoint_id: %s
- bootstrap_status: %s

## Checks
- wave3_closure_closed: %t
- required_repo_domains_found: %t
- required_repo_ids_found: %t
- technical_docs_root_present: %t
- architecture_path_present: %t
- implementation_path_present: %t
- operations_path_present: %t

## Action Baseline
- %s
- %s
- %s
`,
		report.Summary.CheckpointID,
		report.Summary.BootstrapStatus,
		report.Checks.Wave3ClosureClosed,
		report.Checks.RequiredRepoDomainsFound,
		report.Checks.RequiredRepoIDsFound,
		report.Checks.TechnicalDocsRootPresent,
		report.Checks.ArchitecturePathPresent,
		report.Checks.ImplementationPathPresent,
		report.Checks.OperationsPathPresent,
		report.Actions[0],
		report.Actions[1],
		report.Actions[2],
	)

	return os.WriteFile(path, []byte(doc), 0o644)
}
