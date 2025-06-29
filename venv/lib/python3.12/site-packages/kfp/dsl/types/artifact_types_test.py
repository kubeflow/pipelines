# Copyright 2021 The Kubeflow Authors
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
"""Tests for kfp.components.types.artifact_types."""

import contextlib
import json
import os
import unittest

from absl.testing import parameterized
from kfp import dsl
from kfp.dsl.types import artifact_types


class ArtifactsTest(unittest.TestCase):

    def test_complex_metrics(self):
        metrics = dsl.ClassificationMetrics()
        metrics.log_roc_data_point(threshold=0.1, tpr=98.2, fpr=96.2)
        metrics.log_roc_data_point(threshold=24.3, tpr=24.5, fpr=98.4)
        metrics.set_confusion_matrix_categories(['dog', 'cat', 'horses'])
        metrics.log_confusion_matrix_row('dog', [2, 6, 0])
        metrics.log_confusion_matrix_cell('cat', 'dog', 3)
        metrics.log_confusion_matrix_cell('horses', 'horses', 3)
        metrics.metadata['test'] = 1.0
        with open(
                os.path.join(
                    os.path.dirname(__file__), 'test_data',
                    'expected_io_types_classification_metrics.json')
        ) as json_file:
            expected_json = json.load(json_file)
            self.assertEqual(expected_json, metrics.metadata)

    def test_complex_metrics_bulk_loading(self):
        metrics = dsl.ClassificationMetrics()
        metrics.log_roc_curve(
            fpr=[85.1, 85.1, 85.1],
            tpr=[52.6, 52.6, 52.6],
            threshold=[53.6, 53.6, 53.6])
        metrics.log_confusion_matrix(['dog', 'cat', 'horses'],
                                     [[2, 6, 0], [3, 5, 6], [5, 7, 8]])
        with open(
                os.path.join(
                    os.path.dirname(__file__), 'test_data',
                    'expected_io_types_bulk_load_classification_metrics.json')
        ) as json_file:
            expected_json = json.load(json_file)
            self.assertEqual(expected_json, metrics.metadata)


@contextlib.contextmanager
def set_temporary_task_root(task_root: str):
    artifact_types.CONTAINER_TASK_ROOT = task_root
    try:
        yield
    finally:
        artifact_types.CONTAINER_TASK_ROOT = None


class TestGetUri(unittest.TestCase):

    def test_raise_if_no_env_var(self):

        with self.assertRaisesRegex(
                RuntimeError,
                r"'dsl\.get_uri' can only be called at task runtime\. The task root is unknown in the current environment\."
        ):
            dsl.get_uri()

    def test_default_gcs(self):
        with set_temporary_task_root(
                '/gcs/my_bucket/123456789/abc-09-14-2023-14-21-53/foo_123456789'
        ):
            self.assertEqual(
                'gs://my_bucket/123456789/abc-09-14-2023-14-21-53/foo_123456789/Output',
                dsl.get_uri())

    def test_default_s3(self):
        with set_temporary_task_root(
                '/s3/my_bucket/123456789/abc-09-14-2023-14-21-53/foo_123456789'
        ):
            self.assertEqual(
                's3://my_bucket/123456789/abc-09-14-2023-14-21-53/foo_123456789/Output',
                dsl.get_uri())

    def test_default_minio(self):
        with set_temporary_task_root(
                '/minio/my_bucket/123456789/abc-09-14-2023-14-21-53/foo_123456789'
        ):
            self.assertEqual(
                'minio://my_bucket/123456789/abc-09-14-2023-14-21-53/foo_123456789/Output',
                dsl.get_uri())

    def test_suffix_arg_gcs(self):
        with set_temporary_task_root(
                '/gcs/my_bucket/123456789/abc-09-14-2023-14-21-53/foo_123456789'
        ):
            self.assertEqual(
                'gs://my_bucket/123456789/abc-09-14-2023-14-21-53/foo_123456789/model',
                dsl.get_uri('model'))

    def test_suffix_arg_tmp_no_suffix(self):
        with set_temporary_task_root('/tmp/kfp_outputs'):
            with self.assertWarnsRegex(
                    RuntimeWarning,
                    r'dsl\.get_uri is not yet supported by the KFP backend\. Please specify a URI explicitly\.'
            ):
                actual = dsl.get_uri('model')
                self.assertEqual('', actual)

    def test_suffix_arg_tmp_with_suffix(self):
        with set_temporary_task_root('/tmp/kfp_outputs'):
            with self.assertWarnsRegex(
                    RuntimeWarning,
                    r'dsl\.get_uri is not yet supported by the KFP backend\. Please specify a URI explicitly\.'
            ):
                actual = dsl.get_uri('model')
                self.assertEqual('', actual)


class TestConvertLocalPathToRemotePath(parameterized.TestCase):

    @parameterized.parameters([{
        'local_path': local_path,
        'expected': expected
    } for local_path, expected in [
        ('/gcs/foo/bar', 'gs://foo/bar'),
        ('/minio/foo/bar', 'minio://foo/bar'),
        ('/s3/foo/bar', 's3://foo/bar'),
        ('/oci/quay.io_org_repo:latest/models',
         'oci://quay.io/org/repo:latest'),
        ('/oci/quay.io_org_repo:latest', 'oci://quay.io/org/repo:latest'),
        ('/tmp/kfp_outputs', '/tmp/kfp_outputs'),
        ('/some/random/path', '/some/random/path'),
    ]])
    def test_gcs(self, local_path, expected):
        actual = artifact_types.convert_local_path_to_remote_path(local_path)
        self.assertEqual(actual, expected)


if __name__ == '__main__':
    unittest.main()
