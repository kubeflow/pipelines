// source: ml_metadata/proto/metadata_store.proto
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

var google_protobuf_struct_pb = require('google-protobuf/google/protobuf/struct_pb.js');
goog.object.extend(proto, google_protobuf_struct_pb);
var google_protobuf_descriptor_pb = require('google-protobuf/google/protobuf/descriptor_pb.js');
goog.object.extend(proto, google_protobuf_descriptor_pb);
goog.exportSymbol('proto.ml_metadata.AnyArtifactStructType', null, global);
goog.exportSymbol('proto.ml_metadata.Artifact', null, global);
goog.exportSymbol('proto.ml_metadata.Artifact.State', null, global);
goog.exportSymbol('proto.ml_metadata.ArtifactStructType', null, global);
goog.exportSymbol('proto.ml_metadata.ArtifactStructType.KindCase', null, global);
goog.exportSymbol('proto.ml_metadata.ArtifactType', null, global);
goog.exportSymbol('proto.ml_metadata.ArtifactType.SystemDefinedBaseType', null, global);
goog.exportSymbol('proto.ml_metadata.Association', null, global);
goog.exportSymbol('proto.ml_metadata.Attribution', null, global);
goog.exportSymbol('proto.ml_metadata.ConnectionConfig', null, global);
goog.exportSymbol('proto.ml_metadata.ConnectionConfig.ConfigCase', null, global);
goog.exportSymbol('proto.ml_metadata.Context', null, global);
goog.exportSymbol('proto.ml_metadata.ContextType', null, global);
goog.exportSymbol('proto.ml_metadata.ContextType.SystemDefinedBaseType', null, global);
goog.exportSymbol('proto.ml_metadata.DictArtifactStructType', null, global);
goog.exportSymbol('proto.ml_metadata.Event', null, global);
goog.exportSymbol('proto.ml_metadata.Event.Path', null, global);
goog.exportSymbol('proto.ml_metadata.Event.Path.Step', null, global);
goog.exportSymbol('proto.ml_metadata.Event.Path.Step.ValueCase', null, global);
goog.exportSymbol('proto.ml_metadata.Event.Type', null, global);
goog.exportSymbol('proto.ml_metadata.Execution', null, global);
goog.exportSymbol('proto.ml_metadata.Execution.State', null, global);
goog.exportSymbol('proto.ml_metadata.ExecutionType', null, global);
goog.exportSymbol('proto.ml_metadata.ExecutionType.SystemDefinedBaseType', null, global);
goog.exportSymbol('proto.ml_metadata.FakeDatabaseConfig', null, global);
goog.exportSymbol('proto.ml_metadata.GrpcChannelArguments', null, global);
goog.exportSymbol('proto.ml_metadata.IntersectionArtifactStructType', null, global);
goog.exportSymbol('proto.ml_metadata.LineageGraph', null, global);
goog.exportSymbol('proto.ml_metadata.LineageGraphQueryOptions', null, global);
goog.exportSymbol('proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint', null, global);
goog.exportSymbol('proto.ml_metadata.LineageGraphQueryOptions.QueryNodesCase', null, global);
goog.exportSymbol('proto.ml_metadata.ListArtifactStructType', null, global);
goog.exportSymbol('proto.ml_metadata.ListOperationNextPageToken', null, global);
goog.exportSymbol('proto.ml_metadata.ListOperationOptions', null, global);
goog.exportSymbol('proto.ml_metadata.ListOperationOptions.OrderByField', null, global);
goog.exportSymbol('proto.ml_metadata.ListOperationOptions.OrderByField.Field', null, global);
goog.exportSymbol('proto.ml_metadata.MetadataStoreClientConfig', null, global);
goog.exportSymbol('proto.ml_metadata.MetadataStoreClientConfig.SSLConfig', null, global);
goog.exportSymbol('proto.ml_metadata.MetadataStoreServerConfig', null, global);
goog.exportSymbol('proto.ml_metadata.MetadataStoreServerConfig.SSLConfig', null, global);
goog.exportSymbol('proto.ml_metadata.MigrationOptions', null, global);
goog.exportSymbol('proto.ml_metadata.MySQLDatabaseConfig', null, global);
goog.exportSymbol('proto.ml_metadata.MySQLDatabaseConfig.SSLOptions', null, global);
goog.exportSymbol('proto.ml_metadata.NoneArtifactStructType', null, global);
goog.exportSymbol('proto.ml_metadata.ParentContext', null, global);
goog.exportSymbol('proto.ml_metadata.PropertyType', null, global);
goog.exportSymbol('proto.ml_metadata.RetryOptions', null, global);
goog.exportSymbol('proto.ml_metadata.SqliteMetadataSourceConfig', null, global);
goog.exportSymbol('proto.ml_metadata.SqliteMetadataSourceConfig.ConnectionMode', null, global);
goog.exportSymbol('proto.ml_metadata.SystemTypeExtension', null, global);
goog.exportSymbol('proto.ml_metadata.TransactionOptions', null, global);
goog.exportSymbol('proto.ml_metadata.TupleArtifactStructType', null, global);
goog.exportSymbol('proto.ml_metadata.UnionArtifactStructType', null, global);
goog.exportSymbol('proto.ml_metadata.Value', null, global);
goog.exportSymbol('proto.ml_metadata.Value.ValueCase', null, global);
goog.exportSymbol('proto.ml_metadata.systemTypeExtension', null, global);
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
proto.ml_metadata.SystemTypeExtension = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.SystemTypeExtension, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.SystemTypeExtension.displayName = 'proto.ml_metadata.SystemTypeExtension';
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
proto.ml_metadata.Value = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_metadata.Value.oneofGroups_);
};
goog.inherits(proto.ml_metadata.Value, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.Value.displayName = 'proto.ml_metadata.Value';
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
proto.ml_metadata.Artifact = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.Artifact, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.Artifact.displayName = 'proto.ml_metadata.Artifact';
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
proto.ml_metadata.ArtifactType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.ArtifactType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ArtifactType.displayName = 'proto.ml_metadata.ArtifactType';
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
proto.ml_metadata.Event = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.Event, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.Event.displayName = 'proto.ml_metadata.Event';
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
proto.ml_metadata.Event.Path = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.Event.Path.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.Event.Path, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.Event.Path.displayName = 'proto.ml_metadata.Event.Path';
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
proto.ml_metadata.Event.Path.Step = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_metadata.Event.Path.Step.oneofGroups_);
};
goog.inherits(proto.ml_metadata.Event.Path.Step, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.Event.Path.Step.displayName = 'proto.ml_metadata.Event.Path.Step';
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
proto.ml_metadata.Execution = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.Execution, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.Execution.displayName = 'proto.ml_metadata.Execution';
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
proto.ml_metadata.ExecutionType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.ExecutionType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ExecutionType.displayName = 'proto.ml_metadata.ExecutionType';
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
proto.ml_metadata.ContextType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.ContextType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ContextType.displayName = 'proto.ml_metadata.ContextType';
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
proto.ml_metadata.Context = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.Context, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.Context.displayName = 'proto.ml_metadata.Context';
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
proto.ml_metadata.Attribution = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.Attribution, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.Attribution.displayName = 'proto.ml_metadata.Attribution';
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
proto.ml_metadata.Association = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.Association, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.Association.displayName = 'proto.ml_metadata.Association';
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
proto.ml_metadata.ParentContext = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.ParentContext, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ParentContext.displayName = 'proto.ml_metadata.ParentContext';
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
proto.ml_metadata.LineageGraph = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.LineageGraph.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.LineageGraph, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.LineageGraph.displayName = 'proto.ml_metadata.LineageGraph';
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
proto.ml_metadata.ArtifactStructType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_metadata.ArtifactStructType.oneofGroups_);
};
goog.inherits(proto.ml_metadata.ArtifactStructType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ArtifactStructType.displayName = 'proto.ml_metadata.ArtifactStructType';
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
proto.ml_metadata.UnionArtifactStructType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.UnionArtifactStructType.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.UnionArtifactStructType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.UnionArtifactStructType.displayName = 'proto.ml_metadata.UnionArtifactStructType';
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
proto.ml_metadata.IntersectionArtifactStructType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.IntersectionArtifactStructType.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.IntersectionArtifactStructType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.IntersectionArtifactStructType.displayName = 'proto.ml_metadata.IntersectionArtifactStructType';
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
proto.ml_metadata.ListArtifactStructType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.ListArtifactStructType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ListArtifactStructType.displayName = 'proto.ml_metadata.ListArtifactStructType';
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
proto.ml_metadata.NoneArtifactStructType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.NoneArtifactStructType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.NoneArtifactStructType.displayName = 'proto.ml_metadata.NoneArtifactStructType';
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
proto.ml_metadata.AnyArtifactStructType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.AnyArtifactStructType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.AnyArtifactStructType.displayName = 'proto.ml_metadata.AnyArtifactStructType';
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
proto.ml_metadata.TupleArtifactStructType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.TupleArtifactStructType.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.TupleArtifactStructType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.TupleArtifactStructType.displayName = 'proto.ml_metadata.TupleArtifactStructType';
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
proto.ml_metadata.DictArtifactStructType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.DictArtifactStructType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.DictArtifactStructType.displayName = 'proto.ml_metadata.DictArtifactStructType';
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
proto.ml_metadata.FakeDatabaseConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.FakeDatabaseConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.FakeDatabaseConfig.displayName = 'proto.ml_metadata.FakeDatabaseConfig';
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
proto.ml_metadata.MySQLDatabaseConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.MySQLDatabaseConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.MySQLDatabaseConfig.displayName = 'proto.ml_metadata.MySQLDatabaseConfig';
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
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.MySQLDatabaseConfig.SSLOptions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.displayName = 'proto.ml_metadata.MySQLDatabaseConfig.SSLOptions';
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
proto.ml_metadata.SqliteMetadataSourceConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.SqliteMetadataSourceConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.SqliteMetadataSourceConfig.displayName = 'proto.ml_metadata.SqliteMetadataSourceConfig';
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
proto.ml_metadata.MigrationOptions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.MigrationOptions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.MigrationOptions.displayName = 'proto.ml_metadata.MigrationOptions';
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
proto.ml_metadata.RetryOptions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.RetryOptions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.RetryOptions.displayName = 'proto.ml_metadata.RetryOptions';
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
proto.ml_metadata.ConnectionConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_metadata.ConnectionConfig.oneofGroups_);
};
goog.inherits(proto.ml_metadata.ConnectionConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ConnectionConfig.displayName = 'proto.ml_metadata.ConnectionConfig';
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
proto.ml_metadata.GrpcChannelArguments = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.GrpcChannelArguments, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.GrpcChannelArguments.displayName = 'proto.ml_metadata.GrpcChannelArguments';
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
proto.ml_metadata.MetadataStoreClientConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.MetadataStoreClientConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.MetadataStoreClientConfig.displayName = 'proto.ml_metadata.MetadataStoreClientConfig';
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
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.MetadataStoreClientConfig.SSLConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.displayName = 'proto.ml_metadata.MetadataStoreClientConfig.SSLConfig';
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
proto.ml_metadata.MetadataStoreServerConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.MetadataStoreServerConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.MetadataStoreServerConfig.displayName = 'proto.ml_metadata.MetadataStoreServerConfig';
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
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.MetadataStoreServerConfig.SSLConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.displayName = 'proto.ml_metadata.MetadataStoreServerConfig.SSLConfig';
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
proto.ml_metadata.ListOperationOptions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.ListOperationOptions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ListOperationOptions.displayName = 'proto.ml_metadata.ListOperationOptions';
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
proto.ml_metadata.ListOperationOptions.OrderByField = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.ListOperationOptions.OrderByField, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ListOperationOptions.OrderByField.displayName = 'proto.ml_metadata.ListOperationOptions.OrderByField';
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
proto.ml_metadata.ListOperationNextPageToken = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_metadata.ListOperationNextPageToken.repeatedFields_, null);
};
goog.inherits(proto.ml_metadata.ListOperationNextPageToken, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.ListOperationNextPageToken.displayName = 'proto.ml_metadata.ListOperationNextPageToken';
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
proto.ml_metadata.TransactionOptions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, 1, null, null);
};
goog.inherits(proto.ml_metadata.TransactionOptions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.TransactionOptions.displayName = 'proto.ml_metadata.TransactionOptions';
}

/**
 * The extensions registered with this message class. This is a map of
 * extension field number to fieldInfo object.
 *
 * For example:
 *     { 123: {fieldIndex: 123, fieldName: {my_field_name: 0}, ctor: proto.example.MyMessage} }
 *
 * fieldName contains the JsCompiler renamed field name property so that it
 * works in OPTIMIZED mode.
 *
 * @type {!Object<number, jspb.ExtensionFieldInfo>}
 */
proto.ml_metadata.TransactionOptions.extensions = {};


/**
 * The extensions registered with this message class. This is a map of
 * extension field number to fieldInfo object.
 *
 * For example:
 *     { 123: {fieldIndex: 123, fieldName: {my_field_name: 0}, ctor: proto.example.MyMessage} }
 *
 * fieldName contains the JsCompiler renamed field name property so that it
 * works in OPTIMIZED mode.
 *
 * @type {!Object<number, jspb.ExtensionFieldBinaryInfo>}
 */
proto.ml_metadata.TransactionOptions.extensionsBinary = {};

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
proto.ml_metadata.LineageGraphQueryOptions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_metadata.LineageGraphQueryOptions.oneofGroups_);
};
goog.inherits(proto.ml_metadata.LineageGraphQueryOptions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.LineageGraphQueryOptions.displayName = 'proto.ml_metadata.LineageGraphQueryOptions';
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
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.displayName = 'proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint';
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
proto.ml_metadata.SystemTypeExtension.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.SystemTypeExtension.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.SystemTypeExtension} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.SystemTypeExtension.toObject = function(includeInstance, msg) {
  var f, obj = {
    typeName: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.SystemTypeExtension}
 */
proto.ml_metadata.SystemTypeExtension.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.SystemTypeExtension;
  return proto.ml_metadata.SystemTypeExtension.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.SystemTypeExtension} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.SystemTypeExtension}
 */
proto.ml_metadata.SystemTypeExtension.deserializeBinaryFromReader = function(msg, reader) {
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
proto.ml_metadata.SystemTypeExtension.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.SystemTypeExtension.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.SystemTypeExtension} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.SystemTypeExtension.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string type_name = 1;
 * @return {string}
 */
proto.ml_metadata.SystemTypeExtension.prototype.getTypeName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.SystemTypeExtension} returns this
 */
proto.ml_metadata.SystemTypeExtension.prototype.setTypeName = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.SystemTypeExtension} returns this
 */
proto.ml_metadata.SystemTypeExtension.prototype.clearTypeName = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.SystemTypeExtension.prototype.hasTypeName = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_metadata.Value.oneofGroups_ = [[1,2,3,4]];

/**
 * @enum {number}
 */
proto.ml_metadata.Value.ValueCase = {
  VALUE_NOT_SET: 0,
  INT_VALUE: 1,
  DOUBLE_VALUE: 2,
  STRING_VALUE: 3,
  STRUCT_VALUE: 4
};

/**
 * @return {proto.ml_metadata.Value.ValueCase}
 */
proto.ml_metadata.Value.prototype.getValueCase = function() {
  return /** @type {proto.ml_metadata.Value.ValueCase} */(jspb.Message.computeOneofCase(this, proto.ml_metadata.Value.oneofGroups_[0]));
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
proto.ml_metadata.Value.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.Value.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.Value} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Value.toObject = function(includeInstance, msg) {
  var f, obj = {
    intValue: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    doubleValue: (f = jspb.Message.getOptionalFloatingPointField(msg, 2)) == null ? undefined : f,
    stringValue: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    structValue: (f = msg.getStructValue()) && google_protobuf_struct_pb.Struct.toObject(includeInstance, f)
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
 * @return {!proto.ml_metadata.Value}
 */
proto.ml_metadata.Value.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.Value;
  return proto.ml_metadata.Value.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.Value} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.Value}
 */
proto.ml_metadata.Value.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setIntValue(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readDouble());
      msg.setDoubleValue(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setStringValue(value);
      break;
    case 4:
      var value = new google_protobuf_struct_pb.Struct;
      reader.readMessage(value,google_protobuf_struct_pb.Struct.deserializeBinaryFromReader);
      msg.setStructValue(value);
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
proto.ml_metadata.Value.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.Value.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.Value} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Value.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeDouble(
      2,
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
  f = message.getStructValue();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      google_protobuf_struct_pb.Struct.serializeBinaryToWriter
    );
  }
};


/**
 * optional int64 int_value = 1;
 * @return {number}
 */
proto.ml_metadata.Value.prototype.getIntValue = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Value} returns this
 */
