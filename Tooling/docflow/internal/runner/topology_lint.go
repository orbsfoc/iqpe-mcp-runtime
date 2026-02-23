package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type topologyFile struct {
	Repositories []map[string]any `yaml:"repositories"`
	Services     []map[string]any `yaml:"services"`
}

type topologyLintReport struct {
	Summary topologyLintSummary `yaml:"summary"`
	Issues  []string            `yaml:"issues,omitempty"`
}

type topologyLintSummary struct {
	TopologyPath  string `yaml:"topology_path"`
	Repositories  int    `yaml:"repositories"`
	Services      int    `yaml:"services"`
	ValidServices int    `yaml:"valid_services"`
	Issues        int    `yaml:"issues"`
	Status        string `yaml:"status"`
}

func TopologyLint(topologyPath, reportPath string) error {
	content, err := os.ReadFile(topologyPath)
	if err != nil {
		return fmt.Errorf("topology-lint read topology: %w", err)
	}

	var topology topologyFile
	if err := yaml.Unmarshal(content, &topology); err != nil {
		return fmt.Errorf("topology-lint parse topology: %w", err)
	}

	issues := make([]string, 0)
	if len(topology.Repositories) == 0 {
		issues = append(issues, "missing:repositories")
	}
	if len(topology.Services) == 0 {
		issues = append(issues, "missing:services")
	}

	repoSet := map[string]bool{}
	for idx, repo := range topology.Repositories {
		repoID := strings.TrimSpace(stringField(repo, "repo_id"))
		domain := strings.TrimSpace(stringField(repo, "domain"))
		if repoID == "" {
			issues = append(issues, fmt.Sprintf("missing:repo_id:index:%d", idx))
			continue
		}
		if repoSet[repoID] {
			issues = append(issues, fmt.Sprintf("duplicate:repo_id:%s", repoID))
			continue
		}
		repoSet[repoID] = true
		if domain == "" {
			issues = append(issues, fmt.Sprintf("missing:repo_domain:%s", repoID))
		}
	}

	allowedStatus := map[string]bool{
		"ACTIVE":     true,
		"DEPRECATED": true,
		"MIGRATING":  true,
	}

	validServices := 0
	for idx, service := range topology.Services {
		serviceID := strings.TrimSpace(stringField(service, "service_id"))
		if serviceID == "" {
			serviceID = strings.TrimSpace(stringField(service, "asset_id"))
		}
		repoID := strings.TrimSpace(stringField(service, "repo_id"))
		path := strings.TrimSpace(stringField(service, "path"))
		ownerTeam := strings.TrimSpace(stringField(service, "owner_team"))
		buildCommand := strings.TrimSpace(stringField(service, "build_command"))
		testCommand := strings.TrimSpace(stringField(service, "test_command"))
		runtimeRef := strings.TrimSpace(stringField(service, "runtime_reference"))
		if runtimeRef == "" {
			runtimeRef = strings.TrimSpace(stringField(service, "runtime_deploy_reference"))
		}
		status := strings.ToUpper(strings.TrimSpace(stringField(service, "status")))

		entryIssues := 0
		if serviceID == "" {
			issues = append(issues, fmt.Sprintf("missing:service_id:index:%d", idx))
			entryIssues++
		}
		if repoID == "" {
			issues = append(issues, fmt.Sprintf("missing:service_repo_id:%s", serviceID))
			entryIssues++
		} else if !repoSet[repoID] {
			issues = append(issues, fmt.Sprintf("unknown:repo_id:%s->%s", serviceID, repoID))
			entryIssues++
		}
		if path == "" {
			issues = append(issues, fmt.Sprintf("missing:path:%s", serviceID))
			entryIssues++
		}
		if ownerTeam == "" {
			issues = append(issues, fmt.Sprintf("missing:owner_team:%s", serviceID))
			entryIssues++
		}
		if buildCommand == "" {
			issues = append(issues, fmt.Sprintf("missing:build_command:%s", serviceID))
			entryIssues++
		}
		if testCommand == "" {
			issues = append(issues, fmt.Sprintf("missing:test_command:%s", serviceID))
			entryIssues++
		}
		if runtimeRef == "" {
			issues = append(issues, fmt.Sprintf("missing:runtime_reference:%s", serviceID))
			entryIssues++
		}
		if !hasNonEmptyField(service, "linked_architecture_ids") {
			issues = append(issues, fmt.Sprintf("missing:linked_architecture_ids:%s", serviceID))
			entryIssues++
		}
		if !hasNonEmptyField(service, "linked_runbook_paths") {
			issues = append(issues, fmt.Sprintf("missing:linked_runbook_paths:%s", serviceID))
			entryIssues++
		}
		if status == "" {
			issues = append(issues, fmt.Sprintf("missing:status:%s", serviceID))
			entryIssues++
		} else if !allowedStatus[status] {
			issues = append(issues, fmt.Sprintf("invalid:status:%s:%s", serviceID, status))
			entryIssues++
		}

		if entryIssues == 0 {
			validServices++
		}
	}

	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}

	report := topologyLintReport{
		Summary: topologyLintSummary{
			TopologyPath:  topologyPath,
			Repositories:  len(topology.Repositories),
			Services:      len(topology.Services),
			ValidServices: validServices,
			Issues:        len(issues),
			Status:        status,
		},
		Issues: issues,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("topology-lint write report: %w", err)
	}

	if status != "pass" {
		return fmt.Errorf("topology-lint failed: %d issue(s), see %s", len(issues), reportPath)
	}

	return nil
}
