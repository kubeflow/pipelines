// source: ml_metadata/proto/metadata_store_service.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global = Function('return this')();

var ml_metadata_proto_metadata_store_pb = require('../../ml_metadata/proto/metadata_store_pb.js');
goog.object.extend(proto, ml_metadata_proto_metadata_store_pb);
goog.exportSymbol('proto.ml_metadata.ArtifactAndType', null, global);
goog.exportSymbol('proto.ml_metadata.ArtifactStruct', null, global);
goog.exportSymbol('proto.ml_metadata.ArtifactStruct.ValueCase', null, global);
goog.exportSymbol('proto.ml_metadata.ArtifactStructList', null, global);
goog.exportSymbol('proto.ml_metadata.ArtifactStructMap', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactByTypeAndNameRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactByTypeAndNameResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactTypeRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactTypeResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactTypesByIDRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactTypesByIDResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactTypesRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactTypesResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsByContextRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsByContextResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsByIDRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsByIDResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsByTypeRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsByTypeResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsByURIRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsByURIResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetArtifactsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetChildrenContextsByContextRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetChildrenContextsByContextResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextByTypeAndNameRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextByTypeAndNameResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextTypeRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextTypeResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextTypesByIDRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextTypesByIDResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextTypesRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextTypesResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsByArtifactRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsByArtifactResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsByExecutionRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsByExecutionResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsByIDRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsByIDResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsByTypeRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsByTypeResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetContextsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetEventsByArtifactIDsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetEventsByArtifactIDsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetEventsByExecutionIDsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetEventsByExecutionIDsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionByTypeAndNameRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionByTypeAndNameResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionTypeRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionTypeResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionTypesByIDRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionTypesByIDResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionTypesRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionTypesResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionsByContextRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionsByContextResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionsByIDRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionsByIDResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionsByTypeRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionsByTypeResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetExecutionsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetLineageGraphRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetLineageGraphResponse', null, global);
goog.exportSymbol('proto.ml_metadata.GetParentContextsByContextRequest', null, global);
goog.exportSymbol('proto.ml_metadata.GetParentContextsByContextResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutArtifactTypeRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutArtifactTypeResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutArtifactsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutArtifactsRequest.Options', null, global);
goog.exportSymbol('proto.ml_metadata.PutArtifactsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutAttributionsAndAssociationsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutAttributionsAndAssociationsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutContextTypeRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutContextTypeResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutContextsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutContextsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutEventsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutEventsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutExecutionRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent', null, global);
goog.exportSymbol('proto.ml_metadata.PutExecutionRequest.Options', null, global);
goog.exportSymbol('proto.ml_metadata.PutExecutionResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutExecutionTypeRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutExecutionTypeResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutExecutionsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutExecutionsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutParentContextsRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutParentContextsResponse', null, global);
goog.exportSymbol('proto.ml_metadata.PutTypesRequest', null, global);
goog.exportSymbol('proto.ml_metadata.PutTypesResponse', null, global);
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.ArtifactAndType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.ArtifactAndType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ArtifactAndType.displayName = 'proto.ml_metadata.ArtifactAndType';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.ArtifactStructMap = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.ArtifactStructMap, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ArtifactStructMap.displayName = 'proto.ml_metadata.ArtifactStructMap';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.ArtifactStructList = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.ArtifactStructList.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.ArtifactStructList, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ArtifactStructList.displayName = 'proto.ml_metadata.ArtifactStructList';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.ArtifactStruct = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_metadata.ArtifactStruct.oneofGroups_);
};
goog.inherits(proto.ml_metadata.ArtifactStruct, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ArtifactStruct.displayName = 'proto.ml_metadata.ArtifactStruct';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutArtifactsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutArtifactsRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutArtifactsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutArtifactsRequest.displayName = 'proto.ml_metadata.PutArtifactsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutArtifactsRequest.Options = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutArtifactsRequest.Options, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutArtifactsRequest.Options.displayName = 'proto.ml_metadata.PutArtifactsRequest.Options';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutArtifactsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutArtifactsResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutArtifactsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutArtifactsResponse.displayName = 'proto.ml_metadata.PutArtifactsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutArtifactTypeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutArtifactTypeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutArtifactTypeRequest.displayName = 'proto.ml_metadata.PutArtifactTypeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutArtifactTypeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutArtifactTypeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutArtifactTypeResponse.displayName = 'proto.ml_metadata.PutArtifactTypeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutExecutionsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutExecutionsRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutExecutionsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutExecutionsRequest.displayName = 'proto.ml_metadata.PutExecutionsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutExecutionsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutExecutionsResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutExecutionsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutExecutionsResponse.displayName = 'proto.ml_metadata.PutExecutionsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutExecutionTypeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutExecutionTypeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutExecutionTypeRequest.displayName = 'proto.ml_metadata.PutExecutionTypeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutExecutionTypeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutExecutionTypeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutExecutionTypeResponse.displayName = 'proto.ml_metadata.PutExecutionTypeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutEventsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutEventsRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutEventsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutEventsRequest.displayName = 'proto.ml_metadata.PutEventsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutEventsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutEventsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutEventsResponse.displayName = 'proto.ml_metadata.PutEventsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutExecutionRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutExecutionRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutExecutionRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutExecutionRequest.displayName = 'proto.ml_metadata.PutExecutionRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.displayName = 'proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutExecutionRequest.Options = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutExecutionRequest.Options, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutExecutionRequest.Options.displayName = 'proto.ml_metadata.PutExecutionRequest.Options';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutExecutionResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutExecutionResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutExecutionResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutExecutionResponse.displayName = 'proto.ml_metadata.PutExecutionResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutTypesRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutTypesRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutTypesRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutTypesRequest.displayName = 'proto.ml_metadata.PutTypesRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutTypesResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutTypesResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutTypesResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutTypesResponse.displayName = 'proto.ml_metadata.PutTypesResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutContextTypeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutContextTypeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutContextTypeRequest.displayName = 'proto.ml_metadata.PutContextTypeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutContextTypeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutContextTypeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutContextTypeResponse.displayName = 'proto.ml_metadata.PutContextTypeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutContextsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutContextsRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutContextsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutContextsRequest.displayName = 'proto.ml_metadata.PutContextsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutContextsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutContextsResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutContextsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutContextsResponse.displayName = 'proto.ml_metadata.PutContextsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutAttributionsAndAssociationsRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutAttributionsAndAssociationsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutAttributionsAndAssociationsRequest.displayName = 'proto.ml_metadata.PutAttributionsAndAssociationsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutAttributionsAndAssociationsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutAttributionsAndAssociationsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutAttributionsAndAssociationsResponse.displayName = 'proto.ml_metadata.PutAttributionsAndAssociationsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutParentContextsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.PutParentContextsRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.PutParentContextsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutParentContextsRequest.displayName = 'proto.ml_metadata.PutParentContextsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.PutParentContextsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.PutParentContextsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.PutParentContextsResponse.displayName = 'proto.ml_metadata.PutParentContextsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsByTypeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsByTypeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsByTypeRequest.displayName = 'proto.ml_metadata.GetArtifactsByTypeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsByTypeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactsByTypeResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsByTypeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsByTypeResponse.displayName = 'proto.ml_metadata.GetArtifactsByTypeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetArtifactByTypeAndNameRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactByTypeAndNameRequest.displayName = 'proto.ml_metadata.GetArtifactByTypeAndNameRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetArtifactByTypeAndNameResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactByTypeAndNameResponse.displayName = 'proto.ml_metadata.GetArtifactByTypeAndNameResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsByIDRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactsByIDRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsByIDRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsByIDRequest.displayName = 'proto.ml_metadata.GetArtifactsByIDRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsByIDResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactsByIDResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsByIDResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsByIDResponse.displayName = 'proto.ml_metadata.GetArtifactsByIDResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsRequest.displayName = 'proto.ml_metadata.GetArtifactsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactsResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsResponse.displayName = 'proto.ml_metadata.GetArtifactsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsByURIRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactsByURIRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsByURIRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsByURIRequest.displayName = 'proto.ml_metadata.GetArtifactsByURIRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsByURIResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactsByURIResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsByURIResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsByURIResponse.displayName = 'proto.ml_metadata.GetArtifactsByURIResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetExecutionsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionsRequest.displayName = 'proto.ml_metadata.GetExecutionsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetExecutionsResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetExecutionsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionsResponse.displayName = 'proto.ml_metadata.GetExecutionsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactTypeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetArtifactTypeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactTypeRequest.displayName = 'proto.ml_metadata.GetArtifactTypeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactTypeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetArtifactTypeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactTypeResponse.displayName = 'proto.ml_metadata.GetArtifactTypeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactTypesRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetArtifactTypesRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactTypesRequest.displayName = 'proto.ml_metadata.GetArtifactTypesRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactTypesResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactTypesResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactTypesResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactTypesResponse.displayName = 'proto.ml_metadata.GetArtifactTypesResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionTypesRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetExecutionTypesRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionTypesRequest.displayName = 'proto.ml_metadata.GetExecutionTypesRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionTypesResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetExecutionTypesResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetExecutionTypesResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionTypesResponse.displayName = 'proto.ml_metadata.GetExecutionTypesResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextTypesRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetContextTypesRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextTypesRequest.displayName = 'proto.ml_metadata.GetContextTypesRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextTypesResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetContextTypesResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetContextTypesResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextTypesResponse.displayName = 'proto.ml_metadata.GetContextTypesResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionsByTypeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetExecutionsByTypeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionsByTypeRequest.displayName = 'proto.ml_metadata.GetExecutionsByTypeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionsByTypeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetExecutionsByTypeResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetExecutionsByTypeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionsByTypeResponse.displayName = 'proto.ml_metadata.GetExecutionsByTypeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetExecutionByTypeAndNameRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionByTypeAndNameRequest.displayName = 'proto.ml_metadata.GetExecutionByTypeAndNameRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetExecutionByTypeAndNameResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionByTypeAndNameResponse.displayName = 'proto.ml_metadata.GetExecutionByTypeAndNameResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionsByIDRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetExecutionsByIDRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetExecutionsByIDRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionsByIDRequest.displayName = 'proto.ml_metadata.GetExecutionsByIDRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionsByIDResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetExecutionsByIDResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetExecutionsByIDResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionsByIDResponse.displayName = 'proto.ml_metadata.GetExecutionsByIDResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionTypeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetExecutionTypeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionTypeRequest.displayName = 'proto.ml_metadata.GetExecutionTypeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionTypeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetExecutionTypeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionTypeResponse.displayName = 'proto.ml_metadata.GetExecutionTypeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetEventsByExecutionIDsRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetEventsByExecutionIDsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetEventsByExecutionIDsRequest.displayName = 'proto.ml_metadata.GetEventsByExecutionIDsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetEventsByExecutionIDsResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetEventsByExecutionIDsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetEventsByExecutionIDsResponse.displayName = 'proto.ml_metadata.GetEventsByExecutionIDsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetEventsByArtifactIDsRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetEventsByArtifactIDsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetEventsByArtifactIDsRequest.displayName = 'proto.ml_metadata.GetEventsByArtifactIDsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetEventsByArtifactIDsResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetEventsByArtifactIDsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetEventsByArtifactIDsResponse.displayName = 'proto.ml_metadata.GetEventsByArtifactIDsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactTypesByIDRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactTypesByIDRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactTypesByIDRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactTypesByIDRequest.displayName = 'proto.ml_metadata.GetArtifactTypesByIDRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactTypesByIDResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactTypesByIDResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactTypesByIDResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactTypesByIDResponse.displayName = 'proto.ml_metadata.GetArtifactTypesByIDResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionTypesByIDRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetExecutionTypesByIDRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetExecutionTypesByIDRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionTypesByIDRequest.displayName = 'proto.ml_metadata.GetExecutionTypesByIDRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionTypesByIDResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetExecutionTypesByIDResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetExecutionTypesByIDResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionTypesByIDResponse.displayName = 'proto.ml_metadata.GetExecutionTypesByIDResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextTypeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetContextTypeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextTypeRequest.displayName = 'proto.ml_metadata.GetContextTypeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextTypeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetContextTypeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextTypeResponse.displayName = 'proto.ml_metadata.GetContextTypeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextTypesByIDRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetContextTypesByIDRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetContextTypesByIDRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextTypesByIDRequest.displayName = 'proto.ml_metadata.GetContextTypesByIDRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextTypesByIDResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetContextTypesByIDResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetContextTypesByIDResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextTypesByIDResponse.displayName = 'proto.ml_metadata.GetContextTypesByIDResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetContextsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsRequest.displayName = 'proto.ml_metadata.GetContextsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetContextsResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetContextsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsResponse.displayName = 'proto.ml_metadata.GetContextsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsByTypeRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetContextsByTypeRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsByTypeRequest.displayName = 'proto.ml_metadata.GetContextsByTypeRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsByTypeResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetContextsByTypeResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetContextsByTypeResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsByTypeResponse.displayName = 'proto.ml_metadata.GetContextsByTypeResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextByTypeAndNameRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetContextByTypeAndNameRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextByTypeAndNameRequest.displayName = 'proto.ml_metadata.GetContextByTypeAndNameRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextByTypeAndNameResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetContextByTypeAndNameResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextByTypeAndNameResponse.displayName = 'proto.ml_metadata.GetContextByTypeAndNameResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsByIDRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetContextsByIDRequest.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetContextsByIDRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsByIDRequest.displayName = 'proto.ml_metadata.GetContextsByIDRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsByIDResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetContextsByIDResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetContextsByIDResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsByIDResponse.displayName = 'proto.ml_metadata.GetContextsByIDResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsByArtifactRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetContextsByArtifactRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsByArtifactRequest.displayName = 'proto.ml_metadata.GetContextsByArtifactRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsByArtifactResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetContextsByArtifactResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetContextsByArtifactResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsByArtifactResponse.displayName = 'proto.ml_metadata.GetContextsByArtifactResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsByExecutionRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetContextsByExecutionRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsByExecutionRequest.displayName = 'proto.ml_metadata.GetContextsByExecutionRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetContextsByExecutionResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetContextsByExecutionResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetContextsByExecutionResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetContextsByExecutionResponse.displayName = 'proto.ml_metadata.GetContextsByExecutionResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetParentContextsByContextRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetParentContextsByContextRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetParentContextsByContextRequest.displayName = 'proto.ml_metadata.GetParentContextsByContextRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetParentContextsByContextResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetParentContextsByContextResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetParentContextsByContextResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetParentContextsByContextResponse.displayName = 'proto.ml_metadata.GetParentContextsByContextResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetChildrenContextsByContextRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetChildrenContextsByContextRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetChildrenContextsByContextRequest.displayName = 'proto.ml_metadata.GetChildrenContextsByContextRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetChildrenContextsByContextResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetChildrenContextsByContextResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetChildrenContextsByContextResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetChildrenContextsByContextResponse.displayName = 'proto.ml_metadata.GetChildrenContextsByContextResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsByContextRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsByContextRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsByContextRequest.displayName = 'proto.ml_metadata.GetArtifactsByContextRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetArtifactsByContextResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetArtifactsByContextResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetArtifactsByContextResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetArtifactsByContextResponse.displayName = 'proto.ml_metadata.GetArtifactsByContextResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionsByContextRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetExecutionsByContextRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionsByContextRequest.displayName = 'proto.ml_metadata.GetExecutionsByContextRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetExecutionsByContextResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.GetExecutionsByContextResponse.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.GetExecutionsByContextResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetExecutionsByContextResponse.displayName = 'proto.ml_metadata.GetExecutionsByContextResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetLineageGraphRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetLineageGraphRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetLineageGraphRequest.displayName = 'proto.ml_metadata.GetLineageGraphRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.ml_metadata.GetLineageGraphResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GetLineageGraphResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GetLineageGraphResponse.displayName = 'proto.ml_metadata.GetLineageGraphResponse';
}



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.ArtifactAndType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ArtifactAndType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ArtifactAndType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactAndType.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifact: (f = msg.getArtifact()) && ml_metadata_proto_metadata_store_pb.Artifact.toObject(includeInstance, f),
    type: (f = msg.getType()) && ml_metadata_proto_metadata_store_pb.ArtifactType.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.ArtifactAndType}
 */
proto.ml_metadata.ArtifactAndType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ArtifactAndType;
  return proto.ml_metadata.ArtifactAndType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ArtifactAndType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ArtifactAndType}
 */
proto.ml_metadata.ArtifactAndType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Artifact;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Artifact.deserializeBinaryFromReader);
      msg.setArtifact(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.ArtifactType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ArtifactType.deserializeBinaryFromReader);
      msg.setType(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.ArtifactAndType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ArtifactAndType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ArtifactAndType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactAndType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifact();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Artifact.serializeBinaryToWriter
    );
  }
  f = message.getType();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.ArtifactType.serializeBinaryToWriter
    );
  }
};


/**
 * optional Artifact artifact = 1;
 * @return {?proto.ml_metadata.Artifact}
 */
