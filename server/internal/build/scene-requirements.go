package build

import (
	"fmt"
	"regexp"
	"strings"
)

var sceneStairsWords = regexp.MustCompile(`(?i)\b(stairs|staircase|stairway)\b`)
var sceneWindowWords = regexp.MustCompile(`(?i)\bwindows?\b`)
var sceneWheelWords = regexp.MustCompile(`(?i)\bwheels?\b`)
var sceneFourWheels = regexp.MustCompile(`(?i)(四|4)\s*个?\s*(真实)?(车轮|轮子)|\b(four|4)\s+(real\s+)?wheels\b`)

// These checks cover explicit names of supplied construction techniques only.
// They do not establish general language understanding or visual fidelity.
func checkSceneRequirementFunction(requirement string, stairs, windows bool, wheels int) error {
	if (strings.Contains(requirement, "楼梯") || sceneStairsWords.MatchString(requirement)) && !stairs {
		return fmt.Errorf("%q needs an attributed stairs construction, not a label or painted staircase", requirement)
	}
	if (strings.Contains(requirement, "窗") || sceneWindowWords.MatchString(requirement)) && !windows {
		return fmt.Errorf("%q needs real openings in an attributed wall/frame; paint is a color patch, not a window. Use frame openings with face front/back/left/right", requirement)
	}
	lower := strings.ToLower(requirement)
	noWheels := strings.Contains(lower, "no wheel") || strings.Contains(lower, "without wheel") || strings.Contains(lower, "无轮") || strings.Contains(lower, "无车轮") || strings.Contains(lower, "没有车轮") || strings.Contains(lower, "不要车轮")
	if !noWheels && (strings.Contains(requirement, "车轮") || strings.Contains(requirement, "轮子") || sceneWheelWords.MatchString(requirement)) {
		if wheels == 0 || sceneFourWheels.MatchString(requirement) && wheels != 4 {
			return fmt.Errorf("%q needs real mounted wheels in attributed wheelbase constructions; an explicit four wheels requires exactly one wheelbase", requirement)
		}
	}
	return nil
}

func checkScenePairedRequirement(requirement string, constructions int) error {
	text := strings.ToLower(requirement)
	for _, phrase := range []string{"双腿", "两条腿", "双臂", "两只手臂", "two legs", "two arms", "pair of legs", "pair of arms"} {
		if strings.Contains(text, phrase) && constructions < 2 {
			return fmt.Errorf("%q needs at least two attributed non-paint constructions; a single solid block labelled as two limbs is insufficient", requirement)
		}
	}
	return nil
}

// With complete typed anchors, attachment components are cheap to check before
// asking the tiler to solve any geometry. Absolute API designs use the full
// occupancy/connector checks instead.
func checkSceneAttachmentGraph(object SceneObject) error {
	ids := map[string]int{}
	parent := make([]int, len(object.Modules))
	ground := false
	for i, m := range object.Modules {
		if m.Attach == nil {
			return nil
		}
		ids[m.ID], parent[i] = i, i
		ground = ground || m.Attach.Anchor == "ground"
	}
	var root func(int) int
	root = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for i, m := range object.Modules {
		parents := append([]string{}, m.Attach.Parents...)
		if m.Attach.Parent != "" {
			parents = append(parents, m.Attach.Parent)
		}
		for _, id := range parents {
			if j, ok := ids[id]; ok {
				parent[root(i)] = root(j)
			}
		}
	}
	groups := map[int][]string{}
	for i, m := range object.Modules {
		groups[root(i)] = append(groups[root(i)], m.ID)
	}
	if !ground {
		return fmt.Errorf("object %s has no ground anchor", object.ID)
	}
	if len(groups) > 1 {
		return fmt.Errorf("object %s has disconnected attachment groups; grounded limbs need a base/top bridge joining their parents, not side contact: %v", object.ID, groups)
	}
	return nil
}
