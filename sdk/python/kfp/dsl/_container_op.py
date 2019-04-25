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

import re
import warnings
from typing import Any, Dict, List, TypeVar, Union, Callable
from kubernetes.client.models import (
    V1Container, V1EnvVar, V1EnvFromSource, V1SecurityContext, V1Probe,
    V1ResourceRequirements, V1VolumeDevice, V1VolumeMount, V1ContainerPort,
    V1Lifecycle, V1Volume
)

from . import _pipeline_param
from ._metadata import ComponentMeta

# generics
T = TypeVar('T')
# type alias: either a string or a list of string
StringOrStringList = Union[str, List[str]]


# util functions
def deprecation_warning(func: Callable, op_name: str,
                        container_name: str) -> Callable:
    """Decorator function to give a pending deprecation warning"""

    def _wrapped(*args, **kwargs):
        warnings.warn(
            '`dsl.ContainerOp.%s` will be removed in future releases. '
            'Use `dsl.ContainerOp.container.%s` instead.' %
            (op_name, container_name), PendingDeprecationWarning)
        return func(*args, **kwargs)

    return _wrapped


def _create_getter_setter(prop):
    """Create a tuple of getter and setter methods for a property in `Container`."""
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


def as_list(value: Any, if_none: Union[None, List] = None) -> List:
    """Convert any value except None to a list if not already a list."""
    if value is None:
        return if_none
    return value if isinstance(value, list) else [value]


def create_and_append(current_list: Union[List[T], None], item: T) -> List[T]:
    """Create a list (if needed) and appends an item to it."""
    current_list = current_list or []
    current_list.append(item)
    return current_list


