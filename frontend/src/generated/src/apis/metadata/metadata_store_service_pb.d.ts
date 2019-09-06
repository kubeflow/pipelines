// package: ml_metadata
// file: src/apis/metadata/metadata_store_service.proto

import * as jspb from "google-protobuf";
import * as src_apis_metadata_metadata_store_pb from "./metadata_store_pb";

export class ArtifactAndType extends jspb.Message {
  hasArtifact(): boolean;
  clearArtifact(): void;
  getArtifact(): src_apis_metadata_metadata_store_pb.Artifact | undefined;
  setArtifact(value?: src_apis_metadata_metadata_store_pb.Artifact): void;

  hasType(): boolean;
  clearType(): void;
  getType(): src_apis_metadata_metadata_store_pb.ArtifactType | undefined;
  setType(value?: src_apis_metadata_metadata_store_pb.ArtifactType): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactAndType.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactAndType): ArtifactAndType.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ArtifactAndType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactAndType;
  static deserializeBinaryFromReader(message: ArtifactAndType, reader: jspb.BinaryReader): ArtifactAndType;
}

export namespace ArtifactAndType {
  export type AsObject = {
    artifact?: src_apis_metadata_metadata_store_pb.Artifact.AsObject,
    type?: src_apis_metadata_metadata_store_pb.ArtifactType.AsObject,
  }
}

export class ArtifactStructMap extends jspb.Message {
  getPropertiesMap(): jspb.Map<string, ArtifactStruct>;
  clearPropertiesMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactStructMap.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactStructMap): ArtifactStructMap.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ArtifactStructMap, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactStructMap;
  static deserializeBinaryFromReader(message: ArtifactStructMap, reader: jspb.BinaryReader): ArtifactStructMap;
}

export namespace ArtifactStructMap {
  export type AsObject = {
    propertiesMap: Array<[string, ArtifactStruct.AsObject]>,
  }
}

export class ArtifactStructList extends jspb.Message {
  clearElementsList(): void;
  getElementsList(): Array<ArtifactStruct>;
  setElementsList(value: Array<ArtifactStruct>): void;
  addElements(value?: ArtifactStruct, index?: number): ArtifactStruct;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactStructList.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactStructList): ArtifactStructList.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ArtifactStructList, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactStructList;
  static deserializeBinaryFromReader(message: ArtifactStructList, reader: jspb.BinaryReader): ArtifactStructList;
}

export namespace ArtifactStructList {
  export type AsObject = {
    elementsList: Array<ArtifactStruct.AsObject>,
  }
}

export class ArtifactStruct extends jspb.Message {
  hasArtifact(): boolean;
  clearArtifact(): void;
  getArtifact(): ArtifactAndType | undefined;
  setArtifact(value?: ArtifactAndType): void;

  hasMap(): boolean;
  clearMap(): void;
  getMap(): ArtifactStructMap | undefined;
  setMap(value?: ArtifactStructMap): void;

  hasList(): boolean;
  clearList(): void;
  getList(): ArtifactStructList | undefined;
  setList(value?: ArtifactStructList): void;

  getValueCase(): ArtifactStruct.ValueCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactStruct.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactStruct): ArtifactStruct.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ArtifactStruct, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactStruct;
  static deserializeBinaryFromReader(message: ArtifactStruct, reader: jspb.BinaryReader): ArtifactStruct;
}

export namespace ArtifactStruct {
  export type AsObject = {
    artifact?: ArtifactAndType.AsObject,
    map?: ArtifactStructMap.AsObject,
    list?: ArtifactStructList.AsObject,
  }

  export enum ValueCase {
    VALUE_NOT_SET = 0,
    ARTIFACT = 1,
    MAP = 2,
    LIST = 3,
  }
}

export class PutArtifactsRequest extends jspb.Message {
  clearArtifactsList(): void;
  getArtifactsList(): Array<src_apis_metadata_metadata_store_pb.Artifact>;
  setArtifactsList(value: Array<src_apis_metadata_metadata_store_pb.Artifact>): void;
  addArtifacts(value?: src_apis_metadata_metadata_store_pb.Artifact, index?: number): src_apis_metadata_metadata_store_pb.Artifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutArtifactsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutArtifactsRequest): PutArtifactsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutArtifactsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutArtifactsRequest;
  static deserializeBinaryFromReader(message: PutArtifactsRequest, reader: jspb.BinaryReader): PutArtifactsRequest;
}

export namespace PutArtifactsRequest {
  export type AsObject = {
    artifactsList: Array<src_apis_metadata_metadata_store_pb.Artifact.AsObject>,
  }
}

export class PutArtifactsResponse extends jspb.Message {
  clearArtifactIdsList(): void;
  getArtifactIdsList(): Array<number>;
  setArtifactIdsList(value: Array<number>): void;
  addArtifactIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutArtifactsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutArtifactsResponse): PutArtifactsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutArtifactsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutArtifactsResponse;
  static deserializeBinaryFromReader(message: PutArtifactsResponse, reader: jspb.BinaryReader): PutArtifactsResponse;
}

export namespace PutArtifactsResponse {
  export type AsObject = {
    artifactIdsList: Array<number>,
  }
}

export class PutArtifactTypeRequest extends jspb.Message {
  hasArtifactType(): boolean;
  clearArtifactType(): void;
  getArtifactType(): src_apis_metadata_metadata_store_pb.ArtifactType | undefined;
  setArtifactType(value?: src_apis_metadata_metadata_store_pb.ArtifactType): void;

  hasCanAddFields(): boolean;
  clearCanAddFields(): void;
  getCanAddFields(): boolean | undefined;
  setCanAddFields(value: boolean): void;

  hasCanDeleteFields(): boolean;
  clearCanDeleteFields(): void;
  getCanDeleteFields(): boolean | undefined;
  setCanDeleteFields(value: boolean): void;

  hasAllFieldsMatch(): boolean;
  clearAllFieldsMatch(): void;
  getAllFieldsMatch(): boolean | undefined;
  setAllFieldsMatch(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutArtifactTypeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutArtifactTypeRequest): PutArtifactTypeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutArtifactTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutArtifactTypeRequest;
  static deserializeBinaryFromReader(message: PutArtifactTypeRequest, reader: jspb.BinaryReader): PutArtifactTypeRequest;
}

export namespace PutArtifactTypeRequest {
  export type AsObject = {
    artifactType?: src_apis_metadata_metadata_store_pb.ArtifactType.AsObject,
    canAddFields?: boolean,
    canDeleteFields?: boolean,
    allFieldsMatch?: boolean,
  }
}

export class PutArtifactTypeResponse extends jspb.Message {
  hasTypeId(): boolean;
  clearTypeId(): void;
  getTypeId(): number | undefined;
  setTypeId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutArtifactTypeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutArtifactTypeResponse): PutArtifactTypeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutArtifactTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutArtifactTypeResponse;
  static deserializeBinaryFromReader(message: PutArtifactTypeResponse, reader: jspb.BinaryReader): PutArtifactTypeResponse;
}

export namespace PutArtifactTypeResponse {
  export type AsObject = {
    typeId?: number,
  }
}

