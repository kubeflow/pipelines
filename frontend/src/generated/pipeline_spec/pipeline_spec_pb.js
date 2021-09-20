// source: pipeline_spec.proto
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

var google_protobuf_any_pb = require('google-protobuf/google/protobuf/any_pb.js');
goog.object.extend(proto, google_protobuf_any_pb);
var google_protobuf_struct_pb = require('google-protobuf/google/protobuf/struct_pb.js');
goog.object.extend(proto, google_protobuf_struct_pb);
var google_rpc_status_pb = require('./google/rpc/status_pb.js');
goog.object.extend(proto, google_rpc_status_pb);
goog.exportSymbol('proto.ml_pipelines.ArtifactIteratorSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ArtifactList', null, global);
goog.exportSymbol('proto.ml_pipelines.ArtifactTypeSchema', null, global);
goog.exportSymbol('proto.ml_pipelines.ArtifactTypeSchema.KindCase', null, global);
goog.exportSymbol('proto.ml_pipelines.ComponentInputsSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ComponentInputsSpec.ParameterSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ComponentOutputsSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ComponentRef', null, global);
goog.exportSymbol('proto.ml_pipelines.ComponentSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ComponentSpec.ImplementationCase', null, global);
goog.exportSymbol('proto.ml_pipelines.DagOutputsSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.KindCase', null, global);
goog.exportSymbol('proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.DagSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ExecutorInput', null, global);
goog.exportSymbol('proto.ml_pipelines.ExecutorInput.Inputs', null, global);
goog.exportSymbol('proto.ml_pipelines.ExecutorInput.OutputParameter', null, global);
goog.exportSymbol('proto.ml_pipelines.ExecutorInput.Outputs', null, global);
goog.exportSymbol('proto.ml_pipelines.ExecutorOutput', null, global);
goog.exportSymbol('proto.ml_pipelines.ParameterIteratorSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.KindCase', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.SpecCase', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineInfo', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineJob', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineJob.RuntimeConfig', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineSpec.RuntimeParameter', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineStateEnum', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineStateEnum.PipelineTaskState', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineTaskFinalStatus', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineTaskInfo', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineTaskSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineTaskSpec.CachingOptions', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineTaskSpec.IteratorCase', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy', null, global);
goog.exportSymbol('proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy', null, global);
goog.exportSymbol('proto.ml_pipelines.PrimitiveType', null, global);
goog.exportSymbol('proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum', null, global);
goog.exportSymbol('proto.ml_pipelines.RuntimeArtifact', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskInputsSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.KindCase', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskInputsSpec.InputParameterSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.KindCase', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskOutputsSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec', null, global);
goog.exportSymbol('proto.ml_pipelines.Value', null, global);
goog.exportSymbol('proto.ml_pipelines.Value.ValueCase', null, global);
goog.exportSymbol('proto.ml_pipelines.ValueOrRuntimeParameter', null, global);
goog.exportSymbol('proto.ml_pipelines.ValueOrRuntimeParameter.ValueCase', null, global);
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
proto.ml_pipelines.PipelineJob = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineJob, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineJob.displayName = 'proto.ml_pipelines.PipelineJob';
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
proto.ml_pipelines.PipelineJob.RuntimeConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineJob.RuntimeConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineJob.RuntimeConfig.displayName = 'proto.ml_pipelines.PipelineJob.RuntimeConfig';
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
proto.ml_pipelines.PipelineSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineSpec.displayName = 'proto.ml_pipelines.PipelineSpec';
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
proto.ml_pipelines.PipelineSpec.RuntimeParameter = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineSpec.RuntimeParameter, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineSpec.RuntimeParameter.displayName = 'proto.ml_pipelines.PipelineSpec.RuntimeParameter';
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
proto.ml_pipelines.ComponentSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_pipelines.ComponentSpec.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.ComponentSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ComponentSpec.displayName = 'proto.ml_pipelines.ComponentSpec';
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
proto.ml_pipelines.DagSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.DagSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.DagSpec.displayName = 'proto.ml_pipelines.DagSpec';
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
proto.ml_pipelines.DagOutputsSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.DagOutputsSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.DagOutputsSpec.displayName = 'proto.ml_pipelines.DagOutputsSpec';
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
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.displayName = 'proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec';
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
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.repeatedFields_, null);
};
goog.inherits(proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.displayName = 'proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec';
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
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.displayName = 'proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec';
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
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.repeatedFields_, null);
};
goog.inherits(proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.displayName = 'proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec';
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
proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.displayName = 'proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec';
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
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.displayName = 'proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec';
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
proto.ml_pipelines.ComponentInputsSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ComponentInputsSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ComponentInputsSpec.displayName = 'proto.ml_pipelines.ComponentInputsSpec';
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
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.displayName = 'proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec';
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
proto.ml_pipelines.ComponentInputsSpec.ParameterSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ComponentInputsSpec.ParameterSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.displayName = 'proto.ml_pipelines.ComponentInputsSpec.ParameterSpec';
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
proto.ml_pipelines.ComponentOutputsSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ComponentOutputsSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ComponentOutputsSpec.displayName = 'proto.ml_pipelines.ComponentOutputsSpec';
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
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.displayName = 'proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec';
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
proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.displayName = 'proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec';
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
proto.ml_pipelines.TaskInputsSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.TaskInputsSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.TaskInputsSpec.displayName = 'proto.ml_pipelines.TaskInputsSpec';
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
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.displayName = 'proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec';
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
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.displayName = 'proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec';
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
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.TaskInputsSpec.InputParameterSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.displayName = 'proto.ml_pipelines.TaskInputsSpec.InputParameterSpec';
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
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.displayName = 'proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec';
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
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.displayName = 'proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus';
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
proto.ml_pipelines.TaskOutputsSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.TaskOutputsSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.TaskOutputsSpec.displayName = 'proto.ml_pipelines.TaskOutputsSpec';
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
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.displayName = 'proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec';
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
proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.displayName = 'proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec';
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
proto.ml_pipelines.PrimitiveType = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PrimitiveType, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PrimitiveType.displayName = 'proto.ml_pipelines.PrimitiveType';
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
proto.ml_pipelines.PipelineTaskSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_pipelines.PipelineTaskSpec.repeatedFields_, proto.ml_pipelines.PipelineTaskSpec.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.PipelineTaskSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineTaskSpec.displayName = 'proto.ml_pipelines.PipelineTaskSpec';
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
proto.ml_pipelines.PipelineTaskSpec.CachingOptions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineTaskSpec.CachingOptions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineTaskSpec.CachingOptions.displayName = 'proto.ml_pipelines.PipelineTaskSpec.CachingOptions';
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
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.displayName = 'proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy';
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
proto.ml_pipelines.ArtifactIteratorSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ArtifactIteratorSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ArtifactIteratorSpec.displayName = 'proto.ml_pipelines.ArtifactIteratorSpec';
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
proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.displayName = 'proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec';
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
proto.ml_pipelines.ParameterIteratorSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ParameterIteratorSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ParameterIteratorSpec.displayName = 'proto.ml_pipelines.ParameterIteratorSpec';
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
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.displayName = 'proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec';
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
proto.ml_pipelines.ComponentRef = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ComponentRef, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ComponentRef.displayName = 'proto.ml_pipelines.ComponentRef';
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
proto.ml_pipelines.PipelineInfo = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineInfo, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineInfo.displayName = 'proto.ml_pipelines.PipelineInfo';
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
proto.ml_pipelines.ArtifactTypeSchema = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_pipelines.ArtifactTypeSchema.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.ArtifactTypeSchema, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ArtifactTypeSchema.displayName = 'proto.ml_pipelines.ArtifactTypeSchema';
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
proto.ml_pipelines.PipelineTaskInfo = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineTaskInfo, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineTaskInfo.displayName = 'proto.ml_pipelines.PipelineTaskInfo';
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
proto.ml_pipelines.ValueOrRuntimeParameter = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_pipelines.ValueOrRuntimeParameter.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.ValueOrRuntimeParameter, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ValueOrRuntimeParameter.displayName = 'proto.ml_pipelines.ValueOrRuntimeParameter';
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
proto.ml_pipelines.PipelineDeploymentConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig';
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.repeatedFields_, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec';
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle';
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.repeatedFields_, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec';
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec';
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig';
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
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec';
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
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec';
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
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec';
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
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec';
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
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.displayName = 'proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec';
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
proto.ml_pipelines.Value = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.ml_pipelines.Value.oneofGroups_);
};
goog.inherits(proto.ml_pipelines.Value, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.Value.displayName = 'proto.ml_pipelines.Value';
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
proto.ml_pipelines.RuntimeArtifact = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.RuntimeArtifact, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.RuntimeArtifact.displayName = 'proto.ml_pipelines.RuntimeArtifact';
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
proto.ml_pipelines.ArtifactList = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.ml_pipelines.ArtifactList.repeatedFields_, null);
};
goog.inherits(proto.ml_pipelines.ArtifactList, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ArtifactList.displayName = 'proto.ml_pipelines.ArtifactList';
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
proto.ml_pipelines.ExecutorInput = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ExecutorInput, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ExecutorInput.displayName = 'proto.ml_pipelines.ExecutorInput';
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
proto.ml_pipelines.ExecutorInput.Inputs = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ExecutorInput.Inputs, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ExecutorInput.Inputs.displayName = 'proto.ml_pipelines.ExecutorInput.Inputs';
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
proto.ml_pipelines.ExecutorInput.OutputParameter = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ExecutorInput.OutputParameter, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ExecutorInput.OutputParameter.displayName = 'proto.ml_pipelines.ExecutorInput.OutputParameter';
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
proto.ml_pipelines.ExecutorInput.Outputs = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ExecutorInput.Outputs, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ExecutorInput.Outputs.displayName = 'proto.ml_pipelines.ExecutorInput.Outputs';
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
proto.ml_pipelines.ExecutorOutput = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.ExecutorOutput, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.ExecutorOutput.displayName = 'proto.ml_pipelines.ExecutorOutput';
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
proto.ml_pipelines.PipelineTaskFinalStatus = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineTaskFinalStatus, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineTaskFinalStatus.displayName = 'proto.ml_pipelines.PipelineTaskFinalStatus';
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
proto.ml_pipelines.PipelineStateEnum = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.ml_pipelines.PipelineStateEnum, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.ml_pipelines.PipelineStateEnum.displayName = 'proto.ml_pipelines.PipelineStateEnum';
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
proto.ml_pipelines.PipelineJob.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineJob.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineJob} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineJob.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    displayName: jspb.Message.getFieldWithDefault(msg, 2, ""),
    pipelineSpec: (f = msg.getPipelineSpec()) && google_protobuf_struct_pb.Struct.toObject(includeInstance, f),
    labelsMap: (f = msg.getLabelsMap()) ? f.toObject(includeInstance, undefined) : [],
    runtimeConfig: (f = msg.getRuntimeConfig()) && proto.ml_pipelines.PipelineJob.RuntimeConfig.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineJob}
 */
proto.ml_pipelines.PipelineJob.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineJob;
  return proto.ml_pipelines.PipelineJob.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineJob} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineJob}
 */
proto.ml_pipelines.PipelineJob.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDisplayName(value);
      break;
    case 7:
      var value = new google_protobuf_struct_pb.Struct;
      reader.readMessage(value,google_protobuf_struct_pb.Struct.deserializeBinaryFromReader);
      msg.setPipelineSpec(value);
      break;
    case 11:
      var value = msg.getLabelsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readString, null, "", "");
         });
      break;
    case 12:
      var value = new proto.ml_pipelines.PipelineJob.RuntimeConfig;
      reader.readMessage(value,proto.ml_pipelines.PipelineJob.RuntimeConfig.deserializeBinaryFromReader);
      msg.setRuntimeConfig(value);
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
proto.ml_pipelines.PipelineJob.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineJob.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineJob} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineJob.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDisplayName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getPipelineSpec();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      google_protobuf_struct_pb.Struct.serializeBinaryToWriter
    );
  }
  f = message.getLabelsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(11, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeString);
  }
  f = message.getRuntimeConfig();
  if (f != null) {
    writer.writeMessage(
      12,
      f,
      proto.ml_pipelines.PipelineJob.RuntimeConfig.serializeBinaryToWriter
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
proto.ml_pipelines.PipelineJob.RuntimeConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineJob.RuntimeConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineJob.RuntimeConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineJob.RuntimeConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    parametersMap: (f = msg.getParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.Value.toObject) : [],
    gcsOutputDirectory: jspb.Message.getFieldWithDefault(msg, 2, "")
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
 * @return {!proto.ml_pipelines.PipelineJob.RuntimeConfig}
 */
proto.ml_pipelines.PipelineJob.RuntimeConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineJob.RuntimeConfig;
  return proto.ml_pipelines.PipelineJob.RuntimeConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineJob.RuntimeConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineJob.RuntimeConfig}
 */
proto.ml_pipelines.PipelineJob.RuntimeConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.Value.deserializeBinaryFromReader, "", new proto.ml_pipelines.Value());
         });
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setGcsOutputDirectory(value);
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
proto.ml_pipelines.PipelineJob.RuntimeConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineJob.RuntimeConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineJob.RuntimeConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineJob.RuntimeConfig.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.Value.serializeBinaryToWriter);
  }
  f = message.getGcsOutputDirectory();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * map<string, Value> parameters = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.Value>}
 */
proto.ml_pipelines.PipelineJob.RuntimeConfig.prototype.getParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.Value>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.PipelineJob.RuntimeConfig} returns this
 */
proto.ml_pipelines.PipelineJob.RuntimeConfig.prototype.clearParametersMap = function() {
  this.getParametersMap().clear();
  return this;};


/**
 * optional string gcs_output_directory = 2;
 * @return {string}
 */
proto.ml_pipelines.PipelineJob.RuntimeConfig.prototype.getGcsOutputDirectory = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineJob.RuntimeConfig} returns this
 */
proto.ml_pipelines.PipelineJob.RuntimeConfig.prototype.setGcsOutputDirectory = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.ml_pipelines.PipelineJob.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineJob} returns this
 */
proto.ml_pipelines.PipelineJob.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string display_name = 2;
 * @return {string}
 */
proto.ml_pipelines.PipelineJob.prototype.getDisplayName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineJob} returns this
 */
proto.ml_pipelines.PipelineJob.prototype.setDisplayName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional google.protobuf.Struct pipeline_spec = 7;
 * @return {?proto.google.protobuf.Struct}
 */
proto.ml_pipelines.PipelineJob.prototype.getPipelineSpec = function() {
  return /** @type{?proto.google.protobuf.Struct} */ (
    jspb.Message.getWrapperField(this, google_protobuf_struct_pb.Struct, 7));
};


