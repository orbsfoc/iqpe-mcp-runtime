# docflow

`docflow` is a Go CLI to run deterministic parts of the documentation refactor pipeline.

## Commands
- `inventory --source <path> --out <file>`
- `model --out <file>`
- `validate --artifacts <dir> [--product-docs-root <dir>] [--technical-docs-root <dir>]`
- `run --source <path> --artifacts <dir>`
- `diagrams --artifacts <dir> --out <dir>`
- `frontend-diagrams --artifacts <dir> --out <dir>`
- `metadata-lint --docs-root <dir> --schema <file> --report <file>`
- `planning-lint --register <file> --report <file>`
- `shadow-lint --register <file> --report <file>`
- `cutover-progress --register <file> --report <file>`
- `wave3-readiness --cutover-progress <file> --shadow-lint <file> --planning-lint <file> --report <file>`
- `wave3-remediation --readiness <file> --register <file> --report <file>`
- `wave3-remove-shadows --readiness <file> --register <file> --report <file>`
- `wave3-closure --readiness <file> --removal <file> --shadow-lint <file> --planning-lint <file> --metadata-lint <file> --register <file> --report <file> --doc <file> --date <yyyy-mm-dd>`
- `wave4-bootstrap --closure <file> --topology <file> --report <file> --doc <file> --date <yyyy-mm-dd>`
- `v2-demo-readiness --demo-root <dir> --report <file>`
- `slice-graph-lint --graph <file> --report <file>`
- `library-search --catalog <file> --report <file> [--query <text>] [--capability <tag>] [--status <status>] [--runtime <runtime>]`
- `library-decision-lint --register <file> --report <file>`
- `tech-policy-eval --policy <file> --report <file> --project-phase <phase> --environment <env> --decision-type <type> --technology <tech> [--context-tags <csv>] [--approval-status <status>]`
- `tech-radar-lint --radar <file> --report <file>`
- `architecture-ops-lint --boundary <file> --report <file>`
- `topology-lint --topology <file> --report <file>`
- `data-boundary-lint --evidence <file> --report <file>`
- `metadata-overview-lint --metadata <file> --overview <file> --report <file>`
- `resource-variance-lint --estimation <file> --usage <file> --report <file> [--request-ratio-threshold <n>] [--token-ratio-threshold <n>]`

