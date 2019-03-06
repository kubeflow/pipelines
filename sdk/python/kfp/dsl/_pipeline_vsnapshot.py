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


from . import _pipeline
from . import _pipeline_param
from . import _pipeline_volume
from ..compiler._k8s_helper import K8sHelper
import re
from typing import Dict


class PipelineVolumeSnapshot(object):
    """Representing a volume snapshot that is passed between pipeline
    operators. That snapshot could be either new (created according to the
    user's needs from a PipelineVolume) or refering to an existing
    VolumeSnapshot.

    A PipelineVolumeSnapshot object may be passed between pipeline functions,
    to start their data management from a snapshot of the dataflow, also
    exposing dependencies.
    """
    def __init__(self, pipeline_volume=None,
                 name: str = None, snapshot: str = None,
                 snapshot_class: str = None,
                 annotations: Dict = None,
                 k8s_resource: Dict = None,
                 create: bool = True, is_exit_handler: bool = False):
        """Create a new instance of PipelineVolumeSnapshot.
        Args:
            pipeline_volume: An existing PipelineVolume
            name: A desired name to refer to the PipelineVolumeSnapshot, and to
                use as base when naming the created resource. This might change
                when the pipeline gets compiled, in case such name is already
                added.
            snapshot: The name of an existing VolumeSnapshot
            snapshot_class: The snapshot class to use for the dynamically
                created snapshot
            annotations: Annotations to be patched on creating PVC
            k8s_resource: An already configured k8s_resource
            create: If a k8s_resource (PVC) is provided, this orders the
                pipeline to create that resource.
            is_exit_handler: Whether it is used as an exit handler.
        Raises:
        AttributeError: if both pipeline_volume and snapshot are set
                        if pipeline_volume is set but it is not a
                            dsl.PipelineVolume
        ValueError: if not inside a pipeline
                    if k8s_resource without metadata.name or generateName is
                        provided
                    if the name is an invalid string
        """
        if not _pipeline.Pipeline.get_default_pipeline():
            raise ValueError("Default pipeline not defined.")

        # Initialize
        self.name = None
        self.snapshot_class = None
        self.deps = set()
        self.k8s_resource = None
        self.new_snap = False
        self.inputs = []
        self.is_exit_handler = False

        if k8s_resource:
            if create and (snapshot_class or snapshot or annotations):
                print("A k8s_resource was provided. Ignoring any other " +
                      "argument.")
            self.k8s_resource = k8s_resource
            # If the resource needs to be created, make sure name or
            # generateName is provided
            if "name" not in k8s_resource["metadata"] and \
               "generateName" not in k8s_resource["metadata"]:
                raise ValueError("k8s_resource must have metadata.name " +
                                 "or metadata.generateName")
            elif name is None and "name" in k8s_resource["metadata"]:
                self.name = name = k8s_resource["metadata"]["name"]
            elif name is None and "generateName" in \
                                  k8s_resource["metadata"]:
                self.name = name = k8s_resource["metadata"]["generateName"]
            else:
                self.name = name

            if create:
                self.new_snap = True
            pipeline_volume = None
            snapshot = None
            snapshot_class = None
        else:
            if pipeline_volume and snapshot:
                raise AttributeError("pipeline_volume and snapshot cannot be "
                                     "both set.")
            if pipeline_volume and not \
               isinstance(pipeline_volume, _pipeline_volume.PipelineVolume):
                raise AttributeError("pipeline_volume needs to be of type " +
                                     "dsl.PipelineVolume")

            if pipeline_volume:
                self.new_snap = True
                if pipeline_volume.pvc is None:
                    self.inputs.append(
                        _pipeline_param.PipelineParam(
                            name="name",
                            op_name=pipeline_volume.name
                        )
                    )

        # Set the name of the PipelineVolumeSnapshot
        name_provided = False
        if name:
            name_provided = True
            valid_name_regex = r'^[A-Za-z][A-Za-z0-9\s_-]*$'
            if not re.match(valid_name_regex, name):
                raise ValueError("Only letters, numbers, spaces, '_', and " +
                                 "'-' are allowed in name. Must begin with " +
                                 "letter: %s" % (name))
            self.name = name

        if not name_provided:
            if pipeline_volume:
                deps = pipeline_volume.deps
                if pipeline_volume.name in deps:
                    deps = deps.remove(pipeline_volume.name)
                self.name = pipeline_volume.name
                if deps:
                    self.name += "-after"
                    for dep in sorted(deps):
                        self.name += "-%s" % dep
                self.name += "-snapshot"
            elif snapshot:
                snapshot_param, snapshot = self._parse_local_argument(snapshot)
                if snapshot_param:
                    self.name = snapshot.split("}}")[0].split(".")[-1]
                else:
                    self.name = snapshot

        # If new, add it to the pipeline and get a unique name
        if self.new_snap:
            self.human_name = self.name
            self.name = _pipeline.Pipeline.get_default_pipeline().add_op(
                self,
                is_exit_handler
            )
            if pipeline_volume:
                self.deps.update(pipeline_volume.deps)

        if annotations:
            for key, val in annotations.items():
                isParam, useVal = self._parse_local_argument(val)
                if isParam:
                    annotations[key] = useVal

        # Keep other information
        self.pipeline_volume = pipeline_volume
        self.snapshot = snapshot
        if snapshot_class:
            self.snapshot_class = snapshot_class
        self.annotations = annotations
        self.is_exit_handler = is_exit_handler

        # Define the k8s_resource if one was not provided
        # If snapshot is set, self.k8s_resource is None.
        if self.k8s_resource is None and snapshot is None:
            k8s_resource = {
                "kind": "VolumeSnapshot",
                "apiVersion": "snapshot.storage.k8s.io/v1alpha1",
            }
            k8s_resource_metadata = {
                "name": "{{workflow.name}}-%s" % self.name,
            }
            if self.annotations:
                k8s_resource_metadata["annotations"] = self.annotations

            source_name = None
            if self.pipeline_volume.pvc:
                source_name = self.pipeline_volume.pvc
            else:
                source_name = "{{inputs.parameters.%s-name}}" % \
                    self.pipeline_volume.name
            k8s_resource_spec = {
                "source": {
                    "kind": "PersistentVolumeClaim",
                    "name": source_name
                }
            }
            if self.snapshot_class:
                k8s_resource_spec["snapshotClassName"] = self.snapshot_class

            k8s_resource["metadata"] = k8s_resource_metadata
            k8s_resource["spec"] = k8s_resource_spec

            self.k8s_resource = k8s_resource

    def _parse_global_argument(self, name: str):
        """Parses a string to check if it is a global PipelineParam.
        Returns:
            A boolean declaring whether name is a PipelineParam
            The name to use in the template
        """
        match = _pipeline_param._extract_pipelineparams(str(name))
        if len(match) == 1:
            if match[0].name not in _pipeline.Pipeline.get_default_pipeline(
                           ).global_params:
                raise AttributeError("Invalid PipelineParam. Field should " +
                                     "be a global PipelineParam or a " +
                                     "hardcoded value.")
            else:
                return True, "{{workflow.arguments.parameters.%s}}" % \
                             match[0].name
        else:
            return False, name

    def _parse_local_argument(self, name: str):
        """Parses a string to check if it is a PipelineParam and use it
        accordingly
        Returns:
            A boolean declaring whether name is a PipelineParam
            The name to use in the template
        """
        match = _pipeline_param._extract_pipelineparams(str(name))
        if len(match) == 1:
            inputs = set(self.inputs)
            match[0].name = K8sHelper.sanitize_k8s_name(match[0].name)
            if match[0].op_name:
                match[0].op_name = K8sHelper.sanitize_k8s_name(
                    match[0].op_name
                )
            inputs.add(match[0])
            self.inputs = list(inputs)
            return True, "{{inputs.parameters.%s}}" % match[0].name
        else:
            return False, name

    def __repr__(self):
        return str({self.__class__.__name__: self.__dict__})
