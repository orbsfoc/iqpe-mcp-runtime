package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iqpe/docflow/internal/contracts"
)

var reqIDPattern = regexp.MustCompile(`([A-Z]{2,}-[A-Z]+-[0-9]{3}|MVP-CORE-[0-9]{3})`)

func Scan(root string) ([]contracts.Document, error) {
	docs := make([]contracts.Document, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		rel := strings.TrimPrefix(path, root)
		rel = strings.TrimPrefix(rel, string(filepath.Separator))
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		title := extractTitle(string(content), d.Name())
		reqMatches := reqIDPattern.FindAllString(string(content), -1)
		docs = append(docs, contracts.Document{
			DocID:          makeDocID(rel),
			Path:           filepath.ToSlash(filepath.Join(filepath.Base(root), rel)),
			Title:          title,
			Concern:        classifyConcern(rel),
			Specificity:    classifySpecificity(rel),
			PhaseRelevance: []string{"MVP", "Pilot"},
			DocRole:        "source-of-truth",
			RequirementIDs: unique(reqMatches),
		})
		return nil
	})
	return docs, err
}

func extractTitle(content, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}
	return strings.TrimSuffix(fallback, filepath.Ext(fallback))
}

func makeDocID(rel string) string {
	id := strings.ToLower(rel)
	id = strings.TrimSuffix(id, filepath.Ext(id))
	replacer := strings.NewReplacer("/", "-", "_", "-", " ", "-")
	return replacer.Replace(filepath.ToSlash(id))
}

func classifyConcern(rel string) string {
	r := strings.ToLower(rel)
	switch {
	case strings.Contains(r, "requirements"):
		return "product"
	case strings.Contains(r, "architecture"):
		return "architecture"
	case strings.Contains(r, "service"):
		return "service"
	default:
		return "product"
	}
}

func classifySpecificity(rel string) string {
	r := strings.ToLower(rel)
	if strings.Contains(r, "technical") || strings.Contains(r, "infrastructure") {
		return "platform-shared"
	}
	return "product-specific"
}

func unique(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
