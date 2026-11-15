"""Keep the pinned TensorBoard base's zlib package patched."""

from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[3]


class TensorboardZlibTest(unittest.TestCase):

    def test_refreshes_zlib_before_dropping_privileges(self):
        content = (ROOT / 'backend/Dockerfile.tensorboard').read_text()
        self.assertIn('apt-get install -y --no-install-recommends zlib1g',
                      content)
        self.assertIn('1:1.3.dfsg+really1.3.1-1+deb13u1', content)
        self.assertLess(
            content.index('apt-get'), content.index('USER 1000:1000'))
        self.assertIn('rm -rf /var/lib/apt/lists/*', content)


if __name__ == '__main__':
    unittest.main()
