package circuit

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// AssemblyReference is a source-indexed research record, not a compilable
// Catalog or Document. Pictured quantities exclude unlisted cables and must
// never be treated as either complete purchasing requirements or family stock.
type AssemblyReference struct {
	ID             string                        `json:"id"`
	Version        string                        `json:"version"`
	Status         string                        `json:"status"`
	KitID          string                        `json:"kit_id"`
	KitName        string                        `json:"kit_name"`
	Title          Text                          `json:"title"`
	Description    Text                          `json:"description"`
	CheckedOn      string                        `json:"checked_on"`
	SourceURL      string                        `json:"source_url"`
	ProductURL     string                        `json:"product_url"`
	PartsSourceURL string                        `json:"parts_source_url"`
	ProgramURL     string                        `json:"program_url"`
	Parts          []AssemblyReferencePart       `json:"parts"`
	Steps          []AssemblyReferenceStep       `json:"steps"`
	Connections    []AssemblyReferenceConnection `json:"connections"`
	Unresolved     []AssemblyReferenceGap        `json:"unresolved"`
	ContentHash    string                        `json:"content_hash"`
}

type AssemblyReferencePart struct {
	ID               string `json:"id"`
	Name             Text   `json:"name"`
	PicturedQuantity int    `json:"pictured_quantity"`
	Kind             string `json:"kind"`
}

type AssemblyReferenceStep struct {
	Number          int            `json:"number"`
	Title           Text           `json:"title"`
	SourceURL       string         `json:"source_url"`
	IntroducedParts map[string]int `json:"introduced_parts"`
}

type AssemblyReferenceConnection struct {
	PartID       string `json:"part_id"`
	ControllerID string `json:"controller_id"`
	Port         string `json:"port"`
}

type AssemblyReferenceGap struct {
	ID     string `json:"id"`
	Detail Text   `json:"detail"`
}

//go:embed references/nezha-v2-gate.json
var nezhaGateReferenceJSON []byte

// AssemblyReferences returns fresh snapshots independently of executable kits.
// The digest identifies our transcription, not the remote source image bytes.
func AssemblyReferences() []AssemblyReference {
	var ref AssemblyReference
	if err := json.Unmarshal(nezhaGateReferenceJSON, &ref); err != nil {
		panic(err)
	}
	if err := ValidateAssemblyReference(ref); err != nil {
		panic(err)
	}
	ref.ContentHash = ""
	data, err := json.Marshal(ref)
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(data)
	ref.ContentHash = hex.EncodeToString(digest[:])
	return []AssemblyReference{ref}
}

// ValidateAssemblyReference checks transcription consistency only. It does
// not infer geometry, compatibility, firmware validity or physical readiness.
func ValidateAssemblyReference(ref AssemblyReference) error {
	validText := func(t Text) bool { return t.EN != "" && t.ZH != "" }
	validURL := func(s string) bool {
		u, err := url.Parse(s)
		return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil
	}
	if ref.ID == "" || ref.Version == "" || ref.KitID == "" || ref.KitName == "" || ref.Status != "research" || !validText(ref.Title) || !validText(ref.Description) {
		return fmt.Errorf("invalid assembly reference identity or status")
	}
	if _, err := time.Parse(time.DateOnly, ref.CheckedOn); err != nil {
		return fmt.Errorf("invalid reference check date")
	}
	for _, link := range []string{ref.SourceURL, ref.ProductURL, ref.PartsSourceURL, ref.ProgramURL} {
		if !validURL(link) {
			return fmt.Errorf("invalid reference source URL")
		}
	}
	if len(ref.Parts) == 0 || len(ref.Parts) > 128 || len(ref.Steps) == 0 || len(ref.Steps) > 128 || len(ref.Unresolved) == 0 {
		return fmt.Errorf("incomplete assembly reference")
	}
	parts := map[string]AssemblyReferencePart{}
	for _, p := range ref.Parts {
		if _, duplicate := parts[p.ID]; duplicate || p.ID == "" || p.PicturedQuantity <= 0 || p.PicturedQuantity > 128 || !validText(p.Name) || (p.Kind != "structure" && p.Kind != "electronics") {
			return fmt.Errorf("invalid reference part %q", p.ID)
		}
		parts[p.ID] = p
	}
	introduced := map[string]int{}
	for i, step := range ref.Steps {
		if step.Number != i+1 || !validText(step.Title) || !validURL(step.SourceURL) || len(step.IntroducedParts) == 0 {
			return fmt.Errorf("invalid reference step %d", i+1)
		}
		for id, n := range step.IntroducedParts {
			if _, exists := parts[id]; !exists || n <= 0 || n > 128 {
				return fmt.Errorf("invalid part %q at reference step %d", id, step.Number)
			}
			introduced[id] += n
		}
	}
	for id, p := range parts {
		if introduced[id] != p.PicturedQuantity {
			return fmt.Errorf("reference quantity mismatch for %q: steps=%d, pictured=%d", id, introduced[id], p.PicturedQuantity)
		}
	}
	ports := map[string]bool{}
	for _, c := range ref.Connections {
		key := c.ControllerID + ":" + c.Port
		if parts[c.PartID].Kind != "electronics" || parts[c.ControllerID].Kind != "electronics" || c.PartID == c.ControllerID || c.Port == "" || ports[key] {
			return fmt.Errorf("invalid reference connection %q", key)
		}
		ports[key] = true
	}
	gaps := map[string]bool{}
	for _, gap := range ref.Unresolved {
		if gap.ID == "" || gaps[gap.ID] || !validText(gap.Detail) {
			return fmt.Errorf("invalid reference gap %q", gap.ID)
		}
		gaps[gap.ID] = true
	}
	return nil
}
