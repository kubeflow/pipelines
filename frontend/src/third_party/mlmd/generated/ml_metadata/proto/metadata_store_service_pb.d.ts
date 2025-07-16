import * as jspb from 'google-protobuf';

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb';
import * as ml_metadata_proto_metadata_store_pb from '../../ml_metadata/proto/metadata_store_pb';

export class ArtifactAndType extends jspb.Message {
  getArtifact(): ml_metadata_proto_metadata_store_pb.Artifact | undefined;
  setArtifact(value?: ml_metadata_proto_metadata_store_pb.Artifact): ArtifactAndType;
  hasArtifact(): boolean;
  clearArtifact(): ArtifactAndType;

  getType(): ml_metadata_proto_metadata_store_pb.ArtifactType | undefined;
  setType(value?: ml_metadata_proto_metadata_store_pb.ArtifactType): ArtifactAndType;
  hasType(): boolean;
  clearType(): ArtifactAndType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactAndType.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactAndType): ArtifactAndType.AsObject;
  static serializeBinaryToWriter(message: ArtifactAndType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactAndType;
  static deserializeBinaryFromReader(
    message: ArtifactAndType,
    reader: jspb.BinaryReader,
  ): ArtifactAndType;
}

export namespace ArtifactAndType {
  export type AsObject = {
    artifact?: ml_metadata_proto_metadata_store_pb.Artifact.AsObject;
    type?: ml_metadata_proto_metadata_store_pb.ArtifactType.AsObject;
  };
}

export class ArtifactStructMap extends jspb.Message {
  getPropertiesMap(): jspb.Map<string, ArtifactStruct>;
  clearPropertiesMap(): ArtifactStructMap;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactStructMap.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactStructMap): ArtifactStructMap.AsObject;
  static serializeBinaryToWriter(message: ArtifactStructMap, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactStructMap;
  static deserializeBinaryFromReader(
    message: ArtifactStructMap,
    reader: jspb.BinaryReader,
  ): ArtifactStructMap;
}

export namespace ArtifactStructMap {
  export type AsObject = {
    propertiesMap: Array<[string, ArtifactStruct.AsObject]>;
  };
}

export class ArtifactStructList extends jspb.Message {
  getElementsList(): Array<ArtifactStruct>;
  setElementsList(value: Array<ArtifactStruct>): ArtifactStructList;
  clearElementsList(): ArtifactStructList;
  addElements(value?: ArtifactStruct, index?: number): ArtifactStruct;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactStructList.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactStructList): ArtifactStructList.AsObject;
  static serializeBinaryToWriter(message: ArtifactStructList, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactStructList;
  static deserializeBinaryFromReader(
    message: ArtifactStructList,
    reader: jspb.BinaryReader,
  ): ArtifactStructList;
}

export namespace ArtifactStructList {
  export type AsObject = {
    elementsList: Array<ArtifactStruct.AsObject>;
  };
}

export class ArtifactStruct extends jspb.Message {
  getArtifact(): ArtifactAndType | undefined;
  setArtifact(value?: ArtifactAndType): ArtifactStruct;
  hasArtifact(): boolean;
  clearArtifact(): ArtifactStruct;

  getMap(): ArtifactStructMap | undefined;
  setMap(value?: ArtifactStructMap): ArtifactStruct;
  hasMap(): boolean;
  clearMap(): ArtifactStruct;

  getList(): ArtifactStructList | undefined;
  setList(value?: ArtifactStructList): ArtifactStruct;
  hasList(): boolean;
  clearList(): ArtifactStruct;

  getValueCase(): ArtifactStruct.ValueCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ArtifactStruct.AsObject;
  static toObject(includeInstance: boolean, msg: ArtifactStruct): ArtifactStruct.AsObject;
  static serializeBinaryToWriter(message: ArtifactStruct, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ArtifactStruct;
  static deserializeBinaryFromReader(
    message: ArtifactStruct,
    reader: jspb.BinaryReader,
  ): ArtifactStruct;
}

export namespace ArtifactStruct {
  export type AsObject = {
    artifact?: ArtifactAndType.AsObject;
    map?: ArtifactStructMap.AsObject;
    list?: ArtifactStructList.AsObject;
  };

  export enum ValueCase {
    VALUE_NOT_SET = 0,
    ARTIFACT = 1,
    MAP = 2,
    LIST = 3,
  }
}

export class PutArtifactsRequest extends jspb.Message {
  getArtifactsList(): Array<ml_metadata_proto_metadata_store_pb.Artifact>;
  setArtifactsList(value: Array<ml_metadata_proto_metadata_store_pb.Artifact>): PutArtifactsRequest;
  clearArtifactsList(): PutArtifactsRequest;
  addArtifacts(
    value?: ml_metadata_proto_metadata_store_pb.Artifact,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Artifact;

  getOptions(): PutArtifactsRequest.Options | undefined;
  setOptions(value?: PutArtifactsRequest.Options): PutArtifactsRequest;
  hasOptions(): boolean;
  clearOptions(): PutArtifactsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutArtifactsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutArtifactsRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): PutArtifactsRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): PutArtifactsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutArtifactsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutArtifactsRequest): PutArtifactsRequest.AsObject;
  static serializeBinaryToWriter(message: PutArtifactsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutArtifactsRequest;
  static deserializeBinaryFromReader(
    message: PutArtifactsRequest,
    reader: jspb.BinaryReader,
  ): PutArtifactsRequest;
}

export namespace PutArtifactsRequest {
  export type AsObject = {
    artifactsList: Array<ml_metadata_proto_metadata_store_pb.Artifact.AsObject>;
    options?: PutArtifactsRequest.Options.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };

  export class Options extends jspb.Message {
    getAbortIfLatestUpdatedTimeChanged(): boolean;
    setAbortIfLatestUpdatedTimeChanged(value: boolean): Options;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Options.AsObject;
    static toObject(includeInstance: boolean, msg: Options): Options.AsObject;
    static serializeBinaryToWriter(message: Options, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Options;
    static deserializeBinaryFromReader(message: Options, reader: jspb.BinaryReader): Options;
  }

  export namespace Options {
    export type AsObject = {
      abortIfLatestUpdatedTimeChanged: boolean;
    };
  }
}

export class PutArtifactsResponse extends jspb.Message {
  getArtifactIdsList(): Array<number>;
  setArtifactIdsList(value: Array<number>): PutArtifactsResponse;
  clearArtifactIdsList(): PutArtifactsResponse;
  addArtifactIds(value: number, index?: number): PutArtifactsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutArtifactsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutArtifactsResponse,
  ): PutArtifactsResponse.AsObject;
  static serializeBinaryToWriter(message: PutArtifactsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutArtifactsResponse;
  static deserializeBinaryFromReader(
    message: PutArtifactsResponse,
    reader: jspb.BinaryReader,
  ): PutArtifactsResponse;
}

export namespace PutArtifactsResponse {
  export type AsObject = {
    artifactIdsList: Array<number>;
  };
}

export class PutArtifactTypeRequest extends jspb.Message {
  getArtifactType(): ml_metadata_proto_metadata_store_pb.ArtifactType | undefined;
  setArtifactType(value?: ml_metadata_proto_metadata_store_pb.ArtifactType): PutArtifactTypeRequest;
  hasArtifactType(): boolean;
  clearArtifactType(): PutArtifactTypeRequest;

  getCanAddFields(): boolean;
  setCanAddFields(value: boolean): PutArtifactTypeRequest;

  getCanOmitFields(): boolean;
  setCanOmitFields(value: boolean): PutArtifactTypeRequest;

  getCanDeleteFields(): boolean;
  setCanDeleteFields(value: boolean): PutArtifactTypeRequest;

