package build

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
)

const SceneRendererVersion = "catalog-glb-views-v1"

type SceneView struct {
	Direction string `json:"direction"`
	SHA256    string `json:"sha256"`
	PNG       []byte `json:"-"`
}

type VisualReviewEvidence struct {
	Renderer    string      `json:"renderer"`
	ContentHash string      `json:"content_hash"`
	Views       []SceneView `json:"views"`
}

type renderVector struct{ x, y, z float64 }
type renderTriangle struct {
	points   [3]renderVector
	material int
}
type renderMaterial struct {
	current bool
	rgb     color.RGBA
}
type renderPart struct {
	triangles []renderTriangle
	materials []renderMaterial
	min, max  renderVector
}
type renderAccessor struct {
	BufferView, ByteOffset, ComponentType, Count int
	Type                                         string
	Sparse                                       json.RawMessage
}
type renderBufferView struct{ Buffer, ByteOffset, ByteLength, ByteStride int }
type renderGLB struct {
	Asset struct{ Version, Generator string }
	Nodes []struct {
		Mesh                                 *int
		Matrix, Translation, Rotation, Scale []float64
	}
	Meshes []struct {
		Primitives []struct {
			Attributes        map[string]int
			Indices, Material *int
			Mode              *int
			Extensions        json.RawMessage
		}
	}
	Materials []struct {
		Name string
		PBR  struct {
			BaseColorFactor []float64 `json:"baseColorFactor"`
		} `json:"pbrMetallicRoughness"`
	}
	Buffers []struct {
		ByteLength int
		URI        string
	}
	BufferViews []renderBufferView
	Accessors   []renderAccessor
}

func decodeRenderPart(raw []byte) (renderPart, error) {
	bad := func() (renderPart, error) { return renderPart{}, fmt.Errorf("unsupported or malformed catalog GLB") }
	if len(raw) < 28 || len(raw) > 16<<20 || binary.LittleEndian.Uint32(raw[:4]) != 0x46546c67 || binary.LittleEndian.Uint32(raw[4:8]) != 2 || int(binary.LittleEndian.Uint32(raw[8:12])) != len(raw) {
		return bad()
	}
	n := int(binary.LittleEndian.Uint32(raw[12:16]))
	if n < 2 || n > len(raw)-28 || binary.LittleEndian.Uint32(raw[16:20]) != 0x4e4f534a {
		return bad()
	}
	var doc renderGLB
	if json.Unmarshal(raw[20:20+n], &doc) != nil {
		return bad()
	}
	b := 20 + n
	length := int(binary.LittleEndian.Uint32(raw[b : b+4]))
	if length != len(raw)-b-8 || binary.LittleEndian.Uint32(raw[b+4:b+8]) != 0x004e4942 {
		return bad()
	}
	data := raw[b+8:]
	if doc.Asset.Version != "2.0" || doc.Asset.Generator != "CHIMII LDraw Catalog Compiler" || len(doc.Nodes) != 1 || doc.Nodes[0].Mesh == nil || *doc.Nodes[0].Mesh != 0 || len(doc.Nodes[0].Matrix)+len(doc.Nodes[0].Translation)+len(doc.Nodes[0].Rotation)+len(doc.Nodes[0].Scale) != 0 || len(doc.Meshes) != 1 || len(doc.Buffers) != 1 || doc.Buffers[0].URI != "" || doc.Buffers[0].ByteLength <= 0 || doc.Buffers[0].ByteLength > len(data) {
		return bad()
	}
	read := func(index, component, width int, kind string) ([]byte, int, error) {
		if index < 0 || index >= len(doc.Accessors) {
			return nil, 0, fmt.Errorf("invalid GLB accessor")
		}
		a := doc.Accessors[index]
		if a.ComponentType != component || a.Type != kind || a.Count <= 0 || a.Count > 1_000_000 || a.BufferView < 0 || a.BufferView >= len(doc.BufferViews) || a.ByteOffset < 0 || len(a.Sparse) > 0 {
			return nil, 0, fmt.Errorf("unsupported GLB accessor")
		}
		v := doc.BufferViews[a.BufferView]
		size := a.Count * width
		if v.Buffer != 0 || v.ByteStride != 0 || v.ByteOffset < 0 || v.ByteLength < 0 || v.ByteOffset > len(data) || v.ByteLength > len(data)-v.ByteOffset || a.ByteOffset > v.ByteLength || size > v.ByteLength-a.ByteOffset {
			return nil, 0, fmt.Errorf("invalid GLB buffer range")
		}
		return data[v.ByteOffset+a.ByteOffset : v.ByteOffset+a.ByteOffset+size], a.Count, nil
	}
	part := renderPart{min: renderVector{math.Inf(1), math.Inf(1), math.Inf(1)}, max: renderVector{math.Inf(-1), math.Inf(-1), math.Inf(-1)}}
	for _, m := range doc.Materials {
		if len(m.PBR.BaseColorFactor) != 4 {
			return bad()
		}
		rgb := color.RGBA{A: 255}
		channels := []*uint8{&rgb.R, &rgb.G, &rgb.B}
		for i, c := range m.PBR.BaseColorFactor {
			if math.IsNaN(c) || math.IsInf(c, 0) || c < 0 || c > 1 {
				return bad()
			}
			if i < 3 {
				v := 12.92 * c
				if c > 0.0031308 {
					v = 1.055*math.Pow(c, 1/2.4) - 0.055
				}
				*channels[i] = uint8(math.Round(v * 255))
			}
		}
		part.materials = append(part.materials, renderMaterial{current: m.Name == "current", rgb: rgb})
	}
	for _, primitive := range doc.Meshes[0].Primitives {
		position, exists := primitive.Attributes["POSITION"]
		if !exists || primitive.Indices == nil || primitive.Material == nil || *primitive.Material < 0 || *primitive.Material >= len(part.materials) || (primitive.Mode != nil && *primitive.Mode != 4) || len(primitive.Extensions) > 0 {
			return bad()
		}
		positions, count, err := read(position, 5126, 12, "VEC3")
		if err != nil {
			return renderPart{}, err
		}
		indices, indexCount, err := read(*primitive.Indices, 5125, 4, "SCALAR")
		if err != nil {
			return renderPart{}, err
		}
		if indexCount%3 != 0 {
			return bad()
		}
		for i := 0; i < indexCount; i += 3 {
			triangle := renderTriangle{material: *primitive.Material}
			for j := 0; j < 3; j++ {
				index := int(binary.LittleEndian.Uint32(indices[(i+j)*4:]))
				if index >= count {
					return bad()
				}
				v := [3]float64{}
				for k := 0; k < 3; k++ {
					v[k] = float64(math.Float32frombits(binary.LittleEndian.Uint32(positions[index*12+k*4:])))
					if math.IsNaN(v[k]) || math.IsInf(v[k], 0) || math.Abs(v[k]) > 100000 {
						return bad()
					}
				}
				p := renderVector{v[0], v[1], v[2]}
				triangle.points[j] = p
				part.min = renderVector{min(part.min.x, p.x), min(part.min.y, p.y), min(part.min.z, p.z)}
				part.max = renderVector{max(part.max.x, p.x), max(part.max.y, p.y), max(part.max.z, p.z)}
			}
			part.triangles = append(part.triangles, triangle)
			if len(part.triangles) > 200000 {
				return bad()
			}
		}
	}
	if len(part.triangles) == 0 {
		return bad()
	}
	return part, nil
}

