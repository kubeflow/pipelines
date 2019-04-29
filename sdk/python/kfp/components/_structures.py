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
    'InputSpec',
    'OutputSpec',

    'InputValuePlaceholder',
    'InputPathPlaceholder',
    'OutputPathPlaceholder',
    'ConcatPlaceholder',
    'IsPresentPlaceholder',
    'IfPlaceholder',

    'ContainerSpec',
    'ContainerImplementation',

    'ComponentSpec',

    'ComponentReference',

    'GraphInputArgument',
    'TaskOutputReference',
    'TaskOutputArgument',

    'EqualsPredicate',
    'NotEqualsPredicate',
    'GreaterThanPredicate',
    'GreaterThanOrEqualPredicate',
    'LessThenPredicate',
    'LessThenOrEqualPredicate',
    'NotPredicate',
    'AndPredicate',
    'OrPredicate',

    'TaskSpec',

    'GraphSpec',
    'GraphImplementation',

    'PipelineRunSpec',
]

from collections import OrderedDict

from typing import Any, Dict, List, Mapping, Optional, Sequence, Union

from .modelbase import ModelBase

from .structures.kubernetes import v1


PrimitiveTypes = Union[str, int, float, bool]
PrimitiveTypesIncludingNone = Optional[PrimitiveTypes]


class InputSpec(ModelBase):
    '''Describes the component input specification'''
    def __init__(self,
        name: str,
        type: Optional[Union[str, Dict, List]] = None,
        description: Optional[str] = None,
        default: Optional[PrimitiveTypes] = None,
        optional: Optional[bool] = False,
    ):
        super().__init__(locals())


class OutputSpec(ModelBase):
    '''Describes the component output specification'''
    def __init__(self,
        name: str,
        type: Optional[Union[str, Dict, List]] = None,
        description: Optional[str] = None,
    ):
        super().__init__(locals())


class InputValuePlaceholder(ModelBase): #Non-standard attr names
    '''Represents the command-line argument placeholder that will be replaced at run-time by the input argument value.'''
    _serialized_names = {
        'input_name': 'inputValue',
    }

    def __init__(self,
        input_name: str,
    ):
        super().__init__(locals())


class InputPathPlaceholder(ModelBase): #Non-standard attr names
    '''Represents the command-line argument placeholder that will be replaced at run-time by a local file path pointing to a file containing the input argument value.'''
    _serialized_names = {
        'input_name': 'inputPath',
    }

    def __init__(self,
        input_name: str,
    ):
        super().__init__(locals())


class OutputPathPlaceholder(ModelBase): #Non-standard attr names
    '''Represents the command-line argument placeholder that will be replaced at run-time by a local file path pointing to a file where the program should write its output data.'''
    _serialized_names = {
        'output_name': 'outputPath',
    }

    def __init__(self,
        output_name: str,
    ):
        super().__init__(locals())


CommandlineArgumentType = Union[
    str,
    InputValuePlaceholder,
    InputPathPlaceholder,
    OutputPathPlaceholder,
    'ConcatPlaceholder',
    'IfPlaceholder',
]


class ConcatPlaceholder(ModelBase): #Non-standard attr names
    '''Represents the command-line argument placeholder that will be replaced at run-time by the concatenated values of its items.'''
    _serialized_names = {
        'items': 'concat',
    }

    def __init__(self,
        items: List[CommandlineArgumentType],
    ):
        super().__init__(locals())


class IsPresentPlaceholder(ModelBase): #Non-standard attr names
    '''Represents the command-line argument placeholder that will be replaced at run-time by a boolean value specifying whether the caller has passed an argument for the specified optional input.'''
    _serialized_names = {
        'input_name': 'isPresent',
    }

    def __init__(self,
        input_name: str,
    ):
        super().__init__(locals())


IfConditionArgumentType = Union[bool, str, IsPresentPlaceholder, InputValuePlaceholder]


class IfPlaceholderStructure(ModelBase): #Non-standard attr names
    '''Used in by the IfPlaceholder - the command-line argument placeholder that will be replaced at run-time by the expanded value of either "then_value" or "else_value" depending on the submissio-time resolved value of the "cond" predicate.'''
    _serialized_names = {
        'condition': 'cond',
        'then_value': 'then',
        'else_value': 'else',
    }

    def __init__(self,
        condition: IfConditionArgumentType,
        then_value: Union[CommandlineArgumentType, List[CommandlineArgumentType]],
        else_value: Optional[Union[CommandlineArgumentType, List[CommandlineArgumentType]]] = None,
    ):
        super().__init__(locals())


