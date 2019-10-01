# Copyright 2019 Google LLC
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


"""
This example demonstrates how to use ResourceOp to specify the value of env var.
"""

import json
import kfp
import kfp.dsl as dsl
from kubernetes import client as k8s_client
from string import Template


_ENV_CONFIG = """
{
    "apiVersion": "v1",
    "kind": "ConfigMap",
    "metadata": {
        "name": "sample-env-config"
    },
    "data": {
        "SPECIAL_LEVEL": "level",
        "SPECIAL_TYPE": "type"
    }
}    
"""

_CONTAINER_MANIFEST = """
{
    "apiVersion": "batch/v1",
    "kind": "Job",
    "metadata": {
        "generateName": "resourceop-basic-job-"
    },
    "spec": {
        "template": {
            "metadata": {
                "name": "resource-basic"
            },
            "spec": {
                "containers": [{
                    "name": "sample-container",
                    "image": "k8s.gcr.io/busybox",
                    "command": ["/bin/sh", "-c", "env"],
                    "envFrom": [{
                        "configMapRef": {
                            "name": "sample-env-config"
                        }
                    }]
                }],
                "restartPolicy": "Never"
            }
        },
        "backoffLimit": 4      
    }
}
"""


@dsl.pipeline(
    name="ResourceOp Basic",
    description="A Basic Example on ResourceOp Usage."
)
def resourceop_basic():
    # Create a ConfigMap.
    env_config_map_op = dsl.ResourceOp(
        name='environment-config',
        k8s_resource=json.loads(_ENV_CONFIG),
        action='create'
    )

    # Referring the ConfigMap in a Container.
    cop = dsl.ResourceOp(
        name='test-step',
        k8s_resource=json.loads(_CONTAINER_MANIFEST),
        action='create'
    ).after(env_config_map_op)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(resourceop_basic, __file__ + '.zip')
