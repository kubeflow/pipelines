import * as $protobuf from "protobufjs";
/** Namespace ml_pipelines. */
export namespace ml_pipelines {

    /** Properties of a PipelineJob. */
    interface IPipelineJob {

        /** PipelineJob name */
        name?: (string|null);

        /** PipelineJob displayName */
        displayName?: (string|null);

        /** PipelineJob pipelineSpec */
        pipelineSpec?: (google.protobuf.IStruct|null);

        /** PipelineJob labels */
        labels?: ({ [k: string]: string }|null);

        /** PipelineJob runtimeConfig */
        runtimeConfig?: (ml_pipelines.PipelineJob.IRuntimeConfig|null);
    }

    /** Represents a PipelineJob. */
    class PipelineJob implements IPipelineJob {

        /**
         * Constructs a new PipelineJob.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IPipelineJob);

        /** PipelineJob name. */
        public name: string;

        /** PipelineJob displayName. */
        public displayName: string;

        /** PipelineJob pipelineSpec. */
        public pipelineSpec?: (google.protobuf.IStruct|null);

        /** PipelineJob labels. */
        public labels: { [k: string]: string };

        /** PipelineJob runtimeConfig. */
        public runtimeConfig?: (ml_pipelines.PipelineJob.IRuntimeConfig|null);

        /**
         * Creates a new PipelineJob instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PipelineJob instance
         */
        public static create(properties?: ml_pipelines.IPipelineJob): ml_pipelines.PipelineJob;

        /**
         * Encodes the specified PipelineJob message. Does not implicitly {@link ml_pipelines.PipelineJob.verify|verify} messages.
         * @param message PipelineJob message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IPipelineJob, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PipelineJob message, length delimited. Does not implicitly {@link ml_pipelines.PipelineJob.verify|verify} messages.
         * @param message PipelineJob message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IPipelineJob, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PipelineJob message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns PipelineJob
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineJob;

        /**
         * Decodes a PipelineJob message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns PipelineJob
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineJob;

        /**
         * Verifies a PipelineJob message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PipelineJob message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PipelineJob
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineJob;

        /**
         * Creates a plain object from a PipelineJob message. Also converts values to other types if specified.
         * @param message PipelineJob
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.PipelineJob, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PipelineJob to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace PipelineJob {

        /** Properties of a RuntimeConfig. */
        interface IRuntimeConfig {

            /** RuntimeConfig parameters */
            parameters?: ({ [k: string]: ml_pipelines.IValue }|null);

            /** RuntimeConfig gcsOutputDirectory */
            gcsOutputDirectory?: (string|null);
        }

        /** Represents a RuntimeConfig. */
        class RuntimeConfig implements IRuntimeConfig {

            /**
             * Constructs a new RuntimeConfig.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.PipelineJob.IRuntimeConfig);

            /** RuntimeConfig parameters. */
            public parameters: { [k: string]: ml_pipelines.IValue };

            /** RuntimeConfig gcsOutputDirectory. */
            public gcsOutputDirectory: string;

            /**
             * Creates a new RuntimeConfig instance using the specified properties.
             * @param [properties] Properties to set
             * @returns RuntimeConfig instance
             */
            public static create(properties?: ml_pipelines.PipelineJob.IRuntimeConfig): ml_pipelines.PipelineJob.RuntimeConfig;

            /**
             * Encodes the specified RuntimeConfig message. Does not implicitly {@link ml_pipelines.PipelineJob.RuntimeConfig.verify|verify} messages.
             * @param message RuntimeConfig message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.PipelineJob.IRuntimeConfig, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified RuntimeConfig message, length delimited. Does not implicitly {@link ml_pipelines.PipelineJob.RuntimeConfig.verify|verify} messages.
             * @param message RuntimeConfig message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.PipelineJob.IRuntimeConfig, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a RuntimeConfig message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns RuntimeConfig
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineJob.RuntimeConfig;

            /**
             * Decodes a RuntimeConfig message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns RuntimeConfig
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineJob.RuntimeConfig;

            /**
             * Verifies a RuntimeConfig message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a RuntimeConfig message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns RuntimeConfig
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineJob.RuntimeConfig;

            /**
             * Creates a plain object from a RuntimeConfig message. Also converts values to other types if specified.
             * @param message RuntimeConfig
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.PipelineJob.RuntimeConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this RuntimeConfig to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of a PipelineSpec. */
    interface IPipelineSpec {

        /** PipelineSpec pipelineInfo */
        pipelineInfo?: (ml_pipelines.IPipelineInfo|null);

        /** PipelineSpec deploymentSpec */
        deploymentSpec?: (google.protobuf.IStruct|null);

        /** PipelineSpec sdkVersion */
        sdkVersion?: (string|null);

        /** PipelineSpec schemaVersion */
        schemaVersion?: (string|null);

        /** PipelineSpec components */
        components?: ({ [k: string]: ml_pipelines.IComponentSpec }|null);

        /** PipelineSpec root */
        root?: (ml_pipelines.IComponentSpec|null);
    }

    /** Represents a PipelineSpec. */
    class PipelineSpec implements IPipelineSpec {

        /**
         * Constructs a new PipelineSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IPipelineSpec);

        /** PipelineSpec pipelineInfo. */
        public pipelineInfo?: (ml_pipelines.IPipelineInfo|null);

        /** PipelineSpec deploymentSpec. */
        public deploymentSpec?: (google.protobuf.IStruct|null);

        /** PipelineSpec sdkVersion. */
        public sdkVersion: string;

        /** PipelineSpec schemaVersion. */
        public schemaVersion: string;

        /** PipelineSpec components. */
        public components: { [k: string]: ml_pipelines.IComponentSpec };

        /** PipelineSpec root. */
        public root?: (ml_pipelines.IComponentSpec|null);

        /**
         * Creates a new PipelineSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PipelineSpec instance
         */
        public static create(properties?: ml_pipelines.IPipelineSpec): ml_pipelines.PipelineSpec;

        /**
         * Encodes the specified PipelineSpec message. Does not implicitly {@link ml_pipelines.PipelineSpec.verify|verify} messages.
         * @param message PipelineSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IPipelineSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PipelineSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineSpec.verify|verify} messages.
         * @param message PipelineSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IPipelineSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PipelineSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns PipelineSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineSpec;

        /**
         * Decodes a PipelineSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns PipelineSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineSpec;

        /**
         * Verifies a PipelineSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PipelineSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PipelineSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineSpec;

        /**
         * Creates a plain object from a PipelineSpec message. Also converts values to other types if specified.
         * @param message PipelineSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.PipelineSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PipelineSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace PipelineSpec {

        /** Properties of a RuntimeParameter. */
        interface IRuntimeParameter {

            /** RuntimeParameter type */
            type?: (ml_pipelines.PrimitiveType.PrimitiveTypeEnum|null);

            /** RuntimeParameter defaultValue */
            defaultValue?: (ml_pipelines.IValue|null);
        }

        /** Represents a RuntimeParameter. */
        class RuntimeParameter implements IRuntimeParameter {

            /**
             * Constructs a new RuntimeParameter.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.PipelineSpec.IRuntimeParameter);

            /** RuntimeParameter type. */
            public type: ml_pipelines.PrimitiveType.PrimitiveTypeEnum;

            /** RuntimeParameter defaultValue. */
            public defaultValue?: (ml_pipelines.IValue|null);

            /**
             * Creates a new RuntimeParameter instance using the specified properties.
             * @param [properties] Properties to set
             * @returns RuntimeParameter instance
             */
            public static create(properties?: ml_pipelines.PipelineSpec.IRuntimeParameter): ml_pipelines.PipelineSpec.RuntimeParameter;

            /**
             * Encodes the specified RuntimeParameter message. Does not implicitly {@link ml_pipelines.PipelineSpec.RuntimeParameter.verify|verify} messages.
             * @param message RuntimeParameter message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.PipelineSpec.IRuntimeParameter, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified RuntimeParameter message, length delimited. Does not implicitly {@link ml_pipelines.PipelineSpec.RuntimeParameter.verify|verify} messages.
             * @param message RuntimeParameter message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.PipelineSpec.IRuntimeParameter, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a RuntimeParameter message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns RuntimeParameter
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineSpec.RuntimeParameter;

            /**
             * Decodes a RuntimeParameter message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns RuntimeParameter
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineSpec.RuntimeParameter;

            /**
             * Verifies a RuntimeParameter message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a RuntimeParameter message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns RuntimeParameter
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineSpec.RuntimeParameter;

            /**
             * Creates a plain object from a RuntimeParameter message. Also converts values to other types if specified.
             * @param message RuntimeParameter
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.PipelineSpec.RuntimeParameter, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this RuntimeParameter to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of a ComponentSpec. */
    interface IComponentSpec {

        /** ComponentSpec inputDefinitions */
        inputDefinitions?: (ml_pipelines.IComponentInputsSpec|null);

        /** ComponentSpec outputDefinitions */
        outputDefinitions?: (ml_pipelines.IComponentOutputsSpec|null);

        /** ComponentSpec dag */
        dag?: (ml_pipelines.IDagSpec|null);

        /** ComponentSpec executorLabel */
        executorLabel?: (string|null);
    }

    /** Represents a ComponentSpec. */
    class ComponentSpec implements IComponentSpec {

        /**
         * Constructs a new ComponentSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IComponentSpec);

        /** ComponentSpec inputDefinitions. */
        public inputDefinitions?: (ml_pipelines.IComponentInputsSpec|null);

        /** ComponentSpec outputDefinitions. */
        public outputDefinitions?: (ml_pipelines.IComponentOutputsSpec|null);

        /** ComponentSpec dag. */
        public dag?: (ml_pipelines.IDagSpec|null);

        /** ComponentSpec executorLabel. */
        public executorLabel?: (string|null);

        /** ComponentSpec implementation. */
        public implementation?: ("dag"|"executorLabel");

        /**
         * Creates a new ComponentSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ComponentSpec instance
         */
        public static create(properties?: ml_pipelines.IComponentSpec): ml_pipelines.ComponentSpec;

        /**
         * Encodes the specified ComponentSpec message. Does not implicitly {@link ml_pipelines.ComponentSpec.verify|verify} messages.
         * @param message ComponentSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IComponentSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ComponentSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentSpec.verify|verify} messages.
         * @param message ComponentSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IComponentSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ComponentSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ComponentSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ComponentSpec;

        /**
         * Decodes a ComponentSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ComponentSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ComponentSpec;

        /**
         * Verifies a ComponentSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ComponentSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ComponentSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ComponentSpec;

        /**
         * Creates a plain object from a ComponentSpec message. Also converts values to other types if specified.
         * @param message ComponentSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ComponentSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ComponentSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of a DagSpec. */
    interface IDagSpec {

        /** DagSpec tasks */
        tasks?: ({ [k: string]: ml_pipelines.IPipelineTaskSpec }|null);

        /** DagSpec outputs */
        outputs?: (ml_pipelines.IDagOutputsSpec|null);
    }

    /** Represents a DagSpec. */
    class DagSpec implements IDagSpec {

        /**
         * Constructs a new DagSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IDagSpec);

        /** DagSpec tasks. */
        public tasks: { [k: string]: ml_pipelines.IPipelineTaskSpec };

        /** DagSpec outputs. */
        public outputs?: (ml_pipelines.IDagOutputsSpec|null);

        /**
         * Creates a new DagSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns DagSpec instance
         */
        public static create(properties?: ml_pipelines.IDagSpec): ml_pipelines.DagSpec;

        /**
         * Encodes the specified DagSpec message. Does not implicitly {@link ml_pipelines.DagSpec.verify|verify} messages.
         * @param message DagSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IDagSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified DagSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagSpec.verify|verify} messages.
         * @param message DagSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IDagSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a DagSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns DagSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.DagSpec;

        /**
         * Decodes a DagSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns DagSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.DagSpec;

        /**
         * Verifies a DagSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a DagSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns DagSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.DagSpec;

        /**
         * Creates a plain object from a DagSpec message. Also converts values to other types if specified.
         * @param message DagSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.DagSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this DagSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of a DagOutputsSpec. */
    interface IDagOutputsSpec {

        /** DagOutputsSpec artifacts */
        artifacts?: ({ [k: string]: ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec }|null);

        /** DagOutputsSpec parameters */
        parameters?: ({ [k: string]: ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec }|null);
    }

    /** Represents a DagOutputsSpec. */
    class DagOutputsSpec implements IDagOutputsSpec {

        /**
         * Constructs a new DagOutputsSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IDagOutputsSpec);

        /** DagOutputsSpec artifacts. */
        public artifacts: { [k: string]: ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec };

        /** DagOutputsSpec parameters. */
        public parameters: { [k: string]: ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec };

        /**
         * Creates a new DagOutputsSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns DagOutputsSpec instance
         */
        public static create(properties?: ml_pipelines.IDagOutputsSpec): ml_pipelines.DagOutputsSpec;

        /**
         * Encodes the specified DagOutputsSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.verify|verify} messages.
         * @param message DagOutputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IDagOutputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified DagOutputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.verify|verify} messages.
         * @param message DagOutputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IDagOutputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a DagOutputsSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns DagOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.DagOutputsSpec;

        /**
         * Decodes a DagOutputsSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns DagOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.DagOutputsSpec;

        /**
         * Verifies a DagOutputsSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a DagOutputsSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns DagOutputsSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.DagOutputsSpec;

        /**
         * Creates a plain object from a DagOutputsSpec message. Also converts values to other types if specified.
         * @param message DagOutputsSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.DagOutputsSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this DagOutputsSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace DagOutputsSpec {

        /** Properties of an ArtifactSelectorSpec. */
        interface IArtifactSelectorSpec {

            /** ArtifactSelectorSpec producerSubtask */
            producerSubtask?: (string|null);

            /** ArtifactSelectorSpec outputArtifactKey */
            outputArtifactKey?: (string|null);
        }

        /** Represents an ArtifactSelectorSpec. */
        class ArtifactSelectorSpec implements IArtifactSelectorSpec {

            /**
             * Constructs a new ArtifactSelectorSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec);

            /** ArtifactSelectorSpec producerSubtask. */
            public producerSubtask: string;

            /** ArtifactSelectorSpec outputArtifactKey. */
            public outputArtifactKey: string;

            /**
             * Creates a new ArtifactSelectorSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ArtifactSelectorSpec instance
             */
            public static create(properties?: ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec): ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec;

            /**
             * Encodes the specified ArtifactSelectorSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.verify|verify} messages.
             * @param message ArtifactSelectorSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ArtifactSelectorSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec.verify|verify} messages.
             * @param message ArtifactSelectorSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an ArtifactSelectorSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ArtifactSelectorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec;

            /**
             * Decodes an ArtifactSelectorSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ArtifactSelectorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec;

            /**
             * Verifies an ArtifactSelectorSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an ArtifactSelectorSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ArtifactSelectorSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec;

            /**
             * Creates a plain object from an ArtifactSelectorSpec message. Also converts values to other types if specified.
             * @param message ArtifactSelectorSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.DagOutputsSpec.ArtifactSelectorSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ArtifactSelectorSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a DagOutputArtifactSpec. */
        interface IDagOutputArtifactSpec {

            /** DagOutputArtifactSpec artifactSelectors */
            artifactSelectors?: (ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec[]|null);
        }

