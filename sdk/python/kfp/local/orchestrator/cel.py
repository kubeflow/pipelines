# Copyright 2024 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Minimal CEL subset evaluator for kfp-emitted trigger_policy.condition
expressions.

The Kubeflow Pipelines compiler emits a narrow subset of CEL for the
``triggerPolicy.condition`` field.  This module implements exactly that subset,
without any use of Python's ``eval``:

    * parameter references: ``inputs.parameter_values['name']``
    * boolean literals, numeric literals, single/double-quoted string literals
    * unary ``!``
    * binary ``&&``, ``||``
    * binary ``==``, ``!=``, ``<``, ``<=``, ``>``, ``>=``
    * casts / funcs: ``int(expr)``, ``string(expr)``, ``bool(expr)``, ``len(expr)``
    * parenthesised sub-expressions

Inputs are resolved through a typed context object rather than by substituting
strings into the expression.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Callable, Dict, List, Optional, Tuple


class CELError(ValueError):
    """Raised for any parse or evaluation error in a CEL expression."""


# ---------------------------------------------------------------------------
# Tokenizer
# ---------------------------------------------------------------------------

_SYMBOL_TOKENS: Tuple[Tuple[str, str], ...] = (
    ('&&', 'ANDAND'),
    ('||', 'OROR'),
    ('==', 'EQEQ'),
    ('!=', 'NEQ'),
    ('<=', 'LE'),
    ('>=', 'GE'),
    ('<', 'LT'),
    ('>', 'GT'),
    ('!', 'BANG'),
    ('(', 'LPAREN'),
    (')', 'RPAREN'),
    ('[', 'LBRACK'),
    (']', 'RBRACK'),
    ('.', 'DOT'),
    (',', 'COMMA'),
)


@dataclass
class _Token:
    kind: str
    value: Any
    pos: int


def _tokenize(source: str) -> List[_Token]:
    tokens: List[_Token] = []
    i = 0
    n = len(source)
    while i < n:
        c = source[i]
        if c.isspace():
            i += 1
            continue

        # string literal
        if c == "'" or c == '"':
            quote = c
            j = i + 1
            chars: List[str] = []
            while j < n and source[j] != quote:
                if source[j] == '\\' and j + 1 < n:
                    esc = source[j + 1]
                    chars.append({
                        'n': '\n',
                        't': '\t',
                        'r': '\r',
                        "'": "'",
                        '"': '"',
                        '\\': '\\',
                    }.get(esc, esc))
                    j += 2
                else:
                    chars.append(source[j])
                    j += 1
            if j >= n:
                raise CELError(
                    f'Unterminated string literal at position {i}: {source!r}')
            tokens.append(_Token('STRING', ''.join(chars), i))
            i = j + 1
            continue

        # number literal
        if c.isdigit() or (c == '-' and i + 1 < n and source[i + 1].isdigit()):
            j = i + 1
            saw_dot = False
            while j < n and (source[j].isdigit() or
                             (source[j] == '.' and not saw_dot)):
                if source[j] == '.':
                    saw_dot = True
                j += 1
            raw = source[i:j]
            value: Any
            try:
                value = float(raw) if saw_dot else int(raw)
            except ValueError as e:
                raise CELError(f'Invalid numeric literal {raw!r}') from e
            tokens.append(_Token('NUMBER', value, i))
            i = j
            continue

        # identifier / keyword
        if c.isalpha() or c == '_':
            j = i + 1
            while j < n and (source[j].isalnum() or source[j] == '_'):
                j += 1
            ident = source[i:j]
            if ident == 'true':
                tokens.append(_Token('BOOL', True, i))
            elif ident == 'false':
                tokens.append(_Token('BOOL', False, i))
            elif ident == 'null':
                tokens.append(_Token('NULL', None, i))
            else:
                tokens.append(_Token('IDENT', ident, i))
            i = j
            continue

        # symbols (longest-match first)
        matched = False
        for sym, kind in _SYMBOL_TOKENS:
            if source.startswith(sym, i):
                tokens.append(_Token(kind, sym, i))
                i += len(sym)
                matched = True
                break
        if matched:
            continue

        raise CELError(f'Unexpected character {c!r} at position {i}')

    tokens.append(_Token('EOF', None, len(source)))
    return tokens


