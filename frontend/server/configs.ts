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
import { randomBytes } from 'crypto';
import * as path from 'path';
import { getArtifactStoreOrigin, parseArtifactStoreEndpoint } from './minio-helper.js';
import { loadJSON } from './utils.js';
import { loadArtifactsProxyConfig, ArtifactsProxyConfig } from './handlers/artifacts.js';
export const BASEPATH = '/pipeline';
export const apiVersion1 = 'v1beta1';
export const apiVersion1Prefix = `apis/${apiVersion1}`;
export const apiVersion2 = 'v2beta1';
export const apiVersion2Prefix = `apis/${apiVersion2}`;

export enum Deployments {
  NOT_SPECIFIED = 'NOT_SPECIFIED',
  KUBEFLOW = 'KUBEFLOW',
  MARKETPLACE = 'MARKETPLACE',
}

/** converts string to bool */
const asBool = (value: string) => ['true', '1'].includes(value.toLowerCase());

function parseOptionalPort(value: string | undefined, name: string): number | undefined {
  if (value === undefined || value.trim() === '') {
    return undefined;
  }
  const normalized = value.trim();
  if (!/^\d+$/.test(normalized)) {
    throw new Error(`${name} must be an integer between 1 and 65535`);
  }
  const port = Number(normalized);
  if (!Number.isSafeInteger(port) || port < 1 || port > 65535) {
    throw new Error(`${name} must be an integer between 1 and 65535`);
  }
  return port;
}