        /** Represents a DagOutputArtifactSpec. */
        class DagOutputArtifactSpec implements IDagOutputArtifactSpec {

            /**
             * Constructs a new DagOutputArtifactSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec);

            /** DagOutputArtifactSpec artifactSelectors. */
            public artifactSelectors: ml_pipelines.DagOutputsSpec.IArtifactSelectorSpec[];

            /**
             * Creates a new DagOutputArtifactSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns DagOutputArtifactSpec instance
             */
            public static create(properties?: ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec): ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec;

            /**
             * Encodes the specified DagOutputArtifactSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.verify|verify} messages.
             * @param message DagOutputArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified DagOutputArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec.verify|verify} messages.
             * @param message DagOutputArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.DagOutputsSpec.IDagOutputArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a DagOutputArtifactSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns DagOutputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec;

            /**
             * Decodes a DagOutputArtifactSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns DagOutputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec;

            /**
             * Verifies a DagOutputArtifactSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a DagOutputArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns DagOutputArtifactSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec;

            /**
             * Creates a plain object from a DagOutputArtifactSpec message. Also converts values to other types if specified.
             * @param message DagOutputArtifactSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.DagOutputsSpec.DagOutputArtifactSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this DagOutputArtifactSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a ParameterSelectorSpec. */
        interface IParameterSelectorSpec {

            /** ParameterSelectorSpec producerSubtask */
            producerSubtask?: (string|null);

            /** ParameterSelectorSpec outputParameterKey */
            outputParameterKey?: (string|null);
        }

        /** Represents a ParameterSelectorSpec. */
        class ParameterSelectorSpec implements IParameterSelectorSpec {

            /**
             * Constructs a new ParameterSelectorSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.DagOutputsSpec.IParameterSelectorSpec);

            /** ParameterSelectorSpec producerSubtask. */
            public producerSubtask: string;

            /** ParameterSelectorSpec outputParameterKey. */
            public outputParameterKey: string;

            /**
             * Creates a new ParameterSelectorSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ParameterSelectorSpec instance
             */
            public static create(properties?: ml_pipelines.DagOutputsSpec.IParameterSelectorSpec): ml_pipelines.DagOutputsSpec.ParameterSelectorSpec;

            /**
             * Encodes the specified ParameterSelectorSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.verify|verify} messages.
             * @param message ParameterSelectorSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.DagOutputsSpec.IParameterSelectorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ParameterSelectorSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ParameterSelectorSpec.verify|verify} messages.
             * @param message ParameterSelectorSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.DagOutputsSpec.IParameterSelectorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a ParameterSelectorSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ParameterSelectorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.DagOutputsSpec.ParameterSelectorSpec;

            /**
             * Decodes a ParameterSelectorSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ParameterSelectorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.DagOutputsSpec.ParameterSelectorSpec;

            /**
             * Verifies a ParameterSelectorSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a ParameterSelectorSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ParameterSelectorSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.DagOutputsSpec.ParameterSelectorSpec;

            /**
             * Creates a plain object from a ParameterSelectorSpec message. Also converts values to other types if specified.
             * @param message ParameterSelectorSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.DagOutputsSpec.ParameterSelectorSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ParameterSelectorSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a ParameterSelectorsSpec. */
        interface IParameterSelectorsSpec {

            /** ParameterSelectorsSpec parameterSelectors */
            parameterSelectors?: (ml_pipelines.DagOutputsSpec.IParameterSelectorSpec[]|null);
        }

        /** Represents a ParameterSelectorsSpec. */
        class ParameterSelectorsSpec implements IParameterSelectorsSpec {

            /**
             * Constructs a new ParameterSelectorsSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec);

            /** ParameterSelectorsSpec parameterSelectors. */
            public parameterSelectors: ml_pipelines.DagOutputsSpec.IParameterSelectorSpec[];

            /**
             * Creates a new ParameterSelectorsSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ParameterSelectorsSpec instance
             */
            public static create(properties?: ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec): ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec;

            /**
             * Encodes the specified ParameterSelectorsSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.verify|verify} messages.
             * @param message ParameterSelectorsSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ParameterSelectorsSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec.verify|verify} messages.
             * @param message ParameterSelectorsSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a ParameterSelectorsSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ParameterSelectorsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec;

            /**
             * Decodes a ParameterSelectorsSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ParameterSelectorsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec;

            /**
             * Verifies a ParameterSelectorsSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a ParameterSelectorsSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ParameterSelectorsSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec;

            /**
             * Creates a plain object from a ParameterSelectorsSpec message. Also converts values to other types if specified.
             * @param message ParameterSelectorsSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.DagOutputsSpec.ParameterSelectorsSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ParameterSelectorsSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a MapParameterSelectorsSpec. */
        interface IMapParameterSelectorsSpec {

            /** MapParameterSelectorsSpec mappedParameters */
            mappedParameters?: ({ [k: string]: ml_pipelines.DagOutputsSpec.IParameterSelectorSpec }|null);
        }

        /** Represents a MapParameterSelectorsSpec. */
        class MapParameterSelectorsSpec implements IMapParameterSelectorsSpec {

            /**
             * Constructs a new MapParameterSelectorsSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.DagOutputsSpec.IMapParameterSelectorsSpec);

            /** MapParameterSelectorsSpec mappedParameters. */
            public mappedParameters: { [k: string]: ml_pipelines.DagOutputsSpec.IParameterSelectorSpec };

            /**
             * Creates a new MapParameterSelectorsSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns MapParameterSelectorsSpec instance
             */
            public static create(properties?: ml_pipelines.DagOutputsSpec.IMapParameterSelectorsSpec): ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec;

            /**
             * Encodes the specified MapParameterSelectorsSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.verify|verify} messages.
             * @param message MapParameterSelectorsSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.DagOutputsSpec.IMapParameterSelectorsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified MapParameterSelectorsSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec.verify|verify} messages.
             * @param message MapParameterSelectorsSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.DagOutputsSpec.IMapParameterSelectorsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a MapParameterSelectorsSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns MapParameterSelectorsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec;

            /**
             * Decodes a MapParameterSelectorsSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns MapParameterSelectorsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec;

            /**
             * Verifies a MapParameterSelectorsSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a MapParameterSelectorsSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns MapParameterSelectorsSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec;

            /**
             * Creates a plain object from a MapParameterSelectorsSpec message. Also converts values to other types if specified.
             * @param message MapParameterSelectorsSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.DagOutputsSpec.MapParameterSelectorsSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this MapParameterSelectorsSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a DagOutputParameterSpec. */
        interface IDagOutputParameterSpec {

            /** DagOutputParameterSpec valueFromParameter */
            valueFromParameter?: (ml_pipelines.DagOutputsSpec.IParameterSelectorSpec|null);

            /** DagOutputParameterSpec valueFromOneof */
            valueFromOneof?: (ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec|null);
        }

        /** Represents a DagOutputParameterSpec. */
        class DagOutputParameterSpec implements IDagOutputParameterSpec {

            /**
             * Constructs a new DagOutputParameterSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec);

            /** DagOutputParameterSpec valueFromParameter. */
            public valueFromParameter?: (ml_pipelines.DagOutputsSpec.IParameterSelectorSpec|null);

            /** DagOutputParameterSpec valueFromOneof. */
            public valueFromOneof?: (ml_pipelines.DagOutputsSpec.IParameterSelectorsSpec|null);

            /** DagOutputParameterSpec kind. */
            public kind?: ("valueFromParameter"|"valueFromOneof");

            /**
             * Creates a new DagOutputParameterSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns DagOutputParameterSpec instance
             */
            public static create(properties?: ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec): ml_pipelines.DagOutputsSpec.DagOutputParameterSpec;

            /**
             * Encodes the specified DagOutputParameterSpec message. Does not implicitly {@link ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.verify|verify} messages.
             * @param message DagOutputParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified DagOutputParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.DagOutputsSpec.DagOutputParameterSpec.verify|verify} messages.
             * @param message DagOutputParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.DagOutputsSpec.IDagOutputParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a DagOutputParameterSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns DagOutputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.DagOutputsSpec.DagOutputParameterSpec;

            /**
             * Decodes a DagOutputParameterSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns DagOutputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.DagOutputsSpec.DagOutputParameterSpec;

            /**
             * Verifies a DagOutputParameterSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a DagOutputParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns DagOutputParameterSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.DagOutputsSpec.DagOutputParameterSpec;

            /**
             * Creates a plain object from a DagOutputParameterSpec message. Also converts values to other types if specified.
             * @param message DagOutputParameterSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.DagOutputsSpec.DagOutputParameterSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this DagOutputParameterSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of a ComponentInputsSpec. */
    interface IComponentInputsSpec {

        /** ComponentInputsSpec artifacts */
        artifacts?: ({ [k: string]: ml_pipelines.ComponentInputsSpec.IArtifactSpec }|null);

        /** ComponentInputsSpec parameters */
        parameters?: ({ [k: string]: ml_pipelines.ComponentInputsSpec.IParameterSpec }|null);
    }

    /** Represents a ComponentInputsSpec. */
    class ComponentInputsSpec implements IComponentInputsSpec {

        /**
         * Constructs a new ComponentInputsSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IComponentInputsSpec);

        /** ComponentInputsSpec artifacts. */
        public artifacts: { [k: string]: ml_pipelines.ComponentInputsSpec.IArtifactSpec };

        /** ComponentInputsSpec parameters. */
        public parameters: { [k: string]: ml_pipelines.ComponentInputsSpec.IParameterSpec };

        /**
         * Creates a new ComponentInputsSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ComponentInputsSpec instance
         */
        public static create(properties?: ml_pipelines.IComponentInputsSpec): ml_pipelines.ComponentInputsSpec;

        /**
         * Encodes the specified ComponentInputsSpec message. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.verify|verify} messages.
         * @param message ComponentInputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IComponentInputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ComponentInputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.verify|verify} messages.
         * @param message ComponentInputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IComponentInputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ComponentInputsSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ComponentInputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ComponentInputsSpec;

        /**
         * Decodes a ComponentInputsSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ComponentInputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ComponentInputsSpec;

        /**
         * Verifies a ComponentInputsSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ComponentInputsSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ComponentInputsSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ComponentInputsSpec;

        /**
         * Creates a plain object from a ComponentInputsSpec message. Also converts values to other types if specified.
         * @param message ComponentInputsSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ComponentInputsSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ComponentInputsSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace ComponentInputsSpec {

        /** Properties of an ArtifactSpec. */
        interface IArtifactSpec {

            /** ArtifactSpec artifactType */
            artifactType?: (ml_pipelines.IArtifactTypeSchema|null);
        }

        /** Represents an ArtifactSpec. */
        class ArtifactSpec implements IArtifactSpec {

            /**
             * Constructs a new ArtifactSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.ComponentInputsSpec.IArtifactSpec);

            /** ArtifactSpec artifactType. */
            public artifactType?: (ml_pipelines.IArtifactTypeSchema|null);

            /**
             * Creates a new ArtifactSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ArtifactSpec instance
             */
            public static create(properties?: ml_pipelines.ComponentInputsSpec.IArtifactSpec): ml_pipelines.ComponentInputsSpec.ArtifactSpec;

            /**
             * Encodes the specified ArtifactSpec message. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.ArtifactSpec.verify|verify} messages.
             * @param message ArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.ComponentInputsSpec.IArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.ArtifactSpec.verify|verify} messages.
             * @param message ArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.ComponentInputsSpec.IArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an ArtifactSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ComponentInputsSpec.ArtifactSpec;

            /**
             * Decodes an ArtifactSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ComponentInputsSpec.ArtifactSpec;

            /**
             * Verifies an ArtifactSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an ArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ArtifactSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.ComponentInputsSpec.ArtifactSpec;

            /**
             * Creates a plain object from an ArtifactSpec message. Also converts values to other types if specified.
             * @param message ArtifactSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.ComponentInputsSpec.ArtifactSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ArtifactSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a ParameterSpec. */
        interface IParameterSpec {

            /** ParameterSpec type */
            type?: (ml_pipelines.PrimitiveType.PrimitiveTypeEnum|null);
        }

        /** Represents a ParameterSpec. */
        class ParameterSpec implements IParameterSpec {

            /**
             * Constructs a new ParameterSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.ComponentInputsSpec.IParameterSpec);

            /** ParameterSpec type. */
            public type: ml_pipelines.PrimitiveType.PrimitiveTypeEnum;

            /**
             * Creates a new ParameterSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ParameterSpec instance
             */
            public static create(properties?: ml_pipelines.ComponentInputsSpec.IParameterSpec): ml_pipelines.ComponentInputsSpec.ParameterSpec;

            /**
             * Encodes the specified ParameterSpec message. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.ParameterSpec.verify|verify} messages.
             * @param message ParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.ComponentInputsSpec.IParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentInputsSpec.ParameterSpec.verify|verify} messages.
             * @param message ParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.ComponentInputsSpec.IParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a ParameterSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ComponentInputsSpec.ParameterSpec;

            /**
             * Decodes a ParameterSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ComponentInputsSpec.ParameterSpec;

            /**
             * Verifies a ParameterSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a ParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ParameterSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.ComponentInputsSpec.ParameterSpec;

            /**
             * Creates a plain object from a ParameterSpec message. Also converts values to other types if specified.
             * @param message ParameterSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.ComponentInputsSpec.ParameterSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ParameterSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of a ComponentOutputsSpec. */
    interface IComponentOutputsSpec {

        /** ComponentOutputsSpec artifacts */
        artifacts?: ({ [k: string]: ml_pipelines.ComponentOutputsSpec.IArtifactSpec }|null);

        /** ComponentOutputsSpec parameters */
        parameters?: ({ [k: string]: ml_pipelines.ComponentOutputsSpec.IParameterSpec }|null);
    }

    /** Represents a ComponentOutputsSpec. */
    class ComponentOutputsSpec implements IComponentOutputsSpec {

        /**
         * Constructs a new ComponentOutputsSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IComponentOutputsSpec);

        /** ComponentOutputsSpec artifacts. */
        public artifacts: { [k: string]: ml_pipelines.ComponentOutputsSpec.IArtifactSpec };

        /** ComponentOutputsSpec parameters. */
        public parameters: { [k: string]: ml_pipelines.ComponentOutputsSpec.IParameterSpec };

        /**
         * Creates a new ComponentOutputsSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ComponentOutputsSpec instance
         */
        public static create(properties?: ml_pipelines.IComponentOutputsSpec): ml_pipelines.ComponentOutputsSpec;

        /**
         * Encodes the specified ComponentOutputsSpec message. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.verify|verify} messages.
         * @param message ComponentOutputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IComponentOutputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ComponentOutputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.verify|verify} messages.
         * @param message ComponentOutputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IComponentOutputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ComponentOutputsSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ComponentOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ComponentOutputsSpec;

        /**
         * Decodes a ComponentOutputsSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ComponentOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ComponentOutputsSpec;

        /**
         * Verifies a ComponentOutputsSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ComponentOutputsSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ComponentOutputsSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ComponentOutputsSpec;

        /**
         * Creates a plain object from a ComponentOutputsSpec message. Also converts values to other types if specified.
         * @param message ComponentOutputsSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ComponentOutputsSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ComponentOutputsSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace ComponentOutputsSpec {

        /** Properties of an ArtifactSpec. */
        interface IArtifactSpec {

            /** ArtifactSpec artifactType */
            artifactType?: (ml_pipelines.IArtifactTypeSchema|null);

