package main

import (
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	selectionMethod = "rebrickable-set-frequency-v1"
	minimumSetYear  = 2000
)

var excludedCategoryIDs = map[int]bool{
	13: true, 27: true, 28: true, 41: true, 42: true, 48: true, 50: true,
	57: true, 58: true, 59: true, 60: true, 61: true, 62: true, 63: true,
	64: true, 65: true, 69: true, 70: true, 71: true, 72: true, 73: true,
	74: true, 75: true, 77: true, 78: true,
}

var requiredParts = map[string]struct {
	Name     string
	Category string
}{
	"3001.dat":    {Name: "Brick 2 x 4", Category: "Bricks"},
	"3003.dat":    {Name: "Brick 2 x 2", Category: "Bricks"},
	"3004.dat":    {Name: "Brick 1 x 2", Category: "Bricks"},
	"3005.dat":    {Name: "Brick 1 x 1", Category: "Bricks"},
	"3020.dat":    {Name: "Plate 2 x 4", Category: "Plates"},
	"3022.dat":    {Name: "Plate 2 x 2", Category: "Plates"},
	"3023.dat":    {Name: "Plate 1 x 2", Category: "Plates"},
	"3039.dat":    {Name: "Brick Sloped 45° 2 x 2", Category: "Bricks Sloped"},
	"4600.dat":    {Name: "Plate Special 2 x 2 with Wheel Holders", Category: "Wheels and Tyres"},
	"4624c04.dat": {Name: "Wheel Rim 6.4 x 8 with Tyre 8/75 x 8", Category: "Wheels and Tyres"},
}

type sourceSnapshot struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type selection struct {
	Method                 string                    `json:"method"`
	GeneratedAt            string                    `json:"generated_at"`
	MinimumSetYear         int                       `json:"minimum_set_year"`
	LatestInventoryOnly    bool                      `json:"latest_inventory_only"`
	ExcludeSpares          bool                      `json:"exclude_spares"`
	ExcludeDecoratedParts  bool                      `json:"exclude_decorated_parts"`
	ExactOfficialLDrawOnly bool                      `json:"exact_official_ldraw_only"`
	ExcludedCategoryIDs    []int                     `json:"excluded_category_ids"`
	RequiredLDrawIDs       []string                  `json:"required_ldraw_ids"`
	Sources                map[string]sourceSnapshot `json:"sources"`
}