export class PutExecutionsRequest extends jspb.Message {
  clearExecutionsList(): void;
  getExecutionsList(): Array<src_apis_metadata_metadata_store_pb.Execution>;
  setExecutionsList(value: Array<src_apis_metadata_metadata_store_pb.Execution>): void;
  addExecutions(value?: src_apis_metadata_metadata_store_pb.Execution, index?: number): src_apis_metadata_metadata_store_pb.Execution;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutExecutionsRequest): PutExecutionsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutExecutionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionsRequest;
  static deserializeBinaryFromReader(message: PutExecutionsRequest, reader: jspb.BinaryReader): PutExecutionsRequest;
}

export namespace PutExecutionsRequest {
  export type AsObject = {
    executionsList: Array<src_apis_metadata_metadata_store_pb.Execution.AsObject>,
  }
}

export class PutExecutionsResponse extends jspb.Message {
  clearExecutionIdsList(): void;
  getExecutionIdsList(): Array<number>;
  setExecutionIdsList(value: Array<number>): void;
  addExecutionIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutExecutionsResponse): PutExecutionsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutExecutionsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionsResponse;
  static deserializeBinaryFromReader(message: PutExecutionsResponse, reader: jspb.BinaryReader): PutExecutionsResponse;
}

export namespace PutExecutionsResponse {
  export type AsObject = {
    executionIdsList: Array<number>,
  }
}

export class PutExecutionTypeRequest extends jspb.Message {
  hasExecutionType(): boolean;
  clearExecutionType(): void;
  getExecutionType(): src_apis_metadata_metadata_store_pb.ExecutionType | undefined;
  setExecutionType(value?: src_apis_metadata_metadata_store_pb.ExecutionType): void;

  hasCanAddFields(): boolean;
  clearCanAddFields(): void;
  getCanAddFields(): boolean | undefined;
  setCanAddFields(value: boolean): void;

  hasCanDeleteFields(): boolean;
  clearCanDeleteFields(): void;
  getCanDeleteFields(): boolean | undefined;
  setCanDeleteFields(value: boolean): void;

  hasAllFieldsMatch(): boolean;
  clearAllFieldsMatch(): void;
  getAllFieldsMatch(): boolean | undefined;
  setAllFieldsMatch(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionTypeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutExecutionTypeRequest): PutExecutionTypeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutExecutionTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionTypeRequest;
  static deserializeBinaryFromReader(message: PutExecutionTypeRequest, reader: jspb.BinaryReader): PutExecutionTypeRequest;
}

export namespace PutExecutionTypeRequest {
  export type AsObject = {
    executionType?: src_apis_metadata_metadata_store_pb.ExecutionType.AsObject,
    canAddFields?: boolean,
    canDeleteFields?: boolean,
    allFieldsMatch?: boolean,
  }
}

export class PutExecutionTypeResponse extends jspb.Message {
  hasTypeId(): boolean;
  clearTypeId(): void;
  getTypeId(): number | undefined;
  setTypeId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionTypeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutExecutionTypeResponse): PutExecutionTypeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutExecutionTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionTypeResponse;
  static deserializeBinaryFromReader(message: PutExecutionTypeResponse, reader: jspb.BinaryReader): PutExecutionTypeResponse;
}

export namespace PutExecutionTypeResponse {
  export type AsObject = {
    typeId?: number,
  }
}

export class PutEventsRequest extends jspb.Message {
  clearEventsList(): void;
  getEventsList(): Array<src_apis_metadata_metadata_store_pb.Event>;
  setEventsList(value: Array<src_apis_metadata_metadata_store_pb.Event>): void;
  addEvents(value?: src_apis_metadata_metadata_store_pb.Event, index?: number): src_apis_metadata_metadata_store_pb.Event;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutEventsRequest): PutEventsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutEventsRequest;
  static deserializeBinaryFromReader(message: PutEventsRequest, reader: jspb.BinaryReader): PutEventsRequest;
}

export namespace PutEventsRequest {
  export type AsObject = {
    eventsList: Array<src_apis_metadata_metadata_store_pb.Event.AsObject>,
  }
}

export class PutEventsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutEventsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutEventsResponse): PutEventsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutEventsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutEventsResponse;
  static deserializeBinaryFromReader(message: PutEventsResponse, reader: jspb.BinaryReader): PutEventsResponse;
}

export namespace PutEventsResponse {
  export type AsObject = {
  }
}

export class PutExecutionRequest extends jspb.Message {
  hasExecution(): boolean;
  clearExecution(): void;
  getExecution(): src_apis_metadata_metadata_store_pb.Execution | undefined;
  setExecution(value?: src_apis_metadata_metadata_store_pb.Execution): void;

  clearArtifactEventPairsList(): void;
  getArtifactEventPairsList(): Array<PutExecutionRequest.ArtifactAndEvent>;
  setArtifactEventPairsList(value: Array<PutExecutionRequest.ArtifactAndEvent>): void;
  addArtifactEventPairs(value?: PutExecutionRequest.ArtifactAndEvent, index?: number): PutExecutionRequest.ArtifactAndEvent;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutExecutionRequest): PutExecutionRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutExecutionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionRequest;
  static deserializeBinaryFromReader(message: PutExecutionRequest, reader: jspb.BinaryReader): PutExecutionRequest;
}

export namespace PutExecutionRequest {
  export type AsObject = {
    execution?: src_apis_metadata_metadata_store_pb.Execution.AsObject,
    artifactEventPairsList: Array<PutExecutionRequest.ArtifactAndEvent.AsObject>,
  }

  export class ArtifactAndEvent extends jspb.Message {
    hasArtifact(): boolean;
    clearArtifact(): void;
    getArtifact(): src_apis_metadata_metadata_store_pb.Artifact | undefined;
    setArtifact(value?: src_apis_metadata_metadata_store_pb.Artifact): void;

    hasEvent(): boolean;
    clearEvent(): void;
    getEvent(): src_apis_metadata_metadata_store_pb.Event | undefined;
    setEvent(value?: src_apis_metadata_metadata_store_pb.Event): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ArtifactAndEvent.AsObject;
    static toObject(includeInstance: boolean, msg: ArtifactAndEvent): ArtifactAndEvent.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ArtifactAndEvent, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ArtifactAndEvent;
    static deserializeBinaryFromReader(message: ArtifactAndEvent, reader: jspb.BinaryReader): ArtifactAndEvent;
  }

  export namespace ArtifactAndEvent {
    export type AsObject = {
      artifact?: src_apis_metadata_metadata_store_pb.Artifact.AsObject,
      event?: src_apis_metadata_metadata_store_pb.Event.AsObject,
    }
  }
}

export class PutExecutionResponse extends jspb.Message {
  hasExecutionId(): boolean;
  clearExecutionId(): void;
  getExecutionId(): number | undefined;
  setExecutionId(value: number): void;

  clearArtifactIdsList(): void;
  getArtifactIdsList(): Array<number>;
  setArtifactIdsList(value: Array<number>): void;
  addArtifactIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutExecutionResponse): PutExecutionResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutExecutionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionResponse;
  static deserializeBinaryFromReader(message: PutExecutionResponse, reader: jspb.BinaryReader): PutExecutionResponse;
}

export namespace PutExecutionResponse {
  export type AsObject = {
    executionId?: number,
    artifactIdsList: Array<number>,
  }
}

export class PutTypesRequest extends jspb.Message {
  clearArtifactTypesList(): void;
  getArtifactTypesList(): Array<src_apis_metadata_metadata_store_pb.ArtifactType>;
  setArtifactTypesList(value: Array<src_apis_metadata_metadata_store_pb.ArtifactType>): void;
  addArtifactTypes(value?: src_apis_metadata_metadata_store_pb.ArtifactType, index?: number): src_apis_metadata_metadata_store_pb.ArtifactType;