            /** ArtifactSpec properties */
            properties?: ({ [k: string]: ml_pipelines.IValueOrRuntimeParameter }|null);

            /** ArtifactSpec customProperties */
            customProperties?: ({ [k: string]: ml_pipelines.IValueOrRuntimeParameter }|null);

            /** ArtifactSpec metadata */
            metadata?: (google.protobuf.IStruct|null);
        }

        /** Represents an ArtifactSpec. */
        class ArtifactSpec implements IArtifactSpec {

            /**
             * Constructs a new ArtifactSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.ComponentOutputsSpec.IArtifactSpec);

            /** ArtifactSpec artifactType. */
            public artifactType?: (ml_pipelines.IArtifactTypeSchema|null);

            /** ArtifactSpec properties. */
            public properties: { [k: string]: ml_pipelines.IValueOrRuntimeParameter };

            /** ArtifactSpec customProperties. */
            public customProperties: { [k: string]: ml_pipelines.IValueOrRuntimeParameter };

            /** ArtifactSpec metadata. */
            public metadata?: (google.protobuf.IStruct|null);

            /**
             * Creates a new ArtifactSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ArtifactSpec instance
             */
            public static create(properties?: ml_pipelines.ComponentOutputsSpec.IArtifactSpec): ml_pipelines.ComponentOutputsSpec.ArtifactSpec;

            /**
             * Encodes the specified ArtifactSpec message. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.ArtifactSpec.verify|verify} messages.
             * @param message ArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.ComponentOutputsSpec.IArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.ArtifactSpec.verify|verify} messages.
             * @param message ArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.ComponentOutputsSpec.IArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an ArtifactSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ComponentOutputsSpec.ArtifactSpec;

            /**
             * Decodes an ArtifactSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ComponentOutputsSpec.ArtifactSpec;

            /**
             * Verifies an ArtifactSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an ArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ArtifactSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.ComponentOutputsSpec.ArtifactSpec;

            /**
             * Creates a plain object from an ArtifactSpec message. Also converts values to other types if specified.
             * @param message ArtifactSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.ComponentOutputsSpec.ArtifactSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ArtifactSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a ParameterSpec. */
        interface IParameterSpec {

            /** ParameterSpec type */
            type?: (ml_pipelines.PrimitiveType.PrimitiveTypeEnum|null);
        }

        /** Represents a ParameterSpec. */
        class ParameterSpec implements IParameterSpec {

            /**
             * Constructs a new ParameterSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.ComponentOutputsSpec.IParameterSpec);

            /** ParameterSpec type. */
            public type: ml_pipelines.PrimitiveType.PrimitiveTypeEnum;

            /**
             * Creates a new ParameterSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ParameterSpec instance
             */
            public static create(properties?: ml_pipelines.ComponentOutputsSpec.IParameterSpec): ml_pipelines.ComponentOutputsSpec.ParameterSpec;

            /**
             * Encodes the specified ParameterSpec message. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.ParameterSpec.verify|verify} messages.
             * @param message ParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.ComponentOutputsSpec.IParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.ComponentOutputsSpec.ParameterSpec.verify|verify} messages.
             * @param message ParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.ComponentOutputsSpec.IParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a ParameterSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ComponentOutputsSpec.ParameterSpec;

            /**
             * Decodes a ParameterSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ComponentOutputsSpec.ParameterSpec;

            /**
             * Verifies a ParameterSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a ParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ParameterSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.ComponentOutputsSpec.ParameterSpec;

            /**
             * Creates a plain object from a ParameterSpec message. Also converts values to other types if specified.
             * @param message ParameterSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.ComponentOutputsSpec.ParameterSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ParameterSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of a TaskInputsSpec. */
    interface ITaskInputsSpec {

        /** TaskInputsSpec parameters */
        parameters?: ({ [k: string]: ml_pipelines.TaskInputsSpec.IInputParameterSpec }|null);

        /** TaskInputsSpec artifacts */
        artifacts?: ({ [k: string]: ml_pipelines.TaskInputsSpec.IInputArtifactSpec }|null);
    }

    /** Represents a TaskInputsSpec. */
    class TaskInputsSpec implements ITaskInputsSpec {

        /**
         * Constructs a new TaskInputsSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.ITaskInputsSpec);

        /** TaskInputsSpec parameters. */
        public parameters: { [k: string]: ml_pipelines.TaskInputsSpec.IInputParameterSpec };

        /** TaskInputsSpec artifacts. */
        public artifacts: { [k: string]: ml_pipelines.TaskInputsSpec.IInputArtifactSpec };

        /**
         * Creates a new TaskInputsSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns TaskInputsSpec instance
         */
        public static create(properties?: ml_pipelines.ITaskInputsSpec): ml_pipelines.TaskInputsSpec;

        /**
         * Encodes the specified TaskInputsSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.verify|verify} messages.
         * @param message TaskInputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.ITaskInputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified TaskInputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.verify|verify} messages.
         * @param message TaskInputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.ITaskInputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a TaskInputsSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns TaskInputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.TaskInputsSpec;

        /**
         * Decodes a TaskInputsSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns TaskInputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.TaskInputsSpec;

        /**
         * Verifies a TaskInputsSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a TaskInputsSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns TaskInputsSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.TaskInputsSpec;

        /**
         * Creates a plain object from a TaskInputsSpec message. Also converts values to other types if specified.
         * @param message TaskInputsSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.TaskInputsSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this TaskInputsSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace TaskInputsSpec {

        /** Properties of an InputArtifactSpec. */
        interface IInputArtifactSpec {

            /** InputArtifactSpec taskOutputArtifact */
            taskOutputArtifact?: (ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec|null);

            /** InputArtifactSpec componentInputArtifact */
            componentInputArtifact?: (string|null);
        }

        /** Represents an InputArtifactSpec. */
        class InputArtifactSpec implements IInputArtifactSpec {

            /**
             * Constructs a new InputArtifactSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.TaskInputsSpec.IInputArtifactSpec);

            /** InputArtifactSpec taskOutputArtifact. */
            public taskOutputArtifact?: (ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec|null);

            /** InputArtifactSpec componentInputArtifact. */
            public componentInputArtifact?: (string|null);

            /** InputArtifactSpec kind. */
            public kind?: ("taskOutputArtifact"|"componentInputArtifact");

            /**
             * Creates a new InputArtifactSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns InputArtifactSpec instance
             */
            public static create(properties?: ml_pipelines.TaskInputsSpec.IInputArtifactSpec): ml_pipelines.TaskInputsSpec.InputArtifactSpec;

            /**
             * Encodes the specified InputArtifactSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputArtifactSpec.verify|verify} messages.
             * @param message InputArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.TaskInputsSpec.IInputArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified InputArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputArtifactSpec.verify|verify} messages.
             * @param message InputArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.TaskInputsSpec.IInputArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an InputArtifactSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns InputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.TaskInputsSpec.InputArtifactSpec;

            /**
             * Decodes an InputArtifactSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns InputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.TaskInputsSpec.InputArtifactSpec;

            /**
             * Verifies an InputArtifactSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an InputArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns InputArtifactSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.TaskInputsSpec.InputArtifactSpec;

            /**
             * Creates a plain object from an InputArtifactSpec message. Also converts values to other types if specified.
             * @param message InputArtifactSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.TaskInputsSpec.InputArtifactSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this InputArtifactSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        namespace InputArtifactSpec {

            /** Properties of a TaskOutputArtifactSpec. */
            interface ITaskOutputArtifactSpec {

                /** TaskOutputArtifactSpec producerTask */
                producerTask?: (string|null);

                /** TaskOutputArtifactSpec outputArtifactKey */
                outputArtifactKey?: (string|null);
            }

            /** Represents a TaskOutputArtifactSpec. */
            class TaskOutputArtifactSpec implements ITaskOutputArtifactSpec {

                /**
                 * Constructs a new TaskOutputArtifactSpec.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec);

                /** TaskOutputArtifactSpec producerTask. */
                public producerTask: string;

                /** TaskOutputArtifactSpec outputArtifactKey. */
                public outputArtifactKey: string;

                /**
                 * Creates a new TaskOutputArtifactSpec instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns TaskOutputArtifactSpec instance
                 */
                public static create(properties?: ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec): ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec;

                /**
                 * Encodes the specified TaskOutputArtifactSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.verify|verify} messages.
                 * @param message TaskOutputArtifactSpec message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified TaskOutputArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec.verify|verify} messages.
                 * @param message TaskOutputArtifactSpec message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: ml_pipelines.TaskInputsSpec.InputArtifactSpec.ITaskOutputArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a TaskOutputArtifactSpec message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns TaskOutputArtifactSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec;

                /**
                 * Decodes a TaskOutputArtifactSpec message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns TaskOutputArtifactSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec;

                /**
                 * Verifies a TaskOutputArtifactSpec message.
                 * @param message Plain object to verify
                 * @returns `null` if valid, otherwise the reason why it is not
                 */
                public static verify(message: { [k: string]: any }): (string|null);

                /**
                 * Creates a TaskOutputArtifactSpec message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns TaskOutputArtifactSpec
                 */
                public static fromObject(object: { [k: string]: any }): ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec;

                /**
                 * Creates a plain object from a TaskOutputArtifactSpec message. Also converts values to other types if specified.
                 * @param message TaskOutputArtifactSpec
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: ml_pipelines.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this TaskOutputArtifactSpec to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };
            }
        }

        /** Properties of an InputParameterSpec. */
        interface IInputParameterSpec {

            /** InputParameterSpec taskOutputParameter */
            taskOutputParameter?: (ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec|null);

            /** InputParameterSpec runtimeValue */
            runtimeValue?: (ml_pipelines.IValueOrRuntimeParameter|null);

            /** InputParameterSpec componentInputParameter */
            componentInputParameter?: (string|null);

            /** InputParameterSpec taskFinalStatus */
            taskFinalStatus?: (ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus|null);

            /** InputParameterSpec parameterExpressionSelector */
            parameterExpressionSelector?: (string|null);
        }

        /** Represents an InputParameterSpec. */
        class InputParameterSpec implements IInputParameterSpec {

            /**
             * Constructs a new InputParameterSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.TaskInputsSpec.IInputParameterSpec);

            /** InputParameterSpec taskOutputParameter. */
            public taskOutputParameter?: (ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec|null);

            /** InputParameterSpec runtimeValue. */
            public runtimeValue?: (ml_pipelines.IValueOrRuntimeParameter|null);

            /** InputParameterSpec componentInputParameter. */
            public componentInputParameter?: (string|null);

            /** InputParameterSpec taskFinalStatus. */
            public taskFinalStatus?: (ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus|null);

            /** InputParameterSpec parameterExpressionSelector. */
            public parameterExpressionSelector: string;

            /** InputParameterSpec kind. */
            public kind?: ("taskOutputParameter"|"runtimeValue"|"componentInputParameter"|"taskFinalStatus");

            /**
             * Creates a new InputParameterSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns InputParameterSpec instance
             */
            public static create(properties?: ml_pipelines.TaskInputsSpec.IInputParameterSpec): ml_pipelines.TaskInputsSpec.InputParameterSpec;

            /**
             * Encodes the specified InputParameterSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.verify|verify} messages.
             * @param message InputParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.TaskInputsSpec.IInputParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified InputParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.verify|verify} messages.
             * @param message InputParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.TaskInputsSpec.IInputParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an InputParameterSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns InputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.TaskInputsSpec.InputParameterSpec;

            /**
             * Decodes an InputParameterSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns InputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.TaskInputsSpec.InputParameterSpec;

            /**
             * Verifies an InputParameterSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an InputParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns InputParameterSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.TaskInputsSpec.InputParameterSpec;

            /**
             * Creates a plain object from an InputParameterSpec message. Also converts values to other types if specified.
             * @param message InputParameterSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.TaskInputsSpec.InputParameterSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this InputParameterSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        namespace InputParameterSpec {

            /** Properties of a TaskOutputParameterSpec. */
            interface ITaskOutputParameterSpec {

                /** TaskOutputParameterSpec producerTask */
                producerTask?: (string|null);

                /** TaskOutputParameterSpec outputParameterKey */
                outputParameterKey?: (string|null);
            }

            /** Represents a TaskOutputParameterSpec. */
            class TaskOutputParameterSpec implements ITaskOutputParameterSpec {

                /**
                 * Constructs a new TaskOutputParameterSpec.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec);

                /** TaskOutputParameterSpec producerTask. */
                public producerTask: string;

                /** TaskOutputParameterSpec outputParameterKey. */
                public outputParameterKey: string;

                /**
                 * Creates a new TaskOutputParameterSpec instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns TaskOutputParameterSpec instance
                 */
                public static create(properties?: ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec): ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec;

                /**
                 * Encodes the specified TaskOutputParameterSpec message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.verify|verify} messages.
                 * @param message TaskOutputParameterSpec message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified TaskOutputParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec.verify|verify} messages.
                 * @param message TaskOutputParameterSpec message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskOutputParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a TaskOutputParameterSpec message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns TaskOutputParameterSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec;

                /**
                 * Decodes a TaskOutputParameterSpec message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns TaskOutputParameterSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec;

                /**
                 * Verifies a TaskOutputParameterSpec message.
                 * @param message Plain object to verify
                 * @returns `null` if valid, otherwise the reason why it is not
                 */
                public static verify(message: { [k: string]: any }): (string|null);

                /**
                 * Creates a TaskOutputParameterSpec message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns TaskOutputParameterSpec
                 */
                public static fromObject(object: { [k: string]: any }): ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec;

                /**
                 * Creates a plain object from a TaskOutputParameterSpec message. Also converts values to other types if specified.
                 * @param message TaskOutputParameterSpec
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this TaskOutputParameterSpec to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };
            }

            /** Properties of a TaskFinalStatus. */
            interface ITaskFinalStatus {

                /** TaskFinalStatus producerTask */
                producerTask?: (string|null);
            }

            /** Represents a TaskFinalStatus. */
            class TaskFinalStatus implements ITaskFinalStatus {

                /**
                 * Constructs a new TaskFinalStatus.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus);

                /** TaskFinalStatus producerTask. */
                public producerTask: string;

                /**
                 * Creates a new TaskFinalStatus instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns TaskFinalStatus instance
                 */
                public static create(properties?: ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus): ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus;

                /**
                 * Encodes the specified TaskFinalStatus message. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.verify|verify} messages.
                 * @param message TaskFinalStatus message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified TaskFinalStatus message, length delimited. Does not implicitly {@link ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus.verify|verify} messages.
                 * @param message TaskFinalStatus message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: ml_pipelines.TaskInputsSpec.InputParameterSpec.ITaskFinalStatus, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a TaskFinalStatus message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns TaskFinalStatus
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus;

                /**
                 * Decodes a TaskFinalStatus message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns TaskFinalStatus
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus;

                /**
                 * Verifies a TaskFinalStatus message.
                 * @param message Plain object to verify
                 * @returns `null` if valid, otherwise the reason why it is not
                 */
                public static verify(message: { [k: string]: any }): (string|null);

                /**
                 * Creates a TaskFinalStatus message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns TaskFinalStatus
                 */
                public static fromObject(object: { [k: string]: any }): ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus;

