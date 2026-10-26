# Build planning and compilation

Build Studio uses the existing PostgreSQL job queue, an LLM planner, versioned construction scenes, target-shape designs or certified module recipes, and a catalog-backed compiler. New ideas use version-4 scenes with compilation, bounded repair and a separate model review. Legacy parameter edits compile without a model call. The planner can return `ready`, `clarify`, `reply`, or `unsupported`; an absent named module does not make a static subject unsupported.

`build_session` retains its existing status values. `phase` distinguishes planning from compilation; `recipe` stores a draft or complete recipe, and `revision` identifies an answer round. Migration 283 is required before starting the new server. Historical creations remain readable because they contain their finished plan; queued sessions are planned by the new planner.

## Deployment boundary

Do not run old and new backend Build workers concurrently. Old workers do not understand the new recipe checkpoint or clarification lifecycle and can still claim the shared queue. Stop/drain old backend workers, apply migration 283, then start the new backend. Queued sessions and expired leases are resumed by the new worker. This restriction concerns backend workers; the response retains label-based question options for installed clients. Do not roll back migration 283 while new sessions are active; stop workers and resolve those sessions first.

## Execution and recovery

- Requests enqueue work without reading, locking or freezing inventory. The worker reads current availability in a short, read-only transaction before planning/compilation. Version-4 scenes use unlimited quantities for design and report shortages against configured inventory; shortages never block generation. Version-2/3 API recipes retain quantity constraints for one simultaneous model; creating another model never reserves or consumes parts. New session/creation `inventory_snapshot` columns contain `{}` for the existing database contract, and new plans omit the optional legacy `inventory` field. Historical snapshots remain readable.
- Each clarification releases the worker and completes its current job execution. Answer submission checks the question ID/revision, merges history, increments the revision, and reactivates the same job with a fresh retry budget. At most two clarification rounds are allowed.
- Legacy clients may send displayed option labels; new clients send stable choice IDs. Free text is supported. Retransmitting the same answer is idempotent; a changed stale answer is rejected.
- A complete recipe is checkpointed before compiling. Legacy compilation retries reuse it without requesting another model response. Scene repair reservations and revisions are checkpointed under the same lease fence. Two repair attempts include malformed patch corrections; retries cannot reset that budget. An accepted review is reused only for the same review policy, compiled content and source idea, subject, summary, requirements and constraints. Obsolete approvals are removed and checkpointed before requesting a new review; they are not sent back as evidence. Metadata updates do not mutate the caller's previous recipe. A changed module-library version fails explicitly rather than silently compiling a saved recipe with different geometry.
- Worker writes lock session before job and verify the live lease. Completion, pause, failure and retry updates are transactional. Cancel invalidates the job lease before marking the session failed with `BUILD_CANCELLED`.
- Module compilation tries only declared equivalent attachment ports, at most four alternatives. Shape compilation uses bounded tiling search and local seam retiling, preserving target cells, holes and feature colors. Its context timeout is eight seconds, inside the existing worker execution/lease budget. Budget exhaustion returns `BUILD_SEARCH_LIMIT`, not an impossibility claim.

## Construction scenes

`scene.go` defines recipe v4 / `SceneSpec` v1, with architecture, vehicle, spacecraft, creature, figure and landscape families. Families describe intent; they do not select fixed subject templates. The model chooses scale, palette and a supported pose unless the source specifies them. Construction techniques include slabs, walls, hollow frames, real face openings, stairs, stepped roofs/tapers, elliptical extrusions, local shapes, colored patches and a catalog-backed four-wheel chassis.

`construction-scene-v2` adds generic supported platforms and complete storeys. Platforms contain 2x2 piers at six-stud pitch, transverse lower beams and a full upper deck. Storeys combine perimeter walls, interior piers and that ceiling; face openings preserve piers/lintels, and `ceiling_openings: [{x,z,width,depth}]` cuts both ceiling layers for stair access. Transverse trimmers support the opening edges without filling them. Full bearing on parent surfaces is required for their typed top attachments. These are deterministic target volumes, still subject to catalog tiling and all final mechanics checks, not pre-approved physical assemblies. Thin paint may retain a parent's window/door holes through matching openings instead of coloring a solid rectangle across a void. No extra service or executable model code is introduced.

