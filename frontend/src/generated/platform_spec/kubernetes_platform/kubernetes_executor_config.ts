/* eslint-disable */
import Long from 'long';
import _m0 from 'protobufjs/minimal';
import { Struct } from './google/protobuf/struct';

export const protobufPackage = 'kfp_kubernetes';

export interface KubernetesExecutorConfig {
  secretAsVolume: SecretAsVolume[];
  secretAsEnv: SecretAsEnv[];
  pvcMount: PvcMount[];
  nodeSelector: NodeSelector | undefined;
}

export interface SecretAsVolume {
  /** Name of the Secret. */
  secretName: string;
  /** Container path to mount the Secret data. */
  mountPath: string;
}

export interface SecretAsEnv {
  /** Name of the Secret. */
  secretName: string;
  keyToEnv: SecretAsEnv_SecretKeyToEnvMap[];
}

export interface SecretAsEnv_SecretKeyToEnvMap {
  /** Corresponds to a key of the Secret.data field. */
  secretKey: string;
  /** Env var to which secret_key's data should be set. */
  envVar: string;
}

/** Represents an upstream task's output parameter. */
export interface TaskOutputParameterSpec {
  /**
   * The name of the upstream task which produces the output parameter that
   * matches with the `output_parameter_key`.
   */
  producerTask: string;
  /** The key of [TaskOutputsSpec.parameters][] map of the producer task. */
  outputParameterKey: string;
}

export interface PvcMount {
  /** Output parameter from an upstream task. */
  taskOutputParameter: TaskOutputParameterSpec | undefined;
  /** A constant value. */
  constant: string | undefined;
  /** Pass the input parameter from parent component input parameter. */
  componentInputParameter: string | undefined;
  /** Container path to which the PVC should be mounted. */
  mountPath: string;
}

export interface CreatePvc {
  /** Name of the PVC, if not dynamically generated. */
  pvcName: string | undefined;
  /**
   * Suffix for a dynamically generated PVC name of the form
   * {{workflow.name}}-<pvc_name_suffix>.
   */
  pvcNameSuffix: string | undefined;
  /** Corresponds to PersistentVolumeClaim.spec.accessMode field. */
  accessModes: string[];
  /** Corresponds to PersistentVolumeClaim.spec.resources.requests.storage field. */
  size: string;
  /** If true, corresponds to omitted PersistentVolumeClaim.spec.storageClassName. */
  defaultStorageClass: boolean;
  /**
   * Corresponds to PersistentVolumeClaim.spec.storageClassName string field.
   * Should only be used if default_storage_class is false.
   */
  storageClassName: string;
  /** Corresponds to PersistentVolumeClaim.spec.volumeName field. */
  volumeName: string;
  /** Corresponds to PersistentVolumeClaim.metadata.annotations field. */
  annotations: { [key: string]: any } | undefined;
}

export interface DeletePvc {
  /** Output parameter from an upstream task. */
  taskOutputParameter: TaskOutputParameterSpec | undefined;
  /** A constant value. */
  constant: string | undefined;
  /** Pass the input parameter from parent component input parameter. */
  componentInputParameter: string | undefined;
}

export interface NodeSelector {
  /**
   * map of label key to label value
   * corresponds to Pod.spec.nodeSelector field https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling
   */
  labels: { [key: string]: string };
}

export interface NodeSelector_LabelsEntry {
  key: string;
  value: string;
}

const baseKubernetesExecutorConfig: object = {};

