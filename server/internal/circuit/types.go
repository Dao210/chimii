// Package circuit owns the electronic construction domain. It has no dependency
// on the brick compiler, HTTP, database, or a particular model provider.
package circuit

type Text struct {
	EN string `json:"en"`
	ZH string `json:"zh"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Port struct {
	ID        string `json:"id"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction,omitempty"`
	Connector string `json:"connector,omitempty"`
}

type Part struct {
	ID             string `json:"id"`
	ManufacturerID string `json:"manufacturer_id"`
	Name           Text   `json:"name"`
	Purpose        Text   `json:"purpose"`
	Kind           string `json:"kind"`
	Ports          []Port `json:"ports"`
	// Body is an occupancy polygon in the same grid as the terminals.
	Body     []Point `json:"body"`
	Quantity int     `json:"quantity"`
	// Conductive groups describe wires only, never an IC's inferred internals.
	Conductive       [][]string `json:"conductive"`
	Marking          string     `json:"marking,omitempty"`
	SpecificationURL string     `json:"specification_url,omitempty"`
}

// Connections describe complete keyed cables, not bare electrical pins. Power
// conductors inside a BOSON cable are not inferred from the signal direction.
type Connection struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	CableID string `json:"cable_id,omitempty"`
}

type PurchaseLink struct {
	Label Text   `json:"label"`
	URL   string `json:"url"`
}

type HardwareProfile struct {
	Manufacturer  string         `json:"manufacturer"`
	Model         string         `json:"model"`
	Notes         Text           `json:"notes"`
	PurchaseLinks []PurchaseLink `json:"purchase_links"`
	CheckedOn     string         `json:"checked_on"`
}

type Placement struct {
	ID       string `json:"id"`
	PartID   string `json:"part_id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Rotation int    `json:"rotation"`
	Layer    int    `json:"layer"`
}

type Source struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Page   int    `json:"page"`
	SHA256 string `json:"sha256"`
}

type Step struct {
	ID           string   `json:"id"`
	Title        Text     `json:"title"`
	Instruction  Text     `json:"instruction"`
	PlacementIDs []string `json:"placement_ids"`
}

type Project struct {
	ID              string      `json:"id"`
	Title           Text        `json:"title"`
	Description     Text        `json:"description"`
	Explanation     Text        `json:"explanation"`
	TestInstruction Text        `json:"test_instruction"`
	Troubleshooting []Text      `json:"troubleshooting"`
	Source          Source      `json:"source"`
	Placements      []Placement `json:"placements"`
	Steps           []Step      `json:"steps"`
	// ExpectedNets independently encode the reviewed electrical topology. Only
	// functional terminals are listed; wiring terminals are derived from layout.
	ExpectedNets [][]string   `json:"expected_nets"`
	Connections  []Connection `json:"connections,omitempty"`
}

type Catalog struct {
	Version          string           `json:"version"`
	KitID            string           `json:"kit_id"`
	Name             string           `json:"name"`
	Columns          int              `json:"columns"`
	Rows             int              `json:"rows"`
	MinimumAge       int              `json:"minimum_age"`
	Parts            []Part           `json:"parts"`
	Projects         []Project        `json:"projects"`
	Preparation      []Text           `json:"preparation"`
	ConnectionSystem string           `json:"connection_system,omitempty"`
	Hardware         *HardwareProfile `json:"hardware,omitempty"`
}

type Issue struct {
	Code        string `json:"code"`
	PlacementID string `json:"placement_id,omitempty"`
	PartID      string `json:"part_id,omitempty"`
	Required    int    `json:"required,omitempty"`
	Available   int    `json:"available,omitempty"`
}

type Report struct {
	Passed               bool           `json:"passed"`
	Issues               []Issue        `json:"issues"`
	Nets                 [][]string     `json:"nets"`
	UsedParts            map[string]int `json:"used_parts"`
	PhysicalVerification string         `json:"physical_verification"`
}

type Document struct {
	Version          int            `json:"version"`
	CatalogVersion   string         `json:"catalog_version"`
	KitID            string         `json:"kit_id"`
	Prompt           string         `json:"prompt"`
	Title            string         `json:"title"`
	Planner          string         `json:"planner"`
	Project          Project        `json:"project"`
	Parts            []Part         `json:"parts"`
	Columns          int            `json:"columns"`
	Rows             int            `json:"rows"`
	Preparation      []Text         `json:"preparation"`
	Inventory        map[string]int `json:"inventory"`
	Validation       Report         `json:"validation"`
	ContentHash      string         `json:"content_hash"`
	ConnectionSystem string         `json:"connection_system,omitempty"`
}
