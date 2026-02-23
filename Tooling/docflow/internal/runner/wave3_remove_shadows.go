package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type wave3RemovalReport struct {
	Summary wave3RemovalSummary `yaml:"summary"`
	Entries []wave3RemovalEntry `yaml:"entries"`
}

type wave3RemovalSummary struct {
	ReadinessReportPath string `yaml:"readiness_report_path"`
	RegisterPath        string `yaml:"register_path"`
	EntriesAttempted    int    `yaml:"entries_attempted"`
	EntriesRemoved      int    `yaml:"entries_removed"`
	EntriesSkipped      int    `yaml:"entries_skipped"`
	Errors              int    `yaml:"errors"`
	Status              string `yaml:"status"`
}

type wave3RemovalEntry struct {
	EntryID             string   `yaml:"entry_id"`
	Source              string   `yaml:"source"`
	Target              string   `yaml:"target"`
	PreviousShadowState string   `yaml:"previous_shadow_state"`
	NewShadowState      string   `yaml:"new_shadow_state"`
	PreviousCutover     string   `yaml:"previous_cutover_status"`
	NewCutover          string   `yaml:"new_cutover_status"`
	Removed             bool     `yaml:"removed"`
	Skipped             bool     `yaml:"skipped"`
	Notes               []string `yaml:"notes"`
	Errors              []string `yaml:"errors"`
}

func Wave3RemoveShadows(readinessPath, registerPath, reportPath string) error {
	readinessContent, err := os.ReadFile(readinessPath)
	if err != nil {
		return fmt.Errorf("wave3-remove-shadows read readiness report: %w", err)
	}

	var readiness wave3ReadinessReport
	if err := yaml.Unmarshal(readinessContent, &readiness); err != nil {
		return fmt.Errorf("wave3-remove-shadows parse readiness report: %w", err)
	}

	if strings.TrimSpace(readiness.Summary.GoNoGo) != "go" {
		return fmt.Errorf("wave3-remove-shadows requires wave3-readiness go state")
	}

	registerContent, err := os.ReadFile(registerPath)
	if err != nil {
		return fmt.Errorf("wave3-remove-shadows read cutover register: %w", err)
	}

	var register technicalCutoverRegisterFile
	if err := yaml.Unmarshal(registerContent, &register); err != nil {
		return fmt.Errorf("wave3-remove-shadows parse cutover register: %w", err)
	}

	readyByID := map[string]wave3ReadinessEntry{}
	for _, entry := range readiness.Entries {
		readyByID[strings.TrimSpace(entry.EntryID)] = entry
	}

	productRoot := filepath.Clean(filepath.Join("../../", filepath.FromSlash("Docs/RefactoredProductDocs")))
	reportEntries := make([]wave3RemovalEntry, 0, len(register.TechnicalDocsCutover.Entries))
	entriesAttempted := 0
	entriesRemoved := 0
	entriesSkipped := 0
	totalErrors := 0

	for index := range register.TechnicalDocsCutover.Entries {
		entry := &register.TechnicalDocsCutover.Entries[index]
		item := wave3RemovalEntry{
			EntryID:             entry.EntryID,
			Source:              entry.Source,
			Target:              entry.Target,
			PreviousShadowState: entry.ProductShadowState,
			NewShadowState:      entry.ProductShadowState,
			PreviousCutover:     entry.CutoverStatus,
			NewCutover:          entry.CutoverStatus,
			Notes:               make([]string, 0),
			Errors:              make([]string, 0),
		}

		readinessEntry, ok := readyByID[strings.TrimSpace(entry.EntryID)]
		if !ok || !readinessEntry.RemovalReady {
			item.Skipped = true
			item.Notes = append(item.Notes, "entry not marked removal_ready in readiness report")
			reportEntries = append(reportEntries, item)
			entriesSkipped++
			continue
		}

		entriesAttempted++
		sourcePath := filepath.Clean(filepath.Join("../../", filepath.FromSlash(entry.Source)))
		if sourcePath == productRoot || !isPathUnder(sourcePath, productRoot) {
			item.Errors = append(item.Errors, "unsafe_source_path_outside_product_docs_root")
			totalErrors += len(item.Errors)
			reportEntries = append(reportEntries, item)
			continue
		}

		if exists(sourcePath) {
			if err := os.RemoveAll(sourcePath); err != nil {
				item.Errors = append(item.Errors, fmt.Sprintf("remove_source_failed: %v", err))
				totalErrors += len(item.Errors)
				reportEntries = append(reportEntries, item)
				continue
			}
			item.Notes = append(item.Notes, "source path removed")
		} else {
			item.Notes = append(item.Notes, "source path already absent")
		}

		entry.ProductShadowState = "removed"
		entry.CutoverStatus = "completed"
		item.NewShadowState = entry.ProductShadowState
		item.NewCutover = entry.CutoverStatus
		item.Removed = true
		entriesRemoved++
		reportEntries = append(reportEntries, item)
	}

	status := "pass"
	if totalErrors > 0 {
		status = "fail"
	}

	report := wave3RemovalReport{
		Summary: wave3RemovalSummary{
			ReadinessReportPath: readinessPath,
			RegisterPath:        registerPath,
			EntriesAttempted:    entriesAttempted,
			EntriesRemoved:      entriesRemoved,
			EntriesSkipped:      entriesSkipped,
			Errors:              totalErrors,
			Status:              status,
		},
		Entries: reportEntries,
	}

	if err := iohelper.WriteYAML(registerPath, register); err != nil {
		return fmt.Errorf("wave3-remove-shadows write updated register: %w", err)
	}
	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("wave3-remove-shadows write report: %w", err)
	}

	if totalErrors > 0 {
		return fmt.Errorf("wave3-remove-shadows encountered %d errors (see %s)", totalErrors, reportPath)
	}

	return nil
}

func isPathUnder(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}