export const KubernetesExecutorConfig = {
  encode(message: KubernetesExecutorConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.secretAsVolume) {
      SecretAsVolume.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.secretAsEnv) {
      SecretAsEnv.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.pvcMount) {
      PvcMount.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    if (message.nodeSelector !== undefined) {
      NodeSelector.encode(message.nodeSelector, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): KubernetesExecutorConfig {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseKubernetesExecutorConfig } as KubernetesExecutorConfig;
    message.secretAsVolume = [];
    message.secretAsEnv = [];
    message.pvcMount = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.secretAsVolume.push(SecretAsVolume.decode(reader, reader.uint32()));
          break;
        case 2:
          message.secretAsEnv.push(SecretAsEnv.decode(reader, reader.uint32()));
          break;
        case 3:
          message.pvcMount.push(PvcMount.decode(reader, reader.uint32()));
          break;
        case 4:
          message.nodeSelector = NodeSelector.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): KubernetesExecutorConfig {
    const message = { ...baseKubernetesExecutorConfig } as KubernetesExecutorConfig;
    message.secretAsVolume = (object.secretAsVolume ?? []).map((e: any) =>
      SecretAsVolume.fromJSON(e),
    );
    message.secretAsEnv = (object.secretAsEnv ?? []).map((e: any) => SecretAsEnv.fromJSON(e));
    message.pvcMount = (object.pvcMount ?? []).map((e: any) => PvcMount.fromJSON(e));
    message.nodeSelector =
      object.nodeSelector !== undefined && object.nodeSelector !== null
        ? NodeSelector.fromJSON(object.nodeSelector)
        : undefined;
    return message;
  },

  toJSON(message: KubernetesExecutorConfig): unknown {
    const obj: any = {};
    if (message.secretAsVolume) {
      obj.secretAsVolume = message.secretAsVolume.map((e) =>
        e ? SecretAsVolume.toJSON(e) : undefined,
      );
    } else {
      obj.secretAsVolume = [];
    }
    if (message.secretAsEnv) {
      obj.secretAsEnv = message.secretAsEnv.map((e) => (e ? SecretAsEnv.toJSON(e) : undefined));
    } else {
      obj.secretAsEnv = [];
    }
    if (message.pvcMount) {
      obj.pvcMount = message.pvcMount.map((e) => (e ? PvcMount.toJSON(e) : undefined));
    } else {
      obj.pvcMount = [];
    }
    message.nodeSelector !== undefined &&
      (obj.nodeSelector = message.nodeSelector
        ? NodeSelector.toJSON(message.nodeSelector)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<KubernetesExecutorConfig>, I>>(
    object: I,
  ): KubernetesExecutorConfig {
    const message = { ...baseKubernetesExecutorConfig } as KubernetesExecutorConfig;
    message.secretAsVolume = object.secretAsVolume?.map((e) => SecretAsVolume.fromPartial(e)) || [];
    message.secretAsEnv = object.secretAsEnv?.map((e) => SecretAsEnv.fromPartial(e)) || [];
    message.pvcMount = object.pvcMount?.map((e) => PvcMount.fromPartial(e)) || [];
    message.nodeSelector =
      object.nodeSelector !== undefined && object.nodeSelector !== null
        ? NodeSelector.fromPartial(object.nodeSelector)
        : undefined;
    return message;
  },
};

const baseSecretAsVolume: object = { secretName: '', mountPath: '' };

export const SecretAsVolume = {
  encode(message: SecretAsVolume, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.secretName !== '') {
      writer.uint32(10).string(message.secretName);
    }
    if (message.mountPath !== '') {
      writer.uint32(18).string(message.mountPath);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SecretAsVolume {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseSecretAsVolume } as SecretAsVolume;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.secretName = reader.string();
          break;
        case 2:
          message.mountPath = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SecretAsVolume {
    const message = { ...baseSecretAsVolume } as SecretAsVolume;
    message.secretName =
      object.secretName !== undefined && object.secretName !== null
        ? String(object.secretName)
        : '';
    message.mountPath =
      object.mountPath !== undefined && object.mountPath !== null ? String(object.mountPath) : '';
    return message;
  },

  toJSON(message: SecretAsVolume): unknown {
    const obj: any = {};
    message.secretName !== undefined && (obj.secretName = message.secretName);
    message.mountPath !== undefined && (obj.mountPath = message.mountPath);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SecretAsVolume>, I>>(object: I): SecretAsVolume {
    const message = { ...baseSecretAsVolume } as SecretAsVolume;
    message.secretName = object.secretName ?? '';
    message.mountPath = object.mountPath ?? '';
    return message;
  },
};

const baseSecretAsEnv: object = { secretName: '' };

export const SecretAsEnv = {
  encode(message: SecretAsEnv, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.secretName !== '') {
      writer.uint32(10).string(message.secretName);
    }
    for (const v of message.keyToEnv) {
      SecretAsEnv_SecretKeyToEnvMap.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SecretAsEnv {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseSecretAsEnv } as SecretAsEnv;
    message.keyToEnv = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.secretName = reader.string();
          break;
        case 2:
          message.keyToEnv.push(SecretAsEnv_SecretKeyToEnvMap.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SecretAsEnv {
    const message = { ...baseSecretAsEnv } as SecretAsEnv;
    message.secretName =
      object.secretName !== undefined && object.secretName !== null
        ? String(object.secretName)
        : '';
    message.keyToEnv = (object.keyToEnv ?? []).map((e: any) =>
      SecretAsEnv_SecretKeyToEnvMap.fromJSON(e),
    );
    return message;
  },

  toJSON(message: SecretAsEnv): unknown {
    const obj: any = {};
    message.secretName !== undefined && (obj.secretName = message.secretName);
    if (message.keyToEnv) {
      obj.keyToEnv = message.keyToEnv.map((e) =>
        e ? SecretAsEnv_SecretKeyToEnvMap.toJSON(e) : undefined,
      );
    } else {
      obj.keyToEnv = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SecretAsEnv>, I>>(object: I): SecretAsEnv {
    const message = { ...baseSecretAsEnv } as SecretAsEnv;
    message.secretName = object.secretName ?? '';
    message.keyToEnv =
      object.keyToEnv?.map((e) => SecretAsEnv_SecretKeyToEnvMap.fromPartial(e)) || [];
    return message;
  },
};

const baseSecretAsEnv_SecretKeyToEnvMap: object = { secretKey: '', envVar: '' };

export const SecretAsEnv_SecretKeyToEnvMap = {
  encode(
    message: SecretAsEnv_SecretKeyToEnvMap,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.secretKey !== '') {
      writer.uint32(10).string(message.secretKey);
    }
    if (message.envVar !== '') {
      writer.uint32(18).string(message.envVar);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SecretAsEnv_SecretKeyToEnvMap {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseSecretAsEnv_SecretKeyToEnvMap } as SecretAsEnv_SecretKeyToEnvMap;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.secretKey = reader.string();
          break;
        case 2:
          message.envVar = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SecretAsEnv_SecretKeyToEnvMap {
    const message = { ...baseSecretAsEnv_SecretKeyToEnvMap } as SecretAsEnv_SecretKeyToEnvMap;
    message.secretKey =
      object.secretKey !== undefined && object.secretKey !== null ? String(object.secretKey) : '';
    message.envVar =
      object.envVar !== undefined && object.envVar !== null ? String(object.envVar) : '';
    return message;
  },

  toJSON(message: SecretAsEnv_SecretKeyToEnvMap): unknown {
    const obj: any = {};
    message.secretKey !== undefined && (obj.secretKey = message.secretKey);
    message.envVar !== undefined && (obj.envVar = message.envVar);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SecretAsEnv_SecretKeyToEnvMap>, I>>(
    object: I,
  ): SecretAsEnv_SecretKeyToEnvMap {
    const message = { ...baseSecretAsEnv_SecretKeyToEnvMap } as SecretAsEnv_SecretKeyToEnvMap;
    message.secretKey = object.secretKey ?? '';
    message.envVar = object.envVar ?? '';
    return message;
  },
};

const baseTaskOutputParameterSpec: object = { producerTask: '', outputParameterKey: '' };

export const TaskOutputParameterSpec = {
  encode(message: TaskOutputParameterSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.producerTask !== '') {
      writer.uint32(10).string(message.producerTask);
    }
    if (message.outputParameterKey !== '') {
      writer.uint32(18).string(message.outputParameterKey);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskOutputParameterSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseTaskOutputParameterSpec } as TaskOutputParameterSpec;
    while (reader.pos < end) {
      const tag = reader.uint32();
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
  },

  fromJSON(object: any): TaskOutputParameterSpec {
    const message = { ...baseTaskOutputParameterSpec } as TaskOutputParameterSpec;
    message.producerTask =
      object.producerTask !== undefined && object.producerTask !== null
        ? String(object.producerTask)
        : '';
    message.outputParameterKey =
      object.outputParameterKey !== undefined && object.outputParameterKey !== null
        ? String(object.outputParameterKey)
        : '';
    return message;
  },

  toJSON(message: TaskOutputParameterSpec): unknown {
    const obj: any = {};
    message.producerTask !== undefined && (obj.producerTask = message.producerTask);
    message.outputParameterKey !== undefined &&
      (obj.outputParameterKey = message.outputParameterKey);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TaskOutputParameterSpec>, I>>(
    object: I,
  ): TaskOutputParameterSpec {
    const message = { ...baseTaskOutputParameterSpec } as TaskOutputParameterSpec;
    message.producerTask = object.producerTask ?? '';
    message.outputParameterKey = object.outputParameterKey ?? '';
    return message;
  },
};

const basePvcMount: object = { mountPath: '' };

export const PvcMount = {
  encode(message: PvcMount, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.taskOutputParameter !== undefined) {
      TaskOutputParameterSpec.encode(
        message.taskOutputParameter,
        writer.uint32(10).fork(),
      ).ldelim();
    }
    if (message.constant !== undefined) {
      writer.uint32(18).string(message.constant);
    }
    if (message.componentInputParameter !== undefined) {
      writer.uint32(26).string(message.componentInputParameter);
    }
    if (message.mountPath !== '') {
      writer.uint32(34).string(message.mountPath);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PvcMount {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...basePvcMount } as PvcMount;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.taskOutputParameter = TaskOutputParameterSpec.decode(reader, reader.uint32());
          break;
        case 2:
          message.constant = reader.string();
          break;
        case 3:
          message.componentInputParameter = reader.string();
          break;
        case 4:
          message.mountPath = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PvcMount {
    const message = { ...basePvcMount } as PvcMount;
    message.taskOutputParameter =
      object.taskOutputParameter !== undefined && object.taskOutputParameter !== null
        ? TaskOutputParameterSpec.fromJSON(object.taskOutputParameter)
        : undefined;
    message.constant =
      object.constant !== undefined && object.constant !== null
        ? String(object.constant)
        : undefined;
    message.componentInputParameter =
      object.componentInputParameter !== undefined && object.componentInputParameter !== null
        ? String(object.componentInputParameter)
        : undefined;
    message.mountPath =
      object.mountPath !== undefined && object.mountPath !== null ? String(object.mountPath) : '';
    return message;
  },

  toJSON(message: PvcMount): unknown {
    const obj: any = {};
    message.taskOutputParameter !== undefined &&
      (obj.taskOutputParameter = message.taskOutputParameter
        ? TaskOutputParameterSpec.toJSON(message.taskOutputParameter)
        : undefined);
    message.constant !== undefined && (obj.constant = message.constant);
    message.componentInputParameter !== undefined &&
      (obj.componentInputParameter = message.componentInputParameter);
    message.mountPath !== undefined && (obj.mountPath = message.mountPath);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PvcMount>, I>>(object: I): PvcMount {
    const message = { ...basePvcMount } as PvcMount;
    message.taskOutputParameter =
      object.taskOutputParameter !== undefined && object.taskOutputParameter !== null
        ? TaskOutputParameterSpec.fromPartial(object.taskOutputParameter)
        : undefined;
    message.constant = object.constant ?? undefined;
    message.componentInputParameter = object.componentInputParameter ?? undefined;
    message.mountPath = object.mountPath ?? '';
    return message;
  },
};

const baseCreatePvc: object = {
  accessModes: '',
  size: '',
  defaultStorageClass: false,
  storageClassName: '',
  volumeName: '',
};

export const CreatePvc = {
  encode(message: CreatePvc, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pvcName !== undefined) {
      writer.uint32(10).string(message.pvcName);
    }
    if (message.pvcNameSuffix !== undefined) {
      writer.uint32(18).string(message.pvcNameSuffix);
    }
    for (const v of message.accessModes) {
      writer.uint32(26).string(v!);
    }
    if (message.size !== '') {
      writer.uint32(34).string(message.size);
    }
    if (message.defaultStorageClass === true) {
      writer.uint32(40).bool(message.defaultStorageClass);
    }
    if (message.storageClassName !== '') {
      writer.uint32(50).string(message.storageClassName);
    }
    if (message.volumeName !== '') {
      writer.uint32(58).string(message.volumeName);
    }
    if (message.annotations !== undefined) {
      Struct.encode(Struct.wrap(message.annotations), writer.uint32(66).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreatePvc {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseCreatePvc } as CreatePvc;
    message.accessModes = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pvcName = reader.string();
          break;
        case 2:
          message.pvcNameSuffix = reader.string();
          break;
        case 3:
          message.accessModes.push(reader.string());
          break;
        case 4:
          message.size = reader.string();
          break;
        case 5:
          message.defaultStorageClass = reader.bool();
          break;
        case 6:
          message.storageClassName = reader.string();
          break;
        case 7:
          message.volumeName = reader.string();
          break;
        case 8:
          message.annotations = Struct.unwrap(Struct.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CreatePvc {
    const message = { ...baseCreatePvc } as CreatePvc;
    message.pvcName =
      object.pvcName !== undefined && object.pvcName !== null ? String(object.pvcName) : undefined;
    message.pvcNameSuffix =
      object.pvcNameSuffix !== undefined && object.pvcNameSuffix !== null
        ? String(object.pvcNameSuffix)
        : undefined;
    message.accessModes = (object.accessModes ?? []).map((e: any) => String(e));
    message.size = object.size !== undefined && object.size !== null ? String(object.size) : '';
    message.defaultStorageClass =
      object.defaultStorageClass !== undefined && object.defaultStorageClass !== null
        ? Boolean(object.defaultStorageClass)
        : false;
    message.storageClassName =
      object.storageClassName !== undefined && object.storageClassName !== null
        ? String(object.storageClassName)
        : '';
    message.volumeName =
      object.volumeName !== undefined && object.volumeName !== null
        ? String(object.volumeName)
        : '';
    message.annotations = typeof object.annotations === 'object' ? object.annotations : undefined;
    return message;
  },

  toJSON(message: CreatePvc): unknown {
    const obj: any = {};
    message.pvcName !== undefined && (obj.pvcName = message.pvcName);
    message.pvcNameSuffix !== undefined && (obj.pvcNameSuffix = message.pvcNameSuffix);
    if (message.accessModes) {
      obj.accessModes = message.accessModes.map((e) => e);
    } else {
      obj.accessModes = [];
    }
    message.size !== undefined && (obj.size = message.size);
    message.defaultStorageClass !== undefined &&
      (obj.defaultStorageClass = message.defaultStorageClass);
    message.storageClassName !== undefined && (obj.storageClassName = message.storageClassName);
    message.volumeName !== undefined && (obj.volumeName = message.volumeName);
    message.annotations !== undefined && (obj.annotations = message.annotations);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<CreatePvc>, I>>(object: I): CreatePvc {
    const message = { ...baseCreatePvc } as CreatePvc;
    message.pvcName = object.pvcName ?? undefined;
    message.pvcNameSuffix = object.pvcNameSuffix ?? undefined;
    message.accessModes = object.accessModes?.map((e) => e) || [];
    message.size = object.size ?? '';
    message.defaultStorageClass = object.defaultStorageClass ?? false;
    message.storageClassName = object.storageClassName ?? '';
    message.volumeName = object.volumeName ?? '';
    message.annotations = object.annotations ?? undefined;
    return message;
  },
};

const baseDeletePvc: object = {};

export const DeletePvc = {
  encode(message: DeletePvc, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.taskOutputParameter !== undefined) {
      TaskOutputParameterSpec.encode(
        message.taskOutputParameter,
        writer.uint32(10).fork(),
      ).ldelim();
    }
    if (message.constant !== undefined) {
      writer.uint32(18).string(message.constant);
    }
    if (message.componentInputParameter !== undefined) {
      writer.uint32(26).string(message.componentInputParameter);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeletePvc {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseDeletePvc } as DeletePvc;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.taskOutputParameter = TaskOutputParameterSpec.decode(reader, reader.uint32());
          break;
        case 2:
          message.constant = reader.string();
          break;
        case 3:
          message.componentInputParameter = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DeletePvc {
    const message = { ...baseDeletePvc } as DeletePvc;
    message.taskOutputParameter =
      object.taskOutputParameter !== undefined && object.taskOutputParameter !== null
        ? TaskOutputParameterSpec.fromJSON(object.taskOutputParameter)
        : undefined;
    message.constant =
      object.constant !== undefined && object.constant !== null
        ? String(object.constant)
        : undefined;
    message.componentInputParameter =
      object.componentInputParameter !== undefined && object.componentInputParameter !== null
        ? String(object.componentInputParameter)
        : undefined;
    return message;
  },

  toJSON(message: DeletePvc): unknown {
    const obj: any = {};
    message.taskOutputParameter !== undefined &&
      (obj.taskOutputParameter = message.taskOutputParameter
        ? TaskOutputParameterSpec.toJSON(message.taskOutputParameter)
        : undefined);
    message.constant !== undefined && (obj.constant = message.constant);
    message.componentInputParameter !== undefined &&
      (obj.componentInputParameter = message.componentInputParameter);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DeletePvc>, I>>(object: I): DeletePvc {
    const message = { ...baseDeletePvc } as DeletePvc;
    message.taskOutputParameter =
      object.taskOutputParameter !== undefined && object.taskOutputParameter !== null
        ? TaskOutputParameterSpec.fromPartial(object.taskOutputParameter)
        : undefined;
    message.constant = object.constant ?? undefined;
    message.componentInputParameter = object.componentInputParameter ?? undefined;
    return message;
  },
};

const baseNodeSelector: object = {};

export const NodeSelector = {
  encode(message: NodeSelector, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.labels).forEach(([key, value]) => {
      NodeSelector_LabelsEntry.encode(
        { key: key as any, value },
        writer.uint32(10).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): NodeSelector {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseNodeSelector } as NodeSelector;
    message.labels = {};
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          const entry1 = NodeSelector_LabelsEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.labels[entry1.key] = entry1.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): NodeSelector {
    const message = { ...baseNodeSelector } as NodeSelector;
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>(
      (acc, [key, value]) => {
        acc[key] = String(value);
        return acc;
      },
      {},
    );
    return message;
  },

  toJSON(message: NodeSelector): unknown {
    const obj: any = {};
    obj.labels = {};
    if (message.labels) {
      Object.entries(message.labels).forEach(([k, v]) => {
        obj.labels[k] = v;
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<NodeSelector>, I>>(object: I): NodeSelector {
    const message = { ...baseNodeSelector } as NodeSelector;
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = String(value);
        }
        return acc;
      },
      {},
    );
    return message;
  },
};

const baseNodeSelector_LabelsEntry: object = { key: '', value: '' };

export const NodeSelector_LabelsEntry = {
  encode(message: NodeSelector_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== '') {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== '') {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): NodeSelector_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseNodeSelector_LabelsEntry } as NodeSelector_LabelsEntry;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): NodeSelector_LabelsEntry {
    const message = { ...baseNodeSelector_LabelsEntry } as NodeSelector_LabelsEntry;
    message.key = object.key !== undefined && object.key !== null ? String(object.key) : '';
    message.value = object.value !== undefined && object.value !== null ? String(object.value) : '';
    return message;
  },

  toJSON(message: NodeSelector_LabelsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<NodeSelector_LabelsEntry>, I>>(
    object: I,
  ): NodeSelector_LabelsEntry {
    const message = { ...baseNodeSelector_LabelsEntry } as NodeSelector_LabelsEntry;
    message.key = object.key ?? '';
    message.value = object.value ?? '';
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPartial<U>>
  : T extends {}
  ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin
  ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & Record<Exclude<keyof I, KeysOfUnion<P>>, never>;

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}
