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

import unittest
import json
from pprint import pprint
import kfp
import kfp_server_api
from google.cloud import storage

from .tensorboard_minio import my_pipeline
from kfp.samples.test.utils import run_pipeline_func, TestCase, KfpMlmdClient


def verify(run: kfp_server_api.ApiRun, mlmd_connection_config, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')

    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)
    # uncomment to debug
    # pprint(tasks)
    vis = tasks['create-tensorboard-visualization']
    pprint(vis)
    mlpipeline_ui_metadata = vis.outputs.artifacts[0]
    t.assertEqual(mlpipeline_ui_metadata.name, 'mlpipeline-ui-metadata')

    # download artifact content
    storage_client = storage.Client()
    blob = storage.Blob.from_string(mlpipeline_ui_metadata.uri, storage_client)
    data = blob.download_as_text(storage_client)
    data = json.loads(data)
    t.assertTrue(data["outputs"][0]["source"])
    # source is a URI that is generated differently each run
    data["outputs"][0]["source"] = "<redacted>"
    t.assertEqual(
        {
            "outputs": [{
                "type":
                    "tensorboard",
                "source":
                    "<redacted>",
                "image":
                    "gcr.io/deeplearning-platform-release/tf2-cpu.2-3:latest",
                "pod_template_spec": {
                    "spec": {
                        "containers": [{
                            "env": [{
                                "name": "AWS_ACCESS_KEY_ID",
                                "valueFrom": {
                                    "secretKeyRef": {
                                        "name": "mlpipeline-minio-artifact",
                                        "key": "accesskey"
                                    }
                                }
                            }, {
                                "name": "AWS_SECRET_ACCESS_KEY",
                                "valueFrom": {
                                    "secretKeyRef": {
                                        "name": "mlpipeline-minio-artifact",
                                        "key": "secretkey"
                                    }
                                }
                            }, {
                                "name": "AWS_REGION",
                                "value": "minio"
                            }, {
                                "name": "S3_ENDPOINT",
                                "value": "minio-service:9000"
                            }, {
                                "name": "S3_USE_HTTPS",
                                "value": "0"
                            }, {
                                "name": "S3_VERIFY_SSL",
                                "value": "0"
                            }]
                        }]
                    }
                }
            }]
        }, data)


run_pipeline_func([
    TestCase(
        pipeline_func=my_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),
])
