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

// TODO: refractor k8s helper module so that api that interact with k8s can be
// mocked and tested. There is currently no way to mock k8s APIs as
// `k8s-helper.isInCluster` is a constant that is generated when the module is
// first loaded.

// @ts-ignore
import {
  Core_v1Api,
  Custom_objectsApi,
  KubeConfig,
  V1DeleteOptions,
} from '@kubernetes/client-node';
import * as crypto from 'crypto-js';
import * as fs from 'fs';
import { PartialArgoWorkflow } from './workflow-helper';

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

// Constants for argo workflow
const workflowGroup = 'argoproj.io';
const workflowVersion = 'v1alpha1';
const workflowPlural = 'workflows';

/** Default pod template spec used to create tensorboard viewer. */
export const defaultPodTemplateSpec = {
  spec: {
    containers: [{}],
  },
};

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
export async function newTensorboardInstance(
  logdir: string,
  tfversion: string,
  podTemplateSpec: object = defaultPodTemplateSpec,
): Promise<void> {
  if (!k8sV1CustomObjectClient) {
    throw new Error('Cannot access kubernetes Custom Object API');
  }
  const currentPod = await getTensorboardInstance(logdir);
  if (currentPod.podAddress) {
    if (tfversion === currentPod.tfVersion) {
      return;
    } else {
      throw new Error(
        `There's already an existing tensorboard instance with a different version ${currentPod.tfVersion}`,
      );
    }
  }

  const body = {
    apiVersion: viewerGroup + '/' + viewerVersion,
    kind: 'Viewer',
    metadata: {
      name: getNameOfViewerResource(logdir),
      namespace,
    },
    spec: {
      podTemplateSpec,
      tensorboardSpec: {
        logDir: logdir,
        tensorflowImage: 'tensorflow/tensorflow:' + tfversion,
      },
      type: 'tensorboard',
    },
  };
  await k8sV1CustomObjectClient.createNamespacedCustomObject(
    viewerGroup,
    viewerVersion,
    namespace,
    viewerPlural,
    body,
  );
}

/**
 * Finds a running Tensorboard instance created via CRD with the given logdir
 * and returns its dns address and its version
 */
export async function getTensorboardInstance(
  logdir: string,
): Promise<{ podAddress: string; tfVersion: string }> {
  if (!k8sV1CustomObjectClient) {
    throw new Error('Cannot access kubernetes Custom Object API');
  }

  return await k8sV1CustomObjectClient
    .getNamespacedCustomObject(
      viewerGroup,
      viewerVersion,
      namespace,
      viewerPlural,
      getNameOfViewerResource(logdir),
    )
    .then(
      // Viewer CRD pod has tensorboard instance running at port 6006 while
      // viewer CRD service has tensorboard instance running at port 80. Since
      // we return service address here (instead of pod address), so use 80.
      (viewer: any) => {
        if (
          viewer &&
          viewer.body &&
          viewer.body.spec.tensorboardSpec.logDir === logdir &&
          viewer.body.spec.type === 'tensorboard'
        ) {
          const address = `http://${viewer.body.metadata.name}-service.${namespace}.svc.cluster.local:80/tensorboard/${viewer.body.metadata.name}/`;
          const version = viewer.body.spec.tensorboardSpec.tensorflowImage
            ? viewer.body.spec.tensorboardSpec.tensorflowImage.replace('tensorflow/tensorflow:', '')
            : '';
          return { podAddress: address, tfVersion: version };
        } else {
          return { podAddress: '', tfVersion: '' };
        }
      },
      // No existing custom object with the given name, i.e., no existing
      // tensorboard instance.
      (_: any) => {
        return { podAddress: '', tfVersion: '' };
      },
    );
}

/**
 * Find a running Tensorboard instance with the given logdir, delete the instance
 * and returns the deleted podAddress
 */

export async function deleteTensorboardInstance(logdir: string): Promise<void> {
  if (!k8sV1CustomObjectClient) {
    throw new Error('Cannot access kubernetes Custom Object API');
  }
  const currentPod = await getTensorboardInstance(logdir);
  if (!currentPod.podAddress) {
    return;
  }

  const viewerName = getNameOfViewerResource(logdir);
  const deleteOption = new V1DeleteOptions();

  await k8sV1CustomObjectClient.deleteNamespacedCustomObject(
    viewerGroup,
    viewerVersion,
    namespace,
    viewerPlural,
    viewerName,
    deleteOption,
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
      const tensorboardInstance = await getTensorboardInstance(logdir);
      const tensorboardAddress = tensorboardInstance.podAddress;
      if (tensorboardAddress) {
        resolve(encodeURIComponent(tensorboardAddress));
      }
    }, 1000);
  });
}

export function getPodLogs(podName: string, podNamespace?: string): Promise<string> {
  if (!k8sV1Client) {
    throw new Error('Cannot access kubernetes API');
  }
  return (k8sV1Client.readNamespacedPodLog(podName, podNamespace || namespace, 'main') as any).then(
    (response: any) => (response && response.body ? response.body.toString() : ''),
    (error: any) => {
      throw new Error(JSON.stringify(error.body));
    },
  );
}

/**
 * Retrieves the argo workflow CRD.
 * @param workflowName name of the argo workflow
 */
export async function getArgoWorkflow(workflowName: string): Promise<PartialArgoWorkflow> {
  if (!k8sV1CustomObjectClient) {
    throw new Error('Cannot access kubernetes Custom Object API');
  }

  const res = await k8sV1CustomObjectClient.getNamespacedCustomObject(
    workflowGroup,
    workflowVersion,
    namespace,
    workflowPlural,
    workflowName,
  );

  if (res.response.statusCode == null) {
    throw new Error(`Unable to query workflow:${workflowName}: No status code present.`);
  }

  if (res.response.statusCode >= 400) {
    throw new Error(`Unable to query workflow:${workflowName}: Access denied.`);
  }
  return res.body;
}

/**
 * Retrieves k8s secret by key and decode from base64.
 * @param name name of the secret
 * @param key key in the secret
 */
export async function getK8sSecret(name: string, key: string) {
  if (!k8sV1Client) {
    throw new Error('Cannot access kubernetes API');
  }

  const k8sSecret = await k8sV1Client.readNamespacedSecret(name, namespace);
  const secretb64 = k8sSecret.body.data[key];
  const buff = new Buffer(secretb64, 'base64');
  return buff.toString('ascii');
}
