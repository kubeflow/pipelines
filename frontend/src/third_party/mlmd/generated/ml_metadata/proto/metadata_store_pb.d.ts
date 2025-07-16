import * as jspb from 'google-protobuf';

import * as google_protobuf_any_pb from 'google-protobuf/google/protobuf/any_pb';
import * as google_protobuf_struct_pb from 'google-protobuf/google/protobuf/struct_pb';
import * as google_protobuf_descriptor_pb from 'google-protobuf/google/protobuf/descriptor_pb';

export class SystemTypeExtension extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): SystemTypeExtension;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SystemTypeExtension.AsObject;
  static toObject(includeInstance: boolean, msg: SystemTypeExtension): SystemTypeExtension.AsObject;
  static serializeBinaryToWriter(message: SystemTypeExtension, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SystemTypeExtension;
  static deserializeBinaryFromReader(
    message: SystemTypeExtension,
    reader: jspb.BinaryReader,
  ): SystemTypeExtension;
}

export namespace SystemTypeExtension {
  export type AsObject = {
    typeName: string;
  };
}

export class Value extends jspb.Message {
  getIntValue(): number;
  setIntValue(value: number): Value;

  getDoubleValue(): number;
  setDoubleValue(value: number): Value;

  getStringValue(): string;
  setStringValue(value: string): Value;

  getStructValue(): google_protobuf_struct_pb.Struct | undefined;
  setStructValue(value?: google_protobuf_struct_pb.Struct): Value;
  hasStructValue(): boolean;
  clearStructValue(): Value;

  getProtoValue(): google_protobuf_any_pb.Any | undefined;
  setProtoValue(value?: google_protobuf_any_pb.Any): Value;
  hasProtoValue(): boolean;
  clearProtoValue(): Value;

  getBoolValue(): boolean;
  setBoolValue(value: boolean): Value;

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
    intValue: number;
    doubleValue: number;
    stringValue: string;
    structValue?: google_protobuf_struct_pb.Struct.AsObject;
    protoValue?: google_protobuf_any_pb.Any.AsObject;
    boolValue: boolean;
  };

  export enum ValueCase {
    VALUE_NOT_SET = 0,
    INT_VALUE = 1,
    DOUBLE_VALUE = 2,
    STRING_VALUE = 3,
    STRUCT_VALUE = 4,
    PROTO_VALUE = 5,
    BOOL_VALUE = 6,
  }
}

export class Artifact extends jspb.Message {
  getId(): number;
  setId(value: number): Artifact;

  getName(): string;
  setName(value: string): Artifact;

  getTypeId(): number;
  setTypeId(value: number): Artifact;

  getType(): string;
  setType(value: string): Artifact;

  getUri(): string;
  setUri(value: string): Artifact;

  getExternalId(): string;
  setExternalId(value: string): Artifact;

  getPropertiesMap(): jspb.Map<string, Value>;
  clearPropertiesMap(): Artifact;

  getCustomPropertiesMap(): jspb.Map<string, Value>;
  clearCustomPropertiesMap(): Artifact;

  getState(): Artifact.State;
  setState(value: Artifact.State): Artifact;

  getCreateTimeSinceEpoch(): number;
  setCreateTimeSinceEpoch(value: number): Artifact;

  getLastUpdateTimeSinceEpoch(): number;
  setLastUpdateTimeSinceEpoch(value: number): Artifact;

  getSystemMetadata(): google_protobuf_any_pb.Any | undefined;
  setSystemMetadata(value?: google_protobuf_any_pb.Any): Artifact;
  hasSystemMetadata(): boolean;
  clearSystemMetadata(): Artifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Artifact.AsObject;
  static toObject(includeInstance: boolean, msg: Artifact): Artifact.AsObject;
  static serializeBinaryToWriter(message: Artifact, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Artifact;
  static deserializeBinaryFromReader(message: Artifact, reader: jspb.BinaryReader): Artifact;
}

export namespace Artifact {
  export type AsObject = {
    id: number;
    name: string;
    typeId: number;
    type: string;
    uri: string;
    externalId: string;
    propertiesMap: Array<[string, Value.AsObject]>;
    customPropertiesMap: Array<[string, Value.AsObject]>;
    state: Artifact.State;
    createTimeSinceEpoch: number;
    lastUpdateTimeSinceEpoch: number;
    systemMetadata?: google_protobuf_any_pb.Any.AsObject;
  };

  export enum State {
    UNKNOWN = 0,
    PENDING = 1,
    LIVE = 2,
    MARKED_FOR_DELETION = 3,
    DELETED = 4,
    ABANDONED = 5,
    REFERENCE = 6,
  }
}

export class ArtifactType extends jspb.Message {
  getId(): number;
  setId(value: number): ArtifactType;

  getName(): string;
  setName(value: string): ArtifactType;

  getVersion(): string;
  setVersion(value: string): ArtifactType;

  getDescription(): string;
  setDescription(value: string): ArtifactType;

  getExternalId(): string;
  setExternalId(value: string): ArtifactType;

  getPropertiesMap(): jspb.Map<string, PropertyType>;
  clearPropertiesMap(): ArtifactType;

  getBaseType(): ArtifactType.SystemDefinedBaseType;
  setBaseType(value: ArtifactType.SystemDefinedBaseType): ArtifactType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactType.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactType): ArtifactType.AsObject;
  static serializeBinaryToWriter(message: ArtifactType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactType;
  static deserializeBinaryFromReader(
    message: ArtifactType,
    reader: jspb.BinaryReader,
  ): ArtifactType;
}

export namespace ArtifactType {
  export type AsObject = {
    id: number;
    name: string;
    version: string;
    description: string;
    externalId: string;
    propertiesMap: Array<[string, PropertyType]>;
    baseType: ArtifactType.SystemDefinedBaseType;
  };

  export enum SystemDefinedBaseType {
    UNSET = 0,
    DATASET = 1,
    MODEL = 2,
    METRICS = 3,
    STATISTICS = 4,
  }
}

export class Event extends jspb.Message {
  getArtifactId(): number;
  setArtifactId(value: number): Event;

  getExecutionId(): number;
  setExecutionId(value: number): Event;

  getPath(): Event.Path | undefined;
  setPath(value?: Event.Path): Event;
  hasPath(): boolean;
  clearPath(): Event;

  getType(): Event.Type;
  setType(value: Event.Type): Event;

  getMillisecondsSinceEpoch(): number;
  setMillisecondsSinceEpoch(value: number): Event;

  getSystemMetadata(): google_protobuf_any_pb.Any | undefined;
  setSystemMetadata(value?: google_protobuf_any_pb.Any): Event;
  hasSystemMetadata(): boolean;
  clearSystemMetadata(): Event;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Event.AsObject;
  static toObject(includeInstance: boolean, msg: Event): Event.AsObject;
  static serializeBinaryToWriter(message: Event, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Event;
  static deserializeBinaryFromReader(message: Event, reader: jspb.BinaryReader): Event;
}

export namespace Event {
  export type AsObject = {
    artifactId: number;
    executionId: number;
    path?: Event.Path.AsObject;
    type: Event.Type;
    millisecondsSinceEpoch: number;
    systemMetadata?: google_protobuf_any_pb.Any.AsObject;
  };

