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

from __future__ import annotations
import unittest
import unittest.mock as mock
import kfp
import kfp_server_api

from .metrics_visualization_v2 import metrics_visualization_pipeline
from kfp.samples.test.utils import KfpTask, run_pipeline_func, TestCase
from ml_metadata.proto import Execution


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    task_names = [*tasks.keys()]
    t.assertCountEqual(task_names, [
        'wine-classification', 'iris-sgdclassifier', 'digit-classification',
        'html-visualization', 'markdown-visualization'
    ], 'task names')

    wine_classification = tasks['wine-classification']
    iris_sgdclassifier = tasks['iris-sgdclassifier']
    digit_classification = tasks['digit-classification']
    html_visualization = tasks['html-visualization']
    markdown_visualization = tasks['markdown-visualization']

    t.assertEqual(
        {
            'name': 'wine-classification',
            'inputs': {},
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'metrics',
                        'confidenceMetrics': {
                            'list': [{
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
                        }
                    },
                    'name': 'metrics',
                    'type': 'system.ClassificationMetrics'
                }],
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, wine_classification.get_dict())
    t.assertEqual(
        {
            'inputs': {
                'parameters': {
                    'test_samples_fraction': 0.3
                }
            },
            'name': 'iris-sgdclassifier',
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'metrics',
                        'confusionMatrix': {
                            'struct': {
                                'annotationSpecs': [{
                                    'displayName': 'Setosa'
                                }, {
                                    'displayName': 'Versicolour'
                                }, {
                                    'displayName': 'Virginica'
                                }],
                                'rows': [
                                    {  # these numbers can be random during execution
                                        'row': [mock.ANY, mock.ANY, mock.ANY]
                                    },
                                    {
                                        'row': [mock.ANY, mock.ANY, mock.ANY]
                                    },
                                    {
                                        'row': [mock.ANY, mock.ANY, mock.ANY]
                                    }
                                ]
                            }
                        }
                    },
                    'name': 'metrics',
                    'type': 'system.ClassificationMetrics'
                }],
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        },
        iris_sgdclassifier.get_dict())
    rows = iris_sgdclassifier.get_dict()['outputs']['artifacts'][0]['metadata'][
        'confusionMatrix']['struct']['rows']
    for i, row in enumerate(rows):
        for j, item in enumerate(row['row']):
            t.assertIsInstance(
                item, float,
                f'value of confusion matrix row {i}, col {j} is not a number')

    t.assertEqual(
        {
            'name': 'digit-classification',
            'inputs': {},
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'metrics',
                        'accuracy': 92.0,
                    },
                    'name': 'metrics',
                    'type': 'system.Metrics'
                }],
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, digit_classification.get_dict())

    t.assertEqual(
        {
            'name': 'html-visualization',
            'inputs': {},
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'html_artifact'
                    },
                    'name': 'html_artifact',
                    'type': 'system.HTML'
                }],
            },
            'state': Execution.State.COMPLETE,
            'type': 'system.ContainerExecution'
        }, html_visualization.get_dict())

    t.assertEqual(
        {
            'name': 'markdown-visualization',
            'inputs': {},
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'markdown_artifact'
                    },
                    'name': 'markdown_artifact',
                    'type': 'system.Markdown'
                }],
            },
            'state': Execution.State.COMPLETE,
            'type': 'system.ContainerExecution'
        }, markdown_visualization.get_dict())


run_pipeline_func([
    TestCase(
        pipeline_func=metrics_visualization_pipeline,
        verify_func=verify,
        mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE),
])
