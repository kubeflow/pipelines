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

from google.protobuf import json_format
from kfp import dsl
from kfp import kubernetes
import pytest


class TestSecurityContext:

    def test_security_context_all_fields(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.set_security_context(
                task,
                run_as_user=65534,
                run_as_group=0,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'securityContext': {
                                    'runAsUser': '65534',
                                    'runAsGroup': '0',
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_security_context_negative_run_as_user(self):
        with pytest.raises(
            ValueError,
            match=r'Argument for "run_as_user" must be greater than or equal to 0. Got invalid input: -1.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.set_security_context(task, run_as_user=-1)

    def test_security_context_negative_run_as_group(self):
        with pytest.raises(
            ValueError,
            match=r'Argument for "run_as_group" must be greater than or equal to 0. Got invalid input: -2.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.set_security_context(task, run_as_group=-2)

    def test_security_context_no_arguments(self):
        with pytest.raises(
            ValueError,
            match=r'At least one security context field must be provided.',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.set_security_context(task)

    def test_security_context_run_as_user_only(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.set_security_context(task, run_as_user=65534)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'securityContext': {
                                    'runAsUser': '65534',
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_security_context_run_as_group_only(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.set_security_context(task, run_as_group=0)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'securityContext': {
                                    'runAsGroup': '0',
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_security_context_bool_run_as_user(self):
        with pytest.raises(
            TypeError,
            match=r'Argument for "run_as_user" must be an int, not bool',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.set_security_context(task, run_as_user=True)

    def test_security_context_bool_run_as_group(self):
        with pytest.raises(
            TypeError,
            match=r'Argument for "run_as_group" must be an int, not bool',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.set_security_context(task, run_as_group=True)

    def test_security_context_run_as_non_root_true(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.set_security_context(task, run_as_non_root=True)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'securityContext': {
                                    'runAsNonRoot': True,
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_security_context_run_as_non_root_false(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.set_security_context(task, run_as_non_root=False)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'securityContext': {
                                    'runAsNonRoot': False,
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_security_context_all_fields_with_non_root(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.set_security_context(
                task,
                run_as_user=65534,
                run_as_group=0,
                run_as_non_root=True,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'securityContext': {
                                    'runAsUser': '65534',
                                    'runAsGroup': '0',
                                    'runAsNonRoot': True,
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_security_context_run_as_non_root_wrong_type(self):
        with pytest.raises(
            TypeError,
            match=r'Argument for "run_as_non_root" must be a bool',
        ):

            @dsl.pipeline
            def my_pipeline():
                task = print_greeting()
                kubernetes.set_security_context(task, run_as_non_root=1)

    def test_security_context_run_as_user_zero(self):

        @dsl.pipeline
        def my_pipeline():
            task = print_greeting()
            kubernetes.set_security_context(task, run_as_user=0)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-print-greeting': {
                                'securityContext': {
                                    'runAsUser': '0',
                                }
                            }
                        }
                    }
                }
            }
        }


@dsl.component
def print_greeting():
    print('hello world')