  export class Path extends jspb.Message {
    getStepsList(): Array<Event.Path.Step>;
    setStepsList(value: Array<Event.Path.Step>): Path;
    clearStepsList(): Path;
    addSteps(value?: Event.Path.Step, index?: number): Event.Path.Step;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Path.AsObject;
    static toObject(includeInstance: boolean, msg: Path): Path.AsObject;
    static serializeBinaryToWriter(message: Path, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Path;
    static deserializeBinaryFromReader(message: Path, reader: jspb.BinaryReader): Path;
  }

  export namespace Path {
    export type AsObject = {
      stepsList: Array<Event.Path.Step.AsObject>;
    };

    export class Step extends jspb.Message {
      getIndex(): number;
      setIndex(value: number): Step;

      getKey(): string;
      setKey(value: string): Step;

      getValueCase(): Step.ValueCase;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): Step.AsObject;
      static toObject(includeInstance: boolean, msg: Step): Step.AsObject;
      static serializeBinaryToWriter(message: Step, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): Step;
      static deserializeBinaryFromReader(message: Step, reader: jspb.BinaryReader): Step;
    }

    export namespace Step {
      export type AsObject = {
        index: number;
        key: string;
      };

      export enum ValueCase {
        VALUE_NOT_SET = 0,
        INDEX = 1,
        KEY = 2,
      }
    }
  }

  export enum Type {
    UNKNOWN = 0,
    DECLARED_OUTPUT = 1,
    DECLARED_INPUT = 2,
    INPUT = 3,
    OUTPUT = 4,
    INTERNAL_INPUT = 5,
    INTERNAL_OUTPUT = 6,
    PENDING_OUTPUT = 7,
  }
}

export class Execution extends jspb.Message {
  getId(): number;
  setId(value: number): Execution;

  getName(): string;
  setName(value: string): Execution;

  getTypeId(): number;
  setTypeId(value: number): Execution;

  getType(): string;
  setType(value: string): Execution;

  getExternalId(): string;
  setExternalId(value: string): Execution;

  getLastKnownState(): Execution.State;
  setLastKnownState(value: Execution.State): Execution;

  getPropertiesMap(): jspb.Map<string, Value>;
  clearPropertiesMap(): Execution;

  getCustomPropertiesMap(): jspb.Map<string, Value>;
  clearCustomPropertiesMap(): Execution;

  getCreateTimeSinceEpoch(): number;
  setCreateTimeSinceEpoch(value: number): Execution;

  getLastUpdateTimeSinceEpoch(): number;
  setLastUpdateTimeSinceEpoch(value: number): Execution;

  getSystemMetadata(): google_protobuf_any_pb.Any | undefined;
  setSystemMetadata(value?: google_protobuf_any_pb.Any): Execution;
  hasSystemMetadata(): boolean;
  clearSystemMetadata(): Execution;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Execution.AsObject;
  static toObject(includeInstance: boolean, msg: Execution): Execution.AsObject;
  static serializeBinaryToWriter(message: Execution, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Execution;
  static deserializeBinaryFromReader(message: Execution, reader: jspb.BinaryReader): Execution;
}

export namespace Execution {
  export type AsObject = {
    id: number;
    name: string;
    typeId: number;
    type: string;
    externalId: string;
    lastKnownState: Execution.State;
    propertiesMap: Array<[string, Value.AsObject]>;
    customPropertiesMap: Array<[string, Value.AsObject]>;
    createTimeSinceEpoch: number;
    lastUpdateTimeSinceEpoch: number;
    systemMetadata?: google_protobuf_any_pb.Any.AsObject;
  };

  export enum State {
    UNKNOWN = 0,
    NEW = 1,
    RUNNING = 2,
    COMPLETE = 3,
    FAILED = 4,
    CACHED = 5,
    CANCELED = 6,
  }
}

export class ExecutionType extends jspb.Message {
  getId(): number;
  setId(value: number): ExecutionType;

  getName(): string;
  setName(value: string): ExecutionType;

  getVersion(): string;
  setVersion(value: string): ExecutionType;

  getDescription(): string;
  setDescription(value: string): ExecutionType;

  getExternalId(): string;
  setExternalId(value: string): ExecutionType;

  getPropertiesMap(): jspb.Map<string, PropertyType>;
  clearPropertiesMap(): ExecutionType;

  getInputType(): ArtifactStructType | undefined;
  setInputType(value?: ArtifactStructType): ExecutionType;
  hasInputType(): boolean;
  clearInputType(): ExecutionType;

  getOutputType(): ArtifactStructType | undefined;
  setOutputType(value?: ArtifactStructType): ExecutionType;
  hasOutputType(): boolean;
  clearOutputType(): ExecutionType;

  getBaseType(): ExecutionType.SystemDefinedBaseType;
  setBaseType(value: ExecutionType.SystemDefinedBaseType): ExecutionType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExecutionType.AsObject;
  static toObject(includeInstance: boolean, msg: ExecutionType): ExecutionType.AsObject;
  static serializeBinaryToWriter(message: ExecutionType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExecutionType;
  static deserializeBinaryFromReader(
    message: ExecutionType,
    reader: jspb.BinaryReader,
  ): ExecutionType;
}

export namespace ExecutionType {
  export type AsObject = {
    id: number;
    name: string;
    version: string;
    description: string;
    externalId: string;
    propertiesMap: Array<[string, PropertyType]>;
    inputType?: ArtifactStructType.AsObject;
    outputType?: ArtifactStructType.AsObject;
    baseType: ExecutionType.SystemDefinedBaseType;
  };

  export enum SystemDefinedBaseType {
    UNSET = 0,
    TRAIN = 1,
    TRANSFORM = 2,
    PROCESS = 3,
    EVALUATE = 4,
    DEPLOY = 5,
  }
}

export class ContextType extends jspb.Message {
  getId(): number;
  setId(value: number): ContextType;

  getName(): string;
  setName(value: string): ContextType;

  getVersion(): string;
  setVersion(value: string): ContextType;

  getDescription(): string;
  setDescription(value: string): ContextType;

  getExternalId(): string;
  setExternalId(value: string): ContextType;

  getPropertiesMap(): jspb.Map<string, PropertyType>;
  clearPropertiesMap(): ContextType;

  getBaseType(): ContextType.SystemDefinedBaseType;
  setBaseType(value: ContextType.SystemDefinedBaseType): ContextType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ContextType.AsObject;
  static toObject(includeInstance: boolean, msg: ContextType): ContextType.AsObject;
  static serializeBinaryToWriter(message: ContextType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ContextType;
  static deserializeBinaryFromReader(message: ContextType, reader: jspb.BinaryReader): ContextType;
}

export namespace ContextType {
  export type AsObject = {
    id: number;
    name: string;
    version: string;
    description: string;
    externalId: string;
    propertiesMap: Array<[string, PropertyType]>;
    baseType: ContextType.SystemDefinedBaseType;
  };