  clearExecutionTypesList(): void;
  getExecutionTypesList(): Array<src_apis_metadata_metadata_store_pb.ExecutionType>;
  setExecutionTypesList(value: Array<src_apis_metadata_metadata_store_pb.ExecutionType>): void;
  addExecutionTypes(value?: src_apis_metadata_metadata_store_pb.ExecutionType, index?: number): src_apis_metadata_metadata_store_pb.ExecutionType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutTypesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutTypesRequest): PutTypesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutTypesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutTypesRequest;
  static deserializeBinaryFromReader(message: PutTypesRequest, reader: jspb.BinaryReader): PutTypesRequest;
}

export namespace PutTypesRequest {
  export type AsObject = {
    artifactTypesList: Array<src_apis_metadata_metadata_store_pb.ArtifactType.AsObject>,
    executionTypesList: Array<src_apis_metadata_metadata_store_pb.ExecutionType.AsObject>,
  }
}

export class PutTypesResponse extends jspb.Message {
  clearArtifactTypeIdsList(): void;
  getArtifactTypeIdsList(): Array<number>;
  setArtifactTypeIdsList(value: Array<number>): void;
  addArtifactTypeIds(value: number, index?: number): number;

  clearExecutionTypeIdsList(): void;
  getExecutionTypeIdsList(): Array<number>;
  setExecutionTypeIdsList(value: Array<number>): void;
  addExecutionTypeIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutTypesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutTypesResponse): PutTypesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutTypesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutTypesResponse;
  static deserializeBinaryFromReader(message: PutTypesResponse, reader: jspb.BinaryReader): PutTypesResponse;
}

export namespace PutTypesResponse {
  export type AsObject = {
    artifactTypeIdsList: Array<number>,
    executionTypeIdsList: Array<number>,
  }
}

export class PutContextTypeRequest extends jspb.Message {
  hasContextType(): boolean;
  clearContextType(): void;
  getContextType(): src_apis_metadata_metadata_store_pb.ContextType | undefined;
  setContextType(value?: src_apis_metadata_metadata_store_pb.ContextType): void;

  hasCanAddFields(): boolean;
  clearCanAddFields(): void;
  getCanAddFields(): boolean | undefined;
  setCanAddFields(value: boolean): void;

  hasCanDeleteFields(): boolean;
  clearCanDeleteFields(): void;
  getCanDeleteFields(): boolean | undefined;
  setCanDeleteFields(value: boolean): void;

  hasAllFieldsMatch(): boolean;
  clearAllFieldsMatch(): void;
  getAllFieldsMatch(): boolean | undefined;
  setAllFieldsMatch(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutContextTypeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutContextTypeRequest): PutContextTypeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutContextTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutContextTypeRequest;
  static deserializeBinaryFromReader(message: PutContextTypeRequest, reader: jspb.BinaryReader): PutContextTypeRequest;
}

export namespace PutContextTypeRequest {
  export type AsObject = {
    contextType?: src_apis_metadata_metadata_store_pb.ContextType.AsObject,
    canAddFields?: boolean,
    canDeleteFields?: boolean,
    allFieldsMatch?: boolean,
  }
}

export class PutContextTypeResponse extends jspb.Message {
  hasTypeId(): boolean;
  clearTypeId(): void;
  getTypeId(): number | undefined;
  setTypeId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutContextTypeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutContextTypeResponse): PutContextTypeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutContextTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutContextTypeResponse;
  static deserializeBinaryFromReader(message: PutContextTypeResponse, reader: jspb.BinaryReader): PutContextTypeResponse;
}

export namespace PutContextTypeResponse {
  export type AsObject = {
    typeId?: number,
  }
}

export class PutContextsRequest extends jspb.Message {
  clearContextsList(): void;
  getContextsList(): Array<src_apis_metadata_metadata_store_pb.Context>;
  setContextsList(value: Array<src_apis_metadata_metadata_store_pb.Context>): void;
  addContexts(value?: src_apis_metadata_metadata_store_pb.Context, index?: number): src_apis_metadata_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutContextsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutContextsRequest): PutContextsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutContextsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutContextsRequest;
  static deserializeBinaryFromReader(message: PutContextsRequest, reader: jspb.BinaryReader): PutContextsRequest;
}

export namespace PutContextsRequest {
  export type AsObject = {
    contextsList: Array<src_apis_metadata_metadata_store_pb.Context.AsObject>,
  }
}

export class PutContextsResponse extends jspb.Message {
  clearContextIdsList(): void;
  getContextIdsList(): Array<number>;
  setContextIdsList(value: Array<number>): void;
  addContextIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutContextsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutContextsResponse): PutContextsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutContextsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutContextsResponse;
  static deserializeBinaryFromReader(message: PutContextsResponse, reader: jspb.BinaryReader): PutContextsResponse;
}

export namespace PutContextsResponse {
  export type AsObject = {
    contextIdsList: Array<number>,
  }
}

export class PutAttributionsAndAssociationsRequest extends jspb.Message {
  clearAttributionsList(): void;
  getAttributionsList(): Array<src_apis_metadata_metadata_store_pb.Attribution>;
  setAttributionsList(value: Array<src_apis_metadata_metadata_store_pb.Attribution>): void;
  addAttributions(value?: src_apis_metadata_metadata_store_pb.Attribution, index?: number): src_apis_metadata_metadata_store_pb.Attribution;

  clearAssociationsList(): void;
  getAssociationsList(): Array<src_apis_metadata_metadata_store_pb.Association>;
  setAssociationsList(value: Array<src_apis_metadata_metadata_store_pb.Association>): void;
  addAssociations(value?: src_apis_metadata_metadata_store_pb.Association, index?: number): src_apis_metadata_metadata_store_pb.Association;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutAttributionsAndAssociationsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutAttributionsAndAssociationsRequest): PutAttributionsAndAssociationsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutAttributionsAndAssociationsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutAttributionsAndAssociationsRequest;
  static deserializeBinaryFromReader(message: PutAttributionsAndAssociationsRequest, reader: jspb.BinaryReader): PutAttributionsAndAssociationsRequest;
}

export namespace PutAttributionsAndAssociationsRequest {
  export type AsObject = {
    attributionsList: Array<src_apis_metadata_metadata_store_pb.Attribution.AsObject>,
    associationsList: Array<src_apis_metadata_metadata_store_pb.Association.AsObject>,
  }
}

export class PutAttributionsAndAssociationsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutAttributionsAndAssociationsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutAttributionsAndAssociationsResponse): PutAttributionsAndAssociationsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutAttributionsAndAssociationsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutAttributionsAndAssociationsResponse;
  static deserializeBinaryFromReader(message: PutAttributionsAndAssociationsResponse, reader: jspb.BinaryReader): PutAttributionsAndAssociationsResponse;
}

export namespace PutAttributionsAndAssociationsResponse {
  export type AsObject = {
  }
}

export class PutParentContextsRequest extends jspb.Message {
  clearParentContextsList(): void;
  getParentContextsList(): Array<src_apis_metadata_metadata_store_pb.ParentContext>;
  setParentContextsList(value: Array<src_apis_metadata_metadata_store_pb.ParentContext>): void;
  addParentContexts(value?: src_apis_metadata_metadata_store_pb.ParentContext, index?: number): src_apis_metadata_metadata_store_pb.ParentContext;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutParentContextsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutParentContextsRequest): PutParentContextsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutParentContextsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutParentContextsRequest;
  static deserializeBinaryFromReader(message: PutParentContextsRequest, reader: jspb.BinaryReader): PutParentContextsRequest;
}