  getAllFieldsMatch(): boolean;
  setAllFieldsMatch(value: boolean): PutArtifactTypeRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutArtifactTypeRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutArtifactTypeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutArtifactTypeRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutArtifactTypeRequest,
  ): PutArtifactTypeRequest.AsObject;
  static serializeBinaryToWriter(message: PutArtifactTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutArtifactTypeRequest;
  static deserializeBinaryFromReader(
    message: PutArtifactTypeRequest,
    reader: jspb.BinaryReader,
  ): PutArtifactTypeRequest;
}

export namespace PutArtifactTypeRequest {
  export type AsObject = {
    artifactType?: ml_metadata_proto_metadata_store_pb.ArtifactType.AsObject;
    canAddFields: boolean;
    canOmitFields: boolean;
    canDeleteFields: boolean;
    allFieldsMatch: boolean;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class PutArtifactTypeResponse extends jspb.Message {
  getTypeId(): number;
  setTypeId(value: number): PutArtifactTypeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutArtifactTypeResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutArtifactTypeResponse,
  ): PutArtifactTypeResponse.AsObject;
  static serializeBinaryToWriter(message: PutArtifactTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutArtifactTypeResponse;
  static deserializeBinaryFromReader(
    message: PutArtifactTypeResponse,
    reader: jspb.BinaryReader,
  ): PutArtifactTypeResponse;
}

export namespace PutArtifactTypeResponse {
  export type AsObject = {
    typeId: number;
  };
}

export class PutExecutionsRequest extends jspb.Message {
  getExecutionsList(): Array<ml_metadata_proto_metadata_store_pb.Execution>;
  setExecutionsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Execution>,
  ): PutExecutionsRequest;
  clearExecutionsList(): PutExecutionsRequest;
  addExecutions(
    value?: ml_metadata_proto_metadata_store_pb.Execution,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Execution;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutExecutionsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutExecutionsRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): PutExecutionsRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): PutExecutionsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutExecutionsRequest,
  ): PutExecutionsRequest.AsObject;
  static serializeBinaryToWriter(message: PutExecutionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionsRequest;
  static deserializeBinaryFromReader(
    message: PutExecutionsRequest,
    reader: jspb.BinaryReader,
  ): PutExecutionsRequest;
}

export namespace PutExecutionsRequest {
  export type AsObject = {
    executionsList: Array<ml_metadata_proto_metadata_store_pb.Execution.AsObject>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PutExecutionsResponse extends jspb.Message {
  getExecutionIdsList(): Array<number>;
  setExecutionIdsList(value: Array<number>): PutExecutionsResponse;
  clearExecutionIdsList(): PutExecutionsResponse;
  addExecutionIds(value: number, index?: number): PutExecutionsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutExecutionsResponse,
  ): PutExecutionsResponse.AsObject;
  static serializeBinaryToWriter(message: PutExecutionsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionsResponse;
  static deserializeBinaryFromReader(
    message: PutExecutionsResponse,
    reader: jspb.BinaryReader,
  ): PutExecutionsResponse;
}

export namespace PutExecutionsResponse {
  export type AsObject = {
    executionIdsList: Array<number>;
  };
}

export class PutExecutionTypeRequest extends jspb.Message {
  getExecutionType(): ml_metadata_proto_metadata_store_pb.ExecutionType | undefined;
  setExecutionType(
    value?: ml_metadata_proto_metadata_store_pb.ExecutionType,
  ): PutExecutionTypeRequest;
  hasExecutionType(): boolean;
  clearExecutionType(): PutExecutionTypeRequest;

  getCanAddFields(): boolean;
  setCanAddFields(value: boolean): PutExecutionTypeRequest;

  getCanOmitFields(): boolean;
  setCanOmitFields(value: boolean): PutExecutionTypeRequest;

  getCanDeleteFields(): boolean;
  setCanDeleteFields(value: boolean): PutExecutionTypeRequest;

  getAllFieldsMatch(): boolean;
  setAllFieldsMatch(value: boolean): PutExecutionTypeRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutExecutionTypeRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutExecutionTypeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionTypeRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutExecutionTypeRequest,
  ): PutExecutionTypeRequest.AsObject;
  static serializeBinaryToWriter(message: PutExecutionTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionTypeRequest;
  static deserializeBinaryFromReader(
    message: PutExecutionTypeRequest,
    reader: jspb.BinaryReader,
  ): PutExecutionTypeRequest;
}

export namespace PutExecutionTypeRequest {
  export type AsObject = {
    executionType?: ml_metadata_proto_metadata_store_pb.ExecutionType.AsObject;
    canAddFields: boolean;
    canOmitFields: boolean;
    canDeleteFields: boolean;
    allFieldsMatch: boolean;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class PutExecutionTypeResponse extends jspb.Message {
  getTypeId(): number;
  setTypeId(value: number): PutExecutionTypeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionTypeResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutExecutionTypeResponse,
  ): PutExecutionTypeResponse.AsObject;
  static serializeBinaryToWriter(
    message: PutExecutionTypeResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionTypeResponse;
  static deserializeBinaryFromReader(
    message: PutExecutionTypeResponse,
    reader: jspb.BinaryReader,
  ): PutExecutionTypeResponse;
}

export namespace PutExecutionTypeResponse {
  export type AsObject = {
    typeId: number;
  };
}

export class PutEventsRequest extends jspb.Message {
  getEventsList(): Array<ml_metadata_proto_metadata_store_pb.Event>;
  setEventsList(value: Array<ml_metadata_proto_metadata_store_pb.Event>): PutEventsRequest;
  clearEventsList(): PutEventsRequest;
  addEvents(
    value?: ml_metadata_proto_metadata_store_pb.Event,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Event;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutEventsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutEventsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutEventsRequest): PutEventsRequest.AsObject;
  static serializeBinaryToWriter(message: PutEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutEventsRequest;
  static deserializeBinaryFromReader(
    message: PutEventsRequest,
    reader: jspb.BinaryReader,
  ): PutEventsRequest;
}

export namespace PutEventsRequest {
  export type AsObject = {
    eventsList: Array<ml_metadata_proto_metadata_store_pb.Event.AsObject>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class PutEventsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutEventsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutEventsResponse): PutEventsResponse.AsObject;
  static serializeBinaryToWriter(message: PutEventsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutEventsResponse;
  static deserializeBinaryFromReader(
    message: PutEventsResponse,
    reader: jspb.BinaryReader,
  ): PutEventsResponse;
}

export namespace PutEventsResponse {
  export type AsObject = {};
}

export class PutExecutionRequest extends jspb.Message {
  getExecution(): ml_metadata_proto_metadata_store_pb.Execution | undefined;
  setExecution(value?: ml_metadata_proto_metadata_store_pb.Execution): PutExecutionRequest;
  hasExecution(): boolean;
  clearExecution(): PutExecutionRequest;

  getArtifactEventPairsList(): Array<PutExecutionRequest.ArtifactAndEvent>;
  setArtifactEventPairsList(
    value: Array<PutExecutionRequest.ArtifactAndEvent>,
  ): PutExecutionRequest;
  clearArtifactEventPairsList(): PutExecutionRequest;
  addArtifactEventPairs(
    value?: PutExecutionRequest.ArtifactAndEvent,
    index?: number,
  ): PutExecutionRequest.ArtifactAndEvent;

  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(value: Array<ml_metadata_proto_metadata_store_pb.Context>): PutExecutionRequest;
  clearContextsList(): PutExecutionRequest;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  getOptions(): PutExecutionRequest.Options | undefined;
  setOptions(value?: PutExecutionRequest.Options): PutExecutionRequest;
  hasOptions(): boolean;
  clearOptions(): PutExecutionRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutExecutionRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutExecutionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutExecutionRequest): PutExecutionRequest.AsObject;
  static serializeBinaryToWriter(message: PutExecutionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionRequest;
  static deserializeBinaryFromReader(
    message: PutExecutionRequest,
    reader: jspb.BinaryReader,
  ): PutExecutionRequest;
}

export namespace PutExecutionRequest {
  export type AsObject = {
    execution?: ml_metadata_proto_metadata_store_pb.Execution.AsObject;
    artifactEventPairsList: Array<PutExecutionRequest.ArtifactAndEvent.AsObject>;
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
    options?: PutExecutionRequest.Options.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };

  export class ArtifactAndEvent extends jspb.Message {
    getArtifact(): ml_metadata_proto_metadata_store_pb.Artifact | undefined;
    setArtifact(value?: ml_metadata_proto_metadata_store_pb.Artifact): ArtifactAndEvent;
    hasArtifact(): boolean;
    clearArtifact(): ArtifactAndEvent;

    getEvent(): ml_metadata_proto_metadata_store_pb.Event | undefined;
    setEvent(value?: ml_metadata_proto_metadata_store_pb.Event): ArtifactAndEvent;
    hasEvent(): boolean;
    clearEvent(): ArtifactAndEvent;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ArtifactAndEvent.AsObject;
    static toObject(includeInstance: boolean, msg: ArtifactAndEvent): ArtifactAndEvent.AsObject;
    static serializeBinaryToWriter(message: ArtifactAndEvent, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ArtifactAndEvent;
    static deserializeBinaryFromReader(
      message: ArtifactAndEvent,
      reader: jspb.BinaryReader,
    ): ArtifactAndEvent;
  }

  export namespace ArtifactAndEvent {
    export type AsObject = {
      artifact?: ml_metadata_proto_metadata_store_pb.Artifact.AsObject;
      event?: ml_metadata_proto_metadata_store_pb.Event.AsObject;
    };
  }

  export class Options extends jspb.Message {
    getReuseContextIfAlreadyExist(): boolean;
    setReuseContextIfAlreadyExist(value: boolean): Options;

    getReuseArtifactIfAlreadyExistByExternalId(): boolean;
    setReuseArtifactIfAlreadyExistByExternalId(value: boolean): Options;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Options.AsObject;
    static toObject(includeInstance: boolean, msg: Options): Options.AsObject;
    static serializeBinaryToWriter(message: Options, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Options;
    static deserializeBinaryFromReader(message: Options, reader: jspb.BinaryReader): Options;
  }

  export namespace Options {
    export type AsObject = {
      reuseContextIfAlreadyExist: boolean;
      reuseArtifactIfAlreadyExistByExternalId: boolean;
    };
  }
}

export class PutExecutionResponse extends jspb.Message {
  getExecutionId(): number;
  setExecutionId(value: number): PutExecutionResponse;

  getArtifactIdsList(): Array<number>;
  setArtifactIdsList(value: Array<number>): PutExecutionResponse;
  clearArtifactIdsList(): PutExecutionResponse;
  addArtifactIds(value: number, index?: number): PutExecutionResponse;

  getContextIdsList(): Array<number>;
  setContextIdsList(value: Array<number>): PutExecutionResponse;
  clearContextIdsList(): PutExecutionResponse;
  addContextIds(value: number, index?: number): PutExecutionResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutExecutionResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutExecutionResponse,
  ): PutExecutionResponse.AsObject;
  static serializeBinaryToWriter(message: PutExecutionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutExecutionResponse;
  static deserializeBinaryFromReader(
    message: PutExecutionResponse,
    reader: jspb.BinaryReader,
  ): PutExecutionResponse;
}

export namespace PutExecutionResponse {
  export type AsObject = {
    executionId: number;
    artifactIdsList: Array<number>;
    contextIdsList: Array<number>;
  };
}

export class PutLineageSubgraphRequest extends jspb.Message {
  getExecutionsList(): Array<ml_metadata_proto_metadata_store_pb.Execution>;
  setExecutionsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Execution>,
  ): PutLineageSubgraphRequest;
  clearExecutionsList(): PutLineageSubgraphRequest;
  addExecutions(
    value?: ml_metadata_proto_metadata_store_pb.Execution,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Execution;

  getArtifactsList(): Array<ml_metadata_proto_metadata_store_pb.Artifact>;
  setArtifactsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Artifact>,
  ): PutLineageSubgraphRequest;
  clearArtifactsList(): PutLineageSubgraphRequest;
  addArtifacts(
    value?: ml_metadata_proto_metadata_store_pb.Artifact,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Artifact;

  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Context>,
  ): PutLineageSubgraphRequest;
  clearContextsList(): PutLineageSubgraphRequest;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  getEventEdgesList(): Array<PutLineageSubgraphRequest.EventEdge>;
  setEventEdgesList(value: Array<PutLineageSubgraphRequest.EventEdge>): PutLineageSubgraphRequest;
  clearEventEdgesList(): PutLineageSubgraphRequest;
  addEventEdges(
    value?: PutLineageSubgraphRequest.EventEdge,
    index?: number,
  ): PutLineageSubgraphRequest.EventEdge;

  getOptions(): PutLineageSubgraphRequest.Options | undefined;
  setOptions(value?: PutLineageSubgraphRequest.Options): PutLineageSubgraphRequest;
  hasOptions(): boolean;
  clearOptions(): PutLineageSubgraphRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutLineageSubgraphRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutLineageSubgraphRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutLineageSubgraphRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutLineageSubgraphRequest,
  ): PutLineageSubgraphRequest.AsObject;
  static serializeBinaryToWriter(
    message: PutLineageSubgraphRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): PutLineageSubgraphRequest;
  static deserializeBinaryFromReader(
    message: PutLineageSubgraphRequest,
    reader: jspb.BinaryReader,
  ): PutLineageSubgraphRequest;
}

export namespace PutLineageSubgraphRequest {
  export type AsObject = {
    executionsList: Array<ml_metadata_proto_metadata_store_pb.Execution.AsObject>;
    artifactsList: Array<ml_metadata_proto_metadata_store_pb.Artifact.AsObject>;
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
    eventEdgesList: Array<PutLineageSubgraphRequest.EventEdge.AsObject>;
    options?: PutLineageSubgraphRequest.Options.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };

  export class EventEdge extends jspb.Message {
    getExecutionIndex(): number;
    setExecutionIndex(value: number): EventEdge;

    getArtifactIndex(): number;
    setArtifactIndex(value: number): EventEdge;

    getEvent(): ml_metadata_proto_metadata_store_pb.Event | undefined;
    setEvent(value?: ml_metadata_proto_metadata_store_pb.Event): EventEdge;
    hasEvent(): boolean;
    clearEvent(): EventEdge;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): EventEdge.AsObject;
    static toObject(includeInstance: boolean, msg: EventEdge): EventEdge.AsObject;
    static serializeBinaryToWriter(message: EventEdge, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): EventEdge;
    static deserializeBinaryFromReader(message: EventEdge, reader: jspb.BinaryReader): EventEdge;
  }

  export namespace EventEdge {
    export type AsObject = {
      executionIndex: number;
      artifactIndex: number;
      event?: ml_metadata_proto_metadata_store_pb.Event.AsObject;
    };
  }

  export class Options extends jspb.Message {
    getReuseContextIfAlreadyExist(): boolean;
    setReuseContextIfAlreadyExist(value: boolean): Options;

    getReuseArtifactIfAlreadyExistByExternalId(): boolean;
    setReuseArtifactIfAlreadyExistByExternalId(value: boolean): Options;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Options.AsObject;
    static toObject(includeInstance: boolean, msg: Options): Options.AsObject;
    static serializeBinaryToWriter(message: Options, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Options;
    static deserializeBinaryFromReader(message: Options, reader: jspb.BinaryReader): Options;
  }

  export namespace Options {
    export type AsObject = {
      reuseContextIfAlreadyExist: boolean;
      reuseArtifactIfAlreadyExistByExternalId: boolean;
    };
  }
}

export class PutLineageSubgraphResponse extends jspb.Message {
  getExecutionIdsList(): Array<number>;
  setExecutionIdsList(value: Array<number>): PutLineageSubgraphResponse;
  clearExecutionIdsList(): PutLineageSubgraphResponse;
  addExecutionIds(value: number, index?: number): PutLineageSubgraphResponse;

  getArtifactIdsList(): Array<number>;
  setArtifactIdsList(value: Array<number>): PutLineageSubgraphResponse;
  clearArtifactIdsList(): PutLineageSubgraphResponse;
  addArtifactIds(value: number, index?: number): PutLineageSubgraphResponse;

  getContextIdsList(): Array<number>;
  setContextIdsList(value: Array<number>): PutLineageSubgraphResponse;
  clearContextIdsList(): PutLineageSubgraphResponse;
  addContextIds(value: number, index?: number): PutLineageSubgraphResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutLineageSubgraphResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutLineageSubgraphResponse,
  ): PutLineageSubgraphResponse.AsObject;
  static serializeBinaryToWriter(
    message: PutLineageSubgraphResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): PutLineageSubgraphResponse;
  static deserializeBinaryFromReader(
    message: PutLineageSubgraphResponse,
    reader: jspb.BinaryReader,
  ): PutLineageSubgraphResponse;
}

export namespace PutLineageSubgraphResponse {
  export type AsObject = {
    executionIdsList: Array<number>;
    artifactIdsList: Array<number>;
    contextIdsList: Array<number>;
  };
}

export class PutTypesRequest extends jspb.Message {
  getArtifactTypesList(): Array<ml_metadata_proto_metadata_store_pb.ArtifactType>;
  setArtifactTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ArtifactType>,
  ): PutTypesRequest;
  clearArtifactTypesList(): PutTypesRequest;
  addArtifactTypes(
    value?: ml_metadata_proto_metadata_store_pb.ArtifactType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ArtifactType;

  getExecutionTypesList(): Array<ml_metadata_proto_metadata_store_pb.ExecutionType>;
  setExecutionTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ExecutionType>,
  ): PutTypesRequest;
  clearExecutionTypesList(): PutTypesRequest;
  addExecutionTypes(
    value?: ml_metadata_proto_metadata_store_pb.ExecutionType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ExecutionType;

  getContextTypesList(): Array<ml_metadata_proto_metadata_store_pb.ContextType>;
  setContextTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ContextType>,
  ): PutTypesRequest;
  clearContextTypesList(): PutTypesRequest;
  addContextTypes(
    value?: ml_metadata_proto_metadata_store_pb.ContextType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ContextType;

  getCanAddFields(): boolean;
  setCanAddFields(value: boolean): PutTypesRequest;

  getCanOmitFields(): boolean;
  setCanOmitFields(value: boolean): PutTypesRequest;

  getCanDeleteFields(): boolean;
  setCanDeleteFields(value: boolean): PutTypesRequest;

  getAllFieldsMatch(): boolean;
  setAllFieldsMatch(value: boolean): PutTypesRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutTypesRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutTypesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutTypesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutTypesRequest): PutTypesRequest.AsObject;
  static serializeBinaryToWriter(message: PutTypesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutTypesRequest;
  static deserializeBinaryFromReader(
    message: PutTypesRequest,
    reader: jspb.BinaryReader,
  ): PutTypesRequest;
}

export namespace PutTypesRequest {
  export type AsObject = {
    artifactTypesList: Array<ml_metadata_proto_metadata_store_pb.ArtifactType.AsObject>;
    executionTypesList: Array<ml_metadata_proto_metadata_store_pb.ExecutionType.AsObject>;
    contextTypesList: Array<ml_metadata_proto_metadata_store_pb.ContextType.AsObject>;
    canAddFields: boolean;
    canOmitFields: boolean;
    canDeleteFields: boolean;
    allFieldsMatch: boolean;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class PutTypesResponse extends jspb.Message {
  getArtifactTypeIdsList(): Array<number>;
  setArtifactTypeIdsList(value: Array<number>): PutTypesResponse;
  clearArtifactTypeIdsList(): PutTypesResponse;
  addArtifactTypeIds(value: number, index?: number): PutTypesResponse;

  getExecutionTypeIdsList(): Array<number>;
  setExecutionTypeIdsList(value: Array<number>): PutTypesResponse;
  clearExecutionTypeIdsList(): PutTypesResponse;
  addExecutionTypeIds(value: number, index?: number): PutTypesResponse;

  getContextTypeIdsList(): Array<number>;
  setContextTypeIdsList(value: Array<number>): PutTypesResponse;
  clearContextTypeIdsList(): PutTypesResponse;
  addContextTypeIds(value: number, index?: number): PutTypesResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutTypesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutTypesResponse): PutTypesResponse.AsObject;
  static serializeBinaryToWriter(message: PutTypesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutTypesResponse;
  static deserializeBinaryFromReader(
    message: PutTypesResponse,
    reader: jspb.BinaryReader,
  ): PutTypesResponse;
}

export namespace PutTypesResponse {
  export type AsObject = {
    artifactTypeIdsList: Array<number>;
    executionTypeIdsList: Array<number>;
    contextTypeIdsList: Array<number>;
  };
}

export class PutContextTypeRequest extends jspb.Message {
  getContextType(): ml_metadata_proto_metadata_store_pb.ContextType | undefined;
  setContextType(value?: ml_metadata_proto_metadata_store_pb.ContextType): PutContextTypeRequest;
  hasContextType(): boolean;
  clearContextType(): PutContextTypeRequest;

  getCanAddFields(): boolean;
  setCanAddFields(value: boolean): PutContextTypeRequest;

  getCanOmitFields(): boolean;
  setCanOmitFields(value: boolean): PutContextTypeRequest;

  getCanDeleteFields(): boolean;
  setCanDeleteFields(value: boolean): PutContextTypeRequest;

  getAllFieldsMatch(): boolean;
  setAllFieldsMatch(value: boolean): PutContextTypeRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutContextTypeRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutContextTypeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutContextTypeRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutContextTypeRequest,
  ): PutContextTypeRequest.AsObject;
  static serializeBinaryToWriter(message: PutContextTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutContextTypeRequest;
  static deserializeBinaryFromReader(
    message: PutContextTypeRequest,
    reader: jspb.BinaryReader,
  ): PutContextTypeRequest;
}

export namespace PutContextTypeRequest {
  export type AsObject = {
    contextType?: ml_metadata_proto_metadata_store_pb.ContextType.AsObject;
    canAddFields: boolean;
    canOmitFields: boolean;
    canDeleteFields: boolean;
    allFieldsMatch: boolean;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class PutContextTypeResponse extends jspb.Message {
  getTypeId(): number;
  setTypeId(value: number): PutContextTypeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutContextTypeResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutContextTypeResponse,
  ): PutContextTypeResponse.AsObject;
  static serializeBinaryToWriter(message: PutContextTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutContextTypeResponse;
  static deserializeBinaryFromReader(
    message: PutContextTypeResponse,
    reader: jspb.BinaryReader,
  ): PutContextTypeResponse;
}

export namespace PutContextTypeResponse {
  export type AsObject = {
    typeId: number;
  };
}

export class PutContextsRequest extends jspb.Message {
  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(value: Array<ml_metadata_proto_metadata_store_pb.Context>): PutContextsRequest;
  clearContextsList(): PutContextsRequest;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutContextsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutContextsRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): PutContextsRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): PutContextsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutContextsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PutContextsRequest): PutContextsRequest.AsObject;
  static serializeBinaryToWriter(message: PutContextsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutContextsRequest;
  static deserializeBinaryFromReader(
    message: PutContextsRequest,
    reader: jspb.BinaryReader,
  ): PutContextsRequest;
}

export namespace PutContextsRequest {
  export type AsObject = {
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PutContextsResponse extends jspb.Message {
  getContextIdsList(): Array<number>;
  setContextIdsList(value: Array<number>): PutContextsResponse;
  clearContextIdsList(): PutContextsResponse;
  addContextIds(value: number, index?: number): PutContextsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutContextsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PutContextsResponse): PutContextsResponse.AsObject;
  static serializeBinaryToWriter(message: PutContextsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PutContextsResponse;
  static deserializeBinaryFromReader(
    message: PutContextsResponse,
    reader: jspb.BinaryReader,
  ): PutContextsResponse;
}

export namespace PutContextsResponse {
  export type AsObject = {
    contextIdsList: Array<number>;
  };
}

export class PutAttributionsAndAssociationsRequest extends jspb.Message {
  getAttributionsList(): Array<ml_metadata_proto_metadata_store_pb.Attribution>;
  setAttributionsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Attribution>,
  ): PutAttributionsAndAssociationsRequest;
  clearAttributionsList(): PutAttributionsAndAssociationsRequest;
  addAttributions(
    value?: ml_metadata_proto_metadata_store_pb.Attribution,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Attribution;

  getAssociationsList(): Array<ml_metadata_proto_metadata_store_pb.Association>;
  setAssociationsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Association>,
  ): PutAttributionsAndAssociationsRequest;
  clearAssociationsList(): PutAttributionsAndAssociationsRequest;
  addAssociations(
    value?: ml_metadata_proto_metadata_store_pb.Association,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Association;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutAttributionsAndAssociationsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutAttributionsAndAssociationsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutAttributionsAndAssociationsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutAttributionsAndAssociationsRequest,
  ): PutAttributionsAndAssociationsRequest.AsObject;
  static serializeBinaryToWriter(
    message: PutAttributionsAndAssociationsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): PutAttributionsAndAssociationsRequest;
  static deserializeBinaryFromReader(
    message: PutAttributionsAndAssociationsRequest,
    reader: jspb.BinaryReader,
  ): PutAttributionsAndAssociationsRequest;
}

export namespace PutAttributionsAndAssociationsRequest {
  export type AsObject = {
    attributionsList: Array<ml_metadata_proto_metadata_store_pb.Attribution.AsObject>;
    associationsList: Array<ml_metadata_proto_metadata_store_pb.Association.AsObject>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class PutAttributionsAndAssociationsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutAttributionsAndAssociationsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutAttributionsAndAssociationsResponse,
  ): PutAttributionsAndAssociationsResponse.AsObject;
  static serializeBinaryToWriter(
    message: PutAttributionsAndAssociationsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): PutAttributionsAndAssociationsResponse;
  static deserializeBinaryFromReader(
    message: PutAttributionsAndAssociationsResponse,
    reader: jspb.BinaryReader,
  ): PutAttributionsAndAssociationsResponse;
}

export namespace PutAttributionsAndAssociationsResponse {
  export type AsObject = {};
}

export class PutParentContextsRequest extends jspb.Message {
  getParentContextsList(): Array<ml_metadata_proto_metadata_store_pb.ParentContext>;
  setParentContextsList(
    value: Array<ml_metadata_proto_metadata_store_pb.ParentContext>,
  ): PutParentContextsRequest;
  clearParentContextsList(): PutParentContextsRequest;
  addParentContexts(
    value?: ml_metadata_proto_metadata_store_pb.ParentContext,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ParentContext;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): PutParentContextsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): PutParentContextsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutParentContextsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutParentContextsRequest,
  ): PutParentContextsRequest.AsObject;
  static serializeBinaryToWriter(
    message: PutParentContextsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): PutParentContextsRequest;
  static deserializeBinaryFromReader(
    message: PutParentContextsRequest,
    reader: jspb.BinaryReader,
  ): PutParentContextsRequest;
}

export namespace PutParentContextsRequest {
  export type AsObject = {
    parentContextsList: Array<ml_metadata_proto_metadata_store_pb.ParentContext.AsObject>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class PutParentContextsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PutParentContextsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: PutParentContextsResponse,
  ): PutParentContextsResponse.AsObject;
  static serializeBinaryToWriter(
    message: PutParentContextsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): PutParentContextsResponse;
  static deserializeBinaryFromReader(
    message: PutParentContextsResponse,
    reader: jspb.BinaryReader,
  ): PutParentContextsResponse;
}

export namespace PutParentContextsResponse {
  export type AsObject = {};
}

export class GetArtifactsByTypeRequest extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): GetArtifactsByTypeRequest;

  getTypeVersion(): string;
  setTypeVersion(value: string): GetArtifactsByTypeRequest;

  getOptions(): ml_metadata_proto_metadata_store_pb.ListOperationOptions | undefined;
  setOptions(
    value?: ml_metadata_proto_metadata_store_pb.ListOperationOptions,
  ): GetArtifactsByTypeRequest;
  hasOptions(): boolean;
  clearOptions(): GetArtifactsByTypeRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactsByTypeRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactsByTypeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByTypeRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByTypeRequest,
  ): GetArtifactsByTypeRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactsByTypeRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByTypeRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactsByTypeRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactsByTypeRequest;
}

export namespace GetArtifactsByTypeRequest {
  export type AsObject = {
    typeName: string;
    typeVersion: string;
    options?: ml_metadata_proto_metadata_store_pb.ListOperationOptions.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactsByTypeResponse extends jspb.Message {
  getArtifactsList(): Array<ml_metadata_proto_metadata_store_pb.Artifact>;
  setArtifactsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Artifact>,
  ): GetArtifactsByTypeResponse;
  clearArtifactsList(): GetArtifactsByTypeResponse;
  addArtifacts(
    value?: ml_metadata_proto_metadata_store_pb.Artifact,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Artifact;

  getNextPageToken(): string;
  setNextPageToken(value: string): GetArtifactsByTypeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByTypeResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByTypeResponse,
  ): GetArtifactsByTypeResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactsByTypeResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByTypeResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactsByTypeResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactsByTypeResponse;
}

export namespace GetArtifactsByTypeResponse {
  export type AsObject = {
    artifactsList: Array<ml_metadata_proto_metadata_store_pb.Artifact.AsObject>;
    nextPageToken: string;
  };
}

export class GetArtifactByTypeAndNameRequest extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): GetArtifactByTypeAndNameRequest;

  getTypeVersion(): string;
  setTypeVersion(value: string): GetArtifactByTypeAndNameRequest;

  getArtifactName(): string;
  setArtifactName(value: string): GetArtifactByTypeAndNameRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactByTypeAndNameRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactByTypeAndNameRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactByTypeAndNameRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactByTypeAndNameRequest,
  ): GetArtifactByTypeAndNameRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactByTypeAndNameRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactByTypeAndNameRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactByTypeAndNameRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactByTypeAndNameRequest;
}

export namespace GetArtifactByTypeAndNameRequest {
  export type AsObject = {
    typeName: string;
    typeVersion: string;
    artifactName: string;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactByTypeAndNameResponse extends jspb.Message {
  getArtifact(): ml_metadata_proto_metadata_store_pb.Artifact | undefined;
  setArtifact(
    value?: ml_metadata_proto_metadata_store_pb.Artifact,
  ): GetArtifactByTypeAndNameResponse;
  hasArtifact(): boolean;
  clearArtifact(): GetArtifactByTypeAndNameResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactByTypeAndNameResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactByTypeAndNameResponse,
  ): GetArtifactByTypeAndNameResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactByTypeAndNameResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactByTypeAndNameResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactByTypeAndNameResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactByTypeAndNameResponse;
}

export namespace GetArtifactByTypeAndNameResponse {
  export type AsObject = {
    artifact?: ml_metadata_proto_metadata_store_pb.Artifact.AsObject;
  };
}

export class GetArtifactsByIDRequest extends jspb.Message {
  getArtifactIdsList(): Array<number>;
  setArtifactIdsList(value: Array<number>): GetArtifactsByIDRequest;
  clearArtifactIdsList(): GetArtifactsByIDRequest;
  addArtifactIds(value: number, index?: number): GetArtifactsByIDRequest;