proto.ml_metadata.ArtifactAndType.prototype.getArtifact = function() {
  return /** @type{?proto.ml_metadata.Artifact} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.Artifact, 1));
};


/**
 * @param {?proto.ml_metadata.Artifact|undefined} value
 * @return {!proto.ml_metadata.ArtifactAndType} returns this
*/
proto.ml_metadata.ArtifactAndType.prototype.setArtifact = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactAndType} returns this
 */
proto.ml_metadata.ArtifactAndType.prototype.clearArtifact = function() {
  return this.setArtifact(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactAndType.prototype.hasArtifact = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ArtifactType type = 2;
 * @return {?proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.ArtifactAndType.prototype.getType = function() {
  return /** @type{?proto.ml_metadata.ArtifactType} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ArtifactType, 2));
};


/**
 * @param {?proto.ml_metadata.ArtifactType|undefined} value
 * @return {!proto.ml_metadata.ArtifactAndType} returns this
*/
proto.ml_metadata.ArtifactAndType.prototype.setType = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactAndType} returns this
 */
proto.ml_metadata.ArtifactAndType.prototype.clearType = function() {
  return this.setType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactAndType.prototype.hasType = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.ArtifactStructMap.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ArtifactStructMap.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ArtifactStructMap} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactStructMap.toObject = function(includeInstance, msg) {
  var f, obj = {
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, proto.ml_metadata.ArtifactStruct.toObject) : []
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.ArtifactStructMap}
 */
proto.ml_metadata.ArtifactStructMap.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ArtifactStructMap;
  return proto.ml_metadata.ArtifactStructMap.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ArtifactStructMap} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ArtifactStructMap}
 */
proto.ml_metadata.ArtifactStructMap.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_metadata.ArtifactStruct.deserializeBinaryFromReader, "", new proto.ml_metadata.ArtifactStruct());
         });
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.ArtifactStructMap.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ArtifactStructMap.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ArtifactStructMap} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactStructMap.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_metadata.ArtifactStruct.serializeBinaryToWriter);
  }
};


/**
 * map<string, ArtifactStruct> properties = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.ArtifactStruct>}
 */
proto.ml_metadata.ArtifactStructMap.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.ArtifactStruct>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_metadata.ArtifactStruct));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.ArtifactStructMap} returns this
 */
proto.ml_metadata.ArtifactStructMap.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.ArtifactStructList.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.ArtifactStructList.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ArtifactStructList.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ArtifactStructList} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactStructList.toObject = function(includeInstance, msg) {
  var f, obj = {
    elementsList: jspb.Message.toObjectList(msg.getElementsList(),
    proto.ml_metadata.ArtifactStruct.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.ArtifactStructList}
 */
proto.ml_metadata.ArtifactStructList.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ArtifactStructList;
  return proto.ml_metadata.ArtifactStructList.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ArtifactStructList} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ArtifactStructList}
 */
proto.ml_metadata.ArtifactStructList.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ArtifactStruct;
      reader.readMessage(value,proto.ml_metadata.ArtifactStruct.deserializeBinaryFromReader);
      msg.addElements(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.ArtifactStructList.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ArtifactStructList.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ArtifactStructList} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactStructList.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getElementsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.ml_metadata.ArtifactStruct.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ArtifactStruct elements = 1;
 * @return {!Array<!proto.ml_metadata.ArtifactStruct>}
 */
proto.ml_metadata.ArtifactStructList.prototype.getElementsList = function() {
  return /** @type{!Array<!proto.ml_metadata.ArtifactStruct>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.ArtifactStruct, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ArtifactStruct>} value
 * @return {!proto.ml_metadata.ArtifactStructList} returns this
*/
proto.ml_metadata.ArtifactStructList.prototype.setElementsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ArtifactStruct=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ArtifactStruct}
 */
proto.ml_metadata.ArtifactStructList.prototype.addElements = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ArtifactStruct, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.ArtifactStructList} returns this
 */
proto.ml_metadata.ArtifactStructList.prototype.clearElementsList = function() {
  return this.setElementsList([]);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_metadata.ArtifactStruct.oneofGroups_ = [[1,2,3]];

/**
 * @enum {number}
 */
proto.ml_metadata.ArtifactStruct.ValueCase = {
  VALUE_NOT_SET: 0,
  ARTIFACT: 1,
  MAP: 2,
  LIST: 3
};

/**
 * @return {proto.ml_metadata.ArtifactStruct.ValueCase}
 */
proto.ml_metadata.ArtifactStruct.prototype.getValueCase = function() {
  return /** @type {proto.ml_metadata.ArtifactStruct.ValueCase} */(jspb.Message.computeOneofCase(this, proto.ml_metadata.ArtifactStruct.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.ArtifactStruct.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ArtifactStruct.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ArtifactStruct} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactStruct.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifact: (f = msg.getArtifact()) && proto.ml_metadata.ArtifactAndType.toObject(includeInstance, f),
    map: (f = msg.getMap()) && proto.ml_metadata.ArtifactStructMap.toObject(includeInstance, f),
    list: (f = msg.getList()) && proto.ml_metadata.ArtifactStructList.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.ArtifactStruct}
 */
proto.ml_metadata.ArtifactStruct.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ArtifactStruct;
  return proto.ml_metadata.ArtifactStruct.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ArtifactStruct} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ArtifactStruct}
 */
proto.ml_metadata.ArtifactStruct.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ArtifactAndType;
      reader.readMessage(value,proto.ml_metadata.ArtifactAndType.deserializeBinaryFromReader);
      msg.setArtifact(value);
      break;
    case 2:
      var value = new proto.ml_metadata.ArtifactStructMap;
      reader.readMessage(value,proto.ml_metadata.ArtifactStructMap.deserializeBinaryFromReader);
      msg.setMap(value);
      break;
    case 3:
      var value = new proto.ml_metadata.ArtifactStructList;
      reader.readMessage(value,proto.ml_metadata.ArtifactStructList.deserializeBinaryFromReader);
      msg.setList(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.ArtifactStruct.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ArtifactStruct.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ArtifactStruct} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactStruct.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifact();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_metadata.ArtifactAndType.serializeBinaryToWriter
    );
  }
  f = message.getMap();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_metadata.ArtifactStructMap.serializeBinaryToWriter
    );
  }
  f = message.getList();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_metadata.ArtifactStructList.serializeBinaryToWriter
    );
  }
};


/**
 * optional ArtifactAndType artifact = 1;
 * @return {?proto.ml_metadata.ArtifactAndType}
 */
proto.ml_metadata.ArtifactStruct.prototype.getArtifact = function() {
  return /** @type{?proto.ml_metadata.ArtifactAndType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ArtifactAndType, 1));
};


/**
 * @param {?proto.ml_metadata.ArtifactAndType|undefined} value
 * @return {!proto.ml_metadata.ArtifactStruct} returns this
*/
proto.ml_metadata.ArtifactStruct.prototype.setArtifact = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.ml_metadata.ArtifactStruct.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStruct} returns this
 */
proto.ml_metadata.ArtifactStruct.prototype.clearArtifact = function() {
  return this.setArtifact(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStruct.prototype.hasArtifact = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ArtifactStructMap map = 2;
 * @return {?proto.ml_metadata.ArtifactStructMap}
 */
proto.ml_metadata.ArtifactStruct.prototype.getMap = function() {
  return /** @type{?proto.ml_metadata.ArtifactStructMap} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ArtifactStructMap, 2));
};


/**
 * @param {?proto.ml_metadata.ArtifactStructMap|undefined} value
 * @return {!proto.ml_metadata.ArtifactStruct} returns this
*/
proto.ml_metadata.ArtifactStruct.prototype.setMap = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.ml_metadata.ArtifactStruct.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStruct} returns this
 */
proto.ml_metadata.ArtifactStruct.prototype.clearMap = function() {
  return this.setMap(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStruct.prototype.hasMap = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional ArtifactStructList list = 3;
 * @return {?proto.ml_metadata.ArtifactStructList}
 */
proto.ml_metadata.ArtifactStruct.prototype.getList = function() {
  return /** @type{?proto.ml_metadata.ArtifactStructList} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ArtifactStructList, 3));
};


/**
 * @param {?proto.ml_metadata.ArtifactStructList|undefined} value
 * @return {!proto.ml_metadata.ArtifactStruct} returns this
*/
proto.ml_metadata.ArtifactStruct.prototype.setList = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.ml_metadata.ArtifactStruct.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStruct} returns this
 */
proto.ml_metadata.ArtifactStruct.prototype.clearList = function() {
  return this.setList(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStruct.prototype.hasList = function() {
  return jspb.Message.getField(this, 3) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutArtifactsRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutArtifactsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutArtifactsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutArtifactsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsList: jspb.Message.toObjectList(msg.getArtifactsList(),
    ml_metadata_proto_metadata_store_pb.Artifact.toObject, includeInstance),
    options: (f = msg.getOptions()) && proto.ml_metadata.PutArtifactsRequest.Options.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutArtifactsRequest}
 */
proto.ml_metadata.PutArtifactsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutArtifactsRequest;
  return proto.ml_metadata.PutArtifactsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutArtifactsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutArtifactsRequest}
 */
proto.ml_metadata.PutArtifactsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Artifact;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Artifact.deserializeBinaryFromReader);
      msg.addArtifacts(value);
      break;
    case 2:
      var value = new proto.ml_metadata.PutArtifactsRequest.Options;
      reader.readMessage(value,proto.ml_metadata.PutArtifactsRequest.Options.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutArtifactsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutArtifactsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutArtifactsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Artifact.serializeBinaryToWriter
    );
  }
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_metadata.PutArtifactsRequest.Options.serializeBinaryToWriter
    );
  }
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutArtifactsRequest.Options.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutArtifactsRequest.Options.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutArtifactsRequest.Options} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactsRequest.Options.toObject = function(includeInstance, msg) {
  var f, obj = {
    abortIfLatestUpdatedTimeChanged: (f = jspb.Message.getBooleanField(msg, 1)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutArtifactsRequest.Options}
 */
proto.ml_metadata.PutArtifactsRequest.Options.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutArtifactsRequest.Options;
  return proto.ml_metadata.PutArtifactsRequest.Options.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutArtifactsRequest.Options} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutArtifactsRequest.Options}
 */
proto.ml_metadata.PutArtifactsRequest.Options.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setAbortIfLatestUpdatedTimeChanged(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutArtifactsRequest.Options.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutArtifactsRequest.Options.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutArtifactsRequest.Options} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactsRequest.Options.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {boolean} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool abort_if_latest_updated_time_changed = 1;
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactsRequest.Options.prototype.getAbortIfLatestUpdatedTimeChanged = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutArtifactsRequest.Options} returns this
 */
proto.ml_metadata.PutArtifactsRequest.Options.prototype.setAbortIfLatestUpdatedTimeChanged = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutArtifactsRequest.Options} returns this
 */
proto.ml_metadata.PutArtifactsRequest.Options.prototype.clearAbortIfLatestUpdatedTimeChanged = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactsRequest.Options.prototype.hasAbortIfLatestUpdatedTimeChanged = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * repeated Artifact artifacts = 1;
 * @return {!Array<!proto.ml_metadata.Artifact>}
 */
proto.ml_metadata.PutArtifactsRequest.prototype.getArtifactsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Artifact>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Artifact, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Artifact>} value
 * @return {!proto.ml_metadata.PutArtifactsRequest} returns this
*/
proto.ml_metadata.PutArtifactsRequest.prototype.setArtifactsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Artifact=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Artifact}
 */
proto.ml_metadata.PutArtifactsRequest.prototype.addArtifacts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Artifact, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutArtifactsRequest} returns this
 */
proto.ml_metadata.PutArtifactsRequest.prototype.clearArtifactsList = function() {
  return this.setArtifactsList([]);
};


/**
 * optional Options options = 2;
 * @return {?proto.ml_metadata.PutArtifactsRequest.Options}
 */
proto.ml_metadata.PutArtifactsRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.PutArtifactsRequest.Options} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.PutArtifactsRequest.Options, 2));
};


/**
 * @param {?proto.ml_metadata.PutArtifactsRequest.Options|undefined} value
 * @return {!proto.ml_metadata.PutArtifactsRequest} returns this
*/
proto.ml_metadata.PutArtifactsRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.PutArtifactsRequest} returns this
 */
proto.ml_metadata.PutArtifactsRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactsRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutArtifactsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutArtifactsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutArtifactsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutArtifactsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutArtifactsResponse}
 */
proto.ml_metadata.PutArtifactsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutArtifactsResponse;
  return proto.ml_metadata.PutArtifactsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutArtifactsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutArtifactsResponse}
 */
proto.ml_metadata.PutArtifactsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addArtifactIds(values[i]);
      }
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutArtifactsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutArtifactsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutArtifactsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
};


/**
 * repeated int64 artifact_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.PutArtifactsResponse.prototype.getArtifactIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.PutArtifactsResponse} returns this
 */
proto.ml_metadata.PutArtifactsResponse.prototype.setArtifactIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.PutArtifactsResponse} returns this
 */
proto.ml_metadata.PutArtifactsResponse.prototype.addArtifactIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutArtifactsResponse} returns this
 */
proto.ml_metadata.PutArtifactsResponse.prototype.clearArtifactIdsList = function() {
  return this.setArtifactIdsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutArtifactTypeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutArtifactTypeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactTypeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactType: (f = msg.getArtifactType()) && ml_metadata_proto_metadata_store_pb.ArtifactType.toObject(includeInstance, f),
    canAddFields: (f = jspb.Message.getBooleanField(msg, 2)) == null ? undefined : f,
    canOmitFields: (f = jspb.Message.getBooleanField(msg, 5)) == null ? undefined : f,
    canDeleteFields: (f = jspb.Message.getBooleanField(msg, 3)) == null ? undefined : f,
    allFieldsMatch: jspb.Message.getBooleanFieldWithDefault(msg, 4, true)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutArtifactTypeRequest}
 */
proto.ml_metadata.PutArtifactTypeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutArtifactTypeRequest;
  return proto.ml_metadata.PutArtifactTypeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutArtifactTypeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutArtifactTypeRequest}
 */
proto.ml_metadata.PutArtifactTypeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ArtifactType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ArtifactType.deserializeBinaryFromReader);
      msg.setArtifactType(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanAddFields(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanOmitFields(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanDeleteFields(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setAllFieldsMatch(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutArtifactTypeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutArtifactTypeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactTypeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactType();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ArtifactType.serializeBinaryToWriter
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeBool(
      2,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeBool(
      5,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeBool(
      3,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeBool(
      4,
      f
    );
  }
};


/**
 * optional ArtifactType artifact_type = 1;
 * @return {?proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.getArtifactType = function() {
  return /** @type{?proto.ml_metadata.ArtifactType} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ArtifactType, 1));
};


/**
 * @param {?proto.ml_metadata.ArtifactType|undefined} value
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
*/
proto.ml_metadata.PutArtifactTypeRequest.prototype.setArtifactType = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.clearArtifactType = function() {
  return this.setArtifactType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.hasArtifactType = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional bool can_add_fields = 2;
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.getCanAddFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.setCanAddFields = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.clearCanAddFields = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.hasCanAddFields = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional bool can_omit_fields = 5;
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.getCanOmitFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.setCanOmitFields = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.clearCanOmitFields = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.hasCanOmitFields = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional bool can_delete_fields = 3;
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.getCanDeleteFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.setCanDeleteFields = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.clearCanDeleteFields = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.hasCanDeleteFields = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional bool all_fields_match = 4;
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.getAllFieldsMatch = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, true));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.setAllFieldsMatch = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutArtifactTypeRequest} returns this
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.clearAllFieldsMatch = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeRequest.prototype.hasAllFieldsMatch = function() {
  return jspb.Message.getField(this, 4) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutArtifactTypeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutArtifactTypeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutArtifactTypeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactTypeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutArtifactTypeResponse}
 */
proto.ml_metadata.PutArtifactTypeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutArtifactTypeResponse;
  return proto.ml_metadata.PutArtifactTypeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutArtifactTypeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutArtifactTypeResponse}
 */
proto.ml_metadata.PutArtifactTypeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTypeId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutArtifactTypeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutArtifactTypeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutArtifactTypeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutArtifactTypeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
};


/**
 * optional int64 type_id = 1;
 * @return {number}
 */
proto.ml_metadata.PutArtifactTypeResponse.prototype.getTypeId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.PutArtifactTypeResponse} returns this
 */
proto.ml_metadata.PutArtifactTypeResponse.prototype.setTypeId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutArtifactTypeResponse} returns this
 */
proto.ml_metadata.PutArtifactTypeResponse.prototype.clearTypeId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutArtifactTypeResponse.prototype.hasTypeId = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutExecutionsRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutExecutionsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutExecutionsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutExecutionsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionsList: jspb.Message.toObjectList(msg.getExecutionsList(),
    ml_metadata_proto_metadata_store_pb.Execution.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutExecutionsRequest}
 */
proto.ml_metadata.PutExecutionsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutExecutionsRequest;
  return proto.ml_metadata.PutExecutionsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutExecutionsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutExecutionsRequest}
 */
proto.ml_metadata.PutExecutionsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Execution;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Execution.deserializeBinaryFromReader);
      msg.addExecutions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutExecutionsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutExecutionsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutExecutionsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Execution.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Execution executions = 1;
 * @return {!Array<!proto.ml_metadata.Execution>}
 */
