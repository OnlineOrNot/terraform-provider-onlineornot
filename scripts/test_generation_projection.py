import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).with_name("project-generation-openapi.py")
spec = importlib.util.spec_from_file_location("projection", SCRIPT)
projection = importlib.util.module_from_spec(spec)
spec.loader.exec_module(projection)


def envelope(success):
    return {
        "type": "object",
        "properties": {
            "success": {"type": "boolean", "enum": [success]},
            "result": {"type": "object", "properties": {"id": {"type": "string"}}} if success else {"type": "null"},
            "errors": {"type": "array", "items": {"type": "object"}},
            "messages": {"type": "array", "items": {"type": "object"}},
        },
        "required": ["success", "result", "errors", "messages"],
    }


def fixture():
    return {
        "paths": {"/v1/checks": {"get": {"operationId": "listChecks", "responses": {
            "200": {"content": {"application/json": {"schema": {"anyOf": [
                envelope(True), {"$ref": projection.ERROR_REF},
            ]}}}},
            "400": {"content": {"application/json": {"schema": {"$ref": projection.ERROR_REF}}}},
        }}}},
        "components": {"schemas": {"PublicApiErrorResponse": envelope(False)}},
    }


def response_schema(document):
    return document["paths"]["/v1/checks"]["get"]["responses"]["200"]["content"]["application/json"]["schema"]


