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

function getNameOfViewerResource(logdir: string): string {
  // TODO: find some hash function with shorter resulting message.
  return 'viewer-' + crypto.SHA1(logdir);
}

/**
 * Create Tensorboard instance via CRD with the given logdir if there is no
 * existing Tensorboard instance.
 */
export async function newTensorboardInstance(logdir: string): Promise<void> {
  if (!k8sV1CustomObjectClient) {
    throw new Error('Cannot access kubernetes Custom Object API');
  }
  const currentPod = await getTensorboardInstance(logdir);
  if (currentPod) {
    return;
  }

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
    }
  };
  await k8sV1CustomObjectClient.createNamespacedCustomObject(viewerGroup,
    viewerVersion, namespace, viewerPlural, body);
}

/**
 * Finds a running Tensorboard instance created via CRD with the given logdir
 * and returns its dns address.
 */
export async function getTensorboardInstance(logdir: string): Promise<string> {
  if (!k8sV1CustomObjectClient) {
    throw new Error('Cannot access kubernetes Custom Object API');
  }

  return await (k8sV1CustomObjectClient.getNamespacedCustomObject(
    viewerGroup, viewerVersion, namespace, viewerPlural,
    getNameOfViewerResource(logdir))).then(
      // Viewer CRD pod has tensorboard instance running at port 6006 while
      // viewer CRD service has tensorboard instance running at port 80. Since
      // we return service address here (instead of pod address), so use 80.
      (viewer: any) => (
        viewer && viewer.body &&
        viewer.body.spec.tensorboardSpec.logDir == logdir &&
        viewer.body.spec.type == 'tensorboard') ?
          `http://${viewer.body.metadata.name}-service.${namespace}.svc.cluster.local:80/tensorboard/${viewer.body.metadata.name}/` : '',
      // No existing custom object with the given name, i.e., no existing
      // tensorboard instance.
      (error: any) => ''
    );
}

/**
 * Polls every second for a running Tensorboard instance with the given logdir,
 * and returns the address of one if found, or rejects if a timeout expires.
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