  getPopulateArtifactTypes(): boolean;
  setPopulateArtifactTypes(value: boolean): GetArtifactsByIDRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactsByIDRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactsByIDRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByIDRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByIDRequest,
  ): GetArtifactsByIDRequest.AsObject;
  static serializeBinaryToWriter(message: GetArtifactsByIDRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByIDRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactsByIDRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactsByIDRequest;
}

export namespace GetArtifactsByIDRequest {
  export type AsObject = {
    artifactIdsList: Array<number>;
    populateArtifactTypes: boolean;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactsByIDResponse extends jspb.Message {
  getArtifactsList(): Array<ml_metadata_proto_metadata_store_pb.Artifact>;
  setArtifactsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Artifact>,
  ): GetArtifactsByIDResponse;
  clearArtifactsList(): GetArtifactsByIDResponse;
  addArtifacts(
    value?: ml_metadata_proto_metadata_store_pb.Artifact,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Artifact;

  getArtifactTypesList(): Array<ml_metadata_proto_metadata_store_pb.ArtifactType>;
  setArtifactTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ArtifactType>,
  ): GetArtifactsByIDResponse;
  clearArtifactTypesList(): GetArtifactsByIDResponse;
  addArtifactTypes(
    value?: ml_metadata_proto_metadata_store_pb.ArtifactType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ArtifactType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByIDResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByIDResponse,
  ): GetArtifactsByIDResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactsByIDResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByIDResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactsByIDResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactsByIDResponse;
}

export namespace GetArtifactsByIDResponse {
  export type AsObject = {
    artifactsList: Array<ml_metadata_proto_metadata_store_pb.Artifact.AsObject>;
    artifactTypesList: Array<ml_metadata_proto_metadata_store_pb.ArtifactType.AsObject>;
  };
}

export class GetArtifactsRequest extends jspb.Message {
  getOptions(): ml_metadata_proto_metadata_store_pb.ListOperationOptions | undefined;
  setOptions(value?: ml_metadata_proto_metadata_store_pb.ListOperationOptions): GetArtifactsRequest;
  hasOptions(): boolean;
  clearOptions(): GetArtifactsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetArtifactsRequest): GetArtifactsRequest.AsObject;
  static serializeBinaryToWriter(message: GetArtifactsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactsRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactsRequest;
}

export namespace GetArtifactsRequest {
  export type AsObject = {
    options?: ml_metadata_proto_metadata_store_pb.ListOperationOptions.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactsResponse extends jspb.Message {
  getArtifactsList(): Array<ml_metadata_proto_metadata_store_pb.Artifact>;
  setArtifactsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Artifact>,
  ): GetArtifactsResponse;
  clearArtifactsList(): GetArtifactsResponse;
  addArtifacts(
    value?: ml_metadata_proto_metadata_store_pb.Artifact,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Artifact;

  getNextPageToken(): string;
  setNextPageToken(value: string): GetArtifactsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsResponse,
  ): GetArtifactsResponse.AsObject;
  static serializeBinaryToWriter(message: GetArtifactsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactsResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactsResponse;
}

export namespace GetArtifactsResponse {
  export type AsObject = {
    artifactsList: Array<ml_metadata_proto_metadata_store_pb.Artifact.AsObject>;
    nextPageToken: string;
  };
}

export class GetArtifactsByURIRequest extends jspb.Message {
  getUrisList(): Array<string>;
  setUrisList(value: Array<string>): GetArtifactsByURIRequest;
  clearUrisList(): GetArtifactsByURIRequest;
  addUris(value: string, index?: number): GetArtifactsByURIRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactsByURIRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactsByURIRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByURIRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByURIRequest,
  ): GetArtifactsByURIRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactsByURIRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByURIRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactsByURIRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactsByURIRequest;
}

export namespace GetArtifactsByURIRequest {
  export type AsObject = {
    urisList: Array<string>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactsByURIResponse extends jspb.Message {
  getArtifactsList(): Array<ml_metadata_proto_metadata_store_pb.Artifact>;
  setArtifactsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Artifact>,
  ): GetArtifactsByURIResponse;
  clearArtifactsList(): GetArtifactsByURIResponse;
  addArtifacts(
    value?: ml_metadata_proto_metadata_store_pb.Artifact,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Artifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByURIResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByURIResponse,
  ): GetArtifactsByURIResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactsByURIResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByURIResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactsByURIResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactsByURIResponse;
}

export namespace GetArtifactsByURIResponse {
  export type AsObject = {
    artifactsList: Array<ml_metadata_proto_metadata_store_pb.Artifact.AsObject>;
  };
}

export class GetExecutionsRequest extends jspb.Message {
  getOptions(): ml_metadata_proto_metadata_store_pb.ListOperationOptions | undefined;
  setOptions(
    value?: ml_metadata_proto_metadata_store_pb.ListOperationOptions,
  ): GetExecutionsRequest;
  hasOptions(): boolean;
  clearOptions(): GetExecutionsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsRequest,
  ): GetExecutionsRequest.AsObject;
  static serializeBinaryToWriter(message: GetExecutionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionsRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionsRequest;
}

export namespace GetExecutionsRequest {
  export type AsObject = {
    options?: ml_metadata_proto_metadata_store_pb.ListOperationOptions.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionsResponse extends jspb.Message {
  getExecutionsList(): Array<ml_metadata_proto_metadata_store_pb.Execution>;
  setExecutionsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Execution>,
  ): GetExecutionsResponse;
  clearExecutionsList(): GetExecutionsResponse;
  addExecutions(
    value?: ml_metadata_proto_metadata_store_pb.Execution,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Execution;

  getNextPageToken(): string;
  setNextPageToken(value: string): GetExecutionsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsResponse,
  ): GetExecutionsResponse.AsObject;
  static serializeBinaryToWriter(message: GetExecutionsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionsResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionsResponse;
}

export namespace GetExecutionsResponse {
  export type AsObject = {
    executionsList: Array<ml_metadata_proto_metadata_store_pb.Execution.AsObject>;
    nextPageToken: string;
  };
}

export class GetArtifactTypeRequest extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): GetArtifactTypeRequest;