proto.ml_metadata.Value.prototype.setIntValue = function(value) {
  return jspb.Message.setOneofField(this, 1, proto.ml_metadata.Value.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Value} returns this
 */
proto.ml_metadata.Value.prototype.clearIntValue = function() {
  return jspb.Message.setOneofField(this, 1, proto.ml_metadata.Value.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Value.prototype.hasIntValue = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional double double_value = 2;
 * @return {number}
 */
proto.ml_metadata.Value.prototype.getDoubleValue = function() {
  return /** @type {number} */ (jspb.Message.getFloatingPointFieldWithDefault(this, 2, 0.0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Value} returns this
 */
proto.ml_metadata.Value.prototype.setDoubleValue = function(value) {
  return jspb.Message.setOneofField(this, 2, proto.ml_metadata.Value.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Value} returns this
 */
proto.ml_metadata.Value.prototype.clearDoubleValue = function() {
  return jspb.Message.setOneofField(this, 2, proto.ml_metadata.Value.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Value.prototype.hasDoubleValue = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string string_value = 3;
 * @return {string}
 */
proto.ml_metadata.Value.prototype.getStringValue = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.Value} returns this
 */
proto.ml_metadata.Value.prototype.setStringValue = function(value) {
  return jspb.Message.setOneofField(this, 3, proto.ml_metadata.Value.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Value} returns this
 */
proto.ml_metadata.Value.prototype.clearStringValue = function() {
  return jspb.Message.setOneofField(this, 3, proto.ml_metadata.Value.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Value.prototype.hasStringValue = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional google.protobuf.Struct struct_value = 4;
 * @return {?proto.google.protobuf.Struct}
 */
proto.ml_metadata.Value.prototype.getStructValue = function() {
  return /** @type{?proto.google.protobuf.Struct} */ (
    jspb.Message.getWrapperField(this, google_protobuf_struct_pb.Struct, 4));
};


/**
 * @param {?proto.google.protobuf.Struct|undefined} value
 * @return {!proto.ml_metadata.Value} returns this
*/
proto.ml_metadata.Value.prototype.setStructValue = function(value) {
  return jspb.Message.setOneofWrapperField(this, 4, proto.ml_metadata.Value.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.Value} returns this
 */
proto.ml_metadata.Value.prototype.clearStructValue = function() {
  return this.setStructValue(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Value.prototype.hasStructValue = function() {
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
proto.ml_metadata.Artifact.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.Artifact.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.Artifact} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Artifact.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    name: (f = jspb.Message.getField(msg, 7)) == null ? undefined : f,
    typeId: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    type: (f = jspb.Message.getField(msg, 8)) == null ? undefined : f,
    uri: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, proto.ml_metadata.Value.toObject) : [],
    customPropertiesMap: (f = msg.getCustomPropertiesMap()) ? f.toObject(includeInstance, proto.ml_metadata.Value.toObject) : [],
    state: (f = jspb.Message.getField(msg, 6)) == null ? undefined : f,
    createTimeSinceEpoch: (f = jspb.Message.getField(msg, 9)) == null ? undefined : f,
    lastUpdateTimeSinceEpoch: (f = jspb.Message.getField(msg, 10)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.Artifact}
 */
proto.ml_metadata.Artifact.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.Artifact;
  return proto.ml_metadata.Artifact.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.Artifact} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.Artifact}
 */
proto.ml_metadata.Artifact.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setId(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTypeId(value);
      break;
    case 8:
      var value = /** @type {string} */ (reader.readString());
      msg.setType(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setUri(value);
      break;
    case 4:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_metadata.Value.deserializeBinaryFromReader, "", new proto.ml_metadata.Value());
         });
      break;
    case 5:
      var value = msg.getCustomPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_metadata.Value.deserializeBinaryFromReader, "", new proto.ml_metadata.Value());
         });
      break;
    case 6:
      var value = /** @type {!proto.ml_metadata.Artifact.State} */ (reader.readEnum());
      msg.setState(value);
      break;
    case 9:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setCreateTimeSinceEpoch(value);
      break;
    case 10:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setLastUpdateTimeSinceEpoch(value);
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
proto.ml_metadata.Artifact.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.Artifact.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.Artifact} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Artifact.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 7));
  if (f != null) {
    writer.writeString(
      7,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 8));
  if (f != null) {
    writer.writeString(
      8,
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
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(4, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_metadata.Value.serializeBinaryToWriter);
  }
  f = message.getCustomPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_metadata.Value.serializeBinaryToWriter);
  }
  f = /** @type {!proto.ml_metadata.Artifact.State} */ (jspb.Message.getField(message, 6));
  if (f != null) {
    writer.writeEnum(
      6,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 9));
  if (f != null) {
    writer.writeInt64(
      9,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 10));
  if (f != null) {
    writer.writeInt64(
      10,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.ml_metadata.Artifact.State = {
  UNKNOWN: 0,
  PENDING: 1,
  LIVE: 2,
  MARKED_FOR_DELETION: 3,
  DELETED: 4
};

/**
 * optional int64 id = 1;
 * @return {number}
 */
proto.ml_metadata.Artifact.prototype.getId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.setId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Artifact.prototype.hasId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string name = 7;
 * @return {string}
 */
proto.ml_metadata.Artifact.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.setName = function(value) {
  return jspb.Message.setField(this, 7, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearName = function() {
  return jspb.Message.setField(this, 7, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Artifact.prototype.hasName = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional int64 type_id = 2;
 * @return {number}
 */
proto.ml_metadata.Artifact.prototype.getTypeId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.setTypeId = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearTypeId = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Artifact.prototype.hasTypeId = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string type = 8;
 * @return {string}
 */
proto.ml_metadata.Artifact.prototype.getType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 8, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.setType = function(value) {
  return jspb.Message.setField(this, 8, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearType = function() {
  return jspb.Message.setField(this, 8, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Artifact.prototype.hasType = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional string uri = 3;
 * @return {string}
 */
proto.ml_metadata.Artifact.prototype.getUri = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.setUri = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearUri = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Artifact.prototype.hasUri = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * map<string, Value> properties = 4;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.Value>}
 */
proto.ml_metadata.Artifact.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.Value>} */ (
      jspb.Message.getMapField(this, 4, opt_noLazyCreate,
      proto.ml_metadata.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * map<string, Value> custom_properties = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.Value>}
 */
proto.ml_metadata.Artifact.prototype.getCustomPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.Value>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      proto.ml_metadata.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearCustomPropertiesMap = function() {
  this.getCustomPropertiesMap().clear();
  return this;};


/**
 * optional State state = 6;
 * @return {!proto.ml_metadata.Artifact.State}
 */
proto.ml_metadata.Artifact.prototype.getState = function() {
  return /** @type {!proto.ml_metadata.Artifact.State} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {!proto.ml_metadata.Artifact.State} value
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.setState = function(value) {
  return jspb.Message.setField(this, 6, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearState = function() {
  return jspb.Message.setField(this, 6, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Artifact.prototype.hasState = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional int64 create_time_since_epoch = 9;
 * @return {number}
 */
proto.ml_metadata.Artifact.prototype.getCreateTimeSinceEpoch = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 9, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.setCreateTimeSinceEpoch = function(value) {
  return jspb.Message.setField(this, 9, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearCreateTimeSinceEpoch = function() {
  return jspb.Message.setField(this, 9, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Artifact.prototype.hasCreateTimeSinceEpoch = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional int64 last_update_time_since_epoch = 10;
 * @return {number}
 */
proto.ml_metadata.Artifact.prototype.getLastUpdateTimeSinceEpoch = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 10, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.setLastUpdateTimeSinceEpoch = function(value) {
  return jspb.Message.setField(this, 10, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Artifact} returns this
 */
proto.ml_metadata.Artifact.prototype.clearLastUpdateTimeSinceEpoch = function() {
  return jspb.Message.setField(this, 10, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Artifact.prototype.hasLastUpdateTimeSinceEpoch = function() {
  return jspb.Message.getField(this, 10) != null;
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
proto.ml_metadata.ArtifactType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ArtifactType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ArtifactType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactType.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    name: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    version: (f = jspb.Message.getField(msg, 4)) == null ? undefined : f,
    description: (f = jspb.Message.getField(msg, 5)) == null ? undefined : f,
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, undefined) : [],
    baseType: (f = jspb.Message.getField(msg, 6)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.ArtifactType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ArtifactType;
  return proto.ml_metadata.ArtifactType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ArtifactType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.ArtifactType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setVersion(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setDescription(value);
      break;
    case 3:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readEnum, null, "", 0);
         });
      break;
    case 6:
      var value = /** @type {!proto.ml_metadata.ArtifactType.SystemDefinedBaseType} */ (reader.readEnum());
      msg.setBaseType(value);
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
proto.ml_metadata.ArtifactType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ArtifactType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ArtifactType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
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
  f = /** @type {string} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeString(
      4,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(3, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeEnum);
  }
  f = /** @type {!proto.ml_metadata.ArtifactType.SystemDefinedBaseType} */ (jspb.Message.getField(message, 6));
  if (f != null) {
    writer.writeEnum(
      6,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.ml_metadata.ArtifactType.SystemDefinedBaseType = {
  UNSET: 0,
  DATASET: 1,
  MODEL: 2,
  METRICS: 3,
  STATISTICS: 4
};

/**
 * optional int64 id = 1;
 * @return {number}
 */
proto.ml_metadata.ArtifactType.prototype.getId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.setId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.clearId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactType.prototype.hasId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.ml_metadata.ArtifactType.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.setName = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.clearName = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactType.prototype.hasName = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string version = 4;
 * @return {string}
 */
proto.ml_metadata.ArtifactType.prototype.getVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.setVersion = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.clearVersion = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactType.prototype.hasVersion = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional string description = 5;
 * @return {string}
 */
proto.ml_metadata.ArtifactType.prototype.getDescription = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.setDescription = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.clearDescription = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactType.prototype.hasDescription = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * map<string, PropertyType> properties = 3;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.PropertyType>}
 */
proto.ml_metadata.ArtifactType.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.PropertyType>} */ (
      jspb.Message.getMapField(this, 3, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * optional SystemDefinedBaseType base_type = 6;
 * @return {!proto.ml_metadata.ArtifactType.SystemDefinedBaseType}
 */
proto.ml_metadata.ArtifactType.prototype.getBaseType = function() {
  return /** @type {!proto.ml_metadata.ArtifactType.SystemDefinedBaseType} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {!proto.ml_metadata.ArtifactType.SystemDefinedBaseType} value
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.setBaseType = function(value) {
  return jspb.Message.setField(this, 6, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ArtifactType} returns this
 */
proto.ml_metadata.ArtifactType.prototype.clearBaseType = function() {
  return jspb.Message.setField(this, 6, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactType.prototype.hasBaseType = function() {
  return jspb.Message.getField(this, 6) != null;
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
proto.ml_metadata.Event.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.Event.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.Event} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Event.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    executionId: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    path: (f = msg.getPath()) && proto.ml_metadata.Event.Path.toObject(includeInstance, f),
    type: (f = jspb.Message.getField(msg, 4)) == null ? undefined : f,
    millisecondsSinceEpoch: (f = jspb.Message.getField(msg, 5)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.Event}
 */
proto.ml_metadata.Event.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.Event;
  return proto.ml_metadata.Event.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.Event} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.Event}
 */
proto.ml_metadata.Event.deserializeBinaryFromReader = function(msg, reader) {
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
      var value = /** @type {number} */ (reader.readInt64());
      msg.setExecutionId(value);
      break;
    case 3:
      var value = new proto.ml_metadata.Event.Path;
      reader.readMessage(value,proto.ml_metadata.Event.Path.deserializeBinaryFromReader);
      msg.setPath(value);
      break;
    case 4:
      var value = /** @type {!proto.ml_metadata.Event.Type} */ (reader.readEnum());
      msg.setType(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setMillisecondsSinceEpoch(value);
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
proto.ml_metadata.Event.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.Event.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.Event} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Event.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
  f = message.getPath();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_metadata.Event.Path.serializeBinaryToWriter
    );
  }
  f = /** @type {!proto.ml_metadata.Event.Type} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeEnum(
      4,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeInt64(
      5,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.ml_metadata.Event.Type = {
  UNKNOWN: 0,
  DECLARED_OUTPUT: 1,
  DECLARED_INPUT: 2,
  INPUT: 3,
  OUTPUT: 4,
  INTERNAL_INPUT: 5,
  INTERNAL_OUTPUT: 6
};


/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.Event.Path.repeatedFields_ = [1];



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
proto.ml_metadata.Event.Path.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.Event.Path.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.Event.Path} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Event.Path.toObject = function(includeInstance, msg) {
  var f, obj = {
    stepsList: jspb.Message.toObjectList(msg.getStepsList(),
    proto.ml_metadata.Event.Path.Step.toObject, includeInstance)
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
 * @return {!proto.ml_metadata.Event.Path}
 */
proto.ml_metadata.Event.Path.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.Event.Path;
  return proto.ml_metadata.Event.Path.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.Event.Path} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.Event.Path}
 */
proto.ml_metadata.Event.Path.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.Event.Path.Step;
      reader.readMessage(value,proto.ml_metadata.Event.Path.Step.deserializeBinaryFromReader);
      msg.addSteps(value);
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
proto.ml_metadata.Event.Path.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.Event.Path.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.Event.Path} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Event.Path.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getStepsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.ml_metadata.Event.Path.Step.serializeBinaryToWriter
    );
  }
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_metadata.Event.Path.Step.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.ml_metadata.Event.Path.Step.ValueCase = {
  VALUE_NOT_SET: 0,
  INDEX: 1,
  KEY: 2
};

/**
 * @return {proto.ml_metadata.Event.Path.Step.ValueCase}
 */
proto.ml_metadata.Event.Path.Step.prototype.getValueCase = function() {
  return /** @type {proto.ml_metadata.Event.Path.Step.ValueCase} */(jspb.Message.computeOneofCase(this, proto.ml_metadata.Event.Path.Step.oneofGroups_[0]));
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
proto.ml_metadata.Event.Path.Step.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.Event.Path.Step.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.Event.Path.Step} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Event.Path.Step.toObject = function(includeInstance, msg) {
  var f, obj = {
    index: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    key: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.Event.Path.Step}
 */
proto.ml_metadata.Event.Path.Step.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.Event.Path.Step;
  return proto.ml_metadata.Event.Path.Step.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.Event.Path.Step} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.Event.Path.Step}
 */
proto.ml_metadata.Event.Path.Step.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setIndex(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setKey(value);
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
proto.ml_metadata.Event.Path.Step.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.Event.Path.Step.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.Event.Path.Step} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Event.Path.Step.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
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
};


/**
 * optional int64 index = 1;
 * @return {number}
 */
proto.ml_metadata.Event.Path.Step.prototype.getIndex = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Event.Path.Step} returns this
 */
proto.ml_metadata.Event.Path.Step.prototype.setIndex = function(value) {
  return jspb.Message.setOneofField(this, 1, proto.ml_metadata.Event.Path.Step.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Event.Path.Step} returns this
 */
proto.ml_metadata.Event.Path.Step.prototype.clearIndex = function() {
  return jspb.Message.setOneofField(this, 1, proto.ml_metadata.Event.Path.Step.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Event.Path.Step.prototype.hasIndex = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string key = 2;
 * @return {string}
 */
proto.ml_metadata.Event.Path.Step.prototype.getKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.Event.Path.Step} returns this
 */
proto.ml_metadata.Event.Path.Step.prototype.setKey = function(value) {
  return jspb.Message.setOneofField(this, 2, proto.ml_metadata.Event.Path.Step.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Event.Path.Step} returns this
 */
proto.ml_metadata.Event.Path.Step.prototype.clearKey = function() {
  return jspb.Message.setOneofField(this, 2, proto.ml_metadata.Event.Path.Step.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Event.Path.Step.prototype.hasKey = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * repeated Step steps = 1;
 * @return {!Array<!proto.ml_metadata.Event.Path.Step>}
 */
proto.ml_metadata.Event.Path.prototype.getStepsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Event.Path.Step>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.Event.Path.Step, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.Event.Path.Step>} value
 * @return {!proto.ml_metadata.Event.Path} returns this
*/
proto.ml_metadata.Event.Path.prototype.setStepsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.Event.Path.Step=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Event.Path.Step}
 */
proto.ml_metadata.Event.Path.prototype.addSteps = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.Event.Path.Step, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.Event.Path} returns this
 */
proto.ml_metadata.Event.Path.prototype.clearStepsList = function() {
  return this.setStepsList([]);
};


/**
 * optional int64 artifact_id = 1;
 * @return {number}
 */
proto.ml_metadata.Event.prototype.getArtifactId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Event} returns this
 */
proto.ml_metadata.Event.prototype.setArtifactId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Event} returns this
 */
proto.ml_metadata.Event.prototype.clearArtifactId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Event.prototype.hasArtifactId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional int64 execution_id = 2;
 * @return {number}
 */
proto.ml_metadata.Event.prototype.getExecutionId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Event} returns this
 */
proto.ml_metadata.Event.prototype.setExecutionId = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Event} returns this
 */
proto.ml_metadata.Event.prototype.clearExecutionId = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Event.prototype.hasExecutionId = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional Path path = 3;
 * @return {?proto.ml_metadata.Event.Path}
 */
proto.ml_metadata.Event.prototype.getPath = function() {
  return /** @type{?proto.ml_metadata.Event.Path} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.Event.Path, 3));
};


/**
 * @param {?proto.ml_metadata.Event.Path|undefined} value
 * @return {!proto.ml_metadata.Event} returns this
*/
proto.ml_metadata.Event.prototype.setPath = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.Event} returns this
 */
proto.ml_metadata.Event.prototype.clearPath = function() {
  return this.setPath(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Event.prototype.hasPath = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional Type type = 4;
 * @return {!proto.ml_metadata.Event.Type}
 */
proto.ml_metadata.Event.prototype.getType = function() {
  return /** @type {!proto.ml_metadata.Event.Type} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {!proto.ml_metadata.Event.Type} value
 * @return {!proto.ml_metadata.Event} returns this
 */
proto.ml_metadata.Event.prototype.setType = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Event} returns this
 */
proto.ml_metadata.Event.prototype.clearType = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Event.prototype.hasType = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional int64 milliseconds_since_epoch = 5;
 * @return {number}
 */
proto.ml_metadata.Event.prototype.getMillisecondsSinceEpoch = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Event} returns this
 */
proto.ml_metadata.Event.prototype.setMillisecondsSinceEpoch = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Event} returns this
 */
proto.ml_metadata.Event.prototype.clearMillisecondsSinceEpoch = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Event.prototype.hasMillisecondsSinceEpoch = function() {
  return jspb.Message.getField(this, 5) != null;
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
proto.ml_metadata.Execution.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.Execution.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.Execution} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Execution.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    name: (f = jspb.Message.getField(msg, 6)) == null ? undefined : f,
    typeId: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    type: (f = jspb.Message.getField(msg, 7)) == null ? undefined : f,
    lastKnownState: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, proto.ml_metadata.Value.toObject) : [],
    customPropertiesMap: (f = msg.getCustomPropertiesMap()) ? f.toObject(includeInstance, proto.ml_metadata.Value.toObject) : [],
    createTimeSinceEpoch: (f = jspb.Message.getField(msg, 8)) == null ? undefined : f,
    lastUpdateTimeSinceEpoch: (f = jspb.Message.getField(msg, 9)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.Execution}
 */
proto.ml_metadata.Execution.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.Execution;
  return proto.ml_metadata.Execution.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.Execution} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.Execution}
 */
proto.ml_metadata.Execution.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setId(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTypeId(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setType(value);
      break;
    case 3:
      var value = /** @type {!proto.ml_metadata.Execution.State} */ (reader.readEnum());
      msg.setLastKnownState(value);
      break;
    case 4:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_metadata.Value.deserializeBinaryFromReader, "", new proto.ml_metadata.Value());
         });
      break;
    case 5:
      var value = msg.getCustomPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_metadata.Value.deserializeBinaryFromReader, "", new proto.ml_metadata.Value());
         });
      break;
    case 8:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setCreateTimeSinceEpoch(value);
      break;
    case 9:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setLastUpdateTimeSinceEpoch(value);
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
proto.ml_metadata.Execution.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.Execution.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.Execution} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Execution.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 6));
  if (f != null) {
    writer.writeString(
      6,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 7));
  if (f != null) {
    writer.writeString(
      7,
      f
    );
  }
  f = /** @type {!proto.ml_metadata.Execution.State} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeEnum(
      3,
      f
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(4, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_metadata.Value.serializeBinaryToWriter);
  }
  f = message.getCustomPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_metadata.Value.serializeBinaryToWriter);
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 8));
  if (f != null) {
    writer.writeInt64(
      8,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 9));
  if (f != null) {
    writer.writeInt64(
      9,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.ml_metadata.Execution.State = {
  UNKNOWN: 0,
  NEW: 1,
  RUNNING: 2,
  COMPLETE: 3,
  FAILED: 4,
  CACHED: 5,
  CANCELED: 6
};

/**
 * optional int64 id = 1;
 * @return {number}
 */
