# OpenAPI generation compatibility

`make generate-schemas` and `go generate` fetch and verify `schema.lock.json`,
check operation parity against the original `openapi.json`, then write
`openapi.generation.json` for the pinned OpenAPI generator v0.3.0. The projection
also checks the pinned digest before reading schemas. Neither the lock nor the
canonical spec is changed. The projection is not an API contract or a runtime
client input.

The generator skips non-null object unions, sometimes with exit status zero.
For HTTP 200 application/json response roots only, the projection accepts exactly
two `anyOf` alternatives:

- A flat object with required `success`, `result`, `errors`, and `messages`, with
  `success` constrained to boolean `enum: [true]`.
- A flat failure envelope with boolean `enum: [false]`, null `result`, and the same
  four required fields, equal to the canonical `PublicApiErrorResponse` component.
  That public compatibility name represents upstream `ResourceApiErrorResponse`.

Either branch order and inline or pure local references work. The success branch
is copied to the operation's response, without editing shared components,
response metadata, other statuses, or any result/payload schema. Unrelated
components are never projected. This includes the existing `AnyCheck`
discriminated `oneOf` payload.

Unrecognized root compositions, conditional schemas, ambiguous discriminators,
missing canonical errors, ref siblings, external refs, unresolved refs, and
reference-chain cycles fail closed. Conditional/composed alternatives are
intentionally unsupported. Payload unions remain unchanged, not arbitrarily
stripped; this script does not certify generator support for them. Review full
attribute mappings when the canonical schema changes, rather than trusting the
generator's exit status or resource names.

## Tests

```sh
make test-generation
python3 scripts/test-generation-compatibility.py BASELINE.json CANDIDATE.json \
  --baseline-lock BASELINE.lock.json --candidate-lock CANDIDATE.lock.json
```

The integration command verifies both supplied SHA256 locks, checks operation
parity on both original inputs, generates the prior verified baseline as-is and
the projected candidate, and runs both pinned generator stages, including
framework resources and data sources. All outputs are temporary and removed.
It requires exactly 9 resources and 5 data sources, compares their full recursive
attribute structures, and verifies both source files remain byte-identical.
For unpublished upstream exports, use separate local lock files containing the
independently confirmed `sha256`; do not change the repository lock to bypass
verification. Normal generation always uses `schema.lock.json`.