New generated constructions require typed relative attachments. Normal parts attach to ground or a parent's top; several top parents must have equal height. Parents belong to the same object. Side anchors are only for embedded colored patches, which replace color inside existing volume without inventing side connectors. Independent objects must each be grounded and connected. Actual part connectors, global collisions and earlier-step support still determine acceptance.

Window requirements need actual openings in attributed walls/frames/storeys; a color patch is insufficient. Stairs require the stairs technique; wheels require actual wheelbases, and explicitly four wheels require one wheelbase. Explicit color words in attributed requirements are checked against declared construction colors. These checks cover a bounded set of words and techniques, not unrestricted semantic understanding.

`FitGeneratedScene` adjusts generated dimensions to minimum construction rules for piers/lintels, roof/taper tiers, stairs, platform/storey ranges and wheelbase geometry. Out-of-range input is rejected rather than clipped into the budget. It preserves every opening, feature color and construction. This is only a generation/repair step; API input and authoritative compilation remain strict. Frame openings may declare `face: front|back|left|right`, with `x` measured along that wall. Window pairs need a two-stud pier and each opening needs a three-plate lintel. Face paint stays exposed on the requested parent face; distinct same-color marks on one parent retain a one-cell gap instead of merging or being buried. Explicit paired limbs require at least two non-paint constructions, with duplicate requirement attributions rejected. Rectangular tiling excludes catalog entries whose actual X/Z bounds or bottom height disagree with occupancy/connector geometry.

Scenes are bounded to 8 objects, 64 constructions, 1200 total parts, 800 parts per object and 32768 target cells. A compilation shares a 96000-node search budget and 20-second deadline across all objects and attachment resolution. The worker deadline is 540 seconds; its lease is 570 seconds. Planning is bounded to 120 seconds with one 90-second format correction; legacy text review uses 45 seconds, image review and repair 90 seconds, all inside the worker deadline. These are limits, not expected latency.

Build planning and geometry repair use explicit task reasoning (`CHIMII_BUILD_PLANNER_EFFORT`, default `low`); visual review uses `CHIMII_BUILD_REVIEW_EFFORT` (default `low`). The requested thinking token budget is provider-dependent; DeepSeek ignores that field. The HTTP context, compiler budgets and persisted two-repair limit remain authoritative. Utility-title calls retain their existing behavior. Configuration changes require a server restart. The two-case real-provider evaluation timed out at high effort; low remains the default pending broader quality/latency evaluation, and is not a quality guarantee.

Canonical delivery includes MPD, real placements/connections/steps, scene document/hash, validation evidence and an optional frozen shortage list. The UI shows design descriptions, dimensions calculated from catalog occupancy, review status, current inventory comparison, BOM CSV, saved building progress and instanced GLB rendering for large models. Complete previews use actual part colors; the building workflow highlights new parts. Authenticated catalog GLB loads use the shared API client's origin, authorization and workspace context.

New scenes use `SceneSpec.version=2` within recipe v4. Each construction declares a role, and subject constructions must form an explicit interface graph that cannot rely on a display support as an intermediary. Final connector edges and real bricks crossing both target volumes establish interface evidence; merely sharing a pedestal does not. Historical scene v1 remains readable and compilable.

Planner and repair inputs include `catalog_profile`: eligible part counts, longest rectangular brick and the necessary long-beam condition for full platform/storey pier grids. The ten-part fallback does not advertise the long beams needed for large open floors. Asset-only entries do not increase this capability.

The scene compiler tiles horizontal support frontiers and freezes verified lower parts. A bounded, per-generation prefix cache can reuse unchanged lower geometry during repairs; catalog or lower-target changes invalidate it. All stages share the existing global node, part and deadline limits.