/**
 * @param {?proto.google.protobuf.Struct|undefined} value
 * @return {!proto.ml_pipelines.PipelineJob} returns this
*/
proto.ml_pipelines.PipelineJob.prototype.setPipelineSpec = function(value) {
  return jspb.Message.setWrapperField(this, 7, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineJob} returns this
 */
proto.ml_pipelines.PipelineJob.prototype.clearPipelineSpec = function() {
  return this.setPipelineSpec(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineJob.prototype.hasPipelineSpec = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * map<string, string> labels = 11;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,string>}
 */
proto.ml_pipelines.PipelineJob.prototype.getLabelsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,string>} */ (
      jspb.Message.getMapField(this, 11, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.PipelineJob} returns this
 */
proto.ml_pipelines.PipelineJob.prototype.clearLabelsMap = function() {
  this.getLabelsMap().clear();
  return this;};


/**
 * optional RuntimeConfig runtime_config = 12;
 * @return {?proto.ml_pipelines.PipelineJob.RuntimeConfig}
 */
proto.ml_pipelines.PipelineJob.prototype.getRuntimeConfig = function() {
  return /** @type{?proto.ml_pipelines.PipelineJob.RuntimeConfig} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineJob.RuntimeConfig, 12));
};


/**
 * @param {?proto.ml_pipelines.PipelineJob.RuntimeConfig|undefined} value
 * @return {!proto.ml_pipelines.PipelineJob} returns this
*/
proto.ml_pipelines.PipelineJob.prototype.setRuntimeConfig = function(value) {
  return jspb.Message.setWrapperField(this, 12, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineJob} returns this
 */
proto.ml_pipelines.PipelineJob.prototype.clearRuntimeConfig = function() {
  return this.setRuntimeConfig(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineJob.prototype.hasRuntimeConfig = function() {
  return jspb.Message.getField(this, 12) != null;
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
proto.ml_pipelines.PipelineSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    pipelineInfo: (f = msg.getPipelineInfo()) && proto.ml_pipelines.PipelineInfo.toObject(includeInstance, f),
    deploymentSpec: (f = msg.getDeploymentSpec()) && google_protobuf_struct_pb.Struct.toObject(includeInstance, f),
    sdkVersion: jspb.Message.getFieldWithDefault(msg, 4, ""),
    schemaVersion: jspb.Message.getFieldWithDefault(msg, 5, ""),
    componentsMap: (f = msg.getComponentsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ComponentSpec.toObject) : [],
    root: (f = msg.getRoot()) && proto.ml_pipelines.ComponentSpec.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineSpec}
 */
proto.ml_pipelines.PipelineSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineSpec;
  return proto.ml_pipelines.PipelineSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineSpec}
 */
proto.ml_pipelines.PipelineSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.PipelineInfo;
      reader.readMessage(value,proto.ml_pipelines.PipelineInfo.deserializeBinaryFromReader);
      msg.setPipelineInfo(value);
      break;
    case 7:
      var value = new google_protobuf_struct_pb.Struct;
      reader.readMessage(value,google_protobuf_struct_pb.Struct.deserializeBinaryFromReader);
      msg.setDeploymentSpec(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setSdkVersion(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setSchemaVersion(value);
      break;
    case 8:
      var value = msg.getComponentsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ComponentSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.ComponentSpec());
         });
      break;
    case 9:
      var value = new proto.ml_pipelines.ComponentSpec;
      reader.readMessage(value,proto.ml_pipelines.ComponentSpec.deserializeBinaryFromReader);
      msg.setRoot(value);
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
proto.ml_pipelines.PipelineSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPipelineInfo();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.PipelineInfo.serializeBinaryToWriter
    );
  }
  f = message.getDeploymentSpec();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      google_protobuf_struct_pb.Struct.serializeBinaryToWriter
    );
  }
  f = message.getSdkVersion();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getSchemaVersion();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getComponentsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(8, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ComponentSpec.serializeBinaryToWriter);
  }
  f = message.getRoot();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      proto.ml_pipelines.ComponentSpec.serializeBinaryToWriter
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
proto.ml_pipelines.PipelineSpec.RuntimeParameter.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineSpec.RuntimeParameter.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineSpec.RuntimeParameter} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineSpec.RuntimeParameter.toObject = function(includeInstance, msg) {
  var f, obj = {
    type: jspb.Message.getFieldWithDefault(msg, 1, 0),
    defaultValue: (f = msg.getDefaultValue()) && proto.ml_pipelines.Value.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineSpec.RuntimeParameter}
 */
proto.ml_pipelines.PipelineSpec.RuntimeParameter.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineSpec.RuntimeParameter;
  return proto.ml_pipelines.PipelineSpec.RuntimeParameter.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineSpec.RuntimeParameter} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineSpec.RuntimeParameter}
 */
proto.ml_pipelines.PipelineSpec.RuntimeParameter.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} */ (reader.readEnum());
      msg.setType(value);
      break;
    case 2:
      var value = new proto.ml_pipelines.Value;
      reader.readMessage(value,proto.ml_pipelines.Value.deserializeBinaryFromReader);
      msg.setDefaultValue(value);
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
proto.ml_pipelines.PipelineSpec.RuntimeParameter.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineSpec.RuntimeParameter.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineSpec.RuntimeParameter} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineSpec.RuntimeParameter.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getType();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
  f = message.getDefaultValue();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.Value.serializeBinaryToWriter
    );
  }
};


/**
 * optional PrimitiveType.PrimitiveTypeEnum type = 1;
 * @return {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum}
 */
proto.ml_pipelines.PipelineSpec.RuntimeParameter.prototype.getType = function() {
  return /** @type {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} value
 * @return {!proto.ml_pipelines.PipelineSpec.RuntimeParameter} returns this
 */
proto.ml_pipelines.PipelineSpec.RuntimeParameter.prototype.setType = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};


/**
 * optional Value default_value = 2;
 * @return {?proto.ml_pipelines.Value}
 */
proto.ml_pipelines.PipelineSpec.RuntimeParameter.prototype.getDefaultValue = function() {
  return /** @type{?proto.ml_pipelines.Value} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.Value, 2));
};


/**
 * @param {?proto.ml_pipelines.Value|undefined} value
 * @return {!proto.ml_pipelines.PipelineSpec.RuntimeParameter} returns this
*/
proto.ml_pipelines.PipelineSpec.RuntimeParameter.prototype.setDefaultValue = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineSpec.RuntimeParameter} returns this
 */
proto.ml_pipelines.PipelineSpec.RuntimeParameter.prototype.clearDefaultValue = function() {
  return this.setDefaultValue(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineSpec.RuntimeParameter.prototype.hasDefaultValue = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional PipelineInfo pipeline_info = 1;
 * @return {?proto.ml_pipelines.PipelineInfo}
 */
proto.ml_pipelines.PipelineSpec.prototype.getPipelineInfo = function() {
  return /** @type{?proto.ml_pipelines.PipelineInfo} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineInfo, 1));
};


/**
 * @param {?proto.ml_pipelines.PipelineInfo|undefined} value
 * @return {!proto.ml_pipelines.PipelineSpec} returns this
*/
proto.ml_pipelines.PipelineSpec.prototype.setPipelineInfo = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineSpec} returns this
 */
proto.ml_pipelines.PipelineSpec.prototype.clearPipelineInfo = function() {
  return this.setPipelineInfo(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineSpec.prototype.hasPipelineInfo = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional google.protobuf.Struct deployment_spec = 7;
 * @return {?proto.google.protobuf.Struct}
 */
proto.ml_pipelines.PipelineSpec.prototype.getDeploymentSpec = function() {
  return /** @type{?proto.google.protobuf.Struct} */ (
    jspb.Message.getWrapperField(this, google_protobuf_struct_pb.Struct, 7));
};


/**
 * @param {?proto.google.protobuf.Struct|undefined} value
 * @return {!proto.ml_pipelines.PipelineSpec} returns this
*/
proto.ml_pipelines.PipelineSpec.prototype.setDeploymentSpec = function(value) {
  return jspb.Message.setWrapperField(this, 7, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineSpec} returns this
 */
proto.ml_pipelines.PipelineSpec.prototype.clearDeploymentSpec = function() {
  return this.setDeploymentSpec(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineSpec.prototype.hasDeploymentSpec = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional string sdk_version = 4;
 * @return {string}
 */
proto.ml_pipelines.PipelineSpec.prototype.getSdkVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineSpec} returns this
 */
proto.ml_pipelines.PipelineSpec.prototype.setSdkVersion = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string schema_version = 5;
 * @return {string}
 */
proto.ml_pipelines.PipelineSpec.prototype.getSchemaVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineSpec} returns this
 */
proto.ml_pipelines.PipelineSpec.prototype.setSchemaVersion = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * map<string, ComponentSpec> components = 8;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ComponentSpec>}
 */
proto.ml_pipelines.PipelineSpec.prototype.getComponentsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ComponentSpec>} */ (
      jspb.Message.getMapField(this, 8, opt_noLazyCreate,
      proto.ml_pipelines.ComponentSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.PipelineSpec} returns this
 */
proto.ml_pipelines.PipelineSpec.prototype.clearComponentsMap = function() {
  this.getComponentsMap().clear();
  return this;};


/**
 * optional ComponentSpec root = 9;
 * @return {?proto.ml_pipelines.ComponentSpec}
 */
proto.ml_pipelines.PipelineSpec.prototype.getRoot = function() {
  return /** @type{?proto.ml_pipelines.ComponentSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ComponentSpec, 9));
};


/**
 * @param {?proto.ml_pipelines.ComponentSpec|undefined} value
 * @return {!proto.ml_pipelines.PipelineSpec} returns this
*/
proto.ml_pipelines.PipelineSpec.prototype.setRoot = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineSpec} returns this
 */
proto.ml_pipelines.PipelineSpec.prototype.clearRoot = function() {
  return this.setRoot(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineSpec.prototype.hasRoot = function() {
  return jspb.Message.getField(this, 9) != null;
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_pipelines.ComponentSpec.oneofGroups_ = [[3,4]];

/**
 * @enum {number}
 */
proto.ml_pipelines.ComponentSpec.ImplementationCase = {
  IMPLEMENTATION_NOT_SET: 0,
  DAG: 3,
  EXECUTOR_LABEL: 4
};

/**
 * @return {proto.ml_pipelines.ComponentSpec.ImplementationCase}
 */
proto.ml_pipelines.ComponentSpec.prototype.getImplementationCase = function() {
  return /** @type {proto.ml_pipelines.ComponentSpec.ImplementationCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.ComponentSpec.oneofGroups_[0]));
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
proto.ml_pipelines.ComponentSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ComponentSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ComponentSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    inputDefinitions: (f = msg.getInputDefinitions()) && proto.ml_pipelines.ComponentInputsSpec.toObject(includeInstance, f),
    outputDefinitions: (f = msg.getOutputDefinitions()) && proto.ml_pipelines.ComponentOutputsSpec.toObject(includeInstance, f),
    dag: (f = msg.getDag()) && proto.ml_pipelines.DagSpec.toObject(includeInstance, f),
    executorLabel: jspb.Message.getFieldWithDefault(msg, 4, "")
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
 * @return {!proto.ml_pipelines.ComponentSpec}
 */
proto.ml_pipelines.ComponentSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ComponentSpec;
  return proto.ml_pipelines.ComponentSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ComponentSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ComponentSpec}
 */
proto.ml_pipelines.ComponentSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.ComponentInputsSpec;
      reader.readMessage(value,proto.ml_pipelines.ComponentInputsSpec.deserializeBinaryFromReader);
      msg.setInputDefinitions(value);
      break;
    case 2:
      var value = new proto.ml_pipelines.ComponentOutputsSpec;
      reader.readMessage(value,proto.ml_pipelines.ComponentOutputsSpec.deserializeBinaryFromReader);
      msg.setOutputDefinitions(value);
      break;
    case 3:
      var value = new proto.ml_pipelines.DagSpec;
      reader.readMessage(value,proto.ml_pipelines.DagSpec.deserializeBinaryFromReader);
      msg.setDag(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setExecutorLabel(value);
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
proto.ml_pipelines.ComponentSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ComponentSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ComponentSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getInputDefinitions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.ComponentInputsSpec.serializeBinaryToWriter
    );
  }
  f = message.getOutputDefinitions();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.ComponentOutputsSpec.serializeBinaryToWriter
    );
  }
  f = message.getDag();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_pipelines.DagSpec.serializeBinaryToWriter
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


/**
 * optional ComponentInputsSpec input_definitions = 1;
 * @return {?proto.ml_pipelines.ComponentInputsSpec}
 */
proto.ml_pipelines.ComponentSpec.prototype.getInputDefinitions = function() {
  return /** @type{?proto.ml_pipelines.ComponentInputsSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ComponentInputsSpec, 1));
};


/**
 * @param {?proto.ml_pipelines.ComponentInputsSpec|undefined} value
 * @return {!proto.ml_pipelines.ComponentSpec} returns this
*/
proto.ml_pipelines.ComponentSpec.prototype.setInputDefinitions = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ComponentSpec} returns this
 */
proto.ml_pipelines.ComponentSpec.prototype.clearInputDefinitions = function() {
  return this.setInputDefinitions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ComponentSpec.prototype.hasInputDefinitions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ComponentOutputsSpec output_definitions = 2;
 * @return {?proto.ml_pipelines.ComponentOutputsSpec}
 */
proto.ml_pipelines.ComponentSpec.prototype.getOutputDefinitions = function() {
  return /** @type{?proto.ml_pipelines.ComponentOutputsSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ComponentOutputsSpec, 2));
};


/**
 * @param {?proto.ml_pipelines.ComponentOutputsSpec|undefined} value
 * @return {!proto.ml_pipelines.ComponentSpec} returns this
*/
proto.ml_pipelines.ComponentSpec.prototype.setOutputDefinitions = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ComponentSpec} returns this
 */
proto.ml_pipelines.ComponentSpec.prototype.clearOutputDefinitions = function() {
  return this.setOutputDefinitions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ComponentSpec.prototype.hasOutputDefinitions = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional DagSpec dag = 3;
 * @return {?proto.ml_pipelines.DagSpec}
 */
proto.ml_pipelines.ComponentSpec.prototype.getDag = function() {
  return /** @type{?proto.ml_pipelines.DagSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.DagSpec, 3));
};


/**
 * @param {?proto.ml_pipelines.DagSpec|undefined} value
 * @return {!proto.ml_pipelines.ComponentSpec} returns this
*/
proto.ml_pipelines.ComponentSpec.prototype.setDag = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.ml_pipelines.ComponentSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ComponentSpec} returns this
 */
proto.ml_pipelines.ComponentSpec.prototype.clearDag = function() {
  return this.setDag(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ComponentSpec.prototype.hasDag = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string executor_label = 4;
 * @return {string}
 */
proto.ml_pipelines.ComponentSpec.prototype.getExecutorLabel = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ComponentSpec} returns this
 */
proto.ml_pipelines.ComponentSpec.prototype.setExecutorLabel = function(value) {
  return jspb.Message.setOneofField(this, 4, proto.ml_pipelines.ComponentSpec.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.ComponentSpec} returns this
 */
proto.ml_pipelines.ComponentSpec.prototype.clearExecutorLabel = function() {
  return jspb.Message.setOneofField(this, 4, proto.ml_pipelines.ComponentSpec.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ComponentSpec.prototype.hasExecutorLabel = function() {
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
proto.ml_pipelines.DagSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.DagSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.DagSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    tasksMap: (f = msg.getTasksMap()) ? f.toObject(includeInstance, proto.ml_pipelines.PipelineTaskSpec.toObject) : [],
    outputs: (f = msg.getOutputs()) && proto.ml_pipelines.DagOutputsSpec.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.DagSpec}
 */
proto.ml_pipelines.DagSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.DagSpec;
  return proto.ml_pipelines.DagSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.DagSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.DagSpec}
 */
proto.ml_pipelines.DagSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getTasksMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.PipelineTaskSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.PipelineTaskSpec());
         });
      break;
    case 2:
      var value = new proto.ml_pipelines.DagOutputsSpec;
      reader.readMessage(value,proto.ml_pipelines.DagOutputsSpec.deserializeBinaryFromReader);
      msg.setOutputs(value);
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
proto.ml_pipelines.DagSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.DagSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.DagSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTasksMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.PipelineTaskSpec.serializeBinaryToWriter);
  }
  f = message.getOutputs();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.DagOutputsSpec.serializeBinaryToWriter
    );
  }
};


/**
 * map<string, PipelineTaskSpec> tasks = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.PipelineTaskSpec>}
 */
proto.ml_pipelines.DagSpec.prototype.getTasksMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.PipelineTaskSpec>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.PipelineTaskSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.DagSpec} returns this
 */
proto.ml_pipelines.DagSpec.prototype.clearTasksMap = function() {
  this.getTasksMap().clear();
  return this;};


/**
 * optional DagOutputsSpec outputs = 2;
 * @return {?proto.ml_pipelines.DagOutputsSpec}
 */
proto.ml_pipelines.DagSpec.prototype.getOutputs = function() {
  return /** @type{?proto.ml_pipelines.DagOutputsSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.DagOutputsSpec, 2));
};


/**
 * @param {?proto.ml_pipelines.DagOutputsSpec|undefined} value
 * @return {!proto.ml_pipelines.DagSpec} returns this
*/
proto.ml_pipelines.DagSpec.prototype.setOutputs = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.DagSpec} returns this
 */
