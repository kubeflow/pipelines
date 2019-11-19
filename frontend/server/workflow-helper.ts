
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
import {PassThrough} from 'stream';
import {ClientOptions as MinioClientOptions} from 'minio';
import {getK8sSecret, getArgoWorkflow, getPodLogs} from './k8s-helper';
import {createMinioClient, IMinioRequestConfig, getObjectStream} from './minio-helper';


export interface IPartialArgoWorkflow {
    status: {
        nodes?: IArgoWorkflowStatusNode
    }
}

export interface IArgoWorkflowStatusNode {
    [key: string]: IArgoWorkflowStatusNodeInfo;
}

export interface IArgoWorkflowStatusNodeInfo {
    outputs?: {
        artifacts?: IArtifactRecord[]
    }
}

export interface IArtifactRecord {
    archiveLogs?: boolean;
    name: string;
    s3?: IS3Artifact;
}
  
export interface IS3Artifact {
    accessKeySecret?: ISecretSelector;
    bucket: string;
    endpoint: string;
    insecure: boolean;
    key: string;
    secretKeySecret?: ISecretSelector;
}

export interface ISecretSelector {
    key: string;
    name: string;
}

/**
 * Returns the k8s access key and secret used to connect to the s3 artifactory.
 * @param s3artifact s3artifact object describing the s3 artifactory config for argo workflow. 
 */
async function getMinioClientSecrets({accessKeySecret, secretKeySecret}: IS3Artifact) {
    if (!accessKeySecret || !secretKeySecret) {
        return {}
    }
    const accessKey = await getK8sSecret(accessKeySecret.name, accessKeySecret.key);
    const secretKey = await getK8sSecret(secretKeySecret.name, secretKeySecret.key);
    return {accessKey, secretKey};
}

/**
 * Split an uri into host and port.
 * @param uri uri to split
 * @param insecure if port is not provided in uri, return port depending on whether ssl is enabled.
 */  
function urlSplit(uri: string, insecure: boolean) {
    let chunks = uri.split(":");
    if (chunks.length==1) 
        return {host: chunks[0], port: !!insecure ? 80 : 443};
    return {host: chunks[0], port: parseInt(chunks[1], 10)};
}

/**
 * Infers workflow name from pod name.
 * @param podName name of the pod.
 */
function workflowNameFromPodName(podName: string) {
    let chunks = podName.split("-");
    chunks.pop();
    return chunks.join("-");
}

export class PodLogsHandler {
    fromConfig?: (podName: string) => Promise<IMinioRequestConfig>;

    async getPodLogs(podName: string) {
        try {
            // retrieve from k8s
            const stream = new PassThrough();
            stream.end(await getPodLogs(podName));
            console.log(`Getting logs for pod:${podName}.`)
            return stream;
        } catch (k8sError) {
            console.error(`Unable to get logs for pod:${podName}: ${k8sError}`);
            return this.getPodLogsFromArchive(podName);
        }
    }

    async getPodLogsFromArchive(podName: string) {
        try {
            // try argo workflow crd status
            const request = await this.fromWorkflow(podName);
            const stream = await getObjectStream(request);
            console.log(`Getting logs for pod:${podName} from ${request.bucket}/${request.key}.`)
            return stream;
        } catch (workflowError) {
            if (!!this.fromConfig) {
                try {
                    const request = await this.fromConfig(podName);
                    const stream = await getObjectStream(request);
                    console.log(`Getting logs for pod:${podName} from ${request.bucket}/${request.key}.`)
                    return stream;
                } catch (configError) {
                    console.error(`Unable to get logs for pod:${podName}: ${configError}`);
                    throw new Error(`Unable to retrieve logs from ${podName}: ${workflowError}, ${configError}`)
                }
            }
            console.error(`Unable to get logs for pod:${podName}: ${workflowError}`);
            console.error(workflowError);
            throw new Error(`Unable to retrieve logs from ${podName}: ${workflowError}`)
        }        
    }

    setFallbackHandler(minioOptions: MinioClientOptions, bucket: string, prefix: string) {
        const client = createMinioClient(minioOptions);
        this.fromConfig = async function(podName: string): Promise<IMinioRequestConfig> {
            const workflowName = workflowNameFromPodName(podName);
            return {
                bucket,
                key: `${prefix}/${workflowName}/${podName}/main.log`,
                client: await client
            };
        }
        return this;
    }

    async fromWorkflow(podName: string): Promise<IMinioRequestConfig> {
        const workflow = await getArgoWorkflow(workflowNameFromPodName(podName));
    
        // check if required fields are available
        if (!workflow.status || !workflow.status.nodes || 
            !workflow.status.nodes[podName] || 
            !workflow.status.nodes[podName].outputs ||
            !workflow.status.nodes[podName].outputs.artifacts)
          throw new Error('Unable to find pod info in workflow status to retrieve logs.')
      
        const artifacts: IArtifactRecord[] = workflow.status.nodes[podName].outputs.artifacts;
        const archiveLogs: IArtifactRecord[] = artifacts.filter((artifact: any) => artifact.archiveLogs)
      
        if (archiveLogs.length === 0) 
          throw new Error('Unable to find pod log archive information from workflow status.')
        
        const s3Artifact = archiveLogs[0].s3;
        if (!s3Artifact) 
            throw new Error('Unable to find s3 artifact info from workflow status.')
       
        const {host, port} = urlSplit(s3Artifact.endpoint, s3Artifact.insecure);
        const {accessKey, secretKey} = await getMinioClientSecrets(s3Artifact);
        const client = await createMinioClient({
            endPoint: host,
            port,
            accessKey,
            secretKey,
            useSSL: !s3Artifact.insecure,
        });
        return {
            bucket: s3Artifact.bucket,
            key: s3Artifact.key,
            client,
        };
    }
}

const podLogsHandler = new PodLogsHandler();
export default podLogsHandler;