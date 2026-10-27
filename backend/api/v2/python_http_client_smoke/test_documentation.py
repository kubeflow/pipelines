"""Formatting regressions for generated Python client documentation."""

import unittest
from pathlib import Path


class TestGeneratedDocumentation(unittest.TestCase):
    """Check generated Markdown against repository formatting hooks."""

    def test_markdown_has_canonical_whitespace(self) -> None:
        """Require clean line endings and exactly one final newline."""
        root = Path(__file__).resolve().parent.parent / 'python_http_client'
        documents = sorted(root.rglob('*.md'))
        self.assertTrue(
            documents, 'Generated Markdown documentation is missing'
        )
        for document in documents:
            with self.subTest(document=document.name):
                text = document.read_text()
                self.assertTrue(text.endswith('\n'), 'Missing final newline')
                self.assertFalse(text.endswith('\n\n'), 'Blank lines at EOF')
                self.assertTrue(
                    all(line == line.rstrip() for line in text.splitlines()),
                    'Trailing whitespace',
                )