type manifestPart struct {
	Rank          int    `json:"rank"`
	LDrawID       string `json:"ldraw_id"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	SetCount      int    `json:"set_count"`
	TotalQuantity int64  `json:"total_quantity"`
}

type manifest struct {
	SchemaVersion int            `json:"schema_version"`
	KitID         string         `json:"kit_id"`
	PartCount     int            `json:"part_count"`
	Selection     selection      `json:"selection"`
	Parts         []manifestPart `json:"parts"`
}

type partMetadata struct {
	Name       string
	CategoryID int
}

type partScore struct {
	PartNum       string
	SetCount      int
	TotalQuantity int64
}

type inputPaths struct {
	archive       string
	inventories   string
	inventoryPart string
	parts         string
	sets          string
	categories    string
	out           string
	count         int
}

func main() {
	paths := parseFlags()
	result, err := buildManifest(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ldraw-rank:", err)
		os.Exit(1)
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ldraw-rank:", err)
		os.Exit(1)
	}
	raw = append(raw, '\n')
	if paths.out == "" {
		_, _ = os.Stdout.Write(raw)
		return
	}
	if err := os.WriteFile(paths.out, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ldraw-rank:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d ranked LDraw parts to %s\n", len(result.Parts), paths.out)
}

func parseFlags() inputPaths {
	var paths inputPaths
	flag.StringVar(&paths.archive, "archive", "", "pinned official LDraw complete.zip")
	flag.StringVar(&paths.inventories, "inventories", "", "Rebrickable inventories.csv.gz")
	flag.StringVar(&paths.inventoryPart, "inventory-parts", "", "Rebrickable inventory_parts.csv.gz")
	flag.StringVar(&paths.parts, "parts", "", "Rebrickable parts.csv.gz")
	flag.StringVar(&paths.sets, "sets", "", "Rebrickable sets.csv.gz")
	flag.StringVar(&paths.categories, "categories", "", "Rebrickable part_categories.csv.gz")
	flag.StringVar(&paths.out, "out", "", "output manifest; stdout when empty")
	flag.IntVar(&paths.count, "count", 1000, "number of ranked parts")
	flag.Parse()
	return paths
}

func buildManifest(paths inputPaths) (manifest, error) {
	if paths.archive == "" || paths.inventories == "" || paths.inventoryPart == "" || paths.parts == "" || paths.sets == "" || paths.categories == "" {
		return manifest{}, errors.New("archive and all five Rebrickable CSV inputs are required")
	}
	if paths.count <= 0 {
		return manifest{}, errors.New("count must be positive")
	}

	setYears, err := loadSetYears(paths.sets)
	if err != nil {
		return manifest{}, err
	}
	latestInventories, err := loadLatestInventories(paths.inventories, setYears)
	if err != nil {
		return manifest{}, err
	}
	metadata, err := loadPartMetadata(paths.parts)
	if err != nil {
		return manifest{}, err
	}
	categories, err := loadCategories(paths.categories)
	if err != nil {
		return manifest{}, err
	}
	officialParts, err := loadOfficialLDrawParts(paths.archive)
	if err != nil {
		return manifest{}, err
	}
	scores, err := rankInventoryParts(paths.inventoryPart, latestInventories)
	if err != nil {
		return manifest{}, err
	}

	filtered := make([]partScore, 0, len(scores))
	for partNum, score := range scores {
		meta, ok := metadata[partNum]
		if !ok || excludedCategoryIDs[meta.CategoryID] || decorated(meta.Name) {
			continue
		}
		ldrawID := strings.ToLower(partNum) + ".dat"
		if !officialParts[ldrawID] {
			continue
		}
		score.PartNum = strings.ToLower(partNum)
		filtered = append(filtered, score)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].SetCount != filtered[j].SetCount {
			return filtered[i].SetCount > filtered[j].SetCount
		}
		if filtered[i].TotalQuantity != filtered[j].TotalQuantity {
			return filtered[i].TotalQuantity > filtered[j].TotalQuantity
		}
		return filtered[i].PartNum < filtered[j].PartNum
	})
	if len(filtered) < paths.count {
		return manifest{}, fmt.Errorf("only %d eligible official LDraw parts found; need %d", len(filtered), paths.count)
	}

	parts := make([]manifestPart, 0, paths.count)
	for index, score := range filtered[:paths.count] {
		meta := metadata[score.PartNum]
		parts = append(parts, manifestPart{
			Rank: index + 1, LDrawID: score.PartNum + ".dat", Name: meta.Name,
			Category: categories[meta.CategoryID], SetCount: score.SetCount, TotalQuantity: score.TotalQuantity,
		})
	}
	present := make(map[string]bool, len(parts))
	for _, part := range parts {
		present[part.LDrawID] = true
	}
	requiredIDs := make([]string, 0, len(requiredParts))
	for id := range requiredParts {
		requiredIDs = append(requiredIDs, id)
	}
	sort.Strings(requiredIDs)
	replacementIndex := len(parts) - 1
	for _, id := range requiredIDs {
		if present[id] {
			continue
		}
		if !officialParts[id] {
			return manifest{}, fmt.Errorf("required LDraw part %s is not official in the pinned archive", id)
		}
		required := requiredParts[id]
		for replacementIndex >= 0 && requiredParts[parts[replacementIndex].LDrawID].Name != "" {
			replacementIndex--
		}
		if replacementIndex < 0 {
			return manifest{}, errors.New("required parts exceed requested manifest size")
		}
		parts[replacementIndex] = manifestPart{
			Rank: replacementIndex + 1, LDrawID: id, Name: required.Name, Category: required.Category,
		}
		present[id] = true
		replacementIndex--
	}

	excluded := make([]int, 0, len(excludedCategoryIDs))
	for id := range excludedCategoryIDs {
		excluded = append(excluded, id)
	}
	sort.Ints(excluded)
	sources := map[string]sourceSnapshot{}
	for name, source := range map[string]struct{ path, url string }{
		"ldraw_complete":  {paths.archive, "https://library.ldraw.org/library/updates/complete.zip"},
		"inventories":     {paths.inventories, "https://cdn.rebrickable.com/media/downloads/inventories.csv.gz"},
		"inventory_parts": {paths.inventoryPart, "https://cdn.rebrickable.com/media/downloads/inventory_parts.csv.gz"},
		"parts":           {paths.parts, "https://cdn.rebrickable.com/media/downloads/parts.csv.gz"},
		"sets":            {paths.sets, "https://cdn.rebrickable.com/media/downloads/sets.csv.gz"},
		"part_categories": {paths.categories, "https://cdn.rebrickable.com/media/downloads/part_categories.csv.gz"},
	} {
		hash, err := fileSHA256(source.path)
		if err != nil {
			return manifest{}, err
		}
		sources[name] = sourceSnapshot{URL: source.url, SHA256: hash}
	}

	return manifest{
		SchemaVersion: 1,
		KitID:         "chimii-starter-1000-v1",
		PartCount:     len(parts),
		Selection: selection{
			Method: selectionMethod, GeneratedAt: time.Now().UTC().Format(time.RFC3339), MinimumSetYear: minimumSetYear,
			LatestInventoryOnly: true, ExcludeSpares: true, ExcludeDecoratedParts: true, ExactOfficialLDrawOnly: true,
			ExcludedCategoryIDs: excluded, RequiredLDrawIDs: requiredIDs, Sources: sources,
		},
		Parts: parts,
	}, nil
}

func loadSetYears(path string) (map[string]int, error) {
	result := map[string]int{}
	err := readGzipCSV(path, func(row []string) error {
		if len(row) < 3 {
			return nil
		}
		year, err := strconv.Atoi(row[2])
		if err != nil {
			return nil
		}
		result[row[0]] = year
		return nil
	})
	return result, err
}

func loadLatestInventories(path string, setYears map[string]int) (map[int]bool, error) {
	type candidate struct{ id, version int }
	bySet := map[string]candidate{}
	err := readGzipCSV(path, func(row []string) error {
		if len(row) < 3 || setYears[row[2]] < minimumSetYear {
			return nil
		}
		id, idErr := strconv.Atoi(row[0])
		version, versionErr := strconv.Atoi(row[1])
		if idErr != nil || versionErr != nil {
			return nil
		}
		current, ok := bySet[row[2]]
		if !ok || version > current.version || (version == current.version && id > current.id) {
			bySet[row[2]] = candidate{id: id, version: version}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := make(map[int]bool, len(bySet))
	for _, item := range bySet {
		result[item.id] = true
	}
	return result, nil
}

func loadPartMetadata(path string) (map[string]partMetadata, error) {
	result := map[string]partMetadata{}
	err := readGzipCSV(path, func(row []string) error {
		if len(row) < 3 {
			return nil
		}
		categoryID, err := strconv.Atoi(row[2])
		if err != nil {
			return nil
		}
		result[strings.ToLower(row[0])] = partMetadata{Name: row[1], CategoryID: categoryID}
		return nil
	})
	return result, err
}

func loadCategories(path string) (map[int]string, error) {
	result := map[int]string{}
	err := readGzipCSV(path, func(row []string) error {
		if len(row) < 2 {
			return nil
		}
		id, err := strconv.Atoi(row[0])
		if err == nil {
			result[id] = row[1]
		}
		return nil
	})
	return result, err
}

func loadOfficialLDrawParts(path string) (map[string]bool, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	result := map[string]bool{}
	for _, file := range archive.File {
		name := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(file.Name, "ldraw/"), "\\", "/"))
		if strings.HasPrefix(name, "parts/") && !strings.Contains(strings.TrimPrefix(name, "parts/"), "/") && strings.HasSuffix(name, ".dat") {
			result[strings.TrimPrefix(name, "parts/")] = true
		}
	}
	return result, nil
}

func rankInventoryParts(path string, selectedInventories map[int]bool) (map[string]partScore, error) {
	result := map[string]partScore{}
	currentInventory := -1
	currentParts := map[string]int64{}
	flush := func() {
		if currentInventory < 0 || !selectedInventories[currentInventory] {
			currentParts = map[string]int64{}
			return
		}
		for partNum, quantity := range currentParts {
			score := result[partNum]
			score.SetCount++
			score.TotalQuantity += quantity
			result[partNum] = score
		}
		currentParts = map[string]int64{}
	}
	err := readGzipCSV(path, func(row []string) error {
		if len(row) < 5 {
			return nil
		}
		inventoryID, err := strconv.Atoi(row[0])
		if err != nil {
			return nil
		}
		if currentInventory >= 0 && inventoryID < currentInventory {
			return errors.New("inventory_parts.csv must be ordered by inventory_id")
		}
		if inventoryID != currentInventory {
			flush()
			currentInventory = inventoryID
		}
		if !selectedInventories[inventoryID] || strings.EqualFold(row[4], "true") {
			return nil
		}
		quantity, err := strconv.ParseInt(row[3], 10, 64)
		if err != nil || quantity <= 0 {
			return nil
		}
		partNum := strings.ToLower(row[1])
		currentParts[partNum] += quantity
		return nil
	})
	flush()
	return result, err
}

func decorated(name string) bool {
	value := strings.ToLower(name)
	return strings.Contains(value, "sticker") || strings.Contains(value, "pattern") || strings.Contains(value, "printed")
}

func readGzipCSV(path string, visit func([]string) error) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	reader := csv.NewReader(compressed)
	reader.ReuseRecord = true
	if _, err := reader.Read(); err != nil {
		return err
	}
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := visit(row); err != nil {
			return err
		}
	}
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