## Example
```bash
go run ./cmd/docflow inventory --source ../../SampleProductDocs --out ../../Docs/RefactoredProductDocs/artifacts/p01-inventory.yaml
go run ./cmd/docflow model --out ../../Docs/RefactoredProductDocs/artifacts/p02-model.yaml
go run ./cmd/docflow validate --artifacts ../../Docs/RefactoredProductDocs/artifacts --product-docs-root ../../Docs/RefactoredProductDocs --technical-docs-root ../../Docs/RefactoredTechnicalDocs
go run ./cmd/docflow diagrams --artifacts ../../Docs/RefactoredProductDocs/artifacts --out ../../Docs/RefactoredProductDocs/08-diagrams-generated
go run ./cmd/docflow frontend-diagrams --artifacts ../../Docs/RefactoredProductDocs/artifacts --out ../../Docs/RefactoredProductDocs/08-diagrams-generated
go run ./cmd/docflow metadata-lint --docs-root ../../Docs/RefactoredProductDocs --schema ../../Docs/RefactoredProductDocs/contracts/doc-metadata.schema.yaml --report ../../Docs/RefactoredProductDocs/artifacts/metadata-lint-report.yaml
go run ./cmd/docflow planning-lint --register ../../Docs/RefactoredProductDocs/artifacts/intent-implementation-planning-register.yaml --report ../../Docs/RefactoredProductDocs/artifacts/planning-lint-report.yaml
go run ./cmd/docflow shadow-lint --register ../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml --report ../../Docs/RefactoredProductDocs/artifacts/shadow-lint-report.yaml
go run ./cmd/docflow cutover-progress --register ../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml --report ../../Docs/RefactoredProductDocs/artifacts/cutover-progress-report.yaml
go run ./cmd/docflow wave3-readiness --cutover-progress ../../Docs/RefactoredProductDocs/artifacts/cutover-progress-report.yaml --shadow-lint ../../Docs/RefactoredProductDocs/artifacts/shadow-lint-report.yaml --planning-lint ../../Docs/RefactoredProductDocs/artifacts/planning-lint-report.yaml --report ../../Docs/RefactoredProductDocs/artifacts/wave3-readiness-report.yaml
go run ./cmd/docflow wave3-remediation --readiness ../../Docs/RefactoredProductDocs/artifacts/wave3-readiness-report.yaml --register ../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml --report ../../Docs/RefactoredProductDocs/artifacts/wave3-remediation-report.yaml
go run ./cmd/docflow wave3-remove-shadows --readiness ../../Docs/RefactoredProductDocs/artifacts/wave3-readiness-report.yaml --register ../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml --report ../../Docs/RefactoredProductDocs/artifacts/wave3-remove-shadows-report.yaml
go run ./cmd/docflow wave3-closure --readiness ../../Docs/RefactoredProductDocs/artifacts/wave3-readiness-report.yaml --removal ../../Docs/RefactoredProductDocs/artifacts/wave3-remove-shadows-report.yaml --shadow-lint ../../Docs/RefactoredProductDocs/artifacts/shadow-lint-report.yaml --planning-lint ../../Docs/RefactoredProductDocs/artifacts/planning-lint-report.yaml --metadata-lint ../../Docs/RefactoredProductDocs/artifacts/metadata-lint-report.yaml --register ../../Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml --report ../../Docs/RefactoredProductDocs/artifacts/wave3-closure-report.yaml --doc ../../Docs/RefactoredProductDocs/07-roadmaps/wave3-closure-2026-02-21.md --date 2026-02-21
go run ./cmd/docflow wave4-bootstrap --closure ../../Docs/RefactoredProductDocs/artifacts/wave3-closure-report.yaml --topology ../../Docs/RefactoredProductDocs/artifacts/multi-repo-topology.yaml --report ../../Docs/RefactoredProductDocs/artifacts/wave4-bootstrap-report.yaml --doc ../../Docs/RefactoredProductDocs/07-roadmaps/wave4-bootstrap-2026-02-21.md --date 2026-02-21
go run ./cmd/docflow v2-demo-readiness --demo-root ../../portfolio/iqpe-product-template/demo-project-v3 --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/v2-demo-readiness-report.yaml
go run ./cmd/docflow slice-graph-lint --graph ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/session-plan-orchestration.yaml --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/slice-graph-lint-report.yaml
go run ./cmd/docflow library-search --catalog ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/library-catalog.yaml --query shared --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/library-search-report.yaml
go run ./cmd/docflow library-decision-lint --register ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/new-library-decisions.yaml --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/library-decision-lint-report.yaml
go run ./cmd/docflow tech-policy-eval --policy ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/phase-tech-policy.yaml --project-phase MVP --environment DEMO --decision-type ORCHESTRATION --technology docker-compose --approval-status APPROVED --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/tech-policy-eval-report.yaml
go run ./cmd/docflow tech-radar-lint --radar ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/tech-radar-summary.yaml --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/tech-radar-lint-report.yaml
go run ./cmd/docflow architecture-ops-lint --boundary ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/architecture-operations-boundary.yaml --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/architecture-ops-lint-report.yaml
go run ./cmd/docflow topology-lint --topology ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/portfolio-topology.yaml --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/topology-lint-report.yaml
go run ./cmd/docflow data-boundary-lint --evidence ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/data-boundary-evidence.yaml --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/data-boundary-lint-report.yaml
go run ./cmd/docflow metadata-overview-lint --metadata ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/code-unit-metadata.yaml --overview ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/human-readable-overviews.yaml --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/metadata-overview-lint-report.yaml
go run ./cmd/docflow resource-variance-lint --estimation ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/plan-resource-estimation.yaml --usage ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/developer-agent-usage-report.yaml --report ../../portfolio/iqpe-product-template/demo-project-v3/artifacts/resource-variance-summary.yaml
```

## Validation Behavior
- Checks required pipeline artifacts exist.
- Enforces service-to-guide coverage:
	- Parses service IDs from technical docs first (`<technical-docs-root>/00-architecture/services/service-catalog.md`) and falls back to product docs (`<product-docs-root>/03-services/service-catalog.md`).
	- Verifies each service has at least one mapped guide in `Docs/RefactoredProductDocs/artifacts/service-tech-implementation-links.yaml`.
- Enforces context budget governance:
	- Validates `Docs/RefactoredProductDocs/artifacts/context-budget-policy.yaml` exists and parses.
	- Requires `policy_id`, `default_profile`, at least one profile, positive budget limits, and required report fields.
	- When policy enables reporting, validates `Docs/RefactoredProductDocs/artifacts/context-budget-report.yaml` and required report fields.
