# Copyright 2026 The Kubeflow Authors
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

import pytest
from google.protobuf import json_format
from kfp import dsl
from kfp import kubernetes


class TestResourceClaim:

    def test_add_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_resource_claim(
                task,
                resource_claim_template_name='gpu-claim-template',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podResourceClaims': [{
                                    'resourceClaimTemplateName':
                                        'gpu-claim-template',
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_two(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_resource_claim(
                task,
                resource_claim_template_name='gpu-claim-template',
            )
            kubernetes.add_resource_claim(
                task,
                resource_claim_template_name='nic-claim-template',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podResourceClaims': [
                                    {
                                        'resourceClaimTemplateName':
                                            'gpu-claim-template',
                                    },
                                    {
                                        'resourceClaimTemplateName':
                                            'nic-claim-template',
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_respects_other_configuration(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration(
                task,
                key='key1',
                operator='Equal',
                value='value1',
            )
            kubernetes.add_resource_claim(
                task,
                resource_claim_template_name='gpu-claim-template',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'tolerations': [{
                                    'key': 'key1',
                                    'operator': 'Equal',
                                    'value': 'value1',
                                }],
                                'podResourceClaims': [{
                                    'resourceClaimTemplateName':
                                        'gpu-claim-template',
                                }],
                            }
                        }
                    }
                }
            }
        }

    def test_coexists_with_set_accelerator_limit(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            task.set_accelerator_limit(1)
            kubernetes.add_resource_claim(
                task,
                resource_claim_template_name='gpu-claim-template',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podResourceClaims': [{
                                    'resourceClaimTemplateName':
                                        'gpu-claim-template',
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_empty_name_raises_error(self):

        with pytest.raises(ValueError):

            @dsl.pipeline
            def my_pipeline():
                task = comp()
                kubernetes.add_resource_claim(
                    task,
                    resource_claim_template_name='',
                )


class TestResourceClaimJSON:

    def test_pipeline_input_parameter(self):

        @dsl.pipeline
        def my_pipeline(resource_claim: dict):
            task = comp()
            kubernetes.add_resource_claim_json(
                task,
                resource_claim_json=resource_claim,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podResourceClaims': [{
                                    'resourceClaimJson': {
                                        'componentInputParameter':
                                            'resource_claim'
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_static_resource_claim_config(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_resource_claim_json(
                task,
                resource_claim_json=kubernetes.ResourceClaimConfig(
                    resource_claim_template_name='gpu-claim-template',
                ),
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podResourceClaims': [{
                                    'resourceClaimJson': {
                                        'runtimeValue': {
                                            'constant': {
                                                'resourceClaimTemplateName':
                                                    'gpu-claim-template'
                                            }
                                        }
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_static_resource_claim_config_list(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_resource_claim_json(
                task,
                resource_claim_json=[
                    kubernetes.ResourceClaimConfig(
                        resource_claim_template_name='gpu-claim-template',
                    ),
                    kubernetes.ResourceClaimConfig(
                        resource_claim_template_name='nic-claim-template',
                    ),
                ],
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podResourceClaims': [
                                    {
                                        'resourceClaimJson': {
                                            'runtimeValue': {
                                                'constant': {
                                                    'resourceClaimTemplateName':
                                                        'gpu-claim-template'
                                                }
                                            }
                                        }
                                    },
                                    {
                                        'resourceClaimJson': {
                                            'runtimeValue': {
                                                'constant': {
                                                    'resourceClaimTemplateName':
                                                        'nic-claim-template'
                                                }
                                            }
                                        }
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_invalid_type_raises_error(self):

        with pytest.raises(ValueError):

            @dsl.pipeline
            def my_pipeline():
                task = comp()
                kubernetes.add_resource_claim_json(
                    task,
                    resource_claim_json=42,
                )

    def test_upstream_task_output(self):

        @dsl.pipeline
        def my_pipeline():
            t1 = comp()
            t2 = comp_with_output()
            kubernetes.add_resource_claim_json(
                t1,
                resource_claim_json=t2.output,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podResourceClaims': [{
                                    'resourceClaimJson': {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output'
                                        }
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }


class TestResourceClaimConfig:

    def test_valid_creation(self):
        config = kubernetes.ResourceClaimConfig(
            resource_claim_template_name='gpu-claim-template',
        )
        assert config.resource_claim_template_name == 'gpu-claim-template'

    def test_empty_name_raises_error(self):
        with pytest.raises(ValueError):
            kubernetes.ResourceClaimConfig(
                resource_claim_template_name='',
            )

    def test_missing_name_raises_error(self):
        with pytest.raises(TypeError):
            kubernetes.ResourceClaimConfig()

    def test_to_dict(self):
        config = kubernetes.ResourceClaimConfig(
            resource_claim_template_name='gpu-claim-template',
        )
        assert config.to_dict() == {
            'resourceClaimTemplateName': 'gpu-claim-template',
        }


@dsl.component
def comp():
    pass


@dsl.component()
def comp_with_output() -> str:
    pass
