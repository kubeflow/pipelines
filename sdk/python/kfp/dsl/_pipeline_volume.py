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
from . import _pipeline_vsnapshot
from ..compiler._k8s_helper import K8sHelper
from kubernetes import client as k8s_client
import re
from typing import List, Dict


VOLUME_MODE_RWO = ["ReadWriteOnce"]
VOLUME_MODE_RWM = ["ReadWriteMany"]
VOLUME_MODE_ROM = ["ReadOnlyMany"]


class PipelineVolume(object):
    """Representing a volume that is passed between pipeline operators.
    That volume could be either new (created according to the user's needs from
    scratch or from a VolumeSnapshot) or refering to an existing pvc.

    A PipelineVolume object can be used as an extention of the pipeline
    function's filesystem. It may then be passed between pipeline functions,
    exposing dependencies.
    """
    def __init__(self, name: str = None, pvc: str = None,
                 size: str = None, storage_class: str = None,
                 data_source=None, mode: List[str] = VOLUME_MODE_RWM,
                 annotations: Dict = None,
                 k8s_resource: k8s_client.V1PersistentVolumeClaim = None,
                 create: bool = True, is_exit_handler: bool = False):
        """Create a new instance of PipelineVolume.
        Args:
            name: A desired name to refer to the PipelineVolume, and to use as
                base when naming the created resource. This might change when
                the pipeline gets compiled, in case the name is already added.
            pvc: The name of an existing PVC
            size: The size of the PVC which will be created
            storage_class: The storage class to use for the dynamically created
                PVC (requires size or data_source)
            data_source: The name of an existing VolumeSnapshot or a
                PipelineVolumeSnapshot object
            mode: The access mode for the mounted PVC
            annotations: Annotations to be patched on creating PVC
            k8s_resource: An already configured k8s_resource
            create: If a k8s_resource (PVC) is provided, this orders the
                pipeline to create that resource.
            is_exit_handler: Whether it is used as an exit handler.
        Raises:
        AttributeError: if pvc is set with size or data_source
                        if storage_class is set along with pvc
                        if storage_class is set without size or data_source
                        if annotations are set without size or data_source
                        if nothing of size, data_source and pvc is set
        ValueError: if not inside a pipeline
                    if k8s_resource without metadata.name or generateName is
                        provided
                    if size is an invalid memory string (when not a
                        PipelineParam)
                    if the name is an invalid string
        """
        if not _pipeline.Pipeline.get_default_pipeline():
            raise ValueError("Default pipeline not defined.")

        # Initialize
        self.name = None
        self.pvc = None
        self.size = None
        self.storage_class = None
        self.data_source = None
        self.data_source_name = None
        self.mode = None
        self.deps = set()
        self.k8s_resource = None
        self.from_snapshot = False
        self.from_scratch = False
        self.inputs = []
        self.is_exit_handler = False

        if k8s_resource:
            if create and (size or pvc or data_source or storage_class
                           or mode):
                print("A k8s_resource was provided. Ignoring any other " +
                      "argument.")
            self.k8s_resource = k8s_resource
            # If the resource needs to be created, make sure name or
            # generateName is provided. Also find out whether this is a PVC
            # created from a snapshot, or an empty one.
            if create:
                if k8s_resource.metadata.name == "" and \
                   k8s_resource.metadata.generateName == "":
                    raise ValueError("k8s_resource must have metadata.name or "
                                     "metadata.generateName.")
                if name is None and k8s_resource.metadata.name != "":
                    self.name = name = k8s_resource.metadata.name
                elif name is None and k8s_resource.metadata.generateName != "":
                    self.name = name = k8s_resource.metadata.generateName
                else:
                    self.name = name
                if hasattr(k8s_resource.spec, "data_source"):
                    self.from_snapshot = True
                else:
                    self.from_scratch = True
            pvc = None
            size = None
            storage_class = None
            data_source = None
            mode = None
        else:  # Attribute checking
            if pvc and (data_source or size):
                raise AttributeError("pvc cannot be set with size or "
                                     "data_source")
            elif pvc and storage_class:
                raise AttributeError("pvc and storage_class cannot be both "
                                     "set.")
            elif storage_class and size is None and data_source is None:
                raise AttributeError("storage_class requires size or "
                                     "data_source.")
            elif annotations and not (size or data_source):
                raise AttributeError("Please provide size or data_source, when"
                                     " providing annotations")
            elif not (size or pvc or data_source):
                raise AttributeError("Please provide size, data_source or pvc")

            # Find out if data_source, size and pvc are PipelineParams
            ds_param = False
            size_param = False
            pvc_param = False
            if data_source:
                ds_param, data_source = self._parse_local_argument(data_source)
            if size:
                size_param, size = self._parse_local_argument(size)
                if not size_param:
                    self._validate_memory_string(size)
            if pvc:
                pvc_param, pvc = self._parse_global_argument(pvc)

            if data_source:
                self.data_source = data_source
                self.data_source_name = data_source
                self.from_snapshot = True
                if isinstance(data_source,
                              _pipeline_vsnapshot.PipelineVolumeSnapshot):
                    if self.data_source.new_snap:
                        self.deps.add(data_source.name)
                        self.inputs.append(
                            _pipeline_param.PipelineParam(
                                name="name",
                                op_name=self.data_source.name
                            )
                        )
                        self.inputs.append(
                            _pipeline_param.PipelineParam(
                                name="size",
                                op_name=self.data_source.name
                            )
                        )
                        self.data_source_name = data_source.name
                        data_source = "{{inputs.parameters.%s-name}}" % \
                                      data_source.name
                        if size is None:
                            size_param = True
                            size = "{{inputs.parameters.%s-size}}" % \
                                   self.data_source.name
                    else:
                        if data_source.snapshot is None:
                            self.data_source_name = data_source.name
                            data_source = data_source.name
                        else:
                            self.data_source_name = data_source.snapshot
                            data_source = data_source.snapshot
            elif size:
                self.from_scratch = True

        # Set the name of the PipelineVolume
        name_provided = False
        if name:
            name_provided = True
            valid_name_regex = r'^[A-Za-z][A-Za-z0-9\s_-]*$'
            if not re.match(valid_name_regex, name):
                raise ValueError("Only letters, numbers, spaces, '_', and '-' "
                                 "are allowed in name. Must begin with letter:"
                                 " %s" % (name))
            self.name = name

        if not name_provided:
            if data_source:
                if ds_param:
                    self.data_source_name = str(data_source).split(
                        "}}")[0].split(".")[-1]
                if size and not size_param:
                    self.name = "%s-%s-clone" % (
                        self.data_source_name,
                        size.lower()
                    )
                else:
                    self.name = "%s-clone" % self.data_source_name
            elif size:
                if not size_param:
                    self.name = "volume-%s" % size.lower()
                else:
                    self.name = "volume-%s" % (
                        str(size).split("}}")[0].split(".")[-1]
                    )
            elif pvc:
                if pvc_param:
                    self.name = pvc.split("}}")[0].split(".")[-1]
                else:
                    self.name = pvc

        # If new, add it to the pipeline and get a unique name
        # Also add self.name in dependencies if it is a new resource
        # Every other operator using this PipelineVolume
        # will need this dependency to operate properly
        if (self.from_scratch or self.from_snapshot):
            self.human_name = self.name
            self.name = _pipeline.Pipeline.get_default_pipeline().add_op(
                self,
                is_exit_handler
            )
            self.deps.add(str(self.name))

        if annotations:
            for key, val in annotations.items():
                isParam, useVal = self._parse_local_argument(val)
                if isParam:
                    annotations[key] = useVal

        # Keep other information
        self.size = size
        self.pvc = pvc
        if storage_class:
            _, storage_class = self._parse_local_argument(storage_class)
            self.storage_class = storage_class
        self.mode = mode
        self.annotations = annotations
        self.is_exit_handler = is_exit_handler

        # Define the k8s_resource if one was not provided
        # If pvc is set, self.k8s_resource is None.
        if self.k8s_resource is None and pvc is None:
            k8s_resource = None
            k8s_resource_metadata = k8s_client.V1ObjectMeta(
                name="{{workflow.name}}-%s" % self.name,
                annotations=self.annotations
            )
            k8s_resource_spec = None
            if self.from_scratch:
                requested_resources = k8s_client.V1ResourceRequirements(
                    requests={"storage": self.size}
                )
                k8s_resource_spec = k8s_client.V1PersistentVolumeClaimSpec(
                    access_modes=self.mode,
                    resources=requested_resources
                )
                if self.storage_class:
                    k8s_resource_spec.storage_class_name = self.storage_class
                k8s_resource = k8s_client.V1PersistentVolumeClaim(
                    api_version="v1",
                    kind="PersistentVolumeClaim",
                    metadata=k8s_resource_metadata,
                    spec=k8s_resource_spec
                )
            elif self.from_snapshot:
                requested_resources = k8s_client.V1ResourceRequirements(
                    requests={"storage": self.size}
                )
                snapshot = k8s_client.V1TypedLocalObjectReference(
                    api_group="snapshot.storage.k8s.io",
                    kind="VolumeSnapshot",
                    name=data_source
                )
                k8s_resource_spec = k8s_client.V1PersistentVolumeClaimSpec(
                    access_modes=self.mode,
                    storage_class_name=self.storage_class,
                    data_source=snapshot,
                    resources=requested_resources
                )
                k8s_resource = k8s_client.V1PersistentVolumeClaim(
                    api_version="v1",
                    kind="PersistentVolumeClaim",
                    metadata=k8s_resource_metadata,
                    spec=k8s_resource_spec
                )
            self.k8s_resource = k8s_resource

    def after(self, *newdeps, **kwargs):
        """Creates a duplicate of self with the required dependecies excluding
        the redundant dependenices. Optionally choose a new name.
        Args:
            *ops: Pipeline operators to add as dependencies
            **kwargs: Currently we support 'name' kwarg only
        """
        def implies(newdep, olddep):
            if newdep.name == olddep:
                return True
            for parentdep_name in newdep.deps:
                # If this dep is a PipelineVolume,
                # avoid never-ending recursion
                if parentdep_name == newdep.name:
                    continue
                if parentdep_name == olddep:
                    return True
                else:
                    parentdep = _pipeline.Pipeline.get_default_pipeline(
                    ).ops[parentdep_name]
                    if parentdep:
                        if implies(parentdep, olddep):
                            return True
            return False

        ret = self.__class__(
            pvc=self.pvc,
            k8s_resource=self.k8s_resource,
            create=False
        )
        ret.name = self.name

        for newdep in newdeps:
            ret.deps.add(newdep.name)
            for olddep in self.deps:
                if not implies(newdep, olddep):
                    ret.deps.add(olddep)

        name = self.name
        for key, value in kwargs:
            if key == 'name':
                param_info, name = self._parse_argument(str(name))
                if not param_info:
                    name = K8sHelper.sanitize_k8s_name(name)
            else:
                print("Ignoring '%s' kwarg: feature not available" % key)
        ret.name = name

        return ret

    def snapshot(self, name: str = None, snapshot_class: str = None):
        """Create a snapshot from this PipelineVolume"""
        return _pipeline_vsnapshot.PipelineVolumeSnapshot(
            pipeline_volume=self,
            name=name,
            snapshot_class=snapshot_class
        )

    def _validate_memory_string(self, memory_string):
        """Validate a given string is valid for memory request or limit."""
        if re.match(r'^[0-9]+(E|Ei|P|Pi|T|Ti|G|Gi|M|Mi|K|Ki){0,1}$',
                    memory_string) is None:
            raise ValueError('Invalid memory string. Should be an integer, ' +
                             'or integer followed by one of ' +
                             '"E|Ei|P|Pi|T|Ti|G|Gi|M|Mi|K|Ki"')

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
