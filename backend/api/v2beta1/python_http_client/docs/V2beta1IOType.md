# V2beta1IOType

Describes the I/O relationship between Artifacts/Parameters and Tasks. There are a couple of instances where input/outputs have special types such as in the case of LoopArguments or dsl.Collected outputs.   - UNSPECIFIED: For validation  - COMPONENT_DEFAULT_INPUT: This is used for inputs that are provided via default parameters in the component input definitions  - TASK_OUTPUT_INPUT: This is used for inputs that are provided via upstream tasks. In the sdk this appears as: TaskInputsSpec.kind.task_output_parameter & TaskInputsSpec.kind.task_output_artifact  - COMPONENT_INPUT: Used for inputs that are passed from parent tasks.  - RUNTIME_VALUE_INPUT: Hardcoded values passed as arguments to the task.  - COLLECTED_INPUTS: Used for dsl.Collected Usage of this type indicates that all Artifacts within the IOArtifact.artifacts are inputs collected from sub tasks with ITERATOR_OUTPUT outputs.  - ITERATOR_INPUT: In a for loop task, introduced via ParallelFor, this type is used to indicate whether this resolved input belongs to a parameterIterator or artifactIterator. In such a case the \"artifacts\" field for IOArtifact.artifacts is the list of resolved items for this parallelFor.  - ITERATOR_INPUT_RAW: Hardcoded iterator parameters. Raw Iterator inputs have no producer  - ITERATOR_OUTPUT: When an output is produced by a Runtime Iteration Task This value is use to differentiate between standard inputs  - OUTPUT: All other output types fall under this type.  - ONE_OF_OUTPUT: An output of a Conditions branch.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