class Container(V1Container):
    """
    A wrapper over k8s container definition object (io.k8s.api.core.v1.Container),
    which is used to represent the `container` property in argo's workflow 
    template (io.argoproj.workflow.v1alpha1.Template).
    
    `Container` class also comes with utility functions to set and update the 
    the various properties for a k8s container definition.

    NOTE: A notable difference is that `name` is not required and will not be 
    processed for `Container` (in contrast to `V1Container` where `name` is a
    required property). 

    See: 
    - https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_container.py
    - https://github.com/argoproj/argo/blob/master/api/openapi-spec/swagger.json

    
    Example:

      from kfp.dsl import ContainerOp
      from kubernetes.client.models import V1EnvVar
      

      # creates a operation
      op = ContainerOp(name='bash-ops', 
                       image='busybox:latest', 
                       command=['echo'], 
                       arguments=['$MSG'])

      # returns a `Container` object from `ContainerOp`
      # and add an environment variable to `Container`
      op.container.add_env_variable(V1EnvVar(name='MSG', value='hello world'))

    """
    """
    Attributes:
      swagger_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    # remove `name` from swagger_types so `name` is not generated in the JSON
    swagger_types = {
        key: value
        for key, value in V1Container.swagger_types.items() if key != 'name'
    }
    attribute_map = {
        key: value
        for key, value in V1Container.attribute_map.items() if key != 'name'
    }

    def __init__(self, image: str, command: List[str], args: List[str],
                 **kwargs):
        """Creates a new instance of `Container`.
        
        Args:
            image {str}: image to use, e.g. busybox:latest
            command {List[str]}: entrypoint array.  Not executed within a shell.
            args {List[str]}: arguments to entrypoint.
            **kwargs: keyword arguments for `V1Container`
        """
        # set name to '' if name is not provided
        # k8s container MUST have a name
        # argo workflow template does not need a name for container def
        if not kwargs.get('name'):
            kwargs['name'] = ''

        super(Container, self).__init__(
            image=image, command=command, args=args, **kwargs)

    def _validate_memory_string(self, memory_string):
        """Validate a given string is valid for memory request or limit."""

        if isinstance(memory_string, _pipeline_param.PipelineParam):
            if memory_string.value:
                memory_string = memory_string.value
            else:
                return

        if re.match(r'^[0-9]+(E|Ei|P|Pi|T|Ti|G|Gi|M|Mi|K|Ki){0,1}$',
                    memory_string) is None:
            raise ValueError(
                'Invalid memory string. Should be an integer, or integer followed '
                'by one of "E|Ei|P|Pi|T|Ti|G|Gi|M|Mi|K|Ki"')

    def _validate_cpu_string(self, cpu_string):
        "Validate a given string is valid for cpu request or limit."

        if isinstance(cpu_string, _pipeline_param.PipelineParam):
            if cpu_string.value:
                cpu_string = cpu_string.value
            else:
                return

        if re.match(r'^[0-9]+m$', cpu_string) is not None:
            return

        try:
            float(cpu_string)
        except ValueError:
            raise ValueError(
                'Invalid cpu string. Should be float or integer, or integer followed '
                'by "m".')

    def _validate_positive_number(self, str_value, param_name):
        "Validate a given string is in positive integer format."

        if isinstance(str_value, _pipeline_param.PipelineParam):
            if str_value.value:
                str_value = str_value.value
            else:
                return

        try:
            int_value = int(str_value)
        except ValueError:
            raise ValueError(
                'Invalid {}. Should be integer.'.format(param_name))

        if int_value <= 0:
            raise ValueError('{} must be positive integer.'.format(param_name))

    def add_resource_limit(self, resource_name, value):
        """Add the resource limit of the container.

        Args:
          resource_name: The name of the resource. It can be cpu, memory, etc.
          value: The string value of the limit.
        """

        self.resources = self.resources or V1ResourceRequirements()
        self.resources.limits = self.resources.limits or {}
        self.resources.limits.update({resource_name: value})
        return self

    def add_resource_request(self, resource_name, value):
        """Add the resource request of the container.

        Args:
          resource_name: The name of the resource. It can be cpu, memory, etc.
          value: The string value of the request.
        """

        self.resources = self.resources or V1ResourceRequirements()
        self.resources.requests = self.resources.requests or {}
        self.resources.requests.update({resource_name: value})
        return self

    def set_memory_request(self, memory):
        """Set memory request (minimum) for this operator.

        Args:
          memory: a string which can be a number or a number followed by one of
                  "E", "P", "T", "G", "M", "K".
        """

        self._validate_memory_string(memory)
        return self.add_resource_request("memory", memory)

    def set_memory_limit(self, memory):
        """Set memory limit (maximum) for this operator.

        Args:
          memory: a string which can be a number or a number followed by one of
                  "E", "P", "T", "G", "M", "K".
        """
        self._validate_memory_string(memory)
        return self.add_resource_limit("memory", memory)

    def set_cpu_request(self, cpu):
        """Set cpu request (minimum) for this operator.

        Args:
          cpu: A string which can be a number or a number followed by "m", which means 1/1000.
        """

        self._validate_cpu_string(cpu)
        return self.add_resource_request("cpu", cpu)

    def set_cpu_limit(self, cpu):
        """Set cpu limit (maximum) for this operator.

        Args:
          cpu: A string which can be a number or a number followed by "m", which means 1/1000.
        """

        self._validate_cpu_string(cpu)
        return self.add_resource_limit("cpu", cpu)

    def set_gpu_limit(self, gpu, vendor="nvidia"):
        """Set gpu limit for the operator. This function add '<vendor>.com/gpu' into resource limit. 
        Note that there is no need to add GPU request. GPUs are only supposed to be specified in 
        the limits section. See https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/.

        Args:
          gpu: A string which must be a positive number.
          vendor: Optional. A string which is the vendor of the requested gpu. The supported values 
            are: 'nvidia' (default), and 'amd'. 
        """

        self._validate_positive_number(gpu, 'gpu')
        if vendor != 'nvidia' and vendor != 'amd':
            raise ValueError('vendor can only be nvidia or amd.')

        return self.add_resource_limit("%s.com/gpu" % vendor, gpu)

    def add_volume_mount(self, volume_mount):
        """Add volume to the container

        Args:
          volume_mount: Kubernetes volume mount
          For detailed spec, check volume mount definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_volume_mount.py
        """

        if not isinstance(volume_mount, V1VolumeMount):
            raise ValueError(
                'invalid argument. Must be of instance `V1VolumeMount`.')

        self.volume_mounts = create_and_append(self.volume_mounts,
                                               volume_mount)
        return self

    def add_volume_devices(self, volume_device):
        """
        Add a block device to be used by the container.

        Args:
          volume_device: Kubernetes volume device
          For detailed spec, volume device definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_volume_device.py
        """

        if not isinstance(volume_device, V1VolumeDevice):
            raise ValueError(
                'invalid argument. Must be of instance `V1VolumeDevice`.')

        self.volume_devices = create_and_append(self.volume_devices,
                                                volume_device)
        return self

    def add_env_variable(self, env_variable):
        """Add environment variable to the container.

        Args:
          env_variable: Kubernetes environment variable
          For detailed spec, check environment variable definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_env_var.py
        """

        if not isinstance(env_variable, V1EnvVar):
            raise ValueError(
                'invalid argument. Must be of instance `V1EnvVar`.')

        self.env = create_and_append(self.env, env_variable)
        return self

    def add_env_from(self, env_from):
        """Add a source to populate environment variables int the container.

        Args:
          env_from: Kubernetes environment from source
          For detailed spec, check environment from source definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_env_var_source.py
        """

        if not isinstance(env_from, V1EnvFromSource):
            raise ValueError(
                'invalid argument. Must be of instance `V1EnvFromSource`.')

        self.env_from = create_and_append(self.env_from, env_from)
        return self

    def set_image_pull_policy(self, image_pull_policy):
        """Set image pull policy for the container.

        Args:
          image_pull_policy: One of `Always`, `Never`, `IfNotPresent`. 
        """
        if image_pull_policy not in ['Always', 'Never', 'IfNotPresent']:
            raise ValueError(
                'Invalid imagePullPolicy. Must be one of `Always`, `Never`, `IfNotPresent`.'
            )

        self.image_pull_policy = image_pull_policy
        return self

    def add_port(self, container_port):
        """Add a container port to the container.

        Args:
          container_port: Kubernetes container port
          For detailed spec, check container port definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_container_port.py
        """

        if not isinstance(container_port, V1ContainerPort):
            raise ValueError(
                'invalid argument. Must be of instance `V1ContainerPort`.')

        self.ports = create_and_append(self.ports, container_port)
        return self

    def set_security_context(self, security_context):
        """Set security configuration to be applied on the container.  

        Args:
          security_context: Kubernetes security context
          For detailed spec, check security context definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_security_context.py
        """

        if not isinstance(security_context, V1SecurityContext):
            raise ValueError(
                'invalid argument. Must be of instance `V1SecurityContext`.')

        self.security_context = security_context
        return self

    def set_stdin(self, stdin=True):
        """
        Whether this container should allocate a buffer for stdin in the container 
        runtime. If this is not set, reads from stdin in the container will always 
        result in EOF.

        Args:
          stdin: boolean flag
        """

        self.stdin = stdin
        return self

    def set_stdin_once(self, stdin_once=True):
        """
        Whether the container runtime should close the stdin channel after it has 
        been opened by a single attach. When stdin is true the stdin stream will 
        remain open across multiple attach sessions. If stdinOnce is set to true, 
        stdin is opened on container start, is empty until the first client attaches 
        to stdin, and then remains open and accepts data until the client 
        disconnects, at which time stdin is closed and remains closed until the 
        container is restarted. If this flag is false, a container processes that 
        reads from stdin will never receive an EOF. 

        Args:
          stdin_once: boolean flag
        """

        self.stdin_once = stdin_once
        return self

    def set_termination_message_path(self, termination_message_path):
        """
        Path at which the file to which the container's termination message will be 
        written is mounted into the container's filesystem. Message written is 
        intended to be brief final status, such as an assertion failure message. 
        Will be truncated by the node if greater than 4096 bytes. The total message 
        length across all containers will be limited to 12kb. 

        Args:
          termination_message_path: path for the termination message
        """
        self.termination_message_path = termination_message_path
        return self

    def set_termination_message_policy(self, termination_message_policy):
        """
        Indicate how the termination message should be populated. File will use the 
        contents of terminationMessagePath to populate the container status message 
        on both success and failure. FallbackToLogsOnError will use the last chunk 
        of container log output if the termination message file is empty and the 
        container exited with an error. The log output is limited to 2048 bytes or 
        80 lines, whichever is smaller.

        Args:
          termination_message_policy: `File` or `FallbackToLogsOnError`
        """
        if termination_message_policy not in ['File', 'FallbackToLogsOnError']:
            raise ValueError(
                'terminationMessagePolicy must be `File` or `FallbackToLogsOnError`'
            )
        self.termination_message_policy = termination_message_policy
        return self

    def set_tty(self, tty=True):
        """
        Whether this container should allocate a TTY for itself, also requires 
        'stdin' to be true.

        Args:
          tty: boolean flag
        """

        self.tty = tty
        return self

    def set_readiness_probe(self, readiness_probe):
        """
        Set a readiness probe for the container.

        Args:
          readiness_probe: Kubernetes readiness probe
          For detailed spec, check probe definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_probe.py
        """

        if not isinstance(readiness_probe, V1Probe):
            raise ValueError(
                'invalid argument. Must be of instance `V1Probe`.')

        self.readiness_probe = readiness_probe
        return self

    def set_liveness_probe(self, liveness_probe):
        """
        Set a liveness probe for the container.

        Args:
          liveness_probe: Kubernetes liveness probe
          For detailed spec, check probe definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_probe.py
        """

        if not isinstance(liveness_probe, V1Probe):
            raise ValueError(
                'invalid argument. Must be of instance `V1Probe`.')

        self.liveness_probe = liveness_probe
        return self

    def set_lifecycle(self, lifecycle):
        """
        Setup a lifecycle config for the container.

        Args:
          lifecycle: Kubernetes lifecycle
          For detailed spec, lifecycle definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_lifecycle.py
        """

        if not isinstance(lifecycle, V1Lifecycle):
            raise ValueError(
                'invalid argument. Must be of instance `V1Lifecycle`.')

        self.lifecycle = lifecycle
        return self


class Sidecar(Container):
    """
    Represents an argo workflow sidecar (io.argoproj.workflow.v1alpha1.Sidecar) 
    to be used in `sidecars` property in argo's workflow template 
    (io.argoproj.workflow.v1alpha1.Template). 

    `Sidecar` inherits from `Container` class with an addition of `mirror_volume_mounts`
    attribute (`mirrorVolumeMounts` property).

    See https://github.com/argoproj/argo/blob/master/api/openapi-spec/swagger.json

    Example

        from kfp.dsl import ContainerOp, Sidecar

        # creates a `ContainerOp` and adds a redis `Sidecar`
        op = (ContainerOp(name='foo-op', image='busybox:latest')
                .add_sidecar(
                    Sidecar(name='redis', image='redis:alpine')))

    """
    """
    Attributes:
      swagger_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    # adds `mirror_volume_mounts` to `Sidecar` swagger definition
    # NOTE inherits definition from `V1Container` rather than `Container`
    #      because `Container` has no `name` property.
    swagger_types = dict(
        **V1Container.swagger_types, mirror_volume_mounts='bool')

    attribute_map = dict(
        **V1Container.attribute_map, mirror_volume_mounts='mirrorVolumeMounts')

    def __init__(self,
                 name: str,
                 image: str,
                 command: StringOrStringList = None,
                 args: StringOrStringList = None,
                 mirror_volume_mounts: bool = None,
                 **kwargs):
        """Creates a new instance of `Sidecar`.
        
        Args:
            name {str}: unique name for the sidecar container
            image {str}: image to use for the sidecar container, e.g. redis:alpine
            command {StringOrStringList}: entrypoint array.  Not executed within a shell.
            args {StringOrStringList}: arguments to the entrypoint. 
            mirror_volume_mounts {bool}: MirrorVolumeMounts will mount the same 
                volumes specified in the main container to the sidecar (including artifacts), 
                at the same mountPaths. This enables dind daemon to partially see the same 
                filesystem as the main container in order to use features such as docker 
                volume binding
            **kwargs: keyword arguments available for `Container` 

        """
        super().__init__(
            name=name,
            image=image,
            command=as_list(command),
            args=as_list(args),
            **kwargs)

        self.mirror_volume_mounts = mirror_volume_mounts

    def set_mirror_volume_mounts(self, mirror_volume_mounts=True):
        """
        Setting mirrorVolumeMounts to true will mount the same volumes specified 
        in the main container to the sidecar (including artifacts), at the same 
        mountPaths. This enables dind daemon to partially see the same filesystem 
        as the main container in order to use features such as docker volume 
        binding.
        
        Args:
            mirror_volume_mounts: boolean flag
        """

        self.mirror_volume_mounts = mirror_volume_mounts
        return self

    @property
    def inputs(self):
        """A list of PipelineParam found in the Sidecar object."""
        return _pipeline_param.extract_pipelineparams_from_any(self)


