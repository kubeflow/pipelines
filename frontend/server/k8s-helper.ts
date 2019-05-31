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

// @ts-ignore
import {Core_v1Api, Custom_objectsApi, KubeConfig} from '@kubernetes/client-node';
import * as crypto from 'crypto-js';
import * as fs from 'fs';
import * as Utils from './utils';

// If this is running inside a k8s Pod, its namespace should be written at this
// path, this is also how we can tell whether we're running in the cluster.
const namespaceFilePath = '/var/run/secrets/kubernetes.io/serviceaccount/namespace';
let namespace = '';
let k8sV1Client: Core_v1Api | null = null;
let k8sV1CustomObjectClient: Custom_objectsApi | null = null;

// Constants for creating customer resource Viewer.
const viewerGroup = 'kubeflow.org';
const viewerVersion = 'v1beta1';
const viewerPlural = 'viewers';

export const isInCluster = fs.existsSync(namespaceFilePath);

if (isInCluster) {
  namespace = fs.readFileSync(namespaceFilePath, 'utf-8');
  const kc = new KubeConfig();
  kc.loadFromDefault();
  k8sV1Client = kc.makeApiClient(Core_v1Api);
  k8sV1CustomObjectClient = kc.makeApiClient(Custom_objectsApi);
}

/**
 * Creates a new Tensorboard pod.
 */
export async function newTensorboardPod(logdir: string): Promise<void> {
  if (!k8sV1Client) {
    throw new Error('Cannot access kubernetes API');
  }
  const currentPod = await getTensorboardAddress(logdir);
  if (currentPod) {
    return;
  }

  // TODO: take the configuration below to a separate file
  const pod = {
    kind: 'Pod',
    metadata: {
      generateName: 'tensorboard-',
    },
    spec: {
      containers: [{
        args: [
          'tensorboard',
          '--logdir',
          logdir,
        ],
        image: 'tensorflow/tensorflow',
        name: 'tensorflow',
        ports: [{
          containerPort: 6006,
        }],
        env: [{
          name: 'GOOGLE_APPLICATION_CREDENTIALS',
          value: '/secret/gcp-credentials/user-gcp-sa.json'
        }],
        volumeMounts: [{
          mountPath: '/secret/gcp-credentials',
          name: 'gcp-credentials',
        }],
      }],
      volumes: [{
        name: 'gcp-credentials',
        secret: {
          secretName: 'user-gcp-sa',
        },
      }],
    },
  };

  await k8sV1Client.createNamespacedPod(namespace, pod as any);
}

/**
 * Finds a running Tensorboard pod with the given logdir as an argument and
 * returns its pod IP and port.
 */
export async function getTensorboardAddress(logdir: string): Promise<string> {
  if (!k8sV1Client) {
    throw new Error('Cannot access kubernetes API');
  }
  const pods = (await k8sV1Client.listNamespacedPod(namespace)).body.items;
  const args = ['tensorboard', '--logdir', logdir];
  const pod = pods.find((p) =>
    p.status.phase === 'Running' &&
    !p.metadata.deletionTimestamp && // Terminating/terminated pods have this set
    !!p.spec.containers.find((c) => Utils.equalArrays(c.args, args)));
  return pod && pod.status.podIP ? `http://${pod.status.podIP}:6006` : '';
}

/**
 * Polls every second for a running Tensorboard pod with the given logdir, and
 * returns the address of one if found, or rejects if a timeout expires.
 */
export function waitForTensorboard(logdir: string, timeout: number): Promise<string> {
  const start = Date.now();
  return new Promise((resolve, reject) => {
    setInterval(async () => {
      if (Date.now() - start > timeout) {
        reject('Timed out waiting for tensorboard');
      }
      const tensorboardAddress = await getTensorboardAddress(logdir);
      if (tensorboardAddress) {
        resolve(encodeURIComponent(tensorboardAddress));
      }
    }, 1000);
  });
}

export function getPodLogs(podName: string): Promise<string> {
  if (!k8sV1Client) {
    throw new Error('Cannot access kubernetes API');
  }
  return (k8sV1Client.readNamespacedPodLog(podName, namespace, 'main') as any)
    .then(
      (response: any) => (response && response.body) ? response.body.toString() : '',
      (error: any) => {throw new Error(JSON.stringify(error.body));}
    );
}

function getNameOfViewerResource(logdir: string): string {
  return 'viewer-' + crypto.SHA1(logdir) + '-v1';
}

/**
* Create Tensorboard pod via CRD with the given logdir if there is no existing
* Tensorboard pod.
*/
export async function newTensorboardInstance(logdir: string): Promise<void> {
  if (!k8sV1CustomObjectClient) {
    throw new Error('Cannot access kubernetes Custom Object API');
  }
  const currentPod = await getTensorboardInstance(logdir);
  if (currentPod) {
    return;
  }

  // TODO: take the configuration below to a separate file
  // Name of the viewer resource is based on logDir.
  const body = {
    apiVersion: viewerGroup + '/' + viewerVersion,
    kind: 'Viewer',
    metadata: {
      name: getNameOfViewerResource(logdir),
      namespace: namespace,
    },
    spec: {
      type: 'tensorboard',
      tensorboardSpec: {
        logDir: logdir,
      },
      PodTemplateSpec: {
        spec: {
          containers: [{
            env: [{
              name: 'GOOGLE_APPLICATION_CREDENTIALS',
              value: '/secret/gcp-credentials/user-gcp-sa.json'
            }],
            volumeMounts: [{
              mountPath: '/secret/gcp-credentials',
              name: 'gcp-credentials',
            }],
          }],
          volumes: [{
            name: 'gcp-credentials',
            volumeSource: {
              secret: {
                secretName: 'user-gcp-sa',
              }
            },
          }],
        }
      }
    }
  };
  await k8sV1CustomObjectClient.createNamespacedCustomObject(viewerGroup,
    viewerVersion, namespace, viewerPlural, body);
}

/**
* Finds a running Tensorboard pod created via CRD with the give logdir and
* returns its pod IP and port.
*/
export async function getTensorboardInstance(logdir: string): Promise<string> {
  if (!k8sV1CustomObjectClient) {
    throw new Error('Cannot access kubernetes Custom Object API');
  }

  const viewers = (await k8sV1CustomObjectClient.listNamespacedCustomObject(
    viewerGroup, viewerVersion, namespace, viewerPlural)).body.items;
  const viewer = viewers.find((v) =>
    v.metadata.name == getNameOfViewerResource(logdir) &&
    v.spec.tensorboardSpec.logDir == logdir &&
    v.spec.type == 'tensorboard');
  return viewer ? `http://${viewer.metadata.name}-service.kubeflow.svc.cluster.local:6006` : '';
}

/**
 * Polls every second for a running Tensorboard pod with the given logdir, and
 * returns the address of one if found, or rejects if a timeout expires.
 */
export function waitForTensorboardInstance(logdir: string, timeout: number): Promise<string> {
  const start = Date.now();
  return new Promise((resolve, reject) => {
    setInterval(async () => {
      if (Date.now() - start > timeout) {
        reject('Timed out waiting for tensorboard');
      }
      const tensorboardAddress = await getTensorboardInstance(logdir);
      if (tensorboardAddress) {
        resolve(encodeURIComponent(tensorboardAddress));
      }
    }, 1000);
  });
}