proto.ml_pipelines.DagSpec.prototype.clearOutputs = function() {
  return this.setOutputs(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.DagSpec.prototype.hasOutputs = function() {
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
proto.ml_pipelines.DagOutputsSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.DagOutputsSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.DagOutputsSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsMap: (f = msg.getArtifactsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.toObject) : [],
    parametersMap: (f = msg.getParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.toObject) : []
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
 * @return {!proto.ml_pipelines.DagOutputsSpec}
 */
proto.ml_pipelines.DagOutputsSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.DagOutputsSpec;
  return proto.ml_pipelines.DagOutputsSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.DagOutputsSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.DagOutputsSpec}
 */
proto.ml_pipelines.DagOutputsSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getArtifactsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec());
         });
      break;
    case 2:
      var value = msg.getParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec());
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
proto.ml_pipelines.DagOutputsSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.DagOutputsSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.DagOutputsSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.serializeBinaryToWriter);
  }
  f = message.getParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.serializeBinaryToWriter);
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
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    producerSubtask: jspb.Message.getFieldWithDefault(msg, 1, ""),
    outputArtifactKey: jspb.Message.getFieldWithDefault(msg, 2, "")
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
 * @return {!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec}
 */
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec;
  return proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec}
 */
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProducerSubtask(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setOutputArtifactKey(value);
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
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProducerSubtask();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getOutputArtifactKey();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string producer_subtask = 1;
 * @return {string}
 */
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.prototype.getProducerSubtask = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.prototype.setProducerSubtask = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string output_artifact_key = 2;
 * @return {string}
 */
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.prototype.getOutputArtifactKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.prototype.setOutputArtifactKey = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.repeatedFields_ = [1];



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
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactSelectorsList: jspb.Message.toObjectList(msg.getArtifactSelectorsList(),
    proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.toObject, includeInstance)
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
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec;
  return proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec;
      reader.readMessage(value,proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.deserializeBinaryFromReader);
      msg.addArtifactSelectors(value);
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
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactSelectorsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ArtifactSelectorSpec artifact_selectors = 1;
 * @return {!Array<!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec>}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.prototype.getArtifactSelectorsList = function() {
  return /** @type{!Array<!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec, 1));
};


/**
 * @param {!Array<!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec>} value
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} returns this
*/
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.prototype.setArtifactSelectorsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.prototype.addArtifactSelectors = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.prototype.clearArtifactSelectorsList = function() {
  return this.setArtifactSelectorsList([]);
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
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    producerSubtask: jspb.Message.getFieldWithDefault(msg, 1, ""),
    outputParameterKey: jspb.Message.getFieldWithDefault(msg, 2, "")
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
 * @return {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec}
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec;
  return proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec}
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProducerSubtask(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setOutputParameterKey(value);
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
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProducerSubtask();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getOutputParameterKey();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string producer_subtask = 1;
 * @return {string}
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.prototype.getProducerSubtask = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.prototype.setProducerSubtask = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string output_parameter_key = 2;
 * @return {string}
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.prototype.getOutputParameterKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.prototype.setOutputParameterKey = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.repeatedFields_ = [1];



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
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    parameterSelectorsList: jspb.Message.toObjectList(msg.getParameterSelectorsList(),
    proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.toObject, includeInstance)
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
 * @return {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec}
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec;
  return proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec}
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec;
      reader.readMessage(value,proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.deserializeBinaryFromReader);
      msg.addParameterSelectors(value);
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
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getParameterSelectorsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ParameterSelectorSpec parameter_selectors = 1;
 * @return {!Array<!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec>}
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.prototype.getParameterSelectorsList = function() {
  return /** @type{!Array<!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec, 1));
};


/**
 * @param {!Array<!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec>} value
 * @return {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} returns this
*/
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.prototype.setParameterSelectorsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec}
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.prototype.addParameterSelectors = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.prototype.clearParameterSelectorsList = function() {
  return this.setParameterSelectorsList([]);
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
proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    mappedParametersMap: (f = msg.getMappedParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.toObject) : []
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
 * @return {!proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec}
 */
proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec;
  return proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec}
 */
proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 2:
      var value = msg.getMappedParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec());
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
proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMappedParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.serializeBinaryToWriter);
  }
};


/**
 * map<string, ParameterSelectorSpec> mapped_parameters = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec>}
 */
proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.prototype.getMappedParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.prototype.clearMappedParametersMap = function() {
  this.getMappedParametersMap().clear();
  return this;};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.KindCase = {
  KIND_NOT_SET: 0,
  VALUE_FROM_PARAMETER: 1,
  VALUE_FROM_ONEOF: 2
};

/**
 * @return {proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.KindCase}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.getKindCase = function() {
  return /** @type {proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.KindCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.oneofGroups_[0]));
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
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    valueFromParameter: (f = msg.getValueFromParameter()) && proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.toObject(includeInstance, f),
    valueFromOneof: (f = msg.getValueFromOneof()) && proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec;
  return proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec;
      reader.readMessage(value,proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.deserializeBinaryFromReader);
      msg.setValueFromParameter(value);
      break;
    case 2:
      var value = new proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec;
      reader.readMessage(value,proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.deserializeBinaryFromReader);
      msg.setValueFromOneof(value);
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
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getValueFromParameter();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.serializeBinaryToWriter
    );
  }
  f = message.getValueFromOneof();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.serializeBinaryToWriter
    );
  }
};


/**
 * optional ParameterSelectorSpec value_from_parameter = 1;
 * @return {?proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.getValueFromParameter = function() {
  return /** @type{?proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec, 1));
};


/**
 * @param {?proto.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec|undefined} value
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} returns this
*/
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.setValueFromParameter = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.clearValueFromParameter = function() {
  return this.setValueFromParameter(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.hasValueFromParameter = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ParameterSelectorsSpec value_from_oneof = 2;
 * @return {?proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.getValueFromOneof = function() {
  return /** @type{?proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec, 2));
};


/**
 * @param {?proto.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec|undefined} value
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} returns this
*/
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.setValueFromOneof = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.clearValueFromOneof = function() {
  return this.setValueFromOneof(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.prototype.hasValueFromOneof = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * map<string, DagOutputArtifactSpec> artifacts = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec>}
 */
proto.ml_pipelines.DagOutputsSpec.prototype.getArtifactsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.DagOutputsSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.prototype.clearArtifactsMap = function() {
  this.getArtifactsMap().clear();
  return this;};


/**
 * map<string, DagOutputParameterSpec> parameters = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec>}
 */
proto.ml_pipelines.DagOutputsSpec.prototype.getParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.DagOutputsSpec} returns this
 */
proto.ml_pipelines.DagOutputsSpec.prototype.clearParametersMap = function() {
  this.getParametersMap().clear();
  return this;};





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
proto.ml_pipelines.ComponentInputsSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ComponentInputsSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ComponentInputsSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentInputsSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsMap: (f = msg.getArtifactsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.toObject) : [],
    parametersMap: (f = msg.getParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.toObject) : []
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
 * @return {!proto.ml_pipelines.ComponentInputsSpec}
 */
proto.ml_pipelines.ComponentInputsSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ComponentInputsSpec;
  return proto.ml_pipelines.ComponentInputsSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ComponentInputsSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ComponentInputsSpec}
 */
proto.ml_pipelines.ComponentInputsSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getArtifactsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec());
         });
      break;
    case 2:
      var value = msg.getParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.ComponentInputsSpec.ParameterSpec());
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
proto.ml_pipelines.ComponentInputsSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ComponentInputsSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ComponentInputsSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentInputsSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.serializeBinaryToWriter);
  }
  f = message.getParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.serializeBinaryToWriter);
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
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactType: (f = msg.getArtifactType()) && proto.ml_pipelines.ArtifactTypeSchema.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec}
 */
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec;
  return proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec}
 */
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.ArtifactTypeSchema;
      reader.readMessage(value,proto.ml_pipelines.ArtifactTypeSchema.deserializeBinaryFromReader);
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
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactType();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.ArtifactTypeSchema.serializeBinaryToWriter
    );
  }
};


/**
 * optional ArtifactTypeSchema artifact_type = 1;
 * @return {?proto.ml_pipelines.ArtifactTypeSchema}
 */
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.prototype.getArtifactType = function() {
  return /** @type{?proto.ml_pipelines.ArtifactTypeSchema} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ArtifactTypeSchema, 1));
};


/**
 * @param {?proto.ml_pipelines.ArtifactTypeSchema|undefined} value
 * @return {!proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec} returns this
*/
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.prototype.setArtifactType = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec} returns this
 */
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.prototype.clearArtifactType = function() {
  return this.setArtifactType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec.prototype.hasArtifactType = function() {
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
proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ComponentInputsSpec.ParameterSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    type: jspb.Message.getFieldWithDefault(msg, 1, 0)
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
 * @return {!proto.ml_pipelines.ComponentInputsSpec.ParameterSpec}
 */
proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ComponentInputsSpec.ParameterSpec;
  return proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ComponentInputsSpec.ParameterSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ComponentInputsSpec.ParameterSpec}
 */
proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} */ (reader.readEnum());
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
proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ComponentInputsSpec.ParameterSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getType();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
};


/**
 * optional PrimitiveType.PrimitiveTypeEnum type = 1;
 * @return {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum}
 */
proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.prototype.getType = function() {
  return /** @type {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} value
 * @return {!proto.ml_pipelines.ComponentInputsSpec.ParameterSpec} returns this
 */
proto.ml_pipelines.ComponentInputsSpec.ParameterSpec.prototype.setType = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};


/**
 * map<string, ArtifactSpec> artifacts = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec>}
 */
proto.ml_pipelines.ComponentInputsSpec.prototype.getArtifactsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.ComponentInputsSpec.ArtifactSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ComponentInputsSpec} returns this
 */
proto.ml_pipelines.ComponentInputsSpec.prototype.clearArtifactsMap = function() {
  this.getArtifactsMap().clear();
  return this;};


/**
 * map<string, ParameterSpec> parameters = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ComponentInputsSpec.ParameterSpec>}
 */
proto.ml_pipelines.ComponentInputsSpec.prototype.getParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ComponentInputsSpec.ParameterSpec>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.ComponentInputsSpec.ParameterSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ComponentInputsSpec} returns this
 */
proto.ml_pipelines.ComponentInputsSpec.prototype.clearParametersMap = function() {
  this.getParametersMap().clear();
  return this;};





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
proto.ml_pipelines.ComponentOutputsSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ComponentOutputsSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ComponentOutputsSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentOutputsSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsMap: (f = msg.getArtifactsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.toObject) : [],
    parametersMap: (f = msg.getParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.toObject) : []
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
 * @return {!proto.ml_pipelines.ComponentOutputsSpec}
 */
proto.ml_pipelines.ComponentOutputsSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ComponentOutputsSpec;
  return proto.ml_pipelines.ComponentOutputsSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ComponentOutputsSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ComponentOutputsSpec}
 */
proto.ml_pipelines.ComponentOutputsSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getArtifactsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec());
         });
      break;
    case 2:
      var value = msg.getParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec());
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
proto.ml_pipelines.ComponentOutputsSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ComponentOutputsSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ComponentOutputsSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentOutputsSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.serializeBinaryToWriter);
  }
  f = message.getParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.serializeBinaryToWriter);
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
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactType: (f = msg.getArtifactType()) && proto.ml_pipelines.ArtifactTypeSchema.toObject(includeInstance, f),
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ValueOrRuntimeParameter.toObject) : [],
    customPropertiesMap: (f = msg.getCustomPropertiesMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ValueOrRuntimeParameter.toObject) : [],
    metadata: (f = msg.getMetadata()) && google_protobuf_struct_pb.Struct.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec}
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec;
  return proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec}
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.ArtifactTypeSchema;
      reader.readMessage(value,proto.ml_pipelines.ArtifactTypeSchema.deserializeBinaryFromReader);
      msg.setArtifactType(value);
      break;
    case 2:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader, "", new proto.ml_pipelines.ValueOrRuntimeParameter());
         });
      break;
    case 3:
      var value = msg.getCustomPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader, "", new proto.ml_pipelines.ValueOrRuntimeParameter());
         });
      break;
    case 4:
      var value = new google_protobuf_struct_pb.Struct;
      reader.readMessage(value,google_protobuf_struct_pb.Struct.deserializeBinaryFromReader);
      msg.setMetadata(value);
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
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactType();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.ArtifactTypeSchema.serializeBinaryToWriter
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter);
  }
  f = message.getCustomPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(3, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter);
  }
  f = message.getMetadata();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      google_protobuf_struct_pb.Struct.serializeBinaryToWriter
    );
  }
};


/**
 * optional ArtifactTypeSchema artifact_type = 1;
 * @return {?proto.ml_pipelines.ArtifactTypeSchema}
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.getArtifactType = function() {
  return /** @type{?proto.ml_pipelines.ArtifactTypeSchema} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ArtifactTypeSchema, 1));
};


/**
 * @param {?proto.ml_pipelines.ArtifactTypeSchema|undefined} value
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec} returns this
*/
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.setArtifactType = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec} returns this
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.clearArtifactType = function() {
  return this.setArtifactType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.hasArtifactType = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * map<string, ValueOrRuntimeParameter> properties = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>}
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.ValueOrRuntimeParameter));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec} returns this
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * map<string, ValueOrRuntimeParameter> custom_properties = 3;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>}
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.getCustomPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>} */ (
      jspb.Message.getMapField(this, 3, opt_noLazyCreate,
      proto.ml_pipelines.ValueOrRuntimeParameter));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec} returns this
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.clearCustomPropertiesMap = function() {
  this.getCustomPropertiesMap().clear();
  return this;};


/**
 * optional google.protobuf.Struct metadata = 4;
 * @return {?proto.google.protobuf.Struct}
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.getMetadata = function() {
  return /** @type{?proto.google.protobuf.Struct} */ (
    jspb.Message.getWrapperField(this, google_protobuf_struct_pb.Struct, 4));
};


/**
 * @param {?proto.google.protobuf.Struct|undefined} value
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec} returns this
*/
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.setMetadata = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec} returns this
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.clearMetadata = function() {
  return this.setMetadata(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.prototype.hasMetadata = function() {
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
proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    type: jspb.Message.getFieldWithDefault(msg, 1, 0)
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
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec}
 */
proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec;
  return proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec}
 */
proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} */ (reader.readEnum());
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
proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getType();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
};


/**
 * optional PrimitiveType.PrimitiveTypeEnum type = 1;
 * @return {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum}
 */
proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.prototype.getType = function() {
  return /** @type {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} value
 * @return {!proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec} returns this
 */
proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec.prototype.setType = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};


/**
 * map<string, ArtifactSpec> artifacts = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec>}
 */
proto.ml_pipelines.ComponentOutputsSpec.prototype.getArtifactsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.ComponentOutputsSpec.ArtifactSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ComponentOutputsSpec} returns this
 */
proto.ml_pipelines.ComponentOutputsSpec.prototype.clearArtifactsMap = function() {
  this.getArtifactsMap().clear();
  return this;};


/**
 * map<string, ParameterSpec> parameters = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec>}
 */
proto.ml_pipelines.ComponentOutputsSpec.prototype.getParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.ComponentOutputsSpec.ParameterSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ComponentOutputsSpec} returns this
 */
proto.ml_pipelines.ComponentOutputsSpec.prototype.clearParametersMap = function() {
  this.getParametersMap().clear();
  return this;};





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
proto.ml_pipelines.TaskInputsSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.TaskInputsSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.TaskInputsSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    parametersMap: (f = msg.getParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.toObject) : [],
    artifactsMap: (f = msg.getArtifactsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.toObject) : []
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
 * @return {!proto.ml_pipelines.TaskInputsSpec}
 */
proto.ml_pipelines.TaskInputsSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.TaskInputsSpec;
  return proto.ml_pipelines.TaskInputsSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.TaskInputsSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.TaskInputsSpec}
 */
