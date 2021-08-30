import * as jspb from 'google-protobuf'

import * as google_protobuf_any_pb from 'google-protobuf/google/protobuf/any_pb';
import * as google_protobuf_struct_pb from 'google-protobuf/google/protobuf/struct_pb';
import * as google_rpc_status_pb from './google/rpc/status_pb';


export class PipelineJob extends jspb.Message {
  getName(): string;
  setName(value: string): PipelineJob;

  getDisplayName(): string;
  setDisplayName(value: string): PipelineJob;

  getPipelineSpec(): google_protobuf_struct_pb.Struct | undefined;
  setPipelineSpec(value?: google_protobuf_struct_pb.Struct): PipelineJob;
  hasPipelineSpec(): boolean;
  clearPipelineSpec(): PipelineJob;

  getLabelsMap(): jspb.Map<string, string>;
  clearLabelsMap(): PipelineJob;

  getRuntimeConfig(): PipelineJob.RuntimeConfig | undefined;
  setRuntimeConfig(value?: PipelineJob.RuntimeConfig): PipelineJob;
  hasRuntimeConfig(): boolean;
  clearRuntimeConfig(): PipelineJob;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PipelineJob.AsObject;
  static toObject(includeInstance: boolean, msg: PipelineJob): PipelineJob.AsObject;
  static serializeBinaryToWriter(message: PipelineJob, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PipelineJob;
  static deserializeBinaryFromReader(message: PipelineJob, reader: jspb.BinaryReader): PipelineJob;
}

export namespace PipelineJob {
  export type AsObject = {
    name: string,
    displayName: string,
    pipelineSpec?: google_protobuf_struct_pb.Struct.AsObject,
    labelsMap: Array<[string, string]>,
    runtimeConfig?: PipelineJob.RuntimeConfig.AsObject,
  }

  export class RuntimeConfig extends jspb.Message {
    getParametersMap(): jspb.Map<string, Value>;
    clearParametersMap(): RuntimeConfig;

    getGcsOutputDirectory(): string;
    setGcsOutputDirectory(value: string): RuntimeConfig;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): RuntimeConfig.AsObject;
    static toObject(includeInstance: boolean, msg: RuntimeConfig): RuntimeConfig.AsObject;
    static serializeBinaryToWriter(message: RuntimeConfig, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): RuntimeConfig;
    static deserializeBinaryFromReader(message: RuntimeConfig, reader: jspb.BinaryReader): RuntimeConfig;
  }

  export namespace RuntimeConfig {
    export type AsObject = {
      parametersMap: Array<[string, Value.AsObject]>,
      gcsOutputDirectory: string,
    }
  }

}

export class PipelineSpec extends jspb.Message {
  getPipelineInfo(): PipelineInfo | undefined;
  setPipelineInfo(value?: PipelineInfo): PipelineSpec;
  hasPipelineInfo(): boolean;
  clearPipelineInfo(): PipelineSpec;

  getDeploymentSpec(): google_protobuf_struct_pb.Struct | undefined;
  setDeploymentSpec(value?: google_protobuf_struct_pb.Struct): PipelineSpec;
  hasDeploymentSpec(): boolean;
  clearDeploymentSpec(): PipelineSpec;

  getSdkVersion(): string;
  setSdkVersion(value: string): PipelineSpec;

  getSchemaVersion(): string;
  setSchemaVersion(value: string): PipelineSpec;

  getComponentsMap(): jspb.Map<string, ComponentSpec>;
  clearComponentsMap(): PipelineSpec;

  getRoot(): ComponentSpec | undefined;
  setRoot(value?: ComponentSpec): PipelineSpec;
  hasRoot(): boolean;
  clearRoot(): PipelineSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PipelineSpec.AsObject;
  static toObject(includeInstance: boolean, msg: PipelineSpec): PipelineSpec.AsObject;
  static serializeBinaryToWriter(message: PipelineSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PipelineSpec;
  static deserializeBinaryFromReader(message: PipelineSpec, reader: jspb.BinaryReader): PipelineSpec;
}

export namespace PipelineSpec {
  export type AsObject = {
    pipelineInfo?: PipelineInfo.AsObject,
    deploymentSpec?: google_protobuf_struct_pb.Struct.AsObject,
    sdkVersion: string,
    schemaVersion: string,
    componentsMap: Array<[string, ComponentSpec.AsObject]>,
    root?: ComponentSpec.AsObject,
  }

  export class RuntimeParameter extends jspb.Message {
    getType(): PrimitiveType.PrimitiveTypeEnum;
    setType(value: PrimitiveType.PrimitiveTypeEnum): RuntimeParameter;

    getDefaultValue(): Value | undefined;
    setDefaultValue(value?: Value): RuntimeParameter;
    hasDefaultValue(): boolean;
    clearDefaultValue(): RuntimeParameter;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): RuntimeParameter.AsObject;
    static toObject(includeInstance: boolean, msg: RuntimeParameter): RuntimeParameter.AsObject;
    static serializeBinaryToWriter(message: RuntimeParameter, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): RuntimeParameter;
    static deserializeBinaryFromReader(message: RuntimeParameter, reader: jspb.BinaryReader): RuntimeParameter;
  }

  export namespace RuntimeParameter {
    export type AsObject = {
      type: PrimitiveType.PrimitiveTypeEnum,
      defaultValue?: Value.AsObject,
    }
  }

}

export class ComponentSpec extends jspb.Message {
  getInputDefinitions(): ComponentInputsSpec | undefined;
  setInputDefinitions(value?: ComponentInputsSpec): ComponentSpec;
  hasInputDefinitions(): boolean;
  clearInputDefinitions(): ComponentSpec;

  getOutputDefinitions(): ComponentOutputsSpec | undefined;
  setOutputDefinitions(value?: ComponentOutputsSpec): ComponentSpec;
  hasOutputDefinitions(): boolean;
  clearOutputDefinitions(): ComponentSpec;

  getDag(): DagSpec | undefined;
  setDag(value?: DagSpec): ComponentSpec;
  hasDag(): boolean;
  clearDag(): ComponentSpec;

  getExecutorLabel(): string;
  setExecutorLabel(value: string): ComponentSpec;

  getImplementationCase(): ComponentSpec.ImplementationCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ComponentSpec.AsObject;
  static toObject(includeInstance: boolean, msg: ComponentSpec): ComponentSpec.AsObject;
  static serializeBinaryToWriter(message: ComponentSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ComponentSpec;
  static deserializeBinaryFromReader(message: ComponentSpec, reader: jspb.BinaryReader): ComponentSpec;
}

export namespace ComponentSpec {
  export type AsObject = {
    inputDefinitions?: ComponentInputsSpec.AsObject,
    outputDefinitions?: ComponentOutputsSpec.AsObject,
    dag?: DagSpec.AsObject,
    executorLabel: string,
  }

  export enum ImplementationCase { 
    IMPLEMENTATION_NOT_SET = 0,
    DAG = 3,
    EXECUTOR_LABEL = 4,
  }
}

export class DagSpec extends jspb.Message {
  getTasksMap(): jspb.Map<string, PipelineTaskSpec>;
  clearTasksMap(): DagSpec;

  getOutputs(): DagOutputsSpec | undefined;
  setOutputs(value?: DagOutputsSpec): DagSpec;
  hasOutputs(): boolean;
  clearOutputs(): DagSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DagSpec.AsObject;
  static toObject(includeInstance: boolean, msg: DagSpec): DagSpec.AsObject;
  static serializeBinaryToWriter(message: DagSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DagSpec;
  static deserializeBinaryFromReader(message: DagSpec, reader: jspb.BinaryReader): DagSpec;
}

export namespace DagSpec {
  export type AsObject = {
    tasksMap: Array<[string, PipelineTaskSpec.AsObject]>,
    outputs?: DagOutputsSpec.AsObject,
  }
}

export class DagOutputsSpec extends jspb.Message {
  getArtifactsMap(): jspb.Map<string, DagOutputsSpec.DagOutputArtifactSpec>;
  clearArtifactsMap(): DagOutputsSpec;

