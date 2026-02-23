package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iqpe/docflow/internal/contracts"
	iohelper "github.com/iqpe/docflow/internal/io"
	"github.com/iqpe/docflow/internal/scanner"
	"gopkg.in/yaml.v3"
)

var serviceIDPattern = regexp.MustCompile(`(?m)^-\s+(SVC-[A-Z]+-[0-9]+):`)
var generatedDocIDPattern = regexp.MustCompile(`^[A-Z0-9-]+-GEN-[A-Z0-9-]+$`)

type serviceImplementationLinksFile struct {
	ServiceImplementationLinks []serviceImplementationLink `yaml:"service_implementation_links"`
}

type serviceImplementationLink struct {
	ServiceID string   `yaml:"service_id"`
	Guides    []string `yaml:"guides"`
}

type contextBudgetPolicy struct {
	PolicyID             string                       `yaml:"policy_id"`
	Status               string                       `yaml:"status"`
	DefaultProfile       string                       `yaml:"default_profile"`
	Profiles             map[string]contextBudgetSpec `yaml:"profiles"`
	Rules                map[string]bool              `yaml:"rules"`
	RequiredReportFields []string                     `yaml:"required_report_fields"`
}

type contextBudgetSpec struct {
	MaxPromptTokens     int     `yaml:"max_prompt_tokens"`
	MaxCompletionTokens int     `yaml:"max_completion_tokens"`
	MaxLLMCalls         int     `yaml:"max_llm_calls"`
	MinToolToLLMRatio   float64 `yaml:"min_tool_to_llm_ratio"`
}

type frontendAgreementRegisterFile struct {
	AgreementRegister frontendAgreementRegister `yaml:"agreement_register"`
}

type frontendAgreementRegister struct {
	Pages []frontendPage `yaml:"pages"`
}

type frontendPage struct {
	PageID               string   `yaml:"page_id"`
	PageName             string   `yaml:"page_name"`
	Area                 string   `yaml:"area"`
	Status               string   `yaml:"status"`
	ProductSignoff       string   `yaml:"product_signoff"`
	TechnicalSignoff     string   `yaml:"technical_signoff"`
	ContentSpecRef       string   `yaml:"content_spec_ref"`
	VisualSpecRef        string   `yaml:"visual_spec_ref"`
	WireframeRef         string   `yaml:"wireframe_ref"`
	ServiceLinks         []string `yaml:"service_links"`
	ImplementationGuides []string `yaml:"implementation_guides"`
	Tags                 []string `yaml:"tags"`
}

type planningRegisterFile struct {
	PlanningRegister planningRegister `yaml:"planning_register"`
}

type planningRegister struct {
	Plans []planningPlan `yaml:"plans"`
}

type planningPlan struct {
	PlanID              string   `yaml:"plan_id"`
	FeatureIDs          []string `yaml:"feature_ids"`
	ADRIDs              []string `yaml:"adr_ids"`
	ServiceIDs          []string `yaml:"service_ids"`
	ComponentIDs        []string `yaml:"component_ids"`
	ImplementationUnits []string `yaml:"implementation_units"`
	Phase               string   `yaml:"phase"`
	ProductReview       string   `yaml:"product_review"`
	ArchitectureReview  string   `yaml:"architecture_review"`
	EngineeringReview   string   `yaml:"engineering_review"`
	Status              string   `yaml:"status"`
}

type validationOptions struct {
	ProductDocsRoot   string
	TechnicalDocsRoot string
}

type technicalCutoverRegisterFile struct {
	TechnicalDocsCutover technicalCutoverRegister `yaml:"technical_docs_cutover"`
}

type technicalCutoverRegister struct {
	Version       any                     `yaml:"version,omitempty"`
	GovernanceDoc string                  `yaml:"governance_doc,omitempty"`
	Entries       []technicalCutoverEntry `yaml:"entries"`
}

