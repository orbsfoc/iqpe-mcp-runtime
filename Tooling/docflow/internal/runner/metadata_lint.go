package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type metadataSchemaFile struct {
	MetadataSchema metadataSchema `yaml:"metadata_schema"`
}

type metadataSchema struct {
	RequiredFields []string            `yaml:"required_fields"`
	Enums          map[string][]string `yaml:"enums"`
}

type metadataLintReport struct {
	Summary metadataLintSummary     `yaml:"summary"`
	Files   []metadataLintFileIssue `yaml:"files"`
}

type metadataLintSummary struct {
	DocsRoot        string `yaml:"docs_root"`
	SchemaPath      string `yaml:"schema_path"`
	FilesChecked    int    `yaml:"files_checked"`
	FilesWithErrors int    `yaml:"files_with_errors"`
	Violations      int    `yaml:"violations"`
	Status          string `yaml:"status"`
}

type metadataLintFileIssue struct {
	File       string   `yaml:"file"`
	Violations []string `yaml:"violations"`
}

func MetadataLint(docsRoot, schemaPath, reportOut string) error {
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	var schemaFile metadataSchemaFile
	if err := yaml.Unmarshal(schemaContent, &schemaFile); err != nil {
		return fmt.Errorf("parse schema: %w", err)
	}

	if len(schemaFile.MetadataSchema.RequiredFields) == 0 {
		return fmt.Errorf("schema has no required_fields")
	}

	issues := make([]metadataLintFileIssue, 0)
	docIDToFile := map[string]string{}
	filesChecked := 0
	violationsTotal := 0

	err = filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		filesChecked++
		content, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, metadataLintFileIssue{File: path, Violations: []string{fmt.Sprintf("read_error:%v", err)}})
			violationsTotal++
			return nil
		}

		meta, parseViolations := parseFrontmatterMetadata(string(content))
		fileViolations := make([]string, 0, len(parseViolations)+8)
		fileViolations = append(fileViolations, parseViolations...)

		for _, field := range schemaFile.MetadataSchema.RequiredFields {
			if value, ok := meta[field]; !ok || isInvalidRequiredValue(value) {
				fileViolations = append(fileViolations, "missing_required_field:"+field)
			}
		}

		for field, allowed := range schemaFile.MetadataSchema.Enums {
			value, exists := meta[field]
			if !exists || isInvalidRequiredValue(value) {
				continue
			}
			if !valueInEnum(value, allowed) {
				fileViolations = append(fileViolations, "invalid_enum:"+field)
			}
		}

		if docID, ok := meta["doc_id"].(string); ok && strings.TrimSpace(docID) != "" {
			if existing, exists := docIDToFile[docID]; exists && existing != path {
				fileViolations = append(fileViolations, "duplicate_doc_id:"+docID)
			} else {
				docIDToFile[docID] = path
			}
		}

		if len(fileViolations) > 0 {
			sort.Strings(fileViolations)
			issues = append(issues, metadataLintFileIssue{File: path, Violations: fileViolations})
			violationsTotal += len(fileViolations)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk docs root: %w", err)
	}

	report := metadataLintReport{
		Summary: metadataLintSummary{
			DocsRoot:        docsRoot,
			SchemaPath:      schemaPath,
			FilesChecked:    filesChecked,
			FilesWithErrors: len(issues),
			Violations:      violationsTotal,
			Status:          "pass",
		},
		Files: issues,
	}
	if len(issues) > 0 {
		report.Summary.Status = "fail"
	}

	if err := iohelper.WriteYAML(reportOut, report); err != nil {
		return fmt.Errorf("write lint report: %w", err)
	}

	if len(issues) > 0 {
		return fmt.Errorf("metadata-lint failed: %d files with errors (%d violations), see %s", len(issues), violationsTotal, reportOut)
	}
	return nil
}

func parseFrontmatterMetadata(content string) (map[string]any, []string) {
	violations := make([]string, 0)
	if !strings.HasPrefix(content, "---\n") {
		return map[string]any{}, []string{"missing_frontmatter"}
	}

	remaining := content[4:]
	end := strings.Index(remaining, "\n---\n")
	if end == -1 {
		return map[string]any{}, []string{"invalid_frontmatter_termination"}
	}
	frontmatter := remaining[:end]

	meta := map[string]any{}
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		violations = append(violations, "frontmatter_parse_error")
		return map[string]any{}, violations
	}
	return meta, violations
}

func isInvalidRequiredValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return false
	}
}

func valueInEnum(value any, allowed []string) bool {
	allowedSet := map[string]bool{}
	for _, item := range allowed {
		allowedSet[item] = true
	}

	switch v := value.(type) {
	case string:
		return allowedSet[v]
	case []any:
		for _, item := range v {
			s, ok := item.(string)
			if !ok || !allowedSet[s] {
				return false
			}
		}
		return true
	case []string:
		for _, s := range v {
			if !allowedSet[s] {
				return false
			}
		}
		return true
	default:
		return false
	}
}
