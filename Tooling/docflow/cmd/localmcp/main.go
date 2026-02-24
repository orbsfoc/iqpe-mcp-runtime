package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolsListResult struct {
	Tools []toolSpec `json:"tools"`
}

type toolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type toolCallResult struct {
	Content []map[string]string `json:"content"`
}

type serverContext struct {
	Mode          string
	WorkspaceRoot string
}

type mcpActionFile struct {
	Actions []struct {
		ActionID string `yaml:"action_id"`
		Run      string `yaml:"run"`
	} `yaml:"actions"`
}

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

type cliOptions struct {
	Mode      string
	Workspace string
	SelfTest  bool
	Transport string
	Host      string
	Port      string
}

func parseCLIArgs(args []string) (cliOptions, error) {
	opts := cliOptions{
		Transport: "stdio",
		Host:      "127.0.0.1",
		Port:      "8080",
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--self-test":
			opts.SelfTest = true
		case arg == "--stdio":
			continue
		case strings.HasPrefix(arg, "--server="):
			opts.Mode = strings.TrimSpace(strings.TrimPrefix(arg, "--server="))
		case arg == "--server":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --server")
			}
			i++
			opts.Mode = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--workspace="):
			opts.Workspace = strings.TrimSpace(strings.TrimPrefix(arg, "--workspace="))
		case arg == "--workspace":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --workspace")
			}
			i++
			opts.Workspace = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--transport="):
			opts.Transport = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--transport=")))
		case arg == "--transport":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --transport")
			}
			i++
			opts.Transport = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--host="):
			opts.Host = strings.TrimSpace(strings.TrimPrefix(arg, "--host="))
		case arg == "--host":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --host")
			}
			i++
			opts.Host = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--port="):
			opts.Port = strings.TrimSpace(strings.TrimPrefix(arg, "--port="))
		case arg == "--port":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --port")
			}
			i++
			opts.Port = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--log-level="):
			continue
		case arg == "--log-level":
			if i+1 < len(args) {
				i++
			}
		default:
			continue
		}
	}

	if opts.Mode == "" {
		return opts, fmt.Errorf("missing --server")
	}
	if opts.Transport == "" {
		opts.Transport = "stdio"
	}
	if opts.Host == "" {
		opts.Host = "127.0.0.1"
	}
	if opts.Port == "" {
		opts.Port = "8080"
	}

	return opts, nil
}

func main() {
	opts, err := parseCLIArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	if opts.SelfTest {
		fmt.Printf("%s self-test ok\n", opts.Mode)
		return
	}

	root := opts.Workspace
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(2)
		}
		root = cwd
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	ctx := serverContext{Mode: opts.Mode, WorkspaceRoot: rootAbs}

	if strings.EqualFold(opts.Transport, "http") {
		if err := runHTTPServer(ctx, opts.Host, opts.Port); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		return
	}

	if err := runStdioServer(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func runStdioServer(ctx serverContext) error {
	server := buildMCPServer(ctx)
	transport := &mcp.StdioTransport{}
	return server.Run(context.Background(), transport)
}

func runHTTPServer(ctx serverContext, host, port string) error {
	server := buildMCPServer(ctx)
	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return server
	}, nil)

	addr := fmt.Sprintf("%s:%s", host, port)
	fmt.Fprintf(os.Stderr, "localmcp listening mode=%s transport=http addr=%s\n", ctx.Mode, addr)
	return http.ListenAndServe(addr, handler)
}

func buildMCPServer(ctx serverContext) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: ctx.Mode, Version: "0.2.0"}, nil)
	for _, spec := range toolsForMode(ctx.Mode) {
		name := spec.Name
		description := spec.Description
		schema := spec.InputSchema
		server.AddTool(&mcp.Tool{
			Name:        name,
			Description: description,
			InputSchema: schema,
		}, func(callCtx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, err := parseToolArgs(req)
			if err != nil {
				return nil, err
			}
			result, err := callTool(ctx, name, args)
			if err != nil {
				return nil, err
			}
			text := ""
			if len(result.Content) > 0 {
				text = result.Content[0]["text"]
			}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil
		})
	}
	return server
}

