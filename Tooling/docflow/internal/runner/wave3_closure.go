package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type wave3ClosureReport struct {
	Summary  wave3ClosureSummary `yaml:"summary"`
	Gates    wave3ClosureGates   `yaml:"gates"`
	Evidence []string            `yaml:"evidence"`
	Notes    []string            `yaml:"notes"`
}

type wave3ClosureSummary struct {
	CheckpointID     string `yaml:"checkpoint_id"`
	CheckpointDate   string `yaml:"checkpoint_date"`
	ReadinessPath    string `yaml:"readiness_path"`
	RemovalPath      string `yaml:"removal_path"`
	RegisterPath     string `yaml:"register_path"`
	ClosureStatus    string `yaml:"closure_status"`
	TotalEntries     int    `yaml:"total_entries"`
	RemovedEntries   int    `yaml:"removed_entries"`
	CompletedEntries int    `yaml:"completed_entries"`
}

type wave3ClosureGates struct {
	ReadinessGo          bool `yaml:"readiness_go"`
	RemovalReportPass    bool `yaml:"removal_report_pass"`
	ShadowLintPass       bool `yaml:"shadow_lint_pass"`
	PlanningLintPass     bool `yaml:"planning_lint_pass"`
	MetadataLintPass     bool `yaml:"metadata_lint_pass"`
	RegisterAllRemoved   bool `yaml:"register_all_removed"`
	RegisterAllCompleted bool `yaml:"register_all_completed"`
}

func Wave3Closure(readinessPath, removalPath, shadowLintPath, planningLintPath, metadataLintPath, registerPath, reportPath, docPath, checkpointDate string) error {
	readinessContent, err := os.ReadFile(readinessPath)
	if err != nil {
		return fmt.Errorf("wave3-closure read readiness report: %w", err)
	}
	var readiness wave3ReadinessReport
	if err := yaml.Unmarshal(readinessContent, &readiness); err != nil {
		return fmt.Errorf("wave3-closure parse readiness report: %w", err)
	}

	removalContent, err := os.ReadFile(removalPath)
	if err != nil {
		return fmt.Errorf("wave3-closure read removal report: %w", err)
	}
	var removal wave3RemovalReport
	if err := yaml.Unmarshal(removalContent, &removal); err != nil {
		return fmt.Errorf("wave3-closure parse removal report: %w", err)
	}

	shadowPass, err := reportStatusPass(shadowLintPath)
	if err != nil {
		return err
	}
	planningPass, err := reportStatusPass(planningLintPath)
	if err != nil {
		return err
	}
	metadataPass, err := reportStatusPass(metadataLintPath)
	if err != nil {
		return err
	}

	registerContent, err := os.ReadFile(registerPath)
	if err != nil {
		return fmt.Errorf("wave3-closure read cutover register: %w", err)
	}
	var register technicalCutoverRegisterFile
	if err := yaml.Unmarshal(registerContent, &register); err != nil {
		return fmt.Errorf("wave3-closure parse cutover register: %w", err)
	}

	total := len(register.TechnicalDocsCutover.Entries)
	removed := 0
	completed := 0
	for _, entry := range register.TechnicalDocsCutover.Entries {
		if strings.TrimSpace(entry.ProductShadowState) == "removed" {
			removed++
		}
		if strings.TrimSpace(entry.CutoverStatus) == "completed" {
			completed++
		}
	}

	gates := wave3ClosureGates{
		ReadinessGo:          strings.TrimSpace(readiness.Summary.GoNoGo) == "go",
		RemovalReportPass:    strings.TrimSpace(removal.Summary.Status) == "pass" && removal.Summary.Errors == 0,
		ShadowLintPass:       shadowPass,
		PlanningLintPass:     planningPass,
		MetadataLintPass:     metadataPass,
		RegisterAllRemoved:   total > 0 && removed == total,
		RegisterAllCompleted: total > 0 && completed == total,
	}

	closureStatus := "open"
	notes := make([]string, 0)
	if gates.ReadinessGo && gates.RemovalReportPass && gates.ShadowLintPass && gates.PlanningLintPass && gates.MetadataLintPass && gates.RegisterAllRemoved && gates.RegisterAllCompleted {
		closureStatus = "closed"
		notes = append(notes, "wave3 cutover closure gates satisfied")
	} else {
		notes = append(notes, "wave3 closure pending unresolved gates")
	}

	report := wave3ClosureReport{
		Summary: wave3ClosureSummary{
			CheckpointID:     "WAVE3-CLOSURE-CHECKPOINT-001",
			CheckpointDate:   checkpointDate,
			ReadinessPath:    readinessPath,
			RemovalPath:      removalPath,
			RegisterPath:     registerPath,
			ClosureStatus:    closureStatus,
			TotalEntries:     total,
			RemovedEntries:   removed,
			CompletedEntries: completed,
		},
		Gates: gates,
		Evidence: []string{
			readinessPath,
			removalPath,
			shadowLintPath,
			planningLintPath,
			metadataLintPath,
			registerPath,
		},
		Notes: notes,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("wave3-closure write report: %w", err)
	}

	if err := writeWave3ClosureDoc(docPath, report); err != nil {
		return fmt.Errorf("wave3-closure write markdown: %w", err)
	}

	if closureStatus != "closed" {
		return fmt.Errorf("wave3-closure unresolved gates (see %s)", reportPath)
	}

	return nil
}