  getTypeVersion(): string;
  setTypeVersion(value: string): GetArtifactTypeRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactTypeRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactTypeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypeRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactTypeRequest,
  ): GetArtifactTypeRequest.AsObject;
  static serializeBinaryToWriter(message: GetArtifactTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypeRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactTypeRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactTypeRequest;
}

export namespace GetArtifactTypeRequest {
  export type AsObject = {
    typeName: string;
    typeVersion: string;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactTypeResponse extends jspb.Message {
  getArtifactType(): ml_metadata_proto_metadata_store_pb.ArtifactType | undefined;
  setArtifactType(
    value?: ml_metadata_proto_metadata_store_pb.ArtifactType,
  ): GetArtifactTypeResponse;
  hasArtifactType(): boolean;
  clearArtifactType(): GetArtifactTypeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypeResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactTypeResponse,
  ): GetArtifactTypeResponse.AsObject;
  static serializeBinaryToWriter(message: GetArtifactTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypeResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactTypeResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactTypeResponse;
}

export namespace GetArtifactTypeResponse {
  export type AsObject = {
    artifactType?: ml_metadata_proto_metadata_store_pb.ArtifactType.AsObject;
  };
}

export class GetArtifactTypesRequest extends jspb.Message {
  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactTypesRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactTypesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactTypesRequest,
  ): GetArtifactTypesRequest.AsObject;
  static serializeBinaryToWriter(message: GetArtifactTypesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactTypesRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactTypesRequest;
}

export namespace GetArtifactTypesRequest {
  export type AsObject = {
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactTypesResponse extends jspb.Message {
  getArtifactTypesList(): Array<ml_metadata_proto_metadata_store_pb.ArtifactType>;
  setArtifactTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ArtifactType>,
  ): GetArtifactTypesResponse;
  clearArtifactTypesList(): GetArtifactTypesResponse;
  addArtifactTypes(
    value?: ml_metadata_proto_metadata_store_pb.ArtifactType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ArtifactType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactTypesResponse,
  ): GetArtifactTypesResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactTypesResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactTypesResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactTypesResponse;
}

export namespace GetArtifactTypesResponse {
  export type AsObject = {
    artifactTypesList: Array<ml_metadata_proto_metadata_store_pb.ArtifactType.AsObject>;
  };
}

export class GetExecutionTypesRequest extends jspb.Message {
  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionTypesRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionTypesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionTypesRequest,
  ): GetExecutionTypesRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionTypesRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionTypesRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionTypesRequest;
}

export namespace GetExecutionTypesRequest {
  export type AsObject = {
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionTypesResponse extends jspb.Message {
  getExecutionTypesList(): Array<ml_metadata_proto_metadata_store_pb.ExecutionType>;
  setExecutionTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ExecutionType>,
  ): GetExecutionTypesResponse;
  clearExecutionTypesList(): GetExecutionTypesResponse;
  addExecutionTypes(
    value?: ml_metadata_proto_metadata_store_pb.ExecutionType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ExecutionType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionTypesResponse,
  ): GetExecutionTypesResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionTypesResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionTypesResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionTypesResponse;
}

export namespace GetExecutionTypesResponse {
  export type AsObject = {
    executionTypesList: Array<ml_metadata_proto_metadata_store_pb.ExecutionType.AsObject>;
  };
}

export class GetContextTypesRequest extends jspb.Message {
  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextTypesRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextTypesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypesRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextTypesRequest,
  ): GetContextTypesRequest.AsObject;
  static serializeBinaryToWriter(message: GetContextTypesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypesRequest;
  static deserializeBinaryFromReader(
    message: GetContextTypesRequest,
    reader: jspb.BinaryReader,
  ): GetContextTypesRequest;
}

export namespace GetContextTypesRequest {
  export type AsObject = {
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextTypesResponse extends jspb.Message {
  getContextTypesList(): Array<ml_metadata_proto_metadata_store_pb.ContextType>;
  setContextTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ContextType>,
  ): GetContextTypesResponse;
  clearContextTypesList(): GetContextTypesResponse;
  addContextTypes(
    value?: ml_metadata_proto_metadata_store_pb.ContextType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ContextType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypesResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextTypesResponse,
  ): GetContextTypesResponse.AsObject;
  static serializeBinaryToWriter(message: GetContextTypesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypesResponse;
  static deserializeBinaryFromReader(
    message: GetContextTypesResponse,
    reader: jspb.BinaryReader,
  ): GetContextTypesResponse;
}

export namespace GetContextTypesResponse {
  export type AsObject = {
    contextTypesList: Array<ml_metadata_proto_metadata_store_pb.ContextType.AsObject>;
  };
}

export class GetArtifactsByExternalIdsRequest extends jspb.Message {
  getExternalIdsList(): Array<string>;
  setExternalIdsList(value: Array<string>): GetArtifactsByExternalIdsRequest;
  clearExternalIdsList(): GetArtifactsByExternalIdsRequest;
  addExternalIds(value: string, index?: number): GetArtifactsByExternalIdsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactsByExternalIdsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactsByExternalIdsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByExternalIdsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByExternalIdsRequest,
  ): GetArtifactsByExternalIdsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactsByExternalIdsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByExternalIdsRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactsByExternalIdsRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactsByExternalIdsRequest;
}

export namespace GetArtifactsByExternalIdsRequest {
  export type AsObject = {
    externalIdsList: Array<string>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactsByExternalIdsResponse extends jspb.Message {
  getArtifactsList(): Array<ml_metadata_proto_metadata_store_pb.Artifact>;
  setArtifactsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Artifact>,
  ): GetArtifactsByExternalIdsResponse;
  clearArtifactsList(): GetArtifactsByExternalIdsResponse;
  addArtifacts(
    value?: ml_metadata_proto_metadata_store_pb.Artifact,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Artifact;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByExternalIdsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByExternalIdsResponse,
  ): GetArtifactsByExternalIdsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactsByExternalIdsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByExternalIdsResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactsByExternalIdsResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactsByExternalIdsResponse;
}

export namespace GetArtifactsByExternalIdsResponse {
  export type AsObject = {
    artifactsList: Array<ml_metadata_proto_metadata_store_pb.Artifact.AsObject>;
  };
}

export class GetExecutionsByExternalIdsRequest extends jspb.Message {
  getExternalIdsList(): Array<string>;
  setExternalIdsList(value: Array<string>): GetExecutionsByExternalIdsRequest;
  clearExternalIdsList(): GetExecutionsByExternalIdsRequest;
  addExternalIds(value: string, index?: number): GetExecutionsByExternalIdsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionsByExternalIdsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionsByExternalIdsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByExternalIdsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsByExternalIdsRequest,
  ): GetExecutionsByExternalIdsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionsByExternalIdsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByExternalIdsRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionsByExternalIdsRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionsByExternalIdsRequest;
}

export namespace GetExecutionsByExternalIdsRequest {
  export type AsObject = {
    externalIdsList: Array<string>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionsByExternalIdsResponse extends jspb.Message {
  getExecutionsList(): Array<ml_metadata_proto_metadata_store_pb.Execution>;
  setExecutionsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Execution>,
  ): GetExecutionsByExternalIdsResponse;
  clearExecutionsList(): GetExecutionsByExternalIdsResponse;
  addExecutions(
    value?: ml_metadata_proto_metadata_store_pb.Execution,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Execution;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByExternalIdsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsByExternalIdsResponse,
  ): GetExecutionsByExternalIdsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionsByExternalIdsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByExternalIdsResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionsByExternalIdsResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionsByExternalIdsResponse;
}

export namespace GetExecutionsByExternalIdsResponse {
  export type AsObject = {
    executionsList: Array<ml_metadata_proto_metadata_store_pb.Execution.AsObject>;
  };
}

export class GetContextsByExternalIdsRequest extends jspb.Message {
  getExternalIdsList(): Array<string>;
  setExternalIdsList(value: Array<string>): GetContextsByExternalIdsRequest;
  clearExternalIdsList(): GetContextsByExternalIdsRequest;
  addExternalIds(value: string, index?: number): GetContextsByExternalIdsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextsByExternalIdsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextsByExternalIdsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByExternalIdsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByExternalIdsRequest,
  ): GetContextsByExternalIdsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetContextsByExternalIdsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByExternalIdsRequest;
  static deserializeBinaryFromReader(
    message: GetContextsByExternalIdsRequest,
    reader: jspb.BinaryReader,
  ): GetContextsByExternalIdsRequest;
}

export namespace GetContextsByExternalIdsRequest {
  export type AsObject = {
    externalIdsList: Array<string>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextsByExternalIdsResponse extends jspb.Message {
  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Context>,
  ): GetContextsByExternalIdsResponse;
  clearContextsList(): GetContextsByExternalIdsResponse;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByExternalIdsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByExternalIdsResponse,
  ): GetContextsByExternalIdsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetContextsByExternalIdsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByExternalIdsResponse;
  static deserializeBinaryFromReader(
    message: GetContextsByExternalIdsResponse,
    reader: jspb.BinaryReader,
  ): GetContextsByExternalIdsResponse;
}

export namespace GetContextsByExternalIdsResponse {
  export type AsObject = {
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
  };
}

export class GetArtifactTypesByExternalIdsRequest extends jspb.Message {
  getExternalIdsList(): Array<string>;
  setExternalIdsList(value: Array<string>): GetArtifactTypesByExternalIdsRequest;
  clearExternalIdsList(): GetArtifactTypesByExternalIdsRequest;
  addExternalIds(value: string, index?: number): GetArtifactTypesByExternalIdsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactTypesByExternalIdsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactTypesByExternalIdsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesByExternalIdsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactTypesByExternalIdsRequest,
  ): GetArtifactTypesByExternalIdsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactTypesByExternalIdsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesByExternalIdsRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactTypesByExternalIdsRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactTypesByExternalIdsRequest;
}

export namespace GetArtifactTypesByExternalIdsRequest {
  export type AsObject = {
    externalIdsList: Array<string>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactTypesByExternalIdsResponse extends jspb.Message {
  getArtifactTypesList(): Array<ml_metadata_proto_metadata_store_pb.ArtifactType>;
  setArtifactTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ArtifactType>,
  ): GetArtifactTypesByExternalIdsResponse;
  clearArtifactTypesList(): GetArtifactTypesByExternalIdsResponse;
  addArtifactTypes(
    value?: ml_metadata_proto_metadata_store_pb.ArtifactType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ArtifactType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesByExternalIdsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactTypesByExternalIdsResponse,
  ): GetArtifactTypesByExternalIdsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactTypesByExternalIdsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesByExternalIdsResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactTypesByExternalIdsResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactTypesByExternalIdsResponse;
}

export namespace GetArtifactTypesByExternalIdsResponse {
  export type AsObject = {
    artifactTypesList: Array<ml_metadata_proto_metadata_store_pb.ArtifactType.AsObject>;
  };
}

export class GetExecutionTypesByExternalIdsRequest extends jspb.Message {
  getExternalIdsList(): Array<string>;
  setExternalIdsList(value: Array<string>): GetExecutionTypesByExternalIdsRequest;
  clearExternalIdsList(): GetExecutionTypesByExternalIdsRequest;
  addExternalIds(value: string, index?: number): GetExecutionTypesByExternalIdsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionTypesByExternalIdsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionTypesByExternalIdsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesByExternalIdsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionTypesByExternalIdsRequest,
  ): GetExecutionTypesByExternalIdsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionTypesByExternalIdsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesByExternalIdsRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionTypesByExternalIdsRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionTypesByExternalIdsRequest;
}

export namespace GetExecutionTypesByExternalIdsRequest {
  export type AsObject = {
    externalIdsList: Array<string>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionTypesByExternalIdsResponse extends jspb.Message {
  getExecutionTypesList(): Array<ml_metadata_proto_metadata_store_pb.ExecutionType>;
  setExecutionTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ExecutionType>,
  ): GetExecutionTypesByExternalIdsResponse;
  clearExecutionTypesList(): GetExecutionTypesByExternalIdsResponse;
  addExecutionTypes(
    value?: ml_metadata_proto_metadata_store_pb.ExecutionType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ExecutionType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesByExternalIdsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionTypesByExternalIdsResponse,
  ): GetExecutionTypesByExternalIdsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionTypesByExternalIdsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesByExternalIdsResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionTypesByExternalIdsResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionTypesByExternalIdsResponse;
}

export namespace GetExecutionTypesByExternalIdsResponse {
  export type AsObject = {
    executionTypesList: Array<ml_metadata_proto_metadata_store_pb.ExecutionType.AsObject>;
  };
}

export class GetContextTypesByExternalIdsRequest extends jspb.Message {
  getExternalIdsList(): Array<string>;
  setExternalIdsList(value: Array<string>): GetContextTypesByExternalIdsRequest;
  clearExternalIdsList(): GetContextTypesByExternalIdsRequest;
  addExternalIds(value: string, index?: number): GetContextTypesByExternalIdsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextTypesByExternalIdsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextTypesByExternalIdsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypesByExternalIdsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextTypesByExternalIdsRequest,
  ): GetContextTypesByExternalIdsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetContextTypesByExternalIdsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypesByExternalIdsRequest;
  static deserializeBinaryFromReader(
    message: GetContextTypesByExternalIdsRequest,
    reader: jspb.BinaryReader,
  ): GetContextTypesByExternalIdsRequest;
}

export namespace GetContextTypesByExternalIdsRequest {
  export type AsObject = {
    externalIdsList: Array<string>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextTypesByExternalIdsResponse extends jspb.Message {
  getContextTypesList(): Array<ml_metadata_proto_metadata_store_pb.ContextType>;
  setContextTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ContextType>,
  ): GetContextTypesByExternalIdsResponse;
  clearContextTypesList(): GetContextTypesByExternalIdsResponse;
  addContextTypes(
    value?: ml_metadata_proto_metadata_store_pb.ContextType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ContextType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypesByExternalIdsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextTypesByExternalIdsResponse,
  ): GetContextTypesByExternalIdsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetContextTypesByExternalIdsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypesByExternalIdsResponse;
  static deserializeBinaryFromReader(
    message: GetContextTypesByExternalIdsResponse,
    reader: jspb.BinaryReader,
  ): GetContextTypesByExternalIdsResponse;
}

export namespace GetContextTypesByExternalIdsResponse {
  export type AsObject = {
    contextTypesList: Array<ml_metadata_proto_metadata_store_pb.ContextType.AsObject>;
  };
}

export class GetExecutionsByTypeRequest extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): GetExecutionsByTypeRequest;

  getTypeVersion(): string;
  setTypeVersion(value: string): GetExecutionsByTypeRequest;

  getOptions(): ml_metadata_proto_metadata_store_pb.ListOperationOptions | undefined;
  setOptions(
    value?: ml_metadata_proto_metadata_store_pb.ListOperationOptions,
  ): GetExecutionsByTypeRequest;
  hasOptions(): boolean;
  clearOptions(): GetExecutionsByTypeRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionsByTypeRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionsByTypeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByTypeRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsByTypeRequest,
  ): GetExecutionsByTypeRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionsByTypeRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByTypeRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionsByTypeRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionsByTypeRequest;
}

export namespace GetExecutionsByTypeRequest {
  export type AsObject = {
    typeName: string;
    typeVersion: string;
    options?: ml_metadata_proto_metadata_store_pb.ListOperationOptions.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionsByTypeResponse extends jspb.Message {
  getExecutionsList(): Array<ml_metadata_proto_metadata_store_pb.Execution>;
  setExecutionsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Execution>,
  ): GetExecutionsByTypeResponse;
  clearExecutionsList(): GetExecutionsByTypeResponse;
  addExecutions(
    value?: ml_metadata_proto_metadata_store_pb.Execution,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Execution;

  getNextPageToken(): string;
  setNextPageToken(value: string): GetExecutionsByTypeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByTypeResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsByTypeResponse,
  ): GetExecutionsByTypeResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionsByTypeResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByTypeResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionsByTypeResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionsByTypeResponse;
}

export namespace GetExecutionsByTypeResponse {
  export type AsObject = {
    executionsList: Array<ml_metadata_proto_metadata_store_pb.Execution.AsObject>;
    nextPageToken: string;
  };
}

export class GetExecutionByTypeAndNameRequest extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): GetExecutionByTypeAndNameRequest;

