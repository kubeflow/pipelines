# Copyright 2018-2019 Google LLC
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


from ._pipeline_param import PipelineParam, match_serialized_pipelineparam
from ._pipeline import Pipeline, pipeline, get_pipeline_conf, PipelineConf
from ._container_op import ContainerOp, InputArgumentPath, UserContainer, Sidecar
from ._resource_op import ResourceOp
from ._volume_op import (
    VolumeOp, VOLUME_MODE_RWO, VOLUME_MODE_RWM, VOLUME_MODE_ROM
)
from ._pipeline_volume import PipelineVolume
from ._volume_snapshot_op import VolumeSnapshotOp
from ._ops_group import OpsGroup, ExitHandler, Condition, ParallelFor
from ._component import python_component, graph_component, component

EXECUTION_ID_PLACEHOLDER = '{{workflow.uid}}-{{pod.name}}'
RUN_ID_PLACEHOLDER = '{{workflow.uid}}'
