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


import json
import kfp.dsl as dsl


@dsl.pipeline(
    name="Deploy example MAB",
    description="Multi-armed bandit example"
)
def mabdeploy_seldon():


    mabjson = """
{
    "apiVersion": "machinelearning.seldon.io/v1alpha2",
    "kind": "SeldonDeployment",
    "metadata": {
	"labels": {
	    "app": "seldon"
	},
	"name": "mnist-classifier"
    },
    "spec": {
	"annotations": {
	    "project_name": "kubeflow-seldon",
	    "deployment_version": "v1"
	},
	"name": "mnist-classifier",
	"predictors": [
	    {
		"componentSpecs": [{
		    "spec": {
			"containers": [
			    {
                                "image": "seldonio/deepmnistclassifier_runtime:0.2",
				"name": "tf-model",
                                "volumeMounts": [
                                    {
                                        "mountPath": "/data",
                                        "name": "persistent-storage"
                                    }
                                ]
			    },
			    {
                                "image": "seldonio/skmnistclassifier_runtime:0.2",
				"name": "sk-model",
                                "volumeMounts": [
                                    {
                                        "mountPath": "/data",
                                        "name": "persistent-storage"
                                    }
                                ]
			    },
			    {
                                "image": "seldonio/rmnistclassifier_runtime:0.2",
				"name": "r-model",
                                "volumeMounts": [
                                    {
                                        "mountPath": "/data",
                                        "name": "persistent-storage"
                                    }
                                ]
			    },
			    {
				"image": "seldonio/mab_epsilon_greedy:1.1",
				"name": "eg-router"
			    }
			],
                        "volumes": [
                            {
                                "name": "persistent-storage",
				"volumeSource" : {
                                    "persistentVolumeClaim": {
					"claimName": "nfs-1"
                                    }
				}
                            }
                        ]
		    }
		}],
		"name": "mnist-classifier",
		"replicas": 1,
		"annotations": {
		    "predictor_version": "v1"
		},
		"graph": {
		    "name": "eg-router",
		    "type":"ROUTER",
		    "parameters": [
			{
			    "name": "n_branches",
			    "value": "3",
			    "type": "INT"
			},
			{
			    "name": "epsilon",
			    "value": "0.2",
			    "type": "FLOAT"
			},
			{
			    "name": "verbose",
			    "value": "1",
			    "type": "BOOL"
			}
		    ],
		    "children": [
			{
			    "name": "sk-model",
			    "type": "MODEL",
			    "endpoint":{
				"type":"REST"
			    }
			},
			{
			    "name": "tf-model",
			    "type": "MODEL",
			    "endpoint":{
				"type":"REST"
			    }
			},
			{
			    "name": "r-model",
			    "type": "MODEL",
			    "endpoint":{
				"type":"REST"
			    }
			}
		    ]
		}
	    }
	]
    }
}
"""

    mabdeployment = json.loads(mabjson)

    deploy = dsl.ResourceOp(
        name="deploy",
        k8s_resource=mabdeployment,
        attribute_outputs={"name": "{.metadata.name}"}
    )


if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(mabdeploy_seldon, __file__ + ".tar.gz")