                /**
                 * Creates a plain object from a TaskFinalStatus message. Also converts values to other types if specified.
                 * @param message TaskFinalStatus
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: ml_pipelines.TaskInputsSpec.InputParameterSpec.TaskFinalStatus, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this TaskFinalStatus to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };
            }
        }
    }

    /** Properties of a TaskOutputsSpec. */
    interface ITaskOutputsSpec {

        /** TaskOutputsSpec parameters */
        parameters?: ({ [k: string]: ml_pipelines.TaskOutputsSpec.IOutputParameterSpec }|null);

        /** TaskOutputsSpec artifacts */
        artifacts?: ({ [k: string]: ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec }|null);
    }

    /** Represents a TaskOutputsSpec. */
    class TaskOutputsSpec implements ITaskOutputsSpec {

        /**
         * Constructs a new TaskOutputsSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.ITaskOutputsSpec);

        /** TaskOutputsSpec parameters. */
        public parameters: { [k: string]: ml_pipelines.TaskOutputsSpec.IOutputParameterSpec };

        /** TaskOutputsSpec artifacts. */
        public artifacts: { [k: string]: ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec };

        /**
         * Creates a new TaskOutputsSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns TaskOutputsSpec instance
         */
        public static create(properties?: ml_pipelines.ITaskOutputsSpec): ml_pipelines.TaskOutputsSpec;

        /**
         * Encodes the specified TaskOutputsSpec message. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.verify|verify} messages.
         * @param message TaskOutputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.ITaskOutputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified TaskOutputsSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.verify|verify} messages.
         * @param message TaskOutputsSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.ITaskOutputsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a TaskOutputsSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns TaskOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.TaskOutputsSpec;

        /**
         * Decodes a TaskOutputsSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns TaskOutputsSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.TaskOutputsSpec;

        /**
         * Verifies a TaskOutputsSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a TaskOutputsSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns TaskOutputsSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.TaskOutputsSpec;

        /**
         * Creates a plain object from a TaskOutputsSpec message. Also converts values to other types if specified.
         * @param message TaskOutputsSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.TaskOutputsSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this TaskOutputsSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace TaskOutputsSpec {

        /** Properties of an OutputArtifactSpec. */
        interface IOutputArtifactSpec {

            /** OutputArtifactSpec artifactType */
            artifactType?: (ml_pipelines.IArtifactTypeSchema|null);

            /** OutputArtifactSpec properties */
            properties?: ({ [k: string]: ml_pipelines.IValueOrRuntimeParameter }|null);

            /** OutputArtifactSpec customProperties */
            customProperties?: ({ [k: string]: ml_pipelines.IValueOrRuntimeParameter }|null);
        }

        /** Represents an OutputArtifactSpec. */
        class OutputArtifactSpec implements IOutputArtifactSpec {

            /**
             * Constructs a new OutputArtifactSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec);

            /** OutputArtifactSpec artifactType. */
            public artifactType?: (ml_pipelines.IArtifactTypeSchema|null);

            /** OutputArtifactSpec properties. */
            public properties: { [k: string]: ml_pipelines.IValueOrRuntimeParameter };

            /** OutputArtifactSpec customProperties. */
            public customProperties: { [k: string]: ml_pipelines.IValueOrRuntimeParameter };

            /**
             * Creates a new OutputArtifactSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns OutputArtifactSpec instance
             */
            public static create(properties?: ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec): ml_pipelines.TaskOutputsSpec.OutputArtifactSpec;

            /**
             * Encodes the specified OutputArtifactSpec message. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.verify|verify} messages.
             * @param message OutputArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified OutputArtifactSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.OutputArtifactSpec.verify|verify} messages.
             * @param message OutputArtifactSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.TaskOutputsSpec.IOutputArtifactSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an OutputArtifactSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns OutputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.TaskOutputsSpec.OutputArtifactSpec;

            /**
             * Decodes an OutputArtifactSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns OutputArtifactSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.TaskOutputsSpec.OutputArtifactSpec;

            /**
             * Verifies an OutputArtifactSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an OutputArtifactSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns OutputArtifactSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.TaskOutputsSpec.OutputArtifactSpec;

            /**
             * Creates a plain object from an OutputArtifactSpec message. Also converts values to other types if specified.
             * @param message OutputArtifactSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.TaskOutputsSpec.OutputArtifactSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this OutputArtifactSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of an OutputParameterSpec. */
        interface IOutputParameterSpec {

            /** OutputParameterSpec type */
            type?: (ml_pipelines.PrimitiveType.PrimitiveTypeEnum|null);
        }

        /** Represents an OutputParameterSpec. */
        class OutputParameterSpec implements IOutputParameterSpec {

            /**
             * Constructs a new OutputParameterSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.TaskOutputsSpec.IOutputParameterSpec);

            /** OutputParameterSpec type. */
            public type: ml_pipelines.PrimitiveType.PrimitiveTypeEnum;

            /**
             * Creates a new OutputParameterSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns OutputParameterSpec instance
             */
            public static create(properties?: ml_pipelines.TaskOutputsSpec.IOutputParameterSpec): ml_pipelines.TaskOutputsSpec.OutputParameterSpec;

            /**
             * Encodes the specified OutputParameterSpec message. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.OutputParameterSpec.verify|verify} messages.
             * @param message OutputParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.TaskOutputsSpec.IOutputParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified OutputParameterSpec message, length delimited. Does not implicitly {@link ml_pipelines.TaskOutputsSpec.OutputParameterSpec.verify|verify} messages.
             * @param message OutputParameterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.TaskOutputsSpec.IOutputParameterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an OutputParameterSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns OutputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.TaskOutputsSpec.OutputParameterSpec;

            /**
             * Decodes an OutputParameterSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns OutputParameterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.TaskOutputsSpec.OutputParameterSpec;

            /**
             * Verifies an OutputParameterSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an OutputParameterSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns OutputParameterSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.TaskOutputsSpec.OutputParameterSpec;

            /**
             * Creates a plain object from an OutputParameterSpec message. Also converts values to other types if specified.
             * @param message OutputParameterSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.TaskOutputsSpec.OutputParameterSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this OutputParameterSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of a PrimitiveType. */
    interface IPrimitiveType {
    }

    /** Represents a PrimitiveType. */
    class PrimitiveType implements IPrimitiveType {

        /**
         * Constructs a new PrimitiveType.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IPrimitiveType);

        /**
         * Creates a new PrimitiveType instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PrimitiveType instance
         */
        public static create(properties?: ml_pipelines.IPrimitiveType): ml_pipelines.PrimitiveType;

        /**
         * Encodes the specified PrimitiveType message. Does not implicitly {@link ml_pipelines.PrimitiveType.verify|verify} messages.
         * @param message PrimitiveType message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IPrimitiveType, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PrimitiveType message, length delimited. Does not implicitly {@link ml_pipelines.PrimitiveType.verify|verify} messages.
         * @param message PrimitiveType message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IPrimitiveType, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PrimitiveType message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns PrimitiveType
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PrimitiveType;

        /**
         * Decodes a PrimitiveType message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns PrimitiveType
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PrimitiveType;

        /**
         * Verifies a PrimitiveType message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PrimitiveType message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PrimitiveType
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.PrimitiveType;

        /**
         * Creates a plain object from a PrimitiveType message. Also converts values to other types if specified.
         * @param message PrimitiveType
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.PrimitiveType, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PrimitiveType to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace PrimitiveType {

        /** PrimitiveTypeEnum enum. */
        enum PrimitiveTypeEnum {
            PRIMITIVE_TYPE_UNSPECIFIED = 0,
            INT = 1,
            DOUBLE = 2,
            STRING = 3
        }
    }

    /** Properties of a PipelineTaskSpec. */
    interface IPipelineTaskSpec {

        /** PipelineTaskSpec taskInfo */
        taskInfo?: (ml_pipelines.IPipelineTaskInfo|null);

        /** PipelineTaskSpec inputs */
        inputs?: (ml_pipelines.ITaskInputsSpec|null);

        /** PipelineTaskSpec dependentTasks */
        dependentTasks?: (string[]|null);

        /** PipelineTaskSpec cachingOptions */
        cachingOptions?: (ml_pipelines.PipelineTaskSpec.ICachingOptions|null);

        /** PipelineTaskSpec componentRef */
        componentRef?: (ml_pipelines.IComponentRef|null);

        /** PipelineTaskSpec triggerPolicy */
        triggerPolicy?: (ml_pipelines.PipelineTaskSpec.ITriggerPolicy|null);

        /** PipelineTaskSpec artifactIterator */
        artifactIterator?: (ml_pipelines.IArtifactIteratorSpec|null);

        /** PipelineTaskSpec parameterIterator */
        parameterIterator?: (ml_pipelines.IParameterIteratorSpec|null);
    }

    /** Represents a PipelineTaskSpec. */
    class PipelineTaskSpec implements IPipelineTaskSpec {

        /**
         * Constructs a new PipelineTaskSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IPipelineTaskSpec);

        /** PipelineTaskSpec taskInfo. */
        public taskInfo?: (ml_pipelines.IPipelineTaskInfo|null);

        /** PipelineTaskSpec inputs. */
        public inputs?: (ml_pipelines.ITaskInputsSpec|null);

        /** PipelineTaskSpec dependentTasks. */
        public dependentTasks: string[];

        /** PipelineTaskSpec cachingOptions. */
        public cachingOptions?: (ml_pipelines.PipelineTaskSpec.ICachingOptions|null);

        /** PipelineTaskSpec componentRef. */
        public componentRef?: (ml_pipelines.IComponentRef|null);

        /** PipelineTaskSpec triggerPolicy. */
        public triggerPolicy?: (ml_pipelines.PipelineTaskSpec.ITriggerPolicy|null);

        /** PipelineTaskSpec artifactIterator. */
        public artifactIterator?: (ml_pipelines.IArtifactIteratorSpec|null);

        /** PipelineTaskSpec parameterIterator. */
        public parameterIterator?: (ml_pipelines.IParameterIteratorSpec|null);

        /** PipelineTaskSpec iterator. */
        public iterator?: ("artifactIterator"|"parameterIterator");

        /**
         * Creates a new PipelineTaskSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PipelineTaskSpec instance
         */
        public static create(properties?: ml_pipelines.IPipelineTaskSpec): ml_pipelines.PipelineTaskSpec;

        /**
         * Encodes the specified PipelineTaskSpec message. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.verify|verify} messages.
         * @param message PipelineTaskSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IPipelineTaskSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PipelineTaskSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.verify|verify} messages.
         * @param message PipelineTaskSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IPipelineTaskSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PipelineTaskSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns PipelineTaskSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineTaskSpec;

        /**
         * Decodes a PipelineTaskSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns PipelineTaskSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineTaskSpec;

        /**
         * Verifies a PipelineTaskSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PipelineTaskSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PipelineTaskSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineTaskSpec;

        /**
         * Creates a plain object from a PipelineTaskSpec message. Also converts values to other types if specified.
         * @param message PipelineTaskSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.PipelineTaskSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PipelineTaskSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace PipelineTaskSpec {

        /** Properties of a CachingOptions. */
        interface ICachingOptions {

            /** CachingOptions enableCache */
            enableCache?: (boolean|null);
        }

        /** Represents a CachingOptions. */
        class CachingOptions implements ICachingOptions {

            /**
             * Constructs a new CachingOptions.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.PipelineTaskSpec.ICachingOptions);

            /** CachingOptions enableCache. */
            public enableCache: boolean;

            /**
             * Creates a new CachingOptions instance using the specified properties.
             * @param [properties] Properties to set
             * @returns CachingOptions instance
             */
            public static create(properties?: ml_pipelines.PipelineTaskSpec.ICachingOptions): ml_pipelines.PipelineTaskSpec.CachingOptions;

            /**
             * Encodes the specified CachingOptions message. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.CachingOptions.verify|verify} messages.
             * @param message CachingOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.PipelineTaskSpec.ICachingOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified CachingOptions message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.CachingOptions.verify|verify} messages.
             * @param message CachingOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.PipelineTaskSpec.ICachingOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a CachingOptions message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns CachingOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineTaskSpec.CachingOptions;

            /**
             * Decodes a CachingOptions message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns CachingOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineTaskSpec.CachingOptions;

            /**
             * Verifies a CachingOptions message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a CachingOptions message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns CachingOptions
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineTaskSpec.CachingOptions;

            /**
             * Creates a plain object from a CachingOptions message. Also converts values to other types if specified.
             * @param message CachingOptions
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.PipelineTaskSpec.CachingOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this CachingOptions to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a TriggerPolicy. */
        interface ITriggerPolicy {

            /** TriggerPolicy condition */
            condition?: (string|null);

            /** TriggerPolicy strategy */
            strategy?: (ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy|null);
        }

        /** Represents a TriggerPolicy. */
        class TriggerPolicy implements ITriggerPolicy {

            /**
             * Constructs a new TriggerPolicy.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.PipelineTaskSpec.ITriggerPolicy);

            /** TriggerPolicy condition. */
            public condition: string;

            /** TriggerPolicy strategy. */
            public strategy: ml_pipelines.PipelineTaskSpec.TriggerPolicy.TriggerStrategy;

            /**
             * Creates a new TriggerPolicy instance using the specified properties.
             * @param [properties] Properties to set
             * @returns TriggerPolicy instance
             */
            public static create(properties?: ml_pipelines.PipelineTaskSpec.ITriggerPolicy): ml_pipelines.PipelineTaskSpec.TriggerPolicy;

            /**
             * Encodes the specified TriggerPolicy message. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.TriggerPolicy.verify|verify} messages.
             * @param message TriggerPolicy message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.PipelineTaskSpec.ITriggerPolicy, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified TriggerPolicy message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskSpec.TriggerPolicy.verify|verify} messages.
             * @param message TriggerPolicy message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.PipelineTaskSpec.ITriggerPolicy, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a TriggerPolicy message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns TriggerPolicy
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineTaskSpec.TriggerPolicy;

            /**
             * Decodes a TriggerPolicy message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns TriggerPolicy
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineTaskSpec.TriggerPolicy;

            /**
             * Verifies a TriggerPolicy message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a TriggerPolicy message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns TriggerPolicy
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineTaskSpec.TriggerPolicy;

            /**
             * Creates a plain object from a TriggerPolicy message. Also converts values to other types if specified.
             * @param message TriggerPolicy
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.PipelineTaskSpec.TriggerPolicy, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this TriggerPolicy to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        namespace TriggerPolicy {

            /** TriggerStrategy enum. */
            enum TriggerStrategy {
                TRIGGER_STRATEGY_UNSPECIFIED = 0,
                ALL_UPSTREAM_TASKS_SUCCEEDED = 1,
                ALL_UPSTREAM_TASKS_COMPLETED = 2
            }
        }
    }

    /** Properties of an ArtifactIteratorSpec. */
    interface IArtifactIteratorSpec {

        /** ArtifactIteratorSpec items */
        items?: (ml_pipelines.ArtifactIteratorSpec.IItemsSpec|null);

        /** ArtifactIteratorSpec itemInput */
        itemInput?: (string|null);
    }

    /** Represents an ArtifactIteratorSpec. */
    class ArtifactIteratorSpec implements IArtifactIteratorSpec {

        /**
         * Constructs a new ArtifactIteratorSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IArtifactIteratorSpec);

        /** ArtifactIteratorSpec items. */
        public items?: (ml_pipelines.ArtifactIteratorSpec.IItemsSpec|null);

