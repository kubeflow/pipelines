# Copyright 2024 The Kubeflow Authors
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
"""Tests for graph_utils.py."""
from typing import Any, Dict
import unittest

from google.protobuf import json_format
from kfp.local import graph_utils
from kfp.pipeline_spec import pipeline_spec_pb2


class TestBuildAdjacencyList(unittest.TestCase):

    def test_simple(self):
        tasks = {
            k: make_pipeline_task_spec(v)
            for k, v in SIMPLE_TASK_TOPOLOGY.items()
        }
        actual = graph_utils.build_adjacency_list(tasks)
        expected = {'identity': [], 'identity-2': ['identity']}
        self.assertEqual(actual, expected)

    def test_complex(self):
        tasks = {
            k: make_pipeline_task_spec(v)
            for k, v in COMPLEX_TASK_TOPOLOGY.items()
        }
        actual = graph_utils.build_adjacency_list(tasks)
        expected = {
            'add': [],
            'add-2': ['multiply'],
            'divide': ['add-2'],
            'multiply': ['add'],
            'printer': ['add', 'divide', 'multiply']
        }
        self.assertEqual(actual, expected)


class TestTopologicalSort(unittest.TestCase):

    def test_empty_graph(self):
        self.assertEqual(graph_utils.topological_sort({}), [])

    def test_simple_linear_graph(self):
        graph = {'A': ['B'], 'B': ['C'], 'C': []}
        actual = graph_utils.topological_sort(graph)
        expected = ['A', 'B', 'C']
        self.assertEqual(actual, expected)

    def test_separate_components(self):
        graph = {'A': ['B'], 'B': [], 'C': ['D'], 'D': []}
        actual = graph_utils.topological_sort(graph)
        expected = ['C', 'D', 'A', 'B']
        self.assertEqual(actual, expected)

    def test_complex_graph(self):
        graph = {'A': ['B', 'C'], 'B': ['D'], 'C': ['D'], 'D': []}
        actual = graph_utils.topological_sort(graph)
        expected = ['A', 'C', 'B', 'D']
        self.assertEqual(actual, expected)


class TestTopologicalSortTasks(unittest.TestCase):

    def test_simple(self):
        tasks = {
            k: make_pipeline_task_spec(v)
            for k, v in SIMPLE_TASK_TOPOLOGY.items()
        }
        actual = graph_utils.topological_sort_tasks(tasks)
        expected = ['identity-2', 'identity']
        self.assertEqual(actual, expected)

    def test_complex(self):
        tasks = {
            k: make_pipeline_task_spec(v)
            for k, v in COMPLEX_TASK_TOPOLOGY.items()
        }
        actual = graph_utils.topological_sort_tasks(tasks)
        expected = ['printer', 'divide', 'add-2', 'multiply', 'add']
        self.assertEqual(actual, expected)


SIMPLE_TASK_TOPOLOGY = {
    'identity': {
        'cachingOptions': {
            'enableCache': True
        },
        'componentRef': {
            'name': 'comp-identity'
        },
        'inputs': {
            'parameters': {
                'string': {
                    'componentInputParameter': 'string'
                }
            }
        },
        'taskInfo': {
            'name': 'identity'
        }
    },
    'identity-2': {
        'cachingOptions': {
            'enableCache': True
        },
        'componentRef': {
            'name': 'comp-identity-2'
        },
        'dependentTasks': ['identity'],
        'inputs': {
            'parameters': {
                'string': {
                    'taskOutputParameter': {
                        'outputParameterKey': 'Output',
                        'producerTask': 'identity'
                    }
                }
            }
        },
        'taskInfo': {
            'name': 'identity-2'
        }
    }
}

COMPLEX_TASK_TOPOLOGY = {
    'add': {
        'cachingOptions': {
            'enableCache': True
        },
        'componentRef': {
            'name': 'comp-add'
        },
        'inputs': {
            'parameters': {
                'a': {
                    'runtimeValue': {
                        'constant': 1.0
                    }
                },
                'b': {
                    'runtimeValue': {
                        'constant': 2.0
                    }
                }
            }
        },
        'taskInfo': {
            'name': 'add'
        }
    },
    'add-2': {
        'cachingOptions': {
            'enableCache': True
        },
        'componentRef': {
            'name': 'comp-add-2'
        },
        'dependentTasks': ['multiply'],
        'inputs': {
            'parameters': {
                'a': {
                    'taskOutputParameter': {
                        'outputParameterKey': 'Output',
                        'producerTask': 'multiply'
                    }
                },
                'b': {
                    'runtimeValue': {
                        'constant': 4.0
                    }
                }
            }
        },
        'taskInfo': {
            'name': 'add-2'
        }
    },
    'divide': {
        'cachingOptions': {
            'enableCache': True
        },
        'componentRef': {
            'name': 'comp-divide'
        },
        'dependentTasks': ['add-2'],
        'inputs': {
            'parameters': {
                'denominator': {
                    'runtimeValue': {
                        'constant': 2.0
                    }
                },
                'numerator': {
                    'taskOutputParameter': {
                        'outputParameterKey': 'Output',
                        'producerTask': 'add-2'
                    }
                }
            }
        },
        'taskInfo': {
            'name': 'divide'
        }
    },
    'multiply': {
        'cachingOptions': {
            'enableCache': True
        },
        'componentRef': {
            'name': 'comp-multiply'
        },
        'dependentTasks': ['add'],
        'inputs': {
            'parameters': {
                'x': {
                    'taskOutputParameter': {
                        'outputParameterKey': 'Output',
                        'producerTask': 'add'
                    }
                },
                'y': {
                    'runtimeValue': {
                        'constant': 3.0
                    }
                }
            }
        },
        'taskInfo': {
            'name': 'multiply'
        }
    },
    'printer': {
        'cachingOptions': {
            'enableCache': True
        },
        'componentRef': {
            'name': 'comp-printer'
        },
        'dependentTasks': ['add', 'divide', 'multiply'],
        'inputs': {
            'parameters': {
                'flt': {
                    'taskOutputParameter': {
                        'outputParameterKey': 'Output',
                        'producerTask': 'divide'
                    }
                },
                'int1': {
                    'taskOutputParameter': {
                        'outputParameterKey': 'Output',
                        'producerTask': 'add'
                    }
                },
                'int2': {
                    'taskOutputParameter': {
                        'outputParameterKey': 'Output',
                        'producerTask': 'multiply'
                    }
                }
            }
        },
        'taskInfo': {
            'name': 'printer'
        }
    }
}


def make_pipeline_task_spec(
        d: Dict[str, Any]) -> pipeline_spec_pb2.PipelineTaskSpec:
    spec = pipeline_spec_pb2.PipelineTaskSpec()
    json_format.ParseDict(d, spec)
    return spec


if __name__ == '__main__':
    unittest.main()