- Enforces frontend page evolution governance:
	- Validates `Docs/RefactoredProductDocs/artifacts/frontend-page-agreement-register.yaml` exists and parses.
	- Requires per-page fields: IDs, tags, content/visual spec refs, wireframe ref, service links, and implementation guides.
	- Applies acceptance gate: if page status is `accepted`, both `product_signoff` and `technical_signoff` must be `approved`.
- Enforces generated diagram metadata hygiene:
	- Validates every markdown file in `Docs/RefactoredProductDocs/08-diagrams-generated` contains YAML frontmatter.
	- Fails `validate` when generated diagram markdown is missing frontmatter delimiters.
	- Requires generated diagram `doc_id` values to follow `*-GEN-*` convention.
	- Requires generated diagram `doc_id` values to be unique within generated outputs.
- Enforces intent-to-implementation planning gates:
	- Validates `Docs/RefactoredProductDocs/artifacts/intent-implementation-planning-register.yaml` exists and parses.
	- Requires each plan to link feature IDs, ADR IDs, service IDs, component IDs, and implementation units.
	- Applies planning gate: if plan status is `ready-for-build`, product/architecture/engineering reviews must all be `approved`.
- Enforces Wave 2 technical docs cutover semantics (when register exists):
	- Validates `Docs/RefactoredProductDocs/artifacts/technical-docs-cutover-register.yaml` fields and enums.
	- Ensures technical-primary targets exist.
	- Ensures product shadow file semantics align with `product_shadow_state`.

## Planning Lint
- Produces machine-readable planning report (default: `Docs/RefactoredProductDocs/artifacts/planning-lint-report.yaml`).
- Reports missing planning fields and review gate violations per plan.

## Shadow Lint
- Produces machine-readable shadow state report (default: `Docs/RefactoredProductDocs/artifacts/shadow-lint-report.yaml`).
- For `deprecated-ready` markdown entries, requires product shadow markdown to have `status: deprecated` and `deprecated-ready-shadow` tag.
- For `deprecated-ready` directory entries, requires the source directory to exist.

## Cutover Progress
- Produces machine-readable cutover summary (default: `Docs/RefactoredProductDocs/artifacts/cutover-progress-report.yaml`).
- Summarizes entry status counts and `ready_for_removal` candidates for Wave 3 decisions.

## Wave 3 Readiness
- Produces machine-readable go/no-go decision report (default: `Docs/RefactoredProductDocs/artifacts/wave3-readiness-report.yaml`).
- Aggregates `cutover-progress`, `shadow-lint`, and `planning-lint` reports into explicit blockers per cutover entry.

## Wave 3 Remediation
- Produces machine-readable remediation plan report (default: `Docs/RefactoredProductDocs/artifacts/wave3-remediation-report.yaml`).
- Maps each blocked cutover entry to required register state transitions and deterministic re-validation commands.

## Wave 3 Remove Shadows
- Executes shadow-file removal only when `wave3-readiness` is `go`.
- Removes product shadow sources listed in cutover register, updates register `product_shadow_state` to `removed`, and emits `Docs/RefactoredProductDocs/artifacts/wave3-remove-shadows-report.yaml`.

## Wave 3 Closure
- Produces immutable closure checkpoint report: `Docs/RefactoredProductDocs/artifacts/wave3-closure-report.yaml`.
- Produces closure summary document: `Docs/RefactoredProductDocs/07-roadmaps/wave3-closure-2026-02-21.md`.
- Enforces closure gates across readiness, removal, shadow lint, planning lint, metadata lint, and final register state.

## Wave 4 Bootstrap
- Produces machine-readable Wave 4 bootstrap report: `Docs/RefactoredProductDocs/artifacts/wave4-bootstrap-report.yaml`.
- Produces ownership baseline summary: `Docs/RefactoredProductDocs/07-roadmaps/wave4-bootstrap-2026-02-21.md`.
- Enforces baseline gates over Wave 3 closure, multi-repo topology core IDs/domains, and technical docs root paths.

## Metadata Lint
- Validates markdown frontmatter against metadata schema required fields and enum values.
- Detects duplicate `doc_id` values.
- Emits machine-readable report to `artifacts/metadata-lint-report.yaml` (or custom `--report` path).