proto.ml_metadata.PutExecutionsRequest.prototype.getExecutionsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Execution>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Execution, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Execution>} value
 * @return {!proto.ml_metadata.PutExecutionsRequest} returns this
*/
proto.ml_metadata.PutExecutionsRequest.prototype.setExecutionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Execution=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Execution}
 */
proto.ml_metadata.PutExecutionsRequest.prototype.addExecutions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Execution, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutExecutionsRequest} returns this
 */
proto.ml_metadata.PutExecutionsRequest.prototype.clearExecutionsList = function() {
  return this.setExecutionsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutExecutionsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutExecutionsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutExecutionsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutExecutionsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutExecutionsResponse}
 */
proto.ml_metadata.PutExecutionsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutExecutionsResponse;
  return proto.ml_metadata.PutExecutionsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutExecutionsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutExecutionsResponse}
 */
proto.ml_metadata.PutExecutionsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addExecutionIds(values[i]);
      }
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutExecutionsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutExecutionsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutExecutionsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
};


/**
 * repeated int64 execution_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.PutExecutionsResponse.prototype.getExecutionIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.PutExecutionsResponse} returns this
 */
proto.ml_metadata.PutExecutionsResponse.prototype.setExecutionIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.PutExecutionsResponse} returns this
 */
proto.ml_metadata.PutExecutionsResponse.prototype.addExecutionIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutExecutionsResponse} returns this
 */
proto.ml_metadata.PutExecutionsResponse.prototype.clearExecutionIdsList = function() {
  return this.setExecutionIdsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutExecutionTypeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutExecutionTypeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionTypeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionType: (f = msg.getExecutionType()) && ml_metadata_proto_metadata_store_pb.ExecutionType.toObject(includeInstance, f),
    canAddFields: (f = jspb.Message.getBooleanField(msg, 2)) == null ? undefined : f,
    canOmitFields: (f = jspb.Message.getBooleanField(msg, 5)) == null ? undefined : f,
    canDeleteFields: (f = jspb.Message.getBooleanField(msg, 3)) == null ? undefined : f,
    allFieldsMatch: jspb.Message.getBooleanFieldWithDefault(msg, 4, true)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutExecutionTypeRequest}
 */
proto.ml_metadata.PutExecutionTypeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutExecutionTypeRequest;
  return proto.ml_metadata.PutExecutionTypeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutExecutionTypeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutExecutionTypeRequest}
 */
proto.ml_metadata.PutExecutionTypeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ExecutionType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ExecutionType.deserializeBinaryFromReader);
      msg.setExecutionType(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanAddFields(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanOmitFields(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanDeleteFields(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setAllFieldsMatch(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutExecutionTypeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutExecutionTypeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionTypeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionType();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ExecutionType.serializeBinaryToWriter
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeBool(
      2,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeBool(
      5,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeBool(
      3,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeBool(
      4,
      f
    );
  }
};


/**
 * optional ExecutionType execution_type = 1;
 * @return {?proto.ml_metadata.ExecutionType}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.getExecutionType = function() {
  return /** @type{?proto.ml_metadata.ExecutionType} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ExecutionType, 1));
};


/**
 * @param {?proto.ml_metadata.ExecutionType|undefined} value
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
*/
proto.ml_metadata.PutExecutionTypeRequest.prototype.setExecutionType = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.clearExecutionType = function() {
  return this.setExecutionType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.hasExecutionType = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional bool can_add_fields = 2;
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.getCanAddFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.setCanAddFields = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.clearCanAddFields = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.hasCanAddFields = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional bool can_omit_fields = 5;
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.getCanOmitFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.setCanOmitFields = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.clearCanOmitFields = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.hasCanOmitFields = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional bool can_delete_fields = 3;
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.getCanDeleteFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.setCanDeleteFields = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.clearCanDeleteFields = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.hasCanDeleteFields = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional bool all_fields_match = 4;
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.getAllFieldsMatch = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, true));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.setAllFieldsMatch = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionTypeRequest} returns this
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.clearAllFieldsMatch = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeRequest.prototype.hasAllFieldsMatch = function() {
  return jspb.Message.getField(this, 4) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutExecutionTypeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutExecutionTypeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutExecutionTypeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionTypeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutExecutionTypeResponse}
 */
proto.ml_metadata.PutExecutionTypeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutExecutionTypeResponse;
  return proto.ml_metadata.PutExecutionTypeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutExecutionTypeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutExecutionTypeResponse}
 */
proto.ml_metadata.PutExecutionTypeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTypeId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutExecutionTypeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutExecutionTypeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutExecutionTypeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionTypeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
};


/**
 * optional int64 type_id = 1;
 * @return {number}
 */
proto.ml_metadata.PutExecutionTypeResponse.prototype.getTypeId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.PutExecutionTypeResponse} returns this
 */
proto.ml_metadata.PutExecutionTypeResponse.prototype.setTypeId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionTypeResponse} returns this
 */
proto.ml_metadata.PutExecutionTypeResponse.prototype.clearTypeId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionTypeResponse.prototype.hasTypeId = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutEventsRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutEventsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutEventsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutEventsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutEventsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    eventsList: jspb.Message.toObjectList(msg.getEventsList(),
    ml_metadata_proto_metadata_store_pb.Event.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutEventsRequest}
 */
proto.ml_metadata.PutEventsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutEventsRequest;
  return proto.ml_metadata.PutEventsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutEventsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutEventsRequest}
 */
proto.ml_metadata.PutEventsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Event;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Event.deserializeBinaryFromReader);
      msg.addEvents(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutEventsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutEventsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutEventsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutEventsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEventsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Event.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Event events = 1;
 * @return {!Array<!proto.ml_metadata.Event>}
 */
proto.ml_metadata.PutEventsRequest.prototype.getEventsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Event>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Event, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Event>} value
 * @return {!proto.ml_metadata.PutEventsRequest} returns this
*/
proto.ml_metadata.PutEventsRequest.prototype.setEventsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Event=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Event}
 */
proto.ml_metadata.PutEventsRequest.prototype.addEvents = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Event, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutEventsRequest} returns this
 */
proto.ml_metadata.PutEventsRequest.prototype.clearEventsList = function() {
  return this.setEventsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutEventsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutEventsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutEventsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutEventsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutEventsResponse}
 */
proto.ml_metadata.PutEventsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutEventsResponse;
  return proto.ml_metadata.PutEventsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutEventsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutEventsResponse}
 */
proto.ml_metadata.PutEventsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutEventsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutEventsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutEventsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutEventsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutExecutionRequest.repeatedFields_ = [2,3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutExecutionRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutExecutionRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutExecutionRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    execution: (f = msg.getExecution()) && ml_metadata_proto_metadata_store_pb.Execution.toObject(includeInstance, f),
    artifactEventPairsList: jspb.Message.toObjectList(msg.getArtifactEventPairsList(),
    proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.toObject, includeInstance),
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    ml_metadata_proto_metadata_store_pb.Context.toObject, includeInstance),
    options: (f = msg.getOptions()) && proto.ml_metadata.PutExecutionRequest.Options.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutExecutionRequest}
 */
proto.ml_metadata.PutExecutionRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutExecutionRequest;
  return proto.ml_metadata.PutExecutionRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutExecutionRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutExecutionRequest}
 */
proto.ml_metadata.PutExecutionRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Execution;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Execution.deserializeBinaryFromReader);
      msg.setExecution(value);
      break;
    case 2:
      var value = new proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent;
      reader.readMessage(value,proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.deserializeBinaryFromReader);
      msg.addArtifactEventPairs(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    case 4:
      var value = new proto.ml_metadata.PutExecutionRequest.Options;
      reader.readMessage(value,proto.ml_metadata.PutExecutionRequest.Options.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutExecutionRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutExecutionRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutExecutionRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecution();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Execution.serializeBinaryToWriter
    );
  }
  f = message.getArtifactEventPairsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.serializeBinaryToWriter
    );
  }
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.ml_metadata.PutExecutionRequest.Options.serializeBinaryToWriter
    );
  }
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifact: (f = msg.getArtifact()) && ml_metadata_proto_metadata_store_pb.Artifact.toObject(includeInstance, f),
    event: (f = msg.getEvent()) && ml_metadata_proto_metadata_store_pb.Event.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent}
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent;
  return proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent}
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Artifact;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Artifact.deserializeBinaryFromReader);
      msg.setArtifact(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.Event;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Event.deserializeBinaryFromReader);
      msg.setEvent(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifact();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Artifact.serializeBinaryToWriter
    );
  }
  f = message.getEvent();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.Event.serializeBinaryToWriter
    );
  }
};


/**
 * optional Artifact artifact = 1;
 * @return {?proto.ml_metadata.Artifact}
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.getArtifact = function() {
  return /** @type{?proto.ml_metadata.Artifact} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.Artifact, 1));
};


/**
 * @param {?proto.ml_metadata.Artifact|undefined} value
 * @return {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent} returns this
*/
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.setArtifact = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent} returns this
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.clearArtifact = function() {
  return this.setArtifact(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.hasArtifact = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional Event event = 2;
 * @return {?proto.ml_metadata.Event}
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.getEvent = function() {
  return /** @type{?proto.ml_metadata.Event} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.Event, 2));
};


/**
 * @param {?proto.ml_metadata.Event|undefined} value
 * @return {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent} returns this
*/
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.setEvent = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent} returns this
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.clearEvent = function() {
  return this.setEvent(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent.prototype.hasEvent = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutExecutionRequest.Options.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutExecutionRequest.Options.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutExecutionRequest.Options} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionRequest.Options.toObject = function(includeInstance, msg) {
  var f, obj = {
    reuseContextIfAlreadyExist: (f = jspb.Message.getBooleanField(msg, 1)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutExecutionRequest.Options}
 */
proto.ml_metadata.PutExecutionRequest.Options.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutExecutionRequest.Options;
  return proto.ml_metadata.PutExecutionRequest.Options.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutExecutionRequest.Options} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutExecutionRequest.Options}
 */
proto.ml_metadata.PutExecutionRequest.Options.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setReuseContextIfAlreadyExist(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutExecutionRequest.Options.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutExecutionRequest.Options.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutExecutionRequest.Options} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionRequest.Options.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {boolean} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool reuse_context_if_already_exist = 1;
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionRequest.Options.prototype.getReuseContextIfAlreadyExist = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutExecutionRequest.Options} returns this
 */
proto.ml_metadata.PutExecutionRequest.Options.prototype.setReuseContextIfAlreadyExist = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionRequest.Options} returns this
 */
proto.ml_metadata.PutExecutionRequest.Options.prototype.clearReuseContextIfAlreadyExist = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionRequest.Options.prototype.hasReuseContextIfAlreadyExist = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional Execution execution = 1;
 * @return {?proto.ml_metadata.Execution}
 */
proto.ml_metadata.PutExecutionRequest.prototype.getExecution = function() {
  return /** @type{?proto.ml_metadata.Execution} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.Execution, 1));
};


/**
 * @param {?proto.ml_metadata.Execution|undefined} value
 * @return {!proto.ml_metadata.PutExecutionRequest} returns this
*/
proto.ml_metadata.PutExecutionRequest.prototype.setExecution = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionRequest} returns this
 */
proto.ml_metadata.PutExecutionRequest.prototype.clearExecution = function() {
  return this.setExecution(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionRequest.prototype.hasExecution = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * repeated ArtifactAndEvent artifact_event_pairs = 2;
 * @return {!Array<!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent>}
 */
proto.ml_metadata.PutExecutionRequest.prototype.getArtifactEventPairsList = function() {
  return /** @type{!Array<!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent, 2));
};


/**
 * @param {!Array<!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent>} value
 * @return {!proto.ml_metadata.PutExecutionRequest} returns this
*/
proto.ml_metadata.PutExecutionRequest.prototype.setArtifactEventPairsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent}
 */
proto.ml_metadata.PutExecutionRequest.prototype.addArtifactEventPairs = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.ml_metadata.PutExecutionRequest.ArtifactAndEvent, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutExecutionRequest} returns this
 */
proto.ml_metadata.PutExecutionRequest.prototype.clearArtifactEventPairsList = function() {
  return this.setArtifactEventPairsList([]);
};


/**
 * repeated Context contexts = 3;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.PutExecutionRequest.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 3));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.PutExecutionRequest} returns this
*/
proto.ml_metadata.PutExecutionRequest.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.PutExecutionRequest.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutExecutionRequest} returns this
 */
proto.ml_metadata.PutExecutionRequest.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};


/**
 * optional Options options = 4;
 * @return {?proto.ml_metadata.PutExecutionRequest.Options}
 */
proto.ml_metadata.PutExecutionRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.PutExecutionRequest.Options} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.PutExecutionRequest.Options, 4));
};


/**
 * @param {?proto.ml_metadata.PutExecutionRequest.Options|undefined} value
 * @return {!proto.ml_metadata.PutExecutionRequest} returns this
*/
proto.ml_metadata.PutExecutionRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionRequest} returns this
 */
proto.ml_metadata.PutExecutionRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 4) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutExecutionResponse.repeatedFields_ = [2,3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutExecutionResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutExecutionResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutExecutionResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    artifactIdsList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f,
    contextIdsList: (f = jspb.Message.getRepeatedField(msg, 3)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutExecutionResponse}
 */
proto.ml_metadata.PutExecutionResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutExecutionResponse;
  return proto.ml_metadata.PutExecutionResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutExecutionResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutExecutionResponse}
 */
proto.ml_metadata.PutExecutionResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setExecutionId(value);
      break;
    case 2:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addArtifactIds(values[i]);
      }
      break;
    case 3:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addContextIds(values[i]);
      }
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutExecutionResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutExecutionResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutExecutionResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutExecutionResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = message.getArtifactIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      2,
      f
    );
  }
  f = message.getContextIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      3,
      f
    );
  }
};


/**
 * optional int64 execution_id = 1;
 * @return {number}
 */
proto.ml_metadata.PutExecutionResponse.prototype.getExecutionId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.PutExecutionResponse} returns this
 */
proto.ml_metadata.PutExecutionResponse.prototype.setExecutionId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutExecutionResponse} returns this
 */
proto.ml_metadata.PutExecutionResponse.prototype.clearExecutionId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutExecutionResponse.prototype.hasExecutionId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * repeated int64 artifact_ids = 2;
 * @return {!Array<number>}
 */
proto.ml_metadata.PutExecutionResponse.prototype.getArtifactIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.PutExecutionResponse} returns this
 */
proto.ml_metadata.PutExecutionResponse.prototype.setArtifactIdsList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.PutExecutionResponse} returns this
 */
proto.ml_metadata.PutExecutionResponse.prototype.addArtifactIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutExecutionResponse} returns this
 */
proto.ml_metadata.PutExecutionResponse.prototype.clearArtifactIdsList = function() {
  return this.setArtifactIdsList([]);
};


/**
 * repeated int64 context_ids = 3;
 * @return {!Array<number>}
 */
proto.ml_metadata.PutExecutionResponse.prototype.getContextIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 3));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.PutExecutionResponse} returns this
 */
proto.ml_metadata.PutExecutionResponse.prototype.setContextIdsList = function(value) {
  return jspb.Message.setField(this, 3, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.PutExecutionResponse} returns this
 */
proto.ml_metadata.PutExecutionResponse.prototype.addContextIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 3, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutExecutionResponse} returns this
 */
proto.ml_metadata.PutExecutionResponse.prototype.clearContextIdsList = function() {
  return this.setContextIdsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutTypesRequest.repeatedFields_ = [1,2,3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutTypesRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutTypesRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutTypesRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutTypesRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactTypesList: jspb.Message.toObjectList(msg.getArtifactTypesList(),
    ml_metadata_proto_metadata_store_pb.ArtifactType.toObject, includeInstance),
    executionTypesList: jspb.Message.toObjectList(msg.getExecutionTypesList(),
    ml_metadata_proto_metadata_store_pb.ExecutionType.toObject, includeInstance),
    contextTypesList: jspb.Message.toObjectList(msg.getContextTypesList(),
    ml_metadata_proto_metadata_store_pb.ContextType.toObject, includeInstance),
    canAddFields: (f = jspb.Message.getBooleanField(msg, 4)) == null ? undefined : f,
    canOmitFields: (f = jspb.Message.getBooleanField(msg, 7)) == null ? undefined : f,
    canDeleteFields: (f = jspb.Message.getBooleanField(msg, 5)) == null ? undefined : f,
    allFieldsMatch: jspb.Message.getBooleanFieldWithDefault(msg, 6, true)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutTypesRequest}
 */
proto.ml_metadata.PutTypesRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutTypesRequest;
  return proto.ml_metadata.PutTypesRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutTypesRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutTypesRequest}
 */
proto.ml_metadata.PutTypesRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ArtifactType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ArtifactType.deserializeBinaryFromReader);
      msg.addArtifactTypes(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.ExecutionType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ExecutionType.deserializeBinaryFromReader);
      msg.addExecutionTypes(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.ContextType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ContextType.deserializeBinaryFromReader);
      msg.addContextTypes(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanAddFields(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanOmitFields(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanDeleteFields(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setAllFieldsMatch(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutTypesRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutTypesRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutTypesRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutTypesRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ArtifactType.serializeBinaryToWriter
    );
  }
  f = message.getExecutionTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.ExecutionType.serializeBinaryToWriter
    );
  }
  f = message.getContextTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.ContextType.serializeBinaryToWriter
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeBool(
      4,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 7));
  if (f != null) {
    writer.writeBool(
      7,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeBool(
      5,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 6));
  if (f != null) {
    writer.writeBool(
      6,
      f
    );
  }
};