func parseToolArgs(req *mcp.CallToolRequest) (map[string]any, error) {
	if req == nil || req.Params == nil || len(req.Params.Arguments) == 0 {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	if args == nil {
		return map[string]any{}, nil
	}
	return args, nil
}

type contentLengthStdioTransport struct{}

func (t *contentLengthStdioTransport) Connect(context.Context) (mcp.Connection, error) {
	return &contentLengthStdioConn{
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
	}, nil
}

type contentLengthStdioConn struct {
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

func (c *contentLengthStdioConn) Read(context.Context) (jsonrpc.Message, error) {
	contentLength := -1
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			n, convErr := strconv.Atoi(strings.TrimSpace(parts[1]))
			if convErr != nil {
				return nil, convErr
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, body); err != nil {
		return nil, err
	}
	return jsonrpc.DecodeMessage(body)
}

func (c *contentLengthStdioConn) Write(_ context.Context, msg jsonrpc.Message) error {
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.writer.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))); err != nil {
		return err
	}
	if _, err := c.writer.Write(data); err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *contentLengthStdioConn) Close() error {
	return nil
}

func (c *contentLengthStdioConn) SessionID() string {
	return ""
}

func readRPCRequest(reader *bufio.Reader) (rpcRequest, error) {
	var req rpcRequest
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return req, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			n, convErr := strconv.Atoi(strings.TrimSpace(parts[1]))
			if convErr != nil {
				return req, convErr
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return req, fmt.Errorf("missing Content-Length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return req, err
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return req, err
	}
	return req, nil
}

func writeRPCResponse(writer *bufio.Writer, resp rpcResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := writer.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))); err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	return writer.Flush()
}

func handleRequest(ctx serverContext, req rpcRequest) rpcResponse {
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": ctx.Mode, "version": "0.1.0"},
		}
		return resp
	case "notifications/initialized":
		return resp
	case "tools/list":
		resp.Result = toolsListResult{Tools: toolsForMode(ctx.Mode)}
		return resp
	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &rpcError{Code: -32602, Message: err.Error()}
			return resp
		}
		result, err := callTool(ctx, params.Name, params.Arguments)
		if err != nil {
			resp.Error = &rpcError{Code: -32000, Message: err.Error()}
			return resp
		}
		resp.Result = result
		return resp
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found"}
		return resp
	}
}

func toolsForMode(mode string) []toolSpec {
	switch mode {
	case "repo-read":
		return []toolSpec{
			toolSpecFrom("list_dir", "List directory entries", "path"),
			toolSpecFrom("read_file", "Read file by line range", "path", "start_line", "end_line"),
			toolSpecFrom("grep_search", "Search regex under root", "root", "query"),
		}
	case "docflow-actions":
		return []toolSpec{
			toolSpecFrom("list_actions", "List configured MCP actions"),
			toolSpecFrom("run_action", "Run action by action_id; extra args are exposed as ENV vars", "action_id"),
			toolSpecFrom("run_script", "Run script from agent-tools/scripts", "script"),
			toolSpecFrom("list_templates", "List available templates and versions"),
			toolSpecFrom("get_template", "Get template by name and optional version", "name"),
			toolSpecFrom("list_skill_versions", "List agent skills and versions"),
			toolSpecFrom("check_skill_version", "Check skill version (defaults to current)", "skill_id"),
		}
	case "docs-graph":
		return []toolSpec{
			toolSpecFrom("queryImpacts", "Query docs impacted by IDs", "ids"),
			toolSpecFrom("getLatestApproved", "Get latest approved doc by type", "doc_type"),
		}
	case "policy":
		return []toolSpec{
			toolSpecFrom("validateOwnership", "Validate ownership metadata", "artifact_path"),
			toolSpecFrom("checkNonCloningRule", "Check non-cloning governance rule"),
		}
	default:
		return []toolSpec{}
	}
}

