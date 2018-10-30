# Copyright 2018 Google LLC
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

__all__ = [
    'InputOrOutputSpec',
    'InputSpec',
    'OutputSpec',
    'DockerContainerSpec',
    'GraphInputReferenceSpec',
    'TaskOutputReferenceSpec',
    'DataValueOrReferenceSpec',
    'TaskSpec',
    'GraphSpec',
    'ImplementationSpec',
    'SourceSpec',
    'ComponentSpec',
]


import copy
from collections import OrderedDict
from typing import Union, List, Sequence, Mapping, Tuple


class InputOrOutputSpec:
    def __init__(self, name:str, type:str=None, description:str=None, required:bool=True, pattern:str=None):
        if not isinstance(name, str):
            raise ValueError('name must be a string')
        self.name = name
        self.type = type
        self.description = description
        self.required = required
        self.pattern = pattern

    @classmethod
    def from_struct(cls, struct:Union[Tuple[str, Mapping],Mapping[str,Mapping],str]):
        #if not isinstance(struct, tuple) and not isinstance(struct, dict) and not isinstance(struct, str):
        #    raise ValueError('InputOrOutputSpec.from_struct only supports tuples, dicts and strings')
        
        #We support two different serialization variants:
        #1: {name: {'type': type}, ...}
        #2: [{'name': name, 'type': type}, ...]
        #1st one looks nicer, but we must take care to preserve the port ordering (prior to version 3.6 Python's dict does not preserve the order of elements).
        if isinstance(struct, tuple): #(name: {'type': type})
            assert(len(struct) == 2)
            (name, spec_dict) = struct
        elif isinstance(struct, dict): #{'name': name, 'type': type}
            spec_dict = copy.deepcopy(struct)
            name = spec_dict.pop('name')
        elif isinstance(struct, str):
            name = struct
            spec_dict = {}
        else:
            raise ValueError('InputOrOutputSpec.from_struct only supports tuples, dicts and strings')
        #port_spec = InputOrOutputSpec(name)
        port_spec = cls(name)
        
        if 'type' in spec_dict:
            port_spec.type = spec_dict.pop('type')
            check_instance_type(port_spec.type, [str, list]) #TODO: Check format further
        
        if 'description' in spec_dict:
            port_spec.description = str(spec_dict.pop('description'))

        if 'required' in spec_dict:
            port_spec.required = bool(spec_dict.pop('required'))
        
        if 'pattern' in spec_dict:
            port_spec.pattern = str(spec_dict.pop('pattern'))
        
        if spec_dict:
            raise ValueError('Found unrecognized properties: {}'.format(spec_dict))
        
        return port_spec
        
    def to_struct(self):
        struct = OrderedDict()
        if self.type:
            struct['type'] = self.type
        if self.description:
            struct['description'] = self.description
        if self.required != True: #Only outputting when not default
            print(self.required)
            struct['required'] = self.required
        if self.pattern:
            struct['pattern'] = self.pattern
        
        return (self.name, struct)
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'


class InputSpec(InputOrOutputSpec):
    pass


class OutputSpec(InputOrOutputSpec):
    pass


class DockerContainerSpec:
    def __init__(self, image:str, command:List=None, arguments:List=None, file_outputs:Mapping[str,str]=None):
        if not isinstance(image, str):
            raise ValueError('image must be a string')
        self.image = image
        self.command = command
        self.arguments = arguments
        self.file_outputs = file_outputs
    
    @staticmethod
    def from_struct(spec_dict:Mapping):
        spec_dict = copy.deepcopy(spec_dict)
        
        image = spec_dict.pop('image')
        
        container_spec = DockerContainerSpec(image)
        
        if 'command' in spec_dict:
            container_spec.command = list(spec_dict.pop('command'))
        if 'arguments' in spec_dict:
            container_spec.arguments = list(spec_dict.pop('arguments'))
        if 'fileOutputs' in spec_dict:
            container_spec.file_outputs = dict(spec_dict.pop('fileOutputs'))

        if spec_dict:
            raise ValueError('Found unrecognized properties: {}'.format(spec_dict))
        
        return container_spec

    def to_struct(self):
        struct = OrderedDict()
        if self.image:
            struct['image'] = self.image
        if self.command:
            struct['command'] = self.command
        if self.arguments:
            struct['arguments'] = self.arguments
        if self.file_outputs:
            struct['fileOutputs'] = self.file_outputs
        
        return struct
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'


def check_instance_type(obj, accepted_types:List[type]):
    if not [accepted_type for accepted_type in accepted_types if isinstance(obj, accepted_type)]:
        raise ValueError('Encountered object of wrong type. Accepted types: {}. Actual type: {}'.format(accepted_types, obj.__class__.__name__))


class GraphInputReferenceSpec:
    def __init__(self, input_name:str):
        self.input_name = input_name

    @staticmethod
    def from_struct(struct:Union[Tuple[str, str],List[str]]):
        if len(struct) != 2 or struct[0].lower() != 'GraphInput'.lower():
            raise ValueError('Error parsing GraphInputReferenceSpec: "{}". The correct format is [GraphInput, Input name].'.format(struct))
        input_name = struct[1]
        check_instance_type(input_name, [str])
        
        return GraphInputReferenceSpec(input_name)

    def to_struct(self):
        return ['GraphInput', self.input_name]
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'


