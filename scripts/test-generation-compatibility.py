#!/usr/bin/env python3
"""Run both pinned generators in temporary directories and compare complete mappings."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parent.parent
OPENAPI = "github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi"
FRAMEWORK = "github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework"


def run(*args):
    subprocess.run([str(arg) for arg in args], cwd=ROOT, check=True)


def mappings(code):
    result = {}
    for kind, count in (("resources", 9), ("datasources", 5)):
        entries = code[kind]
        if len(entries) != count or len({entry["name"] for entry in entries}) != count:
            raise ValueError(f"expected {count} distinct {kind}")
        # Include every nested attribute field: type, defaults, validators,
        # computed/optional/required, descriptions, and nested object structures.
        result[kind] = {entry["name"]: entry["schema"]["attributes"] for entry in entries}
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("baseline", type=Path)
    parser.add_argument("candidate", type=Path)
    parser.add_argument("--baseline-lock", type=Path, required=True)
    parser.add_argument("--candidate-lock", type=Path, default=ROOT / "schema.lock.json")
    args = parser.parse_args()
    generated = []
    with tempfile.TemporaryDirectory(prefix="oon-generation-compat-") as temporary:
        for label, source, lock in (
            ("baseline", args.baseline.resolve(), args.baseline_lock.resolve()),
            ("candidate", args.candidate.resolve(), args.candidate_lock.resolve()),
        ):
            original = source.read_bytes()
            directory = Path(temporary) / label
            directory.mkdir()
            projection = directory / "generation.json"
            # Validate both digests; the prior verified baseline is generated as-is.
            if hashlib.sha256(original).hexdigest() != json.loads(lock.read_text())["sha256"]:
                raise ValueError(f"{label} digest mismatch")
            if label == "candidate":
                run(sys.executable, ROOT / "scripts/project-generation-openapi.py", source, projection, "--lock", lock)
            else:
                projection.write_bytes(original)
            # Operation inventory is always checked against the canonical input.
            run(sys.executable, ROOT / "scripts/check-operation-parity.py", source, ROOT / "operation-parity.json")
            code = directory / "code.json"
            run("go", "run", OPENAPI, "generate", "--config", ROOT / "generator_config.yml", "--output", code, projection)
            generated.append(mappings(json.loads(code.read_text())))
            for kind in ("resources", "data-sources"):
                run("go", "run", FRAMEWORK, "generate", kind, "--input", code, "--output", directory / "framework")
            if source.read_bytes() != original:
                raise ValueError(f"{label} canonical input was modified")
            print(f"{label} SHA256 {hashlib.sha256(original).hexdigest()}", flush=True)
        if generated[0] != generated[1]:
            for kind in generated[0]:
                for name in generated[0][kind]:
                    if generated[0][kind][name] != generated[1][kind].get(name):
                        print(f"Changed attribute structure: {kind}/{name}", file=sys.stderr)
            raise SystemExit("Generated attribute structures differ")
        digest = hashlib.sha256(json.dumps(generated[0], sort_keys=True).encode()).hexdigest()
        print(f"All 9 resource + 5 data-source full attribute structures match; mapping SHA256 {digest}")


if __name__ == "__main__":
    main()
