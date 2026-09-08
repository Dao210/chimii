#!/usr/bin/env python3
"""Offline BrickGPT experiments. No credentials, model downloads, or API writes."""
import argparse
from collections import Counter
import contextlib
import hashlib
import importlib
import json
import math
import os
from pathlib import Path
import subprocess
import sys
import time

LOCK = Path(__file__).with_name("upstream-lock.json")
DATASET_LOCK = Path(__file__).with_name("dataset-lock.json")
BRICK_SIZES = {(1, 1), (1, 2), (1, 4), (1, 6), (1, 8), (2, 2), (2, 4), (2, 6)}


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()).hexdigest()


def read_json(path):
    if Path(path).stat().st_size > 32 * 1024 * 1024:
        raise ValueError("JSON input exceeds the offline evaluation limit")
    return json.loads(Path(path).read_text())


def write_json(path, value):
    data = json.dumps(value, ensure_ascii=False, indent=2, allow_nan=False) + "\n"
    Path(path).write_text(data)


def verify_upstream(root):
    root = Path(root).resolve()
    lock = read_json(LOCK)
    for relative, expected in lock["files"].items():
        if hashlib.sha256((root / relative).read_bytes()).hexdigest() != expected:
            raise ValueError("upstream source differs from pinned revision: " + relative)
    return lock


def verify_dataset(path, revision, split):
    lock = read_json(DATASET_LOCK)
    if revision != lock["revision"] or split not in lock["splits"]:
        raise ValueError("dataset revision or split is not pinned")
    if hashlib.sha256(Path(path).read_bytes()).hexdigest() != lock["splits"][split]["sha256"]:
        raise ValueError("dataset checksum does not match the official pinned split")


def as_bricks(placements, catalog):
    """Translate horizontal origin only; preserve height, order, and real sizes."""
    if not placements or len(placements) > 200:
        raise ValueError("unsupported_part_count")
    for p in placements:
        if any(type(p.get(k)) is not int for k in ("x", "y", "z", "rotation", "step")):
            raise ValueError("non_integer_placement")
    min_x = min(p["x"] for p in placements)
    min_z = min(p["z"] for p in placements)
    bricks, occupied = [], set()
    ids = set()
    for p in placements:
        if not p.get("id") or p["id"] in ids:
            raise ValueError("invalid_placement_id")
        ids.add(p["id"])
        if p["rotation"] % 90 or p["y"] < 0 or p["y"] % 3 or p["step"] <= 0:
            raise ValueError("unsupported_transform")
        part = catalog.get(p["part_id"])
        if (not part or part.get("geometry_profile") != "stud_tube_rect"
                or part.get("plates_y") != 3 or part.get("certification_level") != "certified"):
            raise ValueError("unsupported_part")
        h, w = part["studs_x"], part["studs_z"]
        if p["rotation"] % 180:
            h, w = w, h
        if tuple(sorted((h, w))) not in BRICK_SIZES:
            raise ValueError("unsupported_dimension")
        x, y, z = p["x"] - min_x, p["z"] - min_z, p["y"] // 3
        if x + h > 20 or y + w > 20 or z >= 20:
            raise ValueError("unsupported_grid")
        for a in range(x, x + h):
            for b in range(y, y + w):
                if (a, b, z) in occupied:
                    raise ValueError("collision")
                occupied.add((a, b, z))
        bricks.append({"h": h, "w": w, "x": x, "y": y, "z": z})
    return bricks


@contextlib.contextmanager
def quiet_native_stdout():
    # Gurobi can write license notices from native code. Keep stdout machine-readable.
    sys.stdout.flush()
    saved = os.dup(1)
    with open(os.devnull, "w") as sink:
        os.dup2(sink.fileno(), 1)
        try:
            with contextlib.redirect_stdout(sink):
                yield
        finally:
            os.dup2(saved, 1)
            os.close(saved)


