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
"""Test CustomTrainingJobOp compilation."""

import os

from google_cloud_pipeline_components.v1 import custom_job
from google_cloud_pipeline_components.tests.v1 import utils
from kfp import dsl

import unittest


class TestCustomTrainingJobCompile(unittest.TestCase):

  def test_compile(self):
    @dsl.pipeline
    def pipeline():
      custom_job.CustomTrainingJobOp(
          project="my-project",
          location="us-central1",
          display_name="job_display_name",
          worker_pool_specs=[{
              "machine_spec": {
                  "machine_type": "n1-standard-4",
                  "accelerator_type": "NVIDIA_TESLA_T4",
                  "accelerator_count": 1,
              },
              "replica_count": 1,
              "container_spec": {"image_uri": "alpine"},
          }],
          timeout="1000s",
          restart_job_on_worker_restart=True,
          tensorboard="projects/000000000000/locations/us-central1/tensorboards/000000000000",
          reserved_ip_ranges=["1.0.0.0"],
          enable_web_access=True,
          base_output_directory=(
              "gs://my-vertex-pipelines-bucket/custom_job_outputs/"
          ),
          labels={"k": "v"},
          service_account=(
              "my-vertex-service-account@my-project.iam.gserviceaccount.com"
          ),
      )

    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(__file__),
            "testdata",
            "pipeline_with_custom_job_op.json",
        ),
    )