  getParametersMap(): jspb.Map<string, DagOutputsSpec.DagOutputParameterSpec>;
  clearParametersMap(): DagOutputsSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DagOutputsSpec.AsObject;
  static toObject(includeInstance: boolean, msg: DagOutputsSpec): DagOutputsSpec.AsObject;
  static serializeBinaryToWriter(message: DagOutputsSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DagOutputsSpec;
  static deserializeBinaryFromReader(message: DagOutputsSpec, reader: jspb.BinaryReader): DagOutputsSpec;
}

export namespace DagOutputsSpec {
  export type AsObject = {
    artifactsMap: Array<[string, DagOutputsSpec.DagOutputArtifactSpec.AsObject]>,
    parametersMap: Array<[string, DagOutputsSpec.DagOutputParameterSpec.AsObject]>,
  }

  export class ArtifactSelectorSpec extends jspb.Message {
    getProducerSubtask(): string;
    setProducerSubtask(value: string): ArtifactSelectorSpec;

    getOutputArtifactKey(): string;
    setOutputArtifactKey(value: string): ArtifactSelectorSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ArtifactSelectorSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ArtifactSelectorSpec): ArtifactSelectorSpec.AsObject;
    static serializeBinaryToWriter(message: ArtifactSelectorSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ArtifactSelectorSpec;
    static deserializeBinaryFromReader(message: ArtifactSelectorSpec, reader: jspb.BinaryReader): ArtifactSelectorSpec;
  }

  export namespace ArtifactSelectorSpec {
    export type AsObject = {
      producerSubtask: string,
      outputArtifactKey: string,
    }
  }


  export class DagOutputArtifactSpec extends jspb.Message {
    getArtifactSelectorsList(): Array<DagOutputsSpec.ArtifactSelectorSpec>;
    setArtifactSelectorsList(value: Array<DagOutputsSpec.ArtifactSelectorSpec>): DagOutputArtifactSpec;
    clearArtifactSelectorsList(): DagOutputArtifactSpec;
    addArtifactSelectors(value?: DagOutputsSpec.ArtifactSelectorSpec, index?: number): DagOutputsSpec.ArtifactSelectorSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DagOutputArtifactSpec.AsObject;
    static toObject(includeInstance: boolean, msg: DagOutputArtifactSpec): DagOutputArtifactSpec.AsObject;
    static serializeBinaryToWriter(message: DagOutputArtifactSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DagOutputArtifactSpec;
    static deserializeBinaryFromReader(message: DagOutputArtifactSpec, reader: jspb.BinaryReader): DagOutputArtifactSpec;
  }

  export namespace DagOutputArtifactSpec {
    export type AsObject = {
      artifactSelectorsList: Array<DagOutputsSpec.ArtifactSelectorSpec.AsObject>,
    }
  }


  export class ParameterSelectorSpec extends jspb.Message {
    getProducerSubtask(): string;
    setProducerSubtask(value: string): ParameterSelectorSpec;

    getOutputParameterKey(): string;
    setOutputParameterKey(value: string): ParameterSelectorSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ParameterSelectorSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ParameterSelectorSpec): ParameterSelectorSpec.AsObject;
    static serializeBinaryToWriter(message: ParameterSelectorSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ParameterSelectorSpec;
    static deserializeBinaryFromReader(message: ParameterSelectorSpec, reader: jspb.BinaryReader): ParameterSelectorSpec;
  }

  export namespace ParameterSelectorSpec {
    export type AsObject = {
      producerSubtask: string,
      outputParameterKey: string,
    }
  }


  export class ParameterSelectorsSpec extends jspb.Message {
    getParameterSelectorsList(): Array<DagOutputsSpec.ParameterSelectorSpec>;
    setParameterSelectorsList(value: Array<DagOutputsSpec.ParameterSelectorSpec>): ParameterSelectorsSpec;
    clearParameterSelectorsList(): ParameterSelectorsSpec;
    addParameterSelectors(value?: DagOutputsSpec.ParameterSelectorSpec, index?: number): DagOutputsSpec.ParameterSelectorSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ParameterSelectorsSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ParameterSelectorsSpec): ParameterSelectorsSpec.AsObject;
    static serializeBinaryToWriter(message: ParameterSelectorsSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ParameterSelectorsSpec;
    static deserializeBinaryFromReader(message: ParameterSelectorsSpec, reader: jspb.BinaryReader): ParameterSelectorsSpec;
  }

  export namespace ParameterSelectorsSpec {
    export type AsObject = {
      parameterSelectorsList: Array<DagOutputsSpec.ParameterSelectorSpec.AsObject>,
    }
  }


  export class MapParameterSelectorsSpec extends jspb.Message {
    getMappedParametersMap(): jspb.Map<string, DagOutputsSpec.ParameterSelectorSpec>;
    clearMappedParametersMap(): MapParameterSelectorsSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): MapParameterSelectorsSpec.AsObject;
    static toObject(includeInstance: boolean, msg: MapParameterSelectorsSpec): MapParameterSelectorsSpec.AsObject;
    static serializeBinaryToWriter(message: MapParameterSelectorsSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): MapParameterSelectorsSpec;
    static deserializeBinaryFromReader(message: MapParameterSelectorsSpec, reader: jspb.BinaryReader): MapParameterSelectorsSpec;
  }

  export namespace MapParameterSelectorsSpec {
    export type AsObject = {
      mappedParametersMap: Array<[string, DagOutputsSpec.ParameterSelectorSpec.AsObject]>,
    }
  }


  export class DagOutputParameterSpec extends jspb.Message {
    getValueFromParameter(): DagOutputsSpec.ParameterSelectorSpec | undefined;
    setValueFromParameter(value?: DagOutputsSpec.ParameterSelectorSpec): DagOutputParameterSpec;
    hasValueFromParameter(): boolean;
    clearValueFromParameter(): DagOutputParameterSpec;

    getValueFromOneof(): DagOutputsSpec.ParameterSelectorsSpec | undefined;
    setValueFromOneof(value?: DagOutputsSpec.ParameterSelectorsSpec): DagOutputParameterSpec;
    hasValueFromOneof(): boolean;
    clearValueFromOneof(): DagOutputParameterSpec;

    getKindCase(): DagOutputParameterSpec.KindCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DagOutputParameterSpec.AsObject;
    static toObject(includeInstance: boolean, msg: DagOutputParameterSpec): DagOutputParameterSpec.AsObject;
    static serializeBinaryToWriter(message: DagOutputParameterSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DagOutputParameterSpec;
    static deserializeBinaryFromReader(message: DagOutputParameterSpec, reader: jspb.BinaryReader): DagOutputParameterSpec;
  }

  export namespace DagOutputParameterSpec {
    export type AsObject = {
      valueFromParameter?: DagOutputsSpec.ParameterSelectorSpec.AsObject,
      valueFromOneof?: DagOutputsSpec.ParameterSelectorsSpec.AsObject,
    }

    export enum KindCase { 
      KIND_NOT_SET = 0,
      VALUE_FROM_PARAMETER = 1,
      VALUE_FROM_ONEOF = 2,
    }
  }

}

export class ComponentInputsSpec extends jspb.Message {
  getArtifactsMap(): jspb.Map<string, ComponentInputsSpec.ArtifactSpec>;
  clearArtifactsMap(): ComponentInputsSpec;

  getParametersMap(): jspb.Map<string, ComponentInputsSpec.ParameterSpec>;
  clearParametersMap(): ComponentInputsSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ComponentInputsSpec.AsObject;
  static toObject(includeInstance: boolean, msg: ComponentInputsSpec): ComponentInputsSpec.AsObject;
  static serializeBinaryToWriter(message: ComponentInputsSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ComponentInputsSpec;
  static deserializeBinaryFromReader(message: ComponentInputsSpec, reader: jspb.BinaryReader): ComponentInputsSpec;
}

export namespace ComponentInputsSpec {
  export type AsObject = {
    artifactsMap: Array<[string, ComponentInputsSpec.ArtifactSpec.AsObject]>,
    parametersMap: Array<[string, ComponentInputsSpec.ParameterSpec.AsObject]>,
  }

