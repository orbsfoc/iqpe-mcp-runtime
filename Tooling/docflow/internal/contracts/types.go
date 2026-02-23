package contracts

type Document struct {
	DocID          string   `yaml:"doc_id"`
	Path           string   `yaml:"path"`
	Title          string   `yaml:"title"`
	Concern        string   `yaml:"concern"`
	Specificity    string   `yaml:"specificity"`
	PhaseRelevance []string `yaml:"phase_relevance"`
	DocRole        string   `yaml:"doc_role"`
	RequirementIDs []string `yaml:"requirement_ids"`
}

type Inventory struct {
	Documents []Document       `yaml:"documents"`
	Stats     map[string]any   `yaml:"stats"`
	Conflicts []map[string]any `yaml:"conflicts"`
}

type Model struct {
	Taxonomy      []map[string]any  `yaml:"taxonomy"`
	LinkTypes     []string          `yaml:"link_types"`
	IDConventions map[string]string `yaml:"id_conventions"`
}

type Validation struct {
	Validation map[string]any   `yaml:"validation"`
	Gaps       []map[string]any `yaml:"gaps"`
}