proto.ml_metadata.Execution.prototype.getId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.setId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.clearId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Execution.prototype.hasId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string name = 6;
 * @return {string}
 */
proto.ml_metadata.Execution.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.setName = function(value) {
  return jspb.Message.setField(this, 6, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.clearName = function() {
  return jspb.Message.setField(this, 6, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Execution.prototype.hasName = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional int64 type_id = 2;
 * @return {number}
 */
proto.ml_metadata.Execution.prototype.getTypeId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.setTypeId = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.clearTypeId = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Execution.prototype.hasTypeId = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string type = 7;
 * @return {string}
 */
proto.ml_metadata.Execution.prototype.getType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.setType = function(value) {
  return jspb.Message.setField(this, 7, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.clearType = function() {
  return jspb.Message.setField(this, 7, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Execution.prototype.hasType = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional State last_known_state = 3;
 * @return {!proto.ml_metadata.Execution.State}
 */
proto.ml_metadata.Execution.prototype.getLastKnownState = function() {
  return /** @type {!proto.ml_metadata.Execution.State} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.ml_metadata.Execution.State} value
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.setLastKnownState = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.clearLastKnownState = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Execution.prototype.hasLastKnownState = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * map<string, Value> properties = 4;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.Value>}
 */
proto.ml_metadata.Execution.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.Value>} */ (
      jspb.Message.getMapField(this, 4, opt_noLazyCreate,
      proto.ml_metadata.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * map<string, Value> custom_properties = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.Value>}
 */
proto.ml_metadata.Execution.prototype.getCustomPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.Value>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      proto.ml_metadata.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.clearCustomPropertiesMap = function() {
  this.getCustomPropertiesMap().clear();
  return this;};


/**
 * optional int64 create_time_since_epoch = 8;
 * @return {number}
 */
proto.ml_metadata.Execution.prototype.getCreateTimeSinceEpoch = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 8, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.setCreateTimeSinceEpoch = function(value) {
  return jspb.Message.setField(this, 8, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.clearCreateTimeSinceEpoch = function() {
  return jspb.Message.setField(this, 8, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Execution.prototype.hasCreateTimeSinceEpoch = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional int64 last_update_time_since_epoch = 9;
 * @return {number}
 */
proto.ml_metadata.Execution.prototype.getLastUpdateTimeSinceEpoch = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 9, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.setLastUpdateTimeSinceEpoch = function(value) {
  return jspb.Message.setField(this, 9, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Execution} returns this
 */
proto.ml_metadata.Execution.prototype.clearLastUpdateTimeSinceEpoch = function() {
  return jspb.Message.setField(this, 9, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Execution.prototype.hasLastUpdateTimeSinceEpoch = function() {
  return jspb.Message.getField(this, 9) != null;
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
proto.ml_metadata.ExecutionType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ExecutionType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ExecutionType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ExecutionType.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    name: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    version: (f = jspb.Message.getField(msg, 6)) == null ? undefined : f,
    description: (f = jspb.Message.getField(msg, 7)) == null ? undefined : f,
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, undefined) : [],
    inputType: (f = msg.getInputType()) && proto.ml_metadata.ArtifactStructType.toObject(includeInstance, f),
    outputType: (f = msg.getOutputType()) && proto.ml_metadata.ArtifactStructType.toObject(includeInstance, f),
    baseType: (f = jspb.Message.getField(msg, 8)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.ExecutionType}
 */
proto.ml_metadata.ExecutionType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ExecutionType;
  return proto.ml_metadata.ExecutionType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ExecutionType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ExecutionType}
 */
proto.ml_metadata.ExecutionType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setVersion(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setDescription(value);
      break;
    case 3:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readEnum, null, "", 0);
         });
      break;
    case 4:
      var value = new proto.ml_metadata.ArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader);
      msg.setInputType(value);
      break;
    case 5:
      var value = new proto.ml_metadata.ArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader);
      msg.setOutputType(value);
      break;
    case 8:
      var value = /** @type {!proto.ml_metadata.ExecutionType.SystemDefinedBaseType} */ (reader.readEnum());
      msg.setBaseType(value);
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
proto.ml_metadata.ExecutionType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ExecutionType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ExecutionType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ExecutionType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
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
  f = /** @type {string} */ (jspb.Message.getField(message, 6));
  if (f != null) {
    writer.writeString(
      6,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 7));
  if (f != null) {
    writer.writeString(
      7,
      f
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(3, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeEnum);
  }
  f = message.getInputType();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter
    );
  }
  f = message.getOutputType();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter
    );
  }
  f = /** @type {!proto.ml_metadata.ExecutionType.SystemDefinedBaseType} */ (jspb.Message.getField(message, 8));
  if (f != null) {
    writer.writeEnum(
      8,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.ml_metadata.ExecutionType.SystemDefinedBaseType = {
  UNSET: 0,
  TRAIN: 1,
  TRANSFORM: 2,
  PROCESS: 3,
  EVALUATE: 4,
  DEPLOY: 5
};

/**
 * optional int64 id = 1;
 * @return {number}
 */
proto.ml_metadata.ExecutionType.prototype.getId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.setId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.clearId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ExecutionType.prototype.hasId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.ml_metadata.ExecutionType.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.setName = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.clearName = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ExecutionType.prototype.hasName = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string version = 6;
 * @return {string}
 */
proto.ml_metadata.ExecutionType.prototype.getVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.setVersion = function(value) {
  return jspb.Message.setField(this, 6, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.clearVersion = function() {
  return jspb.Message.setField(this, 6, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ExecutionType.prototype.hasVersion = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional string description = 7;
 * @return {string}
 */
proto.ml_metadata.ExecutionType.prototype.getDescription = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.setDescription = function(value) {
  return jspb.Message.setField(this, 7, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.clearDescription = function() {
  return jspb.Message.setField(this, 7, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ExecutionType.prototype.hasDescription = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * map<string, PropertyType> properties = 3;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.PropertyType>}
 */
proto.ml_metadata.ExecutionType.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.PropertyType>} */ (
      jspb.Message.getMapField(this, 3, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * optional ArtifactStructType input_type = 4;
 * @return {?proto.ml_metadata.ArtifactStructType}
 */
proto.ml_metadata.ExecutionType.prototype.getInputType = function() {
  return /** @type{?proto.ml_metadata.ArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ArtifactStructType, 4));
};


/**
 * @param {?proto.ml_metadata.ArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ExecutionType} returns this
*/
proto.ml_metadata.ExecutionType.prototype.setInputType = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.clearInputType = function() {
  return this.setInputType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ExecutionType.prototype.hasInputType = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional ArtifactStructType output_type = 5;
 * @return {?proto.ml_metadata.ArtifactStructType}
 */
proto.ml_metadata.ExecutionType.prototype.getOutputType = function() {
  return /** @type{?proto.ml_metadata.ArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ArtifactStructType, 5));
};


/**
 * @param {?proto.ml_metadata.ArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ExecutionType} returns this
*/
proto.ml_metadata.ExecutionType.prototype.setOutputType = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.clearOutputType = function() {
  return this.setOutputType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ExecutionType.prototype.hasOutputType = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional SystemDefinedBaseType base_type = 8;
 * @return {!proto.ml_metadata.ExecutionType.SystemDefinedBaseType}
 */
proto.ml_metadata.ExecutionType.prototype.getBaseType = function() {
  return /** @type {!proto.ml_metadata.ExecutionType.SystemDefinedBaseType} */ (jspb.Message.getFieldWithDefault(this, 8, 0));
};


/**
 * @param {!proto.ml_metadata.ExecutionType.SystemDefinedBaseType} value
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.setBaseType = function(value) {
  return jspb.Message.setField(this, 8, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ExecutionType} returns this
 */
proto.ml_metadata.ExecutionType.prototype.clearBaseType = function() {
  return jspb.Message.setField(this, 8, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ExecutionType.prototype.hasBaseType = function() {
  return jspb.Message.getField(this, 8) != null;
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
proto.ml_metadata.ContextType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ContextType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ContextType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ContextType.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    name: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    version: (f = jspb.Message.getField(msg, 4)) == null ? undefined : f,
    description: (f = jspb.Message.getField(msg, 5)) == null ? undefined : f,
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, undefined) : [],
    baseType: (f = jspb.Message.getField(msg, 6)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.ContextType}
 */
proto.ml_metadata.ContextType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ContextType;
  return proto.ml_metadata.ContextType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ContextType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ContextType}
 */
proto.ml_metadata.ContextType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setVersion(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setDescription(value);
      break;
    case 3:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readEnum, null, "", 0);
         });
      break;
    case 6:
      var value = /** @type {!proto.ml_metadata.ContextType.SystemDefinedBaseType} */ (reader.readEnum());
      msg.setBaseType(value);
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
proto.ml_metadata.ContextType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ContextType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ContextType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ContextType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
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
  f = /** @type {string} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeString(
      4,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(3, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeEnum);
  }
  f = /** @type {!proto.ml_metadata.ContextType.SystemDefinedBaseType} */ (jspb.Message.getField(message, 6));
  if (f != null) {
    writer.writeEnum(
      6,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.ml_metadata.ContextType.SystemDefinedBaseType = {
  UNSET: 0
};

/**
 * optional int64 id = 1;
 * @return {number}
 */
proto.ml_metadata.ContextType.prototype.getId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.setId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.clearId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ContextType.prototype.hasId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.ml_metadata.ContextType.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.setName = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.clearName = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ContextType.prototype.hasName = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string version = 4;
 * @return {string}
 */
proto.ml_metadata.ContextType.prototype.getVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.setVersion = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.clearVersion = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ContextType.prototype.hasVersion = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional string description = 5;
 * @return {string}
 */
proto.ml_metadata.ContextType.prototype.getDescription = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.setDescription = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.clearDescription = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ContextType.prototype.hasDescription = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * map<string, PropertyType> properties = 3;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.PropertyType>}
 */
proto.ml_metadata.ContextType.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.PropertyType>} */ (
      jspb.Message.getMapField(this, 3, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * optional SystemDefinedBaseType base_type = 6;
 * @return {!proto.ml_metadata.ContextType.SystemDefinedBaseType}
 */
proto.ml_metadata.ContextType.prototype.getBaseType = function() {
  return /** @type {!proto.ml_metadata.ContextType.SystemDefinedBaseType} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {!proto.ml_metadata.ContextType.SystemDefinedBaseType} value
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.setBaseType = function(value) {
  return jspb.Message.setField(this, 6, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ContextType} returns this
 */
proto.ml_metadata.ContextType.prototype.clearBaseType = function() {
  return jspb.Message.setField(this, 6, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ContextType.prototype.hasBaseType = function() {
  return jspb.Message.getField(this, 6) != null;
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
proto.ml_metadata.Context.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.Context.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.Context} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Context.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    name: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    typeId: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    type: (f = jspb.Message.getField(msg, 6)) == null ? undefined : f,
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, proto.ml_metadata.Value.toObject) : [],
    customPropertiesMap: (f = msg.getCustomPropertiesMap()) ? f.toObject(includeInstance, proto.ml_metadata.Value.toObject) : [],
    createTimeSinceEpoch: (f = jspb.Message.getField(msg, 7)) == null ? undefined : f,
    lastUpdateTimeSinceEpoch: (f = jspb.Message.getField(msg, 8)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.Context.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.Context;
  return proto.ml_metadata.Context.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.Context} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.Context.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTypeId(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setType(value);
      break;
    case 4:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_metadata.Value.deserializeBinaryFromReader, "", new proto.ml_metadata.Value());
         });
      break;
    case 5:
      var value = msg.getCustomPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_metadata.Value.deserializeBinaryFromReader, "", new proto.ml_metadata.Value());
         });
      break;
    case 7:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setCreateTimeSinceEpoch(value);
      break;
    case 8:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setLastUpdateTimeSinceEpoch(value);
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
proto.ml_metadata.Context.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.Context.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.Context} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Context.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
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
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 6));
  if (f != null) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(4, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_metadata.Value.serializeBinaryToWriter);
  }
  f = message.getCustomPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_metadata.Value.serializeBinaryToWriter);
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 7));
  if (f != null) {
    writer.writeInt64(
      7,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 8));
  if (f != null) {
    writer.writeInt64(
      8,
      f
    );
  }
};


/**
 * optional int64 id = 1;
 * @return {number}
 */
proto.ml_metadata.Context.prototype.getId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.setId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.clearId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Context.prototype.hasId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string name = 3;
 * @return {string}
 */
proto.ml_metadata.Context.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.setName = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.clearName = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Context.prototype.hasName = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional int64 type_id = 2;
 * @return {number}
 */
proto.ml_metadata.Context.prototype.getTypeId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.setTypeId = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.clearTypeId = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Context.prototype.hasTypeId = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string type = 6;
 * @return {string}
 */
proto.ml_metadata.Context.prototype.getType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.setType = function(value) {
  return jspb.Message.setField(this, 6, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.clearType = function() {
  return jspb.Message.setField(this, 6, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Context.prototype.hasType = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * map<string, Value> properties = 4;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.Value>}
 */
proto.ml_metadata.Context.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.Value>} */ (
      jspb.Message.getMapField(this, 4, opt_noLazyCreate,
      proto.ml_metadata.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * map<string, Value> custom_properties = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.Value>}
 */
proto.ml_metadata.Context.prototype.getCustomPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.Value>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      proto.ml_metadata.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.clearCustomPropertiesMap = function() {
  this.getCustomPropertiesMap().clear();
  return this;};


/**
 * optional int64 create_time_since_epoch = 7;
 * @return {number}
 */
proto.ml_metadata.Context.prototype.getCreateTimeSinceEpoch = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 7, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.setCreateTimeSinceEpoch = function(value) {
  return jspb.Message.setField(this, 7, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.clearCreateTimeSinceEpoch = function() {
  return jspb.Message.setField(this, 7, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Context.prototype.hasCreateTimeSinceEpoch = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional int64 last_update_time_since_epoch = 8;
 * @return {number}
 */
proto.ml_metadata.Context.prototype.getLastUpdateTimeSinceEpoch = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 8, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.setLastUpdateTimeSinceEpoch = function(value) {
  return jspb.Message.setField(this, 8, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Context} returns this
 */
proto.ml_metadata.Context.prototype.clearLastUpdateTimeSinceEpoch = function() {
  return jspb.Message.setField(this, 8, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Context.prototype.hasLastUpdateTimeSinceEpoch = function() {
  return jspb.Message.getField(this, 8) != null;
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
proto.ml_metadata.Attribution.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.Attribution.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.Attribution} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Attribution.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    contextId: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.Attribution}
 */
proto.ml_metadata.Attribution.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.Attribution;
  return proto.ml_metadata.Attribution.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.Attribution} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.Attribution}
 */
proto.ml_metadata.Attribution.deserializeBinaryFromReader = function(msg, reader) {
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
      var value = /** @type {number} */ (reader.readInt64());
      msg.setContextId(value);
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
proto.ml_metadata.Attribution.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.Attribution.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.Attribution} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Attribution.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
};


/**
 * optional int64 artifact_id = 1;
 * @return {number}
 */
proto.ml_metadata.Attribution.prototype.getArtifactId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Attribution} returns this
 */
proto.ml_metadata.Attribution.prototype.setArtifactId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Attribution} returns this
 */
proto.ml_metadata.Attribution.prototype.clearArtifactId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Attribution.prototype.hasArtifactId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional int64 context_id = 2;
 * @return {number}
 */
proto.ml_metadata.Attribution.prototype.getContextId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Attribution} returns this
 */
proto.ml_metadata.Attribution.prototype.setContextId = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Attribution} returns this
 */
proto.ml_metadata.Attribution.prototype.clearContextId = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Attribution.prototype.hasContextId = function() {
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
proto.ml_metadata.Association.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.Association.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.Association} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Association.toObject = function(includeInstance, msg) {
  var f, obj = {
    executionId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    contextId: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.Association}
 */
proto.ml_metadata.Association.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.Association;
  return proto.ml_metadata.Association.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.Association} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.Association}
 */
proto.ml_metadata.Association.deserializeBinaryFromReader = function(msg, reader) {
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
      var value = /** @type {number} */ (reader.readInt64());
      msg.setContextId(value);
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
proto.ml_metadata.Association.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.Association.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.Association} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.Association.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
};


/**
 * optional int64 execution_id = 1;
 * @return {number}
 */
proto.ml_metadata.Association.prototype.getExecutionId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Association} returns this
 */
proto.ml_metadata.Association.prototype.setExecutionId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Association} returns this
 */
proto.ml_metadata.Association.prototype.clearExecutionId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Association.prototype.hasExecutionId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional int64 context_id = 2;
 * @return {number}
 */
proto.ml_metadata.Association.prototype.getContextId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.Association} returns this
 */
proto.ml_metadata.Association.prototype.setContextId = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.Association} returns this
 */
proto.ml_metadata.Association.prototype.clearContextId = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.Association.prototype.hasContextId = function() {
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
proto.ml_metadata.ParentContext.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ParentContext.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ParentContext} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ParentContext.toObject = function(includeInstance, msg) {
  var f, obj = {
    childId: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    parentId: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.ParentContext}
 */
proto.ml_metadata.ParentContext.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ParentContext;
  return proto.ml_metadata.ParentContext.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ParentContext} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ParentContext}
 */
proto.ml_metadata.ParentContext.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setChildId(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setParentId(value);
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
proto.ml_metadata.ParentContext.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ParentContext.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ParentContext} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ParentContext.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
};


/**
 * optional int64 child_id = 1;
 * @return {number}
 */
proto.ml_metadata.ParentContext.prototype.getChildId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.ParentContext} returns this
 */
proto.ml_metadata.ParentContext.prototype.setChildId = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ParentContext} returns this
 */
proto.ml_metadata.ParentContext.prototype.clearChildId = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ParentContext.prototype.hasChildId = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional int64 parent_id = 2;
 * @return {number}
 */
proto.ml_metadata.ParentContext.prototype.getParentId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.ParentContext} returns this
 */
proto.ml_metadata.ParentContext.prototype.setParentId = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ParentContext} returns this
 */
proto.ml_metadata.ParentContext.prototype.clearParentId = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ParentContext.prototype.hasParentId = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.LineageGraph.repeatedFields_ = [1,2,3,4,5,6,7,8,9];



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
proto.ml_metadata.LineageGraph.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.LineageGraph.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.LineageGraph} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.LineageGraph.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactTypesList: jspb.Message.toObjectList(msg.getArtifactTypesList(),
    proto.ml_metadata.ArtifactType.toObject, includeInstance),
    executionTypesList: jspb.Message.toObjectList(msg.getExecutionTypesList(),
    proto.ml_metadata.ExecutionType.toObject, includeInstance),
    contextTypesList: jspb.Message.toObjectList(msg.getContextTypesList(),
    proto.ml_metadata.ContextType.toObject, includeInstance),
    artifactsList: jspb.Message.toObjectList(msg.getArtifactsList(),
    proto.ml_metadata.Artifact.toObject, includeInstance),
    executionsList: jspb.Message.toObjectList(msg.getExecutionsList(),
    proto.ml_metadata.Execution.toObject, includeInstance),
    contextsList: jspb.Message.toObjectList(msg.getContextsList(),
    proto.ml_metadata.Context.toObject, includeInstance),
    eventsList: jspb.Message.toObjectList(msg.getEventsList(),
    proto.ml_metadata.Event.toObject, includeInstance),
    attributionsList: jspb.Message.toObjectList(msg.getAttributionsList(),
    proto.ml_metadata.Attribution.toObject, includeInstance),
    associationsList: jspb.Message.toObjectList(msg.getAssociationsList(),
    proto.ml_metadata.Association.toObject, includeInstance)
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
 * @return {!proto.ml_metadata.LineageGraph}
 */
proto.ml_metadata.LineageGraph.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.LineageGraph;
  return proto.ml_metadata.LineageGraph.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.LineageGraph} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.LineageGraph}
 */
