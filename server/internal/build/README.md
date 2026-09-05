# Build planning and compilation

Build Studio uses the existing PostgreSQL job queue, a single LLM planner, a versioned `AssemblyRecipe`, and the deterministic compiler. Clear supported ideas normally use one model call. The planner can return `ready`, `clarify`, or `unsupported`; unsupported subjects must not be renamed into a supported template.

`build_session` retains its existing status values. `phase` distinguishes planning from compilation; `recipe` stores a draft or complete recipe, and `revision` identifies an answer round. Migration 283 is required before starting the new server. Historical creations remain readable because they contain their finished plan; queued sessions are planned by the new planner.

## Deployment boundary

Do not run old and new backend Build workers concurrently. Old workers do not understand the new recipe checkpoint or clarification lifecycle and can still claim the shared queue. Stop/drain old backend workers, apply migration 283, then start the new backend. Queued sessions and expired leases are resumed by the new worker. This restriction concerns backend workers; the response retains label-based question options for installed clients. Do not roll back migration 283 while new sessions are active; stop workers and resolve those sessions first.

## Execution and recovery

- The request freezes inventory and enqueues work; no model call runs inside a database transaction.
- Each clarification releases the worker and completes its current job execution. Answer submission checks the question ID/revision, merges history, increments the revision, and reactivates the same job with a fresh retry budget. At most two clarification rounds are allowed.
- Legacy clients may send displayed option labels; new clients send stable choice IDs. Free text is supported. Retransmitting the same answer is idempotent; a changed stale answer is rejected.
- A complete recipe is checkpointed before compiling. Compilation retries reuse it without requesting another model response. A changed module-library version fails explicitly rather than silently compiling a saved recipe with different geometry.
- Worker writes lock session before job and verify the live lease. Completion, pause, failure and retry updates are transactional. Cancel invalidates the job lease before marking the session failed with `BUILD_CANCELLED`.
- The compiler retries only declared equivalent attachment ports, at most four alternative candidates. It does not change module kinds, required features, exact colors or part counts.

## Extending supported shapes

`internal/build/modules.go` is the single registry for module geometry and planner capabilities. Add a module's reviewed local placements, accepted attachment ports, description and tests. Instances choose module IDs, ports and colors; the model never writes part coordinates. Capability filtering checks catalog certification and module BOM quantities without doing a physics solve for every candidate.

Every final composition passes the existing inventory, connector, collision and step-stability checks. Module certification alone never establishes that a composition is safe. Constraint checks cover exact colors, no wheels, exact part count, and required module kinds. Natural-language interpretation and visual recognizability still depend on the model; passing geometry checks does not prove semantic fidelity or real-world safety.

The initial registry contains 10 modules: body, head, eyes, short ears, long ears, wing, tail, roof, robot base and rolling base. It supports simple silhouettes and limited attachments, not arbitrary architecture or articulated mechanisms. Static wings and tails do not imply real flight or motion. Difficulty-based size expansion is not advertised because it has no certified implementation.

## Verification

Run `go test ./internal/build` and the scoped core/views tests. Handler tests have a global database fixture: set `DATABASE_URL` to a fully migrated disposable database, and also set `CHIMII_BUILD_TEST_DATABASE_URL` for the isolated Build schemas. Run handler cases matching `Test(BuildWorkerDB|BuildAnswers|ParseBuild|ToBuild)` with `-race -count=1 -v`; check that tests actually execute rather than the global fixture skipping them.

`e2e/build.spec.ts` uses real HTTP, queue, database and UI with a deterministic local model fixture. Run `node scripts/build-e2e-llm-fixture.mjs` (loopback port 55441), point a disposable backend's `CHIMII_LLM_BASE_URL` to it, and run Playwright with `CHIMII_BUILD_E2E_STUB=1`. Use only a disposable database and test accounts. This verifies flow, not real-provider model quality.

Local microbenchmarks on Apple M4 measured approximately 22 microseconds for a six-part model compilation and 7 microseconds for the initial registry capability filter. These exclude model latency, queueing, network and browser rendering; they are not end-to-end latency guarantees.