class IfPlaceholder(ModelBase): #Non-standard attr names
    '''Represents the command-line argument placeholder that will be replaced at run-time by the expanded value of either "then_value" or "else_value" depending on the submissio-time resolved value of the "cond" predicate.'''
    _serialized_names = {
        'if_structure': 'if',
    }

    def __init__(self,
        if_structure: IfPlaceholderStructure,
    ):
        super().__init__(locals())


class ContainerSpec(ModelBase):
    '''Describes the container component implementation.'''
    _serialized_names = {
        'file_outputs': 'fileOutputs', #TODO: rename to something like legacy_unconfigurable_output_paths
    }

    def __init__(self,
        image: str,
        command: Optional[List[CommandlineArgumentType]] = None,
        args: Optional[List[CommandlineArgumentType]] = None,
        env: Optional[Mapping[str, str]] = None,
        file_outputs: Optional[Mapping[str, str]] = None, #TODO: rename to something like legacy_unconfigurable_output_paths
    ):
        super().__init__(locals())


class ContainerImplementation(ModelBase):
    '''Represents the container component implementation.'''
    def __init__(self,
        container: ContainerSpec,
    ):
        super().__init__(locals())


ImplementationType = Union[ContainerImplementation, 'GraphImplementation']


class MetadataSpec(ModelBase):
    def __init__(self,
        annotations: Optional[Dict[str, str]] = None,
        labels: Optional[Dict[str, str]] = None,
    ):
        super().__init__(locals())


class ComponentSpec(ModelBase):
    '''Component specification. Describes the metadata (name, description, annotations and labels), the interface (inputs and outputs) and the implementation of the component.'''
    def __init__(
        self,
        name: Optional[str] = None, #? Move to metadata?
        description: Optional[str] = None, #? Move to metadata?
        metadata: Optional[MetadataSpec] = None,
        inputs: Optional[List[InputSpec]] = None,
        outputs: Optional[List[OutputSpec]] = None,
        implementation: Optional[ImplementationType] = None,
        version: Optional[str] = 'google.com/cloud/pipelines/component/v1',
        #tags: Optional[Set[str]] = None,
    ):
        super().__init__(locals())
        self._post_init()

    def _post_init(self):
        #Checking input names for uniqueness
        self._inputs_dict = {}
        if self.inputs:
            for input in self.inputs:
                if input.name in self._inputs_dict:
                    raise ValueError('Non-unique input name "{}"'.format(input.name))
                self._inputs_dict[input.name] = input

        #Checking output names for uniqueness
        self._outputs_dict = {}
        if self.outputs:
            for output in self.outputs:
                if output.name in self._outputs_dict:
                    raise ValueError('Non-unique output name "{}"'.format(output.name))
                self._outputs_dict[output.name] = output

        if isinstance(self.implementation, ContainerImplementation):
            container = self.implementation.container

            if container.file_outputs:
                for output_name, path in container.file_outputs.items():
                    if output_name not in self._outputs_dict:
                        raise TypeError('Unconfigurable output entry "{}" references non-existing output.'.format({output_name: path}))

            def verify_arg(arg):
                if arg is None:
                    pass
                elif isinstance(arg, (str, int, float, bool)):
                    pass
                elif isinstance(arg, list):
                    for arg2 in arg:
                        verify_arg(arg2)
                elif isinstance(arg, (InputValuePlaceholder, InputPathPlaceholder, IsPresentPlaceholder)):
                    if arg.input_name not in self._inputs_dict:
                        raise TypeError('Argument "{}" references non-existing input.'.format(arg))
                elif isinstance(arg, OutputPathPlaceholder):
                    if arg.output_name not in self._outputs_dict:
                        raise TypeError('Argument "{}" references non-existing output.'.format(arg))
                elif isinstance(arg, ConcatPlaceholder):
                    for arg2 in arg.items:
                        verify_arg(arg2)
                elif isinstance(arg, IfPlaceholder):
                    verify_arg(arg.if_structure.condition)
                    verify_arg(arg.if_structure.then_value)
                    verify_arg(arg.if_structure.else_value)
                else:
                    raise TypeError('Unexpected argument "{}"'.format(arg))
            
            verify_arg(container.command)
            verify_arg(container.args)

        if isinstance(self.implementation, GraphImplementation):
            graph = self.implementation.graph

            if graph.output_values is not None:
                for output_name, argument in graph.output_values.items():
                    if output_name not in self._outputs_dict:
                        raise TypeError('Graph output argument entry "{}" references non-existing output.'.format({output_name: argument}))

            if graph.tasks is not None:
                for task in graph.tasks.values():
                    if task.arguments is not None:
                        for argument in task.arguments.values():
                            if isinstance(argument, GraphInputArgument) and argument.input_name not in self._inputs_dict:
                                raise TypeError('Argument "{}" references non-existing input.'.format(argument))