def _make_hash_based_id_for_op(op):
    # Generating a unique ID for Op. For class instances, the hash is the object's memory address which is unique.
    return op.human_name + ' ' + hex(2**63 + hash(op))[2:]


# Pointer to a function that generates a unique ID for the Op instance (Possibly by registering the Op instance in some system).
_register_op_handler = _make_hash_based_id_for_op


class BaseOp(object):

    # list of attributes that might have pipeline params - used to generate
    # the input parameters during compilation.
    # Excludes `file_outputs` and `outputs` as they are handled separately
    # in the compilation process to generate the DAGs and task io parameters.
    attrs_with_pipelineparams = [
        'node_selector', 'volumes', 'pod_annotations', 'pod_labels',
        'num_retries', 'sidecars'
    ]

    def __init__(self,
                 name: str,
                 sidecars: List[Sidecar] = None,
                 is_exit_handler: bool = False):
        """Create a new instance of BaseOp

        Args:
          name: the name of the op. It does not have to be unique within a pipeline
              because the pipeline will generates a unique new name in case of conflicts.
          sidecars: the list of `Sidecar` objects describing the sidecar containers to deploy 
                    together with the `main` container.
          is_exit_handler: Whether it is used as an exit handler.
        """

        valid_name_regex = r'^[A-Za-z][A-Za-z0-9\s_-]*$'
        if not re.match(valid_name_regex, name):
            raise ValueError(
                'Only letters, numbers, spaces, "_", and "-"  are allowed in name. Must begin with letter: %s'
                % (name))

        self.is_exit_handler = is_exit_handler

        # human_name must exist to construct operator's name
        self.human_name = name
        # ID of the current Op. Ideally, it should be generated by the compiler that sees the bigger context.
        # However, the ID is used in the task output references (PipelineParams) which can be serialized to strings.
        # Because of this we must obtain a unique ID right now.
        self.name = _register_op_handler(self)

        # TODO: proper k8s definitions so that `convert_k8s_obj_to_json` can be used?
        # `io.argoproj.workflow.v1alpha1.Template` properties
        self.node_selector = {}
        self.volumes = []
        self.pod_annotations = {}
        self.pod_labels = {}
        self.num_retries = 0
        self.sidecars = sidecars or []

        # attributes specific to `BaseOp`
        self._inputs = []
        self.dependent_names = []

    @property
    def inputs(self):
        """List of PipelineParams that will be converted into input parameters
        (io.argoproj.workflow.v1alpha1.Inputs) for the argo workflow.
        """
        # Iterate through and extract all the `PipelineParam` in Op when
        # called the 1st time (because there are in-place updates to `PipelineParam`
        # during compilation - remove in-place updates for easier debugging?)
        if not self._inputs:
            self._inputs = []
            # TODO replace with proper k8s obj?
            for key in self.attrs_with_pipelineparams:
                self._inputs += [
                    param for param in _pipeline_param.
                    extract_pipelineparams_from_any(getattr(self, key))
                ]
            # keep only unique
            self._inputs = list(set(self._inputs))
        return self._inputs

    @inputs.setter
    def inputs(self, value):
        # to support in-place updates
        self._inputs = value

    def apply(self, mod_func):
        """Applies a modifier function to self. The function should return the passed object.
        This is needed to chain "extention methods" to this class.

        Example:
          from kfp.gcp import use_gcp_secret
          task = (
            train_op(...)
              .set_memory_request('1GB')
              .apply(use_gcp_secret('user-gcp-sa'))
              .set_memory_limit('2GB')
          )
        """
        return mod_func(self)

    def after(self, op):
        """Specify explicit dependency on another op."""
        self.dependent_names.append(op.name)
        return self

    def add_volume(self, volume):
        """Add K8s volume to the container

        Args:
          volume: Kubernetes volumes
          For detailed spec, check volume definition
          https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_volume.py
        """
        self.volumes.append(volume)
        return self

    def add_node_selector_constraint(self, label_name, value):
        """Add a constraint for nodeSelector. Each constraint is a key-value pair label. For the 
        container to be eligible to run on a node, the node must have each of the constraints appeared
        as labels.

        Args:
          label_name: The name of the constraint label.
          value: The value of the constraint label.
        """

        self.node_selector[label_name] = value
        return self

    def add_pod_annotation(self, name: str, value: str):
        """Adds a pod's metadata annotation.

        Args:
          name: The name of the annotation.
          value: The value of the annotation.
        """

        self.pod_annotations[name] = value
        return self

    def add_pod_label(self, name: str, value: str):
        """Adds a pod's metadata label.

        Args:
          name: The name of the label.
          value: The value of the label.
        """

        self.pod_labels[name] = value
        return self

    def set_retry(self, num_retries: int):
        """Sets the number of times the task is retried until it's declared failed.

        Args:
          num_retries: Number of times to retry on failures.
        """

        self.num_retries = num_retries
        return self

    def add_sidecar(self, sidecar: Sidecar):
        """Add a sidecar to the Op.

        Args:
          sidecar: SideCar object.
        """

        self.sidecars.append(sidecar)
        return self

    def __repr__(self):
        return str({self.__class__.__name__: self.__dict__})