proto.ml_pipelines.TaskInputsSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.TaskInputsSpec.InputParameterSpec());
         });
      break;
    case 2:
      var value = msg.getArtifactsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec());
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
proto.ml_pipelines.TaskInputsSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.TaskInputsSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.TaskInputsSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.serializeBinaryToWriter);
  }
  f = message.getArtifactsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.serializeBinaryToWriter);
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
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.oneofGroups_ = [[3,4]];

/**
 * @enum {number}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.KindCase = {
  KIND_NOT_SET: 0,
  TASK_OUTPUT_ARTIFACT: 3,
  COMPONENT_INPUT_ARTIFACT: 4
};

/**
 * @return {proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.KindCase}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.getKindCase = function() {
  return /** @type {proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.KindCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.oneofGroups_[0]));
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
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    taskOutputArtifact: (f = msg.getTaskOutputArtifact()) && proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.toObject(includeInstance, f),
    componentInputArtifact: jspb.Message.getFieldWithDefault(msg, 4, "")
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
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec;
  return proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 3:
      var value = new proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec;
      reader.readMessage(value,proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.deserializeBinaryFromReader);
      msg.setTaskOutputArtifact(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setComponentInputArtifact(value);
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
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTaskOutputArtifact();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.serializeBinaryToWriter
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
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    producerTask: jspb.Message.getFieldWithDefault(msg, 1, ""),
    outputArtifactKey: jspb.Message.getFieldWithDefault(msg, 2, "")
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
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec;
  return proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProducerTask(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setOutputArtifactKey(value);
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
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProducerTask();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getOutputArtifactKey();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string producer_task = 1;
 * @return {string}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.prototype.getProducerTask = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.prototype.setProducerTask = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string output_artifact_key = 2;
 * @return {string}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.prototype.getOutputArtifactKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.prototype.setOutputArtifactKey = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional TaskOutputArtifactSpec task_output_artifact = 3;
 * @return {?proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.getTaskOutputArtifact = function() {
  return /** @type{?proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec, 3));
};


/**
 * @param {?proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec|undefined} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec} returns this
*/
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.setTaskOutputArtifact = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.clearTaskOutputArtifact = function() {
  return this.setTaskOutputArtifact(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.hasTaskOutputArtifact = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string component_input_artifact = 4;
 * @return {string}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.getComponentInputArtifact = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.setComponentInputArtifact = function(value) {
  return jspb.Message.setOneofField(this, 4, proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.clearComponentInputArtifact = function() {
  return jspb.Message.setOneofField(this, 4, proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec.prototype.hasComponentInputArtifact = function() {
  return jspb.Message.getField(this, 4) != null;
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.oneofGroups_ = [[1,2,3,5]];

/**
 * @enum {number}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.KindCase = {
  KIND_NOT_SET: 0,
  TASK_OUTPUT_PARAMETER: 1,
  RUNTIME_VALUE: 2,
  COMPONENT_INPUT_PARAMETER: 3,
  TASK_FINAL_STATUS: 5
};

/**
 * @return {proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.KindCase}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.getKindCase = function() {
  return /** @type {proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.KindCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.oneofGroups_[0]));
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
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    taskOutputParameter: (f = msg.getTaskOutputParameter()) && proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.toObject(includeInstance, f),
    runtimeValue: (f = msg.getRuntimeValue()) && proto.ml_pipelines.ValueOrRuntimeParameter.toObject(includeInstance, f),
    componentInputParameter: jspb.Message.getFieldWithDefault(msg, 3, ""),
    taskFinalStatus: (f = msg.getTaskFinalStatus()) && proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.toObject(includeInstance, f),
    parameterExpressionSelector: jspb.Message.getFieldWithDefault(msg, 4, "")
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
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.TaskInputsSpec.InputParameterSpec;
  return proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec;
      reader.readMessage(value,proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.deserializeBinaryFromReader);
      msg.setTaskOutputParameter(value);
      break;
    case 2:
      var value = new proto.ml_pipelines.ValueOrRuntimeParameter;
      reader.readMessage(value,proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader);
      msg.setRuntimeValue(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setComponentInputParameter(value);
      break;
    case 5:
      var value = new proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus;
      reader.readMessage(value,proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.deserializeBinaryFromReader);
      msg.setTaskFinalStatus(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setParameterExpressionSelector(value);
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
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTaskOutputParameter();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.serializeBinaryToWriter
    );
  }
  f = message.getRuntimeValue();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getTaskFinalStatus();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.serializeBinaryToWriter
    );
  }
  f = message.getParameterExpressionSelector();
  if (f.length > 0) {
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
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    producerTask: jspb.Message.getFieldWithDefault(msg, 1, ""),
    outputParameterKey: jspb.Message.getFieldWithDefault(msg, 2, "")
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
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec;
  return proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProducerTask(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setOutputParameterKey(value);
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
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProducerTask();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getOutputParameterKey();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string producer_task = 1;
 * @return {string}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.prototype.getProducerTask = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.prototype.setProducerTask = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string output_parameter_key = 2;
 * @return {string}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.prototype.getOutputParameterKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.prototype.setOutputParameterKey = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
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
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.toObject = function(includeInstance, msg) {
  var f, obj = {
    producerTask: jspb.Message.getFieldWithDefault(msg, 1, "")
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
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus;
  return proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setProducerTask(value);
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
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getProducerTask();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string producer_task = 1;
 * @return {string}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.prototype.getProducerTask = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.prototype.setProducerTask = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional TaskOutputParameterSpec task_output_parameter = 1;
 * @return {?proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.getTaskOutputParameter = function() {
  return /** @type{?proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec, 1));
};


/**
 * @param {?proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec|undefined} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} returns this
*/
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.setTaskOutputParameter = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.clearTaskOutputParameter = function() {
  return this.setTaskOutputParameter(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.hasTaskOutputParameter = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ValueOrRuntimeParameter runtime_value = 2;
 * @return {?proto.ml_pipelines.ValueOrRuntimeParameter}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.getRuntimeValue = function() {
  return /** @type{?proto.ml_pipelines.ValueOrRuntimeParameter} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ValueOrRuntimeParameter, 2));
};


/**
 * @param {?proto.ml_pipelines.ValueOrRuntimeParameter|undefined} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} returns this
*/
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.setRuntimeValue = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.clearRuntimeValue = function() {
  return this.setRuntimeValue(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.hasRuntimeValue = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string component_input_parameter = 3;
 * @return {string}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.getComponentInputParameter = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.setComponentInputParameter = function(value) {
  return jspb.Message.setOneofField(this, 3, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.clearComponentInputParameter = function() {
  return jspb.Message.setOneofField(this, 3, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.hasComponentInputParameter = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional TaskFinalStatus task_final_status = 5;
 * @return {?proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.getTaskFinalStatus = function() {
  return /** @type{?proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus, 5));
};


/**
 * @param {?proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus|undefined} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} returns this
*/
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.setTaskFinalStatus = function(value) {
  return jspb.Message.setOneofWrapperField(this, 5, proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.clearTaskFinalStatus = function() {
  return this.setTaskFinalStatus(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.hasTaskFinalStatus = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional string parameter_expression_selector = 4;
 * @return {string}
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.getParameterExpressionSelector = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.InputParameterSpec.prototype.setParameterExpressionSelector = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * map<string, InputParameterSpec> parameters = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec>}
 */
proto.ml_pipelines.TaskInputsSpec.prototype.getParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.TaskInputsSpec.InputParameterSpec>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.TaskInputsSpec.InputParameterSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.TaskInputsSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.prototype.clearParametersMap = function() {
  this.getParametersMap().clear();
  return this;};


/**
 * map<string, InputArtifactSpec> artifacts = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec>}
 */
proto.ml_pipelines.TaskInputsSpec.prototype.getArtifactsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.TaskInputsSpec.InputArtifactSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.TaskInputsSpec} returns this
 */
proto.ml_pipelines.TaskInputsSpec.prototype.clearArtifactsMap = function() {
  this.getArtifactsMap().clear();
  return this;};





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
proto.ml_pipelines.TaskOutputsSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.TaskOutputsSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.TaskOutputsSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskOutputsSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    parametersMap: (f = msg.getParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.toObject) : [],
    artifactsMap: (f = msg.getArtifactsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.toObject) : []
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
 * @return {!proto.ml_pipelines.TaskOutputsSpec}
 */
proto.ml_pipelines.TaskOutputsSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.TaskOutputsSpec;
  return proto.ml_pipelines.TaskOutputsSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.TaskOutputsSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.TaskOutputsSpec}
 */
proto.ml_pipelines.TaskOutputsSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec());
         });
      break;
    case 2:
      var value = msg.getArtifactsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec());
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
proto.ml_pipelines.TaskOutputsSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.TaskOutputsSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.TaskOutputsSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskOutputsSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.serializeBinaryToWriter);
  }
  f = message.getArtifactsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.serializeBinaryToWriter);
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
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactType: (f = msg.getArtifactType()) && proto.ml_pipelines.ArtifactTypeSchema.toObject(includeInstance, f),
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ValueOrRuntimeParameter.toObject) : [],
    customPropertiesMap: (f = msg.getCustomPropertiesMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ValueOrRuntimeParameter.toObject) : []
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
 * @return {!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec}
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec;
  return proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec}
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.ArtifactTypeSchema;
      reader.readMessage(value,proto.ml_pipelines.ArtifactTypeSchema.deserializeBinaryFromReader);
      msg.setArtifactType(value);
      break;
    case 2:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader, "", new proto.ml_pipelines.ValueOrRuntimeParameter());
         });
      break;
    case 3:
      var value = msg.getCustomPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader, "", new proto.ml_pipelines.ValueOrRuntimeParameter());
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
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactType();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.ArtifactTypeSchema.serializeBinaryToWriter
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter);
  }
  f = message.getCustomPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(3, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter);
  }
};


/**
 * optional ArtifactTypeSchema artifact_type = 1;
 * @return {?proto.ml_pipelines.ArtifactTypeSchema}
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.getArtifactType = function() {
  return /** @type{?proto.ml_pipelines.ArtifactTypeSchema} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ArtifactTypeSchema, 1));
};


/**
 * @param {?proto.ml_pipelines.ArtifactTypeSchema|undefined} value
 * @return {!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} returns this
*/
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.setArtifactType = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} returns this
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.clearArtifactType = function() {
  return this.setArtifactType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.hasArtifactType = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * map<string, ValueOrRuntimeParameter> properties = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>}
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.ValueOrRuntimeParameter));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} returns this
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * map<string, ValueOrRuntimeParameter> custom_properties = 3;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>}
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.getCustomPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>} */ (
      jspb.Message.getMapField(this, 3, opt_noLazyCreate,
      proto.ml_pipelines.ValueOrRuntimeParameter));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} returns this
 */
proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.prototype.clearCustomPropertiesMap = function() {
  this.getCustomPropertiesMap().clear();
  return this;};





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
proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    type: jspb.Message.getFieldWithDefault(msg, 1, 0)
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
 * @return {!proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec}
 */
proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec;
  return proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec}
 */
proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} */ (reader.readEnum());
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
proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getType();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
};


/**
 * optional PrimitiveType.PrimitiveTypeEnum type = 1;
 * @return {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum}
 */
proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.prototype.getType = function() {
  return /** @type {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum} value
 * @return {!proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec} returns this
 */
proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.prototype.setType = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};


/**
 * map<string, OutputParameterSpec> parameters = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec>}
 */
proto.ml_pipelines.TaskOutputsSpec.prototype.getParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.TaskOutputsSpec.OutputParameterSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.TaskOutputsSpec} returns this
 */
proto.ml_pipelines.TaskOutputsSpec.prototype.clearParametersMap = function() {
  this.getParametersMap().clear();
  return this;};


/**
 * map<string, OutputArtifactSpec> artifacts = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec>}
 */
proto.ml_pipelines.TaskOutputsSpec.prototype.getArtifactsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.TaskOutputsSpec} returns this
 */
proto.ml_pipelines.TaskOutputsSpec.prototype.clearArtifactsMap = function() {
  this.getArtifactsMap().clear();
  return this;};





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
proto.ml_pipelines.PrimitiveType.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PrimitiveType.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PrimitiveType} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PrimitiveType.toObject = function(includeInstance, msg) {
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
 * @return {!proto.ml_pipelines.PrimitiveType}
 */
proto.ml_pipelines.PrimitiveType.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PrimitiveType;
  return proto.ml_pipelines.PrimitiveType.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PrimitiveType} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PrimitiveType}
 */
proto.ml_pipelines.PrimitiveType.deserializeBinaryFromReader = function(msg, reader) {
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
proto.ml_pipelines.PrimitiveType.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PrimitiveType.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PrimitiveType} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PrimitiveType.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};


/**
 * @enum {number}
 */
proto.ml_pipelines.PrimitiveType.PrimitiveTypeEnum = {
  PRIMITIVE_TYPE_UNSPECIFIED: 0,
  INT: 1,
  DOUBLE: 2,
  STRING: 3
};


/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_pipelines.PipelineTaskSpec.repeatedFields_ = [5];

/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_pipelines.PipelineTaskSpec.oneofGroups_ = [[9,10]];

/**
 * @enum {number}
 */
proto.ml_pipelines.PipelineTaskSpec.IteratorCase = {
  ITERATOR_NOT_SET: 0,
  ARTIFACT_ITERATOR: 9,
  PARAMETER_ITERATOR: 10
};

/**
 * @return {proto.ml_pipelines.PipelineTaskSpec.IteratorCase}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.getIteratorCase = function() {
  return /** @type {proto.ml_pipelines.PipelineTaskSpec.IteratorCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.PipelineTaskSpec.oneofGroups_[0]));
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
proto.ml_pipelines.PipelineTaskSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineTaskSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineTaskSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    taskInfo: (f = msg.getTaskInfo()) && proto.ml_pipelines.PipelineTaskInfo.toObject(includeInstance, f),
    inputs: (f = msg.getInputs()) && proto.ml_pipelines.TaskInputsSpec.toObject(includeInstance, f),
    dependentTasksList: (f = jspb.Message.getRepeatedField(msg, 5)) == null ? undefined : f,
    cachingOptions: (f = msg.getCachingOptions()) && proto.ml_pipelines.PipelineTaskSpec.CachingOptions.toObject(includeInstance, f),
    componentRef: (f = msg.getComponentRef()) && proto.ml_pipelines.ComponentRef.toObject(includeInstance, f),
    triggerPolicy: (f = msg.getTriggerPolicy()) && proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.toObject(includeInstance, f),
    artifactIterator: (f = msg.getArtifactIterator()) && proto.ml_pipelines.ArtifactIteratorSpec.toObject(includeInstance, f),
    parameterIterator: (f = msg.getParameterIterator()) && proto.ml_pipelines.ParameterIteratorSpec.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineTaskSpec}
 */
proto.ml_pipelines.PipelineTaskSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineTaskSpec;
  return proto.ml_pipelines.PipelineTaskSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineTaskSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineTaskSpec}
 */
