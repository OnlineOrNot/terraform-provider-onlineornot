#!/usr/bin/env bash
set -euo pipefail

repo=$(python3 -c 'import json; print(json.load(open("schema.lock.json"))["repository"])')
commit=$(python3 -c 'import json; print(json.load(open("schema.lock.json"))["commit"])')
path=$(python3 -c 'import json; print(json.load(open("schema.lock.json"))["path"])')
expected=$(python3 -c 'import json; print(json.load(open("schema.lock.json"))["sha256"])')
output=${1:-openapi.json}
temporary="${output}.tmp"

trap 'rm -f "$temporary"' EXIT
curl --fail --silent --show-error --location \
  "https://raw.githubusercontent.com/${repo}/${commit}/${path}" \
  --output "$temporary"

actual=$(python3 - "$temporary" <<'PY'
import hashlib
import sys

with open(sys.argv[1], "rb") as schema:
    print(hashlib.sha256(schema.read()).hexdigest())
PY
)

if [[ "$actual" != "$expected" ]]; then
  echo "OpenAPI schema digest mismatch: expected $expected, got $actual" >&2
  exit 1
fi

mv "$temporary" "$output"
trap - EXIT