type technicalCutoverEntry struct {
	EntryID            string `yaml:"entry_id"`
	Source             string `yaml:"source"`
	Target             string `yaml:"target"`
	TechnicalPrimary   bool   `yaml:"technical_primary"`
	ProductShadowState string `yaml:"product_shadow_state"`
	CutoverStatus      string `yaml:"cutover_status"`
}

func Inventory(sourceRoot, out string) error {
	docs, err := scanner.Scan(sourceRoot)
	if err != nil {
		return err
	}
	inv := contracts.Inventory{
		Documents: docs,
		Stats: map[string]any{
			"total_docs": len(docs),
		},
		Conflicts: []map[string]any{},
	}
	return iohelper.WriteYAML(out, inv)
}

func Model(out string) error {
	model := contracts.Model{
		Taxonomy: []map[string]any{
			{"domain": "governance", "path_pattern": "Docs/RefactoredProductDocs/00-governance/"},
			{"domain": "product-intent", "path_pattern": "Docs/RefactoredProductDocs/01-product-intent/"},
			{"domain": "decisions", "path_pattern": "Docs/RefactoredProductDocs/02-decisions/"},
			{"domain": "services", "path_pattern": "Docs/RefactoredProductDocs/03-services/"},
			{"domain": "components", "path_pattern": "Docs/RefactoredProductDocs/04-components/"},
			{"domain": "environments", "path_pattern": "Docs/RefactoredProductDocs/05-environments/"},
			{"domain": "operations", "path_pattern": "Docs/RefactoredProductDocs/06-operations/"},
			{"domain": "roadmaps", "path_pattern": "Docs/RefactoredProductDocs/07-roadmaps/"},
		},
		LinkTypes: []string{"implements", "depends_on", "supersedes", "constrained_by", "affects_environment"},
		IDConventions: map[string]string{
			"feature":   "FEAT-<AREA>-<NNN>",
			"adr":       "ADR-<NNN>",
			"service":   "SVC-<DOMAIN>-<NN>",
			"component": "CMP-<SERVICE>-<NN>",
		},
	}
	return iohelper.WriteYAML(out, model)
}

func Validate(artifactsDir string) error {
	return ValidateWithRoots(artifactsDir, "../../Docs/RefactoredProductDocs", "../../Docs/RefactoredTechnicalDocs")
}

func ValidateWithRoots(artifactsDir, productDocsRoot, technicalDocsRoot string) error {
	options := validationOptions{
		ProductDocsRoot:   filepath.Clean(filepath.FromSlash(productDocsRoot)),
		TechnicalDocsRoot: filepath.Clean(filepath.FromSlash(technicalDocsRoot)),
	}

	required := []string{
		"p01-inventory.yaml",
		"p02-model.yaml",
		"p03-ownership.yaml",
		"p04-phase-model.yaml",
		"p05-input-integration.yaml",
		"p06-transform-index.yaml",
		"p07-validation.yaml",
		"p08-migration-roadmaps.yaml",
		"p09-go-mcp-blueprint.yaml",
	}

	missing := make([]string, 0)
	for _, file := range required {
		path := filepath.Join(artifactsDir, file)
		if !exists(path) {
			missing = append(missing, file)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required artifacts: %v", missing)
	}

	if err := validateServiceGuideCoverage(options); err != nil {
		return err
	}

	if err := validateContextBudgetPolicy(options); err != nil {
		return err
	}

	if err := validateFrontendPageAgreementRegister(options); err != nil {
		return err
	}

	if err := validateGeneratedDiagramFrontmatter(options); err != nil {
		return err
	}

	if err := validateIntentImplementationPlanning(options); err != nil {
		return err
	}

	if err := validateTechnicalDocsCutover(options); err != nil {
		return err
	}
	return nil
}

func validateServiceGuideCoverage(options validationOptions) error {
	technicalCatalogPath := filepath.Join(options.TechnicalDocsRoot, "00-architecture", "services", "service-catalog.md")
	productCatalogPath := filepath.Join(options.ProductDocsRoot, "03-services", "service-catalog.md")
	linksPath := filepath.Join(options.ProductDocsRoot, "artifacts", "service-tech-implementation-links.yaml")

	serviceCatalogPath := technicalCatalogPath
	if !exists(serviceCatalogPath) {
		serviceCatalogPath = productCatalogPath
	}

	catalogContent, err := os.ReadFile(serviceCatalogPath)
	if err != nil {
		return fmt.Errorf("read service catalog: %w", err)
	}

	matches := serviceIDPattern.FindAllStringSubmatch(string(catalogContent), -1)
	if len(matches) == 0 {
		return fmt.Errorf("no service IDs found in %s", serviceCatalogPath)
	}

	serviceIDs := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			serviceIDs = append(serviceIDs, match[1])
		}
	}

	linksContent, err := os.ReadFile(linksPath)
	if err != nil {
		return fmt.Errorf("read service implementation links: %w", err)
	}

	var links serviceImplementationLinksFile
	if err := yaml.Unmarshal(linksContent, &links); err != nil {
		return fmt.Errorf("parse service implementation links: %w", err)
	}

	guideByService := map[string]int{}
	for _, link := range links.ServiceImplementationLinks {
		guideByService[link.ServiceID] = len(link.Guides)
	}

	missingCoverage := make([]string, 0)
	for _, serviceID := range serviceIDs {
		if guideByService[serviceID] == 0 {
			missingCoverage = append(missingCoverage, serviceID)
		}
	}

	if len(missingCoverage) > 0 {
		return fmt.Errorf("services missing implementation guide coverage: %v", missingCoverage)
	}

	return nil
}

