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
            arguments={'a': 1, 'b': 'x'},
        )
        k2 = self.cache.compute_key(
            component_name='c',
            image='python:3.11',
            full_command=['python', '-c', 'print(1)'],
            arguments={'b': 'x', 'a': 1},
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
            'out_art': dsl.Artifact(
                name='out_art', uri=artifact_path, metadata={'k': 'v'}
            ),
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

    def test_roundtrip_artifact_list_homogeneous(self):
        outputs = {
            'out_list': [
                dsl.Artifact(
                    name='a1', uri='gs://bucket/a1', metadata={'k1': 'v1'}
                ),
                dsl.Artifact(
                    name='a2', uri='gs://bucket/a2', metadata={'k2': 'v2'}
                ),
            ]
        }
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertIsNotNone(got)
        self.assertEqual(len(got['out_list']), 2)
        self.assertIsInstance(got['out_list'][0], dsl.Artifact)
        self.assertEqual(got['out_list'][0].uri, 'gs://bucket/a1')
        self.assertEqual(got['out_list'][0].metadata, {'k1': 'v1'})
        self.assertIsInstance(got['out_list'][1], dsl.Artifact)
        self.assertEqual(got['out_list'][1].uri, 'gs://bucket/a2')
        self.assertEqual(got['out_list'][1].metadata, {'k2': 'v2'})

    def test_roundtrip_artifact_list_leading_none(self):
        outputs = {
            'out_list': [
                None,
                dsl.Artifact(
                    name='a1', uri='gs://bucket/a1', metadata={'k1': 'v1'}
                ),
            ]
        }
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertIsNotNone(got)
        self.assertEqual(len(got['out_list']), 2)
        self.assertIsNone(got['out_list'][0])
        self.assertIsInstance(got['out_list'][1], dsl.Artifact)
        self.assertEqual(got['out_list'][1].uri, 'gs://bucket/a1')

    def test_roundtrip_artifact_list_trailing_none(self):
        outputs = {
            'out_list': [
                dsl.Artifact(
                    name='a1', uri='gs://bucket/a1', metadata={'k1': 'v1'}
                ),
                None,
            ]
        }
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertIsNotNone(got)
        self.assertEqual(len(got['out_list']), 2)
        self.assertIsInstance(got['out_list'][0], dsl.Artifact)
        self.assertEqual(got['out_list'][0].uri, 'gs://bucket/a1')
        self.assertIsNone(got['out_list'][1])

    def test_roundtrip_artifact_list_mixed_non_artifact(self):
        outputs = {
            'out_list': [
                'plain_string',
                dsl.Artifact(name='a1', uri='gs://bucket/a1'),
            ]
        }
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertIsNotNone(got)
        self.assertEqual(len(got['out_list']), 2)
        self.assertEqual(got['out_list'][0], 'plain_string')
        self.assertIsInstance(got['out_list'][1], dsl.Artifact)
        self.assertEqual(got['out_list'][1].uri, 'gs://bucket/a1')

    def test_roundtrip_plain_list_no_marker(self):
        outputs = {'out_list': [1, 'two', 3.0]}
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertEqual(got, outputs)
        entry_path = self.cache._entry_path('k')
        with open(entry_path) as f:
            raw_content = f.read()
        self.assertNotIn(cache._ARTIFACT_LIST_MARKER, raw_content)

    def test_roundtrip_empty_list(self):
        outputs = {'out_list': []}
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertEqual(got, outputs)
        entry_path = self.cache._entry_path('k')
        with open(entry_path) as f:
            raw_content = f.read()
        self.assertNotIn(cache._ARTIFACT_LIST_MARKER, raw_content)

    def test_deserialize_marked_entry_heterogeneous(self):
        art = dsl.Artifact(name='a', uri='gs://bucket/a', metadata={'m': 1})
        art_dict = cache.LocalCache._serialize_artifact(art)
        serialized_entry = {
            cache._ARTIFACT_LIST_MARKER: True,
            'artifacts': [art_dict, None, 'plain'],
        }
        deserialized = cache.LocalCache._deserialize_output(serialized_entry)
        self.assertEqual(len(deserialized), 3)
        self.assertIsInstance(deserialized[0], dsl.Artifact)
        self.assertEqual(deserialized[0].name, 'a')
        self.assertEqual(deserialized[0].uri, 'gs://bucket/a')
        self.assertEqual(deserialized[0].metadata, {'m': 1})
        self.assertIsNone(deserialized[1])
        self.assertEqual(deserialized[2], 'plain')

    def test_deserialize_backward_compatibility_old_format(self):
        art1 = dsl.Artifact(name='a1', uri='gs://bucket/a1')
        art2 = dsl.Artifact(name='a2', uri='gs://bucket/a2')
        old_format_entry = {
            cache._ARTIFACT_LIST_MARKER: True,
            'artifacts': [
                cache.LocalCache._serialize_artifact(art1),
                cache.LocalCache._serialize_artifact(art2),
            ],
        }
        deserialized = cache.LocalCache._deserialize_output(old_format_entry)
        self.assertEqual(len(deserialized), 2)
        self.assertIsInstance(deserialized[0], dsl.Artifact)
        self.assertEqual(deserialized[0].uri, 'gs://bucket/a1')
        self.assertIsInstance(deserialized[1], dsl.Artifact)
        self.assertEqual(deserialized[1].uri, 'gs://bucket/a2')

    def test_roundtrip_subclass_preservation(self):
        dataset = dsl.Dataset(
            name='ds', uri='gs://bucket/ds', metadata={'rows': 100}
        )
        model = dsl.Model(
            name='mdl', uri='gs://bucket/mdl', metadata={'acc': 0.95}
        )
        outputs = {
            'datasets': [dataset, None],
            'model': model,
        }
        self.cache.put('k', outputs)
        got = self.cache.get('k')
        self.assertIsNotNone(got)
        self.assertIsInstance(got['datasets'][0], dsl.Dataset)
        self.assertEqual(got['datasets'][0].name, 'ds')
        self.assertEqual(got['datasets'][0].uri, 'gs://bucket/ds')
        self.assertEqual(got['datasets'][0].metadata, {'rows': 100})
        self.assertIsNone(got['datasets'][1])
        self.assertIsInstance(got['model'], dsl.Model)
        self.assertEqual(got['model'].name, 'mdl')
        self.assertEqual(got['model'].uri, 'gs://bucket/mdl')
        self.assertEqual(got['model'].metadata, {'acc': 0.95})

    def test_deserialize_marked_entry_with_plain_dict(self):
        art = dsl.Artifact(name='a', uri='gs://bucket/a')
        art_dict = cache.LocalCache._serialize_artifact(art)
        plain_dict = {'key': 'value', 'nested': [1, 2]}
        serialized_entry = {
            cache._ARTIFACT_LIST_MARKER: True,
            'artifacts': [art_dict, plain_dict],
        }
        deserialized = cache.LocalCache._deserialize_output(serialized_entry)
        self.assertEqual(len(deserialized), 2)
        self.assertIsInstance(deserialized[0], dsl.Artifact)
        self.assertEqual(deserialized[0].name, 'a')
        self.assertEqual(deserialized[0].uri, 'gs://bucket/a')
        self.assertIsInstance(deserialized[1], dict)
        self.assertEqual(deserialized[1], plain_dict)
        self.assertNotIn(cache._ARTIFACT_MARKER, deserialized[1])


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
