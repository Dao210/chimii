package build

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultBuildDifficulty = 3
	maxRequestedPartCount  = 1000
)

var (
	chineseBlockCountPattern = regexp.MustCompile(`([0-9]{1,4})[[:space:]]*块`)
	chineseBrickCountPattern = regexp.MustCompile(`([0-9]{1,4})[[:space:]]*个[[:space:]]*(积木|乐高)`)
	englishPartCountPattern  = regexp.MustCompile(`(?i)([0-9]{1,4})[[:space:]]*(bricks?|blocks?|pieces?)`)
	chineseDifficultyPattern = regexp.MustCompile(`难度[[:space:]:：-]*([1-5])`)
	chineseLevelPattern      = regexp.MustCompile(`([1-5])[[:space:]]*级[[:space:]]*(难度)?`)
	englishDifficultyPattern = regexp.MustCompile(`(?i)(difficulty|level)[[:space:]:-]*([1-5])`)
	chineseAgePattern        = regexp.MustCompile(`([0-9]{1,2})[[:space:]]*岁`)
	englishAgePattern        = regexp.MustCompile(`(?i)(age|aged)[[:space:]:-]*([0-9]{1,2})`)
	englishYearsOldPattern   = regexp.MustCompile(`(?i)([0-9]{1,2})[[:space:]-]*(years?|yrs?)[[:space:]-]*old`)
)

// ApplyDifficultyPolicy records the deterministic scale decision in recipe
// metadata. An explicitly requested part count always wins; otherwise an
// explicit difficulty wins over the age-derived difficulty.
func ApplyDifficultyPolicy(recipe AssemblyRecipe, prompt string, answers map[string]string) AssemblyRecipe {
	if recipe.Metadata == nil {
		recipe.Metadata = map[string]string{}
	}
	text := buildScaleText(prompt, answers)
	age, hasAge := parseBuilderAge(text)
	difficulty, hasDifficulty := parseDifficulty(text)
	source := "default"
	if !hasDifficulty && hasAge {
		difficulty = difficultyForAge(age)
		hasDifficulty = true
		source = "age"
	} else if hasDifficulty {
		source = "difficulty"
	}
	if !hasDifficulty {
		difficulty = defaultBuildDifficulty
	}
	target := partCountForDifficulty(difficulty)
	if explicit, ok := parseExplicitPartCount(text); ok {
		target = explicit
		source = "explicit"
	}

	recipe.Metadata["difficulty_level"] = strconv.Itoa(difficulty)
	recipe.Metadata["target_part_count"] = strconv.Itoa(target)
	recipe.Metadata["part_count_source"] = source
	if hasAge {
		recipe.Metadata["builder_age"] = strconv.Itoa(age)
	}
	return recipe
}

func buildScaleText(prompt string, answers map[string]string) string {
	keys := make([]string, 0, len(answers))
	for key := range answers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var text strings.Builder
	text.WriteString(prompt)
	for _, key := range keys {
		text.WriteByte('\n')
		text.WriteString(answers[key])
	}
	return text.String()
}

func parseExplicitPartCount(text string) (int, bool) {
	for _, pattern := range []*regexp.Regexp{chineseBlockCountPattern, chineseBrickCountPattern, englishPartCountPattern} {
		match := pattern.FindStringSubmatch(text)
		if len(match) < 2 {
			continue
		}
		count, err := strconv.Atoi(match[1])
		if err == nil && count > 0 && count <= maxRequestedPartCount {
			return count, true
		}
	}
	return 0, false
}

func parseDifficulty(text string) (int, bool) {
	for _, candidate := range []struct {
		pattern *regexp.Regexp
		group   int
	}{
		{chineseDifficultyPattern, 1},
		{chineseLevelPattern, 1},
		{englishDifficultyPattern, 2},
	} {
		match := candidate.pattern.FindStringSubmatch(text)
		if len(match) > candidate.group {
			level, _ := strconv.Atoi(match[candidate.group])
			return level, true
		}
	}
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "高难度"), strings.Contains(lower, "困难"), strings.Contains(lower, "挑战"), strings.Contains(lower, "hard"):
		return 5, true
	case strings.Contains(lower, "中等难度"), strings.Contains(lower, "标准难度"), strings.Contains(lower, "medium"):
		return 3, true
	case strings.Contains(lower, "低难度"), strings.Contains(lower, "简单"), strings.Contains(lower, "容易"), strings.Contains(lower, "easy"):
		return 1, true
	default:
		return 0, false
	}
}