# ---------------------------------------------------------------------------
# AST
# ---------------------------------------------------------------------------


@dataclass
class _Node:
    kind: str
    # each kind uses the fields it cares about, others stay None
    value: Any = None
    op: Optional[str] = None
    left: Optional['_Node'] = None
    right: Optional['_Node'] = None
    target: Optional['_Node'] = None
    name: Optional[str] = None
    index: Optional['_Node'] = None
    args: Optional[List['_Node']] = None


# ---------------------------------------------------------------------------
# Parser (recursive descent)
# ---------------------------------------------------------------------------


class _Parser:

    def __init__(self, tokens: List[_Token]):
        self.tokens = tokens
        self.pos = 0

    def _peek(self, offset: int = 0) -> _Token:
        return self.tokens[self.pos + offset]

    def _eat(self, kind: str) -> _Token:
        tok = self.tokens[self.pos]
        if tok.kind != kind:
            raise CELError(
                f'Expected {kind} at position {tok.pos}, got {tok.kind} ({tok.value!r})'
            )
        self.pos += 1
        return tok

    def _accept(self, *kinds: str) -> Optional[_Token]:
        tok = self.tokens[self.pos]
        if tok.kind in kinds:
            self.pos += 1
            return tok
        return None

    def parse(self) -> _Node:
        node = self._or()
        if self._peek().kind != 'EOF':
            tok = self._peek()
            raise CELError(
                f'Unexpected trailing token {tok.kind} ({tok.value!r}) at position {tok.pos}'
            )
        return node

    def _or(self) -> _Node:
        node = self._and()
        while self._accept('OROR'):
            node = _Node('binop', op='||', left=node, right=self._and())
        return node

    def _and(self) -> _Node:
        node = self._eq()
        while self._accept('ANDAND'):
            node = _Node('binop', op='&&', left=node, right=self._eq())
        return node

    def _eq(self) -> _Node:
        node = self._rel()
        while True:
            tok = self._accept('EQEQ', 'NEQ')
            if not tok:
                return node
            node = _Node('binop', op=tok.value, left=node, right=self._rel())

    def _rel(self) -> _Node:
        node = self._unary()
        while True:
            tok = self._accept('LT', 'LE', 'GT', 'GE')
            if not tok:
                return node
            node = _Node('binop', op=tok.value, left=node, right=self._unary())

    def _unary(self) -> _Node:
        if self._accept('BANG'):
            return _Node('unary', op='!', target=self._unary())
        return self._postfix()

    def _postfix(self) -> _Node:
        node = self._primary()
        while True:
            if self._accept('DOT'):
                name = self._eat('IDENT').value
                node = _Node('attr', target=node, name=name)
            elif self._accept('LBRACK'):
                idx = self._or()
                self._eat('RBRACK')
                node = _Node('index', target=node, index=idx)
            elif self._peek().kind == 'LPAREN' and node.kind == 'ident':
                self._eat('LPAREN')
                args: List[_Node] = []
                if self._peek().kind != 'RPAREN':
                    args.append(self._or())
                    while self._accept('COMMA'):
                        args.append(self._or())
                self._eat('RPAREN')
                node = _Node('call', name=node.value, args=args)
            else:
                return node

    def _primary(self) -> _Node:
        tok = self._peek()
        if tok.kind == 'NUMBER':
            self.pos += 1
            return _Node('literal', value=tok.value)
        if tok.kind == 'STRING':
            self.pos += 1
            return _Node('literal', value=tok.value)
        if tok.kind == 'BOOL':
            self.pos += 1
            return _Node('literal', value=tok.value)
        if tok.kind == 'NULL':
            self.pos += 1
            return _Node('literal', value=None)
        if tok.kind == 'IDENT':
            self.pos += 1
            return _Node('ident', value=tok.value)
        if tok.kind == 'LPAREN':
            self.pos += 1
            inner = self._or()
            self._eat('RPAREN')
            return inner
        raise CELError(
            f'Unexpected token {tok.kind} ({tok.value!r}) at position {tok.pos}'
        )


# ---------------------------------------------------------------------------
# Evaluation
# ---------------------------------------------------------------------------


