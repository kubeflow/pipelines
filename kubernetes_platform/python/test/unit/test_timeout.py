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

from google.protobuf import json_format
from kfp import dsl
from kfp import kubernetes
import pytest


class TestTimeout:

    def test_timeout(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_timeout(
                task,
                seconds=20
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'activeDeadlineSeconds': '20'
                            }
                        }
                    }
                }
            }
        }

    def test_reset_timeout(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_timeout(
                task,
                seconds=20
            )
            kubernetes.set_timeout(
                task,
                seconds=0
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                            }
                        }
                    }
                }
            }
        }

    def test_bad_value_timeout(self):

        with pytest.raises(
                ValueError,
                match=r'Argument for "seconds" must be an integer greater or equals to 0. Got invalid input: -20.',
        ):
            
            @dsl.pipeline
            def my_pipeline():
                task = comp()
                kubernetes.set_timeout(
                    task,
                    seconds=-20
                )


@dsl.component
def comp():
    pass