export namespace PutParentContextsRequest {
  export type AsObject = {
    parentContextsList: Array<src_apis_metadata_metadata_store_pb.ParentContext.AsObject>,
  }
}

export class PutParentContextsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutParentContextsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutParentContextsResponse): PutParentContextsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PutParentContextsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutParentContextsResponse;
  static deserializeBinaryFromReader(message: PutParentContextsResponse, reader: jspb.BinaryReader): PutParentContextsResponse;
}

export namespace PutParentContextsResponse {
  export type AsObject = {
  }
}

export class GetArtifactsByTypeRequest extends jspb.Message {
  hasTypeName(): boolean;
  clearTypeName(): void;
  getTypeName(): string | undefined;
  setTypeName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByTypeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsByTypeRequest): GetArtifactsByTypeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsByTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByTypeRequest;
  static deserializeBinaryFromReader(message: GetArtifactsByTypeRequest, reader: jspb.BinaryReader): GetArtifactsByTypeRequest;
}

export namespace GetArtifactsByTypeRequest {
  export type AsObject = {
    typeName?: string,
  }
}

export class GetArtifactsByTypeResponse extends jspb.Message {
  clearArtifactsList(): void;
  getArtifactsList(): Array<src_apis_metadata_metadata_store_pb.Artifact>;
  setArtifactsList(value: Array<src_apis_metadata_metadata_store_pb.Artifact>): void;
  addArtifacts(value?: src_apis_metadata_metadata_store_pb.Artifact, index?: number): src_apis_metadata_metadata_store_pb.Artifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByTypeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsByTypeResponse): GetArtifactsByTypeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsByTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByTypeResponse;
  static deserializeBinaryFromReader(message: GetArtifactsByTypeResponse, reader: jspb.BinaryReader): GetArtifactsByTypeResponse;
}

export namespace GetArtifactsByTypeResponse {
  export type AsObject = {
    artifactsList: Array<src_apis_metadata_metadata_store_pb.Artifact.AsObject>,
  }
}

export class GetArtifactsByIDRequest extends jspb.Message {
  clearArtifactIdsList(): void;
  getArtifactIdsList(): Array<number>;
  setArtifactIdsList(value: Array<number>): void;
  addArtifactIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByIDRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsByIDRequest): GetArtifactsByIDRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsByIDRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByIDRequest;
  static deserializeBinaryFromReader(message: GetArtifactsByIDRequest, reader: jspb.BinaryReader): GetArtifactsByIDRequest;
}

export namespace GetArtifactsByIDRequest {
  export type AsObject = {
    artifactIdsList: Array<number>,
  }
}

export class GetArtifactsByIDResponse extends jspb.Message {
  clearArtifactsList(): void;
  getArtifactsList(): Array<src_apis_metadata_metadata_store_pb.Artifact>;
  setArtifactsList(value: Array<src_apis_metadata_metadata_store_pb.Artifact>): void;
  addArtifacts(value?: src_apis_metadata_metadata_store_pb.Artifact, index?: number): src_apis_metadata_metadata_store_pb.Artifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByIDResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsByIDResponse): GetArtifactsByIDResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsByIDResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByIDResponse;
  static deserializeBinaryFromReader(message: GetArtifactsByIDResponse, reader: jspb.BinaryReader): GetArtifactsByIDResponse;
}

export namespace GetArtifactsByIDResponse {
  export type AsObject = {
    artifactsList: Array<src_apis_metadata_metadata_store_pb.Artifact.AsObject>,
  }
}

export class GetArtifactsRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsRequest): GetArtifactsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsRequest;
  static deserializeBinaryFromReader(message: GetArtifactsRequest, reader: jspb.BinaryReader): GetArtifactsRequest;
}

export namespace GetArtifactsRequest {
  export type AsObject = {
  }
}

export class GetArtifactsResponse extends jspb.Message {
  clearArtifactsList(): void;
  getArtifactsList(): Array<src_apis_metadata_metadata_store_pb.Artifact>;
  setArtifactsList(value: Array<src_apis_metadata_metadata_store_pb.Artifact>): void;
  addArtifacts(value?: src_apis_metadata_metadata_store_pb.Artifact, index?: number): src_apis_metadata_metadata_store_pb.Artifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsResponse): GetArtifactsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsResponse;
  static deserializeBinaryFromReader(message: GetArtifactsResponse, reader: jspb.BinaryReader): GetArtifactsResponse;
}

export namespace GetArtifactsResponse {
  export type AsObject = {
    artifactsList: Array<src_apis_metadata_metadata_store_pb.Artifact.AsObject>,
  }
}

export class GetArtifactsByURIRequest extends jspb.Message {
  hasUri(): boolean;
  clearUri(): void;
  getUri(): string | undefined;
  setUri(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByURIRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsByURIRequest): GetArtifactsByURIRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsByURIRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByURIRequest;
  static deserializeBinaryFromReader(message: GetArtifactsByURIRequest, reader: jspb.BinaryReader): GetArtifactsByURIRequest;
}

export namespace GetArtifactsByURIRequest {
  export type AsObject = {
    uri?: string,
  }
}

export class GetArtifactsByURIResponse extends jspb.Message {
  clearArtifactsList(): void;
  getArtifactsList(): Array<src_apis_metadata_metadata_store_pb.Artifact>;
  setArtifactsList(value: Array<src_apis_metadata_metadata_store_pb.Artifact>): void;
  addArtifacts(value?: src_apis_metadata_metadata_store_pb.Artifact, index?: number): src_apis_metadata_metadata_store_pb.Artifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByURIResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsByURIResponse): GetArtifactsByURIResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsByURIResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByURIResponse;
  static deserializeBinaryFromReader(message: GetArtifactsByURIResponse, reader: jspb.BinaryReader): GetArtifactsByURIResponse;
}

export namespace GetArtifactsByURIResponse {
  export type AsObject = {
    artifactsList: Array<src_apis_metadata_metadata_store_pb.Artifact.AsObject>,
  }
}

export class GetExecutionsRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionsRequest): GetExecutionsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsRequest;
  static deserializeBinaryFromReader(message: GetExecutionsRequest, reader: jspb.BinaryReader): GetExecutionsRequest;
}

export namespace GetExecutionsRequest {
  export type AsObject = {
  }
}

export class GetExecutionsResponse extends jspb.Message {
  clearExecutionsList(): void;
  getExecutionsList(): Array<src_apis_metadata_metadata_store_pb.Execution>;
  setExecutionsList(value: Array<src_apis_metadata_metadata_store_pb.Execution>): void;
  addExecutions(value?: src_apis_metadata_metadata_store_pb.Execution, index?: number): src_apis_metadata_metadata_store_pb.Execution;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionsResponse): GetExecutionsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsResponse;
  static deserializeBinaryFromReader(message: GetExecutionsResponse, reader: jspb.BinaryReader): GetExecutionsResponse;
}

export namespace GetExecutionsResponse {
  export type AsObject = {
    executionsList: Array<src_apis_metadata_metadata_store_pb.Execution.AsObject>,
  }
}

