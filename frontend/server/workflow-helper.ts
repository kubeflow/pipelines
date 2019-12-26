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
import { PassThrough } from 'stream';
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
    return { host: chunks[0], port: !!insecure ? 80 : 443 };
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

export class PodLogsHandler {
  fromConfig?: (podName: string) => Promise<MinioRequestConfig>;

  async getPodLogs(podName: string, podNamespace?: string) {
    try {
      // retrieve from k8s
      const stream = new PassThrough();
      stream.end(await getPodLogs(podName, podNamespace));
      console.log(`Getting logs for pod:${podName} in namespace ${podNamespace}.`);
      return stream;
    } catch (k8sError) {
      console.error(`Unable to get logs for pod:${podName}:`, k8sError);
      return this.getPodLogsFromArchive(podName);
    }
  }

  // TODO: support pod in a certain namespace
  async getPodLogsFromArchive(podName: string) {
    try {
      // try argo workflow crd status
      const request = await this.fromWorkflow(podName);
      const stream = await getObjectStream(request);
      console.log(`Getting logs for pod:${podName} from ${request.bucket}/${request.key}.`);
      return stream;
    } catch (workflowError) {
      if (!!this.fromConfig) {
        try {
          const request = await this.fromConfig(podName);
          const stream = await getObjectStream(request);
          console.log(`Getting logs for pod:${podName} from ${request.bucket}/${request.key}.`);
          return stream;
        } catch (configError) {
          console.error(`Unable to get logs for pod:${podName} with ARGO_ARCHIVE_* env flags:`, configError);
          throw new Error(
            `Unable to retrieve logs from ${podName} workflow status: ${workflowError}\n` +
            `Unable to retrieve logs with ARGO_ARCHIVE_* env flags: ${configError}`,
          );
        }
      }
      console.error(`Unable to get logs for pod:${podName} from workflow status:`, workflowError);
      throw new Error(`Unable to retrieve logs from ${podName} workflow status: ${workflowError}`);
    }
  }

  setFallbackHandler(minioOptions: MinioClientOptions, bucket: string, prefix: string) {
    this.fromConfig = async (podName: string): Promise<MinioRequestConfig> => {
      // create a new client each time to ensure session token has not expired
      const client = await createMinioClient(minioOptions);
      const workflowName = workflowNameFromPodName(podName);
      return {
        bucket,
        client,
        key: `${prefix}/${workflowName}/${podName}/main.log`,
      };
    };
    return this;
  }

  async fromWorkflow(podName: string): Promise<MinioRequestConfig> {
    let workflow: PartialArgoWorkflow;
    try {
      workflow = await getArgoWorkflow(workflowNameFromPodName(podName));
    } catch (err) {
      throw new Error(`Unable to retrieve workflow status: ${err}.`);
    }

    // check if required fields are available
    if (
      !workflow.status ||
      !workflow.status.nodes ||
      !workflow.status.nodes[podName] ||
      !workflow.status.nodes[podName].outputs ||
      !workflow.status.nodes[podName].outputs.artifacts
    ) {
      throw new Error('Unable to find pod info in workflow status to retrieve logs.');
    }

    const artifacts: ArtifactRecord[] = workflow.status.nodes[podName].outputs.artifacts;
    const archiveLogs: ArtifactRecord[] = artifacts.filter(
      (artifact: any) => artifact.archiveLogs,
    );

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
}

const podLogsHandler = new PodLogsHandler();
export default podLogsHandler;
