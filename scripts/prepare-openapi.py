#!/usr/bin/env python3
"""Project success envelopes for codegen; never modify the pinned contract.

The framework generator cannot map HTTP 200 success/failure anyOf envelopes.
Terraform state describes successful results; the client still checks success
at runtime. Nested resource unions and nullable fields are left unchanged.
"""

import copy
import json
import sys


def prepare(spec):
    result = copy.deepcopy(spec)
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
