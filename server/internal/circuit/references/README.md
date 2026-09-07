# Module product assembly references

## First batch: Nezha V2 ultrasonic gate

`nezha-v2-gate.json` is a manually reviewed transcription of the ELECFREAKS
Nezha Inventor's Kit V2 (EF08288) case 31, checked on 2026-09-07. It is a
research record, not a `Catalog`, `Project`, `Document` or AI assembly recipe.

Sources:

- [V2 product documentation](https://wiki.elecfreaks.com/en/microbit/building-blocks/nezha-inventors-kit-v2/product-description/).
- [Case 31: ultrasonic gate](https://wiki.elecfreaks.com/en/microbit/building-blocks/nezha-inventors-kit-v2/the-ultrasonic-sound-gate/).
- [Pictured parts inventory](https://wiki-media-ef.oss-cn-hongkong.aliyuncs.com/i18n/en/docusaurus-plugin-content-docs/current/microbit/building-blocks/nezha-inventors-kit-v2/images/nezha-inventors-kit-v2-step-31-01.png).
- [Manufacturer-linked MakeCode program](https://makecode.microbit.org/_71MTtr1dH9go).

The 16 assembly images consist of **one parts inventory and 15 numbered
assembly steps**, not 16 assembly steps. All images were visually reviewed.
The inventory shows 16 part types and 36 pieces: 12 structural types / 32 pieces
and four electronic types / four pieces. Every pictured piece is introduced
exactly once by the step quantity ledger. A type may occur in several steps.
The case text specifies ultrasonic sensor → J1 and servo → S1 on the expansion
board. No connector, voltage or cable compatibility is inferred from those
two labels.

Part names describe visible appearance. IDs such as `yellow-medium-beam` are
local reference identifiers, **not manufacturer SKUs, LDraw IDs or evidence
that similarly coloured pieces are interchangeable**. No beam length, gear
tooth count, shaft length, hole coordinate, servo control mode or micro:bit
revision is inferred. The target plan prefers micro:bit V2, but the case image
does not establish that every listed board is V2. V1/V2/Pro evidence is not mixed.

This is not a complete build or purchase BOM: the pictured inventory omits
cables; power, cable lengths and routes require separate review. No family
stock is seeded, reserved or consumed from reference quantities.

## Integration and evidence boundaries

`AssemblyReferences()` lives beside the current circuit compiler and returns a
fresh, validated copy. The existing authenticated `GET /api/circuit/kits` adds
`assembly_references` beside `kits`. Research entries never enter `Catalogs()`
or `FindCatalog()`, so creation and inventory endpoints cannot enable them.
No database schema, queue, model call or generic assembly engine is added.

The shared Web/Desktop page is:

```text
/:workspace/circuit?reference=nezha-v2-ultrasonic-gate
```

It provides a labelled conceptual product sketch, a source-indexed step
browser, pictured quantities, stated wiring and external documentation links.
There are no build-progress or generation writes. The sketch is original
application artwork, not dimensionally checked installation guidance. Source
images are linked, not copied, embedded or redistributed. Any later use of
manufacturer artwork needs its own rights review or replacement artwork.

`content_hash` identifies the Go JSON encoding of **our reference record**
with an empty hash field. It is not a hash of remotely hosted image bytes and
does not establish that those remote images are immutable. Updating the
transcription requires a new reference version and check date.

The TS schema accepts only `status: research` and independently checks part
identity, ordered steps, quantities, endpoint references and URLs. Missing or
malformed optional references become an empty reference list without breaking
the existing Snap/BOSON kit response. An unavailable direct link shows an
explicit reference error instead of opening the generation workflow.

## What remains before a compilable assembly

The structured `unresolved` list is the next collection batch:

1. Exact part identities and board revisions, with dimensions and provenance.
2. Mounting geometry, shaft engagement and fastening requirements.
3. Actual servo type, supported control mode and reviewed motion clearance.
4. Cable types, lengths, routes and the complete power arrangement.
5. Pinned program dependencies, compilation, port binding and calibration.
6. A separately recorded physical build and functional trial.

Once the required engineering data exists, introduce the proposed v3 recipe
and deterministic assembly compiler. Do not upgrade this research record by
changing its status or passing it through the v1/v2 circuit compiler. The
current checks validate transcription consistency, not physical correctness.

## Regression checks

```sh
# From server/; domain checks need no database or external model.
go test ./internal/circuit -count=1

# Explicitly point DATABASE_URL at a migrated disposable database first.
go test ./internal/handler -run '^TestCircuitAssemblyReferenceCatalogBoundary$' -count=1

# From repository root.
pnpm --filter @chimii/core exec vitest run circuit diagnostics/diagnostic-context.test.ts
pnpm --filter @chimii/views exec vitest run circuit
# Requires local Web/API servers and a disposable DATABASE_URL.
pnpm exec playwright test e2e/circuit-assembly-reference.spec.ts
```

The browser test uses the real kits API and checks the full reference route,
all 15 steps, reload, unavailable references, Chinese/narrow layout and zero
creation, conversation or inventory writes. This batch makes no external
model calls and cannot establish physical acceptance.