func validateContextBudgetPolicy(options validationOptions) error {
	policyPath := filepath.Join(options.ProductDocsRoot, "artifacts", "context-budget-policy.yaml")
	content, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("read context budget policy: %w", err)
	}

	var policy contextBudgetPolicy
	if err := yaml.Unmarshal(content, &policy); err != nil {
		return fmt.Errorf("parse context budget policy: %w", err)
	}

	if policy.PolicyID == "" {
		return fmt.Errorf("context budget policy missing policy_id")
	}
	if policy.DefaultProfile == "" {
		return fmt.Errorf("context budget policy missing default_profile")
	}
	if len(policy.Profiles) == 0 {
		return fmt.Errorf("context budget policy has no profiles")
	}
	selected, ok := policy.Profiles[policy.DefaultProfile]
	if !ok {
		return fmt.Errorf("context budget default_profile '%s' not found in profiles", policy.DefaultProfile)
	}
	if selected.MaxPromptTokens <= 0 || selected.MaxCompletionTokens <= 0 || selected.MaxLLMCalls <= 0 {
		return fmt.Errorf("context budget default profile has invalid limits")
	}
	if len(policy.RequiredReportFields) == 0 {
		return fmt.Errorf("context budget policy missing required_report_fields")
	}

	if policy.Rules["require_report_per_run"] {
		reportPath := filepath.Join(options.ProductDocsRoot, "artifacts", "context-budget-report.yaml")
		reportContent, err := os.ReadFile(reportPath)
		if err != nil {
			return fmt.Errorf("read context budget report: %w", err)
		}

		var report map[string]any
		if err := yaml.Unmarshal(reportContent, &report); err != nil {
			return fmt.Errorf("parse context budget report: %w", err)
		}

		missingFields := make([]string, 0)
		for _, field := range policy.RequiredReportFields {
			if _, exists := report[field]; !exists {
				missingFields = append(missingFields, field)
			}
		}
		if len(missingFields) > 0 {
			return fmt.Errorf("context budget report missing required fields: %v", missingFields)
		}
	}

	return nil
}