class TaskOutputReferenceSpec:
    def __init__(self, task_id:str, output_name:str):
        self.task_id = task_id
        self.output_name = output_name

    @staticmethod
    def from_struct(struct:Union[Tuple[str, str, str],List[str]]):
        if len(struct) != 3 or struct[0].lower() != 'TaskOutput'.lower():
            raise ValueError('Error parsing TaskOutputReferenceSpec: "{}". The correct format is [TaskOutput, Task ID, Output name].'.format(struct))
        task_id = struct[1]
        output_name = struct[2]
        
        check_instance_type(task_id, [str])
        check_instance_type(output_name, [str])
        
        return TaskOutputReferenceSpec(task_id, output_name)

    def to_struct(self):
        return ['TaskOutput', self.task_id, self.output_name]
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'


class DataValueOrReferenceSpec:
    def __init__(self, constant_value:str=None, graph_input:str=None, task_output:Tuple[str, str]=None):
        if constant_value != None:
            if graph_input != None or task_output != None:
                raise ValueError('Specify only one argument.')
            self.constant_value = constant_value
        if graph_input != None:
            if constant_value != None or task_output != None:
                raise ValueError('Specify only one argument.')
            self.graph_input = GraphInputReferenceSpec(graph_input)
        if task_output != None:
            if constant_value != None or graph_input != None:
                raise ValueError('Specify only one argument.')
            (task_id, output_name) = task_output
            self.task_output = TaskOutputReferenceSpec(task_id, output_name)

    @staticmethod
    def from_struct(struct:Union[str,Tuple[str, str],Tuple[str, str],List[str]]):
        if isinstance(struct, tuple) or isinstance(struct, list):
            kind = struct[0]
            if kind.lower() == 'GraphInput'.lower():
                return DataValueOrReferenceSpec(graph_input=GraphInputReferenceSpec.from_struct(struct))
            elif kind.lower() == 'TaskOutput'.lower():
                return DataValueOrReferenceSpec(task_output=TaskOutputReferenceSpec.from_struct(struct))
            else:
                raise ValueError('Found unknown input value spec: {}'.format(struct))
        return DataValueOrReferenceSpec(constant_value=str(struct))
        
    def to_struct(self):
        if self.constant_value != None:
            return self.constant_value
        if self.graph_input != None:
            return self.graph_input.to_struct()
        if self.task_output != None:
            return self.task_output.to_struct()
        raise AssertionError('Invalid internal state')
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'


class TaskSpec:
    def __init__(self, componet_id:str, inputValues:Mapping[str,DataValueOrReferenceSpec]=None, enabled:DataValueOrReferenceSpec=None):
        if componet_id == None:
            raise ValueError('componetId is required')
        self.componet_id = componet_id
        self.input_values = inputValues
        self.enabled = enabled

    @staticmethod
    def from_struct(struct:Mapping):
        struct = copy.deepcopy(struct)
        
        componet_id = str(struct.pop('componetId'))
        spec = TaskSpec(componet_id)
        
        if 'inputValues' in struct:
            spec.input_values = OrderedDict([(name, DataValueOrReferenceSpec.from_struct(value)) for name, value in struct.pop('inputValues').items()])
        if 'enabled' in struct:
            spec.enabled = DataValueOrReferenceSpec.from_struct(struct.pop('enabled'))

        if struct:
            raise ValueError('Found unrecognized properties: {}'.format(struct))
        
        return spec

    def to_struct(self):
        struct = OrderedDict()
        
        struct['componetId'] = self.componet_id
        if self.input_values:
            struct['inputValues'] = self.input_values.to_struct()
        if self.enabled:
            struct['enabled'] = self.enabled.to_struct()
        
        return struct
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'


class GraphSpec:
    def __init__(self, tasks:Mapping[str,TaskSpec], outputValues:Mapping[str,DataValueOrReferenceSpec]=None):
        self.tasks = tasks
        self.outputValues = outputValues
    
    @staticmethod
    def from_struct(struct:Mapping):
        struct = copy.deepcopy(struct)
        
        tasks_dict = struct.pop('tasks')
        tasks = OrderedDict([(task_id, TaskSpec.from_struct(task_struct)) for task_id, task_struct in tasks_dict.items()])

        obj = GraphSpec(tasks)

        if 'outputValues' in struct:
            outputValues_dict = struct.pop('outputValues')
            obj.outputValues = OrderedDict([(name, DataValueOrReferenceSpec.from_struct(value_struct)) for name, value_struct in outputValues_dict.items()])

        if struct:
            raise ValueError('Found unrecognized properties: {}'.format(struct))
        
        return obj

    def to_struct(self):
        struct = OrderedDict()
        if self.tasks:
            struct['tasks'] = OrderedDict([(name, task_spec.to_struct()) for name, task_spec in self.tasks.items()])
        if self.outputValues:
            struct['outputValues'] = OrderedDict([(name, value_spec.to_struct()) for name, value_spec in self.outputValues.items()])
        
        return struct
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'