function validateConfiguredEndpointPort(
  endpoint: string | undefined,
  port: number | undefined,
  endpointName: string,
  portName: string,
): void {
  if (!endpoint || port === undefined) {
    return;
  }
  let parsed: URL;
  try {
    parsed = new URL(endpoint.includes('://') ? endpoint : `https://${endpoint}`);
  } catch {
    return;
  }
  if (!parsed.hostname) {
    return;
  }
  const authority = endpoint.replace(/^[a-z][a-z0-9+.-]*:\/\//i, '').split(/[/?#]/, 1)[0];
  const embeddedPort = /:(\d+)$/.exec(authority)?.[1];
  if (embeddedPort && Number(embeddedPort) !== port) {
    throw new Error(
      `${endpointName} port ${embeddedPort} conflicts with ${portName} value ${port}`,
    );
  }
}

function parseAllowedArtifactEndpoints(value: string): string[] {
  if (!value.trim()) {
    return [];
  }
  return value
    .split(',')
    .map((entry) => entry.trim())
    .filter(Boolean)
    .map((configured) => {
      try {
        const endpoint = new URL(configured);
        if (
          !['http:', 'https:'].includes(endpoint.protocol) ||
          endpoint.username ||
          endpoint.password ||
          endpoint.pathname !== '/' ||
          endpoint.search ||
          endpoint.hash
        ) {
          throw new Error();
        }
        return endpoint.origin;
      } catch {
        throw new Error(
          `ALLOWED_ARTIFACT_ENDPOINTS entry must be an absolute HTTP(S) origin: ${configured}`,
        );
      }
    });
}

function getArgoArtifactRepositoryEndpoints(
  host: string,
  namespace: string,
  clusterDomain: string,
  minio: MinioConfigs,
): string[] {
  const dnsLabel = /^[a-z0-9](?:[-a-z0-9]{0,61}[a-z0-9])?$/i;
  if (!dnsLabel.test(host) || !dnsLabel.test(namespace)) {
    return [];
  }

  // The controller manifest uses .svc; profile repositories use CLUSTER_DOMAIN.
  // Derive exact origins only from operator settings, never workflow metadata.
  const suffixes = new Set(['svc']);
  const domain = clusterDomain.replace(/^\./, '');
  if (domain.split('.').every((label) => dnsLabel.test(label))) {
    suffixes.add(domain);
  }
  return [...suffixes]
    .map((suffix) =>
      getArtifactStoreOrigin({ ...minio, endPoint: `${host}.${namespace}.${suffix}` }),
    )
    .filter((endpoint): endpoint is string => endpoint !== undefined);
}

const MIN_TENSORBOARD_PROXY_SIGNING_SECRET_BYTES = 32;
const EPHEMERAL_TENSORBOARD_PROXY_SIGNING_SECRET = randomBytes(
  MIN_TENSORBOARD_PROXY_SIGNING_SECRET_BYTES,
).toString('base64url');

function resolveTensorboardProxySigningSecret(
  configuredSecret: string | undefined,
  minioSecret: string,
): string {
  if (configuredSecret === undefined) {
    return EPHEMERAL_TENSORBOARD_PROXY_SIGNING_SECRET;
  }

  if (configuredSecret === minioSecret) {
    throw new Error(
      'TENSORBOARD_PROXY_SIGNING_SECRET must not reuse MINIO_SECRET_KEY. Configure a dedicated random secret.',
    );
  }

  if (Buffer.byteLength(configuredSecret, 'utf8') < MIN_TENSORBOARD_PROXY_SIGNING_SECRET_BYTES) {
    throw new Error(
      `TENSORBOARD_PROXY_SIGNING_SECRET must be at least ${MIN_TENSORBOARD_PROXY_SIGNING_SECRET_BYTES} bytes.`,
    );
  }

  return configuredSecret;
}

function parseArgs(argv: string[]) {
  if (argv.length < 3) {
    const msg = `\
  Usage: node server.js <static-dir> [port].
         You can specify the API server address using the
         ML_PIPELINE_SERVICE_HOST and ML_PIPELINE_SERVICE_PORT
         env vars.`;
    throw new Error(msg);
  }

  const staticDir = path.resolve(argv[2]);
  const port = parseInt(argv[3] || '3000', 10);
  return { staticDir, port };
}

export type ProcessEnv = NodeJS.ProcessEnv | { [key: string]: string };

export function loadConfigs(argv: string[], env: ProcessEnv): UIConfigs {
  const { staticDir, port } = parseArgs(argv);
  /** All configurable environment variables can be found here. */
  const {
    /** Object storage client credentials (SeaweedFS/MinIO compatible) */
    MINIO_ACCESS_KEY = 'minio',
    MINIO_SECRET_KEY = 'minio123',
    MINIO_PORT = '9000',
    MINIO_HOST = 'seaweedfs',
    MINIO_NAMESPACE = 'kubeflow',
    MINIO_SSL = 'false',
    /** S3-compatible storage credentials */
    AWS_ACCESS_KEY_ID,
    AWS_SECRET_ACCESS_KEY,
    AWS_REGION,
    AWS_S3_ENDPOINT,
    AWS_S3_PORT,
    AWS_SSL = 'true',
    /** http/https base URL */
    HTTP_BASE_URL = '',
    /** By default, allowing access to all domains. Modify this flag to allow querying matching domains */
    ALLOWED_ARTIFACT_DOMAIN_REGEX = '^.*$',
    /** Additional object-store origins allowed for secret-backed provider configs. */
    ALLOWED_ARTIFACT_ENDPOINTS = '',
    /** http/https fetch with this authorization header key (for example: 'Authorization') */
    HTTP_AUTHORIZATION_KEY = '',
    /** http/https fetch with this authorization header value by default when absent in client request at above key */
    HTTP_AUTHORIZATION_DEFAULT_VALUE = '',
    /** API service will listen to this host */
    ML_PIPELINE_SERVICE_HOST = 'localhost',
    /** API service will listen to this port */
    ML_PIPELINE_SERVICE_PORT = '3001',
    /** API service will listen via this transfer protocol */
    ML_PIPELINE_SERVICE_SCHEME = 'http',
    /** path to viewer:tensorboard pod template spec */
    VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH,
    /** Tensorflow image used for tensorboard viewer */
    VIEWER_TENSORBOARD_TF_IMAGE_NAME = 'tensorflow/tensorflow',
    /** Optional shared signing secret for scoped TensorBoard proxy paths */
    TENSORBOARD_PROXY_SIGNING_SECRET,
    /** Whether custom visualizations are allowed to be generated by the frontend */
    ALLOW_CUSTOM_VISUALIZATIONS = 'false',
    /** Envoy service will listen to this host */
    METADATA_ENVOY_SERVICE_SERVICE_HOST = 'localhost',
    /** Envoy service will listen to this port */
    METADATA_ENVOY_SERVICE_SERVICE_PORT = '9090',
    /** Envoy service will listen via this transfer protocol */
    METADATA_ENVOY_SERVICE_SERVICE_SCHEME = 'http',
    /** Is Argo log archive enabled? */
    ARGO_ARCHIVE_LOGS = 'false',
    /** Use minio or s3 client to retrieve archives. */
    ARGO_ARCHIVE_ARTIFACTORY = 'minio',
    /** Bucket to retrive logs from */
    ARGO_ARCHIVE_BUCKETNAME = 'mlpipeline',
    /** This should match the keyFormat specified in the Argo workflow-controller-configmap.
     * It's set here in the manifests:
     * https://github.com/kubeflow/pipelines/blob/7b7918ebf8c30e6ceec99283ef20dbc02fdf6a42/manifests/kustomize/third-party/argo/base/workflow-controller-configmap-patch.yaml#L28
     */
    ARGO_KEYFORMAT = 'artifacts/{{workflow.name}}/{{workflow.creationTimestamp.Y}}/{{workflow.creationTimestamp.m}}/{{workflow.creationTimestamp.d}}/{{pod.name}}',
    /** Argo Workflows lets you specify a unique artifact repository for each
     * namespace by adding an appropriately formatted configmap to the namespace
     * as documented here:
     * https://argo-workflows.readthedocs.io/en/latest/artifact-repository-ref/.
     * Use this field to enable this lookup. It defaults to false.
     */
    ARGO_ARTIFACT_REPOSITORIES_LOOKUP = 'false',
    /** Should use server API for log streaming? */
    STREAM_LOGS_FROM_SERVER_API = 'false',
    /** The main container name of a pod where logs are retrieved */
    POD_LOG_CONTAINER_NAME = 'main',
    /** Disables GKE metadata endpoint. */
    DISABLE_GKE_METADATA = 'true',
    /** Enable authorization checks for multi user mode. */
    ENABLE_AUTHZ = 'false',
    /** Deployment type. */
    DEPLOYMENT: DEPLOYMENT_STR = '',
    /**
     * Set to true to hide the SideNav. When DEPLOYMENT is KUBEFLOW, HIDE_SIDENAV
     * defaults to true if not explicitly set to false.
     */
    HIDE_SIDENAV,
    /**
     * A header user requests have when authenticated. It carries user identity information.
     * The default value works with Google Cloud IAP.
     */
    KUBEFLOW_USERID_HEADER = 'x-goog-authenticated-user-email',
    /**
     * KUBEFLOW_USERID_HEADER's value may have a prefix before user identity.
     * Use this header to specify what the prefix is.
     *
     * e.g. a valid header value for default values can be like `accounts.google.com:user@gmail.com`.
     */
    KUBEFLOW_USERID_PREFIX = 'accounts.google.com:',
    FRONTEND_SERVER_NAMESPACE = 'kubeflow',
    CLUSTER_DOMAIN = '.svc.cluster.local',
  } = env;

  const configuredAwsEndpoint = AWS_S3_ENDPOINT?.trim();
  const configuredAwsPort = parseOptionalPort(AWS_S3_PORT, 'AWS_S3_PORT');
  validateConfiguredEndpointPort(
    configuredAwsEndpoint,
    configuredAwsPort,
    'AWS_S3_ENDPOINT',
    'AWS_S3_PORT',
  );
  const awsUseSSL = asBool(AWS_SSL);
  const parsedAwsEndpoint = configuredAwsEndpoint
    ? parseArtifactStoreEndpoint(configuredAwsEndpoint, !awsUseSSL)
    : undefined;
  if (configuredAwsEndpoint && !parsedAwsEndpoint) {
    throw new Error(
      'AWS_S3_ENDPOINT must be a valid HTTP(S) origin consistent with AWS_SSL, without credentials, a path, query, or fragment',
    );
  }

  const minio: MinioConfigs = {
    accessKey: MINIO_ACCESS_KEY,
    endPoint: MINIO_NAMESPACE ? `${MINIO_HOST}.${MINIO_NAMESPACE}` : MINIO_HOST,
    port: parseOptionalPort(MINIO_PORT, 'MINIO_PORT') ?? 9000,
    secretKey: MINIO_SECRET_KEY,
    useSSL: asBool(MINIO_SSL),
  };

  return {
    argo: {
      archiveArtifactory: ARGO_ARCHIVE_ARTIFACTORY,
      archiveBucketName: ARGO_ARCHIVE_BUCKETNAME,
      archiveLogs: asBool(ARGO_ARCHIVE_LOGS),
      keyFormat: ARGO_KEYFORMAT,
      artifactRepositoriesLookup: asBool(ARGO_ARTIFACT_REPOSITORIES_LOOKUP),
      artifactRepositoryEndpoints: getArgoArtifactRepositoryEndpoints(
        MINIO_HOST,
        MINIO_NAMESPACE,
        CLUSTER_DOMAIN,
        minio,
      ),
    },
    pod: {
      logContainerName: POD_LOG_CONTAINER_NAME,
    },
    artifacts: {
      aws: {
        accessKey: AWS_ACCESS_KEY_ID || '',
        endPoint: parsedAwsEndpoint?.endPoint || 's3.amazonaws.com',
        port: parsedAwsEndpoint?.port ?? configuredAwsPort,
        region: AWS_REGION || 'us-east-1',
        secretKey: AWS_SECRET_ACCESS_KEY || '',
        useSSL: awsUseSSL,
      },
      http: {
        auth: {
          defaultValue: HTTP_AUTHORIZATION_DEFAULT_VALUE,
          key: HTTP_AUTHORIZATION_KEY,
        },
        baseUrl: HTTP_BASE_URL,
      },
      minio,
      proxy: loadArtifactsProxyConfig(env),
      allowOfficialAwsEndpoints: Boolean(configuredAwsEndpoint),
      streamLogsFromServerApi: asBool(STREAM_LOGS_FROM_SERVER_API),
      allowedDomain: ALLOWED_ARTIFACT_DOMAIN_REGEX,
      allowedEndpoints: parseAllowedArtifactEndpoints(ALLOWED_ARTIFACT_ENDPOINTS),
    },
    metadata: {
      envoyService: {
        host: METADATA_ENVOY_SERVICE_SERVICE_HOST,
        port: METADATA_ENVOY_SERVICE_SERVICE_PORT,
        schema: METADATA_ENVOY_SERVICE_SERVICE_SCHEME,
      },
    },
    pipeline: {
      host: ML_PIPELINE_SERVICE_HOST,
      port: ML_PIPELINE_SERVICE_PORT,
      schema: ML_PIPELINE_SERVICE_SCHEME,
    },
    server: {
      apiVersion1Prefix,
      apiVersion2Prefix,
      basePath: BASEPATH,
      deployment:
        DEPLOYMENT_STR.toUpperCase() === Deployments.KUBEFLOW
          ? Deployments.KUBEFLOW
          : DEPLOYMENT_STR.toUpperCase() === Deployments.MARKETPLACE
            ? Deployments.MARKETPLACE
            : Deployments.NOT_SPECIFIED,
      hideSideNav:
        HIDE_SIDENAV === undefined
          ? DEPLOYMENT_STR.toUpperCase() === Deployments.KUBEFLOW
          : asBool(HIDE_SIDENAV),
      port,
      staticDir,
      serverNamespace: FRONTEND_SERVER_NAMESPACE,
    },
    viewer: {
      tensorboard: {
        podTemplateSpec: loadJSON<object>(VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH),
        tfImageName: VIEWER_TENSORBOARD_TF_IMAGE_NAME,
        clusterDomain: CLUSTER_DOMAIN,
        proxySigningSecret: resolveTensorboardProxySigningSecret(
          TENSORBOARD_PROXY_SIGNING_SECRET,
          MINIO_SECRET_KEY,
        ),
      },
    },
    visualizations: {
      allowCustomVisualizations: asBool(ALLOW_CUSTOM_VISUALIZATIONS),
    },
    gkeMetadata: {
      disabled: asBool(DISABLE_GKE_METADATA),
    },
    auth: {
      enabled: asBool(ENABLE_AUTHZ),
      kubeflowUserIdHeader: KUBEFLOW_USERID_HEADER,
      kubeflowUserIdPrefix: KUBEFLOW_USERID_PREFIX,
    },
  };
}

export interface MinioConfigs {
  accessKey: string;
  secretKey: string;
  endPoint: string;
  port: number;
  useSSL: boolean;
}
export interface AWSConfigs {
  endPoint: string;
  port?: number;
  region: string;
  accessKey: string;
  secretKey: string;
  useSSL: boolean;
}
export interface HttpConfigs {
  baseUrl: string;
  auth: {
    key: string;
    defaultValue: string;
  };
}
export interface PipelineConfigs {
  host: string;
  port: string | number;
  schema: string;
}
export interface ViewerTensorboardConfig {
  podTemplateSpec?: object;
  tfImageName: string;
  clusterDomain: string;
  proxySigningSecret: string;
}
export interface ViewerConfigs {
  tensorboard: ViewerTensorboardConfig;
}
export interface VisualizationsConfigs {
  allowCustomVisualizations: boolean;
}
export interface MetadataConfigs {
  envoyService: {
    host: string;
    port: string | number;
    schema: string;
  };
}
export interface ArgoConfigs {
  archiveLogs: boolean;
  archiveArtifactory: string;
  archiveBucketName: string;
  keyFormat: string;
  artifactRepositoriesLookup: boolean;
  /** Exact origins generated for operator-configured Kubernetes artifact repositories. */
  artifactRepositoryEndpoints?: string[];
}
export interface ServerConfigs {
  basePath: string;
  port: string | number;
  staticDir: string;
  apiVersion1Prefix: string;
  apiVersion2Prefix: string;
  deployment: Deployments;
  hideSideNav: boolean;
  // Namespace where the server is deployed
  serverNamespace: string;
}
export interface GkeMetadataConfigs {
  disabled: boolean;
}
export interface AuthConfigs {
  enabled: boolean;
  kubeflowUserIdHeader: string;
  kubeflowUserIdPrefix: string;
}
export interface UIConfigs {
  server: ServerConfigs;
  artifacts: {
    aws: AWSConfigs;
    minio: MinioConfigs;
    http: HttpConfigs;
    proxy: ArtifactsProxyConfig;
    allowOfficialAwsEndpoints: boolean;
    streamLogsFromServerApi: boolean;
    allowedDomain: string;
    allowedEndpoints: string[];
  };
  pod: {
    logContainerName: string;
  };
  argo: ArgoConfigs;
  metadata: MetadataConfigs;
  visualizations: VisualizationsConfigs;
  viewer: ViewerConfigs;
  pipeline: PipelineConfigs;
  gkeMetadata: GkeMetadataConfigs;
  auth: AuthConfigs;
}