  export class ArtifactSpec extends jspb.Message {
    getArtifactType(): ArtifactTypeSchema | undefined;
    setArtifactType(value?: ArtifactTypeSchema): ArtifactSpec;
    hasArtifactType(): boolean;
    clearArtifactType(): ArtifactSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ArtifactSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ArtifactSpec): ArtifactSpec.AsObject;
    static serializeBinaryToWriter(message: ArtifactSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ArtifactSpec;
    static deserializeBinaryFromReader(message: ArtifactSpec, reader: jspb.BinaryReader): ArtifactSpec;
  }

  export namespace ArtifactSpec {
    export type AsObject = {
      artifactType?: ArtifactTypeSchema.AsObject,
    }
  }


  export class ParameterSpec extends jspb.Message {
    getType(): PrimitiveType.PrimitiveTypeEnum;
    setType(value: PrimitiveType.PrimitiveTypeEnum): ParameterSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ParameterSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ParameterSpec): ParameterSpec.AsObject;
    static serializeBinaryToWriter(message: ParameterSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ParameterSpec;
    static deserializeBinaryFromReader(message: ParameterSpec, reader: jspb.BinaryReader): ParameterSpec;
  }

  export namespace ParameterSpec {
    export type AsObject = {
      type: PrimitiveType.PrimitiveTypeEnum,
    }
  }

}

export class ComponentOutputsSpec extends jspb.Message {
  getArtifactsMap(): jspb.Map<string, ComponentOutputsSpec.ArtifactSpec>;
  clearArtifactsMap(): ComponentOutputsSpec;

  getParametersMap(): jspb.Map<string, ComponentOutputsSpec.ParameterSpec>;
  clearParametersMap(): ComponentOutputsSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ComponentOutputsSpec.AsObject;
  static toObject(includeInstance: boolean, msg: ComponentOutputsSpec): ComponentOutputsSpec.AsObject;
  static serializeBinaryToWriter(message: ComponentOutputsSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ComponentOutputsSpec;
  static deserializeBinaryFromReader(message: ComponentOutputsSpec, reader: jspb.BinaryReader): ComponentOutputsSpec;
}

export namespace ComponentOutputsSpec {
  export type AsObject = {
    artifactsMap: Array<[string, ComponentOutputsSpec.ArtifactSpec.AsObject]>,
    parametersMap: Array<[string, ComponentOutputsSpec.ParameterSpec.AsObject]>,
  }

  export class ArtifactSpec extends jspb.Message {
    getArtifactType(): ArtifactTypeSchema | undefined;
    setArtifactType(value?: ArtifactTypeSchema): ArtifactSpec;
    hasArtifactType(): boolean;
    clearArtifactType(): ArtifactSpec;

    getPropertiesMap(): jspb.Map<string, ValueOrRuntimeParameter>;
    clearPropertiesMap(): ArtifactSpec;

    getCustomPropertiesMap(): jspb.Map<string, ValueOrRuntimeParameter>;
    clearCustomPropertiesMap(): ArtifactSpec;

    getMetadata(): google_protobuf_struct_pb.Struct | undefined;
    setMetadata(value?: google_protobuf_struct_pb.Struct): ArtifactSpec;
    hasMetadata(): boolean;
    clearMetadata(): ArtifactSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ArtifactSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ArtifactSpec): ArtifactSpec.AsObject;
    static serializeBinaryToWriter(message: ArtifactSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ArtifactSpec;
    static deserializeBinaryFromReader(message: ArtifactSpec, reader: jspb.BinaryReader): ArtifactSpec;
  }

  export namespace ArtifactSpec {
    export type AsObject = {
      artifactType?: ArtifactTypeSchema.AsObject,
      propertiesMap: Array<[string, ValueOrRuntimeParameter.AsObject]>,
      customPropertiesMap: Array<[string, ValueOrRuntimeParameter.AsObject]>,
      metadata?: google_protobuf_struct_pb.Struct.AsObject,
    }
  }


  export class ParameterSpec extends jspb.Message {
    getType(): PrimitiveType.PrimitiveTypeEnum;
    setType(value: PrimitiveType.PrimitiveTypeEnum): ParameterSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ParameterSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ParameterSpec): ParameterSpec.AsObject;
    static serializeBinaryToWriter(message: ParameterSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ParameterSpec;
    static deserializeBinaryFromReader(message: ParameterSpec, reader: jspb.BinaryReader): ParameterSpec;
  }

  export namespace ParameterSpec {
    export type AsObject = {
      type: PrimitiveType.PrimitiveTypeEnum,
    }
  }

}

export class TaskInputsSpec extends jspb.Message {
  getParametersMap(): jspb.Map<string, TaskInputsSpec.InputParameterSpec>;
  clearParametersMap(): TaskInputsSpec;

  getArtifactsMap(): jspb.Map<string, TaskInputsSpec.InputArtifactSpec>;
  clearArtifactsMap(): TaskInputsSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TaskInputsSpec.AsObject;
  static toObject(includeInstance: boolean, msg: TaskInputsSpec): TaskInputsSpec.AsObject;
  static serializeBinaryToWriter(message: TaskInputsSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TaskInputsSpec;
  static deserializeBinaryFromReader(message: TaskInputsSpec, reader: jspb.BinaryReader): TaskInputsSpec;
}

export namespace TaskInputsSpec {
  export type AsObject = {
    parametersMap: Array<[string, TaskInputsSpec.InputParameterSpec.AsObject]>,
    artifactsMap: Array<[string, TaskInputsSpec.InputArtifactSpec.AsObject]>,
  }

  export class InputArtifactSpec extends jspb.Message {
    getTaskOutputArtifact(): TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec | undefined;
    setTaskOutputArtifact(value?: TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec): InputArtifactSpec;
    hasTaskOutputArtifact(): boolean;
    clearTaskOutputArtifact(): InputArtifactSpec;

    getComponentInputArtifact(): string;
    setComponentInputArtifact(value: string): InputArtifactSpec;

    getKindCase(): InputArtifactSpec.KindCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): InputArtifactSpec.AsObject;
    static toObject(includeInstance: boolean, msg: InputArtifactSpec): InputArtifactSpec.AsObject;
    static serializeBinaryToWriter(message: InputArtifactSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): InputArtifactSpec;
    static deserializeBinaryFromReader(message: InputArtifactSpec, reader: jspb.BinaryReader): InputArtifactSpec;
  }

  export namespace InputArtifactSpec {
    export type AsObject = {
      taskOutputArtifact?: TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.AsObject,
      componentInputArtifact: string,
    }

    export class TaskOutputArtifactSpec extends jspb.Message {
      getProducerTask(): string;
      setProducerTask(value: string): TaskOutputArtifactSpec;

      getOutputArtifactKey(): string;
      setOutputArtifactKey(value: string): TaskOutputArtifactSpec;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): TaskOutputArtifactSpec.AsObject;
      static toObject(includeInstance: boolean, msg: TaskOutputArtifactSpec): TaskOutputArtifactSpec.AsObject;
      static serializeBinaryToWriter(message: TaskOutputArtifactSpec, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): TaskOutputArtifactSpec;
      static deserializeBinaryFromReader(message: TaskOutputArtifactSpec, reader: jspb.BinaryReader): TaskOutputArtifactSpec;
    }

    export namespace TaskOutputArtifactSpec {
      export type AsObject = {
        producerTask: string,
        outputArtifactKey: string,
      }
    }


    export enum KindCase { 
      KIND_NOT_SET = 0,
      TASK_OUTPUT_ARTIFACT = 3,
      COMPONENT_INPUT_ARTIFACT = 4,
    }
  }


  export class InputParameterSpec extends jspb.Message {
    getTaskOutputParameter(): TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec | undefined;
    setTaskOutputParameter(value?: TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec): InputParameterSpec;
    hasTaskOutputParameter(): boolean;
    clearTaskOutputParameter(): InputParameterSpec;

    getRuntimeValue(): ValueOrRuntimeParameter | undefined;
    setRuntimeValue(value?: ValueOrRuntimeParameter): InputParameterSpec;
    hasRuntimeValue(): boolean;
    clearRuntimeValue(): InputParameterSpec;

