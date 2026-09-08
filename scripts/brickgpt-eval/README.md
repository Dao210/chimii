# BrickGPT comparison experiments

This is an offline evaluation tool, not a production generator or a physical
certification system. It reuses Chimii's existing compiler and validator and
calls the pinned upstream force-analysis implementation in a bounded process.
No database, workspace, inventory reservation, application endpoint or creation
record is involved. The normal test target does not download models or invoke
installed agent CLIs.

## Current evidence

See [the first-batch acceptance record](../../tasks/build-eval-first-batch.md).
Keep three evidence types separate: deterministic compiler contracts, an
upstream computational force check, and actual physical assembly. The last
category is not established by these tools.

## Deterministic baseline

From the repository root:

```sh
mkdir -p output/build-eval
make build-eval BUILD_EVAL_ARGS='-output ../output/build-eval/baseline.json'
make build-eval-test
```

`BUILD_EVAL_ARGS` are passed to a command running in `server/`. A writable Go
cache can be selected with `GOCACHE=/tmp/chimii-go-cache` when the system cache
is restricted. `-family tower,thin-wall` selects a subset; unknown selections
fail instead of producing an empty successful run.

The corpus has 10 parameterized families with 10 cases each: towers, thin
walls, holes, bridges, exact counts, missing parts, exact colors, edits
preserving openings, cantilevers and invalid layouts. Sixty cases have a
specific acceptance/rejection expectation. Forty are capability observations;
their rejection is recorded but not asserted to be a regression. These are
hand-authored recipes and layout probes, not live text-planning evaluations.

The JSON report records input/output hashes, inventory inputs, source edits,
placements, validation failures, catalog/validator versions and runtime.
`planning_ms` is null because no planner ran. `compile_including_validation_ms`
includes compilation and its built-in validation; `validation_recheck_ms`
measures an additional independent invocation and must not be subtracted as a
precise breakdown or added to a user-facing latency claim. Compiler percentiles
include failed compile attempts and exclude layout-only probes.

All accepted designs are independently compared with their sampled target
cells, colors and exact count. Preserved edit shapes and input immutability are
checked. Geometry preservation does not establish that a model resembles its
natural-language prompt.

## Pinned upstream physics

```sh
git clone https://github.com/AvaLovelace1/BrickGPT.git /tmp/chimii-brickgpt
git -C /tmp/chimii-brickgpt checkout da01aab83f646a700e2270b66b4c00180523448b
python3.13 -m venv /tmp/chimii-brickgpt-venv
/tmp/chimii-brickgpt-venv/bin/python -m pip install -r scripts/brickgpt-eval/requirements.txt
/tmp/chimii-brickgpt-venv/bin/python scripts/brickgpt-eval/bridge.py compare \
  --upstream /tmp/chimii-brickgpt \
  --input output/build-eval/baseline.json \
  --output output/build-eval/physics.json --prefixes --timeout 15
```

`upstream-lock.json` fixes source-file SHA-256 values. Changed upstream code is
rejected before loading. The numerical dependencies are separate from the
optional model/Blender stack. NumPy 2.2.6 is used for the numerical module;
upstream's NumPy <2 constraint belongs to its combined Blender installation.

The supported intersection is axis-aligned, full-height rectangular bricks
from the upstream eight sizes. Plates, slopes, wheels, other connectors, mixed
plate elevations, collisions and oversized grids are reported as
`not_applicable`; they are never approximated by rectangular full-height bricks.
Translations preserve brick size, vertical origin and source sequence.

The boundary condition is **stud-attached baseplate**. Chimii's existing
validator additionally reasons about a free-standing support footprint and
requires one connected assembly. For example, separate objects attached to one
implicit baseplate can pass upstream while Chimii rejects their disconnection.
Agreement is therefore not a shared ground truth or a claim of physical safety.
The baseplate is not added to a real BOM by this reference experiment.

`--prefixes` checks every individual brick prefix in the recorded placement
order, not just completed layer groups. It does not verify hand clearance or
insertion paths. The entire case has a process deadline; each Gurobi solve also
has a bounded time limit and one thread. Non-optimal termination, unavailable
dependencies, size-limited licenses and timeouts remain distinct from a
calculated unstable structure. A report sets `complete` only after processing
all cases. Partial results remain available after interruption.