func callTool(ctx serverContext, name string, args map[string]any) (toolCallResult, error) {
	switch ctx.Mode {
	case "repo-read":
		return callRepoRead(ctx, name, args)
	case "docflow-actions":
		return callDocflowActions(ctx, name, args)
	case "docs-graph":
		return callDocsGraph(ctx, name, args)
	case "policy":
		return callPolicy(ctx, name, args)
	default:
		return toolCallResult{}, fmt.Errorf("unsupported server mode")
	}
}

func callRepoRead(ctx serverContext, name string, args map[string]any) (toolCallResult, error) {
	switch name {
	case "list_dir":
		rel := stringArg(args, "path")
		target, err := safePath(ctx.WorkspaceRoot, rel)
		if err != nil {
			return toolCallResult{}, err
		}
		entries, err := os.ReadDir(target)
		if err != nil {
			return toolCallResult{}, err
		}
		list := make([]string, 0, len(entries))
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() {
				name += "/"
			}
			list = append(list, name)
		}
		sort.Strings(list)
		payload, _ := json.MarshalIndent(list, "", "  ")
		return textResult(string(payload)), nil
	case "read_file":
		rel := stringArg(args, "path")
		start := intArg(args, "start_line", 1)
		end := intArg(args, "end_line", start)
		target, err := safePath(ctx.WorkspaceRoot, rel)
		if err != nil {
			return toolCallResult{}, err
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return toolCallResult{}, err
		}
		lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
		if start < 1 {
			start = 1
		}
		if end > len(lines) {
			end = len(lines)
		}
		if start > end {
			start = end
		}
		return textResult(strings.Join(lines[start-1:end], "\n")), nil
	case "grep_search":
		rootRel := stringArg(args, "root")
		query := stringArg(args, "query")
		root, err := safePath(ctx.WorkspaceRoot, rootRel)
		if err != nil {
			return toolCallResult{}, err
		}
		pattern, err := regexp.Compile("(?i)" + query)
		if err != nil {
			return toolCallResult{}, err
		}
		results := make([]map[string]any, 0)
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
			for i, line := range lines {
				if pattern.MatchString(line) {
					rel, _ := filepath.Rel(ctx.WorkspaceRoot, path)
					results = append(results, map[string]any{"path": filepath.ToSlash(rel), "line": i + 1, "text": line})
					if len(results) >= 100 {
						return io.EOF
					}
				}
			}
			return nil
		})
		payload, _ := json.MarshalIndent(results, "", "  ")
		return textResult(string(payload)), nil
	default:
		return toolCallResult{}, fmt.Errorf("unsupported tool: %s", name)
	}
}

