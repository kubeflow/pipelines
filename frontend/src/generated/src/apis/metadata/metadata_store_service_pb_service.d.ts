// package: ml_metadata
// file: src/apis/metadata/metadata_store_service.proto

import * as src_apis_metadata_metadata_store_service_pb from "./metadata_store_service_pb";
import {grpc} from "@improbable-eng/grpc-web";

type MetadataStoreServicePutArtifacts = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutArtifactsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutArtifactsResponse;
};

type MetadataStoreServicePutArtifactType = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutArtifactTypeRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutArtifactTypeResponse;
};

type MetadataStoreServicePutExecutions = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutExecutionsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutExecutionsResponse;
};

type MetadataStoreServicePutExecutionType = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutExecutionTypeRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutExecutionTypeResponse;
};

type MetadataStoreServicePutEvents = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutEventsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutEventsResponse;
};

type MetadataStoreServicePutExecution = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutExecutionRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutExecutionResponse;
};

type MetadataStoreServicePutTypes = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutTypesRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutTypesResponse;
};

type MetadataStoreServicePutContextType = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutContextTypeRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutContextTypeResponse;
};

type MetadataStoreServicePutContexts = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutContextsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutContextsResponse;
};

type MetadataStoreServicePutAttributionsAndAssociations = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutAttributionsAndAssociationsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutAttributionsAndAssociationsResponse;
};

type MetadataStoreServicePutParentContexts = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.PutParentContextsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.PutParentContextsResponse;
};

type MetadataStoreServiceGetArtifactType = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactTypeRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactTypeResponse;
};

type MetadataStoreServiceGetArtifactTypesByID = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactTypesByIDRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactTypesByIDResponse;
};

type MetadataStoreServiceGetArtifactTypes = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactTypesRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactTypesResponse;
};

type MetadataStoreServiceGetExecutionType = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionTypeRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionTypeResponse;
};

type MetadataStoreServiceGetExecutionTypesByID = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionTypesByIDRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionTypesByIDResponse;
};

type MetadataStoreServiceGetExecutionTypes = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionTypesRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionTypesResponse;
};

type MetadataStoreServiceGetContextType = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetContextTypeRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetContextTypeResponse;
};

type MetadataStoreServiceGetContextTypesByID = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetContextTypesByIDRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetContextTypesByIDResponse;
};

type MetadataStoreServiceGetArtifacts = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsResponse;
};

type MetadataStoreServiceGetExecutions = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionsResponse;
};

type MetadataStoreServiceGetContexts = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsResponse;
};

type MetadataStoreServiceGetArtifactsByID = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsByIDRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsByIDResponse;
};

type MetadataStoreServiceGetExecutionsByID = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionsByIDRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionsByIDResponse;
};

type MetadataStoreServiceGetContextsByID = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsByIDRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsByIDResponse;
};

type MetadataStoreServiceGetArtifactsByType = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsByTypeRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsByTypeResponse;
};

type MetadataStoreServiceGetExecutionsByType = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionsByTypeRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionsByTypeResponse;
};

type MetadataStoreServiceGetContextsByType = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsByTypeRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsByTypeResponse;
};

type MetadataStoreServiceGetArtifactsByURI = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsByURIRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsByURIResponse;
};

type MetadataStoreServiceGetEventsByExecutionIDs = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetEventsByExecutionIDsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetEventsByExecutionIDsResponse;
};

type MetadataStoreServiceGetEventsByArtifactIDs = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetEventsByArtifactIDsRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetEventsByArtifactIDsResponse;
};

type MetadataStoreServiceGetContextsByArtifact = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsByArtifactRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsByArtifactResponse;
};

type MetadataStoreServiceGetContextsByExecution = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsByExecutionRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetContextsByExecutionResponse;
};

type MetadataStoreServiceGetParentContextsByContext = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetParentContextsByContextRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetParentContextsByContextResponse;
};

type MetadataStoreServiceGetChildrenContextsByContext = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetChildrenContextsByContextRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetChildrenContextsByContextResponse;
};

type MetadataStoreServiceGetArtifactsByContext = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsByContextRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetArtifactsByContextResponse;
};