        /** ArtifactIteratorSpec itemInput. */
        public itemInput: string;

        /**
         * Creates a new ArtifactIteratorSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ArtifactIteratorSpec instance
         */
        public static create(properties?: ml_pipelines.IArtifactIteratorSpec): ml_pipelines.ArtifactIteratorSpec;

        /**
         * Encodes the specified ArtifactIteratorSpec message. Does not implicitly {@link ml_pipelines.ArtifactIteratorSpec.verify|verify} messages.
         * @param message ArtifactIteratorSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IArtifactIteratorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ArtifactIteratorSpec message, length delimited. Does not implicitly {@link ml_pipelines.ArtifactIteratorSpec.verify|verify} messages.
         * @param message ArtifactIteratorSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IArtifactIteratorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an ArtifactIteratorSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ArtifactIteratorSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ArtifactIteratorSpec;

        /**
         * Decodes an ArtifactIteratorSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ArtifactIteratorSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ArtifactIteratorSpec;

        /**
         * Verifies an ArtifactIteratorSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an ArtifactIteratorSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ArtifactIteratorSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ArtifactIteratorSpec;

        /**
         * Creates a plain object from an ArtifactIteratorSpec message. Also converts values to other types if specified.
         * @param message ArtifactIteratorSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ArtifactIteratorSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ArtifactIteratorSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace ArtifactIteratorSpec {

        /** Properties of an ItemsSpec. */
        interface IItemsSpec {

            /** ItemsSpec inputArtifact */
            inputArtifact?: (string|null);
        }

        /** Represents an ItemsSpec. */
        class ItemsSpec implements IItemsSpec {

            /**
             * Constructs a new ItemsSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.ArtifactIteratorSpec.IItemsSpec);

            /** ItemsSpec inputArtifact. */
            public inputArtifact: string;

            /**
             * Creates a new ItemsSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ItemsSpec instance
             */
            public static create(properties?: ml_pipelines.ArtifactIteratorSpec.IItemsSpec): ml_pipelines.ArtifactIteratorSpec.ItemsSpec;

            /**
             * Encodes the specified ItemsSpec message. Does not implicitly {@link ml_pipelines.ArtifactIteratorSpec.ItemsSpec.verify|verify} messages.
             * @param message ItemsSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.ArtifactIteratorSpec.IItemsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ItemsSpec message, length delimited. Does not implicitly {@link ml_pipelines.ArtifactIteratorSpec.ItemsSpec.verify|verify} messages.
             * @param message ItemsSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.ArtifactIteratorSpec.IItemsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an ItemsSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ItemsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ArtifactIteratorSpec.ItemsSpec;

            /**
             * Decodes an ItemsSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ItemsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ArtifactIteratorSpec.ItemsSpec;

            /**
             * Verifies an ItemsSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an ItemsSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ItemsSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.ArtifactIteratorSpec.ItemsSpec;

            /**
             * Creates a plain object from an ItemsSpec message. Also converts values to other types if specified.
             * @param message ItemsSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.ArtifactIteratorSpec.ItemsSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ItemsSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of a ParameterIteratorSpec. */
    interface IParameterIteratorSpec {

        /** ParameterIteratorSpec items */
        items?: (ml_pipelines.ParameterIteratorSpec.IItemsSpec|null);

        /** ParameterIteratorSpec itemInput */
        itemInput?: (string|null);
    }

    /** Represents a ParameterIteratorSpec. */
    class ParameterIteratorSpec implements IParameterIteratorSpec {

        /**
         * Constructs a new ParameterIteratorSpec.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IParameterIteratorSpec);

        /** ParameterIteratorSpec items. */
        public items?: (ml_pipelines.ParameterIteratorSpec.IItemsSpec|null);

        /** ParameterIteratorSpec itemInput. */
        public itemInput: string;

        /**
         * Creates a new ParameterIteratorSpec instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ParameterIteratorSpec instance
         */
        public static create(properties?: ml_pipelines.IParameterIteratorSpec): ml_pipelines.ParameterIteratorSpec;

        /**
         * Encodes the specified ParameterIteratorSpec message. Does not implicitly {@link ml_pipelines.ParameterIteratorSpec.verify|verify} messages.
         * @param message ParameterIteratorSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IParameterIteratorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ParameterIteratorSpec message, length delimited. Does not implicitly {@link ml_pipelines.ParameterIteratorSpec.verify|verify} messages.
         * @param message ParameterIteratorSpec message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IParameterIteratorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ParameterIteratorSpec message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ParameterIteratorSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ParameterIteratorSpec;

        /**
         * Decodes a ParameterIteratorSpec message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ParameterIteratorSpec
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ParameterIteratorSpec;

        /**
         * Verifies a ParameterIteratorSpec message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ParameterIteratorSpec message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ParameterIteratorSpec
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ParameterIteratorSpec;

        /**
         * Creates a plain object from a ParameterIteratorSpec message. Also converts values to other types if specified.
         * @param message ParameterIteratorSpec
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ParameterIteratorSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ParameterIteratorSpec to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace ParameterIteratorSpec {

        /** Properties of an ItemsSpec. */
        interface IItemsSpec {

            /** ItemsSpec raw */
            raw?: (string|null);

            /** ItemsSpec inputParameter */
            inputParameter?: (string|null);
        }

        /** Represents an ItemsSpec. */
        class ItemsSpec implements IItemsSpec {

            /**
             * Constructs a new ItemsSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.ParameterIteratorSpec.IItemsSpec);

            /** ItemsSpec raw. */
            public raw?: (string|null);

            /** ItemsSpec inputParameter. */
            public inputParameter?: (string|null);

            /** ItemsSpec kind. */
            public kind?: ("raw"|"inputParameter");

            /**
             * Creates a new ItemsSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ItemsSpec instance
             */
            public static create(properties?: ml_pipelines.ParameterIteratorSpec.IItemsSpec): ml_pipelines.ParameterIteratorSpec.ItemsSpec;

            /**
             * Encodes the specified ItemsSpec message. Does not implicitly {@link ml_pipelines.ParameterIteratorSpec.ItemsSpec.verify|verify} messages.
             * @param message ItemsSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.ParameterIteratorSpec.IItemsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ItemsSpec message, length delimited. Does not implicitly {@link ml_pipelines.ParameterIteratorSpec.ItemsSpec.verify|verify} messages.
             * @param message ItemsSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.ParameterIteratorSpec.IItemsSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an ItemsSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ItemsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ParameterIteratorSpec.ItemsSpec;

            /**
             * Decodes an ItemsSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ItemsSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ParameterIteratorSpec.ItemsSpec;

            /**
             * Verifies an ItemsSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an ItemsSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ItemsSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.ParameterIteratorSpec.ItemsSpec;

            /**
             * Creates a plain object from an ItemsSpec message. Also converts values to other types if specified.
             * @param message ItemsSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.ParameterIteratorSpec.ItemsSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ItemsSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of a ComponentRef. */
    interface IComponentRef {

        /** ComponentRef name */
        name?: (string|null);
    }

    /** Represents a ComponentRef. */
    class ComponentRef implements IComponentRef {

        /**
         * Constructs a new ComponentRef.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IComponentRef);

        /** ComponentRef name. */
        public name: string;

        /**
         * Creates a new ComponentRef instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ComponentRef instance
         */
        public static create(properties?: ml_pipelines.IComponentRef): ml_pipelines.ComponentRef;

        /**
         * Encodes the specified ComponentRef message. Does not implicitly {@link ml_pipelines.ComponentRef.verify|verify} messages.
         * @param message ComponentRef message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IComponentRef, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ComponentRef message, length delimited. Does not implicitly {@link ml_pipelines.ComponentRef.verify|verify} messages.
         * @param message ComponentRef message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IComponentRef, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ComponentRef message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ComponentRef
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ComponentRef;

        /**
         * Decodes a ComponentRef message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ComponentRef
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ComponentRef;

        /**
         * Verifies a ComponentRef message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ComponentRef message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ComponentRef
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ComponentRef;

        /**
         * Creates a plain object from a ComponentRef message. Also converts values to other types if specified.
         * @param message ComponentRef
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ComponentRef, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ComponentRef to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of a PipelineInfo. */
    interface IPipelineInfo {

        /** PipelineInfo name */
        name?: (string|null);
    }

    /** Represents a PipelineInfo. */
    class PipelineInfo implements IPipelineInfo {

        /**
         * Constructs a new PipelineInfo.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IPipelineInfo);

        /** PipelineInfo name. */
        public name: string;

        /**
         * Creates a new PipelineInfo instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PipelineInfo instance
         */
        public static create(properties?: ml_pipelines.IPipelineInfo): ml_pipelines.PipelineInfo;

        /**
         * Encodes the specified PipelineInfo message. Does not implicitly {@link ml_pipelines.PipelineInfo.verify|verify} messages.
         * @param message PipelineInfo message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IPipelineInfo, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PipelineInfo message, length delimited. Does not implicitly {@link ml_pipelines.PipelineInfo.verify|verify} messages.
         * @param message PipelineInfo message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IPipelineInfo, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PipelineInfo message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns PipelineInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineInfo;

        /**
         * Decodes a PipelineInfo message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns PipelineInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineInfo;

        /**
         * Verifies a PipelineInfo message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PipelineInfo message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PipelineInfo
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineInfo;

        /**
         * Creates a plain object from a PipelineInfo message. Also converts values to other types if specified.
         * @param message PipelineInfo
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.PipelineInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PipelineInfo to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of an ArtifactTypeSchema. */
    interface IArtifactTypeSchema {

        /** ArtifactTypeSchema schemaTitle */
        schemaTitle?: (string|null);

        /** ArtifactTypeSchema schemaUri */
        schemaUri?: (string|null);

        /** ArtifactTypeSchema instanceSchema */
        instanceSchema?: (string|null);

        /** ArtifactTypeSchema schemaVersion */
        schemaVersion?: (string|null);
    }

    /** Represents an ArtifactTypeSchema. */
    class ArtifactTypeSchema implements IArtifactTypeSchema {

        /**
         * Constructs a new ArtifactTypeSchema.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IArtifactTypeSchema);

        /** ArtifactTypeSchema schemaTitle. */
        public schemaTitle?: (string|null);

        /** ArtifactTypeSchema schemaUri. */
        public schemaUri?: (string|null);

        /** ArtifactTypeSchema instanceSchema. */
        public instanceSchema?: (string|null);

        /** ArtifactTypeSchema schemaVersion. */
        public schemaVersion: string;

        /** ArtifactTypeSchema kind. */
        public kind?: ("schemaTitle"|"schemaUri"|"instanceSchema");

        /**
         * Creates a new ArtifactTypeSchema instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ArtifactTypeSchema instance
         */
        public static create(properties?: ml_pipelines.IArtifactTypeSchema): ml_pipelines.ArtifactTypeSchema;

        /**
         * Encodes the specified ArtifactTypeSchema message. Does not implicitly {@link ml_pipelines.ArtifactTypeSchema.verify|verify} messages.
         * @param message ArtifactTypeSchema message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IArtifactTypeSchema, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ArtifactTypeSchema message, length delimited. Does not implicitly {@link ml_pipelines.ArtifactTypeSchema.verify|verify} messages.
         * @param message ArtifactTypeSchema message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IArtifactTypeSchema, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an ArtifactTypeSchema message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ArtifactTypeSchema
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ArtifactTypeSchema;

        /**
         * Decodes an ArtifactTypeSchema message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ArtifactTypeSchema
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ArtifactTypeSchema;

        /**
         * Verifies an ArtifactTypeSchema message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an ArtifactTypeSchema message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ArtifactTypeSchema
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ArtifactTypeSchema;

        /**
         * Creates a plain object from an ArtifactTypeSchema message. Also converts values to other types if specified.
         * @param message ArtifactTypeSchema
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ArtifactTypeSchema, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ArtifactTypeSchema to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of a PipelineTaskInfo. */
    interface IPipelineTaskInfo {

        /** PipelineTaskInfo name */
        name?: (string|null);
    }

    /** Represents a PipelineTaskInfo. */
    class PipelineTaskInfo implements IPipelineTaskInfo {

        /**
         * Constructs a new PipelineTaskInfo.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IPipelineTaskInfo);

        /** PipelineTaskInfo name. */
        public name: string;

        /**
         * Creates a new PipelineTaskInfo instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PipelineTaskInfo instance
         */
        public static create(properties?: ml_pipelines.IPipelineTaskInfo): ml_pipelines.PipelineTaskInfo;

        /**
         * Encodes the specified PipelineTaskInfo message. Does not implicitly {@link ml_pipelines.PipelineTaskInfo.verify|verify} messages.
         * @param message PipelineTaskInfo message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IPipelineTaskInfo, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PipelineTaskInfo message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskInfo.verify|verify} messages.
         * @param message PipelineTaskInfo message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IPipelineTaskInfo, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PipelineTaskInfo message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns PipelineTaskInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineTaskInfo;

        /**
         * Decodes a PipelineTaskInfo message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns PipelineTaskInfo
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineTaskInfo;

        /**
         * Verifies a PipelineTaskInfo message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PipelineTaskInfo message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PipelineTaskInfo
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineTaskInfo;

        /**
         * Creates a plain object from a PipelineTaskInfo message. Also converts values to other types if specified.
         * @param message PipelineTaskInfo
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.PipelineTaskInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PipelineTaskInfo to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of a ValueOrRuntimeParameter. */
    interface IValueOrRuntimeParameter {

        /** ValueOrRuntimeParameter constantValue */
        constantValue?: (ml_pipelines.IValue|null);

        /** ValueOrRuntimeParameter runtimeParameter */
        runtimeParameter?: (string|null);
    }

    /** Represents a ValueOrRuntimeParameter. */
    class ValueOrRuntimeParameter implements IValueOrRuntimeParameter {

        /**
         * Constructs a new ValueOrRuntimeParameter.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IValueOrRuntimeParameter);

        /** ValueOrRuntimeParameter constantValue. */
        public constantValue?: (ml_pipelines.IValue|null);

        /** ValueOrRuntimeParameter runtimeParameter. */
        public runtimeParameter?: (string|null);

        /** ValueOrRuntimeParameter value. */
        public value?: ("constantValue"|"runtimeParameter");

        /**
         * Creates a new ValueOrRuntimeParameter instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ValueOrRuntimeParameter instance
         */
        public static create(properties?: ml_pipelines.IValueOrRuntimeParameter): ml_pipelines.ValueOrRuntimeParameter;

        /**
         * Encodes the specified ValueOrRuntimeParameter message. Does not implicitly {@link ml_pipelines.ValueOrRuntimeParameter.verify|verify} messages.
         * @param message ValueOrRuntimeParameter message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IValueOrRuntimeParameter, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ValueOrRuntimeParameter message, length delimited. Does not implicitly {@link ml_pipelines.ValueOrRuntimeParameter.verify|verify} messages.
         * @param message ValueOrRuntimeParameter message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IValueOrRuntimeParameter, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a ValueOrRuntimeParameter message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ValueOrRuntimeParameter
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ValueOrRuntimeParameter;

        /**
         * Decodes a ValueOrRuntimeParameter message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ValueOrRuntimeParameter
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ValueOrRuntimeParameter;

        /**
         * Verifies a ValueOrRuntimeParameter message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a ValueOrRuntimeParameter message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ValueOrRuntimeParameter
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ValueOrRuntimeParameter;

        /**
         * Creates a plain object from a ValueOrRuntimeParameter message. Also converts values to other types if specified.
         * @param message ValueOrRuntimeParameter
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ValueOrRuntimeParameter, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ValueOrRuntimeParameter to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of a PipelineDeploymentConfig. */
    interface IPipelineDeploymentConfig {

