# CHIMII LDraw catalog compiler

CHIMII maintains two catalog layers from one pinned **official** LDraw Parts
Library archive:

- a ranked 1,000-part Starter Kit synchronized into PostgreSQL by the server;
- ten core fallback parts embedded in the Web/Desktop bundle.

The complete archive and `.dat` source files are never shipped with CHIMII.

The official source is fixed by
`server/internal/ldrawsync/catalog.lock.json`. To regenerate the small embedded
fallback catalog:

```bash
curl -fL -o /tmp/ldraw-complete.zip \
  https://library.ldraw.org/library/updates/complete.zip
cd tools/ldraw-catalog
go run . \
  -archive /tmp/ldraw-complete.zip \
  -out ../../packages/views/build/catalog/catalog.generated.ts
```

The command fails before parsing if the archive SHA-256 differs from the lock.
Update the release and hash only after reviewing the official release and the
`!LICENSE` header of every Starter Kit root part.

The production Starter Kit is fixed by
`server/internal/ldrawsync/starter-kit-1000.json`. Regenerate it from pinned
Rebrickable CSV snapshots with `go run ./rank`; the manifest records every
input URL and SHA-256 plus the ranking/filter rules. The server embeds both
JSON files, so a deployed binary does not depend on repository files.

The parser accepts the geometry records required by official rigid parts:
subfile references, lines, triangles, quads, inherited colours, and nested
integer/decimal transforms. GLB root-node `extras.ldrawLines` (version 1)
contains groups of ordinary lines (two endpoints) and conditional lines (two
endpoints plus two control points), flattened in the same coordinates as the
surfaces. Group colors are sRGB hex strings, `current`, or `edge-current`.
Control points never contribute to mesh bounds, inventory, or connectors.
Surface `baseColorFactor` values are converted from sRGB to linear glTF colors.
BFC winding is not trusted as a visibility rule; surfaces remain double-sided
with deterministic flat normals.

Render revision 2 uses the catalog suffix `-r2`, preserving the pinned source
release and archive hash. Rebuild both the embedded and server catalogs rather
than overwriting the old immutable asset URLs. Web/desktop and mobile use the
bundled assets only when the requested catalog version matches. Old GLBs remain
readable; web/desktop retain their historical geometry-derived outlines.
New assets use Three.js's official conditional-line material. The rendering
preset is `ldraw-studio-v2`.

After regenerating the GLB catalog, regenerate the card thumbnails from the
same Three.js render preset:

```bash
pnpm generate:ldraw-thumbnails
node apps/mobile/scripts/build-maker-renderer.mjs
```

The thumbnail generator renders every catalog part in every supported colour
inside an isolated Playwright page. It writes content-addressed WebP assets and
a TypeScript manifest for the shared web/desktop build page.
