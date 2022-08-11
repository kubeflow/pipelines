/* eslint-disable */
import Long from 'long';
import _m0 from 'protobufjs/minimal';
import { Status } from './google/rpc/status';
import { Struct, Value as Value1 } from './google/protobuf/struct';

export const protobufPackage = 'ml_pipelines';

/** The spec of a pipeline job. */
export interface PipelineJob {
  /** Name of the job. */
  name: string;
  /** User friendly display name */
  displayName: string;
  /** Definition of the pipeline that is being executed. */
  pipelineSpec: { [key: string]: any } | undefined;
  /** The labels with user-defined metadata to organize PipelineJob. */
  labels: { [key: string]: string };
  /** Runtime config of the pipeline. */
  runtimeConfig: PipelineJob_RuntimeConfig | undefined;
}

export interface PipelineJob_LabelsEntry {
  key: string;
  value: string;
}

/** The runtime config of a PipelineJob. */
export interface PipelineJob_RuntimeConfig {
  /**
   * Deprecated. Use [RuntimeConfig.parameter_values][] instead.
   *
   * @deprecated
   */
  parameters: { [key: string]: Value };
  /**
   * A path in a Cloud Storage bucket which will be treated as the root
   * output directory of the pipeline. It is used by the system to
   * generate the paths of output artifacts.
   * This is a GCP-specific optimization.
   */
  gcsOutputDirectory: string;
  /**
   * The runtime parameters of the PipelineJob. The parameters will be
   * passed into [PipelineJob.pipeline_spec][] to replace the placeholders
   * at runtime.
   */
  parameterValues: { [key: string]: any | undefined };
}

export interface PipelineJob_RuntimeConfig_ParametersEntry {
  key: string;
  value: Value | undefined;
}

export interface PipelineJob_RuntimeConfig_ParameterValuesEntry {
  key: string;
  value: any | undefined;
}

/** The spec of a pipeline. */
export interface PipelineSpec {
  /** The metadata of the pipeline. */
  pipelineInfo: PipelineInfo | undefined;
  /**
   * The deployment config of the pipeline.
   * The deployment config can be extended to provide platform specific configs.
   */
  deploymentSpec: { [key: string]: any } | undefined;
  /** The version of the sdk, which compiles the spec. */
  sdkVersion: string;
  /** The version of the schema. */
  schemaVersion: string;
  /** The map of name to definition of all components used in this pipeline. */
  components: { [key: string]: ComponentSpec };
  /**
   * The definition of the main pipeline.  Execution of the pipeline is
   * completed upon the completion of this component.
   */
  root: ComponentSpec | undefined;
  /** Optional field. The default root output directory of the pipeline. */
  defaultPipelineRoot: string;
}

/** The definition of the runtime parameter. */
export interface PipelineSpec_RuntimeParameter {
  /** Required field. The type of the runtime parameter. */
  type: PrimitiveType_PrimitiveTypeEnum;
  /**
   * Optional field. Default value of the runtime parameter. If not set and
   * the runtime parameter value is not provided during runtime, an error will
   * be raised.
   */
  defaultValue: Value | undefined;
}

export interface PipelineSpec_ComponentsEntry {
  key: string;
  value: ComponentSpec | undefined;
}

/** Definition of a component. */
export interface ComponentSpec {
  /** Definition of the input parameters and artifacts of the component. */
  inputDefinitions: ComponentInputsSpec | undefined;
  /** Definition of the output parameters and artifacts of the component. */
  outputDefinitions: ComponentOutputsSpec | undefined;
  dag: DagSpec | undefined;
  executorLabel: string | undefined;
}

/** A DAG contains multiple tasks. */
export interface DagSpec {
  /** The tasks inside the dag. */
  tasks: { [key: string]: PipelineTaskSpec };
  /** Defines how the outputs of the dag are linked to the sub tasks. */
  outputs: DagOutputsSpec | undefined;
}

export interface DagSpec_TasksEntry {
  key: string;
  value: PipelineTaskSpec | undefined;
}

/** Definition of the output artifacts and parameters of the DAG component. */
export interface DagOutputsSpec {
  /** Name to the output artifact channel of the DAG. */
  artifacts: { [key: string]: DagOutputsSpec_DagOutputArtifactSpec };
  /** The name to the output parameter. */
  parameters: { [key: string]: DagOutputsSpec_DagOutputParameterSpec };
}

/** Selects a defined output artifact from a sub task of the DAG. */
export interface DagOutputsSpec_ArtifactSelectorSpec {
  /**
   * The name of the sub task which produces the output that matches with
   * the `output_artifact_key`.
   */
  producerSubtask: string;
  /** The key of [ComponentOutputsSpec.artifacts][] map of the producer task. */
  outputArtifactKey: string;
}

/**
 * Selects a list of output artifacts that will be aggregated to the single
 * output artifact channel of the DAG.
 */
export interface DagOutputsSpec_DagOutputArtifactSpec {
  /**
   * The selected artifacts will be aggregated as output as a single
   * output channel of the DAG.
   */
  artifactSelectors: DagOutputsSpec_ArtifactSelectorSpec[];
}

export interface DagOutputsSpec_ArtifactsEntry {
  key: string;
  value: DagOutputsSpec_DagOutputArtifactSpec | undefined;
}

/** Selects a defined output parameter from a sub task of the DAG. */
export interface DagOutputsSpec_ParameterSelectorSpec {
  /**
   * The name of the sub task which produces the output that matches with
   * the `output_parameter_key`.
   */
  producerSubtask: string;
  /** The key of [ComponentOutputsSpec.parameters][] map of the producer task. */
  outputParameterKey: string;
}

/** Aggregate output parameters from sub tasks into a list object. */
export interface DagOutputsSpec_ParameterSelectorsSpec {
  parameterSelectors: DagOutputsSpec_ParameterSelectorSpec[];
}

/** Aggregates output parameters from sub tasks into a map object. */
export interface DagOutputsSpec_MapParameterSelectorsSpec {
  mappedParameters: { [key: string]: DagOutputsSpec_ParameterSelectorSpec };
}

export interface DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry {
  key: string;
  value: DagOutputsSpec_ParameterSelectorSpec | undefined;
}

/**
 * We support four ways to fan-in output parameters from sub tasks to the DAG
 * parent task.
 * 1. Directly expose a single output parameter from a sub task,
 * 2. (Conditional flow) Expose a list of output from multiple tasks
 * (some might be skipped) but allows only one of the output being generated.
 * 3. Expose a list of outputs from multiple tasks (e.g. iterator flow).
 * 4. Expose the aggregation of output parameters as a name-value map.
 */
export interface DagOutputsSpec_DagOutputParameterSpec {
  /**
   * Returns the sub-task parameter as a DAG parameter.  The selected
   * parameter must have the same type as the DAG parameter type.
   */
  valueFromParameter: DagOutputsSpec_ParameterSelectorSpec | undefined;
  /**
   * Returns one of the sub-task parameters as a DAG parameter. If there are
   * multiple values are available to select, the DAG will fail. All the
   * selected parameters must have the same type as the DAG parameter type.
   */
  valueFromOneof: DagOutputsSpec_ParameterSelectorsSpec | undefined;
}

export interface DagOutputsSpec_ParametersEntry {
  key: string;
  value: DagOutputsSpec_DagOutputParameterSpec | undefined;
}

/** Definition specification of the component input parameters and artifacts. */
export interface ComponentInputsSpec {
  /** Name to artifact input. */
  artifacts: { [key: string]: ComponentInputsSpec_ArtifactSpec };
  /** Name to parameter input. */
  parameters: { [key: string]: ComponentInputsSpec_ParameterSpec };
}

/** Definition of an artifact input. */
export interface ComponentInputsSpec_ArtifactSpec {
  artifactType: ArtifactTypeSchema | undefined;
}

/** Definition of a parameter input. */
export interface ComponentInputsSpec_ParameterSpec {
  /**
   * Specifies an input parameter's type.
   * Deprecated. Use [ParameterSpec.parameter_type][] instead.
   *
   * @deprecated
   */
  type: PrimitiveType_PrimitiveTypeEnum;
  /** Specifies an input parameter's type. */
  parameterType: ParameterType_ParameterTypeEnum;
  /** Optional field. Default value of the input parameter. */
  defaultValue: any | undefined;
}

export interface ComponentInputsSpec_ArtifactsEntry {
  key: string;
  value: ComponentInputsSpec_ArtifactSpec | undefined;
}

export interface ComponentInputsSpec_ParametersEntry {
  key: string;
  value: ComponentInputsSpec_ParameterSpec | undefined;
}

/** Definition specification of the component output parameters and artifacts. */
export interface ComponentOutputsSpec {
  /** Name to artifact output. */
  artifacts: { [key: string]: ComponentOutputsSpec_ArtifactSpec };
  /** Name to parameter output. */
  parameters: { [key: string]: ComponentOutputsSpec_ParameterSpec };
}

/** Definition of an artifact output. */
export interface ComponentOutputsSpec_ArtifactSpec {
  artifactType: ArtifactTypeSchema | undefined;
  /**
   * Deprecated. Use [ArtifactSpec.metadata][] instead.
   *
   * @deprecated
   */
  properties: { [key: string]: ValueOrRuntimeParameter };
  /**
   * Deprecated. Use [ArtifactSpec.metadata][] instead.
   *
   * @deprecated
   */
  customProperties: { [key: string]: ValueOrRuntimeParameter };
  /** Properties of the Artifact. */
  metadata: { [key: string]: any } | undefined;
}

export interface ComponentOutputsSpec_ArtifactSpec_PropertiesEntry {
  key: string;
  value: ValueOrRuntimeParameter | undefined;
}

export interface ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry {
  key: string;
  value: ValueOrRuntimeParameter | undefined;
}

/** Definition of a parameter output. */
export interface ComponentOutputsSpec_ParameterSpec {
  /**
   * Specifies an input parameter's type.
   * Deprecated. Use [ParameterSpec.parameter_type][] instead.
   *
   * @deprecated
   */
  type: PrimitiveType_PrimitiveTypeEnum;
  /** Specifies an output parameter's type. */
  parameterType: ParameterType_ParameterTypeEnum;
}

export interface ComponentOutputsSpec_ArtifactsEntry {
  key: string;
  value: ComponentOutputsSpec_ArtifactSpec | undefined;
}

export interface ComponentOutputsSpec_ParametersEntry {
  key: string;
  value: ComponentOutputsSpec_ParameterSpec | undefined;
}

/** The spec of task inputs. */
export interface TaskInputsSpec {
  /**
   * A map of input parameters which are small values, stored by the system and
   * can be queriable.
   */
  parameters: { [key: string]: TaskInputsSpec_InputParameterSpec };
  /** A map of input artifacts. */
  artifacts: { [key: string]: TaskInputsSpec_InputArtifactSpec };
}

/** The specification of a task input artifact. */
export interface TaskInputsSpec_InputArtifactSpec {
  /**
   * Pass the input artifact from another task within the same parent
   * component.
   */
  taskOutputArtifact: TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec | undefined;
  /** Pass the input artifact from parent component input artifact. */
  componentInputArtifact: string | undefined;
}

export interface TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec {
  /**
   * The name of the upstream task which produces the output that matches
   * with the `output_artifact_key`.
   */
  producerTask: string;
  /** The key of [TaskOutputsSpec.artifacts][] map of the producer task. */
  outputArtifactKey: string;
}

/**
 * Represents an input parameter. The value can be taken from an upstream
 * task's output parameter (if specifying `producer_task` and
 * `output_parameter_key`, or it can be a runtime value, which can either be
 * determined at compile-time, or from a pipeline parameter.
 */
export interface TaskInputsSpec_InputParameterSpec {
  /** Output parameter from an upstream task. */
  taskOutputParameter: TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec | undefined;
  /** A constant value or runtime parameter. */
  runtimeValue: ValueOrRuntimeParameter | undefined;
  /** Pass the input parameter from parent component input parameter. */
  componentInputParameter: string | undefined;
  /** The final status of an uptream task. */
  taskFinalStatus: TaskInputsSpec_InputParameterSpec_TaskFinalStatus | undefined;
  /**
   * Selector expression of Common Expression Language (CEL)
   * that applies to the parameter found from above kind.
   *
   * The expression is applied to the Value type
   * [Value][].  For example,
   * 'size(string_value)' will return the size of the Value.string_value.
   *
   * After applying the selection, the parameter will be returned as a
   * [Value][].  The type of the Value is either deferred from the input
   * definition in the corresponding
   * [ComponentSpec.input_definitions.parameters][], or if not found,
   * automatically deferred as either string value or double value.
   *
   * In addition to the builtin functions in CEL, The value.string_value can
   * be treated as a json string and parsed to the [google.protobuf.Value][]
   * proto message. Then, the CEL expression provided in this field will be
   * used to get the requested field. For examples,
   *  - if Value.string_value is a json array of "[1.1, 2.2, 3.3]",
   *  'parseJson(string_value)[i]' will pass the ith parameter from the list
   *  to the current task, or
   *  - if the Value.string_value is a json map of "{"a": 1.1, "b": 2.2,
   *  "c": 3.3}, 'parseJson(string_value)[key]' will pass the map value from
   *  the struct map to the current task.
   *
   * If unset, the value will be passed directly to the current task.
   */
  parameterExpressionSelector: string;
}

/** Represents an upstream task's output parameter. */
export interface TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec {
  /**
   * The name of the upstream task which produces the output parameter that
   * matches with the `output_parameter_key`.
   */
  producerTask: string;
  /** The key of [TaskOutputsSpec.parameters][] map of the producer task. */
  outputParameterKey: string;
}

/**
 * Represents an upstream task's final status. The field can only be set if
 * the schema version is `2.0.0`. The resolved input parameter will be a
 * json payload in string type.
 */
export interface TaskInputsSpec_InputParameterSpec_TaskFinalStatus {
  /** The name of the upsteram task where the final status is coming from. */
  producerTask: string;
}

export interface TaskInputsSpec_ParametersEntry {
  key: string;
  value: TaskInputsSpec_InputParameterSpec | undefined;
}

export interface TaskInputsSpec_ArtifactsEntry {
  key: string;
  value: TaskInputsSpec_InputArtifactSpec | undefined;
}

/** The spec of task outputs. */
export interface TaskOutputsSpec {
  /**
   * A map of output parameters which are small values, stored by the system and
   * can be queriable. The output key is used
   * by [TaskInputsSpec.InputParameterSpec][] of the downstream task to specify
   * the data dependency. The same key will also be used by
   * [ExecutorInput.Inputs][] to reference the output parameter.
   */
  parameters: { [key: string]: TaskOutputsSpec_OutputParameterSpec };
  /**
   * A map of output artifacts. Keyed by output key. The output key is used
   * by [TaskInputsSpec.InputArtifactSpec][] of the downstream task to specify
   * the data dependency. The same key will also be used by
   * [ExecutorInput.Inputs][] to reference the output artifact.
   */
  artifacts: { [key: string]: TaskOutputsSpec_OutputArtifactSpec };
}

/** The specification of a task output artifact. */
export interface TaskOutputsSpec_OutputArtifactSpec {
  /** The type of the artifact. */
  artifactType: ArtifactTypeSchema | undefined;
  /**
   * The properties of the artifact, which are determined either at
   * compile-time, or at pipeline submission time through runtime parameters
   */
  properties: { [key: string]: ValueOrRuntimeParameter };
  /**
   * The custom properties of the artifact, which are determined either at
   * compile-time, or at pipeline submission time through runtime parameters
   */
  customProperties: { [key: string]: ValueOrRuntimeParameter };
}

export interface TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry {
  key: string;
  value: ValueOrRuntimeParameter | undefined;
}

export interface TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry {
  key: string;
  value: ValueOrRuntimeParameter | undefined;
}

/** Specification for output parameters produced by the task. */
export interface TaskOutputsSpec_OutputParameterSpec {
  /** Required field. The type of the output parameter. */
  type: PrimitiveType_PrimitiveTypeEnum;
}

export interface TaskOutputsSpec_ParametersEntry {
  key: string;
  value: TaskOutputsSpec_OutputParameterSpec | undefined;
}

export interface TaskOutputsSpec_ArtifactsEntry {
  key: string;
  value: TaskOutputsSpec_OutputArtifactSpec | undefined;
}

/**
 * Represent primitive types. The wrapper is needed to give a namespace of
 * enum value so we don't need add `PRIMITIVE_TYPE_` prefix of each enum value.
 *
 * @deprecated
 */
export interface PrimitiveType {}

/**
 * The primitive types.
 * Deprecated. Use [ParameterType.ParameterTypeEnum][] instead.
 *
 * @deprecated
 */
export enum PrimitiveType_PrimitiveTypeEnum {
  PRIMITIVE_TYPE_UNSPECIFIED = 0,
  INT = 1,
  DOUBLE = 2,
  STRING = 3,
  UNRECOGNIZED = -1,
}

export function primitiveType_PrimitiveTypeEnumFromJSON(
  object: any,
): PrimitiveType_PrimitiveTypeEnum {
  switch (object) {
    case 0:
    case 'PRIMITIVE_TYPE_UNSPECIFIED':
      return PrimitiveType_PrimitiveTypeEnum.PRIMITIVE_TYPE_UNSPECIFIED;
    case 1:
    case 'INT':
      return PrimitiveType_PrimitiveTypeEnum.INT;
    case 2:
    case 'DOUBLE':
      return PrimitiveType_PrimitiveTypeEnum.DOUBLE;
    case 3:
    case 'STRING':
      return PrimitiveType_PrimitiveTypeEnum.STRING;
    case -1:
    case 'UNRECOGNIZED':
    default:
      return PrimitiveType_PrimitiveTypeEnum.UNRECOGNIZED;
  }
}

export function primitiveType_PrimitiveTypeEnumToJSON(
  object: PrimitiveType_PrimitiveTypeEnum,
): string {
  switch (object) {
    case PrimitiveType_PrimitiveTypeEnum.PRIMITIVE_TYPE_UNSPECIFIED:
      return 'PRIMITIVE_TYPE_UNSPECIFIED';
    case PrimitiveType_PrimitiveTypeEnum.INT:
      return 'INT';
    case PrimitiveType_PrimitiveTypeEnum.DOUBLE:
      return 'DOUBLE';
    case PrimitiveType_PrimitiveTypeEnum.STRING:
      return 'STRING';
    default:
      return 'UNKNOWN';
  }
}

/**
 * Represent parameter types. The wrapper is needed to give a namespace of
 * enum value so we don't need add `PARAMETER_TYPE_` prefix of each enum value.
 */
export interface ParameterType {}

/** The parameter types. */
export enum ParameterType_ParameterTypeEnum {
  /** PARAMETER_TYPE_ENUM_UNSPECIFIED - Indicates that the parameter type was not specified. */
  PARAMETER_TYPE_ENUM_UNSPECIFIED = 0,
  /**
   * NUMBER_DOUBLE - Indicates that a parameter is a number that is stored in a field of type
   * `double`.
   */
  NUMBER_DOUBLE = 1,
  /**
   * NUMBER_INTEGER - Indicates that a parameter is an integer stored in the `number_field`,
   * which is of type `double`. NUMBER_INTEGER values must be within the range
   * of JavaScript safe integers (-(2^53 - 1) to (2^53 - 1)). If you need to
   * support integers outside the range of JavaScript safe integers, use the
   * `STRING` parameter type to describe your parameter.
   */
  NUMBER_INTEGER = 2,
  /** STRING - Indicates that a parameter is a string. */
  STRING = 3,
  /** BOOLEAN - Indicates that a parameters is a boolean value. */
  BOOLEAN = 4,
  /**
   * LIST - Indicates that a parameter is a list of values. LIST parameters are
   * serialized to JSON when passed as an input or output of a pipeline step.
   */
  LIST = 5,
  /**
   * STRUCT - Indicates that a parameter is a struct value; structs represent a data
   * structure like a Python dictionary or a JSON object. STRUCT parameters
   * are serialized to JSON when passed as an input or output of a pipeline
   * step.
   */
  STRUCT = 6,
  UNRECOGNIZED = -1,
}

export function parameterType_ParameterTypeEnumFromJSON(
  object: any,
): ParameterType_ParameterTypeEnum {
  switch (object) {
    case 0:
    case 'PARAMETER_TYPE_ENUM_UNSPECIFIED':
      return ParameterType_ParameterTypeEnum.PARAMETER_TYPE_ENUM_UNSPECIFIED;
    case 1:
    case 'NUMBER_DOUBLE':
      return ParameterType_ParameterTypeEnum.NUMBER_DOUBLE;
    case 2:
    case 'NUMBER_INTEGER':
      return ParameterType_ParameterTypeEnum.NUMBER_INTEGER;
    case 3:
    case 'STRING':
      return ParameterType_ParameterTypeEnum.STRING;
    case 4:
    case 'BOOLEAN':
      return ParameterType_ParameterTypeEnum.BOOLEAN;
    case 5:
    case 'LIST':
      return ParameterType_ParameterTypeEnum.LIST;
    case 6:
    case 'STRUCT':
      return ParameterType_ParameterTypeEnum.STRUCT;
    case -1:
    case 'UNRECOGNIZED':
    default:
      return ParameterType_ParameterTypeEnum.UNRECOGNIZED;
  }
}

export function parameterType_ParameterTypeEnumToJSON(
  object: ParameterType_ParameterTypeEnum,
): string {
  switch (object) {
    case ParameterType_ParameterTypeEnum.PARAMETER_TYPE_ENUM_UNSPECIFIED:
      return 'PARAMETER_TYPE_ENUM_UNSPECIFIED';
    case ParameterType_ParameterTypeEnum.NUMBER_DOUBLE:
      return 'NUMBER_DOUBLE';
    case ParameterType_ParameterTypeEnum.NUMBER_INTEGER:
      return 'NUMBER_INTEGER';
    case ParameterType_ParameterTypeEnum.STRING:
      return 'STRING';
    case ParameterType_ParameterTypeEnum.BOOLEAN:
      return 'BOOLEAN';
    case ParameterType_ParameterTypeEnum.LIST:
      return 'LIST';
    case ParameterType_ParameterTypeEnum.STRUCT:
      return 'STRUCT';
    default:
      return 'UNKNOWN';
  }
}

/** The spec of a pipeline task. */
export interface PipelineTaskSpec {
  /** Basic info of a pipeline task. */
  taskInfo: PipelineTaskInfo | undefined;
  /** Specification for task inputs which contains parameters and artifacts. */
  inputs: TaskInputsSpec | undefined;
  /**
   * A list of names of upstream tasks that do not provide input
   * artifacts for this task, but nonetheless whose completion this task depends
   * on.
   */
  dependentTasks: string[];
  cachingOptions: PipelineTaskSpec_CachingOptions | undefined;
  /**
   * Reference to a component.  Use this field to define either a DAG or an
   * executor.
   */
  componentRef: ComponentRef | undefined;
  /** Trigger policy of the task. */
  triggerPolicy: PipelineTaskSpec_TriggerPolicy | undefined;
  /** Iterator to iterate over an artifact input. */
  artifactIterator: ArtifactIteratorSpec | undefined;
  /** Iterator to iterate over a parameter input. */
  parameterIterator: ParameterIteratorSpec | undefined;
}

export interface PipelineTaskSpec_CachingOptions {
  /** Whether or not to enable cache for this task. Defaults to false. */
  enableCache: boolean;
}

/**
 * Trigger policy defines how the task gets triggered. If a task is not
 * triggered, it will run into SKIPPED state.
 */
export interface PipelineTaskSpec_TriggerPolicy {
  /**
   * An expression which will be evaluated into a boolean value. True to
   * trigger the task to run. The expression follows the language of
   * [CEL Spec][https://github.com/google/cel-spec]. It can access the data
   * from [ExecutorInput][] message of the task.
   * For example:
   * - `inputs.artifacts['model'][0].properties['accuracy']*100 > 90`
   * - `inputs.parameters['type'] == 'foo' && inputs.parameters['num'] == 1`
   */
  condition: string;
  /**
   * The trigger strategy of this task.  The `strategy` and `condition` are
   * in logic "AND", as a task will only be tested for the `condition` when
   * the `strategy` is meet.
   * Unset or set to default value of TRIGGER_STATEGY_UNDEFINED behaves the
   * same as ALL_UPSTREAM_TASKS_SUCCEEDED.
   */
  strategy: PipelineTaskSpec_TriggerPolicy_TriggerStrategy;
}

/**
 * An enum defines the trigger strategy of when the task will be ready to be
 * triggered.
 * ALL_UPSTREAM_TASKS_SUCCEEDED - all upstream tasks in succeeded state.
 * ALL_UPSTREAM_TASKS_COMPLETED - all upstream tasks in any final state.
 * (Note that CANCELLED is also a final state but job will not trigger new
 * tasks when job is in CANCELLING state, so that the task with the trigger
 * policy at ALL_UPSTREAM_TASKS_COMPLETED will not start when job
 * cancellation is in progress.)
 */
export enum PipelineTaskSpec_TriggerPolicy_TriggerStrategy {
  /** TRIGGER_STRATEGY_UNSPECIFIED - Unspecified.  Behave the same as ALL_UPSTREAM_TASKS_SUCCEEDED. */
  TRIGGER_STRATEGY_UNSPECIFIED = 0,
  /** ALL_UPSTREAM_TASKS_SUCCEEDED - Specifies that all upstream tasks are in succeeded state. */
  ALL_UPSTREAM_TASKS_SUCCEEDED = 1,
  /** ALL_UPSTREAM_TASKS_COMPLETED - Specifies that all upstream tasks are in any final state. */
  ALL_UPSTREAM_TASKS_COMPLETED = 2,
  UNRECOGNIZED = -1,
}

export function pipelineTaskSpec_TriggerPolicy_TriggerStrategyFromJSON(
  object: any,
): PipelineTaskSpec_TriggerPolicy_TriggerStrategy {
  switch (object) {
    case 0:
    case 'TRIGGER_STRATEGY_UNSPECIFIED':
      return PipelineTaskSpec_TriggerPolicy_TriggerStrategy.TRIGGER_STRATEGY_UNSPECIFIED;
    case 1:
    case 'ALL_UPSTREAM_TASKS_SUCCEEDED':
      return PipelineTaskSpec_TriggerPolicy_TriggerStrategy.ALL_UPSTREAM_TASKS_SUCCEEDED;
    case 2:
    case 'ALL_UPSTREAM_TASKS_COMPLETED':
      return PipelineTaskSpec_TriggerPolicy_TriggerStrategy.ALL_UPSTREAM_TASKS_COMPLETED;
    case -1:
    case 'UNRECOGNIZED':
    default:
      return PipelineTaskSpec_TriggerPolicy_TriggerStrategy.UNRECOGNIZED;
  }
}

