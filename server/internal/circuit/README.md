# Electronic construction

This module turns an idea or a selected project into a saved, step-by-step
assembly document for children using exact commercial kits. The catalogues target
**Snap Circuits SC-500** (switched lamp, alarm sound, tabletop FM radio) and
**DFRobot BOSON EDU0080-EN** (five non-programming projects). Quantities are
subsets of each kit, not their full BOMs.
Parts from other brands or kit revisions are not assumed interchangeable.

## Boundaries

```text
Idea / selected project + kit version + saved parts-box revision
              |
     Plan: bounded model intent (optional)
              |
     Compile: reference project + component catalogue
              |
     Validate: inventory + kit-specific reference connections
         Snap: layers / bodies / snaps / netlist / shorts
         BOSON: keyed ports / directions / cables / expected pairs
              |
     Document v1 (Snap) or v2 (BOSON) -> immutable JSONB snapshot
              |
     Shared SVG workbench -> separate progress / family trial history
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

`catalog.json` (Snap) and `boson.json` are server-owned embedded sources of truth. Each Snap component
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

- `GET /api/circuit/catalog?kit_id=...`: catalogue and optional AI availability.
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
layout, instructions, netlist and report. A family has one persistent `circuit_inventory` row per kit. A parent saves
complete, explicitly checked quantities; children inherit read access. A fresh
box contains zeros and is unconfirmed. Updates compare the expected revision.
BOSON creation requires the saved inventory revision; existing Snap API clients
can still supply quantities directly. The current UI always uses saved boxes.
The handler checks quantities and revision before planning and again under a
transaction lock before saving. The snapshot includes the inventory revision
when supplied. Idempotent replay returns the original snapshot even after edits.
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
pnpm --filter @chimii/core exec vitest run circuit
pnpm --filter @chimii/views exec vitest run circuit/circuit-page.test.tsx
pnpm typecheck
pnpm exec playwright test e2e/circuit.spec.ts
```

The browser test uses `TestApiClient`, a disposable account/workspace, real API
persistence, cookie authentication, a child session, and workspace cleanup. Set
the local API, frontend and database environment variables explicitly. Model
unit tests use a fake text generator; they do not establish a live provider's
intent-classification accuracy.

### Live conversation acceptance (2026-09-06)

After explicit user authorization, `TestBuildConversationLivePlanner` passed all
five cases against the configured external gateway, requesting `deepseek-v4-pro`.
The missing local `CHIMII_LLM_DEFAULT_MODEL` initially selected `sonnet-4-6`, which
the gateway rejected with HTTP 400. Setting the local model to a name listed in
that response resolved the configuration mismatch; no planner prompt was changed.

| Case | Observed result | Duration |
| --- | --- | --- |
| Radio routing | `circuit` | 2.63 s |
| Rabbit enclosure with functional radio | `clarify`, asks which independent product to make first | 5.19 s |
| BOSON button light | `ready`, `boson-button-light` | 7.13 s |
| BOSON FM radio | `unsupported`, explains absent receiver capability | 3.68 s |
| Lower brick gate, keep doorway | Wall height 12 to 10; doorway and other shape properties unchanged | 14.71 s |

The run took 33.35 seconds. The live test logs parsed decisions and sanitized
errors, and requires `CHIMII_CONVERSATION_LIVE=1`, explicit utility model
configuration and a migrated disposable `DATABASE_URL`. Normal tests skip it.
Run from `server/` with those variables supplied:
`go test -v ./internal/handler -run '^TestBuildConversationLivePlanner$' -count=1`.
This is one live planning smoke run, including shape-schema validation; it does
not establish repeated-run accuracy, complete artifact generation, deployment,
or physical assembly verification.


## Commercial module adapter

BOSON records whole keyed cables between module sockets. It does not collapse a
three-wire cable into a single electrical conductor. `modules.go` checks port
identity, connector family, direction, single occupancy, cable ownership, full
connectivity and an independently supplied reference connection list. A generic
connector name is insufficient to accept a new module: the exact project and
part IDs must also match. Diagrams use logical coordinates, not mounting scale,
and do not establish cable reach, current budgets or physical safety.

The five BOSON references are button light, inverted button light, dimmer light,
sound fan, and motion-and-sound AND fan. The reviewed kit quick guide (pages 10,
12) and non-programming cards (pages 2, 4, 8) supply the evidence. The long kit
tutorial contains component differences; it is not a blanket compatibility
source. All documents retain `physical_verification: not_tested`.

Additional APIs:

- `GET /api/circuit/kits`: supported exact kits and official purchase channels.
- `GET /api/circuit/inventory/{kitID}`: this parent's family box and edit ability.
- `PUT /api/circuit/inventory/{kitID}`: parent only; `catalog_version`, complete
  `quantities`, `expected_revision`. Stale updates return 409.
- `POST /api/circuit/creations`: optional `inventory_revision`, required for BOSON.
- `GET /api/circuit/creations/{id}/trials`: last 50 reports for this actor's creation.
- `POST /api/circuit/creations/{id}/trials`: `client_request_id`, `hardware_label`,
  `result` (`worked` or `needs_help`), `notes`, `adult_checked: true`. A new report
  requires persisted final-step progress; identical retries replay the old one.

Trial reports append independently of the design. Their server-assigned document
hash binds them to the original snapshot. `family_report` never upgrades platform
hardware evidence. Scope is workspace + parent + child/parent actor. Workspace
and membership removal explicitly clean up both new FK-free tables in the
existing transactional cleanup path. Migrations 284–288 create tables and
single-statement concurrent indexes.

See [procurement and physical acceptance worksheet (Chinese)](HARDWARE-ACCEPTANCE.zh-CN.md)
for exact parts, reviewed sources, source hashes, acquisition limitations and
physical test steps. No kit was purchased or physically assembled in this run.


## Shared creation conversations

The `/circuit` entry now uses the same conversational container and durable Build
queue as `/build`. Circuit documents, family inventories, progress and trial
reports remain in their existing independent tables. The worker dispatches before
loading any brick catalogue. `compileCircuitDocument` and `saveCircuitDocument`
are shared by the legacy circuit endpoint and queued conversation generation.

A model may select a reviewed project, ask a necessary question, explain a listed
project, or report an unsupported request. Explanations use reviewed catalogue
text; no model-authored wiring is rendered as an assembly guide. Fixed projects
still work without a model. New versions do not inherit old progress or family
trial evidence. See `docs/plans/2026-09-05-conversation-build-circuit.md` at the
repository root for the API, migration and verification record.