func renderDot(a, b renderVector) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }
func renderCross(a, b renderVector) renderVector {
	return renderVector{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
}
func renderSub(a, b renderVector) renderVector { return renderVector{a.x - b.x, a.y - b.y, a.z - b.z} }
func renderUnit(a renderVector) renderVector {
	n := math.Sqrt(renderDot(a, a))
	if n == 0 {
		return a
	}
	return renderVector{a.x / n, a.y / n, a.z / n}
}

func renderWorld(p renderVector, placement Placement, part PartSpec) renderVector {
	x, z := p.x, p.z
	switch normalizedRotation(placement.Rotation) {
	case 90:
		x, z = -z, x
	case 180:
		x, z = -x, -z
	case 270:
		x, z = z, -x
	}
	sx, sz := part.StudsX, part.StudsZ
	if placement.Rotation%180 != 0 {
		sx, sz = sz, sx
	}
	return renderVector{float64(placement.X*20+sx*10) + x, float64((placement.Y+part.PlatesY)*8-part.OriginYOffsetLDU) - p.y, float64(placement.Z*20+sz*10+part.OriginCenterZOffsetLDU) + z}
}

// RenderSceneViews renders only versioned catalog surfaces. Missing assets and
// unsupported formats fail explicitly; occupancy boxes are never substituted.
func RenderSceneViews(ctx context.Context, plan BuildPlan, load func(context.Context, string, string) ([]byte, error), palette map[int]uint32) ([]SceneView, error) {
	if len(plan.Placements) == 0 || len(plan.Placements) > MaxSceneParts {
		return nil, fmt.Errorf("scene outside render bounds")
	}
	parts := map[string]renderPart{}
	triangles := 0
	lo, hi := renderVector{math.Inf(1), math.Inf(1), math.Inf(1)}, renderVector{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, p := range plan.Placements {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		spec, ok := plan.Parts[p.PartID]
		if !ok || p.Rotation%90 != 0 {
			return nil, fmt.Errorf("invalid render placement %s", p.ID)
		}
		part, ok := parts[p.PartID]
		if !ok {
			raw, err := load(ctx, plan.CatalogVersion, spec.LDrawID)
			if err != nil {
				return nil, fmt.Errorf("load review geometry %s: %w", spec.LDrawID, err)
			}
			part, err = decodeRenderPart(raw)
			if err != nil {
				return nil, fmt.Errorf("decode review geometry %s: %w", spec.LDrawID, err)
			}
			parts[p.PartID] = part
		}
		if _, ok := palette[p.Color]; !ok {
			return nil, fmt.Errorf("unknown review color %d", p.Color)
		}
		triangles += len(part.triangles)
		if triangles > 3_000_000 {
			return nil, fmt.Errorf("scene review triangle budget exceeded")
		}
		for _, x := range []float64{part.min.x, part.max.x} {
			for _, y := range []float64{part.min.y, part.max.y} {
				for _, z := range []float64{part.min.z, part.max.z} {
					q := renderWorld(renderVector{x, y, z}, p, spec)
					lo = renderVector{min(lo.x, q.x), min(lo.y, q.y), min(lo.z, q.z)}
					hi = renderVector{max(hi.x, q.x), max(hi.y, q.y), max(hi.z, q.z)}
				}
			}
		}
	}
	center := renderVector{(lo.x + hi.x) / 2, (lo.y + hi.y) / 2, (lo.z + hi.z) / 2}
	radius := math.Sqrt(renderDot(renderSub(hi, lo), renderSub(hi, lo))) / 2
	if radius <= 0 || math.IsNaN(radius) || math.IsInf(radius, 0) {
		return nil, fmt.Errorf("empty review bounds")
	}
	const size = 336
	scale := float64(size-24) / (2 * radius)
	cameras := []struct {
		name      string
		direction renderVector
	}{{"front", renderVector{0, 0, -1}}, {"back", renderVector{0, 0, 1}}, {"left", renderVector{-1, 0, 0}}, {"right", renderVector{1, 0, 0}}, {"top", renderVector{0, 1, 0}}, {"isometric", renderVector{1, 0.8, -1}}}
	views := []SceneView{}
	for _, camera := range cameras {
		direction := renderUnit(camera.direction)
		up := renderVector{0, 1, 0}
		if camera.name == "top" {
			up = renderVector{0, 0, 1}
		}
		right := renderUnit(renderCross(direction, up))
		up = renderUnit(renderCross(right, direction))
		canvas := image.NewRGBA(image.Rect(0, 0, size, size))
		for i := range canvas.Pix {
			canvas.Pix[i] = 255
		}
		depth := make([]float64, size*size)
		for i := range depth {
			depth[i] = math.Inf(-1)
		}
		for _, placement := range plan.Placements {
			part := parts[placement.PartID]
			spec := plan.Parts[placement.PartID]
			rgb := palette[placement.Color]
			for index, triangle := range part.triangles {
				if index%256 == 0 {
					if err := ctx.Err(); err != nil {
						return nil, err
					}
				}
				world := [3]renderVector{}
				screen := [3]renderVector{}
				for i, p := range triangle.points {
					world[i] = renderWorld(p, placement, spec)
					q := renderSub(world[i], center)
					screen[i] = renderVector{float64(size)/2 + renderDot(q, right)*scale, float64(size)/2 - renderDot(q, up)*scale, renderDot(q, direction)}
				}
				material := part.materials[triangle.material]
				c := material.rgb
				if material.current {
					c = color.RGBA{uint8(rgb >> 16), uint8(rgb >> 8), uint8(rgb), 255}
				}
				normal := renderUnit(renderCross(renderSub(world[1], world[0]), renderSub(world[2], world[0])))
				shade := 0.65 + 0.35*math.Abs(renderDot(normal, renderUnit(renderVector{-0.4, 1, -0.6})))
				c.R = uint8(float64(c.R) * shade)
				c.G = uint8(float64(c.G) * shade)
				c.B = uint8(float64(c.B) * shade)
				rasterReviewTriangle(canvas, depth, screen, c)
			}
		}
		var out bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.BestSpeed}
		if err := encoder.Encode(&out, canvas); err != nil {
			return nil, err
		}
		hash := sha256.Sum256(out.Bytes())
		views = append(views, SceneView{Direction: camera.name, SHA256: hex.EncodeToString(hash[:]), PNG: out.Bytes()})
	}
	return views, nil
}

func rasterReviewTriangle(canvas *image.RGBA, depth []float64, p [3]renderVector, c color.RGBA) {
	area := (p[1].y-p[2].y)*(p[0].x-p[2].x) + (p[2].x-p[1].x)*(p[0].y-p[2].y)
	if math.Abs(area) < 1e-9 {
		return
	}
	minX := max(0, int(math.Floor(min(p[0].x, p[1].x, p[2].x))))
	maxX := min(canvas.Rect.Dx()-1, int(math.Ceil(max(p[0].x, p[1].x, p[2].x))))
	minY := max(0, int(math.Floor(min(p[0].y, p[1].y, p[2].y))))
	maxY := min(canvas.Rect.Dy()-1, int(math.Ceil(max(p[0].y, p[1].y, p[2].y))))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			a := ((p[1].y-p[2].y)*(px-p[2].x) + (p[2].x-p[1].x)*(py-p[2].y)) / area
			b := ((p[2].y-p[0].y)*(px-p[2].x) + (p[0].x-p[2].x)*(py-p[2].y)) / area
			d := 1 - a - b
			if a < 0 || b < 0 || d < 0 {
				continue
			}
			z := a*p[0].z + b*p[1].z + d*p[2].z
			i := y*canvas.Rect.Dx() + x
			if z > depth[i] {
				depth[i] = z
				canvas.SetRGBA(x, y, c)
			}
		}
	}
}