  export enum SystemDefinedBaseType {
    UNSET = 0,
  }
}

export class Context extends jspb.Message {
  getId(): number;
  setId(value: number): Context;

  getName(): string;
  setName(value: string): Context;

  getTypeId(): number;
  setTypeId(value: number): Context;

  getType(): string;
  setType(value: string): Context;

  getExternalId(): string;
  setExternalId(value: string): Context;

  getPropertiesMap(): jspb.Map<string, Value>;
  clearPropertiesMap(): Context;

  getCustomPropertiesMap(): jspb.Map<string, Value>;
  clearCustomPropertiesMap(): Context;

  getCreateTimeSinceEpoch(): number;
  setCreateTimeSinceEpoch(value: number): Context;

  getLastUpdateTimeSinceEpoch(): number;
  setLastUpdateTimeSinceEpoch(value: number): Context;

  getSystemMetadata(): google_protobuf_any_pb.Any | undefined;
  setSystemMetadata(value?: google_protobuf_any_pb.Any): Context;
  hasSystemMetadata(): boolean;
  clearSystemMetadata(): Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Context.AsObject;
  static toObject(includeInstance: boolean, msg: Context): Context.AsObject;
  static serializeBinaryToWriter(message: Context, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Context;
  static deserializeBinaryFromReader(message: Context, reader: jspb.BinaryReader): Context;
}

export namespace Context {
  export type AsObject = {
    id: number;
    name: string;
    typeId: number;
    type: string;
    externalId: string;
    propertiesMap: Array<[string, Value.AsObject]>;
    customPropertiesMap: Array<[string, Value.AsObject]>;
    createTimeSinceEpoch: number;
    lastUpdateTimeSinceEpoch: number;
    systemMetadata?: google_protobuf_any_pb.Any.AsObject;
  };
}

export class Attribution extends jspb.Message {
  getArtifactId(): number;
  setArtifactId(value: number): Attribution;

  getContextId(): number;
  setContextId(value: number): Attribution;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Attribution.AsObject;
  static toObject(includeInstance: boolean, msg: Attribution): Attribution.AsObject;
  static serializeBinaryToWriter(message: Attribution, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Attribution;
  static deserializeBinaryFromReader(message: Attribution, reader: jspb.BinaryReader): Attribution;
}

export namespace Attribution {
  export type AsObject = {
    artifactId: number;
    contextId: number;
  };
}

export class Association extends jspb.Message {
  getExecutionId(): number;
  setExecutionId(value: number): Association;

  getContextId(): number;
  setContextId(value: number): Association;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Association.AsObject;
  static toObject(includeInstance: boolean, msg: Association): Association.AsObject;
  static serializeBinaryToWriter(message: Association, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Association;
  static deserializeBinaryFromReader(message: Association, reader: jspb.BinaryReader): Association;
}

export namespace Association {
  export type AsObject = {
    executionId: number;
    contextId: number;
  };
}

export class ParentContext extends jspb.Message {
  getChildId(): number;
  setChildId(value: number): ParentContext;

  getParentId(): number;
  setParentId(value: number): ParentContext;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ParentContext.AsObject;
  static toObject(includeInstance: boolean, msg: ParentContext): ParentContext.AsObject;
  static serializeBinaryToWriter(message: ParentContext, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ParentContext;
  static deserializeBinaryFromReader(
    message: ParentContext,
    reader: jspb.BinaryReader,
  ): ParentContext;
}

export namespace ParentContext {
  export type AsObject = {
    childId: number;
    parentId: number;
  };
}

export class LineageGraph extends jspb.Message {
  getArtifactTypesList(): Array<ArtifactType>;
  setArtifactTypesList(value: Array<ArtifactType>): LineageGraph;
  clearArtifactTypesList(): LineageGraph;
  addArtifactTypes(value?: ArtifactType, index?: number): ArtifactType;

  getExecutionTypesList(): Array<ExecutionType>;
  setExecutionTypesList(value: Array<ExecutionType>): LineageGraph;
  clearExecutionTypesList(): LineageGraph;
  addExecutionTypes(value?: ExecutionType, index?: number): ExecutionType;

  getContextTypesList(): Array<ContextType>;
  setContextTypesList(value: Array<ContextType>): LineageGraph;
  clearContextTypesList(): LineageGraph;
  addContextTypes(value?: ContextType, index?: number): ContextType;

  getArtifactsList(): Array<Artifact>;
  setArtifactsList(value: Array<Artifact>): LineageGraph;
  clearArtifactsList(): LineageGraph;
  addArtifacts(value?: Artifact, index?: number): Artifact;

  getExecutionsList(): Array<Execution>;
  setExecutionsList(value: Array<Execution>): LineageGraph;
  clearExecutionsList(): LineageGraph;
  addExecutions(value?: Execution, index?: number): Execution;

  getContextsList(): Array<Context>;
  setContextsList(value: Array<Context>): LineageGraph;
  clearContextsList(): LineageGraph;
  addContexts(value?: Context, index?: number): Context;

  getEventsList(): Array<Event>;
  setEventsList(value: Array<Event>): LineageGraph;
  clearEventsList(): LineageGraph;
  addEvents(value?: Event, index?: number): Event;

  getAttributionsList(): Array<Attribution>;
  setAttributionsList(value: Array<Attribution>): LineageGraph;
  clearAttributionsList(): LineageGraph;
  addAttributions(value?: Attribution, index?: number): Attribution;

  getAssociationsList(): Array<Association>;
  setAssociationsList(value: Array<Association>): LineageGraph;
  clearAssociationsList(): LineageGraph;
  addAssociations(value?: Association, index?: number): Association;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LineageGraph.AsObject;
  static toObject(includeInstance: boolean, msg: LineageGraph): LineageGraph.AsObject;
  static serializeBinaryToWriter(message: LineageGraph, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LineageGraph;
  static deserializeBinaryFromReader(
    message: LineageGraph,
    reader: jspb.BinaryReader,
  ): LineageGraph;
}

export namespace LineageGraph {
  export type AsObject = {
    artifactTypesList: Array<ArtifactType.AsObject>;
    executionTypesList: Array<ExecutionType.AsObject>;
    contextTypesList: Array<ContextType.AsObject>;
    artifactsList: Array<Artifact.AsObject>;
    executionsList: Array<Execution.AsObject>;
    contextsList: Array<Context.AsObject>;
    eventsList: Array<Event.AsObject>;
    attributionsList: Array<Attribution.AsObject>;
    associationsList: Array<Association.AsObject>;
  };
}

export class ArtifactStructType extends jspb.Message {
  getSimple(): ArtifactType | undefined;
  setSimple(value?: ArtifactType): ArtifactStructType;
  hasSimple(): boolean;
  clearSimple(): ArtifactStructType;

