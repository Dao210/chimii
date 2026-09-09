package buildeval

import (
	"context"
	"github.com/chimii-ai/chimii/server/internal/build"
	"testing"
)

// Frozen accepted case identities from 6e987e0. Keep failures in the corpus;
// accepting a different case must never hide the loss of an existing layout.
var baselineAccepted = map[string]bool{
	"tower-01": true, "thin-wall-01": true, "hole-01": true, "bridge-01": true,
	"exact-count-01": true, "edit-opening-01": true, "cantilever-01": true, "tower-02": true,
	"thin-wall-02": true, "hole-02": true, "bridge-02": true, "exact-count-02": true,
	"edit-opening-02": true, "cantilever-02": true, "tower-03": true, "thin-wall-03": true,
	"hole-03": true, "bridge-03": true, "exact-count-03": true, "edit-opening-03": true,
	"cantilever-03": true, "tower-04": true, "thin-wall-04": true, "hole-04": true,
	"bridge-04": true, "exact-count-04": true, "edit-opening-04": true, "cantilever-04": true,
	"tower-05": true, "thin-wall-05": true, "hole-05": true, "bridge-05": true,
	"exact-count-05": true, "edit-opening-05": true, "tower-06": true, "thin-wall-06": true,
	"hole-06": true, "exact-count-06": true, "edit-opening-06": true, "tower-07": true,
	"thin-wall-07": true, "hole-07": true, "exact-count-07": true, "edit-opening-07": true,
	"tower-08": true, "thin-wall-08": true, "hole-08": true, "exact-count-08": true,
	"edit-opening-08": true, "tower-09": true, "thin-wall-09": true, "hole-09": true,
	"exact-count-09": true, "edit-opening-09": true, "tower-10": true, "thin-wall-10": true,
	"hole-10": true, "exact-count-10": true, "edit-opening-10": true,
}
var seamAccepted = map[string]bool{
	"seam-001": true, "seam-002": true, "seam-003": true, "seam-004": true,
	"seam-005": true, "seam-006": true, "seam-007": true, "seam-008": true,
	"seam-009": true, "seam-010": true, "seam-011": true, "seam-012": true,
	"seam-013": true, "seam-014": true, "seam-015": true, "seam-016": true,
	"seam-017": true, "seam-018": true, "seam-019": true, "seam-020": true,
	"seam-021": true, "seam-022": true, "seam-023": true, "seam-024": true,
	"seam-025": true, "seam-026": true, "seam-027": true, "seam-028": true,
	"seam-029": true, "seam-030": true, "seam-031": true, "seam-032": true,
	"seam-033": true, "seam-035": true, "seam-037": true, "seam-038": true,
	"seam-039": true, "seam-040": true, "seam-042": true, "seam-043": true,
	"seam-044": true, "seam-045": true, "seam-046": true, "seam-047": true,
	"seam-048": true, "seam-049": true, "seam-051": true, "seam-052": true,
	"seam-053": true, "seam-054": true, "seam-055": true, "seam-056": true,
	"seam-057": true, "seam-058": true, "seam-059": true, "seam-061": true,
	"seam-062": true, "seam-064": true, "seam-065": true, "seam-066": true,
	"seam-067": true, "seam-068": true, "seam-069": true, "seam-070": true,
	"seam-072": true, "seam-073": true, "seam-074": true, "seam-075": true,
	"seam-076": true, "seam-078": true, "seam-079": true,
}

func TestSeamCorpusPreservesEveryAcceptedInput(t *testing.T) {
	cases := SeamCases()
	if len(cases) != 80 {
		t.Fatal("changed fixed corpus size")
	}
	report := Run(context.Background(), cases, build.StarterCatalog)
	for _, c := range report.Cases {
		if !c.ContractPassed || (seamAccepted[c.ID] && (c.ErrorCode != "" || c.Validation == nil || !c.Validation.Buildable)) {
			t.Errorf("%s: regression %s %v", c.ID, c.ErrorCode, c.InvariantErrors)
		}
	}
}
