# Copyright 2021 Google LLC
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
"""End-to-end test for KFP aiplatform module."""
import json
import os
import shutil
import tempfile
import unittest

import kfp
from kfp.v2.google import aiplatform
from kfp.v2 import compiler
from kfp import components
from kfp.components import _structures as structures
from kfp import dsl


class AiplatformE2ETest(unittest.TestCase):

  def setUp(self):
    with open(
        os.path.join(
            os.path.dirname(__file__), 'testdata',
            'expected_custom_job_pipeline.json'), 'r') as f:
      self._expected_pipeline_json = json.load(f)
      # Correct the sdkVersion
      self._expected_pipeline_json['pipelineSpec'][
          'sdkVersion'] = 'kfp-{}'.format(kfp.__version__)

    # Create a temporary directory
    self._test_dir = tempfile.mkdtemp()
    self.addCleanup(shutil.rmtree, self._test_dir)
    self._old_dir = os.getcwd()
    os.chdir(self._test_dir)
    self.addCleanup(os.chdir, self._old_dir)

    self.maxDiff = None

  def testCompileSimplePipeline(self):
    write_to_gcs = components.load_component_from_text("""
        name: Write to GCS
        inputs:
        - {name: text, type: String, description: 'Content to be written to GCS'}
        outputs:
        - {name: output_text, type: Dataset, description: 'GCS file path'}
        implementation:
          container:
            image: google/cloud-sdk:slim
            command:
            - sh
            - -c
            - |
              set -e -x
              echo "$0" | gsutil cp - "$1"
            - {inputValue: text}
            - {outputUri: output_text}
        """)

    @dsl.pipeline(
        name='dummy-custom-job-pipeline',
        pipeline_root='gs://my-bucket/my-output-dir')
    def custom_job_pipeline():
      task_1 = write_to_gcs(text='hello world')
      custom_echo_job = aiplatform.custom_job(
          name='test-custom-job',
          input_artifacts={'input_text': task_1.outputs['output_text']},
          image_uri='google/cloud-sdk:slim',
          commands=[
              'sh', '-c', 'set -e -x\ngsutil cat "$0"\n',
              structures.InputUriPlaceholder('input_text')
          ])

    compiler.Compiler().compile(custom_job_pipeline, 'pipeline.json')
    with open('pipeline.json', 'r') as f:
      self.assertDictEqual(self._expected_pipeline_json, json.load(f))


if __name__ == '__main__':
  unittest.main()