    getComponentInputParameter(): string;
    setComponentInputParameter(value: string): InputParameterSpec;

    getTaskFinalStatus(): TaskInputsSpec.InputParameterSpec.TaskFinalStatus | undefined;
    setTaskFinalStatus(value?: TaskInputsSpec.InputParameterSpec.TaskFinalStatus): InputParameterSpec;
    hasTaskFinalStatus(): boolean;
    clearTaskFinalStatus(): InputParameterSpec;

    getParameterExpressionSelector(): string;
    setParameterExpressionSelector(value: string): InputParameterSpec;

    getKindCase(): InputParameterSpec.KindCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): InputParameterSpec.AsObject;
    static toObject(includeInstance: boolean, msg: InputParameterSpec): InputParameterSpec.AsObject;
    static serializeBinaryToWriter(message: InputParameterSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): InputParameterSpec;
    static deserializeBinaryFromReader(message: InputParameterSpec, reader: jspb.BinaryReader): InputParameterSpec;
  }

  export namespace InputParameterSpec {
    export type AsObject = {
      taskOutputParameter?: TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.AsObject,
      runtimeValue?: ValueOrRuntimeParameter.AsObject,
      componentInputParameter: string,
      taskFinalStatus?: TaskInputsSpec.InputParameterSpec.TaskFinalStatus.AsObject,
      parameterExpressionSelector: string,
    }

    export class TaskOutputParameterSpec extends jspb.Message {
      getProducerTask(): string;
      setProducerTask(value: string): TaskOutputParameterSpec;

      getOutputParameterKey(): string;
      setOutputParameterKey(value: string): TaskOutputParameterSpec;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): TaskOutputParameterSpec.AsObject;
      static toObject(includeInstance: boolean, msg: TaskOutputParameterSpec): TaskOutputParameterSpec.AsObject;
      static serializeBinaryToWriter(message: TaskOutputParameterSpec, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): TaskOutputParameterSpec;
      static deserializeBinaryFromReader(message: TaskOutputParameterSpec, reader: jspb.BinaryReader): TaskOutputParameterSpec;
    }

    export namespace TaskOutputParameterSpec {
      export type AsObject = {
        producerTask: string,
        outputParameterKey: string,
      }
    }


    export class TaskFinalStatus extends jspb.Message {
      getProducerTask(): string;
      setProducerTask(value: string): TaskFinalStatus;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): TaskFinalStatus.AsObject;
      static toObject(includeInstance: boolean, msg: TaskFinalStatus): TaskFinalStatus.AsObject;
      static serializeBinaryToWriter(message: TaskFinalStatus, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): TaskFinalStatus;
      static deserializeBinaryFromReader(message: TaskFinalStatus, reader: jspb.BinaryReader): TaskFinalStatus;
    }

    export namespace TaskFinalStatus {
      export type AsObject = {
        producerTask: string,
      }
    }


    export enum KindCase { 
      KIND_NOT_SET = 0,
      TASK_OUTPUT_PARAMETER = 1,
      RUNTIME_VALUE = 2,
      COMPONENT_INPUT_PARAMETER = 3,
      TASK_FINAL_STATUS = 5,
    }
  }

}

export class TaskOutputsSpec extends jspb.Message {
  getParametersMap(): jspb.Map<string, TaskOutputsSpec.OutputParameterSpec>;
  clearParametersMap(): TaskOutputsSpec;

  getArtifactsMap(): jspb.Map<string, TaskOutputsSpec.OutputArtifactSpec>;
  clearArtifactsMap(): TaskOutputsSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TaskOutputsSpec.AsObject;
  static toObject(includeInstance: boolean, msg: TaskOutputsSpec): TaskOutputsSpec.AsObject;
  static serializeBinaryToWriter(message: TaskOutputsSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TaskOutputsSpec;
  static deserializeBinaryFromReader(message: TaskOutputsSpec, reader: jspb.BinaryReader): TaskOutputsSpec;
}

export namespace TaskOutputsSpec {
  export type AsObject = {
    parametersMap: Array<[string, TaskOutputsSpec.OutputParameterSpec.AsObject]>,
    artifactsMap: Array<[string, TaskOutputsSpec.OutputArtifactSpec.AsObject]>,
  }

  export class OutputArtifactSpec extends jspb.Message {
    getArtifactType(): ArtifactTypeSchema | undefined;
    setArtifactType(value?: ArtifactTypeSchema): OutputArtifactSpec;
    hasArtifactType(): boolean;
    clearArtifactType(): OutputArtifactSpec;

    getPropertiesMap(): jspb.Map<string, ValueOrRuntimeParameter>;
    clearPropertiesMap(): OutputArtifactSpec;

    getCustomPropertiesMap(): jspb.Map<string, ValueOrRuntimeParameter>;
    clearCustomPropertiesMap(): OutputArtifactSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): OutputArtifactSpec.AsObject;
    static toObject(includeInstance: boolean, msg: OutputArtifactSpec): OutputArtifactSpec.AsObject;
    static serializeBinaryToWriter(message: OutputArtifactSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): OutputArtifactSpec;
    static deserializeBinaryFromReader(message: OutputArtifactSpec, reader: jspb.BinaryReader): OutputArtifactSpec;
  }

  export namespace OutputArtifactSpec {
    export type AsObject = {
      artifactType?: ArtifactTypeSchema.AsObject,
      propertiesMap: Array<[string, ValueOrRuntimeParameter.AsObject]>,
      customPropertiesMap: Array<[string, ValueOrRuntimeParameter.AsObject]>,
    }
  }


  export class OutputParameterSpec extends jspb.Message {
    getType(): PrimitiveType.PrimitiveTypeEnum;
    setType(value: PrimitiveType.PrimitiveTypeEnum): OutputParameterSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): OutputParameterSpec.AsObject;
    static toObject(includeInstance: boolean, msg: OutputParameterSpec): OutputParameterSpec.AsObject;
    static serializeBinaryToWriter(message: OutputParameterSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): OutputParameterSpec;
    static deserializeBinaryFromReader(message: OutputParameterSpec, reader: jspb.BinaryReader): OutputParameterSpec;
  }

  export namespace OutputParameterSpec {
    export type AsObject = {
      type: PrimitiveType.PrimitiveTypeEnum,
    }
  }

}

export class PrimitiveType extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PrimitiveType.AsObject;
  static toObject(includeInstance: boolean, msg: PrimitiveType): PrimitiveType.AsObject;
  static serializeBinaryToWriter(message: PrimitiveType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PrimitiveType;
  static deserializeBinaryFromReader(message: PrimitiveType, reader: jspb.BinaryReader): PrimitiveType;
}

export namespace PrimitiveType {
  export type AsObject = {
  }

  export enum PrimitiveTypeEnum { 
    PRIMITIVE_TYPE_UNSPECIFIED = 0,
    INT = 1,
    DOUBLE = 2,
    STRING = 3,
  }
}

export class PipelineTaskSpec extends jspb.Message {
  getTaskInfo(): PipelineTaskInfo | undefined;
  setTaskInfo(value?: PipelineTaskInfo): PipelineTaskSpec;
  hasTaskInfo(): boolean;
  clearTaskInfo(): PipelineTaskSpec;

  getInputs(): TaskInputsSpec | undefined;
  setInputs(value?: TaskInputsSpec): PipelineTaskSpec;
  hasInputs(): boolean;
  clearInputs(): PipelineTaskSpec;

  getDependentTasksList(): Array<string>;
  setDependentTasksList(value: Array<string>): PipelineTaskSpec;
  clearDependentTasksList(): PipelineTaskSpec;
  addDependentTasks(value: string, index?: number): PipelineTaskSpec;

  getCachingOptions(): PipelineTaskSpec.CachingOptions | undefined;
  setCachingOptions(value?: PipelineTaskSpec.CachingOptions): PipelineTaskSpec;
  hasCachingOptions(): boolean;
  clearCachingOptions(): PipelineTaskSpec;

  getComponentRef(): ComponentRef | undefined;
  setComponentRef(value?: ComponentRef): PipelineTaskSpec;
  hasComponentRef(): boolean;
  clearComponentRef(): PipelineTaskSpec;

