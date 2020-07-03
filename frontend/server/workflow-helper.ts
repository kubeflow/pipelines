// Copyright 2019 Google LLC
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
import path from 'path';
import { PassThrough, Stream } from 'stream';
import { ClientOptions as MinioClientOptions } from 'minio';
import { getK8sSecret, getArgoWorkflow, getPodLogs } from './k8s-helper';
import { createMinioClient, MinioRequestConfig, getObjectStream } from './minio-helper';

export interface PartialArgoWorkflow {
  status: {
    nodes?: ArgoWorkflowStatusNode;
  };
}

export interface ArgoWorkflowStatusNode {
  [key: string]: ArgoWorkflowStatusNodeInfo;
}

export interface ArgoWorkflowStatusNodeInfo {
  outputs?: {
    artifacts?: ArtifactRecord[];
  };
}

export interface ArtifactRecord {
  archiveLogs?: boolean;
  name: string;
  s3?: S3Artifact;
}

export interface S3Artifact {
  accessKeySecret?: SecretSelector;
  bucket: string;
  endpoint: string;
  insecure: boolean;
  key: string;
  secretKeySecret?: SecretSelector;
}

export interface SecretSelector {
  key: string;
  name: string;
}

/**
 * Compose a pod logs stream handler - i.e. a stream handler returns a stream
 * containing the pod logs.
 * @param handler a function that returns a stream.
 * @param fallback a fallback function that returns a stream if the initial handler
 * fails.
 */
export function composePodLogsStreamHandler<T = Stream>(
  handler: (podName: string, namespace?: string) => Promise<T>,
  fallback?: (podName: string, namespace?: string) => Promise<T>,
) {
  return async (podName: string, namespace?: string) => {
    try {
      return await handler(podName, namespace);
    } catch (err) {
      if (fallback) {
        return await fallback(podName, namespace);
      }
      console.warn(err);
      throw err;
    }
  };
}

/**
 * Returns a stream containing the pod logs using kubernetes api.
 * @param podName name of the pod.
 * @param namespace namespace of the pod (uses the same namespace as the server if not provided).
 */
export async function getPodLogsStreamFromK8s(podName: string, namespace?: string) {
  const stream = new PassThrough();
  stream.end(await getPodLogs(podName, namespace));
  console.log(`Getting logs for pod:${podName} in namespace ${namespace}.`);
  return stream;
}

/**
 * Returns a stream containing the pod logs using the information provided in the
 * workflow status (uses k8s api to retrieve the workflow and secrets).
 * @param podName name of the pod.
 * @param namespace namespace of the pod (uses the same namespace as the server if not provided).
 */
export const getPodLogsStreamFromWorkflow = toGetPodLogsStream(
  getPodLogsMinioRequestConfigfromWorkflow,
);

/**
 * Returns a function that retrieves the pod log streams using the provided
 * getMinioRequestConfig function (a MinioRequestConfig object specifies the
 * artifact bucket and key, with the corresponding minio client).
 * @param getMinioRequestConfig function that returns a MinioRequestConfig based
 * on the provided pod name and namespace (optional).
 */
export function toGetPodLogsStream(
  getMinioRequestConfig: (podName: string, namespace?: string) => Promise<MinioRequestConfig>,
) {
  return async (podName: string, namespace?: string) => {
    const request = await getMinioRequestConfig(podName, namespace);
    console.log(`Getting logs for pod:${podName} from ${request.bucket}/${request.key}.`);
    return await getObjectStream(request);
  };
}

/**
 * Returns a MinioRequestConfig with the provided minio options (a MinioRequestConfig
 * object contains the artifact bucket and keys, with the corresponding minio
 * client).
 * @param minioOptions Minio options to create a minio client.
 * @param bucket bucket containing the pod logs artifacts.
 * @param prefix prefix for pod logs artifacts stored in the bucket.
 */
export function createPodLogsMinioRequestConfig(
  minioOptions: MinioClientOptions,
  bucket: string,
  prefix: string,
) {
  // TODO: support pod log artifacts for diff namespace.
  // different bucket/prefix for diff namespace?
  return async (podName: string, _namespace?: string): Promise<MinioRequestConfig> => {
    // create a new client each time to ensure session token has not expired
    const client = await createMinioClient(minioOptions);
    const workflowName = workflowNameFromPodName(podName);
    return {
      bucket,
      client,
      key: path.join(prefix, workflowName, podName, 'main.log'),
    };
  };
}

/**
 * Retrieves the bucket and pod log artifact key (as well as the
 * minio client need to retrieve them) from the corresponding argo workflow status.
 *
 * @param podName name of the pod to retrieve the logs.
 */
export async function getPodLogsMinioRequestConfigfromWorkflow(
  podName: string,
): Promise<MinioRequestConfig> {
  let workflow: PartialArgoWorkflow;
  try {
    workflow = await getArgoWorkflow(workflowNameFromPodName(podName));
  } catch (err) {
    throw new Error(`Unable to retrieve workflow status: ${err}.`);
  }

  let artifacts: ArtifactRecord[] | undefined;
  // check if required fields are available
  if (workflow.status && workflow.status.nodes) {
    const node = workflow.status.nodes[podName];
    if (node && node.outputs && node.outputs.artifacts) {
      artifacts = node.outputs.artifacts;
    }
  }
  if (!artifacts) {
    throw new Error('Unable to find pod info in workflow status to retrieve logs.');
  }

  const archiveLogs: ArtifactRecord[] = artifacts.filter((artifact: any) => artifact.archiveLogs);

  if (archiveLogs.length === 0) {
    throw new Error('Unable to find pod log archive information from workflow status.');
  }

  const s3Artifact = archiveLogs[0].s3;
  if (!s3Artifact) {
    throw new Error('Unable to find s3 artifact info from workflow status.');
  }

  const { host, port } = urlSplit(s3Artifact.endpoint, s3Artifact.insecure);
  const { accessKey, secretKey } = await getMinioClientSecrets(s3Artifact);
  const client = await createMinioClient({
    accessKey,
    endPoint: host,
    port,
    secretKey,
    useSSL: !s3Artifact.insecure,
  });
  return {
    bucket: s3Artifact.bucket,
    client,
    key: s3Artifact.key,
  };
}

/**
 * Returns the k8s access key and secret used to connect to the s3 artifactory.
 * @param s3artifact s3artifact object describing the s3 artifactory config for argo workflow.
 */
async function getMinioClientSecrets({ accessKeySecret, secretKeySecret }: S3Artifact) {
  if (!accessKeySecret || !secretKeySecret) {
    return {};
  }
  const accessKey = await getK8sSecret(accessKeySecret.name, accessKeySecret.key);
  const secretKey = await getK8sSecret(secretKeySecret.name, secretKeySecret.key);
  return { accessKey, secretKey };
}

/**
 * Split an uri into host and port.
 * @param uri uri to split
 * @param insecure if port is not provided in uri, return port depending on whether ssl is enabled.
 */

function urlSplit(uri: string, insecure: boolean) {
  const chunks = uri.split(':');
  if (chunks.length === 1) {
    return { host: chunks[0], port: insecure ? 80 : 443 };
  }
  return { host: chunks[0], port: parseInt(chunks[1], 10) };
}

/**
 * Infers workflow name from pod name.
 * @param podName name of the pod.
 */
function workflowNameFromPodName(podName: string) {
  const chunks = podName.split('-');
  chunks.pop();
  return chunks.join('-');
}
