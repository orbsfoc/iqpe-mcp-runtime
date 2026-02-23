package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type cutoverProgressReport struct {
	Summary cutoverProgressSummary `yaml:"summary"`
	Entries []cutoverProgressEntry `yaml:"entries"`
}

type cutoverProgressSummary struct {
	RegisterPath      string `yaml:"register_path"`
	TotalEntries      int    `yaml:"total_entries"`
	CompletedEntries  int    `yaml:"completed_entries"`
	InProgressEntries int    `yaml:"in_progress_entries"`
	PlannedEntries    int    `yaml:"planned_entries"`
	ReadyForRemoval   int    `yaml:"ready_for_removal"`
	Status            string `yaml:"status"`
}

type cutoverProgressEntry struct {
	EntryID            string `yaml:"entry_id"`
	Source             string `yaml:"source"`
	Target             string `yaml:"target"`
	CutoverStatus      string `yaml:"cutover_status"`
	ProductShadowState string `yaml:"product_shadow_state"`
	TechnicalPrimary   bool   `yaml:"technical_primary"`
	SourceExists       bool   `yaml:"source_exists"`
	TargetExists       bool   `yaml:"target_exists"`
	RemovalReady       bool   `yaml:"removal_ready"`
}

func CutoverProgress(registerPath, reportPath string) error {
	content, err := os.ReadFile(registerPath)
	if err != nil {
		return fmt.Errorf("cutover-progress read register: %w", err)
	}

	var register technicalCutoverRegisterFile
	if err := yaml.Unmarshal(content, &register); err != nil {
		return fmt.Errorf("cutover-progress parse register: %w", err)
	}

	entries := register.TechnicalDocsCutover.Entries
	reportEntries := make([]cutoverProgressEntry, 0, len(entries))

	completed := 0
	inProgress := 0
	planned := 0
	readyForRemoval := 0

	for _, entry := range entries {
		sourcePath := filepath.Clean(filepath.Join("../../", filepath.FromSlash(entry.Source)))
		targetPath := filepath.Clean(filepath.Join("../../", filepath.FromSlash(entry.Target)))
		sourceExists := exists(sourcePath)
		targetExists := exists(targetPath)

		status := strings.TrimSpace(entry.CutoverStatus)
		switch status {
		case "completed":
			completed++
		case "in-progress":
			inProgress++
		case "planned":
			planned++
		}

		removalReady := entry.TechnicalPrimary && targetExists && (entry.ProductShadowState == "deprecated-ready" || entry.ProductShadowState == "removed")
		if removalReady {
			readyForRemoval++
		}

		reportEntries = append(reportEntries, cutoverProgressEntry{
			EntryID:            entry.EntryID,
			Source:             entry.Source,
			Target:             entry.Target,
			CutoverStatus:      entry.CutoverStatus,
			ProductShadowState: entry.ProductShadowState,
			TechnicalPrimary:   entry.TechnicalPrimary,
			SourceExists:       sourceExists,
			TargetExists:       targetExists,
			RemovalReady:       removalReady,
		})
	}

	summaryStatus := "in-progress"
	if len(entries) > 0 && completed == len(entries) {
		summaryStatus = "completed"
	}

	report := cutoverProgressReport{
		Summary: cutoverProgressSummary{
			RegisterPath:      registerPath,
			TotalEntries:      len(entries),
			CompletedEntries:  completed,
			InProgressEntries: inProgress,
			PlannedEntries:    planned,
			ReadyForRemoval:   readyForRemoval,
			Status:            summaryStatus,
		},
		Entries: reportEntries,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("cutover-progress write report: %w", err)
	}

	return nil
}