func reportStatusPass(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("wave3-closure read report %s: %w", path, err)
	}
	var generic map[string]any
	if err := yaml.Unmarshal(content, &generic); err != nil {
		return false, fmt.Errorf("wave3-closure parse report %s: %w", path, err)
	}
	summaryRaw, ok := generic["summary"]
	if !ok {
		return false, fmt.Errorf("wave3-closure missing summary in %s", path)
	}
	summary, ok := summaryRaw.(map[string]any)
	if !ok {
		return false, fmt.Errorf("wave3-closure invalid summary structure in %s", path)
	}
	status, _ := summary["status"].(string)
	return strings.TrimSpace(status) == "pass", nil
}

func writeWave3ClosureDoc(path string, report wave3ClosureReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	doc := fmt.Sprintf(`---
doc_id: SUM-WAVE3-CLOSURE-2026-02-21
title: Wave 3 Closure Checkpoint (2026-02-21)
doc_type: technical-note
concern: process
status: accepted
phase_scope:
- Production
- Scale
owner_role: principal_architect
accountable_role: platform_lead
reviewers:
- tech_lead
- service_owner
tags:
- ai-development
- wave3
- closure
- checkpoint
linked_ids:
- GOV-WAVE2-CUTOVER-001
review_cadence: per-phase-gate
source_of_truth: true
version: '1.0'
specificity: platform-shared
---

# Wave 3 Closure Checkpoint (2026-02-21)

## Decision
- checkpoint_id: %s
- closure_status: %s

## Gate Summary
- readiness_go: %t
- removal_report_pass: %t
- shadow_lint_pass: %t
- planning_lint_pass: %t
- metadata_lint_pass: %t
- register_all_removed: %t
- register_all_completed: %t

## Cutover Totals
- total_entries: %d
- removed_entries: %d
- completed_entries: %d

## Evidence
- %s
- %s
- %s
- %s
- %s
- %s
`,
		report.Summary.CheckpointID,
		report.Summary.ClosureStatus,
		report.Gates.ReadinessGo,
		report.Gates.RemovalReportPass,
		report.Gates.ShadowLintPass,
		report.Gates.PlanningLintPass,
		report.Gates.MetadataLintPass,
		report.Gates.RegisterAllRemoved,
		report.Gates.RegisterAllCompleted,
		report.Summary.TotalEntries,
		report.Summary.RemovedEntries,
		report.Summary.CompletedEntries,
		report.Evidence[0],
		report.Evidence[1],
		report.Evidence[2],
		report.Evidence[3],
		report.Evidence[4],
		report.Evidence[5],
	)

	return os.WriteFile(path, []byte(doc), 0o644)
}