type MetadataStoreServiceGetExecutionsByContext = {
  readonly methodName: string;
  readonly service: typeof MetadataStoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionsByContextRequest;
  readonly responseType: typeof src_apis_metadata_metadata_store_service_pb.GetExecutionsByContextResponse;
};

export class MetadataStoreService {
  static readonly serviceName: string;
  static readonly PutArtifacts: MetadataStoreServicePutArtifacts;
  static readonly PutArtifactType: MetadataStoreServicePutArtifactType;
  static readonly PutExecutions: MetadataStoreServicePutExecutions;
  static readonly PutExecutionType: MetadataStoreServicePutExecutionType;
  static readonly PutEvents: MetadataStoreServicePutEvents;
  static readonly PutExecution: MetadataStoreServicePutExecution;
  static readonly PutTypes: MetadataStoreServicePutTypes;
  static readonly PutContextType: MetadataStoreServicePutContextType;
  static readonly PutContexts: MetadataStoreServicePutContexts;
  static readonly PutAttributionsAndAssociations: MetadataStoreServicePutAttributionsAndAssociations;
  static readonly PutParentContexts: MetadataStoreServicePutParentContexts;
  static readonly GetArtifactType: MetadataStoreServiceGetArtifactType;
  static readonly GetArtifactTypesByID: MetadataStoreServiceGetArtifactTypesByID;
  static readonly GetArtifactTypes: MetadataStoreServiceGetArtifactTypes;
  static readonly GetExecutionType: MetadataStoreServiceGetExecutionType;
  static readonly GetExecutionTypesByID: MetadataStoreServiceGetExecutionTypesByID;
  static readonly GetExecutionTypes: MetadataStoreServiceGetExecutionTypes;
  static readonly GetContextType: MetadataStoreServiceGetContextType;
  static readonly GetContextTypesByID: MetadataStoreServiceGetContextTypesByID;
  static readonly GetArtifacts: MetadataStoreServiceGetArtifacts;
  static readonly GetExecutions: MetadataStoreServiceGetExecutions;
  static readonly GetContexts: MetadataStoreServiceGetContexts;
  static readonly GetArtifactsByID: MetadataStoreServiceGetArtifactsByID;
  static readonly GetExecutionsByID: MetadataStoreServiceGetExecutionsByID;
  static readonly GetContextsByID: MetadataStoreServiceGetContextsByID;
  static readonly GetArtifactsByType: MetadataStoreServiceGetArtifactsByType;
  static readonly GetExecutionsByType: MetadataStoreServiceGetExecutionsByType;
  static readonly GetContextsByType: MetadataStoreServiceGetContextsByType;
  static readonly GetArtifactsByURI: MetadataStoreServiceGetArtifactsByURI;
  static readonly GetEventsByExecutionIDs: MetadataStoreServiceGetEventsByExecutionIDs;
  static readonly GetEventsByArtifactIDs: MetadataStoreServiceGetEventsByArtifactIDs;
  static readonly GetContextsByArtifact: MetadataStoreServiceGetContextsByArtifact;
  static readonly GetContextsByExecution: MetadataStoreServiceGetContextsByExecution;
  static readonly GetParentContextsByContext: MetadataStoreServiceGetParentContextsByContext;
  static readonly GetChildrenContextsByContext: MetadataStoreServiceGetChildrenContextsByContext;
  static readonly GetArtifactsByContext: MetadataStoreServiceGetArtifactsByContext;
  static readonly GetExecutionsByContext: MetadataStoreServiceGetExecutionsByContext;
}

export type ServiceError = { message: string, code: number; metadata: grpc.Metadata }
export type Status = { details: string, code: number; metadata: grpc.Metadata }

interface UnaryResponse {
  cancel(): void;
}
interface ResponseStream<T> {
  cancel(): void;
  on(type: 'data', handler: (message: T) => void): ResponseStream<T>;
  on(type: 'end', handler: (status?: Status) => void): ResponseStream<T>;
  on(type: 'status', handler: (status: Status) => void): ResponseStream<T>;
}
interface RequestStream<T> {
  write(message: T): RequestStream<T>;
  end(): void;
  cancel(): void;
  on(type: 'end', handler: (status?: Status) => void): RequestStream<T>;
  on(type: 'status', handler: (status: Status) => void): RequestStream<T>;
}
interface BidirectionalStream<ReqT, ResT> {
  write(message: ReqT): BidirectionalStream<ReqT, ResT>;
  end(): void;
  cancel(): void;
  on(type: 'data', handler: (message: ResT) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'end', handler: (status?: Status) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'status', handler: (status: Status) => void): BidirectionalStream<ReqT, ResT>;
}

