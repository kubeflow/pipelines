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
from kubernetes.client.models import V1Volume, V1VolumeMount

from ._base_op import (
    StringOrStringList, deprecation_warning, as_list, Container, BaseOp
)
from . import _pipeline_param
from ._metadata import ComponentMeta


def _create_getter_setter(prop):
    """
    Create a tuple of getter and setter methods for a property in `Container`.
    """
    def _getter(self):
        return getattr(self._container, prop)
    def _setter(self, value):
        return setattr(self._container, prop, value)
    return _getter, _setter


def _proxy_container_op_props(cls: "ContainerOp"):
    """Takes the `ContainerOp` class and proxy the PendingDeprecation properties
    in `ContainerOp` to the `Container` instance.
    """
    # properties mapping to proxy: ContainerOps.<prop> => Container.<prop>
    prop_map = dict(image='image', env_variables='env')
    # itera and create class props
    for op_prop, container_prop in prop_map.items():
        # create getter and setter
        _getter, _setter = _create_getter_setter(container_prop)
        # decorate with deprecation warning
        getter = deprecation_warning(_getter, op_prop, container_prop)
        setter = deprecation_warning(_setter, op_prop, container_prop)
        # update attribites with properties
        setattr(cls, op_prop, property(getter, setter))
    return cls


