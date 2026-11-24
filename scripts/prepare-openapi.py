#!/usr/bin/env python3
"""Project success envelopes for codegen; never modify the pinned contract.

The framework generator cannot map HTTP 200 success/failure anyOf envelopes.
Terraform state describes successful results; the client still checks success
at runtime. The generic check keeps its supported string-valued assertion
branch; other nested resource unions and nullable fields are left unchanged.
"""

import copy
import json
import sys


LEGACY_PROJECT_ID_PATTERN = '^(?=.*[a-zA-Z-])[a-zA-Z0-9-]+$'
PROJECT_ID_PATTERN = '^(?:[0-9a-z]{12}|(?=[a-zA-Z0-9-]*[a-zA-Z-])[a-zA-Z0-9-]{16,128})$'


def re2_project_id_pattern():
    """RE2 has no lookahead. Match a 12-character lowercase ID, or at least 16
    ID characters including a letter or hyphen, by enumerating where the first
    letter or hyphen occurs. The schema's maxLength still bounds the total."""
    legacy = ['[0-9]{%d}[a-zA-Z-][a-zA-Z0-9-]{%d,}' % (digits, 15 - digits) for digits in range(16)]
    legacy.append('[0-9]{16,}[a-zA-Z-][a-zA-Z0-9-]*')
    return '^(?:[0-9a-z]{12}|' + '|'.join(legacy) + ')$'


PROJECT_ID_PATTERNS = {
    LEGACY_PROJECT_ID_PATTERN: '^[a-zA-Z0-9-]*[a-zA-Z-][a-zA-Z0-9-]*$',
    PROJECT_ID_PATTERN: re2_project_id_pattern(),
}


def project_id_patterns(value):
    if isinstance(value, dict):
        if value.get('pattern') in PROJECT_ID_PATTERNS:
            value['pattern'] = PROJECT_ID_PATTERNS[value['pattern']]
        for child in value.values():
            project_id_patterns(child)
    elif isinstance(value, list):
        for child in value:
            project_id_patterns(child)


def prepare(spec):
    result = copy.deepcopy(spec)
    project_id_patterns(result)
    # The Terraform assertion model has a string expected value. Keep that
    # branch explicit instead of relying on codegen skipping the whole union
    # and retaining stale generated attributes from a previous run.
    check_input = result.get('components', {}).get('schemas', {}).get('CheckInput', {})
    assertions = check_input.get('properties', {}).get('assertions', {})
    items = assertions.get('items', {})
    branches = items.get('oneOf', [])
    if branches:
        supported = [branch for branch in branches
                     if branch.get('properties', {}).get('expected', {}).get('type') == 'string']
        if len(branches) != 2 or len(supported) != 1 or not any(
                branch.get('properties', {}).get('type', {}).get('enum') == ['STATUS_CODE']
                for branch in branches):
            raise ValueError('Unsupported CheckInput assertion union')
        assertions['items'] = supported[0]
    for path in result['paths'].values():
        for operation in path.values():
            if not isinstance(operation, dict):
                continue
            for status, response in operation.get('responses', {}).items():
                media = response.get('content', {}).get('application/json', {})
                schema = media.get('schema', {})
                branches = schema.get('anyOf', [])
                if not branches:
                    continue
                flags = [b.get('properties', {}).get('success', {}).get('enum') for b in branches]
                if (status != '200' or set(schema) != {'anyOf'} or len(branches) != 2
                        or [True] not in flags or [False] not in flags
                        or any(b.get('type') != 'object' or 'success' not in b.get('required', [])
                               or 'result' not in b.get('properties', {}) for b in branches)):
                    raise ValueError(f"Unsupported response union in {operation.get('operationId')}")
                media['schema'] = branches[flags.index([True])]
    return result


if __name__ == '__main__':
    with open(sys.argv[1]) as source:
        spec = prepare(json.load(source))
    with open(sys.argv[2], 'w') as output:
        json.dump(spec, output, indent=2)
        output.write('\n')