  getUnionType(): UnionArtifactStructType | undefined;
  setUnionType(value?: UnionArtifactStructType): ArtifactStructType;
  hasUnionType(): boolean;
  clearUnionType(): ArtifactStructType;

  getIntersection(): IntersectionArtifactStructType | undefined;
  setIntersection(value?: IntersectionArtifactStructType): ArtifactStructType;
  hasIntersection(): boolean;
  clearIntersection(): ArtifactStructType;

  getList(): ListArtifactStructType | undefined;
  setList(value?: ListArtifactStructType): ArtifactStructType;
  hasList(): boolean;
  clearList(): ArtifactStructType;

  getNone(): NoneArtifactStructType | undefined;
  setNone(value?: NoneArtifactStructType): ArtifactStructType;
  hasNone(): boolean;
  clearNone(): ArtifactStructType;

  getAny(): AnyArtifactStructType | undefined;
  setAny(value?: AnyArtifactStructType): ArtifactStructType;
  hasAny(): boolean;
  clearAny(): ArtifactStructType;

  getTuple(): TupleArtifactStructType | undefined;
  setTuple(value?: TupleArtifactStructType): ArtifactStructType;
  hasTuple(): boolean;
  clearTuple(): ArtifactStructType;

  getDict(): DictArtifactStructType | undefined;
  setDict(value?: DictArtifactStructType): ArtifactStructType;
  hasDict(): boolean;
  clearDict(): ArtifactStructType;

  getKindCase(): ArtifactStructType.KindCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactStructType.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactStructType): ArtifactStructType.AsObject;
  static serializeBinaryToWriter(message: ArtifactStructType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactStructType;
  static deserializeBinaryFromReader(
    message: ArtifactStructType,
    reader: jspb.BinaryReader,
  ): ArtifactStructType;
}

export namespace ArtifactStructType {
  export type AsObject = {
    simple?: ArtifactType.AsObject;
    unionType?: UnionArtifactStructType.AsObject;
    intersection?: IntersectionArtifactStructType.AsObject;
    list?: ListArtifactStructType.AsObject;
    none?: NoneArtifactStructType.AsObject;
    any?: AnyArtifactStructType.AsObject;
    tuple?: TupleArtifactStructType.AsObject;
    dict?: DictArtifactStructType.AsObject;
  };

  export enum KindCase {
    KIND_NOT_SET = 0,
    SIMPLE = 1,
    UNION_TYPE = 2,
    INTERSECTION = 3,
    LIST = 4,
    NONE = 5,
    ANY = 6,
    TUPLE = 7,
    DICT = 8,
  }
}

export class UnionArtifactStructType extends jspb.Message {
  getCandidatesList(): Array<ArtifactStructType>;
  setCandidatesList(value: Array<ArtifactStructType>): UnionArtifactStructType;
  clearCandidatesList(): UnionArtifactStructType;
  addCandidates(value?: ArtifactStructType, index?: number): ArtifactStructType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UnionArtifactStructType.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: UnionArtifactStructType,
  ): UnionArtifactStructType.AsObject;
  static serializeBinaryToWriter(message: UnionArtifactStructType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UnionArtifactStructType;
  static deserializeBinaryFromReader(
    message: UnionArtifactStructType,
    reader: jspb.BinaryReader,
  ): UnionArtifactStructType;
}

export namespace UnionArtifactStructType {
  export type AsObject = {
    candidatesList: Array<ArtifactStructType.AsObject>;
  };
}

export class IntersectionArtifactStructType extends jspb.Message {
  getConstraintsList(): Array<ArtifactStructType>;
  setConstraintsList(value: Array<ArtifactStructType>): IntersectionArtifactStructType;
  clearConstraintsList(): IntersectionArtifactStructType;
  addConstraints(value?: ArtifactStructType, index?: number): ArtifactStructType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): IntersectionArtifactStructType.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: IntersectionArtifactStructType,
  ): IntersectionArtifactStructType.AsObject;
  static serializeBinaryToWriter(
    message: IntersectionArtifactStructType,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): IntersectionArtifactStructType;
  static deserializeBinaryFromReader(
    message: IntersectionArtifactStructType,
    reader: jspb.BinaryReader,
  ): IntersectionArtifactStructType;
}

export namespace IntersectionArtifactStructType {
  export type AsObject = {
    constraintsList: Array<ArtifactStructType.AsObject>;
  };
}

export class ListArtifactStructType extends jspb.Message {
  getElement(): ArtifactStructType | undefined;
  setElement(value?: ArtifactStructType): ListArtifactStructType;
  hasElement(): boolean;
  clearElement(): ListArtifactStructType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListArtifactStructType.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListArtifactStructType,
  ): ListArtifactStructType.AsObject;
  static serializeBinaryToWriter(message: ListArtifactStructType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListArtifactStructType;
  static deserializeBinaryFromReader(
    message: ListArtifactStructType,
    reader: jspb.BinaryReader,
  ): ListArtifactStructType;
}

export namespace ListArtifactStructType {
  export type AsObject = {
    element?: ArtifactStructType.AsObject;
  };
}

export class NoneArtifactStructType extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NoneArtifactStructType.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: NoneArtifactStructType,
  ): NoneArtifactStructType.AsObject;
  static serializeBinaryToWriter(message: NoneArtifactStructType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NoneArtifactStructType;
  static deserializeBinaryFromReader(
    message: NoneArtifactStructType,
    reader: jspb.BinaryReader,
  ): NoneArtifactStructType;
}

export namespace NoneArtifactStructType {
  export type AsObject = {};
}

export class AnyArtifactStructType extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AnyArtifactStructType.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: AnyArtifactStructType,
  ): AnyArtifactStructType.AsObject;
  static serializeBinaryToWriter(message: AnyArtifactStructType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AnyArtifactStructType;
  static deserializeBinaryFromReader(
    message: AnyArtifactStructType,
    reader: jspb.BinaryReader,
  ): AnyArtifactStructType;
}

export namespace AnyArtifactStructType {
  export type AsObject = {};
}

export class TupleArtifactStructType extends jspb.Message {
  getElementsList(): Array<ArtifactStructType>;
  setElementsList(value: Array<ArtifactStructType>): TupleArtifactStructType;
  clearElementsList(): TupleArtifactStructType;
  addElements(value?: ArtifactStructType, index?: number): ArtifactStructType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TupleArtifactStructType.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: TupleArtifactStructType,
  ): TupleArtifactStructType.AsObject;
  static serializeBinaryToWriter(message: TupleArtifactStructType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TupleArtifactStructType;
  static deserializeBinaryFromReader(
    message: TupleArtifactStructType,
    reader: jspb.BinaryReader,
  ): TupleArtifactStructType;
}

export namespace TupleArtifactStructType {
  export type AsObject = {
    elementsList: Array<ArtifactStructType.AsObject>;
  };
}

export class DictArtifactStructType extends jspb.Message {
  getPropertiesMap(): jspb.Map<string, ArtifactStructType>;
  clearPropertiesMap(): DictArtifactStructType;