proto.ml_metadata.LineageGraph.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ArtifactType;
      reader.readMessage(value,proto.ml_metadata.ArtifactType.deserializeBinaryFromReader);
      msg.addArtifactTypes(value);
      break;
    case 2:
      var value = new proto.ml_metadata.ExecutionType;
      reader.readMessage(value,proto.ml_metadata.ExecutionType.deserializeBinaryFromReader);
      msg.addExecutionTypes(value);
      break;
    case 3:
      var value = new proto.ml_metadata.ContextType;
      reader.readMessage(value,proto.ml_metadata.ContextType.deserializeBinaryFromReader);
      msg.addContextTypes(value);
      break;
    case 4:
      var value = new proto.ml_metadata.Artifact;
      reader.readMessage(value,proto.ml_metadata.Artifact.deserializeBinaryFromReader);
      msg.addArtifacts(value);
      break;
    case 5:
      var value = new proto.ml_metadata.Execution;
      reader.readMessage(value,proto.ml_metadata.Execution.deserializeBinaryFromReader);
      msg.addExecutions(value);
      break;
    case 6:
      var value = new proto.ml_metadata.Context;
      reader.readMessage(value,proto.ml_metadata.Context.deserializeBinaryFromReader);
      msg.addContexts(value);
      break;
    case 7:
      var value = new proto.ml_metadata.Event;
      reader.readMessage(value,proto.ml_metadata.Event.deserializeBinaryFromReader);
      msg.addEvents(value);
      break;
    case 8:
      var value = new proto.ml_metadata.Attribution;
      reader.readMessage(value,proto.ml_metadata.Attribution.deserializeBinaryFromReader);
      msg.addAttributions(value);
      break;
    case 9:
      var value = new proto.ml_metadata.Association;
      reader.readMessage(value,proto.ml_metadata.Association.deserializeBinaryFromReader);
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
proto.ml_metadata.LineageGraph.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.LineageGraph.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.LineageGraph} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.LineageGraph.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.ml_metadata.ArtifactType.serializeBinaryToWriter
    );
  }
  f = message.getExecutionTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      proto.ml_metadata.ExecutionType.serializeBinaryToWriter
    );
  }
  f = message.getContextTypesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      proto.ml_metadata.ContextType.serializeBinaryToWriter
    );
  }
  f = message.getArtifactsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      4,
      f,
      proto.ml_metadata.Artifact.serializeBinaryToWriter
    );
  }
  f = message.getExecutionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      5,
      f,
      proto.ml_metadata.Execution.serializeBinaryToWriter
    );
  }
  f = message.getContextsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      6,
      f,
      proto.ml_metadata.Context.serializeBinaryToWriter
    );
  }
  f = message.getEventsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      7,
      f,
      proto.ml_metadata.Event.serializeBinaryToWriter
    );
  }
  f = message.getAttributionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      8,
      f,
      proto.ml_metadata.Attribution.serializeBinaryToWriter
    );
  }
  f = message.getAssociationsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      9,
      f,
      proto.ml_metadata.Association.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ArtifactType artifact_types = 1;
 * @return {!Array<!proto.ml_metadata.ArtifactType>}
 */
proto.ml_metadata.LineageGraph.prototype.getArtifactTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ArtifactType>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.ArtifactType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ArtifactType>} value
 * @return {!proto.ml_metadata.LineageGraph} returns this
*/
proto.ml_metadata.LineageGraph.prototype.setArtifactTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ArtifactType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.LineageGraph.prototype.addArtifactTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ArtifactType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.LineageGraph} returns this
 */
proto.ml_metadata.LineageGraph.prototype.clearArtifactTypesList = function() {
  return this.setArtifactTypesList([]);
};


/**
 * repeated ExecutionType execution_types = 2;
 * @return {!Array<!proto.ml_metadata.ExecutionType>}
 */
proto.ml_metadata.LineageGraph.prototype.getExecutionTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ExecutionType>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.ExecutionType, 2));
};


/**
 * @param {!Array<!proto.ml_metadata.ExecutionType>} value
 * @return {!proto.ml_metadata.LineageGraph} returns this
*/
proto.ml_metadata.LineageGraph.prototype.setExecutionTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.ml_metadata.ExecutionType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ExecutionType}
 */
proto.ml_metadata.LineageGraph.prototype.addExecutionTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.ml_metadata.ExecutionType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.LineageGraph} returns this
 */
proto.ml_metadata.LineageGraph.prototype.clearExecutionTypesList = function() {
  return this.setExecutionTypesList([]);
};


/**
 * repeated ContextType context_types = 3;
 * @return {!Array<!proto.ml_metadata.ContextType>}
 */
proto.ml_metadata.LineageGraph.prototype.getContextTypesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ContextType>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.ContextType, 3));
};


/**
 * @param {!Array<!proto.ml_metadata.ContextType>} value
 * @return {!proto.ml_metadata.LineageGraph} returns this
*/
proto.ml_metadata.LineageGraph.prototype.setContextTypesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.ml_metadata.ContextType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ContextType}
 */
proto.ml_metadata.LineageGraph.prototype.addContextTypes = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.ml_metadata.ContextType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.LineageGraph} returns this
 */
proto.ml_metadata.LineageGraph.prototype.clearContextTypesList = function() {
  return this.setContextTypesList([]);
};


/**
 * repeated Artifact artifacts = 4;
 * @return {!Array<!proto.ml_metadata.Artifact>}
 */
proto.ml_metadata.LineageGraph.prototype.getArtifactsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Artifact>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.Artifact, 4));
};


/**
 * @param {!Array<!proto.ml_metadata.Artifact>} value
 * @return {!proto.ml_metadata.LineageGraph} returns this
*/
proto.ml_metadata.LineageGraph.prototype.setArtifactsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 4, value);
};


/**
 * @param {!proto.ml_metadata.Artifact=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Artifact}
 */
proto.ml_metadata.LineageGraph.prototype.addArtifacts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 4, opt_value, proto.ml_metadata.Artifact, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.LineageGraph} returns this
 */
proto.ml_metadata.LineageGraph.prototype.clearArtifactsList = function() {
  return this.setArtifactsList([]);
};


/**
 * repeated Execution executions = 5;
 * @return {!Array<!proto.ml_metadata.Execution>}
 */
proto.ml_metadata.LineageGraph.prototype.getExecutionsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Execution>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.Execution, 5));
};


/**
 * @param {!Array<!proto.ml_metadata.Execution>} value
 * @return {!proto.ml_metadata.LineageGraph} returns this
*/
proto.ml_metadata.LineageGraph.prototype.setExecutionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 5, value);
};


/**
 * @param {!proto.ml_metadata.Execution=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Execution}
 */
proto.ml_metadata.LineageGraph.prototype.addExecutions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 5, opt_value, proto.ml_metadata.Execution, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.LineageGraph} returns this
 */
proto.ml_metadata.LineageGraph.prototype.clearExecutionsList = function() {
  return this.setExecutionsList([]);
};


/**
 * repeated Context contexts = 6;
 * @return {!Array<!proto.ml_metadata.Context>}
 */
proto.ml_metadata.LineageGraph.prototype.getContextsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Context>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.Context, 6));
};


/**
 * @param {!Array<!proto.ml_metadata.Context>} value
 * @return {!proto.ml_metadata.LineageGraph} returns this
*/
proto.ml_metadata.LineageGraph.prototype.setContextsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 6, value);
};


/**
 * @param {!proto.ml_metadata.Context=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Context}
 */
proto.ml_metadata.LineageGraph.prototype.addContexts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 6, opt_value, proto.ml_metadata.Context, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.LineageGraph} returns this
 */
proto.ml_metadata.LineageGraph.prototype.clearContextsList = function() {
  return this.setContextsList([]);
};


/**
 * repeated Event events = 7;
 * @return {!Array<!proto.ml_metadata.Event>}
 */
proto.ml_metadata.LineageGraph.prototype.getEventsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Event>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.Event, 7));
};


/**
 * @param {!Array<!proto.ml_metadata.Event>} value
 * @return {!proto.ml_metadata.LineageGraph} returns this
*/
proto.ml_metadata.LineageGraph.prototype.setEventsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 7, value);
};


/**
 * @param {!proto.ml_metadata.Event=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Event}
 */
proto.ml_metadata.LineageGraph.prototype.addEvents = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 7, opt_value, proto.ml_metadata.Event, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.LineageGraph} returns this
 */
proto.ml_metadata.LineageGraph.prototype.clearEventsList = function() {
  return this.setEventsList([]);
};


/**
 * repeated Attribution attributions = 8;
 * @return {!Array<!proto.ml_metadata.Attribution>}
 */
proto.ml_metadata.LineageGraph.prototype.getAttributionsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Attribution>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.Attribution, 8));
};


/**
 * @param {!Array<!proto.ml_metadata.Attribution>} value
 * @return {!proto.ml_metadata.LineageGraph} returns this
*/
proto.ml_metadata.LineageGraph.prototype.setAttributionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 8, value);
};


/**
 * @param {!proto.ml_metadata.Attribution=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Attribution}
 */
proto.ml_metadata.LineageGraph.prototype.addAttributions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 8, opt_value, proto.ml_metadata.Attribution, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.LineageGraph} returns this
 */
proto.ml_metadata.LineageGraph.prototype.clearAttributionsList = function() {
  return this.setAttributionsList([]);
};


/**
 * repeated Association associations = 9;
 * @return {!Array<!proto.ml_metadata.Association>}
 */
proto.ml_metadata.LineageGraph.prototype.getAssociationsList = function() {
  return /** @type{!Array<!proto.ml_metadata.Association>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.Association, 9));
};


/**
 * @param {!Array<!proto.ml_metadata.Association>} value
 * @return {!proto.ml_metadata.LineageGraph} returns this
*/
proto.ml_metadata.LineageGraph.prototype.setAssociationsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 9, value);
};


/**
 * @param {!proto.ml_metadata.Association=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.Association}
 */
proto.ml_metadata.LineageGraph.prototype.addAssociations = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 9, opt_value, proto.ml_metadata.Association, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.LineageGraph} returns this
 */
