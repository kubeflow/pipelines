import * as fs from 'fs';
import { Config } from '@kubernetes/client-node';
import * as Utils from './utils';

// If this is running inside a k8s Pod, its namespace should be written at this
// path, this is also how we can tell whether we're running in the cluster.
const namespaceFilePath = '/var/run/secrets/kubernetes.io/serviceaccount/namespace';
let namespace = null;
let k8sV1Client = null;

export const isInCluster = fs.existsSync(namespaceFilePath);

if (isInCluster) {
  namespace = fs.readFileSync(namespaceFilePath);
  k8sV1Client = Config.defaultClient();
}

/**
 * Creates a new Tensorboard pod.
 */
export async function newTensorboardPod(logdir: string): Promise<string> {
  const currentPod = await getTensorboardAddress(logdir);
  if (currentPod) {
    return currentPod;
  }

  const podName = 'tensorboard-' + Utils.generateRandomString(15);

  // TODO: take the configuration below to a separate file
  const pod = {
    kind: 'Pod',
    metadata: {
      name: podName,
    },
    spec: {
      selector: {
        app: podName,
      },
      containers: [{
        args: [
          'tensorboard',
          '--logdir',
          logdir,
        ],
        name: 'tensorflow',
        image: 'tensorflow/tensorflow',
        ports: [{
          containerPort: 6006,
        }],
      }],
    },
  };

  await k8sV1Client.createNamespacedPod(namespace, pod);
}

/**
 * Finds a running Tensorboard pod with the given logdir as an argument and
 * returns its pod IP and port.
 */
export async function getTensorboardAddress(logdir: string) {
  const pods = (await k8sV1Client.listNamespacedPod(namespace)).body.items;
  const args = ['tensorboard', '--logdir', logdir];
  const pod = pods.find(p =>
    p.status.phase === 'Running' &&
    !p.metadata.deletionTimestamp && // Terminating/terminated pods have this set
    p.spec.containers.find(
      c => Utils.equalArrays(c.args, args)));
  return pod && pod.status.podIP ? `http://${pod.status.podIP}:6006` : '';
}

/**
 * Polls every second for a running Tensorboard pod with the given logdir, and
 * returns the address of one if found, or rejects if a timeout expires.
 */
export function waitForTensorboard(logdir: string, timeout: number) {
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
  })
}