        /** PipelineDeploymentConfig executors */
        executors?: ({ [k: string]: ml_pipelines.PipelineDeploymentConfig.IExecutorSpec }|null);
    }

    /** Represents a PipelineDeploymentConfig. */
    class PipelineDeploymentConfig implements IPipelineDeploymentConfig {

        /**
         * Constructs a new PipelineDeploymentConfig.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IPipelineDeploymentConfig);

        /** PipelineDeploymentConfig executors. */
        public executors: { [k: string]: ml_pipelines.PipelineDeploymentConfig.IExecutorSpec };

        /**
         * Creates a new PipelineDeploymentConfig instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PipelineDeploymentConfig instance
         */
        public static create(properties?: ml_pipelines.IPipelineDeploymentConfig): ml_pipelines.PipelineDeploymentConfig;

        /**
         * Encodes the specified PipelineDeploymentConfig message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.verify|verify} messages.
         * @param message PipelineDeploymentConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IPipelineDeploymentConfig, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PipelineDeploymentConfig message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.verify|verify} messages.
         * @param message PipelineDeploymentConfig message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IPipelineDeploymentConfig, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PipelineDeploymentConfig message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns PipelineDeploymentConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig;

        /**
         * Decodes a PipelineDeploymentConfig message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns PipelineDeploymentConfig
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig;

        /**
         * Verifies a PipelineDeploymentConfig message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PipelineDeploymentConfig message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PipelineDeploymentConfig
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig;

        /**
         * Creates a plain object from a PipelineDeploymentConfig message. Also converts values to other types if specified.
         * @param message PipelineDeploymentConfig
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.PipelineDeploymentConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PipelineDeploymentConfig to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace PipelineDeploymentConfig {

        /** Properties of a PipelineContainerSpec. */
        interface IPipelineContainerSpec {

            /** PipelineContainerSpec image */
            image?: (string|null);

            /** PipelineContainerSpec command */
            command?: (string[]|null);

            /** PipelineContainerSpec args */
            args?: (string[]|null);

            /** PipelineContainerSpec lifecycle */
            lifecycle?: (ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle|null);

            /** PipelineContainerSpec resources */
            resources?: (ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec|null);
        }

        /** Represents a PipelineContainerSpec. */
        class PipelineContainerSpec implements IPipelineContainerSpec {

            /**
             * Constructs a new PipelineContainerSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec);

            /** PipelineContainerSpec image. */
            public image: string;

            /** PipelineContainerSpec command. */
            public command: string[];

            /** PipelineContainerSpec args. */
            public args: string[];

            /** PipelineContainerSpec lifecycle. */
            public lifecycle?: (ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle|null);

            /** PipelineContainerSpec resources. */
            public resources?: (ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec|null);

