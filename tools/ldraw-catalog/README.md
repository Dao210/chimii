# CHIMII LDraw catalog compiler

This build-time tool converts only the Starter Kit dependency closure from a
pinned **official** LDraw Parts Library archive into content-addressed GLB
meshes embedded in a lazy-loaded TypeScript catalog. The complete archive and
`.dat` source files are never shipped with CHIMII.

The input is fixed by `catalog.lock.json`. To regenerate:

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

The parser accepts the geometry records required by official rigid parts:
subfile references, lines, triangles, quads, inherited colours, and nested
integer/decimal transforms. GLB meshes use the surface records; line records
are parsed for format compatibility, while runtime outlines are generated from
mesh edges. BFC winding is not trusted as a visibility rule; generated
materials are double-sided and carry deterministic flat normals.
