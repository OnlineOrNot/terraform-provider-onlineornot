#!/usr/bin/env python3
import json
import sys
from collections import Counter
from pathlib import Path

HTTP_METHODS = {"get", "post", "put", "patch", "delete"}
VALID_STATUSES = {"implemented", "planned", "waived"}

schema_path = Path(sys.argv[1] if len(sys.argv) > 1 else "openapi.json")
manifest_path = Path(sys.argv[2] if len(sys.argv) > 2 else "operation-parity.json")

schema = json.loads(schema_path.read_text())
manifest = json.loads(manifest_path.read_text())

schema_operations = []
for path_item in schema["paths"].values():
    for method, operation in path_item.items():
        if method in HTTP_METHODS:
            operation_id = operation.get("operationId")
            if not operation_id:
                raise SystemExit(f"OpenAPI {method.upper()} operation is missing operationId")
            schema_operations.append(operation_id)

manifest_operations = manifest["operations"]
manifest_ids = [entry["operationId"] for entry in manifest_operations]
errors = []

for label, values in (("OpenAPI", schema_operations), ("manifest", manifest_ids)):
    duplicates = sorted(value for value, count in Counter(values).items() if count > 1)
    if duplicates:
        errors.append(f"Duplicate {label} operation IDs: {', '.join(duplicates)}")

missing = sorted(set(schema_operations) - set(manifest_ids))
extra = sorted(set(manifest_ids) - set(schema_operations))
if missing:
    errors.append(f"Operations missing from manifest: {', '.join(missing)}")
if extra:
    errors.append(f"Manifest operations absent from schema: {', '.join(extra)}")

for entry in manifest_operations:
    operation_id = entry.get("operationId", "<missing>")
    status = entry.get("status")
    if status not in VALID_STATUSES:
        errors.append(f"{operation_id}: invalid status {status!r}")
    if not entry.get("terraform"):
        errors.append(f"{operation_id}: terraform mapping is required")
    if status in {"planned", "waived"} and not entry.get("reason"):
        errors.append(f"{operation_id}: {status} operations require a reason")

if errors:
    print("Operation parity check failed:", file=sys.stderr)
    for error in errors:
        print(f"- {error}", file=sys.stderr)
    raise SystemExit(1)

counts = Counter(entry["status"] for entry in manifest_operations)
print(
    f"Mapped all {len(schema_operations)} OpenAPI operations: "
    + ", ".join(f"{status}={counts[status]}" for status in sorted(counts))
)