Scene v2 review receives six PNGs rendered from the exact catalog-revision GLBs (front/back/left/right/top/isometric), with real material colors, placement transforms and depth buffering. Missing/unsupported assets or an image-incompatible provider fail explicitly. Approval is bound to the source requirements, compiled content and renderer policy; image hashes survive the review checkpoint. Historical v1 uses legacy occupancy diagnostics. No review performs physical testing. Geometry acceptance and model review do not establish recognizability, real part/color availability, clutch strength or hands-on assembly. See [the reference comparison and implementation record](../../../tasks/one-sentence-brick-generation.md) for actual provider results and remaining work.

`SceneInspection` also supplies compiled occupancy dimensions, counts of parts assigned to each construction, direct part connections between those groups and the remaining component count after excluding a group's assigned parts. A tile can span construction boundaries; ownership is attribution, not a detachable assembly contract. These observations expose pedestal-only limbs or wing pieces to review without inventing connectors or imposing a universal visual-quality score. After structural repair the UI uses the current model and dimensions instead of potentially stale original prose.

A separate `reference-vertical-equilibrium-v1` estimate supports at most 32 upright, mechanically certified standard bricks with sourced masses, up to 256 contact samples, gravity-only force and two moment balances. Ground reactions cannot pull down. Per-stud clutch (0.98 N in the scene estimate) and the contact envelope are explicit uncalibrated assumptions. The bounded feasibility solver reports `feasible_in_model`, `infeasible_in_model`, or `not_evaluated`; unknown parts/masses, plates, wheels, cancellation and size limits never count as a pass. This estimate is advisory and does not gate geometric buildability or certify physical strength. Shear, torsion, wear, dynamics, insertion and physical testing remain outside this model. See [the current implementation plan](../../../tasks/brick-generation-technical-plan-2026-10-22.md).

## Extending supported shapes

`design.go` defines `DesignSpec` v1: ordered box, ellipse and polygon extrusions, additive/subtractive operations and translated repetitions. Horizontal coordinates are studs; vertical coordinates are plates. The target is bounded to 48 nodes and 8192 cells. `shape-solver.go` tiles the target with reviewed solid stud/tube catalog parts; `shape-repair.go` retiles disconnected seams without filling holes or changing the silhouette. All candidates pass the existing authoritative validator. New shape recipes use `AssemblyRecipe` v3; certified module recipes retain v2. These are two construction providers, not two validators.

`modules.go` remains the registry for certified module geometry, including special wheel assemblies. Module capability descriptions are catalog-filtered, not inventory-filtered; missing reusable parts must not be mislabeled as a missing shape capability. Neither provider accepts invented part IDs or physical brick scaling.

`shape-layout-v2` adapts BrickGPT's cumulative gap and component priorities plus
critical-neighborhood retiling. The first search attempt stays inexpensive;
with configured inventory, later deterministic restarts score continuous seams and contacts on both faces,
and may retile up to eight neighboring parts after pair repair fails. The same
24,000-node budget, eight-second deadline, target cells, feature identity,
colors, exact counts and reusable inventory apply. Upper contacts never replace
earlier-step support. Failed local searches do not modify their input. Unconfigured inventory retains
all six original orientation searches. No Python
or model dependency is added to the application. See
[the integration record](../../../tasks/brickgpt-integration.md)
and [source attribution](THIRD_PARTY_NOTICES.md).

`BuildPlan.document` holds the target design, its hash, solver evidence and source creation/hash. Placements, connections and instructions remain canonical in the surrounding plan. `POST /api/build/sessions` accepts optional `design`, `source_creation_id` and `expected_content_hash`. Source lookup checks workspace, creator and child scope; a mismatched source hash returns 409. Each edit creates a new immutable creation with fresh progress. Existing creations and progress are never overwritten. Shared `design-commands.ts` provides shape commands and undo/redo; the server compiles and validates every published revision.