  getTriggerPolicy(): PipelineTaskSpec.TriggerPolicy | undefined;
  setTriggerPolicy(value?: PipelineTaskSpec.TriggerPolicy): PipelineTaskSpec;
  hasTriggerPolicy(): boolean;
  clearTriggerPolicy(): PipelineTaskSpec;

  getArtifactIterator(): ArtifactIteratorSpec | undefined;
  setArtifactIterator(value?: ArtifactIteratorSpec): PipelineTaskSpec;
  hasArtifactIterator(): boolean;
  clearArtifactIterator(): PipelineTaskSpec;

  getParameterIterator(): ParameterIteratorSpec | undefined;
  setParameterIterator(value?: ParameterIteratorSpec): PipelineTaskSpec;
  hasParameterIterator(): boolean;
  clearParameterIterator(): PipelineTaskSpec;

  getIteratorCase(): PipelineTaskSpec.IteratorCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PipelineTaskSpec.AsObject;
  static toObject(includeInstance: boolean, msg: PipelineTaskSpec): PipelineTaskSpec.AsObject;
  static serializeBinaryToWriter(message: PipelineTaskSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PipelineTaskSpec;
  static deserializeBinaryFromReader(message: PipelineTaskSpec, reader: jspb.BinaryReader): PipelineTaskSpec;
}

export namespace PipelineTaskSpec {
  export type AsObject = {
    taskInfo?: PipelineTaskInfo.AsObject,
    inputs?: TaskInputsSpec.AsObject,
    dependentTasksList: Array<string>,
    cachingOptions?: PipelineTaskSpec.CachingOptions.AsObject,
    componentRef?: ComponentRef.AsObject,
    triggerPolicy?: PipelineTaskSpec.TriggerPolicy.AsObject,
    artifactIterator?: ArtifactIteratorSpec.AsObject,
    parameterIterator?: ParameterIteratorSpec.AsObject,
  }

  export class CachingOptions extends jspb.Message {
    getEnableCache(): boolean;
    setEnableCache(value: boolean): CachingOptions;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CachingOptions.AsObject;
    static toObject(includeInstance: boolean, msg: CachingOptions): CachingOptions.AsObject;
    static serializeBinaryToWriter(message: CachingOptions, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CachingOptions;
    static deserializeBinaryFromReader(message: CachingOptions, reader: jspb.BinaryReader): CachingOptions;
  }

  export namespace CachingOptions {
    export type AsObject = {
      enableCache: boolean,
    }
  }


  export class TriggerPolicy extends jspb.Message {
    getCondition(): string;
    setCondition(value: string): TriggerPolicy;

    getStrategy(): PipelineTaskSpec.TriggerPolicy.TriggerStrategy;
    setStrategy(value: PipelineTaskSpec.TriggerPolicy.TriggerStrategy): TriggerPolicy;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): TriggerPolicy.AsObject;
    static toObject(includeInstance: boolean, msg: TriggerPolicy): TriggerPolicy.AsObject;
    static serializeBinaryToWriter(message: TriggerPolicy, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): TriggerPolicy;
    static deserializeBinaryFromReader(message: TriggerPolicy, reader: jspb.BinaryReader): TriggerPolicy;
  }

  export namespace TriggerPolicy {
    export type AsObject = {
      condition: string,
      strategy: PipelineTaskSpec.TriggerPolicy.TriggerStrategy,
    }

    export enum TriggerStrategy { 
      TRIGGER_STRATEGY_UNSPECIFIED = 0,
      ALL_UPSTREAM_TASKS_SUCCEEDED = 1,
      ALL_UPSTREAM_TASKS_COMPLETED = 2,
    }
  }


  export enum IteratorCase { 
    ITERATOR_NOT_SET = 0,
    ARTIFACT_ITERATOR = 9,
    PARAMETER_ITERATOR = 10,
  }
}

export class ArtifactIteratorSpec extends jspb.Message {
  getItems(): ArtifactIteratorSpec.ItemsSpec | undefined;
  setItems(value?: ArtifactIteratorSpec.ItemsSpec): ArtifactIteratorSpec;
  hasItems(): boolean;
  clearItems(): ArtifactIteratorSpec;

  getItemInput(): string;
  setItemInput(value: string): ArtifactIteratorSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactIteratorSpec.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactIteratorSpec): ArtifactIteratorSpec.AsObject;
  static serializeBinaryToWriter(message: ArtifactIteratorSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactIteratorSpec;
  static deserializeBinaryFromReader(message: ArtifactIteratorSpec, reader: jspb.BinaryReader): ArtifactIteratorSpec;
}

export namespace ArtifactIteratorSpec {
  export type AsObject = {
    items?: ArtifactIteratorSpec.ItemsSpec.AsObject,
    itemInput: string,
  }

  export class ItemsSpec extends jspb.Message {
    getInputArtifact(): string;
    setInputArtifact(value: string): ItemsSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ItemsSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ItemsSpec): ItemsSpec.AsObject;
    static serializeBinaryToWriter(message: ItemsSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ItemsSpec;
    static deserializeBinaryFromReader(message: ItemsSpec, reader: jspb.BinaryReader): ItemsSpec;
  }

  export namespace ItemsSpec {
    export type AsObject = {
      inputArtifact: string,
    }
  }

}

export class ParameterIteratorSpec extends jspb.Message {
  getItems(): ParameterIteratorSpec.ItemsSpec | undefined;
  setItems(value?: ParameterIteratorSpec.ItemsSpec): ParameterIteratorSpec;
  hasItems(): boolean;
  clearItems(): ParameterIteratorSpec;

  getItemInput(): string;
  setItemInput(value: string): ParameterIteratorSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ParameterIteratorSpec.AsObject;
  static toObject(includeInstance: boolean, msg: ParameterIteratorSpec): ParameterIteratorSpec.AsObject;
  static serializeBinaryToWriter(message: ParameterIteratorSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ParameterIteratorSpec;
  static deserializeBinaryFromReader(message: ParameterIteratorSpec, reader: jspb.BinaryReader): ParameterIteratorSpec;
}

export namespace ParameterIteratorSpec {
  export type AsObject = {
    items?: ParameterIteratorSpec.ItemsSpec.AsObject,
    itemInput: string,
  }

  export class ItemsSpec extends jspb.Message {
    getRaw(): string;
    setRaw(value: string): ItemsSpec;

    getInputParameter(): string;
    setInputParameter(value: string): ItemsSpec;

    getKindCase(): ItemsSpec.KindCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ItemsSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ItemsSpec): ItemsSpec.AsObject;
    static serializeBinaryToWriter(message: ItemsSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ItemsSpec;
    static deserializeBinaryFromReader(message: ItemsSpec, reader: jspb.BinaryReader): ItemsSpec;
  }

  export namespace ItemsSpec {
    export type AsObject = {
      raw: string,
      inputParameter: string,
    }

    export enum KindCase { 
      KIND_NOT_SET = 0,
      RAW = 1,
      INPUT_PARAMETER = 2,
    }
  }

}

export class ComponentRef extends jspb.Message {
  getName(): string;
  setName(value: string): ComponentRef;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ComponentRef.AsObject;
  static toObject(includeInstance: boolean, msg: ComponentRef): ComponentRef.AsObject;
  static serializeBinaryToWriter(message: ComponentRef, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ComponentRef;
  static deserializeBinaryFromReader(message: ComponentRef, reader: jspb.BinaryReader): ComponentRef;
}

export namespace ComponentRef {
  export type AsObject = {
    name: string,
  }
}

export class PipelineInfo extends jspb.Message {
  getName(): string;
  setName(value: string): PipelineInfo;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PipelineInfo.AsObject;
  static toObject(includeInstance: boolean, msg: PipelineInfo): PipelineInfo.AsObject;
  static serializeBinaryToWriter(message: PipelineInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PipelineInfo;
  static deserializeBinaryFromReader(message: PipelineInfo, reader: jspb.BinaryReader): PipelineInfo;
}

export namespace PipelineInfo {
  export type AsObject = {
    name: string,
  }
}

export class ArtifactTypeSchema extends jspb.Message {
  getSchemaTitle(): string;
  setSchemaTitle(value: string): ArtifactTypeSchema;

