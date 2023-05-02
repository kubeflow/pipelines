import * as jspb from 'google-protobuf'

import * as google_protobuf_struct_pb from 'google-protobuf/google/protobuf/struct_pb';


export class KubernetesExecutorConfig extends jspb.Message {
  getSecretAsVolumeList(): Array<SecretAsVolume>;
  setSecretAsVolumeList(value: Array<SecretAsVolume>): KubernetesExecutorConfig;
  clearSecretAsVolumeList(): KubernetesExecutorConfig;
  addSecretAsVolume(value?: SecretAsVolume, index?: number): SecretAsVolume;

  getSecretAsEnvList(): Array<SecretAsEnv>;
  setSecretAsEnvList(value: Array<SecretAsEnv>): KubernetesExecutorConfig;
  clearSecretAsEnvList(): KubernetesExecutorConfig;
  addSecretAsEnv(value?: SecretAsEnv, index?: number): SecretAsEnv;

  getPvcMountList(): Array<PvcMount>;
  setPvcMountList(value: Array<PvcMount>): KubernetesExecutorConfig;
  clearPvcMountList(): KubernetesExecutorConfig;
  addPvcMount(value?: PvcMount, index?: number): PvcMount;

  getNodeSelector(): NodeSelector | undefined;
  setNodeSelector(value?: NodeSelector): KubernetesExecutorConfig;
  hasNodeSelector(): boolean;
  clearNodeSelector(): KubernetesExecutorConfig;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): KubernetesExecutorConfig.AsObject;
  static toObject(includeInstance: boolean, msg: KubernetesExecutorConfig): KubernetesExecutorConfig.AsObject;
  static serializeBinaryToWriter(message: KubernetesExecutorConfig, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): KubernetesExecutorConfig;
  static deserializeBinaryFromReader(message: KubernetesExecutorConfig, reader: jspb.BinaryReader): KubernetesExecutorConfig;
}

export namespace KubernetesExecutorConfig {
  export type AsObject = {
    secretAsVolumeList: Array<SecretAsVolume.AsObject>,
    secretAsEnvList: Array<SecretAsEnv.AsObject>,
    pvcMountList: Array<PvcMount.AsObject>,
    nodeSelector?: NodeSelector.AsObject,
  }
}

export class SecretAsVolume extends jspb.Message {
  getSecretName(): string;
  setSecretName(value: string): SecretAsVolume;

  getMountPath(): string;
  setMountPath(value: string): SecretAsVolume;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SecretAsVolume.AsObject;
  static toObject(includeInstance: boolean, msg: SecretAsVolume): SecretAsVolume.AsObject;
  static serializeBinaryToWriter(message: SecretAsVolume, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SecretAsVolume;
  static deserializeBinaryFromReader(message: SecretAsVolume, reader: jspb.BinaryReader): SecretAsVolume;
}

export namespace SecretAsVolume {
  export type AsObject = {
    secretName: string,
    mountPath: string,
  }
}

export class SecretAsEnv extends jspb.Message {
  getSecretName(): string;
  setSecretName(value: string): SecretAsEnv;

  getKeyToEnvList(): Array<SecretAsEnv.SecretKeyToEnvMap>;
  setKeyToEnvList(value: Array<SecretAsEnv.SecretKeyToEnvMap>): SecretAsEnv;
  clearKeyToEnvList(): SecretAsEnv;
  addKeyToEnv(value?: SecretAsEnv.SecretKeyToEnvMap, index?: number): SecretAsEnv.SecretKeyToEnvMap;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SecretAsEnv.AsObject;
  static toObject(includeInstance: boolean, msg: SecretAsEnv): SecretAsEnv.AsObject;
  static serializeBinaryToWriter(message: SecretAsEnv, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SecretAsEnv;
  static deserializeBinaryFromReader(message: SecretAsEnv, reader: jspb.BinaryReader): SecretAsEnv;
}

export namespace SecretAsEnv {
  export type AsObject = {
    secretName: string,
    keyToEnvList: Array<SecretAsEnv.SecretKeyToEnvMap.AsObject>,
  }

  export class SecretKeyToEnvMap extends jspb.Message {
    getSecretKey(): string;
    setSecretKey(value: string): SecretKeyToEnvMap;

    getEnvVar(): string;
    setEnvVar(value: string): SecretKeyToEnvMap;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SecretKeyToEnvMap.AsObject;
    static toObject(includeInstance: boolean, msg: SecretKeyToEnvMap): SecretKeyToEnvMap.AsObject;
    static serializeBinaryToWriter(message: SecretKeyToEnvMap, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SecretKeyToEnvMap;
    static deserializeBinaryFromReader(message: SecretKeyToEnvMap, reader: jspb.BinaryReader): SecretKeyToEnvMap;
  }

  export namespace SecretKeyToEnvMap {
    export type AsObject = {
      secretKey: string,
      envVar: string,
    }
  }

}

export class TaskOutputParameterSpec extends jspb.Message {
  getProducerTask(): string;
  setProducerTask(value: string): TaskOutputParameterSpec;