export function pipelineTaskSpec_TriggerPolicy_TriggerStrategyToJSON(
  object: PipelineTaskSpec_TriggerPolicy_TriggerStrategy,
): string {
  switch (object) {
    case PipelineTaskSpec_TriggerPolicy_TriggerStrategy.TRIGGER_STRATEGY_UNSPECIFIED:
      return 'TRIGGER_STRATEGY_UNSPECIFIED';
    case PipelineTaskSpec_TriggerPolicy_TriggerStrategy.ALL_UPSTREAM_TASKS_SUCCEEDED:
      return 'ALL_UPSTREAM_TASKS_SUCCEEDED';
    case PipelineTaskSpec_TriggerPolicy_TriggerStrategy.ALL_UPSTREAM_TASKS_COMPLETED:
      return 'ALL_UPSTREAM_TASKS_COMPLETED';
    default:
      return 'UNKNOWN';
  }
}

/**
 * The spec of an artifact iterator. It supports fan-out a workflow from a list
 * of artifacts.
 */
export interface ArtifactIteratorSpec {
  /** The items to iterate. */
  items: ArtifactIteratorSpec_ItemsSpec | undefined;
  /**
   * The name of the input artifact channel which has the artifact item from the
   * [items][] collection.
   */
  itemInput: string;
}

/**
 * Specifies the name of the artifact channel which contains the collection of
 * items to iterate. The iterator will create a sub-task for each item of
 * the collection and pass the item as a new input artifact channel as
 * specified by [item_input][].
 */
export interface ArtifactIteratorSpec_ItemsSpec {
  /** The name of the input artifact. */
  inputArtifact: string;
}

/**
 * The spec of a parameter iterator. It supports fan-out a workflow from a
 * string parameter which contains a JSON array.
 */
export interface ParameterIteratorSpec {
  /** The items to iterate. */
  items: ParameterIteratorSpec_ItemsSpec | undefined;
  /**
   * The name of the input parameter which has the item value from the
   * [items][] collection.
   */
  itemInput: string;
}

/** Specifies the spec to decribe the parameter items to iterate. */
export interface ParameterIteratorSpec_ItemsSpec {
  /** The raw JSON array. */
  raw: string | undefined;
  /**
   * The name of the input parameter whose value has the items collection.
   * The parameter must be in STRING type and its content can be parsed
   * as a JSON array.
   */
  inputParameter: string | undefined;
}

export interface ComponentRef {
  /**
   * The name of a component. Refer to the key of the
   * [PipelineSpec.components][] map.
   */
  name: string;
}

/** Basic info of a pipeline. */
export interface PipelineInfo {
  /**
   * Required field. The name of the pipeline.
   * The name will be used to create or find pipeline context in MLMD.
   */
  name: string;
}

/** The definition of a artifact type in MLMD. */
export interface ArtifactTypeSchema {
  /**
   * The name of the type. The format of the title must be:
   * `<namespace>.<title>`.
   * Examples:
   *  - `aiplatform.Model`
   *  - `acme.CustomModel`
   * When this field is set, the type must be pre-registered in the MLMD
   * store.
   */
  schemaTitle: string | undefined;
  /**
   * Points to a YAML file stored on Google Cloud Storage describing the
   * format.
   * Deprecated. Use [PipelineArtifactTypeSchema.schema_title][] or
   * [PipelineArtifactTypeSchema.instance_schema][] instead.
   *
   * @deprecated
   */
  schemaUri: string | undefined;
  /**
   * Contains a raw YAML string, describing the format of
   * the properties of the type.
   */
  instanceSchema: string | undefined;
  /**
   * The schema version of the artifact. If the value is not set, it defaults
   * to the the latest version in the system.
   */
  schemaVersion: string;
}

/** The basic info of a task. */
export interface PipelineTaskInfo {
  /** The display name of the task. */
  name: string;
}

/**
 * Definition for a value or reference to a runtime parameter. A
 * ValueOrRuntimeParameter instance can be either a field value that is
 * determined during compilation time, or a runtime parameter which will be
 * determined during runtime.
 */
export interface ValueOrRuntimeParameter {
  /**
   * Constant value which is determined in compile time.
   * Deprecated. Use [ValueOrRuntimeParameter.constant][] instead.
   *
   * @deprecated
   */
  constantValue: Value | undefined;
  /** The runtime parameter refers to the parent component input parameter. */
  runtimeParameter: string | undefined;
  /** Constant value which is determined in compile time. */
  constant: any | undefined;
}

/**
 * The definition of the deployment config of the pipeline. It contains the
 * the platform specific executor configs for KFP OSS.
 */
export interface PipelineDeploymentConfig {
  /** Map from executor label to executor spec. */
  executors: { [key: string]: PipelineDeploymentConfig_ExecutorSpec };
}

/**
 * The specification on a container invocation.
 * The string fields of the message support string based placeholder contract
 * defined in [ExecutorInput](). The output of the container follows the
 * contract of [ExecutorOutput]().
 */
export interface PipelineDeploymentConfig_PipelineContainerSpec {
  /** The image uri of the container. */
  image: string;
  /**
   * The main entrypoint commands of the container to run. If not provided,
   * fallback to use the entry point command defined in the container image.
   */
  command: string[];
  /** The arguments to pass into the main entrypoint of the container. */
  args: string[];
  /** The lifecycle hooks of the container executor. */
  lifecycle: PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle | undefined;
  resources: PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec | undefined;
  /** Environment variables to be passed to the container. */
  env: PipelineDeploymentConfig_PipelineContainerSpec_EnvVar[];
}

/**
 * The lifecycle hooks of the container.
 * Each hook follows the same I/O contract as the main container entrypoint.
 * See [ExecutorInput]() and [ExecutorOutput]() for details.
 * (-- TODO(b/165323565): add more documentation on caching and lifecycle
 * hooks. --)
 */
export interface PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle {
  /**
   * This hook is invoked before caching check. It can change the properties
   * of the execution and output artifacts before they are used to compute
   * the cache key. The updated metadata will be passed into the main
   * container entrypoint.
   */
  preCacheCheck: PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec | undefined;
}

/** The command and args to execute a program. */
export interface PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec {
  /** The command of the exec program. */
  command: string[];
  /** The args of the exec program. */
  args: string[];
}

/**
 * The specification on the resource requirements of a container execution.
 * This can include specification of vCPU, memory requirements, as well as
 * accelerator types and counts.
 */
export interface PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec {
  /**
   * The limit of the number of vCPU cores. This container execution needs
   * at most cpu_limit vCPU to run.
   */
  cpuLimit: number;
  /**
   * The memory limit in GB. This container execution needs at most
   * memory_limit RAM to run.
   */
  memoryLimit: number;
  accelerator:
    | PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig
    | undefined;
}

/** The specification on the accelerators being attached to this container. */
export interface PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig {
  /** The type of accelerators. */
  type: string;
  /** The number of accelerators. */
  count: number;
}

/**
 * Environment variables to be passed to the container.
 * Represents an environment variable present in a container.
 */
export interface PipelineDeploymentConfig_PipelineContainerSpec_EnvVar {
  /**
   * Name of the environment variable. Must be a valid C identifier. It can
   * be composed of characters such as uppercase, lowercase characters,
   * underscore, digits, but the leading character should be either a
   * letter or an underscore.
   */
  name: string;
  /**
   * Variables that reference a $(VAR_NAME) are expanded using the previous
   * defined environment variables in the container and any environment
   * variables defined by the platform runtime that executes this pipeline.
   * If a variable cannot be resolved, the reference in the input string
   * will be unchanged. The $(VAR_NAME) syntax can be escaped with a double
   * $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
   * regardless of whether the variable exists or not.
   */
  value: string;
}

/** The specification to import or reimport a new artifact to the pipeline. */
export interface PipelineDeploymentConfig_ImporterSpec {
  /** The URI of the artifact. */
  artifactUri: ValueOrRuntimeParameter | undefined;
  /** The type of the artifact. */
  typeSchema: ArtifactTypeSchema | undefined;
  /**
   * The properties of the artifact.
   * Deprecated. Use [ImporterSpec.metadata][] instead.
   *
   * @deprecated
   */
  properties: { [key: string]: ValueOrRuntimeParameter };
  /**
   * The custom properties of the artifact.
   * Deprecated. Use [ImporterSpec.metadata][] instead.
   *
   * @deprecated
   */
  customProperties: { [key: string]: ValueOrRuntimeParameter };
  /** Properties of the Artifact. */
  metadata: { [key: string]: any } | undefined;
  /** Whether or not import an artifact regardless it has been imported before. */
  reimport: boolean;
}

export interface PipelineDeploymentConfig_ImporterSpec_PropertiesEntry {
  key: string;
  value: ValueOrRuntimeParameter | undefined;
}

export interface PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry {
  key: string;
  value: ValueOrRuntimeParameter | undefined;
}

/**
 * ResolverSpec resolves artifacts from historical metadata and returns them
 * to the pipeline as output artifacts of the resolver task. The downstream
 * tasks can consume them as their input artifacts.
 */
export interface PipelineDeploymentConfig_ResolverSpec {
  /**
   * A list of resolver output definitions. The
   * key of the map must be exactly the same as
   * the keys in the [PipelineTaskOutputsSpec.artifacts][] map.
   * At least one output must be defined.
   */
  outputArtifactQueries: { [key: string]: PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec };
}

/** The query to fetch artifacts. */
export interface PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec {
  /**
   * The filter of the artifact query. The supported syntax are:
   * - `in_context("<context name>")`
   * - `artifact_type="<artifact type name>"`
   * - `uri="<uri>"`
   * - `state=<state>`
   * - `name="value"`
   * - `AND` to combine two conditions and returns when both are true.
   * If no `in_context` filter is set, the query will be scoped to the
   * the current pipeline context.
   */
  filter: string;
  /**
   * The maximum number of the artifacts to be returned from the
   * query. If not defined, the default limit is `1`.
   */
  limit: number;
}

export interface PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry {
  key: string;
  value: PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec | undefined;
}

/** @deprecated */
export interface PipelineDeploymentConfig_AIPlatformCustomJobSpec {
  /**
   * API Specification for invoking a Google Cloud AI Platform CustomJob.
   * The fields must match the field names and structures of CustomJob
   * defined in
   * https://cloud.google.com/ai-platform-unified/docs/reference/rest/v1beta1/projects.locations.customJobs.
   * The field types must be either the same, or be a string containing the
   * string based placeholder contract defined in [ExecutorInput](). The
   * placeholders will be replaced with the actual value during the runtime
   * before the job is launched.
   */
  customJob: { [key: string]: any } | undefined;
}

/** The specification of the executor. */
export interface PipelineDeploymentConfig_ExecutorSpec {
  /** Starts a container. */
  container: PipelineDeploymentConfig_PipelineContainerSpec | undefined;
  /** Import an artifact. */
  importer: PipelineDeploymentConfig_ImporterSpec | undefined;
  /** Resolves an existing artifact. */
  resolver: PipelineDeploymentConfig_ResolverSpec | undefined;
  /**
   * Starts a Google Cloud AI Platform CustomJob.
   *
   * @deprecated
   */
  customJob: PipelineDeploymentConfig_AIPlatformCustomJobSpec | undefined;
}

export interface PipelineDeploymentConfig_ExecutorsEntry {
  key: string;
  value: PipelineDeploymentConfig_ExecutorSpec | undefined;
}

/** Value is the value of the field. */
export interface Value {
  /** An integer value */
  intValue: number | undefined;
  /** A double value */
  doubleValue: number | undefined;
  /** A string value */
  stringValue: string | undefined;
}

/** The definition of a runtime artifact. */
export interface RuntimeArtifact {
  /** The name of an artifact. */
  name: string;
  /** The type of the artifact. */
  type: ArtifactTypeSchema | undefined;
  /** The URI of the artifact. */
  uri: string;
  /**
   * The properties of the artifact.
   * Deprecated. Use [RuntimeArtifact.metadata][] instead.
   *
   * @deprecated
   */
  properties: { [key: string]: Value };
  /**
   * The custom properties of the artifact.
   * Deprecated. Use [RuntimeArtifact.metadata][] instead.
   *
   * @deprecated
   */
  customProperties: { [key: string]: Value };
  /** Properties of the Artifact. */
  metadata: { [key: string]: any } | undefined;
}

export interface RuntimeArtifact_PropertiesEntry {
  key: string;
  value: Value | undefined;
}

export interface RuntimeArtifact_CustomPropertiesEntry {
  key: string;
  value: Value | undefined;
}

/** Message that represents a list of artifacts. */
export interface ArtifactList {
  /** A list of artifacts. */
  artifacts: RuntimeArtifact[];
}

/**
 * The input of an executor, which includes all the data that
 * can be passed into the executor spec by a string based placeholder.
 *
 * The string based placeholder uses a JSON path to reference to the data
 * in the [ExecutionInput]().
 *
 * `{{$}}`: prints the full [ExecutorInput]() as a JSON string.
 * `{{$.inputs.artifacts['<name>'].uri}}`: prints the URI of an input
 * artifact.
 * `{{$.inputs.artifacts['<name>'].properties['<property name>']}}`: prints
 * the
 *   property of an input artifact.
 * `{{$.inputs.parameters['<name>']}}`: prints the value of an input
 * parameter.
 * `{{$.outputs.artifacts['<name>'].uri}}: prints the URI of an output artifact.
 * `{{$.outputs.artifacts['<name>'].properties['<property name>']}}`: prints the
 *   property of an output artifact.
 * `{{$.outputs.parameters['<name>'].output_file}}`: prints a file path which
 * points to a file and container can write to it to return the value of the
 * parameter..
 * `{{$.outputs.output_file}}`: prints a file path of the output metadata file
 * which is used to send output metadata from executor to orchestrator. The
 * contract of the output metadata is [ExecutorOutput](). When both parameter
 * output file and executor output metadata files are set by the container, the
 * output metadata file will have higher precedence to set output parameters.
 */
export interface ExecutorInput {
  /** The runtime input artifacts of the task invocation. */
  inputs: ExecutorInput_Inputs | undefined;
  /** The runtime output artifacts of the task invocation. */
  outputs: ExecutorInput_Outputs | undefined;
}

/** The runtime inputs data of the execution. */
export interface ExecutorInput_Inputs {
  /**
   * Input parameters of the execution.
   * Deprecated. Use [ExecutorInput.Inputs.parameter_values][] instead.
   *
   * @deprecated
   */
  parameters: { [key: string]: Value };
  /** Input artifacts of the execution. */
  artifacts: { [key: string]: ArtifactList };
  /** Input parameters of the execution. */
  parameterValues: { [key: string]: any | undefined };
}

export interface ExecutorInput_Inputs_ParametersEntry {
  key: string;
  value: Value | undefined;
}

export interface ExecutorInput_Inputs_ArtifactsEntry {
  key: string;
  value: ArtifactList | undefined;
}

export interface ExecutorInput_Inputs_ParameterValuesEntry {
  key: string;
  value: any | undefined;
}

/** The runtime output parameter. */
export interface ExecutorInput_OutputParameter {
  /**
   * The file path which is used by the executor to pass the parameter value
   * to the system.
   */
  outputFile: string;
}

/** The runtime outputs data of the execution. */
export interface ExecutorInput_Outputs {
  /** The runtime output parameters. */
  parameters: { [key: string]: ExecutorInput_OutputParameter };
  /** The runtime output artifacts. */
  artifacts: { [key: string]: ArtifactList };
  /**
   * The file path of the full output metadata JSON. The schema of the output
   * file is [ExecutorOutput][].
   *
   * When the full output metadata file is set by the container, the output
   * parameter files will be ignored.
   */
  outputFile: string;
}

export interface ExecutorInput_Outputs_ParametersEntry {
  key: string;
  value: ExecutorInput_OutputParameter | undefined;
}

export interface ExecutorInput_Outputs_ArtifactsEntry {
  key: string;
  value: ArtifactList | undefined;
}

/**
 * The schema of the output metadata of an execution. It will be used to parse
 * the output metadata file.
 */
export interface ExecutorOutput {
  /**
   * The values for output parameters.
   * Deprecated. Use [ExecutorOutput.parameter_values][] instead.
   *
   * @deprecated
   */
  parameters: { [key: string]: Value };
  /** The updated metadata for output artifact. */
  artifacts: { [key: string]: ArtifactList };
  /** The values for output parameters. */
  parameterValues: { [key: string]: any | undefined };
}

export interface ExecutorOutput_ParametersEntry {
  key: string;
  value: Value | undefined;
}

export interface ExecutorOutput_ArtifactsEntry {
  key: string;
  value: ArtifactList | undefined;
}

export interface ExecutorOutput_ParameterValuesEntry {
  key: string;
  value: any | undefined;
}

/**
 * The final status of a task. The structure will be passed to input parameter
 * of kind `task_final_status`.
 */
export interface PipelineTaskFinalStatus {
  /**
   * The final state of the task.
   * The value is the string version of [PipelineStateEnum.PipelineTaskState][]
   */
  state: string;
  /** The error of the task. */
  error: Status | undefined;
  /**
   * The pipeline job unique id.
   *
   * @deprecated
   */
  pipelineJobUuid: number;
  /**
   * The pipeline job name from the [PipelineJob.name][].
   *
   * @deprecated
   */
  pipelineJobName: string;
  /**
   * The pipeline job resource name, in the format of
   * `projects/{project}/locations/{location}/pipelineJobs/{pipeline_job}`.
   */
  pipelineJobResourceName: string;
}

export interface PipelineStateEnum {}

export enum PipelineStateEnum_PipelineTaskState {
  TASK_STATE_UNSPECIFIED = 0,
  PENDING = 1,
  RUNNING_DRIVER = 2,
  DRIVER_SUCCEEDED = 3,
  RUNNING_EXECUTOR = 4,
  SUCCEEDED = 5,
  CANCEL_PENDING = 6,
  CANCELLING = 7,
  CANCELLED = 8,
  FAILED = 9,
  /** SKIPPED - Indicates that the task is skipped to run due to a cache hit. */
  SKIPPED = 10,
  /**
   * QUEUED - Indicates that the task was just populated to the DB but not ready to
   * be scheduled.  Once job handler determined the task being ready to
   * be scheduled, the task state will change to PENDING.  The state
   * transition is depicted below:
   *  * QUEUED(not ready to run) --> PENDING(ready to run) --> RUNNING
   */
  QUEUED = 11,
  /**
   * NOT_TRIGGERED - Indicates that the task is not triggered based on the
   * [PipelineTaskSpec.TriggerPolicy.condition][] config.
   */
  NOT_TRIGGERED = 12,
  /**
   * UNSCHEDULABLE - Indicates that the tasks will no longer be schedulable.  Usually a task
   * was set to this state because its all upstream tasks are in final state
   * but the [PipelineTaskSpec.TriggerPolicy.strategy][] disallows the task to
   * be triggered.
   * The difference between `NOT_TRIGGERED` is that `UNSCHEDULABLE` must met
   * [PipelineTaskSpec.TriggerPolicy.strategy][], but must not met the
   * [PipelineTaskSpec.TriggerPolicy.condition][].
   */
  UNSCHEDULABLE = 13,
  UNRECOGNIZED = -1,
}

export function pipelineStateEnum_PipelineTaskStateFromJSON(
  object: any,
): PipelineStateEnum_PipelineTaskState {
  switch (object) {
    case 0:
    case 'TASK_STATE_UNSPECIFIED':
      return PipelineStateEnum_PipelineTaskState.TASK_STATE_UNSPECIFIED;
    case 1:
    case 'PENDING':
      return PipelineStateEnum_PipelineTaskState.PENDING;
    case 2:
    case 'RUNNING_DRIVER':
      return PipelineStateEnum_PipelineTaskState.RUNNING_DRIVER;
    case 3:
    case 'DRIVER_SUCCEEDED':
      return PipelineStateEnum_PipelineTaskState.DRIVER_SUCCEEDED;
    case 4:
    case 'RUNNING_EXECUTOR':
      return PipelineStateEnum_PipelineTaskState.RUNNING_EXECUTOR;
    case 5:
    case 'SUCCEEDED':
      return PipelineStateEnum_PipelineTaskState.SUCCEEDED;
    case 6:
    case 'CANCEL_PENDING':
      return PipelineStateEnum_PipelineTaskState.CANCEL_PENDING;
    case 7:
    case 'CANCELLING':
      return PipelineStateEnum_PipelineTaskState.CANCELLING;
    case 8:
    case 'CANCELLED':
      return PipelineStateEnum_PipelineTaskState.CANCELLED;
    case 9:
    case 'FAILED':
      return PipelineStateEnum_PipelineTaskState.FAILED;
    case 10:
    case 'SKIPPED':
      return PipelineStateEnum_PipelineTaskState.SKIPPED;
    case 11:
    case 'QUEUED':
      return PipelineStateEnum_PipelineTaskState.QUEUED;
    case 12:
    case 'NOT_TRIGGERED':
      return PipelineStateEnum_PipelineTaskState.NOT_TRIGGERED;
    case 13:
    case 'UNSCHEDULABLE':
      return PipelineStateEnum_PipelineTaskState.UNSCHEDULABLE;
    case -1:
    case 'UNRECOGNIZED':
    default:
      return PipelineStateEnum_PipelineTaskState.UNRECOGNIZED;
  }
}

export function pipelineStateEnum_PipelineTaskStateToJSON(
  object: PipelineStateEnum_PipelineTaskState,
): string {
  switch (object) {
    case PipelineStateEnum_PipelineTaskState.TASK_STATE_UNSPECIFIED:
      return 'TASK_STATE_UNSPECIFIED';
    case PipelineStateEnum_PipelineTaskState.PENDING:
      return 'PENDING';
    case PipelineStateEnum_PipelineTaskState.RUNNING_DRIVER:
      return 'RUNNING_DRIVER';
    case PipelineStateEnum_PipelineTaskState.DRIVER_SUCCEEDED:
      return 'DRIVER_SUCCEEDED';
    case PipelineStateEnum_PipelineTaskState.RUNNING_EXECUTOR:
      return 'RUNNING_EXECUTOR';
    case PipelineStateEnum_PipelineTaskState.SUCCEEDED:
      return 'SUCCEEDED';
    case PipelineStateEnum_PipelineTaskState.CANCEL_PENDING:
      return 'CANCEL_PENDING';
    case PipelineStateEnum_PipelineTaskState.CANCELLING:
      return 'CANCELLING';
    case PipelineStateEnum_PipelineTaskState.CANCELLED:
      return 'CANCELLED';
    case PipelineStateEnum_PipelineTaskState.FAILED:
      return 'FAILED';
    case PipelineStateEnum_PipelineTaskState.SKIPPED:
      return 'SKIPPED';
    case PipelineStateEnum_PipelineTaskState.QUEUED:
      return 'QUEUED';
    case PipelineStateEnum_PipelineTaskState.NOT_TRIGGERED:
      return 'NOT_TRIGGERED';
    case PipelineStateEnum_PipelineTaskState.UNSCHEDULABLE:
      return 'UNSCHEDULABLE';
    default:
      return 'UNKNOWN';
  }
}

const basePipelineJob: object = { name: '', displayName: '' };

export const PipelineJob = {
  encode(message: PipelineJob, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== '') {
      writer.uint32(10).string(message.name);
    }
    if (message.displayName !== '') {
      writer.uint32(18).string(message.displayName);
    }
    if (message.pipelineSpec !== undefined) {
      Struct.encode(Struct.wrap(message.pipelineSpec), writer.uint32(58).fork()).ldelim();
    }
    Object.entries(message.labels).forEach(([key, value]) => {
      PipelineJob_LabelsEntry.encode({ key: key as any, value }, writer.uint32(90).fork()).ldelim();
    });
    if (message.runtimeConfig !== undefined) {
      PipelineJob_RuntimeConfig.encode(message.runtimeConfig, writer.uint32(98).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineJob {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineJob } as PipelineJob;
    message.labels = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.displayName = reader.string();
          break;
        case 7:
          message.pipelineSpec = Struct.unwrap(Struct.decode(reader, reader.uint32()));
          break;
        case 11:
          const entry11 = PipelineJob_LabelsEntry.decode(reader, reader.uint32());
          if (entry11.value !== undefined) {
            message.labels[entry11.key] = entry11.value;
          }
          break;
        case 12:
          message.runtimeConfig = PipelineJob_RuntimeConfig.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineJob {
    const message = { ...basePipelineJob } as PipelineJob;
    message.name = object.name !== undefined && object.name !== null ? String(object.name) : '';
    message.displayName =
      object.displayName !== undefined && object.displayName !== null
        ? String(object.displayName)
        : '';
    message.pipelineSpec =
      typeof object.pipelineSpec === 'object' ? object.pipelineSpec : undefined;
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>(
      (acc, [key, value]) => {
        acc[key] = String(value);
        return acc;
      },
      {},
    );
    message.runtimeConfig =
      object.runtimeConfig !== undefined && object.runtimeConfig !== null
        ? PipelineJob_RuntimeConfig.fromJSON(object.runtimeConfig)
        : undefined;
    return message;
  },

  toJSON(message: PipelineJob): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.displayName !== undefined && (obj.displayName = message.displayName);
    message.pipelineSpec !== undefined && (obj.pipelineSpec = message.pipelineSpec);
    obj.labels = {};
    if (message.labels) {
      Object.entries(message.labels).forEach(([k, v]) => {
        obj.labels[k] = v;
      });
    }
    message.runtimeConfig !== undefined &&
      (obj.runtimeConfig = message.runtimeConfig
        ? PipelineJob_RuntimeConfig.toJSON(message.runtimeConfig)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineJob>, I>>(object: I): PipelineJob {
    const message = { ...basePipelineJob } as PipelineJob;
    message.name = object.name ?? '';
    message.displayName = object.displayName ?? '';
    message.pipelineSpec = object.pipelineSpec ?? undefined;
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = String(value);
        }
        return acc;
      },
      {},
    );
    message.runtimeConfig =
      object.runtimeConfig !== undefined && object.runtimeConfig !== null
        ? PipelineJob_RuntimeConfig.fromPartial(object.runtimeConfig)
        : undefined;
    return message;
  },
};

const basePipelineJob_LabelsEntry: object = { key: '', value: '' };

