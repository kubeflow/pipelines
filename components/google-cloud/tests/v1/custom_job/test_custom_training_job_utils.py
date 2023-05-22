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
"""Test custom training job creation utilities."""

import os

from google_cloud_pipeline_components.v1 import custom_job
from google_cloud_pipeline_components.tests.v1 import utils
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output

import unittest


@dsl.component
def sum_numbers(a: int, b: int) -> int:
  return a + b


class TestCreateCustomTrainingJobFromComponentCompile(unittest.TestCase):

  def test_compile_converted_custom_job_no_args(self):
    sum_numbers_custom_job = (
        custom_job.create_custom_training_job_from_component(sum_numbers)
    )

    @dsl.pipeline
    def pipeline():
      sum_numbers_custom_job(
          a=1,
          b=2,
          project="my-project",
          location="us-central1",
      )

    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(__file__),
            "testdata",
            "pipeline_with_converted_custom_job_no_args.json",
        ),
    )

  def test_compile_converted_custom_job_no_args(self):
    @dsl.component
    def make_artifact(
        out_artifact: Output[Artifact],
    ):
      ...

    @dsl.component
    def dummy_component(
        in_param: int,
        in_artifact: Input[Artifact],
        out_artifact: Output[Artifact],
    ) -> str:
      pass

    dummy_component_custom_job = (
        custom_job.create_custom_training_job_from_component(dummy_component)
    )

    @dsl.pipeline
    def pipeline():
      task1 = make_artifact()
      dummy_component_custom_job(
          in_param=1,
          in_artifact=task1.outputs["out_artifact"],
          project="my-project",
          location="us-central1",
      )

    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(__file__),
            "testdata",
            "pipeline_with_converted_custom_job_no_args_with_artifact_and_params.json",
        ),
    )

  def test_compile_converted_custom_job_with_args(self):
    sum_numbers_custom_job = custom_job.create_custom_training_job_from_component(
        sum_numbers,
        display_name="job_display_name",
        machine_type="a2-ultragpu-1g",
        accelerator_type="NVIDIA_TESLA_A100",
        accelerator_count=10,
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

    @dsl.pipeline
    def pipeline():
      sum_numbers_custom_job(
          a=1,
          b=2,
          project="my-project",
          location="us-central1",
      )

    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(__file__),
            "testdata",
            "pipeline_with_converted_custom_job_with_args.json",
        ),
    )

  def test_create_custom_training_job_op_from_component_is_deprecated(self):
    with self.assertWarnsRegex(
        DeprecationWarning,
        r"'create_custom_training_job_op_from_component' is deprecated\. Please"
        r" use 'create_custom_training_job_from_component' instead\.",
    ):
      custom_job.create_custom_training_job_op_from_component(sum_numbers)