export class GetArtifactTypeRequest extends jspb.Message {
  hasTypeName(): boolean;
  clearTypeName(): void;
  getTypeName(): string | undefined;
  setTypeName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactTypeRequest): GetArtifactTypeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypeRequest;
  static deserializeBinaryFromReader(message: GetArtifactTypeRequest, reader: jspb.BinaryReader): GetArtifactTypeRequest;
}

export namespace GetArtifactTypeRequest {
  export type AsObject = {
    typeName?: string,
  }
}

export class GetArtifactTypeResponse extends jspb.Message {
  hasArtifactType(): boolean;
  clearArtifactType(): void;
  getArtifactType(): src_apis_metadata_metadata_store_pb.ArtifactType | undefined;
  setArtifactType(value?: src_apis_metadata_metadata_store_pb.ArtifactType): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactTypeResponse): GetArtifactTypeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypeResponse;
  static deserializeBinaryFromReader(message: GetArtifactTypeResponse, reader: jspb.BinaryReader): GetArtifactTypeResponse;
}

export namespace GetArtifactTypeResponse {
  export type AsObject = {
    artifactType?: src_apis_metadata_metadata_store_pb.ArtifactType.AsObject,
  }
}

export class GetArtifactTypesRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactTypesRequest): GetArtifactTypesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactTypesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesRequest;
  static deserializeBinaryFromReader(message: GetArtifactTypesRequest, reader: jspb.BinaryReader): GetArtifactTypesRequest;
}

export namespace GetArtifactTypesRequest {
  export type AsObject = {
  }
}

export class GetArtifactTypesResponse extends jspb.Message {
  clearArtifactTypesList(): void;
  getArtifactTypesList(): Array<src_apis_metadata_metadata_store_pb.ArtifactType>;
  setArtifactTypesList(value: Array<src_apis_metadata_metadata_store_pb.ArtifactType>): void;
  addArtifactTypes(value?: src_apis_metadata_metadata_store_pb.ArtifactType, index?: number): src_apis_metadata_metadata_store_pb.ArtifactType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactTypesResponse): GetArtifactTypesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactTypesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesResponse;
  static deserializeBinaryFromReader(message: GetArtifactTypesResponse, reader: jspb.BinaryReader): GetArtifactTypesResponse;
}

export namespace GetArtifactTypesResponse {
  export type AsObject = {
    artifactTypesList: Array<src_apis_metadata_metadata_store_pb.ArtifactType.AsObject>,
  }
}

export class GetExecutionTypesRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionTypesRequest): GetExecutionTypesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionTypesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesRequest;
  static deserializeBinaryFromReader(message: GetExecutionTypesRequest, reader: jspb.BinaryReader): GetExecutionTypesRequest;
}

export namespace GetExecutionTypesRequest {
  export type AsObject = {
  }
}

export class GetExecutionTypesResponse extends jspb.Message {
  clearExecutionTypesList(): void;
  getExecutionTypesList(): Array<src_apis_metadata_metadata_store_pb.ExecutionType>;
  setExecutionTypesList(value: Array<src_apis_metadata_metadata_store_pb.ExecutionType>): void;
  addExecutionTypes(value?: src_apis_metadata_metadata_store_pb.ExecutionType, index?: number): src_apis_metadata_metadata_store_pb.ExecutionType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionTypesResponse): GetExecutionTypesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionTypesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesResponse;
  static deserializeBinaryFromReader(message: GetExecutionTypesResponse, reader: jspb.BinaryReader): GetExecutionTypesResponse;
}

export namespace GetExecutionTypesResponse {
  export type AsObject = {
    executionTypesList: Array<src_apis_metadata_metadata_store_pb.ExecutionType.AsObject>,
  }
}

export class GetExecutionsByTypeRequest extends jspb.Message {
  hasTypeName(): boolean;
  clearTypeName(): void;
  getTypeName(): string | undefined;
  setTypeName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByTypeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionsByTypeRequest): GetExecutionsByTypeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionsByTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByTypeRequest;
  static deserializeBinaryFromReader(message: GetExecutionsByTypeRequest, reader: jspb.BinaryReader): GetExecutionsByTypeRequest;
}

export namespace GetExecutionsByTypeRequest {
  export type AsObject = {
    typeName?: string,
  }
}

export class GetExecutionsByTypeResponse extends jspb.Message {
  clearExecutionsList(): void;
  getExecutionsList(): Array<src_apis_metadata_metadata_store_pb.Execution>;
  setExecutionsList(value: Array<src_apis_metadata_metadata_store_pb.Execution>): void;
  addExecutions(value?: src_apis_metadata_metadata_store_pb.Execution, index?: number): src_apis_metadata_metadata_store_pb.Execution;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByTypeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionsByTypeResponse): GetExecutionsByTypeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionsByTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByTypeResponse;
  static deserializeBinaryFromReader(message: GetExecutionsByTypeResponse, reader: jspb.BinaryReader): GetExecutionsByTypeResponse;
}

export namespace GetExecutionsByTypeResponse {
  export type AsObject = {
    executionsList: Array<src_apis_metadata_metadata_store_pb.Execution.AsObject>,
  }
}

export class GetExecutionsByIDRequest extends jspb.Message {
  clearExecutionIdsList(): void;
  getExecutionIdsList(): Array<number>;
  setExecutionIdsList(value: Array<number>): void;
  addExecutionIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByIDRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionsByIDRequest): GetExecutionsByIDRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionsByIDRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByIDRequest;
  static deserializeBinaryFromReader(message: GetExecutionsByIDRequest, reader: jspb.BinaryReader): GetExecutionsByIDRequest;
}

export namespace GetExecutionsByIDRequest {
  export type AsObject = {
    executionIdsList: Array<number>,
  }
}

export class GetExecutionsByIDResponse extends jspb.Message {
  clearExecutionsList(): void;
  getExecutionsList(): Array<src_apis_metadata_metadata_store_pb.Execution>;
  setExecutionsList(value: Array<src_apis_metadata_metadata_store_pb.Execution>): void;
  addExecutions(value?: src_apis_metadata_metadata_store_pb.Execution, index?: number): src_apis_metadata_metadata_store_pb.Execution;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByIDResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionsByIDResponse): GetExecutionsByIDResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionsByIDResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByIDResponse;
  static deserializeBinaryFromReader(message: GetExecutionsByIDResponse, reader: jspb.BinaryReader): GetExecutionsByIDResponse;
}

export namespace GetExecutionsByIDResponse {
  export type AsObject = {
    executionsList: Array<src_apis_metadata_metadata_store_pb.Execution.AsObject>,
  }
}

export class GetExecutionTypeRequest extends jspb.Message {
  hasTypeName(): boolean;
  clearTypeName(): void;
  getTypeName(): string | undefined;
  setTypeName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionTypeRequest): GetExecutionTypeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypeRequest;
  static deserializeBinaryFromReader(message: GetExecutionTypeRequest, reader: jspb.BinaryReader): GetExecutionTypeRequest;
}

export namespace GetExecutionTypeRequest {
  export type AsObject = {
    typeName?: string,
  }
}

