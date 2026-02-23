package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/iqpe/docflow/internal/diagrams"
	"github.com/iqpe/docflow/internal/runner"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: docflow <inventory|model|validate|run|diagrams|frontend-diagrams|metadata-lint|planning-lint|shadow-lint|cutover-progress|wave3-readiness|wave3-remediation|wave3-remove-shadows|wave3-closure|wave4-bootstrap|v2-demo-readiness> [flags]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "inventory":
		cmd := flag.NewFlagSet("inventory", flag.ExitOnError)
		source := cmd.String("source", "", "source docs root")
		out := cmd.String("out", "", "output yaml file")
		_ = cmd.Parse(os.Args[2:])
		must(*source != "" && *out != "", "inventory requires --source and --out")
		mustNoErr(runner.Inventory(*source, *out))
	case "model":
		cmd := flag.NewFlagSet("model", flag.ExitOnError)
		out := cmd.String("out", "", "output yaml file")
		_ = cmd.Parse(os.Args[2:])
		must(*out != "", "model requires --out")
		mustNoErr(runner.Model(*out))
	case "validate":
		cmd := flag.NewFlagSet("validate", flag.ExitOnError)
		artifacts := cmd.String("artifacts", "", "artifacts directory")
		productDocsRoot := cmd.String("product-docs-root", "../../Docs/RefactoredProductDocs", "product docs root")
		technicalDocsRoot := cmd.String("technical-docs-root", "../../Docs/RefactoredTechnicalDocs", "technical docs root")
		_ = cmd.Parse(os.Args[2:])
		must(*artifacts != "", "validate requires --artifacts")
		mustNoErr(runner.ValidateWithRoots(*artifacts, *productDocsRoot, *technicalDocsRoot))
	case "run":
		cmd := flag.NewFlagSet("run", flag.ExitOnError)
		source := cmd.String("source", "", "source docs root")
		artifacts := cmd.String("artifacts", "", "artifacts directory")
		_ = cmd.Parse(os.Args[2:])
		must(*source != "" && *artifacts != "", "run requires --source and --artifacts")
		mustNoErr(runner.Run(*source, *artifacts))
	case "diagrams":
		cmd := flag.NewFlagSet("diagrams", flag.ExitOnError)
		artifacts := cmd.String("artifacts", "", "artifacts directory (contains adr-links.yaml)")
		out := cmd.String("out", "", "output directory for generated diagram markdown")
		_ = cmd.Parse(os.Args[2:])
		must(*artifacts != "" && *out != "", "diagrams requires --artifacts and --out")
		mustNoErr(diagrams.Generate(*artifacts, *out))
	case "frontend-diagrams":
		cmd := flag.NewFlagSet("frontend-diagrams", flag.ExitOnError)
		artifacts := cmd.String("artifacts", "", "artifacts directory (contains frontend-page-architecture-tags.yaml)")
		out := cmd.String("out", "", "output directory for generated frontend diagram markdown")
		_ = cmd.Parse(os.Args[2:])
		must(*artifacts != "" && *out != "", "frontend-diagrams requires --artifacts and --out")
		mustNoErr(diagrams.GenerateFrontend(*artifacts, *out))
	case "metadata-lint":
		cmd := flag.NewFlagSet("metadata-lint", flag.ExitOnError)
		docsRoot := cmd.String("docs-root", "../../Docs/RefactoredProductDocs", "docs root to lint")
		schema := cmd.String("schema", "../../Docs/RefactoredProductDocs/contracts/doc-metadata.schema.yaml", "metadata schema yaml")
		report := cmd.String("report", "../../Docs/RefactoredProductDocs/artifacts/metadata-lint-report.yaml", "output lint report path")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.MetadataLint(*docsRoot, *schema, *report))
	case "planning-lint":
		cmd := flag.NewFlagSet("planning-lint", flag.ExitOnError)
		register := cmd.String("register", "../../Docs/RefactoredProductDocs/artifacts/intent-implementation-planning-register.yaml", "planning register yaml")
		report := cmd.String("report", "../../Docs/RefactoredProductDocs/artifacts/planning-lint-report.yaml", "output planning lint report path")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.PlanningLint(*register, *report))
	case "shadow-lint":
		cmd := flag.NewFlagSet("shadow-lint", flag.ExitOnError)
		register := cmd.String("register", "../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml", "technical cutover register yaml")
		report := cmd.String("report", "../../Docs/RefactoredProductDocs/artifacts/shadow-lint-report.yaml", "output shadow lint report path")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.ShadowLint(*register, *report))
	case "cutover-progress":
		cmd := flag.NewFlagSet("cutover-progress", flag.ExitOnError)
		register := cmd.String("register", "../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml", "technical cutover register yaml")
		report := cmd.String("report", "../../Docs/RefactoredProductDocs/artifacts/cutover-progress-report.yaml", "output cutover progress report path")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.CutoverProgress(*register, *report))
	case "wave3-readiness":
		cmd := flag.NewFlagSet("wave3-readiness", flag.ExitOnError)
		cutoverProgress := cmd.String("cutover-progress", "../../Docs/RefactoredProductDocs/artifacts/cutover-progress-report.yaml", "cutover progress report yaml")
		shadowLint := cmd.String("shadow-lint", "../../Docs/RefactoredProductDocs/artifacts/shadow-lint-report.yaml", "shadow lint report yaml")
		planningLint := cmd.String("planning-lint", "../../Docs/RefactoredProductDocs/artifacts/planning-lint-report.yaml", "planning lint report yaml")
		report := cmd.String("report", "../../Docs/RefactoredProductDocs/artifacts/wave3-readiness-report.yaml", "output wave3 readiness report path")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.Wave3Readiness(*cutoverProgress, *shadowLint, *planningLint, *report))
	case "wave3-remediation":
		cmd := flag.NewFlagSet("wave3-remediation", flag.ExitOnError)
		readiness := cmd.String("readiness", "../../Docs/RefactoredProductDocs/artifacts/wave3-readiness-report.yaml", "wave3 readiness report yaml")
		register := cmd.String("register", "../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml", "technical cutover register yaml")
		report := cmd.String("report", "../../Docs/RefactoredProductDocs/artifacts/wave3-remediation-report.yaml", "output wave3 remediation report path")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.Wave3Remediation(*readiness, *register, *report))
	case "wave3-remove-shadows":
		cmd := flag.NewFlagSet("wave3-remove-shadows", flag.ExitOnError)
		readiness := cmd.String("readiness", "../../Docs/RefactoredProductDocs/artifacts/wave3-readiness-report.yaml", "wave3 readiness report yaml")
		register := cmd.String("register", "../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml", "technical cutover register yaml")
		report := cmd.String("report", "../../Docs/RefactoredProductDocs/artifacts/wave3-remove-shadows-report.yaml", "output wave3 removal report path")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.Wave3RemoveShadows(*readiness, *register, *report))
	case "wave3-closure":
		cmd := flag.NewFlagSet("wave3-closure", flag.ExitOnError)
		readiness := cmd.String("readiness", "../../Docs/RefactoredProductDocs/artifacts/wave3-readiness-report.yaml", "wave3 readiness report yaml")
		removal := cmd.String("removal", "../../Docs/RefactoredProductDocs/artifacts/wave3-remove-shadows-report.yaml", "wave3 removal report yaml")
		shadowLint := cmd.String("shadow-lint", "../../Docs/RefactoredProductDocs/artifacts/shadow-lint-report.yaml", "shadow lint report yaml")
		planningLint := cmd.String("planning-lint", "../../Docs/RefactoredProductDocs/artifacts/planning-lint-report.yaml", "planning lint report yaml")
		metadataLint := cmd.String("metadata-lint", "../../Docs/RefactoredProductDocs/artifacts/metadata-lint-report.yaml", "metadata lint report yaml")
		register := cmd.String("register", "../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml", "technical cutover register yaml")
		report := cmd.String("report", "../../Docs/RefactoredProductDocs/artifacts/wave3-closure-report.yaml", "output wave3 closure report path")
		doc := cmd.String("doc", "../../Docs/RefactoredProductDocs/07-roadmaps/wave3-closure-2026-02-21.md", "output wave3 closure markdown path")
		date := cmd.String("date", "2026-02-21", "closure checkpoint date")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.Wave3Closure(*readiness, *removal, *shadowLint, *planningLint, *metadataLint, *register, *report, *doc, *date))
	case "wave4-bootstrap":
		cmd := flag.NewFlagSet("wave4-bootstrap", flag.ExitOnError)
		closure := cmd.String("closure", "../../Docs/RefactoredProductDocs/artifacts/wave3-closure-report.yaml", "wave3 closure report yaml")
		topology := cmd.String("topology", "../../Docs/RefactoredProductDocs/artifacts/multi-repo-topology.yaml", "multi-repo topology yaml")
		report := cmd.String("report", "../../Docs/RefactoredProductDocs/artifacts/wave4-bootstrap-report.yaml", "output wave4 bootstrap report path")
		doc := cmd.String("doc", "../../Docs/RefactoredProductDocs/07-roadmaps/wave4-bootstrap-2026-02-21.md", "output wave4 bootstrap markdown path")
		date := cmd.String("date", "2026-02-21", "wave4 bootstrap checkpoint date")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.Wave4Bootstrap(*closure, *topology, *report, *doc, *date))
	case "v2-demo-readiness":
		cmd := flag.NewFlagSet("v2-demo-readiness", flag.ExitOnError)
		demoRoot := cmd.String("demo-root", "../../portfolio/iqpe-product-template/demo-project-v3", "demo project root")
		report := cmd.String("report", "../../portfolio/iqpe-product-template/demo-project-v3/artifacts/v2-demo-readiness-report.yaml", "output v2 demo readiness report path")
		_ = cmd.Parse(os.Args[2:])
		mustNoErr(runner.V2DemoReadiness(*demoRoot, *report))
	default:
		fmt.Printf("unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func must(condition bool, message string) {
	if !condition {
		fmt.Println(message)
		os.Exit(1)
	}
}

func mustNoErr(err error) {
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
