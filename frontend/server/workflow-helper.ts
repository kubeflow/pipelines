// Copyright 2019 The Kubeflow Authors
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
import { PassThrough, Stream } from 'stream';
import { ClientOptions as MinioClientOptions } from 'minio';
import { getK8sSecret, getArgoWorkflow, getPodLogs } from './k8s-helper';
import { createMinioClient, MinioRequestConfig, getObjectStream } from './minio-helper';

export interface PartialArgoWorkflow {
  status: {
    artifactRepositoryRef?: ArtifactRepositoryRef;
    nodes?: ArgoWorkflowStatusNode;
  };
}

export interface ArtifactRepositoryRef {
  artifactRepository?: ArtifactRepository;
}

export interface ArtifactRepository {
  archiveLogs?: boolean;
  s3?: S3Artifact;
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
  name?: string;
  s3: S3Key;
}

export interface S3Key {
  key: string;
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
  handler: (podName: string, createdAt: string, namespace?: string) => Promise<T>,
  fallback?: (podName: string, createdAt: string, namespace?: string) => Promise<T>,
) {
  return async (podName: string, createdAt: string, namespace?: string) => {
    try {
      return await handler(podName, createdAt, namespace);
    } catch (err) {
      if (fallback) {
        return await fallback(podName, createdAt, namespace);
      }
      console.warn(err);
      throw err;
    }
  };
}

/**
 * Returns a stream containing the pod logs using kubernetes api.
 * @param podName name of the pod.
 * @param createdAt YYYY-MM-DD run was created. Not used.
 * @param namespace namespace of the pod (uses the same namespace as the server if not provided).
 * @param containerName container's name of the pod, the default value is 'main'.
 */
export async function getPodLogsStreamFromK8s(
  podName: string,
  createdAt: string,
  namespace?: string,
  containerName: string = 'main',
) {
  const stream = new PassThrough();
  stream.end(await getPodLogs(podName, namespace, containerName));
  console.log(
    `Getting logs for pod, ${podName}, in namespace, ${namespace}, by calling the Kubernetes API.`,
  );
  return stream;
}

/**
 * Returns a stream containing the pod logs using the information provided in the
 * workflow status (uses k8s api to retrieve the workflow and secrets).
 * @param podName name of the pod.
 * @param createdAt YYYY-MM-DD run was created. Not used.
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
  getMinioRequestConfig: (
    podName: string,
    createdAt: string,
    namespace?: string,
  ) => Promise<MinioRequestConfig>,
) {
  return async (podName: string, createdAt: string, namespace?: string) => {
    const request = await getMinioRequestConfig(podName, createdAt, namespace);
    console.log(`Getting logs for pod, ${podName}, from ${request.bucket}/${request.key}.`);
    return await getObjectStream(request);
  };
}

/**
 * Returns a MinioRequestConfig with the provided minio options (a MinioRequestConfig
 * object contains the artifact bucket and keys, with the corresponding minio
 * client).
 * @param minioOptions Minio options to create a minio client.
 * @param bucket bucket containing the pod logs artifacts.
 * @param keyFormat the keyFormat for pod logs artifacts stored in the bucket.
 */
export function createPodLogsMinioRequestConfig(
  minioOptions: MinioClientOptions,
  bucket: string,
  keyFormat: string,
) {
  return async (
    podName: string,
    createdAt: string,
    namespace: string = '',
  ): Promise<MinioRequestConfig> => {
    // create a new client each time to ensure session token has not expired
    const client = await createMinioClient(minioOptions, 's3');
    const createdAtArray = createdAt.split('-');

    let key: string = keyFormat
      .replace(/\s+/g, '') // Remove all whitespace.
      .replace('{{workflow.name}}', podName.replace(/-system-container-impl-.*/, ''))
      .replace('{{workflow.creationTimestamp.Y}}', createdAtArray[0])
      .replace('{{workflow.creationTimestamp.m}}', createdAtArray[1])
      .replace('{{workflow.creationTimestamp.d}}', createdAtArray[2])
      .replace('{{pod.name}}', podName)
      .replace('{{workflow.namespace}}', namespace);

    if (!key.endsWith('/')) {
      key = key + '/';
    }
    key = key + 'main.log';

    // If there are unresolved template tags in the keyFormat, throw an error
    // that surfaces in the frontend's console log.
    if (key.includes('{') || key.includes('}')) {
      throw new Error(
        `keyFormat, which is defined in config.ts or through the ARGO_KEYFORMAT env var, appears to include template tags that are not supported. ` +
          `The resulting log key, ${key}, includes unresolved template tags and is therefore invalid.`,
      );
    }

    const regex = /^[a-zA-Z0-9\-._/]+$/; // Allow letters, numbers, -, ., _, /
    if (!regex.test(key)) {
      throw new Error(
        `The log key, ${key}, which is derived from keyFormat in config.ts or through the ARGO_KEYFORMAT env var, is an invalid path. ` +
          `Supported characters include: letters, numbers, -, ., _, and /.`,
      );
    }

    return { bucket, client, key };
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
  // We should probably parameterize this replace statement. It's brittle to
  // changes in implementation. But brittle is better than completely broken.
  let workflowName = podName.replace(/-system-container-impl-.*/, '');
  try {
    workflow = await getArgoWorkflow(workflowName);
  } catch (err) {
    throw new Error(`Unable to retrieve workflow status: ${err}.`);
  }

  // archiveLogs can be set globally for the workflow as a whole and / or for
  // each individual task. The compiler sets it globally so we look for it in
  // the global field, which is documented here:
  // https://argo-workflows.readthedocs.io/en/release-3.4/fields/#workflow
  if (!workflow.status.artifactRepositoryRef?.artifactRepository?.archiveLogs) {
    throw new Error('Unable to retrieve logs from artifact store; archiveLogs is disabled.');
  }

  let artifacts: ArtifactRecord[] | undefined;
  if (workflow.status && workflow.status.nodes) {
    const nodeName = podName.replace('-system-container-impl', '');
    const node = workflow.status.nodes[nodeName];
    artifacts = node?.outputs?.artifacts || undefined;
  }
  if (!artifacts) {
    throw new Error('Unable to find corresponding log artifact in node.');
  }

  const logKey =
    artifacts.find((artifact: ArtifactRecord) => artifact.name === 'main-logs')?.s3.key || false;
  if (!logKey) {
    throw new Error('No artifact named "main-logs" for node.');
  }

  const s3Artifact = workflow.status.artifactRepositoryRef.artifactRepository.s3 || false;
  if (!s3Artifact) {
    throw new Error('Unable to find artifact repository information from workflow status.');
  }

  const { host, port } = urlSplit(s3Artifact.endpoint, s3Artifact.insecure);
  const { accessKey, secretKey } = await getMinioClientSecrets(s3Artifact);

  const client = await createMinioClient(
    {
      accessKey,
      // TODO: endPoint needs to be set to 'localhost' for local development.
      // start-proxy-and-server.sh sets MINIO_HOST=localhost, but it doesn't
      // seem to be respected when running the server in development mode.
      // Investigate and fix this.
      endPoint: host,
      port,
      secretKey,
      useSSL: !s3Artifact.insecure,
    },
    's3',
  );
  return {
    bucket: s3Artifact.bucket,
    client,
    key: logKey,
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