export class GetExecutionTypeResponse extends jspb.Message {
  hasExecutionType(): boolean;
  clearExecutionType(): void;
  getExecutionType(): src_apis_metadata_metadata_store_pb.ExecutionType | undefined;
  setExecutionType(value?: src_apis_metadata_metadata_store_pb.ExecutionType): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionTypeResponse): GetExecutionTypeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypeResponse;
  static deserializeBinaryFromReader(message: GetExecutionTypeResponse, reader: jspb.BinaryReader): GetExecutionTypeResponse;
}

export namespace GetExecutionTypeResponse {
  export type AsObject = {
    executionType?: src_apis_metadata_metadata_store_pb.ExecutionType.AsObject,
  }
}

export class GetEventsByExecutionIDsRequest extends jspb.Message {
  clearExecutionIdsList(): void;
  getExecutionIdsList(): Array<number>;
  setExecutionIdsList(value: Array<number>): void;
  addExecutionIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEventsByExecutionIDsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetEventsByExecutionIDsRequest): GetEventsByExecutionIDsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetEventsByExecutionIDsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEventsByExecutionIDsRequest;
  static deserializeBinaryFromReader(message: GetEventsByExecutionIDsRequest, reader: jspb.BinaryReader): GetEventsByExecutionIDsRequest;
}

export namespace GetEventsByExecutionIDsRequest {
  export type AsObject = {
    executionIdsList: Array<number>,
  }
}

export class GetEventsByExecutionIDsResponse extends jspb.Message {
  clearEventsList(): void;
  getEventsList(): Array<src_apis_metadata_metadata_store_pb.Event>;
  setEventsList(value: Array<src_apis_metadata_metadata_store_pb.Event>): void;
  addEvents(value?: src_apis_metadata_metadata_store_pb.Event, index?: number): src_apis_metadata_metadata_store_pb.Event;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEventsByExecutionIDsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetEventsByExecutionIDsResponse): GetEventsByExecutionIDsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetEventsByExecutionIDsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEventsByExecutionIDsResponse;
  static deserializeBinaryFromReader(message: GetEventsByExecutionIDsResponse, reader: jspb.BinaryReader): GetEventsByExecutionIDsResponse;
}

export namespace GetEventsByExecutionIDsResponse {
  export type AsObject = {
    eventsList: Array<src_apis_metadata_metadata_store_pb.Event.AsObject>,
  }
}

export class GetEventsByArtifactIDsRequest extends jspb.Message {
  clearArtifactIdsList(): void;
  getArtifactIdsList(): Array<number>;
  setArtifactIdsList(value: Array<number>): void;
  addArtifactIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEventsByArtifactIDsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetEventsByArtifactIDsRequest): GetEventsByArtifactIDsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetEventsByArtifactIDsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEventsByArtifactIDsRequest;
  static deserializeBinaryFromReader(message: GetEventsByArtifactIDsRequest, reader: jspb.BinaryReader): GetEventsByArtifactIDsRequest;
}

export namespace GetEventsByArtifactIDsRequest {
  export type AsObject = {
    artifactIdsList: Array<number>,
  }
}

export class GetEventsByArtifactIDsResponse extends jspb.Message {
  clearEventsList(): void;
  getEventsList(): Array<src_apis_metadata_metadata_store_pb.Event>;
  setEventsList(value: Array<src_apis_metadata_metadata_store_pb.Event>): void;
  addEvents(value?: src_apis_metadata_metadata_store_pb.Event, index?: number): src_apis_metadata_metadata_store_pb.Event;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEventsByArtifactIDsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetEventsByArtifactIDsResponse): GetEventsByArtifactIDsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetEventsByArtifactIDsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEventsByArtifactIDsResponse;
  static deserializeBinaryFromReader(message: GetEventsByArtifactIDsResponse, reader: jspb.BinaryReader): GetEventsByArtifactIDsResponse;
}

export namespace GetEventsByArtifactIDsResponse {
  export type AsObject = {
    eventsList: Array<src_apis_metadata_metadata_store_pb.Event.AsObject>,
  }
}

export class GetArtifactTypesByIDRequest extends jspb.Message {
  clearTypeIdsList(): void;
  getTypeIdsList(): Array<number>;
  setTypeIdsList(value: Array<number>): void;
  addTypeIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesByIDRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactTypesByIDRequest): GetArtifactTypesByIDRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactTypesByIDRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesByIDRequest;
  static deserializeBinaryFromReader(message: GetArtifactTypesByIDRequest, reader: jspb.BinaryReader): GetArtifactTypesByIDRequest;
}

export namespace GetArtifactTypesByIDRequest {
  export type AsObject = {
    typeIdsList: Array<number>,
  }
}

export class GetArtifactTypesByIDResponse extends jspb.Message {
  clearArtifactTypesList(): void;
  getArtifactTypesList(): Array<src_apis_metadata_metadata_store_pb.ArtifactType>;
  setArtifactTypesList(value: Array<src_apis_metadata_metadata_store_pb.ArtifactType>): void;
  addArtifactTypes(value?: src_apis_metadata_metadata_store_pb.ArtifactType, index?: number): src_apis_metadata_metadata_store_pb.ArtifactType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesByIDResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactTypesByIDResponse): GetArtifactTypesByIDResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactTypesByIDResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesByIDResponse;
  static deserializeBinaryFromReader(message: GetArtifactTypesByIDResponse, reader: jspb.BinaryReader): GetArtifactTypesByIDResponse;
}

export namespace GetArtifactTypesByIDResponse {
  export type AsObject = {
    artifactTypesList: Array<src_apis_metadata_metadata_store_pb.ArtifactType.AsObject>,
  }
}

export class GetExecutionTypesByIDRequest extends jspb.Message {
  clearTypeIdsList(): void;
  getTypeIdsList(): Array<number>;
  setTypeIdsList(value: Array<number>): void;
  addTypeIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesByIDRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionTypesByIDRequest): GetExecutionTypesByIDRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionTypesByIDRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesByIDRequest;
  static deserializeBinaryFromReader(message: GetExecutionTypesByIDRequest, reader: jspb.BinaryReader): GetExecutionTypesByIDRequest;
}

export namespace GetExecutionTypesByIDRequest {
  export type AsObject = {
    typeIdsList: Array<number>,
  }
}

export class GetExecutionTypesByIDResponse extends jspb.Message {
  clearExecutionTypesList(): void;
  getExecutionTypesList(): Array<src_apis_metadata_metadata_store_pb.ExecutionType>;
  setExecutionTypesList(value: Array<src_apis_metadata_metadata_store_pb.ExecutionType>): void;
  addExecutionTypes(value?: src_apis_metadata_metadata_store_pb.ExecutionType, index?: number): src_apis_metadata_metadata_store_pb.ExecutionType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesByIDResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionTypesByIDResponse): GetExecutionTypesByIDResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionTypesByIDResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesByIDResponse;
  static deserializeBinaryFromReader(message: GetExecutionTypesByIDResponse, reader: jspb.BinaryReader): GetExecutionTypesByIDResponse;
}

export namespace GetExecutionTypesByIDResponse {
  export type AsObject = {
    executionTypesList: Array<src_apis_metadata_metadata_store_pb.ExecutionType.AsObject>,
  }
}

export class GetContextTypeRequest extends jspb.Message {
  hasTypeName(): boolean;
  clearTypeName(): void;
  getTypeName(): string | undefined;
  setTypeName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextTypeRequest): GetContextTypeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypeRequest;
  static deserializeBinaryFromReader(message: GetContextTypeRequest, reader: jspb.BinaryReader): GetContextTypeRequest;
}

