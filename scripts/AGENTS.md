# Contract and Validation Scripts

Python and Bash tooling for digest-pinned OpenAPI generation and API-free Terraform example validation.

## Where to Look

| Task | File | Input/output |
|------|------|--------------|
| Download exact contract | `fetch-openapi.sh` | Root `schema.lock.json` -> digest-verified output file |
| Adapt response envelopes for generator | `prepare-openapi.py` | Original OpenAPI -> generator-only projection |
| Check operation coverage | `check-operation-parity.py` | OpenAPI + `operation-parity.json` -> validation result |
| Validate shipped examples | `check-examples.py` | Local provider + copied `examples/` -> Terraform validation |
| Projection regression coverage | `test_prepare_openapi.py` | Supported/unsupported union shapes and preservation |

## Contract Invariants

- Run these scripts via root make targets or from the repository root. Schema fetch reads `schema.lock.json` relative to the working directory.
- Fetch writes a temporary file, compares SHA-256, then moves it into place. Keep mismatch failure and temporary-file cleanup intact.
- `prepare` deep-copies its input. Only a supported top-level HTTP 200 response `anyOf` with success/failure object branches is projected to the success branch.
- Unsupported response unions raise an error. Nested resource unions and nullable fields remain unchanged; this script is not a general schema simplifier.
- The projection only serves code generation; runtime API clients still validate success/failure envelopes.
- Parity covers GET/POST/PUT/PATCH/DELETE operation IDs. Duplicate IDs, missing/extra mappings, and unsupported dispositions are errors.
- Every manifest entry needs a Terraform mapping; `planned` and `waived` entries also need a reason. A passing check establishes coverage, not implementation correctness.

## Example Isolation

- `check-examples.py` builds the current provider into a temporary platform-specific filesystem mirror, copies examples and companion files, then overrides only provider selection.
- The mirror has no registry fallback. Preserve that guarantee so validation cannot silently use a released provider.
- Child processes strip inherited `TF_*` and `ONLINEORNOT_*` variables and receive an isolated CLI configuration. The script resolves `TF_ACC_TERRAFORM_PATH` before stripping them.
- Only `terraform init -backend=false` and `terraform validate` run; no plan/apply or live credentials are needed. Temporary directories are removed when the script exits.

## Verification

From the root: `python3 -m unittest discover -s scripts -p 'test_*.py'` checks projection behavior. Use `make check-contract` for pinned operation coverage and `make check-examples` when modifying Terraform examples or their validation runner.
