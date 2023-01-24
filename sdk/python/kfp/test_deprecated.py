import os
import sys
import unittest
from unittest import mock


class TestImportFromDeprecated(unittest.TestCase):

    def tearDown(self):
        """Remove deprecated modules from sys.modules.

        Cleans up relevant global state so that tests are independent.
        """

        modules = [
            m for m in sys.modules.keys() if m.startswith('kfp.deprecated')
        ]
        for m in modules:
            del sys.modules[m]

    @mock.patch.dict(os.environ, {}, clear=True)
    def test_import_module_fails(self):

        with self.assertRaisesRegex(ImportError,
                                    "cannot import from 'kfp.deprecated'"):
            pass

    @mock.patch.dict(os.environ, {}, clear=True)
    def test_import_object_fails(self):
        with self.assertRaisesRegex(ImportError,
                                    "cannot import from 'kfp.deprecated'"):
            pass

    @mock.patch.dict(os.environ, {'ENV': 'TEST'}, clear=True)
    def test_import_module_succeeds_with_env_var(self):
        pass

    @mock.patch.dict(os.environ, {'ENV': 'TEST'}, clear=True)
    def test_import_object_succeeds_with_env_var(self):
        pass
