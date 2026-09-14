import copy
import importlib.util
from pathlib import Path
import unittest

module = importlib.util.spec_from_file_location('prepare_openapi', Path(__file__).with_name('prepare-openapi.py'))
prepare_openapi = importlib.util.module_from_spec(module)
module.loader.exec_module(prepare_openapi)


class PrepareOpenAPITest(unittest.TestCase):
    def fixture(self):
        success = {'type': 'object', 'required': ['success', 'result'], 'properties': {
            'success': {'type': 'boolean', 'enum': [True]},
            'result': {'anyOf': [{'$ref': '#/components/schemas/Check'}, {'type': 'null'}]},
        }}
        failure = {'type': 'object', 'required': ['success', 'result'], 'properties': {
            'success': {'type': 'boolean', 'enum': [False]}, 'result': {'type': 'null'},
        }}
        schema = {'anyOf': [failure, success]}
        return {'paths': {'/checks': {'get': {'operationId': 'getCheck', 'responses': {
            '200': {'content': {'application/json': {'schema': schema}}},
        }}}}}, schema, success

    def test_success_mapping_without_mutating_contract(self):
        spec, _, success = self.fixture()
        original = copy.deepcopy(spec)
        prepared = prepare_openapi.prepare(spec)
        mapped = prepared['paths']['/checks']['get']['responses']['200']['content']['application/json']['schema']
        self.assertEqual(mapped, success)
        self.assertEqual(spec, original)
        self.assertEqual(prepare_openapi.prepare(prepared), prepared)

    def test_rejects_ambiguous_or_unsupported_union(self):
        for change in ('duplicate', 'missing_flag', 'sibling'):
            with self.subTest(change=change):
                spec, schema, success = self.fixture()
                if change == 'duplicate':
                    schema['anyOf'].append(copy.deepcopy(success))
                elif change == 'missing_flag':
                    del success['properties']['success']
                else:
                    schema['description'] = 'Cannot silently discard sibling constraints'
                with self.assertRaisesRegex(ValueError, 'Unsupported response union in getCheck'):
                    prepare_openapi.prepare(spec)
