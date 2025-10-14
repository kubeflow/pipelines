# Copyright 2025 The Kubeflow Authors
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

import kfp
from kfp import dsl
from kfp.kubernetes.node_affinity import add_node_affinity, add_node_affinity_json

@dsl.component
def print_hello_with_json_affinity():
    pass

@dsl.component
def print_hello_with_preferred_affinity():
    pass

@dsl.component
def print_hello_with_empty_json():
    pass

@dsl.pipeline(name="test-node-affinity")
def my_pipeline():
    
   # Task 1: JSON-based node affinity with preferred scheduling
    task1 = print_hello_with_json_affinity()
    task1 = add_node_affinity_json(
        task1,
        {
            "preferredDuringSchedulingIgnoredDuringExecution": [
                {
                    "weight": 100,
                    "preference": {
                        "matchExpressions": [
                            {
                                "key": "disktype",
                                "operator": "In",
                                "values": ["ssd"]
                            }
                        ]
                    }
                }
            ]
        }
    )    

    # Task 2: Preferred scheduling with weight
    task2 = print_hello_with_preferred_affinity()
    task2 = add_node_affinity(
        task2,
        match_expressions=[
            {"key": "zone", "operator": "In", "values": ["us-west-1"]}
        ],
        weight=50
    )    

    # Task 3: Empty JSON (should not set any affinity)
    task3 = print_hello_with_empty_json()
    task3 = add_node_affinity_json(task3, {})

    # Task 4: Complex JSON with multiple preferred terms
    task4 = print_hello_with_json_affinity()
    task4 = add_node_affinity_json(
        task4,
        {
            "preferredDuringSchedulingIgnoredDuringExecution": [
                {
                    "weight": 100,
                    "preference": {
                        "matchExpressions": [
                            {
                                "key": "zone",
                                "operator": "In",
                                "values": ["us-west-1"]
                            }
                        ]
                    }
                },
                {
                    "weight": 50,
                    "preference": {
                        "matchExpressions": [
                            {
                                "key": "instance-type",
                                "operator": "In",
                                "values": ["n1-standard-4"]
                            }
                        ]
                    }
                }
            ]
        }
    )
