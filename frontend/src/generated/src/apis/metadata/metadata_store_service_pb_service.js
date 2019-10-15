// package: ml_metadata
// file: src/apis/metadata/metadata_store_service.proto

var src_apis_metadata_metadata_store_service_pb = require("./metadata_store_service_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var MetadataStoreService = (function () {
  function MetadataStoreService() {}
  MetadataStoreService.serviceName = "ml_metadata.MetadataStoreService";
  return MetadataStoreService;
}());

MetadataStoreService.PutArtifacts = {
  methodName: "PutArtifacts",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutArtifactsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutArtifactsResponse
};

MetadataStoreService.PutArtifactType = {
  methodName: "PutArtifactType",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutArtifactTypeRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutArtifactTypeResponse
};

MetadataStoreService.PutExecutions = {
  methodName: "PutExecutions",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutExecutionsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutExecutionsResponse
};

MetadataStoreService.PutExecutionType = {
  methodName: "PutExecutionType",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutExecutionTypeRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutExecutionTypeResponse
};

MetadataStoreService.PutEvents = {
  methodName: "PutEvents",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutEventsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutEventsResponse
};

MetadataStoreService.PutExecution = {
  methodName: "PutExecution",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutExecutionRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutExecutionResponse
};

MetadataStoreService.PutTypes = {
  methodName: "PutTypes",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutTypesRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutTypesResponse
};

MetadataStoreService.PutContextType = {
  methodName: "PutContextType",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutContextTypeRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutContextTypeResponse
};

MetadataStoreService.PutContexts = {
  methodName: "PutContexts",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutContextsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutContextsResponse
};

MetadataStoreService.PutAttributionsAndAssociations = {
  methodName: "PutAttributionsAndAssociations",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutAttributionsAndAssociationsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutAttributionsAndAssociationsResponse
};

MetadataStoreService.PutParentContexts = {
  methodName: "PutParentContexts",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.PutParentContextsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.PutParentContextsResponse
};

MetadataStoreService.GetArtifactType = {
  methodName: "GetArtifactType",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetArtifactTypeRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetArtifactTypeResponse
};

MetadataStoreService.GetArtifactTypesByID = {
  methodName: "GetArtifactTypesByID",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesByIDRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesByIDResponse
};

MetadataStoreService.GetArtifactTypes = {
  methodName: "GetArtifactTypes",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetArtifactTypesResponse
};

MetadataStoreService.GetExecutionType = {
  methodName: "GetExecutionType",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetExecutionTypeRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetExecutionTypeResponse
};

MetadataStoreService.GetExecutionTypesByID = {
  methodName: "GetExecutionTypesByID",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesByIDRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesByIDResponse
};

MetadataStoreService.GetExecutionTypes = {
  methodName: "GetExecutionTypes",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetExecutionTypesResponse
};

MetadataStoreService.GetContextType = {
  methodName: "GetContextType",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetContextTypeRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetContextTypeResponse
};

MetadataStoreService.GetContextTypesByID = {
  methodName: "GetContextTypesByID",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetContextTypesByIDRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetContextTypesByIDResponse
};

MetadataStoreService.GetArtifacts = {
  methodName: "GetArtifacts",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetArtifactsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetArtifactsResponse
};

MetadataStoreService.GetExecutions = {
  methodName: "GetExecutions",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetExecutionsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetExecutionsResponse
};

MetadataStoreService.GetContexts = {
  methodName: "GetContexts",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetContextsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetContextsResponse
};

MetadataStoreService.GetArtifactsByID = {
  methodName: "GetArtifactsByID",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetArtifactsByIDRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetArtifactsByIDResponse
};

MetadataStoreService.GetExecutionsByID = {
  methodName: "GetExecutionsByID",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetExecutionsByIDRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetExecutionsByIDResponse
};

MetadataStoreService.GetContextsByID = {
  methodName: "GetContextsByID",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetContextsByIDRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetContextsByIDResponse
};

MetadataStoreService.GetArtifactsByType = {
  methodName: "GetArtifactsByType",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetArtifactsByTypeRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetArtifactsByTypeResponse
};

MetadataStoreService.GetExecutionsByType = {
  methodName: "GetExecutionsByType",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetExecutionsByTypeRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetExecutionsByTypeResponse
};

MetadataStoreService.GetContextsByType = {
  methodName: "GetContextsByType",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetContextsByTypeRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetContextsByTypeResponse
};

MetadataStoreService.GetArtifactsByURI = {
  methodName: "GetArtifactsByURI",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetArtifactsByURIRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetArtifactsByURIResponse
};