Every final composition passes the existing connector, collision and step-stability checks; version-2/3 recipes also enforce configured inventory. Module certification alone never establishes that a composition is safe. Constraint checks cover exact colors, no wheels, exact part count, and required module kinds. Natural-language interpretation and visual recognizability still depend on the model; passing geometry checks does not prove semantic fidelity or real-world safety.

The module registry has 10 reviewed constructions. The shape provider handles new static silhouettes, including clocks, rings and extruded polygon outlines, without adding subject-specific compiler templates. It does not implement articulated mechanisms, a working clock, unrestricted part orientations, mesh reconstruction, custom manufactured parts or physical validation. Only the sampled target volume is proven preserved; natural-language fidelity is not established by the geometry checks. See `tasks/technical-design-freeform-build.md` for the staged architecture and remaining scope.

## Verification

Offline comparison against BrickGPT is available through `make build-eval`
and `scripts/brickgpt-eval/`. It uses this compiler/validator without changing
production state. Use `-corpus seams` for the additional 80 fixed inventory
cases and `make build-brickgpt-check` to verify a local `server/BrickGPT` source
snapshot without loading its dependencies. See [the experiment guide](../../../scripts/brickgpt-eval/README.md)
and [the first-batch record](../../../tasks/build-eval-first-batch.md) for
dataset, baseplate, license, model-access and physical-verification boundaries.

Run `go test ./internal/build` and the scoped core/views tests. `TestBuildSceneLiveGeneration` is opt-in via `CHIMII_SCENE_LIVE=1`; set explicit model configuration and output directory. `TestBuildSceneCatalogFixture` compiles an explicit saved scene into a disposable synchronized catalog. `e2e/build-scene.spec.ts` requires `CHIMII_SCENE_FIXTURE_JSON` and verifies persisted rendering/BOM/progress, separately from model quality. Handler tests have a global database fixture: set `DATABASE_URL` to a fully migrated disposable database, and also set `CHIMII_BUILD_TEST_DATABASE_URL` for the isolated Build schemas. Run handler cases matching `Test(BuildWorkerDB|BuildAnswers|ParseBuild|ToBuild)` with `-race -count=1 -v`; check that tests actually execute rather than the global fixture skipping them.

`e2e/build.spec.ts` and `e2e/build-shapes.spec.ts` use real HTTP, queue, database and UI with a deterministic local model fixture. Run `node scripts/build-e2e-llm-fixture.mjs` (loopback port 55441), point a disposable backend's `CHIMII_LLM_BASE_URL` to it, and run Playwright with `CHIMII_BUILD_E2E_STUB=1`. `CHIMII_BUILD_WEBGL_TEST=1` selects a fresh Chrome browser (Metal on macOS) and requires the actual GLB renderer. Use only a disposable database and test accounts. This verifies flow, not real-provider model quality.

Local microbenchmarks on Apple M4 measured approximately 22 microseconds for a six-part model compilation and 7 microseconds for the initial registry capability filter. These exclude model latency, queueing, network and browser rendering; they are not end-to-end latency guarantees.

## Building progress and list payloads

Migration 289 adds `current_step` (0 means not started), `completed_at`, and
`progress_revision` to the creation row. Recipe, plan, validation and inventory
snapshots remain unchanged. The existing workspace/member locking protocol
protects progress writes from concurrent workspace deletion.

- `GET /api/build/creations?view=summary`: latest 60 summaries, without recipe,
  placements or MPD. The original full list remains available for installed clients.
- `GET /api/build/creations/{id}/progress`: a small progress response.
- `PUT /api/build/creations/{id}/progress`: `current_step` (1-based),
  `expected_revision`, optional `completed`. Stale revisions return 409;
  completion requires the final step and an explicit user action.

The UI separates preview from building. Preview controls never save; a saved
step is restored from the creation detail page. A completed creation can be
reviewed without clearing its completion timestamp. Parent and child access
follow the existing creation visibility rules.