/**
 * repeated ArtifactType artifact_types = 1;
 * @return {!Array<!proto.ml_metadata.ArtifactType>}
 */
proto.ml_metadata.PutTypesRequest.prototype.getArtifactTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ArtifactType>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ArtifactType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ArtifactType>} value
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
*/
proto.ml_metadata.PutTypesRequest.prototype.setArtifactTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ArtifactType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.PutTypesRequest.prototype.addArtifactTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ArtifactType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.clearArtifactTypesList = function() {
  return this.setArtifactTypesList([]);
};


/**
 * repeated ExecutionType execution_types = 2;
 * @return {!Array<!proto.ml_metadata.ExecutionType>}
 */
proto.ml_metadata.PutTypesRequest.prototype.getExecutionTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ExecutionType>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ExecutionType, 2));
};


/**
 * @param {!Array<!proto.ml_metadata.ExecutionType>} value
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
*/
proto.ml_metadata.PutTypesRequest.prototype.setExecutionTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.ml_metadata.ExecutionType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ExecutionType}
 */
proto.ml_metadata.PutTypesRequest.prototype.addExecutionTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.ml_metadata.ExecutionType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.clearExecutionTypesList = function() {
  return this.setExecutionTypesList([]);
};


/**
 * repeated ContextType context_types = 3;
 * @return {!Array<!proto.ml_metadata.ContextType>}
 */
proto.ml_metadata.PutTypesRequest.prototype.getContextTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ContextType>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ContextType, 3));
};


/**
 * @param {!Array<!proto.ml_metadata.ContextType>} value
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
*/
proto.ml_metadata.PutTypesRequest.prototype.setContextTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.ml_metadata.ContextType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ContextType}
 */
proto.ml_metadata.PutTypesRequest.prototype.addContextTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.ml_metadata.ContextType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.clearContextTypesList = function() {
  return this.setContextTypesList([]);
};


/**
 * optional bool can_add_fields = 4;
 * @return {boolean}
 */
proto.ml_metadata.PutTypesRequest.prototype.getCanAddFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.setCanAddFields = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.clearCanAddFields = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutTypesRequest.prototype.hasCanAddFields = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional bool can_omit_fields = 7;
 * @return {boolean}
 */
proto.ml_metadata.PutTypesRequest.prototype.getCanOmitFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.setCanOmitFields = function(value) {
  return jspb.Message.setField(this, 7, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.clearCanOmitFields = function() {
  return jspb.Message.setField(this, 7, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutTypesRequest.prototype.hasCanOmitFields = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional bool can_delete_fields = 5;
 * @return {boolean}
 */
proto.ml_metadata.PutTypesRequest.prototype.getCanDeleteFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.setCanDeleteFields = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.clearCanDeleteFields = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutTypesRequest.prototype.hasCanDeleteFields = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional bool all_fields_match = 6;
 * @return {boolean}
 */
proto.ml_metadata.PutTypesRequest.prototype.getAllFieldsMatch = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, true));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.setAllFieldsMatch = function(value) {
  return jspb.Message.setField(this, 6, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutTypesRequest} returns this
 */
proto.ml_metadata.PutTypesRequest.prototype.clearAllFieldsMatch = function() {
  return jspb.Message.setField(this, 6, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutTypesRequest.prototype.hasAllFieldsMatch = function() {
  return jspb.Message.getField(this, 6) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutTypesResponse.repeatedFields_ = [1,2,3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutTypesResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutTypesResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutTypesResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutTypesResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactTypeIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    executionTypeIdsList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f,
    contextTypeIdsList: (f = jspb.Message.getRepeatedField(msg, 3)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutTypesResponse}
 */
proto.ml_metadata.PutTypesResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutTypesResponse;
  return proto.ml_metadata.PutTypesResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutTypesResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutTypesResponse}
 */
proto.ml_metadata.PutTypesResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addArtifactTypeIds(values[i]);
      }
      break;
    case 2:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addExecutionTypeIds(values[i]);
      }
      break;
    case 3:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addContextTypeIds(values[i]);
      }
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutTypesResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutTypesResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutTypesResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutTypesResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactTypeIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
  f = message.getExecutionTypeIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      2,
      f
    );
  }
  f = message.getContextTypeIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      3,
      f
    );
  }
};


/**
 * repeated int64 artifact_type_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.PutTypesResponse.prototype.getArtifactTypeIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.PutTypesResponse} returns this
 */
proto.ml_metadata.PutTypesResponse.prototype.setArtifactTypeIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.PutTypesResponse} returns this
 */
proto.ml_metadata.PutTypesResponse.prototype.addArtifactTypeIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutTypesResponse} returns this
 */
proto.ml_metadata.PutTypesResponse.prototype.clearArtifactTypeIdsList = function() {
  return this.setArtifactTypeIdsList([]);
};


/**
 * repeated int64 execution_type_ids = 2;
 * @return {!Array<number>}
 */
proto.ml_metadata.PutTypesResponse.prototype.getExecutionTypeIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.PutTypesResponse} returns this
 */
proto.ml_metadata.PutTypesResponse.prototype.setExecutionTypeIdsList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.PutTypesResponse} returns this
 */
proto.ml_metadata.PutTypesResponse.prototype.addExecutionTypeIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutTypesResponse} returns this
 */
proto.ml_metadata.PutTypesResponse.prototype.clearExecutionTypeIdsList = function() {
  return this.setExecutionTypeIdsList([]);
};


/**
 * repeated int64 context_type_ids = 3;
 * @return {!Array<number>}
 */
proto.ml_metadata.PutTypesResponse.prototype.getContextTypeIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 3));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.PutTypesResponse} returns this
 */
proto.ml_metadata.PutTypesResponse.prototype.setContextTypeIdsList = function(value) {
  return jspb.Message.setField(this, 3, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.PutTypesResponse} returns this
 */
proto.ml_metadata.PutTypesResponse.prototype.addContextTypeIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 3, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutTypesResponse} returns this
 */
proto.ml_metadata.PutTypesResponse.prototype.clearContextTypeIdsList = function() {
  return this.setContextTypeIdsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutContextTypeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutContextTypeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutContextTypeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextType: (f = msg.getContextType()) && ml_metadata_proto_metadata_store_pb.ContextType.toObject(includeInstance, f),
    canAddFields: (f = jspb.Message.getBooleanField(msg, 2)) == null ? undefined : f,
    canOmitFields: (f = jspb.Message.getBooleanField(msg, 5)) == null ? undefined : f,
    canDeleteFields: (f = jspb.Message.getBooleanField(msg, 3)) == null ? undefined : f,
    allFieldsMatch: jspb.Message.getBooleanFieldWithDefault(msg, 4, true)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutContextTypeRequest}
 */
proto.ml_metadata.PutContextTypeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutContextTypeRequest;
  return proto.ml_metadata.PutContextTypeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutContextTypeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutContextTypeRequest}
 */
proto.ml_metadata.PutContextTypeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ContextType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ContextType.deserializeBinaryFromReader);
      msg.setContextType(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanAddFields(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanOmitFields(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCanDeleteFields(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setAllFieldsMatch(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutContextTypeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutContextTypeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutContextTypeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextType();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ContextType.serializeBinaryToWriter
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeBool(
      2,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeBool(
      5,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeBool(
      3,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeBool(
      4,
      f
    );
  }
};


/**
 * optional ContextType context_type = 1;
 * @return {?proto.ml_metadata.ContextType}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.getContextType = function() {
  return /** @type{?proto.ml_metadata.ContextType} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ContextType, 1));
};


/**
 * @param {?proto.ml_metadata.ContextType|undefined} value
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
*/
proto.ml_metadata.PutContextTypeRequest.prototype.setContextType = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
 */
proto.ml_metadata.PutContextTypeRequest.prototype.clearContextType = function() {
  return this.setContextType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.hasContextType = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional bool can_add_fields = 2;
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.getCanAddFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
 */
proto.ml_metadata.PutContextTypeRequest.prototype.setCanAddFields = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
 */
proto.ml_metadata.PutContextTypeRequest.prototype.clearCanAddFields = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.hasCanAddFields = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional bool can_omit_fields = 5;
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.getCanOmitFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
 */
proto.ml_metadata.PutContextTypeRequest.prototype.setCanOmitFields = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
 */
proto.ml_metadata.PutContextTypeRequest.prototype.clearCanOmitFields = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.hasCanOmitFields = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional bool can_delete_fields = 3;
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.getCanDeleteFields = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
 */
proto.ml_metadata.PutContextTypeRequest.prototype.setCanDeleteFields = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
 */
proto.ml_metadata.PutContextTypeRequest.prototype.clearCanDeleteFields = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.hasCanDeleteFields = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional bool all_fields_match = 4;
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.getAllFieldsMatch = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, true));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
 */
proto.ml_metadata.PutContextTypeRequest.prototype.setAllFieldsMatch = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutContextTypeRequest} returns this
 */
proto.ml_metadata.PutContextTypeRequest.prototype.clearAllFieldsMatch = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeRequest.prototype.hasAllFieldsMatch = function() {
  return jspb.Message.getField(this, 4) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutContextTypeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutContextTypeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutContextTypeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutContextTypeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutContextTypeResponse}
 */
proto.ml_metadata.PutContextTypeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutContextTypeResponse;
  return proto.ml_metadata.PutContextTypeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutContextTypeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutContextTypeResponse}
 */
proto.ml_metadata.PutContextTypeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTypeId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutContextTypeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutContextTypeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutContextTypeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutContextTypeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
};


/**
 * optional int64 type_id = 1;
 * @return {number}
 */
proto.ml_metadata.PutContextTypeResponse.prototype.getTypeId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.PutContextTypeResponse} returns this
 */
proto.ml_metadata.PutContextTypeResponse.prototype.setTypeId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.PutContextTypeResponse} returns this
 */
proto.ml_metadata.PutContextTypeResponse.prototype.clearTypeId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.PutContextTypeResponse.prototype.hasTypeId = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutContextsRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutContextsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutContextsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutContextsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutContextsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    ml_metadata_proto_metadata_store_pb.Context.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutContextsRequest}
 */
proto.ml_metadata.PutContextsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutContextsRequest;
  return proto.ml_metadata.PutContextsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutContextsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutContextsRequest}
 */
proto.ml_metadata.PutContextsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutContextsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutContextsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutContextsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutContextsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Context contexts = 1;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.PutContextsRequest.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.PutContextsRequest} returns this
*/
proto.ml_metadata.PutContextsRequest.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.PutContextsRequest.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutContextsRequest} returns this
 */
proto.ml_metadata.PutContextsRequest.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutContextsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutContextsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutContextsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutContextsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutContextsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutContextsResponse}
 */
proto.ml_metadata.PutContextsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutContextsResponse;
  return proto.ml_metadata.PutContextsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutContextsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutContextsResponse}
 */
proto.ml_metadata.PutContextsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addContextIds(values[i]);
      }
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutContextsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutContextsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutContextsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutContextsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
};


/**
 * repeated int64 context_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.PutContextsResponse.prototype.getContextIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.PutContextsResponse} returns this
 */
proto.ml_metadata.PutContextsResponse.prototype.setContextIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.PutContextsResponse} returns this
 */
proto.ml_metadata.PutContextsResponse.prototype.addContextIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutContextsResponse} returns this
 */
proto.ml_metadata.PutContextsResponse.prototype.clearContextIdsList = function() {
  return this.setContextIdsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.repeatedFields_ = [1,2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutAttributionsAndAssociationsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutAttributionsAndAssociationsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    attributionsList: jspb.Message.toObjectList(msg.getAttributionsList(),
    ml_metadata_proto_metadata_store_pb.Attribution.toObject, includeInstance),
    associationsList: jspb.Message.toObjectList(msg.getAssociationsList(),
    ml_metadata_proto_metadata_store_pb.Association.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutAttributionsAndAssociationsRequest}
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutAttributionsAndAssociationsRequest;
  return proto.ml_metadata.PutAttributionsAndAssociationsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutAttributionsAndAssociationsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutAttributionsAndAssociationsRequest}
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Attribution;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Attribution.deserializeBinaryFromReader);
      msg.addAttributions(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.Association;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Association.deserializeBinaryFromReader);
      msg.addAssociations(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutAttributionsAndAssociationsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutAttributionsAndAssociationsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getAttributionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Attribution.serializeBinaryToWriter
    );
  }
  f = message.getAssociationsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.Association.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Attribution attributions = 1;
 * @return {!Array<!proto.ml_metadata.Attribution>}
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.getAttributionsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Attribution>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Attribution, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Attribution>} value
 * @return {!proto.ml_metadata.PutAttributionsAndAssociationsRequest} returns this
*/
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.setAttributionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Attribution=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Attribution}
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.addAttributions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Attribution, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutAttributionsAndAssociationsRequest} returns this
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.clearAttributionsList = function() {
  return this.setAttributionsList([]);
};


/**
 * repeated Association associations = 2;
 * @return {!Array<!proto.ml_metadata.Association>}
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.getAssociationsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Association>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Association, 2));
};


/**
 * @param {!Array<!proto.ml_metadata.Association>} value
 * @return {!proto.ml_metadata.PutAttributionsAndAssociationsRequest} returns this
*/
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.setAssociationsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.ml_metadata.Association=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Association}
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.addAssociations = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.ml_metadata.Association, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutAttributionsAndAssociationsRequest} returns this
 */
proto.ml_metadata.PutAttributionsAndAssociationsRequest.prototype.clearAssociationsList = function() {
  return this.setAssociationsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutAttributionsAndAssociationsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutAttributionsAndAssociationsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutAttributionsAndAssociationsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutAttributionsAndAssociationsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutAttributionsAndAssociationsResponse}
 */
proto.ml_metadata.PutAttributionsAndAssociationsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutAttributionsAndAssociationsResponse;
  return proto.ml_metadata.PutAttributionsAndAssociationsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutAttributionsAndAssociationsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutAttributionsAndAssociationsResponse}
 */
proto.ml_metadata.PutAttributionsAndAssociationsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutAttributionsAndAssociationsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutAttributionsAndAssociationsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutAttributionsAndAssociationsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutAttributionsAndAssociationsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.PutParentContextsRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutParentContextsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutParentContextsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutParentContextsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutParentContextsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    parentContextsList: jspb.Message.toObjectList(msg.getParentContextsList(),
    ml_metadata_proto_metadata_store_pb.ParentContext.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutParentContextsRequest}
 */
proto.ml_metadata.PutParentContextsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutParentContextsRequest;
  return proto.ml_metadata.PutParentContextsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutParentContextsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutParentContextsRequest}
 */
proto.ml_metadata.PutParentContextsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ParentContext;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ParentContext.deserializeBinaryFromReader);
      msg.addParentContexts(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutParentContextsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutParentContextsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutParentContextsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutParentContextsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getParentContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ParentContext.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ParentContext parent_contexts = 1;
 * @return {!Array<!proto.ml_metadata.ParentContext>}
 */
proto.ml_metadata.PutParentContextsRequest.prototype.getParentContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.ParentContext>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ParentContext, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ParentContext>} value
 * @return {!proto.ml_metadata.PutParentContextsRequest} returns this
*/
proto.ml_metadata.PutParentContextsRequest.prototype.setParentContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ParentContext=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ParentContext}
 */
proto.ml_metadata.PutParentContextsRequest.prototype.addParentContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ParentContext, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.PutParentContextsRequest} returns this
 */
proto.ml_metadata.PutParentContextsRequest.prototype.clearParentContextsList = function() {
  return this.setParentContextsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.PutParentContextsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.PutParentContextsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.PutParentContextsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutParentContextsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.PutParentContextsResponse}
 */
proto.ml_metadata.PutParentContextsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.PutParentContextsResponse;
  return proto.ml_metadata.PutParentContextsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.PutParentContextsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.PutParentContextsResponse}
 */
proto.ml_metadata.PutParentContextsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.PutParentContextsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.PutParentContextsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.PutParentContextsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.PutParentContextsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsByTypeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsByTypeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByTypeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    typeVersion: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    options: (f = msg.getOptions()) && ml_metadata_proto_metadata_store_pb.ListOperationOptions.toObject(includeInstance, f),
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsByTypeRequest;
  return proto.ml_metadata.GetArtifactsByTypeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsByTypeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeVersion(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.ListOperationOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ListOperationOptions.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    case 4:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsByTypeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsByTypeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByTypeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string type_version = 2;
 * @return {string}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.getTypeVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.setTypeVersion = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.clearTypeVersion = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.hasTypeVersion = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional ListOperationOptions options = 3;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ListOperationOptions, 3));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest} returns this
*/
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional TransactionOptions transaction_options = 4;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 4));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest} returns this
*/
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByTypeRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 4) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactsByTypeResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsByTypeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsByTypeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByTypeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsList: jspb.Message.toObjectList(msg.getArtifactsList(),
    ml_metadata_proto_metadata_store_pb.Artifact.toObject, includeInstance),
    nextPageToken: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsByTypeResponse}
 */
proto.ml_metadata.GetArtifactsByTypeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsByTypeResponse;
  return proto.ml_metadata.GetArtifactsByTypeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsByTypeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsByTypeResponse}
 */
