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
import {
  getK8sSecret,
  getArgoWorkflow,
  getPodLogs,
  getConfigMap,
  getServerNamespace,
} from './k8s-helper.js';
import { createMinioClient, MinioRequestConfig, getObjectStream } from './minio-helper.js';
import * as JsYaml from 'js-yaml';

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
        console.warn(`Primary pod-log source failed; falling back to archive: ${err}`);
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
export async function getPodLogsStreamFromWorkflow(
  podName: string,
  createdAt: string,
  namespace: string | undefined,
  options: WorkflowLogStoreOptions & { authEnabled: boolean },
) {
  return toGetPodLogsStream((name, created, ns) =>
    getPodLogsMinioRequestConfigfromWorkflow(name, created, ns, options),
  )(podName, createdAt, namespace);
}

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

/** PartialArtifactRepositoriesValue is used to deserialize the contents of the
 * artifact-repositories configmap.
 */
interface PartialArtifactRepositoriesValue {
  s3?: {
    keyFormat: string;
  };
  gcs?: {
    keyFormat: string;
  };
  oss?: {
    keyFormat: string;
  };
  artifactory?: {
    keyFormat: string;
  };
}

/**
 * getKeyFormatFromArtifactRepositories attempts to retrieve an
 * artifact-repositories configmap from a specified namespace. It then parses
 * the configmap and returns a keyFormat value in its data field.
 * @param namespace namespace of the configmap
 */
export async function getKeyFormatFromArtifactRepositories(
  namespace: string,
): Promise<string | undefined> {
  try {
    const [configMap, k8sError] = await getConfigMap('artifact-repositories', namespace);
    if (configMap === undefined) {
      throw k8sError;
    }
    const artifactRepositories = configMap?.data?.['artifact-repositories'];
    if (artifactRepositories === undefined) {
      throw new Error(
        `artifact-repositories configmap in ${namespace} namespace is missing an artifact-repositories field.`,
      );
    }
    const artifactRepositoriesValue = JsYaml.load(
      artifactRepositories,
    ) as PartialArtifactRepositoriesValue;
    if ('s3' in artifactRepositoriesValue) {
      return artifactRepositoriesValue.s3?.keyFormat;
    } else if ('gcs' in artifactRepositoriesValue) {
      return artifactRepositoriesValue.gcs?.keyFormat;
    } else if ('oss' in artifactRepositoriesValue) {
      return artifactRepositoriesValue.oss?.keyFormat;
    } else if ('artifactory' in artifactRepositoriesValue) {
      return artifactRepositoriesValue.artifactory?.keyFormat;
    } else {
      throw new Error(
        'artifact-repositories configmap missing one of [s3|gcs|oss|artifactory] fields.',
      );
    }
  } catch (error) {
    console.log(error);
    return undefined;
  }
}

/**
 * Derives the deterministic, namespace-scoped object-key prefix implied by a
 * keyFormat template, or returns null when the template does not place
 * {{workflow.namespace}} as a complete '/'-delimited path segment ahead of every
 * caller-influenced field. When null, no safe namespace prefix can be derived and
 * callers must fail closed.
 *
 * In multi-user mode the archived-log object key must be provably confined to the
 * authorized namespace. Merely containing {{workflow.namespace}} somewhere is not
 * sufficient: if a caller-controlled field (pod name, workflow name, creation
 * timestamp) precedes it, the resolved namespace value is not an unambiguous
 * prefix, and a key belonging to another namespace could coincidentally contain
 * the authorized namespace as a later segment (e.g. as a workflow-name segment).
 * Requiring the namespace tag as a bounded segment before all such fields yields
 * a fixed prefix ("<static>/<namespace>") that scopes every resolved key to the
 * namespace's own subtree.
 */
