package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type lockFile struct {
	SchemaVersion int      `json:"schema_version"`
	Release       string   `json:"release"`
	SourceURL     string   `json:"source_url"`
	ArchiveSHA256 string   `json:"archive_sha256"`
	RootParts     []string `json:"root_parts"`
}

type vec3 struct{ X, Y, Z float64 }

type transform struct {
	M [9]float64
	T vec3
}

type triangle struct {
	A, B, C  vec3
	Material string
}

type parsedLine struct {
	Kind      byte
	Color     int
	Transform transform
	Points    []vec3
	Reference string
}

type sourceFile struct {
	Path    string
	Content []byte
	Lines   []parsedLine
}

type archiveLibrary struct {
	reader *zip.ReadCloser
	files  map[string]*zip.File
	cache  map[string]*sourceFile
}

type colorValue struct {
	Value string
	Edge  string
}

type catalogAsset struct {
	ID            string
	Hash          string
	AssetName     string
	Data          string
	TriangleCount int
	Dependencies  []string
	Bounds        [6]float64
}

type materialContext struct {
	Surface string
	Edge    string
}

func main() {
	archivePath := flag.String("archive", "", "path to the pinned complete.zip")
	lockPath := flag.String("lock", "catalog.lock.json", "catalog lock file")
	outPath := flag.String("out", "../../packages/views/build/catalog/catalog.generated.ts", "generated TypeScript output")
	check := flag.Bool("check", false, "verify that the output is already current")
	flag.Parse()

	if *archivePath == "" {
		fatal(errors.New("-archive is required"))
	}
	lock, err := readLock(*lockPath)
	if err != nil {
		fatal(err)
	}
	if err := verifyArchive(*archivePath, lock.ArchiveSHA256); err != nil {
		fatal(err)
	}
	library, err := openLibrary(*archivePath)
	if err != nil {
		fatal(err)
	}
	defer library.close()
	colors, err := library.colors()
	if err != nil {
		fatal(err)
	}

	assets := make([]catalogAsset, 0, len(lock.RootParts))
	for _, partID := range lock.RootParts {
		asset, err := library.compilePart(partID, colors)
		if err != nil {
			fatal(fmt.Errorf("compile %s: %w", partID, err))
		}
		assets = append(assets, asset)
	}
	generated, err := renderTypeScript(lock, assets)
	if err != nil {
		fatal(err)
	}
	if *check {
		existing, err := os.ReadFile(*outPath)
		if err != nil {
			fatal(err)
		}
		if !bytes.Equal(existing, generated) {
			fatal(errors.New("generated catalog is stale"))
		}
		fmt.Printf("catalog is current: %d parts\n", len(assets))
		return
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*outPath, generated, 0o644); err != nil {
		fatal(err)
	}
	for _, asset := range assets {
		fmt.Printf("%s  %d triangles  %d dependencies  %s\n", asset.ID, asset.TriangleCount, len(asset.Dependencies), asset.AssetName)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ldraw-catalog:", err)
	os.Exit(1)
}

func readLock(path string) (lockFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return lockFile{}, err
	}
	var lock lockFile
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&lock); err != nil {
		return lockFile{}, err
	}
	if lock.SchemaVersion != 1 || lock.Release == "" || lock.SourceURL == "" || len(lock.RootParts) == 0 {
		return lockFile{}, errors.New("invalid catalog lock")
	}
	if len(lock.ArchiveSHA256) != 64 {
		return lockFile{}, errors.New("invalid archive SHA-256")
	}
	seen := map[string]bool{}
	for _, id := range lock.RootParts {
		id = normalizeName(id)
		if id == "" || seen[id] {
			return lockFile{}, fmt.Errorf("invalid or duplicate root part %q", id)
		}
		seen[id] = true
	}
	return lock, nil
}

func verifyArchive(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("archive SHA-256 mismatch: got %s, want %s", actual, expected)
	}
	return nil
}