  getTypeVersion(): string;
  setTypeVersion(value: string): GetExecutionByTypeAndNameRequest;

  getExecutionName(): string;
  setExecutionName(value: string): GetExecutionByTypeAndNameRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionByTypeAndNameRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionByTypeAndNameRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionByTypeAndNameRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionByTypeAndNameRequest,
  ): GetExecutionByTypeAndNameRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionByTypeAndNameRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionByTypeAndNameRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionByTypeAndNameRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionByTypeAndNameRequest;
}

export namespace GetExecutionByTypeAndNameRequest {
  export type AsObject = {
    typeName: string;
    typeVersion: string;
    executionName: string;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionByTypeAndNameResponse extends jspb.Message {
  getExecution(): ml_metadata_proto_metadata_store_pb.Execution | undefined;
  setExecution(
    value?: ml_metadata_proto_metadata_store_pb.Execution,
  ): GetExecutionByTypeAndNameResponse;
  hasExecution(): boolean;
  clearExecution(): GetExecutionByTypeAndNameResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionByTypeAndNameResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionByTypeAndNameResponse,
  ): GetExecutionByTypeAndNameResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionByTypeAndNameResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionByTypeAndNameResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionByTypeAndNameResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionByTypeAndNameResponse;
}

export namespace GetExecutionByTypeAndNameResponse {
  export type AsObject = {
    execution?: ml_metadata_proto_metadata_store_pb.Execution.AsObject;
  };
}

export class GetExecutionsByIDRequest extends jspb.Message {
  getExecutionIdsList(): Array<number>;
  setExecutionIdsList(value: Array<number>): GetExecutionsByIDRequest;
  clearExecutionIdsList(): GetExecutionsByIDRequest;
  addExecutionIds(value: number, index?: number): GetExecutionsByIDRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionsByIDRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionsByIDRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByIDRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsByIDRequest,
  ): GetExecutionsByIDRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionsByIDRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByIDRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionsByIDRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionsByIDRequest;
}

export namespace GetExecutionsByIDRequest {
  export type AsObject = {
    executionIdsList: Array<number>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionsByIDResponse extends jspb.Message {
  getExecutionsList(): Array<ml_metadata_proto_metadata_store_pb.Execution>;
  setExecutionsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Execution>,
  ): GetExecutionsByIDResponse;
  clearExecutionsList(): GetExecutionsByIDResponse;
  addExecutions(
    value?: ml_metadata_proto_metadata_store_pb.Execution,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Execution;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByIDResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsByIDResponse,
  ): GetExecutionsByIDResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionsByIDResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByIDResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionsByIDResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionsByIDResponse;
}

export namespace GetExecutionsByIDResponse {
  export type AsObject = {
    executionsList: Array<ml_metadata_proto_metadata_store_pb.Execution.AsObject>;
  };
}

export class GetExecutionTypeRequest extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): GetExecutionTypeRequest;

