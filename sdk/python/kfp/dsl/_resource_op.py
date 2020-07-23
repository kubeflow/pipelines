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


from typing import Dict

from ._container_op import BaseOp
from . import _pipeline_param


class Resource(object):
    """
    A wrapper over Argo ResourceTemplate definition object
    (io.argoproj.workflow.v1alpha1.ResourceTemplate)
    which is used to represent the `resource` property in argo's workflow
    template (io.argoproj.workflow.v1alpha1.Template).
    """
    swagger_types = {
        "action": "str",
        "merge_strategy": "str",
        "success_condition": "str",
        "failure_condition": "str",
        "manifest": "str"
    }
    openapi_types = {
        "action": "str",
        "merge_strategy": "str",
        "success_condition": "str",
        "failure_condition": "str",
        "manifest": "str"
    }
    attribute_map = {
        "action": "action",
        "merge_strategy": "mergeStrategy",
        "success_condition": "successCondition",
        "failure_condition": "failureCondition",
        "manifest": "manifest"
    }

    def __init__(self,
                 action: str = None,
                 merge_strategy: str = None,
                 success_condition: str = None,
                 failure_condition: str = None,
                 manifest: str = None):
        """Create a new instance of Resource"""
        self.action = action
        self.merge_strategy = merge_strategy
        self.success_condition = success_condition
        self.failure_condition = failure_condition
        self.manifest = manifest


class ResourceOp(BaseOp):
    """Represents an op which will be translated into a resource template

    Args:
        k8s_resource: A k8s resource which will be submitted to the cluster
        action: One of "create"/"delete"/"apply"/"patch"
            (default is "create")
        merge_strategy: The merge strategy for the "apply" action
        success_condition: The successCondition of the template
        failure_condition: The failureCondition of the template
            For more info see:
            https://github.com/argoproj/argo/blob/master/examples/k8s-jobs.yaml
        attribute_outputs: Maps output labels to resource's json paths,
            similarly to file_outputs of ContainerOp
        kwargs: name, sidecars. See BaseOp definition

    Raises:
        ValueError: if not inside a pipeline
            if the name is an invalid string
            if no k8s_resource is provided
            if merge_strategy is set without "apply" action
    """

    def __init__(self,
                 k8s_resource=None,
                 action: str = "create",
                 merge_strategy: str = None,
                 success_condition: str = None,
                 failure_condition: str = None,
                 attribute_outputs: Dict[str, str] = None,
                 **kwargs):

        super().__init__(**kwargs)
        self.attrs_with_pipelineparams = list(self.attrs_with_pipelineparams)
        self.attrs_with_pipelineparams.extend([
            "_resource", "k8s_resource", "attribute_outputs"
        ])

        if k8s_resource is None:
            raise ValueError("You need to provide a k8s_resource.")

        if merge_strategy and action != "apply":
            raise ValueError("You can't set merge_strategy when action != 'apply'")
        
        # if action is delete, there should not be any outputs, success_condition, and failure_condition
        if action == "delete" and (success_condition or failure_condition or attribute_outputs):
            raise ValueError("You can't set success_condition, failure_condition, or attribute_outputs when action == 'delete'")

        init_resource = {
            "action": action,
            "merge_strategy": merge_strategy,
            "success_condition": success_condition,
            "failure_condition": failure_condition
        }
        # `resource` prop in `io.argoproj.workflow.v1alpha1.Template`
        self._resource = Resource(**init_resource)

        self.k8s_resource = k8s_resource

        # if action is delete, there should not be any outputs, success_condition, and failure_condition
        if action == "delete":
            self.attribute_outputs = {}
            self.outputs = {}
            self.output = None
            return

        # Set attribute_outputs
        extra_attribute_outputs = \
            attribute_outputs if attribute_outputs else {}
        self.attribute_outputs = \
            self.attribute_outputs if hasattr(self, "attribute_outputs") \
            else {}
        self.attribute_outputs.update(extra_attribute_outputs)
        # Add name and manifest if not specified by the user
        if "name" not in self.attribute_outputs:
            self.attribute_outputs["name"] = "{.metadata.name}"
        if "manifest" not in self.attribute_outputs:
            self.attribute_outputs["manifest"] = "{}"

        # Set outputs
        self.outputs = {
            name: _pipeline_param.PipelineParam(name, op_name=self.name)
            for name in self.attribute_outputs.keys()
        }
        # If user set a single attribute_output, set self.output as that
        # parameter, else set it as the resource name
        self.output = self.outputs["name"]
        if len(extra_attribute_outputs) == 1:
            self.output = self.outputs[list(extra_attribute_outputs)[0]]

    @property
    def resource(self):
        """`Resource` object that represents the `resource` property in
        `io.argoproj.workflow.v1alpha1.Template`.
        """
        return self._resource

    def delete(self):
        """Returns a ResourceOp which deletes the resource."""
        if self.resource.action == "delete":
            raise ValueError("This operation is already a resource deletion.")

        k8s_resource = dict()
        if isinstance(self.k8s_resource, dict):
            k8s_resource["apiVersion"] = self.k8s_resource["apiVersion"]
            k8s_resource["kind"] = self.k8s_resource["kind"]
        else:
            k8s_resource["apiVersion"] = self.k8s_resource.api_version
            k8s_resource["kind"] = self.k8s_resource.kind
        k8s_resource["metadata"] = {"name": self.outputs["name"]}

        return ResourceOp(name="del-%s" % self.name,
                          action="delete",
                          k8s_resource=k8s_resource)
