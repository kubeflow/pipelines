# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""One deliberately narrow expression helper for artifact workflow tests.

Supports string/bool references, literals, equality, boolean operators
and injected status functions. Unsupported syntax or coercion fails
loudly; this is not a runner and hosted CI remains required.
"""

import re

EXPRESSION = re.compile(r'\$\{\{(.*?)\}\}', re.DOTALL)
TOKEN = re.compile(
    r"\s+|(?P<literal>'(?:[^']|'')*')|"
    r'(?P<reference>(?:inputs|steps|github|env)(?:\.[\w-]+)+)|'
    r'(?P<word>true|false|null|always|cancelled|success|failure)\b|'
    r'(?P<operator>==|!=|&&|\|\||!|\(|\))')


def evaluate(expression, context, statuses=None):
    match = EXPRESSION.fullmatch(expression.strip())
    expression = match.group(1) if match else expression
    tokens = []
    position = 0
    while position < len(expression):
        match = TOKEN.match(expression, position)
        if not match:
            raise AssertionError(f'Unsupported expression: {expression}')
        position = match.end()
        if match.lastgroup:
            tokens.append((match.lastgroup, match.group()))
    index = 0

    def take(value):
        nonlocal index
        if index < len(tokens) and tokens[index][1] == value:
            index += 1
            return True
        return False

    def atom():
        nonlocal index
        if take('!'):
            return not atom()
        if take('('):
            value = disjunction()
            if not take(')'):
                raise AssertionError('Missing expression closing parenthesis')
            return value
        if index == len(tokens):
            raise AssertionError('Missing expression operand')
        kind, token = tokens[index]
        index += 1
        if kind == 'literal':
            return token[1:-1].replace("''", "'")
        if kind == 'reference':
            value = context
            for part in token.split('.'):
                value = value.get(part, '') if isinstance(value, dict) else ''
            if value is not None and type(value) not in (str, bool):
                raise AssertionError(f'Unsupported expression value: {token}')
            return value
        if kind == 'word':
            if token in ('true', 'false', 'null'):
                return {'true': True, 'false': False, 'null': None}[token]
            if not take('(') or not take(')') or token not in (statuses or {}):
                raise AssertionError(f'Unsupported status function: {token}')
            return statuses[token]()
        raise AssertionError(f'Unsupported expression operand: {token}')

    def equality():
        value = atom()
        while index < len(tokens) and tokens[index][1] in ('==', '!='):
            negate = take('!=')
            if not negate:
                take('==')
            other = atom()
            if type(value) is not type(other):
                raise AssertionError(
                    'Mixed-type Actions coercion is unsupported')
            # GitHub string equality ignores case; Python equality does not.
            equal = (
                value.casefold() == other.casefold()
                if isinstance(value, str) else value == other)
            value = not equal if negate else equal
        return value

    def conjunction():
        value = equality()
        while take('&&'):
            other = equality()
            value = other if value else value
        return value

    def disjunction():
        value = conjunction()
        while take('||'):
            other = conjunction()
            value = value if value else other
        return value

    value = disjunction()
    if index != len(tokens):
        raise AssertionError(f'Unsupported expression suffix: {tokens[index:]}')
    return value


def render(value, context, statuses=None):

    def substitute(match):
        result = evaluate(match.group(1), context, statuses)
        if result is None:
            return ''
        if isinstance(result, bool):
            return str(result).lower()
        return result

    return EXPRESSION.sub(substitute, str(value))