  getSchemaUri(): string;
  setSchemaUri(value: string): ArtifactTypeSchema;

  getInstanceSchema(): string;
  setInstanceSchema(value: string): ArtifactTypeSchema;

  getSchemaVersion(): string;
  setSchemaVersion(value: string): ArtifactTypeSchema;

  getKindCase(): ArtifactTypeSchema.KindCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactTypeSchema.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactTypeSchema): ArtifactTypeSchema.AsObject;
  static serializeBinaryToWriter(message: ArtifactTypeSchema, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactTypeSchema;
  static deserializeBinaryFromReader(message: ArtifactTypeSchema, reader: jspb.BinaryReader): ArtifactTypeSchema;
}

export namespace ArtifactTypeSchema {
  export type AsObject = {
    schemaTitle: string,
    schemaUri: string,
    instanceSchema: string,
    schemaVersion: string,
  }

  export enum KindCase { 
    KIND_NOT_SET = 0,
    SCHEMA_TITLE = 1,
    SCHEMA_URI = 2,
    INSTANCE_SCHEMA = 3,
  }
}

export class PipelineTaskInfo extends jspb.Message {
  getName(): string;
  setName(value: string): PipelineTaskInfo;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PipelineTaskInfo.AsObject;
  static toObject(includeInstance: boolean, msg: PipelineTaskInfo): PipelineTaskInfo.AsObject;
  static serializeBinaryToWriter(message: PipelineTaskInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PipelineTaskInfo;
  static deserializeBinaryFromReader(message: PipelineTaskInfo, reader: jspb.BinaryReader): PipelineTaskInfo;
}

export namespace PipelineTaskInfo {
  export type AsObject = {
    name: string,
  }
}

export class ValueOrRuntimeParameter extends jspb.Message {
  getConstantValue(): Value | undefined;
  setConstantValue(value?: Value): ValueOrRuntimeParameter;
  hasConstantValue(): boolean;
  clearConstantValue(): ValueOrRuntimeParameter;

  getRuntimeParameter(): string;
  setRuntimeParameter(value: string): ValueOrRuntimeParameter;

  getValueCase(): ValueOrRuntimeParameter.ValueCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ValueOrRuntimeParameter.AsObject;
  static toObject(includeInstance: boolean, msg: ValueOrRuntimeParameter): ValueOrRuntimeParameter.AsObject;
  static serializeBinaryToWriter(message: ValueOrRuntimeParameter, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ValueOrRuntimeParameter;
  static deserializeBinaryFromReader(message: ValueOrRuntimeParameter, reader: jspb.BinaryReader): ValueOrRuntimeParameter;
}

export namespace ValueOrRuntimeParameter {
  export type AsObject = {
    constantValue?: Value.AsObject,
    runtimeParameter: string,
  }

  export enum ValueCase { 
    VALUE_NOT_SET = 0,
    CONSTANT_VALUE = 1,
    RUNTIME_PARAMETER = 2,
  }
}

export class PipelineDeploymentConfig extends jspb.Message {
  getExecutorsMap(): jspb.Map<string, PipelineDeploymentConfig.ExecutorSpec>;
  clearExecutorsMap(): PipelineDeploymentConfig;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PipelineDeploymentConfig.AsObject;
  static toObject(includeInstance: boolean, msg: PipelineDeploymentConfig): PipelineDeploymentConfig.AsObject;
  static serializeBinaryToWriter(message: PipelineDeploymentConfig, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PipelineDeploymentConfig;
  static deserializeBinaryFromReader(message: PipelineDeploymentConfig, reader: jspb.BinaryReader): PipelineDeploymentConfig;
}

export namespace PipelineDeploymentConfig {
  export type AsObject = {
    executorsMap: Array<[string, PipelineDeploymentConfig.ExecutorSpec.AsObject]>,
  }

  export class PipelineContainerSpec extends jspb.Message {
    getImage(): string;
    setImage(value: string): PipelineContainerSpec;

    getCommandList(): Array<string>;
    setCommandList(value: Array<string>): PipelineContainerSpec;
    clearCommandList(): PipelineContainerSpec;
    addCommand(value: string, index?: number): PipelineContainerSpec;

    getArgsList(): Array<string>;
    setArgsList(value: Array<string>): PipelineContainerSpec;
    clearArgsList(): PipelineContainerSpec;
    addArgs(value: string, index?: number): PipelineContainerSpec;

    getLifecycle(): PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle | undefined;
    setLifecycle(value?: PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle): PipelineContainerSpec;
    hasLifecycle(): boolean;
    clearLifecycle(): PipelineContainerSpec;

    getResources(): PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec | undefined;
    setResources(value?: PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec): PipelineContainerSpec;
    hasResources(): boolean;
    clearResources(): PipelineContainerSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): PipelineContainerSpec.AsObject;
    static toObject(includeInstance: boolean, msg: PipelineContainerSpec): PipelineContainerSpec.AsObject;
    static serializeBinaryToWriter(message: PipelineContainerSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): PipelineContainerSpec;
    static deserializeBinaryFromReader(message: PipelineContainerSpec, reader: jspb.BinaryReader): PipelineContainerSpec;
  }

  export namespace PipelineContainerSpec {
    export type AsObject = {
      image: string,
      commandList: Array<string>,
      argsList: Array<string>,
      lifecycle?: PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.AsObject,
      resources?: PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AsObject,
    }

    export class Lifecycle extends jspb.Message {
      getPreCacheCheck(): PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec | undefined;
      setPreCacheCheck(value?: PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec): Lifecycle;
      hasPreCacheCheck(): boolean;
      clearPreCacheCheck(): Lifecycle;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): Lifecycle.AsObject;
      static toObject(includeInstance: boolean, msg: Lifecycle): Lifecycle.AsObject;
      static serializeBinaryToWriter(message: Lifecycle, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): Lifecycle;
      static deserializeBinaryFromReader(message: Lifecycle, reader: jspb.BinaryReader): Lifecycle;
    }

    export namespace Lifecycle {
      export type AsObject = {
        preCacheCheck?: PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.AsObject,
      }

      export class Exec extends jspb.Message {
        getCommandList(): Array<string>;
        setCommandList(value: Array<string>): Exec;
        clearCommandList(): Exec;
        addCommand(value: string, index?: number): Exec;

        getArgsList(): Array<string>;
        setArgsList(value: Array<string>): Exec;
        clearArgsList(): Exec;
        addArgs(value: string, index?: number): Exec;

        serializeBinary(): Uint8Array;
        toObject(includeInstance?: boolean): Exec.AsObject;
        static toObject(includeInstance: boolean, msg: Exec): Exec.AsObject;
        static serializeBinaryToWriter(message: Exec, writer: jspb.BinaryWriter): void;
        static deserializeBinary(bytes: Uint8Array): Exec;
        static deserializeBinaryFromReader(message: Exec, reader: jspb.BinaryReader): Exec;
      }

      export namespace Exec {
        export type AsObject = {
          commandList: Array<string>,
          argsList: Array<string>,
        }
      }

    }


    export class ResourceSpec extends jspb.Message {
      getCpuLimit(): number;
      setCpuLimit(value: number): ResourceSpec;

      getMemoryLimit(): number;
      setMemoryLimit(value: number): ResourceSpec;

      getAccelerator(): PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig | undefined;
      setAccelerator(value?: PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig): ResourceSpec;
      hasAccelerator(): boolean;
      clearAccelerator(): ResourceSpec;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): ResourceSpec.AsObject;
      static toObject(includeInstance: boolean, msg: ResourceSpec): ResourceSpec.AsObject;
      static serializeBinaryToWriter(message: ResourceSpec, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): ResourceSpec;
      static deserializeBinaryFromReader(message: ResourceSpec, reader: jspb.BinaryReader): ResourceSpec;
    }

    export namespace ResourceSpec {
      export type AsObject = {
        cpuLimit: number,
        memoryLimit: number,
        accelerator?: PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.AsObject,
      }

      export class AcceleratorConfig extends jspb.Message {
        getType(): string;
        setType(value: string): AcceleratorConfig;

