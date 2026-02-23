package runner

import (
	"fmt"
	"os"
	"path/filepath"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type v2DemoReadinessReport struct {
	Summary  v2DemoReadinessSummary `yaml:"summary"`
	Features []v2FeatureCheck       `yaml:"features"`
}

type v2DemoReadinessSummary struct {
	DemoRoot         string `yaml:"demo_root"`
	Checks           int    `yaml:"checks"`
	MissingArtifacts int    `yaml:"missing_artifacts"`
	SchemaViolations int    `yaml:"schema_violations"`
	Status           string `yaml:"status"`
}

type v2FeatureCheck struct {
	FeatureID string   `yaml:"feature_id"`
	Artifact  string   `yaml:"artifact"`
	Status    string   `yaml:"status"`
	Issues    []string `yaml:"issues,omitempty"`
}

type v2ArtifactRule struct {
	FeatureID    string
	ArtifactPath string
	RequiredKeys []string
}

func V2DemoReadiness(demoRoot, reportPath string) error {
	rules := []v2ArtifactRule{
		{FeatureID: "feature-01", ArtifactPath: "artifacts/session-plan-orchestration.yaml", RequiredKeys: []string{"sessions", "execution_tracks"}},
		{FeatureID: "feature-02", ArtifactPath: "artifacts/library-catalog.yaml", RequiredKeys: []string{"libraries"}},
		{FeatureID: "feature-03", ArtifactPath: "artifacts/new-library-decisions.yaml", RequiredKeys: []string{"decisions"}},
		{FeatureID: "feature-04", ArtifactPath: "artifacts/phase-tech-policy.yaml", RequiredKeys: []string{"phase_policy"}},
		{FeatureID: "feature-05", ArtifactPath: "artifacts/tech-radar-summary.yaml", RequiredKeys: []string{"radar_entries", "release_summary"}},
		{FeatureID: "feature-06", ArtifactPath: "artifacts/architecture-operations-boundary.yaml", RequiredKeys: []string{"architecture_artifacts", "operations_artifacts"}},
		{FeatureID: "feature-07", ArtifactPath: "artifacts/portfolio-topology.yaml", RequiredKeys: []string{"repositories", "services"}},
		{FeatureID: "feature-08", ArtifactPath: "artifacts/data-boundary-evidence.yaml", RequiredKeys: []string{"agent_local_data", "mcp_data", "evidence_links"}},
		{FeatureID: "feature-09", ArtifactPath: "artifacts/code-unit-metadata.yaml", RequiredKeys: []string{"units"}},
		{FeatureID: "feature-09", ArtifactPath: "artifacts/human-readable-overviews.yaml", RequiredKeys: []string{"system_overview", "product_overview", "architecture_overview", "operations_overview"}},
		{FeatureID: "feature-10", ArtifactPath: "artifacts/plan-resource-estimation.yaml", RequiredKeys: []string{"plan_estimations"}},
		{FeatureID: "feature-10", ArtifactPath: "artifacts/developer-agent-usage-report.yaml", RequiredKeys: []string{"agent_reports", "summary"}},
	}

	checks := make([]v2FeatureCheck, 0, len(rules))
	missingArtifacts := 0
	schemaViolations := 0

	for _, rule := range rules {
		fullPath := filepath.Join(demoRoot, filepath.FromSlash(rule.ArtifactPath))
		item := v2FeatureCheck{
			FeatureID: rule.FeatureID,
			Artifact:  rule.ArtifactPath,
			Status:    "pass",
			Issues:    make([]string, 0),
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			item.Status = "fail"
			item.Issues = append(item.Issues, "missing:artifact")
			missingArtifacts++
			checks = append(checks, item)
			continue
		}

		var payload map[string]any
		if err := yaml.Unmarshal(content, &payload); err != nil {
			item.Status = "fail"
			item.Issues = append(item.Issues, "invalid:yaml")
			schemaViolations++
			checks = append(checks, item)
			continue
		}

		for _, key := range rule.RequiredKeys {
			if _, ok := payload[key]; !ok {
				item.Status = "fail"
				item.Issues = append(item.Issues, fmt.Sprintf("missing:key:%s", key))
				schemaViolations++
			}
		}

		checks = append(checks, item)
	}

	status := "pass"
	if missingArtifacts > 0 || schemaViolations > 0 {
		status = "fail"
	}

	report := v2DemoReadinessReport{
		Summary: v2DemoReadinessSummary{
			DemoRoot:         demoRoot,
			Checks:           len(rules),
			MissingArtifacts: missingArtifacts,
			SchemaViolations: schemaViolations,
			Status:           status,
		},
		Features: checks,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("v2-demo-readiness write report: %w", err)
	}

	if status != "pass" {
		return fmt.Errorf("v2-demo-readiness failed: missing_artifacts=%d schema_violations=%d, see %s", missingArtifacts, schemaViolations, reportPath)
	}

	return nil
}