proto.ml_metadata.GetArtifactsByTypeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Artifact;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Artifact.deserializeBinaryFromReader);
      msg.addArtifacts(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsByTypeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsByTypeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByTypeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Artifact.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated Artifact artifacts = 1;
 * @return {!Array<!proto.ml_metadata.Artifact>}
 */
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.getArtifactsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Artifact>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Artifact, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Artifact>} value
 * @return {!proto.ml_metadata.GetArtifactsByTypeResponse} returns this
*/
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.setArtifactsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Artifact=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Artifact}
 */
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.addArtifacts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Artifact, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactsByTypeResponse} returns this
 */
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.clearArtifactsList = function() {
  return this.setArtifactsList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactsByTypeResponse} returns this
 */
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByTypeResponse} returns this
 */
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.clearNextPageToken = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByTypeResponse.prototype.hasNextPageToken = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactByTypeAndNameRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    typeVersion: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    artifactName: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactByTypeAndNameRequest;
  return proto.ml_metadata.GetArtifactByTypeAndNameRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeVersion(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setArtifactName(value);
      break;
    case 4:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactByTypeAndNameRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string type_version = 3;
 * @return {string}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.getTypeVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.setTypeVersion = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.clearTypeVersion = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.hasTypeVersion = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string artifact_name = 2;
 * @return {string}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.getArtifactName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.setArtifactName = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.clearArtifactName = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.hasArtifactName = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional TransactionOptions transaction_options = 4;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 4));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} returns this
*/
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactByTypeAndNameRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 4) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactByTypeAndNameResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactByTypeAndNameResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifact: (f = msg.getArtifact()) && ml_metadata_proto_metadata_store_pb.Artifact.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameResponse}
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactByTypeAndNameResponse;
  return proto.ml_metadata.GetArtifactByTypeAndNameResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactByTypeAndNameResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameResponse}
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Artifact;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Artifact.deserializeBinaryFromReader);
      msg.setArtifact(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactByTypeAndNameResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactByTypeAndNameResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifact();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Artifact.serializeBinaryToWriter
    );
  }
};


/**
 * optional Artifact artifact = 1;
 * @return {?proto.ml_metadata.Artifact}
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse.prototype.getArtifact = function() {
  return /** @type{?proto.ml_metadata.Artifact} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.Artifact, 1));
};


/**
 * @param {?proto.ml_metadata.Artifact|undefined} value
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameResponse} returns this
*/
proto.ml_metadata.GetArtifactByTypeAndNameResponse.prototype.setArtifact = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactByTypeAndNameResponse} returns this
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse.prototype.clearArtifact = function() {
  return this.setArtifact(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactByTypeAndNameResponse.prototype.hasArtifact = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactsByIDRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsByIDRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsByIDRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsByIDRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByIDRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsByIDRequest}
 */
proto.ml_metadata.GetArtifactsByIDRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsByIDRequest;
  return proto.ml_metadata.GetArtifactsByIDRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsByIDRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsByIDRequest}
 */
proto.ml_metadata.GetArtifactsByIDRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addArtifactIds(values[i]);
      }
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsByIDRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsByIDRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsByIDRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByIDRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated int64 artifact_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.GetArtifactsByIDRequest.prototype.getArtifactIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.GetArtifactsByIDRequest} returns this
 */
proto.ml_metadata.GetArtifactsByIDRequest.prototype.setArtifactIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.GetArtifactsByIDRequest} returns this
 */
proto.ml_metadata.GetArtifactsByIDRequest.prototype.addArtifactIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactsByIDRequest} returns this
 */
proto.ml_metadata.GetArtifactsByIDRequest.prototype.clearArtifactIdsList = function() {
  return this.setArtifactIdsList([]);
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetArtifactsByIDRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactsByIDRequest} returns this
*/
proto.ml_metadata.GetArtifactsByIDRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByIDRequest} returns this
 */
proto.ml_metadata.GetArtifactsByIDRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByIDRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactsByIDResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsByIDResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsByIDResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsByIDResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByIDResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsList: jspb.Message.toObjectList(msg.getArtifactsList(),
    ml_metadata_proto_metadata_store_pb.Artifact.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsByIDResponse}
 */
proto.ml_metadata.GetArtifactsByIDResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsByIDResponse;
  return proto.ml_metadata.GetArtifactsByIDResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsByIDResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsByIDResponse}
 */
proto.ml_metadata.GetArtifactsByIDResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Artifact;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Artifact.deserializeBinaryFromReader);
      msg.addArtifacts(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsByIDResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsByIDResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsByIDResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByIDResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Artifact.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Artifact artifacts = 1;
 * @return {!Array<!proto.ml_metadata.Artifact>}
 */
proto.ml_metadata.GetArtifactsByIDResponse.prototype.getArtifactsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Artifact>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Artifact, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Artifact>} value
 * @return {!proto.ml_metadata.GetArtifactsByIDResponse} returns this
*/
proto.ml_metadata.GetArtifactsByIDResponse.prototype.setArtifactsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Artifact=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Artifact}
 */
proto.ml_metadata.GetArtifactsByIDResponse.prototype.addArtifacts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Artifact, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactsByIDResponse} returns this
 */
proto.ml_metadata.GetArtifactsByIDResponse.prototype.clearArtifactsList = function() {
  return this.setArtifactsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    options: (f = msg.getOptions()) && ml_metadata_proto_metadata_store_pb.ListOperationOptions.toObject(includeInstance, f),
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsRequest}
 */
proto.ml_metadata.GetArtifactsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsRequest;
  return proto.ml_metadata.GetArtifactsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsRequest}
 */
proto.ml_metadata.GetArtifactsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ListOperationOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ListOperationOptions.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional ListOperationOptions options = 1;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.GetArtifactsRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ListOperationOptions, 1));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactsRequest} returns this
*/
proto.ml_metadata.GetArtifactsRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsRequest} returns this
 */
proto.ml_metadata.GetArtifactsRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetArtifactsRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactsRequest} returns this
*/
proto.ml_metadata.GetArtifactsRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsRequest} returns this
 */
proto.ml_metadata.GetArtifactsRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsList: jspb.Message.toObjectList(msg.getArtifactsList(),
    ml_metadata_proto_metadata_store_pb.Artifact.toObject, includeInstance),
    nextPageToken: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsResponse}
 */
proto.ml_metadata.GetArtifactsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsResponse;
  return proto.ml_metadata.GetArtifactsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsResponse}
 */
proto.ml_metadata.GetArtifactsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Artifact;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Artifact.deserializeBinaryFromReader);
      msg.addArtifacts(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Artifact.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated Artifact artifacts = 1;
 * @return {!Array<!proto.ml_metadata.Artifact>}
 */
proto.ml_metadata.GetArtifactsResponse.prototype.getArtifactsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Artifact>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Artifact, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Artifact>} value
 * @return {!proto.ml_metadata.GetArtifactsResponse} returns this
*/
proto.ml_metadata.GetArtifactsResponse.prototype.setArtifactsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Artifact=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Artifact}
 */
proto.ml_metadata.GetArtifactsResponse.prototype.addArtifacts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Artifact, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactsResponse} returns this
 */
proto.ml_metadata.GetArtifactsResponse.prototype.clearArtifactsList = function() {
  return this.setArtifactsList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.ml_metadata.GetArtifactsResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactsResponse} returns this
 */
proto.ml_metadata.GetArtifactsResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsResponse} returns this
 */
proto.ml_metadata.GetArtifactsResponse.prototype.clearNextPageToken = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsResponse.prototype.hasNextPageToken = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactsByURIRequest.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsByURIRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsByURIRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsByURIRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByURIRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    urisList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsByURIRequest}
 */
proto.ml_metadata.GetArtifactsByURIRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsByURIRequest;
  return proto.ml_metadata.GetArtifactsByURIRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsByURIRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsByURIRequest}
 */
proto.ml_metadata.GetArtifactsByURIRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.addUris(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsByURIRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsByURIRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsByURIRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByURIRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUrisList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      2,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated string uris = 2;
 * @return {!Array<string>}
 */
proto.ml_metadata.GetArtifactsByURIRequest.prototype.getUrisList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.ml_metadata.GetArtifactsByURIRequest} returns this
 */
proto.ml_metadata.GetArtifactsByURIRequest.prototype.setUrisList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.GetArtifactsByURIRequest} returns this
 */
proto.ml_metadata.GetArtifactsByURIRequest.prototype.addUris = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactsByURIRequest} returns this
 */
proto.ml_metadata.GetArtifactsByURIRequest.prototype.clearUrisList = function() {
  return this.setUrisList([]);
};


/**
 * optional TransactionOptions transaction_options = 3;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetArtifactsByURIRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 3));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactsByURIRequest} returns this
*/
proto.ml_metadata.GetArtifactsByURIRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByURIRequest} returns this
 */
proto.ml_metadata.GetArtifactsByURIRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByURIRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactsByURIResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsByURIResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsByURIResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsByURIResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByURIResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsList: jspb.Message.toObjectList(msg.getArtifactsList(),
    ml_metadata_proto_metadata_store_pb.Artifact.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsByURIResponse}
 */
proto.ml_metadata.GetArtifactsByURIResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsByURIResponse;
  return proto.ml_metadata.GetArtifactsByURIResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsByURIResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsByURIResponse}
 */
proto.ml_metadata.GetArtifactsByURIResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Artifact;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Artifact.deserializeBinaryFromReader);
      msg.addArtifacts(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsByURIResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsByURIResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsByURIResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByURIResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Artifact.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Artifact artifacts = 1;
 * @return {!Array<!proto.ml_metadata.Artifact>}
 */
proto.ml_metadata.GetArtifactsByURIResponse.prototype.getArtifactsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Artifact>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Artifact, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Artifact>} value
 * @return {!proto.ml_metadata.GetArtifactsByURIResponse} returns this
*/
proto.ml_metadata.GetArtifactsByURIResponse.prototype.setArtifactsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Artifact=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Artifact}
 */
proto.ml_metadata.GetArtifactsByURIResponse.prototype.addArtifacts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Artifact, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactsByURIResponse} returns this
 */
proto.ml_metadata.GetArtifactsByURIResponse.prototype.clearArtifactsList = function() {
  return this.setArtifactsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    options: (f = msg.getOptions()) && ml_metadata_proto_metadata_store_pb.ListOperationOptions.toObject(includeInstance, f),
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionsRequest}
 */
proto.ml_metadata.GetExecutionsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionsRequest;
  return proto.ml_metadata.GetExecutionsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionsRequest}
 */
proto.ml_metadata.GetExecutionsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ListOperationOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ListOperationOptions.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional ListOperationOptions options = 1;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.GetExecutionsRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ListOperationOptions, 1));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionsRequest} returns this
*/
proto.ml_metadata.GetExecutionsRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsRequest} returns this
 */
proto.ml_metadata.GetExecutionsRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetExecutionsRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionsRequest} returns this
*/
proto.ml_metadata.GetExecutionsRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsRequest} returns this
 */
proto.ml_metadata.GetExecutionsRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetExecutionsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionsList: jspb.Message.toObjectList(msg.getExecutionsList(),
    ml_metadata_proto_metadata_store_pb.Execution.toObject, includeInstance),
    nextPageToken: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionsResponse}
 */
proto.ml_metadata.GetExecutionsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionsResponse;
  return proto.ml_metadata.GetExecutionsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionsResponse}
 */
proto.ml_metadata.GetExecutionsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Execution;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Execution.deserializeBinaryFromReader);
      msg.addExecutions(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Execution.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated Execution executions = 1;
 * @return {!Array<!proto.ml_metadata.Execution>}
 */
proto.ml_metadata.GetExecutionsResponse.prototype.getExecutionsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Execution>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Execution, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Execution>} value
 * @return {!proto.ml_metadata.GetExecutionsResponse} returns this
*/
proto.ml_metadata.GetExecutionsResponse.prototype.setExecutionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Execution=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Execution}
 */
proto.ml_metadata.GetExecutionsResponse.prototype.addExecutions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Execution, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetExecutionsResponse} returns this
 */
proto.ml_metadata.GetExecutionsResponse.prototype.clearExecutionsList = function() {
  return this.setExecutionsList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.ml_metadata.GetExecutionsResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionsResponse} returns this
 */
proto.ml_metadata.GetExecutionsResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsResponse} returns this
 */
proto.ml_metadata.GetExecutionsResponse.prototype.clearNextPageToken = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsResponse.prototype.hasNextPageToken = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactTypeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactTypeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    typeVersion: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactTypeRequest}
 */
proto.ml_metadata.GetArtifactTypeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactTypeRequest;
  return proto.ml_metadata.GetArtifactTypeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactTypeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactTypeRequest}
 */
proto.ml_metadata.GetArtifactTypeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeVersion(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactTypeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactTypeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string type_version = 2;
 * @return {string}
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.getTypeVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.setTypeVersion = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.clearTypeVersion = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.hasTypeVersion = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional TransactionOptions transaction_options = 3;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 3));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactTypeRequest} returns this
*/
proto.ml_metadata.GetArtifactTypeRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactTypeRequest} returns this
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactTypeRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactTypeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactTypeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactTypeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactType: (f = msg.getArtifactType()) && ml_metadata_proto_metadata_store_pb.ArtifactType.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactTypeResponse}
 */
proto.ml_metadata.GetArtifactTypeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactTypeResponse;
  return proto.ml_metadata.GetArtifactTypeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactTypeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactTypeResponse}
 */
proto.ml_metadata.GetArtifactTypeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ArtifactType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ArtifactType.deserializeBinaryFromReader);
      msg.setArtifactType(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactTypeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactTypeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactTypeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactType();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ArtifactType.serializeBinaryToWriter
    );
  }
};


/**
 * optional ArtifactType artifact_type = 1;
 * @return {?proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.GetArtifactTypeResponse.prototype.getArtifactType = function() {
  return /** @type{?proto.ml_metadata.ArtifactType} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ArtifactType, 1));
};


/**
 * @param {?proto.ml_metadata.ArtifactType|undefined} value
 * @return {!proto.ml_metadata.GetArtifactTypeResponse} returns this
*/
proto.ml_metadata.GetArtifactTypeResponse.prototype.setArtifactType = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactTypeResponse} returns this
 */
proto.ml_metadata.GetArtifactTypeResponse.prototype.clearArtifactType = function() {
  return this.setArtifactType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactTypeResponse.prototype.hasArtifactType = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactTypesRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactTypesRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactTypesRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypesRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactTypesRequest}
 */
proto.ml_metadata.GetArtifactTypesRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactTypesRequest;
  return proto.ml_metadata.GetArtifactTypesRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactTypesRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactTypesRequest}
 */
proto.ml_metadata.GetArtifactTypesRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactTypesRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactTypesRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactTypesRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypesRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional TransactionOptions transaction_options = 1;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetArtifactTypesRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 1));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactTypesRequest} returns this
*/
proto.ml_metadata.GetArtifactTypesRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactTypesRequest} returns this
 */
proto.ml_metadata.GetArtifactTypesRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactTypesRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactTypesResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactTypesResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactTypesResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactTypesResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypesResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactTypesList: jspb.Message.toObjectList(msg.getArtifactTypesList(),
    ml_metadata_proto_metadata_store_pb.ArtifactType.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactTypesResponse}
 */
proto.ml_metadata.GetArtifactTypesResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactTypesResponse;
  return proto.ml_metadata.GetArtifactTypesResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactTypesResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactTypesResponse}
 */
proto.ml_metadata.GetArtifactTypesResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ArtifactType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ArtifactType.deserializeBinaryFromReader);
      msg.addArtifactTypes(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactTypesResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactTypesResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactTypesResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypesResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ArtifactType.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ArtifactType artifact_types = 1;
 * @return {!Array<!proto.ml_metadata.ArtifactType>}
 */
proto.ml_metadata.GetArtifactTypesResponse.prototype.getArtifactTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ArtifactType>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ArtifactType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ArtifactType>} value
 * @return {!proto.ml_metadata.GetArtifactTypesResponse} returns this
*/
proto.ml_metadata.GetArtifactTypesResponse.prototype.setArtifactTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ArtifactType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.GetArtifactTypesResponse.prototype.addArtifactTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ArtifactType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactTypesResponse} returns this
 */
proto.ml_metadata.GetArtifactTypesResponse.prototype.clearArtifactTypesList = function() {
  return this.setArtifactTypesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionTypesRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionTypesRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionTypesRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypesRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionTypesRequest}
 */
proto.ml_metadata.GetExecutionTypesRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionTypesRequest;
  return proto.ml_metadata.GetExecutionTypesRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionTypesRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionTypesRequest}
 */
proto.ml_metadata.GetExecutionTypesRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionTypesRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionTypesRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionTypesRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypesRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional TransactionOptions transaction_options = 1;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetExecutionTypesRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 1));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionTypesRequest} returns this
*/
proto.ml_metadata.GetExecutionTypesRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionTypesRequest} returns this
 */
