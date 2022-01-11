# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Tests for google_cloud_pipeline_components.types."""

import unittest
from google_cloud_pipeline_components.types import artifact_types

class ArtifactsTypesTest(unittest.TestCase):

    def test_vertex_model(self):
        model = artifact_types.VertexModel(uri='foo')
        self.assertEqual('foo', model.uri)

    def test_vertex_endpoint(self):
        endpoint = artifact_types.VertexEndpoint(uri='foo')
        self.assertEqual('foo', endpoint.uri)

    def test_vertex_batch_prediction_job(self):
        bp_job = artifact_types.VertexBatchPredictionJob(uri='foo')
        self.assertEqual('foo', bp_job.uri)

    def test_vertex_dataset(self):
        dataset = artifact_types.VertexDataset(uri='foo')
        self.assertEqual('foo', dataset.uri)

    def test_bqml_model(self):
        bqml_model = artifact_types.BQMLModel(uri='foo')
        self.assertEqual('foo', bqml_model.uri)

    def test_bq_table(self):
        bq_table = artifact_types.BQTable(uri='foo')
        self.assertEqual('foo', bq_table.uri)