class ContainerOp(BaseOp):
    """
    Represents an op implemented by a container image.

    Example

        from kfp import dsl
        from kubernetes.client.models import V1EnvVar


        @dsl.pipeline(
            name='foo',
            description='hello world')
        def foo_pipeline(tag: str, pull_image_policy: str):

            # any attributes can be parameterized (both serialized string or actual PipelineParam)
            op = dsl.ContainerOp(name='foo', 
                                image='busybox:%s' % tag,
                                # pass in sidecars list
                                sidecars=[dsl.Sidecar('print', 'busybox:latest', command='echo "hello"')],
                                # pass in k8s container kwargs
                                container_kwargs={'env': [V1EnvVar('foo', 'bar')]})

            # set `imagePullPolicy` property for `container` with `PipelineParam` 
            op.container.set_pull_image_policy(pull_image_policy)

            # add sidecar with parameterized image tag
            # sidecar follows the argo sidecar swagger spec
            op.add_sidecar(dsl.Sidecar('redis', 'redis:%s' % tag).set_image_pull_policy('Always'))
    
    """

    # list of attributes that might have pipeline params - used to generate
    # the input parameters during compilation.
    # Excludes `file_outputs` and `outputs` as they are handled separately
    # in the compilation process to generate the DAGs and task io parameters.

    def __init__(self,
                 image: str,
                 command: StringOrStringList = None,
                 arguments: StringOrStringList = None,
                 container_kwargs: Dict = None,
                 file_outputs: Dict[str, str] = None,
                 volumes: Dict[str, V1Volume] = None,
                 **kwargs):
        """Create a new instance of ContainerOp.

        Args:
          image: the container image name, such as 'python:3.5-jessie'
          command: the command to run in the container.
              If None, uses default CMD in defined in container.
          arguments: the arguments of the command. The command can include "%s" and supply
              a PipelineParam as the string replacement. For example, ('echo %s' % input_param).
              At container run time the argument will be 'echo param_value'.
          container_kwargs: the dict of additional keyword arguments to pass to the
                            op's `Container` definition.
          file_outputs: Maps output labels to local file paths. At pipeline run time,
              the value of a PipelineParam is saved to its corresponding local file. It's
              one way for outside world to receive outputs of the container.
          volumes: Dictionary for the user to match a path on the op's fs with a
              V1Volume or it inherited type.
              E.g {"/my/path": vol, "/mnt": other_op.volumes["/output"]}.
          kwargs: name, sidecars & is_exit_hander. See BaseOp definition
        """

        super().__init__(**kwargs)
        self.attrs_with_pipelineparams = list(self.attrs_with_pipelineparams)
        self.attrs_with_pipelineparams.append('_container')

        # convert to list if not a list
        command = as_list(command)
        arguments = as_list(arguments)

        # `container` prop in `io.argoproj.workflow.v1alpha1.Template`
        container_kwargs = container_kwargs or {}
        self._container = Container(
            image=image, args=arguments, command=command, **container_kwargs)

        # NOTE for backward compatibility (remove in future?)
        # proxy old ContainerOp callables to Container

        # attributes to NOT proxy
        ignore_set = frozenset(['to_dict', 'to_str'])

        # decorator func to proxy a method in `Container` into `ContainerOp`
        def _proxy(proxy_attr):
            """Decorator func to proxy to ContainerOp.container"""

            def _decorated(*args, **kwargs):
                # execute method
                ret = getattr(self._container, proxy_attr)(*args, **kwargs)
                if ret == self._container:
                    return self
                return ret

            return deprecation_warning(_decorated, proxy_attr, proxy_attr)

        # iter thru container and attach a proxy func to the container method
        for attr_to_proxy in dir(self._container):
            func = getattr(self._container, attr_to_proxy)
            # ignore private methods
            if hasattr(func, '__call__') and (attr_to_proxy[0] != '_') and (
                    attr_to_proxy not in ignore_set):
                # only proxy public callables
                setattr(self, attr_to_proxy, _proxy(attr_to_proxy))

        # attributes specific to `ContainerOp`
        self.file_outputs = file_outputs
        self._metadata = None

        self.outputs = {}
        if file_outputs:
            self.outputs = {
                name: _pipeline_param.PipelineParam(name, op_name=self.name)
                for name in file_outputs.keys()
            }

        self.output = None
        if len(self.outputs) == 1:
            self.output = list(self.outputs.values())[0]

        self.volumes = volumes
        if volumes:
            for mount_path, volume in volumes.items():
                self.add_volume(volume)
                self._container.add_volume_mount(V1VolumeMount(
                    name=volume.name,
                    mount_path=mount_path
                ))

        self.volume = None
        if self.volumes and len(self.volumes) == 1:
            self.volume = list(self.volumes.values())[0]

    @property
    def command(self):
        return self._container.command

    @command.setter
    def command(self, value):
        self._container.command = as_list(value)

    @property
    def arguments(self):
        return self._container.args

    @arguments.setter
    def arguments(self, value):
        self._container.args = as_list(value)

    @property
    def container(self):
        """`Container` object that represents the `container` property in 
        `io.argoproj.workflow.v1alpha1.Template`. Can be used to update the
        container configurations. 
        
        Example:
            import kfp.dsl as dsl
            from kubernetes.client.models import V1EnvVar
    
            @dsl.pipeline(name='example_pipeline')
            def immediate_value_pipeline():
                op1 = (dsl.ContainerOp(name='example', image='nginx:alpine')
                          .container
                            .add_env_variable(V1EnvVar(name='HOST', value='foo.bar'))
                            .add_env_variable(V1EnvVar(name='PORT', value='80'))
                            .parent # return the parent `ContainerOp`
                        )
        """
        return self._container

    def _set_metadata(self, metadata):
        '''_set_metadata passes the containerop the metadata information
        and configures the right output
        Args:
          metadata (ComponentMeta): component metadata
        '''
        if not isinstance(metadata, ComponentMeta):
            raise ValueError('_set_medata is expecting ComponentMeta.')

        self._metadata = metadata

        if self.file_outputs:
            for output in self.file_outputs.keys():
                output_type = self.outputs[output].param_type
                for output_meta in self._metadata.outputs:
                    if output_meta.name == output:
                        output_type = output_meta.param_type
                self.outputs[output].param_type = output_type

            self.output = None
            if len(self.outputs) == 1:
                self.output = list(self.outputs.values())[0]

# proxy old ContainerOp properties to ContainerOp.container
# with PendingDeprecationWarning.
ContainerOp = _proxy_container_op_props(ContainerOp)