func openLibrary(path string) (*archiveLibrary, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	library := &archiveLibrary{reader: r, files: map[string]*zip.File{}, cache: map[string]*sourceFile{}}
	for _, f := range r.File {
		name := normalizeArchivePath(f.Name)
		if name != "" {
			library.files[name] = f
		}
	}
	return library, nil
}

func (l *archiveLibrary) close() {
	if l.reader != nil {
		_ = l.reader.Close()
	}
}

func normalizeArchivePath(name string) string {
	name = normalizeName(name)
	name = strings.TrimPrefix(name, "./")
	name = strings.TrimPrefix(name, "ldraw/")
	return name
}

func normalizeName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
}

func (l *archiveLibrary) source(path string) (*sourceFile, error) {
	path = normalizeArchivePath(path)
	if cached := l.cache[path]; cached != nil {
		return cached, nil
	}
	zf := l.files[path]
	if zf == nil {
		return nil, fmt.Errorf("source %q not found", path)
	}
	r, err := zf.Open()
	if err != nil {
		return nil, err
	}
	const maxSourceSize = 8 << 20
	raw, err := io.ReadAll(io.LimitReader(r, maxSourceSize+1))
	_ = r.Close()
	if err != nil {
		return nil, err
	}
	if len(raw) > maxSourceSize {
		return nil, fmt.Errorf("source %q exceeds %d bytes", path, maxSourceSize)
	}
	lines, err := parseSource(raw)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	result := &sourceFile{Path: path, Content: raw, Lines: lines}
	l.cache[path] = result
	return result, nil
}

func parseSource(raw []byte) ([]parsedLine, error) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	rows := strings.Split(text, "\n")
	result := make([]parsedLine, 0, len(rows))
	for rowIndex, row := range rows {
		fields := strings.Fields(row)
		if len(fields) == 0 || fields[0] == "0" || fields[0] == "5" {
			continue
		}
		kind := fields[0]
		need := map[string]int{"1": 15, "2": 8, "3": 11, "4": 14}[kind]
		if need == 0 {
			continue
		}
		if len(fields) < need {
			return nil, fmt.Errorf("line %d: type %s has %d fields", rowIndex+1, kind, len(fields))
		}
		color, err := parseInt(fields[1])
		if err != nil {
			return nil, fmt.Errorf("line %d color: %w", rowIndex+1, err)
		}
		line := parsedLine{Kind: kind[0], Color: color}
		switch kind {
		case "1":
			values, err := parseFloats(fields[2:14])
			if err != nil {
				return nil, fmt.Errorf("line %d transform: %w", rowIndex+1, err)
			}
			line.Transform = transform{
				T: vec3{values[0], values[1], values[2]},
				M: [9]float64{values[3], values[4], values[5], values[6], values[7], values[8], values[9], values[10], values[11]},
			}
			line.Reference = normalizeName(strings.Join(fields[14:], " "))
		case "2", "3", "4":
			values, err := parseFloats(fields[2:need])
			if err != nil {
				return nil, fmt.Errorf("line %d points: %w", rowIndex+1, err)
			}
			for i := 0; i < len(values); i += 3 {
				line.Points = append(line.Points, vec3{values[i], values[i+1], values[i+2]})
			}
		}
		result = append(result, line)
	}
	return result, nil
}

func parseInt(value string) (int, error) {
	base := 10
	if strings.HasPrefix(strings.ToLower(value), "0x") {
		base = 0
	}
	n, err := strconv.ParseInt(value, base, 32)
	return int(n), err
}

func parseFloats(values []string) ([]float64, error) {
	result := make([]float64, len(values))
	for i, value := range values {
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		result[i] = n
	}
	return result, nil
}

func identity() transform {
	return transform{M: [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}}
}

func (t transform) point(p vec3) vec3 {
	return vec3{
		X: t.M[0]*p.X + t.M[1]*p.Y + t.M[2]*p.Z + t.T.X,
		Y: t.M[3]*p.X + t.M[4]*p.Y + t.M[5]*p.Z + t.T.Y,
		Z: t.M[6]*p.X + t.M[7]*p.Y + t.M[8]*p.Z + t.T.Z,
	}
}