export namespace GetContextTypeRequest {
  export type AsObject = {
    typeName?: string,
  }
}

export class GetContextTypeResponse extends jspb.Message {
  hasContextType(): boolean;
  clearContextType(): void;
  getContextType(): src_apis_metadata_metadata_store_pb.ContextType | undefined;
  setContextType(value?: src_apis_metadata_metadata_store_pb.ContextType): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextTypeResponse): GetContextTypeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypeResponse;
  static deserializeBinaryFromReader(message: GetContextTypeResponse, reader: jspb.BinaryReader): GetContextTypeResponse;
}

export namespace GetContextTypeResponse {
  export type AsObject = {
    contextType?: src_apis_metadata_metadata_store_pb.ContextType.AsObject,
  }
}

export class GetContextTypesByIDRequest extends jspb.Message {
  clearTypeIdsList(): void;
  getTypeIdsList(): Array<number>;
  setTypeIdsList(value: Array<number>): void;
  addTypeIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypesByIDRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextTypesByIDRequest): GetContextTypesByIDRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextTypesByIDRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypesByIDRequest;
  static deserializeBinaryFromReader(message: GetContextTypesByIDRequest, reader: jspb.BinaryReader): GetContextTypesByIDRequest;
}

export namespace GetContextTypesByIDRequest {
  export type AsObject = {
    typeIdsList: Array<number>,
  }
}

export class GetContextTypesByIDResponse extends jspb.Message {
  clearContextTypesList(): void;
  getContextTypesList(): Array<src_apis_metadata_metadata_store_pb.ContextType>;
  setContextTypesList(value: Array<src_apis_metadata_metadata_store_pb.ContextType>): void;
  addContextTypes(value?: src_apis_metadata_metadata_store_pb.ContextType, index?: number): src_apis_metadata_metadata_store_pb.ContextType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypesByIDResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextTypesByIDResponse): GetContextTypesByIDResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextTypesByIDResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypesByIDResponse;
  static deserializeBinaryFromReader(message: GetContextTypesByIDResponse, reader: jspb.BinaryReader): GetContextTypesByIDResponse;
}

export namespace GetContextTypesByIDResponse {
  export type AsObject = {
    contextTypesList: Array<src_apis_metadata_metadata_store_pb.ContextType.AsObject>,
  }
}

export class GetContextsRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsRequest): GetContextsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsRequest;
  static deserializeBinaryFromReader(message: GetContextsRequest, reader: jspb.BinaryReader): GetContextsRequest;
}

export namespace GetContextsRequest {
  export type AsObject = {
  }
}

export class GetContextsResponse extends jspb.Message {
  clearContextsList(): void;
  getContextsList(): Array<src_apis_metadata_metadata_store_pb.Context>;
  setContextsList(value: Array<src_apis_metadata_metadata_store_pb.Context>): void;
  addContexts(value?: src_apis_metadata_metadata_store_pb.Context, index?: number): src_apis_metadata_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsResponse): GetContextsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsResponse;
  static deserializeBinaryFromReader(message: GetContextsResponse, reader: jspb.BinaryReader): GetContextsResponse;
}

export namespace GetContextsResponse {
  export type AsObject = {
    contextsList: Array<src_apis_metadata_metadata_store_pb.Context.AsObject>,
  }
}

export class GetContextsByTypeRequest extends jspb.Message {
  hasTypeName(): boolean;
  clearTypeName(): void;
  getTypeName(): string | undefined;
  setTypeName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByTypeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsByTypeRequest): GetContextsByTypeRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsByTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByTypeRequest;
  static deserializeBinaryFromReader(message: GetContextsByTypeRequest, reader: jspb.BinaryReader): GetContextsByTypeRequest;
}

export namespace GetContextsByTypeRequest {
  export type AsObject = {
    typeName?: string,
  }
}

export class GetContextsByTypeResponse extends jspb.Message {
  clearContextsList(): void;
  getContextsList(): Array<src_apis_metadata_metadata_store_pb.Context>;
  setContextsList(value: Array<src_apis_metadata_metadata_store_pb.Context>): void;
  addContexts(value?: src_apis_metadata_metadata_store_pb.Context, index?: number): src_apis_metadata_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByTypeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsByTypeResponse): GetContextsByTypeResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsByTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByTypeResponse;
  static deserializeBinaryFromReader(message: GetContextsByTypeResponse, reader: jspb.BinaryReader): GetContextsByTypeResponse;
}

export namespace GetContextsByTypeResponse {
  export type AsObject = {
    contextsList: Array<src_apis_metadata_metadata_store_pb.Context.AsObject>,
  }
}

export class GetContextsByIDRequest extends jspb.Message {
  clearContextIdsList(): void;
  getContextIdsList(): Array<number>;
  setContextIdsList(value: Array<number>): void;
  addContextIds(value: number, index?: number): number;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByIDRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsByIDRequest): GetContextsByIDRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsByIDRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByIDRequest;
  static deserializeBinaryFromReader(message: GetContextsByIDRequest, reader: jspb.BinaryReader): GetContextsByIDRequest;
}

export namespace GetContextsByIDRequest {
  export type AsObject = {
    contextIdsList: Array<number>,
  }
}

export class GetContextsByIDResponse extends jspb.Message {
  clearContextsList(): void;
  getContextsList(): Array<src_apis_metadata_metadata_store_pb.Context>;
  setContextsList(value: Array<src_apis_metadata_metadata_store_pb.Context>): void;
  addContexts(value?: src_apis_metadata_metadata_store_pb.Context, index?: number): src_apis_metadata_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByIDResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsByIDResponse): GetContextsByIDResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsByIDResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByIDResponse;
  static deserializeBinaryFromReader(message: GetContextsByIDResponse, reader: jspb.BinaryReader): GetContextsByIDResponse;
}

export namespace GetContextsByIDResponse {
  export type AsObject = {
    contextsList: Array<src_apis_metadata_metadata_store_pb.Context.AsObject>,
  }
}

export class GetContextsByArtifactRequest extends jspb.Message {
  hasArtifactId(): boolean;
  clearArtifactId(): void;
  getArtifactId(): number | undefined;
  setArtifactId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByArtifactRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsByArtifactRequest): GetContextsByArtifactRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsByArtifactRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByArtifactRequest;
  static deserializeBinaryFromReader(message: GetContextsByArtifactRequest, reader: jspb.BinaryReader): GetContextsByArtifactRequest;
}

export namespace GetContextsByArtifactRequest {
  export type AsObject = {
    artifactId?: number,
  }
}

export class GetContextsByArtifactResponse extends jspb.Message {
  clearContextsList(): void;
  getContextsList(): Array<src_apis_metadata_metadata_store_pb.Context>;
  setContextsList(value: Array<src_apis_metadata_metadata_store_pb.Context>): void;
  addContexts(value?: src_apis_metadata_metadata_store_pb.Context, index?: number): src_apis_metadata_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByArtifactResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsByArtifactResponse): GetContextsByArtifactResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsByArtifactResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByArtifactResponse;
  static deserializeBinaryFromReader(message: GetContextsByArtifactResponse, reader: jspb.BinaryReader): GetContextsByArtifactResponse;
}

export namespace GetContextsByArtifactResponse {
  export type AsObject = {
    contextsList: Array<src_apis_metadata_metadata_store_pb.Context.AsObject>,
  }
}