            /**
             * Creates a new PipelineContainerSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns PipelineContainerSpec instance
             */
            public static create(properties?: ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec;

            /**
             * Encodes the specified PipelineContainerSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.verify|verify} messages.
             * @param message PipelineContainerSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified PipelineContainerSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.verify|verify} messages.
             * @param message PipelineContainerSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a PipelineContainerSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns PipelineContainerSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec;

            /**
             * Decodes a PipelineContainerSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns PipelineContainerSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec;

            /**
             * Verifies a PipelineContainerSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a PipelineContainerSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns PipelineContainerSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec;

            /**
             * Creates a plain object from a PipelineContainerSpec message. Also converts values to other types if specified.
             * @param message PipelineContainerSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this PipelineContainerSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        namespace PipelineContainerSpec {

            /** Properties of a Lifecycle. */
            interface ILifecycle {

                /** Lifecycle preCacheCheck */
                preCacheCheck?: (ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec|null);
            }

            /** Represents a Lifecycle. */
            class Lifecycle implements ILifecycle {

                /**
                 * Constructs a new Lifecycle.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle);

                /** Lifecycle preCacheCheck. */
                public preCacheCheck?: (ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec|null);

                /**
                 * Creates a new Lifecycle instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns Lifecycle instance
                 */
                public static create(properties?: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle;

                /**
                 * Encodes the specified Lifecycle message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.verify|verify} messages.
                 * @param message Lifecycle message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified Lifecycle message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.verify|verify} messages.
                 * @param message Lifecycle message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ILifecycle, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a Lifecycle message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns Lifecycle
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle;

                /**
                 * Decodes a Lifecycle message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns Lifecycle
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle;

                /**
                 * Verifies a Lifecycle message.
                 * @param message Plain object to verify
                 * @returns `null` if valid, otherwise the reason why it is not
                 */
                public static verify(message: { [k: string]: any }): (string|null);

                /**
                 * Creates a Lifecycle message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns Lifecycle
                 */
                public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle;

                /**
                 * Creates a plain object from a Lifecycle message. Also converts values to other types if specified.
                 * @param message Lifecycle
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this Lifecycle to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };
            }

            namespace Lifecycle {

                /** Properties of an Exec. */
                interface IExec {

                    /** Exec command */
                    command?: (string[]|null);

                    /** Exec args */
                    args?: (string[]|null);
                }

                /** Represents an Exec. */
                class Exec implements IExec {

                    /**
                     * Constructs a new Exec.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec);

                    /** Exec command. */
                    public command: string[];

                    /** Exec args. */
                    public args: string[];

                    /**
                     * Creates a new Exec instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns Exec instance
                     */
                    public static create(properties?: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec;

                    /**
                     * Encodes the specified Exec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.verify|verify} messages.
                     * @param message Exec message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified Exec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec.verify|verify} messages.
                     * @param message Exec message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.IExec, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes an Exec message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns Exec
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec;

                    /**
                     * Decodes an Exec message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns Exec
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec;

                    /**
                     * Verifies an Exec message.
                     * @param message Plain object to verify
                     * @returns `null` if valid, otherwise the reason why it is not
                     */
                    public static verify(message: { [k: string]: any }): (string|null);

                    /**
                     * Creates an Exec message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns Exec
                     */
                    public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec;

                    /**
                     * Creates a plain object from an Exec message. Also converts values to other types if specified.
                     * @param message Exec
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.Lifecycle.Exec, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this Exec to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };
                }
            }

            /** Properties of a ResourceSpec. */
            interface IResourceSpec {

                /** ResourceSpec cpuLimit */
                cpuLimit?: (number|null);

                /** ResourceSpec memoryLimit */
                memoryLimit?: (number|null);

                /** ResourceSpec accelerator */
                accelerator?: (ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig|null);
            }

            /** Represents a ResourceSpec. */
            class ResourceSpec implements IResourceSpec {

                /**
                 * Constructs a new ResourceSpec.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec);

                /** ResourceSpec cpuLimit. */
                public cpuLimit: number;

                /** ResourceSpec memoryLimit. */
                public memoryLimit: number;

                /** ResourceSpec accelerator. */
                public accelerator?: (ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig|null);

                /**
                 * Creates a new ResourceSpec instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ResourceSpec instance
                 */
                public static create(properties?: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec;

                /**
                 * Encodes the specified ResourceSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.verify|verify} messages.
                 * @param message ResourceSpec message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ResourceSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.verify|verify} messages.
                 * @param message ResourceSpec message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.IResourceSpec, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a ResourceSpec message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ResourceSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec;

                /**
                 * Decodes a ResourceSpec message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ResourceSpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec;

                /**
                 * Verifies a ResourceSpec message.
                 * @param message Plain object to verify
                 * @returns `null` if valid, otherwise the reason why it is not
                 */
                public static verify(message: { [k: string]: any }): (string|null);

                /**
                 * Creates a ResourceSpec message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ResourceSpec
                 */
                public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec;

                /**
                 * Creates a plain object from a ResourceSpec message. Also converts values to other types if specified.
                 * @param message ResourceSpec
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ResourceSpec to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };
            }

            namespace ResourceSpec {

                /** Properties of an AcceleratorConfig. */
                interface IAcceleratorConfig {

                    /** AcceleratorConfig type */
                    type?: (string|null);

                    /** AcceleratorConfig count */
                    count?: (number|Long|null);
                }

                /** Represents an AcceleratorConfig. */
                class AcceleratorConfig implements IAcceleratorConfig {

                    /**
                     * Constructs a new AcceleratorConfig.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig);

                    /** AcceleratorConfig type. */
                    public type: string;

                    /** AcceleratorConfig count. */
                    public count: (number|Long);

                    /**
                     * Creates a new AcceleratorConfig instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns AcceleratorConfig instance
                     */
                    public static create(properties?: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig;

                    /**
                     * Encodes the specified AcceleratorConfig message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.verify|verify} messages.
                     * @param message AcceleratorConfig message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified AcceleratorConfig message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig.verify|verify} messages.
                     * @param message AcceleratorConfig message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.IAcceleratorConfig, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes an AcceleratorConfig message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns AcceleratorConfig
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig;

                    /**
                     * Decodes an AcceleratorConfig message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns AcceleratorConfig
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig;

                    /**
                     * Verifies an AcceleratorConfig message.
                     * @param message Plain object to verify
                     * @returns `null` if valid, otherwise the reason why it is not
                     */
                    public static verify(message: { [k: string]: any }): (string|null);

                    /**
                     * Creates an AcceleratorConfig message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns AcceleratorConfig
                     */
                    public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig;

                    /**
                     * Creates a plain object from an AcceleratorConfig message. Also converts values to other types if specified.
                     * @param message AcceleratorConfig
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: ml_pipelines.PipelineDeploymentConfig.PipelineContainerSpec.ResourceSpec.AcceleratorConfig, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this AcceleratorConfig to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };
                }
            }
        }

        /** Properties of an ImporterSpec. */
        interface IImporterSpec {

            /** ImporterSpec artifactUri */
            artifactUri?: (ml_pipelines.IValueOrRuntimeParameter|null);

            /** ImporterSpec typeSchema */
            typeSchema?: (ml_pipelines.IArtifactTypeSchema|null);

            /** ImporterSpec properties */
            properties?: ({ [k: string]: ml_pipelines.IValueOrRuntimeParameter }|null);

            /** ImporterSpec customProperties */
            customProperties?: ({ [k: string]: ml_pipelines.IValueOrRuntimeParameter }|null);

            /** ImporterSpec metadata */
            metadata?: (google.protobuf.IStruct|null);

            /** ImporterSpec reimport */
            reimport?: (boolean|null);
        }

        /** Represents an ImporterSpec. */
        class ImporterSpec implements IImporterSpec {

            /**
             * Constructs a new ImporterSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.PipelineDeploymentConfig.IImporterSpec);

            /** ImporterSpec artifactUri. */
            public artifactUri?: (ml_pipelines.IValueOrRuntimeParameter|null);

            /** ImporterSpec typeSchema. */
            public typeSchema?: (ml_pipelines.IArtifactTypeSchema|null);

            /** ImporterSpec properties. */
            public properties: { [k: string]: ml_pipelines.IValueOrRuntimeParameter };

            /** ImporterSpec customProperties. */
            public customProperties: { [k: string]: ml_pipelines.IValueOrRuntimeParameter };

            /** ImporterSpec metadata. */
            public metadata?: (google.protobuf.IStruct|null);

            /** ImporterSpec reimport. */
            public reimport: boolean;

            /**
             * Creates a new ImporterSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ImporterSpec instance
             */
            public static create(properties?: ml_pipelines.PipelineDeploymentConfig.IImporterSpec): ml_pipelines.PipelineDeploymentConfig.ImporterSpec;

            /**
             * Encodes the specified ImporterSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ImporterSpec.verify|verify} messages.
             * @param message ImporterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.PipelineDeploymentConfig.IImporterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ImporterSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ImporterSpec.verify|verify} messages.
             * @param message ImporterSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.IImporterSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an ImporterSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ImporterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.ImporterSpec;

            /**
             * Decodes an ImporterSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ImporterSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.ImporterSpec;

            /**
             * Verifies an ImporterSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an ImporterSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ImporterSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.ImporterSpec;

            /**
             * Creates a plain object from an ImporterSpec message. Also converts values to other types if specified.
             * @param message ImporterSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.PipelineDeploymentConfig.ImporterSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ImporterSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a ResolverSpec. */
        interface IResolverSpec {

            /** ResolverSpec outputArtifactQueries */
            outputArtifactQueries?: ({ [k: string]: ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec }|null);
        }

        /** Represents a ResolverSpec. */
        class ResolverSpec implements IResolverSpec {

            /**
             * Constructs a new ResolverSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.PipelineDeploymentConfig.IResolverSpec);

            /** ResolverSpec outputArtifactQueries. */
            public outputArtifactQueries: { [k: string]: ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec };

            /**
             * Creates a new ResolverSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ResolverSpec instance
             */
            public static create(properties?: ml_pipelines.PipelineDeploymentConfig.IResolverSpec): ml_pipelines.PipelineDeploymentConfig.ResolverSpec;

            /**
             * Encodes the specified ResolverSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ResolverSpec.verify|verify} messages.
             * @param message ResolverSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.PipelineDeploymentConfig.IResolverSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ResolverSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ResolverSpec.verify|verify} messages.
             * @param message ResolverSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.IResolverSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a ResolverSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ResolverSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.ResolverSpec;

            /**
             * Decodes a ResolverSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ResolverSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.ResolverSpec;

            /**
             * Verifies a ResolverSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a ResolverSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ResolverSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.ResolverSpec;

            /**
             * Creates a plain object from a ResolverSpec message. Also converts values to other types if specified.
             * @param message ResolverSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.PipelineDeploymentConfig.ResolverSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ResolverSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        namespace ResolverSpec {

            /** Properties of an ArtifactQuerySpec. */
            interface IArtifactQuerySpec {

                /** ArtifactQuerySpec filter */
                filter?: (string|null);

                /** ArtifactQuerySpec limit */
                limit?: (number|null);
            }

            /** Represents an ArtifactQuerySpec. */
            class ArtifactQuerySpec implements IArtifactQuerySpec {

                /**
                 * Constructs a new ArtifactQuerySpec.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec);

                /** ArtifactQuerySpec filter. */
                public filter: string;

                /** ArtifactQuerySpec limit. */
                public limit: number;

                /**
                 * Creates a new ArtifactQuerySpec instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ArtifactQuerySpec instance
                 */
                public static create(properties?: ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec): ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec;

                /**
                 * Encodes the specified ArtifactQuerySpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.verify|verify} messages.
                 * @param message ArtifactQuerySpec message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ArtifactQuerySpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec.verify|verify} messages.
                 * @param message ArtifactQuerySpec message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.ResolverSpec.IArtifactQuerySpec, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an ArtifactQuerySpec message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ArtifactQuerySpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec;

                /**
                 * Decodes an ArtifactQuerySpec message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ArtifactQuerySpec
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec;

                /**
                 * Verifies an ArtifactQuerySpec message.
                 * @param message Plain object to verify
                 * @returns `null` if valid, otherwise the reason why it is not
                 */
                public static verify(message: { [k: string]: any }): (string|null);

                /**
                 * Creates an ArtifactQuerySpec message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ArtifactQuerySpec
                 */
                public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec;

                /**
                 * Creates a plain object from an ArtifactQuerySpec message. Also converts values to other types if specified.
                 * @param message ArtifactQuerySpec
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: ml_pipelines.PipelineDeploymentConfig.ResolverSpec.ArtifactQuerySpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ArtifactQuerySpec to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };
            }
        }

        /** Properties of a AIPlatformCustomJobSpec. */
        interface IAIPlatformCustomJobSpec {

            /** AIPlatformCustomJobSpec customJob */
            customJob?: (google.protobuf.IStruct|null);
        }

        /** Represents a AIPlatformCustomJobSpec. */
        class AIPlatformCustomJobSpec implements IAIPlatformCustomJobSpec {

            /**
             * Constructs a new AIPlatformCustomJobSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec);

            /** AIPlatformCustomJobSpec customJob. */
            public customJob?: (google.protobuf.IStruct|null);

            /**
             * Creates a new AIPlatformCustomJobSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns AIPlatformCustomJobSpec instance
             */
            public static create(properties?: ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec): ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec;

            /**
             * Encodes the specified AIPlatformCustomJobSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.verify|verify} messages.
             * @param message AIPlatformCustomJobSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified AIPlatformCustomJobSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec.verify|verify} messages.
             * @param message AIPlatformCustomJobSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a AIPlatformCustomJobSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns AIPlatformCustomJobSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec;

            /**
             * Decodes a AIPlatformCustomJobSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns AIPlatformCustomJobSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec;

            /**
             * Verifies a AIPlatformCustomJobSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a AIPlatformCustomJobSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns AIPlatformCustomJobSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec;

            /**
             * Creates a plain object from a AIPlatformCustomJobSpec message. Also converts values to other types if specified.
             * @param message AIPlatformCustomJobSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.PipelineDeploymentConfig.AIPlatformCustomJobSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this AIPlatformCustomJobSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of an ExecutorSpec. */
        interface IExecutorSpec {

            /** ExecutorSpec container */
            container?: (ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec|null);

            /** ExecutorSpec importer */
            importer?: (ml_pipelines.PipelineDeploymentConfig.IImporterSpec|null);

            /** ExecutorSpec resolver */
            resolver?: (ml_pipelines.PipelineDeploymentConfig.IResolverSpec|null);

            /** ExecutorSpec customJob */
            customJob?: (ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec|null);
        }

        /** Represents an ExecutorSpec. */
        class ExecutorSpec implements IExecutorSpec {

            /**
             * Constructs a new ExecutorSpec.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.PipelineDeploymentConfig.IExecutorSpec);

            /** ExecutorSpec container. */
            public container?: (ml_pipelines.PipelineDeploymentConfig.IPipelineContainerSpec|null);

            /** ExecutorSpec importer. */
            public importer?: (ml_pipelines.PipelineDeploymentConfig.IImporterSpec|null);

            /** ExecutorSpec resolver. */
            public resolver?: (ml_pipelines.PipelineDeploymentConfig.IResolverSpec|null);

            /** ExecutorSpec customJob. */
            public customJob?: (ml_pipelines.PipelineDeploymentConfig.IAIPlatformCustomJobSpec|null);

            /** ExecutorSpec spec. */
            public spec?: ("container"|"importer"|"resolver"|"customJob");

            /**
             * Creates a new ExecutorSpec instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ExecutorSpec instance
             */
            public static create(properties?: ml_pipelines.PipelineDeploymentConfig.IExecutorSpec): ml_pipelines.PipelineDeploymentConfig.ExecutorSpec;

            /**
             * Encodes the specified ExecutorSpec message. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.verify|verify} messages.
             * @param message ExecutorSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.PipelineDeploymentConfig.IExecutorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ExecutorSpec message, length delimited. Does not implicitly {@link ml_pipelines.PipelineDeploymentConfig.ExecutorSpec.verify|verify} messages.
             * @param message ExecutorSpec message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.PipelineDeploymentConfig.IExecutorSpec, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an ExecutorSpec message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ExecutorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineDeploymentConfig.ExecutorSpec;

            /**
             * Decodes an ExecutorSpec message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ExecutorSpec
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineDeploymentConfig.ExecutorSpec;

            /**
             * Verifies an ExecutorSpec message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an ExecutorSpec message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ExecutorSpec
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineDeploymentConfig.ExecutorSpec;

            /**
             * Creates a plain object from an ExecutorSpec message. Also converts values to other types if specified.
             * @param message ExecutorSpec
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.PipelineDeploymentConfig.ExecutorSpec, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ExecutorSpec to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of a Value. */
    interface IValue {

        /** Value intValue */
        intValue?: (number|Long|null);

        /** Value doubleValue */
        doubleValue?: (number|null);

        /** Value stringValue */
        stringValue?: (string|null);
    }

    /** Represents a Value. */
    class Value implements IValue {

        /**
         * Constructs a new Value.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IValue);

        /** Value intValue. */
        public intValue?: (number|Long|null);

        /** Value doubleValue. */
        public doubleValue?: (number|null);

        /** Value stringValue. */
        public stringValue?: (string|null);

        /** Value value. */
        public value?: ("intValue"|"doubleValue"|"stringValue");

        /**
         * Creates a new Value instance using the specified properties.
         * @param [properties] Properties to set
         * @returns Value instance
         */
        public static create(properties?: ml_pipelines.IValue): ml_pipelines.Value;

        /**
         * Encodes the specified Value message. Does not implicitly {@link ml_pipelines.Value.verify|verify} messages.
         * @param message Value message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IValue, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified Value message, length delimited. Does not implicitly {@link ml_pipelines.Value.verify|verify} messages.
         * @param message Value message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IValue, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a Value message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns Value
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.Value;

        /**
         * Decodes a Value message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns Value
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.Value;

        /**
         * Verifies a Value message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a Value message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns Value
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.Value;

        /**
         * Creates a plain object from a Value message. Also converts values to other types if specified.
         * @param message Value
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.Value, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this Value to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of a RuntimeArtifact. */
    interface IRuntimeArtifact {

        /** RuntimeArtifact name */
        name?: (string|null);

        /** RuntimeArtifact type */
        type?: (ml_pipelines.IArtifactTypeSchema|null);

        /** RuntimeArtifact uri */
        uri?: (string|null);

        /** RuntimeArtifact properties */
        properties?: ({ [k: string]: ml_pipelines.IValue }|null);

        /** RuntimeArtifact customProperties */
        customProperties?: ({ [k: string]: ml_pipelines.IValue }|null);

        /** RuntimeArtifact metadata */
        metadata?: (google.protobuf.IStruct|null);
    }

    /** Represents a RuntimeArtifact. */
    class RuntimeArtifact implements IRuntimeArtifact {

        /**
         * Constructs a new RuntimeArtifact.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IRuntimeArtifact);

        /** RuntimeArtifact name. */
        public name: string;

        /** RuntimeArtifact type. */
        public type?: (ml_pipelines.IArtifactTypeSchema|null);

        /** RuntimeArtifact uri. */
        public uri: string;

        /** RuntimeArtifact properties. */
        public properties: { [k: string]: ml_pipelines.IValue };

        /** RuntimeArtifact customProperties. */
        public customProperties: { [k: string]: ml_pipelines.IValue };

        /** RuntimeArtifact metadata. */
        public metadata?: (google.protobuf.IStruct|null);

        /**
         * Creates a new RuntimeArtifact instance using the specified properties.
         * @param [properties] Properties to set
         * @returns RuntimeArtifact instance
         */
        public static create(properties?: ml_pipelines.IRuntimeArtifact): ml_pipelines.RuntimeArtifact;

        /**
         * Encodes the specified RuntimeArtifact message. Does not implicitly {@link ml_pipelines.RuntimeArtifact.verify|verify} messages.
         * @param message RuntimeArtifact message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IRuntimeArtifact, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified RuntimeArtifact message, length delimited. Does not implicitly {@link ml_pipelines.RuntimeArtifact.verify|verify} messages.
         * @param message RuntimeArtifact message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IRuntimeArtifact, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a RuntimeArtifact message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns RuntimeArtifact
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.RuntimeArtifact;

        /**
         * Decodes a RuntimeArtifact message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns RuntimeArtifact
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.RuntimeArtifact;

        /**
         * Verifies a RuntimeArtifact message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a RuntimeArtifact message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns RuntimeArtifact
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.RuntimeArtifact;

        /**
         * Creates a plain object from a RuntimeArtifact message. Also converts values to other types if specified.
         * @param message RuntimeArtifact
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.RuntimeArtifact, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this RuntimeArtifact to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of an ArtifactList. */
    interface IArtifactList {

        /** ArtifactList artifacts */
        artifacts?: (ml_pipelines.IRuntimeArtifact[]|null);
    }

    /** Represents an ArtifactList. */
    class ArtifactList implements IArtifactList {

        /**
         * Constructs a new ArtifactList.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IArtifactList);

        /** ArtifactList artifacts. */
        public artifacts: ml_pipelines.IRuntimeArtifact[];

        /**
         * Creates a new ArtifactList instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ArtifactList instance
         */
        public static create(properties?: ml_pipelines.IArtifactList): ml_pipelines.ArtifactList;

        /**
         * Encodes the specified ArtifactList message. Does not implicitly {@link ml_pipelines.ArtifactList.verify|verify} messages.
         * @param message ArtifactList message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IArtifactList, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ArtifactList message, length delimited. Does not implicitly {@link ml_pipelines.ArtifactList.verify|verify} messages.
         * @param message ArtifactList message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IArtifactList, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an ArtifactList message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ArtifactList
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ArtifactList;

        /**
         * Decodes an ArtifactList message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ArtifactList
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ArtifactList;

        /**
         * Verifies an ArtifactList message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an ArtifactList message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ArtifactList
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ArtifactList;

        /**
         * Creates a plain object from an ArtifactList message. Also converts values to other types if specified.
         * @param message ArtifactList
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ArtifactList, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ArtifactList to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of an ExecutorInput. */
    interface IExecutorInput {

        /** ExecutorInput inputs */
        inputs?: (ml_pipelines.ExecutorInput.IInputs|null);

        /** ExecutorInput outputs */
        outputs?: (ml_pipelines.ExecutorInput.IOutputs|null);
    }

    /** Represents an ExecutorInput. */
    class ExecutorInput implements IExecutorInput {

        /**
         * Constructs a new ExecutorInput.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IExecutorInput);

        /** ExecutorInput inputs. */
        public inputs?: (ml_pipelines.ExecutorInput.IInputs|null);

        /** ExecutorInput outputs. */
        public outputs?: (ml_pipelines.ExecutorInput.IOutputs|null);

        /**
         * Creates a new ExecutorInput instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ExecutorInput instance
         */
        public static create(properties?: ml_pipelines.IExecutorInput): ml_pipelines.ExecutorInput;

        /**
         * Encodes the specified ExecutorInput message. Does not implicitly {@link ml_pipelines.ExecutorInput.verify|verify} messages.
         * @param message ExecutorInput message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IExecutorInput, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ExecutorInput message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorInput.verify|verify} messages.
         * @param message ExecutorInput message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IExecutorInput, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an ExecutorInput message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ExecutorInput
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ExecutorInput;

        /**
         * Decodes an ExecutorInput message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ExecutorInput
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ExecutorInput;

        /**
         * Verifies an ExecutorInput message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an ExecutorInput message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ExecutorInput
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ExecutorInput;

        /**
         * Creates a plain object from an ExecutorInput message. Also converts values to other types if specified.
         * @param message ExecutorInput
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ExecutorInput, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ExecutorInput to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace ExecutorInput {

        /** Properties of an Inputs. */
        interface IInputs {

            /** Inputs parameters */
            parameters?: ({ [k: string]: ml_pipelines.IValue }|null);

            /** Inputs artifacts */
            artifacts?: ({ [k: string]: ml_pipelines.IArtifactList }|null);
        }

        /** Represents an Inputs. */
        class Inputs implements IInputs {

            /**
             * Constructs a new Inputs.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.ExecutorInput.IInputs);

            /** Inputs parameters. */
            public parameters: { [k: string]: ml_pipelines.IValue };

            /** Inputs artifacts. */
            public artifacts: { [k: string]: ml_pipelines.IArtifactList };

            /**
             * Creates a new Inputs instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Inputs instance
             */
            public static create(properties?: ml_pipelines.ExecutorInput.IInputs): ml_pipelines.ExecutorInput.Inputs;

            /**
             * Encodes the specified Inputs message. Does not implicitly {@link ml_pipelines.ExecutorInput.Inputs.verify|verify} messages.
             * @param message Inputs message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.ExecutorInput.IInputs, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Inputs message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorInput.Inputs.verify|verify} messages.
             * @param message Inputs message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.ExecutorInput.IInputs, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an Inputs message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Inputs
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ExecutorInput.Inputs;

            /**
             * Decodes an Inputs message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Inputs
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ExecutorInput.Inputs;

            /**
             * Verifies an Inputs message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an Inputs message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Inputs
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.ExecutorInput.Inputs;

            /**
             * Creates a plain object from an Inputs message. Also converts values to other types if specified.
             * @param message Inputs
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.ExecutorInput.Inputs, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Inputs to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of an OutputParameter. */
        interface IOutputParameter {

            /** OutputParameter outputFile */
            outputFile?: (string|null);
        }

        /** Represents an OutputParameter. */
        class OutputParameter implements IOutputParameter {

            /**
             * Constructs a new OutputParameter.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.ExecutorInput.IOutputParameter);

            /** OutputParameter outputFile. */
            public outputFile: string;

            /**
             * Creates a new OutputParameter instance using the specified properties.
             * @param [properties] Properties to set
             * @returns OutputParameter instance
             */
            public static create(properties?: ml_pipelines.ExecutorInput.IOutputParameter): ml_pipelines.ExecutorInput.OutputParameter;

            /**
             * Encodes the specified OutputParameter message. Does not implicitly {@link ml_pipelines.ExecutorInput.OutputParameter.verify|verify} messages.
             * @param message OutputParameter message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.ExecutorInput.IOutputParameter, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified OutputParameter message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorInput.OutputParameter.verify|verify} messages.
             * @param message OutputParameter message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.ExecutorInput.IOutputParameter, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an OutputParameter message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns OutputParameter
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ExecutorInput.OutputParameter;

            /**
             * Decodes an OutputParameter message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns OutputParameter
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ExecutorInput.OutputParameter;

            /**
             * Verifies an OutputParameter message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an OutputParameter message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns OutputParameter
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.ExecutorInput.OutputParameter;

            /**
             * Creates a plain object from an OutputParameter message. Also converts values to other types if specified.
             * @param message OutputParameter
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.ExecutorInput.OutputParameter, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this OutputParameter to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of an Outputs. */
        interface IOutputs {

            /** Outputs parameters */
            parameters?: ({ [k: string]: ml_pipelines.ExecutorInput.IOutputParameter }|null);

            /** Outputs artifacts */
            artifacts?: ({ [k: string]: ml_pipelines.IArtifactList }|null);

            /** Outputs outputFile */
            outputFile?: (string|null);
        }

        /** Represents an Outputs. */
        class Outputs implements IOutputs {

            /**
             * Constructs a new Outputs.
             * @param [properties] Properties to set
             */
            constructor(properties?: ml_pipelines.ExecutorInput.IOutputs);

            /** Outputs parameters. */
            public parameters: { [k: string]: ml_pipelines.ExecutorInput.IOutputParameter };

            /** Outputs artifacts. */
            public artifacts: { [k: string]: ml_pipelines.IArtifactList };

            /** Outputs outputFile. */
            public outputFile: string;

            /**
             * Creates a new Outputs instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Outputs instance
             */
            public static create(properties?: ml_pipelines.ExecutorInput.IOutputs): ml_pipelines.ExecutorInput.Outputs;

            /**
             * Encodes the specified Outputs message. Does not implicitly {@link ml_pipelines.ExecutorInput.Outputs.verify|verify} messages.
             * @param message Outputs message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: ml_pipelines.ExecutorInput.IOutputs, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Outputs message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorInput.Outputs.verify|verify} messages.
             * @param message Outputs message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: ml_pipelines.ExecutorInput.IOutputs, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an Outputs message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Outputs
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ExecutorInput.Outputs;

            /**
             * Decodes an Outputs message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Outputs
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ExecutorInput.Outputs;

            /**
             * Verifies an Outputs message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an Outputs message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Outputs
             */
            public static fromObject(object: { [k: string]: any }): ml_pipelines.ExecutorInput.Outputs;

            /**
             * Creates a plain object from an Outputs message. Also converts values to other types if specified.
             * @param message Outputs
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: ml_pipelines.ExecutorInput.Outputs, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Outputs to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Properties of an ExecutorOutput. */
    interface IExecutorOutput {

        /** ExecutorOutput parameters */
        parameters?: ({ [k: string]: ml_pipelines.IValue }|null);

        /** ExecutorOutput artifacts */
        artifacts?: ({ [k: string]: ml_pipelines.IArtifactList }|null);
    }

    /** Represents an ExecutorOutput. */
    class ExecutorOutput implements IExecutorOutput {

        /**
         * Constructs a new ExecutorOutput.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IExecutorOutput);

        /** ExecutorOutput parameters. */
        public parameters: { [k: string]: ml_pipelines.IValue };

        /** ExecutorOutput artifacts. */
        public artifacts: { [k: string]: ml_pipelines.IArtifactList };

        /**
         * Creates a new ExecutorOutput instance using the specified properties.
         * @param [properties] Properties to set
         * @returns ExecutorOutput instance
         */
        public static create(properties?: ml_pipelines.IExecutorOutput): ml_pipelines.ExecutorOutput;

        /**
         * Encodes the specified ExecutorOutput message. Does not implicitly {@link ml_pipelines.ExecutorOutput.verify|verify} messages.
         * @param message ExecutorOutput message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IExecutorOutput, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified ExecutorOutput message, length delimited. Does not implicitly {@link ml_pipelines.ExecutorOutput.verify|verify} messages.
         * @param message ExecutorOutput message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IExecutorOutput, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes an ExecutorOutput message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns ExecutorOutput
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.ExecutorOutput;

        /**
         * Decodes an ExecutorOutput message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns ExecutorOutput
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.ExecutorOutput;

        /**
         * Verifies an ExecutorOutput message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates an ExecutorOutput message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns ExecutorOutput
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.ExecutorOutput;

        /**
         * Creates a plain object from an ExecutorOutput message. Also converts values to other types if specified.
         * @param message ExecutorOutput
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.ExecutorOutput, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this ExecutorOutput to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of a PipelineTaskFinalStatus. */
    interface IPipelineTaskFinalStatus {

        /** PipelineTaskFinalStatus state */
        state?: (string|null);

        /** PipelineTaskFinalStatus error */
        error?: (google.rpc.IStatus|null);
    }

    /** Represents a PipelineTaskFinalStatus. */
    class PipelineTaskFinalStatus implements IPipelineTaskFinalStatus {

        /**
         * Constructs a new PipelineTaskFinalStatus.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IPipelineTaskFinalStatus);

        /** PipelineTaskFinalStatus state. */
        public state: string;

        /** PipelineTaskFinalStatus error. */
        public error?: (google.rpc.IStatus|null);

        /**
         * Creates a new PipelineTaskFinalStatus instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PipelineTaskFinalStatus instance
         */
        public static create(properties?: ml_pipelines.IPipelineTaskFinalStatus): ml_pipelines.PipelineTaskFinalStatus;

        /**
         * Encodes the specified PipelineTaskFinalStatus message. Does not implicitly {@link ml_pipelines.PipelineTaskFinalStatus.verify|verify} messages.
         * @param message PipelineTaskFinalStatus message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IPipelineTaskFinalStatus, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PipelineTaskFinalStatus message, length delimited. Does not implicitly {@link ml_pipelines.PipelineTaskFinalStatus.verify|verify} messages.
         * @param message PipelineTaskFinalStatus message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IPipelineTaskFinalStatus, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PipelineTaskFinalStatus message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns PipelineTaskFinalStatus
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineTaskFinalStatus;

        /**
         * Decodes a PipelineTaskFinalStatus message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns PipelineTaskFinalStatus
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineTaskFinalStatus;

        /**
         * Verifies a PipelineTaskFinalStatus message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PipelineTaskFinalStatus message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PipelineTaskFinalStatus
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineTaskFinalStatus;

        /**
         * Creates a plain object from a PipelineTaskFinalStatus message. Also converts values to other types if specified.
         * @param message PipelineTaskFinalStatus
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.PipelineTaskFinalStatus, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PipelineTaskFinalStatus to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    /** Properties of a PipelineStateEnum. */
    interface IPipelineStateEnum {
    }

    /** Represents a PipelineStateEnum. */
    class PipelineStateEnum implements IPipelineStateEnum {

        /**
         * Constructs a new PipelineStateEnum.
         * @param [properties] Properties to set
         */
        constructor(properties?: ml_pipelines.IPipelineStateEnum);

        /**
         * Creates a new PipelineStateEnum instance using the specified properties.
         * @param [properties] Properties to set
         * @returns PipelineStateEnum instance
         */
        public static create(properties?: ml_pipelines.IPipelineStateEnum): ml_pipelines.PipelineStateEnum;

        /**
         * Encodes the specified PipelineStateEnum message. Does not implicitly {@link ml_pipelines.PipelineStateEnum.verify|verify} messages.
         * @param message PipelineStateEnum message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encode(message: ml_pipelines.IPipelineStateEnum, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Encodes the specified PipelineStateEnum message, length delimited. Does not implicitly {@link ml_pipelines.PipelineStateEnum.verify|verify} messages.
         * @param message PipelineStateEnum message or plain object to encode
         * @param [writer] Writer to encode to
         * @returns Writer
         */
        public static encodeDelimited(message: ml_pipelines.IPipelineStateEnum, writer?: $protobuf.Writer): $protobuf.Writer;

        /**
         * Decodes a PipelineStateEnum message from the specified reader or buffer.
         * @param reader Reader or buffer to decode from
         * @param [length] Message length if known beforehand
         * @returns PipelineStateEnum
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ml_pipelines.PipelineStateEnum;

        /**
         * Decodes a PipelineStateEnum message from the specified reader or buffer, length delimited.
         * @param reader Reader or buffer to decode from
         * @returns PipelineStateEnum
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ml_pipelines.PipelineStateEnum;

        /**
         * Verifies a PipelineStateEnum message.
         * @param message Plain object to verify
         * @returns `null` if valid, otherwise the reason why it is not
         */
        public static verify(message: { [k: string]: any }): (string|null);

        /**
         * Creates a PipelineStateEnum message from a plain object. Also converts values to their respective internal types.
         * @param object Plain object
         * @returns PipelineStateEnum
         */
        public static fromObject(object: { [k: string]: any }): ml_pipelines.PipelineStateEnum;

        /**
         * Creates a plain object from a PipelineStateEnum message. Also converts values to other types if specified.
         * @param message PipelineStateEnum
         * @param [options] Conversion options
         * @returns Plain object
         */
        public static toObject(message: ml_pipelines.PipelineStateEnum, options?: $protobuf.IConversionOptions): { [k: string]: any };

        /**
         * Converts this PipelineStateEnum to JSON.
         * @returns JSON object
         */
        public toJSON(): { [k: string]: any };
    }

    namespace PipelineStateEnum {

        /** PipelineTaskState enum. */
        enum PipelineTaskState {
            TASK_STATE_UNSPECIFIED = 0,
            PENDING = 1,
            RUNNING_DRIVER = 2,
            DRIVER_SUCCEEDED = 3,
            RUNNING_EXECUTOR = 4,
            SUCCEEDED = 5,
            CANCEL_PENDING = 6,
            CANCELLING = 7,
            CANCELLED = 8,
            FAILED = 9,
            SKIPPED = 10,
            QUEUED = 11,
            NOT_TRIGGERED = 12,
            UNSCHEDULABLE = 13
        }
    }
}

/** Namespace google. */
export namespace google {

    /** Namespace protobuf. */
    namespace protobuf {

        /** Properties of an Any. */
        interface IAny {

            /** Any type_url */
            type_url?: (string|null);

            /** Any value */
            value?: (Uint8Array|null);
        }

        /** Represents an Any. */
        class Any implements IAny {

            /**
             * Constructs a new Any.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IAny);

            /** Any type_url. */
            public type_url: string;

            /** Any value. */
            public value: Uint8Array;

            /**
             * Creates a new Any instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Any instance
             */
            public static create(properties?: google.protobuf.IAny): google.protobuf.Any;

            /**
             * Encodes the specified Any message. Does not implicitly {@link google.protobuf.Any.verify|verify} messages.
             * @param message Any message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IAny, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Any message, length delimited. Does not implicitly {@link google.protobuf.Any.verify|verify} messages.
             * @param message Any message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IAny, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an Any message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Any
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.Any;

            /**
             * Decodes an Any message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Any
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.Any;

            /**
             * Verifies an Any message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an Any message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Any
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.Any;

            /**
             * Creates a plain object from an Any message. Also converts values to other types if specified.
             * @param message Any
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.Any, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Any to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a Struct. */
        interface IStruct {

            /** Struct fields */
            fields?: ({ [k: string]: google.protobuf.IValue }|null);
        }

        /** Represents a Struct. */
        class Struct implements IStruct {

            /**
             * Constructs a new Struct.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IStruct);

            /** Struct fields. */
            public fields: { [k: string]: google.protobuf.IValue };

            /**
             * Creates a new Struct instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Struct instance
             */
            public static create(properties?: google.protobuf.IStruct): google.protobuf.Struct;

            /**
             * Encodes the specified Struct message. Does not implicitly {@link google.protobuf.Struct.verify|verify} messages.
             * @param message Struct message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IStruct, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Struct message, length delimited. Does not implicitly {@link google.protobuf.Struct.verify|verify} messages.
             * @param message Struct message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IStruct, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a Struct message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Struct
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.Struct;

            /**
             * Decodes a Struct message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Struct
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.Struct;

            /**
             * Verifies a Struct message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a Struct message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Struct
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.Struct;

            /**
             * Creates a plain object from a Struct message. Also converts values to other types if specified.
             * @param message Struct
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.Struct, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Struct to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** Properties of a Value. */
        interface IValue {

            /** Value nullValue */
            nullValue?: (google.protobuf.NullValue|null);

            /** Value numberValue */
            numberValue?: (number|null);

            /** Value stringValue */
            stringValue?: (string|null);

            /** Value boolValue */
            boolValue?: (boolean|null);

            /** Value structValue */
            structValue?: (google.protobuf.IStruct|null);

            /** Value listValue */
            listValue?: (google.protobuf.IListValue|null);
        }

        /** Represents a Value. */
        class Value implements IValue {

            /**
             * Constructs a new Value.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IValue);

            /** Value nullValue. */
            public nullValue?: (google.protobuf.NullValue|null);

            /** Value numberValue. */
            public numberValue?: (number|null);

            /** Value stringValue. */
            public stringValue?: (string|null);

            /** Value boolValue. */
            public boolValue?: (boolean|null);

            /** Value structValue. */
            public structValue?: (google.protobuf.IStruct|null);

            /** Value listValue. */
            public listValue?: (google.protobuf.IListValue|null);

            /** Value kind. */
            public kind?: ("nullValue"|"numberValue"|"stringValue"|"boolValue"|"structValue"|"listValue");

            /**
             * Creates a new Value instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Value instance
             */
            public static create(properties?: google.protobuf.IValue): google.protobuf.Value;

            /**
             * Encodes the specified Value message. Does not implicitly {@link google.protobuf.Value.verify|verify} messages.
             * @param message Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Value message, length delimited. Does not implicitly {@link google.protobuf.Value.verify|verify} messages.
             * @param message Value message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a Value message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.Value;

            /**
             * Decodes a Value message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Value
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.Value;

            /**
             * Verifies a Value message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a Value message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Value
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.Value;

            /**
             * Creates a plain object from a Value message. Also converts values to other types if specified.
             * @param message Value
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.Value, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Value to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }

        /** NullValue enum. */
        enum NullValue {
            NULL_VALUE = 0
        }

        /** Properties of a ListValue. */
        interface IListValue {

            /** ListValue values */
            values?: (google.protobuf.IValue[]|null);
        }

        /** Represents a ListValue. */
        class ListValue implements IListValue {

            /**
             * Constructs a new ListValue.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IListValue);

            /** ListValue values. */
            public values: google.protobuf.IValue[];

            /**
             * Creates a new ListValue instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ListValue instance
             */
            public static create(properties?: google.protobuf.IListValue): google.protobuf.ListValue;

            /**
             * Encodes the specified ListValue message. Does not implicitly {@link google.protobuf.ListValue.verify|verify} messages.
             * @param message ListValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IListValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ListValue message, length delimited. Does not implicitly {@link google.protobuf.ListValue.verify|verify} messages.
             * @param message ListValue message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IListValue, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a ListValue message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ListValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.ListValue;

            /**
             * Decodes a ListValue message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ListValue
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.ListValue;

            /**
             * Verifies a ListValue message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a ListValue message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ListValue
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.ListValue;

            /**
             * Creates a plain object from a ListValue message. Also converts values to other types if specified.
             * @param message ListValue
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.ListValue, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ListValue to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }

    /** Namespace rpc. */
    namespace rpc {

        /** Properties of a Status. */
        interface IStatus {

            /** Status code */
            code?: (number|null);

            /** Status message */
            message?: (string|null);

            /** Status details */
            details?: (google.protobuf.IAny[]|null);
        }

        /** Represents a Status. */
        class Status implements IStatus {

            /**
             * Constructs a new Status.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.rpc.IStatus);

            /** Status code. */
            public code: number;

            /** Status message. */
            public message: string;

            /** Status details. */
            public details: google.protobuf.IAny[];

            /**
             * Creates a new Status instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Status instance
             */
            public static create(properties?: google.rpc.IStatus): google.rpc.Status;

            /**
             * Encodes the specified Status message. Does not implicitly {@link google.rpc.Status.verify|verify} messages.
             * @param message Status message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.rpc.IStatus, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Status message, length delimited. Does not implicitly {@link google.rpc.Status.verify|verify} messages.
             * @param message Status message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.rpc.IStatus, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a Status message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Status
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.rpc.Status;

            /**
             * Decodes a Status message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Status
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.rpc.Status;

            /**
             * Verifies a Status message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            public static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates a Status message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Status
             */
            public static fromObject(object: { [k: string]: any }): google.rpc.Status;

            /**
             * Creates a plain object from a Status message. Also converts values to other types if specified.
             * @param message Status
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.rpc.Status, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Status to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };
        }
    }
}
