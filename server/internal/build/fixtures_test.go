package build

import "strings"

func PlanRecipe(prompt string, answers map[string]string) AssemblyRecipe {
	joined := strings.ToLower(prompt + " " + answers["idea"] + " " + answers["movement"])
	archetype := "creature"
	features := []string{"friendly-face", "stable-feet"}
	switch {
	case containsAny(joined, "车", "轮", "跑", "car", "wheel", "race"):
		archetype = "racer"
		features = []string{"rolling-base", "driver-cabin"}
	case containsAny(joined, "飞", "翼", "翅", "鸟", "dragon", "fly", "wing"):
		archetype = "flyer"
		features = []string{"wide-wings", "balanced-tail"}
	case containsAny(joined, "机器人", "机械", "robot", "mech"):
		archetype = "robot"
		features = []string{"friendly-face", "strong-arms", "stable-feet"}
	}
	title := map[string]string{"racer": "闪电探险车", "flyer": "云朵飞行兽", "robot": "勇气机器人", "creature": "摇尾巴积木朋友"}[archetype]
	recipe := ExampleRecipe(archetype)
	recipe.Title = title
	recipe.Prompt = strings.TrimSpace(prompt)
	recipe.Features = features
	return recipe
}

func containsAny(value string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(value, word) {
			return true
		}
	}
	return false
}

func placementsFor(recipe AssemblyRecipe) []Placement {
	if len(recipe.Modules) == 0 {
		recipe = ExampleRecipe(recipe.Archetype)
	}
	p, _ := ExpandRecipe(recipe)
	return p
}
func AvailableArchetypes(i InventorySnapshot) []string {
	var out []string
	for _, a := range []string{"racer", "flyer", "robot", "creature"} {
		r := ExampleRecipe(a)
		p, _ := ExpandRecipe(r)
		if Validate(resolveInventoryColors(p, i), i).Buildable {
			out = append(out, a)
		}
	}
	return out
}