from ._pipeline_volume import PipelineVolume #The import is here to prevent circular reference problems.


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
                 name: str,
                 image: str,
                 command: StringOrStringList = None,
                 arguments: StringOrStringList = None,
                 sidecars: List[Sidecar] = None,
                 container_kwargs: Dict = None,
                 file_outputs: Dict[str, str] = None,
                 output_artifact_paths : Dict[str, str]=None,
                 is_exit_handler=False,
                 pvolumes: Dict[str, V1Volume] = None,
        ):
        """Create a new instance of ContainerOp.

        Args:
          name: the name of the op. It does not have to be unique within a pipeline
              because the pipeline will generates a unique new name in case of conflicts.
          image: the container image name, such as 'python:3.5-jessie'
          command: the command to run in the container.
              If None, uses default CMD in defined in container.
          arguments: the arguments of the command. The command can include "%s" and supply
              a PipelineParam as the string replacement. For example, ('echo %s' % input_param).
              At container run time the argument will be 'echo param_value'.
          sidecars: the list of `Sidecar` objects describing the sidecar containers to deploy 
                    together with the `main` container.
          container_kwargs: the dict of additional keyword arguments to pass to the
                            op's `Container` definition.
          file_outputs: Maps output labels to local file paths. At pipeline run time,
              the value of a PipelineParam is saved to its corresponding local file. It's
              one way for outside world to receive outputs of the container.
          output_artifact_paths: Maps output artifact labels to local artifact file paths.
              It has the following default artifact paths during compile time.
              {'mlpipeline-ui-metadata': '/mlpipeline-ui-metadata.json',
               'mlpipeline-metrics': '/mlpipeline-metrics.json'}
          is_exit_handler: Whether it is used as an exit handler.
          pvolumes: Dictionary for the user to match a path on the op's fs with a
              V1Volume or it inherited type.
              E.g {"/my/path": vol, "/mnt": other_op.volumes["/output"]}.
        """

        super().__init__(name=name, sidecars=sidecars, is_exit_handler=is_exit_handler)
        self.attrs_with_pipelineparams = BaseOp.attrs_with_pipelineparams + ['_container'] #Copying the BaseOp class variable!

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
        self.output_artifact_paths = output_artifact_paths or {}

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

        self.pvolumes = {}
        if pvolumes:
            for mount_path, pvolume in pvolumes.items():
                if hasattr(pvolume, "dependent_names"): #TODO: Replace with type check
                    self.dependent_names.extend(pvolume.dependent_names)
                else:
                    pvolume = PipelineVolume(volume=pvolume)
                self.pvolumes[mount_path] = pvolume.after(self)
                self.add_volume(pvolume)
                self._container.add_volume_mount(V1VolumeMount(
                    name=pvolume.name,
                    mount_path=mount_path
                ))

        self.pvolume = None
        if self.pvolumes and len(self.pvolumes) == 1:
            self.pvolume = list(self.pvolumes.values())[0]

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