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
import unittest.mock as mock
from pprint import pprint
import kfp
import kfp_server_api

from .metrics_visualization_v2 import metrics_visualization_pipeline
from .util import run_pipeline_func, TestCase, KfpMlmdClient


def verify(
    run: kfp_server_api.ApiRun, mlmd_connection_config, argo_workflow_name: str,
    **kwargs
):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(argo_workflow_name=argo_workflow_name)

    task_names = [*tasks.keys()]
    t.assertCountEqual(
        task_names,
        ['wine-classification', 'iris-sgdclassifier', 'digit-classification'],
        'task names'
    )

    wine_classification: KfpTask = tasks.get('wine-classification')
    iris_sgdclassifier: KfpTask = tasks.get('iris-sgdclassifier')
    digit_classification: KfpTask = tasks.get('digit-classification')

    pprint('======= wine classification task =======')
    pprint(wine_classification.get_dict())
    pprint('======= iris sgdclassifier task =======')
    pprint(iris_sgdclassifier.get_dict())
    pprint('======= digit classification task =======')
    pprint(digit_classification.get_dict())

    t.assertEqual(
        wine_classification.get_dict(), {
            'inputs': {
                'artifacts': [],
                'parameters': {}
            },
            'name': 'wine-classification',
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'confidenceMetrics': [{
                            'confidenceThreshold': 2.0,
                            'falsePositiveRate': 0.0,
                            'recall': 0.0
                        }, {
                            'confidenceThreshold': 1.0,
                            'falsePositiveRate': 0.0,
                            'recall': 0.33962264150943394
                        }, {
                            'confidenceThreshold': 0.9,
                            'falsePositiveRate': 0.0,
                            'recall': 0.6037735849056604
                        }, {
                            'confidenceThreshold': 0.8,
                            'falsePositiveRate': 0.0,
                            'recall': 0.8490566037735849
                        }, {
                            'confidenceThreshold': 0.6,
                            'falsePositiveRate': 0.0,
                            'recall': 0.8867924528301887
                        }, {
                            'confidenceThreshold': 0.5,
                            'falsePositiveRate': 0.0125,
                            'recall': 0.9245283018867925
                        }, {
                            'confidenceThreshold': 0.4,
                            'falsePositiveRate': 0.075,
                            'recall': 0.9622641509433962
                        }, {
                            'confidenceThreshold': 0.3,
                            'falsePositiveRate': 0.0875,
                            'recall': 1.0
                        }, {
                            'confidenceThreshold': 0.2,
                            'falsePositiveRate': 0.2375,
                            'recall': 1.0
                        }, {
                            'confidenceThreshold': 0.1,
                            'falsePositiveRate': 0.475,
                            'recall': 1.0
                        }, {
                            'confidenceThreshold': 0.0,
                            'falsePositiveRate': 1.0,
                            'recall': 1.0
                        }]
                    },
                    'name': 'metrics',
                    'type': 'system.ClassificationMetrics'
                }],
                'parameters': {}
            },
            'type': 'kfp.ContainerExecution'
        }
    )
    t.assertEqual(
        iris_sgdclassifier.get_dict(), {
            'inputs': {
                'artifacts': [],
                'parameters': {
                    'test_samples_fraction': 0.3
                }
            },
            'name': 'iris-sgdclassifier',
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'confusionMatrix': {
                            'annotationSpecs': [{
                                'displayName': 'Setosa'
                            }, {
                                'displayName': 'Versicolour'
                            }, {
                                'displayName': 'Virginica'
                            }],
                            'rows': [{ # these numbers can be random during execution
                                'row': [mock.ANY, mock.ANY, mock.ANY]
                            }, {
                                'row': [mock.ANY, mock.ANY, mock.ANY]
                            }, {
                                'row': [mock.ANY, mock.ANY, mock.ANY]
                            }]
                        }
                    },
                    'name': 'metrics',
                    'type': 'system.ClassificationMetrics'
                }],
                'parameters': {}
            },
            'type': 'kfp.ContainerExecution'
        }
    )
    rows = iris_sgdclassifier.get_dict(
    )['outputs']['artifacts'][0]['metadata']['confusionMatrix']['rows']
    for i, row in enumerate(rows):
        for j, item in enumerate(row['row']):
            t.assertIsInstance(
                item, float,
                f'value of confusion matrix row {i}, col {j} is not a number'
            )

    t.assertEqual(
        digit_classification.get_dict(), {
            'inputs': {
                'artifacts': [],
                'parameters': {}
            },
            'name': 'digit-classification',
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'accuracy': 92.0
                    },
                    'name': 'metrics',
                    'type': 'system.Metrics'
                }],
                'parameters': {}
            },
            'type': 'kfp.ContainerExecution'
        }
    )


run_pipeline_func([
    TestCase(
        pipeline_func=metrics_visualization_pipeline,
        verify_func=verify,
        mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE
    ),
])