func callDocflowActions(ctx serverContext, name string, args map[string]any) (toolCallResult, error) {
	actionsPath := filepath.Join(ctx.WorkspaceRoot, "Tooling", "agent-tools", "mcp-actions.yaml")
	scriptsDir := filepath.Join(ctx.WorkspaceRoot, "Tooling", "agent-tools", "scripts")
	templatesPath := filepath.Join(ctx.WorkspaceRoot, "Tooling", "agent-tools", "template-registry.yaml")
	skillVersionsPath := filepath.Join(ctx.WorkspaceRoot, "Tooling", "agent-skills", "skill-versions.yaml")
	actionMap, err := loadActions(actionsPath)
	if err != nil {
		return toolCallResult{}, err
	}
	switch name {
	case "list_actions":
		payload, _ := json.MarshalIndent(actionMap, "", "  ")
		return textResult(string(payload)), nil
	case "run_action":
		actionID := stringArg(args, "action_id")
		runCmd, ok := actionMap[actionID]
		if !ok {
			return toolCallResult{}, fmt.Errorf("unknown action_id")
		}
		env := map[string]string{}
		for key, value := range args {
			if key == "action_id" {
				continue
			}
			env[argKeyToEnv(key)] = fmt.Sprint(value)
		}
		return runCommandWithEnv(ctx.WorkspaceRoot, runCmd, env)
	case "run_script":
		scriptName := stringArg(args, "script")
		scriptPath := filepath.Clean(filepath.Join(scriptsDir, scriptName))
		if !strings.HasPrefix(scriptPath, scriptsDir) {
			return toolCallResult{}, fmt.Errorf("script outside allowed directory")
		}
		if _, statErr := os.Stat(scriptPath); statErr != nil {
			return toolCallResult{}, statErr
		}
		return runCommand(ctx.WorkspaceRoot, scriptPath)
	case "list_templates":
		templates, loadErr := loadTemplateRegistry(templatesPath)
		if loadErr != nil {
			return toolCallResult{}, loadErr
		}
		payload, _ := json.MarshalIndent(templates, "", "  ")
		return textResult(string(payload)), nil
	case "get_template":
		nameArg := stringArg(args, "name")
		versionArg := stringArg(args, "version")
		templates, loadErr := loadTemplateRegistry(templatesPath)
		if loadErr != nil {
			return toolCallResult{}, loadErr
		}
		selected, selErr := selectTemplate(templates, nameArg, versionArg)
		if selErr != nil {
			return toolCallResult{}, selErr
		}
		targetPath, pathErr := safePath(ctx.WorkspaceRoot, selected.Path)
		if pathErr != nil {
			return toolCallResult{}, pathErr
		}
		content, readErr := os.ReadFile(targetPath)
		if readErr != nil {
			return toolCallResult{}, readErr
		}
		payload, _ := json.MarshalIndent(map[string]any{
			"name":    selected.Name,
			"version": selected.Version,
			"path":    selected.Path,
			"latest":  selected.Latest,
			"content": string(content),
		}, "", "  ")
		return textResult(string(payload)), nil
	case "list_skill_versions":
		skills, loadErr := loadSkillVersions(skillVersionsPath)
		if loadErr != nil {
			return toolCallResult{}, loadErr
		}
		payload, _ := json.MarshalIndent(skills, "", "  ")
		return textResult(string(payload)), nil
	case "check_skill_version":
		skillID := stringArg(args, "skill_id")
		expected := stringArg(args, "expected_version")
		skills, loadErr := loadSkillVersions(skillVersionsPath)
		if loadErr != nil {
			return toolCallResult{}, loadErr
		}
		entry, ok := skills[skillID]
		if !ok {
			return toolCallResult{}, fmt.Errorf("unknown skill_id")
		}
		result := map[string]any{"skill_id": skillID, "name": entry.Name, "version": entry.Version, "ok": true}
		if expected != "" {
			result["expected_version"] = expected
			result["ok"] = strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(entry.Version))
		}
		payload, _ := json.MarshalIndent(result, "", "  ")
		return textResult(string(payload)), nil
	default:
		return toolCallResult{}, fmt.Errorf("unsupported tool: %s", name)
	}
}