proto.ml_pipelines.PipelineTaskSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.PipelineTaskInfo;
      reader.readMessage(value,proto.ml_pipelines.PipelineTaskInfo.deserializeBinaryFromReader);
      msg.setTaskInfo(value);
      break;
    case 2:
      var value = new proto.ml_pipelines.TaskInputsSpec;
      reader.readMessage(value,proto.ml_pipelines.TaskInputsSpec.deserializeBinaryFromReader);
      msg.setInputs(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.addDependentTasks(value);
      break;
    case 6:
      var value = new proto.ml_pipelines.PipelineTaskSpec.CachingOptions;
      reader.readMessage(value,proto.ml_pipelines.PipelineTaskSpec.CachingOptions.deserializeBinaryFromReader);
      msg.setCachingOptions(value);
      break;
    case 7:
      var value = new proto.ml_pipelines.ComponentRef;
      reader.readMessage(value,proto.ml_pipelines.ComponentRef.deserializeBinaryFromReader);
      msg.setComponentRef(value);
      break;
    case 8:
      var value = new proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy;
      reader.readMessage(value,proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.deserializeBinaryFromReader);
      msg.setTriggerPolicy(value);
      break;
    case 9:
      var value = new proto.ml_pipelines.ArtifactIteratorSpec;
      reader.readMessage(value,proto.ml_pipelines.ArtifactIteratorSpec.deserializeBinaryFromReader);
      msg.setArtifactIterator(value);
      break;
    case 10:
      var value = new proto.ml_pipelines.ParameterIteratorSpec;
      reader.readMessage(value,proto.ml_pipelines.ParameterIteratorSpec.deserializeBinaryFromReader);
      msg.setParameterIterator(value);
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
proto.ml_pipelines.PipelineTaskSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineTaskSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineTaskSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTaskInfo();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.PipelineTaskInfo.serializeBinaryToWriter
    );
  }
  f = message.getInputs();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.TaskInputsSpec.serializeBinaryToWriter
    );
  }
  f = message.getDependentTasksList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      5,
      f
    );
  }
  f = message.getCachingOptions();
  if (f != null) {
    writer.writeMessage(
      6,
      f,
      proto.ml_pipelines.PipelineTaskSpec.CachingOptions.serializeBinaryToWriter
    );
  }
  f = message.getComponentRef();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      proto.ml_pipelines.ComponentRef.serializeBinaryToWriter
    );
  }
  f = message.getTriggerPolicy();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.serializeBinaryToWriter
    );
  }
  f = message.getArtifactIterator();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      proto.ml_pipelines.ArtifactIteratorSpec.serializeBinaryToWriter
    );
  }
  f = message.getParameterIterator();
  if (f != null) {
    writer.writeMessage(
      10,
      f,
      proto.ml_pipelines.ParameterIteratorSpec.serializeBinaryToWriter
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
proto.ml_pipelines.PipelineTaskSpec.CachingOptions.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineTaskSpec.CachingOptions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineTaskSpec.CachingOptions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskSpec.CachingOptions.toObject = function(includeInstance, msg) {
  var f, obj = {
    enableCache: jspb.Message.getBooleanFieldWithDefault(msg, 1, false)
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
 * @return {!proto.ml_pipelines.PipelineTaskSpec.CachingOptions}
 */
proto.ml_pipelines.PipelineTaskSpec.CachingOptions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineTaskSpec.CachingOptions;
  return proto.ml_pipelines.PipelineTaskSpec.CachingOptions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineTaskSpec.CachingOptions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineTaskSpec.CachingOptions}
 */
proto.ml_pipelines.PipelineTaskSpec.CachingOptions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setEnableCache(value);
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
proto.ml_pipelines.PipelineTaskSpec.CachingOptions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineTaskSpec.CachingOptions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineTaskSpec.CachingOptions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskSpec.CachingOptions.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEnableCache();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
};


/**
 * optional bool enable_cache = 1;
 * @return {boolean}
 */
proto.ml_pipelines.PipelineTaskSpec.CachingOptions.prototype.getEnableCache = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec.CachingOptions} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.CachingOptions.prototype.setEnableCache = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
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
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.toObject = function(includeInstance, msg) {
  var f, obj = {
    condition: jspb.Message.getFieldWithDefault(msg, 1, ""),
    strategy: jspb.Message.getFieldWithDefault(msg, 2, 0)
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
 * @return {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy}
 */
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy;
  return proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy}
 */
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCondition(value);
      break;
    case 2:
      var value = /** @type {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy} */ (reader.readEnum());
      msg.setStrategy(value);
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
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCondition();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getStrategy();
  if (f !== 0.0) {
    writer.writeEnum(
      2,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy = {
  TRIGGER_STRATEGY_UNSPECIFIED: 0,
  ALL_UPSTREAM_TASKS_SUCCEEDED: 1,
  ALL_UPSTREAM_TASKS_COMPLETED: 2
};

/**
 * optional string condition = 1;
 * @return {string}
 */
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.prototype.getCondition = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.prototype.setCondition = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional TriggerStrategy strategy = 2;
 * @return {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy}
 */
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.prototype.getStrategy = function() {
  return /** @type {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy.prototype.setStrategy = function(value) {
  return jspb.Message.setProto3EnumField(this, 2, value);
};


/**
 * optional PipelineTaskInfo task_info = 1;
 * @return {?proto.ml_pipelines.PipelineTaskInfo}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.getTaskInfo = function() {
  return /** @type{?proto.ml_pipelines.PipelineTaskInfo} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineTaskInfo, 1));
};


/**
 * @param {?proto.ml_pipelines.PipelineTaskInfo|undefined} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
*/
proto.ml_pipelines.PipelineTaskSpec.prototype.setTaskInfo = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.clearTaskInfo = function() {
  return this.setTaskInfo(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.hasTaskInfo = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional TaskInputsSpec inputs = 2;
 * @return {?proto.ml_pipelines.TaskInputsSpec}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.getInputs = function() {
  return /** @type{?proto.ml_pipelines.TaskInputsSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.TaskInputsSpec, 2));
};


/**
 * @param {?proto.ml_pipelines.TaskInputsSpec|undefined} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
*/
proto.ml_pipelines.PipelineTaskSpec.prototype.setInputs = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.clearInputs = function() {
  return this.setInputs(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.hasInputs = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * repeated string dependent_tasks = 5;
 * @return {!Array<string>}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.getDependentTasksList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 5));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.setDependentTasksList = function(value) {
  return jspb.Message.setField(this, 5, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.addDependentTasks = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 5, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.clearDependentTasksList = function() {
  return this.setDependentTasksList([]);
};


/**
 * optional CachingOptions caching_options = 6;
 * @return {?proto.ml_pipelines.PipelineTaskSpec.CachingOptions}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.getCachingOptions = function() {
  return /** @type{?proto.ml_pipelines.PipelineTaskSpec.CachingOptions} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineTaskSpec.CachingOptions, 6));
};


/**
 * @param {?proto.ml_pipelines.PipelineTaskSpec.CachingOptions|undefined} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
*/
proto.ml_pipelines.PipelineTaskSpec.prototype.setCachingOptions = function(value) {
  return jspb.Message.setWrapperField(this, 6, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.clearCachingOptions = function() {
  return this.setCachingOptions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.hasCachingOptions = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional ComponentRef component_ref = 7;
 * @return {?proto.ml_pipelines.ComponentRef}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.getComponentRef = function() {
  return /** @type{?proto.ml_pipelines.ComponentRef} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ComponentRef, 7));
};


/**
 * @param {?proto.ml_pipelines.ComponentRef|undefined} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
*/
proto.ml_pipelines.PipelineTaskSpec.prototype.setComponentRef = function(value) {
  return jspb.Message.setWrapperField(this, 7, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.clearComponentRef = function() {
  return this.setComponentRef(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.hasComponentRef = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional TriggerPolicy trigger_policy = 8;
 * @return {?proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.getTriggerPolicy = function() {
  return /** @type{?proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy, 8));
};


/**
 * @param {?proto.ml_pipelines.PipelineTaskSpec.TriggerPolicy|undefined} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
*/
proto.ml_pipelines.PipelineTaskSpec.prototype.setTriggerPolicy = function(value) {
  return jspb.Message.setWrapperField(this, 8, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.clearTriggerPolicy = function() {
  return this.setTriggerPolicy(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.hasTriggerPolicy = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional ArtifactIteratorSpec artifact_iterator = 9;
 * @return {?proto.ml_pipelines.ArtifactIteratorSpec}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.getArtifactIterator = function() {
  return /** @type{?proto.ml_pipelines.ArtifactIteratorSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ArtifactIteratorSpec, 9));
};


/**
 * @param {?proto.ml_pipelines.ArtifactIteratorSpec|undefined} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
*/
proto.ml_pipelines.PipelineTaskSpec.prototype.setArtifactIterator = function(value) {
  return jspb.Message.setOneofWrapperField(this, 9, proto.ml_pipelines.PipelineTaskSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.clearArtifactIterator = function() {
  return this.setArtifactIterator(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.hasArtifactIterator = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional ParameterIteratorSpec parameter_iterator = 10;
 * @return {?proto.ml_pipelines.ParameterIteratorSpec}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.getParameterIterator = function() {
  return /** @type{?proto.ml_pipelines.ParameterIteratorSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ParameterIteratorSpec, 10));
};


/**
 * @param {?proto.ml_pipelines.ParameterIteratorSpec|undefined} value
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
*/
proto.ml_pipelines.PipelineTaskSpec.prototype.setParameterIterator = function(value) {
  return jspb.Message.setOneofWrapperField(this, 10, proto.ml_pipelines.PipelineTaskSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineTaskSpec} returns this
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.clearParameterIterator = function() {
  return this.setParameterIterator(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineTaskSpec.prototype.hasParameterIterator = function() {
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
proto.ml_pipelines.ArtifactIteratorSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ArtifactIteratorSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ArtifactIteratorSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ArtifactIteratorSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    items: (f = msg.getItems()) && proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.toObject(includeInstance, f),
    itemInput: jspb.Message.getFieldWithDefault(msg, 2, "")
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
 * @return {!proto.ml_pipelines.ArtifactIteratorSpec}
 */
proto.ml_pipelines.ArtifactIteratorSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ArtifactIteratorSpec;
  return proto.ml_pipelines.ArtifactIteratorSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ArtifactIteratorSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ArtifactIteratorSpec}
 */
proto.ml_pipelines.ArtifactIteratorSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec;
      reader.readMessage(value,proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.deserializeBinaryFromReader);
      msg.setItems(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setItemInput(value);
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
proto.ml_pipelines.ArtifactIteratorSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ArtifactIteratorSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ArtifactIteratorSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ArtifactIteratorSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getItems();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.serializeBinaryToWriter
    );
  }
  f = message.getItemInput();
  if (f.length > 0) {
    writer.writeString(
      2,
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
proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    inputArtifact: jspb.Message.getFieldWithDefault(msg, 1, "")
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
 * @return {!proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec}
 */
proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec;
  return proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec}
 */
proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setInputArtifact(value);
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
proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getInputArtifact();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string input_artifact = 1;
 * @return {string}
 */
proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.prototype.getInputArtifact = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec} returns this
 */
proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.prototype.setInputArtifact = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional ItemsSpec items = 1;
 * @return {?proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec}
 */
proto.ml_pipelines.ArtifactIteratorSpec.prototype.getItems = function() {
  return /** @type{?proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec, 1));
};


/**
 * @param {?proto.ml_pipelines.ArtifactIteratorSpec.ItemsSpec|undefined} value
 * @return {!proto.ml_pipelines.ArtifactIteratorSpec} returns this
*/
proto.ml_pipelines.ArtifactIteratorSpec.prototype.setItems = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ArtifactIteratorSpec} returns this
 */
proto.ml_pipelines.ArtifactIteratorSpec.prototype.clearItems = function() {
  return this.setItems(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ArtifactIteratorSpec.prototype.hasItems = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string item_input = 2;
 * @return {string}
 */
proto.ml_pipelines.ArtifactIteratorSpec.prototype.getItemInput = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ArtifactIteratorSpec} returns this
 */
proto.ml_pipelines.ArtifactIteratorSpec.prototype.setItemInput = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
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
proto.ml_pipelines.ParameterIteratorSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ParameterIteratorSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ParameterIteratorSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ParameterIteratorSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    items: (f = msg.getItems()) && proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.toObject(includeInstance, f),
    itemInput: jspb.Message.getFieldWithDefault(msg, 2, "")
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
 * @return {!proto.ml_pipelines.ParameterIteratorSpec}
 */
proto.ml_pipelines.ParameterIteratorSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ParameterIteratorSpec;
  return proto.ml_pipelines.ParameterIteratorSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ParameterIteratorSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ParameterIteratorSpec}
 */
proto.ml_pipelines.ParameterIteratorSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec;
      reader.readMessage(value,proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.deserializeBinaryFromReader);
      msg.setItems(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setItemInput(value);
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
proto.ml_pipelines.ParameterIteratorSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ParameterIteratorSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ParameterIteratorSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ParameterIteratorSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getItems();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.serializeBinaryToWriter
    );
  }
  f = message.getItemInput();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
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
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.KindCase = {
  KIND_NOT_SET: 0,
  RAW: 1,
  INPUT_PARAMETER: 2
};

/**
 * @return {proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.KindCase}
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.getKindCase = function() {
  return /** @type {proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.KindCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.oneofGroups_[0]));
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
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    raw: jspb.Message.getFieldWithDefault(msg, 1, ""),
    inputParameter: jspb.Message.getFieldWithDefault(msg, 2, "")
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
 * @return {!proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec}
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec;
  return proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec}
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setRaw(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setInputParameter(value);
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
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.serializeBinaryToWriter = function(message, writer) {
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
};


/**
 * optional string raw = 1;
 * @return {string}
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.getRaw = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec} returns this
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.setRaw = function(value) {
  return jspb.Message.setOneofField(this, 1, proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec} returns this
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.clearRaw = function() {
  return jspb.Message.setOneofField(this, 1, proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.hasRaw = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string input_parameter = 2;
 * @return {string}
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.getInputParameter = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec} returns this
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.setInputParameter = function(value) {
  return jspb.Message.setOneofField(this, 2, proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec} returns this
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.clearInputParameter = function() {
  return jspb.Message.setOneofField(this, 2, proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec.prototype.hasInputParameter = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional ItemsSpec items = 1;
 * @return {?proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec}
 */
proto.ml_pipelines.ParameterIteratorSpec.prototype.getItems = function() {
  return /** @type{?proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec, 1));
};


/**
 * @param {?proto.ml_pipelines.ParameterIteratorSpec.ItemsSpec|undefined} value
 * @return {!proto.ml_pipelines.ParameterIteratorSpec} returns this
*/
proto.ml_pipelines.ParameterIteratorSpec.prototype.setItems = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ParameterIteratorSpec} returns this
 */
proto.ml_pipelines.ParameterIteratorSpec.prototype.clearItems = function() {
  return this.setItems(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ParameterIteratorSpec.prototype.hasItems = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string item_input = 2;
 * @return {string}
 */
proto.ml_pipelines.ParameterIteratorSpec.prototype.getItemInput = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ParameterIteratorSpec} returns this
 */
proto.ml_pipelines.ParameterIteratorSpec.prototype.setItemInput = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
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
proto.ml_pipelines.ComponentRef.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ComponentRef.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ComponentRef} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentRef.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, "")
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
 * @return {!proto.ml_pipelines.ComponentRef}
 */
proto.ml_pipelines.ComponentRef.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ComponentRef;
  return proto.ml_pipelines.ComponentRef.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ComponentRef} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ComponentRef}
 */
proto.ml_pipelines.ComponentRef.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
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
proto.ml_pipelines.ComponentRef.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ComponentRef.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ComponentRef} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ComponentRef.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.ml_pipelines.ComponentRef.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ComponentRef} returns this
 */
proto.ml_pipelines.ComponentRef.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
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
proto.ml_pipelines.PipelineInfo.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineInfo.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineInfo} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineInfo.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, "")
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
 * @return {!proto.ml_pipelines.PipelineInfo}
 */
proto.ml_pipelines.PipelineInfo.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineInfo;
  return proto.ml_pipelines.PipelineInfo.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineInfo} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineInfo}
 */
proto.ml_pipelines.PipelineInfo.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
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
proto.ml_pipelines.PipelineInfo.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineInfo.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineInfo} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineInfo.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.ml_pipelines.PipelineInfo.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineInfo} returns this
 */
proto.ml_pipelines.PipelineInfo.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_pipelines.ArtifactTypeSchema.oneofGroups_ = [[1,2,3]];

/**
 * @enum {number}
 */
proto.ml_pipelines.ArtifactTypeSchema.KindCase = {
  KIND_NOT_SET: 0,
  SCHEMA_TITLE: 1,
  SCHEMA_URI: 2,
  INSTANCE_SCHEMA: 3
};

/**
 * @return {proto.ml_pipelines.ArtifactTypeSchema.KindCase}
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.getKindCase = function() {
  return /** @type {proto.ml_pipelines.ArtifactTypeSchema.KindCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.ArtifactTypeSchema.oneofGroups_[0]));
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
proto.ml_pipelines.ArtifactTypeSchema.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ArtifactTypeSchema.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ArtifactTypeSchema} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ArtifactTypeSchema.toObject = function(includeInstance, msg) {
  var f, obj = {
    schemaTitle: jspb.Message.getFieldWithDefault(msg, 1, ""),
    schemaUri: jspb.Message.getFieldWithDefault(msg, 2, ""),
    instanceSchema: jspb.Message.getFieldWithDefault(msg, 3, ""),
    schemaVersion: jspb.Message.getFieldWithDefault(msg, 4, "")
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
 * @return {!proto.ml_pipelines.ArtifactTypeSchema}
 */
proto.ml_pipelines.ArtifactTypeSchema.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ArtifactTypeSchema;
  return proto.ml_pipelines.ArtifactTypeSchema.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ArtifactTypeSchema} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ArtifactTypeSchema}
 */
proto.ml_pipelines.ArtifactTypeSchema.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setSchemaTitle(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setSchemaUri(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setInstanceSchema(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setSchemaVersion(value);
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
proto.ml_pipelines.ArtifactTypeSchema.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ArtifactTypeSchema.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ArtifactTypeSchema} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ArtifactTypeSchema.serializeBinaryToWriter = function(message, writer) {
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
  f = message.getSchemaVersion();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
};


/**
 * optional string schema_title = 1;
 * @return {string}
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.getSchemaTitle = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ArtifactTypeSchema} returns this
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.setSchemaTitle = function(value) {
  return jspb.Message.setOneofField(this, 1, proto.ml_pipelines.ArtifactTypeSchema.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.ArtifactTypeSchema} returns this
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.clearSchemaTitle = function() {
  return jspb.Message.setOneofField(this, 1, proto.ml_pipelines.ArtifactTypeSchema.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.hasSchemaTitle = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string schema_uri = 2;
 * @return {string}
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.getSchemaUri = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ArtifactTypeSchema} returns this
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.setSchemaUri = function(value) {
  return jspb.Message.setOneofField(this, 2, proto.ml_pipelines.ArtifactTypeSchema.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.ArtifactTypeSchema} returns this
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.clearSchemaUri = function() {
  return jspb.Message.setOneofField(this, 2, proto.ml_pipelines.ArtifactTypeSchema.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.hasSchemaUri = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string instance_schema = 3;
 * @return {string}
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.getInstanceSchema = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ArtifactTypeSchema} returns this
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.setInstanceSchema = function(value) {
  return jspb.Message.setOneofField(this, 3, proto.ml_pipelines.ArtifactTypeSchema.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.ArtifactTypeSchema} returns this
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.clearInstanceSchema = function() {
  return jspb.Message.setOneofField(this, 3, proto.ml_pipelines.ArtifactTypeSchema.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.hasInstanceSchema = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string schema_version = 4;
 * @return {string}
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.getSchemaVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ArtifactTypeSchema} returns this
 */
proto.ml_pipelines.ArtifactTypeSchema.prototype.setSchemaVersion = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
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
proto.ml_pipelines.PipelineTaskInfo.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineTaskInfo.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineTaskInfo} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskInfo.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, "")
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
 * @return {!proto.ml_pipelines.PipelineTaskInfo}
 */
proto.ml_pipelines.PipelineTaskInfo.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineTaskInfo;
  return proto.ml_pipelines.PipelineTaskInfo.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineTaskInfo} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineTaskInfo}
 */
proto.ml_pipelines.PipelineTaskInfo.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
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
proto.ml_pipelines.PipelineTaskInfo.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineTaskInfo.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineTaskInfo} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskInfo.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.ml_pipelines.PipelineTaskInfo.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineTaskInfo} returns this
 */
proto.ml_pipelines.PipelineTaskInfo.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_pipelines.ValueOrRuntimeParameter.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.ml_pipelines.ValueOrRuntimeParameter.ValueCase = {
  VALUE_NOT_SET: 0,
  CONSTANT_VALUE: 1,
  RUNTIME_PARAMETER: 2
};

/**
 * @return {proto.ml_pipelines.ValueOrRuntimeParameter.ValueCase}
 */
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.getValueCase = function() {
  return /** @type {proto.ml_pipelines.ValueOrRuntimeParameter.ValueCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.ValueOrRuntimeParameter.oneofGroups_[0]));
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
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ValueOrRuntimeParameter.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ValueOrRuntimeParameter} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ValueOrRuntimeParameter.toObject = function(includeInstance, msg) {
  var f, obj = {
    constantValue: (f = msg.getConstantValue()) && proto.ml_pipelines.Value.toObject(includeInstance, f),
    runtimeParameter: jspb.Message.getFieldWithDefault(msg, 2, "")
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
 * @return {!proto.ml_pipelines.ValueOrRuntimeParameter}
 */
proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ValueOrRuntimeParameter;
  return proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ValueOrRuntimeParameter} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ValueOrRuntimeParameter}
 */
proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.Value;
      reader.readMessage(value,proto.ml_pipelines.Value.deserializeBinaryFromReader);
      msg.setConstantValue(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setRuntimeParameter(value);
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
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ValueOrRuntimeParameter} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getConstantValue();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.Value.serializeBinaryToWriter
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
 * optional Value constant_value = 1;
 * @return {?proto.ml_pipelines.Value}
 */
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.getConstantValue = function() {
  return /** @type{?proto.ml_pipelines.Value} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.Value, 1));
};


/**
 * @param {?proto.ml_pipelines.Value|undefined} value
 * @return {!proto.ml_pipelines.ValueOrRuntimeParameter} returns this
*/
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.setConstantValue = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.ml_pipelines.ValueOrRuntimeParameter.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ValueOrRuntimeParameter} returns this
 */
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.clearConstantValue = function() {
  return this.setConstantValue(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.hasConstantValue = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string runtime_parameter = 2;
 * @return {string}
 */
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.getRuntimeParameter = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ValueOrRuntimeParameter} returns this
 */
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.setRuntimeParameter = function(value) {
  return jspb.Message.setOneofField(this, 2, proto.ml_pipelines.ValueOrRuntimeParameter.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.ValueOrRuntimeParameter} returns this
 */
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.clearRuntimeParameter = function() {
  return jspb.Message.setOneofField(this, 2, proto.ml_pipelines.ValueOrRuntimeParameter.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ValueOrRuntimeParameter.prototype.hasRuntimeParameter = function() {
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
proto.ml_pipelines.PipelineDeploymentConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    executorsMap: (f = msg.getExecutorsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.toObject) : []
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig}
 */
proto.ml_pipelines.PipelineDeploymentConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig;
  return proto.ml_pipelines.PipelineDeploymentConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig}
 */
proto.ml_pipelines.PipelineDeploymentConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getExecutorsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec());
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
proto.ml_pipelines.PipelineDeploymentConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getExecutorsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.serializeBinaryToWriter);
  }
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.repeatedFields_ = [2,3];



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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    image: jspb.Message.getFieldWithDefault(msg, 1, ""),
    commandList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f,
    argsList: (f = jspb.Message.getRepeatedField(msg, 3)) == null ? undefined : f,
    lifecycle: (f = msg.getLifecycle()) && proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.toObject(includeInstance, f),
    resources: (f = msg.getResources()) && proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec;
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setImage(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.addCommand(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.addArgs(value);
      break;
    case 4:
      var value = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle;
      reader.readMessage(value,proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.deserializeBinaryFromReader);
      msg.setLifecycle(value);
      break;
    case 5:
      var value = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec;
      reader.readMessage(value,proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.deserializeBinaryFromReader);
      msg.setResources(value);
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getImage();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getCommandList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      2,
      f
    );
  }
  f = message.getArgsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      3,
      f
    );
  }
  f = message.getLifecycle();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.serializeBinaryToWriter
    );
  }
  f = message.getResources();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.serializeBinaryToWriter
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.toObject = function(includeInstance, msg) {
  var f, obj = {
    preCacheCheck: (f = msg.getPreCacheCheck()) && proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle;
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec;
      reader.readMessage(value,proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.deserializeBinaryFromReader);
      msg.setPreCacheCheck(value);
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPreCacheCheck();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.serializeBinaryToWriter
    );
  }
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.repeatedFields_ = [2,3];



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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.toObject = function(includeInstance, msg) {
  var f, obj = {
    commandList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f,
    argsList: (f = jspb.Message.getRepeatedField(msg, 3)) == null ? undefined : f
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec;
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.addCommand(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.addArgs(value);
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCommandList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      2,
      f
    );
  }
  f = message.getArgsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      3,
      f
    );
  }
};


/**
 * repeated string command = 2;
 * @return {!Array<string>}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.getCommandList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.setCommandList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.addCommand = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.clearCommandList = function() {
  return this.setCommandList([]);
};


/**
 * repeated string args = 3;
 * @return {!Array<string>}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.getArgsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 3));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.setArgsList = function(value) {
  return jspb.Message.setField(this, 3, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.addArgs = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 3, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.prototype.clearArgsList = function() {
  return this.setArgsList([]);
};


/**
 * optional Exec pre_cache_check = 1;
 * @return {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.prototype.getPreCacheCheck = function() {
  return /** @type{?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec, 1));
};


/**
 * @param {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.prototype.setPreCacheCheck = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.prototype.clearPreCacheCheck = function() {
  return this.setPreCacheCheck(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.prototype.hasPreCacheCheck = function() {
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    cpuLimit: jspb.Message.getFloatingPointFieldWithDefault(msg, 1, 0.0),
    memoryLimit: jspb.Message.getFloatingPointFieldWithDefault(msg, 2, 0.0),
    accelerator: (f = msg.getAccelerator()) && proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec;
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readDouble());
      msg.setCpuLimit(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readDouble());
      msg.setMemoryLimit(value);
      break;
    case 3:
      var value = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig;
      reader.readMessage(value,proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.deserializeBinaryFromReader);
      msg.setAccelerator(value);
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCpuLimit();
  if (f !== 0.0) {
    writer.writeDouble(
      1,
      f
    );
  }
  f = message.getMemoryLimit();
  if (f !== 0.0) {
    writer.writeDouble(
      2,
      f
    );
  }
  f = message.getAccelerator();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.serializeBinaryToWriter
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.toObject = function(includeInstance, msg) {
  var f, obj = {
    type: jspb.Message.getFieldWithDefault(msg, 1, ""),
    count: jspb.Message.getFieldWithDefault(msg, 2, 0)
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig;
  return proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setType(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setCount(value);
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
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getType();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getCount();
  if (f !== 0) {
    writer.writeInt64(
      2,
      f
    );
  }
};


/**
 * optional string type = 1;
 * @return {string}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.prototype.getType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.prototype.setType = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional int64 count = 2;
 * @return {number}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.prototype.getCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.prototype.setCount = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional double cpu_limit = 1;
 * @return {number}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.getCpuLimit = function() {
  return /** @type {number} */ (jspb.Message.getFloatingPointFieldWithDefault(this, 1, 0.0));
};


/**
 * @param {number} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.setCpuLimit = function(value) {
  return jspb.Message.setProto3FloatField(this, 1, value);
};


/**
 * optional double memory_limit = 2;
 * @return {number}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.getMemoryLimit = function() {
  return /** @type {number} */ (jspb.Message.getFloatingPointFieldWithDefault(this, 2, 0.0));
};


/**
 * @param {number} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.setMemoryLimit = function(value) {
  return jspb.Message.setProto3FloatField(this, 2, value);
};


/**
 * optional AcceleratorConfig accelerator = 3;
 * @return {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.getAccelerator = function() {
  return /** @type{?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig, 3));
};


/**
 * @param {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.setAccelerator = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.clearAccelerator = function() {
  return this.setAccelerator(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.prototype.hasAccelerator = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string image = 1;
 * @return {string}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.getImage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.setImage = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated string command = 2;
 * @return {!Array<string>}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.getCommandList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.setCommandList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.addCommand = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.clearCommandList = function() {
  return this.setCommandList([]);
};


/**
 * repeated string args = 3;
 * @return {!Array<string>}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.getArgsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 3));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.setArgsList = function(value) {
  return jspb.Message.setField(this, 3, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.addArgs = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 3, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.clearArgsList = function() {
  return this.setArgsList([]);
};


/**
 * optional Lifecycle lifecycle = 4;
 * @return {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.getLifecycle = function() {
  return /** @type{?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle, 4));
};


/**
 * @param {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.setLifecycle = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.clearLifecycle = function() {
  return this.setLifecycle(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.hasLifecycle = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional ResourceSpec resources = 5;
 * @return {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.getResources = function() {
  return /** @type{?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec, 5));
};


/**
 * @param {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.setResources = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.clearResources = function() {
  return this.setResources(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.prototype.hasResources = function() {
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
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactUri: (f = msg.getArtifactUri()) && proto.ml_pipelines.ValueOrRuntimeParameter.toObject(includeInstance, f),
    typeSchema: (f = msg.getTypeSchema()) && proto.ml_pipelines.ArtifactTypeSchema.toObject(includeInstance, f),
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ValueOrRuntimeParameter.toObject) : [],
    customPropertiesMap: (f = msg.getCustomPropertiesMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ValueOrRuntimeParameter.toObject) : [],
    metadata: (f = msg.getMetadata()) && google_protobuf_struct_pb.Struct.toObject(includeInstance, f),
    reimport: jspb.Message.getBooleanFieldWithDefault(msg, 5, false)
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec;
  return proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.ValueOrRuntimeParameter;
      reader.readMessage(value,proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader);
      msg.setArtifactUri(value);
      break;
    case 2:
      var value = new proto.ml_pipelines.ArtifactTypeSchema;
      reader.readMessage(value,proto.ml_pipelines.ArtifactTypeSchema.deserializeBinaryFromReader);
      msg.setTypeSchema(value);
      break;
    case 3:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader, "", new proto.ml_pipelines.ValueOrRuntimeParameter());
         });
      break;
    case 4:
      var value = msg.getCustomPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ValueOrRuntimeParameter.deserializeBinaryFromReader, "", new proto.ml_pipelines.ValueOrRuntimeParameter());
         });
      break;
    case 6:
      var value = new google_protobuf_struct_pb.Struct;
      reader.readMessage(value,google_protobuf_struct_pb.Struct.deserializeBinaryFromReader);
      msg.setMetadata(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setReimport(value);
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
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactUri();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter
    );
  }
  f = message.getTypeSchema();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.ArtifactTypeSchema.serializeBinaryToWriter
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(3, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter);
  }
  f = message.getCustomPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(4, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ValueOrRuntimeParameter.serializeBinaryToWriter);
  }
  f = message.getMetadata();
  if (f != null) {
    writer.writeMessage(
      6,
      f,
      google_protobuf_struct_pb.Struct.serializeBinaryToWriter
    );
  }
  f = message.getReimport();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
};


/**
 * optional ValueOrRuntimeParameter artifact_uri = 1;
 * @return {?proto.ml_pipelines.ValueOrRuntimeParameter}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.getArtifactUri = function() {
  return /** @type{?proto.ml_pipelines.ValueOrRuntimeParameter} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ValueOrRuntimeParameter, 1));
};


/**
 * @param {?proto.ml_pipelines.ValueOrRuntimeParameter|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.setArtifactUri = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.clearArtifactUri = function() {
  return this.setArtifactUri(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.hasArtifactUri = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ArtifactTypeSchema type_schema = 2;
 * @return {?proto.ml_pipelines.ArtifactTypeSchema}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.getTypeSchema = function() {
  return /** @type{?proto.ml_pipelines.ArtifactTypeSchema} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ArtifactTypeSchema, 2));
};


/**
 * @param {?proto.ml_pipelines.ArtifactTypeSchema|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.setTypeSchema = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.clearTypeSchema = function() {
  return this.setTypeSchema(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.hasTypeSchema = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * map<string, ValueOrRuntimeParameter> properties = 3;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>} */ (
      jspb.Message.getMapField(this, 3, opt_noLazyCreate,
      proto.ml_pipelines.ValueOrRuntimeParameter));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * map<string, ValueOrRuntimeParameter> custom_properties = 4;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.getCustomPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ValueOrRuntimeParameter>} */ (
      jspb.Message.getMapField(this, 4, opt_noLazyCreate,
      proto.ml_pipelines.ValueOrRuntimeParameter));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.clearCustomPropertiesMap = function() {
  this.getCustomPropertiesMap().clear();
  return this;};


/**
 * optional google.protobuf.Struct metadata = 6;
 * @return {?proto.google.protobuf.Struct}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.getMetadata = function() {
  return /** @type{?proto.google.protobuf.Struct} */ (
    jspb.Message.getWrapperField(this, google_protobuf_struct_pb.Struct, 6));
};


/**
 * @param {?proto.google.protobuf.Struct|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.setMetadata = function(value) {
  return jspb.Message.setWrapperField(this, 6, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.clearMetadata = function() {
  return this.setMetadata(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.hasMetadata = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional bool reimport = 5;
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.getReimport = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.prototype.setReimport = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
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
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    outputArtifactQueriesMap: (f = msg.getOutputArtifactQueriesMap()) ? f.toObject(includeInstance, proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.toObject) : []
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec;
  return proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getOutputArtifactQueriesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.deserializeBinaryFromReader, "", new proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec());
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
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOutputArtifactQueriesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.serializeBinaryToWriter);
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
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    filter: jspb.Message.getFieldWithDefault(msg, 1, ""),
    limit: jspb.Message.getFieldWithDefault(msg, 2, 0)
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec;
  return proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setFilter(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setLimit(value);
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
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getFilter();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getLimit();
  if (f !== 0) {
    writer.writeInt32(
      2,
      f
    );
  }
};


/**
 * optional string filter = 1;
 * @return {string}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.prototype.getFilter = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.prototype.setFilter = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional int32 limit = 2;
 * @return {number}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.prototype.getLimit = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.prototype.setLimit = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * map<string, ArtifactQuerySpec> output_artifact_queries = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec>}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.prototype.getOutputArtifactQueriesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.prototype.clearOutputArtifactQueriesMap = function() {
  this.getOutputArtifactQueriesMap().clear();
  return this;};





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
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    customJob: (f = msg.getCustomJob()) && google_protobuf_struct_pb.Struct.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec;
  return proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new google_protobuf_struct_pb.Struct;
      reader.readMessage(value,google_protobuf_struct_pb.Struct.deserializeBinaryFromReader);
      msg.setCustomJob(value);
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
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCustomJob();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      google_protobuf_struct_pb.Struct.serializeBinaryToWriter
    );
  }
};


/**
 * optional google.protobuf.Struct custom_job = 1;
 * @return {?proto.google.protobuf.Struct}
 */
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.prototype.getCustomJob = function() {
  return /** @type{?proto.google.protobuf.Struct} */ (
    jspb.Message.getWrapperField(this, google_protobuf_struct_pb.Struct, 1));
};


/**
 * @param {?proto.google.protobuf.Struct|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.prototype.setCustomJob = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.prototype.clearCustomJob = function() {
  return this.setCustomJob(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.prototype.hasCustomJob = function() {
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
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.oneofGroups_ = [[1,2,3,4]];

/**
 * @enum {number}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.SpecCase = {
  SPEC_NOT_SET: 0,
  CONTAINER: 1,
  IMPORTER: 2,
  RESOLVER: 3,
  CUSTOM_JOB: 4
};

/**
 * @return {proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.SpecCase}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.getSpecCase = function() {
  return /** @type {proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.SpecCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.oneofGroups_[0]));
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
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.toObject = function(includeInstance, msg) {
  var f, obj = {
    container: (f = msg.getContainer()) && proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.toObject(includeInstance, f),
    importer: (f = msg.getImporter()) && proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.toObject(includeInstance, f),
    resolver: (f = msg.getResolver()) && proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.toObject(includeInstance, f),
    customJob: (f = msg.getCustomJob()) && proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec;
  return proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec;
      reader.readMessage(value,proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.deserializeBinaryFromReader);
      msg.setContainer(value);
      break;
    case 2:
      var value = new proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec;
      reader.readMessage(value,proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.deserializeBinaryFromReader);
      msg.setImporter(value);
      break;
    case 3:
      var value = new proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec;
      reader.readMessage(value,proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.deserializeBinaryFromReader);
      msg.setResolver(value);
      break;
    case 4:
      var value = new proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec;
      reader.readMessage(value,proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.deserializeBinaryFromReader);
      msg.setCustomJob(value);
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
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getContainer();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.serializeBinaryToWriter
    );
  }
  f = message.getImporter();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.serializeBinaryToWriter
    );
  }
  f = message.getResolver();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.serializeBinaryToWriter
    );
  }
  f = message.getCustomJob();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.serializeBinaryToWriter
    );
  }
};


/**
 * optional PipelineContainerSpec container = 1;
 * @return {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.getContainer = function() {
  return /** @type{?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec, 1));
};


/**
 * @param {?proto.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.setContainer = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.clearContainer = function() {
  return this.setContainer(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.hasContainer = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ImporterSpec importer = 2;
 * @return {?proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.getImporter = function() {
  return /** @type{?proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec, 2));
};


/**
 * @param {?proto.ml_pipelines.PipelineDeploymentConfig.ImporterSpec|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.setImporter = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.clearImporter = function() {
  return this.setImporter(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.hasImporter = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional ResolverSpec resolver = 3;
 * @return {?proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.getResolver = function() {
  return /** @type{?proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec, 3));
};


/**
 * @param {?proto.ml_pipelines.PipelineDeploymentConfig.ResolverSpec|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.setResolver = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.clearResolver = function() {
  return this.setResolver(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.hasResolver = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional AIPlatformCustomJobSpec custom_job = 4;
 * @return {?proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.getCustomJob = function() {
  return /** @type{?proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec, 4));
};


/**
 * @param {?proto.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec|undefined} value
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} returns this
*/
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.setCustomJob = function(value) {
  return jspb.Message.setOneofWrapperField(this, 4, proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.clearCustomJob = function() {
  return this.setCustomJob(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.prototype.hasCustomJob = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * map<string, ExecutorSpec> executors = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec>}
 */
proto.ml_pipelines.PipelineDeploymentConfig.prototype.getExecutorsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.PipelineDeploymentConfig} returns this
 */
proto.ml_pipelines.PipelineDeploymentConfig.prototype.clearExecutorsMap = function() {
  this.getExecutorsMap().clear();
  return this;};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.ml_pipelines.Value.oneofGroups_ = [[1,2,3]];

/**
 * @enum {number}
 */
proto.ml_pipelines.Value.ValueCase = {
  VALUE_NOT_SET: 0,
  INT_VALUE: 1,
  DOUBLE_VALUE: 2,
  STRING_VALUE: 3
};

/**
 * @return {proto.ml_pipelines.Value.ValueCase}
 */
proto.ml_pipelines.Value.prototype.getValueCase = function() {
  return /** @type {proto.ml_pipelines.Value.ValueCase} */(jspb.Message.computeOneofCase(this, proto.ml_pipelines.Value.oneofGroups_[0]));
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
proto.ml_pipelines.Value.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.Value.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.Value} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.Value.toObject = function(includeInstance, msg) {
  var f, obj = {
    intValue: jspb.Message.getFieldWithDefault(msg, 1, 0),
    doubleValue: jspb.Message.getFloatingPointFieldWithDefault(msg, 2, 0.0),
    stringValue: jspb.Message.getFieldWithDefault(msg, 3, "")
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
 * @return {!proto.ml_pipelines.Value}
 */
proto.ml_pipelines.Value.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.Value;
  return proto.ml_pipelines.Value.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.Value} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.Value}
 */
proto.ml_pipelines.Value.deserializeBinaryFromReader = function(msg, reader) {
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
proto.ml_pipelines.Value.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.Value.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.Value} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.Value.serializeBinaryToWriter = function(message, writer) {
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
};


/**
 * optional int64 int_value = 1;
 * @return {number}
 */
proto.ml_pipelines.Value.prototype.getIntValue = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.ml_pipelines.Value} returns this
 */
proto.ml_pipelines.Value.prototype.setIntValue = function(value) {
  return jspb.Message.setOneofField(this, 1, proto.ml_pipelines.Value.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.Value} returns this
 */
proto.ml_pipelines.Value.prototype.clearIntValue = function() {
  return jspb.Message.setOneofField(this, 1, proto.ml_pipelines.Value.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.Value.prototype.hasIntValue = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional double double_value = 2;
 * @return {number}
 */
proto.ml_pipelines.Value.prototype.getDoubleValue = function() {
  return /** @type {number} */ (jspb.Message.getFloatingPointFieldWithDefault(this, 2, 0.0));
};


/**
 * @param {number} value
 * @return {!proto.ml_pipelines.Value} returns this
 */
proto.ml_pipelines.Value.prototype.setDoubleValue = function(value) {
  return jspb.Message.setOneofField(this, 2, proto.ml_pipelines.Value.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.Value} returns this
 */
proto.ml_pipelines.Value.prototype.clearDoubleValue = function() {
  return jspb.Message.setOneofField(this, 2, proto.ml_pipelines.Value.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.Value.prototype.hasDoubleValue = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string string_value = 3;
 * @return {string}
 */
proto.ml_pipelines.Value.prototype.getStringValue = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.Value} returns this
 */
proto.ml_pipelines.Value.prototype.setStringValue = function(value) {
  return jspb.Message.setOneofField(this, 3, proto.ml_pipelines.Value.oneofGroups_[0], value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.ml_pipelines.Value} returns this
 */
proto.ml_pipelines.Value.prototype.clearStringValue = function() {
  return jspb.Message.setOneofField(this, 3, proto.ml_pipelines.Value.oneofGroups_[0], undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.Value.prototype.hasStringValue = function() {
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
proto.ml_pipelines.RuntimeArtifact.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.RuntimeArtifact.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.RuntimeArtifact} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.RuntimeArtifact.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    type: (f = msg.getType()) && proto.ml_pipelines.ArtifactTypeSchema.toObject(includeInstance, f),
    uri: jspb.Message.getFieldWithDefault(msg, 3, ""),
    propertiesMap: (f = msg.getPropertiesMap()) ? f.toObject(includeInstance, proto.ml_pipelines.Value.toObject) : [],
    customPropertiesMap: (f = msg.getCustomPropertiesMap()) ? f.toObject(includeInstance, proto.ml_pipelines.Value.toObject) : [],
    metadata: (f = msg.getMetadata()) && google_protobuf_struct_pb.Struct.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.RuntimeArtifact}
 */
proto.ml_pipelines.RuntimeArtifact.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.RuntimeArtifact;
  return proto.ml_pipelines.RuntimeArtifact.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.RuntimeArtifact} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.RuntimeArtifact}
 */
proto.ml_pipelines.RuntimeArtifact.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = new proto.ml_pipelines.ArtifactTypeSchema;
      reader.readMessage(value,proto.ml_pipelines.ArtifactTypeSchema.deserializeBinaryFromReader);
      msg.setType(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setUri(value);
      break;
    case 4:
      var value = msg.getPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.Value.deserializeBinaryFromReader, "", new proto.ml_pipelines.Value());
         });
      break;
    case 5:
      var value = msg.getCustomPropertiesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.Value.deserializeBinaryFromReader, "", new proto.ml_pipelines.Value());
         });
      break;
    case 6:
      var value = new google_protobuf_struct_pb.Struct;
      reader.readMessage(value,google_protobuf_struct_pb.Struct.deserializeBinaryFromReader);
      msg.setMetadata(value);
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
proto.ml_pipelines.RuntimeArtifact.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.RuntimeArtifact.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.RuntimeArtifact} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.RuntimeArtifact.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getType();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.ArtifactTypeSchema.serializeBinaryToWriter
    );
  }
  f = message.getUri();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(4, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.Value.serializeBinaryToWriter);
  }
  f = message.getCustomPropertiesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.Value.serializeBinaryToWriter);
  }
  f = message.getMetadata();
  if (f != null) {
    writer.writeMessage(
      6,
      f,
      google_protobuf_struct_pb.Struct.serializeBinaryToWriter
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.ml_pipelines.RuntimeArtifact.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.RuntimeArtifact} returns this
 */
proto.ml_pipelines.RuntimeArtifact.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional ArtifactTypeSchema type = 2;
 * @return {?proto.ml_pipelines.ArtifactTypeSchema}
 */
proto.ml_pipelines.RuntimeArtifact.prototype.getType = function() {
  return /** @type{?proto.ml_pipelines.ArtifactTypeSchema} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ArtifactTypeSchema, 2));
};


/**
 * @param {?proto.ml_pipelines.ArtifactTypeSchema|undefined} value
 * @return {!proto.ml_pipelines.RuntimeArtifact} returns this
*/
proto.ml_pipelines.RuntimeArtifact.prototype.setType = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.RuntimeArtifact} returns this
 */
proto.ml_pipelines.RuntimeArtifact.prototype.clearType = function() {
  return this.setType(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.RuntimeArtifact.prototype.hasType = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string uri = 3;
 * @return {string}
 */
proto.ml_pipelines.RuntimeArtifact.prototype.getUri = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.RuntimeArtifact} returns this
 */
proto.ml_pipelines.RuntimeArtifact.prototype.setUri = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * map<string, Value> properties = 4;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.Value>}
 */
proto.ml_pipelines.RuntimeArtifact.prototype.getPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.Value>} */ (
      jspb.Message.getMapField(this, 4, opt_noLazyCreate,
      proto.ml_pipelines.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.RuntimeArtifact} returns this
 */
proto.ml_pipelines.RuntimeArtifact.prototype.clearPropertiesMap = function() {
  this.getPropertiesMap().clear();
  return this;};


/**
 * map<string, Value> custom_properties = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.Value>}
 */
proto.ml_pipelines.RuntimeArtifact.prototype.getCustomPropertiesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.Value>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      proto.ml_pipelines.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.RuntimeArtifact} returns this
 */