proto.ml_metadata.LineageGraph.prototype.clearAssociationsList = function() {
  return this.setAssociationsList([]);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_metadata.ArtifactStructType.oneofGroups_ = [[1,2,3,4,5,6,7,8]];

/**
 * @enum {number}
 */
proto.ml_metadata.ArtifactStructType.KindCase = {
  KIND_NOT_SET: 0,
  SIMPLE: 1,
  UNION_TYPE: 2,
  INTERSECTION: 3,
  LIST: 4,
  NONE: 5,
  ANY: 6,
  TUPLE: 7,
  DICT: 8
};

/**
 * @return {proto.ml_metadata.ArtifactStructType.KindCase}
 */
proto.ml_metadata.ArtifactStructType.prototype.getKindCase = function() {
  return /** @type {proto.ml_metadata.ArtifactStructType.KindCase} */(jspb.Message.computeOneofCase(this, proto.ml_metadata.ArtifactStructType.oneofGroups_[0]));
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
proto.ml_metadata.ArtifactStructType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ArtifactStructType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ArtifactStructType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactStructType.toObject = function(includeInstance, msg) {
  var f, obj = {
    simple: (f = msg.getSimple()) && proto.ml_metadata.ArtifactType.toObject(includeInstance, f),
    unionType: (f = msg.getUnionType()) && proto.ml_metadata.UnionArtifactStructType.toObject(includeInstance, f),
    intersection: (f = msg.getIntersection()) && proto.ml_metadata.IntersectionArtifactStructType.toObject(includeInstance, f),
    list: (f = msg.getList()) && proto.ml_metadata.ListArtifactStructType.toObject(includeInstance, f),
    none: (f = msg.getNone()) && proto.ml_metadata.NoneArtifactStructType.toObject(includeInstance, f),
    any: (f = msg.getAny()) && proto.ml_metadata.AnyArtifactStructType.toObject(includeInstance, f),
    tuple: (f = msg.getTuple()) && proto.ml_metadata.TupleArtifactStructType.toObject(includeInstance, f),
    dict: (f = msg.getDict()) && proto.ml_metadata.DictArtifactStructType.toObject(includeInstance, f)
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
 * @return {!proto.ml_metadata.ArtifactStructType}
 */
proto.ml_metadata.ArtifactStructType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ArtifactStructType;
  return proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ArtifactStructType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ArtifactStructType}
 */
proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ArtifactType;
      reader.readMessage(value,proto.ml_metadata.ArtifactType.deserializeBinaryFromReader);
      msg.setSimple(value);
      break;
    case 2:
      var value = new proto.ml_metadata.UnionArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.UnionArtifactStructType.deserializeBinaryFromReader);
      msg.setUnionType(value);
      break;
    case 3:
      var value = new proto.ml_metadata.IntersectionArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.IntersectionArtifactStructType.deserializeBinaryFromReader);
      msg.setIntersection(value);
      break;
    case 4:
      var value = new proto.ml_metadata.ListArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.ListArtifactStructType.deserializeBinaryFromReader);
      msg.setList(value);
      break;
    case 5:
      var value = new proto.ml_metadata.NoneArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.NoneArtifactStructType.deserializeBinaryFromReader);
      msg.setNone(value);
      break;
    case 6:
      var value = new proto.ml_metadata.AnyArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.AnyArtifactStructType.deserializeBinaryFromReader);
      msg.setAny(value);
      break;
    case 7:
      var value = new proto.ml_metadata.TupleArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.TupleArtifactStructType.deserializeBinaryFromReader);
      msg.setTuple(value);
      break;
    case 8:
      var value = new proto.ml_metadata.DictArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.DictArtifactStructType.deserializeBinaryFromReader);
      msg.setDict(value);
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
proto.ml_metadata.ArtifactStructType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ArtifactStructType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSimple();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_metadata.ArtifactType.serializeBinaryToWriter
    );
  }
  f = message.getUnionType();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_metadata.UnionArtifactStructType.serializeBinaryToWriter
    );
  }
  f = message.getIntersection();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_metadata.IntersectionArtifactStructType.serializeBinaryToWriter
    );
  }
  f = message.getList();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.ml_metadata.ListArtifactStructType.serializeBinaryToWriter
    );
  }
  f = message.getNone();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      proto.ml_metadata.NoneArtifactStructType.serializeBinaryToWriter
    );
  }
  f = message.getAny();
  if (f != null) {
    writer.writeMessage(
      6,
      f,
      proto.ml_metadata.AnyArtifactStructType.serializeBinaryToWriter
    );
  }
  f = message.getTuple();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      proto.ml_metadata.TupleArtifactStructType.serializeBinaryToWriter
    );
  }
  f = message.getDict();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      proto.ml_metadata.DictArtifactStructType.serializeBinaryToWriter
    );
  }
};


/**
 * optional ArtifactType simple = 1;
 * @return {?proto.ml_metadata.ArtifactType}
 */
