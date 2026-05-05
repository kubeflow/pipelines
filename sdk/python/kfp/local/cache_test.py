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
"""Tests for kfp.local.cache."""
import os
import tempfile
import threading
import unittest

from kfp import dsl
from kfp.local import cache


class LocalCacheKeyTest(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp(prefix='kfp-cache-test-')
        self.addCleanup(self._rmtree)
        self.cache = cache.LocalCache(cache_root=self.tmpdir)

    def _rmtree(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_same_inputs_same_key(self):
        k1 = self.cache.compute_key(
            component_name='c',
            image='python:3.11',
            full_command=['python', '-c', 'print(1)'],
            arguments={
                'a': 1,
                'b': 'x'
            },
        )
        k2 = self.cache.compute_key(
            component_name='c',
            image='python:3.11',
            full_command=['python', '-c', 'print(1)'],
            arguments={
                'b': 'x',
                'a': 1
            },
        )
        self.assertEqual(k1, k2)

    def test_different_args_different_key(self):
        k1 = self.cache.compute_key(
            component_name='c',
            image='python:3.11',
            full_command=['python'],
            arguments={'a': 1},
        )
        k2 = self.cache.compute_key(
            component_name='c',
            image='python:3.11',
            full_command=['python'],
            arguments={'a': 2},
        )
        self.assertNotEqual(k1, k2)

    def test_different_image_different_key(self):
        k1 = self.cache.compute_key(
            component_name='c',
            image='python:3.11',
            full_command=['python'],
            arguments={},
        )
        k2 = self.cache.compute_key(
            component_name='c',
            image='python:3.11-slim',
            full_command=['python'],
            arguments={},
        )
        self.assertNotEqual(k1, k2)

    def test_different_command_different_key(self):
        k1 = self.cache.compute_key(
            component_name='c',
            image='python:3.11',
            full_command=['python', 'one.py'],
            arguments={},
        )
        k2 = self.cache.compute_key(
            component_name='c',
            image='python:3.11',
            full_command=['python', 'two.py'],
            arguments={},
        )
        self.assertNotEqual(k1, k2)

    def test_custom_cache_key_overrides(self):
        # If caller provides a custom key, args/image should not matter.
        k1 = self.cache.compute_key(
            component_name='c',
            image='python:3.11',
            full_command=['python', 'one.py'],
            arguments={'a': 1},
            custom_cache_key='MY_KEY',
        )
        k2 = self.cache.compute_key(
            component_name='c',
            image='python:3.11-slim',
            full_command=['python', 'two.py'],
            arguments={'a': 2},
            custom_cache_key='MY_KEY',
        )
        self.assertEqual(k1, k2)

    def test_custom_cache_key_still_scoped_by_component(self):
        k1 = self.cache.compute_key(
            component_name='c1',
            image='img',
            full_command=[],
            arguments={},
            custom_cache_key='K',
        )
        k2 = self.cache.compute_key(
            component_name='c2',
            image='img',
            full_command=[],
            arguments={},
            custom_cache_key='K',
        )
        self.assertNotEqual(k1, k2)

    def test_artifact_argument_in_key(self):
        a1 = dsl.Artifact(name='a', uri='/tmp/a')
        a2 = dsl.Artifact(name='a', uri='/tmp/b')
        k1 = self.cache.compute_key(
            component_name='c',
            image='img',
            full_command=[],
            arguments={'art': a1},
        )
        k2 = self.cache.compute_key(
            component_name='c',
            image='img',
            full_command=[],
            arguments={'art': a2},
        )
        self.assertNotEqual(k1, k2)


class LocalCachePutGetTest(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp(prefix='kfp-cache-test-')
        self.addCleanup(self._rmtree)
        self.cache = cache.LocalCache(cache_root=self.tmpdir)

    def _rmtree(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_miss_returns_none(self):
        self.assertIsNone(self.cache.get('unknown'))

    def test_roundtrip_parameters(self):
        outputs = {'out_str': 'hello', 'out_int': 42, 'out_list': [1, 2, 3]}
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertEqual(got, outputs)

    def test_roundtrip_artifact(self):
        artifact_path = os.path.join(self.tmpdir, 'art.txt')
        with open(artifact_path, 'w') as f:
            f.write('hi')
        outputs = {
            'out_art':
                dsl.Artifact(
                    name='out_art', uri=artifact_path, metadata={'k': 'v'}),
        }
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertIsNotNone(got)
        self.assertIsInstance(got['out_art'], dsl.Artifact)
        self.assertEqual(got['out_art'].uri, artifact_path)
        self.assertEqual(got['out_art'].metadata, {'k': 'v'})

    def test_roundtrip_artifact_missing_file_is_miss(self):
        artifact_path = os.path.join(self.tmpdir, 'gone.txt')
        with open(artifact_path, 'w') as f:
            f.write('hi')
        outputs = {'out_art': dsl.Artifact(name='a', uri=artifact_path)}
        self.cache.put('k', outputs)
        # Delete the file — cache should now miss.
        os.remove(artifact_path)
        self.assertIsNone(self.cache.get('k'))

    def test_remote_artifact_uri_is_hit(self):
        outputs = {'out_art': dsl.Artifact(name='a', uri='gs://bucket/a')}
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertIsNotNone(got)
        self.assertEqual(got['out_art'].uri, 'gs://bucket/a')

    def test_atomic_write_no_partial(self):
        # Concurrent writers should not corrupt entries.
        def writer(i):
            self.cache.put(f'k{i}', {'i': i})

        threads = [
            threading.Thread(target=writer, args=(i,)) for i in range(20)
        ]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        for i in range(20):
            self.assertEqual(self.cache.get(f'k{i}'), {'i': i})


class LocalCacheSingletonTest(unittest.TestCase):
    """Tests for the module-level singleton helpers that gate caching behind
    the LocalExecutionConfig."""

    def setUp(self):
        cache.reset_local_cache_singleton()
        self.tmpdir = tempfile.mkdtemp(prefix='kfp-cache-test-')
        self.addCleanup(self._cleanup)

    def _cleanup(self):
        import shutil

        from kfp.local import config as local_config
        shutil.rmtree(self.tmpdir, ignore_errors=True)
        local_config.LocalExecutionConfig.instance = None
        cache.reset_local_cache_singleton()

    def test_none_when_no_config(self):
        from kfp.local import config as local_config
        local_config.LocalExecutionConfig.instance = None
        self.assertIsNone(cache.get_local_cache())

    def test_none_when_caching_disabled(self):
        from kfp.local import config as local_config
        local_config.init(
            runner=local_config.SubprocessRunner(use_venv=False),
            pipeline_root=self.tmpdir,
            enable_caching=False,
        )
        self.assertIsNone(cache.get_local_cache())

    def test_singleton_when_enabled(self):
        from kfp.local import config as local_config
        local_config.init(
            runner=local_config.SubprocessRunner(use_venv=False),
            pipeline_root=self.tmpdir,
            enable_caching=True,
        )
        c1 = cache.get_local_cache()
        c2 = cache.get_local_cache()
        self.assertIsNotNone(c1)
        self.assertIs(c1, c2)
        # Defaults to pipeline_root/.kfp_cache
        self.assertTrue(c1.cache_root.endswith('.kfp_cache'), msg=c1.cache_root)

    def test_custom_cache_root_respected(self):
        from kfp.local import config as local_config
        custom = os.path.join(self.tmpdir, 'custom-cache')
        local_config.init(
            runner=local_config.SubprocessRunner(use_venv=False),
            pipeline_root=self.tmpdir,
            enable_caching=True,
            cache_root=custom,
        )
        c = cache.get_local_cache()
        self.assertIsNotNone(c)
        self.assertEqual(c.cache_root, custom)


if __name__ == '__main__':
    unittest.main()
