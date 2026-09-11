## 0.1.0 (Unreleased)

FEATURES:

BUG FIXES:

- Correct DNS/TCP assertion type validators to match their API endpoints. Reject unsupported DNS record types `PTR`/`SRV`/`CAA`, DNS protocol `HTTPS`, TCP IP family `Any`, and HTTP-only assertion types on DNS/TCP checks. Preserve all supported comparisons and required empty assertion strings.