func callDocsGraph(ctx serverContext, name string, args map[string]any) (toolCallResult, error) {
	files := collectDocsGraphFiles(ctx)

	switch name {
	case "queryImpacts":
		rawIDs, _ := args["ids"].([]any)
		ids := make([]string, 0)
		for _, item := range rawIDs {
			if text, ok := item.(string); ok {
				ids = append(ids, text)
			}
		}
		phase := strings.ToLower(stringArg(args, "phase"))
		affectedDocs := make([]string, 0)
		affectedDocDetails := make([]map[string]any, 0)
		svcSet := map[string]bool{}
		svcPattern := regexp.MustCompile(`SVC-[A-Z-]+-[0-9]+`)
		for _, path := range files {
			contentBytes, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			text := string(contentBytes)
			low := strings.ToLower(text)
			if phase != "" && !strings.Contains(low, phase) {
				continue
			}
			matched := false
			for _, id := range ids {
				if strings.Contains(text, id) {
					matched = true
					break
				}
			}
			if matched {
				rel, _ := filepath.Rel(ctx.WorkspaceRoot, path)
				relSlash := filepath.ToSlash(rel)
				affectedDocs = append(affectedDocs, relSlash)
				sourceMeta := docsSourceMeta(relSlash)
				affectedDocDetails = append(affectedDocDetails, map[string]any{
					"path":         relSlash,
					"source":       sourceMeta,
					"matched_ids":  matchedIDs(text, ids),
					"phase_filter": phase,
				})
				for _, svc := range svcPattern.FindAllString(text, -1) {
					svcSet[svc] = true
				}
			}
		}
		svcs := make([]string, 0, len(svcSet))
		for svc := range svcSet {
			svcs = append(svcs, svc)
		}
		sort.Strings(affectedDocs)
		sort.Strings(svcs)
		sort.SliceStable(affectedDocDetails, func(i, j int) bool {
			leftPath, _ := affectedDocDetails[i]["path"].(string)
			rightPath, _ := affectedDocDetails[j]["path"].(string)
			return leftPath < rightPath
		})
		payload, _ := json.MarshalIndent(map[string]any{
			"affected_docs":          affectedDocs,
			"affected_docs_metadata": affectedDocDetails,
			"affected_services":      svcs,
		}, "", "  ")
		return textResult(string(payload)), nil
	case "getLatestApproved":
		docType := stringArg(args, "doc_type")
		latest := map[string]any{"doc_id": "", "version": "", "owner": "", "path": "", "source": map[string]any{}}
		bestVersion := ""
		bestSourceRank := 999
		bestPath := ""
		for _, path := range files {
			contentBytes, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			meta := parseFrontmatterMap(string(contentBytes))
			if !strings.EqualFold(meta["doc_type"], docType) || !strings.EqualFold(meta["status"], "accepted") {
				continue
			}

			rel, _ := filepath.Rel(ctx.WorkspaceRoot, path)
			relSlash := filepath.ToSlash(rel)
			candidateVersion := strings.TrimSpace(meta["version"])
			candidateRank := docsSourcePriority(relSlash)

			shouldSelect := false
			if latest["path"] == "" {
				shouldSelect = true
			} else {
				versionCmp := compareVersion(candidateVersion, bestVersion)
				if versionCmp > 0 {
					shouldSelect = true
				} else if versionCmp == 0 {
					if candidateRank < bestSourceRank {
						shouldSelect = true
					} else if candidateRank == bestSourceRank && relSlash > bestPath {
						shouldSelect = true
					}
				}
			}

			if shouldSelect {
				rel, _ := filepath.Rel(ctx.WorkspaceRoot, path)
				relSlash := filepath.ToSlash(rel)
				sourceMeta := docsSourceMeta(relSlash)
				latest = map[string]any{
					"doc_id":  meta["doc_id"],
					"version": candidateVersion,
					"owner":   meta["owner_role"],
					"path":    relSlash,
					"source":  sourceMeta,
				}
				bestVersion = candidateVersion
				bestSourceRank = candidateRank
				bestPath = relSlash
			}
		}
		payload, _ := json.MarshalIndent(latest, "", "  ")
		return textResult(string(payload)), nil
	default:
		return toolCallResult{}, fmt.Errorf("unsupported tool: %s", name)
	}
}

func collectDocsGraphFiles(ctx serverContext) []string {
	roots := docsGraphRoots(ctx)
	files := make([]string, 0)
	seen := map[string]bool{}
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
				cleaned := filepath.Clean(path)
				if !seen[cleaned] {
					seen[cleaned] = true
					files = append(files, cleaned)
				}
			}
			return nil
		})
	}
	sort.Strings(files)
	return files
}