proto.ml_metadata.GetExecutionTypesRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionTypesRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetExecutionTypesResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionTypesResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionTypesResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionTypesResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypesResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionTypesList: jspb.Message.toObjectList(msg.getExecutionTypesList(),
    ml_metadata_proto_metadata_store_pb.ExecutionType.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionTypesResponse}
 */
proto.ml_metadata.GetExecutionTypesResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionTypesResponse;
  return proto.ml_metadata.GetExecutionTypesResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionTypesResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionTypesResponse}
 */
proto.ml_metadata.GetExecutionTypesResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ExecutionType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ExecutionType.deserializeBinaryFromReader);
      msg.addExecutionTypes(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionTypesResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionTypesResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionTypesResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypesResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ExecutionType.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ExecutionType execution_types = 1;
 * @return {!Array<!proto.ml_metadata.ExecutionType>}
 */
proto.ml_metadata.GetExecutionTypesResponse.prototype.getExecutionTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ExecutionType>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ExecutionType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ExecutionType>} value
 * @return {!proto.ml_metadata.GetExecutionTypesResponse} returns this
*/
proto.ml_metadata.GetExecutionTypesResponse.prototype.setExecutionTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ExecutionType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ExecutionType}
 */
proto.ml_metadata.GetExecutionTypesResponse.prototype.addExecutionTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ExecutionType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetExecutionTypesResponse} returns this
 */
proto.ml_metadata.GetExecutionTypesResponse.prototype.clearExecutionTypesList = function() {
  return this.setExecutionTypesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextTypesRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextTypesRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextTypesRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypesRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextTypesRequest}
 */
proto.ml_metadata.GetContextTypesRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextTypesRequest;
  return proto.ml_metadata.GetContextTypesRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextTypesRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextTypesRequest}
 */
proto.ml_metadata.GetContextTypesRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextTypesRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextTypesRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextTypesRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypesRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional TransactionOptions transaction_options = 1;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetContextTypesRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 1));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextTypesRequest} returns this
*/
proto.ml_metadata.GetContextTypesRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextTypesRequest} returns this
 */
proto.ml_metadata.GetContextTypesRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextTypesRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetContextTypesResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextTypesResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextTypesResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextTypesResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypesResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextTypesList: jspb.Message.toObjectList(msg.getContextTypesList(),
    ml_metadata_proto_metadata_store_pb.ContextType.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextTypesResponse}
 */
proto.ml_metadata.GetContextTypesResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextTypesResponse;
  return proto.ml_metadata.GetContextTypesResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextTypesResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextTypesResponse}
 */
proto.ml_metadata.GetContextTypesResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ContextType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ContextType.deserializeBinaryFromReader);
      msg.addContextTypes(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextTypesResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextTypesResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextTypesResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypesResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ContextType.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ContextType context_types = 1;
 * @return {!Array<!proto.ml_metadata.ContextType>}
 */
proto.ml_metadata.GetContextTypesResponse.prototype.getContextTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ContextType>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ContextType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ContextType>} value
 * @return {!proto.ml_metadata.GetContextTypesResponse} returns this
*/
proto.ml_metadata.GetContextTypesResponse.prototype.setContextTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ContextType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ContextType}
 */
proto.ml_metadata.GetContextTypesResponse.prototype.addContextTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ContextType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetContextTypesResponse} returns this
 */
proto.ml_metadata.GetContextTypesResponse.prototype.clearContextTypesList = function() {
  return this.setContextTypesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionsByTypeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionsByTypeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByTypeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    typeVersion: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    options: (f = msg.getOptions()) && ml_metadata_proto_metadata_store_pb.ListOperationOptions.toObject(includeInstance, f),
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionsByTypeRequest;
  return proto.ml_metadata.GetExecutionsByTypeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionsByTypeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeVersion(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.ListOperationOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ListOperationOptions.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    case 4:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionsByTypeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionsByTypeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByTypeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string type_version = 2;
 * @return {string}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.getTypeVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.setTypeVersion = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.clearTypeVersion = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.hasTypeVersion = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional ListOperationOptions options = 3;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ListOperationOptions, 3));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest} returns this
*/
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional TransactionOptions transaction_options = 4;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 4));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest} returns this
*/
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByTypeRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 4) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetExecutionsByTypeResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionsByTypeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionsByTypeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByTypeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionsList: jspb.Message.toObjectList(msg.getExecutionsList(),
    ml_metadata_proto_metadata_store_pb.Execution.toObject, includeInstance),
    nextPageToken: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionsByTypeResponse}
 */
proto.ml_metadata.GetExecutionsByTypeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionsByTypeResponse;
  return proto.ml_metadata.GetExecutionsByTypeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionsByTypeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionsByTypeResponse}
 */
proto.ml_metadata.GetExecutionsByTypeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Execution;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Execution.deserializeBinaryFromReader);
      msg.addExecutions(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionsByTypeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionsByTypeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByTypeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Execution.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated Execution executions = 1;
 * @return {!Array<!proto.ml_metadata.Execution>}
 */
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.getExecutionsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Execution>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Execution, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Execution>} value
 * @return {!proto.ml_metadata.GetExecutionsByTypeResponse} returns this
*/
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.setExecutionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Execution=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Execution}
 */
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.addExecutions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Execution, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetExecutionsByTypeResponse} returns this
 */
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.clearExecutionsList = function() {
  return this.setExecutionsList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionsByTypeResponse} returns this
 */
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByTypeResponse} returns this
 */
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.clearNextPageToken = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByTypeResponse.prototype.hasNextPageToken = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionByTypeAndNameRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    typeVersion: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    executionName: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionByTypeAndNameRequest;
  return proto.ml_metadata.GetExecutionByTypeAndNameRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeVersion(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setExecutionName(value);
      break;
    case 4:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionByTypeAndNameRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string type_version = 3;
 * @return {string}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.getTypeVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.setTypeVersion = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.clearTypeVersion = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.hasTypeVersion = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string execution_name = 2;
 * @return {string}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.getExecutionName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.setExecutionName = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.clearExecutionName = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.hasExecutionName = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional TransactionOptions transaction_options = 4;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 4));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} returns this
*/
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionByTypeAndNameRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 4) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionByTypeAndNameResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionByTypeAndNameResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    execution: (f = msg.getExecution()) && ml_metadata_proto_metadata_store_pb.Execution.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameResponse}
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionByTypeAndNameResponse;
  return proto.ml_metadata.GetExecutionByTypeAndNameResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionByTypeAndNameResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameResponse}
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Execution;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Execution.deserializeBinaryFromReader);
      msg.setExecution(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionByTypeAndNameResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionByTypeAndNameResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecution();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Execution.serializeBinaryToWriter
    );
  }
};


/**
 * optional Execution execution = 1;
 * @return {?proto.ml_metadata.Execution}
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse.prototype.getExecution = function() {
  return /** @type{?proto.ml_metadata.Execution} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.Execution, 1));
};


/**
 * @param {?proto.ml_metadata.Execution|undefined} value
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameResponse} returns this
*/
proto.ml_metadata.GetExecutionByTypeAndNameResponse.prototype.setExecution = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionByTypeAndNameResponse} returns this
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse.prototype.clearExecution = function() {
  return this.setExecution(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionByTypeAndNameResponse.prototype.hasExecution = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetExecutionsByIDRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionsByIDRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionsByIDRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionsByIDRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByIDRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionsByIDRequest}
 */
proto.ml_metadata.GetExecutionsByIDRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionsByIDRequest;
  return proto.ml_metadata.GetExecutionsByIDRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionsByIDRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionsByIDRequest}
 */
proto.ml_metadata.GetExecutionsByIDRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addExecutionIds(values[i]);
      }
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionsByIDRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionsByIDRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionsByIDRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByIDRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated int64 execution_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.GetExecutionsByIDRequest.prototype.getExecutionIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.GetExecutionsByIDRequest} returns this
 */
proto.ml_metadata.GetExecutionsByIDRequest.prototype.setExecutionIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.GetExecutionsByIDRequest} returns this
 */
proto.ml_metadata.GetExecutionsByIDRequest.prototype.addExecutionIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetExecutionsByIDRequest} returns this
 */
proto.ml_metadata.GetExecutionsByIDRequest.prototype.clearExecutionIdsList = function() {
  return this.setExecutionIdsList([]);
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetExecutionsByIDRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionsByIDRequest} returns this
*/
proto.ml_metadata.GetExecutionsByIDRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByIDRequest} returns this
 */
proto.ml_metadata.GetExecutionsByIDRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByIDRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetExecutionsByIDResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionsByIDResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionsByIDResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionsByIDResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByIDResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionsList: jspb.Message.toObjectList(msg.getExecutionsList(),
    ml_metadata_proto_metadata_store_pb.Execution.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionsByIDResponse}
 */
proto.ml_metadata.GetExecutionsByIDResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionsByIDResponse;
  return proto.ml_metadata.GetExecutionsByIDResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionsByIDResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionsByIDResponse}
 */
proto.ml_metadata.GetExecutionsByIDResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Execution;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Execution.deserializeBinaryFromReader);
      msg.addExecutions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionsByIDResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionsByIDResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionsByIDResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByIDResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Execution.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Execution executions = 1;
 * @return {!Array<!proto.ml_metadata.Execution>}
 */
proto.ml_metadata.GetExecutionsByIDResponse.prototype.getExecutionsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Execution>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Execution, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Execution>} value
 * @return {!proto.ml_metadata.GetExecutionsByIDResponse} returns this
*/
proto.ml_metadata.GetExecutionsByIDResponse.prototype.setExecutionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Execution=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Execution}
 */
proto.ml_metadata.GetExecutionsByIDResponse.prototype.addExecutions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Execution, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetExecutionsByIDResponse} returns this
 */
proto.ml_metadata.GetExecutionsByIDResponse.prototype.clearExecutionsList = function() {
  return this.setExecutionsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionTypeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionTypeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    typeVersion: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionTypeRequest}
 */
proto.ml_metadata.GetExecutionTypeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionTypeRequest;
  return proto.ml_metadata.GetExecutionTypeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionTypeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionTypeRequest}
 */
proto.ml_metadata.GetExecutionTypeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeVersion(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionTypeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionTypeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string type_version = 2;
 * @return {string}
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.getTypeVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.setTypeVersion = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.clearTypeVersion = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.hasTypeVersion = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional TransactionOptions transaction_options = 3;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 3));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionTypeRequest} returns this
*/
proto.ml_metadata.GetExecutionTypeRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionTypeRequest} returns this
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionTypeRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionTypeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionTypeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionTypeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionType: (f = msg.getExecutionType()) && ml_metadata_proto_metadata_store_pb.ExecutionType.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionTypeResponse}
 */
proto.ml_metadata.GetExecutionTypeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionTypeResponse;
  return proto.ml_metadata.GetExecutionTypeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionTypeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionTypeResponse}
 */
proto.ml_metadata.GetExecutionTypeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ExecutionType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ExecutionType.deserializeBinaryFromReader);
      msg.setExecutionType(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionTypeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionTypeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionTypeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionType();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ExecutionType.serializeBinaryToWriter
    );
  }
};


/**
 * optional ExecutionType execution_type = 1;
 * @return {?proto.ml_metadata.ExecutionType}
 */
proto.ml_metadata.GetExecutionTypeResponse.prototype.getExecutionType = function() {
  return /** @type{?proto.ml_metadata.ExecutionType} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ExecutionType, 1));
};


/**
 * @param {?proto.ml_metadata.ExecutionType|undefined} value
 * @return {!proto.ml_metadata.GetExecutionTypeResponse} returns this
*/
proto.ml_metadata.GetExecutionTypeResponse.prototype.setExecutionType = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionTypeResponse} returns this
 */
proto.ml_metadata.GetExecutionTypeResponse.prototype.clearExecutionType = function() {
  return this.setExecutionType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionTypeResponse.prototype.hasExecutionType = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetEventsByExecutionIDsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetEventsByExecutionIDsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsRequest}
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetEventsByExecutionIDsRequest;
  return proto.ml_metadata.GetEventsByExecutionIDsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetEventsByExecutionIDsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsRequest}
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addExecutionIds(values[i]);
      }
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetEventsByExecutionIDsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetEventsByExecutionIDsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated int64 execution_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.getExecutionIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsRequest} returns this
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.setExecutionIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsRequest} returns this
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.addExecutionIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsRequest} returns this
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.clearExecutionIdsList = function() {
  return this.setExecutionIdsList([]);
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsRequest} returns this
*/
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsRequest} returns this
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetEventsByExecutionIDsRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetEventsByExecutionIDsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetEventsByExecutionIDsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    eventsList: jspb.Message.toObjectList(msg.getEventsList(),
    ml_metadata_proto_metadata_store_pb.Event.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsResponse}
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetEventsByExecutionIDsResponse;
  return proto.ml_metadata.GetEventsByExecutionIDsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetEventsByExecutionIDsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsResponse}
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Event;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Event.deserializeBinaryFromReader);
      msg.addEvents(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetEventsByExecutionIDsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetEventsByExecutionIDsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEventsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Event.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Event events = 1;
 * @return {!Array<!proto.ml_metadata.Event>}
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.prototype.getEventsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Event>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Event, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Event>} value
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsResponse} returns this
*/
proto.ml_metadata.GetEventsByExecutionIDsResponse.prototype.setEventsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Event=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Event}
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.prototype.addEvents = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Event, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetEventsByExecutionIDsResponse} returns this
 */
proto.ml_metadata.GetEventsByExecutionIDsResponse.prototype.clearEventsList = function() {
  return this.setEventsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetEventsByArtifactIDsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetEventsByArtifactIDsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsRequest}
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetEventsByArtifactIDsRequest;
  return proto.ml_metadata.GetEventsByArtifactIDsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetEventsByArtifactIDsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsRequest}
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addArtifactIds(values[i]);
      }
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetEventsByArtifactIDsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetEventsByArtifactIDsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated int64 artifact_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.getArtifactIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsRequest} returns this
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.setArtifactIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsRequest} returns this
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.addArtifactIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsRequest} returns this
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.clearArtifactIdsList = function() {
  return this.setArtifactIdsList([]);
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsRequest} returns this
*/
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsRequest} returns this
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetEventsByArtifactIDsRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetEventsByArtifactIDsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetEventsByArtifactIDsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    eventsList: jspb.Message.toObjectList(msg.getEventsList(),
    ml_metadata_proto_metadata_store_pb.Event.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsResponse}
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetEventsByArtifactIDsResponse;
  return proto.ml_metadata.GetEventsByArtifactIDsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetEventsByArtifactIDsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsResponse}
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Event;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Event.deserializeBinaryFromReader);
      msg.addEvents(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetEventsByArtifactIDsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetEventsByArtifactIDsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEventsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Event.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Event events = 1;
 * @return {!Array<!proto.ml_metadata.Event>}
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.prototype.getEventsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Event>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Event, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Event>} value
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsResponse} returns this
*/
proto.ml_metadata.GetEventsByArtifactIDsResponse.prototype.setEventsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Event=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Event}
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.prototype.addEvents = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Event, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetEventsByArtifactIDsResponse} returns this
 */
proto.ml_metadata.GetEventsByArtifactIDsResponse.prototype.clearEventsList = function() {
  return this.setEventsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactTypesByIDRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactTypesByIDRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactTypesByIDRequest}
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactTypesByIDRequest;
  return proto.ml_metadata.GetArtifactTypesByIDRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactTypesByIDRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactTypesByIDRequest}
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addTypeIds(values[i]);
      }
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactTypesByIDRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactTypesByIDRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTypeIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated int64 type_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.getTypeIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.GetArtifactTypesByIDRequest} returns this
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.setTypeIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.GetArtifactTypesByIDRequest} returns this
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.addTypeIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactTypesByIDRequest} returns this
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.clearTypeIdsList = function() {
  return this.setTypeIdsList([]);
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactTypesByIDRequest} returns this
*/
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactTypesByIDRequest} returns this
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactTypesByIDRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactTypesByIDResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactTypesByIDResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactTypesList: jspb.Message.toObjectList(msg.getArtifactTypesList(),
    ml_metadata_proto_metadata_store_pb.ArtifactType.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactTypesByIDResponse}
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactTypesByIDResponse;
  return proto.ml_metadata.GetArtifactTypesByIDResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactTypesByIDResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactTypesByIDResponse}
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ArtifactType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ArtifactType.deserializeBinaryFromReader);
      msg.addArtifactTypes(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactTypesByIDResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactTypesByIDResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ArtifactType.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ArtifactType artifact_types = 1;
 * @return {!Array<!proto.ml_metadata.ArtifactType>}
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.prototype.getArtifactTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ArtifactType>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ArtifactType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ArtifactType>} value
 * @return {!proto.ml_metadata.GetArtifactTypesByIDResponse} returns this
*/
proto.ml_metadata.GetArtifactTypesByIDResponse.prototype.setArtifactTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ArtifactType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.prototype.addArtifactTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ArtifactType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactTypesByIDResponse} returns this
 */