func parseBuilderAge(text string) (int, bool) {
	for _, candidate := range []struct {
		pattern *regexp.Regexp
		group   int
	}{
		{chineseAgePattern, 1},
		{englishAgePattern, 2},
		{englishYearsOldPattern, 1},
	} {
		match := candidate.pattern.FindStringSubmatch(text)
		if len(match) > candidate.group {
			age, _ := strconv.Atoi(match[candidate.group])
			if age > 0 {
				return age, true
			}
		}
	}
	return 0, false
}

func difficultyForAge(age int) int {
	switch {
	case age <= 6:
		return 1
	case age <= 8:
		return 2
	case age <= 10:
		return 3
	case age <= 12:
		return 4
	default:
		return 5
	}
}

func partCountForDifficulty(difficulty int) int {
	switch difficulty {
	case 1:
		return 24
	case 2:
		return 48
	case 4:
		return 120
	case 5:
		return 160
	default:
		return 80
	}
}

func targetPartCount(recipe AssemblyRecipe) int {
	if recipe.Metadata == nil {
		return 0
	}
	target, err := strconv.Atoi(recipe.Metadata["target_part_count"])
	if err != nil || target <= 0 {
		return 0
	}
	return target
}

func scalePlacementsToTarget(placements []Placement, recipe AssemblyRecipe) []Placement {
	return scalePlacementsToTargetWithCatalog(placements, recipe, StarterCatalog)
}

func scalePlacementsToTargetWithCatalog(placements []Placement, recipe AssemblyRecipe, catalog PartCatalog) []Placement {
	target := targetPartCount(recipe)
	if target == 0 || target == len(placements) || len(placements) == 0 {
		return placements
	}
	if target < len(placements) {
		if recipe.Archetype == "robot" {
			return shrinkRobotPlacementsWithCatalog(placements, target, catalog)
		}
		return normalizePlacementSteps(append([]Placement(nil), placements[:target]...))
	}

	scaled := append([]Placement(nil), placements...)
	anchor := scaled[0]
	maxTop := topYWith(anchor, catalog)
	maxStep := anchor.Step
	for _, placement := range scaled[1:] {
		if top := topYWith(placement, catalog); top > maxTop {
			anchor = placement
			maxTop = top
		}
		if placement.Step > maxStep {
			maxStep = placement.Step
		}
	}
	for len(scaled) < target {
		next := anchor
		next.ID = fmt.Sprintf("difficulty-%04d", len(scaled)+1)
		next.Y = topYWith(anchor, catalog)
		next.Step = maxStep + len(scaled) - len(placements) + 1
		next.Module = "difficulty-detail"
		scaled = append(scaled, next)
		anchor = next
	}
	return scaled
}

func shrinkRobotPlacements(placements []Placement, target int) []Placement {
	return shrinkRobotPlacementsWithCatalog(placements, target, StarterCatalog)
}

func shrinkRobotPlacementsWithCatalog(placements []Placement, target int, catalog PartCatalog) []Placement {
	if target <= 0 {
		return nil
	}
	moduleCounts := map[string]int{}
	dominantModule := ""
	for _, placement := range placements {
		moduleCounts[placement.Module]++
		if moduleCounts[placement.Module] > moduleCounts[dominantModule] {
			dominantModule = placement.Module
		}
	}
	dominantCount := moduleCounts[dominantModule]
	otherCount := len(placements) - dominantCount
	keepDominant := target - otherCount
	if keepDominant <= 0 {
		return normalizePlacementSteps(append([]Placement(nil), placements[:target]...))
	}

	originalDominantTop := 0
	keptDominantTop := 0
	kept := make([]Placement, 0, target)
	for _, placement := range placements {
		if placement.Module == dominantModule {
			if top := topYWith(placement, catalog); top > originalDominantTop {
				originalDominantTop = top
			}
			if keepDominant == 0 {
				continue
			}
			keepDominant--
			if top := topYWith(placement, catalog); top > keptDominantTop {
				keptDominantTop = top
			}
		}
		kept = append(kept, placement)
	}
	shift := originalDominantTop - keptDominantTop
	if shift > 0 {
		for index := range kept {
			if kept[index].Module != dominantModule && kept[index].Y >= originalDominantTop {
				kept[index].Y -= shift
			}
		}
	}
	return normalizePlacementSteps(kept)
}

func normalizePlacementSteps(placements []Placement) []Placement {
	steps := make([]int, 0, len(placements))
	seen := map[int]bool{}
	for _, placement := range placements {
		if !seen[placement.Step] {
			seen[placement.Step] = true
			steps = append(steps, placement.Step)
		}
	}
	sort.Ints(steps)
	normalized := make(map[int]int, len(steps))
	for index, step := range steps {
		normalized[step] = index + 1
	}
	for index := range placements {
		placements[index].Step = normalized[placements[index].Step]
	}
	return placements
}
