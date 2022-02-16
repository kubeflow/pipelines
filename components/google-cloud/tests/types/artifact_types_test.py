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
    model = artifact_types.VertexModel(
        name="bar", uri="foo", model_resource_name="fake_model_resource_name")
    self.assertEqual(
        {
            "artifacts": [{
                "uri": "foo",
                "metadata": {
                    "resourceName": "fake_model_resource_name"
                }
            }]
        }, model.to_executor_output_artifact({"artifacts":[{}]}))

  def test_vertex_endpoint(self):
    endpoint = artifact_types.VertexEndpoint(
        name="bar",
        uri="foo",
        endpoint_resource_name="fake_endpoint_resource_name")
    self.assertEqual(
        {
            "artifacts": [{
                "uri": "foo",
                "metadata": {
                    "resourceName": "fake_endpoint_resource_name"
                }
            }]
        }, endpoint.to_executor_output_artifact({"artifacts":[{}]}))

  def test_vertex_batch_prediction_job(self):
    bp_job = artifact_types.VertexBatchPredictionJob(
        name="bar",
        uri="foo",
        job_resource_name="fake_job_name",
        gcs_output_directory="fake_gcs_output_directory")
    self.assertEqual(
        {
            "artifacts": [{
                "uri": "foo",
                "metadata": {
                    "resourceName": "fake_job_name",
                    "bigqueryOutputTable": None,
                    "bigqueryOutputDataset": None,
                    "gcsOutputDirectory": "fake_gcs_output_directory"
                }
            }]
        }, bp_job.to_executor_output_artifact({"artifacts":[{}]}))

  def test_vertex_dataset(self):
    dataset = artifact_types.VertexDataset(
        name="bar",
        uri="foo",
        dataset_resource_name="fake_dataset_resource_name")
    self.assertEqual("foo", dataset.uri)

  def test_bqml_model(self):
    bqml_model = artifact_types.BQMLModel(
        name="bar",
        project_id="fake_project_id",
        dataset_id="fake_dataset_id",
        model_id="fake_model_id")
    self.assertEqual(
        {
            "artifacts": [{
                "uri":
                    "https://www.googleapis.com/bigquery/v2/projects/fake_project_id/datasets/fake_dataset_id/models/fake_model_id",
                "metadata": {
                    "projectId": "fake_project_id",
                    "datasetId": "fake_dataset_id",
                    "modelId": "fake_model_id"
                }
            }]
        }, bqml_model.to_executor_output_artifact({"artifacts":[{}]}))

  def test_bq_table(self):
    bq_table = artifact_types.BQTable(
        name="bar",
        project_id="fake_project_id",
        dataset_id="fake_dataset_id",
        table_id="fake_table_id")
    self.assertEqual(
        {
            "artifacts": [{
                "uri":
                    "https://www.googleapis.com/bigquery/v2/projects/fake_project_id/datasets/fake_dataset_id/tables/fake_table_id",
                "metadata": {
                    "projectId": "fake_project_id",
                    "datasetId": "fake_dataset_id",
                    "tableId": "fake_table_id"
                }
            }]
        }, bq_table.to_executor_output_artifact({"artifacts":[{}]}))
