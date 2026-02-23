package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type techRadarFile struct {
	RadarEntries   []map[string]any `yaml:"radar_entries"`
	ReleaseSummary map[string]any   `yaml:"release_summary"`
}

type techRadarLintReport struct {
	Summary techRadarLintSummary       `yaml:"summary"`
	Entries []techRadarLintEntryResult `yaml:"entries"`
	Issues  []string                   `yaml:"issues,omitempty"`
}

type techRadarLintSummary struct {
	RadarPath    string `yaml:"radar_path"`
	Entries      int    `yaml:"entries"`
	Issues       int    `yaml:"issues"`
	Adopt        int    `yaml:"adopt"`
	Trial        int    `yaml:"trial"`
	Hold         int    `yaml:"hold"`
	Retire       int    `yaml:"retire"`
	ReleaseRisks int    `yaml:"release_risks"`
	Status       string `yaml:"status"`
}

type techRadarLintEntryResult struct {
	ItemID      string   `yaml:"item_id"`
	Technology  string   `yaml:"technology"`
	Disposition string   `yaml:"disposition"`
	Result      string   `yaml:"result"`
	Issues      []string `yaml:"issues,omitempty"`
}

func TechRadarLint(radarPath, reportPath string) error {
	content, err := os.ReadFile(radarPath)
	if err != nil {
		return fmt.Errorf("tech-radar-lint read radar: %w", err)
	}

	var radar techRadarFile
	if err := yaml.Unmarshal(content, &radar); err != nil {
		return fmt.Errorf("tech-radar-lint parse radar: %w", err)
	}

	issues := make([]string, 0)
	entryResults := make([]techRadarLintEntryResult, 0, len(radar.RadarEntries))

	allowedDispositions := map[string]bool{
		"ADOPT":  true,
		"TRIAL":  true,
		"HOLD":   true,
		"RETIRE": true,
	}

	adopt := 0
	trial := 0
	hold := 0
	retire := 0

	if len(radar.RadarEntries) == 0 {
		issues = append(issues, "missing:radar_entries")
	}

	for idx, entry := range radar.RadarEntries {
		itemID := strings.TrimSpace(stringField(entry, "item_id"))
		tech := strings.TrimSpace(stringField(entry, "technology"))
		disposition := strings.ToUpper(strings.TrimSpace(stringField(entry, "disposition")))
		if disposition == "" {
			disposition = strings.ToUpper(strings.TrimSpace(stringField(entry, "ring")))
		}

		result := techRadarLintEntryResult{
			ItemID:      itemID,
			Technology:  tech,
			Disposition: disposition,
			Result:      "pass",
			Issues:      make([]string, 0),
		}

		if itemID == "" {
			result.Issues = append(result.Issues, fmt.Sprintf("missing:item_id:index:%d", idx))
		}
		if tech == "" {
			result.Issues = append(result.Issues, "missing:technology")
		}
		if disposition == "" {
			result.Issues = append(result.Issues, "missing:disposition")
		} else if !allowedDispositions[disposition] {
			result.Issues = append(result.Issues, fmt.Sprintf("invalid:disposition:%s", disposition))
		}

		switch disposition {
		case "ADOPT":
			adopt++
		case "TRIAL":
			trial++
		case "HOLD":
			hold++
		case "RETIRE":
			retire++
		}

		if len(result.Issues) > 0 {
			result.Result = "fail"
			issues = append(issues, result.Issues...)
		}

		entryResults = append(entryResults, result)
	}

	checkpoint := strings.TrimSpace(stringField(radar.ReleaseSummary, "checkpoint"))
	if checkpoint == "" {
		issues = append(issues, "missing:release_summary.checkpoint")
	}
	if !hasNonEmptyField(radar.ReleaseSummary, "updated_items") {
		issues = append(issues, "missing:release_summary.updated_items")
	}

	releaseRisks := hold + retire
	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}

	report := techRadarLintReport{
		Summary: techRadarLintSummary{
			RadarPath:    radarPath,
			Entries:      len(radar.RadarEntries),
			Issues:       len(issues),
			Adopt:        adopt,
			Trial:        trial,
			Hold:         hold,
			Retire:       retire,
			ReleaseRisks: releaseRisks,
			Status:       status,
		},
		Entries: entryResults,
		Issues:  issues,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("tech-radar-lint write report: %w", err)
	}

	if status != "pass" {
		return fmt.Errorf("tech-radar-lint failed: %d issue(s), see %s", len(issues), reportPath)
	}

	return nil
}