export class MetadataStoreServiceClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  putArtifacts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutArtifactsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutArtifactsResponse|null) => void
  ): UnaryResponse;
  putArtifacts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutArtifactsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutArtifactsResponse|null) => void
  ): UnaryResponse;
  putArtifactType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutArtifactTypeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutArtifactTypeResponse|null) => void
  ): UnaryResponse;
  putArtifactType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutArtifactTypeRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutArtifactTypeResponse|null) => void
  ): UnaryResponse;
  putExecutions(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionsResponse|null) => void
  ): UnaryResponse;
  putExecutions(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionsResponse|null) => void
  ): UnaryResponse;
  putExecutionType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionTypeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionTypeResponse|null) => void
  ): UnaryResponse;
  putExecutionType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionTypeRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionTypeResponse|null) => void
  ): UnaryResponse;
  putEvents(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutEventsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutEventsResponse|null) => void
  ): UnaryResponse;
  putEvents(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutEventsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutEventsResponse|null) => void
  ): UnaryResponse;
  putExecution(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionResponse|null) => void
  ): UnaryResponse;
  putExecution(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutExecutionResponse|null) => void
  ): UnaryResponse;
  putTypes(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutTypesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutTypesResponse|null) => void
  ): UnaryResponse;
  putTypes(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutTypesRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutTypesResponse|null) => void
  ): UnaryResponse;
  putContextType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutContextTypeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutContextTypeResponse|null) => void
  ): UnaryResponse;
  putContextType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutContextTypeRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutContextTypeResponse|null) => void
  ): UnaryResponse;
  putContexts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutContextsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutContextsResponse|null) => void
  ): UnaryResponse;
  putContexts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutContextsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutContextsResponse|null) => void
  ): UnaryResponse;
  putAttributionsAndAssociations(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutAttributionsAndAssociationsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutAttributionsAndAssociationsResponse|null) => void
  ): UnaryResponse;
  putAttributionsAndAssociations(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutAttributionsAndAssociationsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutAttributionsAndAssociationsResponse|null) => void
  ): UnaryResponse;
  putParentContexts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutParentContextsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutParentContextsResponse|null) => void
  ): UnaryResponse;
  putParentContexts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.PutParentContextsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.PutParentContextsResponse|null) => void
  ): UnaryResponse;
  getArtifactType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypeResponse|null) => void
  ): UnaryResponse;
  getArtifactType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypeRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypeResponse|null) => void
  ): UnaryResponse;
  getArtifactTypesByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesByIDRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesByIDResponse|null) => void
  ): UnaryResponse;
  getArtifactTypesByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesByIDRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesByIDResponse|null) => void
  ): UnaryResponse;
  getArtifactTypes(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesResponse|null) => void
  ): UnaryResponse;
  getArtifactTypes(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesResponse|null) => void
  ): UnaryResponse;
  getExecutionType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypeResponse|null) => void
  ): UnaryResponse;
  getExecutionType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypeRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypeResponse|null) => void
  ): UnaryResponse;
  getExecutionTypesByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesByIDRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesByIDResponse|null) => void
  ): UnaryResponse;
  getExecutionTypesByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesByIDRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesByIDResponse|null) => void
  ): UnaryResponse;
  getExecutionTypes(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesResponse|null) => void
  ): UnaryResponse;
  getExecutionTypes(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesResponse|null) => void
  ): UnaryResponse;
  getContextType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextTypeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextTypeResponse|null) => void
  ): UnaryResponse;
  getContextType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextTypeRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextTypeResponse|null) => void
  ): UnaryResponse;
  getContextTypesByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextTypesByIDRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextTypesByIDResponse|null) => void
  ): UnaryResponse;
  getContextTypesByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextTypesByIDRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextTypesByIDResponse|null) => void
  ): UnaryResponse;
  getArtifacts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsResponse|null) => void
  ): UnaryResponse;
  getArtifacts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsResponse|null) => void
  ): UnaryResponse;
  getExecutions(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsResponse|null) => void
  ): UnaryResponse;
  getExecutions(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsResponse|null) => void
  ): UnaryResponse;
  getContexts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsResponse|null) => void
  ): UnaryResponse;
  getContexts(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsResponse|null) => void
  ): UnaryResponse;
  getArtifactsByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByIDRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByIDResponse|null) => void
  ): UnaryResponse;
  getArtifactsByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByIDRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByIDResponse|null) => void
  ): UnaryResponse;
  getExecutionsByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByIDRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByIDResponse|null) => void
  ): UnaryResponse;
  getExecutionsByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByIDRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByIDResponse|null) => void
  ): UnaryResponse;
  getContextsByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByIDRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByIDResponse|null) => void
  ): UnaryResponse;
  getContextsByID(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByIDRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByIDResponse|null) => void
  ): UnaryResponse;
  getArtifactsByType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByTypeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByTypeResponse|null) => void
  ): UnaryResponse;
  getArtifactsByType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByTypeRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByTypeResponse|null) => void
  ): UnaryResponse;
  getExecutionsByType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByTypeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByTypeResponse|null) => void
  ): UnaryResponse;
  getExecutionsByType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByTypeRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByTypeResponse|null) => void
  ): UnaryResponse;
  getContextsByType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByTypeRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByTypeResponse|null) => void
  ): UnaryResponse;
  getContextsByType(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByTypeRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByTypeResponse|null) => void
  ): UnaryResponse;
  getArtifactsByURI(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByURIRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByURIResponse|null) => void
  ): UnaryResponse;
  getArtifactsByURI(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByURIRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByURIResponse|null) => void
  ): UnaryResponse;
  getEventsByExecutionIDs(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetEventsByExecutionIDsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetEventsByExecutionIDsResponse|null) => void
  ): UnaryResponse;
  getEventsByExecutionIDs(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetEventsByExecutionIDsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetEventsByExecutionIDsResponse|null) => void
  ): UnaryResponse;
  getEventsByArtifactIDs(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetEventsByArtifactIDsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetEventsByArtifactIDsResponse|null) => void
  ): UnaryResponse;
  getEventsByArtifactIDs(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetEventsByArtifactIDsRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetEventsByArtifactIDsResponse|null) => void
  ): UnaryResponse;
  getContextsByArtifact(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByArtifactRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByArtifactResponse|null) => void
  ): UnaryResponse;
  getContextsByArtifact(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByArtifactRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByArtifactResponse|null) => void
  ): UnaryResponse;
  getContextsByExecution(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByExecutionRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByExecutionResponse|null) => void
  ): UnaryResponse;
  getContextsByExecution(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByExecutionRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetContextsByExecutionResponse|null) => void
  ): UnaryResponse;
  getParentContextsByContext(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetParentContextsByContextRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetParentContextsByContextResponse|null) => void
  ): UnaryResponse;
  getParentContextsByContext(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetParentContextsByContextRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetParentContextsByContextResponse|null) => void
  ): UnaryResponse;
  getChildrenContextsByContext(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetChildrenContextsByContextRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetChildrenContextsByContextResponse|null) => void
  ): UnaryResponse;
  getChildrenContextsByContext(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetChildrenContextsByContextRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetChildrenContextsByContextResponse|null) => void
  ): UnaryResponse;
  getArtifactsByContext(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByContextRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByContextResponse|null) => void
  ): UnaryResponse;
  getArtifactsByContext(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByContextRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetArtifactsByContextResponse|null) => void
  ): UnaryResponse;
  getExecutionsByContext(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByContextRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByContextResponse|null) => void
  ): UnaryResponse;
  getExecutionsByContext(
    requestMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByContextRequest,
    callback: (error: ServiceError|null, responseMessage: src_apis_metadata_metadata_store_service_pb.GetExecutionsByContextResponse|null) => void
  ): UnaryResponse;
}