def analyze_physics(root, payload):
    verify_upstream(root)
    sys.path.insert(0, str(Path(root).resolve() / "src"))
    import numpy as np
    import gurobipy as gp
    from brickgpt.data import Brick, BrickStructure, brick_library
    analysis = importlib.import_module("brickgpt.stability_analysis.stability_analysis")

    # Record the solver termination state before upstream closes the model.
    # Upstream returns an all-ones array for non-optimal termination, which
    # must not be counted as a demonstrated unstable structure.
    states = []
    class ModelProxy:
        def __init__(self, model, state):
            object.__setattr__(self, "model", model)
            object.__setattr__(self, "state", state)

        def __getattr__(self, key):
            return getattr(self.model, key)

        def __setattr__(self, key, value):
            setattr(self.model, key, value)

        def close(self):
            self.state["status"] = self.model.Status
            self.model.close()

    class SolverAPI:
        def __getattr__(self, key):
            return getattr(gp, key)

        def Model(self, name):
            state = {}
            model = gp.Model(name)
            model.setParam("OutputFlag", 0)
            model.setParam("TimeLimit", min(payload["timeout"], 10))
            model.setParam("Threads", 1)
            state["proxy"] = ModelProxy(model, state)
            states.append(state)
            return state["proxy"]

    analysis.gp = SolverAPI()
    bricks = [Brick(**b) for b in payload["bricks"]]
    ids = payload["placement_ids"]
    end_points = list(range(1, len(bricks) + 1)) if payload["prefixes"] else [len(bricks)]
    results = []
    try:
        for end in end_points:
            structure = BrickStructure(bricks[:end])
            if structure.has_collisions() or structure.has_out_of_bounds_bricks():
                return {"status": "invalid_input", "prefixes": results}
            if structure.has_floating_bricks():
                results.append({"count": end, "status": "fail", "reason": "floating_brick"})
                continue
            scores, variables, constraints, total, solve = analysis.stability_score(
                structure.to_json(), brick_library, analysis.StabilityConfig())
            state = states[-1]
            status = state.get("status")
            if status is None:
                status = state["proxy"].Status
                state["proxy"].close()
            if status != gp.GRB.OPTIMAL:
                return {"status": "non_optimal", "solver_status": status, "prefixes": results}
            loads = [float(np.max(scores[b.slice])) for b in bricks[:end]]
            if not all(math.isfinite(x) for x in loads):
                return {"status": "invalid_solver_output", "prefixes": results}
            results.append({"count": end, "status": "pass" if max(loads) < 1 else "fail",
                            "maximum_upstream_score": max(loads),
                            "unstable_placement_ids": [ids[i] for i, score in enumerate(loads) if score >= 1],
                            "variables": variables, "constraints": constraints,
                            "analysis_ms": total * 1000, "solve_ms": solve * 1000})
    finally:
        for state in states:
            if "status" not in state:
                state["proxy"].close()
    return {"status": "pass" if all(x["status"] == "pass" for x in results) else "fail", "prefixes": results,
            "gurobi_version": ".".join(map(str, gp.gurobi.version())), "numpy_version": np.__version__}


def physics_worker(args):
    payload = json.load(sys.stdin)
    try:
        with quiet_native_stdout():
            result = analyze_physics(args.upstream, payload)
    except ImportError as exc:
        result = {"status": "dependency_missing", "module": exc.name}
    except Exception as exc:
        # Gurobi's error code is diagnostic; never echo license/account details.
        code = getattr(exc, "errno", None)
        result = {"status": "solver_size_limit" if code == 10010 else "solver_error", "error_type": type(exc).__name__}
        if code is not None:
            result["error_code"] = code
    print(json.dumps(result, allow_nan=False))


def compare(args):
    lock = verify_upstream(args.upstream)
    report = read_json(args.input)
    if report.get("version") != 1:
        raise ValueError("unsupported input report")
    output = {"version": 1, "upstream_revision": lock["revision"], "input_sha256": hashlib.sha256(Path(args.input).read_bytes()).hexdigest(),
              "boundary_condition": "stud-attached-baseplate", "comparison": "different support conditions from Chimii free-standing validator",
              "scope": "offline static force reference; no physical or product certification", "prefixes_requested": args.prefixes,
              "complete": False, "expected_cases": len(report["cases"]), "cases": [], "summary": {}}
    for case in report["cases"]:
        start = time.perf_counter()
        result = {"id": case["id"], "input_hash": case["input_hash"], "chimii_accepted": case.get("validation", {}).get("buildable", False)}
        try:
            if not case.get("placements"):
                raise ValueError("no_candidate")
            bricks = as_bricks(case["placements"], report["catalog"])
            payload = {"bricks": bricks, "placement_ids": [p["id"] for p in case["placements"]], "prefixes": args.prefixes, "timeout": args.timeout}
            child = subprocess.run([sys.executable, __file__, "_physics", "--upstream", args.upstream], input=json.dumps(payload),
                                   text=True, capture_output=True, timeout=args.timeout, check=False)
            if child.returncode:
                result.update(status="worker_error", exit_code=child.returncode)
            else:
                try:
                    worker_result = json.loads(child.stdout)
                    if not isinstance(worker_result, dict) or worker_result.get("status") not in {
                        "pass", "fail", "non_optimal", "invalid_input", "invalid_solver_output",
                        "dependency_missing", "solver_size_limit", "solver_error"}:
                        raise ValueError("invalid worker result")
                    result.update(worker_result)
                except (TypeError, ValueError):
                    result.update(status="invalid_worker_output")
        except ValueError as exc:
            result.update(status="not_applicable", reason=str(exc))
        except subprocess.TimeoutExpired:
            result.update(status="timeout")
        result["elapsed_ms"] = (time.perf_counter() - start) * 1000
        output["cases"].append(result)
        status = result["status"]
        output["summary"][status] = output["summary"].get(status, 0) + 1
        write_json(args.output, output) # Keep completed results if interrupted.
    output["complete"] = True
    write_json(args.output, output)
    print(json.dumps(output["summary"], sort_keys=True))