export function namespaceScopedKeyPrefix(keyFormat: string, namespace: string): string | null {
  if (!namespace) {
    return null;
  }
  const template = keyFormat.replace(/\s+/g, '');
  const namespaceTag = '{{workflow.namespace}}';
  // Multiple namespace tags make the effective boundary ambiguous and can leave
  // unresolved tags in generated keys.
  if (template.split(namespaceTag).length !== 2) {
    return null;
  }
  // The namespace tag must be a complete '/'-delimited path segment.
  const namespaceSegment = /(^|\/)\{\{workflow\.namespace\}\}(\/|$)/;
  if (!namespaceSegment.test(template)) {
    return null;
  }
  const namespaceIndex = template.indexOf(namespaceTag);
  const staticPrefix = template.slice(0, namespaceIndex);
  // No template tag of any kind may precede the namespace boundary. Limiting
  // this to the tags understood today would let a newly supported or custom tag
  // silently become caller-controlled input ahead of the authorization prefix.
  if (/\{\{[^{}]+\}\}/.test(staticPrefix) || staticPrefix.split('/').includes('..')) {
    return null;
  }
  // Everything up to the namespace tag is static (verified above to contain no
  // caller-controlled field); append the substituted namespace to form the
  // deterministic prefix.
  return staticPrefix + namespace;
}

/**
 * Returns a MinioRequestConfig with the provided minio options (a
 * MinioRequestConfig object contains the artifact bucket and keys, with the
 * corresponding minio client).
 * @param minioOptions Minio options to create a minio client.
 * @param bucket bucket containing the pod logs artifacts.
 * @param keyFormatDefault the default keyFormat for pod logs artifacts stored
 * in the bucket. This is overriden if there's an "artifact-repositories"
 * configmap in the target namespace with a keyFormat field.
 */