class ProjectionTests(unittest.TestCase):
    def test_both_orders_inline_and_referenced_branches(self):
        for reverse in (False, True):
            for refs in (False, True):
                with self.subTest(reverse=reverse, refs=refs):
                    document = fixture()
                    if refs:
                        document["components"]["schemas"]["Success"] = envelope(True)
                        response_schema(document)["anyOf"][0] = {"$ref": "#/components/schemas/Success"}
                    else:
                        response_schema(document)["anyOf"][1] = envelope(False)
                    if reverse:
                        response_schema(document)["anyOf"].reverse()
                    original = copy.deepcopy(document)
                    projected, selected = projection.project(document)
                    self.assertEqual(selected, [("get", "/v1/checks")])
                    expected = copy.deepcopy(document)
                    response_schema(expected).clear()
                    response_schema(expected).update(envelope(True))
                    self.assertEqual(projected, expected)
                    self.assertEqual(document, original)

    def test_shared_response_and_schema_refs_are_not_mutated(self):
        document = fixture()
        schemas = document["components"]["schemas"]
        schemas["Envelope"] = copy.deepcopy(response_schema(document))
        response_schema(document).clear()
        response_schema(document)["$ref"] = "#/components/schemas/Envelope"
        responses = document["paths"]["/v1/checks"]["get"]["responses"]
        document["components"]["responses"] = {"Shared": responses["200"]}
        responses["200"] = {"$ref": "#/components/responses/Shared"}
        document["paths"]["/v1/checks"]["post"] = copy.deepcopy(document["paths"]["/v1/checks"]["get"])
        original = copy.deepcopy(document)
        result, selected = projection.project(document)
        self.assertEqual(len(selected), 2)
        self.assertEqual(response_schema(result), envelope(True))
        self.assertEqual(result["components"], original["components"])
        self.assertEqual(document, original)

    def test_unsupported_envelopes_fail_closed(self):
        mutations = {
            "oneOf": lambda s: s.update(oneOf=s.pop("anyOf")),
            "malformed union": lambda s: s.update(anyOf=None),
            "third branch": lambda s: s["anyOf"].append(envelope(True)),
            "two successes": lambda s: s["anyOf"].__setitem__(1, envelope(True)),
            "two failures": lambda s: s["anyOf"].__setitem__(0, envelope(False)),
            "optional discriminator": lambda s: s["anyOf"][0]["required"].remove("success"),
            "optional result": lambda s: s["anyOf"][0]["required"].remove("result"),
            "nonliteral": lambda s: s["anyOf"][0]["properties"]["success"].pop("enum"),
            "numeric literal": lambda s: s["anyOf"][0]["properties"]["success"].update(enum=[1]),
            "mixed boolean": lambda s: s["anyOf"][0]["properties"]["success"].update(enum=[True, False]),
            "root condition": lambda s: s.update(**{"if": {"type": "object"}}),
            "branch condition": lambda s: s["anyOf"][0].update(**{"if": {"type": "object"}}),
            "root sibling": lambda s: s.update(type="object"),
            "mixed branch": lambda s: s["anyOf"].__setitem__(1, {"type": "string"}),
        }
        for label, mutate in mutations.items():
            with self.subTest(label=label):
                document = fixture()
                mutate(response_schema(document))
                with self.assertRaises(ValueError):
                    projection.project(document)

    def test_failure_requires_canonical_shape(self):
        for field in ("success", "result", "errors", "messages"):
            document = fixture()
            document["components"]["schemas"]["PublicApiErrorResponse"]["required"].remove(field)
            with self.assertRaises(ValueError):
                projection.project(document)
        document = fixture()
        failure = envelope(False)
        failure["description"] = "different error contract"
        response_schema(document)["anyOf"][1] = failure
        with self.assertRaisesRegex(ValueError, "not canonical"):
            projection.project(document)

    def test_bad_references(self):
        for ref in ("https://example.invalid/schema.json", "#/missing", "#/components/schemas/Cycle"):
            document = fixture()
            document["components"]["schemas"]["Cycle"] = {"$ref": "#/components/schemas/Cycle2"}
            document["components"]["schemas"]["Cycle2"] = {"$ref": "#/components/schemas/Cycle"}
            response_schema(document)["anyOf"][0] = {"$ref": ref}
            with self.assertRaises(ValueError):
                projection.project(document)
        document = fixture()
        response_schema(document)["anyOf"][1]["description"] = "ref sibling"
        with self.assertRaises(ValueError):
            projection.project(document)

    def test_escaped_pointer(self):
        document = fixture()
        document["components"]["schemas"]["a/b~c"] = envelope(True)
        response_schema(document)["anyOf"][0] = {"$ref": "#/components/schemas/a~1b~0c"}
        self.assertEqual(response_schema(projection.project(document)[0]), envelope(True))

    def test_unrelated_unions_and_statuses_are_preserved(self):
        document = fixture()
        success = response_schema(document)["anyOf"][0]
        union = {"anyOf": [{"type": "object"}, {"type": "string"}]}
        success["properties"]["result"]["properties"]["unsupported"] = union
        document["components"]["schemas"]["Unrelated"] = union
        document["paths"]["/v1/checks"]["get"]["responses"]["201"] = {"content": {"application/json": {"schema": union}}}
        result, _ = projection.project(document)
        self.assertEqual(response_schema(result), success)
        self.assertEqual(result["components"], document["components"])
        self.assertEqual(result["paths"]["/v1/checks"]["get"]["responses"]["201"], document["paths"]["/v1/checks"]["get"]["responses"]["201"])
        document = fixture()
        response_schema(document)["anyOf"][0]["properties"]["result"] = union
        self.assertEqual(response_schema(projection.project(document)[0])["properties"]["result"], union)
        # A success-only canonical schema needs no projection.
        self.assertEqual(projection.project(result), (result, []))

    def test_cli_digest_checked_before_projection_and_input_unchanged(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, output, lock = (root / name for name in ("input.json", "output.json", "lock.json"))
            raw = json.dumps(fixture()).encode()
            source.write_bytes(raw)
            lock.write_text(json.dumps({"sha256": "wrong"}))
            command = [sys.executable, str(SCRIPT), str(source), str(output), "--lock", str(lock)]
            failed = subprocess.run(command, capture_output=True, text=True)
            self.assertNotEqual(failed.returncode, 0)
            self.assertIn("digest mismatch", failed.stderr)
            self.assertFalse(output.exists())
            lock.write_text(json.dumps({"sha256": hashlib.sha256(raw).hexdigest()}))
            subprocess.run(command, check=True, capture_output=True)
            self.assertEqual(source.read_bytes(), raw)
            self.assertEqual(response_schema(json.loads(output.read_text())), envelope(True))
            command[3] = str(source)
            self.assertNotEqual(subprocess.run(command, capture_output=True).returncode, 0)
            self.assertEqual(source.read_bytes(), raw)
            alias = root / "alias.json"
            alias.hardlink_to(source)
            command[3] = str(alias)
            self.assertNotEqual(subprocess.run(command, capture_output=True).returncode, 0)
            self.assertEqual(source.read_bytes(), raw)


if __name__ == "__main__":
    unittest.main()