proto.ml_metadata.GetArtifactTypesByIDResponse.prototype.clearArtifactTypesList = function() {
  return this.setArtifactTypesList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionTypesByIDRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionTypesByIDRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionTypesByIDRequest}
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionTypesByIDRequest;
  return proto.ml_metadata.GetExecutionTypesByIDRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionTypesByIDRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionTypesByIDRequest}
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addTypeIds(values[i]);
      }
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionTypesByIDRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionTypesByIDRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTypeIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated int64 type_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.getTypeIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.GetExecutionTypesByIDRequest} returns this
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.setTypeIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.GetExecutionTypesByIDRequest} returns this
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.addTypeIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetExecutionTypesByIDRequest} returns this
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.clearTypeIdsList = function() {
  return this.setTypeIdsList([]);
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionTypesByIDRequest} returns this
*/
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionTypesByIDRequest} returns this
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionTypesByIDRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionTypesByIDResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionTypesByIDResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionTypesList: jspb.Message.toObjectList(msg.getExecutionTypesList(),
    ml_metadata_proto_metadata_store_pb.ExecutionType.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionTypesByIDResponse}
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionTypesByIDResponse;
  return proto.ml_metadata.GetExecutionTypesByIDResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionTypesByIDResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionTypesByIDResponse}
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ExecutionType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ExecutionType.deserializeBinaryFromReader);
      msg.addExecutionTypes(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionTypesByIDResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionTypesByIDResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ExecutionType.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ExecutionType execution_types = 1;
 * @return {!Array<!proto.ml_metadata.ExecutionType>}
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.prototype.getExecutionTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ExecutionType>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ExecutionType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ExecutionType>} value
 * @return {!proto.ml_metadata.GetExecutionTypesByIDResponse} returns this
*/
proto.ml_metadata.GetExecutionTypesByIDResponse.prototype.setExecutionTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ExecutionType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ExecutionType}
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.prototype.addExecutionTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ExecutionType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetExecutionTypesByIDResponse} returns this
 */
proto.ml_metadata.GetExecutionTypesByIDResponse.prototype.clearExecutionTypesList = function() {
  return this.setExecutionTypesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextTypeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextTypeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextTypeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    typeVersion: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextTypeRequest}
 */
proto.ml_metadata.GetContextTypeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextTypeRequest;
  return proto.ml_metadata.GetContextTypeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextTypeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextTypeRequest}
 */
proto.ml_metadata.GetContextTypeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeVersion(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextTypeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextTypeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextTypeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.GetContextTypeRequest.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetContextTypeRequest} returns this
 */
proto.ml_metadata.GetContextTypeRequest.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextTypeRequest} returns this
 */
proto.ml_metadata.GetContextTypeRequest.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextTypeRequest.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string type_version = 2;
 * @return {string}
 */
proto.ml_metadata.GetContextTypeRequest.prototype.getTypeVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetContextTypeRequest} returns this
 */
proto.ml_metadata.GetContextTypeRequest.prototype.setTypeVersion = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextTypeRequest} returns this
 */
proto.ml_metadata.GetContextTypeRequest.prototype.clearTypeVersion = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextTypeRequest.prototype.hasTypeVersion = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional TransactionOptions transaction_options = 3;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetContextTypeRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 3));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextTypeRequest} returns this
*/
proto.ml_metadata.GetContextTypeRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextTypeRequest} returns this
 */
proto.ml_metadata.GetContextTypeRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextTypeRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextTypeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextTypeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextTypeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextType: (f = msg.getContextType()) && ml_metadata_proto_metadata_store_pb.ContextType.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextTypeResponse}
 */
proto.ml_metadata.GetContextTypeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextTypeResponse;
  return proto.ml_metadata.GetContextTypeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextTypeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextTypeResponse}
 */
proto.ml_metadata.GetContextTypeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ContextType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ContextType.deserializeBinaryFromReader);
      msg.setContextType(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextTypeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextTypeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextTypeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextType();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ContextType.serializeBinaryToWriter
    );
  }
};


/**
 * optional ContextType context_type = 1;
 * @return {?proto.ml_metadata.ContextType}
 */
proto.ml_metadata.GetContextTypeResponse.prototype.getContextType = function() {
  return /** @type{?proto.ml_metadata.ContextType} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ContextType, 1));
};


/**
 * @param {?proto.ml_metadata.ContextType|undefined} value
 * @return {!proto.ml_metadata.GetContextTypeResponse} returns this
*/
proto.ml_metadata.GetContextTypeResponse.prototype.setContextType = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextTypeResponse} returns this
 */
proto.ml_metadata.GetContextTypeResponse.prototype.clearContextType = function() {
  return this.setContextType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextTypeResponse.prototype.hasContextType = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetContextTypesByIDRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextTypesByIDRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextTypesByIDRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextTypesByIDRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypesByIDRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextTypesByIDRequest}
 */
proto.ml_metadata.GetContextTypesByIDRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextTypesByIDRequest;
  return proto.ml_metadata.GetContextTypesByIDRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextTypesByIDRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextTypesByIDRequest}
 */
proto.ml_metadata.GetContextTypesByIDRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addTypeIds(values[i]);
      }
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextTypesByIDRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextTypesByIDRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextTypesByIDRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypesByIDRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTypeIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated int64 type_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.GetContextTypesByIDRequest.prototype.getTypeIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.GetContextTypesByIDRequest} returns this
 */
proto.ml_metadata.GetContextTypesByIDRequest.prototype.setTypeIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.GetContextTypesByIDRequest} returns this
 */
proto.ml_metadata.GetContextTypesByIDRequest.prototype.addTypeIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetContextTypesByIDRequest} returns this
 */
proto.ml_metadata.GetContextTypesByIDRequest.prototype.clearTypeIdsList = function() {
  return this.setTypeIdsList([]);
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetContextTypesByIDRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextTypesByIDRequest} returns this
*/
proto.ml_metadata.GetContextTypesByIDRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextTypesByIDRequest} returns this
 */
proto.ml_metadata.GetContextTypesByIDRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextTypesByIDRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetContextTypesByIDResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextTypesByIDResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextTypesByIDResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextTypesByIDResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypesByIDResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextTypesList: jspb.Message.toObjectList(msg.getContextTypesList(),
    ml_metadata_proto_metadata_store_pb.ContextType.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextTypesByIDResponse}
 */
proto.ml_metadata.GetContextTypesByIDResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextTypesByIDResponse;
  return proto.ml_metadata.GetContextTypesByIDResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextTypesByIDResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextTypesByIDResponse}
 */
proto.ml_metadata.GetContextTypesByIDResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ContextType;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ContextType.deserializeBinaryFromReader);
      msg.addContextTypes(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextTypesByIDResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextTypesByIDResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextTypesByIDResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextTypesByIDResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ContextType.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ContextType context_types = 1;
 * @return {!Array<!proto.ml_metadata.ContextType>}
 */
proto.ml_metadata.GetContextTypesByIDResponse.prototype.getContextTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ContextType>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.ContextType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ContextType>} value
 * @return {!proto.ml_metadata.GetContextTypesByIDResponse} returns this
*/
proto.ml_metadata.GetContextTypesByIDResponse.prototype.setContextTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ContextType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ContextType}
 */
proto.ml_metadata.GetContextTypesByIDResponse.prototype.addContextTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ContextType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetContextTypesByIDResponse} returns this
 */
proto.ml_metadata.GetContextTypesByIDResponse.prototype.clearContextTypesList = function() {
  return this.setContextTypesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    options: (f = msg.getOptions()) && ml_metadata_proto_metadata_store_pb.ListOperationOptions.toObject(includeInstance, f),
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsRequest}
 */
proto.ml_metadata.GetContextsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsRequest;
  return proto.ml_metadata.GetContextsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsRequest}
 */
proto.ml_metadata.GetContextsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.ListOperationOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ListOperationOptions.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional ListOperationOptions options = 1;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.GetContextsRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ListOperationOptions, 1));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextsRequest} returns this
*/
proto.ml_metadata.GetContextsRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextsRequest} returns this
 */
proto.ml_metadata.GetContextsRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetContextsRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextsRequest} returns this
*/
proto.ml_metadata.GetContextsRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextsRequest} returns this
 */
proto.ml_metadata.GetContextsRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetContextsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    ml_metadata_proto_metadata_store_pb.Context.toObject, includeInstance),
    nextPageToken: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsResponse}
 */
proto.ml_metadata.GetContextsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsResponse;
  return proto.ml_metadata.GetContextsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsResponse}
 */
proto.ml_metadata.GetContextsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated Context contexts = 1;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.GetContextsResponse.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.GetContextsResponse} returns this
*/
proto.ml_metadata.GetContextsResponse.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.GetContextsResponse.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetContextsResponse} returns this
 */
proto.ml_metadata.GetContextsResponse.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.ml_metadata.GetContextsResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetContextsResponse} returns this
 */
proto.ml_metadata.GetContextsResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextsResponse} returns this
 */
proto.ml_metadata.GetContextsResponse.prototype.clearNextPageToken = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsResponse.prototype.hasNextPageToken = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsByTypeRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsByTypeRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByTypeRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    options: (f = msg.getOptions()) && ml_metadata_proto_metadata_store_pb.ListOperationOptions.toObject(includeInstance, f),
    typeVersion: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsByTypeRequest}
 */
proto.ml_metadata.GetContextsByTypeRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsByTypeRequest;
  return proto.ml_metadata.GetContextsByTypeRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsByTypeRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsByTypeRequest}
 */
proto.ml_metadata.GetContextsByTypeRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeName(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.ListOperationOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ListOperationOptions.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeVersion(value);
      break;
    case 4:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsByTypeRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsByTypeRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByTypeRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetContextsByTypeRequest} returns this
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByTypeRequest} returns this
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ListOperationOptions options = 2;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ListOperationOptions, 2));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextsByTypeRequest} returns this
*/
proto.ml_metadata.GetContextsByTypeRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByTypeRequest} returns this
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string type_version = 3;
 * @return {string}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.getTypeVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetContextsByTypeRequest} returns this
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.setTypeVersion = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByTypeRequest} returns this
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.clearTypeVersion = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.hasTypeVersion = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional TransactionOptions transaction_options = 4;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 4));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextsByTypeRequest} returns this
*/
proto.ml_metadata.GetContextsByTypeRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByTypeRequest} returns this
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByTypeRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 4) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetContextsByTypeResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsByTypeResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsByTypeResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsByTypeResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByTypeResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    ml_metadata_proto_metadata_store_pb.Context.toObject, includeInstance),
    nextPageToken: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsByTypeResponse}
 */
proto.ml_metadata.GetContextsByTypeResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsByTypeResponse;
  return proto.ml_metadata.GetContextsByTypeResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsByTypeResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsByTypeResponse}
 */
proto.ml_metadata.GetContextsByTypeResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsByTypeResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsByTypeResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsByTypeResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByTypeResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated Context contexts = 1;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.GetContextsByTypeResponse.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.GetContextsByTypeResponse} returns this
*/
proto.ml_metadata.GetContextsByTypeResponse.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.GetContextsByTypeResponse.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetContextsByTypeResponse} returns this
 */
proto.ml_metadata.GetContextsByTypeResponse.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.ml_metadata.GetContextsByTypeResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetContextsByTypeResponse} returns this
 */
proto.ml_metadata.GetContextsByTypeResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByTypeResponse} returns this
 */
proto.ml_metadata.GetContextsByTypeResponse.prototype.clearNextPageToken = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByTypeResponse.prototype.hasNextPageToken = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextByTypeAndNameRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextByTypeAndNameRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    typeVersion: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    contextName: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextByTypeAndNameRequest;
  return proto.ml_metadata.GetContextByTypeAndNameRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextByTypeAndNameRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setTypeVersion(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setContextName(value);
      break;
    case 4:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextByTypeAndNameRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextByTypeAndNameRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string type_version = 3;
 * @return {string}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.getTypeVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.setTypeVersion = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.clearTypeVersion = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.hasTypeVersion = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string context_name = 2;
 * @return {string}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.getContextName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.setContextName = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.clearContextName = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.hasContextName = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional TransactionOptions transaction_options = 4;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 4));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest} returns this
*/
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextByTypeAndNameRequest} returns this
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextByTypeAndNameRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 4) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextByTypeAndNameResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextByTypeAndNameResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextByTypeAndNameResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextByTypeAndNameResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    context: (f = msg.getContext()) && ml_metadata_proto_metadata_store_pb.Context.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextByTypeAndNameResponse}
 */
proto.ml_metadata.GetContextByTypeAndNameResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextByTypeAndNameResponse;
  return proto.ml_metadata.GetContextByTypeAndNameResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextByTypeAndNameResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextByTypeAndNameResponse}
 */
proto.ml_metadata.GetContextByTypeAndNameResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.setContext(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextByTypeAndNameResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextByTypeAndNameResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextByTypeAndNameResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextByTypeAndNameResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContext();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
};


/**
 * optional Context context = 1;
 * @return {?proto.ml_metadata.Context}
 */
proto.ml_metadata.GetContextByTypeAndNameResponse.prototype.getContext = function() {
  return /** @type{?proto.ml_metadata.Context} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 1));
};


/**
 * @param {?proto.ml_metadata.Context|undefined} value
 * @return {!proto.ml_metadata.GetContextByTypeAndNameResponse} returns this
*/
proto.ml_metadata.GetContextByTypeAndNameResponse.prototype.setContext = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextByTypeAndNameResponse} returns this
 */
proto.ml_metadata.GetContextByTypeAndNameResponse.prototype.clearContext = function() {
  return this.setContext(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextByTypeAndNameResponse.prototype.hasContext = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetContextsByIDRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsByIDRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsByIDRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsByIDRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByIDRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsByIDRequest}
 */
proto.ml_metadata.GetContextsByIDRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsByIDRequest;
  return proto.ml_metadata.GetContextsByIDRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsByIDRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsByIDRequest}
 */
proto.ml_metadata.GetContextsByIDRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addContextIds(values[i]);
      }
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsByIDRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsByIDRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsByIDRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByIDRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated int64 context_ids = 1;
 * @return {!Array<number>}
 */
proto.ml_metadata.GetContextsByIDRequest.prototype.getContextIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.GetContextsByIDRequest} returns this
 */
proto.ml_metadata.GetContextsByIDRequest.prototype.setContextIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.GetContextsByIDRequest} returns this
 */
proto.ml_metadata.GetContextsByIDRequest.prototype.addContextIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetContextsByIDRequest} returns this
 */
proto.ml_metadata.GetContextsByIDRequest.prototype.clearContextIdsList = function() {
  return this.setContextIdsList([]);
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetContextsByIDRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextsByIDRequest} returns this
*/
proto.ml_metadata.GetContextsByIDRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByIDRequest} returns this
 */
proto.ml_metadata.GetContextsByIDRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByIDRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetContextsByIDResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsByIDResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsByIDResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsByIDResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByIDResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    ml_metadata_proto_metadata_store_pb.Context.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsByIDResponse}
 */
proto.ml_metadata.GetContextsByIDResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsByIDResponse;
  return proto.ml_metadata.GetContextsByIDResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsByIDResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsByIDResponse}
 */
proto.ml_metadata.GetContextsByIDResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsByIDResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsByIDResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsByIDResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByIDResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Context contexts = 1;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.GetContextsByIDResponse.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.GetContextsByIDResponse} returns this
*/
proto.ml_metadata.GetContextsByIDResponse.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.GetContextsByIDResponse.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetContextsByIDResponse} returns this
 */
proto.ml_metadata.GetContextsByIDResponse.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsByArtifactRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsByArtifactRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsByArtifactRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByArtifactRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsByArtifactRequest}
 */
proto.ml_metadata.GetContextsByArtifactRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsByArtifactRequest;
  return proto.ml_metadata.GetContextsByArtifactRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsByArtifactRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsByArtifactRequest}
 */
proto.ml_metadata.GetContextsByArtifactRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setArtifactId(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsByArtifactRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsByArtifactRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsByArtifactRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByArtifactRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional int64 artifact_id = 1;
 * @return {number}
 */
proto.ml_metadata.GetContextsByArtifactRequest.prototype.getArtifactId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.GetContextsByArtifactRequest} returns this
 */
proto.ml_metadata.GetContextsByArtifactRequest.prototype.setArtifactId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByArtifactRequest} returns this
 */
proto.ml_metadata.GetContextsByArtifactRequest.prototype.clearArtifactId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByArtifactRequest.prototype.hasArtifactId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetContextsByArtifactRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextsByArtifactRequest} returns this
*/
proto.ml_metadata.GetContextsByArtifactRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByArtifactRequest} returns this
 */
proto.ml_metadata.GetContextsByArtifactRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByArtifactRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetContextsByArtifactResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsByArtifactResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsByArtifactResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsByArtifactResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByArtifactResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    ml_metadata_proto_metadata_store_pb.Context.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsByArtifactResponse}
 */
proto.ml_metadata.GetContextsByArtifactResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsByArtifactResponse;
  return proto.ml_metadata.GetContextsByArtifactResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsByArtifactResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsByArtifactResponse}
 */