proto.ml_pipelines.RuntimeArtifact.prototype.clearCustomPropertiesMap = function() {
  this.getCustomPropertiesMap().clear();
  return this;};


/**
 * optional google.protobuf.Struct metadata = 6;
 * @return {?proto.google.protobuf.Struct}
 */
proto.ml_pipelines.RuntimeArtifact.prototype.getMetadata = function() {
  return /** @type{?proto.google.protobuf.Struct} */ (
    jspb.Message.getWrapperField(this, google_protobuf_struct_pb.Struct, 6));
};


/**
 * @param {?proto.google.protobuf.Struct|undefined} value
 * @return {!proto.ml_pipelines.RuntimeArtifact} returns this
*/
proto.ml_pipelines.RuntimeArtifact.prototype.setMetadata = function(value) {
  return jspb.Message.setWrapperField(this, 6, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.RuntimeArtifact} returns this
 */
proto.ml_pipelines.RuntimeArtifact.prototype.clearMetadata = function() {
  return this.setMetadata(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.RuntimeArtifact.prototype.hasMetadata = function() {
  return jspb.Message.getField(this, 6) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.ml_pipelines.ArtifactList.repeatedFields_ = [1];



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
proto.ml_pipelines.ArtifactList.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ArtifactList.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ArtifactList} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ArtifactList.toObject = function(includeInstance, msg) {
  var f, obj = {
    artifactsList: jspb.Message.toObjectList(msg.getArtifactsList(),
    proto.ml_pipelines.RuntimeArtifact.toObject, includeInstance)
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
 * @return {!proto.ml_pipelines.ArtifactList}
 */
proto.ml_pipelines.ArtifactList.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ArtifactList;
  return proto.ml_pipelines.ArtifactList.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ArtifactList} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ArtifactList}
 */
proto.ml_pipelines.ArtifactList.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.RuntimeArtifact;
      reader.readMessage(value,proto.ml_pipelines.RuntimeArtifact.deserializeBinaryFromReader);
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
proto.ml_pipelines.ArtifactList.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ArtifactList.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ArtifactList} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ArtifactList.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getArtifactsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.ml_pipelines.RuntimeArtifact.serializeBinaryToWriter
    );
  }
};


