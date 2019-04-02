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


from kubernetes.client.models import (
    V1Volume, V1PersistentVolumeClaimVolumeSource,
    V1ObjectMeta, V1TypedLocalObjectReference
)

from . import _pipeline
from ._pipeline_param import sanitize_k8s_name, match_serialized_pipelineparam
from ._volume_snapshot_op import VolumeSnapshotOp


class PipelineVolume(V1Volume):
    """Representing a volume that is passed between pipeline operators and is
    to be mounted by a ContainerOp or its inherited type.

    A PipelineVolume object can be used as an extention of the pipeline
    function's filesystem. It may then be passed between ContainerOps,
    exposing dependencies.
    """
    def __init__(self,
                 pvc: str = None,
                 volume: V1Volume = None,
                 **kwargs):
        """Create a new instance of PipelineVolume.

        Args:
            pvc: The name of an existing PVC
            volume: Create a deep copy out of a V1Volume or PipelineVolume
                with no deps
        Raises:
            ValueError: if pvc is not None and name is None
                        if volume is not None and kwargs is not None
                        if pvc is not None and kwargs.pop("name") is not None
        """
        if pvc and "name" not in kwargs:
            raise ValueError("Please provide name.")
        elif volume and kwargs:
            raise ValueError("You can't pass a volume along with other "
                             "kwargs.")

        init_volume = {}
        if volume:
            init_volume = {attr: getattr(volume, attr)
                           for attr in self.attribute_map.keys()}
        else:
            init_volume = {"name": kwargs.pop("name")
                           if "name" in kwargs else None}
            if pvc and kwargs:
                raise ValueError("You can only pass 'name' along with 'pvc'.")
            elif pvc and not kwargs:
                pvc_volume_source = V1PersistentVolumeClaimVolumeSource(
                    claim_name=pvc
                )
                init_volume["persistent_volume_claim"] = pvc_volume_source
        super().__init__(**init_volume, **kwargs)
        self.deps = []

    def after(self, *ops):
        """Creates a duplicate of self with the required dependecies excluding
        the redundant dependenices.
        Args:
            *ops: Pipeline operators to add as dependencies
        """
        def implies(newdep, olddep):
            if newdep.name == olddep:
                return True
            for parentdep_name in newdep.deps:
                if parentdep_name == olddep:
                    return True
                else:
                    parentdep = _pipeline.Pipeline.get_default_pipeline(
                    ).ops[parentdep_name]
                    if parentdep:
                        if implies(parentdep, olddep):
                            return True
            return False

        ret = self.__class__(volume=self)
        ret.deps = [op.name for op in ops]

        for olddep in self.deps:
            implied = False
            for newdep in ops:
                implied = implies(newdep, olddep)
                if implied:
                    break
            if not implied:
                ret.deps.append(olddep)

        return ret

    def snapshot(self,
                 resource_name: str = None,
                 name: str = None):
        """Create a VolumeSnapshot out of this PipelineVolume"""
        if not hasattr(self, "persistent_volume_claim"):
            raise ValueError("The volume must be referencing a PVC.")

        # Generate VolumeSnapshotOp name
        if not name:
            name = "%s-snapshot" % self.name
        # Generate VolumeSnapshot name
        if not resource_name:
            tmp_name = self.persistent_volume_claim.claim_name
            if self.deps:
                tmp_name = "%s-after" % tmp_name
                for dep in sorted(self.deps):
                    tmp_name = "%s-%s" % (tmp_name, sanitize_k8s_name(dep))
            resource_name = "%s-snapshot" % tmp_name

        # Create the k8s_resource
        if not match_serialized_pipelineparam(str(resource_name)):
            resource_name = sanitize_k8s_name(resource_name)
        snapshot_metadata = V1ObjectMeta(
            name=resource_name,
        )
        source = V1TypedLocalObjectReference(
            kind="PersistentVolumeClaim",
            name=self.persistent_volume_claim.claim_name
        )
        k8s_resource = {
            "apiVersion": "snapshot.storage.k8s.io/v1alpha1",
            "kind": "VolumeSnapshot",
            "metadata": snapshot_metadata,
            "spec": {"source": source}
        }
        VSOp = VolumeSnapshotOp(
            name=name,
            k8s_resource=k8s_resource
        )
        VSOp.deps.extend(self.deps)

        return VSOp