export class GetContextsByExecutionRequest extends jspb.Message {
  hasExecutionId(): boolean;
  clearExecutionId(): void;
  getExecutionId(): number | undefined;
  setExecutionId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByExecutionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsByExecutionRequest): GetContextsByExecutionRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsByExecutionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByExecutionRequest;
  static deserializeBinaryFromReader(message: GetContextsByExecutionRequest, reader: jspb.BinaryReader): GetContextsByExecutionRequest;
}

export namespace GetContextsByExecutionRequest {
  export type AsObject = {
    executionId?: number,
  }
}

export class GetContextsByExecutionResponse extends jspb.Message {
  clearContextsList(): void;
  getContextsList(): Array<src_apis_metadata_metadata_store_pb.Context>;
  setContextsList(value: Array<src_apis_metadata_metadata_store_pb.Context>): void;
  addContexts(value?: src_apis_metadata_metadata_store_pb.Context, index?: number): src_apis_metadata_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByExecutionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsByExecutionResponse): GetContextsByExecutionResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetContextsByExecutionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByExecutionResponse;
  static deserializeBinaryFromReader(message: GetContextsByExecutionResponse, reader: jspb.BinaryReader): GetContextsByExecutionResponse;
}

export namespace GetContextsByExecutionResponse {
  export type AsObject = {
    contextsList: Array<src_apis_metadata_metadata_store_pb.Context.AsObject>,
  }
}

export class GetParentContextsByContextRequest extends jspb.Message {
  hasContextId(): boolean;
  clearContextId(): void;
  getContextId(): number | undefined;
  setContextId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetParentContextsByContextRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetParentContextsByContextRequest): GetParentContextsByContextRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetParentContextsByContextRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetParentContextsByContextRequest;
  static deserializeBinaryFromReader(message: GetParentContextsByContextRequest, reader: jspb.BinaryReader): GetParentContextsByContextRequest;
}

export namespace GetParentContextsByContextRequest {
  export type AsObject = {
    contextId?: number,
  }
}

export class GetParentContextsByContextResponse extends jspb.Message {
  clearContextsList(): void;
  getContextsList(): Array<src_apis_metadata_metadata_store_pb.Context>;
  setContextsList(value: Array<src_apis_metadata_metadata_store_pb.Context>): void;
  addContexts(value?: src_apis_metadata_metadata_store_pb.Context, index?: number): src_apis_metadata_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetParentContextsByContextResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetParentContextsByContextResponse): GetParentContextsByContextResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetParentContextsByContextResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetParentContextsByContextResponse;
  static deserializeBinaryFromReader(message: GetParentContextsByContextResponse, reader: jspb.BinaryReader): GetParentContextsByContextResponse;
}

export namespace GetParentContextsByContextResponse {
  export type AsObject = {
    contextsList: Array<src_apis_metadata_metadata_store_pb.Context.AsObject>,
  }
}

export class GetChildrenContextsByContextRequest extends jspb.Message {
  hasContextId(): boolean;
  clearContextId(): void;
  getContextId(): number | undefined;
  setContextId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetChildrenContextsByContextRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetChildrenContextsByContextRequest): GetChildrenContextsByContextRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetChildrenContextsByContextRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetChildrenContextsByContextRequest;
  static deserializeBinaryFromReader(message: GetChildrenContextsByContextRequest, reader: jspb.BinaryReader): GetChildrenContextsByContextRequest;
}

export namespace GetChildrenContextsByContextRequest {
  export type AsObject = {
    contextId?: number,
  }
}

export class GetChildrenContextsByContextResponse extends jspb.Message {
  clearContextsList(): void;
  getContextsList(): Array<src_apis_metadata_metadata_store_pb.Context>;
  setContextsList(value: Array<src_apis_metadata_metadata_store_pb.Context>): void;
  addContexts(value?: src_apis_metadata_metadata_store_pb.Context, index?: number): src_apis_metadata_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetChildrenContextsByContextResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetChildrenContextsByContextResponse): GetChildrenContextsByContextResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetChildrenContextsByContextResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetChildrenContextsByContextResponse;
  static deserializeBinaryFromReader(message: GetChildrenContextsByContextResponse, reader: jspb.BinaryReader): GetChildrenContextsByContextResponse;
}

export namespace GetChildrenContextsByContextResponse {
  export type AsObject = {
    contextsList: Array<src_apis_metadata_metadata_store_pb.Context.AsObject>,
  }
}

export class GetArtifactsByContextRequest extends jspb.Message {
  hasContextId(): boolean;
  clearContextId(): void;
  getContextId(): number | undefined;
  setContextId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByContextRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsByContextRequest): GetArtifactsByContextRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsByContextRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByContextRequest;
  static deserializeBinaryFromReader(message: GetArtifactsByContextRequest, reader: jspb.BinaryReader): GetArtifactsByContextRequest;
}

export namespace GetArtifactsByContextRequest {
  export type AsObject = {
    contextId?: number,
  }
}

export class GetArtifactsByContextResponse extends jspb.Message {
  clearArtifactsList(): void;
  getArtifactsList(): Array<src_apis_metadata_metadata_store_pb.Artifact>;
  setArtifactsList(value: Array<src_apis_metadata_metadata_store_pb.Artifact>): void;
  addArtifacts(value?: src_apis_metadata_metadata_store_pb.Artifact, index?: number): src_apis_metadata_metadata_store_pb.Artifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByContextResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsByContextResponse): GetArtifactsByContextResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetArtifactsByContextResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByContextResponse;
  static deserializeBinaryFromReader(message: GetArtifactsByContextResponse, reader: jspb.BinaryReader): GetArtifactsByContextResponse;
}

export namespace GetArtifactsByContextResponse {
  export type AsObject = {
    artifactsList: Array<src_apis_metadata_metadata_store_pb.Artifact.AsObject>,
  }
}

export class GetExecutionsByContextRequest extends jspb.Message {
  hasContextId(): boolean;
  clearContextId(): void;
  getContextId(): number | undefined;
  setContextId(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByContextRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionsByContextRequest): GetExecutionsByContextRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionsByContextRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByContextRequest;
  static deserializeBinaryFromReader(message: GetExecutionsByContextRequest, reader: jspb.BinaryReader): GetExecutionsByContextRequest;
}

export namespace GetExecutionsByContextRequest {
  export type AsObject = {
    contextId?: number,
  }
}

export class GetExecutionsByContextResponse extends jspb.Message {
  clearExecutionsList(): void;
  getExecutionsList(): Array<src_apis_metadata_metadata_store_pb.Execution>;
  setExecutionsList(value: Array<src_apis_metadata_metadata_store_pb.Execution>): void;
  addExecutions(value?: src_apis_metadata_metadata_store_pb.Execution, index?: number): src_apis_metadata_metadata_store_pb.Execution;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByContextResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetExecutionsByContextResponse): GetExecutionsByContextResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetExecutionsByContextResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByContextResponse;
  static deserializeBinaryFromReader(message: GetExecutionsByContextResponse, reader: jspb.BinaryReader): GetExecutionsByContextResponse;
}

export namespace GetExecutionsByContextResponse {
  export type AsObject = {
    executionsList: Array<src_apis_metadata_metadata_store_pb.Execution.AsObject>,
  }
}