func docsGraphRoots(ctx serverContext) []string {
	configured := strings.TrimSpace(os.Getenv("DOCS_GRAPH_ROOTS"))
	defaultRoots := []string{
		"Docs/RefactoredProductDocs",
		"Docs/RefactoredTechnicalDocs",
		"Docs/POCProductDocs",
		".github/skills/local-mcp-setup/corporate-docs",
		"Docs/ConcretePOCProduct",
	}
	entries := make([]string, 0)
	if configured == "" {
		entries = append(entries, defaultRoots...)
		if docsRoot, err := safePath(ctx.WorkspaceRoot, "Docs"); err == nil {
			if info, statErr := os.Stat(docsRoot); statErr == nil && info.IsDir() {
				entries = append(entries, "Docs")
			}
		}
	} else {
		for _, item := range strings.Split(configured, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				entries = append(entries, trimmed)
			}
		}
	}

	roots := make([]string, 0)
	seen := map[string]bool{}
	for _, entry := range entries {
		var target string
		if filepath.IsAbs(entry) {
			cleaned := filepath.Clean(entry)
			rel, err := filepath.Rel(ctx.WorkspaceRoot, cleaned)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				continue
			}
			target = cleaned
		} else {
			resolved, err := safePath(ctx.WorkspaceRoot, entry)
			if err != nil {
				continue
			}
			target = resolved
		}
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			cleaned := filepath.Clean(target)
			if !seen[cleaned] {
				seen[cleaned] = true
				roots = append(roots, cleaned)
			}
		}
	}
	return roots
}

func docsSourcePriority(relPath string) int {
	normalized := strings.ToLower(filepath.ToSlash(strings.TrimSpace(relPath)))
	switch {
	case strings.HasPrefix(normalized, "docs/demoarchitecturedocs/"):
		return 0
	case strings.HasPrefix(normalized, "docs/refactoredproductdocs/"):
		return 1
	case strings.HasPrefix(normalized, "docs/refactoredtechnicaldocs/"):
		return 2
	case strings.HasPrefix(normalized, ".github/skills/local-mcp-setup/corporate-docs/"):
		return 3
	case strings.HasPrefix(normalized, "docs/pocproductdocs/"):
		return 4
	case strings.HasPrefix(normalized, "docs/concretepocproduct/"):
		return 5
	default:
		return 6
	}
}

func docsSourceMeta(relPath string) map[string]any {
	normalized := strings.ToLower(filepath.ToSlash(strings.TrimSpace(relPath)))
	meta := map[string]any{
		"source_id":    "workspace-other",
		"source_label": "Workspace Other",
		"source_root":  "",
		"source_rank":  6,
	}
	switch {
	case strings.HasPrefix(normalized, "docs/demoarchitecturedocs/"):
		meta["source_id"] = "demo-merged-architecture-docs"
		meta["source_label"] = "Demo Merged Architecture Docs"
		meta["source_root"] = "Docs/DemoArchitectureDocs"
		meta["source_rank"] = 0
	case strings.HasPrefix(normalized, "docs/refactoredproductdocs/"):
		meta["source_id"] = "refactored-product-docs"
		meta["source_label"] = "Refactored Product Docs"
		meta["source_root"] = "Docs/RefactoredProductDocs"
		meta["source_rank"] = 1
	case strings.HasPrefix(normalized, "docs/refactoredtechnicaldocs/"):
		meta["source_id"] = "refactored-technical-docs"
		meta["source_label"] = "Refactored Technical Docs"
		meta["source_root"] = "Docs/RefactoredTechnicalDocs"
		meta["source_rank"] = 2
	case strings.HasPrefix(normalized, ".github/skills/local-mcp-setup/corporate-docs/"):
		meta["source_id"] = "corporate-docs-bundle"
		meta["source_label"] = "Corporate Docs Bundle"
		meta["source_root"] = ".github/skills/local-mcp-setup/corporate-docs"
		meta["source_rank"] = 3
	case strings.HasPrefix(normalized, "docs/pocproductdocs/"):
		meta["source_id"] = "poc-product-docs"
		meta["source_label"] = "POC Product Docs"
		meta["source_root"] = "Docs/POCProductDocs"
		meta["source_rank"] = 4
	case strings.HasPrefix(normalized, "docs/concretepocproduct/"):
		meta["source_id"] = "concrete-poc-product"
		meta["source_label"] = "Concrete POC Product"
		meta["source_root"] = "Docs/ConcretePOCProduct"
		meta["source_rank"] = 5
	}
	meta["path"] = filepath.ToSlash(strings.TrimSpace(relPath))
	return meta
}

