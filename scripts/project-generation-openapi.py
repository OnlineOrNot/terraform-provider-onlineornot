#!/usr/bin/env python3
"""Project verified HTTP 200 resource envelopes for tfplugingen-openapi 0.3.0.

Only response roots are projected. Payload schemas and published components are
never rewritten; unsupported payload unions remain the generator's responsibility.
"""
import argparse
import copy
import hashlib
import json
from pathlib import Path

METHODS = {"get", "post", "put", "patch", "delete", "head", "options", "trace"}
ANNOTATIONS = {"title", "description", "example", "examples", "deprecated"}
OBJECT_KEYS = ANNOTATIONS | {"type", "properties", "required", "additionalProperties"}
ENVELOPE_FIELDS = {"success", "result", "errors", "messages"}
# Public compatibility name for upstream ResourceApiErrorResponse.
ERROR_REF = "#/components/schemas/PublicApiErrorResponse"
COMPOSITION = {"anyOf", "oneOf", "allOf", "if", "then", "else", "not"}


def resolve(document, node, seen=()):
    """Resolve only pure local JSON pointers, rejecting cycles and ref siblings."""
    while isinstance(node, dict) and "$ref" in node:
        ref = node["$ref"]
        if set(node) != {"$ref"} or not isinstance(ref, str) or not ref.startswith("#/"):
            raise ValueError(f"unsupported reference: {node!r}")
        if ref in seen:
            raise ValueError(f"cyclic reference: {ref}")
        seen = (*seen, ref)
        node = document
        try:
            for part in ref[2:].split("/"):
                node = node[part.replace("~1", "/").replace("~0", "~")]
        except (KeyError, TypeError):
            raise ValueError(f"unresolved reference: {ref}") from None
    if not isinstance(node, dict):
        raise ValueError("expected a schema/response object")
    return node


def literal_boolean(document, node, value):
    node = resolve(document, node)
    # bool is checked by identity: Python considers 1 == True and 0 == False.
    return (
        set(node) <= ANNOTATIONS | {"type", "enum", "default"}
        and node.get("type") == "boolean"
        and isinstance(node.get("enum"), list)
        and len(node["enum"]) == 1
        and node["enum"][0] is value
    )


def envelope(document, node, success):
    if set(node) - OBJECT_KEYS or node.get("type") != "object":
        return False
    props = node.get("properties", {})
    required = node.get("required", [])
    if not isinstance(props, dict) or not isinstance(required, list):
        return False
    if not ENVELOPE_FIELDS <= props.keys() or not ENVELOPE_FIELDS <= set(required):
        return False
    if not literal_boolean(document, props["success"], success):
        return False
    for field in ("errors", "messages"):
        if resolve(document, props[field]).get("type") != "array":
            return False
    if success:
        # Selection proves the envelope discriminator, not the payload shape.
        # In particular AnyCheck is an existing discriminated oneOf payload.
        # Leave all payload schemas intact for the pinned generator to handle.
        return True
    result = resolve(document, props["result"])
    return set(props) == ENVELOPE_FIELDS and result == {"type": "null"}


def select_success(document, schema):
    schema = resolve(document, schema)
    if not COMPOSITION.intersection(schema):
        return None
    alternatives = schema.get("anyOf")
    if (
        set(schema) - (ANNOTATIONS | {"anyOf"})
        or not isinstance(alternatives, list)
        or len(alternatives) != 2
    ):
        raise ValueError("unsupported HTTP 200 response composition")
    branches = [resolve(document, branch) for branch in alternatives]
    successes = [branch for branch in branches if envelope(document, branch, True)]
    failures = [branch for branch in branches if envelope(document, branch, False)]
    if len(successes) != 1 or len(failures) != 1:
        raise ValueError("expected one required success:true envelope and one success:false envelope")
    canonical_error = resolve(document, {"$ref": ERROR_REF})
    if failures[0] != canonical_error:
        raise ValueError("failure branch is not canonical PublicApiErrorResponse")
    return successes[0]


def project(document):
    projected = copy.deepcopy(document)
    selected = []
    for path, path_item in document["paths"].items():
        path_item = resolve(document, path_item)
        for method, operation in path_item.items():
            if method not in METHODS:
                continue
            response = operation.get("responses", {}).get("200")
            if response is None:
                continue
            response = resolve(document, response)
            media = response.get("content", {}).get("application/json", {})
            if "schema" not in media:
                continue
            try:
                success = select_success(document, media["schema"])
            except ValueError as error:
                raise ValueError(f"{method.upper()} {path}: {error}") from error
            if success is not None:
                # Inline at the operation, never edit a shared response/component.
                target = copy.deepcopy(resolve(projected, projected["paths"][path]))
                target[method]["responses"]["200"] = copy.deepcopy(response)
                target[method]["responses"]["200"]["content"]["application/json"]["schema"] = copy.deepcopy(success)
                projected["paths"][path] = target
                selected.append((method, path))
    return projected, selected


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--lock", type=Path, default=Path("schema.lock.json"))
    args = parser.parse_args()
    if args.input.resolve() == args.output.resolve() or (
        args.output.exists() and args.input.samefile(args.output)
    ):
        parser.error("input and output must differ")
    raw = args.input.read_bytes()
    expected = json.loads(args.lock.read_text())["sha256"]
    actual = hashlib.sha256(raw).hexdigest()
    if actual != expected:
        parser.error(f"OpenAPI schema digest mismatch: expected {expected}, got {actual}")
    try:
        projected, selected = project(json.loads(raw))
    except ValueError as error:
        parser.error(str(error))
    args.output.write_text(json.dumps(projected, indent=2) + "\n")
    print(f"Projected {len(selected)} HTTP 200 envelopes for generation only ({actual})")
    for method, path in selected:
        print(f"  {method.upper()} {path}")


if __name__ == "__main__":
    main()
