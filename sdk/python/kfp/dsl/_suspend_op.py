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


from typing import Dict, Optional

from ._container_op import BaseOp
from . import _pipeline_param


class Suspend(object):
    """
    A wrapper over Argo SuspendTemplate definition object
    (io.argoproj.workflow.v1alpha1.SuspendTemplate)
    which is used to represent the `suspend` property in argo's workflow
    template (io.argoproj.workflow.v1alpha1.Template).
    """
    """
    Attributes:
      swagger_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    swagger_types = {
        "duration": "str"
    }
    attribute_map = {
        "duration": "duration"
    }

    def __init__(self,
                 duration: str = None):
        """Create a new instance of Suspend"""
        self.duration = duration


class SuspendOp(BaseOp):
    """Represents an op which will be translated into a suspend template"""

    def __init__(self,
                 duration: Optional[int] = None,
                 **kwargs):
        """Create a new instance of SuspendOp.

        Args:
            duration: Amount of time in seconds to wait until automtically resume the workflow.
                In case of None it will wait for manual resume.
            kwargs: name, sidecars. See BaseOp definition
        Raises:
        ValueError: if not inside a pipeline
                    if the name is an invalid string
                    if the duration is set and is not an integer
        """

        super().__init__(**kwargs)
        self.attrs_with_pipelineparams = list(self.attrs_with_pipelineparams)
        self.attrs_with_pipelineparams.extend([
            "_suspend"
        ])

        if duration and not isinstance(duration, int):
            raise ValueError("Duration must be an integer or None.")

        init_suspend = {
            "duration": str(duration) if duration else ''
        }
        # `suspend` prop in `io.argoproj.workflow.v1alpha1.Template`
        self._suspend = Suspend(**init_suspend)

        self.attribute_outputs = {}
        self.outputs = {}
        self.output = None

    @property
    def suspend(self):
        """`Suspend` object that represents the `suspend` property in
        `io.argoproj.workflow.v1alpha1.Template`.
        """
        return self._suspend
