import * as grpcWeb from 'grpc-web';

import * as ml_metadata_proto_metadata_store_service_pb from '../../ml_metadata/proto/metadata_store_service_pb';

export class MetadataStoreServiceClient {
  constructor(
    hostname: string,
    credentials?: null | { [index: string]: string },
    options?: null | { [index: string]: any },
  );

  putArtifactType(
    request: ml_metadata_proto_metadata_store_service_pb.PutArtifactTypeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutArtifactTypeResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutArtifactTypeResponse>;

  putExecutionType(
    request: ml_metadata_proto_metadata_store_service_pb.PutExecutionTypeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutExecutionTypeResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutExecutionTypeResponse>;

  putContextType(
    request: ml_metadata_proto_metadata_store_service_pb.PutContextTypeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutContextTypeResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutContextTypeResponse>;

  putTypes(
    request: ml_metadata_proto_metadata_store_service_pb.PutTypesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutTypesResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutTypesResponse>;

  putArtifacts(
    request: ml_metadata_proto_metadata_store_service_pb.PutArtifactsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutArtifactsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutArtifactsResponse>;

  putExecutions(
    request: ml_metadata_proto_metadata_store_service_pb.PutExecutionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutExecutionsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutExecutionsResponse>;

  putEvents(
    request: ml_metadata_proto_metadata_store_service_pb.PutEventsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutEventsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutEventsResponse>;

  putExecution(
    request: ml_metadata_proto_metadata_store_service_pb.PutExecutionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutExecutionResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutExecutionResponse>;

  putLineageSubgraph(
    request: ml_metadata_proto_metadata_store_service_pb.PutLineageSubgraphRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutLineageSubgraphResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutLineageSubgraphResponse>;

  putContexts(
    request: ml_metadata_proto_metadata_store_service_pb.PutContextsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutContextsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutContextsResponse>;

  putAttributionsAndAssociations(
    request: ml_metadata_proto_metadata_store_service_pb.PutAttributionsAndAssociationsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutAttributionsAndAssociationsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutAttributionsAndAssociationsResponse>;

  putParentContexts(
    request: ml_metadata_proto_metadata_store_service_pb.PutParentContextsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.PutParentContextsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.PutParentContextsResponse>;

  getArtifactType(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypeResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactTypeResponse>;

  getArtifactTypesByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByIDRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByIDResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByIDResponse>;

  getArtifactTypes(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesResponse>;

  getExecutionType(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypeResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionTypeResponse>;

  getExecutionTypesByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByIDRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByIDResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByIDResponse>;

  getExecutionTypes(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesResponse>;

  getContextType(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextTypeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextTypeResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextTypeResponse>;

  getContextTypesByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextTypesByIDRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextTypesByIDResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextTypesByIDResponse>;

  getContextTypes(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextTypesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextTypesResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextTypesResponse>;

  getArtifacts(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactsResponse>;

  getExecutions(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionsResponse>;

  getContexts(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextsResponse>;

  getArtifactsByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByIDRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByIDResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByIDResponse>;

  getExecutionsByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByIDRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByIDResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionsByIDResponse>;

  getContextsByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByIDRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextsByIDResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextsByIDResponse>;

  getArtifactsByType(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByTypeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByTypeResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByTypeResponse>;

  getExecutionsByType(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByTypeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByTypeResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionsByTypeResponse>;

  getContextsByType(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByTypeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextsByTypeResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextsByTypeResponse>;

  getArtifactByTypeAndName(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactByTypeAndNameRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactByTypeAndNameResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactByTypeAndNameResponse>;

  getExecutionByTypeAndName(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionByTypeAndNameRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionByTypeAndNameResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionByTypeAndNameResponse>;

  getContextByTypeAndName(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextByTypeAndNameRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextByTypeAndNameResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextByTypeAndNameResponse>;

  getArtifactsByURI(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByURIRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByURIResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByURIResponse>;

  getEventsByExecutionIDs(
    request: ml_metadata_proto_metadata_store_service_pb.GetEventsByExecutionIDsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetEventsByExecutionIDsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetEventsByExecutionIDsResponse>;

  getEventsByArtifactIDs(
    request: ml_metadata_proto_metadata_store_service_pb.GetEventsByArtifactIDsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetEventsByArtifactIDsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetEventsByArtifactIDsResponse>;

  getArtifactsByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByExternalIdsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByExternalIdsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByExternalIdsResponse>;

  getExecutionsByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByExternalIdsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByExternalIdsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionsByExternalIdsResponse>;

  getContextsByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByExternalIdsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextsByExternalIdsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextsByExternalIdsResponse>;

  getArtifactTypesByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByExternalIdsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByExternalIdsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByExternalIdsResponse>;

  getExecutionTypesByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByExternalIdsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByExternalIdsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByExternalIdsResponse>;

  getContextTypesByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextTypesByExternalIdsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextTypesByExternalIdsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextTypesByExternalIdsResponse>;

  getContextsByArtifact(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByArtifactRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextsByArtifactResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextsByArtifactResponse>;

  getContextsByExecution(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByExecutionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetContextsByExecutionResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetContextsByExecutionResponse>;

  getParentContextsByContext(
    request: ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextResponse>;

  getChildrenContextsByContext(
    request: ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextResponse>;

  getParentContextsByContexts(
    request: ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextsResponse>;

  getChildrenContextsByContexts(
    request: ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextsResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextsResponse>;

  getArtifactsByContext(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByContextRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByContextResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByContextResponse>;

  getExecutionsByContext(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByContextRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByContextResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetExecutionsByContextResponse>;

  getLineageGraph(
    request: ml_metadata_proto_metadata_store_service_pb.GetLineageGraphRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetLineageGraphResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetLineageGraphResponse>;

  getLineageSubgraph(
    request: ml_metadata_proto_metadata_store_service_pb.GetLineageSubgraphRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (
      err: grpcWeb.RpcError,
      response: ml_metadata_proto_metadata_store_service_pb.GetLineageSubgraphResponse,
    ) => void,
  ): grpcWeb.ClientReadableStream<ml_metadata_proto_metadata_store_service_pb.GetLineageSubgraphResponse>;
}

export class MetadataStoreServicePromiseClient {
  constructor(
    hostname: string,
    credentials?: null | { [index: string]: string },
    options?: null | { [index: string]: any },
  );

  putArtifactType(
    request: ml_metadata_proto_metadata_store_service_pb.PutArtifactTypeRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutArtifactTypeResponse>;

  putExecutionType(
    request: ml_metadata_proto_metadata_store_service_pb.PutExecutionTypeRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutExecutionTypeResponse>;

  putContextType(
    request: ml_metadata_proto_metadata_store_service_pb.PutContextTypeRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutContextTypeResponse>;

  putTypes(
    request: ml_metadata_proto_metadata_store_service_pb.PutTypesRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutTypesResponse>;

  putArtifacts(
    request: ml_metadata_proto_metadata_store_service_pb.PutArtifactsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutArtifactsResponse>;

  putExecutions(
    request: ml_metadata_proto_metadata_store_service_pb.PutExecutionsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutExecutionsResponse>;

  putEvents(
    request: ml_metadata_proto_metadata_store_service_pb.PutEventsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutEventsResponse>;

  putExecution(
    request: ml_metadata_proto_metadata_store_service_pb.PutExecutionRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutExecutionResponse>;

  putLineageSubgraph(
    request: ml_metadata_proto_metadata_store_service_pb.PutLineageSubgraphRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutLineageSubgraphResponse>;

  putContexts(
    request: ml_metadata_proto_metadata_store_service_pb.PutContextsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutContextsResponse>;

  putAttributionsAndAssociations(
    request: ml_metadata_proto_metadata_store_service_pb.PutAttributionsAndAssociationsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutAttributionsAndAssociationsResponse>;

  putParentContexts(
    request: ml_metadata_proto_metadata_store_service_pb.PutParentContextsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.PutParentContextsResponse>;

  getArtifactType(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypeRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactTypeResponse>;

  getArtifactTypesByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByIDRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByIDResponse>;

  getArtifactTypes(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesResponse>;

  getExecutionType(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypeRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionTypeResponse>;

  getExecutionTypesByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByIDRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByIDResponse>;

  getExecutionTypes(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesResponse>;

  getContextType(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextTypeRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextTypeResponse>;

  getContextTypesByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextTypesByIDRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextTypesByIDResponse>;

  getContextTypes(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextTypesRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextTypesResponse>;

  getArtifacts(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactsResponse>;

  getExecutions(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionsResponse>;

  getContexts(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextsResponse>;

  getArtifactsByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByIDRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByIDResponse>;

  getExecutionsByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByIDRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionsByIDResponse>;

  getContextsByID(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByIDRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextsByIDResponse>;

  getArtifactsByType(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByTypeRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByTypeResponse>;

  getExecutionsByType(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByTypeRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionsByTypeResponse>;

  getContextsByType(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByTypeRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextsByTypeResponse>;

  getArtifactByTypeAndName(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactByTypeAndNameRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactByTypeAndNameResponse>;

  getExecutionByTypeAndName(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionByTypeAndNameRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionByTypeAndNameResponse>;

  getContextByTypeAndName(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextByTypeAndNameRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextByTypeAndNameResponse>;

  getArtifactsByURI(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByURIRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByURIResponse>;

  getEventsByExecutionIDs(
    request: ml_metadata_proto_metadata_store_service_pb.GetEventsByExecutionIDsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetEventsByExecutionIDsResponse>;

  getEventsByArtifactIDs(
    request: ml_metadata_proto_metadata_store_service_pb.GetEventsByArtifactIDsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetEventsByArtifactIDsResponse>;

  getArtifactsByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByExternalIdsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByExternalIdsResponse>;

  getExecutionsByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByExternalIdsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionsByExternalIdsResponse>;

  getContextsByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByExternalIdsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextsByExternalIdsResponse>;

  getArtifactTypesByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByExternalIdsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactTypesByExternalIdsResponse>;

  getExecutionTypesByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByExternalIdsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionTypesByExternalIdsResponse>;

  getContextTypesByExternalIds(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextTypesByExternalIdsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextTypesByExternalIdsResponse>;

  getContextsByArtifact(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByArtifactRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextsByArtifactResponse>;

  getContextsByExecution(
    request: ml_metadata_proto_metadata_store_service_pb.GetContextsByExecutionRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetContextsByExecutionResponse>;

  getParentContextsByContext(
    request: ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextResponse>;

  getChildrenContextsByContext(
    request: ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextResponse>;

  getParentContextsByContexts(
    request: ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetParentContextsByContextsResponse>;

  getChildrenContextsByContexts(
    request: ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextsRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetChildrenContextsByContextsResponse>;

  getArtifactsByContext(
    request: ml_metadata_proto_metadata_store_service_pb.GetArtifactsByContextRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetArtifactsByContextResponse>;

  getExecutionsByContext(
    request: ml_metadata_proto_metadata_store_service_pb.GetExecutionsByContextRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetExecutionsByContextResponse>;

  getLineageGraph(
    request: ml_metadata_proto_metadata_store_service_pb.GetLineageGraphRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetLineageGraphResponse>;

  getLineageSubgraph(
    request: ml_metadata_proto_metadata_store_service_pb.GetLineageSubgraphRequest,
    metadata?: grpcWeb.Metadata,
  ): Promise<ml_metadata_proto_metadata_store_service_pb.GetLineageSubgraphResponse>;
}
