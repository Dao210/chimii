#!/usr/bin/env python3
"""Bounded BrickGPT candidate generation using explicitly supplied local weights."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

from bridge import digest, quiet_native_stdout, verify_upstream, write_json


def weight_manifest(args):
    def hashes(root, paths):
        result = {}
        for path in sorted(paths):
            hasher = hashlib.sha256()
            with path.open("rb") as stream:
                while block := stream.read(1024 * 1024):
                    hasher.update(block)
            result[str(path.relative_to(root))] = hasher.hexdigest()
        return result
    base, adapter = Path(args.base_model), Path(args.adapter)
    return {"base": hashes(base, list(base.glob("*.json")) + list(base.glob("*.safetensors"))),
            "adapter": hashes(adapter, list(adapter.glob("*.json")) + list(adapter.glob("*.safetensors")))}


def preflight(args):
    lock = verify_upstream(args.upstream)
    base, adapter = Path(args.base_model), Path(args.adapter)
    if not (base / "config.json").is_file() or not list(base.glob("*.safetensors")):
        return {"status": "base_model_missing", "required": "authorized local meta-llama/Llama-3.2-1B-Instruct directory"}
    required = ["adapter_config.json", "adapter_model.safetensors", "tokenizer.json", "tokenizer_config.json"]
    if any(not (adapter / name).is_file() for name in required):
        return {"status": "adapter_missing", "required": "local AvaLovelace/BrickGPT adapter and tokenizer files"}
    config = json.loads((adapter / "adapter_config.json").read_text())
    if config.get("base_model_name_or_path") != "meta-llama/Llama-3.2-1B-Instruct":
        return {"status": "incompatible_base_model"}
    return {"status": "ready", "upstream_revision": lock["revision"]}


def worker(args):
    # Loading remote code or silently using ambient Hugging Face credentials is
    # outside this prototype. Both base model and adapter must already exist.
    os.environ["HF_HUB_OFFLINE"] = "1"
    os.environ["HF_HUB_DISABLE_IMPLICIT_TOKEN"] = "1"
    os.environ["TRANSFORMERS_OFFLINE"] = "1"
    os.environ["TOKENIZERS_PARALLELISM"] = "false"
    sys.path.insert(0, str(Path(args.upstream).resolve() / "src"))
    from transformers import AutoModelForCausalLM, AutoTokenizer, set_seed
    from peft import PeftModel
    from brickgpt.models.llm import LLM
    import brickgpt.models.brickgpt as upstream

    class LocalLLM(LLM):
        def __init__(self, model_name, device):
            self.device = device
            self.tokenizer = AutoTokenizer.from_pretrained(args.adapter, local_files_only=True, trust_remote_code=False)
            base = AutoModelForCausalLM.from_pretrained(args.base_model, local_files_only=True, trust_remote_code=False)
            self.model = PeftModel.from_pretrained(base, args.adapter, local_files_only=True).to(device).eval()
            self.kv_cache = self.kv_cache_saved = self.input_ids_cache = self.input_ids_cache_saved = None

    # Replace only local weight loading; reuse upstream rejection/rollback logic.
    upstream.LLM = LocalLLM
    set_seed(args.seed)
    start = time.perf_counter()
    model = upstream.BrickGPT(upstream.BrickGPTConfig(model_name_or_path=args.adapter, device=args.device,
                              max_bricks=args.max_bricks, max_brick_rejections=20, max_regenerations=3, use_gurobi=True))
    load_ms = (time.perf_counter() - start) * 1000
    start = time.perf_counter()
    generated = model(args.prompt)
    bricks = generated["bricks"]
    # A result after retry exhaustion can still be unstable. Keep the candidate
    # for diagnosis and expose this check separately from Chimii acceptance.
    stable = bool(bricks.is_stable()) if len(bricks) else False
    return {"status": "candidate_generated", "load_ms": load_ms, "generation_and_final_check_ms": (time.perf_counter() - start) * 1000,
            "upstream_final_stable": stable, "bricks": bricks.to_txt(), "n_regenerations": generated["n_regenerations"],
            "rejection_reasons": dict(generated["rejection_reasons"])}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--upstream", required=True)
    p.add_argument("--base-model", required=True)
    p.add_argument("--adapter", required=True)
    p.add_argument("--prompt", required=True, help="English geometry description; no live translation is performed")
    p.add_argument("--output", required=True)
    p.add_argument("--seed", type=int, default=42)
    p.add_argument("--device", choices=["cpu", "mps", "cuda"], default="cpu")
    p.add_argument("--timeout", type=float, default=180)
    p.add_argument("--max-bricks", type=int, default=100)
    p.add_argument("--worker", action="store_true", help=argparse.SUPPRESS)
    args = p.parse_args()
    if not 1 <= args.max_bricks <= 200 or not 1 <= args.timeout <= 600 or not 1 <= len(args.prompt) <= 1000:
        p.error("invalid part, prompt or time budget")
    checked = preflight(args)
    if checked["status"] != "ready":
        write_json(args.output, {"version": 1, "candidates": []})
        write_json(args.output + ".status.json", checked)
        print(json.dumps(checked))
        return 2
    if args.worker:
        try:
            with quiet_native_stdout():
                result = worker(args)
        except ImportError as exc:
            result = {"status": "dependency_missing", "module": exc.name}
        except Exception as exc:
            result = {"status": "generation_error", "error_type": type(exc).__name__}
        print(json.dumps(result))
        return 0
    start = time.perf_counter()
    weights = weight_manifest(args)
    try:
        child = subprocess.run([sys.executable, __file__, *sys.argv[1:], "--worker"], capture_output=True, text=True, timeout=args.timeout)
        result = json.loads(child.stdout) if child.returncode == 0 else {"status": "worker_error", "exit_code": child.returncode}
        if not isinstance(result, dict) or result.get("status") not in {"candidate_generated", "worker_error", "generation_error", "dependency_missing"}:
            raise ValueError("invalid worker output")
        if result["status"] == "candidate_generated" and (not isinstance(result.get("bricks"), str) or type(result.get("upstream_final_stable")) is not bool):
            raise ValueError("candidate output missing bricks or final check")
    except subprocess.TimeoutExpired:
        result = {"status": "timeout"}
    except (TypeError, ValueError):
        result = {"status": "invalid_worker_output"}
    result.update(upstream_revision=checked["upstream_revision"], elapsed_ms=(time.perf_counter()-start)*1000, seed=args.seed,
                  scope="local text generation; requires independent Chimii acceptance; attached-baseplate physics", weights=weights)
    if result["status"] == "candidate_generated":
        source = {"kind": "brickgpt", "revision": checked["upstream_revision"], "seed": args.seed,
                  "model": "AvaLovelace/BrickGPT", "model_hash": digest(weights)}
        candidate = {"id": "brickgpt-"+digest([args.prompt, source, args.max_bricks])[:16],
                     "prompt": args.prompt, "bricks": result.pop("bricks"), "color": 1,
                     "inventory": {"configured": False, "items": []}, "source": source}
        write_json(args.output, {"version": 1, "candidates": [candidate]})
    else:
        write_json(args.output, {"version": 1, "candidates": []})
    write_json(args.output + ".status.json", result)
    print(json.dumps(result))
    return 0 if result["status"] == "candidate_generated" else 2


if __name__ == "__main__":
    raise SystemExit(main())