class ComponentReference(ModelBase):
    '''Component reference. Contains information that can be used to locate and load a component by name, digest or URL'''
    def __init__(self,
        name: Optional[str] = None,
        digest: Optional[str] = None,
        tag: Optional[str] = None,
        url: Optional[str] = None,
    ):
        super().__init__(locals())
        self._post_init()
    
    def _post_init(self) -> None:
        if not any([self.name, self.digest, self.tag, self.url]):
            raise TypeError('Need at least one argument.')


class GraphInputArgument(ModelBase):
    '''Represents the component argument value that comes from the graph component input.'''
    _serialized_names = {
        'input_name': 'graphInput',
    }

    def __init__(self,
        input_name: str,
    ):
        super().__init__(locals())


class TaskOutputReference(ModelBase):
    '''References the output of some task (the scope is a single graph).'''
    _serialized_names = {
        'task_id': 'taskId',
        'output_name': 'outputName',
    }

    def __init__(self,
        task_id: str,
        output_name: str,
    ):
        super().__init__(locals())


class TaskOutputArgument(ModelBase): #Has additional constructor for convenience
    '''Represents the component argument value that comes from the output of another task.'''
    _serialized_names = {
        'task_output': 'taskOutput',
    }

    def __init__(self,
        task_output: TaskOutputReference,
    ):
        super().__init__(locals())

    @staticmethod
    def construct(
        task_id: str,
        output_name: str,
    ) -> 'TaskOutputArgument':
        return TaskOutputArgument(TaskOutputReference(
            task_id=task_id,
            output_name=output_name,
        ))


ArgumentType = Union[PrimitiveTypes, GraphInputArgument, TaskOutputArgument]


class TwoOperands(ModelBase):
    def __init__(self,
        op1: ArgumentType,
        op2: ArgumentType,
    ):
        super().__init__(locals())


class BinaryPredicate(ModelBase): #abstract base type
    def __init__(self,
        operands: TwoOperands
    ):
        super().__init__(locals())


class EqualsPredicate(BinaryPredicate):
    '''Represents the "equals" comparison predicate.'''
    _serialized_names = {'operands': '=='}


class NotEqualsPredicate(BinaryPredicate):
    '''Represents the "not equals" comparison predicate.'''
    _serialized_names = {'operands': '!='}


class GreaterThanPredicate(BinaryPredicate):
    '''Represents the "greater than" comparison predicate.'''
    _serialized_names = {'operands': '>'}


class GreaterThanOrEqualPredicate(BinaryPredicate):
    '''Represents the "greater than or equal" comparison predicate.'''
    _serialized_names = {'operands': '>='}


class LessThenPredicate(BinaryPredicate):
    '''Represents the "less than" comparison predicate.'''
    _serialized_names = {'operands': '<'}


class LessThenOrEqualPredicate(BinaryPredicate):
    '''Represents the "less than or equal" comparison predicate.'''
    _serialized_names = { 'operands': '<='}


PredicateType = Union[
    ArgumentType,
    EqualsPredicate, NotEqualsPredicate, GreaterThanPredicate, GreaterThanOrEqualPredicate, LessThenPredicate, LessThenOrEqualPredicate,
    'NotPredicate', 'AndPredicate', 'OrPredicate',
]


class TwoBooleanOperands(ModelBase):
    def __init__(self,
        op1: PredicateType,
        op2: PredicateType,
    ):
        super().__init__(locals())


class NotPredicate(ModelBase):
    '''Represents the "not" logical operation.'''
    _serialized_names = {'operand': 'not'}

    def __init__(self,
        operand: PredicateType
    ):
        super().__init__(locals())