export function createPodLogsMinioRequestConfig(
  minioOptions: MinioClientOptions,
  bucket: string,
  keyFormatDefault: string,
  artifactRepositoriesLookup: boolean,
  authEnabled: boolean = false,
) {
  return async (
    podName: string,
    createdAt: string,
    namespace: string = '',
  ): Promise<MinioRequestConfig> => {
    // Standalone callers historically omit podnamespace. The shipped archive
    // layout is namespace-scoped, so resolve that omission to the frontend's
    // operator-controlled server namespace rather than generating a `//` key.
    const archiveNamespace = namespace || getServerNamespace() || '';
    // create a new client each time to ensure session token has not expired
    const client = await createMinioClient(minioOptions, 's3');
    const createdAtArray = createdAt.split('-');

    // If artifactRepositoriesLookup is enabled, try to extract they keyformat
    // from the configmap. Otherwise, just used the default keyFormat specified
    // in configs.ts.
    let keyFormatFromConfigMap = undefined;
    // A namespace owner can control this ConfigMap. In multi-user mode it must
    // not define the authorization boundary for a shared-store object read;
    // use only the operator-controlled default key format instead.
    if (artifactRepositoriesLookup && !authEnabled) {
      keyFormatFromConfigMap = await getKeyFormatFromArtifactRepositories(archiveNamespace);
    }
    let key: string;
    if (keyFormatFromConfigMap !== undefined) {
      key = keyFormatFromConfigMap;
    } else {
      key = keyFormatDefault;
    }

    // Fail closed in multi-user mode: the caller-supplied podname/createdat are
    // interpolated into the object key, but the namespace access check upstream
    // only authorizes `namespace`. Require the key template to scope the key to
    // {{workflow.namespace}} as a deterministic prefix (a complete '/'-delimited
    // segment ahead of every caller-controlled field), so any resolved key stays
    // confined to the authorized namespace's subtree. Merely *containing* the tag
    // is not enough: placing it adjacent to a caller-controlled field, e.g.
    // "{{workflow.namespace}}{{pod.name}}", or after one, e.g.
    // "{{pod.name}}/{{workflow.namespace}}", would let a tenant reach keys under
    // another namespace, so those templates are rejected.
    if (authEnabled && namespaceScopedKeyPrefix(key, archiveNamespace) === null) {
      throw new Error(
        `Refusing to read archived pod logs: the keyFormat, which is defined in config.ts or through the ` +
          `ARGO_KEYFORMAT env var, does not place {{workflow.namespace}} as a complete '/'-delimited path ` +
          `segment ahead of every caller-controlled field (pod name, workflow name, creation timestamp). In ` +
          `multi-user mode the log key must be scoped to the authorized namespace as a deterministic prefix; ` +
          `otherwise archived logs from other namespaces in the shared bucket would be readable.`,
      );
    }

    key = key
      .replace(/\s+/g, '') // Remove all whitespace.
      .replace('{{workflow.name}}', podName.replace(/-system-container-impl-.*/, ''))
      .replace('{{workflow.creationTimestamp.Y}}', createdAtArray[0])
      .replace('{{workflow.creationTimestamp.m}}', createdAtArray[1])
      .replace('{{workflow.creationTimestamp.d}}', createdAtArray[2])
      .replace('{{pod.name}}', podName)
      .replace('{{workflow.namespace}}', archiveNamespace);

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

    // The regex above permits '.' and '/', so reject any '..' segment. The
    // caller-supplied podname is interpolated into the key, and a '..' would
    // let it climb out of the (namespace-scoped) key prefix.
    if (key.split('/').includes('..')) {
      throw new Error(
        `The log key, ${key}, contains a '..' path segment and is therefore invalid.`,
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
export interface WorkflowLogStoreOptions {
  authEnabled?: boolean;
  trustedKeyFormat?: string;
  trustedBucket?: string;
  // Kubernetes cluster DNS suffix (e.g. '.svc.cluster.local'), used to treat the
  // short, '.svc', and fully-qualified forms of a Service host as equivalent when
  // comparing the workflow-recorded endpoint against the trusted store.
  clusterDomain?: string;
  trustedStore?: {
    endPoint: string;
    port?: number;
    region?: string;
    useSSL?: boolean;
    accessKey?: string;
    secretKey?: string;
  };
}

export async function getPodLogsMinioRequestConfigfromWorkflow(
  podName: string,
  _createdAt?: string,
  namespace?: string,
  options: WorkflowLogStoreOptions = {},
): Promise<MinioRequestConfig> {
  const {
    authEnabled = false,
    trustedKeyFormat = '',
    trustedBucket = '',
    clusterDomain = '.svc.cluster.local',
    trustedStore,
  } = options;
  let workflow: PartialArgoWorkflow;
  // We should probably parameterize this replace statement. It's brittle to
  // changes in implementation. But brittle is better than completely broken.
  let workflowName = podName.replace(/-system-container-impl-.*/, '');
  try {
    workflow = await getArgoWorkflow(workflowName, namespace);
  } catch (err) {
    throw new Error(`Unable to retrieve workflow status: ${err}.`, { cause: err });
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

  // Fail closed in multi-user mode: confine the workflow-derived log key to the
  // authorized namespace. `logKey` comes from the workflow status (shaped by the
  // per-namespace artifact-repositories keyFormat, which a tenant may be able to
  // influence) and is read below with the shared object-store credentials, so
  // without this check a tenant could point the main-logs artifact at another
  // namespace's log object and read it. Require the key to begin with the
  // deterministic namespace-scoped prefix derived from the trusted (operator)
  // keyFormat, and reject any '..' segment. This mirrors the confinement applied
  // to the configured archive fallback in createPodLogsMinioRequestConfig. If the
  // trusted keyFormat does not scope by {{workflow.namespace}} ahead of every
  // caller-controlled field, no safe prefix exists and we fail closed.
  const serverNamespace = getServerNamespace();
  const validateWorkflowScope = authEnabled;
  const usesSharedCredentials = authEnabled && Boolean(namespace && namespace !== serverNamespace);
  if (validateWorkflowScope) {
    const prefix = namespaceScopedKeyPrefix(trustedKeyFormat, namespace || '');
    if (prefix === null || !logKey.startsWith(prefix + '/') || logKey.split('/').includes('..')) {
      throw new Error(
        `Refusing to read archived pod logs: the workflow-recorded log key "${logKey}" is not confined to ` +
          `the authorized namespace under the namespace-scoped prefix derived from the trusted keyFormat. In ` +
          `multi-user mode the archive key must begin with that deterministic namespace prefix.`,
      );
    }
  }

  const s3Artifact = workflow.status.artifactRepositoryRef.artifactRepository.s3 || false;
  if (!s3Artifact) {
    throw new Error('Unable to find artifact repository information from workflow status.');
  }

  const workflowEndpoint = parseArtifactStoreEndpoint(s3Artifact.endpoint, s3Artifact.insecure);
  if (!workflowEndpoint) {
    throw new Error('Artifact repository endpoint is invalid or conflicts with its TLS setting.');
  }

  if (validateWorkflowScope) {
    const trustedEndpoint = trustedStore
      ? parseArtifactStoreEndpoint(
          trustedStore.endPoint,
          trustedStore.useSSL === false,
          trustedStore.port,
        )
      : undefined;
    if (
      !trustedEndpoint ||
      canonicalArtifactOrigin(workflowEndpoint, clusterDomain) !==
        canonicalArtifactOrigin(trustedEndpoint, clusterDomain)
    ) {
      throw new Error(
        'Refusing to read archived pod logs from a workflow-controlled artifact endpoint.',
      );
    }
    if (!trustedBucket || s3Artifact.bucket !== trustedBucket) {
      throw new Error(
        'Refusing to read archived pod logs from a workflow-controlled artifact bucket.',
      );
    }
  }

  // Security: Only read the object-store credential Secret from the server's own
  // namespace. In multi-user deployments the run namespace is a customer/user
  // namespace, and the ml-pipeline-ui service account may not read Secrets there;
  // for those runs we instead use the frontend server's own configured
  // object-store credentials (MINIO_ACCESS_KEY / MINIO_SECRET_KEY, with the same
  // defaults as configs.ts). Those credentials own the shared bucket (SeaweedFS
  // in the kubeflow namespace), so the workflow-status log path works for user
  // namespaces against the shared store instead of building a doomed anonymous
  // client. See: https://github.com/kubeflow/pipelines/pull/12860
  //
  // Multi-user mode always supplies an explicit namespace (the pod-logs handler
  // rejects requests without one), so an omitted namespace only occurs in
  // standalone mode, where the run is effectively in the server namespace. We
  // therefore treat a missing namespace as the server namespace and read the
  // workflow-referenced Secret so custom object-store credentials are honored,
  // while still refusing to read Secrets from any user namespace.
  let accessKey: string | undefined;
  let secretKey: string | undefined;
  if (namespace && namespace === serverNamespace) {
    // Explicit server-namespace run (including multi-user runs whose namespace is
    // the server namespace): read the Secret and use whatever it yields, exactly
    // as before.
    ({ accessKey, secretKey } = await getMinioClientSecrets(s3Artifact, namespace));
  } else if (!namespace && serverNamespace) {
    // Standalone run with an omitted namespace: read the workflow-referenced
    // Secret from the server namespace, falling back to the frontend's configured
    // env credentials only when the artifact repository does not reference a
    // Secret (getMinioClientSecrets returns no credentials).
    const { accessKey: readAccessKey = undefined, secretKey: readSecretKey = undefined } =
      await getMinioClientSecrets(s3Artifact, serverNamespace);
    accessKey = readAccessKey || process.env.MINIO_ACCESS_KEY || 'minio';
    secretKey = readSecretKey || process.env.MINIO_SECRET_KEY || 'minio123';
  } else {
    // Cross-namespace (user-namespace) run, or an unknown server namespace: never
    // read a user-namespace Secret; use the frontend's own configured credentials.
    accessKey = usesSharedCredentials
      ? trustedStore?.accessKey
      : process.env.MINIO_ACCESS_KEY || 'minio';
    secretKey = usesSharedCredentials
      ? trustedStore?.secretKey
      : process.env.MINIO_SECRET_KEY || 'minio123';
  }

  const client = await createMinioClient(
    {
      accessKey,
      // TODO: endPoint needs to be set to 'localhost' for local development.
      // start-proxy-and-server.sh sets MINIO_HOST=localhost, but it doesn't
      // seem to be respected when running the server in development mode.
      // Investigate and fix this.
      endPoint: workflowEndpoint.host,
      port: workflowEndpoint.port,
      ...(usesSharedCredentials && trustedStore?.region ? { region: trustedStore.region } : {}),
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
 * Reduces a Kubernetes Service host to its shortest equivalent form so that the
 * short (`svc.ns`), `.svc` (`svc.ns.svc`), and fully-qualified
 * (`svc.ns.svc.<clusterDomain>`) spellings of the same in-cluster Service compare
 * equal. This is used only for the trusted-endpoint equality check; the real,
 * workflow-recorded host is still what the client connects to. A trailing dot
 * (absolute DNS name) is also stripped.
 */
function canonicalizeServiceHost(host: string, clusterDomain: string): string {
  let h = host.toLowerCase().replace(/\.+$/, '');
  const domain = (clusterDomain || '').toLowerCase().replace(/^\.+/, '').replace(/\.+$/, '');
  if (domain && h.endsWith('.' + domain)) {
    h = h.slice(0, h.length - domain.length - 1);
  }
  if (h.endsWith('.svc')) {
    h = h.slice(0, h.length - '.svc'.length);
  }
  return h;
}

/**
 * Builds an origin string (`<scheme>//<canonical-host>:<port>`) for endpoint
 * equality, canonicalizing the Service host so DNS-equivalent spellings match.
 */
function canonicalArtifactOrigin(
  endpoint: { host: string; port: number; origin: string },
  clusterDomain: string,
): string {
  const scheme = endpoint.origin.slice(0, endpoint.origin.indexOf('//'));
  return `${scheme}//${canonicalizeServiceHost(endpoint.host, clusterDomain)}:${endpoint.port}`;
}

function parseArtifactStoreEndpoint(
  endpoint: string,
  insecure: boolean,
  configuredPort?: number,
): { host: string; port: number; origin: string } | undefined {
  try {
    const protocol = insecure ? 'http:' : 'https:';
    const hasExplicitProtocol = endpoint.includes('://');
    const parsed = new URL(hasExplicitProtocol ? endpoint : `${protocol}//${endpoint}`);
    if (
      !['http:', 'https:'].includes(parsed.protocol) ||
      (hasExplicitProtocol && parsed.protocol !== protocol) ||
      parsed.username ||
      parsed.password ||
      (parsed.pathname && parsed.pathname !== '/') ||
      parsed.search ||
      parsed.hash
    ) {
      return undefined;
    }
    const port = parsed.port ? Number(parsed.port) : configuredPort || (insecure ? 80 : 443);
    return {
      host: parsed.hostname,
      port,
      origin: `${protocol}//${parsed.hostname.toLowerCase()}:${port}`,
    };
  } catch {
    return undefined;
  }
}

/**
 * Returns the k8s access key and secret used to connect to the s3 artifactory.
 * @param s3artifact s3artifact object describing the s3 artifactory config for argo workflow.
 */
async function getMinioClientSecrets(
  { accessKeySecret, secretKeySecret }: S3Artifact,
  namespace?: string,
) {
  if (!accessKeySecret || !secretKeySecret) {
    return {};
  }
  const accessKey = await getK8sSecret(accessKeySecret.name, accessKeySecret.key, namespace);
  const secretKey = await getK8sSecret(secretKeySecret.name, secretKeySecret.key, namespace);
  return { accessKey, secretKey };
}
