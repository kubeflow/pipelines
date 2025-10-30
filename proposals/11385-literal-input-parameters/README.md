# KEP-11385: Literal Input Parameters
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User stories](#user-stories)
  - [Design Details](#design-details)
  - [Test Plan](#test-plan)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)

## Summary
It is not currently possible to limit pipeline input to a specified set of values. Often pipeline authors want to restrict input in order to reduce unintended behavior and pipeline failure. This KEP proposes extending the valid pipeline input types to include Python Literal values, from the package `typing.Literal`. This change will involve updates to the SDK compiler, the pipeline run server and the UI input display.

## Motivation
Valid pipeline input currently includes string, int, float, boolean, list and struct, as well as two custom types – TaskFinalStatus and TaskConfig. None of these choices allows a pipeline author to restrict input to one or more pre-determined options. When a pipeline run is executed with unintended inputs, the resulting failure unexpected behavior not only wastes time and resources but also creates a frustrating user experience.
The current workaround is to validate input at the component level. But this validation does not execute until the executor runs the component code, at which point time and resources have still been wasted running the pipeline system and DAG drivers. When an invalid value is used with a pipeline that takes a Literal input set, the pipeline will fail during either SDK compilation or when it is submitted to the pipeline run server, depending on when input is provided. 

### What is the Literal type in Python, and why was it chosen here?
The Python `typing.Literal` type is used to encapsulate one or more specific values in a single variable or parameter. More technical details on the implementation can be found [here](https://typing.python.org/en/latest/spec/literal.html), and the `typing.Literal` [PEP](https://peps.python.org/pep-0586/ ) contains more background information. While a Literal can contain multiple types (ie string, int) within a single value, for our purposes Literals used should contain one or more values of a single type. Typing.Literal was chosen over Enum, because the more complex support found in Enum – Enum members are distinct objects, with built-in methods including iteration and comparison – is unnecessary for this use case. Note that **Literal** is used throughout this document to refer to `typing.Literal` type values.

### Goals
The goal of this KEP is to expand supported pipeline input to include `typing.Literal`:
- A pipeline author can write pipelines and pipeline components with input Literal parameters, and the SDK compiler can parse Literal parameters and output a pipeline YAML containing the Literal elements.
- When a user submits a pipeline YAML file with literal inputs and its corresponding runtime values to the API server, the pipeline run server validates the input before submitting the run. If the input is not present in the Literal parameter, the pipeline run server rejects the request and returns an error.
- When a user runs a pipeline with Literal input via the UI, they should be able to select a valid input from a drop-down menu in the runtime parameter box. 

### Non-Goals
This KEP is not proposing the creation of an additional custom data type for Literal parameters.

## Proposal
### User Stories
#### Story #1:
I am a machine learning engineer writing a pipeline with components that utilize GPU accelerators, and I have two specific options that can be used: `nvidia-tesla-k80` or `nvidia-tesla-p100`. If any other option is input, the component fails. Even if I set component-level input validation, the pipeline still will not fail until this component executes. This is the 10th component in the pipeline, and this failure wastes a lot of time and resources. I want to restrict my input to `nvidia-tesla-k80` and `nvidia-tesla-p100`.

#### Story #2:
I am a pipeline author concerned about malicious users inputting values that could cause my pipeline to run malicious code. I want to limit the input to a specific set of predetermined values to prevent this.

### Design Details
In order to streamline the design and also to account for there being no `typing.Literal` counterpart in Go, a Literal parameter is represented similarly to a more typical parameter (e.g. int, string) with the only difference being an additional `literals` field:
#### InputSpec
`literals: Optional[List[Any]]` is added to the InputSpec class definition in `kfp/dsl/structures.py`:
```aiignore
class InputSpec:
    type: Union[str, dict]
    default: Optional[Any] = None
    literals: Optional[List[Any]] = None
    optional: bool = False
    is_artifact_list: bool = False
    description: Optional[str] = None
```
For example, a pipeline component with a Literal parameter would look like this:
```aiignore
@dsl.component()
def component(input: Literal["a", "b", "c"]):
    print(input)
```
The parameter InputSpec would be populated with the following:
```aiignore
type: STRING
default: None
literals: ["a", "b", "c"]
optional: False
is_artifact_list: False
description: None
```
And the pipeline YAML would look like this:
```aiignore
components:
  component:
    inputDefinitions:
      parameters:
        input:  
          literals: ["a", "b", "c"]
          parameterType: STRING
```
#### ComponentInputsSpec_ParameterSpec 
Optional field `Literals []*structpb.Value` is added to `ComponentInputsSpec_ParameterSpec` located in `api/v2alpha1/pipeline_spec.proto`:


```aiignore
type ComponentInputsSpec_ParameterSpec struct {
	state protoimpl.MessageState
	Type PrimitiveType_PrimitiveTypeEnum
	ParameterType ParameterType_ParameterTypeEnum
	DefaultValue *structpb.Value 
	Literals []*structpb.Value
	IsOptional bool
	Description   string
}
```

#### SDK Compiler
The following changes to the SDK compiler should be implemented first because the API server changes require a compiled pipeline YAML file.
- The current scope of this KEP extends to implementing Literal[string] and Literal[int] parameters
- If a Literal parameter contains elements of multiple types, the compiler will throw an error: `“KFP supports Literals of a single type only.”`
  - Two examples of valid Literals that KFP would not compile: `Literal[“a”, 10, “b”]`; `Literal[False, “a”, 10]`
- Updates to the SDK compiler should be made primarily to the following files: `kfp/dsl/structures.py` and `kfp/dsl/types/type_utils.py`
- After the pipeline spec is populated, the SDK compiler should iterate through pipeline and component-level input and check that every element of a pipeline-level input Literal parameter is a valid input to the corresponding component Literal parameter. If not, compilation fails. 
#### API Server
The following changes to the API server should be implemented first because the pipeline run server changes require a compiled pipeline YAML file.
- Extend the pipeline run server runtime parameter validation logic to check the “literals” field of a ComponentInputsSpec_ParameterSpec. If “literals” is non-empty, then the runtime parameter should be checked against the valid input options. This logic lives in `v2_template.validatePipelineJobInputs()`.
#### UI
The update to the UI should be implemented last, because it depends on the SDK and API server changes.
- When a pipeline YAML file containing a pipeline with Literal input is uploaded via the UI, the box for each input parameter will display a drop-down list containing the one or more values contained within the Literal parameter for the user to click. The user should not be able to type an input in the box - options should be selection-only.
### Test Plan
- Each of the following test cases can be implemented with string, int and float Literal parameters. 
- Hard-coded input refers to input passed into the pipeline function call. This is in contrast to runtime input, which is passed in when the compiled pipeline is submitted to the API server.
#### The following test cases are expected to pass: 
|                                                           | SDK Compiler | Workflow Compiler | Execute Pipeline E2E | API Server: Verify Pipeline Run |
|-----------------------------------------------------------|--------------|-------------------|----------------------|----------------------------|
| Pipeline-level Literal input:<br/> valid hard-coded input | ✓            | ✓                 | ✓                    | ✓                          |
| Pipeline-level Literal input:<br/>no input (SDK only)     | ✓            | X                 | X                    | ✓                          |
| Component-level Literal input                             | ✓            | ✓                 | ✓                    | ✓                          |
| Pipeline & component-level:<br/> valid hard-coded input   | ✓            | ✓                 | ✓                    | ✓                          |
| Pipeline-level Literal input:<br/> valid runtime input    | X            | ✓                 | ✓                    | ✓                          |

#### Verify failure on the following:
|                                                                               | SDK Compiler | Workflow Compiler | Execute Pipeline E2E | API Server: Verify Pipeline Run|
|-------------------------------------------------------------------------------|--------------|-------------------|----------------------|-----------------------|
| Pipeline & component-level Literal input:<br/> Literal elements do not match. | ✓            | X                 | X                    | X                     |
| Pipeline-level Literal input:<br/>invalid hard-coded input                    | ✓            | X                 | X                    | X                     |
| Pipeline-level Literal input:<br/>invalid runtime input                       | X            | ✓                 | ✓                    | ✓                     |

#### Additional testing
- Add frontend integration test case to verify that the UI displays drop-down menu for Literal input.
## Implementation History
- Initial proposal: 2024-11-18
## Drawbacks
Unlike an Enum, Literal parameters do not have a built-in method for iteration or comparison. The choice to use Literal over Enum was made because of Literal's simplicity, but in some cases users could find this frustrating.