/**
 * repeated RuntimeArtifact artifacts = 1;
 * @return {!Array<!proto.ml_pipelines.RuntimeArtifact>}
 */
proto.ml_pipelines.ArtifactList.prototype.getArtifactsList = function() {
  return /** @type{!Array<!proto.ml_pipelines.RuntimeArtifact>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.ml_pipelines.RuntimeArtifact, 1));
};


/**
 * @param {!Array<!proto.ml_pipelines.RuntimeArtifact>} value
 * @return {!proto.ml_pipelines.ArtifactList} returns this
*/
proto.ml_pipelines.ArtifactList.prototype.setArtifactsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.ml_pipelines.RuntimeArtifact=} opt_value
 * @param {number=} opt_index
 * @return {!proto.ml_pipelines.RuntimeArtifact}
 */
proto.ml_pipelines.ArtifactList.prototype.addArtifacts = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.ml_pipelines.RuntimeArtifact, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.ml_pipelines.ArtifactList} returns this
 */
proto.ml_pipelines.ArtifactList.prototype.clearArtifactsList = function() {
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
proto.ml_pipelines.ExecutorInput.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ExecutorInput.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ExecutorInput} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorInput.toObject = function(includeInstance, msg) {
  var f, obj = {
    inputs: (f = msg.getInputs()) && proto.ml_pipelines.ExecutorInput.Inputs.toObject(includeInstance, f),
    outputs: (f = msg.getOutputs()) && proto.ml_pipelines.ExecutorInput.Outputs.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.ExecutorInput}
 */
proto.ml_pipelines.ExecutorInput.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ExecutorInput;
  return proto.ml_pipelines.ExecutorInput.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ExecutorInput} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ExecutorInput}
 */
