# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0.
"""Direct tests of Python compatibility generation."""

import importlib.util
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest

SPEC = importlib.util.spec_from_file_location(
    'generate_python_compat',
    Path(__file__).with_name('generate_python_compat.py'))
GENERATOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GENERATOR)


class TestGeneratePythonCompat(unittest.TestCase):

    def make_client(self, root, source):
        (root / 'models').mkdir()
        (root / '__init__.py').write_text('# package imports\n')
        (root / 'models/__init__.py').write_text('# model imports\n')
        (root / 'models/v2_widget_v2.py').write_text(source)

    def test_aliases_replace_only_first_version_and_are_idempotent(self):
        with TemporaryDirectory() as directory:
            root = Path(directory)
            self.make_client(root, 'class V2WidgetV2:\n    pass\n')
            GENERATOR.generate(root)
            alias = root / 'models/v2beta1_widget_v2.py'
            expected = ('from kfp.server_api.models.v2_widget_v2 import '
                        'V2WidgetV2 as V2beta1WidgetV2\n')
            self.assertIn(expected, alias.read_text())
            self.assertFalse(
                (root / 'models/v2beta1_widget_v2beta1.py').exists())
            before = {p: p.read_bytes() for p in root.rglob('*.py')}
            GENERATOR.generate(root)
            self.assertEqual(before,
                             {p: p.read_bytes() for p in root.rglob('*.py')})
            for initializer in (root / '__init__.py',
                                root / 'models/__init__.py'):
                self.assertEqual(initializer.read_text().count(expected), 1)

    def test_invalid_layout_does_not_partially_write_aliases(self):
        for source in ('class Other: pass\n',
                       'class V2First: pass\nclass V2Second: pass\n'):
            with self.subTest(source=source), TemporaryDirectory() as directory:
                root = Path(directory)
                self.make_client(root, source)
                before = {p: p.read_bytes() for p in root.rglob('*.py')}
                with self.assertRaisesRegex(ValueError,
                                            'regenerate the v2 client'):
                    GENERATOR.generate(root)
                self.assertEqual(before,
                                 {p: p.read_bytes() for p in root.rglob('*.py')})

    def test_missing_models_explains_prerequisite(self):
        with TemporaryDirectory() as directory:
            with self.assertRaisesRegex(ValueError,
                                        'generate the v2 Python client'):
                GENERATOR.generate(Path(directory))


if __name__ == '__main__':
    unittest.main()
