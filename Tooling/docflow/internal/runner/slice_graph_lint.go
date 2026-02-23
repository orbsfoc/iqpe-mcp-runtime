package runner

import (
	"fmt"
	"os"
	"strings"

	iohelper "github.com/iqpe/docflow/internal/io"
	"gopkg.in/yaml.v3"
)

type sliceGraphFile struct {
	Sessions        []sliceSession     `yaml:"sessions"`
	ExecutionTracks executionTrackSets `yaml:"execution_tracks"`
}

type sliceSession struct {
	SessionID string   `yaml:"session_id"`
	StoryID   string   `yaml:"story_id"`
	Phase     string   `yaml:"phase"`
	DependsOn []string `yaml:"depends_on"`
	Status    string   `yaml:"status"`
}

type executionTrackSets struct {
	Sequenced  []string `yaml:"sequenced"`
	Concurrent []string `yaml:"concurrent"`
}

type sliceGraphLintReport struct {
	Summary sliceGraphLintSummary `yaml:"summary"`
	Issues  []string              `yaml:"issues"`
}

type sliceGraphLintSummary struct {
	GraphPath        string `yaml:"graph_path"`
	Sessions         int    `yaml:"sessions"`
	SequencedTracks  int    `yaml:"sequenced_tracks"`
	ConcurrentTracks int    `yaml:"concurrent_tracks"`
	Issues           int    `yaml:"issues"`
	Status           string `yaml:"status"`
}

func SliceGraphLint(graphPath, reportPath string) error {
	content, err := os.ReadFile(graphPath)
	if err != nil {
		return fmt.Errorf("slice-graph-lint read graph: %w", err)
	}

	var graph sliceGraphFile
	if err := yaml.Unmarshal(content, &graph); err != nil {
		return fmt.Errorf("slice-graph-lint parse graph: %w", err)
	}

	issues := make([]string, 0)
	if len(graph.Sessions) == 0 {
		issues = append(issues, "missing:sessions")
	}

	sessionSet := map[string]bool{}
	for _, session := range graph.Sessions {
		sid := strings.TrimSpace(session.SessionID)
		if sid == "" {
			issues = append(issues, "missing:session_id")
			continue
		}
		if sessionSet[sid] {
			issues = append(issues, fmt.Sprintf("duplicate:session_id:%s", sid))
		} else {
			sessionSet[sid] = true
		}
		for _, dep := range session.DependsOn {
			if strings.TrimSpace(dep) == sid {
				issues = append(issues, fmt.Sprintf("invalid:self_dependency:%s", sid))
			}
		}
	}

	trackSet := map[string]string{}
	for _, sid := range graph.ExecutionTracks.Sequenced {
		if strings.TrimSpace(sid) == "" {
			issues = append(issues, "missing:sequenced_track_session_id")
			continue
		}
		trackSet[sid] = "sequenced"
		if !sessionSet[sid] {
			issues = append(issues, fmt.Sprintf("unknown:sequenced_session:%s", sid))
		}
	}

	for _, sid := range graph.ExecutionTracks.Concurrent {
		if strings.TrimSpace(sid) == "" {
			issues = append(issues, "missing:concurrent_track_session_id")
			continue
		}
		if prev, ok := trackSet[sid]; ok {
			issues = append(issues, fmt.Sprintf("invalid:session_in_multiple_tracks:%s:%s+concurrent", sid, prev))
		}
		if !sessionSet[sid] {
			issues = append(issues, fmt.Sprintf("unknown:concurrent_session:%s", sid))
		}
	}

	for _, session := range graph.Sessions {
		for _, dep := range session.DependsOn {
			if !sessionSet[dep] {
				issues = append(issues, fmt.Sprintf("unknown:dependency:%s->%s", session.SessionID, dep))
			}
		}
	}

	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}

	report := sliceGraphLintReport{
		Summary: sliceGraphLintSummary{
			GraphPath:        graphPath,
			Sessions:         len(graph.Sessions),
			SequencedTracks:  len(graph.ExecutionTracks.Sequenced),
			ConcurrentTracks: len(graph.ExecutionTracks.Concurrent),
			Issues:           len(issues),
			Status:           status,
		},
		Issues: issues,
	}

	if err := iohelper.WriteYAML(reportPath, report); err != nil {
		return fmt.Errorf("slice-graph-lint write report: %w", err)
	}

	if len(issues) > 0 {
		return fmt.Errorf("slice-graph-lint failed: %d issue(s), see %s", len(issues), reportPath)
	}

	return nil
}