proto.ml_metadata.ArtifactStructType.prototype.getSimple = function() {
  return /** @type{?proto.ml_metadata.ArtifactType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ArtifactType, 1));
};


/**
 * @param {?proto.ml_metadata.ArtifactType|undefined} value
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
*/
proto.ml_metadata.ArtifactStructType.prototype.setSimple = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.ml_metadata.ArtifactStructType.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
 */
proto.ml_metadata.ArtifactStructType.prototype.clearSimple = function() {
  return this.setSimple(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStructType.prototype.hasSimple = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional UnionArtifactStructType union_type = 2;
 * @return {?proto.ml_metadata.UnionArtifactStructType}
 */
proto.ml_metadata.ArtifactStructType.prototype.getUnionType = function() {
  return /** @type{?proto.ml_metadata.UnionArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.UnionArtifactStructType, 2));
};


/**
 * @param {?proto.ml_metadata.UnionArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
*/
proto.ml_metadata.ArtifactStructType.prototype.setUnionType = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.ml_metadata.ArtifactStructType.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
 */
proto.ml_metadata.ArtifactStructType.prototype.clearUnionType = function() {
  return this.setUnionType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStructType.prototype.hasUnionType = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional IntersectionArtifactStructType intersection = 3;
 * @return {?proto.ml_metadata.IntersectionArtifactStructType}
 */
proto.ml_metadata.ArtifactStructType.prototype.getIntersection = function() {
  return /** @type{?proto.ml_metadata.IntersectionArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.IntersectionArtifactStructType, 3));
};


/**
 * @param {?proto.ml_metadata.IntersectionArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
*/
proto.ml_metadata.ArtifactStructType.prototype.setIntersection = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.ml_metadata.ArtifactStructType.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
 */
proto.ml_metadata.ArtifactStructType.prototype.clearIntersection = function() {
  return this.setIntersection(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStructType.prototype.hasIntersection = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional ListArtifactStructType list = 4;
 * @return {?proto.ml_metadata.ListArtifactStructType}
 */
proto.ml_metadata.ArtifactStructType.prototype.getList = function() {
  return /** @type{?proto.ml_metadata.ListArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ListArtifactStructType, 4));
};


/**
 * @param {?proto.ml_metadata.ListArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
*/
proto.ml_metadata.ArtifactStructType.prototype.setList = function(value) {
  return jspb.Message.setOneofWrapperField(this, 4, proto.ml_metadata.ArtifactStructType.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
 */
proto.ml_metadata.ArtifactStructType.prototype.clearList = function() {
  return this.setList(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStructType.prototype.hasList = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional NoneArtifactStructType none = 5;
 * @return {?proto.ml_metadata.NoneArtifactStructType}
 */
proto.ml_metadata.ArtifactStructType.prototype.getNone = function() {
  return /** @type{?proto.ml_metadata.NoneArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.NoneArtifactStructType, 5));
};


/**
 * @param {?proto.ml_metadata.NoneArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
*/
proto.ml_metadata.ArtifactStructType.prototype.setNone = function(value) {
  return jspb.Message.setOneofWrapperField(this, 5, proto.ml_metadata.ArtifactStructType.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
 */
proto.ml_metadata.ArtifactStructType.prototype.clearNone = function() {
  return this.setNone(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStructType.prototype.hasNone = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional AnyArtifactStructType any = 6;
 * @return {?proto.ml_metadata.AnyArtifactStructType}
 */
proto.ml_metadata.ArtifactStructType.prototype.getAny = function() {
  return /** @type{?proto.ml_metadata.AnyArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.AnyArtifactStructType, 6));
};


/**
 * @param {?proto.ml_metadata.AnyArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
*/
proto.ml_metadata.ArtifactStructType.prototype.setAny = function(value) {
  return jspb.Message.setOneofWrapperField(this, 6, proto.ml_metadata.ArtifactStructType.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
 */
proto.ml_metadata.ArtifactStructType.prototype.clearAny = function() {
  return this.setAny(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStructType.prototype.hasAny = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional TupleArtifactStructType tuple = 7;
 * @return {?proto.ml_metadata.TupleArtifactStructType}
 */
proto.ml_metadata.ArtifactStructType.prototype.getTuple = function() {
  return /** @type{?proto.ml_metadata.TupleArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.TupleArtifactStructType, 7));
};


/**
 * @param {?proto.ml_metadata.TupleArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
*/
proto.ml_metadata.ArtifactStructType.prototype.setTuple = function(value) {
  return jspb.Message.setOneofWrapperField(this, 7, proto.ml_metadata.ArtifactStructType.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
 */
proto.ml_metadata.ArtifactStructType.prototype.clearTuple = function() {
  return this.setTuple(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStructType.prototype.hasTuple = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional DictArtifactStructType dict = 8;
 * @return {?proto.ml_metadata.DictArtifactStructType}
 */
proto.ml_metadata.ArtifactStructType.prototype.getDict = function() {
  return /** @type{?proto.ml_metadata.DictArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.DictArtifactStructType, 8));
};


/**
 * @param {?proto.ml_metadata.DictArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
*/
proto.ml_metadata.ArtifactStructType.prototype.setDict = function(value) {
  return jspb.Message.setOneofWrapperField(this, 8, proto.ml_metadata.ArtifactStructType.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ArtifactStructType} returns this
 */
proto.ml_metadata.ArtifactStructType.prototype.clearDict = function() {
  return this.setDict(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ArtifactStructType.prototype.hasDict = function() {
  return jspb.Message.getField(this, 8) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.UnionArtifactStructType.repeatedFields_ = [1];



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
proto.ml_metadata.UnionArtifactStructType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.UnionArtifactStructType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.UnionArtifactStructType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.UnionArtifactStructType.toObject = function(includeInstance, msg) {
  var f, obj = {
    candidatesList: jspb.Message.toObjectList(msg.getCandidatesList(),
    proto.ml_metadata.ArtifactStructType.toObject, includeInstance)
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
 * @return {!proto.ml_metadata.UnionArtifactStructType}
 */
proto.ml_metadata.UnionArtifactStructType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.UnionArtifactStructType;
  return proto.ml_metadata.UnionArtifactStructType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.UnionArtifactStructType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.UnionArtifactStructType}
 */
proto.ml_metadata.UnionArtifactStructType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader);
      msg.addCandidates(value);
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
proto.ml_metadata.UnionArtifactStructType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.UnionArtifactStructType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.UnionArtifactStructType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.UnionArtifactStructType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCandidatesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ArtifactStructType candidates = 1;
 * @return {!Array<!proto.ml_metadata.ArtifactStructType>}
 */
proto.ml_metadata.UnionArtifactStructType.prototype.getCandidatesList = function() {
  return /** @type{!Array<!proto.ml_metadata.ArtifactStructType>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.ArtifactStructType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ArtifactStructType>} value
 * @return {!proto.ml_metadata.UnionArtifactStructType} returns this
*/
proto.ml_metadata.UnionArtifactStructType.prototype.setCandidatesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ArtifactStructType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ArtifactStructType}
 */
proto.ml_metadata.UnionArtifactStructType.prototype.addCandidates = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ArtifactStructType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.UnionArtifactStructType} returns this
 */
proto.ml_metadata.UnionArtifactStructType.prototype.clearCandidatesList = function() {
  return this.setCandidatesList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.IntersectionArtifactStructType.repeatedFields_ = [1];



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
proto.ml_metadata.IntersectionArtifactStructType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.IntersectionArtifactStructType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.IntersectionArtifactStructType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.IntersectionArtifactStructType.toObject = function(includeInstance, msg) {
  var f, obj = {
    constraintsList: jspb.Message.toObjectList(msg.getConstraintsList(),
    proto.ml_metadata.ArtifactStructType.toObject, includeInstance)
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
 * @return {!proto.ml_metadata.IntersectionArtifactStructType}
 */
proto.ml_metadata.IntersectionArtifactStructType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.IntersectionArtifactStructType;
  return proto.ml_metadata.IntersectionArtifactStructType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.IntersectionArtifactStructType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.IntersectionArtifactStructType}
 */
proto.ml_metadata.IntersectionArtifactStructType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader);
      msg.addConstraints(value);
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
proto.ml_metadata.IntersectionArtifactStructType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.IntersectionArtifactStructType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.IntersectionArtifactStructType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.IntersectionArtifactStructType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getConstraintsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ArtifactStructType constraints = 1;
 * @return {!Array<!proto.ml_metadata.ArtifactStructType>}
 */
proto.ml_metadata.IntersectionArtifactStructType.prototype.getConstraintsList = function() {
  return /** @type{!Array<!proto.ml_metadata.ArtifactStructType>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.ArtifactStructType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ArtifactStructType>} value
 * @return {!proto.ml_metadata.IntersectionArtifactStructType} returns this
*/
proto.ml_metadata.IntersectionArtifactStructType.prototype.setConstraintsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ArtifactStructType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ArtifactStructType}
 */
proto.ml_metadata.IntersectionArtifactStructType.prototype.addConstraints = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ArtifactStructType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.IntersectionArtifactStructType} returns this
 */
proto.ml_metadata.IntersectionArtifactStructType.prototype.clearConstraintsList = function() {
  return this.setConstraintsList([]);
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
proto.ml_metadata.ListArtifactStructType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ListArtifactStructType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ListArtifactStructType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ListArtifactStructType.toObject = function(includeInstance, msg) {
  var f, obj = {
    element: (f = msg.getElement()) && proto.ml_metadata.ArtifactStructType.toObject(includeInstance, f)
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
 * @return {!proto.ml_metadata.ListArtifactStructType}
 */
proto.ml_metadata.ListArtifactStructType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ListArtifactStructType;
  return proto.ml_metadata.ListArtifactStructType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ListArtifactStructType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ListArtifactStructType}
 */
proto.ml_metadata.ListArtifactStructType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader);
      msg.setElement(value);
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
proto.ml_metadata.ListArtifactStructType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ListArtifactStructType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ListArtifactStructType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ListArtifactStructType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getElement();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter
    );
  }
};


/**
 * optional ArtifactStructType element = 1;
 * @return {?proto.ml_metadata.ArtifactStructType}
 */
proto.ml_metadata.ListArtifactStructType.prototype.getElement = function() {
  return /** @type{?proto.ml_metadata.ArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ArtifactStructType, 1));
};


/**
 * @param {?proto.ml_metadata.ArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.ListArtifactStructType} returns this
*/
proto.ml_metadata.ListArtifactStructType.prototype.setElement = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ListArtifactStructType} returns this
 */
proto.ml_metadata.ListArtifactStructType.prototype.clearElement = function() {
  return this.setElement(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListArtifactStructType.prototype.hasElement = function() {
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
proto.ml_metadata.NoneArtifactStructType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.NoneArtifactStructType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.NoneArtifactStructType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.NoneArtifactStructType.toObject = function(includeInstance, msg) {
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
 * @return {!proto.ml_metadata.NoneArtifactStructType}
 */
proto.ml_metadata.NoneArtifactStructType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.NoneArtifactStructType;
  return proto.ml_metadata.NoneArtifactStructType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.NoneArtifactStructType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.NoneArtifactStructType}
 */
proto.ml_metadata.NoneArtifactStructType.deserializeBinaryFromReader = function(msg, reader) {
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
proto.ml_metadata.NoneArtifactStructType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.NoneArtifactStructType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.NoneArtifactStructType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.NoneArtifactStructType.serializeBinaryToWriter = function(message, writer) {
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
proto.ml_metadata.AnyArtifactStructType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.AnyArtifactStructType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.AnyArtifactStructType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.AnyArtifactStructType.toObject = function(includeInstance, msg) {
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
 * @return {!proto.ml_metadata.AnyArtifactStructType}
 */
proto.ml_metadata.AnyArtifactStructType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.AnyArtifactStructType;
  return proto.ml_metadata.AnyArtifactStructType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.AnyArtifactStructType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.AnyArtifactStructType}
 */
proto.ml_metadata.AnyArtifactStructType.deserializeBinaryFromReader = function(msg, reader) {
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
proto.ml_metadata.AnyArtifactStructType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.AnyArtifactStructType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.AnyArtifactStructType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.AnyArtifactStructType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.TupleArtifactStructType.repeatedFields_ = [1];



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
proto.ml_metadata.TupleArtifactStructType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.TupleArtifactStructType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.TupleArtifactStructType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.TupleArtifactStructType.toObject = function(includeInstance, msg) {
  var f, obj = {
    elementsList: jspb.Message.toObjectList(msg.getElementsList(),
    proto.ml_metadata.ArtifactStructType.toObject, includeInstance)
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
 * @return {!proto.ml_metadata.TupleArtifactStructType}
 */
proto.ml_metadata.TupleArtifactStructType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.TupleArtifactStructType;
  return proto.ml_metadata.TupleArtifactStructType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.TupleArtifactStructType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.TupleArtifactStructType}
 */
proto.ml_metadata.TupleArtifactStructType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader);
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
proto.ml_metadata.TupleArtifactStructType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.TupleArtifactStructType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.TupleArtifactStructType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.TupleArtifactStructType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getElementsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ArtifactStructType elements = 1;
 * @return {!Array<!proto.ml_metadata.ArtifactStructType>}
 */
proto.ml_metadata.TupleArtifactStructType.prototype.getElementsList = function() {
  return /** @type{!Array<!proto.ml_metadata.ArtifactStructType>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_metadata.ArtifactStructType, 1));
};


/**
 * @param {!Array<!proto.ml_metadata.ArtifactStructType>} value
 * @return {!proto.ml_metadata.TupleArtifactStructType} returns this
*/
proto.ml_metadata.TupleArtifactStructType.prototype.setElementsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_metadata.ArtifactStructType=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ArtifactStructType}
 */
proto.ml_metadata.TupleArtifactStructType.prototype.addElements = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_metadata.ArtifactStructType, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.TupleArtifactStructType} returns this
 */
proto.ml_metadata.TupleArtifactStructType.prototype.clearElementsList = function() {
  return this.setElementsList([]);
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
proto.ml_metadata.DictArtifactStructType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.DictArtifactStructType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.DictArtifactStructType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.DictArtifactStructType.toObject = function(includeInstance, msg) {
  var f, obj = {
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, proto.ml_metadata.ArtifactStructType.toObject) : [],
    noneTypeNotRequired: (f = jspb.Message.getBooleanField(msg, 2)) == null ? undefined : f,
    extraPropertiesType: (f = msg.getExtraPropertiesType()) && proto.ml_metadata.ArtifactStructType.toObject(includeInstance, f)
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
 * @return {!proto.ml_metadata.DictArtifactStructType}
 */
proto.ml_metadata.DictArtifactStructType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.DictArtifactStructType;
  return proto.ml_metadata.DictArtifactStructType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.DictArtifactStructType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.DictArtifactStructType}
 */
proto.ml_metadata.DictArtifactStructType.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader, "", new proto.ml_metadata.ArtifactStructType());
         });
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setNoneTypeNotRequired(value);
      break;
    case 3:
      var value = new proto.ml_metadata.ArtifactStructType;
      reader.readMessage(value,proto.ml_metadata.ArtifactStructType.deserializeBinaryFromReader);
      msg.setExtraPropertiesType(value);
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
proto.ml_metadata.DictArtifactStructType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.DictArtifactStructType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.DictArtifactStructType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.DictArtifactStructType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter);
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeBool(
      2,
      f
    );
  }
  f = message.getExtraPropertiesType();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_metadata.ArtifactStructType.serializeBinaryToWriter
    );
  }
};


/**
 * map<string, ArtifactStructType> properties = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_metadata.ArtifactStructType>}
 */
proto.ml_metadata.DictArtifactStructType.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_metadata.ArtifactStructType>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_metadata.ArtifactStructType));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_metadata.DictArtifactStructType} returns this
 */
proto.ml_metadata.DictArtifactStructType.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * optional bool none_type_not_required = 2;
 * @return {boolean}
 */
proto.ml_metadata.DictArtifactStructType.prototype.getNoneTypeNotRequired = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.DictArtifactStructType} returns this
 */
proto.ml_metadata.DictArtifactStructType.prototype.setNoneTypeNotRequired = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.DictArtifactStructType} returns this
 */
proto.ml_metadata.DictArtifactStructType.prototype.clearNoneTypeNotRequired = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.DictArtifactStructType.prototype.hasNoneTypeNotRequired = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional ArtifactStructType extra_properties_type = 3;
 * @return {?proto.ml_metadata.ArtifactStructType}
 */
proto.ml_metadata.DictArtifactStructType.prototype.getExtraPropertiesType = function() {
  return /** @type{?proto.ml_metadata.ArtifactStructType} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ArtifactStructType, 3));
};


/**
 * @param {?proto.ml_metadata.ArtifactStructType|undefined} value
 * @return {!proto.ml_metadata.DictArtifactStructType} returns this
*/
proto.ml_metadata.DictArtifactStructType.prototype.setExtraPropertiesType = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.DictArtifactStructType} returns this
 */
proto.ml_metadata.DictArtifactStructType.prototype.clearExtraPropertiesType = function() {
  return this.setExtraPropertiesType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.DictArtifactStructType.prototype.hasExtraPropertiesType = function() {
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
proto.ml_metadata.FakeDatabaseConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.FakeDatabaseConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.FakeDatabaseConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.FakeDatabaseConfig.toObject = function(includeInstance, msg) {
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
 * @return {!proto.ml_metadata.FakeDatabaseConfig}
 */
proto.ml_metadata.FakeDatabaseConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.FakeDatabaseConfig;
  return proto.ml_metadata.FakeDatabaseConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.FakeDatabaseConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.FakeDatabaseConfig}
 */
proto.ml_metadata.FakeDatabaseConfig.deserializeBinaryFromReader = function(msg, reader) {
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
proto.ml_metadata.FakeDatabaseConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.FakeDatabaseConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.FakeDatabaseConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.FakeDatabaseConfig.serializeBinaryToWriter = function(message, writer) {
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
proto.ml_metadata.MySQLDatabaseConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.MySQLDatabaseConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.MySQLDatabaseConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MySQLDatabaseConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    host: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    port: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    database: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    user: (f = jspb.Message.getField(msg, 4)) == null ? undefined : f,
    password: (f = jspb.Message.getField(msg, 5)) == null ? undefined : f,
    socket: (f = jspb.Message.getField(msg, 6)) == null ? undefined : f,
    sslOptions: (f = msg.getSslOptions()) && proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.toObject(includeInstance, f),
    skipDbCreation: (f = jspb.Message.getBooleanField(msg, 8)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.MySQLDatabaseConfig}
 */
proto.ml_metadata.MySQLDatabaseConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.MySQLDatabaseConfig;
  return proto.ml_metadata.MySQLDatabaseConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.MySQLDatabaseConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig}
 */
proto.ml_metadata.MySQLDatabaseConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setHost(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setPort(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setDatabase(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setUser(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setPassword(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setSocket(value);
      break;
    case 7:
      var value = new proto.ml_metadata.MySQLDatabaseConfig.SSLOptions;
      reader.readMessage(value,proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.deserializeBinaryFromReader);
      msg.setSslOptions(value);
      break;
    case 8:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSkipDbCreation(value);
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
proto.ml_metadata.MySQLDatabaseConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.MySQLDatabaseConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.MySQLDatabaseConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MySQLDatabaseConfig.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeUint32(
      2,
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
  f = /** @type {string} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeString(
      4,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeString(
      5,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 6));
  if (f != null) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getSslOptions();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.serializeBinaryToWriter
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 8));
  if (f != null) {
    writer.writeBool(
      8,
      f
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
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.toObject = function(includeInstance, msg) {
  var f, obj = {
    key: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    cert: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    ca: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    capath: (f = jspb.Message.getField(msg, 4)) == null ? undefined : f,
    cipher: (f = jspb.Message.getField(msg, 5)) == null ? undefined : f,
    verifyServerCert: (f = jspb.Message.getBooleanField(msg, 6)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.MySQLDatabaseConfig.SSLOptions;
  return proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setKey(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setCert(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setCa(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setCapath(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setCipher(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setVerifyServerCert(value);
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
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.serializeBinaryToWriter = function(message, writer) {
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
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeString(
      4,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeString(
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
 * optional string key = 1;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.getKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.setKey = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.clearKey = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.hasKey = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string cert = 2;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.getCert = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.setCert = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.clearCert = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.hasCert = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string ca = 3;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.getCa = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.setCa = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.clearCa = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.hasCa = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string capath = 4;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.getCapath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.setCapath = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.clearCapath = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.hasCapath = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional string cipher = 5;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.getCipher = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.setCipher = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.clearCipher = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.hasCipher = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional bool verify_server_cert = 6;
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.getVerifyServerCert = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.setVerifyServerCert = function(value) {
  return jspb.Message.setField(this, 6, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.clearVerifyServerCert = function() {
  return jspb.Message.setField(this, 6, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.SSLOptions.prototype.hasVerifyServerCert = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional string host = 1;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.getHost = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.setHost = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.clearHost = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.hasHost = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional uint32 port = 2;
 * @return {number}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.getPort = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.setPort = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.clearPort = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.hasPort = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string database = 3;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.getDatabase = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.setDatabase = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.clearDatabase = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.hasDatabase = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string user = 4;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.getUser = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.setUser = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.clearUser = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.hasUser = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional string password = 5;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.getPassword = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.setPassword = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.clearPassword = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.hasPassword = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional string socket = 6;
 * @return {string}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.getSocket = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.setSocket = function(value) {
  return jspb.Message.setField(this, 6, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.clearSocket = function() {
  return jspb.Message.setField(this, 6, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.hasSocket = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional SSLOptions ssl_options = 7;
 * @return {?proto.ml_metadata.MySQLDatabaseConfig.SSLOptions}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.getSslOptions = function() {
  return /** @type{?proto.ml_metadata.MySQLDatabaseConfig.SSLOptions} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.MySQLDatabaseConfig.SSLOptions, 7));
};


/**
 * @param {?proto.ml_metadata.MySQLDatabaseConfig.SSLOptions|undefined} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
*/
proto.ml_metadata.MySQLDatabaseConfig.prototype.setSslOptions = function(value) {
  return jspb.Message.setWrapperField(this, 7, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.clearSslOptions = function() {
  return this.setSslOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.hasSslOptions = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional bool skip_db_creation = 8;
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.getSkipDbCreation = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 8, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.setSkipDbCreation = function(value) {
  return jspb.Message.setField(this, 8, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MySQLDatabaseConfig} returns this
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.clearSkipDbCreation = function() {
  return jspb.Message.setField(this, 8, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MySQLDatabaseConfig.prototype.hasSkipDbCreation = function() {
  return jspb.Message.getField(this, 8) != null;
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
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.SqliteMetadataSourceConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.SqliteMetadataSourceConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.SqliteMetadataSourceConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    filenameUri: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    connectionMode: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.SqliteMetadataSourceConfig}
 */
proto.ml_metadata.SqliteMetadataSourceConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.SqliteMetadataSourceConfig;
  return proto.ml_metadata.SqliteMetadataSourceConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.SqliteMetadataSourceConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.SqliteMetadataSourceConfig}
 */
proto.ml_metadata.SqliteMetadataSourceConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setFilenameUri(value);
      break;
    case 2:
      var value = /** @type {!proto.ml_metadata.SqliteMetadataSourceConfig.ConnectionMode} */ (reader.readEnum());
      msg.setConnectionMode(value);
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
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.SqliteMetadataSourceConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.SqliteMetadataSourceConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.SqliteMetadataSourceConfig.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {!proto.ml_metadata.SqliteMetadataSourceConfig.ConnectionMode} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeEnum(
      2,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.ml_metadata.SqliteMetadataSourceConfig.ConnectionMode = {
  UNKNOWN: 0,
  READONLY: 1,
  READWRITE: 2,
  READWRITE_OPENCREATE: 3
};

/**
 * optional string filename_uri = 1;
 * @return {string}
 */
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.getFilenameUri = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.SqliteMetadataSourceConfig} returns this
 */
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.setFilenameUri = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.SqliteMetadataSourceConfig} returns this
 */
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.clearFilenameUri = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.hasFilenameUri = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ConnectionMode connection_mode = 2;
 * @return {!proto.ml_metadata.SqliteMetadataSourceConfig.ConnectionMode}
 */
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.getConnectionMode = function() {
  return /** @type {!proto.ml_metadata.SqliteMetadataSourceConfig.ConnectionMode} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {!proto.ml_metadata.SqliteMetadataSourceConfig.ConnectionMode} value
 * @return {!proto.ml_metadata.SqliteMetadataSourceConfig} returns this
 */
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.setConnectionMode = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.SqliteMetadataSourceConfig} returns this
 */
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.clearConnectionMode = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.SqliteMetadataSourceConfig.prototype.hasConnectionMode = function() {
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
proto.ml_metadata.MigrationOptions.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.MigrationOptions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.MigrationOptions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MigrationOptions.toObject = function(includeInstance, msg) {
  var f, obj = {
    enableUpgradeMigration: (f = jspb.Message.getBooleanField(msg, 3)) == null ? undefined : f,
    downgradeToSchemaVersion: jspb.Message.getFieldWithDefault(msg, 2, -1)
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
 * @return {!proto.ml_metadata.MigrationOptions}
 */
proto.ml_metadata.MigrationOptions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.MigrationOptions;
  return proto.ml_metadata.MigrationOptions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.MigrationOptions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.MigrationOptions}
 */
proto.ml_metadata.MigrationOptions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setEnableUpgradeMigration(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setDowngradeToSchemaVersion(value);
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
proto.ml_metadata.MigrationOptions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.MigrationOptions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.MigrationOptions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MigrationOptions.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {boolean} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeBool(
      3,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
};


/**
 * optional bool enable_upgrade_migration = 3;
 * @return {boolean}
 */
proto.ml_metadata.MigrationOptions.prototype.getEnableUpgradeMigration = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.MigrationOptions} returns this
 */
proto.ml_metadata.MigrationOptions.prototype.setEnableUpgradeMigration = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MigrationOptions} returns this
 */
proto.ml_metadata.MigrationOptions.prototype.clearEnableUpgradeMigration = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MigrationOptions.prototype.hasEnableUpgradeMigration = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional int64 downgrade_to_schema_version = 2;
 * @return {number}
 */
proto.ml_metadata.MigrationOptions.prototype.getDowngradeToSchemaVersion = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, -1));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.MigrationOptions} returns this
 */
proto.ml_metadata.MigrationOptions.prototype.setDowngradeToSchemaVersion = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MigrationOptions} returns this
 */
proto.ml_metadata.MigrationOptions.prototype.clearDowngradeToSchemaVersion = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MigrationOptions.prototype.hasDowngradeToSchemaVersion = function() {
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
proto.ml_metadata.RetryOptions.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.RetryOptions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.RetryOptions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.RetryOptions.toObject = function(includeInstance, msg) {
  var f, obj = {
    maxNumRetries: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.RetryOptions}
 */
proto.ml_metadata.RetryOptions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.RetryOptions;
  return proto.ml_metadata.RetryOptions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.RetryOptions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.RetryOptions}
 */
proto.ml_metadata.RetryOptions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setMaxNumRetries(value);
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
proto.ml_metadata.RetryOptions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.RetryOptions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.RetryOptions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.RetryOptions.serializeBinaryToWriter = function(message, writer) {
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
 * optional int64 max_num_retries = 1;
 * @return {number}
 */
proto.ml_metadata.RetryOptions.prototype.getMaxNumRetries = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.RetryOptions} returns this
 */
proto.ml_metadata.RetryOptions.prototype.setMaxNumRetries = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.RetryOptions} returns this
 */
proto.ml_metadata.RetryOptions.prototype.clearMaxNumRetries = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.RetryOptions.prototype.hasMaxNumRetries = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_metadata.ConnectionConfig.oneofGroups_ = [[1,2,3]];

/**
 * @enum {number}
 */
proto.ml_metadata.ConnectionConfig.ConfigCase = {
  CONFIG_NOT_SET: 0,
  FAKE_DATABASE: 1,
  MYSQL: 2,
  SQLITE: 3
};

/**
 * @return {proto.ml_metadata.ConnectionConfig.ConfigCase}
 */
proto.ml_metadata.ConnectionConfig.prototype.getConfigCase = function() {
  return /** @type {proto.ml_metadata.ConnectionConfig.ConfigCase} */(jspb.Message.computeOneofCase(this, proto.ml_metadata.ConnectionConfig.oneofGroups_[0]));
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
proto.ml_metadata.ConnectionConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ConnectionConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ConnectionConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ConnectionConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    fakeDatabase: (f = msg.getFakeDatabase()) && proto.ml_metadata.FakeDatabaseConfig.toObject(includeInstance, f),
    mysql: (f = msg.getMysql()) && proto.ml_metadata.MySQLDatabaseConfig.toObject(includeInstance, f),
    sqlite: (f = msg.getSqlite()) && proto.ml_metadata.SqliteMetadataSourceConfig.toObject(includeInstance, f),
    retryOptions: (f = msg.getRetryOptions()) && proto.ml_metadata.RetryOptions.toObject(includeInstance, f)
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
 * @return {!proto.ml_metadata.ConnectionConfig}
 */
proto.ml_metadata.ConnectionConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ConnectionConfig;
  return proto.ml_metadata.ConnectionConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ConnectionConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ConnectionConfig}
 */
proto.ml_metadata.ConnectionConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.FakeDatabaseConfig;
      reader.readMessage(value,proto.ml_metadata.FakeDatabaseConfig.deserializeBinaryFromReader);
      msg.setFakeDatabase(value);
      break;
    case 2:
      var value = new proto.ml_metadata.MySQLDatabaseConfig;
      reader.readMessage(value,proto.ml_metadata.MySQLDatabaseConfig.deserializeBinaryFromReader);
      msg.setMysql(value);
      break;
    case 3:
      var value = new proto.ml_metadata.SqliteMetadataSourceConfig;
      reader.readMessage(value,proto.ml_metadata.SqliteMetadataSourceConfig.deserializeBinaryFromReader);
      msg.setSqlite(value);
      break;
    case 4:
      var value = new proto.ml_metadata.RetryOptions;
      reader.readMessage(value,proto.ml_metadata.RetryOptions.deserializeBinaryFromReader);
      msg.setRetryOptions(value);
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
proto.ml_metadata.ConnectionConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ConnectionConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ConnectionConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ConnectionConfig.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getFakeDatabase();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_metadata.FakeDatabaseConfig.serializeBinaryToWriter
    );
  }
  f = message.getMysql();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_metadata.MySQLDatabaseConfig.serializeBinaryToWriter
    );
  }
  f = message.getSqlite();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_metadata.SqliteMetadataSourceConfig.serializeBinaryToWriter
    );
  }
  f = message.getRetryOptions();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.ml_metadata.RetryOptions.serializeBinaryToWriter
    );
  }
};


/**
 * optional FakeDatabaseConfig fake_database = 1;
 * @return {?proto.ml_metadata.FakeDatabaseConfig}
 */
proto.ml_metadata.ConnectionConfig.prototype.getFakeDatabase = function() {
  return /** @type{?proto.ml_metadata.FakeDatabaseConfig} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.FakeDatabaseConfig, 1));
};


/**
 * @param {?proto.ml_metadata.FakeDatabaseConfig|undefined} value
 * @return {!proto.ml_metadata.ConnectionConfig} returns this
*/
proto.ml_metadata.ConnectionConfig.prototype.setFakeDatabase = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.ml_metadata.ConnectionConfig.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ConnectionConfig} returns this
 */
proto.ml_metadata.ConnectionConfig.prototype.clearFakeDatabase = function() {
  return this.setFakeDatabase(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ConnectionConfig.prototype.hasFakeDatabase = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional MySQLDatabaseConfig mysql = 2;
 * @return {?proto.ml_metadata.MySQLDatabaseConfig}
 */
proto.ml_metadata.ConnectionConfig.prototype.getMysql = function() {
  return /** @type{?proto.ml_metadata.MySQLDatabaseConfig} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.MySQLDatabaseConfig, 2));
};


/**
 * @param {?proto.ml_metadata.MySQLDatabaseConfig|undefined} value
 * @return {!proto.ml_metadata.ConnectionConfig} returns this
*/
proto.ml_metadata.ConnectionConfig.prototype.setMysql = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.ml_metadata.ConnectionConfig.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ConnectionConfig} returns this
 */
proto.ml_metadata.ConnectionConfig.prototype.clearMysql = function() {
  return this.setMysql(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ConnectionConfig.prototype.hasMysql = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional SqliteMetadataSourceConfig sqlite = 3;
 * @return {?proto.ml_metadata.SqliteMetadataSourceConfig}
 */
proto.ml_metadata.ConnectionConfig.prototype.getSqlite = function() {
  return /** @type{?proto.ml_metadata.SqliteMetadataSourceConfig} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.SqliteMetadataSourceConfig, 3));
};


/**
 * @param {?proto.ml_metadata.SqliteMetadataSourceConfig|undefined} value
 * @return {!proto.ml_metadata.ConnectionConfig} returns this
*/
proto.ml_metadata.ConnectionConfig.prototype.setSqlite = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.ml_metadata.ConnectionConfig.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ConnectionConfig} returns this
 */
proto.ml_metadata.ConnectionConfig.prototype.clearSqlite = function() {
  return this.setSqlite(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ConnectionConfig.prototype.hasSqlite = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional RetryOptions retry_options = 4;
 * @return {?proto.ml_metadata.RetryOptions}
 */
proto.ml_metadata.ConnectionConfig.prototype.getRetryOptions = function() {
  return /** @type{?proto.ml_metadata.RetryOptions} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.RetryOptions, 4));
};


/**
 * @param {?proto.ml_metadata.RetryOptions|undefined} value
 * @return {!proto.ml_metadata.ConnectionConfig} returns this
*/
proto.ml_metadata.ConnectionConfig.prototype.setRetryOptions = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ConnectionConfig} returns this
 */
proto.ml_metadata.ConnectionConfig.prototype.clearRetryOptions = function() {
  return this.setRetryOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ConnectionConfig.prototype.hasRetryOptions = function() {
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
proto.ml_metadata.GrpcChannelArguments.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.GrpcChannelArguments.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.GrpcChannelArguments} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GrpcChannelArguments.toObject = function(includeInstance, msg) {
  var f, obj = {
    maxReceiveMessageLength: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    http2MaxPingStrikes: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.GrpcChannelArguments}
 */
proto.ml_metadata.GrpcChannelArguments.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.GrpcChannelArguments;
  return proto.ml_metadata.GrpcChannelArguments.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.GrpcChannelArguments} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.GrpcChannelArguments}
 */
proto.ml_metadata.GrpcChannelArguments.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setMaxReceiveMessageLength(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setHttp2MaxPingStrikes(value);
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
proto.ml_metadata.GrpcChannelArguments.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.GrpcChannelArguments.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.GrpcChannelArguments} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.GrpcChannelArguments.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
};


/**
 * optional int64 max_receive_message_length = 1;
 * @return {number}
 */
proto.ml_metadata.GrpcChannelArguments.prototype.getMaxReceiveMessageLength = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.GrpcChannelArguments} returns this
 */
proto.ml_metadata.GrpcChannelArguments.prototype.setMaxReceiveMessageLength = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GrpcChannelArguments} returns this
 */
proto.ml_metadata.GrpcChannelArguments.prototype.clearMaxReceiveMessageLength = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GrpcChannelArguments.prototype.hasMaxReceiveMessageLength = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional int64 http2_max_ping_strikes = 2;
 * @return {number}
 */
proto.ml_metadata.GrpcChannelArguments.prototype.getHttp2MaxPingStrikes = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.GrpcChannelArguments} returns this
 */
proto.ml_metadata.GrpcChannelArguments.prototype.setHttp2MaxPingStrikes = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.GrpcChannelArguments} returns this
 */
proto.ml_metadata.GrpcChannelArguments.prototype.clearHttp2MaxPingStrikes = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.GrpcChannelArguments.prototype.hasHttp2MaxPingStrikes = function() {
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
proto.ml_metadata.MetadataStoreClientConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.MetadataStoreClientConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.MetadataStoreClientConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MetadataStoreClientConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    host: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    port: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    sslConfig: (f = msg.getSslConfig()) && proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.toObject(includeInstance, f),
    channelArguments: (f = msg.getChannelArguments()) && proto.ml_metadata.GrpcChannelArguments.toObject(includeInstance, f),
    clientTimeoutSec: (f = jspb.Message.getOptionalFloatingPointField(msg, 5)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.MetadataStoreClientConfig}
 */
proto.ml_metadata.MetadataStoreClientConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.MetadataStoreClientConfig;
  return proto.ml_metadata.MetadataStoreClientConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.MetadataStoreClientConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig}
 */
proto.ml_metadata.MetadataStoreClientConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setHost(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setPort(value);
      break;
    case 3:
      var value = new proto.ml_metadata.MetadataStoreClientConfig.SSLConfig;
      reader.readMessage(value,proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.deserializeBinaryFromReader);
      msg.setSslConfig(value);
      break;
    case 4:
      var value = new proto.ml_metadata.GrpcChannelArguments;
      reader.readMessage(value,proto.ml_metadata.GrpcChannelArguments.deserializeBinaryFromReader);
      msg.setChannelArguments(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readDouble());
      msg.setClientTimeoutSec(value);
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
proto.ml_metadata.MetadataStoreClientConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.MetadataStoreClientConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.MetadataStoreClientConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MetadataStoreClientConfig.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {string} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeString(
      1,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeUint32(
      2,
      f
    );
  }
  f = message.getSslConfig();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.serializeBinaryToWriter
    );
  }
  f = message.getChannelArguments();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.ml_metadata.GrpcChannelArguments.serializeBinaryToWriter
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 5));
  if (f != null) {
    writer.writeDouble(
      5,
      f
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
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    clientKey: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    serverCert: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    customCa: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig}
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.MetadataStoreClientConfig.SSLConfig;
  return proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig}
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setClientKey(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setServerCert(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setCustomCa(value);
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
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.serializeBinaryToWriter = function(message, writer) {
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
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
};


/**
 * optional string client_key = 1;
 * @return {string}
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.getClientKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.setClientKey = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.clearClientKey = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.hasClientKey = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string server_cert = 2;
 * @return {string}
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.getServerCert = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.setServerCert = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.clearServerCert = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.hasServerCert = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string custom_ca = 3;
 * @return {string}
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.getCustomCa = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.setCustomCa = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.clearCustomCa = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreClientConfig.SSLConfig.prototype.hasCustomCa = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string host = 1;
 * @return {string}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.getHost = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.setHost = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.clearHost = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.hasHost = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional uint32 port = 2;
 * @return {number}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.getPort = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.setPort = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.clearPort = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.hasPort = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional SSLConfig ssl_config = 3;
 * @return {?proto.ml_metadata.MetadataStoreClientConfig.SSLConfig}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.getSslConfig = function() {
  return /** @type{?proto.ml_metadata.MetadataStoreClientConfig.SSLConfig} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.MetadataStoreClientConfig.SSLConfig, 3));
};


/**
 * @param {?proto.ml_metadata.MetadataStoreClientConfig.SSLConfig|undefined} value
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
*/
proto.ml_metadata.MetadataStoreClientConfig.prototype.setSslConfig = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.clearSslConfig = function() {
  return this.setSslConfig(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.hasSslConfig = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional GrpcChannelArguments channel_arguments = 4;
 * @return {?proto.ml_metadata.GrpcChannelArguments}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.getChannelArguments = function() {
  return /** @type{?proto.ml_metadata.GrpcChannelArguments} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.GrpcChannelArguments, 4));
};


/**
 * @param {?proto.ml_metadata.GrpcChannelArguments|undefined} value
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
*/
proto.ml_metadata.MetadataStoreClientConfig.prototype.setChannelArguments = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.clearChannelArguments = function() {
  return this.setChannelArguments(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.hasChannelArguments = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional double client_timeout_sec = 5;
 * @return {number}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.getClientTimeoutSec = function() {
  return /** @type {number} */ (jspb.Message.getFloatingPointFieldWithDefault(this, 5, 0.0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.setClientTimeoutSec = function(value) {
  return jspb.Message.setField(this, 5, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreClientConfig} returns this
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.clearClientTimeoutSec = function() {
  return jspb.Message.setField(this, 5, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreClientConfig.prototype.hasClientTimeoutSec = function() {
  return jspb.Message.getField(this, 5) != null;
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
proto.ml_metadata.MetadataStoreServerConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.MetadataStoreServerConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.MetadataStoreServerConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MetadataStoreServerConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    connectionConfig: (f = msg.getConnectionConfig()) && proto.ml_metadata.ConnectionConfig.toObject(includeInstance, f),
    migrationOptions: (f = msg.getMigrationOptions()) && proto.ml_metadata.MigrationOptions.toObject(includeInstance, f),
    sslConfig: (f = msg.getSslConfig()) && proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.toObject(includeInstance, f)
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
 * @return {!proto.ml_metadata.MetadataStoreServerConfig}
 */
proto.ml_metadata.MetadataStoreServerConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.MetadataStoreServerConfig;
  return proto.ml_metadata.MetadataStoreServerConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.MetadataStoreServerConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.MetadataStoreServerConfig}
 */
proto.ml_metadata.MetadataStoreServerConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ConnectionConfig;
      reader.readMessage(value,proto.ml_metadata.ConnectionConfig.deserializeBinaryFromReader);
      msg.setConnectionConfig(value);
      break;
    case 3:
      var value = new proto.ml_metadata.MigrationOptions;
      reader.readMessage(value,proto.ml_metadata.MigrationOptions.deserializeBinaryFromReader);
      msg.setMigrationOptions(value);
      break;
    case 2:
      var value = new proto.ml_metadata.MetadataStoreServerConfig.SSLConfig;
      reader.readMessage(value,proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.deserializeBinaryFromReader);
      msg.setSslConfig(value);
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
proto.ml_metadata.MetadataStoreServerConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.MetadataStoreServerConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.MetadataStoreServerConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MetadataStoreServerConfig.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getConnectionConfig();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_metadata.ConnectionConfig.serializeBinaryToWriter
    );
  }
  f = message.getMigrationOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_metadata.MigrationOptions.serializeBinaryToWriter
    );
  }
  f = message.getSslConfig();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.serializeBinaryToWriter
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
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    serverKey: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    serverCert: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    customCa: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    clientVerify: (f = jspb.Message.getBooleanField(msg, 4)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.MetadataStoreServerConfig.SSLConfig;
  return proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setServerKey(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setServerCert(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setCustomCa(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setClientVerify(value);
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
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.serializeBinaryToWriter = function(message, writer) {
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
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
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
 * optional string server_key = 1;
 * @return {string}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.getServerKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.setServerKey = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.clearServerKey = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.hasServerKey = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string server_cert = 2;
 * @return {string}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.getServerCert = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.setServerCert = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.clearServerCert = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.hasServerCert = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string custom_ca = 3;
 * @return {string}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.getCustomCa = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.setCustomCa = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.clearCustomCa = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.hasCustomCa = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional bool client_verify = 4;
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.getClientVerify = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.setClientVerify = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.clearClientVerify = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreServerConfig.SSLConfig.prototype.hasClientVerify = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional ConnectionConfig connection_config = 1;
 * @return {?proto.ml_metadata.ConnectionConfig}
 */
proto.ml_metadata.MetadataStoreServerConfig.prototype.getConnectionConfig = function() {
  return /** @type{?proto.ml_metadata.ConnectionConfig} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ConnectionConfig, 1));
};


/**
 * @param {?proto.ml_metadata.ConnectionConfig|undefined} value
 * @return {!proto.ml_metadata.MetadataStoreServerConfig} returns this
*/
proto.ml_metadata.MetadataStoreServerConfig.prototype.setConnectionConfig = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreServerConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.prototype.clearConnectionConfig = function() {
  return this.setConnectionConfig(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreServerConfig.prototype.hasConnectionConfig = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional MigrationOptions migration_options = 3;
 * @return {?proto.ml_metadata.MigrationOptions}
 */
proto.ml_metadata.MetadataStoreServerConfig.prototype.getMigrationOptions = function() {
  return /** @type{?proto.ml_metadata.MigrationOptions} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.MigrationOptions, 3));
};


/**
 * @param {?proto.ml_metadata.MigrationOptions|undefined} value
 * @return {!proto.ml_metadata.MetadataStoreServerConfig} returns this
*/
proto.ml_metadata.MetadataStoreServerConfig.prototype.setMigrationOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreServerConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.prototype.clearMigrationOptions = function() {
  return this.setMigrationOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreServerConfig.prototype.hasMigrationOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional SSLConfig ssl_config = 2;
 * @return {?proto.ml_metadata.MetadataStoreServerConfig.SSLConfig}
 */
proto.ml_metadata.MetadataStoreServerConfig.prototype.getSslConfig = function() {
  return /** @type{?proto.ml_metadata.MetadataStoreServerConfig.SSLConfig} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.MetadataStoreServerConfig.SSLConfig, 2));
};


/**
 * @param {?proto.ml_metadata.MetadataStoreServerConfig.SSLConfig|undefined} value
 * @return {!proto.ml_metadata.MetadataStoreServerConfig} returns this
*/
proto.ml_metadata.MetadataStoreServerConfig.prototype.setSslConfig = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.MetadataStoreServerConfig} returns this
 */
proto.ml_metadata.MetadataStoreServerConfig.prototype.clearSslConfig = function() {
  return this.setSslConfig(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.MetadataStoreServerConfig.prototype.hasSslConfig = function() {
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
proto.ml_metadata.ListOperationOptions.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ListOperationOptions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ListOperationOptions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ListOperationOptions.toObject = function(includeInstance, msg) {
  var f, obj = {
    maxResultSize: jspb.Message.getFieldWithDefault(msg, 1, 20),
    orderByField: (f = msg.getOrderByField()) && proto.ml_metadata.ListOperationOptions.OrderByField.toObject(includeInstance, f),
    nextPageToken: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f,
    filterQuery: (f = jspb.Message.getField(msg, 4)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.ListOperationOptions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ListOperationOptions;
  return proto.ml_metadata.ListOperationOptions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ListOperationOptions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.ListOperationOptions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxResultSize(value);
      break;
    case 2:
      var value = new proto.ml_metadata.ListOperationOptions.OrderByField;
      reader.readMessage(value,proto.ml_metadata.ListOperationOptions.OrderByField.deserializeBinaryFromReader);
      msg.setOrderByField(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setNextPageToken(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setFilterQuery(value);
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
proto.ml_metadata.ListOperationOptions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ListOperationOptions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ListOperationOptions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ListOperationOptions.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt32(
      1,
      f
    );
  }
  f = message.getOrderByField();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_metadata.ListOperationOptions.OrderByField.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 4));
  if (f != null) {
    writer.writeString(
      4,
      f
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
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ListOperationOptions.OrderByField.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ListOperationOptions.OrderByField} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ListOperationOptions.OrderByField.toObject = function(includeInstance, msg) {
  var f, obj = {
    field: jspb.Message.getFieldWithDefault(msg, 1, 3),
    isAsc: jspb.Message.getBooleanFieldWithDefault(msg, 2, true)
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
 * @return {!proto.ml_metadata.ListOperationOptions.OrderByField}
 */
proto.ml_metadata.ListOperationOptions.OrderByField.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ListOperationOptions.OrderByField;
  return proto.ml_metadata.ListOperationOptions.OrderByField.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ListOperationOptions.OrderByField} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ListOperationOptions.OrderByField}
 */
proto.ml_metadata.ListOperationOptions.OrderByField.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.ml_metadata.ListOperationOptions.OrderByField.Field} */ (reader.readEnum());
      msg.setField(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsAsc(value);
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
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ListOperationOptions.OrderByField.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ListOperationOptions.OrderByField} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ListOperationOptions.OrderByField.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {!proto.ml_metadata.ListOperationOptions.OrderByField.Field} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeEnum(
      1,
      f
    );
  }
  f = /** @type {boolean} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeBool(
      2,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.ml_metadata.ListOperationOptions.OrderByField.Field = {
  FIELD_UNSPECIFIED: 0,
  CREATE_TIME: 1,
  LAST_UPDATE_TIME: 2,
  ID: 3
};

/**
 * optional Field field = 1;
 * @return {!proto.ml_metadata.ListOperationOptions.OrderByField.Field}
 */
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.getField = function() {
  return /** @type {!proto.ml_metadata.ListOperationOptions.OrderByField.Field} */ (jspb.Message.getFieldWithDefault(this, 1, 3));
};


/**
 * @param {!proto.ml_metadata.ListOperationOptions.OrderByField.Field} value
 * @return {!proto.ml_metadata.ListOperationOptions.OrderByField} returns this
 */
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.setField = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ListOperationOptions.OrderByField} returns this
 */
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.clearField = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.hasField = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional bool is_asc = 2;
 * @return {boolean}
 */
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.getIsAsc = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, true));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_metadata.ListOperationOptions.OrderByField} returns this
 */
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.setIsAsc = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ListOperationOptions.OrderByField} returns this
 */
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.clearIsAsc = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListOperationOptions.OrderByField.prototype.hasIsAsc = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional int32 max_result_size = 1;
 * @return {number}
 */
proto.ml_metadata.ListOperationOptions.prototype.getMaxResultSize = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 20));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.ListOperationOptions} returns this
 */
proto.ml_metadata.ListOperationOptions.prototype.setMaxResultSize = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ListOperationOptions} returns this
 */
proto.ml_metadata.ListOperationOptions.prototype.clearMaxResultSize = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListOperationOptions.prototype.hasMaxResultSize = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional OrderByField order_by_field = 2;
 * @return {?proto.ml_metadata.ListOperationOptions.OrderByField}
 */
proto.ml_metadata.ListOperationOptions.prototype.getOrderByField = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions.OrderByField} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ListOperationOptions.OrderByField, 2));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions.OrderByField|undefined} value
 * @return {!proto.ml_metadata.ListOperationOptions} returns this
*/
proto.ml_metadata.ListOperationOptions.prototype.setOrderByField = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ListOperationOptions} returns this
 */
proto.ml_metadata.ListOperationOptions.prototype.clearOrderByField = function() {
  return this.setOrderByField(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListOperationOptions.prototype.hasOrderByField = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string next_page_token = 3;
 * @return {string}
 */
proto.ml_metadata.ListOperationOptions.prototype.getNextPageToken = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ListOperationOptions} returns this
 */
proto.ml_metadata.ListOperationOptions.prototype.setNextPageToken = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ListOperationOptions} returns this
 */
proto.ml_metadata.ListOperationOptions.prototype.clearNextPageToken = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListOperationOptions.prototype.hasNextPageToken = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string filter_query = 4;
 * @return {string}
 */
proto.ml_metadata.ListOperationOptions.prototype.getFilterQuery = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.ListOperationOptions} returns this
 */
proto.ml_metadata.ListOperationOptions.prototype.setFilterQuery = function(value) {
  return jspb.Message.setField(this, 4, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ListOperationOptions} returns this
 */
proto.ml_metadata.ListOperationOptions.prototype.clearFilterQuery = function() {
  return jspb.Message.setField(this, 4, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListOperationOptions.prototype.hasFilterQuery = function() {
  return jspb.Message.getField(this, 4) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_metadata.ListOperationNextPageToken.repeatedFields_ = [4];



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
proto.ml_metadata.ListOperationNextPageToken.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.ListOperationNextPageToken.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.ListOperationNextPageToken} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ListOperationNextPageToken.toObject = function(includeInstance, msg) {
  var f, obj = {
    idOffset: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    fieldOffset: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    setOptions: (f = msg.getSetOptions()) && proto.ml_metadata.ListOperationOptions.toObject(includeInstance, f),
    listedIdsList: (f = jspb.Message.getRepeatedField(msg, 4)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.ListOperationNextPageToken}
 */
proto.ml_metadata.ListOperationNextPageToken.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.ListOperationNextPageToken;
  return proto.ml_metadata.ListOperationNextPageToken.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.ListOperationNextPageToken} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.ListOperationNextPageToken}
 */
proto.ml_metadata.ListOperationNextPageToken.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setIdOffset(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setFieldOffset(value);
      break;
    case 3:
      var value = new proto.ml_metadata.ListOperationOptions;
      reader.readMessage(value,proto.ml_metadata.ListOperationOptions.deserializeBinaryFromReader);
      msg.setSetOptions(value);
      break;
    case 4:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt64() : [reader.readInt64()]);
      for (var i = 0; i < values.length; i++) {
        msg.addListedIds(values[i]);
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
proto.ml_metadata.ListOperationNextPageToken.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.ListOperationNextPageToken.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.ListOperationNextPageToken} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.ListOperationNextPageToken.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = /** @type {number} */ (jspb.Message.getField(message, 2));
  if (f != null) {
    writer.writeInt64(
      2,
      f
    );
  }
  f = message.getSetOptions();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_metadata.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = message.getListedIdsList();
  if (f.length > 0) {
    writer.writeRepeatedInt64(
      4,
      f
    );
  }
};


/**
 * optional int64 id_offset = 1;
 * @return {number}
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.getIdOffset = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.ListOperationNextPageToken} returns this
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.setIdOffset = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ListOperationNextPageToken} returns this
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.clearIdOffset = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.hasIdOffset = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional int64 field_offset = 2;
 * @return {number}
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.getFieldOffset = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.ListOperationNextPageToken} returns this
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.setFieldOffset = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.ListOperationNextPageToken} returns this
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.clearFieldOffset = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.hasFieldOffset = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional ListOperationOptions set_options = 3;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.getSetOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ListOperationOptions, 3));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.ListOperationNextPageToken} returns this
*/
proto.ml_metadata.ListOperationNextPageToken.prototype.setSetOptions = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.ListOperationNextPageToken} returns this
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.clearSetOptions = function() {
  return this.setSetOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.hasSetOptions = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * repeated int64 listed_ids = 4;
 * @return {!Array<number>}
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.getListedIdsList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 4));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.ml_metadata.ListOperationNextPageToken} returns this
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.setListedIdsList = function(value) {
  return jspb.Message.setField(this, 4, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.ml_metadata.ListOperationNextPageToken} returns this
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.addListedIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 4, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_metadata.ListOperationNextPageToken} returns this
 */
proto.ml_metadata.ListOperationNextPageToken.prototype.clearListedIdsList = function() {
  return this.setListedIdsList([]);
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
proto.ml_metadata.TransactionOptions.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.TransactionOptions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.TransactionOptions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.TransactionOptions.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  jspb.Message.toObjectExtension(/** @type {!jspb.Message} */ (msg), obj,
      proto.ml_metadata.TransactionOptions.extensions, proto.ml_metadata.TransactionOptions.prototype.getExtension,
      includeInstance);
  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.TransactionOptions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.TransactionOptions;
  return proto.ml_metadata.TransactionOptions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.TransactionOptions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.TransactionOptions}
 */
proto.ml_metadata.TransactionOptions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      jspb.Message.readBinaryExtension(msg, reader,
        proto.ml_metadata.TransactionOptions.extensionsBinary,
        proto.ml_metadata.TransactionOptions.prototype.getExtension,
        proto.ml_metadata.TransactionOptions.prototype.setExtension);
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.ml_metadata.TransactionOptions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.TransactionOptions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.TransactionOptions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.TransactionOptions.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  jspb.Message.serializeBinaryExtensions(message, writer,
    proto.ml_metadata.TransactionOptions.extensionsBinary, proto.ml_metadata.TransactionOptions.prototype.getExtension);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_metadata.LineageGraphQueryOptions.oneofGroups_ = [[1]];

/**
 * @enum {number}
 */
proto.ml_metadata.LineageGraphQueryOptions.QueryNodesCase = {
  QUERY_NODES_NOT_SET: 0,
  ARTIFACTS_OPTIONS: 1
};

/**
 * @return {proto.ml_metadata.LineageGraphQueryOptions.QueryNodesCase}
 */
proto.ml_metadata.LineageGraphQueryOptions.prototype.getQueryNodesCase = function() {
  return /** @type {proto.ml_metadata.LineageGraphQueryOptions.QueryNodesCase} */(jspb.Message.computeOneofCase(this, proto.ml_metadata.LineageGraphQueryOptions.oneofGroups_[0]));
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
proto.ml_metadata.LineageGraphQueryOptions.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.LineageGraphQueryOptions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.LineageGraphQueryOptions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.LineageGraphQueryOptions.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsOptions: (f = msg.getArtifactsOptions()) && proto.ml_metadata.ListOperationOptions.toObject(includeInstance, f),
    stopConditions: (f = msg.getStopConditions()) && proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.toObject(includeInstance, f)
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
 * @return {!proto.ml_metadata.LineageGraphQueryOptions}
 */
proto.ml_metadata.LineageGraphQueryOptions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.LineageGraphQueryOptions;
  return proto.ml_metadata.LineageGraphQueryOptions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.LineageGraphQueryOptions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.LineageGraphQueryOptions}
 */
proto.ml_metadata.LineageGraphQueryOptions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_metadata.ListOperationOptions;
      reader.readMessage(value,proto.ml_metadata.ListOperationOptions.deserializeBinaryFromReader);
      msg.setArtifactsOptions(value);
      break;
    case 2:
      var value = new proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint;
      reader.readMessage(value,proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.deserializeBinaryFromReader);
      msg.setStopConditions(value);
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
proto.ml_metadata.LineageGraphQueryOptions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.LineageGraphQueryOptions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.LineageGraphQueryOptions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.LineageGraphQueryOptions.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsOptions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_metadata.ListOperationOptions.serializeBinaryToWriter
    );
  }
  f = message.getStopConditions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.serializeBinaryToWriter
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
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.toObject = function(includeInstance, msg) {
  var f, obj = {
    maxNumHops: (f = jspb.Message.getField(msg, 1)) == null ? undefined : f,
    boundaryArtifacts: (f = jspb.Message.getField(msg, 2)) == null ? undefined : f,
    boundaryExecutions: (f = jspb.Message.getField(msg, 3)) == null ? undefined : f
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
 * @return {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint}
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint;
  return proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint}
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setMaxNumHops(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setBoundaryArtifacts(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setBoundaryExecutions(value);
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
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = /** @type {number} */ (jspb.Message.getField(message, 1));
  if (f != null) {
    writer.writeInt64(
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
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
};


/**
 * optional int64 max_num_hops = 1;
 * @return {number}
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.getMaxNumHops = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} returns this
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.setMaxNumHops = function(value) {
  return jspb.Message.setField(this, 1, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} returns this
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.clearMaxNumHops = function() {
  return jspb.Message.setField(this, 1, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.hasMaxNumHops = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string boundary_artifacts = 2;
 * @return {string}
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.getBoundaryArtifacts = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} returns this
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.setBoundaryArtifacts = function(value) {
  return jspb.Message.setField(this, 2, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} returns this
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.clearBoundaryArtifacts = function() {
  return jspb.Message.setField(this, 2, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.hasBoundaryArtifacts = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string boundary_executions = 3;
 * @return {string}
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.getBoundaryExecutions = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} returns this
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.setBoundaryExecutions = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} returns this
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.clearBoundaryExecutions = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint.prototype.hasBoundaryExecutions = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional ListOperationOptions artifacts_options = 1;
 * @return {?proto.ml_metadata.ListOperationOptions}
 */
proto.ml_metadata.LineageGraphQueryOptions.prototype.getArtifactsOptions = function() {
  return /** @type{?proto.ml_metadata.ListOperationOptions} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.ListOperationOptions, 1));
};


/**
 * @param {?proto.ml_metadata.ListOperationOptions|undefined} value
 * @return {!proto.ml_metadata.LineageGraphQueryOptions} returns this
*/
proto.ml_metadata.LineageGraphQueryOptions.prototype.setArtifactsOptions = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.ml_metadata.LineageGraphQueryOptions.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.LineageGraphQueryOptions} returns this
 */
proto.ml_metadata.LineageGraphQueryOptions.prototype.clearArtifactsOptions = function() {
  return this.setArtifactsOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.LineageGraphQueryOptions.prototype.hasArtifactsOptions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional BoundaryConstraint stop_conditions = 2;
 * @return {?proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint}
 */
proto.ml_metadata.LineageGraphQueryOptions.prototype.getStopConditions = function() {
  return /** @type{?proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint} */ (
    jspb.Message.getWrapperField(this, proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint, 2));
};


/**
 * @param {?proto.ml_metadata.LineageGraphQueryOptions.BoundaryConstraint|undefined} value
 * @return {!proto.ml_metadata.LineageGraphQueryOptions} returns this
*/
proto.ml_metadata.LineageGraphQueryOptions.prototype.setStopConditions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_metadata.LineageGraphQueryOptions} returns this
 */
proto.ml_metadata.LineageGraphQueryOptions.prototype.clearStopConditions = function() {
  return this.setStopConditions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_metadata.LineageGraphQueryOptions.prototype.hasStopConditions = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * @enum {number}
 */
proto.ml_metadata.PropertyType = {
  UNKNOWN: 0,
  INT: 1,
  DOUBLE: 2,
  STRING: 3,
  STRUCT: 4
};


/**
 * A tuple of {field number, class constructor} for the extension
 * field named `systemTypeExtension`.
 * @type {!jspb.ExtensionFieldInfo<!proto.ml_metadata.SystemTypeExtension>}
 */
proto.ml_metadata.systemTypeExtension = new jspb.ExtensionFieldInfo(
    384560917,
    {systemTypeExtension: 0},
    proto.ml_metadata.SystemTypeExtension,
     /** @type {?function((boolean|undefined),!jspb.Message=): !Object} */ (
         proto.ml_metadata.SystemTypeExtension.toObject),
    0);

google_protobuf_descriptor_pb.EnumValueOptions.extensionsBinary[384560917] = new jspb.ExtensionFieldBinaryInfo(
    proto.ml_metadata.systemTypeExtension,
    jspb.BinaryReader.prototype.readMessage,
    jspb.BinaryWriter.prototype.writeMessage,
    proto.ml_metadata.SystemTypeExtension.serializeBinaryToWriter,
    proto.ml_metadata.SystemTypeExtension.deserializeBinaryFromReader,
    false);
// This registers the extension field with the extended class, so that
// toObject() will function correctly.
google_protobuf_descriptor_pb.EnumValueOptions.extensions[384560917] = proto.ml_metadata.systemTypeExtension;

goog.object.extend(exports, proto.ml_metadata);