        getCount(): number;
        setCount(value: number): AcceleratorConfig;

        serializeBinary(): Uint8Array;
        toObject(includeInstance?: boolean): AcceleratorConfig.AsObject;
        static toObject(includeInstance: boolean, msg: AcceleratorConfig): AcceleratorConfig.AsObject;
        static serializeBinaryToWriter(message: AcceleratorConfig, writer: jspb.BinaryWriter): void;
        static deserializeBinary(bytes: Uint8Array): AcceleratorConfig;
        static deserializeBinaryFromReader(message: AcceleratorConfig, reader: jspb.BinaryReader): AcceleratorConfig;
      }

      export namespace AcceleratorConfig {
        export type AsObject = {
          type: string,
          count: number,
        }
      }

    }

  }


  export class ImporterSpec extends jspb.Message {
    getArtifactUri(): ValueOrRuntimeParameter | undefined;
    setArtifactUri(value?: ValueOrRuntimeParameter): ImporterSpec;
    hasArtifactUri(): boolean;
    clearArtifactUri(): ImporterSpec;

    getTypeSchema(): ArtifactTypeSchema | undefined;
    setTypeSchema(value?: ArtifactTypeSchema): ImporterSpec;
    hasTypeSchema(): boolean;
    clearTypeSchema(): ImporterSpec;

    getPropertiesMap(): jspb.Map<string, ValueOrRuntimeParameter>;
    clearPropertiesMap(): ImporterSpec;

    getCustomPropertiesMap(): jspb.Map<string, ValueOrRuntimeParameter>;
    clearCustomPropertiesMap(): ImporterSpec;

    getMetadata(): google_protobuf_struct_pb.Struct | undefined;
    setMetadata(value?: google_protobuf_struct_pb.Struct): ImporterSpec;
    hasMetadata(): boolean;
    clearMetadata(): ImporterSpec;

    getReimport(): boolean;
    setReimport(value: boolean): ImporterSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ImporterSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ImporterSpec): ImporterSpec.AsObject;
    static serializeBinaryToWriter(message: ImporterSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ImporterSpec;
    static deserializeBinaryFromReader(message: ImporterSpec, reader: jspb.BinaryReader): ImporterSpec;
  }

  export namespace ImporterSpec {
    export type AsObject = {
      artifactUri?: ValueOrRuntimeParameter.AsObject,
      typeSchema?: ArtifactTypeSchema.AsObject,
      propertiesMap: Array<[string, ValueOrRuntimeParameter.AsObject]>,
      customPropertiesMap: Array<[string, ValueOrRuntimeParameter.AsObject]>,
      metadata?: google_protobuf_struct_pb.Struct.AsObject,
      reimport: boolean,
    }
  }


  export class ResolverSpec extends jspb.Message {
    getOutputArtifactQueriesMap(): jspb.Map<string, PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec>;
    clearOutputArtifactQueriesMap(): ResolverSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ResolverSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ResolverSpec): ResolverSpec.AsObject;
    static serializeBinaryToWriter(message: ResolverSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ResolverSpec;
    static deserializeBinaryFromReader(message: ResolverSpec, reader: jspb.BinaryReader): ResolverSpec;
  }

  export namespace ResolverSpec {
    export type AsObject = {
      outputArtifactQueriesMap: Array<[string, PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.AsObject]>,
    }

    export class ArtifactQuerySpec extends jspb.Message {
      getFilter(): string;
      setFilter(value: string): ArtifactQuerySpec;

      getLimit(): number;
      setLimit(value: number): ArtifactQuerySpec;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): ArtifactQuerySpec.AsObject;
      static toObject(includeInstance: boolean, msg: ArtifactQuerySpec): ArtifactQuerySpec.AsObject;
      static serializeBinaryToWriter(message: ArtifactQuerySpec, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): ArtifactQuerySpec;
      static deserializeBinaryFromReader(message: ArtifactQuerySpec, reader: jspb.BinaryReader): ArtifactQuerySpec;
    }

    export namespace ArtifactQuerySpec {
      export type AsObject = {
        filter: string,
        limit: number,
      }
    }

  }


  export class AIPlatformCustomJobSpec extends jspb.Message {
    getCustomJob(): google_protobuf_struct_pb.Struct | undefined;
    setCustomJob(value?: google_protobuf_struct_pb.Struct): AIPlatformCustomJobSpec;
    hasCustomJob(): boolean;
    clearCustomJob(): AIPlatformCustomJobSpec;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): AIPlatformCustomJobSpec.AsObject;
    static toObject(includeInstance: boolean, msg: AIPlatformCustomJobSpec): AIPlatformCustomJobSpec.AsObject;
    static serializeBinaryToWriter(message: AIPlatformCustomJobSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): AIPlatformCustomJobSpec;
    static deserializeBinaryFromReader(message: AIPlatformCustomJobSpec, reader: jspb.BinaryReader): AIPlatformCustomJobSpec;
  }

  export namespace AIPlatformCustomJobSpec {
    export type AsObject = {
      customJob?: google_protobuf_struct_pb.Struct.AsObject,
    }
  }


  export class ExecutorSpec extends jspb.Message {
    getContainer(): PipelineDeploymentConfig.PipelineContainerSpec | undefined;
    setContainer(value?: PipelineDeploymentConfig.PipelineContainerSpec): ExecutorSpec;
    hasContainer(): boolean;
    clearContainer(): ExecutorSpec;

    getImporter(): PipelineDeploymentConfig.ImporterSpec | undefined;
    setImporter(value?: PipelineDeploymentConfig.ImporterSpec): ExecutorSpec;
    hasImporter(): boolean;
    clearImporter(): ExecutorSpec;

    getResolver(): PipelineDeploymentConfig.ResolverSpec | undefined;
    setResolver(value?: PipelineDeploymentConfig.ResolverSpec): ExecutorSpec;
    hasResolver(): boolean;
    clearResolver(): ExecutorSpec;

    getCustomJob(): PipelineDeploymentConfig.AIPlatformCustomJobSpec | undefined;
    setCustomJob(value?: PipelineDeploymentConfig.AIPlatformCustomJobSpec): ExecutorSpec;
    hasCustomJob(): boolean;
    clearCustomJob(): ExecutorSpec;

    getSpecCase(): ExecutorSpec.SpecCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ExecutorSpec.AsObject;
    static toObject(includeInstance: boolean, msg: ExecutorSpec): ExecutorSpec.AsObject;
    static serializeBinaryToWriter(message: ExecutorSpec, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ExecutorSpec;
    static deserializeBinaryFromReader(message: ExecutorSpec, reader: jspb.BinaryReader): ExecutorSpec;
  }

  export namespace ExecutorSpec {
    export type AsObject = {
      container?: PipelineDeploymentConfig.PipelineContainerSpec.AsObject,
      importer?: PipelineDeploymentConfig.ImporterSpec.AsObject,
      resolver?: PipelineDeploymentConfig.ResolverSpec.AsObject,
      customJob?: PipelineDeploymentConfig.AIPlatformCustomJobSpec.AsObject,
    }

    export enum SpecCase { 
      SPEC_NOT_SET = 0,
      CONTAINER = 1,
      IMPORTER = 2,
      RESOLVER = 3,
      CUSTOM_JOB = 4,
    }
  }

}

export class Value extends jspb.Message {
  getIntValue(): number;
  setIntValue(value: number): Value;

  getDoubleValue(): number;
  setDoubleValue(value: number): Value;

  getStringValue(): string;
  setStringValue(value: string): Value;

  getValueCase(): Value.ValueCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Value.AsObject;
  static toObject(includeInstance: boolean, msg: Value): Value.AsObject;
  static serializeBinaryToWriter(message: Value, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Value;
  static deserializeBinaryFromReader(message: Value, reader: jspb.BinaryReader): Value;
}

export namespace Value {
  export type AsObject = {
    intValue: number,
    doubleValue: number,
    stringValue: string,
  }