  getNoneTypeNotRequired(): boolean;
  setNoneTypeNotRequired(value: boolean): DictArtifactStructType;

  getExtraPropertiesType(): ArtifactStructType | undefined;
  setExtraPropertiesType(value?: ArtifactStructType): DictArtifactStructType;
  hasExtraPropertiesType(): boolean;
  clearExtraPropertiesType(): DictArtifactStructType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DictArtifactStructType.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: DictArtifactStructType,
  ): DictArtifactStructType.AsObject;
  static serializeBinaryToWriter(message: DictArtifactStructType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DictArtifactStructType;
  static deserializeBinaryFromReader(
    message: DictArtifactStructType,
    reader: jspb.BinaryReader,
  ): DictArtifactStructType;
}

export namespace DictArtifactStructType {
  export type AsObject = {
    propertiesMap: Array<[string, ArtifactStructType.AsObject]>;
    noneTypeNotRequired: boolean;
    extraPropertiesType?: ArtifactStructType.AsObject;
  };
}

export class FakeDatabaseConfig extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FakeDatabaseConfig.AsObject;
  static toObject(includeInstance: boolean, msg: FakeDatabaseConfig): FakeDatabaseConfig.AsObject;
  static serializeBinaryToWriter(message: FakeDatabaseConfig, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FakeDatabaseConfig;
  static deserializeBinaryFromReader(
    message: FakeDatabaseConfig,
    reader: jspb.BinaryReader,
  ): FakeDatabaseConfig;
}

export namespace FakeDatabaseConfig {
  export type AsObject = {};
}

export class MySQLDatabaseConfig extends jspb.Message {
  getHost(): string;
  setHost(value: string): MySQLDatabaseConfig;

  getPort(): number;
  setPort(value: number): MySQLDatabaseConfig;

  getDatabase(): string;
  setDatabase(value: string): MySQLDatabaseConfig;

  getUser(): string;
  setUser(value: string): MySQLDatabaseConfig;

  getPassword(): string;
  setPassword(value: string): MySQLDatabaseConfig;

  getSocket(): string;
  setSocket(value: string): MySQLDatabaseConfig;

  getSslOptions(): MySQLDatabaseConfig.SSLOptions | undefined;
  setSslOptions(value?: MySQLDatabaseConfig.SSLOptions): MySQLDatabaseConfig;
  hasSslOptions(): boolean;
  clearSslOptions(): MySQLDatabaseConfig;

  getSkipDbCreation(): boolean;
  setSkipDbCreation(value: boolean): MySQLDatabaseConfig;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MySQLDatabaseConfig.AsObject;
  static toObject(includeInstance: boolean, msg: MySQLDatabaseConfig): MySQLDatabaseConfig.AsObject;
  static serializeBinaryToWriter(message: MySQLDatabaseConfig, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MySQLDatabaseConfig;
  static deserializeBinaryFromReader(
    message: MySQLDatabaseConfig,
    reader: jspb.BinaryReader,
  ): MySQLDatabaseConfig;
}

export namespace MySQLDatabaseConfig {
  export type AsObject = {
    host: string;
    port: number;
    database: string;
    user: string;
    password: string;
    socket: string;
    sslOptions?: MySQLDatabaseConfig.SSLOptions.AsObject;
    skipDbCreation: boolean;
  };

  export class SSLOptions extends jspb.Message {
    getKey(): string;
    setKey(value: string): SSLOptions;

    getCert(): string;
    setCert(value: string): SSLOptions;

    getCa(): string;
    setCa(value: string): SSLOptions;

    getCapath(): string;
    setCapath(value: string): SSLOptions;

    getCipher(): string;
    setCipher(value: string): SSLOptions;

    getVerifyServerCert(): boolean;
    setVerifyServerCert(value: boolean): SSLOptions;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SSLOptions.AsObject;
    static toObject(includeInstance: boolean, msg: SSLOptions): SSLOptions.AsObject;
    static serializeBinaryToWriter(message: SSLOptions, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SSLOptions;
    static deserializeBinaryFromReader(message: SSLOptions, reader: jspb.BinaryReader): SSLOptions;
  }

  export namespace SSLOptions {
    export type AsObject = {
      key: string;
      cert: string;
      ca: string;
      capath: string;
      cipher: string;
      verifyServerCert: boolean;
    };
  }
}

export class SqliteMetadataSourceConfig extends jspb.Message {
  getFilenameUri(): string;
  setFilenameUri(value: string): SqliteMetadataSourceConfig;

  getConnectionMode(): SqliteMetadataSourceConfig.ConnectionMode;
  setConnectionMode(value: SqliteMetadataSourceConfig.ConnectionMode): SqliteMetadataSourceConfig;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SqliteMetadataSourceConfig.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: SqliteMetadataSourceConfig,
  ): SqliteMetadataSourceConfig.AsObject;
  static serializeBinaryToWriter(
    message: SqliteMetadataSourceConfig,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): SqliteMetadataSourceConfig;
  static deserializeBinaryFromReader(
    message: SqliteMetadataSourceConfig,
    reader: jspb.BinaryReader,
  ): SqliteMetadataSourceConfig;
}

export namespace SqliteMetadataSourceConfig {
  export type AsObject = {
    filenameUri: string;
    connectionMode: SqliteMetadataSourceConfig.ConnectionMode;
  };

  export enum ConnectionMode {
    UNKNOWN = 0,
    READONLY = 1,
    READWRITE = 2,
    READWRITE_OPENCREATE = 3,
  }
}

export class PostgreSQLDatabaseConfig extends jspb.Message {
  getHost(): string;
  setHost(value: string): PostgreSQLDatabaseConfig;

  getHostaddr(): string;
  setHostaddr(value: string): PostgreSQLDatabaseConfig;

  getPort(): string;
  setPort(value: string): PostgreSQLDatabaseConfig;

  getUser(): string;
  setUser(value: string): PostgreSQLDatabaseConfig;

  getPassword(): string;
  setPassword(value: string): PostgreSQLDatabaseConfig;

  getPassfile(): string;
  setPassfile(value: string): PostgreSQLDatabaseConfig;

  getDbname(): string;
  setDbname(value: string): PostgreSQLDatabaseConfig;

  getSkipDbCreation(): boolean;
  setSkipDbCreation(value: boolean): PostgreSQLDatabaseConfig;

