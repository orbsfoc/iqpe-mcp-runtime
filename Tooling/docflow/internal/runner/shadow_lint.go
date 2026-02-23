package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type shadowLintReport struct {
	Summary shadowLintSummary `yaml:"summary"`
	Entries []shadowLintEntry `yaml:"entries"`
}

type shadowLintSummary struct {
	RegisterPath    string `yaml:"register_path"`
	EntriesChecked  int    `yaml:"entries_checked"`
	EntriesWithViol int    `yaml:"entries_with_violations"`
	Violations      int    `yaml:"violations"`
	Status          string `yaml:"status"`
}

type shadowLintEntry struct {
	EntryID     string   `yaml:"entry_id"`
	Source      string   `yaml:"source"`
	ShadowState string   `yaml:"product_shadow_state"`
	Violations  []string `yaml:"violations"`
}

func ShadowLint(registerPath, reportPath string) error {
	content, err := os.ReadFile(registerPath)
	if err != nil {
		return fmt.Errorf("shadow-lint read register: %w", err)
	}

	var register technicalCutoverRegisterFile
	if err := yaml.Unmarshal(content, &register); err != nil {
		return fmt.Errorf("shadow-lint parse register: %w", err)
	}

	entries := register.TechnicalDocsCutover.Entries
	reportEntries := make([]shadowLintEntry, 0, len(entries))
	violationsCount := 0
	entriesWithViol := 0

	for _, entry := range entries {
		reportEntry := shadowLintEntry{
			EntryID:     entry.EntryID,
			Source:      entry.Source,
			ShadowState: entry.ProductShadowState,
			Violations:  make([]string, 0),
		}

		sourcePath := filepath.Clean(filepath.Join("../../", filepath.FromSlash(entry.Source)))
		isMarkdown := strings.HasSuffix(strings.ToLower(sourcePath), ".md")
		isDir := isDirectory(sourcePath)

		switch entry.ProductShadowState {
		case "deprecated-ready":
			if !exists(sourcePath) {
				reportEntry.Violations = append(reportEntry.Violations, "missing_source_file")
				break
			}
			if isDir {
				break
			}
			if !isMarkdown {
				reportEntry.Violations = append(reportEntry.Violations, "deprecated_ready_requires_markdown_source")
				break
			}
			metadata, err := readFrontmatterMap(sourcePath)
			if err != nil {
				reportEntry.Violations = append(reportEntry.Violations, "frontmatter_parse_error")
				break
			}
			status, _ := metadata["status"].(string)
			if strings.TrimSpace(status) != "deprecated" {
				reportEntry.Violations = append(reportEntry.Violations, "status_not_deprecated")
			}
			if !hasTag(metadata, "deprecated-ready-shadow") {
				reportEntry.Violations = append(reportEntry.Violations, "missing_tag_deprecated_ready_shadow")
			}
		case "shadow-active":
			if !exists(sourcePath) {
				reportEntry.Violations = append(reportEntry.Violations, "missing_source_file")
			}
		case "removed":
			if exists(sourcePath) {
				reportEntry.Violations = append(reportEntry.Violations, "source_should_be_removed")
			}
		default:
			reportEntry.Violations = append(reportEntry.Violations, "invalid_shadow_state")
		}

		if len(reportEntry.Violations) > 0 {
			entriesWithViol++
			violationsCount += len(reportEntry.Violations)
		}
		reportEntries = append(reportEntries, reportEntry)
	}

	status := "pass"
	if violationsCount > 0 {
		status = "fail"
	}

	report := shadowLintReport{
		Summary: shadowLintSummary{
			RegisterPath:    registerPath,
			EntriesChecked:  len(entries),
			EntriesWithViol: entriesWithViol,
			Violations:      violationsCount,
			Status:          status,
		},
		Entries: reportEntries,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("shadow-lint write report: %w", err)
	}

	if violationsCount > 0 {
		return fmt.Errorf("shadow-lint failed: %d violations, see %s", violationsCount, reportPath)
	}

	return nil
}

func readFrontmatterMap(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	trimmed := strings.TrimSpace(strings.TrimPrefix(text, "\ufeff"))
	if !strings.HasPrefix(trimmed, "---\n") {
		return nil, fmt.Errorf("no frontmatter")
	}
	remainder := strings.TrimPrefix(trimmed, "---\n")
	end := strings.Index(remainder, "\n---\n")
	if end < 0 {
		return nil, fmt.Errorf("unterminated frontmatter")
	}
	frontmatter := remainder[:end]
	metadata := map[string]any{}
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

func hasTag(metadata map[string]any, expected string) bool {
	tagsRaw, ok := metadata["tags"]
	if !ok {
		return false
	}
	tags, ok := tagsRaw.([]any)
	if !ok {
		return false
	}
	for _, tag := range tags {
		if value, ok := tag.(string); ok && strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