class _ParameterValues:
    """Indexable view over a task's already-resolved parameter arguments."""

    def __init__(self, values: Dict[str, Any]):
        self._values = values

    def __getitem__(self, key: str) -> Any:
        if key not in self._values:
            raise CELError(
                f"parameter_values[{key!r}] is not bound in this task's inputs")
        return self._values[key]


class _InputsNamespace:
    """Provides the ``inputs.parameter_values`` and ``inputs.artifacts`` entry
    points expected by kfp-emitted CEL expressions."""

    def __init__(self,
                 parameter_values: Dict[str, Any],
                 artifacts: Optional[Dict[str, Any]] = None):
        self.parameter_values = _ParameterValues(parameter_values)
        # Expose artifacts as a plain indexable dict; the compiler does not
        # currently emit artifact-valued conditions but we keep the namespace
        # symmetric.
        self.artifacts = artifacts or {}


_BUILTIN_FUNCS: Dict[str, Callable[[Any], Any]] = {
    'int': lambda v: int(v),
    'string': lambda v: str(v),
    'bool': lambda v: bool(v),
    'len': lambda v: len(v),
}


def _eval(node: _Node, env: Dict[str, Any]) -> Any:
    kind = node.kind
    if kind == 'literal':
        return node.value
    if kind == 'ident':
        name = node.value
        if name not in env:
            raise CELError(f'Undefined identifier {name!r}')
        return env[name]
    if kind == 'attr':
        target = _eval(node.target, env)
        try:
            return getattr(target, node.name)
        except AttributeError:
            if isinstance(target, dict) and node.name in target:
                return target[node.name]
            raise CELError(
                f'Value {type(target).__name__} has no attribute {node.name!r}')
    if kind == 'index':
        target = _eval(node.target, env)
        key = _eval(node.index, env)
        try:
            return target[key]
        except (KeyError, IndexError, TypeError) as e:
            raise CELError(f'Cannot index {target!r} with {key!r}: {e}') from e
    if kind == 'call':
        func = _BUILTIN_FUNCS.get(node.name)
        if func is None:
            raise CELError(f'Unsupported function {node.name!r}')
        if len(node.args) != 1:
            raise CELError(
                f'Function {node.name!r} takes 1 argument, got {len(node.args)}'
            )
        arg = _eval(node.args[0], env)
        try:
            return func(arg)
        except Exception as e:
            raise CELError(f'{node.name}({arg!r}) failed: {e}') from e
    if kind == 'unary':
        val = _eval(node.target, env)
        if node.op == '!':
            return not bool(val)
        raise CELError(f'Unknown unary operator {node.op!r}')
    if kind == 'binop':
        op = node.op
        # short-circuit boolean ops
        if op == '&&':
            return bool(_eval(node.left, env)) and bool(_eval(node.right, env))
        if op == '||':
            return bool(_eval(node.left, env)) or bool(_eval(node.right, env))
        left = _eval(node.left, env)
        right = _eval(node.right, env)
        if op == '==':
            return left == right
        if op == '!=':
            return left != right
        if op == '<':
            return left < right
        if op == '<=':
            return left <= right
        if op == '>':
            return left > right
        if op == '>=':
            return left >= right
        raise CELError(f'Unknown binary operator {op!r}')
    raise CELError(f'Unknown node kind {kind!r}')


def evaluate(
    expression: str,
    parameter_values: Dict[str, Any],
    artifacts: Optional[Dict[str, Any]] = None,
) -> bool:
    """Evaluate a compiler-emitted CEL condition.

    Args:
        expression: The ``triggerPolicy.condition`` string.
        parameter_values: Mapping from parameter name
            (e.g. ``'pipelinechannel--flip-coin-Output'``) to the resolved
            Python value for that parameter.
        artifacts: Optional mapping for ``inputs.artifacts[...]`` lookups.

    Returns:
        The boolean result of the expression.

    Raises:
        CELError: If the expression cannot be parsed or evaluated.
    """
    if expression is None or not expression.strip():
        return True
    tokens = _tokenize(expression)
    ast = _Parser(tokens).parse()
    env = {'inputs': _InputsNamespace(parameter_values, artifacts)}
    return bool(_eval(ast, env))