  getSsloption(): PostgreSQLDatabaseConfig.SSLOptions | undefined;
  setSsloption(value?: PostgreSQLDatabaseConfig.SSLOptions): PostgreSQLDatabaseConfig;
  hasSsloption(): boolean;
  clearSsloption(): PostgreSQLDatabaseConfig;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PostgreSQLDatabaseConfig.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PostgreSQLDatabaseConfig,
  ): PostgreSQLDatabaseConfig.AsObject;
  static serializeBinaryToWriter(
    message: PostgreSQLDatabaseConfig,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): PostgreSQLDatabaseConfig;
  static deserializeBinaryFromReader(
    message: PostgreSQLDatabaseConfig,
    reader: jspb.BinaryReader,
  ): PostgreSQLDatabaseConfig;
}

export namespace PostgreSQLDatabaseConfig {
  export type AsObject = {
    host: string;
    hostaddr: string;
    port: string;
    user: string;
    password: string;
    passfile: string;
    dbname: string;
    skipDbCreation: boolean;
    ssloption?: PostgreSQLDatabaseConfig.SSLOptions.AsObject;
  };

  export class SSLOptions extends jspb.Message {
    getSslmode(): string;
    setSslmode(value: string): SSLOptions;

    getSslcert(): string;
    setSslcert(value: string): SSLOptions;

    getSslkey(): string;
    setSslkey(value: string): SSLOptions;

    getSslpassword(): string;
    setSslpassword(value: string): SSLOptions;

    getSslrootcert(): string;
    setSslrootcert(value: string): SSLOptions;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SSLOptions.AsObject;
    static toObject(includeInstance: boolean, msg: SSLOptions): SSLOptions.AsObject;
    static serializeBinaryToWriter(message: SSLOptions, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SSLOptions;
    static deserializeBinaryFromReader(message: SSLOptions, reader: jspb.BinaryReader): SSLOptions;
  }

  export namespace SSLOptions {
    export type AsObject = {
      sslmode: string;
      sslcert: string;
      sslkey: string;
      sslpassword: string;
      sslrootcert: string;
    };
  }
}

export class MigrationOptions extends jspb.Message {
  getEnableUpgradeMigration(): boolean;
  setEnableUpgradeMigration(value: boolean): MigrationOptions;

  getDowngradeToSchemaVersion(): number;
  setDowngradeToSchemaVersion(value: number): MigrationOptions;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MigrationOptions.AsObject;
  static toObject(includeInstance: boolean, msg: MigrationOptions): MigrationOptions.AsObject;
  static serializeBinaryToWriter(message: MigrationOptions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MigrationOptions;
  static deserializeBinaryFromReader(
    message: MigrationOptions,
    reader: jspb.BinaryReader,
  ): MigrationOptions;
}

export namespace MigrationOptions {
  export type AsObject = {
    enableUpgradeMigration: boolean;
    downgradeToSchemaVersion: number;
  };
}

export class RetryOptions extends jspb.Message {
  getMaxNumRetries(): number;
  setMaxNumRetries(value: number): RetryOptions;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RetryOptions.AsObject;
  static toObject(includeInstance: boolean, msg: RetryOptions): RetryOptions.AsObject;
  static serializeBinaryToWriter(message: RetryOptions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RetryOptions;
  static deserializeBinaryFromReader(
    message: RetryOptions,
    reader: jspb.BinaryReader,
  ): RetryOptions;
}

export namespace RetryOptions {
  export type AsObject = {
    maxNumRetries: number;
  };
}

export class ConnectionConfig extends jspb.Message {
  getFakeDatabase(): FakeDatabaseConfig | undefined;
  setFakeDatabase(value?: FakeDatabaseConfig): ConnectionConfig;
  hasFakeDatabase(): boolean;
  clearFakeDatabase(): ConnectionConfig;

  getMysql(): MySQLDatabaseConfig | undefined;
  setMysql(value?: MySQLDatabaseConfig): ConnectionConfig;
  hasMysql(): boolean;
  clearMysql(): ConnectionConfig;

  getSqlite(): SqliteMetadataSourceConfig | undefined;
  setSqlite(value?: SqliteMetadataSourceConfig): ConnectionConfig;
  hasSqlite(): boolean;
  clearSqlite(): ConnectionConfig;

  getPostgresql(): PostgreSQLDatabaseConfig | undefined;
  setPostgresql(value?: PostgreSQLDatabaseConfig): ConnectionConfig;
  hasPostgresql(): boolean;
  clearPostgresql(): ConnectionConfig;

  getRetryOptions(): RetryOptions | undefined;
  setRetryOptions(value?: RetryOptions): ConnectionConfig;
  hasRetryOptions(): boolean;
  clearRetryOptions(): ConnectionConfig;

  getConfigCase(): ConnectionConfig.ConfigCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ConnectionConfig.AsObject;
  static toObject(includeInstance: boolean, msg: ConnectionConfig): ConnectionConfig.AsObject;
  static serializeBinaryToWriter(message: ConnectionConfig, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ConnectionConfig;
  static deserializeBinaryFromReader(
    message: ConnectionConfig,
    reader: jspb.BinaryReader,
  ): ConnectionConfig;
}

export namespace ConnectionConfig {
  export type AsObject = {
    fakeDatabase?: FakeDatabaseConfig.AsObject;
    mysql?: MySQLDatabaseConfig.AsObject;
    sqlite?: SqliteMetadataSourceConfig.AsObject;
    postgresql?: PostgreSQLDatabaseConfig.AsObject;
    retryOptions?: RetryOptions.AsObject;
  };

  export enum ConfigCase {
    CONFIG_NOT_SET = 0,
    FAKE_DATABASE = 1,
    MYSQL = 2,
    SQLITE = 3,
    POSTGRESQL = 5,
  }
}

export class GrpcChannelArguments extends jspb.Message {
  getMaxReceiveMessageLength(): number;
  setMaxReceiveMessageLength(value: number): GrpcChannelArguments;

  getHttp2MaxPingStrikes(): number;
  setHttp2MaxPingStrikes(value: number): GrpcChannelArguments;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GrpcChannelArguments.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GrpcChannelArguments,
  ): GrpcChannelArguments.AsObject;
  static serializeBinaryToWriter(message: GrpcChannelArguments, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GrpcChannelArguments;
  static deserializeBinaryFromReader(
    message: GrpcChannelArguments,
    reader: jspb.BinaryReader,
  ): GrpcChannelArguments;
}

export namespace GrpcChannelArguments {
  export type AsObject = {
    maxReceiveMessageLength: number;
    http2MaxPingStrikes: number;
  };
}

export class MetadataStoreClientConfig extends jspb.Message {
  getHost(): string;
  setHost(value: string): MetadataStoreClientConfig;

  getPort(): number;
  setPort(value: number): MetadataStoreClientConfig;

  getSslConfig(): MetadataStoreClientConfig.SSLConfig | undefined;
  setSslConfig(value?: MetadataStoreClientConfig.SSLConfig): MetadataStoreClientConfig;
  hasSslConfig(): boolean;
  clearSslConfig(): MetadataStoreClientConfig;