export const PipelineJob_LabelsEntry = {
  encode(message: PipelineJob_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== '') {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineJob_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineJob_LabelsEntry } as PipelineJob_LabelsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineJob_LabelsEntry {
    const message = { ...basePipelineJob_LabelsEntry } as PipelineJob_LabelsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value = object.value !== undefined && object.value !== null ? String(object.value) : '';
    return message;
  },

  toJSON(message: PipelineJob_LabelsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineJob_LabelsEntry>, I>>(
    object: I,
  ): PipelineJob_LabelsEntry {
    const message = { ...basePipelineJob_LabelsEntry } as PipelineJob_LabelsEntry;
    message.key = object.key ?? '';
    message.value = object.value ?? '';
    return message;
  },
};

const basePipelineJob_RuntimeConfig: object = { gcsOutputDirectory: '' };

export const PipelineJob_RuntimeConfig = {
  encode(message: PipelineJob_RuntimeConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.parameters).forEach(([key, value]) => {
      PipelineJob_RuntimeConfig_ParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    if (message.gcsOutputDirectory !== '') {
      writer.uint32(18).string(message.gcsOutputDirectory);
    }
    Object.entries(message.parameterValues).forEach(([key, value]) => {
      if (value !== undefined) {
        PipelineJob_RuntimeConfig_ParameterValuesEntry.encode(
          { key: key as any, value },
          writer.uint32(26).fork(),
        ).ldelim();
      }
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineJob_RuntimeConfig {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineJob_RuntimeConfig } as PipelineJob_RuntimeConfig;
    message.parameters = {};
    message.parameterValues = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = PipelineJob_RuntimeConfig_ParametersEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.parameters[entry1.key] = entry1.value;
          }
          break;
        case 2:
          message.gcsOutputDirectory = reader.string();
          break;
        case 3:
          const entry3 = PipelineJob_RuntimeConfig_ParameterValuesEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry3.value !== undefined) {
            message.parameterValues[entry3.key] = entry3.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineJob_RuntimeConfig {
    const message = { ...basePipelineJob_RuntimeConfig } as PipelineJob_RuntimeConfig;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{ [key: string]: Value }>(
      (acc, [key, value]) => {
        acc[key] = Value.fromJSON(value);
        return acc;
      },
      {},
    );
    message.gcsOutputDirectory =
      object.gcsOutputDirectory !== undefined && object.gcsOutputDirectory !== null
        ? String(object.gcsOutputDirectory)
        : '';
    message.parameterValues = Object.entries(object.parameterValues ?? {}).reduce<{
      [key: string]: any | undefined;
    }>((acc, [key, value]) => {
      acc[key] = value as any | undefined;
      return acc;
    }, {});
    return message;
  },

  toJSON(message: PipelineJob_RuntimeConfig): unknown {
    const obj: any = {};
    obj.parameters = {};
    if (message.parameters) {
      Object.entries(message.parameters).forEach(([k, v]) => {
        obj.parameters[k] = Value.toJSON(v);
      });
    }
    message.gcsOutputDirectory !== undefined &&
      (obj.gcsOutputDirectory = message.gcsOutputDirectory);
    obj.parameterValues = {};
    if (message.parameterValues) {
      Object.entries(message.parameterValues).forEach(([k, v]) => {
        obj.parameterValues[k] = v;
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineJob_RuntimeConfig>, I>>(
    object: I,
  ): PipelineJob_RuntimeConfig {
    const message = { ...basePipelineJob_RuntimeConfig } as PipelineJob_RuntimeConfig;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{ [key: string]: Value }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = Value.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    message.gcsOutputDirectory = object.gcsOutputDirectory ?? '';
    message.parameterValues = Object.entries(object.parameterValues ?? {}).reduce<{
      [key: string]: any | undefined;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = value;
      }
      return acc;
    }, {});
    return message;
  },
};

const basePipelineJob_RuntimeConfig_ParametersEntry: object = { key: '' };

export const PipelineJob_RuntimeConfig_ParametersEntry = {
  encode(
    message: PipelineJob_RuntimeConfig_ParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Value.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineJob_RuntimeConfig_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineJob_RuntimeConfig_ParametersEntry,
    } as PipelineJob_RuntimeConfig_ParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = Value.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineJob_RuntimeConfig_ParametersEntry {
    const message = {
      ...basePipelineJob_RuntimeConfig_ParametersEntry,
    } as PipelineJob_RuntimeConfig_ParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: PipelineJob_RuntimeConfig_ParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? Value.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineJob_RuntimeConfig_ParametersEntry>, I>>(
    object: I,
  ): PipelineJob_RuntimeConfig_ParametersEntry {
    const message = {
      ...basePipelineJob_RuntimeConfig_ParametersEntry,
    } as PipelineJob_RuntimeConfig_ParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const basePipelineJob_RuntimeConfig_ParameterValuesEntry: object = { key: '' };

export const PipelineJob_RuntimeConfig_ParameterValuesEntry = {
  encode(
    message: PipelineJob_RuntimeConfig_ParameterValuesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Value1.encode(Value1.wrap(message.value), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineJob_RuntimeConfig_ParameterValuesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineJob_RuntimeConfig_ParameterValuesEntry,
    } as PipelineJob_RuntimeConfig_ParameterValuesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = Value1.unwrap(Value1.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineJob_RuntimeConfig_ParameterValuesEntry {
    const message = {
      ...basePipelineJob_RuntimeConfig_ParameterValuesEntry,
    } as PipelineJob_RuntimeConfig_ParameterValuesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value = object.value;
    return message;
  },

  toJSON(message: PipelineJob_RuntimeConfig_ParameterValuesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineJob_RuntimeConfig_ParameterValuesEntry>, I>>(
    object: I,
  ): PipelineJob_RuntimeConfig_ParameterValuesEntry {
    const message = {
      ...basePipelineJob_RuntimeConfig_ParameterValuesEntry,
    } as PipelineJob_RuntimeConfig_ParameterValuesEntry;
    message.key = object.key ?? '';
    message.value = object.value ?? undefined;
    return message;
  },
};

const basePipelineSpec: object = { sdkVersion: '', schemaVersion: '', defaultPipelineRoot: '' };

export const PipelineSpec = {
  encode(message: PipelineSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pipelineInfo !== undefined) {
      PipelineInfo.encode(message.pipelineInfo, writer.uint32(10).fork()).ldelim();
    }
    if (message.deploymentSpec !== undefined) {
      Struct.encode(Struct.wrap(message.deploymentSpec), writer.uint32(58).fork()).ldelim();
    }
    if (message.sdkVersion !== '') {
      writer.uint32(34).string(message.sdkVersion);
    }
    if (message.schemaVersion !== '') {
      writer.uint32(42).string(message.schemaVersion);
    }
    Object.entries(message.components).forEach(([key, value]) => {
      PipelineSpec_ComponentsEntry.encode(
        { key: key as any, value },
        writer.uint32(66).fork(),
      ).ldelim();
    });
    if (message.root !== undefined) {
      ComponentSpec.encode(message.root, writer.uint32(74).fork()).ldelim();
    }
    if (message.defaultPipelineRoot !== '') {
      writer.uint32(82).string(message.defaultPipelineRoot);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineSpec } as PipelineSpec;
    message.components = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pipelineInfo = PipelineInfo.decode(reader, reader.uint32());
          break;
        case 7:
          message.deploymentSpec = Struct.unwrap(Struct.decode(reader, reader.uint32()));
          break;
        case 4:
          message.sdkVersion = reader.string();
          break;
        case 5:
          message.schemaVersion = reader.string();
          break;
        case 8:
          const entry8 = PipelineSpec_ComponentsEntry.decode(reader, reader.uint32());
          if (entry8.value !== undefined) {
            message.components[entry8.key] = entry8.value;
          }
          break;
        case 9:
          message.root = ComponentSpec.decode(reader, reader.uint32());
          break;
        case 10:
          message.defaultPipelineRoot = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineSpec {
    const message = { ...basePipelineSpec } as PipelineSpec;
    message.pipelineInfo =
      object.pipelineInfo !== undefined && object.pipelineInfo !== null
        ? PipelineInfo.fromJSON(object.pipelineInfo)
        : undefined;
    message.deploymentSpec =
      typeof object.deploymentSpec === 'object' ? object.deploymentSpec : undefined;
    message.sdkVersion =
      object.sdkVersion !== undefined && object.sdkVersion !== null
        ? String(object.sdkVersion)
        : '';
    message.schemaVersion =
      object.schemaVersion !== undefined && object.schemaVersion !== null
        ? String(object.schemaVersion)
        : '';
    message.components = Object.entries(object.components ?? {}).reduce<{
      [key: string]: ComponentSpec;
    }>((acc, [key, value]) => {
      acc[key] = ComponentSpec.fromJSON(value);
      return acc;
    }, {});
    message.root =
      object.root !== undefined && object.root !== null
        ? ComponentSpec.fromJSON(object.root)
        : undefined;
    message.defaultPipelineRoot =
      object.defaultPipelineRoot !== undefined && object.defaultPipelineRoot !== null
        ? String(object.defaultPipelineRoot)
        : '';
    return message;
  },

  toJSON(message: PipelineSpec): unknown {
    const obj: any = {};
    message.pipelineInfo !== undefined &&
      (obj.pipelineInfo = message.pipelineInfo
        ? PipelineInfo.toJSON(message.pipelineInfo)
        : undefined);
    message.deploymentSpec !== undefined && (obj.deploymentSpec = message.deploymentSpec);
    message.sdkVersion !== undefined && (obj.sdkVersion = message.sdkVersion);
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
    obj.components = {};
    if (message.components) {
      Object.entries(message.components).forEach(([k, v]) => {
        obj.components[k] = ComponentSpec.toJSON(v);
      });
    }
    message.root !== undefined &&
      (obj.root = message.root ? ComponentSpec.toJSON(message.root) : undefined);
    message.defaultPipelineRoot !== undefined &&
      (obj.defaultPipelineRoot = message.defaultPipelineRoot);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineSpec>, I>>(object: I): PipelineSpec {
    const message = { ...basePipelineSpec } as PipelineSpec;
    message.pipelineInfo =
      object.pipelineInfo !== undefined && object.pipelineInfo !== null
        ? PipelineInfo.fromPartial(object.pipelineInfo)
        : undefined;
    message.deploymentSpec = object.deploymentSpec ?? undefined;
    message.sdkVersion = object.sdkVersion ?? '';
    message.schemaVersion = object.schemaVersion ?? '';
    message.components = Object.entries(object.components ?? {}).reduce<{
      [key: string]: ComponentSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ComponentSpec.fromPartial(value);
      }
      return acc;
    }, {});
    message.root =
      object.root !== undefined && object.root !== null
        ? ComponentSpec.fromPartial(object.root)
        : undefined;
    message.defaultPipelineRoot = object.defaultPipelineRoot ?? '';
    return message;
  },
};

const basePipelineSpec_RuntimeParameter: object = { type: 0 };

export const PipelineSpec_RuntimeParameter = {
  encode(
    message: PipelineSpec_RuntimeParameter,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.defaultValue !== undefined) {
      Value.encode(message.defaultValue, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineSpec_RuntimeParameter {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineSpec_RuntimeParameter } as PipelineSpec_RuntimeParameter;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.int32() as any;
          break;
        case 2:
          message.defaultValue = Value.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineSpec_RuntimeParameter {
    const message = { ...basePipelineSpec_RuntimeParameter } as PipelineSpec_RuntimeParameter;
    message.type =
      object.type !== undefined && object.type !== null
        ? primitiveType_PrimitiveTypeEnumFromJSON(object.type)
        : 0;
    message.defaultValue =
      object.defaultValue !== undefined && object.defaultValue !== null
        ? Value.fromJSON(object.defaultValue)
        : undefined;
    return message;
  },

  toJSON(message: PipelineSpec_RuntimeParameter): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = primitiveType_PrimitiveTypeEnumToJSON(message.type));
    message.defaultValue !== undefined &&
      (obj.defaultValue = message.defaultValue ? Value.toJSON(message.defaultValue) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineSpec_RuntimeParameter>, I>>(
    object: I,
  ): PipelineSpec_RuntimeParameter {
    const message = { ...basePipelineSpec_RuntimeParameter } as PipelineSpec_RuntimeParameter;
    message.type = object.type ?? 0;
    message.defaultValue =
      object.defaultValue !== undefined && object.defaultValue !== null
        ? Value.fromPartial(object.defaultValue)
        : undefined;
    return message;
  },
};

const basePipelineSpec_ComponentsEntry: object = { key: '' };

export const PipelineSpec_ComponentsEntry = {
  encode(
    message: PipelineSpec_ComponentsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ComponentSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineSpec_ComponentsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineSpec_ComponentsEntry } as PipelineSpec_ComponentsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ComponentSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineSpec_ComponentsEntry {
    const message = { ...basePipelineSpec_ComponentsEntry } as PipelineSpec_ComponentsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: PipelineSpec_ComponentsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ComponentSpec.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineSpec_ComponentsEntry>, I>>(
    object: I,
  ): PipelineSpec_ComponentsEntry {
    const message = { ...basePipelineSpec_ComponentsEntry } as PipelineSpec_ComponentsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseComponentSpec: object = {};

export const ComponentSpec = {
  encode(message: ComponentSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.inputDefinitions !== undefined) {
      ComponentInputsSpec.encode(message.inputDefinitions, writer.uint32(10).fork()).ldelim();
    }
    if (message.outputDefinitions !== undefined) {
      ComponentOutputsSpec.encode(message.outputDefinitions, writer.uint32(18).fork()).ldelim();
    }
    if (message.dag !== undefined) {
      DagSpec.encode(message.dag, writer.uint32(26).fork()).ldelim();
    }
    if (message.executorLabel !== undefined) {
      writer.uint32(34).string(message.executorLabel);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseComponentSpec } as ComponentSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.inputDefinitions = ComponentInputsSpec.decode(reader, reader.uint32());
          break;
        case 2:
          message.outputDefinitions = ComponentOutputsSpec.decode(reader, reader.uint32());
          break;
        case 3:
          message.dag = DagSpec.decode(reader, reader.uint32());
          break;
        case 4:
          message.executorLabel = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentSpec {
    const message = { ...baseComponentSpec } as ComponentSpec;
    message.inputDefinitions =
      object.inputDefinitions !== undefined && object.inputDefinitions !== null
        ? ComponentInputsSpec.fromJSON(object.inputDefinitions)
        : undefined;
    message.outputDefinitions =
      object.outputDefinitions !== undefined && object.outputDefinitions !== null
        ? ComponentOutputsSpec.fromJSON(object.outputDefinitions)
        : undefined;
    message.dag =
      object.dag !== undefined && object.dag !== null ? DagSpec.fromJSON(object.dag) : undefined;
    message.executorLabel =
      object.executorLabel !== undefined && object.executorLabel !== null
        ? String(object.executorLabel)
        : undefined;
    return message;
  },

  toJSON(message: ComponentSpec): unknown {
    const obj: any = {};
    message.inputDefinitions !== undefined &&
      (obj.inputDefinitions = message.inputDefinitions
        ? ComponentInputsSpec.toJSON(message.inputDefinitions)
        : undefined);
    message.outputDefinitions !== undefined &&
      (obj.outputDefinitions = message.outputDefinitions
        ? ComponentOutputsSpec.toJSON(message.outputDefinitions)
        : undefined);
    message.dag !== undefined && (obj.dag = message.dag ? DagSpec.toJSON(message.dag) : undefined);
    message.executorLabel !== undefined && (obj.executorLabel = message.executorLabel);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentSpec>, I>>(object: I): ComponentSpec {
    const message = { ...baseComponentSpec } as ComponentSpec;
    message.inputDefinitions =
      object.inputDefinitions !== undefined && object.inputDefinitions !== null
        ? ComponentInputsSpec.fromPartial(object.inputDefinitions)
        : undefined;
    message.outputDefinitions =
      object.outputDefinitions !== undefined && object.outputDefinitions !== null
        ? ComponentOutputsSpec.fromPartial(object.outputDefinitions)
        : undefined;
    message.dag =
      object.dag !== undefined && object.dag !== null ? DagSpec.fromPartial(object.dag) : undefined;
    message.executorLabel = object.executorLabel ?? undefined;
    return message;
  },
};

const baseDagSpec: object = {};

export const DagSpec = {
  encode(message: DagSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.tasks).forEach(([key, value]) => {
      DagSpec_TasksEntry.encode({ key: key as any, value }, writer.uint32(10).fork()).ldelim();
    });
    if (message.outputs !== undefined) {
      DagOutputsSpec.encode(message.outputs, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseDagSpec } as DagSpec;
    message.tasks = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = DagSpec_TasksEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.tasks[entry1.key] = entry1.value;
          }
          break;
        case 2:
          message.outputs = DagOutputsSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagSpec {
    const message = { ...baseDagSpec } as DagSpec;
    message.tasks = Object.entries(object.tasks ?? {}).reduce<{ [key: string]: PipelineTaskSpec }>(
      (acc, [key, value]) => {
        acc[key] = PipelineTaskSpec.fromJSON(value);
        return acc;
      },
      {},
    );
    message.outputs =
      object.outputs !== undefined && object.outputs !== null
        ? DagOutputsSpec.fromJSON(object.outputs)
        : undefined;
    return message;
  },

  toJSON(message: DagSpec): unknown {
    const obj: any = {};
    obj.tasks = {};
    if (message.tasks) {
      Object.entries(message.tasks).forEach(([k, v]) => {
        obj.tasks[k] = PipelineTaskSpec.toJSON(v);
      });
    }
    message.outputs !== undefined &&
      (obj.outputs = message.outputs ? DagOutputsSpec.toJSON(message.outputs) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagSpec>, I>>(object: I): DagSpec {
    const message = { ...baseDagSpec } as DagSpec;
    message.tasks = Object.entries(object.tasks ?? {}).reduce<{ [key: string]: PipelineTaskSpec }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = PipelineTaskSpec.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    message.outputs =
      object.outputs !== undefined && object.outputs !== null
        ? DagOutputsSpec.fromPartial(object.outputs)
        : undefined;
    return message;
  },
};

const baseDagSpec_TasksEntry: object = { key: '' };

export const DagSpec_TasksEntry = {
  encode(message: DagSpec_TasksEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      PipelineTaskSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagSpec_TasksEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseDagSpec_TasksEntry } as DagSpec_TasksEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = PipelineTaskSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagSpec_TasksEntry {
    const message = { ...baseDagSpec_TasksEntry } as DagSpec_TasksEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? PipelineTaskSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: DagSpec_TasksEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? PipelineTaskSpec.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagSpec_TasksEntry>, I>>(object: I): DagSpec_TasksEntry {
    const message = { ...baseDagSpec_TasksEntry } as DagSpec_TasksEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? PipelineTaskSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseDagOutputsSpec: object = {};

export const DagOutputsSpec = {
  encode(message: DagOutputsSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.artifacts).forEach(([key, value]) => {
      DagOutputsSpec_ArtifactsEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    Object.entries(message.parameters).forEach(([key, value]) => {
      DagOutputsSpec_ParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagOutputsSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseDagOutputsSpec } as DagOutputsSpec;
    message.artifacts = {};
    message.parameters = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = DagOutputsSpec_ArtifactsEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.artifacts[entry1.key] = entry1.value;
          }
          break;
        case 2:
          const entry2 = DagOutputsSpec_ParametersEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.parameters[entry2.key] = entry2.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec {
    const message = { ...baseDagOutputsSpec } as DagOutputsSpec;
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: DagOutputsSpec_DagOutputArtifactSpec;
    }>((acc, [key, value]) => {
      acc[key] = DagOutputsSpec_DagOutputArtifactSpec.fromJSON(value);
      return acc;
    }, {});
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: DagOutputsSpec_DagOutputParameterSpec;
    }>((acc, [key, value]) => {
      acc[key] = DagOutputsSpec_DagOutputParameterSpec.fromJSON(value);
      return acc;
    }, {});
    return message;
  },

  toJSON(message: DagOutputsSpec): unknown {
    const obj: any = {};
    obj.artifacts = {};
    if (message.artifacts) {
      Object.entries(message.artifacts).forEach(([k, v]) => {
        obj.artifacts[k] = DagOutputsSpec_DagOutputArtifactSpec.toJSON(v);
      });
    }
    obj.parameters = {};
    if (message.parameters) {
      Object.entries(message.parameters).forEach(([k, v]) => {
        obj.parameters[k] = DagOutputsSpec_DagOutputParameterSpec.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagOutputsSpec>, I>>(object: I): DagOutputsSpec {
    const message = { ...baseDagOutputsSpec } as DagOutputsSpec;
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: DagOutputsSpec_DagOutputArtifactSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = DagOutputsSpec_DagOutputArtifactSpec.fromPartial(value);
      }
      return acc;
    }, {});
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: DagOutputsSpec_DagOutputParameterSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = DagOutputsSpec_DagOutputParameterSpec.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

const baseDagOutputsSpec_ArtifactSelectorSpec: object = {
  producerSubtask: '',
  outputArtifactKey: '',
};

export const DagOutputsSpec_ArtifactSelectorSpec = {
  encode(
    message: DagOutputsSpec_ArtifactSelectorSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.producerSubtask !== '') {
      writer.uint32(10).string(message.producerSubtask);
    }
    if (message.outputArtifactKey !== '') {
      writer.uint32(18).string(message.outputArtifactKey);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagOutputsSpec_ArtifactSelectorSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseDagOutputsSpec_ArtifactSelectorSpec,
    } as DagOutputsSpec_ArtifactSelectorSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.producerSubtask = reader.string();
          break;
        case 2:
          message.outputArtifactKey = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec_ArtifactSelectorSpec {
    const message = {
      ...baseDagOutputsSpec_ArtifactSelectorSpec,
    } as DagOutputsSpec_ArtifactSelectorSpec;
    message.producerSubtask =
      object.producerSubtask !== undefined && object.producerSubtask !== null
        ? String(object.producerSubtask)
        : '';
    message.outputArtifactKey =
      object.outputArtifactKey !== undefined && object.outputArtifactKey !== null
        ? String(object.outputArtifactKey)
        : '';
    return message;
  },

  toJSON(message: DagOutputsSpec_ArtifactSelectorSpec): unknown {
    const obj: any = {};
    message.producerSubtask !== undefined && (obj.producerSubtask = message.producerSubtask);
    message.outputArtifactKey !== undefined && (obj.outputArtifactKey = message.outputArtifactKey);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagOutputsSpec_ArtifactSelectorSpec>, I>>(
    object: I,
  ): DagOutputsSpec_ArtifactSelectorSpec {
    const message = {
      ...baseDagOutputsSpec_ArtifactSelectorSpec,
    } as DagOutputsSpec_ArtifactSelectorSpec;
    message.producerSubtask = object.producerSubtask ?? '';
    message.outputArtifactKey = object.outputArtifactKey ?? '';
    return message;
  },
};

const baseDagOutputsSpec_DagOutputArtifactSpec: object = {};

export const DagOutputsSpec_DagOutputArtifactSpec = {
  encode(
    message: DagOutputsSpec_DagOutputArtifactSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    for (const v of message.artifactSelectors) {
      DagOutputsSpec_ArtifactSelectorSpec.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagOutputsSpec_DagOutputArtifactSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseDagOutputsSpec_DagOutputArtifactSpec,
    } as DagOutputsSpec_DagOutputArtifactSpec;
    message.artifactSelectors = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.artifactSelectors.push(
            DagOutputsSpec_ArtifactSelectorSpec.decode(reader, reader.uint32()),
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec_DagOutputArtifactSpec {
    const message = {
      ...baseDagOutputsSpec_DagOutputArtifactSpec,
    } as DagOutputsSpec_DagOutputArtifactSpec;
    message.artifactSelectors = (object.artifactSelectors ?? []).map((e: any) =>
      DagOutputsSpec_ArtifactSelectorSpec.fromJSON(e),
    );
    return message;
  },

  toJSON(message: DagOutputsSpec_DagOutputArtifactSpec): unknown {
    const obj: any = {};
    if (message.artifactSelectors) {
      obj.artifactSelectors = message.artifactSelectors.map((e) =>
        e ? DagOutputsSpec_ArtifactSelectorSpec.toJSON(e) : undefined,
      );
    } else {
      obj.artifactSelectors = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagOutputsSpec_DagOutputArtifactSpec>, I>>(
    object: I,
  ): DagOutputsSpec_DagOutputArtifactSpec {
    const message = {
      ...baseDagOutputsSpec_DagOutputArtifactSpec,
    } as DagOutputsSpec_DagOutputArtifactSpec;
    message.artifactSelectors =
      object.artifactSelectors?.map((e) => DagOutputsSpec_ArtifactSelectorSpec.fromPartial(e)) ||
      [];
    return message;
  },
};

const baseDagOutputsSpec_ArtifactsEntry: object = { key: '' };

export const DagOutputsSpec_ArtifactsEntry = {
  encode(
    message: DagOutputsSpec_ArtifactsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      DagOutputsSpec_DagOutputArtifactSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagOutputsSpec_ArtifactsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseDagOutputsSpec_ArtifactsEntry } as DagOutputsSpec_ArtifactsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = DagOutputsSpec_DagOutputArtifactSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec_ArtifactsEntry {
    const message = { ...baseDagOutputsSpec_ArtifactsEntry } as DagOutputsSpec_ArtifactsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? DagOutputsSpec_DagOutputArtifactSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: DagOutputsSpec_ArtifactsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? DagOutputsSpec_DagOutputArtifactSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagOutputsSpec_ArtifactsEntry>, I>>(
    object: I,
  ): DagOutputsSpec_ArtifactsEntry {
    const message = { ...baseDagOutputsSpec_ArtifactsEntry } as DagOutputsSpec_ArtifactsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? DagOutputsSpec_DagOutputArtifactSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseDagOutputsSpec_ParameterSelectorSpec: object = {
  producerSubtask: '',
  outputParameterKey: '',
};

export const DagOutputsSpec_ParameterSelectorSpec = {
  encode(
    message: DagOutputsSpec_ParameterSelectorSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.producerSubtask !== '') {
      writer.uint32(10).string(message.producerSubtask);
    }
    if (message.outputParameterKey !== '') {
      writer.uint32(18).string(message.outputParameterKey);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagOutputsSpec_ParameterSelectorSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseDagOutputsSpec_ParameterSelectorSpec,
    } as DagOutputsSpec_ParameterSelectorSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.producerSubtask = reader.string();
          break;
        case 2:
          message.outputParameterKey = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec_ParameterSelectorSpec {
    const message = {
      ...baseDagOutputsSpec_ParameterSelectorSpec,
    } as DagOutputsSpec_ParameterSelectorSpec;
    message.producerSubtask =
      object.producerSubtask !== undefined && object.producerSubtask !== null
        ? String(object.producerSubtask)
        : '';
    message.outputParameterKey =
      object.outputParameterKey !== undefined && object.outputParameterKey !== null
        ? String(object.outputParameterKey)
        : '';
    return message;
  },

  toJSON(message: DagOutputsSpec_ParameterSelectorSpec): unknown {
    const obj: any = {};
    message.producerSubtask !== undefined && (obj.producerSubtask = message.producerSubtask);
    message.outputParameterKey !== undefined &&
      (obj.outputParameterKey = message.outputParameterKey);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagOutputsSpec_ParameterSelectorSpec>, I>>(
    object: I,
  ): DagOutputsSpec_ParameterSelectorSpec {
    const message = {
      ...baseDagOutputsSpec_ParameterSelectorSpec,
    } as DagOutputsSpec_ParameterSelectorSpec;
    message.producerSubtask = object.producerSubtask ?? '';
    message.outputParameterKey = object.outputParameterKey ?? '';
    return message;
  },
};

const baseDagOutputsSpec_ParameterSelectorsSpec: object = {};

export const DagOutputsSpec_ParameterSelectorsSpec = {
  encode(
    message: DagOutputsSpec_ParameterSelectorsSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    for (const v of message.parameterSelectors) {
      DagOutputsSpec_ParameterSelectorSpec.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagOutputsSpec_ParameterSelectorsSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseDagOutputsSpec_ParameterSelectorsSpec,
    } as DagOutputsSpec_ParameterSelectorsSpec;
    message.parameterSelectors = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parameterSelectors.push(
            DagOutputsSpec_ParameterSelectorSpec.decode(reader, reader.uint32()),
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec_ParameterSelectorsSpec {
    const message = {
      ...baseDagOutputsSpec_ParameterSelectorsSpec,
    } as DagOutputsSpec_ParameterSelectorsSpec;
    message.parameterSelectors = (object.parameterSelectors ?? []).map((e: any) =>
      DagOutputsSpec_ParameterSelectorSpec.fromJSON(e),
    );
    return message;
  },

  toJSON(message: DagOutputsSpec_ParameterSelectorsSpec): unknown {
    const obj: any = {};
    if (message.parameterSelectors) {
      obj.parameterSelectors = message.parameterSelectors.map((e) =>
        e ? DagOutputsSpec_ParameterSelectorSpec.toJSON(e) : undefined,
      );
    } else {
      obj.parameterSelectors = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagOutputsSpec_ParameterSelectorsSpec>, I>>(
    object: I,
  ): DagOutputsSpec_ParameterSelectorsSpec {
    const message = {
      ...baseDagOutputsSpec_ParameterSelectorsSpec,
    } as DagOutputsSpec_ParameterSelectorsSpec;
    message.parameterSelectors =
      object.parameterSelectors?.map((e) => DagOutputsSpec_ParameterSelectorSpec.fromPartial(e)) ||
      [];
    return message;
  },
};

const baseDagOutputsSpec_MapParameterSelectorsSpec: object = {};

export const DagOutputsSpec_MapParameterSelectorsSpec = {
  encode(
    message: DagOutputsSpec_MapParameterSelectorsSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    Object.entries(message.mappedParameters).forEach(([key, value]) => {
      DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): DagOutputsSpec_MapParameterSelectorsSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseDagOutputsSpec_MapParameterSelectorsSpec,
    } as DagOutputsSpec_MapParameterSelectorsSpec;
    message.mappedParameters = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          const entry2 = DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry2.value !== undefined) {
            message.mappedParameters[entry2.key] = entry2.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec_MapParameterSelectorsSpec {
    const message = {
      ...baseDagOutputsSpec_MapParameterSelectorsSpec,
    } as DagOutputsSpec_MapParameterSelectorsSpec;
    message.mappedParameters = Object.entries(object.mappedParameters ?? {}).reduce<{
      [key: string]: DagOutputsSpec_ParameterSelectorSpec;
    }>((acc, [key, value]) => {
      acc[key] = DagOutputsSpec_ParameterSelectorSpec.fromJSON(value);
      return acc;
    }, {});
    return message;
  },

  toJSON(message: DagOutputsSpec_MapParameterSelectorsSpec): unknown {
    const obj: any = {};
    obj.mappedParameters = {};
    if (message.mappedParameters) {
      Object.entries(message.mappedParameters).forEach(([k, v]) => {
        obj.mappedParameters[k] = DagOutputsSpec_ParameterSelectorSpec.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagOutputsSpec_MapParameterSelectorsSpec>, I>>(
    object: I,
  ): DagOutputsSpec_MapParameterSelectorsSpec {
    const message = {
      ...baseDagOutputsSpec_MapParameterSelectorsSpec,
    } as DagOutputsSpec_MapParameterSelectorsSpec;
    message.mappedParameters = Object.entries(object.mappedParameters ?? {}).reduce<{
      [key: string]: DagOutputsSpec_ParameterSelectorSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = DagOutputsSpec_ParameterSelectorSpec.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

const baseDagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry: object = { key: '' };

export const DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry = {
  encode(
    message: DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      DagOutputsSpec_ParameterSelectorSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseDagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry,
    } as DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = DagOutputsSpec_ParameterSelectorSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry {
    const message = {
      ...baseDagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry,
    } as DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? DagOutputsSpec_ParameterSelectorSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? DagOutputsSpec_ParameterSelectorSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry>, I>,
  >(object: I): DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry {
    const message = {
      ...baseDagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry,
    } as DagOutputsSpec_MapParameterSelectorsSpec_MappedParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? DagOutputsSpec_ParameterSelectorSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseDagOutputsSpec_DagOutputParameterSpec: object = {};

export const DagOutputsSpec_DagOutputParameterSpec = {
  encode(
    message: DagOutputsSpec_DagOutputParameterSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.valueFromParameter !== undefined) {
      DagOutputsSpec_ParameterSelectorSpec.encode(
        message.valueFromParameter,
        writer.uint32(10).fork(),
      ).ldelim();
    }
    if (message.valueFromOneof !== undefined) {
      DagOutputsSpec_ParameterSelectorsSpec.encode(
        message.valueFromOneof,
        writer.uint32(18).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagOutputsSpec_DagOutputParameterSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseDagOutputsSpec_DagOutputParameterSpec,
    } as DagOutputsSpec_DagOutputParameterSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.valueFromParameter = DagOutputsSpec_ParameterSelectorSpec.decode(
            reader,
            reader.uint32(),
          );
          break;
        case 2:
          message.valueFromOneof = DagOutputsSpec_ParameterSelectorsSpec.decode(
            reader,
            reader.uint32(),
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec_DagOutputParameterSpec {
    const message = {
      ...baseDagOutputsSpec_DagOutputParameterSpec,
    } as DagOutputsSpec_DagOutputParameterSpec;
    message.valueFromParameter =
      object.valueFromParameter !== undefined && object.valueFromParameter !== null
        ? DagOutputsSpec_ParameterSelectorSpec.fromJSON(object.valueFromParameter)
        : undefined;
    message.valueFromOneof =
      object.valueFromOneof !== undefined && object.valueFromOneof !== null
        ? DagOutputsSpec_ParameterSelectorsSpec.fromJSON(object.valueFromOneof)
        : undefined;
    return message;
  },

  toJSON(message: DagOutputsSpec_DagOutputParameterSpec): unknown {
    const obj: any = {};
    message.valueFromParameter !== undefined &&
      (obj.valueFromParameter = message.valueFromParameter
        ? DagOutputsSpec_ParameterSelectorSpec.toJSON(message.valueFromParameter)
        : undefined);
    message.valueFromOneof !== undefined &&
      (obj.valueFromOneof = message.valueFromOneof
        ? DagOutputsSpec_ParameterSelectorsSpec.toJSON(message.valueFromOneof)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagOutputsSpec_DagOutputParameterSpec>, I>>(
    object: I,
  ): DagOutputsSpec_DagOutputParameterSpec {
    const message = {
      ...baseDagOutputsSpec_DagOutputParameterSpec,
    } as DagOutputsSpec_DagOutputParameterSpec;
    message.valueFromParameter =
      object.valueFromParameter !== undefined && object.valueFromParameter !== null
        ? DagOutputsSpec_ParameterSelectorSpec.fromPartial(object.valueFromParameter)
        : undefined;
    message.valueFromOneof =
      object.valueFromOneof !== undefined && object.valueFromOneof !== null
        ? DagOutputsSpec_ParameterSelectorsSpec.fromPartial(object.valueFromOneof)
        : undefined;
    return message;
  },
};

const baseDagOutputsSpec_ParametersEntry: object = { key: '' };

export const DagOutputsSpec_ParametersEntry = {
  encode(
    message: DagOutputsSpec_ParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      DagOutputsSpec_DagOutputParameterSpec.encode(
        message.value,
        writer.uint32(18).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DagOutputsSpec_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseDagOutputsSpec_ParametersEntry } as DagOutputsSpec_ParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = DagOutputsSpec_DagOutputParameterSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DagOutputsSpec_ParametersEntry {
    const message = { ...baseDagOutputsSpec_ParametersEntry } as DagOutputsSpec_ParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? DagOutputsSpec_DagOutputParameterSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: DagOutputsSpec_ParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? DagOutputsSpec_DagOutputParameterSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DagOutputsSpec_ParametersEntry>, I>>(
    object: I,
  ): DagOutputsSpec_ParametersEntry {
    const message = { ...baseDagOutputsSpec_ParametersEntry } as DagOutputsSpec_ParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? DagOutputsSpec_DagOutputParameterSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseComponentInputsSpec: object = {};

export const ComponentInputsSpec = {
  encode(message: ComponentInputsSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.artifacts).forEach(([key, value]) => {
      ComponentInputsSpec_ArtifactsEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    Object.entries(message.parameters).forEach(([key, value]) => {
      ComponentInputsSpec_ParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentInputsSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseComponentInputsSpec } as ComponentInputsSpec;
    message.artifacts = {};
    message.parameters = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = ComponentInputsSpec_ArtifactsEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.artifacts[entry1.key] = entry1.value;
          }
          break;
        case 2:
          const entry2 = ComponentInputsSpec_ParametersEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.parameters[entry2.key] = entry2.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentInputsSpec {
    const message = { ...baseComponentInputsSpec } as ComponentInputsSpec;
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ComponentInputsSpec_ArtifactSpec;
    }>((acc, [key, value]) => {
      acc[key] = ComponentInputsSpec_ArtifactSpec.fromJSON(value);
      return acc;
    }, {});
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: ComponentInputsSpec_ParameterSpec;
    }>((acc, [key, value]) => {
      acc[key] = ComponentInputsSpec_ParameterSpec.fromJSON(value);
      return acc;
    }, {});
    return message;
  },

  toJSON(message: ComponentInputsSpec): unknown {
    const obj: any = {};
    obj.artifacts = {};
    if (message.artifacts) {
      Object.entries(message.artifacts).forEach(([k, v]) => {
        obj.artifacts[k] = ComponentInputsSpec_ArtifactSpec.toJSON(v);
      });
    }
    obj.parameters = {};
    if (message.parameters) {
      Object.entries(message.parameters).forEach(([k, v]) => {
        obj.parameters[k] = ComponentInputsSpec_ParameterSpec.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentInputsSpec>, I>>(
    object: I,
  ): ComponentInputsSpec {
    const message = { ...baseComponentInputsSpec } as ComponentInputsSpec;
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ComponentInputsSpec_ArtifactSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ComponentInputsSpec_ArtifactSpec.fromPartial(value);
      }
      return acc;
    }, {});
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: ComponentInputsSpec_ParameterSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ComponentInputsSpec_ParameterSpec.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

const baseComponentInputsSpec_ArtifactSpec: object = {};

export const ComponentInputsSpec_ArtifactSpec = {
  encode(
    message: ComponentInputsSpec_ArtifactSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.artifactType !== undefined) {
      ArtifactTypeSchema.encode(message.artifactType, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentInputsSpec_ArtifactSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseComponentInputsSpec_ArtifactSpec } as ComponentInputsSpec_ArtifactSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.artifactType = ArtifactTypeSchema.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentInputsSpec_ArtifactSpec {
    const message = { ...baseComponentInputsSpec_ArtifactSpec } as ComponentInputsSpec_ArtifactSpec;
    message.artifactType =
      object.artifactType !== undefined && object.artifactType !== null
        ? ArtifactTypeSchema.fromJSON(object.artifactType)
        : undefined;
    return message;
  },

  toJSON(message: ComponentInputsSpec_ArtifactSpec): unknown {
    const obj: any = {};
    message.artifactType !== undefined &&
      (obj.artifactType = message.artifactType
        ? ArtifactTypeSchema.toJSON(message.artifactType)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentInputsSpec_ArtifactSpec>, I>>(
    object: I,
  ): ComponentInputsSpec_ArtifactSpec {
    const message = { ...baseComponentInputsSpec_ArtifactSpec } as ComponentInputsSpec_ArtifactSpec;
    message.artifactType =
      object.artifactType !== undefined && object.artifactType !== null
        ? ArtifactTypeSchema.fromPartial(object.artifactType)
        : undefined;
    return message;
  },
};

const baseComponentInputsSpec_ParameterSpec: object = { type: 0, parameterType: 0 };

export const ComponentInputsSpec_ParameterSpec = {
  encode(
    message: ComponentInputsSpec_ParameterSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.parameterType !== 0) {
      writer.uint32(16).int32(message.parameterType);
    }
    if (message.defaultValue !== undefined) {
      Value1.encode(Value1.wrap(message.defaultValue), writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentInputsSpec_ParameterSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseComponentInputsSpec_ParameterSpec,
    } as ComponentInputsSpec_ParameterSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.int32() as any;
          break;
        case 2:
          message.parameterType = reader.int32() as any;
          break;
        case 3:
          message.defaultValue = Value1.unwrap(Value1.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentInputsSpec_ParameterSpec {
    const message = {
      ...baseComponentInputsSpec_ParameterSpec,
    } as ComponentInputsSpec_ParameterSpec;
    message.type =
      object.type !== undefined && object.type !== null
        ? primitiveType_PrimitiveTypeEnumFromJSON(object.type)
        : 0;
    message.parameterType =
      object.parameterType !== undefined && object.parameterType !== null
        ? parameterType_ParameterTypeEnumFromJSON(object.parameterType)
        : 0;
    message.defaultValue = object.defaultValue;
    return message;
  },

  toJSON(message: ComponentInputsSpec_ParameterSpec): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = primitiveType_PrimitiveTypeEnumToJSON(message.type));
    message.parameterType !== undefined &&
      (obj.parameterType = parameterType_ParameterTypeEnumToJSON(message.parameterType));
    message.defaultValue !== undefined && (obj.defaultValue = message.defaultValue);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentInputsSpec_ParameterSpec>, I>>(
    object: I,
  ): ComponentInputsSpec_ParameterSpec {
    const message = {
      ...baseComponentInputsSpec_ParameterSpec,
    } as ComponentInputsSpec_ParameterSpec;
    message.type = object.type ?? 0;
    message.parameterType = object.parameterType ?? 0;
    message.defaultValue = object.defaultValue ?? undefined;
    return message;
  },
};

const baseComponentInputsSpec_ArtifactsEntry: object = { key: '' };

export const ComponentInputsSpec_ArtifactsEntry = {
  encode(
    message: ComponentInputsSpec_ArtifactsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ComponentInputsSpec_ArtifactSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentInputsSpec_ArtifactsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseComponentInputsSpec_ArtifactsEntry,
    } as ComponentInputsSpec_ArtifactsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ComponentInputsSpec_ArtifactSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentInputsSpec_ArtifactsEntry {
    const message = {
      ...baseComponentInputsSpec_ArtifactsEntry,
    } as ComponentInputsSpec_ArtifactsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentInputsSpec_ArtifactSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ComponentInputsSpec_ArtifactsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? ComponentInputsSpec_ArtifactSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentInputsSpec_ArtifactsEntry>, I>>(
    object: I,
  ): ComponentInputsSpec_ArtifactsEntry {
    const message = {
      ...baseComponentInputsSpec_ArtifactsEntry,
    } as ComponentInputsSpec_ArtifactsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentInputsSpec_ArtifactSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseComponentInputsSpec_ParametersEntry: object = { key: '' };

export const ComponentInputsSpec_ParametersEntry = {
  encode(
    message: ComponentInputsSpec_ParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ComponentInputsSpec_ParameterSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentInputsSpec_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseComponentInputsSpec_ParametersEntry,
    } as ComponentInputsSpec_ParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ComponentInputsSpec_ParameterSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentInputsSpec_ParametersEntry {
    const message = {
      ...baseComponentInputsSpec_ParametersEntry,
    } as ComponentInputsSpec_ParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentInputsSpec_ParameterSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ComponentInputsSpec_ParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? ComponentInputsSpec_ParameterSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentInputsSpec_ParametersEntry>, I>>(
    object: I,
  ): ComponentInputsSpec_ParametersEntry {
    const message = {
      ...baseComponentInputsSpec_ParametersEntry,
    } as ComponentInputsSpec_ParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentInputsSpec_ParameterSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseComponentOutputsSpec: object = {};

export const ComponentOutputsSpec = {
  encode(message: ComponentOutputsSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.artifacts).forEach(([key, value]) => {
      ComponentOutputsSpec_ArtifactsEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    Object.entries(message.parameters).forEach(([key, value]) => {
      ComponentOutputsSpec_ParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentOutputsSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseComponentOutputsSpec } as ComponentOutputsSpec;
    message.artifacts = {};
    message.parameters = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = ComponentOutputsSpec_ArtifactsEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.artifacts[entry1.key] = entry1.value;
          }
          break;
        case 2:
          const entry2 = ComponentOutputsSpec_ParametersEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.parameters[entry2.key] = entry2.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentOutputsSpec {
    const message = { ...baseComponentOutputsSpec } as ComponentOutputsSpec;
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ComponentOutputsSpec_ArtifactSpec;
    }>((acc, [key, value]) => {
      acc[key] = ComponentOutputsSpec_ArtifactSpec.fromJSON(value);
      return acc;
    }, {});
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: ComponentOutputsSpec_ParameterSpec;
    }>((acc, [key, value]) => {
      acc[key] = ComponentOutputsSpec_ParameterSpec.fromJSON(value);
      return acc;
    }, {});
    return message;
  },

  toJSON(message: ComponentOutputsSpec): unknown {
    const obj: any = {};
    obj.artifacts = {};
    if (message.artifacts) {
      Object.entries(message.artifacts).forEach(([k, v]) => {
        obj.artifacts[k] = ComponentOutputsSpec_ArtifactSpec.toJSON(v);
      });
    }
    obj.parameters = {};
    if (message.parameters) {
      Object.entries(message.parameters).forEach(([k, v]) => {
        obj.parameters[k] = ComponentOutputsSpec_ParameterSpec.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentOutputsSpec>, I>>(
    object: I,
  ): ComponentOutputsSpec {
    const message = { ...baseComponentOutputsSpec } as ComponentOutputsSpec;
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ComponentOutputsSpec_ArtifactSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ComponentOutputsSpec_ArtifactSpec.fromPartial(value);
      }
      return acc;
    }, {});
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: ComponentOutputsSpec_ParameterSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ComponentOutputsSpec_ParameterSpec.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

const baseComponentOutputsSpec_ArtifactSpec: object = {};

export const ComponentOutputsSpec_ArtifactSpec = {
  encode(
    message: ComponentOutputsSpec_ArtifactSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.artifactType !== undefined) {
      ArtifactTypeSchema.encode(message.artifactType, writer.uint32(10).fork()).ldelim();
    }
    Object.entries(message.properties).forEach(([key, value]) => {
      ComponentOutputsSpec_ArtifactSpec_PropertiesEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    Object.entries(message.customProperties).forEach(([key, value]) => {
      ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry.encode(
        { key: key as any, value },
        writer.uint32(26).fork(),
      ).ldelim();
    });
    if (message.metadata !== undefined) {
      Struct.encode(Struct.wrap(message.metadata), writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentOutputsSpec_ArtifactSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseComponentOutputsSpec_ArtifactSpec,
    } as ComponentOutputsSpec_ArtifactSpec;
    message.properties = {};
    message.customProperties = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.artifactType = ArtifactTypeSchema.decode(reader, reader.uint32());
          break;
        case 2:
          const entry2 = ComponentOutputsSpec_ArtifactSpec_PropertiesEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry2.value !== undefined) {
            message.properties[entry2.key] = entry2.value;
          }
          break;
        case 3:
          const entry3 = ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry3.value !== undefined) {
            message.customProperties[entry3.key] = entry3.value;
          }
          break;
        case 4:
          message.metadata = Struct.unwrap(Struct.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentOutputsSpec_ArtifactSpec {
    const message = {
      ...baseComponentOutputsSpec_ArtifactSpec,
    } as ComponentOutputsSpec_ArtifactSpec;
    message.artifactType =
      object.artifactType !== undefined && object.artifactType !== null
        ? ArtifactTypeSchema.fromJSON(object.artifactType)
        : undefined;
    message.properties = Object.entries(object.properties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      acc[key] = ValueOrRuntimeParameter.fromJSON(value);
      return acc;
    }, {});
    message.customProperties = Object.entries(object.customProperties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      acc[key] = ValueOrRuntimeParameter.fromJSON(value);
      return acc;
    }, {});
    message.metadata = typeof object.metadata === 'object' ? object.metadata : undefined;
    return message;
  },

  toJSON(message: ComponentOutputsSpec_ArtifactSpec): unknown {
    const obj: any = {};
    message.artifactType !== undefined &&
      (obj.artifactType = message.artifactType
        ? ArtifactTypeSchema.toJSON(message.artifactType)
        : undefined);
    obj.properties = {};
    if (message.properties) {
      Object.entries(message.properties).forEach(([k, v]) => {
        obj.properties[k] = ValueOrRuntimeParameter.toJSON(v);
      });
    }
    obj.customProperties = {};
    if (message.customProperties) {
      Object.entries(message.customProperties).forEach(([k, v]) => {
        obj.customProperties[k] = ValueOrRuntimeParameter.toJSON(v);
      });
    }
    message.metadata !== undefined && (obj.metadata = message.metadata);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentOutputsSpec_ArtifactSpec>, I>>(
    object: I,
  ): ComponentOutputsSpec_ArtifactSpec {
    const message = {
      ...baseComponentOutputsSpec_ArtifactSpec,
    } as ComponentOutputsSpec_ArtifactSpec;
    message.artifactType =
      object.artifactType !== undefined && object.artifactType !== null
        ? ArtifactTypeSchema.fromPartial(object.artifactType)
        : undefined;
    message.properties = Object.entries(object.properties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ValueOrRuntimeParameter.fromPartial(value);
      }
      return acc;
    }, {});
    message.customProperties = Object.entries(object.customProperties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ValueOrRuntimeParameter.fromPartial(value);
      }
      return acc;
    }, {});
    message.metadata = object.metadata ?? undefined;
    return message;
  },
};

const baseComponentOutputsSpec_ArtifactSpec_PropertiesEntry: object = { key: '' };

export const ComponentOutputsSpec_ArtifactSpec_PropertiesEntry = {
  encode(
    message: ComponentOutputsSpec_ArtifactSpec_PropertiesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ValueOrRuntimeParameter.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): ComponentOutputsSpec_ArtifactSpec_PropertiesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseComponentOutputsSpec_ArtifactSpec_PropertiesEntry,
    } as ComponentOutputsSpec_ArtifactSpec_PropertiesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ValueOrRuntimeParameter.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentOutputsSpec_ArtifactSpec_PropertiesEntry {
    const message = {
      ...baseComponentOutputsSpec_ArtifactSpec_PropertiesEntry,
    } as ComponentOutputsSpec_ArtifactSpec_PropertiesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ComponentOutputsSpec_ArtifactSpec_PropertiesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ValueOrRuntimeParameter.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentOutputsSpec_ArtifactSpec_PropertiesEntry>, I>>(
    object: I,
  ): ComponentOutputsSpec_ArtifactSpec_PropertiesEntry {
    const message = {
      ...baseComponentOutputsSpec_ArtifactSpec_PropertiesEntry,
    } as ComponentOutputsSpec_ArtifactSpec_PropertiesEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry: object = { key: '' };

export const ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry = {
  encode(
    message: ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ValueOrRuntimeParameter.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry,
    } as ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ValueOrRuntimeParameter.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry {
    const message = {
      ...baseComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry,
    } as ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ValueOrRuntimeParameter.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry>, I>,
  >(object: I): ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry {
    const message = {
      ...baseComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry,
    } as ComponentOutputsSpec_ArtifactSpec_CustomPropertiesEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseComponentOutputsSpec_ParameterSpec: object = { type: 0, parameterType: 0 };

export const ComponentOutputsSpec_ParameterSpec = {
  encode(
    message: ComponentOutputsSpec_ParameterSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.parameterType !== 0) {
      writer.uint32(16).int32(message.parameterType);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentOutputsSpec_ParameterSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseComponentOutputsSpec_ParameterSpec,
    } as ComponentOutputsSpec_ParameterSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.int32() as any;
          break;
        case 2:
          message.parameterType = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentOutputsSpec_ParameterSpec {
    const message = {
      ...baseComponentOutputsSpec_ParameterSpec,
    } as ComponentOutputsSpec_ParameterSpec;
    message.type =
      object.type !== undefined && object.type !== null
        ? primitiveType_PrimitiveTypeEnumFromJSON(object.type)
        : 0;
    message.parameterType =
      object.parameterType !== undefined && object.parameterType !== null
        ? parameterType_ParameterTypeEnumFromJSON(object.parameterType)
        : 0;
    return message;
  },

  toJSON(message: ComponentOutputsSpec_ParameterSpec): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = primitiveType_PrimitiveTypeEnumToJSON(message.type));
    message.parameterType !== undefined &&
      (obj.parameterType = parameterType_ParameterTypeEnumToJSON(message.parameterType));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentOutputsSpec_ParameterSpec>, I>>(
    object: I,
  ): ComponentOutputsSpec_ParameterSpec {
    const message = {
      ...baseComponentOutputsSpec_ParameterSpec,
    } as ComponentOutputsSpec_ParameterSpec;
    message.type = object.type ?? 0;
    message.parameterType = object.parameterType ?? 0;
    return message;
  },
};

const baseComponentOutputsSpec_ArtifactsEntry: object = { key: '' };

export const ComponentOutputsSpec_ArtifactsEntry = {
  encode(
    message: ComponentOutputsSpec_ArtifactsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ComponentOutputsSpec_ArtifactSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentOutputsSpec_ArtifactsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseComponentOutputsSpec_ArtifactsEntry,
    } as ComponentOutputsSpec_ArtifactsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ComponentOutputsSpec_ArtifactSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentOutputsSpec_ArtifactsEntry {
    const message = {
      ...baseComponentOutputsSpec_ArtifactsEntry,
    } as ComponentOutputsSpec_ArtifactsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentOutputsSpec_ArtifactSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ComponentOutputsSpec_ArtifactsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? ComponentOutputsSpec_ArtifactSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentOutputsSpec_ArtifactsEntry>, I>>(
    object: I,
  ): ComponentOutputsSpec_ArtifactsEntry {
    const message = {
      ...baseComponentOutputsSpec_ArtifactsEntry,
    } as ComponentOutputsSpec_ArtifactsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentOutputsSpec_ArtifactSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseComponentOutputsSpec_ParametersEntry: object = { key: '' };

export const ComponentOutputsSpec_ParametersEntry = {
  encode(
    message: ComponentOutputsSpec_ParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ComponentOutputsSpec_ParameterSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentOutputsSpec_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseComponentOutputsSpec_ParametersEntry,
    } as ComponentOutputsSpec_ParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ComponentOutputsSpec_ParameterSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentOutputsSpec_ParametersEntry {
    const message = {
      ...baseComponentOutputsSpec_ParametersEntry,
    } as ComponentOutputsSpec_ParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentOutputsSpec_ParameterSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ComponentOutputsSpec_ParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? ComponentOutputsSpec_ParameterSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentOutputsSpec_ParametersEntry>, I>>(
    object: I,
  ): ComponentOutputsSpec_ParametersEntry {
    const message = {
      ...baseComponentOutputsSpec_ParametersEntry,
    } as ComponentOutputsSpec_ParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ComponentOutputsSpec_ParameterSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseTaskInputsSpec: object = {};

export const TaskInputsSpec = {
  encode(message: TaskInputsSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.parameters).forEach(([key, value]) => {
      TaskInputsSpec_ParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    Object.entries(message.artifacts).forEach(([key, value]) => {
      TaskInputsSpec_ArtifactsEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskInputsSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTaskInputsSpec } as TaskInputsSpec;
    message.parameters = {};
    message.artifacts = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = TaskInputsSpec_ParametersEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.parameters[entry1.key] = entry1.value;
          }
          break;
        case 2:
          const entry2 = TaskInputsSpec_ArtifactsEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.artifacts[entry2.key] = entry2.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskInputsSpec {
    const message = { ...baseTaskInputsSpec } as TaskInputsSpec;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: TaskInputsSpec_InputParameterSpec;
    }>((acc, [key, value]) => {
      acc[key] = TaskInputsSpec_InputParameterSpec.fromJSON(value);
      return acc;
    }, {});
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: TaskInputsSpec_InputArtifactSpec;
    }>((acc, [key, value]) => {
      acc[key] = TaskInputsSpec_InputArtifactSpec.fromJSON(value);
      return acc;
    }, {});
    return message;
  },

  toJSON(message: TaskInputsSpec): unknown {
    const obj: any = {};
    obj.parameters = {};
    if (message.parameters) {
      Object.entries(message.parameters).forEach(([k, v]) => {
        obj.parameters[k] = TaskInputsSpec_InputParameterSpec.toJSON(v);
      });
    }
    obj.artifacts = {};
    if (message.artifacts) {
      Object.entries(message.artifacts).forEach(([k, v]) => {
        obj.artifacts[k] = TaskInputsSpec_InputArtifactSpec.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskInputsSpec>, I>>(object: I): TaskInputsSpec {
    const message = { ...baseTaskInputsSpec } as TaskInputsSpec;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: TaskInputsSpec_InputParameterSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = TaskInputsSpec_InputParameterSpec.fromPartial(value);
      }
      return acc;
    }, {});
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: TaskInputsSpec_InputArtifactSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = TaskInputsSpec_InputArtifactSpec.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

const baseTaskInputsSpec_InputArtifactSpec: object = {};

export const TaskInputsSpec_InputArtifactSpec = {
  encode(
    message: TaskInputsSpec_InputArtifactSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.taskOutputArtifact !== undefined) {
      TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec.encode(
        message.taskOutputArtifact,
        writer.uint32(26).fork(),
      ).ldelim();
    }
    if (message.componentInputArtifact !== undefined) {
      writer.uint32(34).string(message.componentInputArtifact);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskInputsSpec_InputArtifactSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTaskInputsSpec_InputArtifactSpec } as TaskInputsSpec_InputArtifactSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 3:
          message.taskOutputArtifact =
            TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec.decode(reader, reader.uint32());
          break;
        case 4:
          message.componentInputArtifact = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskInputsSpec_InputArtifactSpec {
    const message = { ...baseTaskInputsSpec_InputArtifactSpec } as TaskInputsSpec_InputArtifactSpec;
    message.taskOutputArtifact =
      object.taskOutputArtifact !== undefined && object.taskOutputArtifact !== null
        ? TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec.fromJSON(
            object.taskOutputArtifact,
          )
        : undefined;
    message.componentInputArtifact =
      object.componentInputArtifact !== undefined && object.componentInputArtifact !== null
        ? String(object.componentInputArtifact)
        : undefined;
    return message;
  },

  toJSON(message: TaskInputsSpec_InputArtifactSpec): unknown {
    const obj: any = {};
    message.taskOutputArtifact !== undefined &&
      (obj.taskOutputArtifact = message.taskOutputArtifact
        ? TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec.toJSON(message.taskOutputArtifact)
        : undefined);
    message.componentInputArtifact !== undefined &&
      (obj.componentInputArtifact = message.componentInputArtifact);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskInputsSpec_InputArtifactSpec>, I>>(
    object: I,
  ): TaskInputsSpec_InputArtifactSpec {
    const message = { ...baseTaskInputsSpec_InputArtifactSpec } as TaskInputsSpec_InputArtifactSpec;
    message.taskOutputArtifact =
      object.taskOutputArtifact !== undefined && object.taskOutputArtifact !== null
        ? TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec.fromPartial(
            object.taskOutputArtifact,
          )
        : undefined;
    message.componentInputArtifact = object.componentInputArtifact ?? undefined;
    return message;
  },
};

const baseTaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec: object = {
  producerTask: '',
  outputArtifactKey: '',
};

export const TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec = {
  encode(
    message: TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.producerTask !== '') {
      writer.uint32(10).string(message.producerTask);
    }
    if (message.outputArtifactKey !== '') {
      writer.uint32(18).string(message.outputArtifactKey);
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec,
    } as TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.producerTask = reader.string();
          break;
        case 2:
          message.outputArtifactKey = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec {
    const message = {
      ...baseTaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec,
    } as TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec;
    message.producerTask =
      object.producerTask !== undefined && object.producerTask !== null
        ? String(object.producerTask)
        : '';
    message.outputArtifactKey =
      object.outputArtifactKey !== undefined && object.outputArtifactKey !== null
        ? String(object.outputArtifactKey)
        : '';
    return message;
  },

  toJSON(message: TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec): unknown {
    const obj: any = {};
    message.producerTask !== undefined && (obj.producerTask = message.producerTask);
    message.outputArtifactKey !== undefined && (obj.outputArtifactKey = message.outputArtifactKey);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec>, I>,
  >(object: I): TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec {
    const message = {
      ...baseTaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec,
    } as TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec;
    message.producerTask = object.producerTask ?? '';
    message.outputArtifactKey = object.outputArtifactKey ?? '';
    return message;
  },
};

const baseTaskInputsSpec_InputParameterSpec: object = { parameterExpressionSelector: '' };

export const TaskInputsSpec_InputParameterSpec = {
  encode(
    message: TaskInputsSpec_InputParameterSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.taskOutputParameter !== undefined) {
      TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec.encode(
        message.taskOutputParameter,
        writer.uint32(10).fork(),
      ).ldelim();
    }
    if (message.runtimeValue !== undefined) {
      ValueOrRuntimeParameter.encode(message.runtimeValue, writer.uint32(18).fork()).ldelim();
    }
    if (message.componentInputParameter !== undefined) {
      writer.uint32(26).string(message.componentInputParameter);
    }
    if (message.taskFinalStatus !== undefined) {
      TaskInputsSpec_InputParameterSpec_TaskFinalStatus.encode(
        message.taskFinalStatus,
        writer.uint32(42).fork(),
      ).ldelim();
    }
    if (message.parameterExpressionSelector !== '') {
      writer.uint32(34).string(message.parameterExpressionSelector);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskInputsSpec_InputParameterSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTaskInputsSpec_InputParameterSpec,
    } as TaskInputsSpec_InputParameterSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.taskOutputParameter =
            TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec.decode(
              reader,
              reader.uint32(),
            );
          break;
        case 2:
          message.runtimeValue = ValueOrRuntimeParameter.decode(reader, reader.uint32());
          break;
        case 3:
          message.componentInputParameter = reader.string();
          break;
        case 5:
          message.taskFinalStatus = TaskInputsSpec_InputParameterSpec_TaskFinalStatus.decode(
            reader,
            reader.uint32(),
          );
          break;
        case 4:
          message.parameterExpressionSelector = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskInputsSpec_InputParameterSpec {
    const message = {
      ...baseTaskInputsSpec_InputParameterSpec,
    } as TaskInputsSpec_InputParameterSpec;
    message.taskOutputParameter =
      object.taskOutputParameter !== undefined && object.taskOutputParameter !== null
        ? TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec.fromJSON(
            object.taskOutputParameter,
          )
        : undefined;
    message.runtimeValue =
      object.runtimeValue !== undefined && object.runtimeValue !== null
        ? ValueOrRuntimeParameter.fromJSON(object.runtimeValue)
        : undefined;
    message.componentInputParameter =
      object.componentInputParameter !== undefined && object.componentInputParameter !== null
        ? String(object.componentInputParameter)
        : undefined;
    message.taskFinalStatus =
      object.taskFinalStatus !== undefined && object.taskFinalStatus !== null
        ? TaskInputsSpec_InputParameterSpec_TaskFinalStatus.fromJSON(object.taskFinalStatus)
        : undefined;
    message.parameterExpressionSelector =
      object.parameterExpressionSelector !== undefined &&
      object.parameterExpressionSelector !== null
        ? String(object.parameterExpressionSelector)
        : '';
    return message;
  },

  toJSON(message: TaskInputsSpec_InputParameterSpec): unknown {
    const obj: any = {};
    message.taskOutputParameter !== undefined &&
      (obj.taskOutputParameter = message.taskOutputParameter
        ? TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec.toJSON(
            message.taskOutputParameter,
          )
        : undefined);
    message.runtimeValue !== undefined &&
      (obj.runtimeValue = message.runtimeValue
        ? ValueOrRuntimeParameter.toJSON(message.runtimeValue)
        : undefined);
    message.componentInputParameter !== undefined &&
      (obj.componentInputParameter = message.componentInputParameter);
    message.taskFinalStatus !== undefined &&
      (obj.taskFinalStatus = message.taskFinalStatus
        ? TaskInputsSpec_InputParameterSpec_TaskFinalStatus.toJSON(message.taskFinalStatus)
        : undefined);
    message.parameterExpressionSelector !== undefined &&
      (obj.parameterExpressionSelector = message.parameterExpressionSelector);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskInputsSpec_InputParameterSpec>, I>>(
    object: I,
  ): TaskInputsSpec_InputParameterSpec {
    const message = {
      ...baseTaskInputsSpec_InputParameterSpec,
    } as TaskInputsSpec_InputParameterSpec;
    message.taskOutputParameter =
      object.taskOutputParameter !== undefined && object.taskOutputParameter !== null
        ? TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec.fromPartial(
            object.taskOutputParameter,
          )
        : undefined;
    message.runtimeValue =
      object.runtimeValue !== undefined && object.runtimeValue !== null
        ? ValueOrRuntimeParameter.fromPartial(object.runtimeValue)
        : undefined;
    message.componentInputParameter = object.componentInputParameter ?? undefined;
    message.taskFinalStatus =
      object.taskFinalStatus !== undefined && object.taskFinalStatus !== null
        ? TaskInputsSpec_InputParameterSpec_TaskFinalStatus.fromPartial(object.taskFinalStatus)
        : undefined;
    message.parameterExpressionSelector = object.parameterExpressionSelector ?? '';
    return message;
  },
};

const baseTaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec: object = {
  producerTask: '',
  outputParameterKey: '',
};

export const TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec = {
  encode(
    message: TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.producerTask !== '') {
      writer.uint32(10).string(message.producerTask);
    }
    if (message.outputParameterKey !== '') {
      writer.uint32(18).string(message.outputParameterKey);
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec,
    } as TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.producerTask = reader.string();
          break;
        case 2:
          message.outputParameterKey = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec {
    const message = {
      ...baseTaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec,
    } as TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec;
    message.producerTask =
      object.producerTask !== undefined && object.producerTask !== null
        ? String(object.producerTask)
        : '';
    message.outputParameterKey =
      object.outputParameterKey !== undefined && object.outputParameterKey !== null
        ? String(object.outputParameterKey)
        : '';
    return message;
  },

  toJSON(message: TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec): unknown {
    const obj: any = {};
    message.producerTask !== undefined && (obj.producerTask = message.producerTask);
    message.outputParameterKey !== undefined &&
      (obj.outputParameterKey = message.outputParameterKey);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec>, I>,
  >(object: I): TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec {
    const message = {
      ...baseTaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec,
    } as TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec;
    message.producerTask = object.producerTask ?? '';
    message.outputParameterKey = object.outputParameterKey ?? '';
    return message;
  },
};

const baseTaskInputsSpec_InputParameterSpec_TaskFinalStatus: object = { producerTask: '' };

export const TaskInputsSpec_InputParameterSpec_TaskFinalStatus = {
  encode(
    message: TaskInputsSpec_InputParameterSpec_TaskFinalStatus,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.producerTask !== '') {
      writer.uint32(10).string(message.producerTask);
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): TaskInputsSpec_InputParameterSpec_TaskFinalStatus {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTaskInputsSpec_InputParameterSpec_TaskFinalStatus,
    } as TaskInputsSpec_InputParameterSpec_TaskFinalStatus;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.producerTask = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskInputsSpec_InputParameterSpec_TaskFinalStatus {
    const message = {
      ...baseTaskInputsSpec_InputParameterSpec_TaskFinalStatus,
    } as TaskInputsSpec_InputParameterSpec_TaskFinalStatus;
    message.producerTask =
      object.producerTask !== undefined && object.producerTask !== null
        ? String(object.producerTask)
        : '';
    return message;
  },

  toJSON(message: TaskInputsSpec_InputParameterSpec_TaskFinalStatus): unknown {
    const obj: any = {};
    message.producerTask !== undefined && (obj.producerTask = message.producerTask);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskInputsSpec_InputParameterSpec_TaskFinalStatus>, I>>(
    object: I,
  ): TaskInputsSpec_InputParameterSpec_TaskFinalStatus {
    const message = {
      ...baseTaskInputsSpec_InputParameterSpec_TaskFinalStatus,
    } as TaskInputsSpec_InputParameterSpec_TaskFinalStatus;
    message.producerTask = object.producerTask ?? '';
    return message;
  },
};

const baseTaskInputsSpec_ParametersEntry: object = { key: '' };

export const TaskInputsSpec_ParametersEntry = {
  encode(
    message: TaskInputsSpec_ParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      TaskInputsSpec_InputParameterSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskInputsSpec_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTaskInputsSpec_ParametersEntry } as TaskInputsSpec_ParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = TaskInputsSpec_InputParameterSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskInputsSpec_ParametersEntry {
    const message = { ...baseTaskInputsSpec_ParametersEntry } as TaskInputsSpec_ParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? TaskInputsSpec_InputParameterSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: TaskInputsSpec_ParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? TaskInputsSpec_InputParameterSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskInputsSpec_ParametersEntry>, I>>(
    object: I,
  ): TaskInputsSpec_ParametersEntry {
    const message = { ...baseTaskInputsSpec_ParametersEntry } as TaskInputsSpec_ParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? TaskInputsSpec_InputParameterSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseTaskInputsSpec_ArtifactsEntry: object = { key: '' };

export const TaskInputsSpec_ArtifactsEntry = {
  encode(
    message: TaskInputsSpec_ArtifactsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      TaskInputsSpec_InputArtifactSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskInputsSpec_ArtifactsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTaskInputsSpec_ArtifactsEntry } as TaskInputsSpec_ArtifactsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = TaskInputsSpec_InputArtifactSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskInputsSpec_ArtifactsEntry {
    const message = { ...baseTaskInputsSpec_ArtifactsEntry } as TaskInputsSpec_ArtifactsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? TaskInputsSpec_InputArtifactSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: TaskInputsSpec_ArtifactsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? TaskInputsSpec_InputArtifactSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskInputsSpec_ArtifactsEntry>, I>>(
    object: I,
  ): TaskInputsSpec_ArtifactsEntry {
    const message = { ...baseTaskInputsSpec_ArtifactsEntry } as TaskInputsSpec_ArtifactsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? TaskInputsSpec_InputArtifactSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseTaskOutputsSpec: object = {};

export const TaskOutputsSpec = {
  encode(message: TaskOutputsSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.parameters).forEach(([key, value]) => {
      TaskOutputsSpec_ParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    Object.entries(message.artifacts).forEach(([key, value]) => {
      TaskOutputsSpec_ArtifactsEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskOutputsSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTaskOutputsSpec } as TaskOutputsSpec;
    message.parameters = {};
    message.artifacts = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = TaskOutputsSpec_ParametersEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.parameters[entry1.key] = entry1.value;
          }
          break;
        case 2:
          const entry2 = TaskOutputsSpec_ArtifactsEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.artifacts[entry2.key] = entry2.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskOutputsSpec {
    const message = { ...baseTaskOutputsSpec } as TaskOutputsSpec;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: TaskOutputsSpec_OutputParameterSpec;
    }>((acc, [key, value]) => {
      acc[key] = TaskOutputsSpec_OutputParameterSpec.fromJSON(value);
      return acc;
    }, {});
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: TaskOutputsSpec_OutputArtifactSpec;
    }>((acc, [key, value]) => {
      acc[key] = TaskOutputsSpec_OutputArtifactSpec.fromJSON(value);
      return acc;
    }, {});
    return message;
  },

  toJSON(message: TaskOutputsSpec): unknown {
    const obj: any = {};
    obj.parameters = {};
    if (message.parameters) {
      Object.entries(message.parameters).forEach(([k, v]) => {
        obj.parameters[k] = TaskOutputsSpec_OutputParameterSpec.toJSON(v);
      });
    }
    obj.artifacts = {};
    if (message.artifacts) {
      Object.entries(message.artifacts).forEach(([k, v]) => {
        obj.artifacts[k] = TaskOutputsSpec_OutputArtifactSpec.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskOutputsSpec>, I>>(object: I): TaskOutputsSpec {
    const message = { ...baseTaskOutputsSpec } as TaskOutputsSpec;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: TaskOutputsSpec_OutputParameterSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = TaskOutputsSpec_OutputParameterSpec.fromPartial(value);
      }
      return acc;
    }, {});
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: TaskOutputsSpec_OutputArtifactSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = TaskOutputsSpec_OutputArtifactSpec.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

const baseTaskOutputsSpec_OutputArtifactSpec: object = {};

export const TaskOutputsSpec_OutputArtifactSpec = {
  encode(
    message: TaskOutputsSpec_OutputArtifactSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.artifactType !== undefined) {
      ArtifactTypeSchema.encode(message.artifactType, writer.uint32(10).fork()).ldelim();
    }
    Object.entries(message.properties).forEach(([key, value]) => {
      TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    Object.entries(message.customProperties).forEach(([key, value]) => {
      TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry.encode(
        { key: key as any, value },
        writer.uint32(26).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskOutputsSpec_OutputArtifactSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTaskOutputsSpec_OutputArtifactSpec,
    } as TaskOutputsSpec_OutputArtifactSpec;
    message.properties = {};
    message.customProperties = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.artifactType = ArtifactTypeSchema.decode(reader, reader.uint32());
          break;
        case 2:
          const entry2 = TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry2.value !== undefined) {
            message.properties[entry2.key] = entry2.value;
          }
          break;
        case 3:
          const entry3 = TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry3.value !== undefined) {
            message.customProperties[entry3.key] = entry3.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskOutputsSpec_OutputArtifactSpec {
    const message = {
      ...baseTaskOutputsSpec_OutputArtifactSpec,
    } as TaskOutputsSpec_OutputArtifactSpec;
    message.artifactType =
      object.artifactType !== undefined && object.artifactType !== null
        ? ArtifactTypeSchema.fromJSON(object.artifactType)
        : undefined;
    message.properties = Object.entries(object.properties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      acc[key] = ValueOrRuntimeParameter.fromJSON(value);
      return acc;
    }, {});
    message.customProperties = Object.entries(object.customProperties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      acc[key] = ValueOrRuntimeParameter.fromJSON(value);
      return acc;
    }, {});
    return message;
  },

  toJSON(message: TaskOutputsSpec_OutputArtifactSpec): unknown {
    const obj: any = {};
    message.artifactType !== undefined &&
      (obj.artifactType = message.artifactType
        ? ArtifactTypeSchema.toJSON(message.artifactType)
        : undefined);
    obj.properties = {};
    if (message.properties) {
      Object.entries(message.properties).forEach(([k, v]) => {
        obj.properties[k] = ValueOrRuntimeParameter.toJSON(v);
      });
    }
    obj.customProperties = {};
    if (message.customProperties) {
      Object.entries(message.customProperties).forEach(([k, v]) => {
        obj.customProperties[k] = ValueOrRuntimeParameter.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskOutputsSpec_OutputArtifactSpec>, I>>(
    object: I,
  ): TaskOutputsSpec_OutputArtifactSpec {
    const message = {
      ...baseTaskOutputsSpec_OutputArtifactSpec,
    } as TaskOutputsSpec_OutputArtifactSpec;
    message.artifactType =
      object.artifactType !== undefined && object.artifactType !== null
        ? ArtifactTypeSchema.fromPartial(object.artifactType)
        : undefined;
    message.properties = Object.entries(object.properties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ValueOrRuntimeParameter.fromPartial(value);
      }
      return acc;
    }, {});
    message.customProperties = Object.entries(object.customProperties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ValueOrRuntimeParameter.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

const baseTaskOutputsSpec_OutputArtifactSpec_PropertiesEntry: object = { key: '' };

export const TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry = {
  encode(
    message: TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ValueOrRuntimeParameter.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTaskOutputsSpec_OutputArtifactSpec_PropertiesEntry,
    } as TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ValueOrRuntimeParameter.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry {
    const message = {
      ...baseTaskOutputsSpec_OutputArtifactSpec_PropertiesEntry,
    } as TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ValueOrRuntimeParameter.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry>, I>>(
    object: I,
  ): TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry {
    const message = {
      ...baseTaskOutputsSpec_OutputArtifactSpec_PropertiesEntry,
    } as TaskOutputsSpec_OutputArtifactSpec_PropertiesEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseTaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry: object = { key: '' };

export const TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry = {
  encode(
    message: TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ValueOrRuntimeParameter.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry,
    } as TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ValueOrRuntimeParameter.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry {
    const message = {
      ...baseTaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry,
    } as TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ValueOrRuntimeParameter.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry>, I>,
  >(object: I): TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry {
    const message = {
      ...baseTaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry,
    } as TaskOutputsSpec_OutputArtifactSpec_CustomPropertiesEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseTaskOutputsSpec_OutputParameterSpec: object = { type: 0 };

export const TaskOutputsSpec_OutputParameterSpec = {
  encode(
    message: TaskOutputsSpec_OutputParameterSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskOutputsSpec_OutputParameterSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseTaskOutputsSpec_OutputParameterSpec,
    } as TaskOutputsSpec_OutputParameterSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskOutputsSpec_OutputParameterSpec {
    const message = {
      ...baseTaskOutputsSpec_OutputParameterSpec,
    } as TaskOutputsSpec_OutputParameterSpec;
    message.type =
      object.type !== undefined && object.type !== null
        ? primitiveType_PrimitiveTypeEnumFromJSON(object.type)
        : 0;
    return message;
  },

  toJSON(message: TaskOutputsSpec_OutputParameterSpec): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = primitiveType_PrimitiveTypeEnumToJSON(message.type));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskOutputsSpec_OutputParameterSpec>, I>>(
    object: I,
  ): TaskOutputsSpec_OutputParameterSpec {
    const message = {
      ...baseTaskOutputsSpec_OutputParameterSpec,
    } as TaskOutputsSpec_OutputParameterSpec;
    message.type = object.type ?? 0;
    return message;
  },
};

const baseTaskOutputsSpec_ParametersEntry: object = { key: '' };

export const TaskOutputsSpec_ParametersEntry = {
  encode(
    message: TaskOutputsSpec_ParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      TaskOutputsSpec_OutputParameterSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskOutputsSpec_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTaskOutputsSpec_ParametersEntry } as TaskOutputsSpec_ParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = TaskOutputsSpec_OutputParameterSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskOutputsSpec_ParametersEntry {
    const message = { ...baseTaskOutputsSpec_ParametersEntry } as TaskOutputsSpec_ParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? TaskOutputsSpec_OutputParameterSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: TaskOutputsSpec_ParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? TaskOutputsSpec_OutputParameterSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskOutputsSpec_ParametersEntry>, I>>(
    object: I,
  ): TaskOutputsSpec_ParametersEntry {
    const message = { ...baseTaskOutputsSpec_ParametersEntry } as TaskOutputsSpec_ParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? TaskOutputsSpec_OutputParameterSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseTaskOutputsSpec_ArtifactsEntry: object = { key: '' };

export const TaskOutputsSpec_ArtifactsEntry = {
  encode(
    message: TaskOutputsSpec_ArtifactsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      TaskOutputsSpec_OutputArtifactSpec.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskOutputsSpec_ArtifactsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTaskOutputsSpec_ArtifactsEntry } as TaskOutputsSpec_ArtifactsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = TaskOutputsSpec_OutputArtifactSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TaskOutputsSpec_ArtifactsEntry {
    const message = { ...baseTaskOutputsSpec_ArtifactsEntry } as TaskOutputsSpec_ArtifactsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? TaskOutputsSpec_OutputArtifactSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: TaskOutputsSpec_ArtifactsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? TaskOutputsSpec_OutputArtifactSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskOutputsSpec_ArtifactsEntry>, I>>(
    object: I,
  ): TaskOutputsSpec_ArtifactsEntry {
    const message = { ...baseTaskOutputsSpec_ArtifactsEntry } as TaskOutputsSpec_ArtifactsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? TaskOutputsSpec_OutputArtifactSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const basePrimitiveType: object = {};

export const PrimitiveType = {
  encode(_: PrimitiveType, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PrimitiveType {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePrimitiveType } as PrimitiveType;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): PrimitiveType {
    const message = { ...basePrimitiveType } as PrimitiveType;
    return message;
  },

  toJSON(_: PrimitiveType): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PrimitiveType>, I>>(_: I): PrimitiveType {
    const message = { ...basePrimitiveType } as PrimitiveType;
    return message;
  },
};

const baseParameterType: object = {};

export const ParameterType = {
  encode(_: ParameterType, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParameterType {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseParameterType } as ParameterType;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): ParameterType {
    const message = { ...baseParameterType } as ParameterType;
    return message;
  },

  toJSON(_: ParameterType): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ParameterType>, I>>(_: I): ParameterType {
    const message = { ...baseParameterType } as ParameterType;
    return message;
  },
};

const basePipelineTaskSpec: object = { dependentTasks: '' };

export const PipelineTaskSpec = {
  encode(message: PipelineTaskSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.taskInfo !== undefined) {
      PipelineTaskInfo.encode(message.taskInfo, writer.uint32(10).fork()).ldelim();
    }
    if (message.inputs !== undefined) {
      TaskInputsSpec.encode(message.inputs, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.dependentTasks) {
      writer.uint32(42).string(v!);
    }
    if (message.cachingOptions !== undefined) {
      PipelineTaskSpec_CachingOptions.encode(
        message.cachingOptions,
        writer.uint32(50).fork(),
      ).ldelim();
    }
    if (message.componentRef !== undefined) {
      ComponentRef.encode(message.componentRef, writer.uint32(58).fork()).ldelim();
    }
    if (message.triggerPolicy !== undefined) {
      PipelineTaskSpec_TriggerPolicy.encode(
        message.triggerPolicy,
        writer.uint32(66).fork(),
      ).ldelim();
    }
    if (message.artifactIterator !== undefined) {
      ArtifactIteratorSpec.encode(message.artifactIterator, writer.uint32(74).fork()).ldelim();
    }
    if (message.parameterIterator !== undefined) {
      ParameterIteratorSpec.encode(message.parameterIterator, writer.uint32(82).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineTaskSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineTaskSpec } as PipelineTaskSpec;
    message.dependentTasks = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.taskInfo = PipelineTaskInfo.decode(reader, reader.uint32());
          break;
        case 2:
          message.inputs = TaskInputsSpec.decode(reader, reader.uint32());
          break;
        case 5:
          message.dependentTasks.push(reader.string());
          break;
        case 6:
          message.cachingOptions = PipelineTaskSpec_CachingOptions.decode(reader, reader.uint32());
          break;
        case 7:
          message.componentRef = ComponentRef.decode(reader, reader.uint32());
          break;
        case 8:
          message.triggerPolicy = PipelineTaskSpec_TriggerPolicy.decode(reader, reader.uint32());
          break;
        case 9:
          message.artifactIterator = ArtifactIteratorSpec.decode(reader, reader.uint32());
          break;
        case 10:
          message.parameterIterator = ParameterIteratorSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineTaskSpec {
    const message = { ...basePipelineTaskSpec } as PipelineTaskSpec;
    message.taskInfo =
      object.taskInfo !== undefined && object.taskInfo !== null
        ? PipelineTaskInfo.fromJSON(object.taskInfo)
        : undefined;
    message.inputs =
      object.inputs !== undefined && object.inputs !== null
        ? TaskInputsSpec.fromJSON(object.inputs)
        : undefined;
    message.dependentTasks = (object.dependentTasks ?? []).map((e: any) => String(e));
    message.cachingOptions =
      object.cachingOptions !== undefined && object.cachingOptions !== null
        ? PipelineTaskSpec_CachingOptions.fromJSON(object.cachingOptions)
        : undefined;
    message.componentRef =
      object.componentRef !== undefined && object.componentRef !== null
        ? ComponentRef.fromJSON(object.componentRef)
        : undefined;
    message.triggerPolicy =
      object.triggerPolicy !== undefined && object.triggerPolicy !== null
        ? PipelineTaskSpec_TriggerPolicy.fromJSON(object.triggerPolicy)
        : undefined;
    message.artifactIterator =
      object.artifactIterator !== undefined && object.artifactIterator !== null
        ? ArtifactIteratorSpec.fromJSON(object.artifactIterator)
        : undefined;
    message.parameterIterator =
      object.parameterIterator !== undefined && object.parameterIterator !== null
        ? ParameterIteratorSpec.fromJSON(object.parameterIterator)
        : undefined;
    return message;
  },

  toJSON(message: PipelineTaskSpec): unknown {
    const obj: any = {};
    message.taskInfo !== undefined &&
      (obj.taskInfo = message.taskInfo ? PipelineTaskInfo.toJSON(message.taskInfo) : undefined);
    message.inputs !== undefined &&
      (obj.inputs = message.inputs ? TaskInputsSpec.toJSON(message.inputs) : undefined);
    if (message.dependentTasks) {
      obj.dependentTasks = message.dependentTasks.map((e) => e);
    } else {
      obj.dependentTasks = [];
    }
    message.cachingOptions !== undefined &&
      (obj.cachingOptions = message.cachingOptions
        ? PipelineTaskSpec_CachingOptions.toJSON(message.cachingOptions)
        : undefined);
    message.componentRef !== undefined &&
      (obj.componentRef = message.componentRef
        ? ComponentRef.toJSON(message.componentRef)
        : undefined);
    message.triggerPolicy !== undefined &&
      (obj.triggerPolicy = message.triggerPolicy
        ? PipelineTaskSpec_TriggerPolicy.toJSON(message.triggerPolicy)
        : undefined);
    message.artifactIterator !== undefined &&
      (obj.artifactIterator = message.artifactIterator
        ? ArtifactIteratorSpec.toJSON(message.artifactIterator)
        : undefined);
    message.parameterIterator !== undefined &&
      (obj.parameterIterator = message.parameterIterator
        ? ParameterIteratorSpec.toJSON(message.parameterIterator)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineTaskSpec>, I>>(object: I): PipelineTaskSpec {
    const message = { ...basePipelineTaskSpec } as PipelineTaskSpec;
    message.taskInfo =
      object.taskInfo !== undefined && object.taskInfo !== null
        ? PipelineTaskInfo.fromPartial(object.taskInfo)
        : undefined;
    message.inputs =
      object.inputs !== undefined && object.inputs !== null
        ? TaskInputsSpec.fromPartial(object.inputs)
        : undefined;
    message.dependentTasks = object.dependentTasks?.map((e) => e) || [];
    message.cachingOptions =
      object.cachingOptions !== undefined && object.cachingOptions !== null
        ? PipelineTaskSpec_CachingOptions.fromPartial(object.cachingOptions)
        : undefined;
    message.componentRef =
      object.componentRef !== undefined && object.componentRef !== null
        ? ComponentRef.fromPartial(object.componentRef)
        : undefined;
    message.triggerPolicy =
      object.triggerPolicy !== undefined && object.triggerPolicy !== null
        ? PipelineTaskSpec_TriggerPolicy.fromPartial(object.triggerPolicy)
        : undefined;
    message.artifactIterator =
      object.artifactIterator !== undefined && object.artifactIterator !== null
        ? ArtifactIteratorSpec.fromPartial(object.artifactIterator)
        : undefined;
    message.parameterIterator =
      object.parameterIterator !== undefined && object.parameterIterator !== null
        ? ParameterIteratorSpec.fromPartial(object.parameterIterator)
        : undefined;
    return message;
  },
};

const basePipelineTaskSpec_CachingOptions: object = { enableCache: false };

export const PipelineTaskSpec_CachingOptions = {
  encode(
    message: PipelineTaskSpec_CachingOptions,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.enableCache === true) {
      writer.uint32(8).bool(message.enableCache);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineTaskSpec_CachingOptions {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineTaskSpec_CachingOptions } as PipelineTaskSpec_CachingOptions;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.enableCache = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineTaskSpec_CachingOptions {
    const message = { ...basePipelineTaskSpec_CachingOptions } as PipelineTaskSpec_CachingOptions;
    message.enableCache =
      object.enableCache !== undefined && object.enableCache !== null
        ? Boolean(object.enableCache)
        : false;
    return message;
  },

  toJSON(message: PipelineTaskSpec_CachingOptions): unknown {
    const obj: any = {};
    message.enableCache !== undefined && (obj.enableCache = message.enableCache);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineTaskSpec_CachingOptions>, I>>(
    object: I,
  ): PipelineTaskSpec_CachingOptions {
    const message = { ...basePipelineTaskSpec_CachingOptions } as PipelineTaskSpec_CachingOptions;
    message.enableCache = object.enableCache ?? false;
    return message;
  },
};

const basePipelineTaskSpec_TriggerPolicy: object = { condition: '', strategy: 0 };

export const PipelineTaskSpec_TriggerPolicy = {
  encode(
    message: PipelineTaskSpec_TriggerPolicy,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.condition !== '') {
      writer.uint32(10).string(message.condition);
    }
    if (message.strategy !== 0) {
      writer.uint32(16).int32(message.strategy);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineTaskSpec_TriggerPolicy {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineTaskSpec_TriggerPolicy } as PipelineTaskSpec_TriggerPolicy;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.condition = reader.string();
          break;
        case 2:
          message.strategy = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineTaskSpec_TriggerPolicy {
    const message = { ...basePipelineTaskSpec_TriggerPolicy } as PipelineTaskSpec_TriggerPolicy;
    message.condition =
      object.condition !== undefined && object.condition !== null ? String(object.condition) : '';
    message.strategy =
      object.strategy !== undefined && object.strategy !== null
        ? pipelineTaskSpec_TriggerPolicy_TriggerStrategyFromJSON(object.strategy)
        : 0;
    return message;
  },

  toJSON(message: PipelineTaskSpec_TriggerPolicy): unknown {
    const obj: any = {};
    message.condition !== undefined && (obj.condition = message.condition);
    message.strategy !== undefined &&
      (obj.strategy = pipelineTaskSpec_TriggerPolicy_TriggerStrategyToJSON(message.strategy));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineTaskSpec_TriggerPolicy>, I>>(
    object: I,
  ): PipelineTaskSpec_TriggerPolicy {
    const message = { ...basePipelineTaskSpec_TriggerPolicy } as PipelineTaskSpec_TriggerPolicy;
    message.condition = object.condition ?? '';
    message.strategy = object.strategy ?? 0;
    return message;
  },
};

const baseArtifactIteratorSpec: object = { itemInput: '' };

export const ArtifactIteratorSpec = {
  encode(message: ArtifactIteratorSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.items !== undefined) {
      ArtifactIteratorSpec_ItemsSpec.encode(message.items, writer.uint32(10).fork()).ldelim();
    }
    if (message.itemInput !== '') {
      writer.uint32(18).string(message.itemInput);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ArtifactIteratorSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseArtifactIteratorSpec } as ArtifactIteratorSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.items = ArtifactIteratorSpec_ItemsSpec.decode(reader, reader.uint32());
          break;
        case 2:
          message.itemInput = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ArtifactIteratorSpec {
    const message = { ...baseArtifactIteratorSpec } as ArtifactIteratorSpec;
    message.items =
      object.items !== undefined && object.items !== null
        ? ArtifactIteratorSpec_ItemsSpec.fromJSON(object.items)
        : undefined;
    message.itemInput =
      object.itemInput !== undefined && object.itemInput !== null ? String(object.itemInput) : '';
    return message;
  },

  toJSON(message: ArtifactIteratorSpec): unknown {
    const obj: any = {};
    message.items !== undefined &&
      (obj.items = message.items
        ? ArtifactIteratorSpec_ItemsSpec.toJSON(message.items)
        : undefined);
    message.itemInput !== undefined && (obj.itemInput = message.itemInput);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ArtifactIteratorSpec>, I>>(
    object: I,
  ): ArtifactIteratorSpec {
    const message = { ...baseArtifactIteratorSpec } as ArtifactIteratorSpec;
    message.items =
      object.items !== undefined && object.items !== null
        ? ArtifactIteratorSpec_ItemsSpec.fromPartial(object.items)
        : undefined;
    message.itemInput = object.itemInput ?? '';
    return message;
  },
};

const baseArtifactIteratorSpec_ItemsSpec: object = { inputArtifact: '' };

export const ArtifactIteratorSpec_ItemsSpec = {
  encode(
    message: ArtifactIteratorSpec_ItemsSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.inputArtifact !== '') {
      writer.uint32(10).string(message.inputArtifact);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ArtifactIteratorSpec_ItemsSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseArtifactIteratorSpec_ItemsSpec } as ArtifactIteratorSpec_ItemsSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.inputArtifact = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ArtifactIteratorSpec_ItemsSpec {
    const message = { ...baseArtifactIteratorSpec_ItemsSpec } as ArtifactIteratorSpec_ItemsSpec;
    message.inputArtifact =
      object.inputArtifact !== undefined && object.inputArtifact !== null
        ? String(object.inputArtifact)
        : '';
    return message;
  },

  toJSON(message: ArtifactIteratorSpec_ItemsSpec): unknown {
    const obj: any = {};
    message.inputArtifact !== undefined && (obj.inputArtifact = message.inputArtifact);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ArtifactIteratorSpec_ItemsSpec>, I>>(
    object: I,
  ): ArtifactIteratorSpec_ItemsSpec {
    const message = { ...baseArtifactIteratorSpec_ItemsSpec } as ArtifactIteratorSpec_ItemsSpec;
    message.inputArtifact = object.inputArtifact ?? '';
    return message;
  },
};

const baseParameterIteratorSpec: object = { itemInput: '' };

export const ParameterIteratorSpec = {
  encode(message: ParameterIteratorSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.items !== undefined) {
      ParameterIteratorSpec_ItemsSpec.encode(message.items, writer.uint32(10).fork()).ldelim();
    }
    if (message.itemInput !== '') {
      writer.uint32(18).string(message.itemInput);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParameterIteratorSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseParameterIteratorSpec } as ParameterIteratorSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.items = ParameterIteratorSpec_ItemsSpec.decode(reader, reader.uint32());
          break;
        case 2:
          message.itemInput = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ParameterIteratorSpec {
    const message = { ...baseParameterIteratorSpec } as ParameterIteratorSpec;
    message.items =
      object.items !== undefined && object.items !== null
        ? ParameterIteratorSpec_ItemsSpec.fromJSON(object.items)
        : undefined;
    message.itemInput =
      object.itemInput !== undefined && object.itemInput !== null ? String(object.itemInput) : '';
    return message;
  },

  toJSON(message: ParameterIteratorSpec): unknown {
    const obj: any = {};
    message.items !== undefined &&
      (obj.items = message.items
        ? ParameterIteratorSpec_ItemsSpec.toJSON(message.items)
        : undefined);
    message.itemInput !== undefined && (obj.itemInput = message.itemInput);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ParameterIteratorSpec>, I>>(
    object: I,
  ): ParameterIteratorSpec {
    const message = { ...baseParameterIteratorSpec } as ParameterIteratorSpec;
    message.items =
      object.items !== undefined && object.items !== null
        ? ParameterIteratorSpec_ItemsSpec.fromPartial(object.items)
        : undefined;
    message.itemInput = object.itemInput ?? '';
    return message;
  },
};

const baseParameterIteratorSpec_ItemsSpec: object = {};

export const ParameterIteratorSpec_ItemsSpec = {
  encode(
    message: ParameterIteratorSpec_ItemsSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.raw !== undefined) {
      writer.uint32(10).string(message.raw);
    }
    if (message.inputParameter !== undefined) {
      writer.uint32(18).string(message.inputParameter);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParameterIteratorSpec_ItemsSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseParameterIteratorSpec_ItemsSpec } as ParameterIteratorSpec_ItemsSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.raw = reader.string();
          break;
        case 2:
          message.inputParameter = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ParameterIteratorSpec_ItemsSpec {
    const message = { ...baseParameterIteratorSpec_ItemsSpec } as ParameterIteratorSpec_ItemsSpec;
    message.raw = object.raw !== undefined && object.raw !== null ? String(object.raw) : undefined;
    message.inputParameter =
      object.inputParameter !== undefined && object.inputParameter !== null
        ? String(object.inputParameter)
        : undefined;
    return message;
  },

  toJSON(message: ParameterIteratorSpec_ItemsSpec): unknown {
    const obj: any = {};
    message.raw !== undefined && (obj.raw = message.raw);
    message.inputParameter !== undefined && (obj.inputParameter = message.inputParameter);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ParameterIteratorSpec_ItemsSpec>, I>>(
    object: I,
  ): ParameterIteratorSpec_ItemsSpec {
    const message = { ...baseParameterIteratorSpec_ItemsSpec } as ParameterIteratorSpec_ItemsSpec;
    message.raw = object.raw ?? undefined;
    message.inputParameter = object.inputParameter ?? undefined;
    return message;
  },
};

const baseComponentRef: object = { name: '' };

export const ComponentRef = {
  encode(message: ComponentRef, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== '') {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ComponentRef {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseComponentRef } as ComponentRef;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ComponentRef {
    const message = { ...baseComponentRef } as ComponentRef;
    message.name = object.name !== undefined && object.name !== null ? String(object.name) : '';
    return message;
  },

  toJSON(message: ComponentRef): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ComponentRef>, I>>(object: I): ComponentRef {
    const message = { ...baseComponentRef } as ComponentRef;
    message.name = object.name ?? '';
    return message;
  },
};

const basePipelineInfo: object = { name: '' };

export const PipelineInfo = {
  encode(message: PipelineInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== '') {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineInfo {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineInfo } as PipelineInfo;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineInfo {
    const message = { ...basePipelineInfo } as PipelineInfo;
    message.name = object.name !== undefined && object.name !== null ? String(object.name) : '';
    return message;
  },

  toJSON(message: PipelineInfo): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineInfo>, I>>(object: I): PipelineInfo {
    const message = { ...basePipelineInfo } as PipelineInfo;
    message.name = object.name ?? '';
    return message;
  },
};

const baseArtifactTypeSchema: object = { schemaVersion: '' };

export const ArtifactTypeSchema = {
  encode(message: ArtifactTypeSchema, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaTitle !== undefined) {
      writer.uint32(10).string(message.schemaTitle);
    }
    if (message.schemaUri !== undefined) {
      writer.uint32(18).string(message.schemaUri);
    }
    if (message.instanceSchema !== undefined) {
      writer.uint32(26).string(message.instanceSchema);
    }
    if (message.schemaVersion !== '') {
      writer.uint32(34).string(message.schemaVersion);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ArtifactTypeSchema {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseArtifactTypeSchema } as ArtifactTypeSchema;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.schemaTitle = reader.string();
          break;
        case 2:
          message.schemaUri = reader.string();
          break;
        case 3:
          message.instanceSchema = reader.string();
          break;
        case 4:
          message.schemaVersion = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ArtifactTypeSchema {
    const message = { ...baseArtifactTypeSchema } as ArtifactTypeSchema;
    message.schemaTitle =
      object.schemaTitle !== undefined && object.schemaTitle !== null
        ? String(object.schemaTitle)
        : undefined;
    message.schemaUri =
      object.schemaUri !== undefined && object.schemaUri !== null
        ? String(object.schemaUri)
        : undefined;
    message.instanceSchema =
      object.instanceSchema !== undefined && object.instanceSchema !== null
        ? String(object.instanceSchema)
        : undefined;
    message.schemaVersion =
      object.schemaVersion !== undefined && object.schemaVersion !== null
        ? String(object.schemaVersion)
        : '';
    return message;
  },

  toJSON(message: ArtifactTypeSchema): unknown {
    const obj: any = {};
    message.schemaTitle !== undefined && (obj.schemaTitle = message.schemaTitle);
    message.schemaUri !== undefined && (obj.schemaUri = message.schemaUri);
    message.instanceSchema !== undefined && (obj.instanceSchema = message.instanceSchema);
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ArtifactTypeSchema>, I>>(object: I): ArtifactTypeSchema {
    const message = { ...baseArtifactTypeSchema } as ArtifactTypeSchema;
    message.schemaTitle = object.schemaTitle ?? undefined;
    message.schemaUri = object.schemaUri ?? undefined;
    message.instanceSchema = object.instanceSchema ?? undefined;
    message.schemaVersion = object.schemaVersion ?? '';
    return message;
  },
};

const basePipelineTaskInfo: object = { name: '' };

export const PipelineTaskInfo = {
  encode(message: PipelineTaskInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== '') {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineTaskInfo {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineTaskInfo } as PipelineTaskInfo;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineTaskInfo {
    const message = { ...basePipelineTaskInfo } as PipelineTaskInfo;
    message.name = object.name !== undefined && object.name !== null ? String(object.name) : '';
    return message;
  },

  toJSON(message: PipelineTaskInfo): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineTaskInfo>, I>>(object: I): PipelineTaskInfo {
    const message = { ...basePipelineTaskInfo } as PipelineTaskInfo;
    message.name = object.name ?? '';
    return message;
  },
};

const baseValueOrRuntimeParameter: object = {};

export const ValueOrRuntimeParameter = {
  encode(message: ValueOrRuntimeParameter, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.constantValue !== undefined) {
      Value.encode(message.constantValue, writer.uint32(10).fork()).ldelim();
    }
    if (message.runtimeParameter !== undefined) {
      writer.uint32(18).string(message.runtimeParameter);
    }
    if (message.constant !== undefined) {
      Value1.encode(Value1.wrap(message.constant), writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ValueOrRuntimeParameter {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseValueOrRuntimeParameter } as ValueOrRuntimeParameter;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.constantValue = Value.decode(reader, reader.uint32());
          break;
        case 2:
          message.runtimeParameter = reader.string();
          break;
        case 3:
          message.constant = Value1.unwrap(Value1.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ValueOrRuntimeParameter {
    const message = { ...baseValueOrRuntimeParameter } as ValueOrRuntimeParameter;
    message.constantValue =
      object.constantValue !== undefined && object.constantValue !== null
        ? Value.fromJSON(object.constantValue)
        : undefined;
    message.runtimeParameter =
      object.runtimeParameter !== undefined && object.runtimeParameter !== null
        ? String(object.runtimeParameter)
        : undefined;
    message.constant = object.constant;
    return message;
  },

  toJSON(message: ValueOrRuntimeParameter): unknown {
    const obj: any = {};
    message.constantValue !== undefined &&
      (obj.constantValue = message.constantValue ? Value.toJSON(message.constantValue) : undefined);
    message.runtimeParameter !== undefined && (obj.runtimeParameter = message.runtimeParameter);
    message.constant !== undefined && (obj.constant = message.constant);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ValueOrRuntimeParameter>, I>>(
    object: I,
  ): ValueOrRuntimeParameter {
    const message = { ...baseValueOrRuntimeParameter } as ValueOrRuntimeParameter;
    message.constantValue =
      object.constantValue !== undefined && object.constantValue !== null
        ? Value.fromPartial(object.constantValue)
        : undefined;
    message.runtimeParameter = object.runtimeParameter ?? undefined;
    message.constant = object.constant ?? undefined;
    return message;
  },
};

const basePipelineDeploymentConfig: object = {};

export const PipelineDeploymentConfig = {
  encode(message: PipelineDeploymentConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.executors).forEach(([key, value]) => {
      PipelineDeploymentConfig_ExecutorsEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineDeploymentConfig {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineDeploymentConfig } as PipelineDeploymentConfig;
    message.executors = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = PipelineDeploymentConfig_ExecutorsEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.executors[entry1.key] = entry1.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig {
    const message = { ...basePipelineDeploymentConfig } as PipelineDeploymentConfig;
    message.executors = Object.entries(object.executors ?? {}).reduce<{
      [key: string]: PipelineDeploymentConfig_ExecutorSpec;
    }>((acc, [key, value]) => {
      acc[key] = PipelineDeploymentConfig_ExecutorSpec.fromJSON(value);
      return acc;
    }, {});
    return message;
  },

  toJSON(message: PipelineDeploymentConfig): unknown {
    const obj: any = {};
    obj.executors = {};
    if (message.executors) {
      Object.entries(message.executors).forEach(([k, v]) => {
        obj.executors[k] = PipelineDeploymentConfig_ExecutorSpec.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineDeploymentConfig>, I>>(
    object: I,
  ): PipelineDeploymentConfig {
    const message = { ...basePipelineDeploymentConfig } as PipelineDeploymentConfig;
    message.executors = Object.entries(object.executors ?? {}).reduce<{
      [key: string]: PipelineDeploymentConfig_ExecutorSpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = PipelineDeploymentConfig_ExecutorSpec.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

const basePipelineDeploymentConfig_PipelineContainerSpec: object = {
  image: '',
  command: '',
  args: '',
};

export const PipelineDeploymentConfig_PipelineContainerSpec = {
  encode(
    message: PipelineDeploymentConfig_PipelineContainerSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.image !== '') {
      writer.uint32(10).string(message.image);
    }
    for (const v of message.command) {
      writer.uint32(18).string(v!);
    }
    for (const v of message.args) {
      writer.uint32(26).string(v!);
    }
    if (message.lifecycle !== undefined) {
      PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle.encode(
        message.lifecycle,
        writer.uint32(34).fork(),
      ).ldelim();
    }
    if (message.resources !== undefined) {
      PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec.encode(
        message.resources,
        writer.uint32(42).fork(),
      ).ldelim();
    }
    for (const v of message.env) {
      PipelineDeploymentConfig_PipelineContainerSpec_EnvVar.encode(
        v!,
        writer.uint32(50).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_PipelineContainerSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec,
    } as PipelineDeploymentConfig_PipelineContainerSpec;
    message.command = [];
    message.args = [];
    message.env = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.image = reader.string();
          break;
        case 2:
          message.command.push(reader.string());
          break;
        case 3:
          message.args.push(reader.string());
          break;
        case 4:
          message.lifecycle = PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle.decode(
            reader,
            reader.uint32(),
          );
          break;
        case 5:
          message.resources = PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec.decode(
            reader,
            reader.uint32(),
          );
          break;
        case 6:
          message.env.push(
            PipelineDeploymentConfig_PipelineContainerSpec_EnvVar.decode(reader, reader.uint32()),
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_PipelineContainerSpec {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec,
    } as PipelineDeploymentConfig_PipelineContainerSpec;
    message.image = object.image !== undefined && object.image !== null ? String(object.image) : '';
    message.command = (object.command ?? []).map((e: any) => String(e));
    message.args = (object.args ?? []).map((e: any) => String(e));
    message.lifecycle =
      object.lifecycle !== undefined && object.lifecycle !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle.fromJSON(object.lifecycle)
        : undefined;
    message.resources =
      object.resources !== undefined && object.resources !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec.fromJSON(object.resources)
        : undefined;
    message.env = (object.env ?? []).map((e: any) =>
      PipelineDeploymentConfig_PipelineContainerSpec_EnvVar.fromJSON(e),
    );
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_PipelineContainerSpec): unknown {
    const obj: any = {};
    message.image !== undefined && (obj.image = message.image);
    if (message.command) {
      obj.command = message.command.map((e) => e);
    } else {
      obj.command = [];
    }
    if (message.args) {
      obj.args = message.args.map((e) => e);
    } else {
      obj.args = [];
    }
    message.lifecycle !== undefined &&
      (obj.lifecycle = message.lifecycle
        ? PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle.toJSON(message.lifecycle)
        : undefined);
    message.resources !== undefined &&
      (obj.resources = message.resources
        ? PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec.toJSON(message.resources)
        : undefined);
    if (message.env) {
      obj.env = message.env.map((e) =>
        e ? PipelineDeploymentConfig_PipelineContainerSpec_EnvVar.toJSON(e) : undefined,
      );
    } else {
      obj.env = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineDeploymentConfig_PipelineContainerSpec>, I>>(
    object: I,
  ): PipelineDeploymentConfig_PipelineContainerSpec {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec,
    } as PipelineDeploymentConfig_PipelineContainerSpec;
    message.image = object.image ?? '';
    message.command = object.command?.map((e) => e) || [];
    message.args = object.args?.map((e) => e) || [];
    message.lifecycle =
      object.lifecycle !== undefined && object.lifecycle !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle.fromPartial(object.lifecycle)
        : undefined;
    message.resources =
      object.resources !== undefined && object.resources !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec.fromPartial(object.resources)
        : undefined;
    message.env =
      object.env?.map((e) =>
        PipelineDeploymentConfig_PipelineContainerSpec_EnvVar.fromPartial(e),
      ) || [];
    return message;
  },
};

const basePipelineDeploymentConfig_PipelineContainerSpec_Lifecycle: object = {};

export const PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle = {
  encode(
    message: PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.preCacheCheck !== undefined) {
      PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec.encode(
        message.preCacheCheck,
        writer.uint32(10).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_Lifecycle,
    } as PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.preCacheCheck =
            PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec.decode(
              reader,
              reader.uint32(),
            );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_Lifecycle,
    } as PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle;
    message.preCacheCheck =
      object.preCacheCheck !== undefined && object.preCacheCheck !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec.fromJSON(
            object.preCacheCheck,
          )
        : undefined;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle): unknown {
    const obj: any = {};
    message.preCacheCheck !== undefined &&
      (obj.preCacheCheck = message.preCacheCheck
        ? PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec.toJSON(
            message.preCacheCheck,
          )
        : undefined);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle>, I>,
  >(object: I): PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_Lifecycle,
    } as PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle;
    message.preCacheCheck =
      object.preCacheCheck !== undefined && object.preCacheCheck !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec.fromPartial(
            object.preCacheCheck,
          )
        : undefined;
    return message;
  },
};

const basePipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec: object = {
  command: '',
  args: '',
};

export const PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec = {
  encode(
    message: PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    for (const v of message.command) {
      writer.uint32(18).string(v!);
    }
    for (const v of message.args) {
      writer.uint32(26).string(v!);
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec,
    } as PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec;
    message.command = [];
    message.args = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          message.command.push(reader.string());
          break;
        case 3:
          message.args.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec,
    } as PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec;
    message.command = (object.command ?? []).map((e: any) => String(e));
    message.args = (object.args ?? []).map((e: any) => String(e));
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec): unknown {
    const obj: any = {};
    if (message.command) {
      obj.command = message.command.map((e) => e);
    } else {
      obj.command = [];
    }
    if (message.args) {
      obj.args = message.args.map((e) => e);
    } else {
      obj.args = [];
    }
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec>, I>,
  >(object: I): PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec,
    } as PipelineDeploymentConfig_PipelineContainerSpec_Lifecycle_Exec;
    message.command = object.command?.map((e) => e) || [];
    message.args = object.args?.map((e) => e) || [];
    return message;
  },
};

const basePipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec: object = {
  cpuLimit: 0,
  memoryLimit: 0,
};

export const PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec = {
  encode(
    message: PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.cpuLimit !== 0) {
      writer.uint32(9).double(message.cpuLimit);
    }
    if (message.memoryLimit !== 0) {
      writer.uint32(17).double(message.memoryLimit);
    }
    if (message.accelerator !== undefined) {
      PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig.encode(
        message.accelerator,
        writer.uint32(26).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec,
    } as PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.cpuLimit = reader.double();
          break;
        case 2:
          message.memoryLimit = reader.double();
          break;
        case 3:
          message.accelerator =
            PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig.decode(
              reader,
              reader.uint32(),
            );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec,
    } as PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec;
    message.cpuLimit =
      object.cpuLimit !== undefined && object.cpuLimit !== null ? Number(object.cpuLimit) : 0;
    message.memoryLimit =
      object.memoryLimit !== undefined && object.memoryLimit !== null
        ? Number(object.memoryLimit)
        : 0;
    message.accelerator =
      object.accelerator !== undefined && object.accelerator !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig.fromJSON(
            object.accelerator,
          )
        : undefined;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec): unknown {
    const obj: any = {};
    message.cpuLimit !== undefined && (obj.cpuLimit = message.cpuLimit);
    message.memoryLimit !== undefined && (obj.memoryLimit = message.memoryLimit);
    message.accelerator !== undefined &&
      (obj.accelerator = message.accelerator
        ? PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig.toJSON(
            message.accelerator,
          )
        : undefined);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec>, I>,
  >(object: I): PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec,
    } as PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec;
    message.cpuLimit = object.cpuLimit ?? 0;
    message.memoryLimit = object.memoryLimit ?? 0;
    message.accelerator =
      object.accelerator !== undefined && object.accelerator !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig.fromPartial(
            object.accelerator,
          )
        : undefined;
    return message;
  },
};

const basePipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig: object = {
  type: '',
  count: 0,
};

export const PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig = {
  encode(
    message: PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.type !== '') {
      writer.uint32(10).string(message.type);
    }
    if (message.count !== 0) {
      writer.uint32(16).int64(message.count);
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig,
    } as PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.string();
          break;
        case 2:
          message.count = longToNumber(reader.int64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(
    object: any,
  ): PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig,
    } as PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig;
    message.type = object.type !== undefined && object.type !== null ? String(object.type) : '';
    message.count = object.count !== undefined && object.count !== null ? Number(object.count) : 0;
    return message;
  },

  toJSON(
    message: PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig,
  ): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = message.type);
    message.count !== undefined && (obj.count = Math.round(message.count));
    return obj;
  },

  fromPartial<
    I extends Exact<
      DeepPartial<PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig>,
      I
    >,
  >(object: I): PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig,
    } as PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig;
    message.type = object.type ?? '';
    message.count = object.count ?? 0;
    return message;
  },
};

const basePipelineDeploymentConfig_PipelineContainerSpec_EnvVar: object = { name: '', value: '' };

export const PipelineDeploymentConfig_PipelineContainerSpec_EnvVar = {
  encode(
    message: PipelineDeploymentConfig_PipelineContainerSpec_EnvVar,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.name !== '') {
      writer.uint32(10).string(message.name);
    }
    if (message.value !== '') {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_PipelineContainerSpec_EnvVar {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_EnvVar,
    } as PipelineDeploymentConfig_PipelineContainerSpec_EnvVar;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.value = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_PipelineContainerSpec_EnvVar {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_EnvVar,
    } as PipelineDeploymentConfig_PipelineContainerSpec_EnvVar;
    message.name = object.name !== undefined && object.name !== null ? String(object.name) : '';
    message.value = object.value !== undefined && object.value !== null ? String(object.value) : '';
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_PipelineContainerSpec_EnvVar): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<PipelineDeploymentConfig_PipelineContainerSpec_EnvVar>, I>,
  >(object: I): PipelineDeploymentConfig_PipelineContainerSpec_EnvVar {
    const message = {
      ...basePipelineDeploymentConfig_PipelineContainerSpec_EnvVar,
    } as PipelineDeploymentConfig_PipelineContainerSpec_EnvVar;
    message.name = object.name ?? '';
    message.value = object.value ?? '';
    return message;
  },
};

const basePipelineDeploymentConfig_ImporterSpec: object = { reimport: false };

export const PipelineDeploymentConfig_ImporterSpec = {
  encode(
    message: PipelineDeploymentConfig_ImporterSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.artifactUri !== undefined) {
      ValueOrRuntimeParameter.encode(message.artifactUri, writer.uint32(10).fork()).ldelim();
    }
    if (message.typeSchema !== undefined) {
      ArtifactTypeSchema.encode(message.typeSchema, writer.uint32(18).fork()).ldelim();
    }
    Object.entries(message.properties).forEach(([key, value]) => {
      PipelineDeploymentConfig_ImporterSpec_PropertiesEntry.encode(
        { key: key as any, value },
        writer.uint32(26).fork(),
      ).ldelim();
    });
    Object.entries(message.customProperties).forEach(([key, value]) => {
      PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry.encode(
        { key: key as any, value },
        writer.uint32(34).fork(),
      ).ldelim();
    });
    if (message.metadata !== undefined) {
      Struct.encode(Struct.wrap(message.metadata), writer.uint32(50).fork()).ldelim();
    }
    if (message.reimport === true) {
      writer.uint32(40).bool(message.reimport);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineDeploymentConfig_ImporterSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_ImporterSpec,
    } as PipelineDeploymentConfig_ImporterSpec;
    message.properties = {};
    message.customProperties = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.artifactUri = ValueOrRuntimeParameter.decode(reader, reader.uint32());
          break;
        case 2:
          message.typeSchema = ArtifactTypeSchema.decode(reader, reader.uint32());
          break;
        case 3:
          const entry3 = PipelineDeploymentConfig_ImporterSpec_PropertiesEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry3.value !== undefined) {
            message.properties[entry3.key] = entry3.value;
          }
          break;
        case 4:
          const entry4 = PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry4.value !== undefined) {
            message.customProperties[entry4.key] = entry4.value;
          }
          break;
        case 6:
          message.metadata = Struct.unwrap(Struct.decode(reader, reader.uint32()));
          break;
        case 5:
          message.reimport = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_ImporterSpec {
    const message = {
      ...basePipelineDeploymentConfig_ImporterSpec,
    } as PipelineDeploymentConfig_ImporterSpec;
    message.artifactUri =
      object.artifactUri !== undefined && object.artifactUri !== null
        ? ValueOrRuntimeParameter.fromJSON(object.artifactUri)
        : undefined;
    message.typeSchema =
      object.typeSchema !== undefined && object.typeSchema !== null
        ? ArtifactTypeSchema.fromJSON(object.typeSchema)
        : undefined;
    message.properties = Object.entries(object.properties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      acc[key] = ValueOrRuntimeParameter.fromJSON(value);
      return acc;
    }, {});
    message.customProperties = Object.entries(object.customProperties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      acc[key] = ValueOrRuntimeParameter.fromJSON(value);
      return acc;
    }, {});
    message.metadata = typeof object.metadata === 'object' ? object.metadata : undefined;
    message.reimport =
      object.reimport !== undefined && object.reimport !== null ? Boolean(object.reimport) : false;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_ImporterSpec): unknown {
    const obj: any = {};
    message.artifactUri !== undefined &&
      (obj.artifactUri = message.artifactUri
        ? ValueOrRuntimeParameter.toJSON(message.artifactUri)
        : undefined);
    message.typeSchema !== undefined &&
      (obj.typeSchema = message.typeSchema
        ? ArtifactTypeSchema.toJSON(message.typeSchema)
        : undefined);
    obj.properties = {};
    if (message.properties) {
      Object.entries(message.properties).forEach(([k, v]) => {
        obj.properties[k] = ValueOrRuntimeParameter.toJSON(v);
      });
    }
    obj.customProperties = {};
    if (message.customProperties) {
      Object.entries(message.customProperties).forEach(([k, v]) => {
        obj.customProperties[k] = ValueOrRuntimeParameter.toJSON(v);
      });
    }
    message.metadata !== undefined && (obj.metadata = message.metadata);
    message.reimport !== undefined && (obj.reimport = message.reimport);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineDeploymentConfig_ImporterSpec>, I>>(
    object: I,
  ): PipelineDeploymentConfig_ImporterSpec {
    const message = {
      ...basePipelineDeploymentConfig_ImporterSpec,
    } as PipelineDeploymentConfig_ImporterSpec;
    message.artifactUri =
      object.artifactUri !== undefined && object.artifactUri !== null
        ? ValueOrRuntimeParameter.fromPartial(object.artifactUri)
        : undefined;
    message.typeSchema =
      object.typeSchema !== undefined && object.typeSchema !== null
        ? ArtifactTypeSchema.fromPartial(object.typeSchema)
        : undefined;
    message.properties = Object.entries(object.properties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ValueOrRuntimeParameter.fromPartial(value);
      }
      return acc;
    }, {});
    message.customProperties = Object.entries(object.customProperties ?? {}).reduce<{
      [key: string]: ValueOrRuntimeParameter;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ValueOrRuntimeParameter.fromPartial(value);
      }
      return acc;
    }, {});
    message.metadata = object.metadata ?? undefined;
    message.reimport = object.reimport ?? false;
    return message;
  },
};

const basePipelineDeploymentConfig_ImporterSpec_PropertiesEntry: object = { key: '' };

export const PipelineDeploymentConfig_ImporterSpec_PropertiesEntry = {
  encode(
    message: PipelineDeploymentConfig_ImporterSpec_PropertiesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ValueOrRuntimeParameter.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_ImporterSpec_PropertiesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_ImporterSpec_PropertiesEntry,
    } as PipelineDeploymentConfig_ImporterSpec_PropertiesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ValueOrRuntimeParameter.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_ImporterSpec_PropertiesEntry {
    const message = {
      ...basePipelineDeploymentConfig_ImporterSpec_PropertiesEntry,
    } as PipelineDeploymentConfig_ImporterSpec_PropertiesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_ImporterSpec_PropertiesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ValueOrRuntimeParameter.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<PipelineDeploymentConfig_ImporterSpec_PropertiesEntry>, I>,
  >(object: I): PipelineDeploymentConfig_ImporterSpec_PropertiesEntry {
    const message = {
      ...basePipelineDeploymentConfig_ImporterSpec_PropertiesEntry,
    } as PipelineDeploymentConfig_ImporterSpec_PropertiesEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const basePipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry: object = { key: '' };

export const PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry = {
  encode(
    message: PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ValueOrRuntimeParameter.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry,
    } as PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ValueOrRuntimeParameter.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry {
    const message = {
      ...basePipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry,
    } as PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ValueOrRuntimeParameter.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry>, I>,
  >(object: I): PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry {
    const message = {
      ...basePipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry,
    } as PipelineDeploymentConfig_ImporterSpec_CustomPropertiesEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ValueOrRuntimeParameter.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const basePipelineDeploymentConfig_ResolverSpec: object = {};

export const PipelineDeploymentConfig_ResolverSpec = {
  encode(
    message: PipelineDeploymentConfig_ResolverSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    Object.entries(message.outputArtifactQueries).forEach(([key, value]) => {
      PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineDeploymentConfig_ResolverSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_ResolverSpec,
    } as PipelineDeploymentConfig_ResolverSpec;
    message.outputArtifactQueries = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry1.value !== undefined) {
            message.outputArtifactQueries[entry1.key] = entry1.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_ResolverSpec {
    const message = {
      ...basePipelineDeploymentConfig_ResolverSpec,
    } as PipelineDeploymentConfig_ResolverSpec;
    message.outputArtifactQueries = Object.entries(object.outputArtifactQueries ?? {}).reduce<{
      [key: string]: PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec;
    }>((acc, [key, value]) => {
      acc[key] = PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec.fromJSON(value);
      return acc;
    }, {});
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_ResolverSpec): unknown {
    const obj: any = {};
    obj.outputArtifactQueries = {};
    if (message.outputArtifactQueries) {
      Object.entries(message.outputArtifactQueries).forEach(([k, v]) => {
        obj.outputArtifactQueries[k] =
          PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineDeploymentConfig_ResolverSpec>, I>>(
    object: I,
  ): PipelineDeploymentConfig_ResolverSpec {
    const message = {
      ...basePipelineDeploymentConfig_ResolverSpec,
    } as PipelineDeploymentConfig_ResolverSpec;
    message.outputArtifactQueries = Object.entries(object.outputArtifactQueries ?? {}).reduce<{
      [key: string]: PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

const basePipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec: object = {
  filter: '',
  limit: 0,
};

export const PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec = {
  encode(
    message: PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.filter !== '') {
      writer.uint32(10).string(message.filter);
    }
    if (message.limit !== 0) {
      writer.uint32(16).int32(message.limit);
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec,
    } as PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.filter = reader.string();
          break;
        case 2:
          message.limit = reader.int32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec {
    const message = {
      ...basePipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec,
    } as PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec;
    message.filter =
      object.filter !== undefined && object.filter !== null ? String(object.filter) : '';
    message.limit = object.limit !== undefined && object.limit !== null ? Number(object.limit) : 0;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec): unknown {
    const obj: any = {};
    message.filter !== undefined && (obj.filter = message.filter);
    message.limit !== undefined && (obj.limit = Math.round(message.limit));
    return obj;
  },

  fromPartial<
    I extends Exact<DeepPartial<PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec>, I>,
  >(object: I): PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec {
    const message = {
      ...basePipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec,
    } as PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec;
    message.filter = object.filter ?? '';
    message.limit = object.limit ?? 0;
    return message;
  },
};

const basePipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry: object = { key: '' };

export const PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry = {
  encode(
    message: PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec.encode(
        message.value,
        writer.uint32(18).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry,
    } as PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec.decode(
            reader,
            reader.uint32(),
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry {
    const message = {
      ...basePipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry,
    } as PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<
    I extends Exact<
      DeepPartial<PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry>,
      I
    >,
  >(object: I): PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry {
    const message = {
      ...basePipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry,
    } as PipelineDeploymentConfig_ResolverSpec_OutputArtifactQueriesEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? PipelineDeploymentConfig_ResolverSpec_ArtifactQuerySpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const basePipelineDeploymentConfig_AIPlatformCustomJobSpec: object = {};

export const PipelineDeploymentConfig_AIPlatformCustomJobSpec = {
  encode(
    message: PipelineDeploymentConfig_AIPlatformCustomJobSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.customJob !== undefined) {
      Struct.encode(Struct.wrap(message.customJob), writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): PipelineDeploymentConfig_AIPlatformCustomJobSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_AIPlatformCustomJobSpec,
    } as PipelineDeploymentConfig_AIPlatformCustomJobSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.customJob = Struct.unwrap(Struct.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_AIPlatformCustomJobSpec {
    const message = {
      ...basePipelineDeploymentConfig_AIPlatformCustomJobSpec,
    } as PipelineDeploymentConfig_AIPlatformCustomJobSpec;
    message.customJob = typeof object.customJob === 'object' ? object.customJob : undefined;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_AIPlatformCustomJobSpec): unknown {
    const obj: any = {};
    message.customJob !== undefined && (obj.customJob = message.customJob);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineDeploymentConfig_AIPlatformCustomJobSpec>, I>>(
    object: I,
  ): PipelineDeploymentConfig_AIPlatformCustomJobSpec {
    const message = {
      ...basePipelineDeploymentConfig_AIPlatformCustomJobSpec,
    } as PipelineDeploymentConfig_AIPlatformCustomJobSpec;
    message.customJob = object.customJob ?? undefined;
    return message;
  },
};

const basePipelineDeploymentConfig_ExecutorSpec: object = {};

export const PipelineDeploymentConfig_ExecutorSpec = {
  encode(
    message: PipelineDeploymentConfig_ExecutorSpec,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.container !== undefined) {
      PipelineDeploymentConfig_PipelineContainerSpec.encode(
        message.container,
        writer.uint32(10).fork(),
      ).ldelim();
    }
    if (message.importer !== undefined) {
      PipelineDeploymentConfig_ImporterSpec.encode(
        message.importer,
        writer.uint32(18).fork(),
      ).ldelim();
    }
    if (message.resolver !== undefined) {
      PipelineDeploymentConfig_ResolverSpec.encode(
        message.resolver,
        writer.uint32(26).fork(),
      ).ldelim();
    }
    if (message.customJob !== undefined) {
      PipelineDeploymentConfig_AIPlatformCustomJobSpec.encode(
        message.customJob,
        writer.uint32(34).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineDeploymentConfig_ExecutorSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_ExecutorSpec,
    } as PipelineDeploymentConfig_ExecutorSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.container = PipelineDeploymentConfig_PipelineContainerSpec.decode(
            reader,
            reader.uint32(),
          );
          break;
        case 2:
          message.importer = PipelineDeploymentConfig_ImporterSpec.decode(reader, reader.uint32());
          break;
        case 3:
          message.resolver = PipelineDeploymentConfig_ResolverSpec.decode(reader, reader.uint32());
          break;
        case 4:
          message.customJob = PipelineDeploymentConfig_AIPlatformCustomJobSpec.decode(
            reader,
            reader.uint32(),
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_ExecutorSpec {
    const message = {
      ...basePipelineDeploymentConfig_ExecutorSpec,
    } as PipelineDeploymentConfig_ExecutorSpec;
    message.container =
      object.container !== undefined && object.container !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec.fromJSON(object.container)
        : undefined;
    message.importer =
      object.importer !== undefined && object.importer !== null
        ? PipelineDeploymentConfig_ImporterSpec.fromJSON(object.importer)
        : undefined;
    message.resolver =
      object.resolver !== undefined && object.resolver !== null
        ? PipelineDeploymentConfig_ResolverSpec.fromJSON(object.resolver)
        : undefined;
    message.customJob =
      object.customJob !== undefined && object.customJob !== null
        ? PipelineDeploymentConfig_AIPlatformCustomJobSpec.fromJSON(object.customJob)
        : undefined;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_ExecutorSpec): unknown {
    const obj: any = {};
    message.container !== undefined &&
      (obj.container = message.container
        ? PipelineDeploymentConfig_PipelineContainerSpec.toJSON(message.container)
        : undefined);
    message.importer !== undefined &&
      (obj.importer = message.importer
        ? PipelineDeploymentConfig_ImporterSpec.toJSON(message.importer)
        : undefined);
    message.resolver !== undefined &&
      (obj.resolver = message.resolver
        ? PipelineDeploymentConfig_ResolverSpec.toJSON(message.resolver)
        : undefined);
    message.customJob !== undefined &&
      (obj.customJob = message.customJob
        ? PipelineDeploymentConfig_AIPlatformCustomJobSpec.toJSON(message.customJob)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineDeploymentConfig_ExecutorSpec>, I>>(
    object: I,
  ): PipelineDeploymentConfig_ExecutorSpec {
    const message = {
      ...basePipelineDeploymentConfig_ExecutorSpec,
    } as PipelineDeploymentConfig_ExecutorSpec;
    message.container =
      object.container !== undefined && object.container !== null
        ? PipelineDeploymentConfig_PipelineContainerSpec.fromPartial(object.container)
        : undefined;
    message.importer =
      object.importer !== undefined && object.importer !== null
        ? PipelineDeploymentConfig_ImporterSpec.fromPartial(object.importer)
        : undefined;
    message.resolver =
      object.resolver !== undefined && object.resolver !== null
        ? PipelineDeploymentConfig_ResolverSpec.fromPartial(object.resolver)
        : undefined;
    message.customJob =
      object.customJob !== undefined && object.customJob !== null
        ? PipelineDeploymentConfig_AIPlatformCustomJobSpec.fromPartial(object.customJob)
        : undefined;
    return message;
  },
};

const basePipelineDeploymentConfig_ExecutorsEntry: object = { key: '' };

export const PipelineDeploymentConfig_ExecutorsEntry = {
  encode(
    message: PipelineDeploymentConfig_ExecutorsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      PipelineDeploymentConfig_ExecutorSpec.encode(
        message.value,
        writer.uint32(18).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineDeploymentConfig_ExecutorsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...basePipelineDeploymentConfig_ExecutorsEntry,
    } as PipelineDeploymentConfig_ExecutorsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = PipelineDeploymentConfig_ExecutorSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineDeploymentConfig_ExecutorsEntry {
    const message = {
      ...basePipelineDeploymentConfig_ExecutorsEntry,
    } as PipelineDeploymentConfig_ExecutorsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? PipelineDeploymentConfig_ExecutorSpec.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: PipelineDeploymentConfig_ExecutorsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value
        ? PipelineDeploymentConfig_ExecutorSpec.toJSON(message.value)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineDeploymentConfig_ExecutorsEntry>, I>>(
    object: I,
  ): PipelineDeploymentConfig_ExecutorsEntry {
    const message = {
      ...basePipelineDeploymentConfig_ExecutorsEntry,
    } as PipelineDeploymentConfig_ExecutorsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? PipelineDeploymentConfig_ExecutorSpec.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseValue: object = {};

export const Value = {
  encode(message: Value, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.intValue !== undefined) {
      writer.uint32(8).int64(message.intValue);
    }
    if (message.doubleValue !== undefined) {
      writer.uint32(17).double(message.doubleValue);
    }
    if (message.stringValue !== undefined) {
      writer.uint32(26).string(message.stringValue);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Value {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseValue } as Value;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.intValue = longToNumber(reader.int64() as Long);
          break;
        case 2:
          message.doubleValue = reader.double();
          break;
        case 3:
          message.stringValue = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Value {
    const message = { ...baseValue } as Value;
    message.intValue =
      object.intValue !== undefined && object.intValue !== null
        ? Number(object.intValue)
        : undefined;
    message.doubleValue =
      object.doubleValue !== undefined && object.doubleValue !== null
        ? Number(object.doubleValue)
        : undefined;
    message.stringValue =
      object.stringValue !== undefined && object.stringValue !== null
        ? String(object.stringValue)
        : undefined;
    return message;
  },

  toJSON(message: Value): unknown {
    const obj: any = {};
    message.intValue !== undefined && (obj.intValue = Math.round(message.intValue));
    message.doubleValue !== undefined && (obj.doubleValue = message.doubleValue);
    message.stringValue !== undefined && (obj.stringValue = message.stringValue);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Value>, I>>(object: I): Value {
    const message = { ...baseValue } as Value;
    message.intValue = object.intValue ?? undefined;
    message.doubleValue = object.doubleValue ?? undefined;
    message.stringValue = object.stringValue ?? undefined;
    return message;
  },
};

const baseRuntimeArtifact: object = { name: '', uri: '' };

export const RuntimeArtifact = {
  encode(message: RuntimeArtifact, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== '') {
      writer.uint32(10).string(message.name);
    }
    if (message.type !== undefined) {
      ArtifactTypeSchema.encode(message.type, writer.uint32(18).fork()).ldelim();
    }
    if (message.uri !== '') {
      writer.uint32(26).string(message.uri);
    }
    Object.entries(message.properties).forEach(([key, value]) => {
      RuntimeArtifact_PropertiesEntry.encode(
        { key: key as any, value },
        writer.uint32(34).fork(),
      ).ldelim();
    });
    Object.entries(message.customProperties).forEach(([key, value]) => {
      RuntimeArtifact_CustomPropertiesEntry.encode(
        { key: key as any, value },
        writer.uint32(42).fork(),
      ).ldelim();
    });
    if (message.metadata !== undefined) {
      Struct.encode(Struct.wrap(message.metadata), writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RuntimeArtifact {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRuntimeArtifact } as RuntimeArtifact;
    message.properties = {};
    message.customProperties = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.type = ArtifactTypeSchema.decode(reader, reader.uint32());
          break;
        case 3:
          message.uri = reader.string();
          break;
        case 4:
          const entry4 = RuntimeArtifact_PropertiesEntry.decode(reader, reader.uint32());
          if (entry4.value !== undefined) {
            message.properties[entry4.key] = entry4.value;
          }
          break;
        case 5:
          const entry5 = RuntimeArtifact_CustomPropertiesEntry.decode(reader, reader.uint32());
          if (entry5.value !== undefined) {
            message.customProperties[entry5.key] = entry5.value;
          }
          break;
        case 6:
          message.metadata = Struct.unwrap(Struct.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RuntimeArtifact {
    const message = { ...baseRuntimeArtifact } as RuntimeArtifact;
    message.name = object.name !== undefined && object.name !== null ? String(object.name) : '';
    message.type =
      object.type !== undefined && object.type !== null
        ? ArtifactTypeSchema.fromJSON(object.type)
        : undefined;
    message.uri = object.uri !== undefined && object.uri !== null ? String(object.uri) : '';
    message.properties = Object.entries(object.properties ?? {}).reduce<{ [key: string]: Value }>(
      (acc, [key, value]) => {
        acc[key] = Value.fromJSON(value);
        return acc;
      },
      {},
    );
    message.customProperties = Object.entries(object.customProperties ?? {}).reduce<{
      [key: string]: Value;
    }>((acc, [key, value]) => {
      acc[key] = Value.fromJSON(value);
      return acc;
    }, {});
    message.metadata = typeof object.metadata === 'object' ? object.metadata : undefined;
    return message;
  },

  toJSON(message: RuntimeArtifact): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.type !== undefined &&
      (obj.type = message.type ? ArtifactTypeSchema.toJSON(message.type) : undefined);
    message.uri !== undefined && (obj.uri = message.uri);
    obj.properties = {};
    if (message.properties) {
      Object.entries(message.properties).forEach(([k, v]) => {
        obj.properties[k] = Value.toJSON(v);
      });
    }
    obj.customProperties = {};
    if (message.customProperties) {
      Object.entries(message.customProperties).forEach(([k, v]) => {
        obj.customProperties[k] = Value.toJSON(v);
      });
    }
    message.metadata !== undefined && (obj.metadata = message.metadata);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<RuntimeArtifact>, I>>(object: I): RuntimeArtifact {
    const message = { ...baseRuntimeArtifact } as RuntimeArtifact;
    message.name = object.name ?? '';
    message.type =
      object.type !== undefined && object.type !== null
        ? ArtifactTypeSchema.fromPartial(object.type)
        : undefined;
    message.uri = object.uri ?? '';
    message.properties = Object.entries(object.properties ?? {}).reduce<{ [key: string]: Value }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = Value.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    message.customProperties = Object.entries(object.customProperties ?? {}).reduce<{
      [key: string]: Value;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Value.fromPartial(value);
      }
      return acc;
    }, {});
    message.metadata = object.metadata ?? undefined;
    return message;
  },
};

const baseRuntimeArtifact_PropertiesEntry: object = { key: '' };

export const RuntimeArtifact_PropertiesEntry = {
  encode(
    message: RuntimeArtifact_PropertiesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Value.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RuntimeArtifact_PropertiesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseRuntimeArtifact_PropertiesEntry } as RuntimeArtifact_PropertiesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = Value.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RuntimeArtifact_PropertiesEntry {
    const message = { ...baseRuntimeArtifact_PropertiesEntry } as RuntimeArtifact_PropertiesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: RuntimeArtifact_PropertiesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? Value.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<RuntimeArtifact_PropertiesEntry>, I>>(
    object: I,
  ): RuntimeArtifact_PropertiesEntry {
    const message = { ...baseRuntimeArtifact_PropertiesEntry } as RuntimeArtifact_PropertiesEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseRuntimeArtifact_CustomPropertiesEntry: object = { key: '' };

export const RuntimeArtifact_CustomPropertiesEntry = {
  encode(
    message: RuntimeArtifact_CustomPropertiesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Value.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RuntimeArtifact_CustomPropertiesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseRuntimeArtifact_CustomPropertiesEntry,
    } as RuntimeArtifact_CustomPropertiesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = Value.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RuntimeArtifact_CustomPropertiesEntry {
    const message = {
      ...baseRuntimeArtifact_CustomPropertiesEntry,
    } as RuntimeArtifact_CustomPropertiesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: RuntimeArtifact_CustomPropertiesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? Value.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<RuntimeArtifact_CustomPropertiesEntry>, I>>(
    object: I,
  ): RuntimeArtifact_CustomPropertiesEntry {
    const message = {
      ...baseRuntimeArtifact_CustomPropertiesEntry,
    } as RuntimeArtifact_CustomPropertiesEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseArtifactList: object = {};

export const ArtifactList = {
  encode(message: ArtifactList, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.artifacts) {
      RuntimeArtifact.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ArtifactList {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseArtifactList } as ArtifactList;
    message.artifacts = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.artifacts.push(RuntimeArtifact.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ArtifactList {
    const message = { ...baseArtifactList } as ArtifactList;
    message.artifacts = (object.artifacts ?? []).map((e: any) => RuntimeArtifact.fromJSON(e));
    return message;
  },

  toJSON(message: ArtifactList): unknown {
    const obj: any = {};
    if (message.artifacts) {
      obj.artifacts = message.artifacts.map((e) => (e ? RuntimeArtifact.toJSON(e) : undefined));
    } else {
      obj.artifacts = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ArtifactList>, I>>(object: I): ArtifactList {
    const message = { ...baseArtifactList } as ArtifactList;
    message.artifacts = object.artifacts?.map((e) => RuntimeArtifact.fromPartial(e)) || [];
    return message;
  },
};

const baseExecutorInput: object = {};

export const ExecutorInput = {
  encode(message: ExecutorInput, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.inputs !== undefined) {
      ExecutorInput_Inputs.encode(message.inputs, writer.uint32(10).fork()).ldelim();
    }
    if (message.outputs !== undefined) {
      ExecutorInput_Outputs.encode(message.outputs, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorInput {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseExecutorInput } as ExecutorInput;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.inputs = ExecutorInput_Inputs.decode(reader, reader.uint32());
          break;
        case 2:
          message.outputs = ExecutorInput_Outputs.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorInput {
    const message = { ...baseExecutorInput } as ExecutorInput;
    message.inputs =
      object.inputs !== undefined && object.inputs !== null
        ? ExecutorInput_Inputs.fromJSON(object.inputs)
        : undefined;
    message.outputs =
      object.outputs !== undefined && object.outputs !== null
        ? ExecutorInput_Outputs.fromJSON(object.outputs)
        : undefined;
    return message;
  },

  toJSON(message: ExecutorInput): unknown {
    const obj: any = {};
    message.inputs !== undefined &&
      (obj.inputs = message.inputs ? ExecutorInput_Inputs.toJSON(message.inputs) : undefined);
    message.outputs !== undefined &&
      (obj.outputs = message.outputs ? ExecutorInput_Outputs.toJSON(message.outputs) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorInput>, I>>(object: I): ExecutorInput {
    const message = { ...baseExecutorInput } as ExecutorInput;
    message.inputs =
      object.inputs !== undefined && object.inputs !== null
        ? ExecutorInput_Inputs.fromPartial(object.inputs)
        : undefined;
    message.outputs =
      object.outputs !== undefined && object.outputs !== null
        ? ExecutorInput_Outputs.fromPartial(object.outputs)
        : undefined;
    return message;
  },
};

const baseExecutorInput_Inputs: object = {};

export const ExecutorInput_Inputs = {
  encode(message: ExecutorInput_Inputs, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.parameters).forEach(([key, value]) => {
      ExecutorInput_Inputs_ParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    Object.entries(message.artifacts).forEach(([key, value]) => {
      ExecutorInput_Inputs_ArtifactsEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    Object.entries(message.parameterValues).forEach(([key, value]) => {
      if (value !== undefined) {
        ExecutorInput_Inputs_ParameterValuesEntry.encode(
          { key: key as any, value },
          writer.uint32(26).fork(),
        ).ldelim();
      }
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorInput_Inputs {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseExecutorInput_Inputs } as ExecutorInput_Inputs;
    message.parameters = {};
    message.artifacts = {};
    message.parameterValues = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = ExecutorInput_Inputs_ParametersEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.parameters[entry1.key] = entry1.value;
          }
          break;
        case 2:
          const entry2 = ExecutorInput_Inputs_ArtifactsEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.artifacts[entry2.key] = entry2.value;
          }
          break;
        case 3:
          const entry3 = ExecutorInput_Inputs_ParameterValuesEntry.decode(reader, reader.uint32());
          if (entry3.value !== undefined) {
            message.parameterValues[entry3.key] = entry3.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorInput_Inputs {
    const message = { ...baseExecutorInput_Inputs } as ExecutorInput_Inputs;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{ [key: string]: Value }>(
      (acc, [key, value]) => {
        acc[key] = Value.fromJSON(value);
        return acc;
      },
      {},
    );
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ArtifactList;
    }>((acc, [key, value]) => {
      acc[key] = ArtifactList.fromJSON(value);
      return acc;
    }, {});
    message.parameterValues = Object.entries(object.parameterValues ?? {}).reduce<{
      [key: string]: any | undefined;
    }>((acc, [key, value]) => {
      acc[key] = value as any | undefined;
      return acc;
    }, {});
    return message;
  },

  toJSON(message: ExecutorInput_Inputs): unknown {
    const obj: any = {};
    obj.parameters = {};
    if (message.parameters) {
      Object.entries(message.parameters).forEach(([k, v]) => {
        obj.parameters[k] = Value.toJSON(v);
      });
    }
    obj.artifacts = {};
    if (message.artifacts) {
      Object.entries(message.artifacts).forEach(([k, v]) => {
        obj.artifacts[k] = ArtifactList.toJSON(v);
      });
    }
    obj.parameterValues = {};
    if (message.parameterValues) {
      Object.entries(message.parameterValues).forEach(([k, v]) => {
        obj.parameterValues[k] = v;
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorInput_Inputs>, I>>(
    object: I,
  ): ExecutorInput_Inputs {
    const message = { ...baseExecutorInput_Inputs } as ExecutorInput_Inputs;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{ [key: string]: Value }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = Value.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ArtifactList;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ArtifactList.fromPartial(value);
      }
      return acc;
    }, {});
    message.parameterValues = Object.entries(object.parameterValues ?? {}).reduce<{
      [key: string]: any | undefined;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = value;
      }
      return acc;
    }, {});
    return message;
  },
};

const baseExecutorInput_Inputs_ParametersEntry: object = { key: '' };

export const ExecutorInput_Inputs_ParametersEntry = {
  encode(
    message: ExecutorInput_Inputs_ParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Value.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorInput_Inputs_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseExecutorInput_Inputs_ParametersEntry,
    } as ExecutorInput_Inputs_ParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = Value.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorInput_Inputs_ParametersEntry {
    const message = {
      ...baseExecutorInput_Inputs_ParametersEntry,
    } as ExecutorInput_Inputs_ParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ExecutorInput_Inputs_ParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? Value.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorInput_Inputs_ParametersEntry>, I>>(
    object: I,
  ): ExecutorInput_Inputs_ParametersEntry {
    const message = {
      ...baseExecutorInput_Inputs_ParametersEntry,
    } as ExecutorInput_Inputs_ParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseExecutorInput_Inputs_ArtifactsEntry: object = { key: '' };

export const ExecutorInput_Inputs_ArtifactsEntry = {
  encode(
    message: ExecutorInput_Inputs_ArtifactsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ArtifactList.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorInput_Inputs_ArtifactsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseExecutorInput_Inputs_ArtifactsEntry,
    } as ExecutorInput_Inputs_ArtifactsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ArtifactList.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorInput_Inputs_ArtifactsEntry {
    const message = {
      ...baseExecutorInput_Inputs_ArtifactsEntry,
    } as ExecutorInput_Inputs_ArtifactsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ArtifactList.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ExecutorInput_Inputs_ArtifactsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ArtifactList.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorInput_Inputs_ArtifactsEntry>, I>>(
    object: I,
  ): ExecutorInput_Inputs_ArtifactsEntry {
    const message = {
      ...baseExecutorInput_Inputs_ArtifactsEntry,
    } as ExecutorInput_Inputs_ArtifactsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ArtifactList.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseExecutorInput_Inputs_ParameterValuesEntry: object = { key: '' };

export const ExecutorInput_Inputs_ParameterValuesEntry = {
  encode(
    message: ExecutorInput_Inputs_ParameterValuesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Value1.encode(Value1.wrap(message.value), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): ExecutorInput_Inputs_ParameterValuesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseExecutorInput_Inputs_ParameterValuesEntry,
    } as ExecutorInput_Inputs_ParameterValuesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = Value1.unwrap(Value1.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorInput_Inputs_ParameterValuesEntry {
    const message = {
      ...baseExecutorInput_Inputs_ParameterValuesEntry,
    } as ExecutorInput_Inputs_ParameterValuesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value = object.value;
    return message;
  },

  toJSON(message: ExecutorInput_Inputs_ParameterValuesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorInput_Inputs_ParameterValuesEntry>, I>>(
    object: I,
  ): ExecutorInput_Inputs_ParameterValuesEntry {
    const message = {
      ...baseExecutorInput_Inputs_ParameterValuesEntry,
    } as ExecutorInput_Inputs_ParameterValuesEntry;
    message.key = object.key ?? '';
    message.value = object.value ?? undefined;
    return message;
  },
};

const baseExecutorInput_OutputParameter: object = { outputFile: '' };

export const ExecutorInput_OutputParameter = {
  encode(
    message: ExecutorInput_OutputParameter,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.outputFile !== '') {
      writer.uint32(10).string(message.outputFile);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorInput_OutputParameter {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseExecutorInput_OutputParameter } as ExecutorInput_OutputParameter;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.outputFile = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorInput_OutputParameter {
    const message = { ...baseExecutorInput_OutputParameter } as ExecutorInput_OutputParameter;
    message.outputFile =
      object.outputFile !== undefined && object.outputFile !== null
        ? String(object.outputFile)
        : '';
    return message;
  },

  toJSON(message: ExecutorInput_OutputParameter): unknown {
    const obj: any = {};
    message.outputFile !== undefined && (obj.outputFile = message.outputFile);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorInput_OutputParameter>, I>>(
    object: I,
  ): ExecutorInput_OutputParameter {
    const message = { ...baseExecutorInput_OutputParameter } as ExecutorInput_OutputParameter;
    message.outputFile = object.outputFile ?? '';
    return message;
  },
};

const baseExecutorInput_Outputs: object = { outputFile: '' };

export const ExecutorInput_Outputs = {
  encode(message: ExecutorInput_Outputs, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.parameters).forEach(([key, value]) => {
      ExecutorInput_Outputs_ParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    Object.entries(message.artifacts).forEach(([key, value]) => {
      ExecutorInput_Outputs_ArtifactsEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    if (message.outputFile !== '') {
      writer.uint32(26).string(message.outputFile);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorInput_Outputs {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseExecutorInput_Outputs } as ExecutorInput_Outputs;
    message.parameters = {};
    message.artifacts = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = ExecutorInput_Outputs_ParametersEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.parameters[entry1.key] = entry1.value;
          }
          break;
        case 2:
          const entry2 = ExecutorInput_Outputs_ArtifactsEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.artifacts[entry2.key] = entry2.value;
          }
          break;
        case 3:
          message.outputFile = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorInput_Outputs {
    const message = { ...baseExecutorInput_Outputs } as ExecutorInput_Outputs;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: ExecutorInput_OutputParameter;
    }>((acc, [key, value]) => {
      acc[key] = ExecutorInput_OutputParameter.fromJSON(value);
      return acc;
    }, {});
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ArtifactList;
    }>((acc, [key, value]) => {
      acc[key] = ArtifactList.fromJSON(value);
      return acc;
    }, {});
    message.outputFile =
      object.outputFile !== undefined && object.outputFile !== null
        ? String(object.outputFile)
        : '';
    return message;
  },

  toJSON(message: ExecutorInput_Outputs): unknown {
    const obj: any = {};
    obj.parameters = {};
    if (message.parameters) {
      Object.entries(message.parameters).forEach(([k, v]) => {
        obj.parameters[k] = ExecutorInput_OutputParameter.toJSON(v);
      });
    }
    obj.artifacts = {};
    if (message.artifacts) {
      Object.entries(message.artifacts).forEach(([k, v]) => {
        obj.artifacts[k] = ArtifactList.toJSON(v);
      });
    }
    message.outputFile !== undefined && (obj.outputFile = message.outputFile);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorInput_Outputs>, I>>(
    object: I,
  ): ExecutorInput_Outputs {
    const message = { ...baseExecutorInput_Outputs } as ExecutorInput_Outputs;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{
      [key: string]: ExecutorInput_OutputParameter;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ExecutorInput_OutputParameter.fromPartial(value);
      }
      return acc;
    }, {});
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ArtifactList;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ArtifactList.fromPartial(value);
      }
      return acc;
    }, {});
    message.outputFile = object.outputFile ?? '';
    return message;
  },
};

const baseExecutorInput_Outputs_ParametersEntry: object = { key: '' };

export const ExecutorInput_Outputs_ParametersEntry = {
  encode(
    message: ExecutorInput_Outputs_ParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ExecutorInput_OutputParameter.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorInput_Outputs_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseExecutorInput_Outputs_ParametersEntry,
    } as ExecutorInput_Outputs_ParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ExecutorInput_OutputParameter.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorInput_Outputs_ParametersEntry {
    const message = {
      ...baseExecutorInput_Outputs_ParametersEntry,
    } as ExecutorInput_Outputs_ParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ExecutorInput_OutputParameter.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ExecutorInput_Outputs_ParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ExecutorInput_OutputParameter.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorInput_Outputs_ParametersEntry>, I>>(
    object: I,
  ): ExecutorInput_Outputs_ParametersEntry {
    const message = {
      ...baseExecutorInput_Outputs_ParametersEntry,
    } as ExecutorInput_Outputs_ParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ExecutorInput_OutputParameter.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseExecutorInput_Outputs_ArtifactsEntry: object = { key: '' };

export const ExecutorInput_Outputs_ArtifactsEntry = {
  encode(
    message: ExecutorInput_Outputs_ArtifactsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ArtifactList.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorInput_Outputs_ArtifactsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseExecutorInput_Outputs_ArtifactsEntry,
    } as ExecutorInput_Outputs_ArtifactsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ArtifactList.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorInput_Outputs_ArtifactsEntry {
    const message = {
      ...baseExecutorInput_Outputs_ArtifactsEntry,
    } as ExecutorInput_Outputs_ArtifactsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ArtifactList.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ExecutorInput_Outputs_ArtifactsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ArtifactList.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorInput_Outputs_ArtifactsEntry>, I>>(
    object: I,
  ): ExecutorInput_Outputs_ArtifactsEntry {
    const message = {
      ...baseExecutorInput_Outputs_ArtifactsEntry,
    } as ExecutorInput_Outputs_ArtifactsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ArtifactList.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseExecutorOutput: object = {};

export const ExecutorOutput = {
  encode(message: ExecutorOutput, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.parameters).forEach(([key, value]) => {
      ExecutorOutput_ParametersEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    Object.entries(message.artifacts).forEach(([key, value]) => {
      ExecutorOutput_ArtifactsEntry.encode(
        { key: key as any, value },
        writer.uint32(18).fork(),
      ).ldelim();
    });
    Object.entries(message.parameterValues).forEach(([key, value]) => {
      if (value !== undefined) {
        ExecutorOutput_ParameterValuesEntry.encode(
          { key: key as any, value },
          writer.uint32(26).fork(),
        ).ldelim();
      }
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorOutput {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseExecutorOutput } as ExecutorOutput;
    message.parameters = {};
    message.artifacts = {};
    message.parameterValues = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = ExecutorOutput_ParametersEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.parameters[entry1.key] = entry1.value;
          }
          break;
        case 2:
          const entry2 = ExecutorOutput_ArtifactsEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.artifacts[entry2.key] = entry2.value;
          }
          break;
        case 3:
          const entry3 = ExecutorOutput_ParameterValuesEntry.decode(reader, reader.uint32());
          if (entry3.value !== undefined) {
            message.parameterValues[entry3.key] = entry3.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorOutput {
    const message = { ...baseExecutorOutput } as ExecutorOutput;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{ [key: string]: Value }>(
      (acc, [key, value]) => {
        acc[key] = Value.fromJSON(value);
        return acc;
      },
      {},
    );
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ArtifactList;
    }>((acc, [key, value]) => {
      acc[key] = ArtifactList.fromJSON(value);
      return acc;
    }, {});
    message.parameterValues = Object.entries(object.parameterValues ?? {}).reduce<{
      [key: string]: any | undefined;
    }>((acc, [key, value]) => {
      acc[key] = value as any | undefined;
      return acc;
    }, {});
    return message;
  },

  toJSON(message: ExecutorOutput): unknown {
    const obj: any = {};
    obj.parameters = {};
    if (message.parameters) {
      Object.entries(message.parameters).forEach(([k, v]) => {
        obj.parameters[k] = Value.toJSON(v);
      });
    }
    obj.artifacts = {};
    if (message.artifacts) {
      Object.entries(message.artifacts).forEach(([k, v]) => {
        obj.artifacts[k] = ArtifactList.toJSON(v);
      });
    }
    obj.parameterValues = {};
    if (message.parameterValues) {
      Object.entries(message.parameterValues).forEach(([k, v]) => {
        obj.parameterValues[k] = v;
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorOutput>, I>>(object: I): ExecutorOutput {
    const message = { ...baseExecutorOutput } as ExecutorOutput;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{ [key: string]: Value }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = Value.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    message.artifacts = Object.entries(object.artifacts ?? {}).reduce<{
      [key: string]: ArtifactList;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = ArtifactList.fromPartial(value);
      }
      return acc;
    }, {});
    message.parameterValues = Object.entries(object.parameterValues ?? {}).reduce<{
      [key: string]: any | undefined;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = value;
      }
      return acc;
    }, {});
    return message;
  },
};

const baseExecutorOutput_ParametersEntry: object = { key: '' };

export const ExecutorOutput_ParametersEntry = {
  encode(
    message: ExecutorOutput_ParametersEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Value.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorOutput_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseExecutorOutput_ParametersEntry } as ExecutorOutput_ParametersEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = Value.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorOutput_ParametersEntry {
    const message = { ...baseExecutorOutput_ParametersEntry } as ExecutorOutput_ParametersEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ExecutorOutput_ParametersEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? Value.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorOutput_ParametersEntry>, I>>(
    object: I,
  ): ExecutorOutput_ParametersEntry {
    const message = { ...baseExecutorOutput_ParametersEntry } as ExecutorOutput_ParametersEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? Value.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseExecutorOutput_ArtifactsEntry: object = { key: '' };

export const ExecutorOutput_ArtifactsEntry = {
  encode(
    message: ExecutorOutput_ArtifactsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      ArtifactList.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorOutput_ArtifactsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseExecutorOutput_ArtifactsEntry } as ExecutorOutput_ArtifactsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = ArtifactList.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorOutput_ArtifactsEntry {
    const message = { ...baseExecutorOutput_ArtifactsEntry } as ExecutorOutput_ArtifactsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ArtifactList.fromJSON(object.value)
        : undefined;
    return message;
  },

  toJSON(message: ExecutorOutput_ArtifactsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined &&
      (obj.value = message.value ? ArtifactList.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorOutput_ArtifactsEntry>, I>>(
    object: I,
  ): ExecutorOutput_ArtifactsEntry {
    const message = { ...baseExecutorOutput_ArtifactsEntry } as ExecutorOutput_ArtifactsEntry;
    message.key = object.key ?? '';
    message.value =
      object.value !== undefined && object.value !== null
        ? ArtifactList.fromPartial(object.value)
        : undefined;
    return message;
  },
};

const baseExecutorOutput_ParameterValuesEntry: object = { key: '' };

export const ExecutorOutput_ParameterValuesEntry = {
  encode(
    message: ExecutorOutput_ParameterValuesEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Value1.encode(Value1.wrap(message.value), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecutorOutput_ParameterValuesEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseExecutorOutput_ParameterValuesEntry,
    } as ExecutorOutput_ParameterValuesEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = Value1.unwrap(Value1.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExecutorOutput_ParameterValuesEntry {
    const message = {
      ...baseExecutorOutput_ParameterValuesEntry,
    } as ExecutorOutput_ParameterValuesEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value = object.value;
    return message;
  },

  toJSON(message: ExecutorOutput_ParameterValuesEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExecutorOutput_ParameterValuesEntry>, I>>(
    object: I,
  ): ExecutorOutput_ParameterValuesEntry {
    const message = {
      ...baseExecutorOutput_ParameterValuesEntry,
    } as ExecutorOutput_ParameterValuesEntry;
    message.key = object.key ?? '';
    message.value = object.value ?? undefined;
    return message;
  },
};

const basePipelineTaskFinalStatus: object = {
  state: '',
  pipelineJobUuid: 0,
  pipelineJobName: '',
  pipelineJobResourceName: '',
};

export const PipelineTaskFinalStatus = {
  encode(message: PipelineTaskFinalStatus, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.state !== '') {
      writer.uint32(10).string(message.state);
    }
    if (message.error !== undefined) {
      Status.encode(message.error, writer.uint32(18).fork()).ldelim();
    }
    if (message.pipelineJobUuid !== 0) {
      writer.uint32(24).int64(message.pipelineJobUuid);
    }
    if (message.pipelineJobName !== '') {
      writer.uint32(34).string(message.pipelineJobName);
    }
    if (message.pipelineJobResourceName !== '') {
      writer.uint32(42).string(message.pipelineJobResourceName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineTaskFinalStatus {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineTaskFinalStatus } as PipelineTaskFinalStatus;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.state = reader.string();
          break;
        case 2:
          message.error = Status.decode(reader, reader.uint32());
          break;
        case 3:
          message.pipelineJobUuid = longToNumber(reader.int64() as Long);
          break;
        case 4:
          message.pipelineJobName = reader.string();
          break;
        case 5:
          message.pipelineJobResourceName = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PipelineTaskFinalStatus {
    const message = { ...basePipelineTaskFinalStatus } as PipelineTaskFinalStatus;
    message.state = object.state !== undefined && object.state !== null ? String(object.state) : '';
    message.error =
      object.error !== undefined && object.error !== null
        ? Status.fromJSON(object.error)
        : undefined;
    message.pipelineJobUuid =
      object.pipelineJobUuid !== undefined && object.pipelineJobUuid !== null
        ? Number(object.pipelineJobUuid)
        : 0;
    message.pipelineJobName =
      object.pipelineJobName !== undefined && object.pipelineJobName !== null
        ? String(object.pipelineJobName)
        : '';
    message.pipelineJobResourceName =
      object.pipelineJobResourceName !== undefined && object.pipelineJobResourceName !== null
        ? String(object.pipelineJobResourceName)
        : '';
    return message;
  },

  toJSON(message: PipelineTaskFinalStatus): unknown {
    const obj: any = {};
    message.state !== undefined && (obj.state = message.state);
    message.error !== undefined &&
      (obj.error = message.error ? Status.toJSON(message.error) : undefined);
    message.pipelineJobUuid !== undefined &&
      (obj.pipelineJobUuid = Math.round(message.pipelineJobUuid));
    message.pipelineJobName !== undefined && (obj.pipelineJobName = message.pipelineJobName);
    message.pipelineJobResourceName !== undefined &&
      (obj.pipelineJobResourceName = message.pipelineJobResourceName);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineTaskFinalStatus>, I>>(
    object: I,
  ): PipelineTaskFinalStatus {
    const message = { ...basePipelineTaskFinalStatus } as PipelineTaskFinalStatus;
    message.state = object.state ?? '';
    message.error =
      object.error !== undefined && object.error !== null
        ? Status.fromPartial(object.error)
        : undefined;
    message.pipelineJobUuid = object.pipelineJobUuid ?? 0;
    message.pipelineJobName = object.pipelineJobName ?? '';
    message.pipelineJobResourceName = object.pipelineJobResourceName ?? '';
    return message;
  },
};

const basePipelineStateEnum: object = {};

export const PipelineStateEnum = {
  encode(_: PipelineStateEnum, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PipelineStateEnum {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePipelineStateEnum } as PipelineStateEnum;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): PipelineStateEnum {
    const message = { ...basePipelineStateEnum } as PipelineStateEnum;
    return message;
  },

  toJSON(_: PipelineStateEnum): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PipelineStateEnum>, I>>(_: I): PipelineStateEnum {
    const message = { ...basePipelineStateEnum } as PipelineStateEnum;
    return message;
  },
};

declare var self: any | undefined;
declare var window: any | undefined;
declare var global: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== 'undefined') return globalThis;
  if (typeof self !== 'undefined') return self;
  if (typeof window !== 'undefined') return window;
  if (typeof global !== 'undefined') return global;
  throw 'Unable to locate global object';
})();

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPartial<U>>
  : T extends {}
  ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin
  ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & Record<Exclude<keyof I, KeysOfUnion<P>>, never>;

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error('Value is larger than Number.MAX_SAFE_INTEGER');
  }
  return long.toNumber();
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}
