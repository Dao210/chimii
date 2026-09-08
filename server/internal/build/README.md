# Build planning and compilation

Build Studio uses the existing PostgreSQL job queue, a single LLM planner, versioned target-shape designs or certified module recipes, and a catalog-backed compiler. Clear supported ideas normally use one model call. Parameter edits compile without a model call. The planner can return `ready`, `clarify`, or `unsupported`; an absent named module does not make a static subject unsupported.

`build_session` retains its existing status values. `phase` distinguishes planning from compilation; `recipe` stores a draft or complete recipe, and `revision` identifies an answer round. Migration 283 is required before starting the new server. Historical creations remain readable because they contain their finished plan; queued sessions are planned by the new planner.

## Deployment boundary

Do not run old and new backend Build workers concurrently. Old workers do not understand the new recipe checkpoint or clarification lifecycle and can still claim the shared queue. Stop/drain old backend workers, apply migration 283, then start the new backend. Queued sessions and expired leases are resumed by the new worker. This restriction concerns backend workers; the response retains label-based question options for installed clients. Do not roll back migration 283 while new sessions are active; stop workers and resolve those sessions first.

## Execution and recovery

- Requests enqueue work without reading, locking or freezing inventory. The worker reads current availability in a short, read-only transaction before planning/compilation. Quantities constrain one simultaneous model only; creating another model never reserves or consumes parts. New session/creation `inventory_snapshot` columns contain `{}` for the existing database contract, and new plans omit the optional legacy `inventory` field. Historical snapshots remain readable.
- Each clarification releases the worker and completes its current job execution. Answer submission checks the question ID/revision, merges history, increments the revision, and reactivates the same job with a fresh retry budget. At most two clarification rounds are allowed.
- Legacy clients may send displayed option labels; new clients send stable choice IDs. Free text is supported. Retransmitting the same answer is idempotent; a changed stale answer is rejected.
- A complete recipe is checkpointed before compiling. Compilation retries reuse it without requesting another model response. A changed module-library version fails explicitly rather than silently compiling a saved recipe with different geometry.
- Worker writes lock session before job and verify the live lease. Completion, pause, failure and retry updates are transactional. Cancel invalidates the job lease before marking the session failed with `BUILD_CANCELLED`.
- Module compilation tries only declared equivalent attachment ports, at most four alternatives. Shape compilation uses bounded tiling search and local seam retiling, preserving target cells, holes and feature colors. Its context timeout is eight seconds, inside the existing worker execution/lease budget. Budget exhaustion returns `BUILD_SEARCH_LIMIT`, not an impossibility claim.

## Extending supported shapes

`design.go` defines `DesignSpec` v1: ordered box, ellipse and polygon extrusions, additive/subtractive operations and translated repetitions. Horizontal coordinates are studs; vertical coordinates are plates. The target is bounded to 48 nodes and 8192 cells. `shape-solver.go` tiles the target with reviewed solid stud/tube catalog parts; `shape-repair.go` retiles disconnected seams without filling holes or changing the silhouette. All candidates pass the existing authoritative validator. New shape recipes use `AssemblyRecipe` v3; certified module recipes retain v2. These are two construction providers, not two validators.

`modules.go` remains the registry for certified module geometry, including special wheel assemblies. Module capability descriptions are catalog-filtered, not inventory-filtered; missing reusable parts must not be mislabeled as a missing shape capability. Neither provider accepts invented part IDs or physical brick scaling.

`BuildPlan.document` holds the target design, its hash, solver evidence and source creation/hash. Placements, connections and instructions remain canonical in the surrounding plan. `POST /api/build/sessions` accepts optional `design`, `source_creation_id` and `expected_content_hash`. Source lookup checks workspace, creator and child scope; a mismatched source hash returns 409. Each edit creates a new immutable creation with fresh progress. Existing creations and progress are never overwritten. Shared `design-commands.ts` provides shape commands and undo/redo; the server compiles and validates every published revision.

Every final composition passes the existing inventory, connector, collision and step-stability checks. Module certification alone never establishes that a composition is safe. Constraint checks cover exact colors, no wheels, exact part count, and required module kinds. Natural-language interpretation and visual recognizability still depend on the model; passing geometry checks does not prove semantic fidelity or real-world safety.

The module registry has 10 reviewed constructions. The shape provider handles new static silhouettes, including clocks, rings and extruded polygon outlines, without adding subject-specific compiler templates. It does not implement articulated mechanisms, a working clock, unrestricted part orientations, mesh reconstruction, custom manufactured parts or physical validation. Only the sampled target volume is proven preserved; natural-language fidelity is not established by the geometry checks. See `tasks/technical-design-freeform-build.md` for the staged architecture and remaining scope.

## Verification

Offline comparison against BrickGPT is available through `make build-eval`
and `scripts/brickgpt-eval/`. It uses this compiler/validator without changing
production generation. See [the experiment guide](../../../scripts/brickgpt-eval/README.md)
and [the first-batch record](../../../tasks/build-eval-first-batch.md) for
dataset, baseplate, license, model-access and physical-verification boundaries.

Run `go test ./internal/build` and the scoped core/views tests. Handler tests have a global database fixture: set `DATABASE_URL` to a fully migrated disposable database, and also set `CHIMII_BUILD_TEST_DATABASE_URL` for the isolated Build schemas. Run handler cases matching `Test(BuildWorkerDB|BuildAnswers|ParseBuild|ToBuild)` with `-race -count=1 -v`; check that tests actually execute rather than the global fixture skipping them.

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