func validateFrontendPageAgreementRegister(options validationOptions) error {
	registerPath := filepath.Join(options.ProductDocsRoot, "artifacts", "frontend-page-agreement-register.yaml")
	content, err := os.ReadFile(registerPath)
	if err != nil {
		return fmt.Errorf("read frontend page agreement register: %w", err)
	}

	var registerFile frontendAgreementRegisterFile
	if err := yaml.Unmarshal(content, &registerFile); err != nil {
		return fmt.Errorf("parse frontend page agreement register: %w", err)
	}

	pages := registerFile.AgreementRegister.Pages
	if len(pages) == 0 {
		return fmt.Errorf("frontend page agreement register has no pages")
	}

	missing := make([]string, 0)
	gateViolations := make([]string, 0)

	for _, page := range pages {
		if strings.TrimSpace(page.PageID) == "" {
			missing = append(missing, "page missing page_id")
			continue
		}

		if strings.TrimSpace(page.PageName) == "" {
			missing = append(missing, fmt.Sprintf("%s missing page_name", page.PageID))
		}
		if strings.TrimSpace(page.Area) == "" {
			missing = append(missing, fmt.Sprintf("%s missing area", page.PageID))
		}
		if strings.TrimSpace(page.Status) == "" {
			missing = append(missing, fmt.Sprintf("%s missing status", page.PageID))
		}
		if len(page.Tags) == 0 {
			missing = append(missing, fmt.Sprintf("%s missing tags", page.PageID))
		}
		if strings.TrimSpace(page.ContentSpecRef) == "" {
			missing = append(missing, fmt.Sprintf("%s missing content_spec_ref", page.PageID))
		}
		if strings.TrimSpace(page.VisualSpecRef) == "" {
			missing = append(missing, fmt.Sprintf("%s missing visual_spec_ref", page.PageID))
		}
		if strings.TrimSpace(page.WireframeRef) == "" {
			missing = append(missing, fmt.Sprintf("%s missing wireframe_ref", page.PageID))
		}
		if len(page.ServiceLinks) == 0 {
			missing = append(missing, fmt.Sprintf("%s missing service_links", page.PageID))
		}
		if len(page.ImplementationGuides) == 0 {
			missing = append(missing, fmt.Sprintf("%s missing implementation_guides", page.PageID))
		}

		if page.Status == "accepted" {
			if page.ProductSignoff != "approved" {
				gateViolations = append(gateViolations, fmt.Sprintf("%s accepted but product_signoff is %s", page.PageID, page.ProductSignoff))
			}
			if page.TechnicalSignoff != "approved" {
				gateViolations = append(gateViolations, fmt.Sprintf("%s accepted but technical_signoff is %s", page.PageID, page.TechnicalSignoff))
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("frontend page register missing required fields: %v", missing)
	}
	if len(gateViolations) > 0 {
		return fmt.Errorf("frontend page acceptance gate violations: %v", gateViolations)
	}

	return nil
}

func validateGeneratedDiagramFrontmatter(options validationOptions) error {
	generatedDir := filepath.Join(options.ProductDocsRoot, "08-diagrams-generated")

	if !exists(generatedDir) {
		return fmt.Errorf("generated diagrams directory not found: %s", generatedDir)
	}

	missingFrontmatter := make([]string, 0)
	missingDocID := make([]string, 0)
	invalidDocID := make([]string, 0)
	duplicateDocID := make([]string, 0)
	docIDToPath := map[string]string{}
	err := filepath.WalkDir(generatedDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
		normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "\ufeff"))
		if !strings.HasPrefix(normalized, "---\n") {
			missingFrontmatter = append(missingFrontmatter, path)
			return nil
		}

		remainder := strings.TrimPrefix(normalized, "---\n")
		endIdx := strings.Index(remainder, "\n---\n")
		if endIdx < 0 {
			missingFrontmatter = append(missingFrontmatter, path)
			return nil
		}

		frontmatter := remainder[:endIdx]
		var metadata map[string]any
		if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
			missingDocID = append(missingDocID, fmt.Sprintf("%s (frontmatter parse error)", path))
			return nil
		}

		rawDocID, exists := metadata["doc_id"]
		docID, ok := rawDocID.(string)
		docID = strings.TrimSpace(docID)
		if !exists || !ok || docID == "" {
			missingDocID = append(missingDocID, path)
			return nil
		}

		if !generatedDocIDPattern.MatchString(docID) {
			invalidDocID = append(invalidDocID, fmt.Sprintf("%s (%s)", path, docID))
		}

		if existingPath, exists := docIDToPath[docID]; exists {
			duplicateDocID = append(duplicateDocID, fmt.Sprintf("%s (%s) duplicates %s", path, docID, existingPath))
		} else {
			docIDToPath[docID] = path
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scan generated diagrams: %w", err)
	}

	if len(missingFrontmatter) > 0 {
		return fmt.Errorf("generated diagram markdown missing YAML frontmatter: %v", missingFrontmatter)
	}
	if len(missingDocID) > 0 {
		return fmt.Errorf("generated diagram markdown missing doc_id: %v", missingDocID)
	}
	if len(invalidDocID) > 0 {
		return fmt.Errorf("generated diagram markdown doc_id must follow *-GEN-* convention: %v", invalidDocID)
	}
	if len(duplicateDocID) > 0 {
		return fmt.Errorf("generated diagram markdown has duplicate doc_id values: %v", duplicateDocID)
	}

	return nil
}