func matchedIDs(text string, ids []string) []string {
	matches := make([]string, 0)
	for _, id := range ids {
		if strings.Contains(text, id) {
			matches = append(matches, id)
		}
	}
	sort.Strings(matches)
	return matches
}

func callPolicy(ctx serverContext, name string, args map[string]any) (toolCallResult, error) {
	switch name {
	case "validateOwnership":
		rel := stringArg(args, "artifact_path")
		target, err := safePath(ctx.WorkspaceRoot, rel)
		if err != nil {
			return toolCallResult{}, err
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return toolCallResult{}, err
		}
		text := string(content)
		violations := make([]string, 0)
		if strings.HasSuffix(strings.ToLower(target), ".md") {
			if !strings.Contains(text, "owner_role:") {
				violations = append(violations, "missing owner_role")
			}
			if !strings.Contains(text, "accountable_role:") {
				violations = append(violations, "missing accountable_role")
			}
		} else {
			if !strings.Contains(text, "owner") && !strings.Contains(text, "owner_role") {
				violations = append(violations, "missing owner metadata")
			}
		}
		payload, _ := json.MarshalIndent(map[string]any{"ok": len(violations) == 0, "violations": violations}, "", "  ")
		return textResult(string(payload)), nil
	case "checkNonCloningRule":
		target := filepath.Join(ctx.WorkspaceRoot, "Docs", "RefactoredProductDocs", "00-governance", "mcp-server-hosting-and-data-sources.md")
		content, err := os.ReadFile(target)
		if err != nil {
			return toolCallResult{}, err
		}
		ok := strings.Contains(strings.ToLower(string(content)), "must not need to clone repositories")
		violations := []string{}
		if !ok {
			violations = append(violations, "non-cloning rule missing")
		}
		payload, _ := json.MarshalIndent(map[string]any{"ok": ok, "violations": violations}, "", "  ")
		return textResult(string(payload)), nil
	default:
		return toolCallResult{}, fmt.Errorf("unsupported tool: %s", name)
	}
}

func loadActions(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file mcpActionFile
	if err := yaml.Unmarshal(content, &file); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, action := range file.Actions {
		out[action.ActionID] = action.Run
	}
	return out, nil
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
		return templateEntry{}, fmt.Errorf("name is required")
	}
	candidates := make([]templateEntry, 0)
	for _, template := range templates {
		if strings.EqualFold(strings.TrimSpace(template.Name), strings.TrimSpace(name)) {
			candidates = append(candidates, template)
		}
	}
	if len(candidates) == 0 {
		return templateEntry{}, fmt.Errorf("template not found")
	}
	if strings.TrimSpace(version) != "" {
		for _, template := range candidates {
			if strings.EqualFold(strings.TrimSpace(template.Version), strings.TrimSpace(version)) {
				return template, nil
			}
		}
		return templateEntry{}, fmt.Errorf("template version not found")
	}
	for _, template := range candidates {
		if template.Latest {
			return template, nil
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return compareVersion(candidates[i].Version, candidates[j].Version) > 0
	})
	return candidates[0], nil
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

func runCommand(workspace, command string) (toolCallResult, error) {
	return runCommandWithEnv(workspace, command, map[string]string{})
}

