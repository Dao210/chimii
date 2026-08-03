package build

import (
	"fmt"
	"strings"
)

// ExportMPD emits a standards-compatible single-model MPD. The BuildPlan stays
// canonical; MPD is an exchange artifact whose integer LDU transforms can be
// reproduced byte-for-byte from the plan.
func ExportMPD(plan BuildPlan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "0 FILE %s.ldr\n", safeModelName(plan.Title))
	fmt.Fprintln(&b, "0 Name: "+plan.Title)
	fmt.Fprintln(&b, "0 Author: CHIMII Build Studio")
	fmt.Fprintln(&b, "0 // References official LDraw Parts Library identifiers; part files are not embedded")
	fmt.Fprintln(&b, "0 !CHIMII KIT "+plan.KitID)
	lastStep := 0
	for _, p := range plan.Placements {
		if lastStep != 0 && p.Step != lastStep {
			fmt.Fprintln(&b, "0 STEP")
		}
		lastStep = p.Step
		spec := plan.Parts[p.PartID]
		// Plan X/Z are lower occupancy bounds while official LDraw parts use a
		// centered horizontal origin. Convert after orientation so differently
		// sized parts still share the exact stud grid. -Y points up in LDraw.
		sx, sz := spec.StudsX, spec.StudsZ
		if p.Rotation%180 != 0 {
			sx, sz = sz, sx
		}
		x := p.X*20 + sx*10
		y := -p.Y*8 + spec.OriginYOffsetLDU
		z := p.Z*20 + sz*10 + spec.OriginCenterZOffsetLDU
		a, c, g, i := 1, 0, 0, 1
		switch ((p.Rotation % 360) + 360) % 360 {
		case 90:
			a, c, g, i = 0, -1, 1, 0
		case 180:
			a, c, g, i = -1, 0, 0, -1
		case 270:
			a, c, g, i = 0, 1, -1, 0
		}
		fmt.Fprintf(&b, "1 %d %d %d %d %d 0 %d 0 1 0 %d 0 %d %s\n", p.Color, x, y, z, a, c, g, i, spec.LDrawID)
	}
	fmt.Fprintln(&b, "0 NOFILE")
	// The official format specifies CRLF line endings. Readers commonly accept
	// LF, but emitting the canonical form improves interoperability.
	return strings.ReplaceAll(b.String(), "\n", "\r\n")
}

func safeModelName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "chimii-creation"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r > 127 {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