  getTypeVersion(): string;
  setTypeVersion(value: string): GetExecutionTypeRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionTypeRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionTypeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypeRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionTypeRequest,
  ): GetExecutionTypeRequest.AsObject;
  static serializeBinaryToWriter(message: GetExecutionTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypeRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionTypeRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionTypeRequest;
}

export namespace GetExecutionTypeRequest {
  export type AsObject = {
    typeName: string;
    typeVersion: string;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionTypeResponse extends jspb.Message {
  getExecutionType(): ml_metadata_proto_metadata_store_pb.ExecutionType | undefined;
  setExecutionType(
    value?: ml_metadata_proto_metadata_store_pb.ExecutionType,
  ): GetExecutionTypeResponse;
  hasExecutionType(): boolean;
  clearExecutionType(): GetExecutionTypeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypeResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionTypeResponse,
  ): GetExecutionTypeResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionTypeResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypeResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionTypeResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionTypeResponse;
}

export namespace GetExecutionTypeResponse {
  export type AsObject = {
    executionType?: ml_metadata_proto_metadata_store_pb.ExecutionType.AsObject;
  };
}

export class GetEventsByExecutionIDsRequest extends jspb.Message {
  getExecutionIdsList(): Array<number>;
  setExecutionIdsList(value: Array<number>): GetEventsByExecutionIDsRequest;
  clearExecutionIdsList(): GetEventsByExecutionIDsRequest;
  addExecutionIds(value: number, index?: number): GetEventsByExecutionIDsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetEventsByExecutionIDsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetEventsByExecutionIDsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEventsByExecutionIDsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetEventsByExecutionIDsRequest,
  ): GetEventsByExecutionIDsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetEventsByExecutionIDsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetEventsByExecutionIDsRequest;
  static deserializeBinaryFromReader(
    message: GetEventsByExecutionIDsRequest,
    reader: jspb.BinaryReader,
  ): GetEventsByExecutionIDsRequest;
}

export namespace GetEventsByExecutionIDsRequest {
  export type AsObject = {
    executionIdsList: Array<number>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetEventsByExecutionIDsResponse extends jspb.Message {
  getEventsList(): Array<ml_metadata_proto_metadata_store_pb.Event>;
  setEventsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Event>,
  ): GetEventsByExecutionIDsResponse;
  clearEventsList(): GetEventsByExecutionIDsResponse;
  addEvents(
    value?: ml_metadata_proto_metadata_store_pb.Event,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Event;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEventsByExecutionIDsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetEventsByExecutionIDsResponse,
  ): GetEventsByExecutionIDsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetEventsByExecutionIDsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetEventsByExecutionIDsResponse;
  static deserializeBinaryFromReader(
    message: GetEventsByExecutionIDsResponse,
    reader: jspb.BinaryReader,
  ): GetEventsByExecutionIDsResponse;
}

export namespace GetEventsByExecutionIDsResponse {
  export type AsObject = {
    eventsList: Array<ml_metadata_proto_metadata_store_pb.Event.AsObject>;
  };
}

export class GetEventsByArtifactIDsRequest extends jspb.Message {
  getArtifactIdsList(): Array<number>;
  setArtifactIdsList(value: Array<number>): GetEventsByArtifactIDsRequest;
  clearArtifactIdsList(): GetEventsByArtifactIDsRequest;
  addArtifactIds(value: number, index?: number): GetEventsByArtifactIDsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetEventsByArtifactIDsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetEventsByArtifactIDsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEventsByArtifactIDsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetEventsByArtifactIDsRequest,
  ): GetEventsByArtifactIDsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetEventsByArtifactIDsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetEventsByArtifactIDsRequest;
  static deserializeBinaryFromReader(
    message: GetEventsByArtifactIDsRequest,
    reader: jspb.BinaryReader,
  ): GetEventsByArtifactIDsRequest;
}

export namespace GetEventsByArtifactIDsRequest {
  export type AsObject = {
    artifactIdsList: Array<number>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetEventsByArtifactIDsResponse extends jspb.Message {
  getEventsList(): Array<ml_metadata_proto_metadata_store_pb.Event>;
  setEventsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Event>,
  ): GetEventsByArtifactIDsResponse;
  clearEventsList(): GetEventsByArtifactIDsResponse;
  addEvents(
    value?: ml_metadata_proto_metadata_store_pb.Event,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Event;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEventsByArtifactIDsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetEventsByArtifactIDsResponse,
  ): GetEventsByArtifactIDsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetEventsByArtifactIDsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetEventsByArtifactIDsResponse;
  static deserializeBinaryFromReader(
    message: GetEventsByArtifactIDsResponse,
    reader: jspb.BinaryReader,
  ): GetEventsByArtifactIDsResponse;
}

export namespace GetEventsByArtifactIDsResponse {
  export type AsObject = {
    eventsList: Array<ml_metadata_proto_metadata_store_pb.Event.AsObject>;
  };
}

export class GetArtifactTypesByIDRequest extends jspb.Message {
  getTypeIdsList(): Array<number>;
  setTypeIdsList(value: Array<number>): GetArtifactTypesByIDRequest;
  clearTypeIdsList(): GetArtifactTypesByIDRequest;
  addTypeIds(value: number, index?: number): GetArtifactTypesByIDRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactTypesByIDRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactTypesByIDRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesByIDRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactTypesByIDRequest,
  ): GetArtifactTypesByIDRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactTypesByIDRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesByIDRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactTypesByIDRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactTypesByIDRequest;
}

export namespace GetArtifactTypesByIDRequest {
  export type AsObject = {
    typeIdsList: Array<number>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactTypesByIDResponse extends jspb.Message {
  getArtifactTypesList(): Array<ml_metadata_proto_metadata_store_pb.ArtifactType>;
  setArtifactTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ArtifactType>,
  ): GetArtifactTypesByIDResponse;
  clearArtifactTypesList(): GetArtifactTypesByIDResponse;
  addArtifactTypes(
    value?: ml_metadata_proto_metadata_store_pb.ArtifactType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ArtifactType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactTypesByIDResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactTypesByIDResponse,
  ): GetArtifactTypesByIDResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactTypesByIDResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactTypesByIDResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactTypesByIDResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactTypesByIDResponse;
}

export namespace GetArtifactTypesByIDResponse {
  export type AsObject = {
    artifactTypesList: Array<ml_metadata_proto_metadata_store_pb.ArtifactType.AsObject>;
  };
}

export class GetExecutionTypesByIDRequest extends jspb.Message {
  getTypeIdsList(): Array<number>;
  setTypeIdsList(value: Array<number>): GetExecutionTypesByIDRequest;
  clearTypeIdsList(): GetExecutionTypesByIDRequest;
  addTypeIds(value: number, index?: number): GetExecutionTypesByIDRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionTypesByIDRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionTypesByIDRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesByIDRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionTypesByIDRequest,
  ): GetExecutionTypesByIDRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionTypesByIDRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesByIDRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionTypesByIDRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionTypesByIDRequest;
}

export namespace GetExecutionTypesByIDRequest {
  export type AsObject = {
    typeIdsList: Array<number>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionTypesByIDResponse extends jspb.Message {
  getExecutionTypesList(): Array<ml_metadata_proto_metadata_store_pb.ExecutionType>;
  setExecutionTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ExecutionType>,
  ): GetExecutionTypesByIDResponse;
  clearExecutionTypesList(): GetExecutionTypesByIDResponse;
  addExecutionTypes(
    value?: ml_metadata_proto_metadata_store_pb.ExecutionType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ExecutionType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionTypesByIDResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionTypesByIDResponse,
  ): GetExecutionTypesByIDResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionTypesByIDResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionTypesByIDResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionTypesByIDResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionTypesByIDResponse;
}

export namespace GetExecutionTypesByIDResponse {
  export type AsObject = {
    executionTypesList: Array<ml_metadata_proto_metadata_store_pb.ExecutionType.AsObject>;
  };
}

export class GetContextTypeRequest extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): GetContextTypeRequest;

  getTypeVersion(): string;
  setTypeVersion(value: string): GetContextTypeRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextTypeRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextTypeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypeRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextTypeRequest,
  ): GetContextTypeRequest.AsObject;
  static serializeBinaryToWriter(message: GetContextTypeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypeRequest;
  static deserializeBinaryFromReader(
    message: GetContextTypeRequest,
    reader: jspb.BinaryReader,
  ): GetContextTypeRequest;
}

export namespace GetContextTypeRequest {
  export type AsObject = {
    typeName: string;
    typeVersion: string;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextTypeResponse extends jspb.Message {
  getContextType(): ml_metadata_proto_metadata_store_pb.ContextType | undefined;
  setContextType(value?: ml_metadata_proto_metadata_store_pb.ContextType): GetContextTypeResponse;
  hasContextType(): boolean;
  clearContextType(): GetContextTypeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypeResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextTypeResponse,
  ): GetContextTypeResponse.AsObject;
  static serializeBinaryToWriter(message: GetContextTypeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypeResponse;
  static deserializeBinaryFromReader(
    message: GetContextTypeResponse,
    reader: jspb.BinaryReader,
  ): GetContextTypeResponse;
}

export namespace GetContextTypeResponse {
  export type AsObject = {
    contextType?: ml_metadata_proto_metadata_store_pb.ContextType.AsObject;
  };
}

export class GetContextTypesByIDRequest extends jspb.Message {
  getTypeIdsList(): Array<number>;
  setTypeIdsList(value: Array<number>): GetContextTypesByIDRequest;
  clearTypeIdsList(): GetContextTypesByIDRequest;
  addTypeIds(value: number, index?: number): GetContextTypesByIDRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextTypesByIDRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextTypesByIDRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypesByIDRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextTypesByIDRequest,
  ): GetContextTypesByIDRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetContextTypesByIDRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypesByIDRequest;
  static deserializeBinaryFromReader(
    message: GetContextTypesByIDRequest,
    reader: jspb.BinaryReader,
  ): GetContextTypesByIDRequest;
}

export namespace GetContextTypesByIDRequest {
  export type AsObject = {
    typeIdsList: Array<number>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextTypesByIDResponse extends jspb.Message {
  getContextTypesList(): Array<ml_metadata_proto_metadata_store_pb.ContextType>;
  setContextTypesList(
    value: Array<ml_metadata_proto_metadata_store_pb.ContextType>,
  ): GetContextTypesByIDResponse;
  clearContextTypesList(): GetContextTypesByIDResponse;
  addContextTypes(
    value?: ml_metadata_proto_metadata_store_pb.ContextType,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.ContextType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextTypesByIDResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextTypesByIDResponse,
  ): GetContextTypesByIDResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetContextTypesByIDResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextTypesByIDResponse;
  static deserializeBinaryFromReader(
    message: GetContextTypesByIDResponse,
    reader: jspb.BinaryReader,
  ): GetContextTypesByIDResponse;
}

export namespace GetContextTypesByIDResponse {
  export type AsObject = {
    contextTypesList: Array<ml_metadata_proto_metadata_store_pb.ContextType.AsObject>;
  };
}

export class GetContextsRequest extends jspb.Message {
  getOptions(): ml_metadata_proto_metadata_store_pb.ListOperationOptions | undefined;
  setOptions(value?: ml_metadata_proto_metadata_store_pb.ListOperationOptions): GetContextsRequest;
  hasOptions(): boolean;
  clearOptions(): GetContextsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsRequest): GetContextsRequest.AsObject;
  static serializeBinaryToWriter(message: GetContextsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsRequest;
  static deserializeBinaryFromReader(
    message: GetContextsRequest,
    reader: jspb.BinaryReader,
  ): GetContextsRequest;
}

export namespace GetContextsRequest {
  export type AsObject = {
    options?: ml_metadata_proto_metadata_store_pb.ListOperationOptions.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextsResponse extends jspb.Message {
  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(value: Array<ml_metadata_proto_metadata_store_pb.Context>): GetContextsResponse;
  clearContextsList(): GetContextsResponse;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  getNextPageToken(): string;
  setNextPageToken(value: string): GetContextsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetContextsResponse): GetContextsResponse.AsObject;
  static serializeBinaryToWriter(message: GetContextsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsResponse;
  static deserializeBinaryFromReader(
    message: GetContextsResponse,
    reader: jspb.BinaryReader,
  ): GetContextsResponse;
}

export namespace GetContextsResponse {
  export type AsObject = {
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
    nextPageToken: string;
  };
}

export class GetContextsByTypeRequest extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): GetContextsByTypeRequest;

  getOptions(): ml_metadata_proto_metadata_store_pb.ListOperationOptions | undefined;
  setOptions(
    value?: ml_metadata_proto_metadata_store_pb.ListOperationOptions,
  ): GetContextsByTypeRequest;
  hasOptions(): boolean;
  clearOptions(): GetContextsByTypeRequest;

  getTypeVersion(): string;
  setTypeVersion(value: string): GetContextsByTypeRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextsByTypeRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextsByTypeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByTypeRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByTypeRequest,
  ): GetContextsByTypeRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetContextsByTypeRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByTypeRequest;
  static deserializeBinaryFromReader(
    message: GetContextsByTypeRequest,
    reader: jspb.BinaryReader,
  ): GetContextsByTypeRequest;
}

export namespace GetContextsByTypeRequest {
  export type AsObject = {
    typeName: string;
    options?: ml_metadata_proto_metadata_store_pb.ListOperationOptions.AsObject;
    typeVersion: string;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextsByTypeResponse extends jspb.Message {
  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Context>,
  ): GetContextsByTypeResponse;
  clearContextsList(): GetContextsByTypeResponse;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  getNextPageToken(): string;
  setNextPageToken(value: string): GetContextsByTypeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByTypeResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByTypeResponse,
  ): GetContextsByTypeResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetContextsByTypeResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByTypeResponse;
  static deserializeBinaryFromReader(
    message: GetContextsByTypeResponse,
    reader: jspb.BinaryReader,
  ): GetContextsByTypeResponse;
}

export namespace GetContextsByTypeResponse {
  export type AsObject = {
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
    nextPageToken: string;
  };
}

export class GetContextByTypeAndNameRequest extends jspb.Message {
  getTypeName(): string;
  setTypeName(value: string): GetContextByTypeAndNameRequest;