def dataset(args):
    """Sample one structure per object from a local, pinned official Parquet file."""
    verify_dataset(args.input, args.revision, args.split)
    import pyarrow.parquet as pq
    rows = pq.read_table(args.input, columns=["structure_id", "object_id", "category_id", "captions", "bricks"]).to_pylist()
    # Stable hash ordering prevents category-order sampling. Keep explicit object
    # identity so a later train/reference split can exclude these test objects.
    rows.sort(key=lambda r: (hashlib.sha256(r["object_id"].encode()).hexdigest(), r["structure_id"]))
    seen, candidates = set(), []
    for row in rows:
        if row["object_id"] in seen:
            continue
        seen.add(row["object_id"])
        candidates.append({"id": row["structure_id"], "prompt": row["captions"][0], "bricks": row["bricks"], "color": 1,
                           "inventory": {"configured": False, "items": []},
                           "source": {"kind": "stabletext2brick", "revision": args.revision, "split": args.split,
                                      "object_id": row["object_id"], "structure_id": row["structure_id"]}})
        if len(candidates) == args.count:
            break
    write_json(args.output, {"version": 1, "candidates": candidates})
    dimensions = Counter()
    for candidate in candidates:
        per_object = set()
        for line in candidate["bricks"].strip().splitlines():
            a, b = sorted(map(int, line.split()[0].split("x")))
            per_object.add(f"{a}x{b}")
        dimensions.update(per_object)
    write_json(args.output + ".manifest.json", {"dataset": "AvaLovelace/StableText2Brick", "revision": args.revision, "split": args.split,
                                               "parquet_sha256": hashlib.sha256(Path(args.input).read_bytes()).hexdigest(),
                                               "selection": "sha256(object_id), one structure per object; no quality filtering", "sample_count": len(candidates),
                                               "objects_using_dimension": dict(sorted(dimensions.items())),
                                               "objects_over_200_parts": sum(len(c["bricks"].strip().splitlines()) > 200 for c in candidates),
                                               "sample_sha256": hashlib.sha256(Path(args.output).read_bytes()).hexdigest()})
    print(f"Selected {len(candidates)} distinct objects from {len(rows)} structures")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    p = commands.add_parser("compare", help="compare an offline Go report against pinned BrickGPT physics")
    p.add_argument("--upstream", required=True)
    p.add_argument("--input", required=True)
    p.add_argument("--output", required=True)
    p.add_argument("--timeout", type=float, default=15)
    p.add_argument("--prefixes", action="store_true", help="also check every ordered brick prefix; same per-case timeout")
    p.set_defaults(run=compare)
    p = commands.add_parser("_physics", help=argparse.SUPPRESS)
    p.add_argument("--upstream", required=True)
    p.set_defaults(run=physics_worker)
    p = commands.add_parser("dataset", help="convert an official local Parquet split to a reproducible candidate sample")
    p.add_argument("--input", required=True)
    p.add_argument("--output", required=True)
    p.add_argument("--revision", required=True)
    p.add_argument("--split", choices=["test"], required=True)
    p.add_argument("--count", type=int, default=100)
    p.set_defaults(run=dataset)
    args = parser.parse_args()
    if not math.isfinite(getattr(args, "timeout", 15)) or not 0 < getattr(args, "timeout", 15) <= 60:
        parser.error("timeout must be in (0, 60]")
    if not 1 <= getattr(args, "count", 100) <= 1000:
        parser.error("count must be in [1, 1000]")
    args.run(args)


if __name__ == "__main__":
    main()
