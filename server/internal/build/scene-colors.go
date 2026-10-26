package build

import (
	"regexp"
	"strings"
)

// Explicit color words in an attributed requirement are simple verifiable
// evidence. This does not establish general natural-language fidelity.
var sceneColorWords = []struct {
	code  int
	words []string
}{
	{272, []string{"深蓝色", "dark blue"}}, {288, []string{"深绿色", "dark green"}},
	{70, []string{"红棕色", "reddish brown"}}, {72, []string{"深灰色", "dark gray", "dark grey"}},
	{71, []string{"浅灰色", "light bluish gray", "light gray", "light grey"}}, {29, []string{"粉色", "bright pink", "pink"}},
	{28, []string{"深棕黄色", "dark tan"}},
	{25, []string{"橙色", "orange"}}, {19, []string{"棕黄色", "tan"}},
	{15, []string{"白色", "white", "ホワイト", "흰색"}}, {0, []string{"黑色", "black", "ブラック", "검은색"}},
	{1, []string{"蓝色", "blue"}}, {2, []string{"绿色", "green"}}, {4, []string{"红色", "red"}}, {14, []string{"黄色", "yellow"}},
}

func RequiredSceneColors(requirement string) []int {
	text := strings.ToLower(requirement)
	colors := []int{}
	for _, option := range sceneColorWords {
		found := false
		for _, word := range option.words {
			if word[0] < 128 {
				pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
				if pattern.MatchString(text) {
					found = true
					text = pattern.ReplaceAllString(text, " ")
				}
			} else if strings.Contains(text, word) {
				found = true
				text = strings.ReplaceAll(text, word, " ")
			}
		}
		if found {
			colors = append(colors, option.code)
		}
	}
	return colors
}

func SceneColorNames() map[int]string {
	return map[int]string{0: "Black", 1: "Blue", 2: "Green", 4: "Red", 14: "Yellow", 15: "White", 19: "Tan", 25: "Orange", 28: "Dark tan", 29: "Bright pink", 70: "Reddish brown", 71: "Light bluish gray", 72: "Dark bluish gray", 272: "Dark blue", 288: "Dark green"}
}

func SceneModuleColorRequired(recipe AssemblyRecipe, module Construction, color int) bool {
	for _, n := range module.Requirements {
		if n >= 0 && n < len(recipe.Requirements) {
			for _, c := range RequiredSceneColors(recipe.Requirements[n]) {
				if c == color {
					return true
				}
			}
		}
	}
	return false
}