MetadataStoreService.GetEventsByExecutionIDs = {
  methodName: "GetEventsByExecutionIDs",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetEventsByExecutionIDsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetEventsByExecutionIDsResponse
};

MetadataStoreService.GetEventsByArtifactIDs = {
  methodName: "GetEventsByArtifactIDs",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetEventsByArtifactIDsRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetEventsByArtifactIDsResponse
};

MetadataStoreService.GetContextsByArtifact = {
  methodName: "GetContextsByArtifact",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetContextsByArtifactRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetContextsByArtifactResponse
};

MetadataStoreService.GetContextsByExecution = {
  methodName: "GetContextsByExecution",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetContextsByExecutionRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetContextsByExecutionResponse
};

MetadataStoreService.GetParentContextsByContext = {
  methodName: "GetParentContextsByContext",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetParentContextsByContextRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetParentContextsByContextResponse
};

MetadataStoreService.GetChildrenContextsByContext = {
  methodName: "GetChildrenContextsByContext",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetChildrenContextsByContextRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetChildrenContextsByContextResponse
};

MetadataStoreService.GetArtifactsByContext = {
  methodName: "GetArtifactsByContext",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetArtifactsByContextRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetArtifactsByContextResponse
};

MetadataStoreService.GetExecutionsByContext = {
  methodName: "GetExecutionsByContext",
  service: MetadataStoreService,
  requestStream: false,
  responseStream: false,
  requestType: src_apis_metadata_metadata_store_service_pb.GetExecutionsByContextRequest,
  responseType: src_apis_metadata_metadata_store_service_pb.GetExecutionsByContextResponse
};

exports.MetadataStoreService = MetadataStoreService;

function MetadataStoreServiceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

MetadataStoreServiceClient.prototype.putArtifacts = function putArtifacts(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutArtifacts, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putArtifactType = function putArtifactType(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutArtifactType, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putExecutions = function putExecutions(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutExecutions, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putExecutionType = function putExecutionType(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutExecutionType, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putEvents = function putEvents(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutEvents, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putExecution = function putExecution(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutExecution, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putTypes = function putTypes(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutTypes, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putContextType = function putContextType(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutContextType, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putContexts = function putContexts(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutContexts, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putAttributionsAndAssociations = function putAttributionsAndAssociations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutAttributionsAndAssociations, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.putParentContexts = function putParentContexts(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.PutParentContexts, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getArtifactType = function getArtifactType(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetArtifactType, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getArtifactTypesByID = function getArtifactTypesByID(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetArtifactTypesByID, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getArtifactTypes = function getArtifactTypes(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetArtifactTypes, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getExecutionType = function getExecutionType(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetExecutionType, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getExecutionTypesByID = function getExecutionTypesByID(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetExecutionTypesByID, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getExecutionTypes = function getExecutionTypes(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetExecutionTypes, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getContextType = function getContextType(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetContextType, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getContextTypesByID = function getContextTypesByID(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetContextTypesByID, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getArtifacts = function getArtifacts(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetArtifacts, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getExecutions = function getExecutions(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetExecutions, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getContexts = function getContexts(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetContexts, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getArtifactsByID = function getArtifactsByID(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetArtifactsByID, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getExecutionsByID = function getExecutionsByID(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetExecutionsByID, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getContextsByID = function getContextsByID(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetContextsByID, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getArtifactsByType = function getArtifactsByType(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetArtifactsByType, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getExecutionsByType = function getExecutionsByType(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetExecutionsByType, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getContextsByType = function getContextsByType(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetContextsByType, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getArtifactsByURI = function getArtifactsByURI(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetArtifactsByURI, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getEventsByExecutionIDs = function getEventsByExecutionIDs(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetEventsByExecutionIDs, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getEventsByArtifactIDs = function getEventsByArtifactIDs(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetEventsByArtifactIDs, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getContextsByArtifact = function getContextsByArtifact(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetContextsByArtifact, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getContextsByExecution = function getContextsByExecution(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetContextsByExecution, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getParentContextsByContext = function getParentContextsByContext(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetParentContextsByContext, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getChildrenContextsByContext = function getChildrenContextsByContext(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetChildrenContextsByContext, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getArtifactsByContext = function getArtifactsByContext(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetArtifactsByContext, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

MetadataStoreServiceClient.prototype.getExecutionsByContext = function getExecutionsByContext(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(MetadataStoreService.GetExecutionsByContext, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

exports.MetadataStoreServiceClient = MetadataStoreServiceClient;