proto.ml_metadata.GetContextsByArtifactResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsByArtifactResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsByArtifactResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsByArtifactResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByArtifactResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Context contexts = 1;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.GetContextsByArtifactResponse.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.GetContextsByArtifactResponse} returns this
*/
proto.ml_metadata.GetContextsByArtifactResponse.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.GetContextsByArtifactResponse.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetContextsByArtifactResponse} returns this
 */
proto.ml_metadata.GetContextsByArtifactResponse.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsByExecutionRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsByExecutionRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsByExecutionRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByExecutionRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsByExecutionRequest}
 */
proto.ml_metadata.GetContextsByExecutionRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsByExecutionRequest;
  return proto.ml_metadata.GetContextsByExecutionRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsByExecutionRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsByExecutionRequest}
 */
proto.ml_metadata.GetContextsByExecutionRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setExecutionId(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsByExecutionRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsByExecutionRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsByExecutionRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByExecutionRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional int64 execution_id = 1;
 * @return {number}
 */
proto.ml_metadata.GetContextsByExecutionRequest.prototype.getExecutionId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.GetContextsByExecutionRequest} returns this
 */
proto.ml_metadata.GetContextsByExecutionRequest.prototype.setExecutionId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByExecutionRequest} returns this
 */
proto.ml_metadata.GetContextsByExecutionRequest.prototype.clearExecutionId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByExecutionRequest.prototype.hasExecutionId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetContextsByExecutionRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetContextsByExecutionRequest} returns this
*/
proto.ml_metadata.GetContextsByExecutionRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetContextsByExecutionRequest} returns this
 */
proto.ml_metadata.GetContextsByExecutionRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetContextsByExecutionRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetContextsByExecutionResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetContextsByExecutionResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetContextsByExecutionResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetContextsByExecutionResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByExecutionResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    ml_metadata_proto_metadata_store_pb.Context.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetContextsByExecutionResponse}
 */
proto.ml_metadata.GetContextsByExecutionResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetContextsByExecutionResponse;
  return proto.ml_metadata.GetContextsByExecutionResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetContextsByExecutionResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetContextsByExecutionResponse}
 */
proto.ml_metadata.GetContextsByExecutionResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetContextsByExecutionResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetContextsByExecutionResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetContextsByExecutionResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetContextsByExecutionResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Context contexts = 1;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.GetContextsByExecutionResponse.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.GetContextsByExecutionResponse} returns this
*/
proto.ml_metadata.GetContextsByExecutionResponse.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.GetContextsByExecutionResponse.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetContextsByExecutionResponse} returns this
 */
proto.ml_metadata.GetContextsByExecutionResponse.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetParentContextsByContextRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetParentContextsByContextRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetParentContextsByContextRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetParentContextsByContextRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetParentContextsByContextRequest}
 */
proto.ml_metadata.GetParentContextsByContextRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetParentContextsByContextRequest;
  return proto.ml_metadata.GetParentContextsByContextRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetParentContextsByContextRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetParentContextsByContextRequest}
 */
proto.ml_metadata.GetParentContextsByContextRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setContextId(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetParentContextsByContextRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetParentContextsByContextRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetParentContextsByContextRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetParentContextsByContextRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional int64 context_id = 1;
 * @return {number}
 */
proto.ml_metadata.GetParentContextsByContextRequest.prototype.getContextId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.GetParentContextsByContextRequest} returns this
 */
proto.ml_metadata.GetParentContextsByContextRequest.prototype.setContextId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetParentContextsByContextRequest} returns this
 */
proto.ml_metadata.GetParentContextsByContextRequest.prototype.clearContextId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetParentContextsByContextRequest.prototype.hasContextId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetParentContextsByContextRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetParentContextsByContextRequest} returns this
*/
proto.ml_metadata.GetParentContextsByContextRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetParentContextsByContextRequest} returns this
 */
proto.ml_metadata.GetParentContextsByContextRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetParentContextsByContextRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetParentContextsByContextResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetParentContextsByContextResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetParentContextsByContextResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetParentContextsByContextResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetParentContextsByContextResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    ml_metadata_proto_metadata_store_pb.Context.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetParentContextsByContextResponse}
 */
proto.ml_metadata.GetParentContextsByContextResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetParentContextsByContextResponse;
  return proto.ml_metadata.GetParentContextsByContextResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetParentContextsByContextResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetParentContextsByContextResponse}
 */
proto.ml_metadata.GetParentContextsByContextResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetParentContextsByContextResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetParentContextsByContextResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetParentContextsByContextResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetParentContextsByContextResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Context contexts = 1;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.GetParentContextsByContextResponse.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.GetParentContextsByContextResponse} returns this
*/
proto.ml_metadata.GetParentContextsByContextResponse.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.GetParentContextsByContextResponse.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetParentContextsByContextResponse} returns this
 */
proto.ml_metadata.GetParentContextsByContextResponse.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetChildrenContextsByContextRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetChildrenContextsByContextRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetChildrenContextsByContextRequest}
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetChildrenContextsByContextRequest;
  return proto.ml_metadata.GetChildrenContextsByContextRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetChildrenContextsByContextRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetChildrenContextsByContextRequest}
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setContextId(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetChildrenContextsByContextRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetChildrenContextsByContextRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional int64 context_id = 1;
 * @return {number}
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.getContextId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.GetChildrenContextsByContextRequest} returns this
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.setContextId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetChildrenContextsByContextRequest} returns this
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.clearContextId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.hasContextId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetChildrenContextsByContextRequest} returns this
*/
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetChildrenContextsByContextRequest} returns this
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetChildrenContextsByContextRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetChildrenContextsByContextResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetChildrenContextsByContextResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    ml_metadata_proto_metadata_store_pb.Context.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetChildrenContextsByContextResponse}
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetChildrenContextsByContextResponse;
  return proto.ml_metadata.GetChildrenContextsByContextResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetChildrenContextsByContextResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetChildrenContextsByContextResponse}
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Context;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetChildrenContextsByContextResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetChildrenContextsByContextResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Context.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Context contexts = 1;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Context, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.GetChildrenContextsByContextResponse} returns this
*/
proto.ml_metadata.GetChildrenContextsByContextResponse.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetChildrenContextsByContextResponse} returns this
 */
proto.ml_metadata.GetChildrenContextsByContextResponse.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsByContextRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsByContextRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByContextRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    options: (f = msg.getOptions()) && ml_metadata_proto_metadata_store_pb.ListOperationOptions.toObject(includeInstance, f),
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsByContextRequest}
 */
proto.ml_metadata.GetArtifactsByContextRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsByContextRequest;
  return proto.ml_metadata.GetArtifactsByContextRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsByContextRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsByContextRequest}
 */
proto.ml_metadata.GetArtifactsByContextRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setContextId(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.ListOperationOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ListOperationOptions.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsByContextRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsByContextRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByContextRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional int64 context_id = 1;
 * @return {number}
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.getContextId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.GetArtifactsByContextRequest} returns this
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.setContextId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByContextRequest} returns this
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.clearContextId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.hasContextId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ListOperationOptions options = 2;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ListOperationOptions, 2));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactsByContextRequest} returns this
*/
proto.ml_metadata.GetArtifactsByContextRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByContextRequest} returns this
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional TransactionOptions transaction_options = 3;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 3));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetArtifactsByContextRequest} returns this
*/
proto.ml_metadata.GetArtifactsByContextRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByContextRequest} returns this
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByContextRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetArtifactsByContextResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetArtifactsByContextResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetArtifactsByContextResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetArtifactsByContextResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByContextResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsList: jspb.Message.toObjectList(msg.getArtifactsList(),
    ml_metadata_proto_metadata_store_pb.Artifact.toObject, includeInstance),
    nextPageToken: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetArtifactsByContextResponse}
 */
proto.ml_metadata.GetArtifactsByContextResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetArtifactsByContextResponse;
  return proto.ml_metadata.GetArtifactsByContextResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetArtifactsByContextResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetArtifactsByContextResponse}
 */
proto.ml_metadata.GetArtifactsByContextResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Artifact;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Artifact.deserializeBinaryFromReader);
      msg.addArtifacts(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetArtifactsByContextResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetArtifactsByContextResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetArtifactsByContextResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetArtifactsByContextResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Artifact.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated Artifact artifacts = 1;
 * @return {!Array<!proto.ml_metadata.Artifact>}
 */
proto.ml_metadata.GetArtifactsByContextResponse.prototype.getArtifactsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Artifact>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Artifact, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Artifact>} value
 * @return {!proto.ml_metadata.GetArtifactsByContextResponse} returns this
*/
proto.ml_metadata.GetArtifactsByContextResponse.prototype.setArtifactsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Artifact=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Artifact}
 */
proto.ml_metadata.GetArtifactsByContextResponse.prototype.addArtifacts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Artifact, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetArtifactsByContextResponse} returns this
 */
proto.ml_metadata.GetArtifactsByContextResponse.prototype.clearArtifactsList = function() {
  return this.setArtifactsList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.ml_metadata.GetArtifactsByContextResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetArtifactsByContextResponse} returns this
 */
proto.ml_metadata.GetArtifactsByContextResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetArtifactsByContextResponse} returns this
 */
proto.ml_metadata.GetArtifactsByContextResponse.prototype.clearNextPageToken = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetArtifactsByContextResponse.prototype.hasNextPageToken = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionsByContextRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionsByContextRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByContextRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    contextId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    options: (f = msg.getOptions()) && ml_metadata_proto_metadata_store_pb.ListOperationOptions.toObject(includeInstance, f),
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionsByContextRequest}
 */
proto.ml_metadata.GetExecutionsByContextRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionsByContextRequest;
  return proto.ml_metadata.GetExecutionsByContextRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionsByContextRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionsByContextRequest}
 */
proto.ml_metadata.GetExecutionsByContextRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setContextId(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.ListOperationOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.ListOperationOptions.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionsByContextRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionsByContextRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByContextRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional int64 context_id = 1;
 * @return {number}
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.getContextId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.GetExecutionsByContextRequest} returns this
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.setContextId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByContextRequest} returns this
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.clearContextId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.hasContextId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ListOperationOptions options = 2;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.ListOperationOptions, 2));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionsByContextRequest} returns this
*/
proto.ml_metadata.GetExecutionsByContextRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByContextRequest} returns this
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional TransactionOptions transaction_options = 3;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 3));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionsByContextRequest} returns this
*/
proto.ml_metadata.GetExecutionsByContextRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByContextRequest} returns this
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByContextRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.GetExecutionsByContextResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetExecutionsByContextResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetExecutionsByContextResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByContextResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionsList: jspb.Message.toObjectList(msg.getExecutionsList(),
    ml_metadata_proto_metadata_store_pb.Execution.toObject, includeInstance),
    nextPageToken: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetExecutionsByContextResponse}
 */
proto.ml_metadata.GetExecutionsByContextResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetExecutionsByContextResponse;
  return proto.ml_metadata.GetExecutionsByContextResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetExecutionsByContextResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetExecutionsByContextResponse}
 */
proto.ml_metadata.GetExecutionsByContextResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.Execution;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.Execution.deserializeBinaryFromReader);
      msg.addExecutions(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    case 3:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetExecutionsByContextResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetExecutionsByContextResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetExecutionsByContextResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.Execution.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Execution executions = 1;
 * @return {!Array<!proto.ml_metadata.Execution>}
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.getExecutionsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Execution>} */ (
    jspb.Message.getRepeatedWrapperField(this, ml_metadata_proto_metadata_store_pb.Execution, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Execution>} value
 * @return {!proto.ml_metadata.GetExecutionsByContextResponse} returns this
*/
proto.ml_metadata.GetExecutionsByContextResponse.prototype.setExecutionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Execution=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Execution}
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.addExecutions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Execution, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.GetExecutionsByContextResponse} returns this
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.clearExecutionsList = function() {
  return this.setExecutionsList([]);
};


/**
 * optional string next_page_token = 2;
 * @return {string}
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.GetExecutionsByContextResponse} returns this
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.setNextPageToken = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByContextResponse} returns this
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.clearNextPageToken = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.hasNextPageToken = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional TransactionOptions transaction_options = 3;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 3));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetExecutionsByContextResponse} returns this
*/
proto.ml_metadata.GetExecutionsByContextResponse.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetExecutionsByContextResponse} returns this
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetExecutionsByContextResponse.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetLineageGraphRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetLineageGraphRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetLineageGraphRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetLineageGraphRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    options: (f = msg.getOptions()) && ml_metadata_proto_metadata_store_pb.LineageGraphQueryOptions.toObject(includeInstance, f),
    transactionOptions: (f = msg.getTransactionOptions()) && ml_metadata_proto_metadata_store_pb.TransactionOptions.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetLineageGraphRequest}
 */
proto.ml_metadata.GetLineageGraphRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetLineageGraphRequest;
  return proto.ml_metadata.GetLineageGraphRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetLineageGraphRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetLineageGraphRequest}
 */
proto.ml_metadata.GetLineageGraphRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.LineageGraphQueryOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.LineageGraphQueryOptions.deserializeBinaryFromReader);
      msg.setOptions(value);
      break;
    case 2:
      var value = new ml_metadata_proto_metadata_store_pb.TransactionOptions;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.TransactionOptions.deserializeBinaryFromReader);
      msg.setTransactionOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetLineageGraphRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetLineageGraphRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetLineageGraphRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetLineageGraphRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOptions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.LineageGraphQueryOptions.serializeBinaryToWriter
    );
  }
  f = message.getTransactionOptions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      ml_metadata_proto_metadata_store_pb.TransactionOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional LineageGraphQueryOptions options = 1;
 * @return {?proto.ml_metadata.LineageGraphQueryOptions}
 */
proto.ml_metadata.GetLineageGraphRequest.prototype.getOptions = function() {
  return /** @type{?proto.ml_metadata.LineageGraphQueryOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.LineageGraphQueryOptions, 1));
};


/**
 * @param {?proto.ml_metadata.LineageGraphQueryOptions|undefined} value
 * @return {!proto.ml_metadata.GetLineageGraphRequest} returns this
*/
proto.ml_metadata.GetLineageGraphRequest.prototype.setOptions = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetLineageGraphRequest} returns this
 */
proto.ml_metadata.GetLineageGraphRequest.prototype.clearOptions = function() {
  return this.setOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetLineageGraphRequest.prototype.hasOptions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional TransactionOptions transaction_options = 2;
 * @return {?proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.GetLineageGraphRequest.prototype.getTransactionOptions = function() {
  return /** @type{?proto.ml_metadata.TransactionOptions} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.TransactionOptions, 2));
};


/**
 * @param {?proto.ml_metadata.TransactionOptions|undefined} value
 * @return {!proto.ml_metadata.GetLineageGraphRequest} returns this
*/
proto.ml_metadata.GetLineageGraphRequest.prototype.setTransactionOptions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetLineageGraphRequest} returns this
 */
proto.ml_metadata.GetLineageGraphRequest.prototype.clearTransactionOptions = function() {
  return this.setTransactionOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetLineageGraphRequest.prototype.hasTransactionOptions = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.ml_metadata.GetLineageGraphResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GetLineageGraphResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GetLineageGraphResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetLineageGraphResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    subgraph: (f = msg.getSubgraph()) && ml_metadata_proto_metadata_store_pb.LineageGraph.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.GetLineageGraphResponse}
 */
proto.ml_metadata.GetLineageGraphResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GetLineageGraphResponse;
  return proto.ml_metadata.GetLineageGraphResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GetLineageGraphResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GetLineageGraphResponse}
 */
proto.ml_metadata.GetLineageGraphResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new ml_metadata_proto_metadata_store_pb.LineageGraph;
      reader.readMessage(value,ml_metadata_proto_metadata_store_pb.LineageGraph.deserializeBinaryFromReader);
      msg.setSubgraph(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.GetLineageGraphResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GetLineageGraphResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GetLineageGraphResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GetLineageGraphResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSubgraph();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      ml_metadata_proto_metadata_store_pb.LineageGraph.serializeBinaryToWriter
    );
  }
};


/**
 * optional LineageGraph subgraph = 1;
 * @return {?proto.ml_metadata.LineageGraph}
 */
proto.ml_metadata.GetLineageGraphResponse.prototype.getSubgraph = function() {
  return /** @type{?proto.ml_metadata.LineageGraph} */ (
    jspb.Message.getWrapperField(this, ml_metadata_proto_metadata_store_pb.LineageGraph, 1));
};


/**
 * @param {?proto.ml_metadata.LineageGraph|undefined} value
 * @return {!proto.ml_metadata.GetLineageGraphResponse} returns this
*/
proto.ml_metadata.GetLineageGraphResponse.prototype.setSubgraph = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.GetLineageGraphResponse} returns this
 */
proto.ml_metadata.GetLineageGraphResponse.prototype.clearSubgraph = function() {
  return this.setSubgraph(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GetLineageGraphResponse.prototype.hasSubgraph = function() {
  return jspb.Message.getField(this, 1) != null;
};


goog.object.extend(exports, proto.ml_metadata);