`maximum_upstream_score` is the upstream load/failure indicator (larger is
worse; values of 1 indicate failure), not the paper's inverted stability
percentage. We record solver status before upstream closes the model so its
all-ones error result cannot masquerade as a physical finding.

## Official dataset sample

Download the official test split from this fixed revision:

https://huggingface.co/datasets/AvaLovelace/StableText2Brick/resolve/fd4c7b12cf925325665b7d1117615a18f1296b7a/data/test-00000-of-00001.parquet

Expected SHA-256:
`45b1078801f2e1871c9444d186be089d77aa9354bc60a35ab3d8e82710e4ff3e`.
The importer verifies both the revision and file checksum against
`dataset-lock.json` before reading the pinned test split.

```sh
/tmp/chimii-brickgpt-venv/bin/python scripts/brickgpt-eval/bridge.py dataset \
  --input /tmp/stabletext2brick-test.parquet \
  --output output/build-eval/dataset-candidates.json \
  --revision fd4c7b12cf925325665b7d1117615a18f1296b7a --split test --count 100
make build-eval BUILD_EVAL_ARGS='-candidates ../output/build-eval/dataset-candidates.json -output ../output/build-eval/dataset-validation.json'
```

Sampling sorts by SHA-256 of object ID and takes one structure per object. It
does not filter by quality, part count or compatibility. The manifest contains
file/sample hashes, split, revision and part-dimension coverage. If building a
training/reference set later, exclude these object IDs, including other
structures of the same objects; this tool does not train or establish a
train/test-disjoint training pipeline.

The importer accepts the upstream text format only, not arbitrary LDraw or
model-supplied part metadata. It maps the complete candidate into the embedded
`StarterCatalog`, preserves brick ordering, maps one brick layer to three
plates, checks the 200-part/48-plate Chimii limits, then runs the existing
validator with the supplied inventory. Missing parts are not deleted and
floating bricks are not moved onto the ground. All failed candidates remain in
the report denominator. This starter-catalog result says nothing about the
currently deployed database catalog without a separate production audit.

## Optional local text generation

The public BrickGPT weights are a LoRA adapter. Supply an authorized local
`meta-llama/Llama-3.2-1B-Instruct` base model and the public
`AvaLovelace/BrickGPT` adapter/tokenizer directory. Install
`requirements-model.txt` in the isolated environment when these weights are
available. Versions match the pinned upstream model stack.

```sh
/tmp/chimii-brickgpt-venv/bin/python scripts/brickgpt-eval/generate.py \
  --upstream /tmp/chimii-brickgpt \
  --base-model /path/to/authorized/llama-base \
  --adapter /path/to/brickgpt-adapter \
  --prompt 'A small chair with four legs and a backrest.' \
  --seed 42 --max-bricks 100 --timeout 180 --device cpu \
  --output output/build-eval/generated-candidates.json
make build-eval BUILD_EVAL_ARGS='-candidates ../output/build-eval/generated-candidates.json -output ../output/build-eval/generated-validation.json'
```

The generator requires local weights, disables implicit authentication and
network loading, records weight hashes and seed, and reuses upstream rejection
and physics-aware rollback with bounded retries. It starts one model per CLI
invocation; this prototype does not promise production throughput. It records
model loading separately from generation/final checking. Failure writes an
empty candidate batch and a status artifact, invalidating stale output. Even
`candidate_generated` with exit code 0 is only a candidate: the upstream final
check may be false and Chimii can still reject it. No candidate is saved as a
product creation.

Missing base weights return `base_model_missing` without starting inference.
The current experiment has not established real text-generation quality or
latency. Contract tests with test-created outputs are identified as such and
are not model evidence.

## Licenses and boundaries

The upstream repository declares MIT for its main code, dataset and adapter;
retain its license if redistributing those files. Source is loaded from the
explicit checkout and is not vendored here. The Llama base model has its own
community license. Gurobi has separate license terms and the free package can
reject models exceeding its allowed size. No license is purchased by these
scripts. The existing product LDraw catalog retains its own notices.

- [Pinned BrickGPT license](https://github.com/AvaLovelace1/BrickGPT/blob/da01aab83f646a700e2270b66b4c00180523448b/LICENSE)
- [Llama base-model license](https://huggingface.co/meta-llama/Llama-3.2-1B-Instruct/blob/main/LICENSE.txt)
- [Official project and physical-assembly demonstrations](https://avalovelace1.github.io/BrickGPT/)
