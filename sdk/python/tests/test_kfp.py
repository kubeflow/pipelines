import unittest


class KfpTestCase(unittest.TestCase):

    def test_kfp_version(self):
        import kfp
        self.assertTrue(len(kfp.__version__) > 0)