func compose(parent, child transform) transform {
	var result transform
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			for k := 0; k < 3; k++ {
				result.M[row*3+col] += parent.M[row*3+k] * child.M[k*3+col]
			}
		}
	}
	result.T = parent.point(child.T)
	return result
}

func resolveReference(name string, files map[string]*zip.File) (string, error) {
	name = normalizeName(name)
	candidates := []string{}
	switch {
	case strings.HasPrefix(name, "s/"):
		candidates = append(candidates, "parts/"+name)
	case strings.HasPrefix(name, "48/"), strings.HasPrefix(name, "8/"):
		candidates = append(candidates, "p/"+name)
	default:
		candidates = append(candidates, "parts/"+name, "p/"+name)
	}
	for _, candidate := range candidates {
		if files[candidate] != nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("reference %q not found", name)
}

func childMaterial(color int, parent materialContext) materialContext {
	if color == 16 {
		return parent
	}
	return materialContext{Surface: materialName(color), Edge: edgeMaterialName(color)}
}

func geometryMaterial(color int, parent materialContext) string {
	switch color {
	case 16:
		return parent.Surface
	case 24:
		return parent.Edge
	default:
		return materialName(color)
	}
}

func materialName(color int) string {
	if color == 16 {
		return "current"
	}
	if color >= 0x2000000 && color <= 0x2ffffff {
		return fmt.Sprintf("direct-%06x", color&0xffffff)
	}
	return fmt.Sprintf("fixed-%d", color)
}

func edgeMaterialName(color int) string {
	if color == 16 {
		return "edge-current"
	}
	return fmt.Sprintf("edge-fixed-%d", color)
}

func (l *archiveLibrary) flatten(path string, matrix transform, material materialContext, stack map[string]bool, deps map[string]bool, out *[]triangle) error {
	if stack[path] {
		return fmt.Errorf("cyclic reference through %s", path)
	}
	stack[path] = true
	defer delete(stack, path)
	deps[path] = true
	source, err := l.source(path)
	if err != nil {
		return err
	}
	for _, line := range source.Lines {
		switch line.Kind {
		case '1':
			childPath, err := resolveReference(line.Reference, l.files)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			if err := l.flatten(childPath, compose(matrix, line.Transform), childMaterial(line.Color, material), stack, deps, out); err != nil {
				return err
			}
		case '3':
			*out = append(*out, triangle{matrix.point(line.Points[0]), matrix.point(line.Points[1]), matrix.point(line.Points[2]), geometryMaterial(line.Color, material)})
		case '4':
			a, b, c, d := matrix.point(line.Points[0]), matrix.point(line.Points[1]), matrix.point(line.Points[2]), matrix.point(line.Points[3])
			name := geometryMaterial(line.Color, material)
			*out = append(*out, triangle{a, b, c, name}, triangle{a, c, d, name})
		}
	}
	return nil
}

func (l *archiveLibrary) compilePart(partID string, colors map[int]colorValue) (catalogAsset, error) {
	root, err := resolveReference(partID, l.files)
	if err != nil {
		return catalogAsset{}, err
	}
	deps := map[string]bool{}
	triangles := []triangle{}
	if err := l.flatten(root, identity(), materialContext{Surface: "current", Edge: "edge-current"}, map[string]bool{}, deps, &triangles); err != nil {
		return catalogAsset{}, err
	}
	if len(triangles) == 0 {
		return catalogAsset{}, errors.New("part contains no triangles")
	}
	glb, bounds, err := buildGLB(partID, triangles, colors)
	if err != nil {
		return catalogAsset{}, err
	}
	hash := sha256.Sum256(glb)
	dependencyList := make([]string, 0, len(deps))
	for path := range deps {
		dependencyList = append(dependencyList, path)
	}
	sort.Strings(dependencyList)
	return catalogAsset{
		ID: normalizeName(partID), Hash: hex.EncodeToString(hash[:]), AssetName: hex.EncodeToString(hash[:]) + ".glb",
		Data: base64.StdEncoding.EncodeToString(glb), TriangleCount: len(triangles), Dependencies: dependencyList, Bounds: bounds,
	}, nil
}

func (l *archiveLibrary) colors() (map[int]colorValue, error) {
	source, err := l.source("ldconfig.ldr")
	if err != nil {
		return nil, err
	}
	result := map[int]colorValue{}
	for _, row := range strings.Split(strings.ReplaceAll(string(source.Content), "\r\n", "\n"), "\n") {
		fields := strings.Fields(row)
		if len(fields) < 9 || fields[0] != "0" || fields[1] != "!COLOUR" {
			continue
		}
		var code int
		var value, edge string
		for i := 2; i+1 < len(fields); i++ {
			switch fields[i] {
			case "CODE":
				code, _ = strconv.Atoi(fields[i+1])
			case "VALUE":
				value = fields[i+1]
			case "EDGE":
				edge = fields[i+1]
			}
		}
		if value != "" {
			result[code] = colorValue{Value: value, Edge: edge}
		}
	}
	if len(result) == 0 {
		return nil, errors.New("LDConfig.ldr contains no colours")
	}
	return result, nil
}

func renderTypeScript(lock lockFile, assets []catalogAsset) ([]byte, error) {
	sort.Slice(assets, func(i, j int) bool { return assets[i].ID < assets[j].ID })
	var b strings.Builder
	b.WriteString("// Code generated by tools/ldraw-catalog. DO NOT EDIT.\n")
	b.WriteString("// Official LDraw source files are not embedded; each value is a compiled GLB.\n\n")
	fmt.Fprintf(&b, "export const LDRAW_CATALOG_VERSION = %q;\n", "ldraw-official-"+lock.Release+"-"+lock.ArchiveSHA256[:12])
	fmt.Fprintf(&b, "export const LDRAW_SOURCE = { release: %q, archiveSha256: %q, url: %q } as const;\n\n", lock.Release, lock.ArchiveSHA256, lock.SourceURL)
	b.WriteString("export interface LDrawCatalogAsset {\n  hash: string;\n  assetName: string;\n  glbBase64: string;\n  triangleCount: number;\n  dependencies: readonly string[];\n  bounds: readonly [number, number, number, number, number, number];\n}\n\n")
	b.WriteString("export const LDRAW_CATALOG: Readonly<Record<string, LDrawCatalogAsset>> = {\n")
	for _, asset := range assets {
		deps, _ := json.Marshal(asset.Dependencies)
		fmt.Fprintf(&b, "  %q: {\n", asset.ID)
		fmt.Fprintf(&b, "    hash: %q,\n    assetName: %q,\n", asset.Hash, asset.AssetName)
		fmt.Fprintf(&b, "    glbBase64: %q,\n", asset.Data)
		fmt.Fprintf(&b, "    triangleCount: %d,\n    dependencies: %s,\n", asset.TriangleCount, deps)
		fmt.Fprintf(&b, "    bounds: [%g, %g, %g, %g, %g, %g],\n", asset.Bounds[0], asset.Bounds[1], asset.Bounds[2], asset.Bounds[3], asset.Bounds[4], asset.Bounds[5])
		b.WriteString("  },\n")
	}
	b.WriteString("};\n")
	return []byte(b.String()), nil
}

type gltfDocument struct {
	Asset       map[string]string `json:"asset"`
	Scene       int               `json:"scene"`
	Scenes      []map[string]any  `json:"scenes"`
	Nodes       []map[string]any  `json:"nodes"`
	Meshes      []map[string]any  `json:"meshes"`
	Materials   []map[string]any  `json:"materials"`
	Buffers     []map[string]any  `json:"buffers"`
	BufferViews []map[string]any  `json:"bufferViews"`
	Accessors   []map[string]any  `json:"accessors"`
}

func buildGLB(partID string, triangles []triangle, colors map[int]colorValue) ([]byte, [6]float64, error) {
	groups := map[string][]triangle{}
	bounds := [6]float64{math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, tri := range triangles {
		groups[tri.Material] = append(groups[tri.Material], tri)
		for _, p := range []vec3{tri.A, tri.B, tri.C} {
			bounds[0] = math.Min(bounds[0], p.X)
			bounds[1] = math.Min(bounds[1], p.Y)
			bounds[2] = math.Min(bounds[2], p.Z)
			bounds[3] = math.Max(bounds[3], p.X)
			bounds[4] = math.Max(bounds[4], p.Y)
			bounds[5] = math.Max(bounds[5], p.Z)
		}
	}
	materialNames := make([]string, 0, len(groups))
	for name := range groups {
		materialNames = append(materialNames, name)
	}
	sort.Strings(materialNames)

	doc := gltfDocument{
		Asset: map[string]string{"version": "2.0", "generator": "CHIMII LDraw Catalog Compiler"}, Scene: 0,
		Scenes: []map[string]any{{"nodes": []int{0}}}, Nodes: []map[string]any{{"mesh": 0, "name": partID}},
	}
	binChunk := bytes.Buffer{}
	primitives := make([]map[string]any, 0, len(materialNames))
	for materialIndex, name := range materialNames {
		color := materialColor(name, colors)
		doc.Materials = append(doc.Materials, map[string]any{
			"name": name, "doubleSided": true,
			"pbrMetallicRoughness": map[string]any{"baseColorFactor": color, "metallicFactor": 0, "roughnessFactor": 0.38},
		})
		positions := []float32{}
		normals := []float32{}
		indices := []uint32{}
		vertexIndex := map[string]uint32{}
		minPos := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
		maxPos := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
		for _, tri := range groups[name] {
			normal := faceNormal(tri.A, tri.B, tri.C)
			for _, p := range []vec3{tri.A, tri.B, tri.C} {
				key := fmt.Sprintf("%.6f/%.6f/%.6f/%.6f/%.6f/%.6f", p.X, p.Y, p.Z, normal.X, normal.Y, normal.Z)
				index, ok := vertexIndex[key]
				if !ok {
					index = uint32(len(positions) / 3)
					vertexIndex[key] = index
					positions = append(positions, float32(p.X), float32(p.Y), float32(p.Z))
					normals = append(normals, float32(normal.X), float32(normal.Y), float32(normal.Z))
					minPos[0] = math.Min(minPos[0], p.X)
					minPos[1] = math.Min(minPos[1], p.Y)
					minPos[2] = math.Min(minPos[2], p.Z)
					maxPos[0] = math.Max(maxPos[0], p.X)
					maxPos[1] = math.Max(maxPos[1], p.Y)
					maxPos[2] = math.Max(maxPos[2], p.Z)
				}
				indices = append(indices, index)
			}
		}
		positionAccessor := appendFloatAccessor(&doc, &binChunk, positions, "VEC3", minPos[:], maxPos[:])
		normalAccessor := appendFloatAccessor(&doc, &binChunk, normals, "VEC3", nil, nil)
		indexAccessor := appendIndexAccessor(&doc, &binChunk, indices)
		primitives = append(primitives, map[string]any{
			"attributes": map[string]int{"POSITION": positionAccessor, "NORMAL": normalAccessor},
			"indices":    indexAccessor, "material": materialIndex, "mode": 4,
		})
	}
	doc.Meshes = []map[string]any{{"name": partID, "primitives": primitives}}
	doc.Buffers = []map[string]any{{"byteLength": binChunk.Len()}}
	jsonChunk, err := json.Marshal(doc)
	if err != nil {
		return nil, bounds, err
	}
	jsonChunk = pad(jsonChunk, 0x20)
	binBytes := pad(binChunk.Bytes(), 0)
	total := 12 + 8 + len(jsonChunk) + 8 + len(binBytes)
	result := bytes.NewBuffer(make([]byte, 0, total))
	_ = binary.Write(result, binary.LittleEndian, uint32(0x46546c67))
	_ = binary.Write(result, binary.LittleEndian, uint32(2))
	_ = binary.Write(result, binary.LittleEndian, uint32(total))
	_ = binary.Write(result, binary.LittleEndian, uint32(len(jsonChunk)))
	_ = binary.Write(result, binary.LittleEndian, uint32(0x4e4f534a))
	result.Write(jsonChunk)
	_ = binary.Write(result, binary.LittleEndian, uint32(len(binBytes)))
	_ = binary.Write(result, binary.LittleEndian, uint32(0x004e4942))
	result.Write(binBytes)
	return result.Bytes(), bounds, nil
}

func faceNormal(a, b, c vec3) vec3 {
	u := vec3{b.X - a.X, b.Y - a.Y, b.Z - a.Z}
	v := vec3{c.X - a.X, c.Y - a.Y, c.Z - a.Z}
	n := vec3{u.Y*v.Z - u.Z*v.Y, u.Z*v.X - u.X*v.Z, u.X*v.Y - u.Y*v.X}
	length := math.Sqrt(n.X*n.X + n.Y*n.Y + n.Z*n.Z)
	if length == 0 {
		return vec3{Y: 1}
	}
	return vec3{n.X / length, n.Y / length, n.Z / length}
}

func materialColor(name string, colors map[int]colorValue) []float64 {
	hexColor := "#b7c0c8"
	switch {
	case name == "current":
		hexColor = "#b7c0c8"
	case strings.HasPrefix(name, "direct-"):
		hexColor = "#" + strings.TrimPrefix(name, "direct-")
	case strings.HasPrefix(name, "fixed-"):
		code, _ := strconv.Atoi(strings.TrimPrefix(name, "fixed-"))
		if value := colors[code].Value; value != "" {
			hexColor = value
		}
	}
	r, g, b := parseHexColor(hexColor)
	return []float64{r, g, b, 1}
}

func parseHexColor(value string) (float64, float64, float64) {
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 {
		return 0.7, 0.7, 0.7
	}
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return 0.7, 0.7, 0.7
	}
	return float64((n>>16)&0xff) / 255, float64((n>>8)&0xff) / 255, float64(n&0xff) / 255
}

func appendFloatAccessor(doc *gltfDocument, binChunk *bytes.Buffer, values []float32, kind string, min, max []float64) int {
	align4(binChunk)
	offset := binChunk.Len()
	for _, value := range values {
		_ = binary.Write(binChunk, binary.LittleEndian, value)
	}
	view := len(doc.BufferViews)
	doc.BufferViews = append(doc.BufferViews, map[string]any{"buffer": 0, "byteOffset": offset, "byteLength": len(values) * 4, "target": 34962})
	accessor := map[string]any{"bufferView": view, "componentType": 5126, "count": len(values) / 3, "type": kind}
	if min != nil {
		accessor["min"] = min
	}
	if max != nil {
		accessor["max"] = max
	}
	doc.Accessors = append(doc.Accessors, accessor)
	return len(doc.Accessors) - 1
}

func appendIndexAccessor(doc *gltfDocument, binChunk *bytes.Buffer, values []uint32) int {
	align4(binChunk)
	offset := binChunk.Len()
	for _, value := range values {
		_ = binary.Write(binChunk, binary.LittleEndian, value)
	}
	view := len(doc.BufferViews)
	doc.BufferViews = append(doc.BufferViews, map[string]any{"buffer": 0, "byteOffset": offset, "byteLength": len(values) * 4, "target": 34963})
	doc.Accessors = append(doc.Accessors, map[string]any{"bufferView": view, "componentType": 5125, "count": len(values), "type": "SCALAR"})
	return len(doc.Accessors) - 1
}

func align4(buffer *bytes.Buffer) {
	for buffer.Len()%4 != 0 {
		buffer.WriteByte(0)
	}
}

func pad(value []byte, fill byte) []byte {
	result := append([]byte(nil), value...)
	for len(result)%4 != 0 {
		result = append(result, fill)
	}
	return result
}
