package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type libraryCatalogFile struct {
	Libraries []map[string]any `yaml:"libraries"`
}

type librarySearchReport struct {
	Summary librarySearchSummary `yaml:"summary"`
	Matches []map[string]any     `yaml:"matches"`
	Filters map[string]string    `yaml:"filters"`
}

type librarySearchSummary struct {
	CatalogPath string `yaml:"catalog_path"`
	Total       int    `yaml:"total"`
	Matches     int    `yaml:"matches"`
	Status      string `yaml:"status"`
}

func LibrarySearch(catalogPath, reportPath, query, capability, status, runtime string) error {
	content, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("library-search read catalog: %w", err)
	}

	var catalog libraryCatalogFile
	if err := yaml.Unmarshal(content, &catalog); err != nil {
		return fmt.Errorf("library-search parse catalog: %w", err)
	}

	query = strings.ToLower(strings.TrimSpace(query))
	capability = strings.ToLower(strings.TrimSpace(capability))
	status = strings.ToLower(strings.TrimSpace(status))
	runtime = strings.ToLower(strings.TrimSpace(runtime))

	matches := make([]map[string]any, 0)
	for _, lib := range catalog.Libraries {
		if !libraryMatches(lib, query, capability, status, runtime) {
			continue
		}
		matches = append(matches, lib)
	}

	report := librarySearchReport{
		Summary: librarySearchSummary{
			CatalogPath: catalogPath,
			Total:       len(catalog.Libraries),
			Matches:     len(matches),
			Status:      "pass",
		},
		Matches: matches,
		Filters: map[string]string{
			"query":      query,
			"capability": capability,
			"status":     status,
			"runtime":    runtime,
		},
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("library-search write report: %w", err)
	}

	return nil
}

func libraryMatches(lib map[string]any, query, capability, status, runtime string) bool {
	if query != "" {
		if !strings.Contains(strings.ToLower(stringField(lib, "library_id")), query) &&
			!strings.Contains(strings.ToLower(stringField(lib, "asset_id")), query) &&
			!strings.Contains(strings.ToLower(stringField(lib, "name")), query) &&
			!strings.Contains(strings.ToLower(stringField(lib, "owner_team")), query) {
			return false
		}
	}

	if status != "" {
		libStatus := strings.ToLower(stringField(lib, "status"))
		if libStatus == "" {
			libStatus = strings.ToLower(stringField(lib, "stability_level"))
		}
		if libStatus != status {
			return false
		}
	}

	if runtime != "" {
		libRuntime := strings.ToLower(stringField(lib, "language_runtime"))
		if libRuntime == "" {
			libRuntime = strings.ToLower(stringField(lib, "runtime"))
		}
		if !strings.Contains(libRuntime, runtime) {
			return false
		}
	}

	if capability != "" {
		if !sliceContainsString(lib["capability_tags"], capability) {
			return false
		}
	}

	return true
}

func stringField(lib map[string]any, key string) string {
	value, ok := lib[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func sliceContainsString(value any, token string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return true
	}

	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if strings.Contains(strings.ToLower(fmt.Sprintf("%v", item)), token) {
				return true
			}
		}
	case []string:
		for _, item := range v {
			if strings.Contains(strings.ToLower(item), token) {
				return true
			}
		}
	case string:
		return strings.Contains(strings.ToLower(v), token)
	}

	return false
}
