package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type wave3RemediationReport struct {
	Summary wave3RemediationSummary `yaml:"summary"`
	Entries []wave3RemediationEntry `yaml:"entries"`
}

type wave3RemediationSummary struct {
	ReadinessReportPath string   `yaml:"readiness_report_path"`
	RegisterPath        string   `yaml:"register_path"`
	GoNoGo              string   `yaml:"go_no_go"`
	BlockedEntries      int      `yaml:"blocked_entries"`
	RemediationStatus   string   `yaml:"remediation_status"`
	GlobalActions       []string `yaml:"global_actions"`
	ValidationCommands  []string `yaml:"validation_commands"`
}

type wave3RemediationEntry struct {
	EntryID              string   `yaml:"entry_id"`
	Source               string   `yaml:"source"`
	Target               string   `yaml:"target"`
	CurrentShadowState   string   `yaml:"current_shadow_state"`
	DesiredShadowState   string   `yaml:"desired_shadow_state"`
	CurrentCutoverStatus string   `yaml:"current_cutover_status"`
	DesiredCutoverStatus string   `yaml:"desired_cutover_status"`
	Blockers             []string `yaml:"blockers"`
	RequiredTransitions  []string `yaml:"required_transitions"`
	Notes                []string `yaml:"notes"`
}

func Wave3Remediation(readinessPath, registerPath, reportPath string) error {
	readinessContent, err := os.ReadFile(readinessPath)
	if err != nil {
		return fmt.Errorf("wave3-remediation read readiness report: %w", err)
	}

	var readiness wave3ReadinessReport
	if err := yaml.Unmarshal(readinessContent, &readiness); err != nil {
		return fmt.Errorf("wave3-remediation parse readiness report: %w", err)
	}

	registerContent, err := os.ReadFile(registerPath)
	if err != nil {
		return fmt.Errorf("wave3-remediation read cutover register: %w", err)
	}

	var register technicalCutoverRegisterFile
	if err := yaml.Unmarshal(registerContent, &register); err != nil {
		return fmt.Errorf("wave3-remediation parse cutover register: %w", err)
	}

	registerByID := map[string]technicalCutoverEntry{}
	for _, entry := range register.TechnicalDocsCutover.Entries {
		registerByID[strings.TrimSpace(entry.EntryID)] = entry
	}

	blocked := make([]wave3RemediationEntry, 0)
	for _, readinessEntry := range readiness.Entries {
		if readinessEntry.RemovalReady {
			continue
		}
		registerEntry, ok := registerByID[strings.TrimSpace(readinessEntry.EntryID)]
		if !ok {
			return fmt.Errorf("wave3-remediation missing register entry for %s", readinessEntry.EntryID)
		}

		transitions := make([]string, 0)
		notes := make([]string, 0)

		if strings.TrimSpace(registerEntry.ProductShadowState) != "deprecated-ready" {
			transitions = append(transitions, "set product_shadow_state to deprecated-ready")
			notes = append(notes, "for markdown shadow sources ensure frontmatter status=deprecated and tag deprecated-ready-shadow before state flip")
		}

		if strings.TrimSpace(registerEntry.CutoverStatus) != "completed" {
			transitions = append(transitions, "set cutover_status to completed after technical-primary verification and link rewrites")
		}

		transitions = append(transitions, "rerun cutover-progress and wave3-readiness to confirm removal_ready=true")

		blocked = append(blocked, wave3RemediationEntry{
			EntryID:              readinessEntry.EntryID,
			Source:               registerEntry.Source,
			Target:               registerEntry.Target,
			CurrentShadowState:   registerEntry.ProductShadowState,
			DesiredShadowState:   "deprecated-ready",
			CurrentCutoverStatus: registerEntry.CutoverStatus,
			DesiredCutoverStatus: "completed",
			Blockers:             readinessEntry.Blockers,
			RequiredTransitions:  transitions,
			Notes:                notes,
		})
	}

	globalActions := []string{
		"update technical-docs-cutover-register entries listed in this report",
		"execute deterministic lint chain before wave3-readiness rerun",
		"proceed to Wave 3 shadow removal only when wave3-readiness go_no_go=go",
	}

	validationCommands := []string{
		"Tooling/agent-tools/scripts/shadow-lint.sh",
		"Tooling/agent-tools/scripts/planning-lint.sh",
		"Tooling/agent-tools/scripts/metadata-lint.sh",
		"Tooling/agent-tools/scripts/validate-docs.sh",
		"Tooling/agent-tools/scripts/cutover-progress.sh",
		"Tooling/agent-tools/scripts/wave3-readiness.sh",
	}

	status := "ready"
	if len(blocked) > 0 || strings.TrimSpace(readiness.Summary.GoNoGo) != "go" {
		status = "action-required"
	}

	report := wave3RemediationReport{
		Summary: wave3RemediationSummary{
			ReadinessReportPath: readinessPath,
			RegisterPath:        registerPath,
			GoNoGo:              readiness.Summary.GoNoGo,
			BlockedEntries:      len(blocked),
			RemediationStatus:   status,
			GlobalActions:       globalActions,
			ValidationCommands:  validationCommands,
		},
		Entries: blocked,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("wave3-remediation write report: %w", err)
	}

	return nil
}