proto.ml_pipelines.ExecutorInput.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.ml_pipelines.ExecutorInput.Inputs;
      reader.readMessage(value,proto.ml_pipelines.ExecutorInput.Inputs.deserializeBinaryFromReader);
      msg.setInputs(value);
      break;
    case 2:
      var value = new proto.ml_pipelines.ExecutorInput.Outputs;
      reader.readMessage(value,proto.ml_pipelines.ExecutorInput.Outputs.deserializeBinaryFromReader);
      msg.setOutputs(value);
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
proto.ml_pipelines.ExecutorInput.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ExecutorInput.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ExecutorInput} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorInput.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getInputs();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.ml_pipelines.ExecutorInput.Inputs.serializeBinaryToWriter
    );
  }
  f = message.getOutputs();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.ml_pipelines.ExecutorInput.Outputs.serializeBinaryToWriter
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
proto.ml_pipelines.ExecutorInput.Inputs.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ExecutorInput.Inputs.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ExecutorInput.Inputs} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorInput.Inputs.toObject = function(includeInstance, msg) {
  var f, obj = {
    parametersMap: (f = msg.getParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.Value.toObject) : [],
    artifactsMap: (f = msg.getArtifactsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ArtifactList.toObject) : []
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
 * @return {!proto.ml_pipelines.ExecutorInput.Inputs}
 */
proto.ml_pipelines.ExecutorInput.Inputs.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ExecutorInput.Inputs;
  return proto.ml_pipelines.ExecutorInput.Inputs.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ExecutorInput.Inputs} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ExecutorInput.Inputs}
 */
proto.ml_pipelines.ExecutorInput.Inputs.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.Value.deserializeBinaryFromReader, "", new proto.ml_pipelines.Value());
         });
      break;
    case 2:
      var value = msg.getArtifactsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ArtifactList.deserializeBinaryFromReader, "", new proto.ml_pipelines.ArtifactList());
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
proto.ml_pipelines.ExecutorInput.Inputs.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ExecutorInput.Inputs.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ExecutorInput.Inputs} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorInput.Inputs.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.Value.serializeBinaryToWriter);
  }
  f = message.getArtifactsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ArtifactList.serializeBinaryToWriter);
  }
};


/**
 * map<string, Value> parameters = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.Value>}
 */
proto.ml_pipelines.ExecutorInput.Inputs.prototype.getParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.Value>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ExecutorInput.Inputs} returns this
 */
proto.ml_pipelines.ExecutorInput.Inputs.prototype.clearParametersMap = function() {
  this.getParametersMap().clear();
  return this;};


/**
 * map<string, ArtifactList> artifacts = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ArtifactList>}
 */
proto.ml_pipelines.ExecutorInput.Inputs.prototype.getArtifactsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ArtifactList>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.ArtifactList));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ExecutorInput.Inputs} returns this
 */
proto.ml_pipelines.ExecutorInput.Inputs.prototype.clearArtifactsMap = function() {
  this.getArtifactsMap().clear();
  return this;};





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
proto.ml_pipelines.ExecutorInput.OutputParameter.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ExecutorInput.OutputParameter.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ExecutorInput.OutputParameter} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorInput.OutputParameter.toObject = function(includeInstance, msg) {
  var f, obj = {
    outputFile: jspb.Message.getFieldWithDefault(msg, 1, "")
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
 * @return {!proto.ml_pipelines.ExecutorInput.OutputParameter}
 */
proto.ml_pipelines.ExecutorInput.OutputParameter.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ExecutorInput.OutputParameter;
  return proto.ml_pipelines.ExecutorInput.OutputParameter.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ExecutorInput.OutputParameter} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ExecutorInput.OutputParameter}
 */
proto.ml_pipelines.ExecutorInput.OutputParameter.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setOutputFile(value);
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
proto.ml_pipelines.ExecutorInput.OutputParameter.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ExecutorInput.OutputParameter.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ExecutorInput.OutputParameter} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorInput.OutputParameter.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOutputFile();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string output_file = 1;
 * @return {string}
 */
proto.ml_pipelines.ExecutorInput.OutputParameter.prototype.getOutputFile = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ExecutorInput.OutputParameter} returns this
 */
proto.ml_pipelines.ExecutorInput.OutputParameter.prototype.setOutputFile = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
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
proto.ml_pipelines.ExecutorInput.Outputs.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ExecutorInput.Outputs.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ExecutorInput.Outputs} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorInput.Outputs.toObject = function(includeInstance, msg) {
  var f, obj = {
    parametersMap: (f = msg.getParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ExecutorInput.OutputParameter.toObject) : [],
    artifactsMap: (f = msg.getArtifactsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ArtifactList.toObject) : [],
    outputFile: jspb.Message.getFieldWithDefault(msg, 3, "")
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
 * @return {!proto.ml_pipelines.ExecutorInput.Outputs}
 */
proto.ml_pipelines.ExecutorInput.Outputs.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ExecutorInput.Outputs;
  return proto.ml_pipelines.ExecutorInput.Outputs.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ExecutorInput.Outputs} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ExecutorInput.Outputs}
 */
proto.ml_pipelines.ExecutorInput.Outputs.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ExecutorInput.OutputParameter.deserializeBinaryFromReader, "", new proto.ml_pipelines.ExecutorInput.OutputParameter());
         });
      break;
    case 2:
      var value = msg.getArtifactsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ArtifactList.deserializeBinaryFromReader, "", new proto.ml_pipelines.ArtifactList());
         });
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setOutputFile(value);
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
proto.ml_pipelines.ExecutorInput.Outputs.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ExecutorInput.Outputs.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ExecutorInput.Outputs} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorInput.Outputs.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ExecutorInput.OutputParameter.serializeBinaryToWriter);
  }
  f = message.getArtifactsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ArtifactList.serializeBinaryToWriter);
  }
  f = message.getOutputFile();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
};


/**
 * map<string, OutputParameter> parameters = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ExecutorInput.OutputParameter>}
 */
proto.ml_pipelines.ExecutorInput.Outputs.prototype.getParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ExecutorInput.OutputParameter>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.ExecutorInput.OutputParameter));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ExecutorInput.Outputs} returns this
 */
proto.ml_pipelines.ExecutorInput.Outputs.prototype.clearParametersMap = function() {
  this.getParametersMap().clear();
  return this;};


/**
 * map<string, ArtifactList> artifacts = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ArtifactList>}
 */
proto.ml_pipelines.ExecutorInput.Outputs.prototype.getArtifactsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ArtifactList>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.ArtifactList));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ExecutorInput.Outputs} returns this
 */
proto.ml_pipelines.ExecutorInput.Outputs.prototype.clearArtifactsMap = function() {
  this.getArtifactsMap().clear();
  return this;};


/**
 * optional string output_file = 3;
 * @return {string}
 */
proto.ml_pipelines.ExecutorInput.Outputs.prototype.getOutputFile = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.ExecutorInput.Outputs} returns this
 */
proto.ml_pipelines.ExecutorInput.Outputs.prototype.setOutputFile = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional Inputs inputs = 1;
 * @return {?proto.ml_pipelines.ExecutorInput.Inputs}
 */
proto.ml_pipelines.ExecutorInput.prototype.getInputs = function() {
  return /** @type{?proto.ml_pipelines.ExecutorInput.Inputs} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ExecutorInput.Inputs, 1));
};


/**
 * @param {?proto.ml_pipelines.ExecutorInput.Inputs|undefined} value
 * @return {!proto.ml_pipelines.ExecutorInput} returns this
*/
proto.ml_pipelines.ExecutorInput.prototype.setInputs = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ExecutorInput} returns this
 */
proto.ml_pipelines.ExecutorInput.prototype.clearInputs = function() {
  return this.setInputs(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ExecutorInput.prototype.hasInputs = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional Outputs outputs = 2;
 * @return {?proto.ml_pipelines.ExecutorInput.Outputs}
 */
proto.ml_pipelines.ExecutorInput.prototype.getOutputs = function() {
  return /** @type{?proto.ml_pipelines.ExecutorInput.Outputs} */ (
    jspb.Message.getWrapperField(this, proto.ml_pipelines.ExecutorInput.Outputs, 2));
};


/**
 * @param {?proto.ml_pipelines.ExecutorInput.Outputs|undefined} value
 * @return {!proto.ml_pipelines.ExecutorInput} returns this
*/
proto.ml_pipelines.ExecutorInput.prototype.setOutputs = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.ExecutorInput} returns this
 */
proto.ml_pipelines.ExecutorInput.prototype.clearOutputs = function() {
  return this.setOutputs(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.ExecutorInput.prototype.hasOutputs = function() {
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
proto.ml_pipelines.ExecutorOutput.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.ExecutorOutput.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.ExecutorOutput} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorOutput.toObject = function(includeInstance, msg) {
  var f, obj = {
    parametersMap: (f = msg.getParametersMap()) ? f.toObject(includeInstance, proto.ml_pipelines.Value.toObject) : [],
    artifactsMap: (f = msg.getArtifactsMap()) ? f.toObject(includeInstance, proto.ml_pipelines.ArtifactList.toObject) : []
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
 * @return {!proto.ml_pipelines.ExecutorOutput}
 */
proto.ml_pipelines.ExecutorOutput.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.ExecutorOutput;
  return proto.ml_pipelines.ExecutorOutput.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.ExecutorOutput} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.ExecutorOutput}
 */
proto.ml_pipelines.ExecutorOutput.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getParametersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.Value.deserializeBinaryFromReader, "", new proto.ml_pipelines.Value());
         });
      break;
    case 2:
      var value = msg.getArtifactsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.ml_pipelines.ArtifactList.deserializeBinaryFromReader, "", new proto.ml_pipelines.ArtifactList());
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
proto.ml_pipelines.ExecutorOutput.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.ExecutorOutput.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.ExecutorOutput} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.ExecutorOutput.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getParametersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.Value.serializeBinaryToWriter);
  }
  f = message.getArtifactsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.ml_pipelines.ArtifactList.serializeBinaryToWriter);
  }
};


/**
 * map<string, Value> parameters = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.Value>}
 */
proto.ml_pipelines.ExecutorOutput.prototype.getParametersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.Value>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.ml_pipelines.Value));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ExecutorOutput} returns this
 */
proto.ml_pipelines.ExecutorOutput.prototype.clearParametersMap = function() {
  this.getParametersMap().clear();
  return this;};


/**
 * map<string, ArtifactList> artifacts = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.ml_pipelines.ArtifactList>}
 */
proto.ml_pipelines.ExecutorOutput.prototype.getArtifactsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.ml_pipelines.ArtifactList>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      proto.ml_pipelines.ArtifactList));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.ml_pipelines.ExecutorOutput} returns this
 */
proto.ml_pipelines.ExecutorOutput.prototype.clearArtifactsMap = function() {
  this.getArtifactsMap().clear();
  return this;};





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
proto.ml_pipelines.PipelineTaskFinalStatus.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineTaskFinalStatus.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineTaskFinalStatus} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskFinalStatus.toObject = function(includeInstance, msg) {
  var f, obj = {
    state: jspb.Message.getFieldWithDefault(msg, 1, ""),
    error: (f = msg.getError()) && google_rpc_status_pb.Status.toObject(includeInstance, f)
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
 * @return {!proto.ml_pipelines.PipelineTaskFinalStatus}
 */
proto.ml_pipelines.PipelineTaskFinalStatus.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineTaskFinalStatus;
  return proto.ml_pipelines.PipelineTaskFinalStatus.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineTaskFinalStatus} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineTaskFinalStatus}
 */
proto.ml_pipelines.PipelineTaskFinalStatus.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setState(value);
      break;
    case 2:
      var value = new google_rpc_status_pb.Status;
      reader.readMessage(value,google_rpc_status_pb.Status.deserializeBinaryFromReader);
      msg.setError(value);
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
proto.ml_pipelines.PipelineTaskFinalStatus.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineTaskFinalStatus.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineTaskFinalStatus} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineTaskFinalStatus.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getState();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getError();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_rpc_status_pb.Status.serializeBinaryToWriter
    );
  }
};


/**
 * optional string state = 1;
 * @return {string}
 */
proto.ml_pipelines.PipelineTaskFinalStatus.prototype.getState = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.ml_pipelines.PipelineTaskFinalStatus} returns this
 */
proto.ml_pipelines.PipelineTaskFinalStatus.prototype.setState = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional google.rpc.Status error = 2;
 * @return {?proto.google.rpc.Status}
 */
proto.ml_pipelines.PipelineTaskFinalStatus.prototype.getError = function() {
  return /** @type{?proto.google.rpc.Status} */ (
    jspb.Message.getWrapperField(this, google_rpc_status_pb.Status, 2));
};


/**
 * @param {?proto.google.rpc.Status|undefined} value
 * @return {!proto.ml_pipelines.PipelineTaskFinalStatus} returns this
*/
proto.ml_pipelines.PipelineTaskFinalStatus.prototype.setError = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.ml_pipelines.PipelineTaskFinalStatus} returns this
 */
proto.ml_pipelines.PipelineTaskFinalStatus.prototype.clearError = function() {
  return this.setError(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.ml_pipelines.PipelineTaskFinalStatus.prototype.hasError = function() {
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
proto.ml_pipelines.PipelineStateEnum.prototype.toObject = function(opt_includeInstance) {
  return proto.ml_pipelines.PipelineStateEnum.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.ml_pipelines.PipelineStateEnum} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineStateEnum.toObject = function(includeInstance, msg) {
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
 * @return {!proto.ml_pipelines.PipelineStateEnum}
 */
proto.ml_pipelines.PipelineStateEnum.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.ml_pipelines.PipelineStateEnum;
  return proto.ml_pipelines.PipelineStateEnum.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.ml_pipelines.PipelineStateEnum} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.ml_pipelines.PipelineStateEnum}
 */
proto.ml_pipelines.PipelineStateEnum.deserializeBinaryFromReader = function(msg, reader) {
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
proto.ml_pipelines.PipelineStateEnum.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.ml_pipelines.PipelineStateEnum.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.ml_pipelines.PipelineStateEnum} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.ml_pipelines.PipelineStateEnum.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};


/**
 * @enum {number}
 */
proto.ml_pipelines.PipelineStateEnum.PipelineTaskState = {
  TASK_STATE_UNSPECIFIED: 0,
  PENDING: 1,
  RUNNING_DRIVER: 2,
  DRIVER_SUCCEEDED: 3,
  RUNNING_EXECUTOR: 4,
  SUCCEEDED: 5,
  CANCEL_PENDING: 6,
  CANCELLING: 7,
  CANCELLED: 8,
  FAILED: 9,
  SKIPPED: 10,
  QUEUED: 11,
  NOT_TRIGGERED: 12,
  UNSCHEDULABLE: 13
};

goog.object.extend(exports, proto.ml_pipelines);
