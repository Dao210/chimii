# Electronic construction

This module turns an idea or a selected project into a saved, step-by-step
assembly document for children using an existing snap-circuit kit. Its first
catalogue targets **Snap Circuits SC-500**, using only the components needed by
three documented projects: a switched lamp, an alarm sound, and a tabletop FM
radio with volume control. Quantities are a subset of that kit, not its full BOM.
Parts from other brands or kit revisions are not assumed interchangeable.

## Boundaries

```text
Idea / selected project + kit version + available quantities
              |
     Plan: bounded model intent (optional)
              |
     Compile: reference project + component catalogue
              |
     Validate: inventory, snaps, layers, bodies, netlist, shorts
              |
     Document v1 -> immutable JSONB snapshot
              |
     Shared SVG workbench -> separate progress / family observation
```

`internal/circuit` imports no brick compiler, HTTP, database, or provider SDK.
`TextGenerator` is the provider boundary. The handler reuses the platform's
configured text-model client and workspace/parent/child authentication.
The frontend has its own `packages/core/circuit` and `packages/views/circuit`
entry points; Web and Electron mount the same views at `/:workspace/circuit`.
Native mobile UI is outside this version; the Web view supports narrow screens
and an enlarged, horizontally scrollable assembly diagram.

The model chooses one supported project and a short title. It does not supply
electrical connections, coordinates, instructions, or inferred component
substitutions. Unsupported functions must return `unsupported`. The optional
model call has a 20-second deadline. Selecting a project remains available when
the platform model is unconfigured. There is no firmware, flashing, circuit
simulation, background agent job, or general-purpose circuit synthesis here.

## Catalogue and verification evidence

`catalog.json` is the server-owned, embedded source of truth. Each component
defines snap coordinates, rotation, occupancy shape, part ID, and quantity.
Each project separately records its functional-pin netlist, placement layout,
instructions, original PDF URL, page number, and SHA-256 of the reviewed PDF.
Instructions and simplified vector shapes were authored for this application.

| Project | Manufacturer reference | PDF page |
| --- | --- | --- |
| `switch-light` | SC-100 project 1, Electric Light and Switch | 9 |
| `alarm-sound` | SC-100 project 17, Alarm Circuit | 17 |
| `fm-radio` | SC-500 project 307, Adjustable Volume FM Radio | 85 |

Reviewed sources:

- [SC-100 REV-H manual](https://www.elenco.com/wp-content/uploads/2011/03/SC-100_REV-H_071712.pdf), SHA-256 `2a49a880eed07f7cd32a80dc273d1d5b296a5576852a3c9dacf2aed6b4ac3155`.
- [SC-500 combined manual / UC60](https://www.elenco.com/wp-content/uploads/2017/11/UC60.pdf), SHA-256 `9ead0e558c6ca2ebde6e75e3c1d2d71a30053015a67d9ca3a5a5dcef3c9d585f`.

Positions use integer grid coordinates, right-angle rotations, and layers 1–3.
The UI labels grid points A1–G10. Snaps connect only at identical coordinates on
adjacent layers. Conductive groups describe wires; IC internals are not inferred.
Elevated parts require supports, and the independently stored reference netlist
must match the netlist derived from layout. The short check closes S1 without
treating loads as wires. Body checks cover the catalogue's rectangles, bars, and
triangle on a quarter-grid; this is not a dimensional manufacturing model.

The radio has two battery holders and three layers. At D5–F5 it uses **two
2-snap connectors on successive layers**, not a single 3-snap connector. The
F6–F7 bridge sits on layer 3. Preserve these details when changing rendering or
templates; electrical equivalence alone does not establish physical fit.

`validation.passed` means these reference-consistency checks passed. Every
current document has `physical_verification: "not_tested"`. No bench test or
simulation has been performed. A family's `worked` / `needs_help` observation
never changes that evidence. Validate actual stock, assembly, and operation
with the matching kit before claiming hardware verification.

## Persistence and API

Migrations 279–282 add `circuit_creation` and three separate concurrent indexes.
There are no new foreign keys. Workspace deletion and member removal explicitly
clean up circuit creations in the existing application cleanup transactions.

- `GET /api/circuit/catalog`: catalogue and optional AI availability.
- `POST /api/circuit/creations`: `client_request_id`, `kit_id`, `catalog_version`,
  `inventory`, `locale`, and exactly one of `project_id` or `prompt`.
- `GET /api/circuit/creations`: last 50 creations for this actor.
- `GET /api/circuit/creations/{id}`: saved document and progress.
- `PUT /api/circuit/creations/{id}/progress`: `current_step`, `observation`,
  `expected_revision`.

All creations are scoped by workspace, parent user, and child actor (or
`parent`). A repeated request ID returns the original document; a different
payload with that ID returns 409. Insufficient inventory returns 422 and creates
no document. The saved snapshot freezes inventory, catalogue version, source,
layout, instructions, netlist and report. Inventory edits in the project picker
are inputs to this snapshot, not a separate persistent inventory collection.
The content hash is SHA-256 of Go's JSON encoding of `Document` with an empty
`content_hash` field. Compilation detaches nested data from caller-owned maps.

Progress updates use revision comparison, return 409 on concurrent changes, and
leave the document untouched. A family observation is accepted only at the last
step. The UI awaits persistence before navigation or advancement. API schemas
reject malformed or contradictory build documents with a retryable error.

## Independent optimization

1. Tune intent matching in `planner.go`, retaining the bounded output contract.
   Test supported ideas and unsupported additions such as Bluetooth, timers,
   automatic sensors, remote control, and recording against the chosen model.
2. Add a documented project by defining the functional netlist independently
   from the layout, then testing missing parts, shorts, polarity, collisions,
   and step order. Bump the catalogue version for every material change.
3. Improve snap shapes, component identification, zoom, and step instructions
   in the shared workbench without changing saved electrical semantics.
4. Add another commercial kit through its own versioned catalogue and project
   evidence. Confirm dimensions, port polarity, battery arrangement, module
   internals and substitutions; do not alias similar-looking cross-brand parts.

Local checks:

```sh
# From server/; handler tests require a migrated, disposable DATABASE_URL.
go test ./internal/circuit
go test ./internal/handler -run '^TestCircuit' -count=1
go test ./internal/middleware -run '^TestChildCapabilities' -count=1

# From repository root.
pnpm --filter @chimii/core exec vitest run circuit/schemas.test.ts
pnpm --filter @chimii/views exec vitest run circuit/circuit-page.test.tsx
pnpm typecheck
pnpm exec playwright test e2e/circuit.spec.ts
```

The browser test uses `TestApiClient`, a disposable account/workspace, real API
persistence, cookie authentication, a child session, and workspace cleanup. Set
the local API, frontend and database environment variables explicitly. Model
unit tests use a fake text generator; they do not establish a live provider's
intent-classification accuracy.
