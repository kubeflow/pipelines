from kfp.gcp.core.base_op import BaseOp

import unittest

class BaseOpTest(unittest.TestCase):
    def test_init(self):
        op = BaseOp("test/dir")
        self.assertEqual(op.staging_dir, "test/dir")