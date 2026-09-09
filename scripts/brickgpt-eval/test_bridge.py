import argparse
import contextlib
import hashlib
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import bridge
import generate


def part(x=2, z=4, height=3):
    return {"studs_x": x, "studs_z": z, "plates_y": height, "geometry_profile": "stud_tube_rect", "certification_level": "certified"}


def placement(identifier="a", **kwargs):
    return {"id": identifier, "part_id": "brick", "x": -3, "y": 0, "z": -4, "rotation": 0, "step": 1, **kwargs}


class BridgeContracts(unittest.TestCase):
    def test_source_verification_does_not_start_upstream(self):
        output = io.StringIO()
        with patch.object(sys, "argv", ["bridge.py", "verify-source"]), \
                patch.object(bridge, "verify_upstream", return_value={"revision": "pin", "files": {"module.py": "hash"}}) as verify, \
                patch.object(bridge.subprocess, "run") as child, contextlib.redirect_stdout(output):
            bridge.main()
        verify.assert_called_once_with(str(bridge.DEFAULT_UPSTREAM))
        child.assert_not_called()
        self.assertEqual(json.loads(output.getvalue())["status"], "verified")
        self.assertEqual(json.loads(output.getvalue())["files_checked"], 1)

    def test_axis_rotation_height_and_order(self):
        p = [placement(rotation=90), placement("b", y=3, step=2)]
        result = bridge.as_bricks(p, {"brick": part()})
        self.assertEqual(result[0], {"h": 4, "w": 2, "x": 0, "y": 0, "z": 0})
        self.assertEqual(result[1]["z"], 1)
        self.assertEqual(p[0]["x"], -3)

    def test_no_implicit_grounding(self):
        self.assertEqual(bridge.as_bricks([placement(y=6)], {"brick": part()})[0]["z"], 2)

    def test_non_applicable_geometry_is_never_approximated(self):
        for catalog, p, reason in [
            ({"brick": part(height=1)}, placement(), "unsupported_part"),
            ({"brick": part()}, placement(rotation=45), "unsupported_transform"),
            ({"brick": part()}, placement(y=1), "unsupported_transform"),
            ({"brick": part()}, placement(y=-3), "unsupported_transform"),
            ({"brick": part()}, placement(y=60), "unsupported_grid"),
            ({"brick": part(3, 3)}, placement(), "unsupported_dimension"),
            ({}, placement(), "unsupported_part"),
            ({"brick": part()}, placement(x=True), "non_integer_placement"),
        ]:
            with self.subTest(reason=reason), self.assertRaisesRegex(ValueError, reason):
                bridge.as_bricks([p], catalog)

    def test_collision_duplicate_ids_and_empty_candidates(self):
        for p, reason in [([], "unsupported_part_count"), ([placement(), placement("b")], "collision"),
                          ([placement(), placement(y=3)], "invalid_placement_id")]:
            with self.assertRaisesRegex(ValueError, reason):
                bridge.as_bricks(p, {"brick": part()})

    def test_source_pin_rejects_changed_code(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source, lock = root / "module.py", root / "lock.json"
            source.write_text("original")
            lock.write_text(json.dumps({"files": {"module.py": hashlib.sha256(b"original").hexdigest()}}))
            with patch.object(bridge, "LOCK", lock):
                bridge.verify_upstream(root)
                source.write_text("modified")
                with self.assertRaisesRegex(ValueError, "differs from pinned"):
                    bridge.verify_upstream(root)

    def test_dataset_identity_and_checksum_are_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source, lock = root / "sample.parquet", root / "lock.json"
            source.write_bytes(b"synthetic test file")
            lock.write_text(json.dumps({"revision": "pin", "splits": {"test": {"sha256": hashlib.sha256(source.read_bytes()).hexdigest()}}}))
            with patch.object(bridge, "DATASET_LOCK", lock):
                bridge.verify_dataset(source, "pin", "test")
                with self.assertRaisesRegex(ValueError, "not pinned"):
                    bridge.verify_dataset(source, "different", "test")
                source.write_bytes(b"changed")
                with self.assertRaisesRegex(ValueError, "checksum"):
                    bridge.verify_dataset(source, "pin", "test")

    def test_unknown_or_unavailable_physics_remains_in_denominator(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            report = {"version": 1, "catalog": {"brick": part()}, "cases": [
                {"id": "sample", "input_hash": "input", "placements": [placement()], "validation": {"buildable": True}}]}
            source, output = root / "report.json", root / "physics.json"
            source.write_text(json.dumps(report))
            args = argparse.Namespace(upstream=tmp, input=str(source), output=str(output), prefixes=True, timeout=2)
            for returned, expected in [('{}', "invalid_worker_output"), ('null', "invalid_worker_output"),
                                      ('garbage', "invalid_worker_output"), ('{"status":"solver_size_limit"}', "solver_size_limit"),
                                      ('{"status":"non_optimal"}', "non_optimal")]:
                with self.subTest(expected=expected), patch.object(bridge, "verify_upstream", return_value={"revision": "pin"}), \
                        patch.object(bridge.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, returned, "")), contextlib.redirect_stdout(io.StringIO()):
                    bridge.compare(args)
                result = json.loads(output.read_text())
                self.assertEqual(result["summary"], {expected: 1})
                self.assertEqual(len(result["cases"]), 1)
            with patch.object(bridge, "verify_upstream", return_value={"revision": "pin"}), \
                    patch.object(bridge.subprocess, "run", side_effect=subprocess.TimeoutExpired([], 2)), contextlib.redirect_stdout(io.StringIO()):
                bridge.compare(args)
            self.assertEqual(json.loads(output.read_text())["summary"], {"timeout": 1})

    def test_generation_preflight_needs_authorized_base_weights(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(generate, "verify_upstream", return_value={"revision": "pin"}):
            args = argparse.Namespace(upstream=tmp, base_model=tmp, adapter=tmp)
            self.assertEqual(generate.preflight(args)["status"], "base_model_missing")
            root = Path(tmp)
            (root / "config.json").write_text("{}")
            (root / "model.safetensors").write_bytes(b"test-created fake weights")
            self.assertEqual(generate.preflight(args)["status"], "adapter_missing")

    def test_generation_failure_invalidates_old_candidate_artifact(self):
        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp) / "candidate.json"
            output.write_text('{"version":1,"candidates":[{"id":"old"}]}')
            argv = ["generate.py", "--upstream", tmp, "--base-model", tmp, "--adapter", tmp,
                    "--prompt", "small tower", "--output", str(output)]
            with patch.object(sys, "argv", argv), patch.object(generate, "preflight", return_value={"status": "base_model_missing"}), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(generate.main(), 2)
            self.assertEqual(json.loads(output.read_text())["candidates"], [])

    def test_generation_keeps_unstable_candidates_separate_from_acceptance(self):
        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp) / "candidate.json"
            argv = ["generate.py", "--upstream", tmp, "--base-model", tmp, "--adapter", tmp,
                    "--prompt", "small tower", "--output", str(output)]
            response = {"status": "candidate_generated", "bricks": "2x2 (0,0,2)", "upstream_final_stable": False}
            with patch.object(sys, "argv", argv), patch.object(generate, "preflight", return_value={"status": "ready", "upstream_revision": "pin"}), \
                    patch.object(generate, "weight_manifest", return_value={"test": "synthetic"}), \
                    patch.object(generate.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, json.dumps(response), "")), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(generate.main(), 0)
            self.assertFalse(json.loads(Path(str(output)+".status.json").read_text())["upstream_final_stable"])
            candidate = json.loads(output.read_text())["candidates"][0]
            self.assertEqual(candidate["bricks"], response["bricks"])
            self.assertEqual(len(candidate["source"]["model_hash"]), 64)


if __name__ == "__main__":
    unittest.main()
