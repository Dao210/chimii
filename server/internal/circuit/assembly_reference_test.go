package circuit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestAssemblyReferenceEvidenceAndIsolation(t *testing.T) {
	r := AssemblyReferences()[0]
	if err := ValidateAssemblyReference(r); err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, p := range r.Parts {
		total += p.PicturedQuantity
	}
	if len(r.Parts) != 16 || total != 36 || len(r.Steps) != 15 {
		t.Fatalf("source transcription changed: %d types, %d pieces, %d steps", len(r.Parts), total, len(r.Steps))
	}
	if _, executable := FindCatalog(r.KitID); executable {
		t.Fatal("research reference exposed as an executable kit")
	}
	if _, err := Compile(StarterCatalog(), r.ID, "", "", "project", nil); err == nil {
		t.Fatal("research reference accepted as a buildable project")
	}
	hash := r.ContentHash
	r.ContentHash = ""
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	if hash != hex.EncodeToString(digest[:]) {
		t.Fatal("reference digest does not identify the returned snapshot")
	}
	r.Parts[0].PicturedQuantity = 100
	r.Steps[0].IntroducedParts["servo"] = 99
	if fresh := AssemblyReferences()[0]; fresh.ContentHash != hash || fresh.Parts[0].PicturedQuantity != 1 || fresh.Steps[0].IntroducedParts["servo"] != 1 {
		t.Fatal("request mutation changed the reference authority")
	}
}

func TestAssemblyReferenceRejectsBrokenEvidence(t *testing.T) {
	for _, tc := range []struct {
		name           string
		breakReference func(*AssemblyReference)
	}{
		{"unearned readiness", func(r *AssemblyReference) { r.Status = "ready" }},
		{"missing known gaps", func(r *AssemblyReference) { r.Unresolved = nil }},
		{"missing pin", func(r *AssemblyReference) { r.Steps[0].IntroducedParts["black-pin"] = 0 }},
		{"extra pin", func(r *AssemblyReference) { r.Steps[0].IntroducedParts["black-pin"]++ }},
		{"unknown part", func(r *AssemblyReference) { r.Steps[0].IntroducedParts["imaginary"] = 1 }},
		{"duplicate part", func(r *AssemblyReference) { r.Parts[1].ID = r.Parts[0].ID }},
		{"lost step", func(r *AssemblyReference) { r.Steps = r.Steps[1:] }},
		{"structural wire", func(r *AssemblyReference) { r.Connections[0].PartID = "axle" }},
		{"occupied port", func(r *AssemblyReference) { r.Connections[1].Port = "J1" }},
		{"unsafe source", func(r *AssemblyReference) { r.Steps[0].SourceURL = "javascript:alert(1)" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := AssemblyReferences()[0]
			tc.breakReference(&r)
			if ValidateAssemblyReference(r) == nil {
				t.Fatal("broken evidence accepted")
			}
		})
	}
}
