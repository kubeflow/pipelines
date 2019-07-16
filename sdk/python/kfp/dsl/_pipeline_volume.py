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


import hashlib
import json

from kubernetes.client.models import (
    V1Volume, V1PersistentVolumeClaimVolumeSource
)

from . import _pipeline


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
            ValueError: if volume is not None and kwargs is not None
                        if pvc is not None and kwargs.pop("name") is not None
        """
        if volume and kwargs:
            raise ValueError("You can't pass a volume along with other "
                             "kwargs.")

        name_provided = True
        init_volume = {}
        if volume:
            init_volume = {attr: getattr(volume, attr)
                           for attr in self.attribute_map.keys()}
        else:
            if "name" in kwargs:
                init_volume = {"name": kwargs.pop("name")}
            else:
                name_provided = False
                init_volume = {"name": "pvolume-placeholder"}
            if pvc and kwargs:
                raise ValueError("You can only pass 'name' along with 'pvc'.")
            elif pvc and not kwargs:
                pvc_volume_source = V1PersistentVolumeClaimVolumeSource(
                    claim_name=pvc
                )
                init_volume["persistent_volume_claim"] = pvc_volume_source
        super().__init__(**init_volume, **kwargs)
        if not name_provided:
            self.name = "pvolume-%s" % hashlib.sha256(
                bytes(json.dumps(self.to_dict(), sort_keys=True), "utf-8")
            ).hexdigest()
        self.dependent_names = []

    def after(self, *ops):
        """Creates a duplicate of self with the required dependecies excluding
        the redundant dependenices.
        Args:
            *ops: Pipeline operators to add as dependencies
        """
        def implies(newdep, olddep):
            if newdep.name == olddep:
                return True
            for parentdep_name in newdep.dependent_names:
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
        ret.dependent_names = [op.name for op in ops]

        for olddep in self.dependent_names:
            implied = False
            for newdep in ops:
                implied = implies(newdep, olddep)
                if implied:
                    break
            if not implied:
                ret.dependent_names.append(olddep)

        return ret
