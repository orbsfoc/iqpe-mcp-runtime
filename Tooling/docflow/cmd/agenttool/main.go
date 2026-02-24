package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type templateRegistryFile struct {
	Templates []templateEntry `yaml:"templates"`
}

type templateEntry struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Path    string `yaml:"path"`
	Latest  bool   `yaml:"latest"`
}

type skillVersionFile struct {
	Skills []skillVersionEntry `yaml:"skills"`
}

type skillVersionEntry struct {
	SkillID string `yaml:"skill_id"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type techDecisionCandidate struct {
	Value     string `json:"value"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	MatchedOn string `json:"matched_on"`
}

type techDetectionReport struct {
	SpecDir   string         `json:"spec_dir"`
	Detected  map[string]any `json:"detected"`
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp_utc"`
}

type mcpConfigFile struct {
	Servers map[string]mcpServerConfig `json:"servers"`
}

type mcpServerConfig struct {
	Transport string   `json:"transport"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: agenttool <command>")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "workflow-bootstrap":
		workflowBootstrap(os.Args[2:])
	case "workflow-preflight":
		workflowPreflight(os.Args[2:])
	case "spec-tech-detect":
		specTechDetect(os.Args[2:])
	case "template-list":
		templateList(os.Args[2:])
	case "template-get":
		templateGet(os.Args[2:])
	case "skill-version-list":
		skillVersionList(os.Args[2:])
	case "skill-version-check":
		skillVersionCheck(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(2)
	}
}

func workflowPreflight(args []string) {
	flags := flag.NewFlagSet("workflow-preflight", flag.ExitOnError)
	workspaceRoot := flags.String("workspace-root", ".", "workspace root")
	targetRoot := flags.String("target-root", ".", "target project root")
	specDir := flags.String("spec-dir", "", "spec directory")
	outPath := flags.String("out", "", "output JSON path")
	_ = flags.Parse(args)

	if strings.TrimSpace(*specDir) == "" {
		fail(errors.New("spec-dir is required"))
	}

	resolvedTarget, err := filepath.Abs(*targetRoot)
	if err != nil {
		fail(err)
	}
	resolvedSpecDir := *specDir
	if !filepath.IsAbs(resolvedSpecDir) {
		resolvedSpecDir = filepath.Join(resolvedTarget, resolvedSpecDir)
	}
	resolvedSpecDir, err = filepath.Abs(resolvedSpecDir)
	if err != nil {
		fail(err)
	}

	mcpPath := filepath.Join(resolvedTarget, ".vscode", "mcp.json")
	mcpServersPresent := false
	mcpUsesLocalBinary := false
	mcpConfiguredCommandRunnable := false
	mcpConfiguredCommand := ""
	mcpParseError := ""

	requiredServers := []string{"repo-read-local", "docflow-actions-local", "docs-graph-local", "policy-local"}
	if data, readErr := os.ReadFile(mcpPath); readErr == nil {
		var cfg mcpConfigFile
		if err := json.Unmarshal(data, &cfg); err != nil {
			mcpParseError = err.Error()
		} else {
			presentCount := 0
			localBinaryCount := 0
			runnableCount := 0
			for _, serverName := range requiredServers {
				server, ok := cfg.Servers[serverName]
				if !ok {
					continue
				}
				presentCount++
				if strings.Contains(strings.ToLower(server.Command), "iqpe-localmcp") {
					localBinaryCount++
				}
				if commandRunnable(server.Command) {
					runnableCount++
				}
				if serverName == "repo-read-local" {
					mcpConfiguredCommand = strings.TrimSpace(server.Command)
				}
			}
			mcpServersPresent = presentCount == len(requiredServers)
			mcpUsesLocalBinary = localBinaryCount == len(requiredServers)
			mcpConfiguredCommandRunnable = runnableCount == len(requiredServers)
		}
	}

	localBinary := strings.TrimSpace(os.Getenv("MCP_LOCAL_BINARY"))
	if localBinary == "" {
		localBinary = "iqpe-localmcp"
	}
	_, binaryErr := exec.LookPath(localBinary)
	binaryOnPath := binaryErr == nil

	mcpOK := mcpServersPresent && mcpUsesLocalBinary && mcpConfiguredCommandRunnable

	specOK := false
	specCount := 0
	if stat, statErr := os.Stat(resolvedSpecDir); statErr == nil && stat.IsDir() {
		specCount = countSpecFiles(resolvedSpecDir)
		specOK = specCount > 0
	}

	status := "PASS"
	if !mcpOK || !specOK {
		status = "BLOCKED"
	}

	targetOut := strings.TrimSpace(*outPath)
	if targetOut == "" {
		targetOut = filepath.Join(*workspaceRoot, "docs", "tooling", "workflow-preflight.json")
	}
	if !filepath.IsAbs(targetOut) {
		targetOut = filepath.Join(*workspaceRoot, targetOut)
	}
	if err := os.MkdirAll(filepath.Dir(targetOut), 0o755); err != nil {
		fail(err)
	}

	report := map[string]any{
		"status":                      status,
		"timestamp_utc":               time.Now().UTC().Format(time.RFC3339),
		"target_root":                 resolvedTarget,
		"spec_dir":                    resolvedSpecDir,
		"mcp_config_path":             filepath.ToSlash(mcpPath),
		"mcp_ready":                   mcpOK,
		"mcp_servers_present":         mcpServersPresent,
		"mcp_uses_local_binary":       mcpUsesLocalBinary,
		"mcp_config_parse_error":      mcpParseError,
		"mcp_config_command":          mcpConfiguredCommand,
		"mcp_config_command_runnable": mcpConfiguredCommandRunnable,
		"mcp_local_binary":            localBinary,
		"mcp_local_binary_on_path":    binaryOnPath,
		"spec_ready":                  specOK,
		"spec_file_count":             specCount,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(targetOut, data, 0o644); err != nil {
		fail(err)
	}

	printJSON(map[string]any{"status": status, "report": filepath.ToSlash(targetOut)})
}

func specTechDetect(args []string) {
	flags := flag.NewFlagSet("spec-tech-detect", flag.ExitOnError)
	workspaceRoot := flags.String("workspace-root", ".", "workspace root")
	specDir := flags.String("spec-dir", "", "spec directory")
	outPath := flags.String("out", "", "output JSON path")
	_ = flags.Parse(args)

	if strings.TrimSpace(*specDir) == "" {
		fail(errors.New("spec-dir is required"))
	}

	resolvedSpecDir := *specDir
	if !filepath.IsAbs(resolvedSpecDir) {
		resolvedSpecDir = filepath.Join(*workspaceRoot, resolvedSpecDir)
	}
	resolvedSpecDir, err := filepath.Abs(resolvedSpecDir)
	if err != nil {
		fail(err)
	}
	if stat, err := os.Stat(resolvedSpecDir); err != nil || !stat.IsDir() {
		fail(fmt.Errorf("spec-dir not found or not a directory: %s", resolvedSpecDir))
	}

	type keyword struct {
		value  string
		regexp *regexp.Regexp
	}
	compile := func(pattern string) *regexp.Regexp {
		return regexp.MustCompile("(?i)" + pattern)
	}

	backendMatchers := []keyword{{"golang", compile(`\bgo(lang)?\b`)}, {"node", compile(`\bnode(js)?\b`)}, {"java", compile(`\bjava\b`)}, {"dotnet", compile(`\.net|dotnet`)}}
	frontendMatchers := []keyword{{"react", compile(`\breact\b`)}, {"vue", compile(`\bvue\b`)}, {"angular", compile(`\bangular\b`)}}
	dbMatchers := []keyword{{"postgres", compile(`\bpostgres(ql)?\b`)}, {"sqlite", compile(`\bsqlite\b`)}, {"mysql", compile(`\bmysql\b`)}, {"mssql", compile(`\bms\s*sql|sql\s*server\b`)}}
	migrationMatchers := []keyword{{"flyway", compile(`\bflyway\b`)}, {"liquibase", compile(`\bliquibase\b`)}, {"golang-migrate", compile(`\bmigrate\b`)}}

	findFirst := func(filePath, line string, lineNo int, patterns []keyword) *techDecisionCandidate {
		for _, item := range patterns {
			if item.regexp.MatchString(line) {
				return &techDecisionCandidate{Value: item.value, File: filepath.ToSlash(filePath), Line: lineNo, MatchedOn: strings.TrimSpace(line)}
			}
		}
		return nil
	}

	var backend, frontend, database, migration *techDecisionCandidate
	allowedExt := map[string]bool{".md": true, ".yaml": true, ".yml": true, ".json": true, ".txt": true}

	_ = filepath.WalkDir(resolvedSpecDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !allowedExt[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer file.Close()

		rel, _ := filepath.Rel(resolvedSpecDir, path)
		scanner := bufio.NewScanner(file)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			text := scanner.Text()
			if backend == nil {
				backend = findFirst(rel, text, lineNo, backendMatchers)
			}
			if frontend == nil {
				frontend = findFirst(rel, text, lineNo, frontendMatchers)
			}
			if database == nil {
				database = findFirst(rel, text, lineNo, dbMatchers)
			}
			if migration == nil {
				migration = findFirst(rel, text, lineNo, migrationMatchers)
			}
			if backend != nil && frontend != nil && database != nil && migration != nil {
				return io.EOF
			}
		}
		return nil
	})

	detected := map[string]any{}
	if backend != nil {
		detected["backend_runtime"] = backend
	}
	if frontend != nil {
		detected["frontend_framework"] = frontend
	}
	if database != nil {
		detected["persistent_engine"] = database
	}
	if migration != nil {
		detected["migration_tool"] = migration
	}

	status := "PASS"
	if backend == nil || frontend == nil || database == nil {
		status = "BLOCKED"
	}
	report := techDetectionReport{SpecDir: resolvedSpecDir, Detected: detected, Status: status, Timestamp: time.Now().UTC().Format(time.RFC3339)}

	targetOut := strings.TrimSpace(*outPath)
	if targetOut == "" {
		targetOut = filepath.Join(*workspaceRoot, "docs", "tooling", "spec-tech-detect.json")
	}
	if !filepath.IsAbs(targetOut) {
		targetOut = filepath.Join(*workspaceRoot, targetOut)
	}
	if err := os.MkdirAll(filepath.Dir(targetOut), 0o755); err != nil {
		fail(err)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(targetOut, data, 0o644); err != nil {
		fail(err)
	}

	printJSON(map[string]any{"status": status, "report": filepath.ToSlash(targetOut), "detected": detected})
}

func workflowBootstrap(args []string) {
	flags := flag.NewFlagSet("workflow-bootstrap", flag.ExitOnError)
	workspaceRoot := flags.String("workspace-root", ".", "workspace root")
	targetRoot := flags.String("target-root", ".", "target project root")
	specDir := flags.String("spec-dir", "", "spec dir for this run")
	_ = flags.Parse(args)

	resolvedWorkspace, err := filepath.Abs(*workspaceRoot)
	if err != nil {
		fail(err)
	}
	resolvedTarget, err := filepath.Abs(*targetRoot)
	if err != nil {
		fail(err)
	}

	promptsSource := filepath.Join(resolvedWorkspace, "prompts", "productWorkflowPack")
	if _, err := os.Stat(promptsSource); err != nil {
		fail(fmt.Errorf("missing prompt source: %s", promptsSource))
	}

	workflowTarget := filepath.Join(resolvedTarget, ".iqpe-workflow", "productWorkflowPack")
	if err := os.RemoveAll(workflowTarget); err != nil {
		fail(err)
	}
	if err := copyDir(promptsSource, workflowTarget); err != nil {
		fail(err)
	}

	vscodeDir := filepath.Join(resolvedTarget, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0o755); err != nil {
		fail(err)
	}
	mcpTarget := filepath.Join(vscodeDir, "mcp.json")
	mcpCommand, mcpCommandResolved := resolveLocalMCPCommand()
	if _, err := os.Stat(mcpTarget); errors.Is(err, os.ErrNotExist) {
		if err := writeMCPConfig(mcpTarget, mcpCommand); err != nil {
			fail(err)
		}
	}

	reportPath := filepath.Join(resolvedTarget, "docs", "tooling", "bootstrap-report.md")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		fail(err)
	}
	report := strings.Join([]string{
		"# Workflow Bootstrap Report",
		"",
		fmt.Sprintf("- Timestamp (UTC): %s", time.Now().UTC().Format(time.RFC3339)),
		fmt.Sprintf("- Workspace root: %s", resolvedWorkspace),
		fmt.Sprintf("- Target root: %s", resolvedTarget),
		fmt.Sprintf("- SPEC_DIR: %s", defaultValue(*specDir, "<unset>")),
		"",
		"## Applied changes",
		"- Copied prompt pack to `.iqpe-workflow/productWorkflowPack`",
		fmt.Sprintf("- Ensured `.vscode/mcp.json` exists (command: %s)", mcpCommand),
		fmt.Sprintf("- MCP command resolved on system: %t", mcpCommandResolved),
		"- Generated this report",
		"",
		"## Next steps",
		"1) Validate SPEC_DIR via MCP/server checks",
		"2) Start with orchestrator prompt in `.iqpe-workflow/productWorkflowPack/00-orchestrator.md`",
		"3) Run governance readiness checklist",
	}, "\n") + "\n"
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		fail(err)
	}

	printJSON(map[string]any{
		"status":           "PASS",
		"workspace_root":   resolvedWorkspace,
		"target_root":      resolvedTarget,
		"spec_dir":         defaultValue(*specDir, "<unset>"),
		"prompt_pack_path": filepath.ToSlash(filepath.Join(".iqpe-workflow", "productWorkflowPack")),
		"mcp_config_path":  filepath.ToSlash(filepath.Join(".vscode", "mcp.json")),
		"report_path":      filepath.ToSlash(filepath.Join("docs", "tooling", "bootstrap-report.md")),
	})
}

func templateList(args []string) {
	flags := flag.NewFlagSet("template-list", flag.ExitOnError)
	workspaceRoot := flags.String("workspace-root", ".", "workspace root")
	_ = flags.Parse(args)

	registryPath := filepath.Join(*workspaceRoot, "Tooling", "agent-tools", "template-registry.yaml")
	templates, err := loadTemplateRegistry(registryPath)
	if err != nil {
		fail(err)
	}
	printJSON(templates)
}

func templateGet(args []string) {
	flags := flag.NewFlagSet("template-get", flag.ExitOnError)
	workspaceRoot := flags.String("workspace-root", ".", "workspace root")
	name := flags.String("name", "", "template name")
	version := flags.String("version", "", "template version")
	_ = flags.Parse(args)

	registryPath := filepath.Join(*workspaceRoot, "Tooling", "agent-tools", "template-registry.yaml")
	templates, err := loadTemplateRegistry(registryPath)
	if err != nil {
		fail(err)
	}
	selected, err := selectTemplate(templates, *name, *version)
	if err != nil {
		fail(err)
	}

	targetPath := filepath.Join(*workspaceRoot, filepath.FromSlash(selected.Path))
	content, err := os.ReadFile(targetPath)
	if err != nil {
		fail(err)
	}

	printJSON(map[string]any{
		"name":    selected.Name,
		"version": selected.Version,
		"path":    selected.Path,
		"latest":  selected.Latest,
		"content": string(content),
	})
}

func skillVersionList(args []string) {
	flags := flag.NewFlagSet("skill-version-list", flag.ExitOnError)
	workspaceRoot := flags.String("workspace-root", ".", "workspace root")
	_ = flags.Parse(args)

	versionsPath := filepath.Join(*workspaceRoot, "Tooling", "agent-skills", "skill-versions.yaml")
	entries, err := loadSkillVersions(versionsPath)
	if err != nil {
		fail(err)
	}
	printJSON(entries)
}

func skillVersionCheck(args []string) {
	flags := flag.NewFlagSet("skill-version-check", flag.ExitOnError)
	workspaceRoot := flags.String("workspace-root", ".", "workspace root")
	skillID := flags.String("skill-id", "", "skill id")
	expected := flags.String("expected-version", "", "expected version")
	_ = flags.Parse(args)

	if strings.TrimSpace(*skillID) == "" {
		fail(errors.New("skill-id is required"))
	}

	versionsPath := filepath.Join(*workspaceRoot, "Tooling", "agent-skills", "skill-versions.yaml")
	entries, err := loadSkillVersions(versionsPath)
	if err != nil {
		fail(err)
	}

	entry, ok := entries[*skillID]
	if !ok {
		fail(errors.New("unknown skill-id"))
	}

	payload := map[string]any{"skill_id": entry.SkillID, "name": entry.Name, "version": entry.Version, "ok": true}
	if strings.TrimSpace(*expected) != "" {
		payload["expected_version"] = *expected
		payload["ok"] = strings.EqualFold(strings.TrimSpace(*expected), strings.TrimSpace(entry.Version))
	}
	printJSON(payload)
}

func loadTemplateRegistry(path string) ([]templateEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file templateRegistryFile
	if err := yaml.Unmarshal(content, &file); err != nil {
		return nil, err
	}
	return file.Templates, nil
}

func selectTemplate(templates []templateEntry, name, version string) (templateEntry, error) {
	if strings.TrimSpace(name) == "" {
		return templateEntry{}, errors.New("name is required")
	}
	matches := make([]templateEntry, 0)
	for _, item := range templates {
		if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(name)) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return templateEntry{}, errors.New("template not found")
	}
	if strings.TrimSpace(version) != "" {
		for _, item := range matches {
			if strings.EqualFold(strings.TrimSpace(item.Version), strings.TrimSpace(version)) {
				return item, nil
			}
		}
		return templateEntry{}, errors.New("template version not found")
	}
	for _, item := range matches {
		if item.Latest {
			return item, nil
		}
	}
	sort.Slice(matches, func(i, j int) bool { return compareVersion(matches[i].Version, matches[j].Version) > 0 })
	return matches[0], nil
}

func compareVersion(left, right string) int {
	parse := func(raw string) []int {
		cleaned := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(raw), "v"))
		parts := strings.Split(cleaned, ".")
		out := make([]int, 0, len(parts))
		for _, part := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil {
				out = append(out, 0)
			} else {
				out = append(out, n)
			}
		}
		return out
	}

	a := parse(left)
	b := parse(right)
	limit := len(a)
	if len(b) > limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		av := 0
		bv := 0
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

func loadSkillVersions(path string) (map[string]skillVersionEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file skillVersionFile
	if err := yaml.Unmarshal(content, &file); err != nil {
		return nil, err
	}
	out := map[string]skillVersionEntry{}
	for _, entry := range file.Skills {
		out[entry.SkillID] = entry
	}
	return out, nil
}

func printJSON(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(data))
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(0o644)
}

func defaultValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func resolveLocalMCPCommand() (string, bool) {
	if path, err := exec.LookPath("iqpe-localmcp"); err == nil {
		return path, true
	}
	home, err := os.UserHomeDir()
	if err == nil {
		candidates := []string{
			filepath.Join(home, "bin", "iqpe-localmcp"),
			filepath.Join(home, ".local", "bin", "iqpe-localmcp"),
		}
		for _, candidate := range candidates {
			if isExecutableFile(candidate) {
				return candidate, true
			}
		}
	}
	return "iqpe-localmcp", false
}

func commandRunnable(command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return false
	}
	if filepath.IsAbs(trimmed) || strings.Contains(trimmed, string(os.PathSeparator)) {
		return isExecutableFile(trimmed)
	}
	_, err := exec.LookPath(trimmed)
	return err == nil
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}

func writeMCPConfig(path, command string) error {
	config := mcpConfigFile{Servers: map[string]mcpServerConfig{
		"repo-read-local": {
			Transport: "stdio",
			Command:   command,
			Args:      []string{"--server", "repo-read"},
		},
		"docflow-actions-local": {
			Transport: "stdio",
			Command:   command,
			Args:      []string{"--server", "docflow-actions"},
		},
		"docs-graph-local": {
			Transport: "stdio",
			Command:   command,
			Args:      []string{"--server", "docs-graph"},
		},
		"policy-local": {
			Transport: "stdio",
			Command:   command,
			Args:      []string{"--server", "policy"},
		},
	}}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func countSpecFiles(root string) int {
	count := 0
	allowed := map[string]bool{".md": true, ".yaml": true, ".yml": true, ".json": true, ".txt": true}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if allowed[strings.ToLower(filepath.Ext(path))] {
			count++
		}
		return nil
	})
	return count
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(2)
}