  getTypeVersion(): string;
  setTypeVersion(value: string): GetContextByTypeAndNameRequest;

  getContextName(): string;
  setContextName(value: string): GetContextByTypeAndNameRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextByTypeAndNameRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextByTypeAndNameRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextByTypeAndNameRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextByTypeAndNameRequest,
  ): GetContextByTypeAndNameRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetContextByTypeAndNameRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextByTypeAndNameRequest;
  static deserializeBinaryFromReader(
    message: GetContextByTypeAndNameRequest,
    reader: jspb.BinaryReader,
  ): GetContextByTypeAndNameRequest;
}

export namespace GetContextByTypeAndNameRequest {
  export type AsObject = {
    typeName: string;
    typeVersion: string;
    contextName: string;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextByTypeAndNameResponse extends jspb.Message {
  getContext(): ml_metadata_proto_metadata_store_pb.Context | undefined;
  setContext(value?: ml_metadata_proto_metadata_store_pb.Context): GetContextByTypeAndNameResponse;
  hasContext(): boolean;
  clearContext(): GetContextByTypeAndNameResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextByTypeAndNameResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextByTypeAndNameResponse,
  ): GetContextByTypeAndNameResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetContextByTypeAndNameResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextByTypeAndNameResponse;
  static deserializeBinaryFromReader(
    message: GetContextByTypeAndNameResponse,
    reader: jspb.BinaryReader,
  ): GetContextByTypeAndNameResponse;
}

export namespace GetContextByTypeAndNameResponse {
  export type AsObject = {
    context?: ml_metadata_proto_metadata_store_pb.Context.AsObject;
  };
}

export class GetContextsByIDRequest extends jspb.Message {
  getContextIdsList(): Array<number>;
  setContextIdsList(value: Array<number>): GetContextsByIDRequest;
  clearContextIdsList(): GetContextsByIDRequest;
  addContextIds(value: number, index?: number): GetContextsByIDRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextsByIDRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextsByIDRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByIDRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByIDRequest,
  ): GetContextsByIDRequest.AsObject;
  static serializeBinaryToWriter(message: GetContextsByIDRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByIDRequest;
  static deserializeBinaryFromReader(
    message: GetContextsByIDRequest,
    reader: jspb.BinaryReader,
  ): GetContextsByIDRequest;
}

export namespace GetContextsByIDRequest {
  export type AsObject = {
    contextIdsList: Array<number>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextsByIDResponse extends jspb.Message {
  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Context>,
  ): GetContextsByIDResponse;
  clearContextsList(): GetContextsByIDResponse;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByIDResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByIDResponse,
  ): GetContextsByIDResponse.AsObject;
  static serializeBinaryToWriter(message: GetContextsByIDResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByIDResponse;
  static deserializeBinaryFromReader(
    message: GetContextsByIDResponse,
    reader: jspb.BinaryReader,
  ): GetContextsByIDResponse;
}

export namespace GetContextsByIDResponse {
  export type AsObject = {
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
  };
}

export class GetContextsByArtifactRequest extends jspb.Message {
  getArtifactId(): number;
  setArtifactId(value: number): GetContextsByArtifactRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextsByArtifactRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextsByArtifactRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByArtifactRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByArtifactRequest,
  ): GetContextsByArtifactRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetContextsByArtifactRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByArtifactRequest;
  static deserializeBinaryFromReader(
    message: GetContextsByArtifactRequest,
    reader: jspb.BinaryReader,
  ): GetContextsByArtifactRequest;
}

export namespace GetContextsByArtifactRequest {
  export type AsObject = {
    artifactId: number;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextsByArtifactResponse extends jspb.Message {
  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Context>,
  ): GetContextsByArtifactResponse;
  clearContextsList(): GetContextsByArtifactResponse;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByArtifactResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByArtifactResponse,
  ): GetContextsByArtifactResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetContextsByArtifactResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByArtifactResponse;
  static deserializeBinaryFromReader(
    message: GetContextsByArtifactResponse,
    reader: jspb.BinaryReader,
  ): GetContextsByArtifactResponse;
}

export namespace GetContextsByArtifactResponse {
  export type AsObject = {
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
  };
}

export class GetContextsByExecutionRequest extends jspb.Message {
  getExecutionId(): number;
  setExecutionId(value: number): GetContextsByExecutionRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetContextsByExecutionRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetContextsByExecutionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByExecutionRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByExecutionRequest,
  ): GetContextsByExecutionRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetContextsByExecutionRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByExecutionRequest;
  static deserializeBinaryFromReader(
    message: GetContextsByExecutionRequest,
    reader: jspb.BinaryReader,
  ): GetContextsByExecutionRequest;
}

export namespace GetContextsByExecutionRequest {
  export type AsObject = {
    executionId: number;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetContextsByExecutionResponse extends jspb.Message {
  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Context>,
  ): GetContextsByExecutionResponse;
  clearContextsList(): GetContextsByExecutionResponse;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetContextsByExecutionResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetContextsByExecutionResponse,
  ): GetContextsByExecutionResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetContextsByExecutionResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetContextsByExecutionResponse;
  static deserializeBinaryFromReader(
    message: GetContextsByExecutionResponse,
    reader: jspb.BinaryReader,
  ): GetContextsByExecutionResponse;
}

export namespace GetContextsByExecutionResponse {
  export type AsObject = {
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
  };
}

export class GetParentContextsByContextRequest extends jspb.Message {
  getContextId(): number;
  setContextId(value: number): GetParentContextsByContextRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetParentContextsByContextRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetParentContextsByContextRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetParentContextsByContextRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetParentContextsByContextRequest,
  ): GetParentContextsByContextRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetParentContextsByContextRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetParentContextsByContextRequest;
  static deserializeBinaryFromReader(
    message: GetParentContextsByContextRequest,
    reader: jspb.BinaryReader,
  ): GetParentContextsByContextRequest;
}

export namespace GetParentContextsByContextRequest {
  export type AsObject = {
    contextId: number;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetParentContextsByContextResponse extends jspb.Message {
  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Context>,
  ): GetParentContextsByContextResponse;
  clearContextsList(): GetParentContextsByContextResponse;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetParentContextsByContextResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetParentContextsByContextResponse,
  ): GetParentContextsByContextResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetParentContextsByContextResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetParentContextsByContextResponse;
  static deserializeBinaryFromReader(
    message: GetParentContextsByContextResponse,
    reader: jspb.BinaryReader,
  ): GetParentContextsByContextResponse;
}

export namespace GetParentContextsByContextResponse {
  export type AsObject = {
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
  };
}

export class GetChildrenContextsByContextRequest extends jspb.Message {
  getContextId(): number;
  setContextId(value: number): GetChildrenContextsByContextRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetChildrenContextsByContextRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetChildrenContextsByContextRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetChildrenContextsByContextRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetChildrenContextsByContextRequest,
  ): GetChildrenContextsByContextRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetChildrenContextsByContextRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetChildrenContextsByContextRequest;
  static deserializeBinaryFromReader(
    message: GetChildrenContextsByContextRequest,
    reader: jspb.BinaryReader,
  ): GetChildrenContextsByContextRequest;
}

export namespace GetChildrenContextsByContextRequest {
  export type AsObject = {
    contextId: number;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetChildrenContextsByContextResponse extends jspb.Message {
  getContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
  setContextsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Context>,
  ): GetChildrenContextsByContextResponse;
  clearContextsList(): GetChildrenContextsByContextResponse;
  addContexts(
    value?: ml_metadata_proto_metadata_store_pb.Context,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Context;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetChildrenContextsByContextResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetChildrenContextsByContextResponse,
  ): GetChildrenContextsByContextResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetChildrenContextsByContextResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetChildrenContextsByContextResponse;
  static deserializeBinaryFromReader(
    message: GetChildrenContextsByContextResponse,
    reader: jspb.BinaryReader,
  ): GetChildrenContextsByContextResponse;
}

export namespace GetChildrenContextsByContextResponse {
  export type AsObject = {
    contextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
  };
}

export class GetParentContextsByContextsRequest extends jspb.Message {
  getContextIdsList(): Array<number>;
  setContextIdsList(value: Array<number>): GetParentContextsByContextsRequest;
  clearContextIdsList(): GetParentContextsByContextsRequest;
  addContextIds(value: number, index?: number): GetParentContextsByContextsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetParentContextsByContextsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetParentContextsByContextsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetParentContextsByContextsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetParentContextsByContextsRequest,
  ): GetParentContextsByContextsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetParentContextsByContextsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetParentContextsByContextsRequest;
  static deserializeBinaryFromReader(
    message: GetParentContextsByContextsRequest,
    reader: jspb.BinaryReader,
  ): GetParentContextsByContextsRequest;
}

export namespace GetParentContextsByContextsRequest {
  export type AsObject = {
    contextIdsList: Array<number>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetParentContextsByContextsResponse extends jspb.Message {
  getContextsMap(): jspb.Map<number, GetParentContextsByContextsResponse.ParentContextsPerChild>;
  clearContextsMap(): GetParentContextsByContextsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetParentContextsByContextsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetParentContextsByContextsResponse,
  ): GetParentContextsByContextsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetParentContextsByContextsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetParentContextsByContextsResponse;
  static deserializeBinaryFromReader(
    message: GetParentContextsByContextsResponse,
    reader: jspb.BinaryReader,
  ): GetParentContextsByContextsResponse;
}

export namespace GetParentContextsByContextsResponse {
  export type AsObject = {
    contextsMap: Array<
      [number, GetParentContextsByContextsResponse.ParentContextsPerChild.AsObject]
    >;
  };

