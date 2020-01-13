// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

export type Time = string;
export type VolumeDevice = any;
export type Volume = any;
export type EnvFromSource = any;
export type EnvVarSource = any;
export type ResourceRequirements = any;
export type Probe = any;
export type Lifecycle = any;
export type TerminationMessagePolicy = any;
export type PullPolicy = any;
export type SecurityContext = any;
export type PersistentVolumeClaim = any;
export type Affinity = any;

export interface VolumeMount {
  name: string;
  mountPath?: string;
}

export interface ListMeta {
  _continue?: string;
  resourceVersion?: string;
  selfLink?: string;
}

export interface ObjectMeta {
  name?: string;
  generateName?: string;
  namespace?: string;
  selfLink?: string;
  uid?: string;
  resourceVersion?: string;
  generation?: number;
  creationTimestamp?: Time;
  deletionTimestamp?: Time;
  deletionGracePeriodSeconds?: number;
  labels?: { [name: string]: string };
  annotations?: { [name: string]: string };
  ownerReferences?: any[];
  initializers?: any;
  finalizers?: string[];
  clusterName?: string;
}

export interface TypeMeta {
  kind: string;
  apiVersion: string;
}

export interface LocalObjectReference {
  name: string;
}

export interface SecretKeySelector extends LocalObjectReference {
  key: string;
  optional: boolean;
}

export interface ContainerPort {
  name: string;
  hostPort: number;
  containerPort: number;
  protocol: string;
  hostIP: string;
}

export interface EnvVar {
  name: string;
  value: string;
  valueFrom: EnvVarSource;
}

export interface Container {
  name: string;
  image: string;
  command: string[];
  args: string[];
  workingDir: string;
  ports: ContainerPort[];
  envFrom: EnvFromSource[];
  env: EnvVar[];
  resources: ResourceRequirements;
  volumeMounts: VolumeMount[];
  livenessProbe: Probe;
  readinessProbe: Probe;
  lifecycle: Lifecycle;
  terminationMessagePath: string;
  terminationMessagePolicy: TerminationMessagePolicy;
  imagePullPolicy: PullPolicy;
  securityContext: SecurityContext;
  stdin: boolean;
  stdinOnce: boolean;
  tty: boolean;
}

export interface WatchEvent<T> {
  object: T;
  type: 'ADDED' | 'MODIFIED' | 'DELETED' | 'ERROR';
}
