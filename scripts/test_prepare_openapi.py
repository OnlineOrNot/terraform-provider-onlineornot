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

    def test_project_id_pattern_is_re2_compatible_without_changing_contract(self):
        pattern = '^(?=.*[a-zA-Z-])[a-zA-Z0-9-]+$'
        spec = {'paths': {}, 'components': {'schemas': {'ProjectId': {'type': 'string', 'pattern': pattern}}}}
        prepared = prepare_openapi.prepare(spec)
        self.assertEqual(prepared['components']['schemas']['ProjectId']['pattern'], '^[a-zA-Z0-9-]*[a-zA-Z-][a-zA-Z0-9-]*$')
        self.assertEqual(spec['components']['schemas']['ProjectId']['pattern'], pattern)
        import re
        for value in ('1234567890123456', 'sourceProject1234567', '-1234567890123456', '123abc', '', 'bad/value', 'abc_def', 'abcé'):
            self.assertEqual(bool(re.fullmatch(pattern, value)), bool(re.fullmatch(prepared['components']['schemas']['ProjectId']['pattern'], value)))

    def test_short_project_id_pattern_is_re2_compatible_without_changing_contract(self):
        import random
        import re
        pattern = prepare_openapi.PROJECT_ID_PATTERN
        spec = {'paths': {}, 'components': {'schemas': {'ProjectId': {
            'type': 'string', 'minLength': 12, 'maxLength': 128, 'pattern': pattern}}}}
        prepared = prepare_openapi.prepare(spec)
        rewritten = prepared['components']['schemas']['ProjectId']['pattern']
        self.assertNotIn('(?=', rewritten)
        self.assertEqual(spec['components']['schemas']['ProjectId']['pattern'], pattern)
        self.assertEqual(prepare_openapi.prepare(prepared), prepared)

        def valid(candidate, regex):
            return 12 <= len(candidate) <= 128 and bool(re.fullmatch(regex, candidate))

        values = ['izkpm55j334u', 'IZKPM55J334U', 'izkpm55j334', 'izkpm55j334uu', '123456789012',
                  '1234567890123456', '12345678901234567', '123456789012345a', 'a234567890123456',
                  '1234567890123456-', '-23456789012345', 'sourceProject1234567', 'abc_def1234567890',
                  'abcé123456789012', '', 'a' * 128, 'a' * 129, '1' * 127 + 'a', '1' * 128 + 'a']
        rng = random.Random(0)
        for _ in range(20000):
            alphabet = rng.choice(['0123456789', '0123456789a', '0123456789-Z', 'abc012', '0123456789_'])
            values.append(''.join(rng.choice(alphabet) for _ in range(rng.choice([11, 12, 13, 15, 16, 17, 40, 128, 129]))))
        for value in values:
            self.assertEqual(valid(value, pattern), valid(value, rewritten), value)

    def test_check_assertions_keep_supported_string_branch_without_mutating_contract(self):
        legacy = {'type': 'object', 'properties': {'type': {'enum': ['JSON_BODY']}, 'expected': {'type': 'string'}}}
        status = {'type': 'object', 'properties': {'type': {'enum': ['STATUS_CODE']}, 'expected': {'type': 'array', 'items': {'type': 'number'}}}}
        spec = {'paths': {}, 'components': {'schemas': {'CheckInput': {'properties': {'assertions': {'type': 'array', 'items': {'oneOf': [legacy, status]}}}}}}}
        original = copy.deepcopy(spec)
        prepared = prepare_openapi.prepare(spec)
        items = prepared['components']['schemas']['CheckInput']['properties']['assertions']['items']
        self.assertEqual(items, legacy)
        self.assertEqual(spec, original)
        self.assertEqual(prepare_openapi.prepare(prepared), prepared)