  export class ParentContextsPerChild extends jspb.Message {
    getParentContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
    setParentContextsList(
      value: Array<ml_metadata_proto_metadata_store_pb.Context>,
    ): ParentContextsPerChild;
    clearParentContextsList(): ParentContextsPerChild;
    addParentContexts(
      value?: ml_metadata_proto_metadata_store_pb.Context,
      index?: number,
    ): ml_metadata_proto_metadata_store_pb.Context;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ParentContextsPerChild.AsObject;
    static toObject(
      includeInstance: boolean,
      msg: ParentContextsPerChild,
    ): ParentContextsPerChild.AsObject;
    static serializeBinaryToWriter(
      message: ParentContextsPerChild,
      writer: jspb.BinaryWriter,
    ): void;
    static deserializeBinary(bytes: Uint8Array): ParentContextsPerChild;
    static deserializeBinaryFromReader(
      message: ParentContextsPerChild,
      reader: jspb.BinaryReader,
    ): ParentContextsPerChild;
  }

  export namespace ParentContextsPerChild {
    export type AsObject = {
      parentContextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
    };
  }
}

export class GetChildrenContextsByContextsRequest extends jspb.Message {
  getContextIdsList(): Array<number>;
  setContextIdsList(value: Array<number>): GetChildrenContextsByContextsRequest;
  clearContextIdsList(): GetChildrenContextsByContextsRequest;
  addContextIds(value: number, index?: number): GetChildrenContextsByContextsRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetChildrenContextsByContextsRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetChildrenContextsByContextsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetChildrenContextsByContextsRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetChildrenContextsByContextsRequest,
  ): GetChildrenContextsByContextsRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetChildrenContextsByContextsRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetChildrenContextsByContextsRequest;
  static deserializeBinaryFromReader(
    message: GetChildrenContextsByContextsRequest,
    reader: jspb.BinaryReader,
  ): GetChildrenContextsByContextsRequest;
}

export namespace GetChildrenContextsByContextsRequest {
  export type AsObject = {
    contextIdsList: Array<number>;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetChildrenContextsByContextsResponse extends jspb.Message {
  getContextsMap(): jspb.Map<
    number,
    GetChildrenContextsByContextsResponse.ChildrenContextsPerParent
  >;
  clearContextsMap(): GetChildrenContextsByContextsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetChildrenContextsByContextsResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetChildrenContextsByContextsResponse,
  ): GetChildrenContextsByContextsResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetChildrenContextsByContextsResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetChildrenContextsByContextsResponse;
  static deserializeBinaryFromReader(
    message: GetChildrenContextsByContextsResponse,
    reader: jspb.BinaryReader,
  ): GetChildrenContextsByContextsResponse;
}

export namespace GetChildrenContextsByContextsResponse {
  export type AsObject = {
    contextsMap: Array<
      [number, GetChildrenContextsByContextsResponse.ChildrenContextsPerParent.AsObject]
    >;
  };

  export class ChildrenContextsPerParent extends jspb.Message {
    getChildrenContextsList(): Array<ml_metadata_proto_metadata_store_pb.Context>;
    setChildrenContextsList(
      value: Array<ml_metadata_proto_metadata_store_pb.Context>,
    ): ChildrenContextsPerParent;
    clearChildrenContextsList(): ChildrenContextsPerParent;
    addChildrenContexts(
      value?: ml_metadata_proto_metadata_store_pb.Context,
      index?: number,
    ): ml_metadata_proto_metadata_store_pb.Context;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ChildrenContextsPerParent.AsObject;
    static toObject(
      includeInstance: boolean,
      msg: ChildrenContextsPerParent,
    ): ChildrenContextsPerParent.AsObject;
    static serializeBinaryToWriter(
      message: ChildrenContextsPerParent,
      writer: jspb.BinaryWriter,
    ): void;
    static deserializeBinary(bytes: Uint8Array): ChildrenContextsPerParent;
    static deserializeBinaryFromReader(
      message: ChildrenContextsPerParent,
      reader: jspb.BinaryReader,
    ): ChildrenContextsPerParent;
  }

  export namespace ChildrenContextsPerParent {
    export type AsObject = {
      childrenContextsList: Array<ml_metadata_proto_metadata_store_pb.Context.AsObject>;
    };
  }
}

export class GetArtifactsByContextRequest extends jspb.Message {
  getContextId(): number;
  setContextId(value: number): GetArtifactsByContextRequest;

  getOptions(): ml_metadata_proto_metadata_store_pb.ListOperationOptions | undefined;
  setOptions(
    value?: ml_metadata_proto_metadata_store_pb.ListOperationOptions,
  ): GetArtifactsByContextRequest;
  hasOptions(): boolean;
  clearOptions(): GetArtifactsByContextRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetArtifactsByContextRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetArtifactsByContextRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByContextRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByContextRequest,
  ): GetArtifactsByContextRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactsByContextRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByContextRequest;
  static deserializeBinaryFromReader(
    message: GetArtifactsByContextRequest,
    reader: jspb.BinaryReader,
  ): GetArtifactsByContextRequest;
}

export namespace GetArtifactsByContextRequest {
  export type AsObject = {
    contextId: number;
    options?: ml_metadata_proto_metadata_store_pb.ListOperationOptions.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetArtifactsByContextResponse extends jspb.Message {
  getArtifactsList(): Array<ml_metadata_proto_metadata_store_pb.Artifact>;
  setArtifactsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Artifact>,
  ): GetArtifactsByContextResponse;
  clearArtifactsList(): GetArtifactsByContextResponse;
  addArtifacts(
    value?: ml_metadata_proto_metadata_store_pb.Artifact,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Artifact;

  getNextPageToken(): string;
  setNextPageToken(value: string): GetArtifactsByContextResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetArtifactsByContextResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetArtifactsByContextResponse,
  ): GetArtifactsByContextResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetArtifactsByContextResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetArtifactsByContextResponse;
  static deserializeBinaryFromReader(
    message: GetArtifactsByContextResponse,
    reader: jspb.BinaryReader,
  ): GetArtifactsByContextResponse;
}

export namespace GetArtifactsByContextResponse {
  export type AsObject = {
    artifactsList: Array<ml_metadata_proto_metadata_store_pb.Artifact.AsObject>;
    nextPageToken: string;
  };
}

export class GetExecutionsByContextRequest extends jspb.Message {
  getContextId(): number;
  setContextId(value: number): GetExecutionsByContextRequest;

  getOptions(): ml_metadata_proto_metadata_store_pb.ListOperationOptions | undefined;
  setOptions(
    value?: ml_metadata_proto_metadata_store_pb.ListOperationOptions,
  ): GetExecutionsByContextRequest;
  hasOptions(): boolean;
  clearOptions(): GetExecutionsByContextRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionsByContextRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionsByContextRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByContextRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsByContextRequest,
  ): GetExecutionsByContextRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionsByContextRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByContextRequest;
  static deserializeBinaryFromReader(
    message: GetExecutionsByContextRequest,
    reader: jspb.BinaryReader,
  ): GetExecutionsByContextRequest;
}

export namespace GetExecutionsByContextRequest {
  export type AsObject = {
    contextId: number;
    options?: ml_metadata_proto_metadata_store_pb.ListOperationOptions.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetExecutionsByContextResponse extends jspb.Message {
  getExecutionsList(): Array<ml_metadata_proto_metadata_store_pb.Execution>;
  setExecutionsList(
    value: Array<ml_metadata_proto_metadata_store_pb.Execution>,
  ): GetExecutionsByContextResponse;
  clearExecutionsList(): GetExecutionsByContextResponse;
  addExecutions(
    value?: ml_metadata_proto_metadata_store_pb.Execution,
    index?: number,
  ): ml_metadata_proto_metadata_store_pb.Execution;

  getNextPageToken(): string;
  setNextPageToken(value: string): GetExecutionsByContextResponse;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetExecutionsByContextResponse;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetExecutionsByContextResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExecutionsByContextResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetExecutionsByContextResponse,
  ): GetExecutionsByContextResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetExecutionsByContextResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetExecutionsByContextResponse;
  static deserializeBinaryFromReader(
    message: GetExecutionsByContextResponse,
    reader: jspb.BinaryReader,
  ): GetExecutionsByContextResponse;
}

export namespace GetExecutionsByContextResponse {
  export type AsObject = {
    executionsList: Array<ml_metadata_proto_metadata_store_pb.Execution.AsObject>;
    nextPageToken: string;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetLineageGraphRequest extends jspb.Message {
  getOptions(): ml_metadata_proto_metadata_store_pb.LineageGraphQueryOptions | undefined;
  setOptions(
    value?: ml_metadata_proto_metadata_store_pb.LineageGraphQueryOptions,
  ): GetLineageGraphRequest;
  hasOptions(): boolean;
  clearOptions(): GetLineageGraphRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetLineageGraphRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetLineageGraphRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLineageGraphRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetLineageGraphRequest,
  ): GetLineageGraphRequest.AsObject;
  static serializeBinaryToWriter(message: GetLineageGraphRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetLineageGraphRequest;
  static deserializeBinaryFromReader(
    message: GetLineageGraphRequest,
    reader: jspb.BinaryReader,
  ): GetLineageGraphRequest;
}

export namespace GetLineageGraphRequest {
  export type AsObject = {
    options?: ml_metadata_proto_metadata_store_pb.LineageGraphQueryOptions.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetLineageGraphResponse extends jspb.Message {
  getSubgraph(): ml_metadata_proto_metadata_store_pb.LineageGraph | undefined;
  setSubgraph(value?: ml_metadata_proto_metadata_store_pb.LineageGraph): GetLineageGraphResponse;
  hasSubgraph(): boolean;
  clearSubgraph(): GetLineageGraphResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLineageGraphResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetLineageGraphResponse,
  ): GetLineageGraphResponse.AsObject;
  static serializeBinaryToWriter(message: GetLineageGraphResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetLineageGraphResponse;
  static deserializeBinaryFromReader(
    message: GetLineageGraphResponse,
    reader: jspb.BinaryReader,
  ): GetLineageGraphResponse;
}

export namespace GetLineageGraphResponse {
  export type AsObject = {
    subgraph?: ml_metadata_proto_metadata_store_pb.LineageGraph.AsObject;
  };
}

export class GetLineageSubgraphRequest extends jspb.Message {
  getLineageSubgraphQueryOptions():
    | ml_metadata_proto_metadata_store_pb.LineageSubgraphQueryOptions
    | undefined;
  setLineageSubgraphQueryOptions(
    value?: ml_metadata_proto_metadata_store_pb.LineageSubgraphQueryOptions,
  ): GetLineageSubgraphRequest;
  hasLineageSubgraphQueryOptions(): boolean;
  clearLineageSubgraphQueryOptions(): GetLineageSubgraphRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetLineageSubgraphRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetLineageSubgraphRequest;

  getTransactionOptions(): ml_metadata_proto_metadata_store_pb.TransactionOptions | undefined;
  setTransactionOptions(
    value?: ml_metadata_proto_metadata_store_pb.TransactionOptions,
  ): GetLineageSubgraphRequest;
  hasTransactionOptions(): boolean;
  clearTransactionOptions(): GetLineageSubgraphRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLineageSubgraphRequest.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetLineageSubgraphRequest,
  ): GetLineageSubgraphRequest.AsObject;
  static serializeBinaryToWriter(
    message: GetLineageSubgraphRequest,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetLineageSubgraphRequest;
  static deserializeBinaryFromReader(
    message: GetLineageSubgraphRequest,
    reader: jspb.BinaryReader,
  ): GetLineageSubgraphRequest;
}

export namespace GetLineageSubgraphRequest {
  export type AsObject = {
    lineageSubgraphQueryOptions?: ml_metadata_proto_metadata_store_pb.LineageSubgraphQueryOptions.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    transactionOptions?: ml_metadata_proto_metadata_store_pb.TransactionOptions.AsObject;
  };
}

export class GetLineageSubgraphResponse extends jspb.Message {
  getLineageSubgraph(): ml_metadata_proto_metadata_store_pb.LineageGraph | undefined;
  setLineageSubgraph(
    value?: ml_metadata_proto_metadata_store_pb.LineageGraph,
  ): GetLineageSubgraphResponse;
  hasLineageSubgraph(): boolean;
  clearLineageSubgraph(): GetLineageSubgraphResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLineageSubgraphResponse.AsObject;
  static toObject(
    includeInstance: boolean,
    msg: GetLineageSubgraphResponse,
  ): GetLineageSubgraphResponse.AsObject;
  static serializeBinaryToWriter(
    message: GetLineageSubgraphResponse,
    writer: jspb.BinaryWriter,
  ): void;
  static deserializeBinary(bytes: Uint8Array): GetLineageSubgraphResponse;
  static deserializeBinaryFromReader(
    message: GetLineageSubgraphResponse,
    reader: jspb.BinaryReader,
  ): GetLineageSubgraphResponse;
}

export namespace GetLineageSubgraphResponse {
  export type AsObject = {
    lineageSubgraph?: ml_metadata_proto_metadata_store_pb.LineageGraph.AsObject;
  };
}
