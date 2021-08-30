/*eslint-disable block-scoped-var, id-length, no-control-regex, no-magic-numbers, no-prototype-builtins, no-redeclare, no-shadow, no-var, sort-vars*/
"use strict";

var $protobuf = require("protobufjs/minimal");

// Common aliases
var $Reader = $protobuf.Reader, $Writer = $protobuf.Writer, $util = $protobuf.util;

// Exported root namespace
var $root = $protobuf.roots["default"] || ($protobuf.roots["default"] = {});

$root.ml_pipelines = (function() {

    /**
     * Namespace ml_pipelines.
     * @exports ml_pipelines
     * @namespace
     */
    var ml_pipelines = {};

    ml_pipelines.PipelineJob = (function() {

        /**
         * Properties of a PipelineJob.
         * @memberof ml_pipelines
         * @interface IPipelineJob
         * @property {string|null} [name] PipelineJob name
         * @property {string|null} [displayName] PipelineJob displayName
         * @property {google.protobuf.IStruct|null} [pipelineSpec] PipelineJob pipelineSpec
         * @property {Object.<string,string>|null} [labels] PipelineJob labels
         * @property {ml_pipelines.PipelineJob.IRuntimeConfig|null} [runtimeConfig] PipelineJob runtimeConfig
         */

        /**
         * Constructs a new PipelineJob.
         * @memberof ml_pipelines
         * @classdesc Represents a PipelineJob.
         * @implements IPipelineJob
         * @constructor
         * @param {ml_pipelines.IPipelineJob=} [properties] Properties to set
         */
        function PipelineJob(properties) {
            this.labels = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * PipelineJob name.
         * @member {string} name
         * @memberof ml_pipelines.PipelineJob
         * @instance
         */
        PipelineJob.prototype.name = "";

        /**
         * PipelineJob displayName.
         * @member {string} displayName
         * @memberof ml_pipelines.PipelineJob
         * @instance
         */
        PipelineJob.prototype.displayName = "";

        /**
         * PipelineJob pipelineSpec.
         * @member {google.protobuf.IStruct|null|undefined} pipelineSpec
         * @memberof ml_pipelines.PipelineJob
         * @instance
         */
        PipelineJob.prototype.pipelineSpec = null;

        /**
         * PipelineJob labels.
         * @member {Object.<string,string>} labels
         * @memberof ml_pipelines.PipelineJob
         * @instance
         */
        PipelineJob.prototype.labels = $util.emptyObject;

        /**
         * PipelineJob runtimeConfig.
         * @member {ml_pipelines.PipelineJob.IRuntimeConfig|null|undefined} runtimeConfig
         * @memberof ml_pipelines.PipelineJob
         * @instance
         */
        PipelineJob.prototype.runtimeConfig = null;

        /**
         * Creates a new PipelineJob instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.PipelineJob
         * @static
         * @param {ml_pipelines.IPipelineJob=} [properties] Properties to set
         * @returns {ml_pipelines.PipelineJob} PipelineJob instance
         */
        PipelineJob.create = function create(properties) {
            return new PipelineJob(properties);
        };

        /**
         * Encodes the specified PipelineJob message. Does not implicitly {@link ml_pipelines.PipelineJob.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.PipelineJob
         * @static
         * @param {ml_pipelines.IPipelineJob} message PipelineJob message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineJob.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
            if (message.displayName != null && Object.hasOwnProperty.call(message, "displayName"))
                writer.uint32(/* id 2, wireType 2 =*/18).string(message.displayName);
            if (message.pipelineSpec != null && Object.hasOwnProperty.call(message, "pipelineSpec"))
                $root.google.protobuf.Struct.encode(message.pipelineSpec, writer.uint32(/* id 7, wireType 2 =*/58).fork()).ldelim();
            if (message.labels != null && Object.hasOwnProperty.call(message, "labels"))
                for (var keys = Object.keys(message.labels), i = 0; i < keys.length; ++i)
                    writer.uint32(/* id 11, wireType 2 =*/90).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]).uint32(/* id 2, wireType 2 =*/18).string(message.labels[keys[i]]).ldelim();
            if (message.runtimeConfig != null && Object.hasOwnProperty.call(message, "runtimeConfig"))
                $root.ml_pipelines.PipelineJob.RuntimeConfig.encode(message.runtimeConfig, writer.uint32(/* id 12, wireType 2 =*/98).fork()).ldelim();
            return writer;
        };

        /**
         * Encodes the specified PipelineJob message, length delimited. Does not implicitly {@link ml_pipelines.PipelineJob.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.PipelineJob
         * @static
         * @param {ml_pipelines.IPipelineJob} message PipelineJob message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineJob.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a PipelineJob message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.PipelineJob
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.PipelineJob} PipelineJob
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineJob.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineJob(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 2:
                    message.displayName = reader.string();
                    break;
                case 7:
                    message.pipelineSpec = $root.google.protobuf.Struct.decode(reader, reader.uint32());
                    break;
                case 11:
                    if (message.labels === $util.emptyObject)
                        message.labels = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = "";
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = reader.string();
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.labels[key] = value;
                    break;
                case 12:
                    message.runtimeConfig = $root.ml_pipelines.PipelineJob.RuntimeConfig.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a PipelineJob message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.PipelineJob
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.PipelineJob} PipelineJob
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineJob.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a PipelineJob message.
         * @function verify
         * @memberof ml_pipelines.PipelineJob
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        PipelineJob.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.name != null && message.hasOwnProperty("name"))
                if (!$util.isString(message.name))
                    return "name: string expected";
            if (message.displayName != null && message.hasOwnProperty("displayName"))
                if (!$util.isString(message.displayName))
                    return "displayName: string expected";
            if (message.pipelineSpec != null && message.hasOwnProperty("pipelineSpec")) {
                var error = $root.google.protobuf.Struct.verify(message.pipelineSpec);
                if (error)
                    return "pipelineSpec." + error;
            }
            if (message.labels != null && message.hasOwnProperty("labels")) {
                if (!$util.isObject(message.labels))
                    return "labels: object expected";
                var key = Object.keys(message.labels);
                for (var i = 0; i < key.length; ++i)
                    if (!$util.isString(message.labels[key[i]]))
                        return "labels: string{k:string} expected";
            }
            if (message.runtimeConfig != null && message.hasOwnProperty("runtimeConfig")) {
                var error = $root.ml_pipelines.PipelineJob.RuntimeConfig.verify(message.runtimeConfig);
                if (error)
                    return "runtimeConfig." + error;
            }
            return null;
        };

        /**
         * Creates a PipelineJob message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.PipelineJob
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.PipelineJob} PipelineJob
         */
        PipelineJob.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.PipelineJob)
                return object;
            var message = new $root.ml_pipelines.PipelineJob();
            if (object.name != null)
                message.name = String(object.name);
            if (object.displayName != null)
                message.displayName = String(object.displayName);
            if (object.pipelineSpec != null) {
                if (typeof object.pipelineSpec !== "object")
                    throw TypeError(".ml_pipelines.PipelineJob.pipelineSpec: object expected");
                message.pipelineSpec = $root.google.protobuf.Struct.fromObject(object.pipelineSpec);
            }
            if (object.labels) {
                if (typeof object.labels !== "object")
                    throw TypeError(".ml_pipelines.PipelineJob.labels: object expected");
                message.labels = {};
                for (var keys = Object.keys(object.labels), i = 0; i < keys.length; ++i)
                    message.labels[keys[i]] = String(object.labels[keys[i]]);
            }
            if (object.runtimeConfig != null) {
                if (typeof object.runtimeConfig !== "object")
                    throw TypeError(".ml_pipelines.PipelineJob.runtimeConfig: object expected");
                message.runtimeConfig = $root.ml_pipelines.PipelineJob.RuntimeConfig.fromObject(object.runtimeConfig);
            }
            return message;
        };

        /**
         * Creates a plain object from a PipelineJob message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.PipelineJob
         * @static
         * @param {ml_pipelines.PipelineJob} message PipelineJob
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        PipelineJob.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults)
                object.labels = {};
            if (options.defaults) {
                object.name = "";
                object.displayName = "";
                object.pipelineSpec = null;
                object.runtimeConfig = null;
            }
            if (message.name != null && message.hasOwnProperty("name"))
                object.name = message.name;
            if (message.displayName != null && message.hasOwnProperty("displayName"))
                object.displayName = message.displayName;
            if (message.pipelineSpec != null && message.hasOwnProperty("pipelineSpec"))
                object.pipelineSpec = $root.google.protobuf.Struct.toObject(message.pipelineSpec, options);
            var keys2;
            if (message.labels && (keys2 = Object.keys(message.labels)).length) {
                object.labels = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.labels[keys2[j]] = message.labels[keys2[j]];
            }
            if (message.runtimeConfig != null && message.hasOwnProperty("runtimeConfig"))
                object.runtimeConfig = $root.ml_pipelines.PipelineJob.RuntimeConfig.toObject(message.runtimeConfig, options);
            return object;
        };

        /**
         * Converts this PipelineJob to JSON.
         * @function toJSON
         * @memberof ml_pipelines.PipelineJob
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        PipelineJob.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        PipelineJob.RuntimeConfig = (function() {

            /**
             * Properties of a RuntimeConfig.
             * @memberof ml_pipelines.PipelineJob
             * @interface IRuntimeConfig
             * @property {Object.<string,ml_pipelines.IValue>|null} [parameters] RuntimeConfig parameters
             * @property {string|null} [gcsOutputDirectory] RuntimeConfig gcsOutputDirectory
             */

            /**
             * Constructs a new RuntimeConfig.
             * @memberof ml_pipelines.PipelineJob
             * @classdesc Represents a RuntimeConfig.
             * @implements IRuntimeConfig
             * @constructor
             * @param {ml_pipelines.PipelineJob.IRuntimeConfig=} [properties] Properties to set
             */
            function RuntimeConfig(properties) {
                this.parameters = {};
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * RuntimeConfig parameters.
             * @member {Object.<string,ml_pipelines.IValue>} parameters
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @instance
             */
            RuntimeConfig.prototype.parameters = $util.emptyObject;

            /**
             * RuntimeConfig gcsOutputDirectory.
             * @member {string} gcsOutputDirectory
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @instance
             */
            RuntimeConfig.prototype.gcsOutputDirectory = "";

            /**
             * Creates a new RuntimeConfig instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @static
             * @param {ml_pipelines.PipelineJob.IRuntimeConfig=} [properties] Properties to set
             * @returns {ml_pipelines.PipelineJob.RuntimeConfig} RuntimeConfig instance
             */
            RuntimeConfig.create = function create(properties) {
                return new RuntimeConfig(properties);
            };

            /**
             * Encodes the specified RuntimeConfig message. Does not implicitly {@link ml_pipelines.PipelineJob.RuntimeConfig.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @static
             * @param {ml_pipelines.PipelineJob.IRuntimeConfig} message RuntimeConfig message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            RuntimeConfig.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.parameters != null && Object.hasOwnProperty.call(message, "parameters"))
                    for (var keys = Object.keys(message.parameters), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.Value.encode(message.parameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                if (message.gcsOutputDirectory != null && Object.hasOwnProperty.call(message, "gcsOutputDirectory"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.gcsOutputDirectory);
                return writer;
            };

            /**
             * Encodes the specified RuntimeConfig message, length delimited. Does not implicitly {@link ml_pipelines.PipelineJob.RuntimeConfig.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @static
             * @param {ml_pipelines.PipelineJob.IRuntimeConfig} message RuntimeConfig message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            RuntimeConfig.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a RuntimeConfig message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.PipelineJob.RuntimeConfig} RuntimeConfig
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            RuntimeConfig.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineJob.RuntimeConfig(), key, value;
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        if (message.parameters === $util.emptyObject)
                            message.parameters = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.Value.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.parameters[key] = value;
                        break;
                    case 2:
                        message.gcsOutputDirectory = reader.string();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a RuntimeConfig message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.PipelineJob.RuntimeConfig} RuntimeConfig
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            RuntimeConfig.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a RuntimeConfig message.
             * @function verify
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            RuntimeConfig.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.parameters != null && message.hasOwnProperty("parameters")) {
                    if (!$util.isObject(message.parameters))
                        return "parameters: object expected";
                    var key = Object.keys(message.parameters);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.Value.verify(message.parameters[key[i]]);
                        if (error)
                            return "parameters." + error;
                    }
                }
                if (message.gcsOutputDirectory != null && message.hasOwnProperty("gcsOutputDirectory"))
                    if (!$util.isString(message.gcsOutputDirectory))
                        return "gcsOutputDirectory: string expected";
                return null;
            };

            /**
             * Creates a RuntimeConfig message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.PipelineJob.RuntimeConfig} RuntimeConfig
             */
            RuntimeConfig.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.PipelineJob.RuntimeConfig)
                    return object;
                var message = new $root.ml_pipelines.PipelineJob.RuntimeConfig();
                if (object.parameters) {
                    if (typeof object.parameters !== "object")
                        throw TypeError(".ml_pipelines.PipelineJob.RuntimeConfig.parameters: object expected");
                    message.parameters = {};
                    for (var keys = Object.keys(object.parameters), i = 0; i < keys.length; ++i) {
                        if (typeof object.parameters[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.PipelineJob.RuntimeConfig.parameters: object expected");
                        message.parameters[keys[i]] = $root.ml_pipelines.Value.fromObject(object.parameters[keys[i]]);
                    }
                }
                if (object.gcsOutputDirectory != null)
                    message.gcsOutputDirectory = String(object.gcsOutputDirectory);
                return message;
            };

            /**
             * Creates a plain object from a RuntimeConfig message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @static
             * @param {ml_pipelines.PipelineJob.RuntimeConfig} message RuntimeConfig
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            RuntimeConfig.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.objects || options.defaults)
                    object.parameters = {};
                if (options.defaults)
                    object.gcsOutputDirectory = "";
                var keys2;
                if (message.parameters && (keys2 = Object.keys(message.parameters)).length) {
                    object.parameters = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.parameters[keys2[j]] = $root.ml_pipelines.Value.toObject(message.parameters[keys2[j]], options);
                }
                if (message.gcsOutputDirectory != null && message.hasOwnProperty("gcsOutputDirectory"))
                    object.gcsOutputDirectory = message.gcsOutputDirectory;
                return object;
            };

            /**
             * Converts this RuntimeConfig to JSON.
             * @function toJSON
             * @memberof ml_pipelines.PipelineJob.RuntimeConfig
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            RuntimeConfig.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return RuntimeConfig;
        })();

        return PipelineJob;
    })();

    ml_pipelines.PipelineSpec = (function() {

        /**
         * Properties of a PipelineSpec.
         * @memberof ml_pipelines
         * @interface IPipelineSpec
         * @property {ml_pipelines.IPipelineInfo|null} [pipelineInfo] PipelineSpec pipelineInfo
         * @property {google.protobuf.IStruct|null} [deploymentSpec] PipelineSpec deploymentSpec
         * @property {string|null} [sdkVersion] PipelineSpec sdkVersion
         * @property {string|null} [schemaVersion] PipelineSpec schemaVersion
         * @property {Object.<string,ml_pipelines.IComponentSpec>|null} [components] PipelineSpec components
         * @property {ml_pipelines.IComponentSpec|null} [root] PipelineSpec root
         */

        /**
         * Constructs a new PipelineSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a PipelineSpec.
         * @implements IPipelineSpec
         * @constructor
         * @param {ml_pipelines.IPipelineSpec=} [properties] Properties to set
         */
        function PipelineSpec(properties) {
            this.components = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * PipelineSpec pipelineInfo.
         * @member {ml_pipelines.IPipelineInfo|null|undefined} pipelineInfo
         * @memberof ml_pipelines.PipelineSpec
         * @instance
         */
        PipelineSpec.prototype.pipelineInfo = null;

        /**
         * PipelineSpec deploymentSpec.
         * @member {google.protobuf.IStruct|null|undefined} deploymentSpec
         * @memberof ml_pipelines.PipelineSpec
         * @instance
         */
        PipelineSpec.prototype.deploymentSpec = null;

        /**
         * PipelineSpec sdkVersion.
         * @member {string} sdkVersion
         * @memberof ml_pipelines.PipelineSpec
         * @instance
         */
        PipelineSpec.prototype.sdkVersion = "";

        /**
         * PipelineSpec schemaVersion.
         * @member {string} schemaVersion
         * @memberof ml_pipelines.PipelineSpec
         * @instance
         */
        PipelineSpec.prototype.schemaVersion = "";

        /**
         * PipelineSpec components.
         * @member {Object.<string,ml_pipelines.IComponentSpec>} components
         * @memberof ml_pipelines.PipelineSpec
         * @instance
         */
        PipelineSpec.prototype.components = $util.emptyObject;

        /**
         * PipelineSpec root.
         * @member {ml_pipelines.IComponentSpec|null|undefined} root
         * @memberof ml_pipelines.PipelineSpec
         * @instance
         */
        PipelineSpec.prototype.root = null;

        /**
         * Creates a new PipelineSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.PipelineSpec
         * @static
         * @param {ml_pipelines.IPipelineSpec=} [properties] Properties to set
         * @returns {ml_pipelines.PipelineSpec} PipelineSpec instance
         */
        PipelineSpec.create = function create(properties) {
            return new PipelineSpec(properties);
        };

        /**
         * Encodes the specified PipelineSpec message. Does not implicitly {@link ml_pipelines.PipelineSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.PipelineSpec
         * @static
         * @param {ml_pipelines.IPipelineSpec} message PipelineSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.pipelineInfo != null && Object.hasOwnProperty.call(message, "pipelineInfo"))
                $root.ml_pipelines.PipelineInfo.encode(message.pipelineInfo, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
            if (message.sdkVersion != null && Object.hasOwnProperty.call(message, "sdkVersion"))
                writer.uint32(/* id 4, wireType 2 =*/34).string(message.sdkVersion);
            if (message.schemaVersion != null && Object.hasOwnProperty.call(message, "schemaVersion"))
                writer.uint32(/* id 5, wireType 2 =*/42).string(message.schemaVersion);
            if (message.deploymentSpec != null && Object.hasOwnProperty.call(message, "deploymentSpec"))
                $root.google.protobuf.Struct.encode(message.deploymentSpec, writer.uint32(/* id 7, wireType 2 =*/58).fork()).ldelim();
            if (message.components != null && Object.hasOwnProperty.call(message, "components"))
                for (var keys = Object.keys(message.components), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 8, wireType 2 =*/66).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.ComponentSpec.encode(message.components[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.root != null && Object.hasOwnProperty.call(message, "root"))
                $root.ml_pipelines.ComponentSpec.encode(message.root, writer.uint32(/* id 9, wireType 2 =*/74).fork()).ldelim();
            return writer;
        };

        /**
         * Encodes the specified PipelineSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.PipelineSpec
         * @static
         * @param {ml_pipelines.IPipelineSpec} message PipelineSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a PipelineSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.PipelineSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.PipelineSpec} PipelineSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineSpec(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.pipelineInfo = $root.ml_pipelines.PipelineInfo.decode(reader, reader.uint32());
                    break;
                case 7:
                    message.deploymentSpec = $root.google.protobuf.Struct.decode(reader, reader.uint32());
                    break;
                case 4:
                    message.sdkVersion = reader.string();
                    break;
                case 5:
                    message.schemaVersion = reader.string();
                    break;
                case 8:
                    if (message.components === $util.emptyObject)
                        message.components = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.ComponentSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.components[key] = value;
                    break;
                case 9:
                    message.root = $root.ml_pipelines.ComponentSpec.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a PipelineSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.PipelineSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.PipelineSpec} PipelineSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a PipelineSpec message.
         * @function verify
         * @memberof ml_pipelines.PipelineSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        PipelineSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.pipelineInfo != null && message.hasOwnProperty("pipelineInfo")) {
                var error = $root.ml_pipelines.PipelineInfo.verify(message.pipelineInfo);
                if (error)
                    return "pipelineInfo." + error;
            }
            if (message.deploymentSpec != null && message.hasOwnProperty("deploymentSpec")) {
                var error = $root.google.protobuf.Struct.verify(message.deploymentSpec);
                if (error)
                    return "deploymentSpec." + error;
            }
            if (message.sdkVersion != null && message.hasOwnProperty("sdkVersion"))
                if (!$util.isString(message.sdkVersion))
                    return "sdkVersion: string expected";
            if (message.schemaVersion != null && message.hasOwnProperty("schemaVersion"))
                if (!$util.isString(message.schemaVersion))
                    return "schemaVersion: string expected";
            if (message.components != null && message.hasOwnProperty("components")) {
                if (!$util.isObject(message.components))
                    return "components: object expected";
                var key = Object.keys(message.components);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.ComponentSpec.verify(message.components[key[i]]);
                    if (error)
                        return "components." + error;
                }
            }
            if (message.root != null && message.hasOwnProperty("root")) {
                var error = $root.ml_pipelines.ComponentSpec.verify(message.root);
                if (error)
                    return "root." + error;
            }
            return null;
        };

        /**
         * Creates a PipelineSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.PipelineSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.PipelineSpec} PipelineSpec
         */
        PipelineSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.PipelineSpec)
                return object;
            var message = new $root.ml_pipelines.PipelineSpec();
            if (object.pipelineInfo != null) {
                if (typeof object.pipelineInfo !== "object")
                    throw TypeError(".ml_pipelines.PipelineSpec.pipelineInfo: object expected");
                message.pipelineInfo = $root.ml_pipelines.PipelineInfo.fromObject(object.pipelineInfo);
            }
            if (object.deploymentSpec != null) {
                if (typeof object.deploymentSpec !== "object")
                    throw TypeError(".ml_pipelines.PipelineSpec.deploymentSpec: object expected");
                message.deploymentSpec = $root.google.protobuf.Struct.fromObject(object.deploymentSpec);
            }
            if (object.sdkVersion != null)
                message.sdkVersion = String(object.sdkVersion);
            if (object.schemaVersion != null)
                message.schemaVersion = String(object.schemaVersion);
            if (object.components) {
                if (typeof object.components !== "object")
                    throw TypeError(".ml_pipelines.PipelineSpec.components: object expected");
                message.components = {};
                for (var keys = Object.keys(object.components), i = 0; i < keys.length; ++i) {
                    if (typeof object.components[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.PipelineSpec.components: object expected");
                    message.components[keys[i]] = $root.ml_pipelines.ComponentSpec.fromObject(object.components[keys[i]]);
                }
            }
            if (object.root != null) {
                if (typeof object.root !== "object")
                    throw TypeError(".ml_pipelines.PipelineSpec.root: object expected");
                message.root = $root.ml_pipelines.ComponentSpec.fromObject(object.root);
            }
            return message;
        };

        /**
         * Creates a plain object from a PipelineSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.PipelineSpec
         * @static
         * @param {ml_pipelines.PipelineSpec} message PipelineSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        PipelineSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults)
                object.components = {};
            if (options.defaults) {
                object.pipelineInfo = null;
                object.sdkVersion = "";
                object.schemaVersion = "";
                object.deploymentSpec = null;
                object.root = null;
            }
            if (message.pipelineInfo != null && message.hasOwnProperty("pipelineInfo"))
                object.pipelineInfo = $root.ml_pipelines.PipelineInfo.toObject(message.pipelineInfo, options);
            if (message.sdkVersion != null && message.hasOwnProperty("sdkVersion"))
                object.sdkVersion = message.sdkVersion;
            if (message.schemaVersion != null && message.hasOwnProperty("schemaVersion"))
                object.schemaVersion = message.schemaVersion;
            if (message.deploymentSpec != null && message.hasOwnProperty("deploymentSpec"))
                object.deploymentSpec = $root.google.protobuf.Struct.toObject(message.deploymentSpec, options);
            var keys2;
            if (message.components && (keys2 = Object.keys(message.components)).length) {
                object.components = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.components[keys2[j]] = $root.ml_pipelines.ComponentSpec.toObject(message.components[keys2[j]], options);
            }
            if (message.root != null && message.hasOwnProperty("root"))
                object.root = $root.ml_pipelines.ComponentSpec.toObject(message.root, options);
            return object;
        };

        /**
         * Converts this PipelineSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.PipelineSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        PipelineSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        PipelineSpec.RuntimeParameter = (function() {

            /**
             * Properties of a RuntimeParameter.
             * @memberof ml_pipelines.PipelineSpec
             * @interface IRuntimeParameter
             * @property {ml_pipelines.PrimitiveType.PrimitiveTypeEnum|null} [type] RuntimeParameter type
             * @property {ml_pipelines.IValue|null} [defaultValue] RuntimeParameter defaultValue
             */

            /**
             * Constructs a new RuntimeParameter.
             * @memberof ml_pipelines.PipelineSpec
             * @classdesc Represents a RuntimeParameter.
             * @implements IRuntimeParameter
             * @constructor
             * @param {ml_pipelines.PipelineSpec.IRuntimeParameter=} [properties] Properties to set
             */
            function RuntimeParameter(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * RuntimeParameter type.
             * @member {ml_pipelines.PrimitiveType.PrimitiveTypeEnum} type
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @instance
             */
            RuntimeParameter.prototype.type = 0;

            /**
             * RuntimeParameter defaultValue.
             * @member {ml_pipelines.IValue|null|undefined} defaultValue
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @instance
             */
            RuntimeParameter.prototype.defaultValue = null;

            /**
             * Creates a new RuntimeParameter instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @static
             * @param {ml_pipelines.PipelineSpec.IRuntimeParameter=} [properties] Properties to set
             * @returns {ml_pipelines.PipelineSpec.RuntimeParameter} RuntimeParameter instance
             */
            RuntimeParameter.create = function create(properties) {
                return new RuntimeParameter(properties);
            };

            /**
             * Encodes the specified RuntimeParameter message. Does not implicitly {@link ml_pipelines.PipelineSpec.RuntimeParameter.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @static
             * @param {ml_pipelines.PipelineSpec.IRuntimeParameter} message RuntimeParameter message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            RuntimeParameter.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.type != null && Object.hasOwnProperty.call(message, "type"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int32(message.type);
                if (message.defaultValue != null && Object.hasOwnProperty.call(message, "defaultValue"))
                    $root.ml_pipelines.Value.encode(message.defaultValue, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified RuntimeParameter message, length delimited. Does not implicitly {@link ml_pipelines.PipelineSpec.RuntimeParameter.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @static
             * @param {ml_pipelines.PipelineSpec.IRuntimeParameter} message RuntimeParameter message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            RuntimeParameter.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a RuntimeParameter message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.PipelineSpec.RuntimeParameter} RuntimeParameter
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            RuntimeParameter.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineSpec.RuntimeParameter();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.type = reader.int32();
                        break;
                    case 2:
                        message.defaultValue = $root.ml_pipelines.Value.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a RuntimeParameter message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.PipelineSpec.RuntimeParameter} RuntimeParameter
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            RuntimeParameter.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a RuntimeParameter message.
             * @function verify
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            RuntimeParameter.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.type != null && message.hasOwnProperty("type"))
                    switch (message.type) {
                    default:
                        return "type: enum value expected";
                    case 0:
                    case 1:
                    case 2:
                    case 3:
                        break;
                    }
                if (message.defaultValue != null && message.hasOwnProperty("defaultValue")) {
                    var error = $root.ml_pipelines.Value.verify(message.defaultValue);
                    if (error)
                        return "defaultValue." + error;
                }
                return null;
            };

            /**
             * Creates a RuntimeParameter message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.PipelineSpec.RuntimeParameter} RuntimeParameter
             */
            RuntimeParameter.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.PipelineSpec.RuntimeParameter)
                    return object;
                var message = new $root.ml_pipelines.PipelineSpec.RuntimeParameter();
                switch (object.type) {
                case "PRIMITIVE_TYPE_UNSPECIFIED":
                case 0:
                    message.type = 0;
                    break;
                case "INT":
                case 1:
                    message.type = 1;
                    break;
                case "DOUBLE":
                case 2:
                    message.type = 2;
                    break;
                case "STRING":
                case 3:
                    message.type = 3;
                    break;
                }
                if (object.defaultValue != null) {
                    if (typeof object.defaultValue !== "object")
                        throw TypeError(".ml_pipelines.PipelineSpec.RuntimeParameter.defaultValue: object expected");
                    message.defaultValue = $root.ml_pipelines.Value.fromObject(object.defaultValue);
                }
                return message;
            };

            /**
             * Creates a plain object from a RuntimeParameter message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @static
             * @param {ml_pipelines.PipelineSpec.RuntimeParameter} message RuntimeParameter
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            RuntimeParameter.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    object.type = options.enums === String ? "PRIMITIVE_TYPE_UNSPECIFIED" : 0;
                    object.defaultValue = null;
                }
                if (message.type != null && message.hasOwnProperty("type"))
                    object.type = options.enums === String ? $root.ml_pipelines.PrimitiveType.PrimitiveTypeEnum[message.type] : message.type;
                if (message.defaultValue != null && message.hasOwnProperty("defaultValue"))
                    object.defaultValue = $root.ml_pipelines.Value.toObject(message.defaultValue, options);
                return object;
            };

            /**
             * Converts this RuntimeParameter to JSON.
             * @function toJSON
             * @memberof ml_pipelines.PipelineSpec.RuntimeParameter
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            RuntimeParameter.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return RuntimeParameter;
        })();

        return PipelineSpec;
    })();

    ml_pipelines.ComponentSpec = (function() {

        /**
         * Properties of a ComponentSpec.
         * @memberof ml_pipelines
         * @interface IComponentSpec
         * @property {ml_pipelines.IComponentInputsSpec|null} [inputDefinitions] ComponentSpec inputDefinitions
         * @property {ml_pipelines.IComponentOutputsSpec|null} [outputDefinitions] ComponentSpec outputDefinitions
         * @property {ml_pipelines.IDagSpec|null} [dag] ComponentSpec dag
         * @property {string|null} [executorLabel] ComponentSpec executorLabel
         */

        /**
         * Constructs a new ComponentSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a ComponentSpec.
         * @implements IComponentSpec
         * @constructor
         * @param {ml_pipelines.IComponentSpec=} [properties] Properties to set
         */
        function ComponentSpec(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ComponentSpec inputDefinitions.
         * @member {ml_pipelines.IComponentInputsSpec|null|undefined} inputDefinitions
         * @memberof ml_pipelines.ComponentSpec
         * @instance
         */
        ComponentSpec.prototype.inputDefinitions = null;

        /**
         * ComponentSpec outputDefinitions.
         * @member {ml_pipelines.IComponentOutputsSpec|null|undefined} outputDefinitions
         * @memberof ml_pipelines.ComponentSpec
         * @instance
         */
        ComponentSpec.prototype.outputDefinitions = null;

        /**
         * ComponentSpec dag.
         * @member {ml_pipelines.IDagSpec|null|undefined} dag
         * @memberof ml_pipelines.ComponentSpec
         * @instance
         */
        ComponentSpec.prototype.dag = null;

        /**
         * ComponentSpec executorLabel.
         * @member {string|null|undefined} executorLabel
         * @memberof ml_pipelines.ComponentSpec
         * @instance
         */
        ComponentSpec.prototype.executorLabel = null;

        // OneOf field names bound to virtual getters and setters
        var $oneOfFields;

        /**
         * ComponentSpec implementation.
         * @member {"dag"|"executorLabel"|undefined} implementation
         * @memberof ml_pipelines.ComponentSpec
         * @instance
         */
        Object.defineProperty(ComponentSpec.prototype, "implementation", {
            get: $util.oneOfGetter($oneOfFields = ["dag", "executorLabel"]),
            set: $util.oneOfSetter($oneOfFields)
        });

        /**
         * Creates a new ComponentSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ComponentSpec
         * @static
         * @param {ml_pipelines.IComponentSpec=} [properties] Properties to set
         * @returns {ml_pipelines.ComponentSpec} ComponentSpec instance
         */
        ComponentSpec.create = function create(properties) {
            return new ComponentSpec(properties);
        };

        /**
         * Encodes the specified ComponentSpec message. Does not implicitly {@link ml_pipelines.ComponentSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ComponentSpec
         * @static
         * @param {ml_pipelines.IComponentSpec} message ComponentSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ComponentSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.inputDefinitions != null && Object.hasOwnProperty.call(message, "inputDefinitions"))
                $root.ml_pipelines.ComponentInputsSpec.encode(message.inputDefinitions, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
            if (message.outputDefinitions != null && Object.hasOwnProperty.call(message, "outputDefinitions"))
                $root.ml_pipelines.ComponentOutputsSpec.encode(message.outputDefinitions, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
            if (message.dag != null && Object.hasOwnProperty.call(message, "dag"))
                $root.ml_pipelines.DagSpec.encode(message.dag, writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
            if (message.executorLabel != null && Object.hasOwnProperty.call(message, "executorLabel"))
                writer.uint32(/* id 4, wireType 2 =*/34).string(message.executorLabel);
            return writer;
        };

        /**
         * Encodes the specified ComponentSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ComponentSpec
         * @static
         * @param {ml_pipelines.IComponentSpec} message ComponentSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ComponentSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a ComponentSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ComponentSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ComponentSpec} ComponentSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ComponentSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ComponentSpec();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.inputDefinitions = $root.ml_pipelines.ComponentInputsSpec.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.outputDefinitions = $root.ml_pipelines.ComponentOutputsSpec.decode(reader, reader.uint32());
                    break;
                case 3:
                    message.dag = $root.ml_pipelines.DagSpec.decode(reader, reader.uint32());
                    break;
                case 4:
                    message.executorLabel = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a ComponentSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ComponentSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ComponentSpec} ComponentSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ComponentSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a ComponentSpec message.
         * @function verify
         * @memberof ml_pipelines.ComponentSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ComponentSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            var properties = {};
            if (message.inputDefinitions != null && message.hasOwnProperty("inputDefinitions")) {
                var error = $root.ml_pipelines.ComponentInputsSpec.verify(message.inputDefinitions);
                if (error)
                    return "inputDefinitions." + error;
            }
            if (message.outputDefinitions != null && message.hasOwnProperty("outputDefinitions")) {
                var error = $root.ml_pipelines.ComponentOutputsSpec.verify(message.outputDefinitions);
                if (error)
                    return "outputDefinitions." + error;
            }
            if (message.dag != null && message.hasOwnProperty("dag")) {
                properties.implementation = 1;
                {
                    var error = $root.ml_pipelines.DagSpec.verify(message.dag);
                    if (error)
                        return "dag." + error;
                }
            }
            if (message.executorLabel != null && message.hasOwnProperty("executorLabel")) {
                if (properties.implementation === 1)
                    return "implementation: multiple values";
                properties.implementation = 1;
                if (!$util.isString(message.executorLabel))
                    return "executorLabel: string expected";
            }
            return null;
        };

        /**
         * Creates a ComponentSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ComponentSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ComponentSpec} ComponentSpec
         */
        ComponentSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ComponentSpec)
                return object;
            var message = new $root.ml_pipelines.ComponentSpec();
            if (object.inputDefinitions != null) {
                if (typeof object.inputDefinitions !== "object")
                    throw TypeError(".ml_pipelines.ComponentSpec.inputDefinitions: object expected");
                message.inputDefinitions = $root.ml_pipelines.ComponentInputsSpec.fromObject(object.inputDefinitions);
            }
            if (object.outputDefinitions != null) {
                if (typeof object.outputDefinitions !== "object")
                    throw TypeError(".ml_pipelines.ComponentSpec.outputDefinitions: object expected");
                message.outputDefinitions = $root.ml_pipelines.ComponentOutputsSpec.fromObject(object.outputDefinitions);
            }
            if (object.dag != null) {
                if (typeof object.dag !== "object")
                    throw TypeError(".ml_pipelines.ComponentSpec.dag: object expected");
                message.dag = $root.ml_pipelines.DagSpec.fromObject(object.dag);
            }
            if (object.executorLabel != null)
                message.executorLabel = String(object.executorLabel);
            return message;
        };

        /**
         * Creates a plain object from a ComponentSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ComponentSpec
         * @static
         * @param {ml_pipelines.ComponentSpec} message ComponentSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ComponentSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.defaults) {
                object.inputDefinitions = null;
                object.outputDefinitions = null;
            }
            if (message.inputDefinitions != null && message.hasOwnProperty("inputDefinitions"))
                object.inputDefinitions = $root.ml_pipelines.ComponentInputsSpec.toObject(message.inputDefinitions, options);
            if (message.outputDefinitions != null && message.hasOwnProperty("outputDefinitions"))
                object.outputDefinitions = $root.ml_pipelines.ComponentOutputsSpec.toObject(message.outputDefinitions, options);
            if (message.dag != null && message.hasOwnProperty("dag")) {
                object.dag = $root.ml_pipelines.DagSpec.toObject(message.dag, options);
                if (options.oneofs)
                    object.implementation = "dag";
            }
            if (message.executorLabel != null && message.hasOwnProperty("executorLabel")) {
                object.executorLabel = message.executorLabel;
                if (options.oneofs)
                    object.implementation = "executorLabel";
            }
            return object;
        };

        /**
         * Converts this ComponentSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ComponentSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ComponentSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return ComponentSpec;
    })();

    ml_pipelines.DagSpec = (function() {

        /**
         * Properties of a DagSpec.
         * @memberof ml_pipelines
         * @interface IDagSpec
         * @property {Object.<string,ml_pipelines.IPipelineTaskSpec>|null} [tasks] DagSpec tasks
         * @property {ml_pipelines.IDagOutputsSpec|null} [outputs] DagSpec outputs
         */

        /**
         * Constructs a new DagSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a DagSpec.
         * @implements IDagSpec
         * @constructor
         * @param {ml_pipelines.IDagSpec=} [properties] Properties to set
         */
        function DagSpec(properties) {
            this.tasks = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * DagSpec tasks.
         * @member {Object.<string,ml_pipelines.IPipelineTaskSpec>} tasks
         * @memberof ml_pipelines.DagSpec
         * @instance
         */
        DagSpec.prototype.tasks = $util.emptyObject;

        /**
         * DagSpec outputs.
         * @member {ml_pipelines.IDagOutputsSpec|null|undefined} outputs
         * @memberof ml_pipelines.DagSpec
         * @instance
         */
        DagSpec.prototype.outputs = null;

        /**
         * Creates a new DagSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.DagSpec
         * @static
         * @param {ml_pipelines.IDagSpec=} [properties] Properties to set
         * @returns {ml_pipelines.DagSpec} DagSpec instance
         */
        DagSpec.create = function create(properties) {
            return new DagSpec(properties);
        };

        /**
         * Encodes the specified DagSpec message. Does not implicitly {@link ml_pipelines.DagSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.DagSpec
         * @static
         * @param {ml_pipelines.IDagSpec} message DagSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        DagSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.tasks != null && Object.hasOwnProperty.call(message, "tasks"))
                for (var keys = Object.keys(message.tasks), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.PipelineTaskSpec.encode(message.tasks[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.outputs != null && Object.hasOwnProperty.call(message, "outputs"))
                $root.ml_pipelines.DagOutputsSpec.encode(message.outputs, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
            return writer;
        };

        /**
         * Encodes the specified DagSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.DagSpec
         * @static
         * @param {ml_pipelines.IDagSpec} message DagSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        DagSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a DagSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.DagSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.DagSpec} DagSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        DagSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.DagSpec(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    if (message.tasks === $util.emptyObject)
                        message.tasks = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.PipelineTaskSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.tasks[key] = value;
                    break;
                case 2:
                    message.outputs = $root.ml_pipelines.DagOutputsSpec.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a DagSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.DagSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.DagSpec} DagSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        DagSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a DagSpec message.
         * @function verify
         * @memberof ml_pipelines.DagSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        DagSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.tasks != null && message.hasOwnProperty("tasks")) {
                if (!$util.isObject(message.tasks))
                    return "tasks: object expected";
                var key = Object.keys(message.tasks);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.PipelineTaskSpec.verify(message.tasks[key[i]]);
                    if (error)
                        return "tasks." + error;
                }
            }
            if (message.outputs != null && message.hasOwnProperty("outputs")) {
                var error = $root.ml_pipelines.DagOutputsSpec.verify(message.outputs);
                if (error)
                    return "outputs." + error;
            }
            return null;
        };

        /**
         * Creates a DagSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.DagSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.DagSpec} DagSpec
         */
        DagSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.DagSpec)
                return object;
            var message = new $root.ml_pipelines.DagSpec();
            if (object.tasks) {
                if (typeof object.tasks !== "object")
                    throw TypeError(".ml_pipelines.DagSpec.tasks: object expected");
                message.tasks = {};
                for (var keys = Object.keys(object.tasks), i = 0; i < keys.length; ++i) {
                    if (typeof object.tasks[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.DagSpec.tasks: object expected");
                    message.tasks[keys[i]] = $root.ml_pipelines.PipelineTaskSpec.fromObject(object.tasks[keys[i]]);
                }
            }
            if (object.outputs != null) {
                if (typeof object.outputs !== "object")
                    throw TypeError(".ml_pipelines.DagSpec.outputs: object expected");
                message.outputs = $root.ml_pipelines.DagOutputsSpec.fromObject(object.outputs);
            }
            return message;
        };

        /**
         * Creates a plain object from a DagSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.DagSpec
         * @static
         * @param {ml_pipelines.DagSpec} message DagSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        DagSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults)
                object.tasks = {};
            if (options.defaults)
                object.outputs = null;
            var keys2;
            if (message.tasks && (keys2 = Object.keys(message.tasks)).length) {
                object.tasks = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.tasks[keys2[j]] = $root.ml_pipelines.PipelineTaskSpec.toObject(message.tasks[keys2[j]], options);
            }
            if (message.outputs != null && message.hasOwnProperty("outputs"))
                object.outputs = $root.ml_pipelines.DagOutputsSpec.toObject(message.outputs, options);
            return object;
        };

        /**
         * Converts this DagSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.DagSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        DagSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return DagSpec;
    })();

    ml_pipelines.DagOutputsSpec = (function() {

        /**
         * Properties of a DagOutputsSpec.
         * @memberof ml_pipelines
         * @interface IDagOutputsSpec
         * @property {Object.<string,ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec>|null} [artifacts] DagOutputsSpec artifacts
         * @property {Object.<string,ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec>|null} [parameters] DagOutputsSpec parameters
         */

        /**
         * Constructs a new DagOutputsSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a DagOutputsSpec.
         * @implements IDagOutputsSpec
         * @constructor
         * @param {ml_pipelines.IDagOutputsSpec=} [properties] Properties to set
         */
        function DagOutputsSpec(properties) {
            this.artifacts = {};
            this.parameters = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * DagOutputsSpec artifacts.
         * @member {Object.<string,ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec>} artifacts
         * @memberof ml_pipelines.DagOutputsSpec
         * @instance
         */
        DagOutputsSpec.prototype.artifacts = $util.emptyObject;

        /**
         * DagOutputsSpec parameters.
         * @member {Object.<string,ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec>} parameters
         * @memberof ml_pipelines.DagOutputsSpec
         * @instance
         */
        DagOutputsSpec.prototype.parameters = $util.emptyObject;

        /**
         * Creates a new DagOutputsSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.DagOutputsSpec
         * @static
         * @param {ml_pipelines.IDagOutputsSpec=} [properties] Properties to set
         * @returns {ml_pipelines.DagOutputsSpec} DagOutputsSpec instance
         */
        DagOutputsSpec.create = function create(properties) {
            return new DagOutputsSpec(properties);
        };

        /**
         * Encodes the specified DagOutputsSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.DagOutputsSpec
         * @static
         * @param {ml_pipelines.IDagOutputsSpec} message DagOutputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        DagOutputsSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.artifacts != null && Object.hasOwnProperty.call(message, "artifacts"))
                for (var keys = Object.keys(message.artifacts), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.encode(message.artifacts[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.parameters != null && Object.hasOwnProperty.call(message, "parameters"))
                for (var keys = Object.keys(message.parameters), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.encode(message.parameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            return writer;
        };

        /**
         * Encodes the specified DagOutputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.DagOutputsSpec
         * @static
         * @param {ml_pipelines.IDagOutputsSpec} message DagOutputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        DagOutputsSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a DagOutputsSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.DagOutputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.DagOutputsSpec} DagOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        DagOutputsSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.DagOutputsSpec(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    if (message.artifacts === $util.emptyObject)
                        message.artifacts = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.artifacts[key] = value;
                    break;
                case 2:
                    if (message.parameters === $util.emptyObject)
                        message.parameters = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.parameters[key] = value;
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a DagOutputsSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.DagOutputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.DagOutputsSpec} DagOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        DagOutputsSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a DagOutputsSpec message.
         * @function verify
         * @memberof ml_pipelines.DagOutputsSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        DagOutputsSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.artifacts != null && message.hasOwnProperty("artifacts")) {
                if (!$util.isObject(message.artifacts))
                    return "artifacts: object expected";
                var key = Object.keys(message.artifacts);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.verify(message.artifacts[key[i]]);
                    if (error)
                        return "artifacts." + error;
                }
            }
            if (message.parameters != null && message.hasOwnProperty("parameters")) {
                if (!$util.isObject(message.parameters))
                    return "parameters: object expected";
                var key = Object.keys(message.parameters);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.verify(message.parameters[key[i]]);
                    if (error)
                        return "parameters." + error;
                }
            }
            return null;
        };

        /**
         * Creates a DagOutputsSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.DagOutputsSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.DagOutputsSpec} DagOutputsSpec
         */
        DagOutputsSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.DagOutputsSpec)
                return object;
            var message = new $root.ml_pipelines.DagOutputsSpec();
            if (object.artifacts) {
                if (typeof object.artifacts !== "object")
                    throw TypeError(".ml_pipelines.DagOutputsSpec.artifacts: object expected");
                message.artifacts = {};
                for (var keys = Object.keys(object.artifacts), i = 0; i < keys.length; ++i) {
                    if (typeof object.artifacts[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.DagOutputsSpec.artifacts: object expected");
                    message.artifacts[keys[i]] = $root.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.fromObject(object.artifacts[keys[i]]);
                }
            }
            if (object.parameters) {
                if (typeof object.parameters !== "object")
                    throw TypeError(".ml_pipelines.DagOutputsSpec.parameters: object expected");
                message.parameters = {};
                for (var keys = Object.keys(object.parameters), i = 0; i < keys.length; ++i) {
                    if (typeof object.parameters[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.DagOutputsSpec.parameters: object expected");
                    message.parameters[keys[i]] = $root.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.fromObject(object.parameters[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a DagOutputsSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.DagOutputsSpec
         * @static
         * @param {ml_pipelines.DagOutputsSpec} message DagOutputsSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        DagOutputsSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults) {
                object.artifacts = {};
                object.parameters = {};
            }
            var keys2;
            if (message.artifacts && (keys2 = Object.keys(message.artifacts)).length) {
                object.artifacts = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.artifacts[keys2[j]] = $root.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.toObject(message.artifacts[keys2[j]], options);
            }
            if (message.parameters && (keys2 = Object.keys(message.parameters)).length) {
                object.parameters = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.parameters[keys2[j]] = $root.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.toObject(message.parameters[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this DagOutputsSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.DagOutputsSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        DagOutputsSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        DagOutputsSpec.ArtifactSelectorSpec = (function() {

            /**
             * Properties of an ArtifactSelectorSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @interface IArtifactSelectorSpec
             * @property {string|null} [producerSubtask] ArtifactSelectorSpec producerSubtask
             * @property {string|null} [outputArtifactKey] ArtifactSelectorSpec outputArtifactKey
             */

            /**
             * Constructs a new ArtifactSelectorSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @classdesc Represents an ArtifactSelectorSpec.
             * @implements IArtifactSelectorSpec
             * @constructor
             * @param {ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec=} [properties] Properties to set
             */
            function ArtifactSelectorSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ArtifactSelectorSpec producerSubtask.
             * @member {string} producerSubtask
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @instance
             */
            ArtifactSelectorSpec.prototype.producerSubtask = "";

            /**
             * ArtifactSelectorSpec outputArtifactKey.
             * @member {string} outputArtifactKey
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @instance
             */
            ArtifactSelectorSpec.prototype.outputArtifactKey = "";

            /**
             * Creates a new ArtifactSelectorSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec=} [properties] Properties to set
             * @returns {ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} ArtifactSelectorSpec instance
             */
            ArtifactSelectorSpec.create = function create(properties) {
                return new ArtifactSelectorSpec(properties);
            };

            /**
             * Encodes the specified ArtifactSelectorSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec} message ArtifactSelectorSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ArtifactSelectorSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.producerSubtask != null && Object.hasOwnProperty.call(message, "producerSubtask"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.producerSubtask);
                if (message.outputArtifactKey != null && Object.hasOwnProperty.call(message, "outputArtifactKey"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.outputArtifactKey);
                return writer;
            };

            /**
             * Encodes the specified ArtifactSelectorSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec} message ArtifactSelectorSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ArtifactSelectorSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an ArtifactSelectorSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} ArtifactSelectorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ArtifactSelectorSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.producerSubtask = reader.string();
                        break;
                    case 2:
                        message.outputArtifactKey = reader.string();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an ArtifactSelectorSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} ArtifactSelectorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ArtifactSelectorSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an ArtifactSelectorSpec message.
             * @function verify
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ArtifactSelectorSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.producerSubtask != null && message.hasOwnProperty("producerSubtask"))
                    if (!$util.isString(message.producerSubtask))
                        return "producerSubtask: string expected";
                if (message.outputArtifactKey != null && message.hasOwnProperty("outputArtifactKey"))
                    if (!$util.isString(message.outputArtifactKey))
                        return "outputArtifactKey: string expected";
                return null;
            };

            /**
             * Creates an ArtifactSelectorSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} ArtifactSelectorSpec
             */
            ArtifactSelectorSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec)
                    return object;
                var message = new $root.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec();
                if (object.producerSubtask != null)
                    message.producerSubtask = String(object.producerSubtask);
                if (object.outputArtifactKey != null)
                    message.outputArtifactKey = String(object.outputArtifactKey);
                return message;
            };

            /**
             * Creates a plain object from an ArtifactSelectorSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec} message ArtifactSelectorSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ArtifactSelectorSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    object.producerSubtask = "";
                    object.outputArtifactKey = "";
                }
                if (message.producerSubtask != null && message.hasOwnProperty("producerSubtask"))
                    object.producerSubtask = message.producerSubtask;
                if (message.outputArtifactKey != null && message.hasOwnProperty("outputArtifactKey"))
                    object.outputArtifactKey = message.outputArtifactKey;
                return object;
            };

            /**
             * Converts this ArtifactSelectorSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ArtifactSelectorSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ArtifactSelectorSpec;
        })();

        DagOutputsSpec.DagOutputArtifactSpec = (function() {

            /**
             * Properties of a DagOutputArtifactSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @interface IDagOutputArtifactSpec
             * @property {Array.<ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec>|null} [artifactSelectors] DagOutputArtifactSpec artifactSelectors
             */

            /**
             * Constructs a new DagOutputArtifactSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @classdesc Represents a DagOutputArtifactSpec.
             * @implements IDagOutputArtifactSpec
             * @constructor
             * @param {ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec=} [properties] Properties to set
             */
            function DagOutputArtifactSpec(properties) {
                this.artifactSelectors = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * DagOutputArtifactSpec artifactSelectors.
             * @member {Array.<ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec>} artifactSelectors
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @instance
             */
            DagOutputArtifactSpec.prototype.artifactSelectors = $util.emptyArray;

            /**
             * Creates a new DagOutputArtifactSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec=} [properties] Properties to set
             * @returns {ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} DagOutputArtifactSpec instance
             */
            DagOutputArtifactSpec.create = function create(properties) {
                return new DagOutputArtifactSpec(properties);
            };

            /**
             * Encodes the specified DagOutputArtifactSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec} message DagOutputArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            DagOutputArtifactSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.artifactSelectors != null && message.artifactSelectors.length)
                    for (var i = 0; i < message.artifactSelectors.length; ++i)
                        $root.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.encode(message.artifactSelectors[i], writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified DagOutputArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec} message DagOutputArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            DagOutputArtifactSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a DagOutputArtifactSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} DagOutputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            DagOutputArtifactSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        if (!(message.artifactSelectors && message.artifactSelectors.length))
                            message.artifactSelectors = [];
                        message.artifactSelectors.push($root.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.decode(reader, reader.uint32()));
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a DagOutputArtifactSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} DagOutputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            DagOutputArtifactSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a DagOutputArtifactSpec message.
             * @function verify
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            DagOutputArtifactSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.artifactSelectors != null && message.hasOwnProperty("artifactSelectors")) {
                    if (!Array.isArray(message.artifactSelectors))
                        return "artifactSelectors: array expected";
                    for (var i = 0; i < message.artifactSelectors.length; ++i) {
                        var error = $root.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.verify(message.artifactSelectors[i]);
                        if (error)
                            return "artifactSelectors." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a DagOutputArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} DagOutputArtifactSpec
             */
            DagOutputArtifactSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec)
                    return object;
                var message = new $root.ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec();
                if (object.artifactSelectors) {
                    if (!Array.isArray(object.artifactSelectors))
                        throw TypeError(".ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.artifactSelectors: array expected");
                    message.artifactSelectors = [];
                    for (var i = 0; i < object.artifactSelectors.length; ++i) {
                        if (typeof object.artifactSelectors[i] !== "object")
                            throw TypeError(".ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.artifactSelectors: object expected");
                        message.artifactSelectors[i] = $root.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.fromObject(object.artifactSelectors[i]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a DagOutputArtifactSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec} message DagOutputArtifactSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            DagOutputArtifactSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.artifactSelectors = [];
                if (message.artifactSelectors && message.artifactSelectors.length) {
                    object.artifactSelectors = [];
                    for (var j = 0; j < message.artifactSelectors.length; ++j)
                        object.artifactSelectors[j] = $root.ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.toObject(message.artifactSelectors[j], options);
                }
                return object;
            };

            /**
             * Converts this DagOutputArtifactSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            DagOutputArtifactSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return DagOutputArtifactSpec;
        })();

        DagOutputsSpec.ParameterSelectorSpec = (function() {

            /**
             * Properties of a ParameterSelectorSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @interface IParameterSelectorSpec
             * @property {string|null} [producerSubtask] ParameterSelectorSpec producerSubtask
             * @property {string|null} [outputParameterKey] ParameterSelectorSpec outputParameterKey
             */

            /**
             * Constructs a new ParameterSelectorSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @classdesc Represents a ParameterSelectorSpec.
             * @implements IParameterSelectorSpec
             * @constructor
             * @param {ml_pipelines.DagOutputsSpec.IParameterSelectorSpec=} [properties] Properties to set
             */
            function ParameterSelectorSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ParameterSelectorSpec producerSubtask.
             * @member {string} producerSubtask
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @instance
             */
            ParameterSelectorSpec.prototype.producerSubtask = "";

            /**
             * ParameterSelectorSpec outputParameterKey.
             * @member {string} outputParameterKey
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @instance
             */
            ParameterSelectorSpec.prototype.outputParameterKey = "";

            /**
             * Creates a new ParameterSelectorSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IParameterSelectorSpec=} [properties] Properties to set
             * @returns {ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} ParameterSelectorSpec instance
             */
            ParameterSelectorSpec.create = function create(properties) {
                return new ParameterSelectorSpec(properties);
            };

            /**
             * Encodes the specified ParameterSelectorSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IParameterSelectorSpec} message ParameterSelectorSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ParameterSelectorSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.producerSubtask != null && Object.hasOwnProperty.call(message, "producerSubtask"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.producerSubtask);
                if (message.outputParameterKey != null && Object.hasOwnProperty.call(message, "outputParameterKey"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.outputParameterKey);
                return writer;
            };

            /**
             * Encodes the specified ParameterSelectorSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IParameterSelectorSpec} message ParameterSelectorSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ParameterSelectorSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ParameterSelectorSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} ParameterSelectorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ParameterSelectorSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.producerSubtask = reader.string();
                        break;
                    case 2:
                        message.outputParameterKey = reader.string();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ParameterSelectorSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} ParameterSelectorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ParameterSelectorSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ParameterSelectorSpec message.
             * @function verify
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ParameterSelectorSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.producerSubtask != null && message.hasOwnProperty("producerSubtask"))
                    if (!$util.isString(message.producerSubtask))
                        return "producerSubtask: string expected";
                if (message.outputParameterKey != null && message.hasOwnProperty("outputParameterKey"))
                    if (!$util.isString(message.outputParameterKey))
                        return "outputParameterKey: string expected";
                return null;
            };

            /**
             * Creates a ParameterSelectorSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} ParameterSelectorSpec
             */
            ParameterSelectorSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec)
                    return object;
                var message = new $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec();
                if (object.producerSubtask != null)
                    message.producerSubtask = String(object.producerSubtask);
                if (object.outputParameterKey != null)
                    message.outputParameterKey = String(object.outputParameterKey);
                return message;
            };

            /**
             * Creates a plain object from a ParameterSelectorSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.ParameterSelectorSpec} message ParameterSelectorSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ParameterSelectorSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    object.producerSubtask = "";
                    object.outputParameterKey = "";
                }
                if (message.producerSubtask != null && message.hasOwnProperty("producerSubtask"))
                    object.producerSubtask = message.producerSubtask;
                if (message.outputParameterKey != null && message.hasOwnProperty("outputParameterKey"))
                    object.outputParameterKey = message.outputParameterKey;
                return object;
            };

            /**
             * Converts this ParameterSelectorSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ParameterSelectorSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ParameterSelectorSpec;
        })();

        DagOutputsSpec.ParameterSelectorsSpec = (function() {

            /**
             * Properties of a ParameterSelectorsSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @interface IParameterSelectorsSpec
             * @property {Array.<ml_pipelines.DagOutputsSpec.IParameterSelectorSpec>|null} [parameterSelectors] ParameterSelectorsSpec parameterSelectors
             */

            /**
             * Constructs a new ParameterSelectorsSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @classdesc Represents a ParameterSelectorsSpec.
             * @implements IParameterSelectorsSpec
             * @constructor
             * @param {ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec=} [properties] Properties to set
             */
            function ParameterSelectorsSpec(properties) {
                this.parameterSelectors = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ParameterSelectorsSpec parameterSelectors.
             * @member {Array.<ml_pipelines.DagOutputsSpec.IParameterSelectorSpec>} parameterSelectors
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @instance
             */
            ParameterSelectorsSpec.prototype.parameterSelectors = $util.emptyArray;

            /**
             * Creates a new ParameterSelectorsSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec=} [properties] Properties to set
             * @returns {ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} ParameterSelectorsSpec instance
             */
            ParameterSelectorsSpec.create = function create(properties) {
                return new ParameterSelectorsSpec(properties);
            };

            /**
             * Encodes the specified ParameterSelectorsSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec} message ParameterSelectorsSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ParameterSelectorsSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.parameterSelectors != null && message.parameterSelectors.length)
                    for (var i = 0; i < message.parameterSelectors.length; ++i)
                        $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.encode(message.parameterSelectors[i], writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ParameterSelectorsSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec} message ParameterSelectorsSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ParameterSelectorsSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ParameterSelectorsSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} ParameterSelectorsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ParameterSelectorsSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        if (!(message.parameterSelectors && message.parameterSelectors.length))
                            message.parameterSelectors = [];
                        message.parameterSelectors.push($root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.decode(reader, reader.uint32()));
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ParameterSelectorsSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} ParameterSelectorsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ParameterSelectorsSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ParameterSelectorsSpec message.
             * @function verify
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ParameterSelectorsSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.parameterSelectors != null && message.hasOwnProperty("parameterSelectors")) {
                    if (!Array.isArray(message.parameterSelectors))
                        return "parameterSelectors: array expected";
                    for (var i = 0; i < message.parameterSelectors.length; ++i) {
                        var error = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.verify(message.parameterSelectors[i]);
                        if (error)
                            return "parameterSelectors." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a ParameterSelectorsSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} ParameterSelectorsSpec
             */
            ParameterSelectorsSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec)
                    return object;
                var message = new $root.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec();
                if (object.parameterSelectors) {
                    if (!Array.isArray(object.parameterSelectors))
                        throw TypeError(".ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.parameterSelectors: array expected");
                    message.parameterSelectors = [];
                    for (var i = 0; i < object.parameterSelectors.length; ++i) {
                        if (typeof object.parameterSelectors[i] !== "object")
                            throw TypeError(".ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.parameterSelectors: object expected");
                        message.parameterSelectors[i] = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.fromObject(object.parameterSelectors[i]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a ParameterSelectorsSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec} message ParameterSelectorsSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ParameterSelectorsSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.parameterSelectors = [];
                if (message.parameterSelectors && message.parameterSelectors.length) {
                    object.parameterSelectors = [];
                    for (var j = 0; j < message.parameterSelectors.length; ++j)
                        object.parameterSelectors[j] = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.toObject(message.parameterSelectors[j], options);
                }
                return object;
            };

            /**
             * Converts this ParameterSelectorsSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ParameterSelectorsSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ParameterSelectorsSpec;
        })();

        DagOutputsSpec.MapParameterSelectorsSpec = (function() {

            /**
             * Properties of a MapParameterSelectorsSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @interface IMapParameterSelectorsSpec
             * @property {Object.<string,ml_pipelines.DagOutputsSpec.IParameterSelectorSpec>|null} [mappedParameters] MapParameterSelectorsSpec mappedParameters
             */

            /**
             * Constructs a new MapParameterSelectorsSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @classdesc Represents a MapParameterSelectorsSpec.
             * @implements IMapParameterSelectorsSpec
             * @constructor
             * @param {ml_pipelines.DagOutputsSpec.IMapParameterSelectorsSpec=} [properties] Properties to set
             */
            function MapParameterSelectorsSpec(properties) {
                this.mappedParameters = {};
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * MapParameterSelectorsSpec mappedParameters.
             * @member {Object.<string,ml_pipelines.DagOutputsSpec.IParameterSelectorSpec>} mappedParameters
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @instance
             */
            MapParameterSelectorsSpec.prototype.mappedParameters = $util.emptyObject;

            /**
             * Creates a new MapParameterSelectorsSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IMapParameterSelectorsSpec=} [properties] Properties to set
             * @returns {ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec} MapParameterSelectorsSpec instance
             */
            MapParameterSelectorsSpec.create = function create(properties) {
                return new MapParameterSelectorsSpec(properties);
            };

            /**
             * Encodes the specified MapParameterSelectorsSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IMapParameterSelectorsSpec} message MapParameterSelectorsSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            MapParameterSelectorsSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.mappedParameters != null && Object.hasOwnProperty.call(message, "mappedParameters"))
                    for (var keys = Object.keys(message.mappedParameters), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.encode(message.mappedParameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                return writer;
            };

            /**
             * Encodes the specified MapParameterSelectorsSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IMapParameterSelectorsSpec} message MapParameterSelectorsSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            MapParameterSelectorsSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a MapParameterSelectorsSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec} MapParameterSelectorsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            MapParameterSelectorsSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec(), key, value;
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 2:
                        if (message.mappedParameters === $util.emptyObject)
                            message.mappedParameters = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.mappedParameters[key] = value;
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a MapParameterSelectorsSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec} MapParameterSelectorsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            MapParameterSelectorsSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a MapParameterSelectorsSpec message.
             * @function verify
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            MapParameterSelectorsSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.mappedParameters != null && message.hasOwnProperty("mappedParameters")) {
                    if (!$util.isObject(message.mappedParameters))
                        return "mappedParameters: object expected";
                    var key = Object.keys(message.mappedParameters);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.verify(message.mappedParameters[key[i]]);
                        if (error)
                            return "mappedParameters." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a MapParameterSelectorsSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec} MapParameterSelectorsSpec
             */
            MapParameterSelectorsSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec)
                    return object;
                var message = new $root.ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec();
                if (object.mappedParameters) {
                    if (typeof object.mappedParameters !== "object")
                        throw TypeError(".ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.mappedParameters: object expected");
                    message.mappedParameters = {};
                    for (var keys = Object.keys(object.mappedParameters), i = 0; i < keys.length; ++i) {
                        if (typeof object.mappedParameters[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.mappedParameters: object expected");
                        message.mappedParameters[keys[i]] = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.fromObject(object.mappedParameters[keys[i]]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a MapParameterSelectorsSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec} message MapParameterSelectorsSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            MapParameterSelectorsSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.objects || options.defaults)
                    object.mappedParameters = {};
                var keys2;
                if (message.mappedParameters && (keys2 = Object.keys(message.mappedParameters)).length) {
                    object.mappedParameters = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.mappedParameters[keys2[j]] = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.toObject(message.mappedParameters[keys2[j]], options);
                }
                return object;
            };

            /**
             * Converts this MapParameterSelectorsSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            MapParameterSelectorsSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return MapParameterSelectorsSpec;
        })();

        DagOutputsSpec.DagOutputParameterSpec = (function() {

            /**
             * Properties of a DagOutputParameterSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @interface IDagOutputParameterSpec
             * @property {ml_pipelines.DagOutputsSpec.IParameterSelectorSpec|null} [valueFromParameter] DagOutputParameterSpec valueFromParameter
             * @property {ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec|null} [valueFromOneof] DagOutputParameterSpec valueFromOneof
             */

            /**
             * Constructs a new DagOutputParameterSpec.
             * @memberof ml_pipelines.DagOutputsSpec
             * @classdesc Represents a DagOutputParameterSpec.
             * @implements IDagOutputParameterSpec
             * @constructor
             * @param {ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec=} [properties] Properties to set
             */
            function DagOutputParameterSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * DagOutputParameterSpec valueFromParameter.
             * @member {ml_pipelines.DagOutputsSpec.IParameterSelectorSpec|null|undefined} valueFromParameter
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @instance
             */
            DagOutputParameterSpec.prototype.valueFromParameter = null;

            /**
             * DagOutputParameterSpec valueFromOneof.
             * @member {ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec|null|undefined} valueFromOneof
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @instance
             */
            DagOutputParameterSpec.prototype.valueFromOneof = null;

            // OneOf field names bound to virtual getters and setters
            var $oneOfFields;

            /**
             * DagOutputParameterSpec kind.
             * @member {"valueFromParameter"|"valueFromOneof"|undefined} kind
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @instance
             */
            Object.defineProperty(DagOutputParameterSpec.prototype, "kind", {
                get: $util.oneOfGetter($oneOfFields = ["valueFromParameter", "valueFromOneof"]),
                set: $util.oneOfSetter($oneOfFields)
            });

            /**
             * Creates a new DagOutputParameterSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec=} [properties] Properties to set
             * @returns {ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} DagOutputParameterSpec instance
             */
            DagOutputParameterSpec.create = function create(properties) {
                return new DagOutputParameterSpec(properties);
            };

            /**
             * Encodes the specified DagOutputParameterSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec} message DagOutputParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            DagOutputParameterSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.valueFromParameter != null && Object.hasOwnProperty.call(message, "valueFromParameter"))
                    $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.encode(message.valueFromParameter, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.valueFromOneof != null && Object.hasOwnProperty.call(message, "valueFromOneof"))
                    $root.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.encode(message.valueFromOneof, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified DagOutputParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec} message DagOutputParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            DagOutputParameterSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a DagOutputParameterSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} DagOutputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            DagOutputParameterSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.valueFromParameter = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.decode(reader, reader.uint32());
                        break;
                    case 2:
                        message.valueFromOneof = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a DagOutputParameterSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} DagOutputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            DagOutputParameterSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a DagOutputParameterSpec message.
             * @function verify
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            DagOutputParameterSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                var properties = {};
                if (message.valueFromParameter != null && message.hasOwnProperty("valueFromParameter")) {
                    properties.kind = 1;
                    {
                        var error = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.verify(message.valueFromParameter);
                        if (error)
                            return "valueFromParameter." + error;
                    }
                }
                if (message.valueFromOneof != null && message.hasOwnProperty("valueFromOneof")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    {
                        var error = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.verify(message.valueFromOneof);
                        if (error)
                            return "valueFromOneof." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a DagOutputParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} DagOutputParameterSpec
             */
            DagOutputParameterSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec)
                    return object;
                var message = new $root.ml_pipelines.DagOutputsSpec.DagOutputParameterSpec();
                if (object.valueFromParameter != null) {
                    if (typeof object.valueFromParameter !== "object")
                        throw TypeError(".ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.valueFromParameter: object expected");
                    message.valueFromParameter = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.fromObject(object.valueFromParameter);
                }
                if (object.valueFromOneof != null) {
                    if (typeof object.valueFromOneof !== "object")
                        throw TypeError(".ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.valueFromOneof: object expected");
                    message.valueFromOneof = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.fromObject(object.valueFromOneof);
                }
                return message;
            };

            /**
             * Creates a plain object from a DagOutputParameterSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @static
             * @param {ml_pipelines.DagOutputsSpec.DagOutputParameterSpec} message DagOutputParameterSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            DagOutputParameterSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (message.valueFromParameter != null && message.hasOwnProperty("valueFromParameter")) {
                    object.valueFromParameter = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.toObject(message.valueFromParameter, options);
                    if (options.oneofs)
                        object.kind = "valueFromParameter";
                }
                if (message.valueFromOneof != null && message.hasOwnProperty("valueFromOneof")) {
                    object.valueFromOneof = $root.ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.toObject(message.valueFromOneof, options);
                    if (options.oneofs)
                        object.kind = "valueFromOneof";
                }
                return object;
            };

            /**
             * Converts this DagOutputParameterSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.DagOutputsSpec.DagOutputParameterSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            DagOutputParameterSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return DagOutputParameterSpec;
        })();

        return DagOutputsSpec;
    })();

    ml_pipelines.ComponentInputsSpec = (function() {

        /**
         * Properties of a ComponentInputsSpec.
         * @memberof ml_pipelines
         * @interface IComponentInputsSpec
         * @property {Object.<string,ml_pipelines.ComponentInputsSpec.IArtifactSpec>|null} [artifacts] ComponentInputsSpec artifacts
         * @property {Object.<string,ml_pipelines.ComponentInputsSpec.IParameterSpec>|null} [parameters] ComponentInputsSpec parameters
         */

        /**
         * Constructs a new ComponentInputsSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a ComponentInputsSpec.
         * @implements IComponentInputsSpec
         * @constructor
         * @param {ml_pipelines.IComponentInputsSpec=} [properties] Properties to set
         */
        function ComponentInputsSpec(properties) {
            this.artifacts = {};
            this.parameters = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ComponentInputsSpec artifacts.
         * @member {Object.<string,ml_pipelines.ComponentInputsSpec.IArtifactSpec>} artifacts
         * @memberof ml_pipelines.ComponentInputsSpec
         * @instance
         */
        ComponentInputsSpec.prototype.artifacts = $util.emptyObject;

        /**
         * ComponentInputsSpec parameters.
         * @member {Object.<string,ml_pipelines.ComponentInputsSpec.IParameterSpec>} parameters
         * @memberof ml_pipelines.ComponentInputsSpec
         * @instance
         */
        ComponentInputsSpec.prototype.parameters = $util.emptyObject;

        /**
         * Creates a new ComponentInputsSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ComponentInputsSpec
         * @static
         * @param {ml_pipelines.IComponentInputsSpec=} [properties] Properties to set
         * @returns {ml_pipelines.ComponentInputsSpec} ComponentInputsSpec instance
         */
        ComponentInputsSpec.create = function create(properties) {
            return new ComponentInputsSpec(properties);
        };

        /**
         * Encodes the specified ComponentInputsSpec message. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ComponentInputsSpec
         * @static
         * @param {ml_pipelines.IComponentInputsSpec} message ComponentInputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ComponentInputsSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.artifacts != null && Object.hasOwnProperty.call(message, "artifacts"))
                for (var keys = Object.keys(message.artifacts), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.ComponentInputsSpec.ArtifactSpec.encode(message.artifacts[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.parameters != null && Object.hasOwnProperty.call(message, "parameters"))
                for (var keys = Object.keys(message.parameters), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.ComponentInputsSpec.ParameterSpec.encode(message.parameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            return writer;
        };

        /**
         * Encodes the specified ComponentInputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ComponentInputsSpec
         * @static
         * @param {ml_pipelines.IComponentInputsSpec} message ComponentInputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ComponentInputsSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a ComponentInputsSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ComponentInputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ComponentInputsSpec} ComponentInputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ComponentInputsSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ComponentInputsSpec(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    if (message.artifacts === $util.emptyObject)
                        message.artifacts = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.ComponentInputsSpec.ArtifactSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.artifacts[key] = value;
                    break;
                case 2:
                    if (message.parameters === $util.emptyObject)
                        message.parameters = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.ComponentInputsSpec.ParameterSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.parameters[key] = value;
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a ComponentInputsSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ComponentInputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ComponentInputsSpec} ComponentInputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ComponentInputsSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a ComponentInputsSpec message.
         * @function verify
         * @memberof ml_pipelines.ComponentInputsSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ComponentInputsSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.artifacts != null && message.hasOwnProperty("artifacts")) {
                if (!$util.isObject(message.artifacts))
                    return "artifacts: object expected";
                var key = Object.keys(message.artifacts);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.ComponentInputsSpec.ArtifactSpec.verify(message.artifacts[key[i]]);
                    if (error)
                        return "artifacts." + error;
                }
            }
            if (message.parameters != null && message.hasOwnProperty("parameters")) {
                if (!$util.isObject(message.parameters))
                    return "parameters: object expected";
                var key = Object.keys(message.parameters);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.ComponentInputsSpec.ParameterSpec.verify(message.parameters[key[i]]);
                    if (error)
                        return "parameters." + error;
                }
            }
            return null;
        };

        /**
         * Creates a ComponentInputsSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ComponentInputsSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ComponentInputsSpec} ComponentInputsSpec
         */
        ComponentInputsSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ComponentInputsSpec)
                return object;
            var message = new $root.ml_pipelines.ComponentInputsSpec();
            if (object.artifacts) {
                if (typeof object.artifacts !== "object")
                    throw TypeError(".ml_pipelines.ComponentInputsSpec.artifacts: object expected");
                message.artifacts = {};
                for (var keys = Object.keys(object.artifacts), i = 0; i < keys.length; ++i) {
                    if (typeof object.artifacts[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.ComponentInputsSpec.artifacts: object expected");
                    message.artifacts[keys[i]] = $root.ml_pipelines.ComponentInputsSpec.ArtifactSpec.fromObject(object.artifacts[keys[i]]);
                }
            }
            if (object.parameters) {
                if (typeof object.parameters !== "object")
                    throw TypeError(".ml_pipelines.ComponentInputsSpec.parameters: object expected");
                message.parameters = {};
                for (var keys = Object.keys(object.parameters), i = 0; i < keys.length; ++i) {
                    if (typeof object.parameters[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.ComponentInputsSpec.parameters: object expected");
                    message.parameters[keys[i]] = $root.ml_pipelines.ComponentInputsSpec.ParameterSpec.fromObject(object.parameters[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a ComponentInputsSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ComponentInputsSpec
         * @static
         * @param {ml_pipelines.ComponentInputsSpec} message ComponentInputsSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ComponentInputsSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults) {
                object.artifacts = {};
                object.parameters = {};
            }
            var keys2;
            if (message.artifacts && (keys2 = Object.keys(message.artifacts)).length) {
                object.artifacts = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.artifacts[keys2[j]] = $root.ml_pipelines.ComponentInputsSpec.ArtifactSpec.toObject(message.artifacts[keys2[j]], options);
            }
            if (message.parameters && (keys2 = Object.keys(message.parameters)).length) {
                object.parameters = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.parameters[keys2[j]] = $root.ml_pipelines.ComponentInputsSpec.ParameterSpec.toObject(message.parameters[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this ComponentInputsSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ComponentInputsSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ComponentInputsSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        ComponentInputsSpec.ArtifactSpec = (function() {

            /**
             * Properties of an ArtifactSpec.
             * @memberof ml_pipelines.ComponentInputsSpec
             * @interface IArtifactSpec
             * @property {ml_pipelines.IArtifactTypeSchema|null} [artifactType] ArtifactSpec artifactType
             */

            /**
             * Constructs a new ArtifactSpec.
             * @memberof ml_pipelines.ComponentInputsSpec
             * @classdesc Represents an ArtifactSpec.
             * @implements IArtifactSpec
             * @constructor
             * @param {ml_pipelines.ComponentInputsSpec.IArtifactSpec=} [properties] Properties to set
             */
            function ArtifactSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ArtifactSpec artifactType.
             * @member {ml_pipelines.IArtifactTypeSchema|null|undefined} artifactType
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @instance
             */
            ArtifactSpec.prototype.artifactType = null;

            /**
             * Creates a new ArtifactSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @static
             * @param {ml_pipelines.ComponentInputsSpec.IArtifactSpec=} [properties] Properties to set
             * @returns {ml_pipelines.ComponentInputsSpec.ArtifactSpec} ArtifactSpec instance
             */
            ArtifactSpec.create = function create(properties) {
                return new ArtifactSpec(properties);
            };

            /**
             * Encodes the specified ArtifactSpec message. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.ArtifactSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @static
             * @param {ml_pipelines.ComponentInputsSpec.IArtifactSpec} message ArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ArtifactSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.artifactType != null && Object.hasOwnProperty.call(message, "artifactType"))
                    $root.ml_pipelines.ArtifactTypeSchema.encode(message.artifactType, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.ArtifactSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @static
             * @param {ml_pipelines.ComponentInputsSpec.IArtifactSpec} message ArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ArtifactSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an ArtifactSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.ComponentInputsSpec.ArtifactSpec} ArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ArtifactSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ComponentInputsSpec.ArtifactSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.artifactType = $root.ml_pipelines.ArtifactTypeSchema.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an ArtifactSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.ComponentInputsSpec.ArtifactSpec} ArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ArtifactSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an ArtifactSpec message.
             * @function verify
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ArtifactSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.artifactType != null && message.hasOwnProperty("artifactType")) {
                    var error = $root.ml_pipelines.ArtifactTypeSchema.verify(message.artifactType);
                    if (error)
                        return "artifactType." + error;
                }
                return null;
            };

            /**
             * Creates an ArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.ComponentInputsSpec.ArtifactSpec} ArtifactSpec
             */
            ArtifactSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.ComponentInputsSpec.ArtifactSpec)
                    return object;
                var message = new $root.ml_pipelines.ComponentInputsSpec.ArtifactSpec();
                if (object.artifactType != null) {
                    if (typeof object.artifactType !== "object")
                        throw TypeError(".ml_pipelines.ComponentInputsSpec.ArtifactSpec.artifactType: object expected");
                    message.artifactType = $root.ml_pipelines.ArtifactTypeSchema.fromObject(object.artifactType);
                }
                return message;
            };

            /**
             * Creates a plain object from an ArtifactSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @static
             * @param {ml_pipelines.ComponentInputsSpec.ArtifactSpec} message ArtifactSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ArtifactSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.artifactType = null;
                if (message.artifactType != null && message.hasOwnProperty("artifactType"))
                    object.artifactType = $root.ml_pipelines.ArtifactTypeSchema.toObject(message.artifactType, options);
                return object;
            };

            /**
             * Converts this ArtifactSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.ComponentInputsSpec.ArtifactSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ArtifactSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ArtifactSpec;
        })();

        ComponentInputsSpec.ParameterSpec = (function() {

            /**
             * Properties of a ParameterSpec.
             * @memberof ml_pipelines.ComponentInputsSpec
             * @interface IParameterSpec
             * @property {ml_pipelines.PrimitiveType.PrimitiveTypeEnum|null} [type] ParameterSpec type
             */

            /**
             * Constructs a new ParameterSpec.
             * @memberof ml_pipelines.ComponentInputsSpec
             * @classdesc Represents a ParameterSpec.
             * @implements IParameterSpec
             * @constructor
             * @param {ml_pipelines.ComponentInputsSpec.IParameterSpec=} [properties] Properties to set
             */
            function ParameterSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ParameterSpec type.
             * @member {ml_pipelines.PrimitiveType.PrimitiveTypeEnum} type
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @instance
             */
            ParameterSpec.prototype.type = 0;

            /**
             * Creates a new ParameterSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @static
             * @param {ml_pipelines.ComponentInputsSpec.IParameterSpec=} [properties] Properties to set
             * @returns {ml_pipelines.ComponentInputsSpec.ParameterSpec} ParameterSpec instance
             */
            ParameterSpec.create = function create(properties) {
                return new ParameterSpec(properties);
            };

            /**
             * Encodes the specified ParameterSpec message. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.ParameterSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @static
             * @param {ml_pipelines.ComponentInputsSpec.IParameterSpec} message ParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ParameterSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.type != null && Object.hasOwnProperty.call(message, "type"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int32(message.type);
                return writer;
            };

            /**
             * Encodes the specified ParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.ParameterSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @static
             * @param {ml_pipelines.ComponentInputsSpec.IParameterSpec} message ParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ParameterSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ParameterSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.ComponentInputsSpec.ParameterSpec} ParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ParameterSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ComponentInputsSpec.ParameterSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.type = reader.int32();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ParameterSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.ComponentInputsSpec.ParameterSpec} ParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ParameterSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ParameterSpec message.
             * @function verify
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ParameterSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.type != null && message.hasOwnProperty("type"))
                    switch (message.type) {
                    default:
                        return "type: enum value expected";
                    case 0:
                    case 1:
                    case 2:
                    case 3:
                        break;
                    }
                return null;
            };

            /**
             * Creates a ParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.ComponentInputsSpec.ParameterSpec} ParameterSpec
             */
            ParameterSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.ComponentInputsSpec.ParameterSpec)
                    return object;
                var message = new $root.ml_pipelines.ComponentInputsSpec.ParameterSpec();
                switch (object.type) {
                case "PRIMITIVE_TYPE_UNSPECIFIED":
                case 0:
                    message.type = 0;
                    break;
                case "INT":
                case 1:
                    message.type = 1;
                    break;
                case "DOUBLE":
                case 2:
                    message.type = 2;
                    break;
                case "STRING":
                case 3:
                    message.type = 3;
                    break;
                }
                return message;
            };

            /**
             * Creates a plain object from a ParameterSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @static
             * @param {ml_pipelines.ComponentInputsSpec.ParameterSpec} message ParameterSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ParameterSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.type = options.enums === String ? "PRIMITIVE_TYPE_UNSPECIFIED" : 0;
                if (message.type != null && message.hasOwnProperty("type"))
                    object.type = options.enums === String ? $root.ml_pipelines.PrimitiveType.PrimitiveTypeEnum[message.type] : message.type;
                return object;
            };

            /**
             * Converts this ParameterSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.ComponentInputsSpec.ParameterSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ParameterSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ParameterSpec;
        })();

        return ComponentInputsSpec;
    })();

    ml_pipelines.ComponentOutputsSpec = (function() {

        /**
         * Properties of a ComponentOutputsSpec.
         * @memberof ml_pipelines
         * @interface IComponentOutputsSpec
         * @property {Object.<string,ml_pipelines.ComponentOutputsSpec.IArtifactSpec>|null} [artifacts] ComponentOutputsSpec artifacts
         * @property {Object.<string,ml_pipelines.ComponentOutputsSpec.IParameterSpec>|null} [parameters] ComponentOutputsSpec parameters
         */

        /**
         * Constructs a new ComponentOutputsSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a ComponentOutputsSpec.
         * @implements IComponentOutputsSpec
         * @constructor
         * @param {ml_pipelines.IComponentOutputsSpec=} [properties] Properties to set
         */
        function ComponentOutputsSpec(properties) {
            this.artifacts = {};
            this.parameters = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ComponentOutputsSpec artifacts.
         * @member {Object.<string,ml_pipelines.ComponentOutputsSpec.IArtifactSpec>} artifacts
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @instance
         */
        ComponentOutputsSpec.prototype.artifacts = $util.emptyObject;

        /**
         * ComponentOutputsSpec parameters.
         * @member {Object.<string,ml_pipelines.ComponentOutputsSpec.IParameterSpec>} parameters
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @instance
         */
        ComponentOutputsSpec.prototype.parameters = $util.emptyObject;

        /**
         * Creates a new ComponentOutputsSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @static
         * @param {ml_pipelines.IComponentOutputsSpec=} [properties] Properties to set
         * @returns {ml_pipelines.ComponentOutputsSpec} ComponentOutputsSpec instance
         */
        ComponentOutputsSpec.create = function create(properties) {
            return new ComponentOutputsSpec(properties);
        };

        /**
         * Encodes the specified ComponentOutputsSpec message. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @static
         * @param {ml_pipelines.IComponentOutputsSpec} message ComponentOutputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ComponentOutputsSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.artifacts != null && Object.hasOwnProperty.call(message, "artifacts"))
                for (var keys = Object.keys(message.artifacts), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.encode(message.artifacts[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.parameters != null && Object.hasOwnProperty.call(message, "parameters"))
                for (var keys = Object.keys(message.parameters), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.ComponentOutputsSpec.ParameterSpec.encode(message.parameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            return writer;
        };

        /**
         * Encodes the specified ComponentOutputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @static
         * @param {ml_pipelines.IComponentOutputsSpec} message ComponentOutputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ComponentOutputsSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a ComponentOutputsSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ComponentOutputsSpec} ComponentOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ComponentOutputsSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ComponentOutputsSpec(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    if (message.artifacts === $util.emptyObject)
                        message.artifacts = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.artifacts[key] = value;
                    break;
                case 2:
                    if (message.parameters === $util.emptyObject)
                        message.parameters = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.ComponentOutputsSpec.ParameterSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.parameters[key] = value;
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a ComponentOutputsSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ComponentOutputsSpec} ComponentOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ComponentOutputsSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a ComponentOutputsSpec message.
         * @function verify
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ComponentOutputsSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.artifacts != null && message.hasOwnProperty("artifacts")) {
                if (!$util.isObject(message.artifacts))
                    return "artifacts: object expected";
                var key = Object.keys(message.artifacts);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.verify(message.artifacts[key[i]]);
                    if (error)
                        return "artifacts." + error;
                }
            }
            if (message.parameters != null && message.hasOwnProperty("parameters")) {
                if (!$util.isObject(message.parameters))
                    return "parameters: object expected";
                var key = Object.keys(message.parameters);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.ComponentOutputsSpec.ParameterSpec.verify(message.parameters[key[i]]);
                    if (error)
                        return "parameters." + error;
                }
            }
            return null;
        };

        /**
         * Creates a ComponentOutputsSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ComponentOutputsSpec} ComponentOutputsSpec
         */
        ComponentOutputsSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ComponentOutputsSpec)
                return object;
            var message = new $root.ml_pipelines.ComponentOutputsSpec();
            if (object.artifacts) {
                if (typeof object.artifacts !== "object")
                    throw TypeError(".ml_pipelines.ComponentOutputsSpec.artifacts: object expected");
                message.artifacts = {};
                for (var keys = Object.keys(object.artifacts), i = 0; i < keys.length; ++i) {
                    if (typeof object.artifacts[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.ComponentOutputsSpec.artifacts: object expected");
                    message.artifacts[keys[i]] = $root.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.fromObject(object.artifacts[keys[i]]);
                }
            }
            if (object.parameters) {
                if (typeof object.parameters !== "object")
                    throw TypeError(".ml_pipelines.ComponentOutputsSpec.parameters: object expected");
                message.parameters = {};
                for (var keys = Object.keys(object.parameters), i = 0; i < keys.length; ++i) {
                    if (typeof object.parameters[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.ComponentOutputsSpec.parameters: object expected");
                    message.parameters[keys[i]] = $root.ml_pipelines.ComponentOutputsSpec.ParameterSpec.fromObject(object.parameters[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a ComponentOutputsSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @static
         * @param {ml_pipelines.ComponentOutputsSpec} message ComponentOutputsSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ComponentOutputsSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults) {
                object.artifacts = {};
                object.parameters = {};
            }
            var keys2;
            if (message.artifacts && (keys2 = Object.keys(message.artifacts)).length) {
                object.artifacts = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.artifacts[keys2[j]] = $root.ml_pipelines.ComponentOutputsSpec.ArtifactSpec.toObject(message.artifacts[keys2[j]], options);
            }
            if (message.parameters && (keys2 = Object.keys(message.parameters)).length) {
                object.parameters = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.parameters[keys2[j]] = $root.ml_pipelines.ComponentOutputsSpec.ParameterSpec.toObject(message.parameters[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this ComponentOutputsSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ComponentOutputsSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ComponentOutputsSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        ComponentOutputsSpec.ArtifactSpec = (function() {

            /**
             * Properties of an ArtifactSpec.
             * @memberof ml_pipelines.ComponentOutputsSpec
             * @interface IArtifactSpec
             * @property {ml_pipelines.IArtifactTypeSchema|null} [artifactType] ArtifactSpec artifactType
             * @property {Object.<string,ml_pipelines.IValueOrRuntimeParameter>|null} [properties] ArtifactSpec properties
             * @property {Object.<string,ml_pipelines.IValueOrRuntimeParameter>|null} [customProperties] ArtifactSpec customProperties
             * @property {google.protobuf.IStruct|null} [metadata] ArtifactSpec metadata
             */

            /**
             * Constructs a new ArtifactSpec.
             * @memberof ml_pipelines.ComponentOutputsSpec
             * @classdesc Represents an ArtifactSpec.
             * @implements IArtifactSpec
             * @constructor
             * @param {ml_pipelines.ComponentOutputsSpec.IArtifactSpec=} [properties] Properties to set
             */
            function ArtifactSpec(properties) {
                this.properties = {};
                this.customProperties = {};
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ArtifactSpec artifactType.
             * @member {ml_pipelines.IArtifactTypeSchema|null|undefined} artifactType
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @instance
             */
            ArtifactSpec.prototype.artifactType = null;

            /**
             * ArtifactSpec properties.
             * @member {Object.<string,ml_pipelines.IValueOrRuntimeParameter>} properties
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @instance
             */
            ArtifactSpec.prototype.properties = $util.emptyObject;

            /**
             * ArtifactSpec customProperties.
             * @member {Object.<string,ml_pipelines.IValueOrRuntimeParameter>} customProperties
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @instance
             */
            ArtifactSpec.prototype.customProperties = $util.emptyObject;

            /**
             * ArtifactSpec metadata.
             * @member {google.protobuf.IStruct|null|undefined} metadata
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @instance
             */
            ArtifactSpec.prototype.metadata = null;

            /**
             * Creates a new ArtifactSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @static
             * @param {ml_pipelines.ComponentOutputsSpec.IArtifactSpec=} [properties] Properties to set
             * @returns {ml_pipelines.ComponentOutputsSpec.ArtifactSpec} ArtifactSpec instance
             */
            ArtifactSpec.create = function create(properties) {
                return new ArtifactSpec(properties);
            };

            /**
             * Encodes the specified ArtifactSpec message. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.ArtifactSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @static
             * @param {ml_pipelines.ComponentOutputsSpec.IArtifactSpec} message ArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ArtifactSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.artifactType != null && Object.hasOwnProperty.call(message, "artifactType"))
                    $root.ml_pipelines.ArtifactTypeSchema.encode(message.artifactType, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.properties != null && Object.hasOwnProperty.call(message, "properties"))
                    for (var keys = Object.keys(message.properties), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.ValueOrRuntimeParameter.encode(message.properties[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                if (message.customProperties != null && Object.hasOwnProperty.call(message, "customProperties"))
                    for (var keys = Object.keys(message.customProperties), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 3, wireType 2 =*/26).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.ValueOrRuntimeParameter.encode(message.customProperties[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                if (message.metadata != null && Object.hasOwnProperty.call(message, "metadata"))
                    $root.google.protobuf.Struct.encode(message.metadata, writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.ArtifactSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @static
             * @param {ml_pipelines.ComponentOutputsSpec.IArtifactSpec} message ArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ArtifactSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an ArtifactSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.ComponentOutputsSpec.ArtifactSpec} ArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ArtifactSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ComponentOutputsSpec.ArtifactSpec(), key, value;
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.artifactType = $root.ml_pipelines.ArtifactTypeSchema.decode(reader, reader.uint32());
                        break;
                    case 2:
                        if (message.properties === $util.emptyObject)
                            message.properties = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.ValueOrRuntimeParameter.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.properties[key] = value;
                        break;
                    case 3:
                        if (message.customProperties === $util.emptyObject)
                            message.customProperties = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.ValueOrRuntimeParameter.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.customProperties[key] = value;
                        break;
                    case 4:
                        message.metadata = $root.google.protobuf.Struct.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an ArtifactSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.ComponentOutputsSpec.ArtifactSpec} ArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ArtifactSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an ArtifactSpec message.
             * @function verify
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ArtifactSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.artifactType != null && message.hasOwnProperty("artifactType")) {
                    var error = $root.ml_pipelines.ArtifactTypeSchema.verify(message.artifactType);
                    if (error)
                        return "artifactType." + error;
                }
                if (message.properties != null && message.hasOwnProperty("properties")) {
                    if (!$util.isObject(message.properties))
                        return "properties: object expected";
                    var key = Object.keys(message.properties);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.ValueOrRuntimeParameter.verify(message.properties[key[i]]);
                        if (error)
                            return "properties." + error;
                    }
                }
                if (message.customProperties != null && message.hasOwnProperty("customProperties")) {
                    if (!$util.isObject(message.customProperties))
                        return "customProperties: object expected";
                    var key = Object.keys(message.customProperties);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.ValueOrRuntimeParameter.verify(message.customProperties[key[i]]);
                        if (error)
                            return "customProperties." + error;
                    }
                }
                if (message.metadata != null && message.hasOwnProperty("metadata")) {
                    var error = $root.google.protobuf.Struct.verify(message.metadata);
                    if (error)
                        return "metadata." + error;
                }
                return null;
            };

            /**
             * Creates an ArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.ComponentOutputsSpec.ArtifactSpec} ArtifactSpec
             */
            ArtifactSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.ComponentOutputsSpec.ArtifactSpec)
                    return object;
                var message = new $root.ml_pipelines.ComponentOutputsSpec.ArtifactSpec();
                if (object.artifactType != null) {
                    if (typeof object.artifactType !== "object")
                        throw TypeError(".ml_pipelines.ComponentOutputsSpec.ArtifactSpec.artifactType: object expected");
                    message.artifactType = $root.ml_pipelines.ArtifactTypeSchema.fromObject(object.artifactType);
                }
                if (object.properties) {
                    if (typeof object.properties !== "object")
                        throw TypeError(".ml_pipelines.ComponentOutputsSpec.ArtifactSpec.properties: object expected");
                    message.properties = {};
                    for (var keys = Object.keys(object.properties), i = 0; i < keys.length; ++i) {
                        if (typeof object.properties[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.ComponentOutputsSpec.ArtifactSpec.properties: object expected");
                        message.properties[keys[i]] = $root.ml_pipelines.ValueOrRuntimeParameter.fromObject(object.properties[keys[i]]);
                    }
                }
                if (object.customProperties) {
                    if (typeof object.customProperties !== "object")
                        throw TypeError(".ml_pipelines.ComponentOutputsSpec.ArtifactSpec.customProperties: object expected");
                    message.customProperties = {};
                    for (var keys = Object.keys(object.customProperties), i = 0; i < keys.length; ++i) {
                        if (typeof object.customProperties[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.ComponentOutputsSpec.ArtifactSpec.customProperties: object expected");
                        message.customProperties[keys[i]] = $root.ml_pipelines.ValueOrRuntimeParameter.fromObject(object.customProperties[keys[i]]);
                    }
                }
                if (object.metadata != null) {
                    if (typeof object.metadata !== "object")
                        throw TypeError(".ml_pipelines.ComponentOutputsSpec.ArtifactSpec.metadata: object expected");
                    message.metadata = $root.google.protobuf.Struct.fromObject(object.metadata);
                }
                return message;
            };

            /**
             * Creates a plain object from an ArtifactSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @static
             * @param {ml_pipelines.ComponentOutputsSpec.ArtifactSpec} message ArtifactSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ArtifactSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.objects || options.defaults) {
                    object.properties = {};
                    object.customProperties = {};
                }
                if (options.defaults) {
                    object.artifactType = null;
                    object.metadata = null;
                }
                if (message.artifactType != null && message.hasOwnProperty("artifactType"))
                    object.artifactType = $root.ml_pipelines.ArtifactTypeSchema.toObject(message.artifactType, options);
                var keys2;
                if (message.properties && (keys2 = Object.keys(message.properties)).length) {
                    object.properties = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.properties[keys2[j]] = $root.ml_pipelines.ValueOrRuntimeParameter.toObject(message.properties[keys2[j]], options);
                }
                if (message.customProperties && (keys2 = Object.keys(message.customProperties)).length) {
                    object.customProperties = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.customProperties[keys2[j]] = $root.ml_pipelines.ValueOrRuntimeParameter.toObject(message.customProperties[keys2[j]], options);
                }
                if (message.metadata != null && message.hasOwnProperty("metadata"))
                    object.metadata = $root.google.protobuf.Struct.toObject(message.metadata, options);
                return object;
            };

            /**
             * Converts this ArtifactSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.ComponentOutputsSpec.ArtifactSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ArtifactSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ArtifactSpec;
        })();

        ComponentOutputsSpec.ParameterSpec = (function() {

            /**
             * Properties of a ParameterSpec.
             * @memberof ml_pipelines.ComponentOutputsSpec
             * @interface IParameterSpec
             * @property {ml_pipelines.PrimitiveType.PrimitiveTypeEnum|null} [type] ParameterSpec type
             */

            /**
             * Constructs a new ParameterSpec.
             * @memberof ml_pipelines.ComponentOutputsSpec
             * @classdesc Represents a ParameterSpec.
             * @implements IParameterSpec
             * @constructor
             * @param {ml_pipelines.ComponentOutputsSpec.IParameterSpec=} [properties] Properties to set
             */
            function ParameterSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ParameterSpec type.
             * @member {ml_pipelines.PrimitiveType.PrimitiveTypeEnum} type
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @instance
             */
            ParameterSpec.prototype.type = 0;

            /**
             * Creates a new ParameterSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @static
             * @param {ml_pipelines.ComponentOutputsSpec.IParameterSpec=} [properties] Properties to set
             * @returns {ml_pipelines.ComponentOutputsSpec.ParameterSpec} ParameterSpec instance
             */
            ParameterSpec.create = function create(properties) {
                return new ParameterSpec(properties);
            };

            /**
             * Encodes the specified ParameterSpec message. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.ParameterSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @static
             * @param {ml_pipelines.ComponentOutputsSpec.IParameterSpec} message ParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ParameterSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.type != null && Object.hasOwnProperty.call(message, "type"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int32(message.type);
                return writer;
            };

            /**
             * Encodes the specified ParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.ParameterSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @static
             * @param {ml_pipelines.ComponentOutputsSpec.IParameterSpec} message ParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ParameterSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ParameterSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.ComponentOutputsSpec.ParameterSpec} ParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ParameterSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ComponentOutputsSpec.ParameterSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.type = reader.int32();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ParameterSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.ComponentOutputsSpec.ParameterSpec} ParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ParameterSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ParameterSpec message.
             * @function verify
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ParameterSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.type != null && message.hasOwnProperty("type"))
                    switch (message.type) {
                    default:
                        return "type: enum value expected";
                    case 0:
                    case 1:
                    case 2:
                    case 3:
                        break;
                    }
                return null;
            };

            /**
             * Creates a ParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.ComponentOutputsSpec.ParameterSpec} ParameterSpec
             */
            ParameterSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.ComponentOutputsSpec.ParameterSpec)
                    return object;
                var message = new $root.ml_pipelines.ComponentOutputsSpec.ParameterSpec();
                switch (object.type) {
                case "PRIMITIVE_TYPE_UNSPECIFIED":
                case 0:
                    message.type = 0;
                    break;
                case "INT":
                case 1:
                    message.type = 1;
                    break;
                case "DOUBLE":
                case 2:
                    message.type = 2;
                    break;
                case "STRING":
                case 3:
                    message.type = 3;
                    break;
                }
                return message;
            };

            /**
             * Creates a plain object from a ParameterSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @static
             * @param {ml_pipelines.ComponentOutputsSpec.ParameterSpec} message ParameterSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ParameterSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.type = options.enums === String ? "PRIMITIVE_TYPE_UNSPECIFIED" : 0;
                if (message.type != null && message.hasOwnProperty("type"))
                    object.type = options.enums === String ? $root.ml_pipelines.PrimitiveType.PrimitiveTypeEnum[message.type] : message.type;
                return object;
            };

            /**
             * Converts this ParameterSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.ComponentOutputsSpec.ParameterSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ParameterSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ParameterSpec;
        })();

        return ComponentOutputsSpec;
    })();

    ml_pipelines.TaskInputsSpec = (function() {

        /**
         * Properties of a TaskInputsSpec.
         * @memberof ml_pipelines
         * @interface ITaskInputsSpec
         * @property {Object.<string,ml_pipelines.TaskInputsSpec.IInputParameterSpec>|null} [parameters] TaskInputsSpec parameters
         * @property {Object.<string,ml_pipelines.TaskInputsSpec.IInputArtifactSpec>|null} [artifacts] TaskInputsSpec artifacts
         */

        /**
         * Constructs a new TaskInputsSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a TaskInputsSpec.
         * @implements ITaskInputsSpec
         * @constructor
         * @param {ml_pipelines.ITaskInputsSpec=} [properties] Properties to set
         */
        function TaskInputsSpec(properties) {
            this.parameters = {};
            this.artifacts = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * TaskInputsSpec parameters.
         * @member {Object.<string,ml_pipelines.TaskInputsSpec.IInputParameterSpec>} parameters
         * @memberof ml_pipelines.TaskInputsSpec
         * @instance
         */
        TaskInputsSpec.prototype.parameters = $util.emptyObject;

        /**
         * TaskInputsSpec artifacts.
         * @member {Object.<string,ml_pipelines.TaskInputsSpec.IInputArtifactSpec>} artifacts
         * @memberof ml_pipelines.TaskInputsSpec
         * @instance
         */
        TaskInputsSpec.prototype.artifacts = $util.emptyObject;

        /**
         * Creates a new TaskInputsSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.TaskInputsSpec
         * @static
         * @param {ml_pipelines.ITaskInputsSpec=} [properties] Properties to set
         * @returns {ml_pipelines.TaskInputsSpec} TaskInputsSpec instance
         */
        TaskInputsSpec.create = function create(properties) {
            return new TaskInputsSpec(properties);
        };

        /**
         * Encodes the specified TaskInputsSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.TaskInputsSpec
         * @static
         * @param {ml_pipelines.ITaskInputsSpec} message TaskInputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        TaskInputsSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.parameters != null && Object.hasOwnProperty.call(message, "parameters"))
                for (var keys = Object.keys(message.parameters), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.encode(message.parameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.artifacts != null && Object.hasOwnProperty.call(message, "artifacts"))
                for (var keys = Object.keys(message.artifacts), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.encode(message.artifacts[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            return writer;
        };

        /**
         * Encodes the specified TaskInputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.TaskInputsSpec
         * @static
         * @param {ml_pipelines.ITaskInputsSpec} message TaskInputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        TaskInputsSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a TaskInputsSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.TaskInputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.TaskInputsSpec} TaskInputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        TaskInputsSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.TaskInputsSpec(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    if (message.parameters === $util.emptyObject)
                        message.parameters = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.parameters[key] = value;
                    break;
                case 2:
                    if (message.artifacts === $util.emptyObject)
                        message.artifacts = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.artifacts[key] = value;
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a TaskInputsSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.TaskInputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.TaskInputsSpec} TaskInputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        TaskInputsSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a TaskInputsSpec message.
         * @function verify
         * @memberof ml_pipelines.TaskInputsSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        TaskInputsSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.parameters != null && message.hasOwnProperty("parameters")) {
                if (!$util.isObject(message.parameters))
                    return "parameters: object expected";
                var key = Object.keys(message.parameters);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.verify(message.parameters[key[i]]);
                    if (error)
                        return "parameters." + error;
                }
            }
            if (message.artifacts != null && message.hasOwnProperty("artifacts")) {
                if (!$util.isObject(message.artifacts))
                    return "artifacts: object expected";
                var key = Object.keys(message.artifacts);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.verify(message.artifacts[key[i]]);
                    if (error)
                        return "artifacts." + error;
                }
            }
            return null;
        };

        /**
         * Creates a TaskInputsSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.TaskInputsSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.TaskInputsSpec} TaskInputsSpec
         */
        TaskInputsSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.TaskInputsSpec)
                return object;
            var message = new $root.ml_pipelines.TaskInputsSpec();
            if (object.parameters) {
                if (typeof object.parameters !== "object")
                    throw TypeError(".ml_pipelines.TaskInputsSpec.parameters: object expected");
                message.parameters = {};
                for (var keys = Object.keys(object.parameters), i = 0; i < keys.length; ++i) {
                    if (typeof object.parameters[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.TaskInputsSpec.parameters: object expected");
                    message.parameters[keys[i]] = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.fromObject(object.parameters[keys[i]]);
                }
            }
            if (object.artifacts) {
                if (typeof object.artifacts !== "object")
                    throw TypeError(".ml_pipelines.TaskInputsSpec.artifacts: object expected");
                message.artifacts = {};
                for (var keys = Object.keys(object.artifacts), i = 0; i < keys.length; ++i) {
                    if (typeof object.artifacts[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.TaskInputsSpec.artifacts: object expected");
                    message.artifacts[keys[i]] = $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.fromObject(object.artifacts[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a TaskInputsSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.TaskInputsSpec
         * @static
         * @param {ml_pipelines.TaskInputsSpec} message TaskInputsSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        TaskInputsSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults) {
                object.parameters = {};
                object.artifacts = {};
            }
            var keys2;
            if (message.parameters && (keys2 = Object.keys(message.parameters)).length) {
                object.parameters = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.parameters[keys2[j]] = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.toObject(message.parameters[keys2[j]], options);
            }
            if (message.artifacts && (keys2 = Object.keys(message.artifacts)).length) {
                object.artifacts = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.artifacts[keys2[j]] = $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.toObject(message.artifacts[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this TaskInputsSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.TaskInputsSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        TaskInputsSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        TaskInputsSpec.InputArtifactSpec = (function() {

            /**
             * Properties of an InputArtifactSpec.
             * @memberof ml_pipelines.TaskInputsSpec
             * @interface IInputArtifactSpec
             * @property {ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec|null} [taskOutputArtifact] InputArtifactSpec taskOutputArtifact
             * @property {string|null} [componentInputArtifact] InputArtifactSpec componentInputArtifact
             */

            /**
             * Constructs a new InputArtifactSpec.
             * @memberof ml_pipelines.TaskInputsSpec
             * @classdesc Represents an InputArtifactSpec.
             * @implements IInputArtifactSpec
             * @constructor
             * @param {ml_pipelines.TaskInputsSpec.IInputArtifactSpec=} [properties] Properties to set
             */
            function InputArtifactSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * InputArtifactSpec taskOutputArtifact.
             * @member {ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec|null|undefined} taskOutputArtifact
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @instance
             */
            InputArtifactSpec.prototype.taskOutputArtifact = null;

            /**
             * InputArtifactSpec componentInputArtifact.
             * @member {string|null|undefined} componentInputArtifact
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @instance
             */
            InputArtifactSpec.prototype.componentInputArtifact = null;

            // OneOf field names bound to virtual getters and setters
            var $oneOfFields;

            /**
             * InputArtifactSpec kind.
             * @member {"taskOutputArtifact"|"componentInputArtifact"|undefined} kind
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @instance
             */
            Object.defineProperty(InputArtifactSpec.prototype, "kind", {
                get: $util.oneOfGetter($oneOfFields = ["taskOutputArtifact", "componentInputArtifact"]),
                set: $util.oneOfSetter($oneOfFields)
            });

            /**
             * Creates a new InputArtifactSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @static
             * @param {ml_pipelines.TaskInputsSpec.IInputArtifactSpec=} [properties] Properties to set
             * @returns {ml_pipelines.TaskInputsSpec.InputArtifactSpec} InputArtifactSpec instance
             */
            InputArtifactSpec.create = function create(properties) {
                return new InputArtifactSpec(properties);
            };

            /**
             * Encodes the specified InputArtifactSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputArtifactSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @static
             * @param {ml_pipelines.TaskInputsSpec.IInputArtifactSpec} message InputArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            InputArtifactSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.taskOutputArtifact != null && Object.hasOwnProperty.call(message, "taskOutputArtifact"))
                    $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.encode(message.taskOutputArtifact, writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.componentInputArtifact != null && Object.hasOwnProperty.call(message, "componentInputArtifact"))
                    writer.uint32(/* id 4, wireType 2 =*/34).string(message.componentInputArtifact);
                return writer;
            };

            /**
             * Encodes the specified InputArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputArtifactSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @static
             * @param {ml_pipelines.TaskInputsSpec.IInputArtifactSpec} message InputArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            InputArtifactSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an InputArtifactSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.TaskInputsSpec.InputArtifactSpec} InputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            InputArtifactSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 3:
                        message.taskOutputArtifact = $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.decode(reader, reader.uint32());
                        break;
                    case 4:
                        message.componentInputArtifact = reader.string();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an InputArtifactSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.TaskInputsSpec.InputArtifactSpec} InputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            InputArtifactSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an InputArtifactSpec message.
             * @function verify
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            InputArtifactSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                var properties = {};
                if (message.taskOutputArtifact != null && message.hasOwnProperty("taskOutputArtifact")) {
                    properties.kind = 1;
                    {
                        var error = $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.verify(message.taskOutputArtifact);
                        if (error)
                            return "taskOutputArtifact." + error;
                    }
                }
                if (message.componentInputArtifact != null && message.hasOwnProperty("componentInputArtifact")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    if (!$util.isString(message.componentInputArtifact))
                        return "componentInputArtifact: string expected";
                }
                return null;
            };

            /**
             * Creates an InputArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.TaskInputsSpec.InputArtifactSpec} InputArtifactSpec
             */
            InputArtifactSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec)
                    return object;
                var message = new $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec();
                if (object.taskOutputArtifact != null) {
                    if (typeof object.taskOutputArtifact !== "object")
                        throw TypeError(".ml_pipelines.TaskInputsSpec.InputArtifactSpec.taskOutputArtifact: object expected");
                    message.taskOutputArtifact = $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.fromObject(object.taskOutputArtifact);
                }
                if (object.componentInputArtifact != null)
                    message.componentInputArtifact = String(object.componentInputArtifact);
                return message;
            };

            /**
             * Creates a plain object from an InputArtifactSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @static
             * @param {ml_pipelines.TaskInputsSpec.InputArtifactSpec} message InputArtifactSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            InputArtifactSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (message.taskOutputArtifact != null && message.hasOwnProperty("taskOutputArtifact")) {
                    object.taskOutputArtifact = $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.toObject(message.taskOutputArtifact, options);
                    if (options.oneofs)
                        object.kind = "taskOutputArtifact";
                }
                if (message.componentInputArtifact != null && message.hasOwnProperty("componentInputArtifact")) {
                    object.componentInputArtifact = message.componentInputArtifact;
                    if (options.oneofs)
                        object.kind = "componentInputArtifact";
                }
                return object;
            };

            /**
             * Converts this InputArtifactSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            InputArtifactSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            InputArtifactSpec.TaskOutputArtifactSpec = (function() {

                /**
                 * Properties of a TaskOutputArtifactSpec.
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
                 * @interface ITaskOutputArtifactSpec
                 * @property {string|null} [producerTask] TaskOutputArtifactSpec producerTask
                 * @property {string|null} [outputArtifactKey] TaskOutputArtifactSpec outputArtifactKey
                 */

                /**
                 * Constructs a new TaskOutputArtifactSpec.
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec
                 * @classdesc Represents a TaskOutputArtifactSpec.
                 * @implements ITaskOutputArtifactSpec
                 * @constructor
                 * @param {ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec=} [properties] Properties to set
                 */
                function TaskOutputArtifactSpec(properties) {
                    if (properties)
                        for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null)
                                this[keys[i]] = properties[keys[i]];
                }

                /**
                 * TaskOutputArtifactSpec producerTask.
                 * @member {string} producerTask
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @instance
                 */
                TaskOutputArtifactSpec.prototype.producerTask = "";

                /**
                 * TaskOutputArtifactSpec outputArtifactKey.
                 * @member {string} outputArtifactKey
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @instance
                 */
                TaskOutputArtifactSpec.prototype.outputArtifactKey = "";

                /**
                 * Creates a new TaskOutputArtifactSpec instance using the specified properties.
                 * @function create
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec=} [properties] Properties to set
                 * @returns {ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} TaskOutputArtifactSpec instance
                 */
                TaskOutputArtifactSpec.create = function create(properties) {
                    return new TaskOutputArtifactSpec(properties);
                };

                /**
                 * Encodes the specified TaskOutputArtifactSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.verify|verify} messages.
                 * @function encode
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec} message TaskOutputArtifactSpec message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                TaskOutputArtifactSpec.encode = function encode(message, writer) {
                    if (!writer)
                        writer = $Writer.create();
                    if (message.producerTask != null && Object.hasOwnProperty.call(message, "producerTask"))
                        writer.uint32(/* id 1, wireType 2 =*/10).string(message.producerTask);
                    if (message.outputArtifactKey != null && Object.hasOwnProperty.call(message, "outputArtifactKey"))
                        writer.uint32(/* id 2, wireType 2 =*/18).string(message.outputArtifactKey);
                    return writer;
                };

                /**
                 * Encodes the specified TaskOutputArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.verify|verify} messages.
                 * @function encodeDelimited
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec} message TaskOutputArtifactSpec message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                TaskOutputArtifactSpec.encodeDelimited = function encodeDelimited(message, writer) {
                    return this.encode(message, writer).ldelim();
                };

                /**
                 * Decodes a TaskOutputArtifactSpec message from the specified reader or buffer.
                 * @function decode
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} TaskOutputArtifactSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                TaskOutputArtifactSpec.decode = function decode(reader, length) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec();
                    while (reader.pos < end) {
                        var tag = reader.uint32();
                        switch (tag >>> 3) {
                        case 1:
                            message.producerTask = reader.string();
                            break;
                        case 2:
                            message.outputArtifactKey = reader.string();
                            break;
                        default:
                            reader.skipType(tag & 7);
                            break;
                        }
                    }
                    return message;
                };

                /**
                 * Decodes a TaskOutputArtifactSpec message from the specified reader or buffer, length delimited.
                 * @function decodeDelimited
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @returns {ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} TaskOutputArtifactSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                TaskOutputArtifactSpec.decodeDelimited = function decodeDelimited(reader) {
                    if (!(reader instanceof $Reader))
                        reader = new $Reader(reader);
                    return this.decode(reader, reader.uint32());
                };

                /**
                 * Verifies a TaskOutputArtifactSpec message.
                 * @function verify
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @static
                 * @param {Object.<string,*>} message Plain object to verify
                 * @returns {string|null} `null` if valid, otherwise the reason why it is not
                 */
                TaskOutputArtifactSpec.verify = function verify(message) {
                    if (typeof message !== "object" || message === null)
                        return "object expected";
                    if (message.producerTask != null && message.hasOwnProperty("producerTask"))
                        if (!$util.isString(message.producerTask))
                            return "producerTask: string expected";
                    if (message.outputArtifactKey != null && message.hasOwnProperty("outputArtifactKey"))
                        if (!$util.isString(message.outputArtifactKey))
                            return "outputArtifactKey: string expected";
                    return null;
                };

                /**
                 * Creates a TaskOutputArtifactSpec message from a plain object. Also converts values to their respective internal types.
                 * @function fromObject
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @static
                 * @param {Object.<string,*>} object Plain object
                 * @returns {ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} TaskOutputArtifactSpec
                 */
                TaskOutputArtifactSpec.fromObject = function fromObject(object) {
                    if (object instanceof $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec)
                        return object;
                    var message = new $root.ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec();
                    if (object.producerTask != null)
                        message.producerTask = String(object.producerTask);
                    if (object.outputArtifactKey != null)
                        message.outputArtifactKey = String(object.outputArtifactKey);
                    return message;
                };

                /**
                 * Creates a plain object from a TaskOutputArtifactSpec message. Also converts values to other types if specified.
                 * @function toObject
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec} message TaskOutputArtifactSpec
                 * @param {$protobuf.IConversionOptions} [options] Conversion options
                 * @returns {Object.<string,*>} Plain object
                 */
                TaskOutputArtifactSpec.toObject = function toObject(message, options) {
                    if (!options)
                        options = {};
                    var object = {};
                    if (options.defaults) {
                        object.producerTask = "";
                        object.outputArtifactKey = "";
                    }
                    if (message.producerTask != null && message.hasOwnProperty("producerTask"))
                        object.producerTask = message.producerTask;
                    if (message.outputArtifactKey != null && message.hasOwnProperty("outputArtifactKey"))
                        object.outputArtifactKey = message.outputArtifactKey;
                    return object;
                };

                /**
                 * Converts this TaskOutputArtifactSpec to JSON.
                 * @function toJSON
                 * @memberof ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec
                 * @instance
                 * @returns {Object.<string,*>} JSON object
                 */
                TaskOutputArtifactSpec.prototype.toJSON = function toJSON() {
                    return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                };

                return TaskOutputArtifactSpec;
            })();

            return InputArtifactSpec;
        })();

        TaskInputsSpec.InputParameterSpec = (function() {

            /**
             * Properties of an InputParameterSpec.
             * @memberof ml_pipelines.TaskInputsSpec
             * @interface IInputParameterSpec
             * @property {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec|null} [taskOutputParameter] InputParameterSpec taskOutputParameter
             * @property {ml_pipelines.IValueOrRuntimeParameter|null} [runtimeValue] InputParameterSpec runtimeValue
             * @property {string|null} [componentInputParameter] InputParameterSpec componentInputParameter
             * @property {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus|null} [taskFinalStatus] InputParameterSpec taskFinalStatus
             * @property {string|null} [parameterExpressionSelector] InputParameterSpec parameterExpressionSelector
             */

            /**
             * Constructs a new InputParameterSpec.
             * @memberof ml_pipelines.TaskInputsSpec
             * @classdesc Represents an InputParameterSpec.
             * @implements IInputParameterSpec
             * @constructor
             * @param {ml_pipelines.TaskInputsSpec.IInputParameterSpec=} [properties] Properties to set
             */
            function InputParameterSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * InputParameterSpec taskOutputParameter.
             * @member {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec|null|undefined} taskOutputParameter
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @instance
             */
            InputParameterSpec.prototype.taskOutputParameter = null;

            /**
             * InputParameterSpec runtimeValue.
             * @member {ml_pipelines.IValueOrRuntimeParameter|null|undefined} runtimeValue
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @instance
             */
            InputParameterSpec.prototype.runtimeValue = null;

            /**
             * InputParameterSpec componentInputParameter.
             * @member {string|null|undefined} componentInputParameter
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @instance
             */
            InputParameterSpec.prototype.componentInputParameter = null;

            /**
             * InputParameterSpec taskFinalStatus.
             * @member {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus|null|undefined} taskFinalStatus
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @instance
             */
            InputParameterSpec.prototype.taskFinalStatus = null;

            /**
             * InputParameterSpec parameterExpressionSelector.
             * @member {string} parameterExpressionSelector
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @instance
             */
            InputParameterSpec.prototype.parameterExpressionSelector = "";

            // OneOf field names bound to virtual getters and setters
            var $oneOfFields;

            /**
             * InputParameterSpec kind.
             * @member {"taskOutputParameter"|"runtimeValue"|"componentInputParameter"|"taskFinalStatus"|undefined} kind
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @instance
             */
            Object.defineProperty(InputParameterSpec.prototype, "kind", {
                get: $util.oneOfGetter($oneOfFields = ["taskOutputParameter", "runtimeValue", "componentInputParameter", "taskFinalStatus"]),
                set: $util.oneOfSetter($oneOfFields)
            });

            /**
             * Creates a new InputParameterSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @static
             * @param {ml_pipelines.TaskInputsSpec.IInputParameterSpec=} [properties] Properties to set
             * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec} InputParameterSpec instance
             */
            InputParameterSpec.create = function create(properties) {
                return new InputParameterSpec(properties);
            };

            /**
             * Encodes the specified InputParameterSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @static
             * @param {ml_pipelines.TaskInputsSpec.IInputParameterSpec} message InputParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            InputParameterSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.taskOutputParameter != null && Object.hasOwnProperty.call(message, "taskOutputParameter"))
                    $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.encode(message.taskOutputParameter, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.runtimeValue != null && Object.hasOwnProperty.call(message, "runtimeValue"))
                    $root.ml_pipelines.ValueOrRuntimeParameter.encode(message.runtimeValue, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.componentInputParameter != null && Object.hasOwnProperty.call(message, "componentInputParameter"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.componentInputParameter);
                if (message.parameterExpressionSelector != null && Object.hasOwnProperty.call(message, "parameterExpressionSelector"))
                    writer.uint32(/* id 4, wireType 2 =*/34).string(message.parameterExpressionSelector);
                if (message.taskFinalStatus != null && Object.hasOwnProperty.call(message, "taskFinalStatus"))
                    $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.encode(message.taskFinalStatus, writer.uint32(/* id 5, wireType 2 =*/42).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified InputParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @static
             * @param {ml_pipelines.TaskInputsSpec.IInputParameterSpec} message InputParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            InputParameterSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an InputParameterSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec} InputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            InputParameterSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.TaskInputsSpec.InputParameterSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.taskOutputParameter = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.decode(reader, reader.uint32());
                        break;
                    case 2:
                        message.runtimeValue = $root.ml_pipelines.ValueOrRuntimeParameter.decode(reader, reader.uint32());
                        break;
                    case 3:
                        message.componentInputParameter = reader.string();
                        break;
                    case 5:
                        message.taskFinalStatus = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.decode(reader, reader.uint32());
                        break;
                    case 4:
                        message.parameterExpressionSelector = reader.string();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an InputParameterSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec} InputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            InputParameterSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an InputParameterSpec message.
             * @function verify
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            InputParameterSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                var properties = {};
                if (message.taskOutputParameter != null && message.hasOwnProperty("taskOutputParameter")) {
                    properties.kind = 1;
                    {
                        var error = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.verify(message.taskOutputParameter);
                        if (error)
                            return "taskOutputParameter." + error;
                    }
                }
                if (message.runtimeValue != null && message.hasOwnProperty("runtimeValue")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    {
                        var error = $root.ml_pipelines.ValueOrRuntimeParameter.verify(message.runtimeValue);
                        if (error)
                            return "runtimeValue." + error;
                    }
                }
                if (message.componentInputParameter != null && message.hasOwnProperty("componentInputParameter")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    if (!$util.isString(message.componentInputParameter))
                        return "componentInputParameter: string expected";
                }
                if (message.taskFinalStatus != null && message.hasOwnProperty("taskFinalStatus")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    {
                        var error = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.verify(message.taskFinalStatus);
                        if (error)
                            return "taskFinalStatus." + error;
                    }
                }
                if (message.parameterExpressionSelector != null && message.hasOwnProperty("parameterExpressionSelector"))
                    if (!$util.isString(message.parameterExpressionSelector))
                        return "parameterExpressionSelector: string expected";
                return null;
            };

            /**
             * Creates an InputParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec} InputParameterSpec
             */
            InputParameterSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.TaskInputsSpec.InputParameterSpec)
                    return object;
                var message = new $root.ml_pipelines.TaskInputsSpec.InputParameterSpec();
                if (object.taskOutputParameter != null) {
                    if (typeof object.taskOutputParameter !== "object")
                        throw TypeError(".ml_pipelines.TaskInputsSpec.InputParameterSpec.taskOutputParameter: object expected");
                    message.taskOutputParameter = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.fromObject(object.taskOutputParameter);
                }
                if (object.runtimeValue != null) {
                    if (typeof object.runtimeValue !== "object")
                        throw TypeError(".ml_pipelines.TaskInputsSpec.InputParameterSpec.runtimeValue: object expected");
                    message.runtimeValue = $root.ml_pipelines.ValueOrRuntimeParameter.fromObject(object.runtimeValue);
                }
                if (object.componentInputParameter != null)
                    message.componentInputParameter = String(object.componentInputParameter);
                if (object.taskFinalStatus != null) {
                    if (typeof object.taskFinalStatus !== "object")
                        throw TypeError(".ml_pipelines.TaskInputsSpec.InputParameterSpec.taskFinalStatus: object expected");
                    message.taskFinalStatus = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.fromObject(object.taskFinalStatus);
                }
                if (object.parameterExpressionSelector != null)
                    message.parameterExpressionSelector = String(object.parameterExpressionSelector);
                return message;
            };

            /**
             * Creates a plain object from an InputParameterSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @static
             * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec} message InputParameterSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            InputParameterSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.parameterExpressionSelector = "";
                if (message.taskOutputParameter != null && message.hasOwnProperty("taskOutputParameter")) {
                    object.taskOutputParameter = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.toObject(message.taskOutputParameter, options);
                    if (options.oneofs)
                        object.kind = "taskOutputParameter";
                }
                if (message.runtimeValue != null && message.hasOwnProperty("runtimeValue")) {
                    object.runtimeValue = $root.ml_pipelines.ValueOrRuntimeParameter.toObject(message.runtimeValue, options);
                    if (options.oneofs)
                        object.kind = "runtimeValue";
                }
                if (message.componentInputParameter != null && message.hasOwnProperty("componentInputParameter")) {
                    object.componentInputParameter = message.componentInputParameter;
                    if (options.oneofs)
                        object.kind = "componentInputParameter";
                }
                if (message.parameterExpressionSelector != null && message.hasOwnProperty("parameterExpressionSelector"))
                    object.parameterExpressionSelector = message.parameterExpressionSelector;
                if (message.taskFinalStatus != null && message.hasOwnProperty("taskFinalStatus")) {
                    object.taskFinalStatus = $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.toObject(message.taskFinalStatus, options);
                    if (options.oneofs)
                        object.kind = "taskFinalStatus";
                }
                return object;
            };

            /**
             * Converts this InputParameterSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            InputParameterSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            InputParameterSpec.TaskOutputParameterSpec = (function() {

                /**
                 * Properties of a TaskOutputParameterSpec.
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
                 * @interface ITaskOutputParameterSpec
                 * @property {string|null} [producerTask] TaskOutputParameterSpec producerTask
                 * @property {string|null} [outputParameterKey] TaskOutputParameterSpec outputParameterKey
                 */

                /**
                 * Constructs a new TaskOutputParameterSpec.
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
                 * @classdesc Represents a TaskOutputParameterSpec.
                 * @implements ITaskOutputParameterSpec
                 * @constructor
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec=} [properties] Properties to set
                 */
                function TaskOutputParameterSpec(properties) {
                    if (properties)
                        for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null)
                                this[keys[i]] = properties[keys[i]];
                }

                /**
                 * TaskOutputParameterSpec producerTask.
                 * @member {string} producerTask
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @instance
                 */
                TaskOutputParameterSpec.prototype.producerTask = "";

                /**
                 * TaskOutputParameterSpec outputParameterKey.
                 * @member {string} outputParameterKey
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @instance
                 */
                TaskOutputParameterSpec.prototype.outputParameterKey = "";

                /**
                 * Creates a new TaskOutputParameterSpec instance using the specified properties.
                 * @function create
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec=} [properties] Properties to set
                 * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} TaskOutputParameterSpec instance
                 */
                TaskOutputParameterSpec.create = function create(properties) {
                    return new TaskOutputParameterSpec(properties);
                };

                /**
                 * Encodes the specified TaskOutputParameterSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.verify|verify} messages.
                 * @function encode
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec} message TaskOutputParameterSpec message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                TaskOutputParameterSpec.encode = function encode(message, writer) {
                    if (!writer)
                        writer = $Writer.create();
                    if (message.producerTask != null && Object.hasOwnProperty.call(message, "producerTask"))
                        writer.uint32(/* id 1, wireType 2 =*/10).string(message.producerTask);
                    if (message.outputParameterKey != null && Object.hasOwnProperty.call(message, "outputParameterKey"))
                        writer.uint32(/* id 2, wireType 2 =*/18).string(message.outputParameterKey);
                    return writer;
                };

                /**
                 * Encodes the specified TaskOutputParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.verify|verify} messages.
                 * @function encodeDelimited
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec} message TaskOutputParameterSpec message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                TaskOutputParameterSpec.encodeDelimited = function encodeDelimited(message, writer) {
                    return this.encode(message, writer).ldelim();
                };

                /**
                 * Decodes a TaskOutputParameterSpec message from the specified reader or buffer.
                 * @function decode
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} TaskOutputParameterSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                TaskOutputParameterSpec.decode = function decode(reader, length) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec();
                    while (reader.pos < end) {
                        var tag = reader.uint32();
                        switch (tag >>> 3) {
                        case 1:
                            message.producerTask = reader.string();
                            break;
                        case 2:
                            message.outputParameterKey = reader.string();
                            break;
                        default:
                            reader.skipType(tag & 7);
                            break;
                        }
                    }
                    return message;
                };

                /**
                 * Decodes a TaskOutputParameterSpec message from the specified reader or buffer, length delimited.
                 * @function decodeDelimited
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} TaskOutputParameterSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                TaskOutputParameterSpec.decodeDelimited = function decodeDelimited(reader) {
                    if (!(reader instanceof $Reader))
                        reader = new $Reader(reader);
                    return this.decode(reader, reader.uint32());
                };

                /**
                 * Verifies a TaskOutputParameterSpec message.
                 * @function verify
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @static
                 * @param {Object.<string,*>} message Plain object to verify
                 * @returns {string|null} `null` if valid, otherwise the reason why it is not
                 */
                TaskOutputParameterSpec.verify = function verify(message) {
                    if (typeof message !== "object" || message === null)
                        return "object expected";
                    if (message.producerTask != null && message.hasOwnProperty("producerTask"))
                        if (!$util.isString(message.producerTask))
                            return "producerTask: string expected";
                    if (message.outputParameterKey != null && message.hasOwnProperty("outputParameterKey"))
                        if (!$util.isString(message.outputParameterKey))
                            return "outputParameterKey: string expected";
                    return null;
                };

                /**
                 * Creates a TaskOutputParameterSpec message from a plain object. Also converts values to their respective internal types.
                 * @function fromObject
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @static
                 * @param {Object.<string,*>} object Plain object
                 * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} TaskOutputParameterSpec
                 */
                TaskOutputParameterSpec.fromObject = function fromObject(object) {
                    if (object instanceof $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec)
                        return object;
                    var message = new $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec();
                    if (object.producerTask != null)
                        message.producerTask = String(object.producerTask);
                    if (object.outputParameterKey != null)
                        message.outputParameterKey = String(object.outputParameterKey);
                    return message;
                };

                /**
                 * Creates a plain object from a TaskOutputParameterSpec message. Also converts values to other types if specified.
                 * @function toObject
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec} message TaskOutputParameterSpec
                 * @param {$protobuf.IConversionOptions} [options] Conversion options
                 * @returns {Object.<string,*>} Plain object
                 */
                TaskOutputParameterSpec.toObject = function toObject(message, options) {
                    if (!options)
                        options = {};
                    var object = {};
                    if (options.defaults) {
                        object.producerTask = "";
                        object.outputParameterKey = "";
                    }
                    if (message.producerTask != null && message.hasOwnProperty("producerTask"))
                        object.producerTask = message.producerTask;
                    if (message.outputParameterKey != null && message.hasOwnProperty("outputParameterKey"))
                        object.outputParameterKey = message.outputParameterKey;
                    return object;
                };

                /**
                 * Converts this TaskOutputParameterSpec to JSON.
                 * @function toJSON
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec
                 * @instance
                 * @returns {Object.<string,*>} JSON object
                 */
                TaskOutputParameterSpec.prototype.toJSON = function toJSON() {
                    return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                };

                return TaskOutputParameterSpec;
            })();

            InputParameterSpec.TaskFinalStatus = (function() {

                /**
                 * Properties of a TaskFinalStatus.
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
                 * @interface ITaskFinalStatus
                 * @property {string|null} [producerTask] TaskFinalStatus producerTask
                 */

                /**
                 * Constructs a new TaskFinalStatus.
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec
                 * @classdesc Represents a TaskFinalStatus.
                 * @implements ITaskFinalStatus
                 * @constructor
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus=} [properties] Properties to set
                 */
                function TaskFinalStatus(properties) {
                    if (properties)
                        for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null)
                                this[keys[i]] = properties[keys[i]];
                }

                /**
                 * TaskFinalStatus producerTask.
                 * @member {string} producerTask
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @instance
                 */
                TaskFinalStatus.prototype.producerTask = "";

                /**
                 * Creates a new TaskFinalStatus instance using the specified properties.
                 * @function create
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus=} [properties] Properties to set
                 * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} TaskFinalStatus instance
                 */
                TaskFinalStatus.create = function create(properties) {
                    return new TaskFinalStatus(properties);
                };

                /**
                 * Encodes the specified TaskFinalStatus message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.verify|verify} messages.
                 * @function encode
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus} message TaskFinalStatus message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                TaskFinalStatus.encode = function encode(message, writer) {
                    if (!writer)
                        writer = $Writer.create();
                    if (message.producerTask != null && Object.hasOwnProperty.call(message, "producerTask"))
                        writer.uint32(/* id 1, wireType 2 =*/10).string(message.producerTask);
                    return writer;
                };

                /**
                 * Encodes the specified TaskFinalStatus message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.verify|verify} messages.
                 * @function encodeDelimited
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus} message TaskFinalStatus message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                TaskFinalStatus.encodeDelimited = function encodeDelimited(message, writer) {
                    return this.encode(message, writer).ldelim();
                };

                /**
                 * Decodes a TaskFinalStatus message from the specified reader or buffer.
                 * @function decode
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} TaskFinalStatus
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                TaskFinalStatus.decode = function decode(reader, length) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus();
                    while (reader.pos < end) {
                        var tag = reader.uint32();
                        switch (tag >>> 3) {
                        case 1:
                            message.producerTask = reader.string();
                            break;
                        default:
                            reader.skipType(tag & 7);
                            break;
                        }
                    }
                    return message;
                };

                /**
                 * Decodes a TaskFinalStatus message from the specified reader or buffer, length delimited.
                 * @function decodeDelimited
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} TaskFinalStatus
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                TaskFinalStatus.decodeDelimited = function decodeDelimited(reader) {
                    if (!(reader instanceof $Reader))
                        reader = new $Reader(reader);
                    return this.decode(reader, reader.uint32());
                };

                /**
                 * Verifies a TaskFinalStatus message.
                 * @function verify
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @static
                 * @param {Object.<string,*>} message Plain object to verify
                 * @returns {string|null} `null` if valid, otherwise the reason why it is not
                 */
                TaskFinalStatus.verify = function verify(message) {
                    if (typeof message !== "object" || message === null)
                        return "object expected";
                    if (message.producerTask != null && message.hasOwnProperty("producerTask"))
                        if (!$util.isString(message.producerTask))
                            return "producerTask: string expected";
                    return null;
                };

                /**
                 * Creates a TaskFinalStatus message from a plain object. Also converts values to their respective internal types.
                 * @function fromObject
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @static
                 * @param {Object.<string,*>} object Plain object
                 * @returns {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} TaskFinalStatus
                 */
                TaskFinalStatus.fromObject = function fromObject(object) {
                    if (object instanceof $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus)
                        return object;
                    var message = new $root.ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus();
                    if (object.producerTask != null)
                        message.producerTask = String(object.producerTask);
                    return message;
                };

                /**
                 * Creates a plain object from a TaskFinalStatus message. Also converts values to other types if specified.
                 * @function toObject
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @static
                 * @param {ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus} message TaskFinalStatus
                 * @param {$protobuf.IConversionOptions} [options] Conversion options
                 * @returns {Object.<string,*>} Plain object
                 */
                TaskFinalStatus.toObject = function toObject(message, options) {
                    if (!options)
                        options = {};
                    var object = {};
                    if (options.defaults)
                        object.producerTask = "";
                    if (message.producerTask != null && message.hasOwnProperty("producerTask"))
                        object.producerTask = message.producerTask;
                    return object;
                };

                /**
                 * Converts this TaskFinalStatus to JSON.
                 * @function toJSON
                 * @memberof ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus
                 * @instance
                 * @returns {Object.<string,*>} JSON object
                 */
                TaskFinalStatus.prototype.toJSON = function toJSON() {
                    return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                };

                return TaskFinalStatus;
            })();

            return InputParameterSpec;
        })();

        return TaskInputsSpec;
    })();

    ml_pipelines.TaskOutputsSpec = (function() {

        /**
         * Properties of a TaskOutputsSpec.
         * @memberof ml_pipelines
         * @interface ITaskOutputsSpec
         * @property {Object.<string,ml_pipelines.TaskOutputsSpec.IOutputParameterSpec>|null} [parameters] TaskOutputsSpec parameters
         * @property {Object.<string,ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec>|null} [artifacts] TaskOutputsSpec artifacts
         */

        /**
         * Constructs a new TaskOutputsSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a TaskOutputsSpec.
         * @implements ITaskOutputsSpec
         * @constructor
         * @param {ml_pipelines.ITaskOutputsSpec=} [properties] Properties to set
         */
        function TaskOutputsSpec(properties) {
            this.parameters = {};
            this.artifacts = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * TaskOutputsSpec parameters.
         * @member {Object.<string,ml_pipelines.TaskOutputsSpec.IOutputParameterSpec>} parameters
         * @memberof ml_pipelines.TaskOutputsSpec
         * @instance
         */
        TaskOutputsSpec.prototype.parameters = $util.emptyObject;

        /**
         * TaskOutputsSpec artifacts.
         * @member {Object.<string,ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec>} artifacts
         * @memberof ml_pipelines.TaskOutputsSpec
         * @instance
         */
        TaskOutputsSpec.prototype.artifacts = $util.emptyObject;

        /**
         * Creates a new TaskOutputsSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.TaskOutputsSpec
         * @static
         * @param {ml_pipelines.ITaskOutputsSpec=} [properties] Properties to set
         * @returns {ml_pipelines.TaskOutputsSpec} TaskOutputsSpec instance
         */
        TaskOutputsSpec.create = function create(properties) {
            return new TaskOutputsSpec(properties);
        };

        /**
         * Encodes the specified TaskOutputsSpec message. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.TaskOutputsSpec
         * @static
         * @param {ml_pipelines.ITaskOutputsSpec} message TaskOutputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        TaskOutputsSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.parameters != null && Object.hasOwnProperty.call(message, "parameters"))
                for (var keys = Object.keys(message.parameters), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.encode(message.parameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.artifacts != null && Object.hasOwnProperty.call(message, "artifacts"))
                for (var keys = Object.keys(message.artifacts), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.encode(message.artifacts[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            return writer;
        };

        /**
         * Encodes the specified TaskOutputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.TaskOutputsSpec
         * @static
         * @param {ml_pipelines.ITaskOutputsSpec} message TaskOutputsSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        TaskOutputsSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a TaskOutputsSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.TaskOutputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.TaskOutputsSpec} TaskOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        TaskOutputsSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.TaskOutputsSpec(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    if (message.parameters === $util.emptyObject)
                        message.parameters = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.parameters[key] = value;
                    break;
                case 2:
                    if (message.artifacts === $util.emptyObject)
                        message.artifacts = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.artifacts[key] = value;
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a TaskOutputsSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.TaskOutputsSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.TaskOutputsSpec} TaskOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        TaskOutputsSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a TaskOutputsSpec message.
         * @function verify
         * @memberof ml_pipelines.TaskOutputsSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        TaskOutputsSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.parameters != null && message.hasOwnProperty("parameters")) {
                if (!$util.isObject(message.parameters))
                    return "parameters: object expected";
                var key = Object.keys(message.parameters);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.verify(message.parameters[key[i]]);
                    if (error)
                        return "parameters." + error;
                }
            }
            if (message.artifacts != null && message.hasOwnProperty("artifacts")) {
                if (!$util.isObject(message.artifacts))
                    return "artifacts: object expected";
                var key = Object.keys(message.artifacts);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.verify(message.artifacts[key[i]]);
                    if (error)
                        return "artifacts." + error;
                }
            }
            return null;
        };

        /**
         * Creates a TaskOutputsSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.TaskOutputsSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.TaskOutputsSpec} TaskOutputsSpec
         */
        TaskOutputsSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.TaskOutputsSpec)
                return object;
            var message = new $root.ml_pipelines.TaskOutputsSpec();
            if (object.parameters) {
                if (typeof object.parameters !== "object")
                    throw TypeError(".ml_pipelines.TaskOutputsSpec.parameters: object expected");
                message.parameters = {};
                for (var keys = Object.keys(object.parameters), i = 0; i < keys.length; ++i) {
                    if (typeof object.parameters[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.TaskOutputsSpec.parameters: object expected");
                    message.parameters[keys[i]] = $root.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.fromObject(object.parameters[keys[i]]);
                }
            }
            if (object.artifacts) {
                if (typeof object.artifacts !== "object")
                    throw TypeError(".ml_pipelines.TaskOutputsSpec.artifacts: object expected");
                message.artifacts = {};
                for (var keys = Object.keys(object.artifacts), i = 0; i < keys.length; ++i) {
                    if (typeof object.artifacts[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.TaskOutputsSpec.artifacts: object expected");
                    message.artifacts[keys[i]] = $root.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.fromObject(object.artifacts[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a TaskOutputsSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.TaskOutputsSpec
         * @static
         * @param {ml_pipelines.TaskOutputsSpec} message TaskOutputsSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        TaskOutputsSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults) {
                object.parameters = {};
                object.artifacts = {};
            }
            var keys2;
            if (message.parameters && (keys2 = Object.keys(message.parameters)).length) {
                object.parameters = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.parameters[keys2[j]] = $root.ml_pipelines.TaskOutputsSpec.OutputParameterSpec.toObject(message.parameters[keys2[j]], options);
            }
            if (message.artifacts && (keys2 = Object.keys(message.artifacts)).length) {
                object.artifacts = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.artifacts[keys2[j]] = $root.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.toObject(message.artifacts[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this TaskOutputsSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.TaskOutputsSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        TaskOutputsSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        TaskOutputsSpec.OutputArtifactSpec = (function() {

            /**
             * Properties of an OutputArtifactSpec.
             * @memberof ml_pipelines.TaskOutputsSpec
             * @interface IOutputArtifactSpec
             * @property {ml_pipelines.IArtifactTypeSchema|null} [artifactType] OutputArtifactSpec artifactType
             * @property {Object.<string,ml_pipelines.IValueOrRuntimeParameter>|null} [properties] OutputArtifactSpec properties
             * @property {Object.<string,ml_pipelines.IValueOrRuntimeParameter>|null} [customProperties] OutputArtifactSpec customProperties
             */

            /**
             * Constructs a new OutputArtifactSpec.
             * @memberof ml_pipelines.TaskOutputsSpec
             * @classdesc Represents an OutputArtifactSpec.
             * @implements IOutputArtifactSpec
             * @constructor
             * @param {ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec=} [properties] Properties to set
             */
            function OutputArtifactSpec(properties) {
                this.properties = {};
                this.customProperties = {};
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * OutputArtifactSpec artifactType.
             * @member {ml_pipelines.IArtifactTypeSchema|null|undefined} artifactType
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @instance
             */
            OutputArtifactSpec.prototype.artifactType = null;

            /**
             * OutputArtifactSpec properties.
             * @member {Object.<string,ml_pipelines.IValueOrRuntimeParameter>} properties
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @instance
             */
            OutputArtifactSpec.prototype.properties = $util.emptyObject;

            /**
             * OutputArtifactSpec customProperties.
             * @member {Object.<string,ml_pipelines.IValueOrRuntimeParameter>} customProperties
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @instance
             */
            OutputArtifactSpec.prototype.customProperties = $util.emptyObject;

            /**
             * Creates a new OutputArtifactSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @static
             * @param {ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec=} [properties] Properties to set
             * @returns {ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} OutputArtifactSpec instance
             */
            OutputArtifactSpec.create = function create(properties) {
                return new OutputArtifactSpec(properties);
            };

            /**
             * Encodes the specified OutputArtifactSpec message. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @static
             * @param {ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec} message OutputArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OutputArtifactSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.artifactType != null && Object.hasOwnProperty.call(message, "artifactType"))
                    $root.ml_pipelines.ArtifactTypeSchema.encode(message.artifactType, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.properties != null && Object.hasOwnProperty.call(message, "properties"))
                    for (var keys = Object.keys(message.properties), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.ValueOrRuntimeParameter.encode(message.properties[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                if (message.customProperties != null && Object.hasOwnProperty.call(message, "customProperties"))
                    for (var keys = Object.keys(message.customProperties), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 3, wireType 2 =*/26).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.ValueOrRuntimeParameter.encode(message.customProperties[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                return writer;
            };

            /**
             * Encodes the specified OutputArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @static
             * @param {ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec} message OutputArtifactSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OutputArtifactSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an OutputArtifactSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} OutputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OutputArtifactSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec(), key, value;
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.artifactType = $root.ml_pipelines.ArtifactTypeSchema.decode(reader, reader.uint32());
                        break;
                    case 2:
                        if (message.properties === $util.emptyObject)
                            message.properties = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.ValueOrRuntimeParameter.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.properties[key] = value;
                        break;
                    case 3:
                        if (message.customProperties === $util.emptyObject)
                            message.customProperties = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.ValueOrRuntimeParameter.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.customProperties[key] = value;
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an OutputArtifactSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} OutputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OutputArtifactSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an OutputArtifactSpec message.
             * @function verify
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            OutputArtifactSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.artifactType != null && message.hasOwnProperty("artifactType")) {
                    var error = $root.ml_pipelines.ArtifactTypeSchema.verify(message.artifactType);
                    if (error)
                        return "artifactType." + error;
                }
                if (message.properties != null && message.hasOwnProperty("properties")) {
                    if (!$util.isObject(message.properties))
                        return "properties: object expected";
                    var key = Object.keys(message.properties);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.ValueOrRuntimeParameter.verify(message.properties[key[i]]);
                        if (error)
                            return "properties." + error;
                    }
                }
                if (message.customProperties != null && message.hasOwnProperty("customProperties")) {
                    if (!$util.isObject(message.customProperties))
                        return "customProperties: object expected";
                    var key = Object.keys(message.customProperties);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.ValueOrRuntimeParameter.verify(message.customProperties[key[i]]);
                        if (error)
                            return "customProperties." + error;
                    }
                }
                return null;
            };

            /**
             * Creates an OutputArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} OutputArtifactSpec
             */
            OutputArtifactSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec)
                    return object;
                var message = new $root.ml_pipelines.TaskOutputsSpec.OutputArtifactSpec();
                if (object.artifactType != null) {
                    if (typeof object.artifactType !== "object")
                        throw TypeError(".ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.artifactType: object expected");
                    message.artifactType = $root.ml_pipelines.ArtifactTypeSchema.fromObject(object.artifactType);
                }
                if (object.properties) {
                    if (typeof object.properties !== "object")
                        throw TypeError(".ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.properties: object expected");
                    message.properties = {};
                    for (var keys = Object.keys(object.properties), i = 0; i < keys.length; ++i) {
                        if (typeof object.properties[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.properties: object expected");
                        message.properties[keys[i]] = $root.ml_pipelines.ValueOrRuntimeParameter.fromObject(object.properties[keys[i]]);
                    }
                }
                if (object.customProperties) {
                    if (typeof object.customProperties !== "object")
                        throw TypeError(".ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.customProperties: object expected");
                    message.customProperties = {};
                    for (var keys = Object.keys(object.customProperties), i = 0; i < keys.length; ++i) {
                        if (typeof object.customProperties[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.customProperties: object expected");
                        message.customProperties[keys[i]] = $root.ml_pipelines.ValueOrRuntimeParameter.fromObject(object.customProperties[keys[i]]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from an OutputArtifactSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @static
             * @param {ml_pipelines.TaskOutputsSpec.OutputArtifactSpec} message OutputArtifactSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            OutputArtifactSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.objects || options.defaults) {
                    object.properties = {};
                    object.customProperties = {};
                }
                if (options.defaults)
                    object.artifactType = null;
                if (message.artifactType != null && message.hasOwnProperty("artifactType"))
                    object.artifactType = $root.ml_pipelines.ArtifactTypeSchema.toObject(message.artifactType, options);
                var keys2;
                if (message.properties && (keys2 = Object.keys(message.properties)).length) {
                    object.properties = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.properties[keys2[j]] = $root.ml_pipelines.ValueOrRuntimeParameter.toObject(message.properties[keys2[j]], options);
                }
                if (message.customProperties && (keys2 = Object.keys(message.customProperties)).length) {
                    object.customProperties = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.customProperties[keys2[j]] = $root.ml_pipelines.ValueOrRuntimeParameter.toObject(message.customProperties[keys2[j]], options);
                }
                return object;
            };

            /**
             * Converts this OutputArtifactSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.TaskOutputsSpec.OutputArtifactSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            OutputArtifactSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return OutputArtifactSpec;
        })();

        TaskOutputsSpec.OutputParameterSpec = (function() {

            /**
             * Properties of an OutputParameterSpec.
             * @memberof ml_pipelines.TaskOutputsSpec
             * @interface IOutputParameterSpec
             * @property {ml_pipelines.PrimitiveType.PrimitiveTypeEnum|null} [type] OutputParameterSpec type
             */

            /**
             * Constructs a new OutputParameterSpec.
             * @memberof ml_pipelines.TaskOutputsSpec
             * @classdesc Represents an OutputParameterSpec.
             * @implements IOutputParameterSpec
             * @constructor
             * @param {ml_pipelines.TaskOutputsSpec.IOutputParameterSpec=} [properties] Properties to set
             */
            function OutputParameterSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * OutputParameterSpec type.
             * @member {ml_pipelines.PrimitiveType.PrimitiveTypeEnum} type
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @instance
             */
            OutputParameterSpec.prototype.type = 0;

            /**
             * Creates a new OutputParameterSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @static
             * @param {ml_pipelines.TaskOutputsSpec.IOutputParameterSpec=} [properties] Properties to set
             * @returns {ml_pipelines.TaskOutputsSpec.OutputParameterSpec} OutputParameterSpec instance
             */
            OutputParameterSpec.create = function create(properties) {
                return new OutputParameterSpec(properties);
            };

            /**
             * Encodes the specified OutputParameterSpec message. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.OutputParameterSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @static
             * @param {ml_pipelines.TaskOutputsSpec.IOutputParameterSpec} message OutputParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OutputParameterSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.type != null && Object.hasOwnProperty.call(message, "type"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int32(message.type);
                return writer;
            };

            /**
             * Encodes the specified OutputParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.OutputParameterSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @static
             * @param {ml_pipelines.TaskOutputsSpec.IOutputParameterSpec} message OutputParameterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OutputParameterSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an OutputParameterSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.TaskOutputsSpec.OutputParameterSpec} OutputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OutputParameterSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.TaskOutputsSpec.OutputParameterSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.type = reader.int32();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an OutputParameterSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.TaskOutputsSpec.OutputParameterSpec} OutputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OutputParameterSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an OutputParameterSpec message.
             * @function verify
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            OutputParameterSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.type != null && message.hasOwnProperty("type"))
                    switch (message.type) {
                    default:
                        return "type: enum value expected";
                    case 0:
                    case 1:
                    case 2:
                    case 3:
                        break;
                    }
                return null;
            };

            /**
             * Creates an OutputParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.TaskOutputsSpec.OutputParameterSpec} OutputParameterSpec
             */
            OutputParameterSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.TaskOutputsSpec.OutputParameterSpec)
                    return object;
                var message = new $root.ml_pipelines.TaskOutputsSpec.OutputParameterSpec();
                switch (object.type) {
                case "PRIMITIVE_TYPE_UNSPECIFIED":
                case 0:
                    message.type = 0;
                    break;
                case "INT":
                case 1:
                    message.type = 1;
                    break;
                case "DOUBLE":
                case 2:
                    message.type = 2;
                    break;
                case "STRING":
                case 3:
                    message.type = 3;
                    break;
                }
                return message;
            };

            /**
             * Creates a plain object from an OutputParameterSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @static
             * @param {ml_pipelines.TaskOutputsSpec.OutputParameterSpec} message OutputParameterSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            OutputParameterSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.type = options.enums === String ? "PRIMITIVE_TYPE_UNSPECIFIED" : 0;
                if (message.type != null && message.hasOwnProperty("type"))
                    object.type = options.enums === String ? $root.ml_pipelines.PrimitiveType.PrimitiveTypeEnum[message.type] : message.type;
                return object;
            };

            /**
             * Converts this OutputParameterSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.TaskOutputsSpec.OutputParameterSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            OutputParameterSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return OutputParameterSpec;
        })();

        return TaskOutputsSpec;
    })();

    ml_pipelines.PrimitiveType = (function() {

        /**
         * Properties of a PrimitiveType.
         * @memberof ml_pipelines
         * @interface IPrimitiveType
         */

        /**
         * Constructs a new PrimitiveType.
         * @memberof ml_pipelines
         * @classdesc Represents a PrimitiveType.
         * @implements IPrimitiveType
         * @constructor
         * @param {ml_pipelines.IPrimitiveType=} [properties] Properties to set
         */
        function PrimitiveType(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * Creates a new PrimitiveType instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.PrimitiveType
         * @static
         * @param {ml_pipelines.IPrimitiveType=} [properties] Properties to set
         * @returns {ml_pipelines.PrimitiveType} PrimitiveType instance
         */
        PrimitiveType.create = function create(properties) {
            return new PrimitiveType(properties);
        };

        /**
         * Encodes the specified PrimitiveType message. Does not implicitly {@link ml_pipelines.PrimitiveType.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.PrimitiveType
         * @static
         * @param {ml_pipelines.IPrimitiveType} message PrimitiveType message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PrimitiveType.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            return writer;
        };

        /**
         * Encodes the specified PrimitiveType message, length delimited. Does not implicitly {@link ml_pipelines.PrimitiveType.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.PrimitiveType
         * @static
         * @param {ml_pipelines.IPrimitiveType} message PrimitiveType message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PrimitiveType.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a PrimitiveType message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.PrimitiveType
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.PrimitiveType} PrimitiveType
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PrimitiveType.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PrimitiveType();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a PrimitiveType message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.PrimitiveType
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.PrimitiveType} PrimitiveType
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PrimitiveType.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a PrimitiveType message.
         * @function verify
         * @memberof ml_pipelines.PrimitiveType
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        PrimitiveType.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            return null;
        };

        /**
         * Creates a PrimitiveType message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.PrimitiveType
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.PrimitiveType} PrimitiveType
         */
        PrimitiveType.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.PrimitiveType)
                return object;
            return new $root.ml_pipelines.PrimitiveType();
        };

        /**
         * Creates a plain object from a PrimitiveType message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.PrimitiveType
         * @static
         * @param {ml_pipelines.PrimitiveType} message PrimitiveType
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        PrimitiveType.toObject = function toObject() {
            return {};
        };

        /**
         * Converts this PrimitiveType to JSON.
         * @function toJSON
         * @memberof ml_pipelines.PrimitiveType
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        PrimitiveType.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * PrimitiveTypeEnum enum.
         * @name ml_pipelines.PrimitiveType.PrimitiveTypeEnum
         * @enum {number}
         * @property {number} PRIMITIVE_TYPE_UNSPECIFIED=0 PRIMITIVE_TYPE_UNSPECIFIED value
         * @property {number} INT=1 INT value
         * @property {number} DOUBLE=2 DOUBLE value
         * @property {number} STRING=3 STRING value
         */
        PrimitiveType.PrimitiveTypeEnum = (function() {
            var valuesById = {}, values = Object.create(valuesById);
            values[valuesById[0] = "PRIMITIVE_TYPE_UNSPECIFIED"] = 0;
            values[valuesById[1] = "INT"] = 1;
            values[valuesById[2] = "DOUBLE"] = 2;
            values[valuesById[3] = "STRING"] = 3;
            return values;
        })();

        return PrimitiveType;
    })();

    ml_pipelines.PipelineTaskSpec = (function() {

        /**
         * Properties of a PipelineTaskSpec.
         * @memberof ml_pipelines
         * @interface IPipelineTaskSpec
         * @property {ml_pipelines.IPipelineTaskInfo|null} [taskInfo] PipelineTaskSpec taskInfo
         * @property {ml_pipelines.ITaskInputsSpec|null} [inputs] PipelineTaskSpec inputs
         * @property {Array.<string>|null} [dependentTasks] PipelineTaskSpec dependentTasks
         * @property {ml_pipelines.PipelineTaskSpec.ICachingOptions|null} [cachingOptions] PipelineTaskSpec cachingOptions
         * @property {ml_pipelines.IComponentRef|null} [componentRef] PipelineTaskSpec componentRef
         * @property {ml_pipelines.PipelineTaskSpec.ITriggerPolicy|null} [triggerPolicy] PipelineTaskSpec triggerPolicy
         * @property {ml_pipelines.IArtifactIteratorSpec|null} [artifactIterator] PipelineTaskSpec artifactIterator
         * @property {ml_pipelines.IParameterIteratorSpec|null} [parameterIterator] PipelineTaskSpec parameterIterator
         */

        /**
         * Constructs a new PipelineTaskSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a PipelineTaskSpec.
         * @implements IPipelineTaskSpec
         * @constructor
         * @param {ml_pipelines.IPipelineTaskSpec=} [properties] Properties to set
         */
        function PipelineTaskSpec(properties) {
            this.dependentTasks = [];
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * PipelineTaskSpec taskInfo.
         * @member {ml_pipelines.IPipelineTaskInfo|null|undefined} taskInfo
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         */
        PipelineTaskSpec.prototype.taskInfo = null;

        /**
         * PipelineTaskSpec inputs.
         * @member {ml_pipelines.ITaskInputsSpec|null|undefined} inputs
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         */
        PipelineTaskSpec.prototype.inputs = null;

        /**
         * PipelineTaskSpec dependentTasks.
         * @member {Array.<string>} dependentTasks
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         */
        PipelineTaskSpec.prototype.dependentTasks = $util.emptyArray;

        /**
         * PipelineTaskSpec cachingOptions.
         * @member {ml_pipelines.PipelineTaskSpec.ICachingOptions|null|undefined} cachingOptions
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         */
        PipelineTaskSpec.prototype.cachingOptions = null;

        /**
         * PipelineTaskSpec componentRef.
         * @member {ml_pipelines.IComponentRef|null|undefined} componentRef
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         */
        PipelineTaskSpec.prototype.componentRef = null;

        /**
         * PipelineTaskSpec triggerPolicy.
         * @member {ml_pipelines.PipelineTaskSpec.ITriggerPolicy|null|undefined} triggerPolicy
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         */
        PipelineTaskSpec.prototype.triggerPolicy = null;

        /**
         * PipelineTaskSpec artifactIterator.
         * @member {ml_pipelines.IArtifactIteratorSpec|null|undefined} artifactIterator
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         */
        PipelineTaskSpec.prototype.artifactIterator = null;

        /**
         * PipelineTaskSpec parameterIterator.
         * @member {ml_pipelines.IParameterIteratorSpec|null|undefined} parameterIterator
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         */
        PipelineTaskSpec.prototype.parameterIterator = null;

        // OneOf field names bound to virtual getters and setters
        var $oneOfFields;

        /**
         * PipelineTaskSpec iterator.
         * @member {"artifactIterator"|"parameterIterator"|undefined} iterator
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         */
        Object.defineProperty(PipelineTaskSpec.prototype, "iterator", {
            get: $util.oneOfGetter($oneOfFields = ["artifactIterator", "parameterIterator"]),
            set: $util.oneOfSetter($oneOfFields)
        });

        /**
         * Creates a new PipelineTaskSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.PipelineTaskSpec
         * @static
         * @param {ml_pipelines.IPipelineTaskSpec=} [properties] Properties to set
         * @returns {ml_pipelines.PipelineTaskSpec} PipelineTaskSpec instance
         */
        PipelineTaskSpec.create = function create(properties) {
            return new PipelineTaskSpec(properties);
        };

        /**
         * Encodes the specified PipelineTaskSpec message. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.PipelineTaskSpec
         * @static
         * @param {ml_pipelines.IPipelineTaskSpec} message PipelineTaskSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineTaskSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.taskInfo != null && Object.hasOwnProperty.call(message, "taskInfo"))
                $root.ml_pipelines.PipelineTaskInfo.encode(message.taskInfo, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
            if (message.inputs != null && Object.hasOwnProperty.call(message, "inputs"))
                $root.ml_pipelines.TaskInputsSpec.encode(message.inputs, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
            if (message.dependentTasks != null && message.dependentTasks.length)
                for (var i = 0; i < message.dependentTasks.length; ++i)
                    writer.uint32(/* id 5, wireType 2 =*/42).string(message.dependentTasks[i]);
            if (message.cachingOptions != null && Object.hasOwnProperty.call(message, "cachingOptions"))
                $root.ml_pipelines.PipelineTaskSpec.CachingOptions.encode(message.cachingOptions, writer.uint32(/* id 6, wireType 2 =*/50).fork()).ldelim();
            if (message.componentRef != null && Object.hasOwnProperty.call(message, "componentRef"))
                $root.ml_pipelines.ComponentRef.encode(message.componentRef, writer.uint32(/* id 7, wireType 2 =*/58).fork()).ldelim();
            if (message.triggerPolicy != null && Object.hasOwnProperty.call(message, "triggerPolicy"))
                $root.ml_pipelines.PipelineTaskSpec.TriggerPolicy.encode(message.triggerPolicy, writer.uint32(/* id 8, wireType 2 =*/66).fork()).ldelim();
            if (message.artifactIterator != null && Object.hasOwnProperty.call(message, "artifactIterator"))
                $root.ml_pipelines.ArtifactIteratorSpec.encode(message.artifactIterator, writer.uint32(/* id 9, wireType 2 =*/74).fork()).ldelim();
            if (message.parameterIterator != null && Object.hasOwnProperty.call(message, "parameterIterator"))
                $root.ml_pipelines.ParameterIteratorSpec.encode(message.parameterIterator, writer.uint32(/* id 10, wireType 2 =*/82).fork()).ldelim();
            return writer;
        };

        /**
         * Encodes the specified PipelineTaskSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.PipelineTaskSpec
         * @static
         * @param {ml_pipelines.IPipelineTaskSpec} message PipelineTaskSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineTaskSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a PipelineTaskSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.PipelineTaskSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.PipelineTaskSpec} PipelineTaskSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineTaskSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineTaskSpec();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.taskInfo = $root.ml_pipelines.PipelineTaskInfo.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.inputs = $root.ml_pipelines.TaskInputsSpec.decode(reader, reader.uint32());
                    break;
                case 5:
                    if (!(message.dependentTasks && message.dependentTasks.length))
                        message.dependentTasks = [];
                    message.dependentTasks.push(reader.string());
                    break;
                case 6:
                    message.cachingOptions = $root.ml_pipelines.PipelineTaskSpec.CachingOptions.decode(reader, reader.uint32());
                    break;
                case 7:
                    message.componentRef = $root.ml_pipelines.ComponentRef.decode(reader, reader.uint32());
                    break;
                case 8:
                    message.triggerPolicy = $root.ml_pipelines.PipelineTaskSpec.TriggerPolicy.decode(reader, reader.uint32());
                    break;
                case 9:
                    message.artifactIterator = $root.ml_pipelines.ArtifactIteratorSpec.decode(reader, reader.uint32());
                    break;
                case 10:
                    message.parameterIterator = $root.ml_pipelines.ParameterIteratorSpec.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a PipelineTaskSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.PipelineTaskSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.PipelineTaskSpec} PipelineTaskSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineTaskSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a PipelineTaskSpec message.
         * @function verify
         * @memberof ml_pipelines.PipelineTaskSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        PipelineTaskSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            var properties = {};
            if (message.taskInfo != null && message.hasOwnProperty("taskInfo")) {
                var error = $root.ml_pipelines.PipelineTaskInfo.verify(message.taskInfo);
                if (error)
                    return "taskInfo." + error;
            }
            if (message.inputs != null && message.hasOwnProperty("inputs")) {
                var error = $root.ml_pipelines.TaskInputsSpec.verify(message.inputs);
                if (error)
                    return "inputs." + error;
            }
            if (message.dependentTasks != null && message.hasOwnProperty("dependentTasks")) {
                if (!Array.isArray(message.dependentTasks))
                    return "dependentTasks: array expected";
                for (var i = 0; i < message.dependentTasks.length; ++i)
                    if (!$util.isString(message.dependentTasks[i]))
                        return "dependentTasks: string[] expected";
            }
            if (message.cachingOptions != null && message.hasOwnProperty("cachingOptions")) {
                var error = $root.ml_pipelines.PipelineTaskSpec.CachingOptions.verify(message.cachingOptions);
                if (error)
                    return "cachingOptions." + error;
            }
            if (message.componentRef != null && message.hasOwnProperty("componentRef")) {
                var error = $root.ml_pipelines.ComponentRef.verify(message.componentRef);
                if (error)
                    return "componentRef." + error;
            }
            if (message.triggerPolicy != null && message.hasOwnProperty("triggerPolicy")) {
                var error = $root.ml_pipelines.PipelineTaskSpec.TriggerPolicy.verify(message.triggerPolicy);
                if (error)
                    return "triggerPolicy." + error;
            }
            if (message.artifactIterator != null && message.hasOwnProperty("artifactIterator")) {
                properties.iterator = 1;
                {
                    var error = $root.ml_pipelines.ArtifactIteratorSpec.verify(message.artifactIterator);
                    if (error)
                        return "artifactIterator." + error;
                }
            }
            if (message.parameterIterator != null && message.hasOwnProperty("parameterIterator")) {
                if (properties.iterator === 1)
                    return "iterator: multiple values";
                properties.iterator = 1;
                {
                    var error = $root.ml_pipelines.ParameterIteratorSpec.verify(message.parameterIterator);
                    if (error)
                        return "parameterIterator." + error;
                }
            }
            return null;
        };

        /**
         * Creates a PipelineTaskSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.PipelineTaskSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.PipelineTaskSpec} PipelineTaskSpec
         */
        PipelineTaskSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.PipelineTaskSpec)
                return object;
            var message = new $root.ml_pipelines.PipelineTaskSpec();
            if (object.taskInfo != null) {
                if (typeof object.taskInfo !== "object")
                    throw TypeError(".ml_pipelines.PipelineTaskSpec.taskInfo: object expected");
                message.taskInfo = $root.ml_pipelines.PipelineTaskInfo.fromObject(object.taskInfo);
            }
            if (object.inputs != null) {
                if (typeof object.inputs !== "object")
                    throw TypeError(".ml_pipelines.PipelineTaskSpec.inputs: object expected");
                message.inputs = $root.ml_pipelines.TaskInputsSpec.fromObject(object.inputs);
            }
            if (object.dependentTasks) {
                if (!Array.isArray(object.dependentTasks))
                    throw TypeError(".ml_pipelines.PipelineTaskSpec.dependentTasks: array expected");
                message.dependentTasks = [];
                for (var i = 0; i < object.dependentTasks.length; ++i)
                    message.dependentTasks[i] = String(object.dependentTasks[i]);
            }
            if (object.cachingOptions != null) {
                if (typeof object.cachingOptions !== "object")
                    throw TypeError(".ml_pipelines.PipelineTaskSpec.cachingOptions: object expected");
                message.cachingOptions = $root.ml_pipelines.PipelineTaskSpec.CachingOptions.fromObject(object.cachingOptions);
            }
            if (object.componentRef != null) {
                if (typeof object.componentRef !== "object")
                    throw TypeError(".ml_pipelines.PipelineTaskSpec.componentRef: object expected");
                message.componentRef = $root.ml_pipelines.ComponentRef.fromObject(object.componentRef);
            }
            if (object.triggerPolicy != null) {
                if (typeof object.triggerPolicy !== "object")
                    throw TypeError(".ml_pipelines.PipelineTaskSpec.triggerPolicy: object expected");
                message.triggerPolicy = $root.ml_pipelines.PipelineTaskSpec.TriggerPolicy.fromObject(object.triggerPolicy);
            }
            if (object.artifactIterator != null) {
                if (typeof object.artifactIterator !== "object")
                    throw TypeError(".ml_pipelines.PipelineTaskSpec.artifactIterator: object expected");
                message.artifactIterator = $root.ml_pipelines.ArtifactIteratorSpec.fromObject(object.artifactIterator);
            }
            if (object.parameterIterator != null) {
                if (typeof object.parameterIterator !== "object")
                    throw TypeError(".ml_pipelines.PipelineTaskSpec.parameterIterator: object expected");
                message.parameterIterator = $root.ml_pipelines.ParameterIteratorSpec.fromObject(object.parameterIterator);
            }
            return message;
        };

        /**
         * Creates a plain object from a PipelineTaskSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.PipelineTaskSpec
         * @static
         * @param {ml_pipelines.PipelineTaskSpec} message PipelineTaskSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        PipelineTaskSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.arrays || options.defaults)
                object.dependentTasks = [];
            if (options.defaults) {
                object.taskInfo = null;
                object.inputs = null;
                object.cachingOptions = null;
                object.componentRef = null;
                object.triggerPolicy = null;
            }
            if (message.taskInfo != null && message.hasOwnProperty("taskInfo"))
                object.taskInfo = $root.ml_pipelines.PipelineTaskInfo.toObject(message.taskInfo, options);
            if (message.inputs != null && message.hasOwnProperty("inputs"))
                object.inputs = $root.ml_pipelines.TaskInputsSpec.toObject(message.inputs, options);
            if (message.dependentTasks && message.dependentTasks.length) {
                object.dependentTasks = [];
                for (var j = 0; j < message.dependentTasks.length; ++j)
                    object.dependentTasks[j] = message.dependentTasks[j];
            }
            if (message.cachingOptions != null && message.hasOwnProperty("cachingOptions"))
                object.cachingOptions = $root.ml_pipelines.PipelineTaskSpec.CachingOptions.toObject(message.cachingOptions, options);
            if (message.componentRef != null && message.hasOwnProperty("componentRef"))
                object.componentRef = $root.ml_pipelines.ComponentRef.toObject(message.componentRef, options);
            if (message.triggerPolicy != null && message.hasOwnProperty("triggerPolicy"))
                object.triggerPolicy = $root.ml_pipelines.PipelineTaskSpec.TriggerPolicy.toObject(message.triggerPolicy, options);
            if (message.artifactIterator != null && message.hasOwnProperty("artifactIterator")) {
                object.artifactIterator = $root.ml_pipelines.ArtifactIteratorSpec.toObject(message.artifactIterator, options);
                if (options.oneofs)
                    object.iterator = "artifactIterator";
            }
            if (message.parameterIterator != null && message.hasOwnProperty("parameterIterator")) {
                object.parameterIterator = $root.ml_pipelines.ParameterIteratorSpec.toObject(message.parameterIterator, options);
                if (options.oneofs)
                    object.iterator = "parameterIterator";
            }
            return object;
        };

        /**
         * Converts this PipelineTaskSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.PipelineTaskSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        PipelineTaskSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        PipelineTaskSpec.CachingOptions = (function() {

            /**
             * Properties of a CachingOptions.
             * @memberof ml_pipelines.PipelineTaskSpec
             * @interface ICachingOptions
             * @property {boolean|null} [enableCache] CachingOptions enableCache
             */

            /**
             * Constructs a new CachingOptions.
             * @memberof ml_pipelines.PipelineTaskSpec
             * @classdesc Represents a CachingOptions.
             * @implements ICachingOptions
             * @constructor
             * @param {ml_pipelines.PipelineTaskSpec.ICachingOptions=} [properties] Properties to set
             */
            function CachingOptions(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * CachingOptions enableCache.
             * @member {boolean} enableCache
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @instance
             */
            CachingOptions.prototype.enableCache = false;

            /**
             * Creates a new CachingOptions instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @static
             * @param {ml_pipelines.PipelineTaskSpec.ICachingOptions=} [properties] Properties to set
             * @returns {ml_pipelines.PipelineTaskSpec.CachingOptions} CachingOptions instance
             */
            CachingOptions.create = function create(properties) {
                return new CachingOptions(properties);
            };

            /**
             * Encodes the specified CachingOptions message. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.CachingOptions.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @static
             * @param {ml_pipelines.PipelineTaskSpec.ICachingOptions} message CachingOptions message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            CachingOptions.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.enableCache != null && Object.hasOwnProperty.call(message, "enableCache"))
                    writer.uint32(/* id 1, wireType 0 =*/8).bool(message.enableCache);
                return writer;
            };

            /**
             * Encodes the specified CachingOptions message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.CachingOptions.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @static
             * @param {ml_pipelines.PipelineTaskSpec.ICachingOptions} message CachingOptions message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            CachingOptions.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a CachingOptions message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.PipelineTaskSpec.CachingOptions} CachingOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            CachingOptions.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineTaskSpec.CachingOptions();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.enableCache = reader.bool();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a CachingOptions message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.PipelineTaskSpec.CachingOptions} CachingOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            CachingOptions.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a CachingOptions message.
             * @function verify
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            CachingOptions.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.enableCache != null && message.hasOwnProperty("enableCache"))
                    if (typeof message.enableCache !== "boolean")
                        return "enableCache: boolean expected";
                return null;
            };

            /**
             * Creates a CachingOptions message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.PipelineTaskSpec.CachingOptions} CachingOptions
             */
            CachingOptions.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.PipelineTaskSpec.CachingOptions)
                    return object;
                var message = new $root.ml_pipelines.PipelineTaskSpec.CachingOptions();
                if (object.enableCache != null)
                    message.enableCache = Boolean(object.enableCache);
                return message;
            };

            /**
             * Creates a plain object from a CachingOptions message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @static
             * @param {ml_pipelines.PipelineTaskSpec.CachingOptions} message CachingOptions
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            CachingOptions.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.enableCache = false;
                if (message.enableCache != null && message.hasOwnProperty("enableCache"))
                    object.enableCache = message.enableCache;
                return object;
            };

            /**
             * Converts this CachingOptions to JSON.
             * @function toJSON
             * @memberof ml_pipelines.PipelineTaskSpec.CachingOptions
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            CachingOptions.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return CachingOptions;
        })();

        PipelineTaskSpec.TriggerPolicy = (function() {

            /**
             * Properties of a TriggerPolicy.
             * @memberof ml_pipelines.PipelineTaskSpec
             * @interface ITriggerPolicy
             * @property {string|null} [condition] TriggerPolicy condition
             * @property {ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy|null} [strategy] TriggerPolicy strategy
             */

            /**
             * Constructs a new TriggerPolicy.
             * @memberof ml_pipelines.PipelineTaskSpec
             * @classdesc Represents a TriggerPolicy.
             * @implements ITriggerPolicy
             * @constructor
             * @param {ml_pipelines.PipelineTaskSpec.ITriggerPolicy=} [properties] Properties to set
             */
            function TriggerPolicy(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * TriggerPolicy condition.
             * @member {string} condition
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @instance
             */
            TriggerPolicy.prototype.condition = "";

            /**
             * TriggerPolicy strategy.
             * @member {ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy} strategy
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @instance
             */
            TriggerPolicy.prototype.strategy = 0;

            /**
             * Creates a new TriggerPolicy instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @static
             * @param {ml_pipelines.PipelineTaskSpec.ITriggerPolicy=} [properties] Properties to set
             * @returns {ml_pipelines.PipelineTaskSpec.TriggerPolicy} TriggerPolicy instance
             */
            TriggerPolicy.create = function create(properties) {
                return new TriggerPolicy(properties);
            };

            /**
             * Encodes the specified TriggerPolicy message. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.TriggerPolicy.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @static
             * @param {ml_pipelines.PipelineTaskSpec.ITriggerPolicy} message TriggerPolicy message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            TriggerPolicy.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.condition != null && Object.hasOwnProperty.call(message, "condition"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.condition);
                if (message.strategy != null && Object.hasOwnProperty.call(message, "strategy"))
                    writer.uint32(/* id 2, wireType 0 =*/16).int32(message.strategy);
                return writer;
            };

            /**
             * Encodes the specified TriggerPolicy message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.TriggerPolicy.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @static
             * @param {ml_pipelines.PipelineTaskSpec.ITriggerPolicy} message TriggerPolicy message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            TriggerPolicy.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a TriggerPolicy message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.PipelineTaskSpec.TriggerPolicy} TriggerPolicy
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            TriggerPolicy.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineTaskSpec.TriggerPolicy();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.condition = reader.string();
                        break;
                    case 2:
                        message.strategy = reader.int32();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a TriggerPolicy message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.PipelineTaskSpec.TriggerPolicy} TriggerPolicy
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            TriggerPolicy.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a TriggerPolicy message.
             * @function verify
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            TriggerPolicy.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.condition != null && message.hasOwnProperty("condition"))
                    if (!$util.isString(message.condition))
                        return "condition: string expected";
                if (message.strategy != null && message.hasOwnProperty("strategy"))
                    switch (message.strategy) {
                    default:
                        return "strategy: enum value expected";
                    case 0:
                    case 1:
                    case 2:
                        break;
                    }
                return null;
            };

            /**
             * Creates a TriggerPolicy message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.PipelineTaskSpec.TriggerPolicy} TriggerPolicy
             */
            TriggerPolicy.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.PipelineTaskSpec.TriggerPolicy)
                    return object;
                var message = new $root.ml_pipelines.PipelineTaskSpec.TriggerPolicy();
                if (object.condition != null)
                    message.condition = String(object.condition);
                switch (object.strategy) {
                case "TRIGGER_STRATEGY_UNSPECIFIED":
                case 0:
                    message.strategy = 0;
                    break;
                case "ALL_UPSTREAM_TASKS_SUCCEEDED":
                case 1:
                    message.strategy = 1;
                    break;
                case "ALL_UPSTREAM_TASKS_COMPLETED":
                case 2:
                    message.strategy = 2;
                    break;
                }
                return message;
            };

            /**
             * Creates a plain object from a TriggerPolicy message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @static
             * @param {ml_pipelines.PipelineTaskSpec.TriggerPolicy} message TriggerPolicy
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            TriggerPolicy.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    object.condition = "";
                    object.strategy = options.enums === String ? "TRIGGER_STRATEGY_UNSPECIFIED" : 0;
                }
                if (message.condition != null && message.hasOwnProperty("condition"))
                    object.condition = message.condition;
                if (message.strategy != null && message.hasOwnProperty("strategy"))
                    object.strategy = options.enums === String ? $root.ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy[message.strategy] : message.strategy;
                return object;
            };

            /**
             * Converts this TriggerPolicy to JSON.
             * @function toJSON
             * @memberof ml_pipelines.PipelineTaskSpec.TriggerPolicy
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            TriggerPolicy.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * TriggerStrategy enum.
             * @name ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy
             * @enum {number}
             * @property {number} TRIGGER_STRATEGY_UNSPECIFIED=0 TRIGGER_STRATEGY_UNSPECIFIED value
             * @property {number} ALL_UPSTREAM_TASKS_SUCCEEDED=1 ALL_UPSTREAM_TASKS_SUCCEEDED value
             * @property {number} ALL_UPSTREAM_TASKS_COMPLETED=2 ALL_UPSTREAM_TASKS_COMPLETED value
             */
            TriggerPolicy.TriggerStrategy = (function() {
                var valuesById = {}, values = Object.create(valuesById);
                values[valuesById[0] = "TRIGGER_STRATEGY_UNSPECIFIED"] = 0;
                values[valuesById[1] = "ALL_UPSTREAM_TASKS_SUCCEEDED"] = 1;
                values[valuesById[2] = "ALL_UPSTREAM_TASKS_COMPLETED"] = 2;
                return values;
            })();

            return TriggerPolicy;
        })();

        return PipelineTaskSpec;
    })();

    ml_pipelines.ArtifactIteratorSpec = (function() {

        /**
         * Properties of an ArtifactIteratorSpec.
         * @memberof ml_pipelines
         * @interface IArtifactIteratorSpec
         * @property {ml_pipelines.ArtifactIteratorSpec.IItemsSpec|null} [items] ArtifactIteratorSpec items
         * @property {string|null} [itemInput] ArtifactIteratorSpec itemInput
         */

        /**
         * Constructs a new ArtifactIteratorSpec.
         * @memberof ml_pipelines
         * @classdesc Represents an ArtifactIteratorSpec.
         * @implements IArtifactIteratorSpec
         * @constructor
         * @param {ml_pipelines.IArtifactIteratorSpec=} [properties] Properties to set
         */
        function ArtifactIteratorSpec(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ArtifactIteratorSpec items.
         * @member {ml_pipelines.ArtifactIteratorSpec.IItemsSpec|null|undefined} items
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @instance
         */
        ArtifactIteratorSpec.prototype.items = null;

        /**
         * ArtifactIteratorSpec itemInput.
         * @member {string} itemInput
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @instance
         */
        ArtifactIteratorSpec.prototype.itemInput = "";

        /**
         * Creates a new ArtifactIteratorSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @static
         * @param {ml_pipelines.IArtifactIteratorSpec=} [properties] Properties to set
         * @returns {ml_pipelines.ArtifactIteratorSpec} ArtifactIteratorSpec instance
         */
        ArtifactIteratorSpec.create = function create(properties) {
            return new ArtifactIteratorSpec(properties);
        };

        /**
         * Encodes the specified ArtifactIteratorSpec message. Does not implicitly {@link ml_pipelines.ArtifactIteratorSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @static
         * @param {ml_pipelines.IArtifactIteratorSpec} message ArtifactIteratorSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ArtifactIteratorSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.items != null && Object.hasOwnProperty.call(message, "items"))
                $root.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.encode(message.items, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
            if (message.itemInput != null && Object.hasOwnProperty.call(message, "itemInput"))
                writer.uint32(/* id 2, wireType 2 =*/18).string(message.itemInput);
            return writer;
        };

        /**
         * Encodes the specified ArtifactIteratorSpec message, length delimited. Does not implicitly {@link ml_pipelines.ArtifactIteratorSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @static
         * @param {ml_pipelines.IArtifactIteratorSpec} message ArtifactIteratorSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ArtifactIteratorSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes an ArtifactIteratorSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ArtifactIteratorSpec} ArtifactIteratorSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ArtifactIteratorSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ArtifactIteratorSpec();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.items = $root.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.itemInput = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes an ArtifactIteratorSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ArtifactIteratorSpec} ArtifactIteratorSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ArtifactIteratorSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies an ArtifactIteratorSpec message.
         * @function verify
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ArtifactIteratorSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.items != null && message.hasOwnProperty("items")) {
                var error = $root.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.verify(message.items);
                if (error)
                    return "items." + error;
            }
            if (message.itemInput != null && message.hasOwnProperty("itemInput"))
                if (!$util.isString(message.itemInput))
                    return "itemInput: string expected";
            return null;
        };

        /**
         * Creates an ArtifactIteratorSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ArtifactIteratorSpec} ArtifactIteratorSpec
         */
        ArtifactIteratorSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ArtifactIteratorSpec)
                return object;
            var message = new $root.ml_pipelines.ArtifactIteratorSpec();
            if (object.items != null) {
                if (typeof object.items !== "object")
                    throw TypeError(".ml_pipelines.ArtifactIteratorSpec.items: object expected");
                message.items = $root.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.fromObject(object.items);
            }
            if (object.itemInput != null)
                message.itemInput = String(object.itemInput);
            return message;
        };

        /**
         * Creates a plain object from an ArtifactIteratorSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @static
         * @param {ml_pipelines.ArtifactIteratorSpec} message ArtifactIteratorSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ArtifactIteratorSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.defaults) {
                object.items = null;
                object.itemInput = "";
            }
            if (message.items != null && message.hasOwnProperty("items"))
                object.items = $root.ml_pipelines.ArtifactIteratorSpec.ItemsSpec.toObject(message.items, options);
            if (message.itemInput != null && message.hasOwnProperty("itemInput"))
                object.itemInput = message.itemInput;
            return object;
        };

        /**
         * Converts this ArtifactIteratorSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ArtifactIteratorSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ArtifactIteratorSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        ArtifactIteratorSpec.ItemsSpec = (function() {

            /**
             * Properties of an ItemsSpec.
             * @memberof ml_pipelines.ArtifactIteratorSpec
             * @interface IItemsSpec
             * @property {string|null} [inputArtifact] ItemsSpec inputArtifact
             */

            /**
             * Constructs a new ItemsSpec.
             * @memberof ml_pipelines.ArtifactIteratorSpec
             * @classdesc Represents an ItemsSpec.
             * @implements IItemsSpec
             * @constructor
             * @param {ml_pipelines.ArtifactIteratorSpec.IItemsSpec=} [properties] Properties to set
             */
            function ItemsSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ItemsSpec inputArtifact.
             * @member {string} inputArtifact
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @instance
             */
            ItemsSpec.prototype.inputArtifact = "";

            /**
             * Creates a new ItemsSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @static
             * @param {ml_pipelines.ArtifactIteratorSpec.IItemsSpec=} [properties] Properties to set
             * @returns {ml_pipelines.ArtifactIteratorSpec.ItemsSpec} ItemsSpec instance
             */
            ItemsSpec.create = function create(properties) {
                return new ItemsSpec(properties);
            };

            /**
             * Encodes the specified ItemsSpec message. Does not implicitly {@link ml_pipelines.ArtifactIteratorSpec.ItemsSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @static
             * @param {ml_pipelines.ArtifactIteratorSpec.IItemsSpec} message ItemsSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ItemsSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.inputArtifact != null && Object.hasOwnProperty.call(message, "inputArtifact"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.inputArtifact);
                return writer;
            };

            /**
             * Encodes the specified ItemsSpec message, length delimited. Does not implicitly {@link ml_pipelines.ArtifactIteratorSpec.ItemsSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @static
             * @param {ml_pipelines.ArtifactIteratorSpec.IItemsSpec} message ItemsSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ItemsSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an ItemsSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.ArtifactIteratorSpec.ItemsSpec} ItemsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ItemsSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ArtifactIteratorSpec.ItemsSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.inputArtifact = reader.string();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an ItemsSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.ArtifactIteratorSpec.ItemsSpec} ItemsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ItemsSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an ItemsSpec message.
             * @function verify
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ItemsSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.inputArtifact != null && message.hasOwnProperty("inputArtifact"))
                    if (!$util.isString(message.inputArtifact))
                        return "inputArtifact: string expected";
                return null;
            };

            /**
             * Creates an ItemsSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.ArtifactIteratorSpec.ItemsSpec} ItemsSpec
             */
            ItemsSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.ArtifactIteratorSpec.ItemsSpec)
                    return object;
                var message = new $root.ml_pipelines.ArtifactIteratorSpec.ItemsSpec();
                if (object.inputArtifact != null)
                    message.inputArtifact = String(object.inputArtifact);
                return message;
            };

            /**
             * Creates a plain object from an ItemsSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @static
             * @param {ml_pipelines.ArtifactIteratorSpec.ItemsSpec} message ItemsSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ItemsSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.inputArtifact = "";
                if (message.inputArtifact != null && message.hasOwnProperty("inputArtifact"))
                    object.inputArtifact = message.inputArtifact;
                return object;
            };

            /**
             * Converts this ItemsSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.ArtifactIteratorSpec.ItemsSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ItemsSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ItemsSpec;
        })();

        return ArtifactIteratorSpec;
    })();

    ml_pipelines.ParameterIteratorSpec = (function() {

        /**
         * Properties of a ParameterIteratorSpec.
         * @memberof ml_pipelines
         * @interface IParameterIteratorSpec
         * @property {ml_pipelines.ParameterIteratorSpec.IItemsSpec|null} [items] ParameterIteratorSpec items
         * @property {string|null} [itemInput] ParameterIteratorSpec itemInput
         */

        /**
         * Constructs a new ParameterIteratorSpec.
         * @memberof ml_pipelines
         * @classdesc Represents a ParameterIteratorSpec.
         * @implements IParameterIteratorSpec
         * @constructor
         * @param {ml_pipelines.IParameterIteratorSpec=} [properties] Properties to set
         */
        function ParameterIteratorSpec(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ParameterIteratorSpec items.
         * @member {ml_pipelines.ParameterIteratorSpec.IItemsSpec|null|undefined} items
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @instance
         */
        ParameterIteratorSpec.prototype.items = null;

        /**
         * ParameterIteratorSpec itemInput.
         * @member {string} itemInput
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @instance
         */
        ParameterIteratorSpec.prototype.itemInput = "";

        /**
         * Creates a new ParameterIteratorSpec instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @static
         * @param {ml_pipelines.IParameterIteratorSpec=} [properties] Properties to set
         * @returns {ml_pipelines.ParameterIteratorSpec} ParameterIteratorSpec instance
         */
        ParameterIteratorSpec.create = function create(properties) {
            return new ParameterIteratorSpec(properties);
        };

        /**
         * Encodes the specified ParameterIteratorSpec message. Does not implicitly {@link ml_pipelines.ParameterIteratorSpec.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @static
         * @param {ml_pipelines.IParameterIteratorSpec} message ParameterIteratorSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ParameterIteratorSpec.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.items != null && Object.hasOwnProperty.call(message, "items"))
                $root.ml_pipelines.ParameterIteratorSpec.ItemsSpec.encode(message.items, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
            if (message.itemInput != null && Object.hasOwnProperty.call(message, "itemInput"))
                writer.uint32(/* id 2, wireType 2 =*/18).string(message.itemInput);
            return writer;
        };

        /**
         * Encodes the specified ParameterIteratorSpec message, length delimited. Does not implicitly {@link ml_pipelines.ParameterIteratorSpec.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @static
         * @param {ml_pipelines.IParameterIteratorSpec} message ParameterIteratorSpec message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ParameterIteratorSpec.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a ParameterIteratorSpec message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ParameterIteratorSpec} ParameterIteratorSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ParameterIteratorSpec.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ParameterIteratorSpec();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.items = $root.ml_pipelines.ParameterIteratorSpec.ItemsSpec.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.itemInput = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a ParameterIteratorSpec message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ParameterIteratorSpec} ParameterIteratorSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ParameterIteratorSpec.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a ParameterIteratorSpec message.
         * @function verify
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ParameterIteratorSpec.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.items != null && message.hasOwnProperty("items")) {
                var error = $root.ml_pipelines.ParameterIteratorSpec.ItemsSpec.verify(message.items);
                if (error)
                    return "items." + error;
            }
            if (message.itemInput != null && message.hasOwnProperty("itemInput"))
                if (!$util.isString(message.itemInput))
                    return "itemInput: string expected";
            return null;
        };

        /**
         * Creates a ParameterIteratorSpec message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ParameterIteratorSpec} ParameterIteratorSpec
         */
        ParameterIteratorSpec.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ParameterIteratorSpec)
                return object;
            var message = new $root.ml_pipelines.ParameterIteratorSpec();
            if (object.items != null) {
                if (typeof object.items !== "object")
                    throw TypeError(".ml_pipelines.ParameterIteratorSpec.items: object expected");
                message.items = $root.ml_pipelines.ParameterIteratorSpec.ItemsSpec.fromObject(object.items);
            }
            if (object.itemInput != null)
                message.itemInput = String(object.itemInput);
            return message;
        };

        /**
         * Creates a plain object from a ParameterIteratorSpec message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @static
         * @param {ml_pipelines.ParameterIteratorSpec} message ParameterIteratorSpec
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ParameterIteratorSpec.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.defaults) {
                object.items = null;
                object.itemInput = "";
            }
            if (message.items != null && message.hasOwnProperty("items"))
                object.items = $root.ml_pipelines.ParameterIteratorSpec.ItemsSpec.toObject(message.items, options);
            if (message.itemInput != null && message.hasOwnProperty("itemInput"))
                object.itemInput = message.itemInput;
            return object;
        };

        /**
         * Converts this ParameterIteratorSpec to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ParameterIteratorSpec
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ParameterIteratorSpec.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        ParameterIteratorSpec.ItemsSpec = (function() {

            /**
             * Properties of an ItemsSpec.
             * @memberof ml_pipelines.ParameterIteratorSpec
             * @interface IItemsSpec
             * @property {string|null} [raw] ItemsSpec raw
             * @property {string|null} [inputParameter] ItemsSpec inputParameter
             */

            /**
             * Constructs a new ItemsSpec.
             * @memberof ml_pipelines.ParameterIteratorSpec
             * @classdesc Represents an ItemsSpec.
             * @implements IItemsSpec
             * @constructor
             * @param {ml_pipelines.ParameterIteratorSpec.IItemsSpec=} [properties] Properties to set
             */
            function ItemsSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ItemsSpec raw.
             * @member {string|null|undefined} raw
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @instance
             */
            ItemsSpec.prototype.raw = null;

            /**
             * ItemsSpec inputParameter.
             * @member {string|null|undefined} inputParameter
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @instance
             */
            ItemsSpec.prototype.inputParameter = null;

            // OneOf field names bound to virtual getters and setters
            var $oneOfFields;

            /**
             * ItemsSpec kind.
             * @member {"raw"|"inputParameter"|undefined} kind
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @instance
             */
            Object.defineProperty(ItemsSpec.prototype, "kind", {
                get: $util.oneOfGetter($oneOfFields = ["raw", "inputParameter"]),
                set: $util.oneOfSetter($oneOfFields)
            });

            /**
             * Creates a new ItemsSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @static
             * @param {ml_pipelines.ParameterIteratorSpec.IItemsSpec=} [properties] Properties to set
             * @returns {ml_pipelines.ParameterIteratorSpec.ItemsSpec} ItemsSpec instance
             */
            ItemsSpec.create = function create(properties) {
                return new ItemsSpec(properties);
            };

            /**
             * Encodes the specified ItemsSpec message. Does not implicitly {@link ml_pipelines.ParameterIteratorSpec.ItemsSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @static
             * @param {ml_pipelines.ParameterIteratorSpec.IItemsSpec} message ItemsSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ItemsSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.raw != null && Object.hasOwnProperty.call(message, "raw"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.raw);
                if (message.inputParameter != null && Object.hasOwnProperty.call(message, "inputParameter"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.inputParameter);
                return writer;
            };

            /**
             * Encodes the specified ItemsSpec message, length delimited. Does not implicitly {@link ml_pipelines.ParameterIteratorSpec.ItemsSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @static
             * @param {ml_pipelines.ParameterIteratorSpec.IItemsSpec} message ItemsSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ItemsSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an ItemsSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.ParameterIteratorSpec.ItemsSpec} ItemsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ItemsSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ParameterIteratorSpec.ItemsSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.raw = reader.string();
                        break;
                    case 2:
                        message.inputParameter = reader.string();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an ItemsSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.ParameterIteratorSpec.ItemsSpec} ItemsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ItemsSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an ItemsSpec message.
             * @function verify
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ItemsSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                var properties = {};
                if (message.raw != null && message.hasOwnProperty("raw")) {
                    properties.kind = 1;
                    if (!$util.isString(message.raw))
                        return "raw: string expected";
                }
                if (message.inputParameter != null && message.hasOwnProperty("inputParameter")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    if (!$util.isString(message.inputParameter))
                        return "inputParameter: string expected";
                }
                return null;
            };

            /**
             * Creates an ItemsSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.ParameterIteratorSpec.ItemsSpec} ItemsSpec
             */
            ItemsSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.ParameterIteratorSpec.ItemsSpec)
                    return object;
                var message = new $root.ml_pipelines.ParameterIteratorSpec.ItemsSpec();
                if (object.raw != null)
                    message.raw = String(object.raw);
                if (object.inputParameter != null)
                    message.inputParameter = String(object.inputParameter);
                return message;
            };

            /**
             * Creates a plain object from an ItemsSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @static
             * @param {ml_pipelines.ParameterIteratorSpec.ItemsSpec} message ItemsSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ItemsSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (message.raw != null && message.hasOwnProperty("raw")) {
                    object.raw = message.raw;
                    if (options.oneofs)
                        object.kind = "raw";
                }
                if (message.inputParameter != null && message.hasOwnProperty("inputParameter")) {
                    object.inputParameter = message.inputParameter;
                    if (options.oneofs)
                        object.kind = "inputParameter";
                }
                return object;
            };

            /**
             * Converts this ItemsSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.ParameterIteratorSpec.ItemsSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ItemsSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ItemsSpec;
        })();

        return ParameterIteratorSpec;
    })();

    ml_pipelines.ComponentRef = (function() {

        /**
         * Properties of a ComponentRef.
         * @memberof ml_pipelines
         * @interface IComponentRef
         * @property {string|null} [name] ComponentRef name
         */

        /**
         * Constructs a new ComponentRef.
         * @memberof ml_pipelines
         * @classdesc Represents a ComponentRef.
         * @implements IComponentRef
         * @constructor
         * @param {ml_pipelines.IComponentRef=} [properties] Properties to set
         */
        function ComponentRef(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ComponentRef name.
         * @member {string} name
         * @memberof ml_pipelines.ComponentRef
         * @instance
         */
        ComponentRef.prototype.name = "";

        /**
         * Creates a new ComponentRef instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ComponentRef
         * @static
         * @param {ml_pipelines.IComponentRef=} [properties] Properties to set
         * @returns {ml_pipelines.ComponentRef} ComponentRef instance
         */
        ComponentRef.create = function create(properties) {
            return new ComponentRef(properties);
        };

        /**
         * Encodes the specified ComponentRef message. Does not implicitly {@link ml_pipelines.ComponentRef.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ComponentRef
         * @static
         * @param {ml_pipelines.IComponentRef} message ComponentRef message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ComponentRef.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
            return writer;
        };

        /**
         * Encodes the specified ComponentRef message, length delimited. Does not implicitly {@link ml_pipelines.ComponentRef.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ComponentRef
         * @static
         * @param {ml_pipelines.IComponentRef} message ComponentRef message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ComponentRef.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a ComponentRef message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ComponentRef
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ComponentRef} ComponentRef
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ComponentRef.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ComponentRef();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a ComponentRef message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ComponentRef
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ComponentRef} ComponentRef
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ComponentRef.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a ComponentRef message.
         * @function verify
         * @memberof ml_pipelines.ComponentRef
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ComponentRef.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.name != null && message.hasOwnProperty("name"))
                if (!$util.isString(message.name))
                    return "name: string expected";
            return null;
        };

        /**
         * Creates a ComponentRef message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ComponentRef
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ComponentRef} ComponentRef
         */
        ComponentRef.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ComponentRef)
                return object;
            var message = new $root.ml_pipelines.ComponentRef();
            if (object.name != null)
                message.name = String(object.name);
            return message;
        };

        /**
         * Creates a plain object from a ComponentRef message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ComponentRef
         * @static
         * @param {ml_pipelines.ComponentRef} message ComponentRef
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ComponentRef.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.defaults)
                object.name = "";
            if (message.name != null && message.hasOwnProperty("name"))
                object.name = message.name;
            return object;
        };

        /**
         * Converts this ComponentRef to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ComponentRef
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ComponentRef.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return ComponentRef;
    })();

    ml_pipelines.PipelineInfo = (function() {

        /**
         * Properties of a PipelineInfo.
         * @memberof ml_pipelines
         * @interface IPipelineInfo
         * @property {string|null} [name] PipelineInfo name
         */

        /**
         * Constructs a new PipelineInfo.
         * @memberof ml_pipelines
         * @classdesc Represents a PipelineInfo.
         * @implements IPipelineInfo
         * @constructor
         * @param {ml_pipelines.IPipelineInfo=} [properties] Properties to set
         */
        function PipelineInfo(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * PipelineInfo name.
         * @member {string} name
         * @memberof ml_pipelines.PipelineInfo
         * @instance
         */
        PipelineInfo.prototype.name = "";

        /**
         * Creates a new PipelineInfo instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.PipelineInfo
         * @static
         * @param {ml_pipelines.IPipelineInfo=} [properties] Properties to set
         * @returns {ml_pipelines.PipelineInfo} PipelineInfo instance
         */
        PipelineInfo.create = function create(properties) {
            return new PipelineInfo(properties);
        };

        /**
         * Encodes the specified PipelineInfo message. Does not implicitly {@link ml_pipelines.PipelineInfo.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.PipelineInfo
         * @static
         * @param {ml_pipelines.IPipelineInfo} message PipelineInfo message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineInfo.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
            return writer;
        };

        /**
         * Encodes the specified PipelineInfo message, length delimited. Does not implicitly {@link ml_pipelines.PipelineInfo.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.PipelineInfo
         * @static
         * @param {ml_pipelines.IPipelineInfo} message PipelineInfo message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineInfo.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a PipelineInfo message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.PipelineInfo
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.PipelineInfo} PipelineInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineInfo.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineInfo();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a PipelineInfo message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.PipelineInfo
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.PipelineInfo} PipelineInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineInfo.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a PipelineInfo message.
         * @function verify
         * @memberof ml_pipelines.PipelineInfo
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        PipelineInfo.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.name != null && message.hasOwnProperty("name"))
                if (!$util.isString(message.name))
                    return "name: string expected";
            return null;
        };

        /**
         * Creates a PipelineInfo message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.PipelineInfo
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.PipelineInfo} PipelineInfo
         */
        PipelineInfo.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.PipelineInfo)
                return object;
            var message = new $root.ml_pipelines.PipelineInfo();
            if (object.name != null)
                message.name = String(object.name);
            return message;
        };

        /**
         * Creates a plain object from a PipelineInfo message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.PipelineInfo
         * @static
         * @param {ml_pipelines.PipelineInfo} message PipelineInfo
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        PipelineInfo.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.defaults)
                object.name = "";
            if (message.name != null && message.hasOwnProperty("name"))
                object.name = message.name;
            return object;
        };

        /**
         * Converts this PipelineInfo to JSON.
         * @function toJSON
         * @memberof ml_pipelines.PipelineInfo
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        PipelineInfo.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return PipelineInfo;
    })();

    ml_pipelines.ArtifactTypeSchema = (function() {

        /**
         * Properties of an ArtifactTypeSchema.
         * @memberof ml_pipelines
         * @interface IArtifactTypeSchema
         * @property {string|null} [schemaTitle] ArtifactTypeSchema schemaTitle
         * @property {string|null} [schemaUri] ArtifactTypeSchema schemaUri
         * @property {string|null} [instanceSchema] ArtifactTypeSchema instanceSchema
         * @property {string|null} [schemaVersion] ArtifactTypeSchema schemaVersion
         */

        /**
         * Constructs a new ArtifactTypeSchema.
         * @memberof ml_pipelines
         * @classdesc Represents an ArtifactTypeSchema.
         * @implements IArtifactTypeSchema
         * @constructor
         * @param {ml_pipelines.IArtifactTypeSchema=} [properties] Properties to set
         */
        function ArtifactTypeSchema(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ArtifactTypeSchema schemaTitle.
         * @member {string|null|undefined} schemaTitle
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @instance
         */
        ArtifactTypeSchema.prototype.schemaTitle = null;

        /**
         * ArtifactTypeSchema schemaUri.
         * @member {string|null|undefined} schemaUri
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @instance
         */
        ArtifactTypeSchema.prototype.schemaUri = null;

        /**
         * ArtifactTypeSchema instanceSchema.
         * @member {string|null|undefined} instanceSchema
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @instance
         */
        ArtifactTypeSchema.prototype.instanceSchema = null;

        /**
         * ArtifactTypeSchema schemaVersion.
         * @member {string} schemaVersion
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @instance
         */
        ArtifactTypeSchema.prototype.schemaVersion = "";

        // OneOf field names bound to virtual getters and setters
        var $oneOfFields;

        /**
         * ArtifactTypeSchema kind.
         * @member {"schemaTitle"|"schemaUri"|"instanceSchema"|undefined} kind
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @instance
         */
        Object.defineProperty(ArtifactTypeSchema.prototype, "kind", {
            get: $util.oneOfGetter($oneOfFields = ["schemaTitle", "schemaUri", "instanceSchema"]),
            set: $util.oneOfSetter($oneOfFields)
        });

        /**
         * Creates a new ArtifactTypeSchema instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @static
         * @param {ml_pipelines.IArtifactTypeSchema=} [properties] Properties to set
         * @returns {ml_pipelines.ArtifactTypeSchema} ArtifactTypeSchema instance
         */
        ArtifactTypeSchema.create = function create(properties) {
            return new ArtifactTypeSchema(properties);
        };

        /**
         * Encodes the specified ArtifactTypeSchema message. Does not implicitly {@link ml_pipelines.ArtifactTypeSchema.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @static
         * @param {ml_pipelines.IArtifactTypeSchema} message ArtifactTypeSchema message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ArtifactTypeSchema.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.schemaTitle != null && Object.hasOwnProperty.call(message, "schemaTitle"))
                writer.uint32(/* id 1, wireType 2 =*/10).string(message.schemaTitle);
            if (message.schemaUri != null && Object.hasOwnProperty.call(message, "schemaUri"))
                writer.uint32(/* id 2, wireType 2 =*/18).string(message.schemaUri);
            if (message.instanceSchema != null && Object.hasOwnProperty.call(message, "instanceSchema"))
                writer.uint32(/* id 3, wireType 2 =*/26).string(message.instanceSchema);
            if (message.schemaVersion != null && Object.hasOwnProperty.call(message, "schemaVersion"))
                writer.uint32(/* id 4, wireType 2 =*/34).string(message.schemaVersion);
            return writer;
        };

        /**
         * Encodes the specified ArtifactTypeSchema message, length delimited. Does not implicitly {@link ml_pipelines.ArtifactTypeSchema.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @static
         * @param {ml_pipelines.IArtifactTypeSchema} message ArtifactTypeSchema message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ArtifactTypeSchema.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes an ArtifactTypeSchema message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ArtifactTypeSchema} ArtifactTypeSchema
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ArtifactTypeSchema.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ArtifactTypeSchema();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.schemaTitle = reader.string();
                    break;
                case 2:
                    message.schemaUri = reader.string();
                    break;
                case 3:
                    message.instanceSchema = reader.string();
                    break;
                case 4:
                    message.schemaVersion = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes an ArtifactTypeSchema message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ArtifactTypeSchema} ArtifactTypeSchema
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ArtifactTypeSchema.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies an ArtifactTypeSchema message.
         * @function verify
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ArtifactTypeSchema.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            var properties = {};
            if (message.schemaTitle != null && message.hasOwnProperty("schemaTitle")) {
                properties.kind = 1;
                if (!$util.isString(message.schemaTitle))
                    return "schemaTitle: string expected";
            }
            if (message.schemaUri != null && message.hasOwnProperty("schemaUri")) {
                if (properties.kind === 1)
                    return "kind: multiple values";
                properties.kind = 1;
                if (!$util.isString(message.schemaUri))
                    return "schemaUri: string expected";
            }
            if (message.instanceSchema != null && message.hasOwnProperty("instanceSchema")) {
                if (properties.kind === 1)
                    return "kind: multiple values";
                properties.kind = 1;
                if (!$util.isString(message.instanceSchema))
                    return "instanceSchema: string expected";
            }
            if (message.schemaVersion != null && message.hasOwnProperty("schemaVersion"))
                if (!$util.isString(message.schemaVersion))
                    return "schemaVersion: string expected";
            return null;
        };

        /**
         * Creates an ArtifactTypeSchema message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ArtifactTypeSchema} ArtifactTypeSchema
         */
        ArtifactTypeSchema.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ArtifactTypeSchema)
                return object;
            var message = new $root.ml_pipelines.ArtifactTypeSchema();
            if (object.schemaTitle != null)
                message.schemaTitle = String(object.schemaTitle);
            if (object.schemaUri != null)
                message.schemaUri = String(object.schemaUri);
            if (object.instanceSchema != null)
                message.instanceSchema = String(object.instanceSchema);
            if (object.schemaVersion != null)
                message.schemaVersion = String(object.schemaVersion);
            return message;
        };

        /**
         * Creates a plain object from an ArtifactTypeSchema message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @static
         * @param {ml_pipelines.ArtifactTypeSchema} message ArtifactTypeSchema
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ArtifactTypeSchema.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.defaults)
                object.schemaVersion = "";
            if (message.schemaTitle != null && message.hasOwnProperty("schemaTitle")) {
                object.schemaTitle = message.schemaTitle;
                if (options.oneofs)
                    object.kind = "schemaTitle";
            }
            if (message.schemaUri != null && message.hasOwnProperty("schemaUri")) {
                object.schemaUri = message.schemaUri;
                if (options.oneofs)
                    object.kind = "schemaUri";
            }
            if (message.instanceSchema != null && message.hasOwnProperty("instanceSchema")) {
                object.instanceSchema = message.instanceSchema;
                if (options.oneofs)
                    object.kind = "instanceSchema";
            }
            if (message.schemaVersion != null && message.hasOwnProperty("schemaVersion"))
                object.schemaVersion = message.schemaVersion;
            return object;
        };

        /**
         * Converts this ArtifactTypeSchema to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ArtifactTypeSchema
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ArtifactTypeSchema.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return ArtifactTypeSchema;
    })();

    ml_pipelines.PipelineTaskInfo = (function() {

        /**
         * Properties of a PipelineTaskInfo.
         * @memberof ml_pipelines
         * @interface IPipelineTaskInfo
         * @property {string|null} [name] PipelineTaskInfo name
         */

        /**
         * Constructs a new PipelineTaskInfo.
         * @memberof ml_pipelines
         * @classdesc Represents a PipelineTaskInfo.
         * @implements IPipelineTaskInfo
         * @constructor
         * @param {ml_pipelines.IPipelineTaskInfo=} [properties] Properties to set
         */
        function PipelineTaskInfo(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * PipelineTaskInfo name.
         * @member {string} name
         * @memberof ml_pipelines.PipelineTaskInfo
         * @instance
         */
        PipelineTaskInfo.prototype.name = "";

        /**
         * Creates a new PipelineTaskInfo instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.PipelineTaskInfo
         * @static
         * @param {ml_pipelines.IPipelineTaskInfo=} [properties] Properties to set
         * @returns {ml_pipelines.PipelineTaskInfo} PipelineTaskInfo instance
         */
        PipelineTaskInfo.create = function create(properties) {
            return new PipelineTaskInfo(properties);
        };

        /**
         * Encodes the specified PipelineTaskInfo message. Does not implicitly {@link ml_pipelines.PipelineTaskInfo.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.PipelineTaskInfo
         * @static
         * @param {ml_pipelines.IPipelineTaskInfo} message PipelineTaskInfo message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineTaskInfo.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
            return writer;
        };

        /**
         * Encodes the specified PipelineTaskInfo message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskInfo.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.PipelineTaskInfo
         * @static
         * @param {ml_pipelines.IPipelineTaskInfo} message PipelineTaskInfo message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineTaskInfo.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a PipelineTaskInfo message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.PipelineTaskInfo
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.PipelineTaskInfo} PipelineTaskInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineTaskInfo.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineTaskInfo();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a PipelineTaskInfo message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.PipelineTaskInfo
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.PipelineTaskInfo} PipelineTaskInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineTaskInfo.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a PipelineTaskInfo message.
         * @function verify
         * @memberof ml_pipelines.PipelineTaskInfo
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        PipelineTaskInfo.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.name != null && message.hasOwnProperty("name"))
                if (!$util.isString(message.name))
                    return "name: string expected";
            return null;
        };

        /**
         * Creates a PipelineTaskInfo message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.PipelineTaskInfo
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.PipelineTaskInfo} PipelineTaskInfo
         */
        PipelineTaskInfo.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.PipelineTaskInfo)
                return object;
            var message = new $root.ml_pipelines.PipelineTaskInfo();
            if (object.name != null)
                message.name = String(object.name);
            return message;
        };

        /**
         * Creates a plain object from a PipelineTaskInfo message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.PipelineTaskInfo
         * @static
         * @param {ml_pipelines.PipelineTaskInfo} message PipelineTaskInfo
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        PipelineTaskInfo.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.defaults)
                object.name = "";
            if (message.name != null && message.hasOwnProperty("name"))
                object.name = message.name;
            return object;
        };

        /**
         * Converts this PipelineTaskInfo to JSON.
         * @function toJSON
         * @memberof ml_pipelines.PipelineTaskInfo
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        PipelineTaskInfo.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return PipelineTaskInfo;
    })();

    ml_pipelines.ValueOrRuntimeParameter = (function() {

        /**
         * Properties of a ValueOrRuntimeParameter.
         * @memberof ml_pipelines
         * @interface IValueOrRuntimeParameter
         * @property {ml_pipelines.IValue|null} [constantValue] ValueOrRuntimeParameter constantValue
         * @property {string|null} [runtimeParameter] ValueOrRuntimeParameter runtimeParameter
         */

        /**
         * Constructs a new ValueOrRuntimeParameter.
         * @memberof ml_pipelines
         * @classdesc Represents a ValueOrRuntimeParameter.
         * @implements IValueOrRuntimeParameter
         * @constructor
         * @param {ml_pipelines.IValueOrRuntimeParameter=} [properties] Properties to set
         */
        function ValueOrRuntimeParameter(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ValueOrRuntimeParameter constantValue.
         * @member {ml_pipelines.IValue|null|undefined} constantValue
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @instance
         */
        ValueOrRuntimeParameter.prototype.constantValue = null;

        /**
         * ValueOrRuntimeParameter runtimeParameter.
         * @member {string|null|undefined} runtimeParameter
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @instance
         */
        ValueOrRuntimeParameter.prototype.runtimeParameter = null;

        // OneOf field names bound to virtual getters and setters
        var $oneOfFields;

        /**
         * ValueOrRuntimeParameter value.
         * @member {"constantValue"|"runtimeParameter"|undefined} value
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @instance
         */
        Object.defineProperty(ValueOrRuntimeParameter.prototype, "value", {
            get: $util.oneOfGetter($oneOfFields = ["constantValue", "runtimeParameter"]),
            set: $util.oneOfSetter($oneOfFields)
        });

        /**
         * Creates a new ValueOrRuntimeParameter instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @static
         * @param {ml_pipelines.IValueOrRuntimeParameter=} [properties] Properties to set
         * @returns {ml_pipelines.ValueOrRuntimeParameter} ValueOrRuntimeParameter instance
         */
        ValueOrRuntimeParameter.create = function create(properties) {
            return new ValueOrRuntimeParameter(properties);
        };

        /**
         * Encodes the specified ValueOrRuntimeParameter message. Does not implicitly {@link ml_pipelines.ValueOrRuntimeParameter.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @static
         * @param {ml_pipelines.IValueOrRuntimeParameter} message ValueOrRuntimeParameter message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ValueOrRuntimeParameter.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.constantValue != null && Object.hasOwnProperty.call(message, "constantValue"))
                $root.ml_pipelines.Value.encode(message.constantValue, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
            if (message.runtimeParameter != null && Object.hasOwnProperty.call(message, "runtimeParameter"))
                writer.uint32(/* id 2, wireType 2 =*/18).string(message.runtimeParameter);
            return writer;
        };

        /**
         * Encodes the specified ValueOrRuntimeParameter message, length delimited. Does not implicitly {@link ml_pipelines.ValueOrRuntimeParameter.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @static
         * @param {ml_pipelines.IValueOrRuntimeParameter} message ValueOrRuntimeParameter message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ValueOrRuntimeParameter.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a ValueOrRuntimeParameter message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ValueOrRuntimeParameter} ValueOrRuntimeParameter
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ValueOrRuntimeParameter.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ValueOrRuntimeParameter();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.constantValue = $root.ml_pipelines.Value.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.runtimeParameter = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a ValueOrRuntimeParameter message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ValueOrRuntimeParameter} ValueOrRuntimeParameter
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ValueOrRuntimeParameter.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a ValueOrRuntimeParameter message.
         * @function verify
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ValueOrRuntimeParameter.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            var properties = {};
            if (message.constantValue != null && message.hasOwnProperty("constantValue")) {
                properties.value = 1;
                {
                    var error = $root.ml_pipelines.Value.verify(message.constantValue);
                    if (error)
                        return "constantValue." + error;
                }
            }
            if (message.runtimeParameter != null && message.hasOwnProperty("runtimeParameter")) {
                if (properties.value === 1)
                    return "value: multiple values";
                properties.value = 1;
                if (!$util.isString(message.runtimeParameter))
                    return "runtimeParameter: string expected";
            }
            return null;
        };

        /**
         * Creates a ValueOrRuntimeParameter message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ValueOrRuntimeParameter} ValueOrRuntimeParameter
         */
        ValueOrRuntimeParameter.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ValueOrRuntimeParameter)
                return object;
            var message = new $root.ml_pipelines.ValueOrRuntimeParameter();
            if (object.constantValue != null) {
                if (typeof object.constantValue !== "object")
                    throw TypeError(".ml_pipelines.ValueOrRuntimeParameter.constantValue: object expected");
                message.constantValue = $root.ml_pipelines.Value.fromObject(object.constantValue);
            }
            if (object.runtimeParameter != null)
                message.runtimeParameter = String(object.runtimeParameter);
            return message;
        };

        /**
         * Creates a plain object from a ValueOrRuntimeParameter message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @static
         * @param {ml_pipelines.ValueOrRuntimeParameter} message ValueOrRuntimeParameter
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ValueOrRuntimeParameter.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (message.constantValue != null && message.hasOwnProperty("constantValue")) {
                object.constantValue = $root.ml_pipelines.Value.toObject(message.constantValue, options);
                if (options.oneofs)
                    object.value = "constantValue";
            }
            if (message.runtimeParameter != null && message.hasOwnProperty("runtimeParameter")) {
                object.runtimeParameter = message.runtimeParameter;
                if (options.oneofs)
                    object.value = "runtimeParameter";
            }
            return object;
        };

        /**
         * Converts this ValueOrRuntimeParameter to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ValueOrRuntimeParameter
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ValueOrRuntimeParameter.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return ValueOrRuntimeParameter;
    })();

    ml_pipelines.PipelineDeploymentConfig = (function() {

        /**
         * Properties of a PipelineDeploymentConfig.
         * @memberof ml_pipelines
         * @interface IPipelineDeploymentConfig
         * @property {Object.<string,ml_pipelines.PipelineDeploymentConfig.IExecutorSpec>|null} [executors] PipelineDeploymentConfig executors
         */

        /**
         * Constructs a new PipelineDeploymentConfig.
         * @memberof ml_pipelines
         * @classdesc Represents a PipelineDeploymentConfig.
         * @implements IPipelineDeploymentConfig
         * @constructor
         * @param {ml_pipelines.IPipelineDeploymentConfig=} [properties] Properties to set
         */
        function PipelineDeploymentConfig(properties) {
            this.executors = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * PipelineDeploymentConfig executors.
         * @member {Object.<string,ml_pipelines.PipelineDeploymentConfig.IExecutorSpec>} executors
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @instance
         */
        PipelineDeploymentConfig.prototype.executors = $util.emptyObject;

        /**
         * Creates a new PipelineDeploymentConfig instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @static
         * @param {ml_pipelines.IPipelineDeploymentConfig=} [properties] Properties to set
         * @returns {ml_pipelines.PipelineDeploymentConfig} PipelineDeploymentConfig instance
         */
        PipelineDeploymentConfig.create = function create(properties) {
            return new PipelineDeploymentConfig(properties);
        };

        /**
         * Encodes the specified PipelineDeploymentConfig message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @static
         * @param {ml_pipelines.IPipelineDeploymentConfig} message PipelineDeploymentConfig message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineDeploymentConfig.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.executors != null && Object.hasOwnProperty.call(message, "executors"))
                for (var keys = Object.keys(message.executors), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.encode(message.executors[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            return writer;
        };

        /**
         * Encodes the specified PipelineDeploymentConfig message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @static
         * @param {ml_pipelines.IPipelineDeploymentConfig} message PipelineDeploymentConfig message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineDeploymentConfig.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a PipelineDeploymentConfig message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.PipelineDeploymentConfig} PipelineDeploymentConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineDeploymentConfig.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    if (message.executors === $util.emptyObject)
                        message.executors = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.executors[key] = value;
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a PipelineDeploymentConfig message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.PipelineDeploymentConfig} PipelineDeploymentConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineDeploymentConfig.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a PipelineDeploymentConfig message.
         * @function verify
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        PipelineDeploymentConfig.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.executors != null && message.hasOwnProperty("executors")) {
                if (!$util.isObject(message.executors))
                    return "executors: object expected";
                var key = Object.keys(message.executors);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.verify(message.executors[key[i]]);
                    if (error)
                        return "executors." + error;
                }
            }
            return null;
        };

        /**
         * Creates a PipelineDeploymentConfig message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.PipelineDeploymentConfig} PipelineDeploymentConfig
         */
        PipelineDeploymentConfig.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig)
                return object;
            var message = new $root.ml_pipelines.PipelineDeploymentConfig();
            if (object.executors) {
                if (typeof object.executors !== "object")
                    throw TypeError(".ml_pipelines.PipelineDeploymentConfig.executors: object expected");
                message.executors = {};
                for (var keys = Object.keys(object.executors), i = 0; i < keys.length; ++i) {
                    if (typeof object.executors[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.executors: object expected");
                    message.executors[keys[i]] = $root.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.fromObject(object.executors[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a PipelineDeploymentConfig message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @static
         * @param {ml_pipelines.PipelineDeploymentConfig} message PipelineDeploymentConfig
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        PipelineDeploymentConfig.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults)
                object.executors = {};
            var keys2;
            if (message.executors && (keys2 = Object.keys(message.executors)).length) {
                object.executors = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.executors[keys2[j]] = $root.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.toObject(message.executors[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this PipelineDeploymentConfig to JSON.
         * @function toJSON
         * @memberof ml_pipelines.PipelineDeploymentConfig
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        PipelineDeploymentConfig.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        PipelineDeploymentConfig.PipelineContainerSpec = (function() {

            /**
             * Properties of a PipelineContainerSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @interface IPipelineContainerSpec
             * @property {string|null} [image] PipelineContainerSpec image
             * @property {Array.<string>|null} [command] PipelineContainerSpec command
             * @property {Array.<string>|null} [args] PipelineContainerSpec args
             * @property {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle|null} [lifecycle] PipelineContainerSpec lifecycle
             * @property {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec|null} [resources] PipelineContainerSpec resources
             */

            /**
             * Constructs a new PipelineContainerSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @classdesc Represents a PipelineContainerSpec.
             * @implements IPipelineContainerSpec
             * @constructor
             * @param {ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec=} [properties] Properties to set
             */
            function PipelineContainerSpec(properties) {
                this.command = [];
                this.args = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * PipelineContainerSpec image.
             * @member {string} image
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @instance
             */
            PipelineContainerSpec.prototype.image = "";

            /**
             * PipelineContainerSpec command.
             * @member {Array.<string>} command
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @instance
             */
            PipelineContainerSpec.prototype.command = $util.emptyArray;

            /**
             * PipelineContainerSpec args.
             * @member {Array.<string>} args
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @instance
             */
            PipelineContainerSpec.prototype.args = $util.emptyArray;

            /**
             * PipelineContainerSpec lifecycle.
             * @member {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle|null|undefined} lifecycle
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @instance
             */
            PipelineContainerSpec.prototype.lifecycle = null;

            /**
             * PipelineContainerSpec resources.
             * @member {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec|null|undefined} resources
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @instance
             */
            PipelineContainerSpec.prototype.resources = null;

            /**
             * Creates a new PipelineContainerSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec=} [properties] Properties to set
             * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} PipelineContainerSpec instance
             */
            PipelineContainerSpec.create = function create(properties) {
                return new PipelineContainerSpec(properties);
            };

            /**
             * Encodes the specified PipelineContainerSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec} message PipelineContainerSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            PipelineContainerSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.image != null && Object.hasOwnProperty.call(message, "image"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.image);
                if (message.command != null && message.command.length)
                    for (var i = 0; i < message.command.length; ++i)
                        writer.uint32(/* id 2, wireType 2 =*/18).string(message.command[i]);
                if (message.args != null && message.args.length)
                    for (var i = 0; i < message.args.length; ++i)
                        writer.uint32(/* id 3, wireType 2 =*/26).string(message.args[i]);
                if (message.lifecycle != null && Object.hasOwnProperty.call(message, "lifecycle"))
                    $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.encode(message.lifecycle, writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                if (message.resources != null && Object.hasOwnProperty.call(message, "resources"))
                    $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.encode(message.resources, writer.uint32(/* id 5, wireType 2 =*/42).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified PipelineContainerSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec} message PipelineContainerSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            PipelineContainerSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a PipelineContainerSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} PipelineContainerSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            PipelineContainerSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.image = reader.string();
                        break;
                    case 2:
                        if (!(message.command && message.command.length))
                            message.command = [];
                        message.command.push(reader.string());
                        break;
                    case 3:
                        if (!(message.args && message.args.length))
                            message.args = [];
                        message.args.push(reader.string());
                        break;
                    case 4:
                        message.lifecycle = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.decode(reader, reader.uint32());
                        break;
                    case 5:
                        message.resources = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a PipelineContainerSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} PipelineContainerSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            PipelineContainerSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a PipelineContainerSpec message.
             * @function verify
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            PipelineContainerSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.image != null && message.hasOwnProperty("image"))
                    if (!$util.isString(message.image))
                        return "image: string expected";
                if (message.command != null && message.hasOwnProperty("command")) {
                    if (!Array.isArray(message.command))
                        return "command: array expected";
                    for (var i = 0; i < message.command.length; ++i)
                        if (!$util.isString(message.command[i]))
                            return "command: string[] expected";
                }
                if (message.args != null && message.hasOwnProperty("args")) {
                    if (!Array.isArray(message.args))
                        return "args: array expected";
                    for (var i = 0; i < message.args.length; ++i)
                        if (!$util.isString(message.args[i]))
                            return "args: string[] expected";
                }
                if (message.lifecycle != null && message.hasOwnProperty("lifecycle")) {
                    var error = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.verify(message.lifecycle);
                    if (error)
                        return "lifecycle." + error;
                }
                if (message.resources != null && message.hasOwnProperty("resources")) {
                    var error = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.verify(message.resources);
                    if (error)
                        return "resources." + error;
                }
                return null;
            };

            /**
             * Creates a PipelineContainerSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} PipelineContainerSpec
             */
            PipelineContainerSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec)
                    return object;
                var message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec();
                if (object.image != null)
                    message.image = String(object.image);
                if (object.command) {
                    if (!Array.isArray(object.command))
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.command: array expected");
                    message.command = [];
                    for (var i = 0; i < object.command.length; ++i)
                        message.command[i] = String(object.command[i]);
                }
                if (object.args) {
                    if (!Array.isArray(object.args))
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.args: array expected");
                    message.args = [];
                    for (var i = 0; i < object.args.length; ++i)
                        message.args[i] = String(object.args[i]);
                }
                if (object.lifecycle != null) {
                    if (typeof object.lifecycle !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.lifecycle: object expected");
                    message.lifecycle = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.fromObject(object.lifecycle);
                }
                if (object.resources != null) {
                    if (typeof object.resources !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.resources: object expected");
                    message.resources = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.fromObject(object.resources);
                }
                return message;
            };

            /**
             * Creates a plain object from a PipelineContainerSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec} message PipelineContainerSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            PipelineContainerSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults) {
                    object.command = [];
                    object.args = [];
                }
                if (options.defaults) {
                    object.image = "";
                    object.lifecycle = null;
                    object.resources = null;
                }
                if (message.image != null && message.hasOwnProperty("image"))
                    object.image = message.image;
                if (message.command && message.command.length) {
                    object.command = [];
                    for (var j = 0; j < message.command.length; ++j)
                        object.command[j] = message.command[j];
                }
                if (message.args && message.args.length) {
                    object.args = [];
                    for (var j = 0; j < message.args.length; ++j)
                        object.args[j] = message.args[j];
                }
                if (message.lifecycle != null && message.hasOwnProperty("lifecycle"))
                    object.lifecycle = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.toObject(message.lifecycle, options);
                if (message.resources != null && message.hasOwnProperty("resources"))
                    object.resources = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.toObject(message.resources, options);
                return object;
            };

            /**
             * Converts this PipelineContainerSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            PipelineContainerSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            PipelineContainerSpec.Lifecycle = (function() {

                /**
                 * Properties of a Lifecycle.
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
                 * @interface ILifecycle
                 * @property {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec|null} [preCacheCheck] Lifecycle preCacheCheck
                 */

                /**
                 * Constructs a new Lifecycle.
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
                 * @classdesc Represents a Lifecycle.
                 * @implements ILifecycle
                 * @constructor
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle=} [properties] Properties to set
                 */
                function Lifecycle(properties) {
                    if (properties)
                        for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null)
                                this[keys[i]] = properties[keys[i]];
                }

                /**
                 * Lifecycle preCacheCheck.
                 * @member {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec|null|undefined} preCacheCheck
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @instance
                 */
                Lifecycle.prototype.preCacheCheck = null;

                /**
                 * Creates a new Lifecycle instance using the specified properties.
                 * @function create
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle=} [properties] Properties to set
                 * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} Lifecycle instance
                 */
                Lifecycle.create = function create(properties) {
                    return new Lifecycle(properties);
                };

                /**
                 * Encodes the specified Lifecycle message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.verify|verify} messages.
                 * @function encode
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle} message Lifecycle message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                Lifecycle.encode = function encode(message, writer) {
                    if (!writer)
                        writer = $Writer.create();
                    if (message.preCacheCheck != null && Object.hasOwnProperty.call(message, "preCacheCheck"))
                        $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.encode(message.preCacheCheck, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                    return writer;
                };

                /**
                 * Encodes the specified Lifecycle message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.verify|verify} messages.
                 * @function encodeDelimited
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle} message Lifecycle message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                Lifecycle.encodeDelimited = function encodeDelimited(message, writer) {
                    return this.encode(message, writer).ldelim();
                };

                /**
                 * Decodes a Lifecycle message from the specified reader or buffer.
                 * @function decode
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} Lifecycle
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                Lifecycle.decode = function decode(reader, length) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle();
                    while (reader.pos < end) {
                        var tag = reader.uint32();
                        switch (tag >>> 3) {
                        case 1:
                            message.preCacheCheck = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag & 7);
                            break;
                        }
                    }
                    return message;
                };

                /**
                 * Decodes a Lifecycle message from the specified reader or buffer, length delimited.
                 * @function decodeDelimited
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} Lifecycle
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                Lifecycle.decodeDelimited = function decodeDelimited(reader) {
                    if (!(reader instanceof $Reader))
                        reader = new $Reader(reader);
                    return this.decode(reader, reader.uint32());
                };

                /**
                 * Verifies a Lifecycle message.
                 * @function verify
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @static
                 * @param {Object.<string,*>} message Plain object to verify
                 * @returns {string|null} `null` if valid, otherwise the reason why it is not
                 */
                Lifecycle.verify = function verify(message) {
                    if (typeof message !== "object" || message === null)
                        return "object expected";
                    if (message.preCacheCheck != null && message.hasOwnProperty("preCacheCheck")) {
                        var error = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.verify(message.preCacheCheck);
                        if (error)
                            return "preCacheCheck." + error;
                    }
                    return null;
                };

                /**
                 * Creates a Lifecycle message from a plain object. Also converts values to their respective internal types.
                 * @function fromObject
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @static
                 * @param {Object.<string,*>} object Plain object
                 * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} Lifecycle
                 */
                Lifecycle.fromObject = function fromObject(object) {
                    if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle)
                        return object;
                    var message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle();
                    if (object.preCacheCheck != null) {
                        if (typeof object.preCacheCheck !== "object")
                            throw TypeError(".ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.preCacheCheck: object expected");
                        message.preCacheCheck = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.fromObject(object.preCacheCheck);
                    }
                    return message;
                };

                /**
                 * Creates a plain object from a Lifecycle message. Also converts values to other types if specified.
                 * @function toObject
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle} message Lifecycle
                 * @param {$protobuf.IConversionOptions} [options] Conversion options
                 * @returns {Object.<string,*>} Plain object
                 */
                Lifecycle.toObject = function toObject(message, options) {
                    if (!options)
                        options = {};
                    var object = {};
                    if (options.defaults)
                        object.preCacheCheck = null;
                    if (message.preCacheCheck != null && message.hasOwnProperty("preCacheCheck"))
                        object.preCacheCheck = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.toObject(message.preCacheCheck, options);
                    return object;
                };

                /**
                 * Converts this Lifecycle to JSON.
                 * @function toJSON
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                 * @instance
                 * @returns {Object.<string,*>} JSON object
                 */
                Lifecycle.prototype.toJSON = function toJSON() {
                    return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                };

                Lifecycle.Exec = (function() {

                    /**
                     * Properties of an Exec.
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                     * @interface IExec
                     * @property {Array.<string>|null} [command] Exec command
                     * @property {Array.<string>|null} [args] Exec args
                     */

                    /**
                     * Constructs a new Exec.
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle
                     * @classdesc Represents an Exec.
                     * @implements IExec
                     * @constructor
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec=} [properties] Properties to set
                     */
                    function Exec(properties) {
                        this.command = [];
                        this.args = [];
                        if (properties)
                            for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                                if (properties[keys[i]] != null)
                                    this[keys[i]] = properties[keys[i]];
                    }

                    /**
                     * Exec command.
                     * @member {Array.<string>} command
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @instance
                     */
                    Exec.prototype.command = $util.emptyArray;

                    /**
                     * Exec args.
                     * @member {Array.<string>} args
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @instance
                     */
                    Exec.prototype.args = $util.emptyArray;

                    /**
                     * Creates a new Exec instance using the specified properties.
                     * @function create
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @static
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec=} [properties] Properties to set
                     * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} Exec instance
                     */
                    Exec.create = function create(properties) {
                        return new Exec(properties);
                    };

                    /**
                     * Encodes the specified Exec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.verify|verify} messages.
                     * @function encode
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @static
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec} message Exec message or plain object to encode
                     * @param {$protobuf.Writer} [writer] Writer to encode to
                     * @returns {$protobuf.Writer} Writer
                     */
                    Exec.encode = function encode(message, writer) {
                        if (!writer)
                            writer = $Writer.create();
                        if (message.command != null && message.command.length)
                            for (var i = 0; i < message.command.length; ++i)
                                writer.uint32(/* id 2, wireType 2 =*/18).string(message.command[i]);
                        if (message.args != null && message.args.length)
                            for (var i = 0; i < message.args.length; ++i)
                                writer.uint32(/* id 3, wireType 2 =*/26).string(message.args[i]);
                        return writer;
                    };

                    /**
                     * Encodes the specified Exec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.verify|verify} messages.
                     * @function encodeDelimited
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @static
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec} message Exec message or plain object to encode
                     * @param {$protobuf.Writer} [writer] Writer to encode to
                     * @returns {$protobuf.Writer} Writer
                     */
                    Exec.encodeDelimited = function encodeDelimited(message, writer) {
                        return this.encode(message, writer).ldelim();
                    };

                    /**
                     * Decodes an Exec message from the specified reader or buffer.
                     * @function decode
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @static
                     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                     * @param {number} [length] Message length if known beforehand
                     * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} Exec
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    Exec.decode = function decode(reader, length) {
                        if (!(reader instanceof $Reader))
                            reader = $Reader.create(reader);
                        var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec();
                        while (reader.pos < end) {
                            var tag = reader.uint32();
                            switch (tag >>> 3) {
                            case 2:
                                if (!(message.command && message.command.length))
                                    message.command = [];
                                message.command.push(reader.string());
                                break;
                            case 3:
                                if (!(message.args && message.args.length))
                                    message.args = [];
                                message.args.push(reader.string());
                                break;
                            default:
                                reader.skipType(tag & 7);
                                break;
                            }
                        }
                        return message;
                    };

                    /**
                     * Decodes an Exec message from the specified reader or buffer, length delimited.
                     * @function decodeDelimited
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @static
                     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                     * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} Exec
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    Exec.decodeDelimited = function decodeDelimited(reader) {
                        if (!(reader instanceof $Reader))
                            reader = new $Reader(reader);
                        return this.decode(reader, reader.uint32());
                    };

                    /**
                     * Verifies an Exec message.
                     * @function verify
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @static
                     * @param {Object.<string,*>} message Plain object to verify
                     * @returns {string|null} `null` if valid, otherwise the reason why it is not
                     */
                    Exec.verify = function verify(message) {
                        if (typeof message !== "object" || message === null)
                            return "object expected";
                        if (message.command != null && message.hasOwnProperty("command")) {
                            if (!Array.isArray(message.command))
                                return "command: array expected";
                            for (var i = 0; i < message.command.length; ++i)
                                if (!$util.isString(message.command[i]))
                                    return "command: string[] expected";
                        }
                        if (message.args != null && message.hasOwnProperty("args")) {
                            if (!Array.isArray(message.args))
                                return "args: array expected";
                            for (var i = 0; i < message.args.length; ++i)
                                if (!$util.isString(message.args[i]))
                                    return "args: string[] expected";
                        }
                        return null;
                    };

                    /**
                     * Creates an Exec message from a plain object. Also converts values to their respective internal types.
                     * @function fromObject
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @static
                     * @param {Object.<string,*>} object Plain object
                     * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} Exec
                     */
                    Exec.fromObject = function fromObject(object) {
                        if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec)
                            return object;
                        var message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec();
                        if (object.command) {
                            if (!Array.isArray(object.command))
                                throw TypeError(".ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.command: array expected");
                            message.command = [];
                            for (var i = 0; i < object.command.length; ++i)
                                message.command[i] = String(object.command[i]);
                        }
                        if (object.args) {
                            if (!Array.isArray(object.args))
                                throw TypeError(".ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.args: array expected");
                            message.args = [];
                            for (var i = 0; i < object.args.length; ++i)
                                message.args[i] = String(object.args[i]);
                        }
                        return message;
                    };

                    /**
                     * Creates a plain object from an Exec message. Also converts values to other types if specified.
                     * @function toObject
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @static
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec} message Exec
                     * @param {$protobuf.IConversionOptions} [options] Conversion options
                     * @returns {Object.<string,*>} Plain object
                     */
                    Exec.toObject = function toObject(message, options) {
                        if (!options)
                            options = {};
                        var object = {};
                        if (options.arrays || options.defaults) {
                            object.command = [];
                            object.args = [];
                        }
                        if (message.command && message.command.length) {
                            object.command = [];
                            for (var j = 0; j < message.command.length; ++j)
                                object.command[j] = message.command[j];
                        }
                        if (message.args && message.args.length) {
                            object.args = [];
                            for (var j = 0; j < message.args.length; ++j)
                                object.args[j] = message.args[j];
                        }
                        return object;
                    };

                    /**
                     * Converts this Exec to JSON.
                     * @function toJSON
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec
                     * @instance
                     * @returns {Object.<string,*>} JSON object
                     */
                    Exec.prototype.toJSON = function toJSON() {
                        return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                    };

                    return Exec;
                })();

                return Lifecycle;
            })();

            PipelineContainerSpec.ResourceSpec = (function() {

                /**
                 * Properties of a ResourceSpec.
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
                 * @interface IResourceSpec
                 * @property {number|null} [cpuLimit] ResourceSpec cpuLimit
                 * @property {number|null} [memoryLimit] ResourceSpec memoryLimit
                 * @property {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig|null} [accelerator] ResourceSpec accelerator
                 */

                /**
                 * Constructs a new ResourceSpec.
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec
                 * @classdesc Represents a ResourceSpec.
                 * @implements IResourceSpec
                 * @constructor
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec=} [properties] Properties to set
                 */
                function ResourceSpec(properties) {
                    if (properties)
                        for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null)
                                this[keys[i]] = properties[keys[i]];
                }

                /**
                 * ResourceSpec cpuLimit.
                 * @member {number} cpuLimit
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @instance
                 */
                ResourceSpec.prototype.cpuLimit = 0;

                /**
                 * ResourceSpec memoryLimit.
                 * @member {number} memoryLimit
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @instance
                 */
                ResourceSpec.prototype.memoryLimit = 0;

                /**
                 * ResourceSpec accelerator.
                 * @member {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig|null|undefined} accelerator
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @instance
                 */
                ResourceSpec.prototype.accelerator = null;

                /**
                 * Creates a new ResourceSpec instance using the specified properties.
                 * @function create
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec=} [properties] Properties to set
                 * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} ResourceSpec instance
                 */
                ResourceSpec.create = function create(properties) {
                    return new ResourceSpec(properties);
                };

                /**
                 * Encodes the specified ResourceSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.verify|verify} messages.
                 * @function encode
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec} message ResourceSpec message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                ResourceSpec.encode = function encode(message, writer) {
                    if (!writer)
                        writer = $Writer.create();
                    if (message.cpuLimit != null && Object.hasOwnProperty.call(message, "cpuLimit"))
                        writer.uint32(/* id 1, wireType 1 =*/9).double(message.cpuLimit);
                    if (message.memoryLimit != null && Object.hasOwnProperty.call(message, "memoryLimit"))
                        writer.uint32(/* id 2, wireType 1 =*/17).double(message.memoryLimit);
                    if (message.accelerator != null && Object.hasOwnProperty.call(message, "accelerator"))
                        $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.encode(message.accelerator, writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                    return writer;
                };

                /**
                 * Encodes the specified ResourceSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.verify|verify} messages.
                 * @function encodeDelimited
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec} message ResourceSpec message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                ResourceSpec.encodeDelimited = function encodeDelimited(message, writer) {
                    return this.encode(message, writer).ldelim();
                };

                /**
                 * Decodes a ResourceSpec message from the specified reader or buffer.
                 * @function decode
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} ResourceSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                ResourceSpec.decode = function decode(reader, length) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec();
                    while (reader.pos < end) {
                        var tag = reader.uint32();
                        switch (tag >>> 3) {
                        case 1:
                            message.cpuLimit = reader.double();
                            break;
                        case 2:
                            message.memoryLimit = reader.double();
                            break;
                        case 3:
                            message.accelerator = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag & 7);
                            break;
                        }
                    }
                    return message;
                };

                /**
                 * Decodes a ResourceSpec message from the specified reader or buffer, length delimited.
                 * @function decodeDelimited
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} ResourceSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                ResourceSpec.decodeDelimited = function decodeDelimited(reader) {
                    if (!(reader instanceof $Reader))
                        reader = new $Reader(reader);
                    return this.decode(reader, reader.uint32());
                };

                /**
                 * Verifies a ResourceSpec message.
                 * @function verify
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @static
                 * @param {Object.<string,*>} message Plain object to verify
                 * @returns {string|null} `null` if valid, otherwise the reason why it is not
                 */
                ResourceSpec.verify = function verify(message) {
                    if (typeof message !== "object" || message === null)
                        return "object expected";
                    if (message.cpuLimit != null && message.hasOwnProperty("cpuLimit"))
                        if (typeof message.cpuLimit !== "number")
                            return "cpuLimit: number expected";
                    if (message.memoryLimit != null && message.hasOwnProperty("memoryLimit"))
                        if (typeof message.memoryLimit !== "number")
                            return "memoryLimit: number expected";
                    if (message.accelerator != null && message.hasOwnProperty("accelerator")) {
                        var error = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.verify(message.accelerator);
                        if (error)
                            return "accelerator." + error;
                    }
                    return null;
                };

                /**
                 * Creates a ResourceSpec message from a plain object. Also converts values to their respective internal types.
                 * @function fromObject
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @static
                 * @param {Object.<string,*>} object Plain object
                 * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} ResourceSpec
                 */
                ResourceSpec.fromObject = function fromObject(object) {
                    if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec)
                        return object;
                    var message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec();
                    if (object.cpuLimit != null)
                        message.cpuLimit = Number(object.cpuLimit);
                    if (object.memoryLimit != null)
                        message.memoryLimit = Number(object.memoryLimit);
                    if (object.accelerator != null) {
                        if (typeof object.accelerator !== "object")
                            throw TypeError(".ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.accelerator: object expected");
                        message.accelerator = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.fromObject(object.accelerator);
                    }
                    return message;
                };

                /**
                 * Creates a plain object from a ResourceSpec message. Also converts values to other types if specified.
                 * @function toObject
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec} message ResourceSpec
                 * @param {$protobuf.IConversionOptions} [options] Conversion options
                 * @returns {Object.<string,*>} Plain object
                 */
                ResourceSpec.toObject = function toObject(message, options) {
                    if (!options)
                        options = {};
                    var object = {};
                    if (options.defaults) {
                        object.cpuLimit = 0;
                        object.memoryLimit = 0;
                        object.accelerator = null;
                    }
                    if (message.cpuLimit != null && message.hasOwnProperty("cpuLimit"))
                        object.cpuLimit = options.json && !isFinite(message.cpuLimit) ? String(message.cpuLimit) : message.cpuLimit;
                    if (message.memoryLimit != null && message.hasOwnProperty("memoryLimit"))
                        object.memoryLimit = options.json && !isFinite(message.memoryLimit) ? String(message.memoryLimit) : message.memoryLimit;
                    if (message.accelerator != null && message.hasOwnProperty("accelerator"))
                        object.accelerator = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.toObject(message.accelerator, options);
                    return object;
                };

                /**
                 * Converts this ResourceSpec to JSON.
                 * @function toJSON
                 * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                 * @instance
                 * @returns {Object.<string,*>} JSON object
                 */
                ResourceSpec.prototype.toJSON = function toJSON() {
                    return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                };

                ResourceSpec.AcceleratorConfig = (function() {

                    /**
                     * Properties of an AcceleratorConfig.
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                     * @interface IAcceleratorConfig
                     * @property {string|null} [type] AcceleratorConfig type
                     * @property {number|Long|null} [count] AcceleratorConfig count
                     */

                    /**
                     * Constructs a new AcceleratorConfig.
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec
                     * @classdesc Represents an AcceleratorConfig.
                     * @implements IAcceleratorConfig
                     * @constructor
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig=} [properties] Properties to set
                     */
                    function AcceleratorConfig(properties) {
                        if (properties)
                            for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                                if (properties[keys[i]] != null)
                                    this[keys[i]] = properties[keys[i]];
                    }

                    /**
                     * AcceleratorConfig type.
                     * @member {string} type
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @instance
                     */
                    AcceleratorConfig.prototype.type = "";

                    /**
                     * AcceleratorConfig count.
                     * @member {number|Long} count
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @instance
                     */
                    AcceleratorConfig.prototype.count = $util.Long ? $util.Long.fromBits(0,0,false) : 0;

                    /**
                     * Creates a new AcceleratorConfig instance using the specified properties.
                     * @function create
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @static
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig=} [properties] Properties to set
                     * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} AcceleratorConfig instance
                     */
                    AcceleratorConfig.create = function create(properties) {
                        return new AcceleratorConfig(properties);
                    };

                    /**
                     * Encodes the specified AcceleratorConfig message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.verify|verify} messages.
                     * @function encode
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @static
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig} message AcceleratorConfig message or plain object to encode
                     * @param {$protobuf.Writer} [writer] Writer to encode to
                     * @returns {$protobuf.Writer} Writer
                     */
                    AcceleratorConfig.encode = function encode(message, writer) {
                        if (!writer)
                            writer = $Writer.create();
                        if (message.type != null && Object.hasOwnProperty.call(message, "type"))
                            writer.uint32(/* id 1, wireType 2 =*/10).string(message.type);
                        if (message.count != null && Object.hasOwnProperty.call(message, "count"))
                            writer.uint32(/* id 2, wireType 0 =*/16).int64(message.count);
                        return writer;
                    };

                    /**
                     * Encodes the specified AcceleratorConfig message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.verify|verify} messages.
                     * @function encodeDelimited
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @static
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig} message AcceleratorConfig message or plain object to encode
                     * @param {$protobuf.Writer} [writer] Writer to encode to
                     * @returns {$protobuf.Writer} Writer
                     */
                    AcceleratorConfig.encodeDelimited = function encodeDelimited(message, writer) {
                        return this.encode(message, writer).ldelim();
                    };

                    /**
                     * Decodes an AcceleratorConfig message from the specified reader or buffer.
                     * @function decode
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @static
                     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                     * @param {number} [length] Message length if known beforehand
                     * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} AcceleratorConfig
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    AcceleratorConfig.decode = function decode(reader, length) {
                        if (!(reader instanceof $Reader))
                            reader = $Reader.create(reader);
                        var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig();
                        while (reader.pos < end) {
                            var tag = reader.uint32();
                            switch (tag >>> 3) {
                            case 1:
                                message.type = reader.string();
                                break;
                            case 2:
                                message.count = reader.int64();
                                break;
                            default:
                                reader.skipType(tag & 7);
                                break;
                            }
                        }
                        return message;
                    };

                    /**
                     * Decodes an AcceleratorConfig message from the specified reader or buffer, length delimited.
                     * @function decodeDelimited
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @static
                     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                     * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} AcceleratorConfig
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    AcceleratorConfig.decodeDelimited = function decodeDelimited(reader) {
                        if (!(reader instanceof $Reader))
                            reader = new $Reader(reader);
                        return this.decode(reader, reader.uint32());
                    };

                    /**
                     * Verifies an AcceleratorConfig message.
                     * @function verify
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @static
                     * @param {Object.<string,*>} message Plain object to verify
                     * @returns {string|null} `null` if valid, otherwise the reason why it is not
                     */
                    AcceleratorConfig.verify = function verify(message) {
                        if (typeof message !== "object" || message === null)
                            return "object expected";
                        if (message.type != null && message.hasOwnProperty("type"))
                            if (!$util.isString(message.type))
                                return "type: string expected";
                        if (message.count != null && message.hasOwnProperty("count"))
                            if (!$util.isInteger(message.count) && !(message.count && $util.isInteger(message.count.low) && $util.isInteger(message.count.high)))
                                return "count: integer|Long expected";
                        return null;
                    };

                    /**
                     * Creates an AcceleratorConfig message from a plain object. Also converts values to their respective internal types.
                     * @function fromObject
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @static
                     * @param {Object.<string,*>} object Plain object
                     * @returns {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} AcceleratorConfig
                     */
                    AcceleratorConfig.fromObject = function fromObject(object) {
                        if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig)
                            return object;
                        var message = new $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig();
                        if (object.type != null)
                            message.type = String(object.type);
                        if (object.count != null)
                            if ($util.Long)
                                (message.count = $util.Long.fromValue(object.count)).unsigned = false;
                            else if (typeof object.count === "string")
                                message.count = parseInt(object.count, 10);
                            else if (typeof object.count === "number")
                                message.count = object.count;
                            else if (typeof object.count === "object")
                                message.count = new $util.LongBits(object.count.low >>> 0, object.count.high >>> 0).toNumber();
                        return message;
                    };

                    /**
                     * Creates a plain object from an AcceleratorConfig message. Also converts values to other types if specified.
                     * @function toObject
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @static
                     * @param {ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig} message AcceleratorConfig
                     * @param {$protobuf.IConversionOptions} [options] Conversion options
                     * @returns {Object.<string,*>} Plain object
                     */
                    AcceleratorConfig.toObject = function toObject(message, options) {
                        if (!options)
                            options = {};
                        var object = {};
                        if (options.defaults) {
                            object.type = "";
                            if ($util.Long) {
                                var long = new $util.Long(0, 0, false);
                                object.count = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                            } else
                                object.count = options.longs === String ? "0" : 0;
                        }
                        if (message.type != null && message.hasOwnProperty("type"))
                            object.type = message.type;
                        if (message.count != null && message.hasOwnProperty("count"))
                            if (typeof message.count === "number")
                                object.count = options.longs === String ? String(message.count) : message.count;
                            else
                                object.count = options.longs === String ? $util.Long.prototype.toString.call(message.count) : options.longs === Number ? new $util.LongBits(message.count.low >>> 0, message.count.high >>> 0).toNumber() : message.count;
                        return object;
                    };

                    /**
                     * Converts this AcceleratorConfig to JSON.
                     * @function toJSON
                     * @memberof ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig
                     * @instance
                     * @returns {Object.<string,*>} JSON object
                     */
                    AcceleratorConfig.prototype.toJSON = function toJSON() {
                        return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                    };

                    return AcceleratorConfig;
                })();

                return ResourceSpec;
            })();

            return PipelineContainerSpec;
        })();

        PipelineDeploymentConfig.ImporterSpec = (function() {

            /**
             * Properties of an ImporterSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @interface IImporterSpec
             * @property {ml_pipelines.IValueOrRuntimeParameter|null} [artifactUri] ImporterSpec artifactUri
             * @property {ml_pipelines.IArtifactTypeSchema|null} [typeSchema] ImporterSpec typeSchema
             * @property {Object.<string,ml_pipelines.IValueOrRuntimeParameter>|null} [properties] ImporterSpec properties
             * @property {Object.<string,ml_pipelines.IValueOrRuntimeParameter>|null} [customProperties] ImporterSpec customProperties
             * @property {google.protobuf.IStruct|null} [metadata] ImporterSpec metadata
             * @property {boolean|null} [reimport] ImporterSpec reimport
             */

            /**
             * Constructs a new ImporterSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @classdesc Represents an ImporterSpec.
             * @implements IImporterSpec
             * @constructor
             * @param {ml_pipelines.PipelineDeploymentConfig.IImporterSpec=} [properties] Properties to set
             */
            function ImporterSpec(properties) {
                this.properties = {};
                this.customProperties = {};
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ImporterSpec artifactUri.
             * @member {ml_pipelines.IValueOrRuntimeParameter|null|undefined} artifactUri
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @instance
             */
            ImporterSpec.prototype.artifactUri = null;

            /**
             * ImporterSpec typeSchema.
             * @member {ml_pipelines.IArtifactTypeSchema|null|undefined} typeSchema
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @instance
             */
            ImporterSpec.prototype.typeSchema = null;

            /**
             * ImporterSpec properties.
             * @member {Object.<string,ml_pipelines.IValueOrRuntimeParameter>} properties
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @instance
             */
            ImporterSpec.prototype.properties = $util.emptyObject;

            /**
             * ImporterSpec customProperties.
             * @member {Object.<string,ml_pipelines.IValueOrRuntimeParameter>} customProperties
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @instance
             */
            ImporterSpec.prototype.customProperties = $util.emptyObject;

            /**
             * ImporterSpec metadata.
             * @member {google.protobuf.IStruct|null|undefined} metadata
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @instance
             */
            ImporterSpec.prototype.metadata = null;

            /**
             * ImporterSpec reimport.
             * @member {boolean} reimport
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @instance
             */
            ImporterSpec.prototype.reimport = false;

            /**
             * Creates a new ImporterSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IImporterSpec=} [properties] Properties to set
             * @returns {ml_pipelines.PipelineDeploymentConfig.ImporterSpec} ImporterSpec instance
             */
            ImporterSpec.create = function create(properties) {
                return new ImporterSpec(properties);
            };

            /**
             * Encodes the specified ImporterSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ImporterSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IImporterSpec} message ImporterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ImporterSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.artifactUri != null && Object.hasOwnProperty.call(message, "artifactUri"))
                    $root.ml_pipelines.ValueOrRuntimeParameter.encode(message.artifactUri, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.typeSchema != null && Object.hasOwnProperty.call(message, "typeSchema"))
                    $root.ml_pipelines.ArtifactTypeSchema.encode(message.typeSchema, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.properties != null && Object.hasOwnProperty.call(message, "properties"))
                    for (var keys = Object.keys(message.properties), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 3, wireType 2 =*/26).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.ValueOrRuntimeParameter.encode(message.properties[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                if (message.customProperties != null && Object.hasOwnProperty.call(message, "customProperties"))
                    for (var keys = Object.keys(message.customProperties), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 4, wireType 2 =*/34).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.ValueOrRuntimeParameter.encode(message.customProperties[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                if (message.reimport != null && Object.hasOwnProperty.call(message, "reimport"))
                    writer.uint32(/* id 5, wireType 0 =*/40).bool(message.reimport);
                if (message.metadata != null && Object.hasOwnProperty.call(message, "metadata"))
                    $root.google.protobuf.Struct.encode(message.metadata, writer.uint32(/* id 6, wireType 2 =*/50).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ImporterSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ImporterSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IImporterSpec} message ImporterSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ImporterSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an ImporterSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.PipelineDeploymentConfig.ImporterSpec} ImporterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ImporterSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.ImporterSpec(), key, value;
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.artifactUri = $root.ml_pipelines.ValueOrRuntimeParameter.decode(reader, reader.uint32());
                        break;
                    case 2:
                        message.typeSchema = $root.ml_pipelines.ArtifactTypeSchema.decode(reader, reader.uint32());
                        break;
                    case 3:
                        if (message.properties === $util.emptyObject)
                            message.properties = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.ValueOrRuntimeParameter.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.properties[key] = value;
                        break;
                    case 4:
                        if (message.customProperties === $util.emptyObject)
                            message.customProperties = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.ValueOrRuntimeParameter.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.customProperties[key] = value;
                        break;
                    case 6:
                        message.metadata = $root.google.protobuf.Struct.decode(reader, reader.uint32());
                        break;
                    case 5:
                        message.reimport = reader.bool();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an ImporterSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.PipelineDeploymentConfig.ImporterSpec} ImporterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ImporterSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an ImporterSpec message.
             * @function verify
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ImporterSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.artifactUri != null && message.hasOwnProperty("artifactUri")) {
                    var error = $root.ml_pipelines.ValueOrRuntimeParameter.verify(message.artifactUri);
                    if (error)
                        return "artifactUri." + error;
                }
                if (message.typeSchema != null && message.hasOwnProperty("typeSchema")) {
                    var error = $root.ml_pipelines.ArtifactTypeSchema.verify(message.typeSchema);
                    if (error)
                        return "typeSchema." + error;
                }
                if (message.properties != null && message.hasOwnProperty("properties")) {
                    if (!$util.isObject(message.properties))
                        return "properties: object expected";
                    var key = Object.keys(message.properties);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.ValueOrRuntimeParameter.verify(message.properties[key[i]]);
                        if (error)
                            return "properties." + error;
                    }
                }
                if (message.customProperties != null && message.hasOwnProperty("customProperties")) {
                    if (!$util.isObject(message.customProperties))
                        return "customProperties: object expected";
                    var key = Object.keys(message.customProperties);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.ValueOrRuntimeParameter.verify(message.customProperties[key[i]]);
                        if (error)
                            return "customProperties." + error;
                    }
                }
                if (message.metadata != null && message.hasOwnProperty("metadata")) {
                    var error = $root.google.protobuf.Struct.verify(message.metadata);
                    if (error)
                        return "metadata." + error;
                }
                if (message.reimport != null && message.hasOwnProperty("reimport"))
                    if (typeof message.reimport !== "boolean")
                        return "reimport: boolean expected";
                return null;
            };

            /**
             * Creates an ImporterSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.PipelineDeploymentConfig.ImporterSpec} ImporterSpec
             */
            ImporterSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.ImporterSpec)
                    return object;
                var message = new $root.ml_pipelines.PipelineDeploymentConfig.ImporterSpec();
                if (object.artifactUri != null) {
                    if (typeof object.artifactUri !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ImporterSpec.artifactUri: object expected");
                    message.artifactUri = $root.ml_pipelines.ValueOrRuntimeParameter.fromObject(object.artifactUri);
                }
                if (object.typeSchema != null) {
                    if (typeof object.typeSchema !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ImporterSpec.typeSchema: object expected");
                    message.typeSchema = $root.ml_pipelines.ArtifactTypeSchema.fromObject(object.typeSchema);
                }
                if (object.properties) {
                    if (typeof object.properties !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ImporterSpec.properties: object expected");
                    message.properties = {};
                    for (var keys = Object.keys(object.properties), i = 0; i < keys.length; ++i) {
                        if (typeof object.properties[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ImporterSpec.properties: object expected");
                        message.properties[keys[i]] = $root.ml_pipelines.ValueOrRuntimeParameter.fromObject(object.properties[keys[i]]);
                    }
                }
                if (object.customProperties) {
                    if (typeof object.customProperties !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ImporterSpec.customProperties: object expected");
                    message.customProperties = {};
                    for (var keys = Object.keys(object.customProperties), i = 0; i < keys.length; ++i) {
                        if (typeof object.customProperties[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ImporterSpec.customProperties: object expected");
                        message.customProperties[keys[i]] = $root.ml_pipelines.ValueOrRuntimeParameter.fromObject(object.customProperties[keys[i]]);
                    }
                }
                if (object.metadata != null) {
                    if (typeof object.metadata !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ImporterSpec.metadata: object expected");
                    message.metadata = $root.google.protobuf.Struct.fromObject(object.metadata);
                }
                if (object.reimport != null)
                    message.reimport = Boolean(object.reimport);
                return message;
            };

            /**
             * Creates a plain object from an ImporterSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.ImporterSpec} message ImporterSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ImporterSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.objects || options.defaults) {
                    object.properties = {};
                    object.customProperties = {};
                }
                if (options.defaults) {
                    object.artifactUri = null;
                    object.typeSchema = null;
                    object.reimport = false;
                    object.metadata = null;
                }
                if (message.artifactUri != null && message.hasOwnProperty("artifactUri"))
                    object.artifactUri = $root.ml_pipelines.ValueOrRuntimeParameter.toObject(message.artifactUri, options);
                if (message.typeSchema != null && message.hasOwnProperty("typeSchema"))
                    object.typeSchema = $root.ml_pipelines.ArtifactTypeSchema.toObject(message.typeSchema, options);
                var keys2;
                if (message.properties && (keys2 = Object.keys(message.properties)).length) {
                    object.properties = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.properties[keys2[j]] = $root.ml_pipelines.ValueOrRuntimeParameter.toObject(message.properties[keys2[j]], options);
                }
                if (message.customProperties && (keys2 = Object.keys(message.customProperties)).length) {
                    object.customProperties = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.customProperties[keys2[j]] = $root.ml_pipelines.ValueOrRuntimeParameter.toObject(message.customProperties[keys2[j]], options);
                }
                if (message.reimport != null && message.hasOwnProperty("reimport"))
                    object.reimport = message.reimport;
                if (message.metadata != null && message.hasOwnProperty("metadata"))
                    object.metadata = $root.google.protobuf.Struct.toObject(message.metadata, options);
                return object;
            };

            /**
             * Converts this ImporterSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.PipelineDeploymentConfig.ImporterSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ImporterSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ImporterSpec;
        })();

        PipelineDeploymentConfig.ResolverSpec = (function() {

            /**
             * Properties of a ResolverSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @interface IResolverSpec
             * @property {Object.<string,ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec>|null} [outputArtifactQueries] ResolverSpec outputArtifactQueries
             */

            /**
             * Constructs a new ResolverSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @classdesc Represents a ResolverSpec.
             * @implements IResolverSpec
             * @constructor
             * @param {ml_pipelines.PipelineDeploymentConfig.IResolverSpec=} [properties] Properties to set
             */
            function ResolverSpec(properties) {
                this.outputArtifactQueries = {};
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ResolverSpec outputArtifactQueries.
             * @member {Object.<string,ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec>} outputArtifactQueries
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @instance
             */
            ResolverSpec.prototype.outputArtifactQueries = $util.emptyObject;

            /**
             * Creates a new ResolverSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IResolverSpec=} [properties] Properties to set
             * @returns {ml_pipelines.PipelineDeploymentConfig.ResolverSpec} ResolverSpec instance
             */
            ResolverSpec.create = function create(properties) {
                return new ResolverSpec(properties);
            };

            /**
             * Encodes the specified ResolverSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ResolverSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IResolverSpec} message ResolverSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ResolverSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.outputArtifactQueries != null && Object.hasOwnProperty.call(message, "outputArtifactQueries"))
                    for (var keys = Object.keys(message.outputArtifactQueries), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.encode(message.outputArtifactQueries[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                return writer;
            };

            /**
             * Encodes the specified ResolverSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ResolverSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IResolverSpec} message ResolverSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ResolverSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ResolverSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.PipelineDeploymentConfig.ResolverSpec} ResolverSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ResolverSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec(), key, value;
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        if (message.outputArtifactQueries === $util.emptyObject)
                            message.outputArtifactQueries = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.outputArtifactQueries[key] = value;
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ResolverSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.PipelineDeploymentConfig.ResolverSpec} ResolverSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ResolverSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ResolverSpec message.
             * @function verify
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ResolverSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.outputArtifactQueries != null && message.hasOwnProperty("outputArtifactQueries")) {
                    if (!$util.isObject(message.outputArtifactQueries))
                        return "outputArtifactQueries: object expected";
                    var key = Object.keys(message.outputArtifactQueries);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.verify(message.outputArtifactQueries[key[i]]);
                        if (error)
                            return "outputArtifactQueries." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a ResolverSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.PipelineDeploymentConfig.ResolverSpec} ResolverSpec
             */
            ResolverSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec)
                    return object;
                var message = new $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec();
                if (object.outputArtifactQueries) {
                    if (typeof object.outputArtifactQueries !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ResolverSpec.outputArtifactQueries: object expected");
                    message.outputArtifactQueries = {};
                    for (var keys = Object.keys(object.outputArtifactQueries), i = 0; i < keys.length; ++i) {
                        if (typeof object.outputArtifactQueries[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ResolverSpec.outputArtifactQueries: object expected");
                        message.outputArtifactQueries[keys[i]] = $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.fromObject(object.outputArtifactQueries[keys[i]]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a ResolverSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.ResolverSpec} message ResolverSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ResolverSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.objects || options.defaults)
                    object.outputArtifactQueries = {};
                var keys2;
                if (message.outputArtifactQueries && (keys2 = Object.keys(message.outputArtifactQueries)).length) {
                    object.outputArtifactQueries = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.outputArtifactQueries[keys2[j]] = $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.toObject(message.outputArtifactQueries[keys2[j]], options);
                }
                return object;
            };

            /**
             * Converts this ResolverSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ResolverSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            ResolverSpec.ArtifactQuerySpec = (function() {

                /**
                 * Properties of an ArtifactQuerySpec.
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
                 * @interface IArtifactQuerySpec
                 * @property {string|null} [filter] ArtifactQuerySpec filter
                 * @property {number|null} [limit] ArtifactQuerySpec limit
                 */

                /**
                 * Constructs a new ArtifactQuerySpec.
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec
                 * @classdesc Represents an ArtifactQuerySpec.
                 * @implements IArtifactQuerySpec
                 * @constructor
                 * @param {ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec=} [properties] Properties to set
                 */
                function ArtifactQuerySpec(properties) {
                    if (properties)
                        for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null)
                                this[keys[i]] = properties[keys[i]];
                }

                /**
                 * ArtifactQuerySpec filter.
                 * @member {string} filter
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @instance
                 */
                ArtifactQuerySpec.prototype.filter = "";

                /**
                 * ArtifactQuerySpec limit.
                 * @member {number} limit
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @instance
                 */
                ArtifactQuerySpec.prototype.limit = 0;

                /**
                 * Creates a new ArtifactQuerySpec instance using the specified properties.
                 * @function create
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec=} [properties] Properties to set
                 * @returns {ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} ArtifactQuerySpec instance
                 */
                ArtifactQuerySpec.create = function create(properties) {
                    return new ArtifactQuerySpec(properties);
                };

                /**
                 * Encodes the specified ArtifactQuerySpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.verify|verify} messages.
                 * @function encode
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec} message ArtifactQuerySpec message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                ArtifactQuerySpec.encode = function encode(message, writer) {
                    if (!writer)
                        writer = $Writer.create();
                    if (message.filter != null && Object.hasOwnProperty.call(message, "filter"))
                        writer.uint32(/* id 1, wireType 2 =*/10).string(message.filter);
                    if (message.limit != null && Object.hasOwnProperty.call(message, "limit"))
                        writer.uint32(/* id 2, wireType 0 =*/16).int32(message.limit);
                    return writer;
                };

                /**
                 * Encodes the specified ArtifactQuerySpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.verify|verify} messages.
                 * @function encodeDelimited
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec} message ArtifactQuerySpec message or plain object to encode
                 * @param {$protobuf.Writer} [writer] Writer to encode to
                 * @returns {$protobuf.Writer} Writer
                 */
                ArtifactQuerySpec.encodeDelimited = function encodeDelimited(message, writer) {
                    return this.encode(message, writer).ldelim();
                };

                /**
                 * Decodes an ArtifactQuerySpec message from the specified reader or buffer.
                 * @function decode
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} ArtifactQuerySpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                ArtifactQuerySpec.decode = function decode(reader, length) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec();
                    while (reader.pos < end) {
                        var tag = reader.uint32();
                        switch (tag >>> 3) {
                        case 1:
                            message.filter = reader.string();
                            break;
                        case 2:
                            message.limit = reader.int32();
                            break;
                        default:
                            reader.skipType(tag & 7);
                            break;
                        }
                    }
                    return message;
                };

                /**
                 * Decodes an ArtifactQuerySpec message from the specified reader or buffer, length delimited.
                 * @function decodeDelimited
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @returns {ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} ArtifactQuerySpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                ArtifactQuerySpec.decodeDelimited = function decodeDelimited(reader) {
                    if (!(reader instanceof $Reader))
                        reader = new $Reader(reader);
                    return this.decode(reader, reader.uint32());
                };

                /**
                 * Verifies an ArtifactQuerySpec message.
                 * @function verify
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @static
                 * @param {Object.<string,*>} message Plain object to verify
                 * @returns {string|null} `null` if valid, otherwise the reason why it is not
                 */
                ArtifactQuerySpec.verify = function verify(message) {
                    if (typeof message !== "object" || message === null)
                        return "object expected";
                    if (message.filter != null && message.hasOwnProperty("filter"))
                        if (!$util.isString(message.filter))
                            return "filter: string expected";
                    if (message.limit != null && message.hasOwnProperty("limit"))
                        if (!$util.isInteger(message.limit))
                            return "limit: integer expected";
                    return null;
                };

                /**
                 * Creates an ArtifactQuerySpec message from a plain object. Also converts values to their respective internal types.
                 * @function fromObject
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @static
                 * @param {Object.<string,*>} object Plain object
                 * @returns {ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} ArtifactQuerySpec
                 */
                ArtifactQuerySpec.fromObject = function fromObject(object) {
                    if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec)
                        return object;
                    var message = new $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec();
                    if (object.filter != null)
                        message.filter = String(object.filter);
                    if (object.limit != null)
                        message.limit = object.limit | 0;
                    return message;
                };

                /**
                 * Creates a plain object from an ArtifactQuerySpec message. Also converts values to other types if specified.
                 * @function toObject
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @static
                 * @param {ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec} message ArtifactQuerySpec
                 * @param {$protobuf.IConversionOptions} [options] Conversion options
                 * @returns {Object.<string,*>} Plain object
                 */
                ArtifactQuerySpec.toObject = function toObject(message, options) {
                    if (!options)
                        options = {};
                    var object = {};
                    if (options.defaults) {
                        object.filter = "";
                        object.limit = 0;
                    }
                    if (message.filter != null && message.hasOwnProperty("filter"))
                        object.filter = message.filter;
                    if (message.limit != null && message.hasOwnProperty("limit"))
                        object.limit = message.limit;
                    return object;
                };

                /**
                 * Converts this ArtifactQuerySpec to JSON.
                 * @function toJSON
                 * @memberof ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec
                 * @instance
                 * @returns {Object.<string,*>} JSON object
                 */
                ArtifactQuerySpec.prototype.toJSON = function toJSON() {
                    return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                };

                return ArtifactQuerySpec;
            })();

            return ResolverSpec;
        })();

        PipelineDeploymentConfig.AIPlatformCustomJobSpec = (function() {

            /**
             * Properties of a AIPlatformCustomJobSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @interface IAIPlatformCustomJobSpec
             * @property {google.protobuf.IStruct|null} [customJob] AIPlatformCustomJobSpec customJob
             */

            /**
             * Constructs a new AIPlatformCustomJobSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @classdesc Represents a AIPlatformCustomJobSpec.
             * @implements IAIPlatformCustomJobSpec
             * @constructor
             * @param {ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec=} [properties] Properties to set
             */
            function AIPlatformCustomJobSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * AIPlatformCustomJobSpec customJob.
             * @member {google.protobuf.IStruct|null|undefined} customJob
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @instance
             */
            AIPlatformCustomJobSpec.prototype.customJob = null;

            /**
             * Creates a new AIPlatformCustomJobSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec=} [properties] Properties to set
             * @returns {ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} AIPlatformCustomJobSpec instance
             */
            AIPlatformCustomJobSpec.create = function create(properties) {
                return new AIPlatformCustomJobSpec(properties);
            };

            /**
             * Encodes the specified AIPlatformCustomJobSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec} message AIPlatformCustomJobSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            AIPlatformCustomJobSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.customJob != null && Object.hasOwnProperty.call(message, "customJob"))
                    $root.google.protobuf.Struct.encode(message.customJob, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified AIPlatformCustomJobSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec} message AIPlatformCustomJobSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            AIPlatformCustomJobSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a AIPlatformCustomJobSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} AIPlatformCustomJobSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            AIPlatformCustomJobSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.customJob = $root.google.protobuf.Struct.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a AIPlatformCustomJobSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} AIPlatformCustomJobSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            AIPlatformCustomJobSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a AIPlatformCustomJobSpec message.
             * @function verify
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            AIPlatformCustomJobSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.customJob != null && message.hasOwnProperty("customJob")) {
                    var error = $root.google.protobuf.Struct.verify(message.customJob);
                    if (error)
                        return "customJob." + error;
                }
                return null;
            };

            /**
             * Creates a AIPlatformCustomJobSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} AIPlatformCustomJobSpec
             */
            AIPlatformCustomJobSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec)
                    return object;
                var message = new $root.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec();
                if (object.customJob != null) {
                    if (typeof object.customJob !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.customJob: object expected");
                    message.customJob = $root.google.protobuf.Struct.fromObject(object.customJob);
                }
                return message;
            };

            /**
             * Creates a plain object from a AIPlatformCustomJobSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec} message AIPlatformCustomJobSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            AIPlatformCustomJobSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.customJob = null;
                if (message.customJob != null && message.hasOwnProperty("customJob"))
                    object.customJob = $root.google.protobuf.Struct.toObject(message.customJob, options);
                return object;
            };

            /**
             * Converts this AIPlatformCustomJobSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            AIPlatformCustomJobSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return AIPlatformCustomJobSpec;
        })();

        PipelineDeploymentConfig.ExecutorSpec = (function() {

            /**
             * Properties of an ExecutorSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @interface IExecutorSpec
             * @property {ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec|null} [container] ExecutorSpec container
             * @property {ml_pipelines.PipelineDeploymentConfig.IImporterSpec|null} [importer] ExecutorSpec importer
             * @property {ml_pipelines.PipelineDeploymentConfig.IResolverSpec|null} [resolver] ExecutorSpec resolver
             * @property {ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec|null} [customJob] ExecutorSpec customJob
             */

            /**
             * Constructs a new ExecutorSpec.
             * @memberof ml_pipelines.PipelineDeploymentConfig
             * @classdesc Represents an ExecutorSpec.
             * @implements IExecutorSpec
             * @constructor
             * @param {ml_pipelines.PipelineDeploymentConfig.IExecutorSpec=} [properties] Properties to set
             */
            function ExecutorSpec(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ExecutorSpec container.
             * @member {ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec|null|undefined} container
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @instance
             */
            ExecutorSpec.prototype.container = null;

            /**
             * ExecutorSpec importer.
             * @member {ml_pipelines.PipelineDeploymentConfig.IImporterSpec|null|undefined} importer
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @instance
             */
            ExecutorSpec.prototype.importer = null;

            /**
             * ExecutorSpec resolver.
             * @member {ml_pipelines.PipelineDeploymentConfig.IResolverSpec|null|undefined} resolver
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @instance
             */
            ExecutorSpec.prototype.resolver = null;

            /**
             * ExecutorSpec customJob.
             * @member {ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec|null|undefined} customJob
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @instance
             */
            ExecutorSpec.prototype.customJob = null;

            // OneOf field names bound to virtual getters and setters
            var $oneOfFields;

            /**
             * ExecutorSpec spec.
             * @member {"container"|"importer"|"resolver"|"customJob"|undefined} spec
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @instance
             */
            Object.defineProperty(ExecutorSpec.prototype, "spec", {
                get: $util.oneOfGetter($oneOfFields = ["container", "importer", "resolver", "customJob"]),
                set: $util.oneOfSetter($oneOfFields)
            });

            /**
             * Creates a new ExecutorSpec instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IExecutorSpec=} [properties] Properties to set
             * @returns {ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} ExecutorSpec instance
             */
            ExecutorSpec.create = function create(properties) {
                return new ExecutorSpec(properties);
            };

            /**
             * Encodes the specified ExecutorSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IExecutorSpec} message ExecutorSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ExecutorSpec.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.container != null && Object.hasOwnProperty.call(message, "container"))
                    $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.encode(message.container, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.importer != null && Object.hasOwnProperty.call(message, "importer"))
                    $root.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.encode(message.importer, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.resolver != null && Object.hasOwnProperty.call(message, "resolver"))
                    $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.encode(message.resolver, writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.customJob != null && Object.hasOwnProperty.call(message, "customJob"))
                    $root.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.encode(message.customJob, writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ExecutorSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.IExecutorSpec} message ExecutorSpec message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ExecutorSpec.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an ExecutorSpec message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} ExecutorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ExecutorSpec.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.container = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.decode(reader, reader.uint32());
                        break;
                    case 2:
                        message.importer = $root.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.decode(reader, reader.uint32());
                        break;
                    case 3:
                        message.resolver = $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.decode(reader, reader.uint32());
                        break;
                    case 4:
                        message.customJob = $root.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an ExecutorSpec message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} ExecutorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ExecutorSpec.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an ExecutorSpec message.
             * @function verify
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ExecutorSpec.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                var properties = {};
                if (message.container != null && message.hasOwnProperty("container")) {
                    properties.spec = 1;
                    {
                        var error = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.verify(message.container);
                        if (error)
                            return "container." + error;
                    }
                }
                if (message.importer != null && message.hasOwnProperty("importer")) {
                    if (properties.spec === 1)
                        return "spec: multiple values";
                    properties.spec = 1;
                    {
                        var error = $root.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.verify(message.importer);
                        if (error)
                            return "importer." + error;
                    }
                }
                if (message.resolver != null && message.hasOwnProperty("resolver")) {
                    if (properties.spec === 1)
                        return "spec: multiple values";
                    properties.spec = 1;
                    {
                        var error = $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.verify(message.resolver);
                        if (error)
                            return "resolver." + error;
                    }
                }
                if (message.customJob != null && message.hasOwnProperty("customJob")) {
                    if (properties.spec === 1)
                        return "spec: multiple values";
                    properties.spec = 1;
                    {
                        var error = $root.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.verify(message.customJob);
                        if (error)
                            return "customJob." + error;
                    }
                }
                return null;
            };

            /**
             * Creates an ExecutorSpec message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} ExecutorSpec
             */
            ExecutorSpec.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec)
                    return object;
                var message = new $root.ml_pipelines.PipelineDeploymentConfig.ExecutorSpec();
                if (object.container != null) {
                    if (typeof object.container !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.container: object expected");
                    message.container = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.fromObject(object.container);
                }
                if (object.importer != null) {
                    if (typeof object.importer !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.importer: object expected");
                    message.importer = $root.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.fromObject(object.importer);
                }
                if (object.resolver != null) {
                    if (typeof object.resolver !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.resolver: object expected");
                    message.resolver = $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.fromObject(object.resolver);
                }
                if (object.customJob != null) {
                    if (typeof object.customJob !== "object")
                        throw TypeError(".ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.customJob: object expected");
                    message.customJob = $root.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.fromObject(object.customJob);
                }
                return message;
            };

            /**
             * Creates a plain object from an ExecutorSpec message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @static
             * @param {ml_pipelines.PipelineDeploymentConfig.ExecutorSpec} message ExecutorSpec
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ExecutorSpec.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (message.container != null && message.hasOwnProperty("container")) {
                    object.container = $root.ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.toObject(message.container, options);
                    if (options.oneofs)
                        object.spec = "container";
                }
                if (message.importer != null && message.hasOwnProperty("importer")) {
                    object.importer = $root.ml_pipelines.PipelineDeploymentConfig.ImporterSpec.toObject(message.importer, options);
                    if (options.oneofs)
                        object.spec = "importer";
                }
                if (message.resolver != null && message.hasOwnProperty("resolver")) {
                    object.resolver = $root.ml_pipelines.PipelineDeploymentConfig.ResolverSpec.toObject(message.resolver, options);
                    if (options.oneofs)
                        object.spec = "resolver";
                }
                if (message.customJob != null && message.hasOwnProperty("customJob")) {
                    object.customJob = $root.ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.toObject(message.customJob, options);
                    if (options.oneofs)
                        object.spec = "customJob";
                }
                return object;
            };

            /**
             * Converts this ExecutorSpec to JSON.
             * @function toJSON
             * @memberof ml_pipelines.PipelineDeploymentConfig.ExecutorSpec
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ExecutorSpec.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ExecutorSpec;
        })();

        return PipelineDeploymentConfig;
    })();

    ml_pipelines.Value = (function() {

        /**
         * Properties of a Value.
         * @memberof ml_pipelines
         * @interface IValue
         * @property {number|Long|null} [intValue] Value intValue
         * @property {number|null} [doubleValue] Value doubleValue
         * @property {string|null} [stringValue] Value stringValue
         */

        /**
         * Constructs a new Value.
         * @memberof ml_pipelines
         * @classdesc Represents a Value.
         * @implements IValue
         * @constructor
         * @param {ml_pipelines.IValue=} [properties] Properties to set
         */
        function Value(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * Value intValue.
         * @member {number|Long|null|undefined} intValue
         * @memberof ml_pipelines.Value
         * @instance
         */
        Value.prototype.intValue = null;

        /**
         * Value doubleValue.
         * @member {number|null|undefined} doubleValue
         * @memberof ml_pipelines.Value
         * @instance
         */
        Value.prototype.doubleValue = null;

        /**
         * Value stringValue.
         * @member {string|null|undefined} stringValue
         * @memberof ml_pipelines.Value
         * @instance
         */
        Value.prototype.stringValue = null;

        // OneOf field names bound to virtual getters and setters
        var $oneOfFields;

        /**
         * Value value.
         * @member {"intValue"|"doubleValue"|"stringValue"|undefined} value
         * @memberof ml_pipelines.Value
         * @instance
         */
        Object.defineProperty(Value.prototype, "value", {
            get: $util.oneOfGetter($oneOfFields = ["intValue", "doubleValue", "stringValue"]),
            set: $util.oneOfSetter($oneOfFields)
        });

        /**
         * Creates a new Value instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.Value
         * @static
         * @param {ml_pipelines.IValue=} [properties] Properties to set
         * @returns {ml_pipelines.Value} Value instance
         */
        Value.create = function create(properties) {
            return new Value(properties);
        };

        /**
         * Encodes the specified Value message. Does not implicitly {@link ml_pipelines.Value.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.Value
         * @static
         * @param {ml_pipelines.IValue} message Value message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        Value.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.intValue != null && Object.hasOwnProperty.call(message, "intValue"))
                writer.uint32(/* id 1, wireType 0 =*/8).int64(message.intValue);
            if (message.doubleValue != null && Object.hasOwnProperty.call(message, "doubleValue"))
                writer.uint32(/* id 2, wireType 1 =*/17).double(message.doubleValue);
            if (message.stringValue != null && Object.hasOwnProperty.call(message, "stringValue"))
                writer.uint32(/* id 3, wireType 2 =*/26).string(message.stringValue);
            return writer;
        };

        /**
         * Encodes the specified Value message, length delimited. Does not implicitly {@link ml_pipelines.Value.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.Value
         * @static
         * @param {ml_pipelines.IValue} message Value message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        Value.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a Value message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.Value
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.Value} Value
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        Value.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.Value();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.intValue = reader.int64();
                    break;
                case 2:
                    message.doubleValue = reader.double();
                    break;
                case 3:
                    message.stringValue = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a Value message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.Value
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.Value} Value
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        Value.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a Value message.
         * @function verify
         * @memberof ml_pipelines.Value
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        Value.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            var properties = {};
            if (message.intValue != null && message.hasOwnProperty("intValue")) {
                properties.value = 1;
                if (!$util.isInteger(message.intValue) && !(message.intValue && $util.isInteger(message.intValue.low) && $util.isInteger(message.intValue.high)))
                    return "intValue: integer|Long expected";
            }
            if (message.doubleValue != null && message.hasOwnProperty("doubleValue")) {
                if (properties.value === 1)
                    return "value: multiple values";
                properties.value = 1;
                if (typeof message.doubleValue !== "number")
                    return "doubleValue: number expected";
            }
            if (message.stringValue != null && message.hasOwnProperty("stringValue")) {
                if (properties.value === 1)
                    return "value: multiple values";
                properties.value = 1;
                if (!$util.isString(message.stringValue))
                    return "stringValue: string expected";
            }
            return null;
        };

        /**
         * Creates a Value message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.Value
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.Value} Value
         */
        Value.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.Value)
                return object;
            var message = new $root.ml_pipelines.Value();
            if (object.intValue != null)
                if ($util.Long)
                    (message.intValue = $util.Long.fromValue(object.intValue)).unsigned = false;
                else if (typeof object.intValue === "string")
                    message.intValue = parseInt(object.intValue, 10);
                else if (typeof object.intValue === "number")
                    message.intValue = object.intValue;
                else if (typeof object.intValue === "object")
                    message.intValue = new $util.LongBits(object.intValue.low >>> 0, object.intValue.high >>> 0).toNumber();
            if (object.doubleValue != null)
                message.doubleValue = Number(object.doubleValue);
            if (object.stringValue != null)
                message.stringValue = String(object.stringValue);
            return message;
        };

        /**
         * Creates a plain object from a Value message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.Value
         * @static
         * @param {ml_pipelines.Value} message Value
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        Value.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (message.intValue != null && message.hasOwnProperty("intValue")) {
                if (typeof message.intValue === "number")
                    object.intValue = options.longs === String ? String(message.intValue) : message.intValue;
                else
                    object.intValue = options.longs === String ? $util.Long.prototype.toString.call(message.intValue) : options.longs === Number ? new $util.LongBits(message.intValue.low >>> 0, message.intValue.high >>> 0).toNumber() : message.intValue;
                if (options.oneofs)
                    object.value = "intValue";
            }
            if (message.doubleValue != null && message.hasOwnProperty("doubleValue")) {
                object.doubleValue = options.json && !isFinite(message.doubleValue) ? String(message.doubleValue) : message.doubleValue;
                if (options.oneofs)
                    object.value = "doubleValue";
            }
            if (message.stringValue != null && message.hasOwnProperty("stringValue")) {
                object.stringValue = message.stringValue;
                if (options.oneofs)
                    object.value = "stringValue";
            }
            return object;
        };

        /**
         * Converts this Value to JSON.
         * @function toJSON
         * @memberof ml_pipelines.Value
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        Value.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return Value;
    })();

    ml_pipelines.RuntimeArtifact = (function() {

        /**
         * Properties of a RuntimeArtifact.
         * @memberof ml_pipelines
         * @interface IRuntimeArtifact
         * @property {string|null} [name] RuntimeArtifact name
         * @property {ml_pipelines.IArtifactTypeSchema|null} [type] RuntimeArtifact type
         * @property {string|null} [uri] RuntimeArtifact uri
         * @property {Object.<string,ml_pipelines.IValue>|null} [properties] RuntimeArtifact properties
         * @property {Object.<string,ml_pipelines.IValue>|null} [customProperties] RuntimeArtifact customProperties
         * @property {google.protobuf.IStruct|null} [metadata] RuntimeArtifact metadata
         */

        /**
         * Constructs a new RuntimeArtifact.
         * @memberof ml_pipelines
         * @classdesc Represents a RuntimeArtifact.
         * @implements IRuntimeArtifact
         * @constructor
         * @param {ml_pipelines.IRuntimeArtifact=} [properties] Properties to set
         */
        function RuntimeArtifact(properties) {
            this.properties = {};
            this.customProperties = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * RuntimeArtifact name.
         * @member {string} name
         * @memberof ml_pipelines.RuntimeArtifact
         * @instance
         */
        RuntimeArtifact.prototype.name = "";

        /**
         * RuntimeArtifact type.
         * @member {ml_pipelines.IArtifactTypeSchema|null|undefined} type
         * @memberof ml_pipelines.RuntimeArtifact
         * @instance
         */
        RuntimeArtifact.prototype.type = null;

        /**
         * RuntimeArtifact uri.
         * @member {string} uri
         * @memberof ml_pipelines.RuntimeArtifact
         * @instance
         */
        RuntimeArtifact.prototype.uri = "";

        /**
         * RuntimeArtifact properties.
         * @member {Object.<string,ml_pipelines.IValue>} properties
         * @memberof ml_pipelines.RuntimeArtifact
         * @instance
         */
        RuntimeArtifact.prototype.properties = $util.emptyObject;

        /**
         * RuntimeArtifact customProperties.
         * @member {Object.<string,ml_pipelines.IValue>} customProperties
         * @memberof ml_pipelines.RuntimeArtifact
         * @instance
         */
        RuntimeArtifact.prototype.customProperties = $util.emptyObject;

        /**
         * RuntimeArtifact metadata.
         * @member {google.protobuf.IStruct|null|undefined} metadata
         * @memberof ml_pipelines.RuntimeArtifact
         * @instance
         */
        RuntimeArtifact.prototype.metadata = null;

        /**
         * Creates a new RuntimeArtifact instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.RuntimeArtifact
         * @static
         * @param {ml_pipelines.IRuntimeArtifact=} [properties] Properties to set
         * @returns {ml_pipelines.RuntimeArtifact} RuntimeArtifact instance
         */
        RuntimeArtifact.create = function create(properties) {
            return new RuntimeArtifact(properties);
        };

        /**
         * Encodes the specified RuntimeArtifact message. Does not implicitly {@link ml_pipelines.RuntimeArtifact.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.RuntimeArtifact
         * @static
         * @param {ml_pipelines.IRuntimeArtifact} message RuntimeArtifact message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        RuntimeArtifact.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
            if (message.type != null && Object.hasOwnProperty.call(message, "type"))
                $root.ml_pipelines.ArtifactTypeSchema.encode(message.type, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
            if (message.uri != null && Object.hasOwnProperty.call(message, "uri"))
                writer.uint32(/* id 3, wireType 2 =*/26).string(message.uri);
            if (message.properties != null && Object.hasOwnProperty.call(message, "properties"))
                for (var keys = Object.keys(message.properties), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 4, wireType 2 =*/34).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.Value.encode(message.properties[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.customProperties != null && Object.hasOwnProperty.call(message, "customProperties"))
                for (var keys = Object.keys(message.customProperties), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 5, wireType 2 =*/42).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.Value.encode(message.customProperties[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.metadata != null && Object.hasOwnProperty.call(message, "metadata"))
                $root.google.protobuf.Struct.encode(message.metadata, writer.uint32(/* id 6, wireType 2 =*/50).fork()).ldelim();
            return writer;
        };

        /**
         * Encodes the specified RuntimeArtifact message, length delimited. Does not implicitly {@link ml_pipelines.RuntimeArtifact.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.RuntimeArtifact
         * @static
         * @param {ml_pipelines.IRuntimeArtifact} message RuntimeArtifact message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        RuntimeArtifact.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a RuntimeArtifact message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.RuntimeArtifact
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.RuntimeArtifact} RuntimeArtifact
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        RuntimeArtifact.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.RuntimeArtifact(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 2:
                    message.type = $root.ml_pipelines.ArtifactTypeSchema.decode(reader, reader.uint32());
                    break;
                case 3:
                    message.uri = reader.string();
                    break;
                case 4:
                    if (message.properties === $util.emptyObject)
                        message.properties = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.Value.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.properties[key] = value;
                    break;
                case 5:
                    if (message.customProperties === $util.emptyObject)
                        message.customProperties = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.Value.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.customProperties[key] = value;
                    break;
                case 6:
                    message.metadata = $root.google.protobuf.Struct.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a RuntimeArtifact message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.RuntimeArtifact
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.RuntimeArtifact} RuntimeArtifact
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        RuntimeArtifact.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a RuntimeArtifact message.
         * @function verify
         * @memberof ml_pipelines.RuntimeArtifact
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        RuntimeArtifact.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.name != null && message.hasOwnProperty("name"))
                if (!$util.isString(message.name))
                    return "name: string expected";
            if (message.type != null && message.hasOwnProperty("type")) {
                var error = $root.ml_pipelines.ArtifactTypeSchema.verify(message.type);
                if (error)
                    return "type." + error;
            }
            if (message.uri != null && message.hasOwnProperty("uri"))
                if (!$util.isString(message.uri))
                    return "uri: string expected";
            if (message.properties != null && message.hasOwnProperty("properties")) {
                if (!$util.isObject(message.properties))
                    return "properties: object expected";
                var key = Object.keys(message.properties);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.Value.verify(message.properties[key[i]]);
                    if (error)
                        return "properties." + error;
                }
            }
            if (message.customProperties != null && message.hasOwnProperty("customProperties")) {
                if (!$util.isObject(message.customProperties))
                    return "customProperties: object expected";
                var key = Object.keys(message.customProperties);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.Value.verify(message.customProperties[key[i]]);
                    if (error)
                        return "customProperties." + error;
                }
            }
            if (message.metadata != null && message.hasOwnProperty("metadata")) {
                var error = $root.google.protobuf.Struct.verify(message.metadata);
                if (error)
                    return "metadata." + error;
            }
            return null;
        };

        /**
         * Creates a RuntimeArtifact message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.RuntimeArtifact
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.RuntimeArtifact} RuntimeArtifact
         */
        RuntimeArtifact.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.RuntimeArtifact)
                return object;
            var message = new $root.ml_pipelines.RuntimeArtifact();
            if (object.name != null)
                message.name = String(object.name);
            if (object.type != null) {
                if (typeof object.type !== "object")
                    throw TypeError(".ml_pipelines.RuntimeArtifact.type: object expected");
                message.type = $root.ml_pipelines.ArtifactTypeSchema.fromObject(object.type);
            }
            if (object.uri != null)
                message.uri = String(object.uri);
            if (object.properties) {
                if (typeof object.properties !== "object")
                    throw TypeError(".ml_pipelines.RuntimeArtifact.properties: object expected");
                message.properties = {};
                for (var keys = Object.keys(object.properties), i = 0; i < keys.length; ++i) {
                    if (typeof object.properties[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.RuntimeArtifact.properties: object expected");
                    message.properties[keys[i]] = $root.ml_pipelines.Value.fromObject(object.properties[keys[i]]);
                }
            }
            if (object.customProperties) {
                if (typeof object.customProperties !== "object")
                    throw TypeError(".ml_pipelines.RuntimeArtifact.customProperties: object expected");
                message.customProperties = {};
                for (var keys = Object.keys(object.customProperties), i = 0; i < keys.length; ++i) {
                    if (typeof object.customProperties[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.RuntimeArtifact.customProperties: object expected");
                    message.customProperties[keys[i]] = $root.ml_pipelines.Value.fromObject(object.customProperties[keys[i]]);
                }
            }
            if (object.metadata != null) {
                if (typeof object.metadata !== "object")
                    throw TypeError(".ml_pipelines.RuntimeArtifact.metadata: object expected");
                message.metadata = $root.google.protobuf.Struct.fromObject(object.metadata);
            }
            return message;
        };

        /**
         * Creates a plain object from a RuntimeArtifact message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.RuntimeArtifact
         * @static
         * @param {ml_pipelines.RuntimeArtifact} message RuntimeArtifact
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        RuntimeArtifact.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults) {
                object.properties = {};
                object.customProperties = {};
            }
            if (options.defaults) {
                object.name = "";
                object.type = null;
                object.uri = "";
                object.metadata = null;
            }
            if (message.name != null && message.hasOwnProperty("name"))
                object.name = message.name;
            if (message.type != null && message.hasOwnProperty("type"))
                object.type = $root.ml_pipelines.ArtifactTypeSchema.toObject(message.type, options);
            if (message.uri != null && message.hasOwnProperty("uri"))
                object.uri = message.uri;
            var keys2;
            if (message.properties && (keys2 = Object.keys(message.properties)).length) {
                object.properties = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.properties[keys2[j]] = $root.ml_pipelines.Value.toObject(message.properties[keys2[j]], options);
            }
            if (message.customProperties && (keys2 = Object.keys(message.customProperties)).length) {
                object.customProperties = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.customProperties[keys2[j]] = $root.ml_pipelines.Value.toObject(message.customProperties[keys2[j]], options);
            }
            if (message.metadata != null && message.hasOwnProperty("metadata"))
                object.metadata = $root.google.protobuf.Struct.toObject(message.metadata, options);
            return object;
        };

        /**
         * Converts this RuntimeArtifact to JSON.
         * @function toJSON
         * @memberof ml_pipelines.RuntimeArtifact
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        RuntimeArtifact.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return RuntimeArtifact;
    })();

    ml_pipelines.ArtifactList = (function() {

        /**
         * Properties of an ArtifactList.
         * @memberof ml_pipelines
         * @interface IArtifactList
         * @property {Array.<ml_pipelines.IRuntimeArtifact>|null} [artifacts] ArtifactList artifacts
         */

        /**
         * Constructs a new ArtifactList.
         * @memberof ml_pipelines
         * @classdesc Represents an ArtifactList.
         * @implements IArtifactList
         * @constructor
         * @param {ml_pipelines.IArtifactList=} [properties] Properties to set
         */
        function ArtifactList(properties) {
            this.artifacts = [];
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ArtifactList artifacts.
         * @member {Array.<ml_pipelines.IRuntimeArtifact>} artifacts
         * @memberof ml_pipelines.ArtifactList
         * @instance
         */
        ArtifactList.prototype.artifacts = $util.emptyArray;

        /**
         * Creates a new ArtifactList instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ArtifactList
         * @static
         * @param {ml_pipelines.IArtifactList=} [properties] Properties to set
         * @returns {ml_pipelines.ArtifactList} ArtifactList instance
         */
        ArtifactList.create = function create(properties) {
            return new ArtifactList(properties);
        };

        /**
         * Encodes the specified ArtifactList message. Does not implicitly {@link ml_pipelines.ArtifactList.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ArtifactList
         * @static
         * @param {ml_pipelines.IArtifactList} message ArtifactList message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ArtifactList.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.artifacts != null && message.artifacts.length)
                for (var i = 0; i < message.artifacts.length; ++i)
                    $root.ml_pipelines.RuntimeArtifact.encode(message.artifacts[i], writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
            return writer;
        };

        /**
         * Encodes the specified ArtifactList message, length delimited. Does not implicitly {@link ml_pipelines.ArtifactList.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ArtifactList
         * @static
         * @param {ml_pipelines.IArtifactList} message ArtifactList message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ArtifactList.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes an ArtifactList message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ArtifactList
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ArtifactList} ArtifactList
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ArtifactList.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ArtifactList();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    if (!(message.artifacts && message.artifacts.length))
                        message.artifacts = [];
                    message.artifacts.push($root.ml_pipelines.RuntimeArtifact.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes an ArtifactList message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ArtifactList
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ArtifactList} ArtifactList
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ArtifactList.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies an ArtifactList message.
         * @function verify
         * @memberof ml_pipelines.ArtifactList
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ArtifactList.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.artifacts != null && message.hasOwnProperty("artifacts")) {
                if (!Array.isArray(message.artifacts))
                    return "artifacts: array expected";
                for (var i = 0; i < message.artifacts.length; ++i) {
                    var error = $root.ml_pipelines.RuntimeArtifact.verify(message.artifacts[i]);
                    if (error)
                        return "artifacts." + error;
                }
            }
            return null;
        };

        /**
         * Creates an ArtifactList message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ArtifactList
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ArtifactList} ArtifactList
         */
        ArtifactList.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ArtifactList)
                return object;
            var message = new $root.ml_pipelines.ArtifactList();
            if (object.artifacts) {
                if (!Array.isArray(object.artifacts))
                    throw TypeError(".ml_pipelines.ArtifactList.artifacts: array expected");
                message.artifacts = [];
                for (var i = 0; i < object.artifacts.length; ++i) {
                    if (typeof object.artifacts[i] !== "object")
                        throw TypeError(".ml_pipelines.ArtifactList.artifacts: object expected");
                    message.artifacts[i] = $root.ml_pipelines.RuntimeArtifact.fromObject(object.artifacts[i]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from an ArtifactList message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ArtifactList
         * @static
         * @param {ml_pipelines.ArtifactList} message ArtifactList
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ArtifactList.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.arrays || options.defaults)
                object.artifacts = [];
            if (message.artifacts && message.artifacts.length) {
                object.artifacts = [];
                for (var j = 0; j < message.artifacts.length; ++j)
                    object.artifacts[j] = $root.ml_pipelines.RuntimeArtifact.toObject(message.artifacts[j], options);
            }
            return object;
        };

        /**
         * Converts this ArtifactList to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ArtifactList
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ArtifactList.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return ArtifactList;
    })();

    ml_pipelines.ExecutorInput = (function() {

        /**
         * Properties of an ExecutorInput.
         * @memberof ml_pipelines
         * @interface IExecutorInput
         * @property {ml_pipelines.ExecutorInput.IInputs|null} [inputs] ExecutorInput inputs
         * @property {ml_pipelines.ExecutorInput.IOutputs|null} [outputs] ExecutorInput outputs
         */

        /**
         * Constructs a new ExecutorInput.
         * @memberof ml_pipelines
         * @classdesc Represents an ExecutorInput.
         * @implements IExecutorInput
         * @constructor
         * @param {ml_pipelines.IExecutorInput=} [properties] Properties to set
         */
        function ExecutorInput(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ExecutorInput inputs.
         * @member {ml_pipelines.ExecutorInput.IInputs|null|undefined} inputs
         * @memberof ml_pipelines.ExecutorInput
         * @instance
         */
        ExecutorInput.prototype.inputs = null;

        /**
         * ExecutorInput outputs.
         * @member {ml_pipelines.ExecutorInput.IOutputs|null|undefined} outputs
         * @memberof ml_pipelines.ExecutorInput
         * @instance
         */
        ExecutorInput.prototype.outputs = null;

        /**
         * Creates a new ExecutorInput instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ExecutorInput
         * @static
         * @param {ml_pipelines.IExecutorInput=} [properties] Properties to set
         * @returns {ml_pipelines.ExecutorInput} ExecutorInput instance
         */
        ExecutorInput.create = function create(properties) {
            return new ExecutorInput(properties);
        };

        /**
         * Encodes the specified ExecutorInput message. Does not implicitly {@link ml_pipelines.ExecutorInput.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ExecutorInput
         * @static
         * @param {ml_pipelines.IExecutorInput} message ExecutorInput message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ExecutorInput.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.inputs != null && Object.hasOwnProperty.call(message, "inputs"))
                $root.ml_pipelines.ExecutorInput.Inputs.encode(message.inputs, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
            if (message.outputs != null && Object.hasOwnProperty.call(message, "outputs"))
                $root.ml_pipelines.ExecutorInput.Outputs.encode(message.outputs, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
            return writer;
        };

        /**
         * Encodes the specified ExecutorInput message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorInput.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ExecutorInput
         * @static
         * @param {ml_pipelines.IExecutorInput} message ExecutorInput message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ExecutorInput.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes an ExecutorInput message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ExecutorInput
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ExecutorInput} ExecutorInput
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ExecutorInput.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ExecutorInput();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.inputs = $root.ml_pipelines.ExecutorInput.Inputs.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.outputs = $root.ml_pipelines.ExecutorInput.Outputs.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes an ExecutorInput message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ExecutorInput
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ExecutorInput} ExecutorInput
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ExecutorInput.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies an ExecutorInput message.
         * @function verify
         * @memberof ml_pipelines.ExecutorInput
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ExecutorInput.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.inputs != null && message.hasOwnProperty("inputs")) {
                var error = $root.ml_pipelines.ExecutorInput.Inputs.verify(message.inputs);
                if (error)
                    return "inputs." + error;
            }
            if (message.outputs != null && message.hasOwnProperty("outputs")) {
                var error = $root.ml_pipelines.ExecutorInput.Outputs.verify(message.outputs);
                if (error)
                    return "outputs." + error;
            }
            return null;
        };

        /**
         * Creates an ExecutorInput message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ExecutorInput
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ExecutorInput} ExecutorInput
         */
        ExecutorInput.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ExecutorInput)
                return object;
            var message = new $root.ml_pipelines.ExecutorInput();
            if (object.inputs != null) {
                if (typeof object.inputs !== "object")
                    throw TypeError(".ml_pipelines.ExecutorInput.inputs: object expected");
                message.inputs = $root.ml_pipelines.ExecutorInput.Inputs.fromObject(object.inputs);
            }
            if (object.outputs != null) {
                if (typeof object.outputs !== "object")
                    throw TypeError(".ml_pipelines.ExecutorInput.outputs: object expected");
                message.outputs = $root.ml_pipelines.ExecutorInput.Outputs.fromObject(object.outputs);
            }
            return message;
        };

        /**
         * Creates a plain object from an ExecutorInput message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ExecutorInput
         * @static
         * @param {ml_pipelines.ExecutorInput} message ExecutorInput
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ExecutorInput.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.defaults) {
                object.inputs = null;
                object.outputs = null;
            }
            if (message.inputs != null && message.hasOwnProperty("inputs"))
                object.inputs = $root.ml_pipelines.ExecutorInput.Inputs.toObject(message.inputs, options);
            if (message.outputs != null && message.hasOwnProperty("outputs"))
                object.outputs = $root.ml_pipelines.ExecutorInput.Outputs.toObject(message.outputs, options);
            return object;
        };

        /**
         * Converts this ExecutorInput to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ExecutorInput
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ExecutorInput.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        ExecutorInput.Inputs = (function() {

            /**
             * Properties of an Inputs.
             * @memberof ml_pipelines.ExecutorInput
             * @interface IInputs
             * @property {Object.<string,ml_pipelines.IValue>|null} [parameters] Inputs parameters
             * @property {Object.<string,ml_pipelines.IArtifactList>|null} [artifacts] Inputs artifacts
             */

            /**
             * Constructs a new Inputs.
             * @memberof ml_pipelines.ExecutorInput
             * @classdesc Represents an Inputs.
             * @implements IInputs
             * @constructor
             * @param {ml_pipelines.ExecutorInput.IInputs=} [properties] Properties to set
             */
            function Inputs(properties) {
                this.parameters = {};
                this.artifacts = {};
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Inputs parameters.
             * @member {Object.<string,ml_pipelines.IValue>} parameters
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @instance
             */
            Inputs.prototype.parameters = $util.emptyObject;

            /**
             * Inputs artifacts.
             * @member {Object.<string,ml_pipelines.IArtifactList>} artifacts
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @instance
             */
            Inputs.prototype.artifacts = $util.emptyObject;

            /**
             * Creates a new Inputs instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @static
             * @param {ml_pipelines.ExecutorInput.IInputs=} [properties] Properties to set
             * @returns {ml_pipelines.ExecutorInput.Inputs} Inputs instance
             */
            Inputs.create = function create(properties) {
                return new Inputs(properties);
            };

            /**
             * Encodes the specified Inputs message. Does not implicitly {@link ml_pipelines.ExecutorInput.Inputs.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @static
             * @param {ml_pipelines.ExecutorInput.IInputs} message Inputs message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Inputs.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.parameters != null && Object.hasOwnProperty.call(message, "parameters"))
                    for (var keys = Object.keys(message.parameters), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.Value.encode(message.parameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                if (message.artifacts != null && Object.hasOwnProperty.call(message, "artifacts"))
                    for (var keys = Object.keys(message.artifacts), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.ArtifactList.encode(message.artifacts[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                return writer;
            };

            /**
             * Encodes the specified Inputs message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorInput.Inputs.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @static
             * @param {ml_pipelines.ExecutorInput.IInputs} message Inputs message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Inputs.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an Inputs message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.ExecutorInput.Inputs} Inputs
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Inputs.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ExecutorInput.Inputs(), key, value;
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        if (message.parameters === $util.emptyObject)
                            message.parameters = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.Value.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.parameters[key] = value;
                        break;
                    case 2:
                        if (message.artifacts === $util.emptyObject)
                            message.artifacts = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.ArtifactList.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.artifacts[key] = value;
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an Inputs message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.ExecutorInput.Inputs} Inputs
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Inputs.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an Inputs message.
             * @function verify
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Inputs.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.parameters != null && message.hasOwnProperty("parameters")) {
                    if (!$util.isObject(message.parameters))
                        return "parameters: object expected";
                    var key = Object.keys(message.parameters);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.Value.verify(message.parameters[key[i]]);
                        if (error)
                            return "parameters." + error;
                    }
                }
                if (message.artifacts != null && message.hasOwnProperty("artifacts")) {
                    if (!$util.isObject(message.artifacts))
                        return "artifacts: object expected";
                    var key = Object.keys(message.artifacts);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.ArtifactList.verify(message.artifacts[key[i]]);
                        if (error)
                            return "artifacts." + error;
                    }
                }
                return null;
            };

            /**
             * Creates an Inputs message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.ExecutorInput.Inputs} Inputs
             */
            Inputs.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.ExecutorInput.Inputs)
                    return object;
                var message = new $root.ml_pipelines.ExecutorInput.Inputs();
                if (object.parameters) {
                    if (typeof object.parameters !== "object")
                        throw TypeError(".ml_pipelines.ExecutorInput.Inputs.parameters: object expected");
                    message.parameters = {};
                    for (var keys = Object.keys(object.parameters), i = 0; i < keys.length; ++i) {
                        if (typeof object.parameters[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.ExecutorInput.Inputs.parameters: object expected");
                        message.parameters[keys[i]] = $root.ml_pipelines.Value.fromObject(object.parameters[keys[i]]);
                    }
                }
                if (object.artifacts) {
                    if (typeof object.artifacts !== "object")
                        throw TypeError(".ml_pipelines.ExecutorInput.Inputs.artifacts: object expected");
                    message.artifacts = {};
                    for (var keys = Object.keys(object.artifacts), i = 0; i < keys.length; ++i) {
                        if (typeof object.artifacts[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.ExecutorInput.Inputs.artifacts: object expected");
                        message.artifacts[keys[i]] = $root.ml_pipelines.ArtifactList.fromObject(object.artifacts[keys[i]]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from an Inputs message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @static
             * @param {ml_pipelines.ExecutorInput.Inputs} message Inputs
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Inputs.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.objects || options.defaults) {
                    object.parameters = {};
                    object.artifacts = {};
                }
                var keys2;
                if (message.parameters && (keys2 = Object.keys(message.parameters)).length) {
                    object.parameters = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.parameters[keys2[j]] = $root.ml_pipelines.Value.toObject(message.parameters[keys2[j]], options);
                }
                if (message.artifacts && (keys2 = Object.keys(message.artifacts)).length) {
                    object.artifacts = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.artifacts[keys2[j]] = $root.ml_pipelines.ArtifactList.toObject(message.artifacts[keys2[j]], options);
                }
                return object;
            };

            /**
             * Converts this Inputs to JSON.
             * @function toJSON
             * @memberof ml_pipelines.ExecutorInput.Inputs
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Inputs.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return Inputs;
        })();

        ExecutorInput.OutputParameter = (function() {

            /**
             * Properties of an OutputParameter.
             * @memberof ml_pipelines.ExecutorInput
             * @interface IOutputParameter
             * @property {string|null} [outputFile] OutputParameter outputFile
             */

            /**
             * Constructs a new OutputParameter.
             * @memberof ml_pipelines.ExecutorInput
             * @classdesc Represents an OutputParameter.
             * @implements IOutputParameter
             * @constructor
             * @param {ml_pipelines.ExecutorInput.IOutputParameter=} [properties] Properties to set
             */
            function OutputParameter(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * OutputParameter outputFile.
             * @member {string} outputFile
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @instance
             */
            OutputParameter.prototype.outputFile = "";

            /**
             * Creates a new OutputParameter instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @static
             * @param {ml_pipelines.ExecutorInput.IOutputParameter=} [properties] Properties to set
             * @returns {ml_pipelines.ExecutorInput.OutputParameter} OutputParameter instance
             */
            OutputParameter.create = function create(properties) {
                return new OutputParameter(properties);
            };

            /**
             * Encodes the specified OutputParameter message. Does not implicitly {@link ml_pipelines.ExecutorInput.OutputParameter.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @static
             * @param {ml_pipelines.ExecutorInput.IOutputParameter} message OutputParameter message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OutputParameter.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.outputFile != null && Object.hasOwnProperty.call(message, "outputFile"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.outputFile);
                return writer;
            };

            /**
             * Encodes the specified OutputParameter message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorInput.OutputParameter.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @static
             * @param {ml_pipelines.ExecutorInput.IOutputParameter} message OutputParameter message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OutputParameter.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an OutputParameter message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.ExecutorInput.OutputParameter} OutputParameter
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OutputParameter.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ExecutorInput.OutputParameter();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.outputFile = reader.string();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an OutputParameter message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.ExecutorInput.OutputParameter} OutputParameter
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OutputParameter.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an OutputParameter message.
             * @function verify
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            OutputParameter.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.outputFile != null && message.hasOwnProperty("outputFile"))
                    if (!$util.isString(message.outputFile))
                        return "outputFile: string expected";
                return null;
            };

            /**
             * Creates an OutputParameter message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.ExecutorInput.OutputParameter} OutputParameter
             */
            OutputParameter.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.ExecutorInput.OutputParameter)
                    return object;
                var message = new $root.ml_pipelines.ExecutorInput.OutputParameter();
                if (object.outputFile != null)
                    message.outputFile = String(object.outputFile);
                return message;
            };

            /**
             * Creates a plain object from an OutputParameter message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @static
             * @param {ml_pipelines.ExecutorInput.OutputParameter} message OutputParameter
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            OutputParameter.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.outputFile = "";
                if (message.outputFile != null && message.hasOwnProperty("outputFile"))
                    object.outputFile = message.outputFile;
                return object;
            };

            /**
             * Converts this OutputParameter to JSON.
             * @function toJSON
             * @memberof ml_pipelines.ExecutorInput.OutputParameter
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            OutputParameter.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return OutputParameter;
        })();

        ExecutorInput.Outputs = (function() {

            /**
             * Properties of an Outputs.
             * @memberof ml_pipelines.ExecutorInput
             * @interface IOutputs
             * @property {Object.<string,ml_pipelines.ExecutorInput.IOutputParameter>|null} [parameters] Outputs parameters
             * @property {Object.<string,ml_pipelines.IArtifactList>|null} [artifacts] Outputs artifacts
             * @property {string|null} [outputFile] Outputs outputFile
             */

            /**
             * Constructs a new Outputs.
             * @memberof ml_pipelines.ExecutorInput
             * @classdesc Represents an Outputs.
             * @implements IOutputs
             * @constructor
             * @param {ml_pipelines.ExecutorInput.IOutputs=} [properties] Properties to set
             */
            function Outputs(properties) {
                this.parameters = {};
                this.artifacts = {};
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Outputs parameters.
             * @member {Object.<string,ml_pipelines.ExecutorInput.IOutputParameter>} parameters
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @instance
             */
            Outputs.prototype.parameters = $util.emptyObject;

            /**
             * Outputs artifacts.
             * @member {Object.<string,ml_pipelines.IArtifactList>} artifacts
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @instance
             */
            Outputs.prototype.artifacts = $util.emptyObject;

            /**
             * Outputs outputFile.
             * @member {string} outputFile
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @instance
             */
            Outputs.prototype.outputFile = "";

            /**
             * Creates a new Outputs instance using the specified properties.
             * @function create
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @static
             * @param {ml_pipelines.ExecutorInput.IOutputs=} [properties] Properties to set
             * @returns {ml_pipelines.ExecutorInput.Outputs} Outputs instance
             */
            Outputs.create = function create(properties) {
                return new Outputs(properties);
            };

            /**
             * Encodes the specified Outputs message. Does not implicitly {@link ml_pipelines.ExecutorInput.Outputs.verify|verify} messages.
             * @function encode
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @static
             * @param {ml_pipelines.ExecutorInput.IOutputs} message Outputs message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Outputs.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.parameters != null && Object.hasOwnProperty.call(message, "parameters"))
                    for (var keys = Object.keys(message.parameters), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.ExecutorInput.OutputParameter.encode(message.parameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                if (message.artifacts != null && Object.hasOwnProperty.call(message, "artifacts"))
                    for (var keys = Object.keys(message.artifacts), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.ml_pipelines.ArtifactList.encode(message.artifacts[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                if (message.outputFile != null && Object.hasOwnProperty.call(message, "outputFile"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.outputFile);
                return writer;
            };

            /**
             * Encodes the specified Outputs message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorInput.Outputs.verify|verify} messages.
             * @function encodeDelimited
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @static
             * @param {ml_pipelines.ExecutorInput.IOutputs} message Outputs message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Outputs.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an Outputs message from the specified reader or buffer.
             * @function decode
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {ml_pipelines.ExecutorInput.Outputs} Outputs
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Outputs.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ExecutorInput.Outputs(), key, value;
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        if (message.parameters === $util.emptyObject)
                            message.parameters = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.ExecutorInput.OutputParameter.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.parameters[key] = value;
                        break;
                    case 2:
                        if (message.artifacts === $util.emptyObject)
                            message.artifacts = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.ml_pipelines.ArtifactList.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.artifacts[key] = value;
                        break;
                    case 3:
                        message.outputFile = reader.string();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an Outputs message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {ml_pipelines.ExecutorInput.Outputs} Outputs
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Outputs.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an Outputs message.
             * @function verify
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Outputs.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.parameters != null && message.hasOwnProperty("parameters")) {
                    if (!$util.isObject(message.parameters))
                        return "parameters: object expected";
                    var key = Object.keys(message.parameters);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.ExecutorInput.OutputParameter.verify(message.parameters[key[i]]);
                        if (error)
                            return "parameters." + error;
                    }
                }
                if (message.artifacts != null && message.hasOwnProperty("artifacts")) {
                    if (!$util.isObject(message.artifacts))
                        return "artifacts: object expected";
                    var key = Object.keys(message.artifacts);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.ml_pipelines.ArtifactList.verify(message.artifacts[key[i]]);
                        if (error)
                            return "artifacts." + error;
                    }
                }
                if (message.outputFile != null && message.hasOwnProperty("outputFile"))
                    if (!$util.isString(message.outputFile))
                        return "outputFile: string expected";
                return null;
            };

            /**
             * Creates an Outputs message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {ml_pipelines.ExecutorInput.Outputs} Outputs
             */
            Outputs.fromObject = function fromObject(object) {
                if (object instanceof $root.ml_pipelines.ExecutorInput.Outputs)
                    return object;
                var message = new $root.ml_pipelines.ExecutorInput.Outputs();
                if (object.parameters) {
                    if (typeof object.parameters !== "object")
                        throw TypeError(".ml_pipelines.ExecutorInput.Outputs.parameters: object expected");
                    message.parameters = {};
                    for (var keys = Object.keys(object.parameters), i = 0; i < keys.length; ++i) {
                        if (typeof object.parameters[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.ExecutorInput.Outputs.parameters: object expected");
                        message.parameters[keys[i]] = $root.ml_pipelines.ExecutorInput.OutputParameter.fromObject(object.parameters[keys[i]]);
                    }
                }
                if (object.artifacts) {
                    if (typeof object.artifacts !== "object")
                        throw TypeError(".ml_pipelines.ExecutorInput.Outputs.artifacts: object expected");
                    message.artifacts = {};
                    for (var keys = Object.keys(object.artifacts), i = 0; i < keys.length; ++i) {
                        if (typeof object.artifacts[keys[i]] !== "object")
                            throw TypeError(".ml_pipelines.ExecutorInput.Outputs.artifacts: object expected");
                        message.artifacts[keys[i]] = $root.ml_pipelines.ArtifactList.fromObject(object.artifacts[keys[i]]);
                    }
                }
                if (object.outputFile != null)
                    message.outputFile = String(object.outputFile);
                return message;
            };

            /**
             * Creates a plain object from an Outputs message. Also converts values to other types if specified.
             * @function toObject
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @static
             * @param {ml_pipelines.ExecutorInput.Outputs} message Outputs
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Outputs.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.objects || options.defaults) {
                    object.parameters = {};
                    object.artifacts = {};
                }
                if (options.defaults)
                    object.outputFile = "";
                var keys2;
                if (message.parameters && (keys2 = Object.keys(message.parameters)).length) {
                    object.parameters = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.parameters[keys2[j]] = $root.ml_pipelines.ExecutorInput.OutputParameter.toObject(message.parameters[keys2[j]], options);
                }
                if (message.artifacts && (keys2 = Object.keys(message.artifacts)).length) {
                    object.artifacts = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.artifacts[keys2[j]] = $root.ml_pipelines.ArtifactList.toObject(message.artifacts[keys2[j]], options);
                }
                if (message.outputFile != null && message.hasOwnProperty("outputFile"))
                    object.outputFile = message.outputFile;
                return object;
            };

            /**
             * Converts this Outputs to JSON.
             * @function toJSON
             * @memberof ml_pipelines.ExecutorInput.Outputs
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Outputs.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return Outputs;
        })();

        return ExecutorInput;
    })();

    ml_pipelines.ExecutorOutput = (function() {

        /**
         * Properties of an ExecutorOutput.
         * @memberof ml_pipelines
         * @interface IExecutorOutput
         * @property {Object.<string,ml_pipelines.IValue>|null} [parameters] ExecutorOutput parameters
         * @property {Object.<string,ml_pipelines.IArtifactList>|null} [artifacts] ExecutorOutput artifacts
         */

        /**
         * Constructs a new ExecutorOutput.
         * @memberof ml_pipelines
         * @classdesc Represents an ExecutorOutput.
         * @implements IExecutorOutput
         * @constructor
         * @param {ml_pipelines.IExecutorOutput=} [properties] Properties to set
         */
        function ExecutorOutput(properties) {
            this.parameters = {};
            this.artifacts = {};
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * ExecutorOutput parameters.
         * @member {Object.<string,ml_pipelines.IValue>} parameters
         * @memberof ml_pipelines.ExecutorOutput
         * @instance
         */
        ExecutorOutput.prototype.parameters = $util.emptyObject;

        /**
         * ExecutorOutput artifacts.
         * @member {Object.<string,ml_pipelines.IArtifactList>} artifacts
         * @memberof ml_pipelines.ExecutorOutput
         * @instance
         */
        ExecutorOutput.prototype.artifacts = $util.emptyObject;

        /**
         * Creates a new ExecutorOutput instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.ExecutorOutput
         * @static
         * @param {ml_pipelines.IExecutorOutput=} [properties] Properties to set
         * @returns {ml_pipelines.ExecutorOutput} ExecutorOutput instance
         */
        ExecutorOutput.create = function create(properties) {
            return new ExecutorOutput(properties);
        };

        /**
         * Encodes the specified ExecutorOutput message. Does not implicitly {@link ml_pipelines.ExecutorOutput.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.ExecutorOutput
         * @static
         * @param {ml_pipelines.IExecutorOutput} message ExecutorOutput message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ExecutorOutput.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.parameters != null && Object.hasOwnProperty.call(message, "parameters"))
                for (var keys = Object.keys(message.parameters), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.Value.encode(message.parameters[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            if (message.artifacts != null && Object.hasOwnProperty.call(message, "artifacts"))
                for (var keys = Object.keys(message.artifacts), i = 0; i < keys.length; ++i) {
                    writer.uint32(/* id 2, wireType 2 =*/18).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                    $root.ml_pipelines.ArtifactList.encode(message.artifacts[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                }
            return writer;
        };

        /**
         * Encodes the specified ExecutorOutput message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorOutput.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.ExecutorOutput
         * @static
         * @param {ml_pipelines.IExecutorOutput} message ExecutorOutput message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        ExecutorOutput.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes an ExecutorOutput message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.ExecutorOutput
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.ExecutorOutput} ExecutorOutput
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ExecutorOutput.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.ExecutorOutput(), key, value;
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    if (message.parameters === $util.emptyObject)
                        message.parameters = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.Value.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.parameters[key] = value;
                    break;
                case 2:
                    if (message.artifacts === $util.emptyObject)
                        message.artifacts = {};
                    var end2 = reader.uint32() + reader.pos;
                    key = "";
                    value = null;
                    while (reader.pos < end2) {
                        var tag2 = reader.uint32();
                        switch (tag2 >>> 3) {
                        case 1:
                            key = reader.string();
                            break;
                        case 2:
                            value = $root.ml_pipelines.ArtifactList.decode(reader, reader.uint32());
                            break;
                        default:
                            reader.skipType(tag2 & 7);
                            break;
                        }
                    }
                    message.artifacts[key] = value;
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes an ExecutorOutput message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.ExecutorOutput
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.ExecutorOutput} ExecutorOutput
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        ExecutorOutput.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies an ExecutorOutput message.
         * @function verify
         * @memberof ml_pipelines.ExecutorOutput
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        ExecutorOutput.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.parameters != null && message.hasOwnProperty("parameters")) {
                if (!$util.isObject(message.parameters))
                    return "parameters: object expected";
                var key = Object.keys(message.parameters);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.Value.verify(message.parameters[key[i]]);
                    if (error)
                        return "parameters." + error;
                }
            }
            if (message.artifacts != null && message.hasOwnProperty("artifacts")) {
                if (!$util.isObject(message.artifacts))
                    return "artifacts: object expected";
                var key = Object.keys(message.artifacts);
                for (var i = 0; i < key.length; ++i) {
                    var error = $root.ml_pipelines.ArtifactList.verify(message.artifacts[key[i]]);
                    if (error)
                        return "artifacts." + error;
                }
            }
            return null;
        };

        /**
         * Creates an ExecutorOutput message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.ExecutorOutput
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.ExecutorOutput} ExecutorOutput
         */
        ExecutorOutput.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.ExecutorOutput)
                return object;
            var message = new $root.ml_pipelines.ExecutorOutput();
            if (object.parameters) {
                if (typeof object.parameters !== "object")
                    throw TypeError(".ml_pipelines.ExecutorOutput.parameters: object expected");
                message.parameters = {};
                for (var keys = Object.keys(object.parameters), i = 0; i < keys.length; ++i) {
                    if (typeof object.parameters[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.ExecutorOutput.parameters: object expected");
                    message.parameters[keys[i]] = $root.ml_pipelines.Value.fromObject(object.parameters[keys[i]]);
                }
            }
            if (object.artifacts) {
                if (typeof object.artifacts !== "object")
                    throw TypeError(".ml_pipelines.ExecutorOutput.artifacts: object expected");
                message.artifacts = {};
                for (var keys = Object.keys(object.artifacts), i = 0; i < keys.length; ++i) {
                    if (typeof object.artifacts[keys[i]] !== "object")
                        throw TypeError(".ml_pipelines.ExecutorOutput.artifacts: object expected");
                    message.artifacts[keys[i]] = $root.ml_pipelines.ArtifactList.fromObject(object.artifacts[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from an ExecutorOutput message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.ExecutorOutput
         * @static
         * @param {ml_pipelines.ExecutorOutput} message ExecutorOutput
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        ExecutorOutput.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.objects || options.defaults) {
                object.parameters = {};
                object.artifacts = {};
            }
            var keys2;
            if (message.parameters && (keys2 = Object.keys(message.parameters)).length) {
                object.parameters = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.parameters[keys2[j]] = $root.ml_pipelines.Value.toObject(message.parameters[keys2[j]], options);
            }
            if (message.artifacts && (keys2 = Object.keys(message.artifacts)).length) {
                object.artifacts = {};
                for (var j = 0; j < keys2.length; ++j)
                    object.artifacts[keys2[j]] = $root.ml_pipelines.ArtifactList.toObject(message.artifacts[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this ExecutorOutput to JSON.
         * @function toJSON
         * @memberof ml_pipelines.ExecutorOutput
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        ExecutorOutput.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return ExecutorOutput;
    })();

    ml_pipelines.PipelineTaskFinalStatus = (function() {

        /**
         * Properties of a PipelineTaskFinalStatus.
         * @memberof ml_pipelines
         * @interface IPipelineTaskFinalStatus
         * @property {string|null} [state] PipelineTaskFinalStatus state
         * @property {google.rpc.IStatus|null} [error] PipelineTaskFinalStatus error
         */

        /**
         * Constructs a new PipelineTaskFinalStatus.
         * @memberof ml_pipelines
         * @classdesc Represents a PipelineTaskFinalStatus.
         * @implements IPipelineTaskFinalStatus
         * @constructor
         * @param {ml_pipelines.IPipelineTaskFinalStatus=} [properties] Properties to set
         */
        function PipelineTaskFinalStatus(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * PipelineTaskFinalStatus state.
         * @member {string} state
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @instance
         */
        PipelineTaskFinalStatus.prototype.state = "";

        /**
         * PipelineTaskFinalStatus error.
         * @member {google.rpc.IStatus|null|undefined} error
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @instance
         */
        PipelineTaskFinalStatus.prototype.error = null;

        /**
         * Creates a new PipelineTaskFinalStatus instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @static
         * @param {ml_pipelines.IPipelineTaskFinalStatus=} [properties] Properties to set
         * @returns {ml_pipelines.PipelineTaskFinalStatus} PipelineTaskFinalStatus instance
         */
        PipelineTaskFinalStatus.create = function create(properties) {
            return new PipelineTaskFinalStatus(properties);
        };

        /**
         * Encodes the specified PipelineTaskFinalStatus message. Does not implicitly {@link ml_pipelines.PipelineTaskFinalStatus.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @static
         * @param {ml_pipelines.IPipelineTaskFinalStatus} message PipelineTaskFinalStatus message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineTaskFinalStatus.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            if (message.state != null && Object.hasOwnProperty.call(message, "state"))
                writer.uint32(/* id 1, wireType 2 =*/10).string(message.state);
            if (message.error != null && Object.hasOwnProperty.call(message, "error"))
                $root.google.rpc.Status.encode(message.error, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
            return writer;
        };

        /**
         * Encodes the specified PipelineTaskFinalStatus message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskFinalStatus.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @static
         * @param {ml_pipelines.IPipelineTaskFinalStatus} message PipelineTaskFinalStatus message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineTaskFinalStatus.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a PipelineTaskFinalStatus message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.PipelineTaskFinalStatus} PipelineTaskFinalStatus
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineTaskFinalStatus.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineTaskFinalStatus();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                case 1:
                    message.state = reader.string();
                    break;
                case 2:
                    message.error = $root.google.rpc.Status.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a PipelineTaskFinalStatus message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.PipelineTaskFinalStatus} PipelineTaskFinalStatus
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineTaskFinalStatus.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a PipelineTaskFinalStatus message.
         * @function verify
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        PipelineTaskFinalStatus.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            if (message.state != null && message.hasOwnProperty("state"))
                if (!$util.isString(message.state))
                    return "state: string expected";
            if (message.error != null && message.hasOwnProperty("error")) {
                var error = $root.google.rpc.Status.verify(message.error);
                if (error)
                    return "error." + error;
            }
            return null;
        };

        /**
         * Creates a PipelineTaskFinalStatus message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.PipelineTaskFinalStatus} PipelineTaskFinalStatus
         */
        PipelineTaskFinalStatus.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.PipelineTaskFinalStatus)
                return object;
            var message = new $root.ml_pipelines.PipelineTaskFinalStatus();
            if (object.state != null)
                message.state = String(object.state);
            if (object.error != null) {
                if (typeof object.error !== "object")
                    throw TypeError(".ml_pipelines.PipelineTaskFinalStatus.error: object expected");
                message.error = $root.google.rpc.Status.fromObject(object.error);
            }
            return message;
        };

        /**
         * Creates a plain object from a PipelineTaskFinalStatus message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @static
         * @param {ml_pipelines.PipelineTaskFinalStatus} message PipelineTaskFinalStatus
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        PipelineTaskFinalStatus.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            var object = {};
            if (options.defaults) {
                object.state = "";
                object.error = null;
            }
            if (message.state != null && message.hasOwnProperty("state"))
                object.state = message.state;
            if (message.error != null && message.hasOwnProperty("error"))
                object.error = $root.google.rpc.Status.toObject(message.error, options);
            return object;
        };

        /**
         * Converts this PipelineTaskFinalStatus to JSON.
         * @function toJSON
         * @memberof ml_pipelines.PipelineTaskFinalStatus
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        PipelineTaskFinalStatus.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        return PipelineTaskFinalStatus;
    })();

    ml_pipelines.PipelineStateEnum = (function() {

        /**
         * Properties of a PipelineStateEnum.
         * @memberof ml_pipelines
         * @interface IPipelineStateEnum
         */

        /**
         * Constructs a new PipelineStateEnum.
         * @memberof ml_pipelines
         * @classdesc Represents a PipelineStateEnum.
         * @implements IPipelineStateEnum
         * @constructor
         * @param {ml_pipelines.IPipelineStateEnum=} [properties] Properties to set
         */
        function PipelineStateEnum(properties) {
            if (properties)
                for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * Creates a new PipelineStateEnum instance using the specified properties.
         * @function create
         * @memberof ml_pipelines.PipelineStateEnum
         * @static
         * @param {ml_pipelines.IPipelineStateEnum=} [properties] Properties to set
         * @returns {ml_pipelines.PipelineStateEnum} PipelineStateEnum instance
         */
        PipelineStateEnum.create = function create(properties) {
            return new PipelineStateEnum(properties);
        };

        /**
         * Encodes the specified PipelineStateEnum message. Does not implicitly {@link ml_pipelines.PipelineStateEnum.verify|verify} messages.
         * @function encode
         * @memberof ml_pipelines.PipelineStateEnum
         * @static
         * @param {ml_pipelines.IPipelineStateEnum} message PipelineStateEnum message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineStateEnum.encode = function encode(message, writer) {
            if (!writer)
                writer = $Writer.create();
            return writer;
        };

        /**
         * Encodes the specified PipelineStateEnum message, length delimited. Does not implicitly {@link ml_pipelines.PipelineStateEnum.verify|verify} messages.
         * @function encodeDelimited
         * @memberof ml_pipelines.PipelineStateEnum
         * @static
         * @param {ml_pipelines.IPipelineStateEnum} message PipelineStateEnum message or plain object to encode
         * @param {$protobuf.Writer} [writer] Writer to encode to
         * @returns {$protobuf.Writer} Writer
         */
        PipelineStateEnum.encodeDelimited = function encodeDelimited(message, writer) {
            return this.encode(message, writer).ldelim();
        };

        /**
         * Decodes a PipelineStateEnum message from the specified reader or buffer.
         * @function decode
         * @memberof ml_pipelines.PipelineStateEnum
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {ml_pipelines.PipelineStateEnum} PipelineStateEnum
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineStateEnum.decode = function decode(reader, length) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            var end = length === undefined ? reader.len : reader.pos + length, message = new $root.ml_pipelines.PipelineStateEnum();
            while (reader.pos < end) {
                var tag = reader.uint32();
                switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Decodes a PipelineStateEnum message from the specified reader or buffer, length delimited.
         * @function decodeDelimited
         * @memberof ml_pipelines.PipelineStateEnum
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @returns {ml_pipelines.PipelineStateEnum} PipelineStateEnum
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        PipelineStateEnum.decodeDelimited = function decodeDelimited(reader) {
            if (!(reader instanceof $Reader))
                reader = new $Reader(reader);
            return this.decode(reader, reader.uint32());
        };

        /**
         * Verifies a PipelineStateEnum message.
         * @function verify
         * @memberof ml_pipelines.PipelineStateEnum
         * @static
         * @param {Object.<string,*>} message Plain object to verify
         * @returns {string|null} `null` if valid, otherwise the reason why it is not
         */
        PipelineStateEnum.verify = function verify(message) {
            if (typeof message !== "object" || message === null)
                return "object expected";
            return null;
        };

        /**
         * Creates a PipelineStateEnum message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof ml_pipelines.PipelineStateEnum
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {ml_pipelines.PipelineStateEnum} PipelineStateEnum
         */
        PipelineStateEnum.fromObject = function fromObject(object) {
            if (object instanceof $root.ml_pipelines.PipelineStateEnum)
                return object;
            return new $root.ml_pipelines.PipelineStateEnum();
        };

        /**
         * Creates a plain object from a PipelineStateEnum message. Also converts values to other types if specified.
         * @function toObject
         * @memberof ml_pipelines.PipelineStateEnum
         * @static
         * @param {ml_pipelines.PipelineStateEnum} message PipelineStateEnum
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        PipelineStateEnum.toObject = function toObject() {
            return {};
        };

        /**
         * Converts this PipelineStateEnum to JSON.
         * @function toJSON
         * @memberof ml_pipelines.PipelineStateEnum
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        PipelineStateEnum.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * PipelineTaskState enum.
         * @name ml_pipelines.PipelineStateEnum.PipelineTaskState
         * @enum {number}
         * @property {number} TASK_STATE_UNSPECIFIED=0 TASK_STATE_UNSPECIFIED value
         * @property {number} PENDING=1 PENDING value
         * @property {number} RUNNING_DRIVER=2 RUNNING_DRIVER value
         * @property {number} DRIVER_SUCCEEDED=3 DRIVER_SUCCEEDED value
         * @property {number} RUNNING_EXECUTOR=4 RUNNING_EXECUTOR value
         * @property {number} SUCCEEDED=5 SUCCEEDED value
         * @property {number} CANCEL_PENDING=6 CANCEL_PENDING value
         * @property {number} CANCELLING=7 CANCELLING value
         * @property {number} CANCELLED=8 CANCELLED value
         * @property {number} FAILED=9 FAILED value
         * @property {number} SKIPPED=10 SKIPPED value
         * @property {number} QUEUED=11 QUEUED value
         * @property {number} NOT_TRIGGERED=12 NOT_TRIGGERED value
         * @property {number} UNSCHEDULABLE=13 UNSCHEDULABLE value
         */
        PipelineStateEnum.PipelineTaskState = (function() {
            var valuesById = {}, values = Object.create(valuesById);
            values[valuesById[0] = "TASK_STATE_UNSPECIFIED"] = 0;
            values[valuesById[1] = "PENDING"] = 1;
            values[valuesById[2] = "RUNNING_DRIVER"] = 2;
            values[valuesById[3] = "DRIVER_SUCCEEDED"] = 3;
            values[valuesById[4] = "RUNNING_EXECUTOR"] = 4;
            values[valuesById[5] = "SUCCEEDED"] = 5;
            values[valuesById[6] = "CANCEL_PENDING"] = 6;
            values[valuesById[7] = "CANCELLING"] = 7;
            values[valuesById[8] = "CANCELLED"] = 8;
            values[valuesById[9] = "FAILED"] = 9;
            values[valuesById[10] = "SKIPPED"] = 10;
            values[valuesById[11] = "QUEUED"] = 11;
            values[valuesById[12] = "NOT_TRIGGERED"] = 12;
            values[valuesById[13] = "UNSCHEDULABLE"] = 13;
            return values;
        })();

        return PipelineStateEnum;
    })();

    return ml_pipelines;
})();

$root.google = (function() {

    /**
     * Namespace google.
     * @exports google
     * @namespace
     */
    var google = {};

    google.protobuf = (function() {

        /**
         * Namespace protobuf.
         * @memberof google
         * @namespace
         */
        var protobuf = {};

        protobuf.Any = (function() {

            /**
             * Properties of an Any.
             * @memberof google.protobuf
             * @interface IAny
             * @property {string|null} [type_url] Any type_url
             * @property {Uint8Array|null} [value] Any value
             */

            /**
             * Constructs a new Any.
             * @memberof google.protobuf
             * @classdesc Represents an Any.
             * @implements IAny
             * @constructor
             * @param {google.protobuf.IAny=} [properties] Properties to set
             */
            function Any(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Any type_url.
             * @member {string} type_url
             * @memberof google.protobuf.Any
             * @instance
             */
            Any.prototype.type_url = "";

            /**
             * Any value.
             * @member {Uint8Array} value
             * @memberof google.protobuf.Any
             * @instance
             */
            Any.prototype.value = $util.newBuffer([]);

            /**
             * Creates a new Any instance using the specified properties.
             * @function create
             * @memberof google.protobuf.Any
             * @static
             * @param {google.protobuf.IAny=} [properties] Properties to set
             * @returns {google.protobuf.Any} Any instance
             */
            Any.create = function create(properties) {
                return new Any(properties);
            };

            /**
             * Encodes the specified Any message. Does not implicitly {@link google.protobuf.Any.verify|verify} messages.
             * @function encode
             * @memberof google.protobuf.Any
             * @static
             * @param {google.protobuf.IAny} message Any message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Any.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.type_url != null && Object.hasOwnProperty.call(message, "type_url"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.type_url);
                if (message.value != null && Object.hasOwnProperty.call(message, "value"))
                    writer.uint32(/* id 2, wireType 2 =*/18).bytes(message.value);
                return writer;
            };

            /**
             * Encodes the specified Any message, length delimited. Does not implicitly {@link google.protobuf.Any.verify|verify} messages.
             * @function encodeDelimited
             * @memberof google.protobuf.Any
             * @static
             * @param {google.protobuf.IAny} message Any message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Any.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an Any message from the specified reader or buffer.
             * @function decode
             * @memberof google.protobuf.Any
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {google.protobuf.Any} Any
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Any.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.google.protobuf.Any();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.type_url = reader.string();
                        break;
                    case 2:
                        message.value = reader.bytes();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an Any message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof google.protobuf.Any
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {google.protobuf.Any} Any
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Any.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an Any message.
             * @function verify
             * @memberof google.protobuf.Any
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Any.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.type_url != null && message.hasOwnProperty("type_url"))
                    if (!$util.isString(message.type_url))
                        return "type_url: string expected";
                if (message.value != null && message.hasOwnProperty("value"))
                    if (!(message.value && typeof message.value.length === "number" || $util.isString(message.value)))
                        return "value: buffer expected";
                return null;
            };

            /**
             * Creates an Any message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof google.protobuf.Any
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {google.protobuf.Any} Any
             */
            Any.fromObject = function fromObject(object) {
                if (object instanceof $root.google.protobuf.Any)
                    return object;
                var message = new $root.google.protobuf.Any();
                if (object.type_url != null)
                    message.type_url = String(object.type_url);
                if (object.value != null)
                    if (typeof object.value === "string")
                        $util.base64.decode(object.value, message.value = $util.newBuffer($util.base64.length(object.value)), 0);
                    else if (object.value.length)
                        message.value = object.value;
                return message;
            };

            /**
             * Creates a plain object from an Any message. Also converts values to other types if specified.
             * @function toObject
             * @memberof google.protobuf.Any
             * @static
             * @param {google.protobuf.Any} message Any
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Any.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    object.type_url = "";
                    if (options.bytes === String)
                        object.value = "";
                    else {
                        object.value = [];
                        if (options.bytes !== Array)
                            object.value = $util.newBuffer(object.value);
                    }
                }
                if (message.type_url != null && message.hasOwnProperty("type_url"))
                    object.type_url = message.type_url;
                if (message.value != null && message.hasOwnProperty("value"))
                    object.value = options.bytes === String ? $util.base64.encode(message.value, 0, message.value.length) : options.bytes === Array ? Array.prototype.slice.call(message.value) : message.value;
                return object;
            };

            /**
             * Converts this Any to JSON.
             * @function toJSON
             * @memberof google.protobuf.Any
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Any.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return Any;
        })();

        protobuf.Struct = (function() {

            /**
             * Properties of a Struct.
             * @memberof google.protobuf
             * @interface IStruct
             * @property {Object.<string,google.protobuf.IValue>|null} [fields] Struct fields
             */

            /**
             * Constructs a new Struct.
             * @memberof google.protobuf
             * @classdesc Represents a Struct.
             * @implements IStruct
             * @constructor
             * @param {google.protobuf.IStruct=} [properties] Properties to set
             */
            function Struct(properties) {
                this.fields = {};
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Struct fields.
             * @member {Object.<string,google.protobuf.IValue>} fields
             * @memberof google.protobuf.Struct
             * @instance
             */
            Struct.prototype.fields = $util.emptyObject;

            /**
             * Creates a new Struct instance using the specified properties.
             * @function create
             * @memberof google.protobuf.Struct
             * @static
             * @param {google.protobuf.IStruct=} [properties] Properties to set
             * @returns {google.protobuf.Struct} Struct instance
             */
            Struct.create = function create(properties) {
                return new Struct(properties);
            };

            /**
             * Encodes the specified Struct message. Does not implicitly {@link google.protobuf.Struct.verify|verify} messages.
             * @function encode
             * @memberof google.protobuf.Struct
             * @static
             * @param {google.protobuf.IStruct} message Struct message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Struct.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.fields != null && Object.hasOwnProperty.call(message, "fields"))
                    for (var keys = Object.keys(message.fields), i = 0; i < keys.length; ++i) {
                        writer.uint32(/* id 1, wireType 2 =*/10).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]);
                        $root.google.protobuf.Value.encode(message.fields[keys[i]], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim().ldelim();
                    }
                return writer;
            };

            /**
             * Encodes the specified Struct message, length delimited. Does not implicitly {@link google.protobuf.Struct.verify|verify} messages.
             * @function encodeDelimited
             * @memberof google.protobuf.Struct
             * @static
             * @param {google.protobuf.IStruct} message Struct message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Struct.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a Struct message from the specified reader or buffer.
             * @function decode
             * @memberof google.protobuf.Struct
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {google.protobuf.Struct} Struct
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Struct.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.google.protobuf.Struct(), key, value;
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        if (message.fields === $util.emptyObject)
                            message.fields = {};
                        var end2 = reader.uint32() + reader.pos;
                        key = "";
                        value = null;
                        while (reader.pos < end2) {
                            var tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.string();
                                break;
                            case 2:
                                value = $root.google.protobuf.Value.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.fields[key] = value;
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a Struct message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof google.protobuf.Struct
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {google.protobuf.Struct} Struct
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Struct.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a Struct message.
             * @function verify
             * @memberof google.protobuf.Struct
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Struct.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.fields != null && message.hasOwnProperty("fields")) {
                    if (!$util.isObject(message.fields))
                        return "fields: object expected";
                    var key = Object.keys(message.fields);
                    for (var i = 0; i < key.length; ++i) {
                        var error = $root.google.protobuf.Value.verify(message.fields[key[i]]);
                        if (error)
                            return "fields." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a Struct message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof google.protobuf.Struct
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {google.protobuf.Struct} Struct
             */
            Struct.fromObject = function fromObject(object) {
                if (object instanceof $root.google.protobuf.Struct)
                    return object;
                var message = new $root.google.protobuf.Struct();
                if (object.fields) {
                    if (typeof object.fields !== "object")
                        throw TypeError(".google.protobuf.Struct.fields: object expected");
                    message.fields = {};
                    for (var keys = Object.keys(object.fields), i = 0; i < keys.length; ++i) {
                        if (typeof object.fields[keys[i]] !== "object")
                            throw TypeError(".google.protobuf.Struct.fields: object expected");
                        message.fields[keys[i]] = $root.google.protobuf.Value.fromObject(object.fields[keys[i]]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a Struct message. Also converts values to other types if specified.
             * @function toObject
             * @memberof google.protobuf.Struct
             * @static
             * @param {google.protobuf.Struct} message Struct
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Struct.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.objects || options.defaults)
                    object.fields = {};
                var keys2;
                if (message.fields && (keys2 = Object.keys(message.fields)).length) {
                    object.fields = {};
                    for (var j = 0; j < keys2.length; ++j)
                        object.fields[keys2[j]] = $root.google.protobuf.Value.toObject(message.fields[keys2[j]], options);
                }
                return object;
            };

            /**
             * Converts this Struct to JSON.
             * @function toJSON
             * @memberof google.protobuf.Struct
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Struct.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return Struct;
        })();

        protobuf.Value = (function() {

            /**
             * Properties of a Value.
             * @memberof google.protobuf
             * @interface IValue
             * @property {google.protobuf.NullValue|null} [nullValue] Value nullValue
             * @property {number|null} [numberValue] Value numberValue
             * @property {string|null} [stringValue] Value stringValue
             * @property {boolean|null} [boolValue] Value boolValue
             * @property {google.protobuf.IStruct|null} [structValue] Value structValue
             * @property {google.protobuf.IListValue|null} [listValue] Value listValue
             */

            /**
             * Constructs a new Value.
             * @memberof google.protobuf
             * @classdesc Represents a Value.
             * @implements IValue
             * @constructor
             * @param {google.protobuf.IValue=} [properties] Properties to set
             */
            function Value(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Value nullValue.
             * @member {google.protobuf.NullValue|null|undefined} nullValue
             * @memberof google.protobuf.Value
             * @instance
             */
            Value.prototype.nullValue = null;

            /**
             * Value numberValue.
             * @member {number|null|undefined} numberValue
             * @memberof google.protobuf.Value
             * @instance
             */
            Value.prototype.numberValue = null;

            /**
             * Value stringValue.
             * @member {string|null|undefined} stringValue
             * @memberof google.protobuf.Value
             * @instance
             */
            Value.prototype.stringValue = null;

            /**
             * Value boolValue.
             * @member {boolean|null|undefined} boolValue
             * @memberof google.protobuf.Value
             * @instance
             */
            Value.prototype.boolValue = null;

            /**
             * Value structValue.
             * @member {google.protobuf.IStruct|null|undefined} structValue
             * @memberof google.protobuf.Value
             * @instance
             */
            Value.prototype.structValue = null;

            /**
             * Value listValue.
             * @member {google.protobuf.IListValue|null|undefined} listValue
             * @memberof google.protobuf.Value
             * @instance
             */
            Value.prototype.listValue = null;

            // OneOf field names bound to virtual getters and setters
            var $oneOfFields;

            /**
             * Value kind.
             * @member {"nullValue"|"numberValue"|"stringValue"|"boolValue"|"structValue"|"listValue"|undefined} kind
             * @memberof google.protobuf.Value
             * @instance
             */
            Object.defineProperty(Value.prototype, "kind", {
                get: $util.oneOfGetter($oneOfFields = ["nullValue", "numberValue", "stringValue", "boolValue", "structValue", "listValue"]),
                set: $util.oneOfSetter($oneOfFields)
            });

            /**
             * Creates a new Value instance using the specified properties.
             * @function create
             * @memberof google.protobuf.Value
             * @static
             * @param {google.protobuf.IValue=} [properties] Properties to set
             * @returns {google.protobuf.Value} Value instance
             */
            Value.create = function create(properties) {
                return new Value(properties);
            };

            /**
             * Encodes the specified Value message. Does not implicitly {@link google.protobuf.Value.verify|verify} messages.
             * @function encode
             * @memberof google.protobuf.Value
             * @static
             * @param {google.protobuf.IValue} message Value message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Value.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.nullValue != null && Object.hasOwnProperty.call(message, "nullValue"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int32(message.nullValue);
                if (message.numberValue != null && Object.hasOwnProperty.call(message, "numberValue"))
                    writer.uint32(/* id 2, wireType 1 =*/17).double(message.numberValue);
                if (message.stringValue != null && Object.hasOwnProperty.call(message, "stringValue"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.stringValue);
                if (message.boolValue != null && Object.hasOwnProperty.call(message, "boolValue"))
                    writer.uint32(/* id 4, wireType 0 =*/32).bool(message.boolValue);
                if (message.structValue != null && Object.hasOwnProperty.call(message, "structValue"))
                    $root.google.protobuf.Struct.encode(message.structValue, writer.uint32(/* id 5, wireType 2 =*/42).fork()).ldelim();
                if (message.listValue != null && Object.hasOwnProperty.call(message, "listValue"))
                    $root.google.protobuf.ListValue.encode(message.listValue, writer.uint32(/* id 6, wireType 2 =*/50).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified Value message, length delimited. Does not implicitly {@link google.protobuf.Value.verify|verify} messages.
             * @function encodeDelimited
             * @memberof google.protobuf.Value
             * @static
             * @param {google.protobuf.IValue} message Value message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Value.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a Value message from the specified reader or buffer.
             * @function decode
             * @memberof google.protobuf.Value
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {google.protobuf.Value} Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Value.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.google.protobuf.Value();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.nullValue = reader.int32();
                        break;
                    case 2:
                        message.numberValue = reader.double();
                        break;
                    case 3:
                        message.stringValue = reader.string();
                        break;
                    case 4:
                        message.boolValue = reader.bool();
                        break;
                    case 5:
                        message.structValue = $root.google.protobuf.Struct.decode(reader, reader.uint32());
                        break;
                    case 6:
                        message.listValue = $root.google.protobuf.ListValue.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a Value message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof google.protobuf.Value
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {google.protobuf.Value} Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Value.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a Value message.
             * @function verify
             * @memberof google.protobuf.Value
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Value.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                var properties = {};
                if (message.nullValue != null && message.hasOwnProperty("nullValue")) {
                    properties.kind = 1;
                    switch (message.nullValue) {
                    default:
                        return "nullValue: enum value expected";
                    case 0:
                        break;
                    }
                }
                if (message.numberValue != null && message.hasOwnProperty("numberValue")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    if (typeof message.numberValue !== "number")
                        return "numberValue: number expected";
                }
                if (message.stringValue != null && message.hasOwnProperty("stringValue")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    if (!$util.isString(message.stringValue))
                        return "stringValue: string expected";
                }
                if (message.boolValue != null && message.hasOwnProperty("boolValue")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    if (typeof message.boolValue !== "boolean")
                        return "boolValue: boolean expected";
                }
                if (message.structValue != null && message.hasOwnProperty("structValue")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    {
                        var error = $root.google.protobuf.Struct.verify(message.structValue);
                        if (error)
                            return "structValue." + error;
                    }
                }
                if (message.listValue != null && message.hasOwnProperty("listValue")) {
                    if (properties.kind === 1)
                        return "kind: multiple values";
                    properties.kind = 1;
                    {
                        var error = $root.google.protobuf.ListValue.verify(message.listValue);
                        if (error)
                            return "listValue." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a Value message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof google.protobuf.Value
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {google.protobuf.Value} Value
             */
            Value.fromObject = function fromObject(object) {
                if (object instanceof $root.google.protobuf.Value)
                    return object;
                var message = new $root.google.protobuf.Value();
                switch (object.nullValue) {
                case "NULL_VALUE":
                case 0:
                    message.nullValue = 0;
                    break;
                }
                if (object.numberValue != null)
                    message.numberValue = Number(object.numberValue);
                if (object.stringValue != null)
                    message.stringValue = String(object.stringValue);
                if (object.boolValue != null)
                    message.boolValue = Boolean(object.boolValue);
                if (object.structValue != null) {
                    if (typeof object.structValue !== "object")
                        throw TypeError(".google.protobuf.Value.structValue: object expected");
                    message.structValue = $root.google.protobuf.Struct.fromObject(object.structValue);
                }
                if (object.listValue != null) {
                    if (typeof object.listValue !== "object")
                        throw TypeError(".google.protobuf.Value.listValue: object expected");
                    message.listValue = $root.google.protobuf.ListValue.fromObject(object.listValue);
                }
                return message;
            };

            /**
             * Creates a plain object from a Value message. Also converts values to other types if specified.
             * @function toObject
             * @memberof google.protobuf.Value
             * @static
             * @param {google.protobuf.Value} message Value
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Value.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (message.nullValue != null && message.hasOwnProperty("nullValue")) {
                    object.nullValue = options.enums === String ? $root.google.protobuf.NullValue[message.nullValue] : message.nullValue;
                    if (options.oneofs)
                        object.kind = "nullValue";
                }
                if (message.numberValue != null && message.hasOwnProperty("numberValue")) {
                    object.numberValue = options.json && !isFinite(message.numberValue) ? String(message.numberValue) : message.numberValue;
                    if (options.oneofs)
                        object.kind = "numberValue";
                }
                if (message.stringValue != null && message.hasOwnProperty("stringValue")) {
                    object.stringValue = message.stringValue;
                    if (options.oneofs)
                        object.kind = "stringValue";
                }
                if (message.boolValue != null && message.hasOwnProperty("boolValue")) {
                    object.boolValue = message.boolValue;
                    if (options.oneofs)
                        object.kind = "boolValue";
                }
                if (message.structValue != null && message.hasOwnProperty("structValue")) {
                    object.structValue = $root.google.protobuf.Struct.toObject(message.structValue, options);
                    if (options.oneofs)
                        object.kind = "structValue";
                }
                if (message.listValue != null && message.hasOwnProperty("listValue")) {
                    object.listValue = $root.google.protobuf.ListValue.toObject(message.listValue, options);
                    if (options.oneofs)
                        object.kind = "listValue";
                }
                return object;
            };

            /**
             * Converts this Value to JSON.
             * @function toJSON
             * @memberof google.protobuf.Value
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Value.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return Value;
        })();

        /**
         * NullValue enum.
         * @name google.protobuf.NullValue
         * @enum {number}
         * @property {number} NULL_VALUE=0 NULL_VALUE value
         */
        protobuf.NullValue = (function() {
            var valuesById = {}, values = Object.create(valuesById);
            values[valuesById[0] = "NULL_VALUE"] = 0;
            return values;
        })();

        protobuf.ListValue = (function() {

            /**
             * Properties of a ListValue.
             * @memberof google.protobuf
             * @interface IListValue
             * @property {Array.<google.protobuf.IValue>|null} [values] ListValue values
             */

            /**
             * Constructs a new ListValue.
             * @memberof google.protobuf
             * @classdesc Represents a ListValue.
             * @implements IListValue
             * @constructor
             * @param {google.protobuf.IListValue=} [properties] Properties to set
             */
            function ListValue(properties) {
                this.values = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ListValue values.
             * @member {Array.<google.protobuf.IValue>} values
             * @memberof google.protobuf.ListValue
             * @instance
             */
            ListValue.prototype.values = $util.emptyArray;

            /**
             * Creates a new ListValue instance using the specified properties.
             * @function create
             * @memberof google.protobuf.ListValue
             * @static
             * @param {google.protobuf.IListValue=} [properties] Properties to set
             * @returns {google.protobuf.ListValue} ListValue instance
             */
            ListValue.create = function create(properties) {
                return new ListValue(properties);
            };

            /**
             * Encodes the specified ListValue message. Does not implicitly {@link google.protobuf.ListValue.verify|verify} messages.
             * @function encode
             * @memberof google.protobuf.ListValue
             * @static
             * @param {google.protobuf.IListValue} message ListValue message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ListValue.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.values != null && message.values.length)
                    for (var i = 0; i < message.values.length; ++i)
                        $root.google.protobuf.Value.encode(message.values[i], writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ListValue message, length delimited. Does not implicitly {@link google.protobuf.ListValue.verify|verify} messages.
             * @function encodeDelimited
             * @memberof google.protobuf.ListValue
             * @static
             * @param {google.protobuf.IListValue} message ListValue message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ListValue.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ListValue message from the specified reader or buffer.
             * @function decode
             * @memberof google.protobuf.ListValue
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {google.protobuf.ListValue} ListValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ListValue.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.google.protobuf.ListValue();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        if (!(message.values && message.values.length))
                            message.values = [];
                        message.values.push($root.google.protobuf.Value.decode(reader, reader.uint32()));
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ListValue message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof google.protobuf.ListValue
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {google.protobuf.ListValue} ListValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ListValue.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ListValue message.
             * @function verify
             * @memberof google.protobuf.ListValue
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ListValue.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.values != null && message.hasOwnProperty("values")) {
                    if (!Array.isArray(message.values))
                        return "values: array expected";
                    for (var i = 0; i < message.values.length; ++i) {
                        var error = $root.google.protobuf.Value.verify(message.values[i]);
                        if (error)
                            return "values." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a ListValue message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof google.protobuf.ListValue
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {google.protobuf.ListValue} ListValue
             */
            ListValue.fromObject = function fromObject(object) {
                if (object instanceof $root.google.protobuf.ListValue)
                    return object;
                var message = new $root.google.protobuf.ListValue();
                if (object.values) {
                    if (!Array.isArray(object.values))
                        throw TypeError(".google.protobuf.ListValue.values: array expected");
                    message.values = [];
                    for (var i = 0; i < object.values.length; ++i) {
                        if (typeof object.values[i] !== "object")
                            throw TypeError(".google.protobuf.ListValue.values: object expected");
                        message.values[i] = $root.google.protobuf.Value.fromObject(object.values[i]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a ListValue message. Also converts values to other types if specified.
             * @function toObject
             * @memberof google.protobuf.ListValue
             * @static
             * @param {google.protobuf.ListValue} message ListValue
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ListValue.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.values = [];
                if (message.values && message.values.length) {
                    object.values = [];
                    for (var j = 0; j < message.values.length; ++j)
                        object.values[j] = $root.google.protobuf.Value.toObject(message.values[j], options);
                }
                return object;
            };

            /**
             * Converts this ListValue to JSON.
             * @function toJSON
             * @memberof google.protobuf.ListValue
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ListValue.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return ListValue;
        })();

        return protobuf;
    })();

    google.rpc = (function() {

        /**
         * Namespace rpc.
         * @memberof google
         * @namespace
         */
        var rpc = {};

        rpc.Status = (function() {

            /**
             * Properties of a Status.
             * @memberof google.rpc
             * @interface IStatus
             * @property {number|null} [code] Status code
             * @property {string|null} [message] Status message
             * @property {Array.<google.protobuf.IAny>|null} [details] Status details
             */

            /**
             * Constructs a new Status.
             * @memberof google.rpc
             * @classdesc Represents a Status.
             * @implements IStatus
             * @constructor
             * @param {google.rpc.IStatus=} [properties] Properties to set
             */
            function Status(properties) {
                this.details = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Status code.
             * @member {number} code
             * @memberof google.rpc.Status
             * @instance
             */
            Status.prototype.code = 0;

            /**
             * Status message.
             * @member {string} message
             * @memberof google.rpc.Status
             * @instance
             */
            Status.prototype.message = "";

            /**
             * Status details.
             * @member {Array.<google.protobuf.IAny>} details
             * @memberof google.rpc.Status
             * @instance
             */
            Status.prototype.details = $util.emptyArray;

            /**
             * Creates a new Status instance using the specified properties.
             * @function create
             * @memberof google.rpc.Status
             * @static
             * @param {google.rpc.IStatus=} [properties] Properties to set
             * @returns {google.rpc.Status} Status instance
             */
            Status.create = function create(properties) {
                return new Status(properties);
            };

            /**
             * Encodes the specified Status message. Does not implicitly {@link google.rpc.Status.verify|verify} messages.
             * @function encode
             * @memberof google.rpc.Status
             * @static
             * @param {google.rpc.IStatus} message Status message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Status.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.code != null && Object.hasOwnProperty.call(message, "code"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int32(message.code);
                if (message.message != null && Object.hasOwnProperty.call(message, "message"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.message);
                if (message.details != null && message.details.length)
                    for (var i = 0; i < message.details.length; ++i)
                        $root.google.protobuf.Any.encode(message.details[i], writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified Status message, length delimited. Does not implicitly {@link google.rpc.Status.verify|verify} messages.
             * @function encodeDelimited
             * @memberof google.rpc.Status
             * @static
             * @param {google.rpc.IStatus} message Status message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Status.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a Status message from the specified reader or buffer.
             * @function decode
             * @memberof google.rpc.Status
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {google.rpc.Status} Status
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Status.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.google.rpc.Status();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.code = reader.int32();
                        break;
                    case 2:
                        message.message = reader.string();
                        break;
                    case 3:
                        if (!(message.details && message.details.length))
                            message.details = [];
                        message.details.push($root.google.protobuf.Any.decode(reader, reader.uint32()));
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a Status message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof google.rpc.Status
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {google.rpc.Status} Status
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Status.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a Status message.
             * @function verify
             * @memberof google.rpc.Status
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Status.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.code != null && message.hasOwnProperty("code"))
                    if (!$util.isInteger(message.code))
                        return "code: integer expected";
                if (message.message != null && message.hasOwnProperty("message"))
                    if (!$util.isString(message.message))
                        return "message: string expected";
                if (message.details != null && message.hasOwnProperty("details")) {
                    if (!Array.isArray(message.details))
                        return "details: array expected";
                    for (var i = 0; i < message.details.length; ++i) {
                        var error = $root.google.protobuf.Any.verify(message.details[i]);
                        if (error)
                            return "details." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a Status message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof google.rpc.Status
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {google.rpc.Status} Status
             */
            Status.fromObject = function fromObject(object) {
                if (object instanceof $root.google.rpc.Status)
                    return object;
                var message = new $root.google.rpc.Status();
                if (object.code != null)
                    message.code = object.code | 0;
                if (object.message != null)
                    message.message = String(object.message);
                if (object.details) {
                    if (!Array.isArray(object.details))
                        throw TypeError(".google.rpc.Status.details: array expected");
                    message.details = [];
                    for (var i = 0; i < object.details.length; ++i) {
                        if (typeof object.details[i] !== "object")
                            throw TypeError(".google.rpc.Status.details: object expected");
                        message.details[i] = $root.google.protobuf.Any.fromObject(object.details[i]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a Status message. Also converts values to other types if specified.
             * @function toObject
             * @memberof google.rpc.Status
             * @static
             * @param {google.rpc.Status} message Status
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Status.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.details = [];
                if (options.defaults) {
                    object.code = 0;
                    object.message = "";
                }
                if (message.code != null && message.hasOwnProperty("code"))
                    object.code = message.code;
                if (message.message != null && message.hasOwnProperty("message"))
                    object.message = message.message;
                if (message.details && message.details.length) {
                    object.details = [];
                    for (var j = 0; j < message.details.length; ++j)
                        object.details[j] = $root.google.protobuf.Any.toObject(message.details[j], options);
                }
                return object;
            };

            /**
             * Converts this Status to JSON.
             * @function toJSON
             * @memberof google.rpc.Status
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Status.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            return Status;
        })();

        return rpc;
    })();

    return google;
})();

module.exports = $root;