func runCommandWithEnv(workspace, command string, extraEnv map[string]string) (toolCallResult, error) {
	cmd := exec.Command("bash", "-lc", command)
	cmd.Dir = workspace
	if len(extraEnv) > 0 {
		env := os.Environ()
		for key, value := range extraEnv {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	payload := map[string]any{
		"exit_code": 0,
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			payload["exit_code"] = exitErr.ExitCode()
		} else {
			payload["exit_code"] = 1
		}
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	return textResult(string(data)), nil
}

func parseFrontmatterMap(text string) map[string]string {
	out := map[string]string{}
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return out
	}
	rest := strings.TrimPrefix(normalized, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return out
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		if strings.Contains(line, ":") && !strings.HasPrefix(strings.TrimSpace(line), "-") {
			parts := strings.SplitN(line, ":", 2)
			out[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), "'\"")
		}
	}
	return out
}

func safePath(root, rel string) (string, error) {
	target := filepath.Clean(filepath.Join(root, rel))
	relPath, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside workspace")
	}
	return target, nil
}

func stringArg(args map[string]any, key string) string {
	if value, ok := args[key]; ok {
		if text, ok := value.(string); ok {
			return text
		}
	}
	return ""
}

func intArg(args map[string]any, key string, fallback int) int {
	if value, ok := args[key]; ok {
		switch typed := value.(type) {
		case float64:
			return int(typed)
		case int:
			return typed
		case string:
			if n, err := strconv.Atoi(typed); err == nil {
				return n
			}
		}
	}
	return fallback
}

func textResult(text string) toolCallResult {
	return toolCallResult{Content: []map[string]string{{"type": "text", "text": text}}}
}

func toolSpecFrom(name, desc string, required ...string) toolSpec {
	props := map[string]any{}
	requiredOut := make([]any, 0, len(required))
	for _, key := range required {
		requiredOut = append(requiredOut, key)
		props[key] = map[string]any{"type": "string"}
	}
	return toolSpec{
		Name:        name,
		Description: desc,
		InputSchema: map[string]any{"type": "object", "properties": props, "required": requiredOut},
	}
}

func argKeyToEnv(key string) string {
	upper := strings.ToUpper(strings.TrimSpace(key))
	buffer := strings.Builder{}
	lastUnderscore := false
	for _, r := range upper {
		valid := (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if valid {
			buffer.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			buffer.WriteRune('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(buffer.String(), "_")
	if result == "" {
		return "ARG"
	}
	return result
}

func toolsForModeCompat(mode string) []toolSpec {
	switch mode {
	case "repo-read":
		return []toolSpec{toolSpecFrom("list_dir", "List directory entries", "path"), toolSpecFrom("read_file", "Read file by line range", "path", "start_line", "end_line"), toolSpecFrom("grep_search", "Search regex under root", "root", "query")}
	case "docflow-actions":
		return []toolSpec{toolSpecFrom("list_actions", "List configured MCP actions"), toolSpecFrom("run_action", "Run action by action_id; extra args are exposed as ENV vars", "action_id"), toolSpecFrom("run_script", "Run script from agent-tools/scripts", "script"), toolSpecFrom("list_templates", "List available templates and versions"), toolSpecFrom("get_template", "Get template by name and optional version", "name"), toolSpecFrom("list_skill_versions", "List agent skills and versions"), toolSpecFrom("check_skill_version", "Check skill version (defaults to current)", "skill_id")}
	case "docs-graph":
		return []toolSpec{toolSpecFrom("queryImpacts", "Query docs impacted by IDs", "ids"), toolSpecFrom("getLatestApproved", "Get latest approved doc by type", "doc_type")}
	case "policy":
		return []toolSpec{toolSpecFrom("validateOwnership", "Validate ownership metadata", "artifact_path"), toolSpecFrom("checkNonCloningRule", "Check non-cloning governance rule")}
	default:
		return []toolSpec{}
	}
}
