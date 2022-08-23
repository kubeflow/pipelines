# Copyright 2022 The Kubeflow Authors
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
from typing import NamedTuple

from kfp.dsl import component
from kfp.dsl import Metrics


@component
def output_named_tuple(
    dummy: str) -> NamedTuple('Outputs', [
        ('scalar', str),
        ('metrics', Metrics),
    ]):
    scalar = '123'

    import json
    metrics = json.dumps({
        'metrics': [{
            'name': 'accuracy',
            'numberValue': 0.9,
            'format': 'PERCENTAGE',
        }]
    })

    from collections import namedtuple
    output = namedtuple('Outputs', ['scalar', 'metrics'])
    return output(scalar, metrics)