  export enum ValueCase { 
    VALUE_NOT_SET = 0,
    INT_VALUE = 1,
    DOUBLE_VALUE = 2,
    STRING_VALUE = 3,
  }
}

export class RuntimeArtifact extends jspb.Message {
  getName(): string;
  setName(value: string): RuntimeArtifact;

  getType(): ArtifactTypeSchema | undefined;
  setType(value?: ArtifactTypeSchema): RuntimeArtifact;
  hasType(): boolean;
  clearType(): RuntimeArtifact;

  getUri(): string;
  setUri(value: string): RuntimeArtifact;

  getPropertiesMap(): jspb.Map<string, Value>;
  clearPropertiesMap(): RuntimeArtifact;

  getCustomPropertiesMap(): jspb.Map<string, Value>;
  clearCustomPropertiesMap(): RuntimeArtifact;

  getMetadata(): google_protobuf_struct_pb.Struct | undefined;
  setMetadata(value?: google_protobuf_struct_pb.Struct): RuntimeArtifact;
  hasMetadata(): boolean;
  clearMetadata(): RuntimeArtifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RuntimeArtifact.AsObject;
  static toObject(includeInstance: boolean, msg: RuntimeArtifact): RuntimeArtifact.AsObject;
  static serializeBinaryToWriter(message: RuntimeArtifact, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RuntimeArtifact;
  static deserializeBinaryFromReader(message: RuntimeArtifact, reader: jspb.BinaryReader): RuntimeArtifact;
}

export namespace RuntimeArtifact {
  export type AsObject = {
    name: string,
    type?: ArtifactTypeSchema.AsObject,
    uri: string,
    propertiesMap: Array<[string, Value.AsObject]>,
    customPropertiesMap: Array<[string, Value.AsObject]>,
    metadata?: google_protobuf_struct_pb.Struct.AsObject,
  }
}

export class ArtifactList extends jspb.Message {
  getArtifactsList(): Array<RuntimeArtifact>;
  setArtifactsList(value: Array<RuntimeArtifact>): ArtifactList;
  clearArtifactsList(): ArtifactList;
  addArtifacts(value?: RuntimeArtifact, index?: number): RuntimeArtifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactList.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactList): ArtifactList.AsObject;
  static serializeBinaryToWriter(message: ArtifactList, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactList;
  static deserializeBinaryFromReader(message: ArtifactList, reader: jspb.BinaryReader): ArtifactList;
}

export namespace ArtifactList {
  export type AsObject = {
    artifactsList: Array<RuntimeArtifact.AsObject>,
  }
}

export class ExecutorInput extends jspb.Message {
  getInputs(): ExecutorInput.Inputs | undefined;
  setInputs(value?: ExecutorInput.Inputs): ExecutorInput;
  hasInputs(): boolean;
  clearInputs(): ExecutorInput;

  getOutputs(): ExecutorInput.Outputs | undefined;
  setOutputs(value?: ExecutorInput.Outputs): ExecutorInput;
  hasOutputs(): boolean;
  clearOutputs(): ExecutorInput;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExecutorInput.AsObject;
  static toObject(includeInstance: boolean, msg: ExecutorInput): ExecutorInput.AsObject;
  static serializeBinaryToWriter(message: ExecutorInput, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExecutorInput;
  static deserializeBinaryFromReader(message: ExecutorInput, reader: jspb.BinaryReader): ExecutorInput;
}

export namespace ExecutorInput {
  export type AsObject = {
    inputs?: ExecutorInput.Inputs.AsObject,
    outputs?: ExecutorInput.Outputs.AsObject,
  }

  export class Inputs extends jspb.Message {
    getParametersMap(): jspb.Map<string, Value>;
    clearParametersMap(): Inputs;

    getArtifactsMap(): jspb.Map<string, ArtifactList>;
    clearArtifactsMap(): Inputs;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Inputs.AsObject;
    static toObject(includeInstance: boolean, msg: Inputs): Inputs.AsObject;
    static serializeBinaryToWriter(message: Inputs, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Inputs;
    static deserializeBinaryFromReader(message: Inputs, reader: jspb.BinaryReader): Inputs;
  }

  export namespace Inputs {
    export type AsObject = {
      parametersMap: Array<[string, Value.AsObject]>,
      artifactsMap: Array<[string, ArtifactList.AsObject]>,
    }
  }


  export class OutputParameter extends jspb.Message {
    getOutputFile(): string;
    setOutputFile(value: string): OutputParameter;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): OutputParameter.AsObject;
    static toObject(includeInstance: boolean, msg: OutputParameter): OutputParameter.AsObject;
    static serializeBinaryToWriter(message: OutputParameter, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): OutputParameter;
    static deserializeBinaryFromReader(message: OutputParameter, reader: jspb.BinaryReader): OutputParameter;
  }

  export namespace OutputParameter {
    export type AsObject = {
      outputFile: string,
    }
  }


  export class Outputs extends jspb.Message {
    getParametersMap(): jspb.Map<string, ExecutorInput.OutputParameter>;
    clearParametersMap(): Outputs;

    getArtifactsMap(): jspb.Map<string, ArtifactList>;
    clearArtifactsMap(): Outputs;

    getOutputFile(): string;
    setOutputFile(value: string): Outputs;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Outputs.AsObject;
    static toObject(includeInstance: boolean, msg: Outputs): Outputs.AsObject;
    static serializeBinaryToWriter(message: Outputs, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Outputs;
    static deserializeBinaryFromReader(message: Outputs, reader: jspb.BinaryReader): Outputs;
  }

  export namespace Outputs {
    export type AsObject = {
      parametersMap: Array<[string, ExecutorInput.OutputParameter.AsObject]>,
      artifactsMap: Array<[string, ArtifactList.AsObject]>,
      outputFile: string,
    }
  }

}

export class ExecutorOutput extends jspb.Message {
  getParametersMap(): jspb.Map<string, Value>;
  clearParametersMap(): ExecutorOutput;

  getArtifactsMap(): jspb.Map<string, ArtifactList>;
  clearArtifactsMap(): ExecutorOutput;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExecutorOutput.AsObject;
  static toObject(includeInstance: boolean, msg: ExecutorOutput): ExecutorOutput.AsObject;
  static serializeBinaryToWriter(message: ExecutorOutput, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExecutorOutput;
  static deserializeBinaryFromReader(message: ExecutorOutput, reader: jspb.BinaryReader): ExecutorOutput;
}

export namespace ExecutorOutput {
  export type AsObject = {
    parametersMap: Array<[string, Value.AsObject]>,
    artifactsMap: Array<[string, ArtifactList.AsObject]>,
  }
}

export class PipelineTaskFinalStatus extends jspb.Message {
  getState(): string;
  setState(value: string): PipelineTaskFinalStatus;

  getError(): google_rpc_status_pb.Status | undefined;
  setError(value?: google_rpc_status_pb.Status): PipelineTaskFinalStatus;
  hasError(): boolean;
  clearError(): PipelineTaskFinalStatus;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PipelineTaskFinalStatus.AsObject;
  static toObject(includeInstance: boolean, msg: PipelineTaskFinalStatus): PipelineTaskFinalStatus.AsObject;
  static serializeBinaryToWriter(message: PipelineTaskFinalStatus, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PipelineTaskFinalStatus;
  static deserializeBinaryFromReader(message: PipelineTaskFinalStatus, reader: jspb.BinaryReader): PipelineTaskFinalStatus;
}

export namespace PipelineTaskFinalStatus {
  export type AsObject = {
    state: string,
    error?: google_rpc_status_pb.Status.AsObject,
  }
}

export class PipelineStateEnum extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PipelineStateEnum.AsObject;
  static toObject(includeInstance: boolean, msg: PipelineStateEnum): PipelineStateEnum.AsObject;
  static serializeBinaryToWriter(message: PipelineStateEnum, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PipelineStateEnum;
  static deserializeBinaryFromReader(message: PipelineStateEnum, reader: jspb.BinaryReader): PipelineStateEnum;
}

export namespace PipelineStateEnum {
  export type AsObject = {
  }

  export enum PipelineTaskState { 
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
    SKIPPED = 10,
    QUEUED = 11,
    NOT_TRIGGERED = 12,
    UNSCHEDULABLE = 13,
  }
}