  getChannelArguments(): GrpcChannelArguments | undefined;
  setChannelArguments(value?: GrpcChannelArguments): MetadataStoreClientConfig;
  hasChannelArguments(): boolean;
  clearChannelArguments(): MetadataStoreClientConfig;

  getClientTimeoutSec(): number;
  setClientTimeoutSec(value: number): MetadataStoreClientConfig;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MetadataStoreClientConfig.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: MetadataStoreClientConfig,
  ): MetadataStoreClientConfig.AsObject;
  static serializeBinaryToWriter(
    message: MetadataStoreClientConfig,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): MetadataStoreClientConfig;
  static deserializeBinaryFromReader(
    message: MetadataStoreClientConfig,
    reader: jspb.BinaryReader,
  ): MetadataStoreClientConfig;
}

export namespace MetadataStoreClientConfig {
  export type AsObject = {
    host: string;
    port: number;
    sslConfig?: MetadataStoreClientConfig.SSLConfig.AsObject;
    channelArguments?: GrpcChannelArguments.AsObject;
    clientTimeoutSec: number;
  };

  export class SSLConfig extends jspb.Message {
    getClientKey(): string;
    setClientKey(value: string): SSLConfig;

    getServerCert(): string;
    setServerCert(value: string): SSLConfig;

    getCustomCa(): string;
    setCustomCa(value: string): SSLConfig;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SSLConfig.AsObject;
    static toObject(includeInstance: boolean, msg: SSLConfig): SSLConfig.AsObject;
    static serializeBinaryToWriter(message: SSLConfig, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SSLConfig;
    static deserializeBinaryFromReader(message: SSLConfig, reader: jspb.BinaryReader): SSLConfig;
  }

  export namespace SSLConfig {
    export type AsObject = {
      clientKey: string;
      serverCert: string;
      customCa: string;
    };
  }
}

export class MetadataStoreServerConfig extends jspb.Message {
  getConnectionConfig(): ConnectionConfig | undefined;
  setConnectionConfig(value?: ConnectionConfig): MetadataStoreServerConfig;
  hasConnectionConfig(): boolean;
  clearConnectionConfig(): MetadataStoreServerConfig;

  getMigrationOptions(): MigrationOptions | undefined;
  setMigrationOptions(value?: MigrationOptions): MetadataStoreServerConfig;
  hasMigrationOptions(): boolean;
  clearMigrationOptions(): MetadataStoreServerConfig;

  getSslConfig(): MetadataStoreServerConfig.SSLConfig | undefined;
  setSslConfig(value?: MetadataStoreServerConfig.SSLConfig): MetadataStoreServerConfig;
  hasSslConfig(): boolean;
  clearSslConfig(): MetadataStoreServerConfig;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MetadataStoreServerConfig.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: MetadataStoreServerConfig,
  ): MetadataStoreServerConfig.AsObject;
  static serializeBinaryToWriter(
    message: MetadataStoreServerConfig,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): MetadataStoreServerConfig;
  static deserializeBinaryFromReader(
    message: MetadataStoreServerConfig,
    reader: jspb.BinaryReader,
  ): MetadataStoreServerConfig;
}

export namespace MetadataStoreServerConfig {
  export type AsObject = {
    connectionConfig?: ConnectionConfig.AsObject;
    migrationOptions?: MigrationOptions.AsObject;
    sslConfig?: MetadataStoreServerConfig.SSLConfig.AsObject;
  };

  export class SSLConfig extends jspb.Message {
    getServerKey(): string;
    setServerKey(value: string): SSLConfig;

    getServerCert(): string;
    setServerCert(value: string): SSLConfig;

    getCustomCa(): string;
    setCustomCa(value: string): SSLConfig;

    getClientVerify(): boolean;
    setClientVerify(value: boolean): SSLConfig;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SSLConfig.AsObject;
    static toObject(includeInstance: boolean, msg: SSLConfig): SSLConfig.AsObject;
    static serializeBinaryToWriter(message: SSLConfig, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SSLConfig;
    static deserializeBinaryFromReader(message: SSLConfig, reader: jspb.BinaryReader): SSLConfig;
  }

  export namespace SSLConfig {
    export type AsObject = {
      serverKey: string;
      serverCert: string;
      customCa: string;
      clientVerify: boolean;
    };
  }
}

export class ListOperationOptions extends jspb.Message {
  getMaxResultSize(): number;
  setMaxResultSize(value: number): ListOperationOptions;

  getOrderByField(): ListOperationOptions.OrderByField | undefined;
  setOrderByField(value?: ListOperationOptions.OrderByField): ListOperationOptions;
  hasOrderByField(): boolean;
  clearOrderByField(): ListOperationOptions;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListOperationOptions;

  getFilterQuery(): string;
  setFilterQuery(value: string): ListOperationOptions;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListOperationOptions.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListOperationOptions,
  ): ListOperationOptions.AsObject;
  static serializeBinaryToWriter(message: ListOperationOptions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListOperationOptions;
  static deserializeBinaryFromReader(
    message: ListOperationOptions,
    reader: jspb.BinaryReader,
  ): ListOperationOptions;
}

export namespace ListOperationOptions {
  export type AsObject = {
    maxResultSize: number;
    orderByField?: ListOperationOptions.OrderByField.AsObject;
    nextPageToken: string;
    filterQuery: string;
  };

  export class OrderByField extends jspb.Message {
    getField(): ListOperationOptions.OrderByField.Field;
    setField(value: ListOperationOptions.OrderByField.Field): OrderByField;

    getIsAsc(): boolean;
    setIsAsc(value: boolean): OrderByField;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): OrderByField.AsObject;
    static toObject(includeInstance: boolean, msg: OrderByField): OrderByField.AsObject;
    static serializeBinaryToWriter(message: OrderByField, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): OrderByField;
    static deserializeBinaryFromReader(
      message: OrderByField,
      reader: jspb.BinaryReader,
    ): OrderByField;
  }

  export namespace OrderByField {
    export type AsObject = {
      field: ListOperationOptions.OrderByField.Field;
      isAsc: boolean;
    };

    export enum Field {
      FIELD_UNSPECIFIED = 0,
      CREATE_TIME = 1,
      LAST_UPDATE_TIME = 2,
      ID = 3,
    }
  }
}

export class ListOperationNextPageToken extends jspb.Message {
  getIdOffset(): number;
  setIdOffset(value: number): ListOperationNextPageToken;

  getFieldOffset(): number;
  setFieldOffset(value: number): ListOperationNextPageToken;

  getSetOptions(): ListOperationOptions | undefined;
  setSetOptions(value?: ListOperationOptions): ListOperationNextPageToken;
  hasSetOptions(): boolean;
  clearSetOptions(): ListOperationNextPageToken;

