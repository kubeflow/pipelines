# Copyright 2022 Google LLC
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
"""Test pipeline for Vertex AI Notification Email."""

import os
import tempfile

from google_cloud_pipeline_components.v1.vertex_notification_email import VertexNotificationEmailOp
from kfp import compiler
from kfp import dsl

import unittest


class ComponentsCompileTest(unittest.TestCase):

  def test_compile(self):
    @dsl.component
    def print_op(message: str):
      """Prints a message."""
      print(message)

    @dsl.component
    def fail_op(message: str):
      """Fails the pipeline task and triggers the notification."""
      import sys

      print(message)
      sys.exit(1)

    @dsl.pipeline(name="v2-pipeline-with-email-notification")
    def my_pipeline(message: str = "Hello World!"):
      """Test pipeline for Vertex AI Notification Email."""
      exit_task = VertexNotificationEmailOp(
          recipients=[f"managed-pipelines-email-notification-test@google.com"]
      )

      with dsl.ExitHandler(exit_task, name="my-pipeline"):
        print_op(message=message)
        fail_op(message="Task failed.")

    with tempfile.TemporaryDirectory() as temp_dir:
      compiler.Compiler().compile(
          my_pipeline, os.path.join(temp_dir, "pipeline.yaml")
      )