class ImplementationSpec:
    def __init__(self, docker_container=None, graph=None):
        if not docker_container and not graph:
            raise ValueError('Implementation is required')
        if docker_container and graph:
            raise ValueError('Only one implementation can be specified')

        self.docker_container = docker_container
        self.graph = graph
    
    @staticmethod
    def from_struct(spec_dict:Mapping):
        if len(spec_dict) != 1:
            raise ValueError('There must be exactly one implementation')
        
        for name, value in spec_dict.items():
            if name == 'dockerContainer':
                return ImplementationSpec(DockerContainerSpec.from_struct(value))
            elif name == 'graph':
                return ImplementationSpec(GraphSpec.from_struct(value))
            else:
                raise ValueError('Unknown implementation type {}'.format(name))

    def to_struct(self):
        struct = {}
        if self.docker_container:
            struct['dockerContainer'] = self.docker_container.to_struct()
        if self.graph:
            struct['graph'] = self.graph.to_struct()
        
        return struct
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'


class SourceSpec:
    def __init__(self, url:str=None):
        self.url = url
    
    @staticmethod
    def from_struct(struct:Mapping):
        struct = copy.deepcopy(struct)
        spec = SourceSpec()
        
        if 'url' in struct:
            spec.url = struct.pop('url')
            check_instance_type(spec.url, [str])

        if struct:
            raise ValueError('Found unrecognized properties: {}'.format(struct))
        
        return spec

    def to_struct(self):
        struct = OrderedDict()
        if self.url:
            struct['url'] = self.url
        return struct
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'


class ComponentSpec:
    def __init__(
        self,
        implementation:ImplementationSpec,
        name:str=None,
        description:str=None,
        source:Mapping=None,
        inputs:List[InputSpec]=None,
        outputs:List[OutputSpec]=None,
        version:str='google.com/cloud/pipelines/component/v1',
    ):
        if not implementation:
            raise ValueError('Implementation is required')
        self.version = version
        self.name = name
        self.description = description
        self.source = source
        self.inputs = inputs
        self.outputs = outputs
        self.implementation = implementation
    
    @staticmethod
    def _ports_collection_from_struct(struct_collection:Union[Mapping[str,Mapping],List[Mapping]]):
        if isinstance(struct_collection, dict):
            port_struct_iterator = struct_collection.items()
        elif isinstance(struct_collection, list):
            port_struct_iterator = struct_collection
        else:
            check_instance_type(struct_collection, [dict, list])
            #raise ValueError('Unknown inputs collection type: {}'.format(struct.__class__.__name__))
        return list([InputSpec.from_struct(port_struct) for port_struct in port_struct_iterator])
        
    @staticmethod
    def from_struct(struct:Mapping):
        struct = copy.deepcopy(struct)
        
        implementation_struct = struct.pop('implementation')
        implementation_spec = ImplementationSpec.from_struct(implementation_struct)
        
        spec = ComponentSpec(implementation_spec)
        
        if 'version' in struct:
            spec.version = struct.pop('version')
        if 'name' in struct:
            spec.name = struct.pop('name')
        if 'description' in struct:
            spec.description = struct.pop('description')
        if 'source' in struct:
            spec.source = SourceSpec.from_struct(struct.pop('source'))
        if 'inputs' in struct:
            inputs_struct = struct.pop('inputs')
            if isinstance(inputs_struct, dict):
                input_iterator = inputs_struct.items()
            elif isinstance(inputs_struct, list):
                input_iterator = inputs_struct
            else:
                check_instance_type(inputs_struct, [dict, list])
                #raise ValueError('Unknown inputs collection type: {}'.format(inputs_struct.__class__.__name__))
            spec.inputs = [InputSpec.from_struct(input_struct) for input_struct in input_iterator]
        if 'outputs' in struct:
            outputs_struct = struct.pop('outputs')
            if isinstance(outputs_struct, dict):
                output_iterator = outputs_struct.items()
            elif isinstance(outputs_struct, list):
                output_iterator = outputs_struct
            else:
                check_instance_type(outputs_struct, [dict, list])
                #raise ValueError('Unknown outputs collection type: {}'.format(outputs_struct.__class__.__name__))
            spec.outputs = [OutputSpec.from_struct(output_struct) for output_struct in output_iterator]

        if struct:
            raise ValueError('Found unrecognized properties: {}'.format(struct))
        
        return spec

    def to_struct(self):
        struct = OrderedDict()
        
        if self.version:
            struct['version'] = self.version
        if self.name:
            struct['name'] = self.name
        if self.description:
            struct['description'] = self.description
        if self.source:
            struct['source'] = self.source.to_struct()
        if self.inputs:
            struct['inputs'] = OrderedDict([input.to_struct() for input in self.inputs])
        if self.outputs:
            struct['outputs'] = OrderedDict([output.to_struct() for output in self.outputs])
        if self.implementation:
            struct['implementation'] = self.implementation.to_struct()
        
        return struct
    
    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'
