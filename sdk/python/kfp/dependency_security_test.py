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

from pathlib import Path
import pickle
import subprocess
import sys
import tempfile
import unittest
import warnings

if sys.version_info >= (3, 10):
    from pylint.lint import caching
    from pylint.utils import LinterStats


class _CachePayload:

    def __init__(self, marker):
        self.marker = marker

    def __reduce__(self):
        # A harmless filesystem marker proves whether deserialization ran code.
        return Path.touch, (self.marker,)


@unittest.skipIf(sys.version_info < (3, 10),
                 'Patched pylint is a Python 3.10+ development tool.')
class DependencySecurityTest(unittest.TestCase):

    def test_pylint_does_not_execute_pickle_cache(self):
        """GHSA-22qq-2fwx-7p67: old cache files must never execute code."""
        with tempfile.TemporaryDirectory() as directory:
            cache = Path(directory)
            marker = cache / 'executed'
            cache_file = caching._get_pdata_path(Path('sample'), 1, cache)
            cache_file.write_bytes(pickle.dumps(_CachePayload(marker)))
            with warnings.catch_warnings():
                warnings.simplefilter('ignore', UserWarning)
                self.assertIsNone(caching.load_results('sample', cache))
            self.assertFalse(marker.exists(),
                             'loading the cache executed pickle')

    def test_pylint_accepts_repository_configuration(self):
        root = Path(__file__).resolve().parents[3]
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / 'sample.py'
            source.write_text('VALUE = 1\n')
            result = subprocess.run([
                sys.executable, '-m', 'pylint', '--rcfile',
                str(root / '.pylintrc'), '--disable=all', '--enable=E,F',
                '--reports=no', '--score=no', '--persistent=no',
                str(source)
            ],
                                    capture_output=True,
                                    text=True,
                                    check=False)
            self.assertNotIn('unrecognized-option', result.stdout)
            self.assertNotIn('UserWarning', result.stderr)
            self.assertEqual(result.returncode, 0,
                             result.stdout + result.stderr)

    def test_pylint_stats_still_round_trip(self):
        with tempfile.TemporaryDirectory() as directory:
            cache = Path(directory)
            stats = LinterStats()
            stats.statement = 7
            caching.save_results(stats, 'sample', cache)
            restored = caching.load_results('sample', cache)
            self.assertIsNotNone(restored)
            self.assertEqual(restored.statement, 7)


if __name__ == '__main__':
    unittest.main()
