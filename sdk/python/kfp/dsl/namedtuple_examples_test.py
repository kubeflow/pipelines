# Copyright 2026 The Kubeflow Authors
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
"""Checks that user-facing examples avoid keyword-form NamedTuple.

NamedTuple('outputs', a=int) was removed in Python 3.15. CI runs older
Pythons where it still works, so the examples are checked statically.
"""

import ast
import pathlib
import re
import unittest
from typing import Iterator, List, Tuple

_REPO_ROOT = pathlib.Path(__file__).resolve().parents[4]
_DOCS_DIR = _REPO_ROOT / 'docs'
_SAMPLES_DIR = _REPO_ROOT / 'samples'
_PYTHON_BLOCK = re.compile(r'```python\n(.*?)```', re.DOTALL)


def _keyword_namedtuple_lines(source: str) -> List[int]:
    tree = ast.parse(source)
    return [
        node.lineno
        for node in ast.walk(tree)
        if isinstance(node, ast.Call)
        and getattr(node.func, 'id', getattr(node.func, 'attr', None))
        == 'NamedTuple'
        and node.keywords
    ]


def _python_sources() -> Iterator[Tuple[str, str]]:
    for path in sorted(_SAMPLES_DIR.rglob('*.py')):
        yield str(path.relative_to(_REPO_ROOT)), path.read_text()
    for path in sorted(_DOCS_DIR.rglob('*.md')):
        relative = path.relative_to(_REPO_ROOT)
        for index, block in enumerate(_PYTHON_BLOCK.findall(path.read_text())):
            yield f'{relative} (python block {index + 1})', block


@unittest.skipUnless(
    _DOCS_DIR.is_dir() and _SAMPLES_DIR.is_dir(),
    'docs/ and samples/ are only available in a repository checkout',
)
class NamedTupleExamplesTest(unittest.TestCase):
    maxDiff = None

    def test_examples_use_list_form_namedtuple(self):
        offenders = []
        for name, source in _python_sources():
            try:
                lines = _keyword_namedtuple_lines(source)
            except SyntaxError:
                # Docs blocks may be partial snippets that do not parse.
                continue
            offenders.extend(f'{name}:{line}' for line in lines)
        self.assertEqual(
            offenders,
            [],
            "Use NamedTuple('name', [('field', type)]); the keyword form "
            'raises TypeError on Python 3.15.',
        )


if __name__ == '__main__':
    unittest.main()