  getOutputParameterKey(): string;
  setOutputParameterKey(value: string): TaskOutputParameterSpec;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TaskOutputParameterSpec.AsObject;
  static toObject(includeInstance: boolean, msg: TaskOutputParameterSpec): TaskOutputParameterSpec.AsObject;
  static serializeBinaryToWriter(message: TaskOutputParameterSpec, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TaskOutputParameterSpec;
  static deserializeBinaryFromReader(message: TaskOutputParameterSpec, reader: jspb.BinaryReader): TaskOutputParameterSpec;
}

export namespace TaskOutputParameterSpec {
  export type AsObject = {
    producerTask: string,
    outputParameterKey: string,
  }
}

export class PvcMount extends jspb.Message {
  getTaskOutputParameter(): TaskOutputParameterSpec | undefined;
  setTaskOutputParameter(value?: TaskOutputParameterSpec): PvcMount;
  hasTaskOutputParameter(): boolean;
  clearTaskOutputParameter(): PvcMount;

  getConstant(): string;
  setConstant(value: string): PvcMount;

  getComponentInputParameter(): string;
  setComponentInputParameter(value: string): PvcMount;

  getMountPath(): string;
  setMountPath(value: string): PvcMount;

  getPvcReferenceCase(): PvcMount.PvcReferenceCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PvcMount.AsObject;
  static toObject(includeInstance: boolean, msg: PvcMount): PvcMount.AsObject;
  static serializeBinaryToWriter(message: PvcMount, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PvcMount;
  static deserializeBinaryFromReader(message: PvcMount, reader: jspb.BinaryReader): PvcMount;
}

export namespace PvcMount {
  export type AsObject = {
    taskOutputParameter?: TaskOutputParameterSpec.AsObject,
    constant: string,
    componentInputParameter: string,
    mountPath: string,
  }

  export enum PvcReferenceCase { 
    PVC_REFERENCE_NOT_SET = 0,
    TASK_OUTPUT_PARAMETER = 1,
    CONSTANT = 2,
    COMPONENT_INPUT_PARAMETER = 3,
  }
}

export class CreatePvc extends jspb.Message {
  getPvcName(): string;
  setPvcName(value: string): CreatePvc;

  getPvcNameSuffix(): string;
  setPvcNameSuffix(value: string): CreatePvc;

  getAccessModesList(): Array<string>;
  setAccessModesList(value: Array<string>): CreatePvc;
  clearAccessModesList(): CreatePvc;
  addAccessModes(value: string, index?: number): CreatePvc;

  getSize(): string;
  setSize(value: string): CreatePvc;

  getDefaultStorageClass(): boolean;
  setDefaultStorageClass(value: boolean): CreatePvc;

  getStorageClassName(): string;
  setStorageClassName(value: string): CreatePvc;

  getVolumeName(): string;
  setVolumeName(value: string): CreatePvc;

  getAnnotations(): google_protobuf_struct_pb.Struct | undefined;
  setAnnotations(value?: google_protobuf_struct_pb.Struct): CreatePvc;
  hasAnnotations(): boolean;
  clearAnnotations(): CreatePvc;

  getNameCase(): CreatePvc.NameCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreatePvc.AsObject;
  static toObject(includeInstance: boolean, msg: CreatePvc): CreatePvc.AsObject;
  static serializeBinaryToWriter(message: CreatePvc, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreatePvc;
  static deserializeBinaryFromReader(message: CreatePvc, reader: jspb.BinaryReader): CreatePvc;
}

export namespace CreatePvc {
  export type AsObject = {
    pvcName: string,
    pvcNameSuffix: string,
    accessModesList: Array<string>,
    size: string,
    defaultStorageClass: boolean,
    storageClassName: string,
    volumeName: string,
    annotations?: google_protobuf_struct_pb.Struct.AsObject,
  }

  export enum NameCase { 
    NAME_NOT_SET = 0,
    PVC_NAME = 1,
    PVC_NAME_SUFFIX = 2,
  }
}

export class DeletePvc extends jspb.Message {
  getTaskOutputParameter(): TaskOutputParameterSpec | undefined;
  setTaskOutputParameter(value?: TaskOutputParameterSpec): DeletePvc;
  hasTaskOutputParameter(): boolean;
  clearTaskOutputParameter(): DeletePvc;

  getConstant(): string;
  setConstant(value: string): DeletePvc;

  getComponentInputParameter(): string;
  setComponentInputParameter(value: string): DeletePvc;

  getPvcReferenceCase(): DeletePvc.PvcReferenceCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeletePvc.AsObject;
  static toObject(includeInstance: boolean, msg: DeletePvc): DeletePvc.AsObject;
  static serializeBinaryToWriter(message: DeletePvc, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeletePvc;
  static deserializeBinaryFromReader(message: DeletePvc, reader: jspb.BinaryReader): DeletePvc;
}

export namespace DeletePvc {
  export type AsObject = {
    taskOutputParameter?: TaskOutputParameterSpec.AsObject,
    constant: string,
    componentInputParameter: string,
  }

  export enum PvcReferenceCase { 
    PVC_REFERENCE_NOT_SET = 0,
    TASK_OUTPUT_PARAMETER = 1,
    CONSTANT = 2,
    COMPONENT_INPUT_PARAMETER = 3,
  }
}

export class NodeSelector extends jspb.Message {
  getLabelsMap(): jspb.Map<string, string>;
  clearLabelsMap(): NodeSelector;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NodeSelector.AsObject;
  static toObject(includeInstance: boolean, msg: NodeSelector): NodeSelector.AsObject;
  static serializeBinaryToWriter(message: NodeSelector, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NodeSelector;
  static deserializeBinaryFromReader(message: NodeSelector, reader: jspb.BinaryReader): NodeSelector;
}

export namespace NodeSelector {
  export type AsObject = {
    labelsMap: Array<[string, string]>,
  }
}

