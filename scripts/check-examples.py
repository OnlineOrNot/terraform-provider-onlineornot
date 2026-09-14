#!/usr/bin/env python3
"""Validate checked-in examples against the local provider, without API access."""

import os
from pathlib import Path
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parents[1]


def main():
    terraform = shutil.which(os.environ.get("TF_ACC_TERRAFORM_PATH", "terraform"))
    if terraform is None:
        raise SystemExit("Install Terraform or set TF_ACC_TERRAFORM_PATH before running this check.")

    # Arrange: isolate Terraform configuration, provider installation and state.
    with tempfile.TemporaryDirectory(prefix="onlineornot-examples-") as temporary:
        work = Path(temporary)
        platform = subprocess.check_output(
            ["go", "env", "GOOS", "GOARCH"], cwd=ROOT, text=True
        ).split()
        mirror = work / "providers"
        package = mirror / "registry.terraform.io/onlineornot/onlineornot/0.0.1" / "_".join(platform)
        package.mkdir(parents=True)
        subprocess.run(
            ["go", "build", "-o", str(package / "terraform-provider-onlineornot_v0.0.1"), "."],
            cwd=ROOT, check=True,
        )
        cli_config = work / "terraform.rc"
        cli_config.write_text(
            'provider_installation {\n  filesystem_mirror {\n'
            f'    path = "{mirror.as_posix()}"\n'
            '    include = ["registry.terraform.io/onlineornot/onlineornot"]\n'
            '  }\n}\ndisable_checkpoint = true\n'
        )
        env = {
            key: value for key, value in os.environ.items()
            if not key.startswith(("TF_", "ONLINEORNOT_"))
        }
        env.update(TF_CLI_CONFIG_FILE=str(cli_config), TF_IN_AUTOMATION="1", CHECKPOINT_DISABLE="1")
        examples = work / "examples"
        shutil.copytree(ROOT / "examples", examples)
        directories = sorted({path.parent for path in examples.rglob("*.tf")})
        if not directories:
            raise SystemExit("No Terraform examples found.")

        for directory in directories:
            print(f"Validating {directory.relative_to(work)}", flush=True)
            # Only the provider selection is overridden. Resource/data-source examples
            # and their companion scripts are copied unchanged.
            (directory / "local_provider_override.tf").write_text(
                'terraform {\n  required_providers {\n    onlineornot = {\n'
                '      source = "onlineornot/onlineornot"\n      version = "0.0.1"\n'
                '    }\n  }\n}\n'
            )
            # Act and assert: Terraform must accept every configuration. The mirror
            # has no registry fallback, so init cannot substitute a released provider.
            subprocess.run(
                [terraform, "init", "-backend=false", "-input=false", "-no-color"],
                cwd=directory, env=env, check=True,
            )
            subprocess.run(
                [terraform, "validate", "-no-color"], cwd=directory, env=env, check=True,
            )
        print(f"Validated {len(directories)} example configurations.")


if __name__ == "__main__":
    main()
