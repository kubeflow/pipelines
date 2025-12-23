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

import pytest
from google.protobuf import json_format
from kfp import dsl
from kfp import kubernetes


@dsl.component
def comp():
    pass


class TestSecurityContext:

    def test_set_privileged(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                privileged=True,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'privileged': True,
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_privileged_false(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                privileged=False,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'privileged': False,
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_allow_privilege_escalation(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                allow_privilege_escalation=False,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'allowPrivilegeEscalation': False,
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_run_as_user(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                run_as_user=1000,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'runAsUser': '1000',
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_run_as_group(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                run_as_group=1000,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'runAsGroup': '1000',
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_run_as_non_root(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                run_as_non_root=True,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'runAsNonRoot': True,
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_read_only_root_filesystem(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                read_only_root_filesystem=True,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'readOnlyRootFilesystem': True,
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_capabilities_add(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                capabilities_add=['NET_ADMIN', 'SYS_TIME'],
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'capabilities': {
                                        'add': ['NET_ADMIN', 'SYS_TIME'],
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_capabilities_drop(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                capabilities_drop=['ALL'],
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'capabilities': {
                                        'drop': ['ALL'],
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_capabilities_add_and_drop(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                capabilities_add=['NET_ADMIN'],
                capabilities_drop=['SYS_ADMIN'],
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'capabilities': {
                                        'add': ['NET_ADMIN'],
                                        'drop': ['SYS_ADMIN'],
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_se_linux_options(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                se_linux_options_user='system_u',
                se_linux_options_role='system_r',
                se_linux_options_type='container_t',
                se_linux_options_level='s0:c123,c456',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'seLinuxOptions': {
                                        'user': 'system_u',
                                        'role': 'system_r',
                                        'type': 'container_t',
                                        'level': 's0:c123,c456',
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_seccomp_profile_runtime_default(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                seccomp_profile_type='RuntimeDefault',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'seccompProfile': {
                                        'type': 'RuntimeDefault',
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_seccomp_profile_localhost(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                seccomp_profile_type='Localhost',
                seccomp_profile_localhost_profile='profiles/my-profile.json',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'seccompProfile': {
                                        'type': 'Localhost',
                                        'localhostProfile':
                                            'profiles/my-profile.json',
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_set_full_security_context(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                privileged=False,
                allow_privilege_escalation=False,
                run_as_user=1000,
                run_as_group=1000,
                run_as_non_root=True,
                read_only_root_filesystem=True,
                capabilities_add=['NET_BIND_SERVICE'],
                capabilities_drop=['ALL'],
                seccomp_profile_type='RuntimeDefault',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'privileged': False,
                                    'allowPrivilegeEscalation': False,
                                    'runAsUser': '1000',
                                    'runAsGroup': '1000',
                                    'runAsNonRoot': True,
                                    'readOnlyRootFilesystem': True,
                                    'capabilities': {
                                        'add': ['NET_BIND_SERVICE'],
                                        'drop': ['ALL'],
                                    },
                                    'seccompProfile': {
                                        'type': 'RuntimeDefault',
                                    }
                                }
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
            kubernetes.use_secret_as_volume(
                task,
                secret_name='my-secret',
                mount_path='/mnt/my_vol',
            )
            kubernetes.set_security_context(
                task,
                privileged=True,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'secretAsVolume': [{
                                    'secretName': 'my-secret',
                                    'mountPath': '/mnt/my_vol',
                                    'optional': False,
                                    'secretNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'my-secret'
                                        }
                                    }
                                }],
                                'securityContext': {
                                    'privileged': True,
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_invalid_seccomp_profile_type(self):
        with pytest.raises(ValueError) as excinfo:

            @dsl.pipeline
            def my_pipeline():
                task = comp()
                kubernetes.set_security_context(
                    task,
                    seccomp_profile_type='InvalidType',
                )

        assert "Invalid seccomp_profile_type" in str(excinfo.value)

    def test_localhost_profile_without_localhost_type(self):
        with pytest.raises(ValueError) as excinfo:

            @dsl.pipeline
            def my_pipeline():
                task = comp()
                kubernetes.set_security_context(
                    task,
                    seccomp_profile_type='RuntimeDefault',
                    seccomp_profile_localhost_profile='profiles/my-profile.json',
                )

        assert "can only be set when seccomp_profile_type is 'Localhost'" in str(
            excinfo.value)

    def test_set_user_and_group_zero(self):
        """Test that setting user/group to 0 (root) works correctly."""

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_security_context(
                task,
                run_as_user=0,
                run_as_group=0,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'securityContext': {
                                    'runAsUser': '0',
                                    'runAsGroup': '0',
                                }
                            }
                        }
                    }
                }
            }
        }