func validateIntentImplementationPlanning(options validationOptions) error {
	planningPath := filepath.Join(options.ProductDocsRoot, "artifacts", "intent-implementation-planning-register.yaml")
	content, err := os.ReadFile(planningPath)
	if err != nil {
		return fmt.Errorf("read planning register: %w", err)
	}

	var register planningRegisterFile
	if err := yaml.Unmarshal(content, &register); err != nil {
		return fmt.Errorf("parse planning register: %w", err)
	}

	plans := register.PlanningRegister.Plans
	if len(plans) == 0 {
		return fmt.Errorf("planning register has no plans")
	}

	missing := make([]string, 0)
	gateViolations := make([]string, 0)
	seenPlanIDs := map[string]bool{}

	for _, plan := range plans {
		planID := strings.TrimSpace(plan.PlanID)
		if planID == "" {
			missing = append(missing, "plan missing plan_id")
			continue
		}
		if seenPlanIDs[planID] {
			gateViolations = append(gateViolations, fmt.Sprintf("duplicate plan_id %s", planID))
			continue
		}
		seenPlanIDs[planID] = true

		if len(plan.FeatureIDs) == 0 {
			missing = append(missing, fmt.Sprintf("%s missing feature_ids", planID))
		}
		if len(plan.ADRIDs) == 0 {
			missing = append(missing, fmt.Sprintf("%s missing adr_ids", planID))
		}
		if len(plan.ServiceIDs) == 0 {
			missing = append(missing, fmt.Sprintf("%s missing service_ids", planID))
		}
		if len(plan.ComponentIDs) == 0 {
			missing = append(missing, fmt.Sprintf("%s missing component_ids", planID))
		}
		if len(plan.ImplementationUnits) == 0 {
			missing = append(missing, fmt.Sprintf("%s missing implementation_units", planID))
		}
		if strings.TrimSpace(plan.Phase) == "" {
			missing = append(missing, fmt.Sprintf("%s missing phase", planID))
		}
		if strings.TrimSpace(plan.ProductReview) == "" {
			missing = append(missing, fmt.Sprintf("%s missing product_review", planID))
		}
		if strings.TrimSpace(plan.ArchitectureReview) == "" {
			missing = append(missing, fmt.Sprintf("%s missing architecture_review", planID))
		}
		if strings.TrimSpace(plan.EngineeringReview) == "" {
			missing = append(missing, fmt.Sprintf("%s missing engineering_review", planID))
		}
		if strings.TrimSpace(plan.Status) == "" {
			missing = append(missing, fmt.Sprintf("%s missing status", planID))
		}

		if plan.Status == "ready-for-build" {
			if plan.ProductReview != "approved" {
				gateViolations = append(gateViolations, fmt.Sprintf("%s ready-for-build but product_review is %s", planID, plan.ProductReview))
			}
			if plan.ArchitectureReview != "approved" {
				gateViolations = append(gateViolations, fmt.Sprintf("%s ready-for-build but architecture_review is %s", planID, plan.ArchitectureReview))
			}
			if plan.EngineeringReview != "approved" {
				gateViolations = append(gateViolations, fmt.Sprintf("%s ready-for-build but engineering_review is %s", planID, plan.EngineeringReview))
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("planning register missing required fields: %v", missing)
	}
	if len(gateViolations) > 0 {
		return fmt.Errorf("planning register gate violations: %v", gateViolations)
	}

	return nil
}

func validateTechnicalDocsCutover(options validationOptions) error {
	registerPath := filepath.Join(options.ProductDocsRoot, "artifacts", "technical-docs-cutover-register.yaml")
	if !exists(registerPath) {
		return nil
	}

	content, err := os.ReadFile(registerPath)
	if err != nil {
		return fmt.Errorf("read technical cutover register: %w", err)
	}

	var register technicalCutoverRegisterFile
	if err := yaml.Unmarshal(content, &register); err != nil {
		return fmt.Errorf("parse technical cutover register: %w", err)
	}

	entries := register.TechnicalDocsCutover.Entries
	if len(entries) == 0 {
		return fmt.Errorf("technical cutover register has no entries")
	}

	missing := make([]string, 0)
	violations := make([]string, 0)
	seen := map[string]bool{}
	allowedShadowStates := map[string]bool{"shadow-active": true, "deprecated-ready": true, "removed": true}
	allowedCutoverStatuses := map[string]bool{"planned": true, "in-progress": true, "completed": true}

	for _, entry := range entries {
		entryID := strings.TrimSpace(entry.EntryID)
		if entryID == "" {
			missing = append(missing, "entry missing entry_id")
			continue
		}
		if seen[entryID] {
			violations = append(violations, fmt.Sprintf("duplicate entry_id %s", entryID))
			continue
		}
		seen[entryID] = true

		if strings.TrimSpace(entry.Source) == "" {
			missing = append(missing, fmt.Sprintf("%s missing source", entryID))
		}
		if strings.TrimSpace(entry.Target) == "" {
			missing = append(missing, fmt.Sprintf("%s missing target", entryID))
		}
		if strings.TrimSpace(entry.ProductShadowState) == "" {
			missing = append(missing, fmt.Sprintf("%s missing product_shadow_state", entryID))
		}
		if strings.TrimSpace(entry.CutoverStatus) == "" {
			missing = append(missing, fmt.Sprintf("%s missing cutover_status", entryID))
		}

		if !allowedShadowStates[entry.ProductShadowState] {
			violations = append(violations, fmt.Sprintf("%s invalid product_shadow_state %s", entryID, entry.ProductShadowState))
		}
		if !allowedCutoverStatuses[entry.CutoverStatus] {
			violations = append(violations, fmt.Sprintf("%s invalid cutover_status %s", entryID, entry.CutoverStatus))
		}

		targetPath := filepath.Clean(filepath.Join("../../", filepath.FromSlash(entry.Target)))
		if entry.TechnicalPrimary && !exists(targetPath) {
			violations = append(violations, fmt.Sprintf("%s technical primary target not found: %s", entryID, entry.Target))
		}

		sourcePath := filepath.Clean(filepath.Join("../../", filepath.FromSlash(entry.Source)))
		if entry.ProductShadowState == "removed" {
			if exists(sourcePath) {
				violations = append(violations, fmt.Sprintf("%s source should be removed but still exists: %s", entryID, entry.Source))
			}
		} else {
			if !exists(sourcePath) {
				violations = append(violations, fmt.Sprintf("%s source shadow file missing: %s", entryID, entry.Source))
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("technical cutover register missing required fields: %v", missing)
	}
	if len(violations) > 0 {
		return fmt.Errorf("technical cutover register violations: %v", violations)
	}

	return nil
}

func Run(sourceRoot, artifactsDir string) error {
	if err := Inventory(sourceRoot, filepath.Join(artifactsDir, "p01-inventory.generated.yaml")); err != nil {
		return err
	}
	if err := Model(filepath.Join(artifactsDir, "p02-model.generated.yaml")); err != nil {
		return err
	}
	return Validate(artifactsDir)
}