  getListedIdsList(): Array<number>;
  setListedIdsList(value: Array<number>): ListOperationNextPageToken;
  clearListedIdsList(): ListOperationNextPageToken;
  addListedIds(value: number, index?: number): ListOperationNextPageToken;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListOperationNextPageToken.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: ListOperationNextPageToken,
  ): ListOperationNextPageToken.AsObject;
  static serializeBinaryToWriter(
    message: ListOperationNextPageToken,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): ListOperationNextPageToken;
  static deserializeBinaryFromReader(
    message: ListOperationNextPageToken,
    reader: jspb.BinaryReader,
  ): ListOperationNextPageToken;
}

export namespace ListOperationNextPageToken {
  export type AsObject = {
    idOffset: number;
    fieldOffset: number;
    setOptions?: ListOperationOptions.AsObject;
    listedIdsList: Array<number>;
  };
}

export class TransactionOptions extends jspb.Message {
  getTag(): string;
  setTag(value: string): TransactionOptions;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TransactionOptions.AsObject;
  static toObject(includeInstance: boolean, msg: TransactionOptions): TransactionOptions.AsObject;
  static serializeBinaryToWriter(message: TransactionOptions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TransactionOptions;
  static deserializeBinaryFromReader(
    message: TransactionOptions,
    reader: jspb.BinaryReader,
  ): TransactionOptions;
}

export namespace TransactionOptions {
  export type AsObject = {
    tag: string;
  };
}

export class LineageGraphQueryOptions extends jspb.Message {
  getArtifactsOptions(): ListOperationOptions | undefined;
  setArtifactsOptions(value?: ListOperationOptions): LineageGraphQueryOptions;
  hasArtifactsOptions(): boolean;
  clearArtifactsOptions(): LineageGraphQueryOptions;

  getStopConditions(): LineageGraphQueryOptions.BoundaryConstraint | undefined;
  setStopConditions(value?: LineageGraphQueryOptions.BoundaryConstraint): LineageGraphQueryOptions;
  hasStopConditions(): boolean;
  clearStopConditions(): LineageGraphQueryOptions;

  getMaxNodeSize(): number;
  setMaxNodeSize(value: number): LineageGraphQueryOptions;

  getQueryNodesCase(): LineageGraphQueryOptions.QueryNodesCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LineageGraphQueryOptions.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: LineageGraphQueryOptions,
  ): LineageGraphQueryOptions.AsObject;
  static serializeBinaryToWriter(
    message: LineageGraphQueryOptions,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): LineageGraphQueryOptions;
  static deserializeBinaryFromReader(
    message: LineageGraphQueryOptions,
    reader: jspb.BinaryReader,
  ): LineageGraphQueryOptions;
}

export namespace LineageGraphQueryOptions {
  export type AsObject = {
    artifactsOptions?: ListOperationOptions.AsObject;
    stopConditions?: LineageGraphQueryOptions.BoundaryConstraint.AsObject;
    maxNodeSize: number;
  };

  export class BoundaryConstraint extends jspb.Message {
    getMaxNumHops(): number;
    setMaxNumHops(value: number): BoundaryConstraint;

    getBoundaryArtifacts(): string;
    setBoundaryArtifacts(value: string): BoundaryConstraint;

    getBoundaryExecutions(): string;
    setBoundaryExecutions(value: string): BoundaryConstraint;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): BoundaryConstraint.AsObject;
    static toObject(includeInstance: boolean, msg: BoundaryConstraint): BoundaryConstraint.AsObject;
    static serializeBinaryToWriter(message: BoundaryConstraint, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): BoundaryConstraint;
    static deserializeBinaryFromReader(
      message: BoundaryConstraint,
      reader: jspb.BinaryReader,
    ): BoundaryConstraint;
  }

  export namespace BoundaryConstraint {
    export type AsObject = {
      maxNumHops: number;
      boundaryArtifacts: string;
      boundaryExecutions: string;
    };
  }

  export enum QueryNodesCase {
    QUERY_NODES_NOT_SET = 0,
    ARTIFACTS_OPTIONS = 1,
  }
}

export class LineageSubgraphQueryOptions extends jspb.Message {
  getStartingArtifacts(): LineageSubgraphQueryOptions.StartingNodes | undefined;
  setStartingArtifacts(
    value?: LineageSubgraphQueryOptions.StartingNodes,
  ): LineageSubgraphQueryOptions;
  hasStartingArtifacts(): boolean;
  clearStartingArtifacts(): LineageSubgraphQueryOptions;

  getStartingExecutions(): LineageSubgraphQueryOptions.StartingNodes | undefined;
  setStartingExecutions(
    value?: LineageSubgraphQueryOptions.StartingNodes,
  ): LineageSubgraphQueryOptions;
  hasStartingExecutions(): boolean;
  clearStartingExecutions(): LineageSubgraphQueryOptions;

  getMaxNumHops(): number;
  setMaxNumHops(value: number): LineageSubgraphQueryOptions;

  getDirection(): LineageSubgraphQueryOptions.Direction;
  setDirection(value: LineageSubgraphQueryOptions.Direction): LineageSubgraphQueryOptions;

  getStartingNodesCase(): LineageSubgraphQueryOptions.StartingNodesCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LineageSubgraphQueryOptions.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: LineageSubgraphQueryOptions,
  ): LineageSubgraphQueryOptions.AsObject;
  static serializeBinaryToWriter(
    message: LineageSubgraphQueryOptions,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): LineageSubgraphQueryOptions;
  static deserializeBinaryFromReader(
    message: LineageSubgraphQueryOptions,
    reader: jspb.BinaryReader,
  ): LineageSubgraphQueryOptions;
}

export namespace LineageSubgraphQueryOptions {
  export type AsObject = {
    startingArtifacts?: LineageSubgraphQueryOptions.StartingNodes.AsObject;
    startingExecutions?: LineageSubgraphQueryOptions.StartingNodes.AsObject;
    maxNumHops: number;
    direction: LineageSubgraphQueryOptions.Direction;
  };

  export class StartingNodes extends jspb.Message {
    getFilterQuery(): string;
    setFilterQuery(value: string): StartingNodes;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): StartingNodes.AsObject;
    static toObject(includeInstance: boolean, msg: StartingNodes): StartingNodes.AsObject;
    static serializeBinaryToWriter(message: StartingNodes, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): StartingNodes;
    static deserializeBinaryFromReader(
      message: StartingNodes,
      reader: jspb.BinaryReader,
    ): StartingNodes;
  }

  export namespace StartingNodes {
    export type AsObject = {
      filterQuery: string;
    };
  }

  export enum Direction {
    DIRECTION_UNSPECIFIED = 0,
    UPSTREAM = 1,
    DOWNSTREAM = 2,
    BIDIRECTIONAL = 3,
  }

  export enum StartingNodesCase {
    STARTING_NODES_NOT_SET = 0,
    STARTING_ARTIFACTS = 1,
    STARTING_EXECUTIONS = 2,
  }
}

export enum PropertyType {
  UNKNOWN = 0,
  INT = 1,
  DOUBLE = 2,
  STRING = 3,
  STRUCT = 4,
  PROTO = 5,
  BOOLEAN = 6,
}