class AndPredicate(ModelBase):
    '''Represents the "and" logical operation.'''
    _serialized_names = {'operands': 'and'}

    def __init__(self,
        operands: TwoBooleanOperands
    ) :
        super().__init__(locals())

class OrPredicate(ModelBase):
    '''Represents the "or" logical operation.'''
    _serialized_names = {'operands': 'or'}

    def __init__(self,
        operands: TwoBooleanOperands
    ):
        super().__init__(locals())


class TaskSpec(ModelBase):
    '''Task specification. Task is a "configured" component - a component supplied with arguments and other applied configuration changes.'''
    _serialized_names = {
        'component_ref': 'componentRef',
        'is_enabled': 'isEnabled',
        'k8s_container_options': 'k8sContainerOptions',
        'k8s_pod_options': 'k8sPodOptions',
    }

    def __init__(self,
        component_ref: ComponentReference,
        arguments: Optional[Mapping[str, ArgumentType]] = None,
        is_enabled: Optional[PredicateType] = None,
        k8s_container_options: Optional[v1.Container] = None,
        k8s_pod_options: Optional[v1.PodArgoSubset] = None,
    ):
        super().__init__(locals())
        #TODO: If component_ref is resolved to component spec, then check that the arguments correspond to the inputs


class GraphSpec(ModelBase):
    '''Describes the graph component implementation. It represents a graph of component tasks connected to the upstream sources of data using the argument specifications. It also describes the sources of graph output values.'''
    _serialized_names = {
        'output_values': 'outputValues',
    }

    def __init__(self,
        tasks: Mapping[str, TaskSpec],
        output_values: Mapping[str, ArgumentType] = None,
    ):
        super().__init__(locals())
        self._post_init()
    
    def _post_init(self):
        #Checking task output references and preparing the dependency table
        task_dependencies = {}
        for task_id, task in self.tasks.items():
            dependencies = set()
            task_dependencies[task_id] = dependencies
            if task.arguments is not None:
                for argument in task.arguments.values():
                    if isinstance(argument, TaskOutputArgument):
                        dependencies.add(argument.task_output.task_id)
                        if argument.task_output.task_id not in self.tasks:
                            raise TypeError('Argument "{}" references non-existing task.'.format(argument))

        #Topologically sorting tasks to detect cycles
        task_dependents = {k: set() for k in task_dependencies.keys()}
        for task_id, dependencies in task_dependencies.items():
            for dependency in dependencies:
                task_dependents[dependency].add(task_id)
        task_number_of_remaining_dependencies = {k: len(v) for k, v in task_dependencies.items()}
        sorted_tasks = OrderedDict()
        def process_task(task_id):
            if task_number_of_remaining_dependencies[task_id] == 0 and task_id not in sorted_tasks:
                sorted_tasks[task_id] = self.tasks[task_id]
                for dependent_task in task_dependents[task_id]:
                    task_number_of_remaining_dependencies[dependent_task] = task_number_of_remaining_dependencies[dependent_task] - 1
                    process_task(dependent_task)
        for task_id in task_dependencies.keys():
            process_task(task_id)
        if len(sorted_tasks) != len(task_dependencies):
            tasks_with_unsatisfied_dependencies = {k: v for k, v in task_number_of_remaining_dependencies.items() if v > 0}
            task_wth_minimal_number_of_unsatisfied_dependencies = min(tasks_with_unsatisfied_dependencies.keys(), key=lambda task_id: tasks_with_unsatisfied_dependencies[task_id])
            raise ValueError('Task "{}" has cyclical dependency.'.format(task_wth_minimal_number_of_unsatisfied_dependencies))
        
        self._toposorted_tasks = sorted_tasks


class GraphImplementation(ModelBase):
    '''Represents the graph component implementation.'''
    def __init__(self,
        graph: GraphSpec,
    ):
        super().__init__(locals())


class PipelineRunSpec(ModelBase):
    '''The object that can be sent to the backend to start a new Run.'''
    _serialized_names = {
        'root_task': 'rootTask',
        #'on_exit_task': 'onExitTask',
    }

    def __init__(self,
        root_task: TaskSpec,
        #on_exit_task: Optional[TaskSpec] = None,
    ):
        super().__init__(locals())
