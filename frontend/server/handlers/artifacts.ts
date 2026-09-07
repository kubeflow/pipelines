// Copyright 2019-2020 The Kubeflow Authors
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
import { AWSConfigs, HttpConfigs, MinioConfigs, ProcessEnv, UIConfigs } from '../configs.js';
import { Client as MinioClient } from 'minio';
import {
  PreviewStream,
  findFileOnPodVolume,
  parseJSONString,
  isAllowedResourceName,
  openFileWithinRoot,
} from '../utils.js';
import {
  createMinioClient,
  getObjectStream,
  isNoSuchKeyError,
  listObjectsUnderPrefix,
  summarizeDirectoryUnderPrefix,
} from '../minio-helper.js';
import type { MinioRequestConfig } from '../minio-helper.js';
import * as tar from 'tar-stream';
import * as zlib from 'zlib';
import type { IncomingMessage } from 'http';
import { Readable } from 'stream';
import { pipeline as pipelinePromise } from 'stream/promises';
import * as serverInfo from '../helpers/server-info.js';
import { Handler, Request, Response, NextFunction } from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import { HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS } from '../consts.js';
import { URL } from 'url';
import { getGCSClient, listGCSObjectNames, downloadGCSObjectStream } from '../gcs-helper.js';
import type { GCSClient } from '../gcs-helper.js';

import { isAllowedDomain } from './domain-checker.js';
import { getK8sSecret } from '../k8s-helper.js';
import { CredentialBody } from 'google-auth-library';
import { AuthorizeFn } from '../helpers/auth.js';
import { validateArtifactNamespace, buildArtifactUri } from '../helpers/mlmd-validator.js';
import { resolveArtifactCoordinates } from '../helpers/artifact-coordinates.js';
import {
  AuthorizeRequestResources,
  AuthorizeRequestVerb,
} from '../src/generated/apis/auth/index.js';

/**
 * ArtifactsQueryStrings describes the expected query strings key value pairs
 * in the artifact request object.
 */
interface ArtifactsQueryStrings {
  /** artifact source. */
  source: 'minio' | 's3' | 'gcs' | 'http' | 'https' | 'volume';
  /** bucket name. */
  bucket: string;
  /** artifact key/path that is uri encoded.  */
  key: string;
  /** return only the first x characters or bytes. */
  peek?: number;
  /** optional provider info to use to query object store */
  providerInfo?: string;
  namespace?: string;
  /** return the artifact byte-for-byte without archive extraction */
  download?: string;
}

type ArtifactSource = ArtifactsQueryStrings['source'];

const ARTIFACT_SOURCES = new Set<ArtifactSource>(['minio', 's3', 'gcs', 'http', 'https', 'volume']);
const ARTIFACT_QUERY_PARAMETER_NAMES = [
  'source',
  'bucket',
  'key',
  'providerInfo',
  'namespace',
  'peek',
  'download',
] as const;

export interface S3ProviderInfo {
  Provider: string;
  Params: {
    fromEnv: string;
    secretName?: string;
    accessKeyKey?: string;
    secretKeyKey?: string;
    region?: string;
    endpoint?: string;
    disableSSL?: string;
  };
}

export interface GCSProviderInfo {
  Provider: string;
  Params: {
    fromEnv: string;
    secretName?: string;
    tokenKey?: string;
  };
}

function hardenArtifactResponse(response: Response): void {
  response.setHeader('X-Content-Type-Options', 'nosniff');
  response.setHeader('Content-Disposition', 'attachment');
}

const SAFE_ARTIFACT_PROXY_RESPONSE_HEADERS = new Set([
  'accept-ranges',
  'content-encoding',
  'content-length',
  'content-range',
  'etag',
  'last-modified',
]);

function hardenArtifactProxyResponse(proxyResponse: IncomingMessage): void {
  const contentDisposition = hardenUpstreamContentDisposition(
    proxyResponse.headers['content-disposition'],
  );
  for (const header of Object.keys(proxyResponse.headers)) {
    if (!SAFE_ARTIFACT_PROXY_RESPONSE_HEADERS.has(header.toLowerCase())) {
      delete proxyResponse.headers[header];
    }
  }
  proxyResponse.headers['content-type'] = 'application/octet-stream';
  proxyResponse.headers['x-content-type-options'] = 'nosniff';
  proxyResponse.headers['content-disposition'] = contentDisposition;
}

export function sendArtifactError(response: Response, status: number, message: string): void {
  // A stream may fail after successful response headers have already been
  // committed. Express cannot change the status or MIME type at that point,
  // so abort the connection. Ending it normally would emit a valid terminating
  // chunk and make a truncated artifact indistinguishable from a complete one.
  if (response.headersSent || response.destroyed || response.writableEnded) {
    console.error(`[artifacts] aborting committed response: ${message}`);
    response.destroy();
    return;
  }
  hardenArtifactResponse(response);
  response.status(status).type('text/plain').send(message);
}

export function pipePreviewResponse(
  source: Readable,
  response: Response,
  peek: number,
  onError: (error: Error) => void,
): void {
  const preview = new PreviewStream({ peek });
  if (response.destroyed || response.writableEnded) {
    source.destroy();
    preview.destroy();
    return;
  }
  let failed = false;
  const cleanup = () => {
    source.off('error', failOnce);
    preview.off('error', failOnce);
    response.off('error', failOnce);
    response.off('close', failOnPrematureClose);
    response.off('finish', cleanup);
  };
  const failOnce = (error: Error) => {
    if (failed) {
      return;
    }
    failed = true;
    cleanup();
    source.unpipe(preview);
    preview.unpipe(response);
    source.destroy();
    preview.destroy();
    onError(error);
  };
  const failOnPrematureClose = () => {
    if (!response.writableFinished) {
      // The peer has already gone away, so there is no response left to repair
      // and no storage failure to report. Stop upstream work without routing a
      // routine client cancellation through the server-error logger.
      if (failed) {
        return;
      }
      failed = true;
      cleanup();
      source.unpipe(preview);
      preview.unpipe(response);
      source.destroy();
      preview.destroy();
    }
  };
  source.once('error', failOnce);
  preview.once('error', failOnce);
  response.once('error', failOnce);
  response.once('close', failOnPrematureClose);
  response.once('finish', cleanup);
  source.pipe(preview).pipe(response);
}

/**
 * Returns an authorization middleware for artifact endpoints.
 * This middleware handles 3 modes:
 *
 * 1. Standalone KFP deployment without Kubeflow platform (single-tenant):
 *    No Subject Access Review and 100% insecure. The namespace query
 *    parameter is optional and not validated or authorized when
 *    authorization is disabled.
 *
 * 2. Default multi-tenant deployment of KFP within Kubeflow platform:
 *    Namespace parameter is required, its format is validated, and RBAC is
 *    checked (the user is authenticated to access the artifact from the
 *    specific namespace folder on the object storage via Subject Access
 *    Review) before accessing SeaweedFS/storage directly.
 *
 * 3. Artifact PROXY MODE (overhead, disabled by default):
 *    Namespace parameter is required, its format is validated, and RBAC is
 *    checked. This adds significant overhead to each namespace, decreases
 *    scalability, and is prone to many CVEs in the artifact proxy
 *    deployment.
 *
 * Note: Secret-backed provider mode (fromEnv === 'false') names a Kubernetes
 * Secret to source object-store credentials from. The frontend server only
 * honors it when the requested namespace is the server's own namespace, so it
 * never reads Secrets from a customer namespace. In multi-user deployments the
 * provider info is dropped for user namespaces and artifact retrieval falls
 * back to the server's own environment credentials (SeaweedFS in the kubeflow
 * namespace) or the per-namespace artifact proxy.
 * See: https://github.com/kubeflow/pipelines/pull/12860
 *
 * Security: This addresses the vulnerability where the namespace parameter
 * could be manipulated to access artifacts from other namespaces.
 * See https://github.com/kubeflow/pipelines/issues/9889
 *
 * @param authorizeFn The authorization function to validate permissions
 * @param authEnabled Whether authorization is enabled
 * @param kubeflowUserIdHeader The header name containing the user identity
 * @param envoyAddress MLMD Envoy address used for namespace-ownership
 *   validation (#9889). When omitted, the IDOR check is skipped.
 */
export function getArtifactsAuthMiddleware(
  authorizeFn: AuthorizeFn,
  authEnabled: boolean,
  kubeflowUserIdHeader: string,
  envoyAddress?: string,
): Handler {
  return async (request: Request, response: Response, next: NextFunction) => {
    hardenArtifactResponse(response);
    const queryError = validateArtifactQueryParameters(request.query);
    if (queryError) {
      sendArtifactError(response, queryError.status, queryError.message);
      return;
    }

    if (!authEnabled) {
      return next();
    }

    const userId = request.headers[kubeflowUserIdHeader.toLowerCase()];
    if (!userId) {
      console.warn(
        `[SECURITY] Unauthenticated artifact access attempt. Path: ${request.originalUrl}`,
      );
      sendArtifactError(response, 401, 'Authentication required for artifact access');
      return;
    }

    const namespaceParameter = getOptionalRequestString(request.query.namespace, 'namespace');
    if ('error' in namespaceParameter) {
      sendArtifactError(
        response,
        namespaceParameter.error.status,
        namespaceParameter.error.message,
      );
      return;
    }
    const namespace = namespaceParameter.value;

    if (!namespace) {
      console.warn(
        `[SECURITY] Missing namespace parameter. ` +
          `User: ${userId}, Path: ${request.originalUrl}`,
      );
      sendArtifactError(
        response,
        400,
        'Namespace parameter is required when authentication is enabled',
      );
      return;
    }

    if (!isAllowedResourceName(namespace)) {
      console.warn(
        `[SECURITY] Invalid namespace format. ` +
          `User: ${userId}, ` +
          `Namespace: ${namespace}, Path: ${request.originalUrl}`,
      );
      sendArtifactError(response, 400, 'Invalid namespace format');
      return;
    }

    const authError = await authorizeFn(
      {
        verb: AuthorizeRequestVerb.GET,
        resources: AuthorizeRequestResources.VIEWERS,
        namespace: namespace,
      },
      request,
    );

    if (authError) {
      console.warn(
        `[SECURITY] Unauthorized cross-namespace access attempt. ` +
          `User: ${userId}, ` +
          `Namespace: ${namespace}, Path: ${request.originalUrl}, ` +
          `Reason: ${authError.message}`,
      );
      sendArtifactError(response, 403, authError.message);
      return;
    }

    if (envoyAddress) {
      const coords = resolveArtifactCoordinates(request);
      if (coords === null) {
        console.warn(
          `[SECURITY] Malformed percent-encoding in artifact path. ` +
            `User: ${userId}, Path: ${request.path}`,
        );
        sendArtifactError(response, 400, 'Malformed URL encoding in artifact path');
        return;
      }
      const mlmdTrackedSources = new Set(['minio', 's3', 'gcs', 'http', 'https']);
      if (mlmdTrackedSources.has(coords.source) && coords.bucket && coords.key) {
        const artifactUri = buildArtifactUri(coords.source, coords.bucket, coords.key);
        const validation = await validateArtifactNamespace(envoyAddress, artifactUri, namespace);

        if (!validation.valid) {
          console.warn(
            `[SECURITY] IDOR blocked: artifact namespace mismatch. ` +
              `User: ${userId}, ` +
              `Claimed namespace: ${namespace}, ` +
              `Actual namespace: ${validation.actualNamespace}, ` +
              `URI: ${artifactUri}, ` +
              `Path: ${request.path}`,
          );
          sendArtifactError(response, 403, 'Artifact does not belong to the requested namespace');
          return;
        }
      }
    }

    next();
  };
}

/**
 * Returns an artifact handler which retrieve an artifact from the corresponding
 * backend (i.e. gcs, minio, s3, http/https).
 * @param artifactsConfigs configs to retrieve the artifacts from the various backend.
 * @param useParameter get bucket and key from parameter instead of query. When true, expect
 *    to be used in a route like `/artifacts/:source/:bucket/*`.
 * @param tryExtract whether preview responses may extract content from *.tar.gz files.
 * Download routes pass false so S3 and MinIO archives are returned byte-for-byte
 * with an attachment filename; preview routes may extract the first tar entry.
 */
export function getArtifactsHandler({
  artifactsConfigs,
  useParameter,
  tryExtract,
  options,
}: {
  artifactsConfigs: {
    aws: AWSConfigs;
    http: HttpConfigs;
    minio: MinioConfigs;
    allowedDomain: string;
  };
  tryExtract: boolean;
  useParameter: boolean;
  options: UIConfigs;
}): Handler {
  const { aws, http, minio, allowedDomain } = artifactsConfigs;
  return async (req, res) => {
    // Security: artifact bytes are untrusted, user-controlled content. Set the
    // hardening headers before parsing so every early error and storage path is
    // protected. Inline previews use fetch(), which ignores Content-Disposition.
    hardenArtifactResponse(res);
    const artifactRequest = parseArtifactRequest(req, useParameter, options.server.serverNamespace);
    if ('error' in artifactRequest) {
      sendArtifactError(res, artifactRequest.error.status, artifactRequest.error.message);
      return;
    }
    const { source, bucket, key, peek, providerInfo, namespace, download } = artifactRequest;
    const keyBaseName = key.replace(/\/+$/, '').split('/').pop() || 'artifact';
    const setArtifactFilename = (transformed: boolean) => {
      res.setHeader(
        'Content-Disposition',
        buildAttachmentDisposition(transformed ? 'artifact' : keyBaseName),
      );
    };
    if (source !== 'minio' && source !== 's3') {
      setArtifactFilename(false);
    }
    if (!isAllowedResourceName(bucket)) {
      sendArtifactError(res, 500, 'Invalid bucket name');
      return;
    }
    if (key.length > 1024) {
      sendArtifactError(res, 500, 'Object key too long');
      return;
    }
    console.log(`Getting storage artifact at: ${source}: ${bucket}/${key}`);

    // Security: The ml-pipeline-ui service account is only permitted to read
    // Secrets from its own (server) namespace. Secret-backed provider info
    // (fromEnv === 'false') names a Secret to read for object-store
    // credentials; honoring it for a customer/user namespace would read
    // Secrets cross-namespace, which is forbidden. When the requested
    // namespace is not the server's own namespace we drop the provider info so
    // credential resolution falls back to the server's own environment
    // credentials (SeaweedFS in the kubeflow namespace) or, when enabled, the
    // per-namespace artifact proxy. See:
    // https://github.com/kubeflow/pipelines/pull/12860
    // A missing namespace only occurs when auth is disabled (single-tenant): the
    // auth middleware rejects namespace-less requests whenever auth is enabled, so
    // treating it as server-local cannot be triggered by a multi-user caller.
    const allowProviderSecrets = !namespace || namespace === options.server.serverNamespace;
    if (!allowProviderSecrets && providerInfo) {
      console.warn(
        `Ignoring secret-backed provider info for namespace "${namespace}": Secrets may ` +
          `only be read from the server namespace; falling back to environment credentials.`,
      );
    }
    const effectiveProviderInfo = allowProviderSecrets ? providerInfo : '';

    let client: MinioClient;
    switch (source) {
      case 'gcs':
        await getGCSArtifactHandler(
          { bucket, key },
          peek,
          effectiveProviderInfo,
          namespace,
          useParameter || download,
        )(req, res);
        break;
      case 'minio':
        try {
          client = await createMinioClient(minio, 'minio', effectiveProviderInfo, namespace);
        } catch (e) {
          sendArtifactError(res, 500, `Failed to initialize Minio Client for Minio Provider: ${e}`);
          return;
        }
        await getMinioArtifactHandler(
          {
            bucket,
            client,
            key,
            tryExtract: tryExtract && !download,
            onTransformationDetermined: setArtifactFilename,
          },
          peek,
        )(req, res);
        break;
      case 's3':
        try {
          client = await createMinioClient(aws, 's3', effectiveProviderInfo, namespace);
        } catch (e) {
          sendArtifactError(res, 500, `Failed to initialize Minio Client for S3 Provider: ${e}`);
          return;
        }
        await getMinioArtifactHandler(
          {
            bucket,
            client,
            key,
            tryExtract: tryExtract && !download,
            onTransformationDetermined: setArtifactFilename,
          },
          peek,
        )(req, res);
        break;
      case 'http':
      case 'https': {
        const httpUrl = getHttpUrl(source, http.baseUrl || '', bucket, key);
        if (!httpUrl) {
          sendArtifactError(
            res,
            400,
            http.baseUrl.trim()
              ? 'Invalid HTTP artifact path'
              : 'HTTP artifact base URL is not configured',
          );
          return;
        }
        await getHttpArtifactsHandler(allowedDomain, httpUrl, http.auth, peek)(req, res);
        break;
      }
      case 'volume':
        await getVolumeArtifactsHandler(
          {
            bucket,
            key,
          },
          peek,
        )(req, res);
        break;
      default:
        sendArtifactError(res, 500, 'Unknown storage source');
        return;
    }
  };
}

type ArtifactRequest =
  | {
      source: ArtifactSource;
      bucket: string;
      key: string;
      peek: number;
      providerInfo: string;
      namespace: string;
      download: boolean;
    }
  | { error: { status: number; message: string } };

function parseArtifactRequest(
  req: Request,
  useParameter: boolean,
  defaultNamespace: string,
): ArtifactRequest {
  const source = getRequiredRequestString(
    useParameter ? req.params.source : req.query.source,
    'source',
    'Storage source is missing from artifact request',
  );
  if ('error' in source) {
    return source;
  }
  if (!isArtifactSource(source.value)) {
    return { error: { status: 500, message: 'Unknown storage source' } };
  }

  const bucket = getRequiredRequestString(
    useParameter ? req.params.bucket : req.query.bucket,
    'bucket',
    'Storage bucket is missing from artifact request',
  );
  if ('error' in bucket) {
    return bucket;
  }

  const key = getRequiredRequestString(
    useParameter ? req.params[0] : req.query.key,
    'key',
    'Storage key is missing from artifact request',
  );
  if ('error' in key) {
    return key;
  }

  const providerInfo = getOptionalRequestString(req.query.providerInfo, 'providerInfo');
  if ('error' in providerInfo) {
    return providerInfo;
  }

  const namespace = getOptionalRequestString(req.query.namespace, 'namespace');
  if ('error' in namespace) {
    return namespace;
  }

  const peek = getOptionalRequestString(req.query.peek, 'peek');
  if ('error' in peek) {
    return peek;
  }

  const download = getOptionalRequestString(req.query.download, 'download');
  if ('error' in download) {
    return download;
  }
  if (download.value !== undefined && download.value !== 'true' && download.value !== 'false') {
    return { error: { status: 400, message: 'download must be true or false when provided' } };
  }

  return {
    source: source.value,
    bucket: bucket.value,
    key: key.value,
    peek: parsePeekValue(peek.value),
    providerInfo: providerInfo.value ?? '',
    namespace: namespace.value ?? defaultNamespace,
    download: download.value === 'true',
  };
}

function getRequiredRequestString(
  value: unknown,
  name: string,
  missingMessage: string,
): { value: string } | { error: { status: number; message: string } } {
  const optional = getOptionalRequestString(value, name);
  if ('error' in optional) {
    return optional;
  }
  if (!optional.value) {
    return { error: { status: 500, message: missingMessage } };
  }
  return { value: optional.value };
}

function getOptionalRequestString(
  value: unknown,
  name: string,
): { value: string | undefined } | { error: { status: number; message: string } } {
  if (value === undefined) {
    return { value: undefined };
  }
  if (typeof value !== 'string') {
    return { error: { status: 400, message: `${name} must be a single string value` } };
  }
  return { value };
}

function validateArtifactQueryParameters(
  query: Request['query'],
): { status: number; message: string } | undefined {
  for (const name of ARTIFACT_QUERY_PARAMETER_NAMES) {
    const parameter = getOptionalRequestString(query[name], name);
    if ('error' in parameter) {
      return parameter.error;
    }
  }
  return undefined;
}

function parsePeekValue(value: string | undefined): number {
  if (!value) {
    return 0;
  }
  const peek = Number(value);
  return Number.isFinite(peek) && peek > 0 ? peek : 0;
}

function isArtifactSource(source: string): source is ArtifactSource {
  return ARTIFACT_SOURCES.has(source as ArtifactSource);
}

/**
 * Returns the http/https url to retrieve a kfp artifact (of the form: `${source}://${baseUrl}${bucket}/${key}`)
 * @param source "http" or "https".
 * @param baseUrl string to prefix the url.
 * @param bucket name of the bucket.
 * @param key path to the artifact.
 */
function getHttpUrl(source: 'http' | 'https', baseUrl: string, bucket: string, key: string) {
  const configuredBaseUrl = baseUrl.trim().replace(/^\/+/, '');
  if (!configuredBaseUrl) {
    return undefined;
  }
  try {
    const artifactUrl = new URL(`${source}://${configuredBaseUrl}`);
    if (
      key.includes('\\') ||
      key.split('/').some((segment) => segment === '.' || segment === '..')
    ) {
      return undefined;
    }
    const escapedKey = key.replace(/%/g, '%25');
    artifactUrl.pathname = [artifactUrl.pathname.replace(/\/+$/, ''), bucket, escapedKey]
      .filter(Boolean)
      .join('/');
    artifactUrl.search = '';
    artifactUrl.hash = '';
    return artifactUrl.toString();
  } catch {
    return undefined;
  }
}

function getHttpArtifactsHandler(
  allowedDomain: string,
  url: string,
  auth: {
    key: string;
    defaultValue: string;
  } = { key: '', defaultValue: '' },
  peek: number = 0,
) {
  return async (req: Request, res: Response) => {
    const headers: Record<string, string> = {};

    // add authorization header to fetch request if key is non-empty
    if (auth.key.length > 0) {
      // inject original request's value if exists, otherwise default to provided default value
      const headerValue =
        req.headers[auth.key] || req.headers[auth.key.toLowerCase()] || auth.defaultValue;
      headers[auth.key] = Array.isArray(headerValue) ? headerValue[0] : headerValue;
    }
    // Follow redirects manually so every hop is re-checked against the
    // allowlist. Letting fetch auto-follow only validates the first URL, so an
    // allowed host could 3xx the request to an internal address (link-local
    // metadata, cluster services) and exfiltrate the response plus any auth
    // header.
    const maxRedirects = 5;
    let currentUrl = url;
    const credentialOrigin = new URL(url).origin;
    let requestHeaders = headers;
    let response: Awaited<ReturnType<typeof fetch>>;
    for (let hop = 0; ; hop++) {
      const allowedUrl = parseAllowedHttpArtifactUrl(currentUrl, allowedDomain);
      if (!allowedUrl) {
        sendArtifactError(res, 500, 'Domain not allowed.');
        return;
      }
      if (new URL(allowedUrl).origin !== credentialOrigin) {
        requestHeaders = {};
      }
      response = await fetch(allowedUrl, { headers: requestHeaders, redirect: 'manual' });
      const status = response.status ?? 200;
      if (status < 300 || status >= 400) {
        break;
      }
      const location = response.headers?.get('location');
      if (!location) {
        break;
      }
      // We are not streaming this redirect response, so release its body.
      // Node's fetch keeps the connection tied up until GC if the body is left
      // unconsumed, which shows up under redirect-heavy artifact traffic.
      if (response.body) {
        await response.body.cancel().catch(() => undefined);
      }
      if (hop >= maxRedirects) {
        sendArtifactError(res, 500, 'Too many redirects while retrieving artifact');
        return;
      }
      // An allowed host can hand back a malformed Location header; resolve it
      // defensively so a bad value turns into a controlled 500 rather than an
      // unhandled exception escaping the handler.
      try {
        currentUrl = new URL(location, allowedUrl).toString();
      } catch {
        sendArtifactError(res, 500, 'Invalid redirect location while retrieving artifact');
        return;
      }
    }
    if (!response.body) {
      sendArtifactError(res, 500, 'Unable to retrieve artifact: empty response body');
      return;
    }
    const { Readable } = await import('stream');
    const nodeStream = Readable.fromWeb(response.body as any);
    pipePreviewResponse(nodeStream, res, peek, (err) =>
      sendArtifactError(res, 500, `Unable to retrieve artifact: ${err}`),
    );
  };
}

function parseAllowedHttpArtifactUrl(url: string, allowedDomain: string): string | undefined {
  try {
    const parsedUrl = new URL(url);
    if (parsedUrl.protocol !== 'http:' && parsedUrl.protocol !== 'https:') {
      return undefined;
    }
    if (!isAllowedDomain(parsedUrl.toString(), allowedDomain)) {
      return undefined;
    }
    return parsedUrl.toString();
  } catch {
    return undefined;
  }
}

function getMinioArtifactHandler(options: MinioRequestConfig, peek: number = 0) {
  return async (_: Request, res: Response) => {
    let handlingError = false;
    const handleObjectFailure = async (err: unknown) => {
      if (handlingError) {
        return;
      }
      handlingError = true;
      // In KFP v2, output artifacts may be directories (prefixes) rather than
      // single objects. Fall back to packaging the contents of the prefix as
      // a .tar.gz so users can still download them. See
      // https://github.com/kubeflow/pipelines/issues/7809. A provider can
      // surface NoSuchKey either by rejecting getObject or by emitting it on
      // the returned stream, so both paths converge here.
      if (isNoSuchKeyError(err) && !res.headersSent) {
        if (peek > 0) {
          try {
            await previewDirectorySummary(options, res);
          } catch (summaryErr) {
            console.error(summaryErr);
            sendArtifactError(res, 500, `Failed to summarize directory: ${summaryErr}`);
          }
          return;
        }
        try {
          await streamDirectoryAsTarGz(options, res);
        } catch (tarErr) {
          if (tarErr instanceof ArtifactResponseClosedError) {
            return;
          }
          console.error(tarErr);
          sendArtifactError(res, 500, `Failed to get object in bucket: ${tarErr}`);
        }
        return;
      }
      console.error(err);
      sendArtifactError(res, 500, `Failed to get object in bucket: ${err}`);
    };

    try {
      const stream = await getObjectStream({
        ...options,
        onError: (err) => void handleObjectFailure(err),
      });
      pipePreviewResponse(stream, res, peek, (err) => void handleObjectFailure(err));
    } catch (err) {
      await handleObjectFailure(err);
    }
  };
}

async function previewDirectorySummary(
  options: { bucket: string; key: string; client: MinioClient },
  res: Response,
) {
  const { bucket, key, client } = options;
  // Trailing slash so prefix "foo" doesn't also match sibling key "foobar".
  const prefix = key.endsWith('/') ? key : `${key}/`;
  const summary = await summarizeDirectoryUnderPrefix(client, bucket, prefix);
  if (!summary) {
    sendArtifactError(res, 404, `No objects found at ${bucket}/${key}`);
    return;
  }
  const baseName = key.replace(/\/+$/, '').split('/').pop() || 'artifact';
  const countLabel = `${summary.count}${summary.truncated ? '+' : ''}`;
  res
    .type('text/plain')
    .send(`Directory artifact "${baseName}" — ${countLabel} file(s). Download to view contents.\n`);
}

class ArtifactResponseClosedError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'ArtifactResponseClosedError';
  }
}

export async function streamDirectoryAsTarGz(
  options: { bucket: string; key: string; client: MinioClient },
  res: Response,
) {
  const { bucket, key, client } = options;
  // Trailing slash so prefix "foo" doesn't also match sibling key "foobar".
  const prefix = key.endsWith('/') ? key : `${key}/`;
  let pack: ReturnType<typeof tar.pack> | undefined;
  let responseComplete: Promise<void> | undefined;
  const archiveAbort = new AbortController();
  const abortArchive = (error: Error) => {
    if (!archiveAbort.signal.aborted) {
      archiveAbort.abort(error);
    }
  };
  const abortOnPrematureClose = () => {
    if (!res.writableFinished) {
      abortArchive(
        new ArtifactResponseClosedError(
          pack
            ? 'Artifact response closed before archive streaming completed'
            : 'Artifact response closed before archive streaming started',
        ),
      );
    }
  };
  if (res.destroyed) {
    abortArchive(
      new ArtifactResponseClosedError('Artifact response closed before archive streaming started'),
    );
  } else {
    res.once('close', abortOnPrematureClose);
  }
  const iterator = listObjectsUnderPrefix(client, bucket, prefix, archiveAbort.signal);
  const baseName = key.replace(/\/+$/, '').split('/').pop() || 'artifact';

  const startArchiveResponse = () => {
    if (pack && responseComplete) {
      return { pack, responseComplete };
    }
    res.setHeader('Content-Type', 'application/gzip');
    res.setHeader('Content-Disposition', buildAttachmentDisposition(`${baseName}.tar.gz`));
    pack = tar.pack();
    const gzip = zlib.createGzip();
    responseComplete = pipelinePromise(pack, gzip, res);
    // Observe the archive pipeline exactly once. Per-entry operations wait on
    // the abort signal below and remove their listener as soon as they settle;
    // attaching every entry to this archive-lifetime promise would retain two
    // pending promise reactions per object until the whole archive completed.
    void responseComplete.then(
      () => undefined,
      (error) => {
        abortArchive(error);
        pack?.destroy(error);
      },
    );
    return { pack, responseComplete };
  };

  const getObjectWhileResponseOpen = async (name: string) => {
    const objectRequest = client.getObject(bucket, name);
    return waitForArtifactOperation(objectRequest, archiveAbort.signal, (lateStream) =>
      lateStream.destroy(),
    );
  };

  const writeEntry = async ({ name, size }: { name: string; size: number }) => {
    const relativeName = name.startsWith(prefix) ? name.slice(prefix.length) : name;
    const safeName = sanitizeTarEntryName(relativeName);
    if (!safeName) {
      // Skip directory-marker objects (key === prefix) and any keys that
      // sanitize to an empty path.
      return;
    }
    // Resolve the first object before starting the response pipeline. If that
    // lookup fails, the caller can still return a well-formed HTTP error
    // instead of discovering that pipeline teardown already destroyed res.
    const objStream = await getObjectWhileResponseOpen(name);
    const archive = startArchiveResponse();
    const entryComplete = new Promise<void>((resolve, reject) => {
      const entry = archive.pack.entry({ name: safeName, size }, (err) =>
        err ? reject(err) : resolve(),
      );
      objStream.once('error', reject);
      objStream.pipe(entry);
    });
    try {
      await waitForArtifactOperation(entryComplete, archiveAbort.signal);
    } catch (error) {
      objStream.destroy();
      throw error;
    }
  };

  try {
    // Peek the first object before sending headers so an empty prefix can still
    // produce a 404 instead of an empty 200 tarball. The iterator shares the
    // response lifecycle signal, including while a listing page is pending.
    const first = await iterator.next();
    if (first.done) {
      sendArtifactError(res, 404, `No objects found at ${bucket}/${key}`);
      return;
    }
    await writeEntry(first.value);
    for await (const item of iterator) {
      await writeEntry(item);
    }
    const archive = startArchiveResponse();
    archive.pack.finalize();
    await archive.responseComplete;
  } catch (error) {
    const abortReason =
      archiveAbort.signal.aborted && archiveAbort.signal.reason instanceof Error
        ? archiveAbort.signal.reason
        : undefined;
    pack?.destroy(error as Error);
    await responseComplete?.catch(() => undefined);
    throw abortReason ?? error;
  } finally {
    res.off('close', abortOnPrematureClose);
  }
}

/**
 * Wait for one bounded artifact operation while sharing a single archive
 * lifecycle signal. The listener is explicitly removed when the operation
 * settles so a large directory cannot retain one archive-lifetime promise
 * reaction per object.
 */
export function waitForArtifactOperation<T>(
  operation: Promise<T>,
  signal: AbortSignal,
  onLateSuccess?: (value: T) => void,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    let settled = false;
    const cleanup = () => signal.removeEventListener('abort', rejectOnAbort);
    const rejectOnAbort = () => {
      if (settled) {
        return;
      }
      settled = true;
      cleanup();
      reject(
        signal.reason instanceof Error
          ? signal.reason
          : new Error('Artifact archive operation was aborted'),
      );
    };

    if (signal.aborted) {
      rejectOnAbort();
    } else {
      signal.addEventListener('abort', rejectOnAbort, { once: true });
    }

    void operation.then(
      (value) => {
        if (settled) {
          try {
            onLateSuccess?.(value);
          } catch {
            // The response is already gone; best-effort cleanup must not
            // create a new unhandled rejection.
          }
          return;
        }
        settled = true;
        cleanup();
        resolve(value);
      },
      (error) => {
        if (settled) {
          return;
        }
        settled = true;
        cleanup();
        reject(error);
      },
    );
  });
}

// Builds a `Content-Disposition: attachment` header that is safe to pass to
// `res.setHeader` regardless of the user-controlled filename. The legacy
// `filename=` parameter is reduced to an ASCII-only form so older clients
// don't see broken quoting; the modern `filename*` parameter carries the
// real name via RFC 5987 percent-encoding (UTF-8). Without this, a key
// containing quotes, control characters, or anything outside latin-1 could
// cause `setHeader` to throw or produce a malformed download name.
function buildAttachmentDisposition(filename: string): string {
  // Path separators have no place in a filename and are not valid in either
  // disposition parameter.
  const stripped = filename.replace(/[/\\]+/g, '_');
  const asciiFallback = stripped.replace(/[^A-Za-z0-9._-]/g, '_') || 'artifact';
  // encodeURIComponent leaves a few characters (', (, ), *) unencoded that
  // RFC 5987's `attr-char` set excludes; encode them explicitly so the
  // result conforms to `ext-value` from RFC 5987.
  const rfc5987Encoded = encodeURIComponent(stripped).replace(
    /['()*]/g,
    (c) => '%' + c.charCodeAt(0).toString(16).toUpperCase(),
  );
  return `attachment; filename="${asciiFallback}"; filename*=UTF-8''${rfc5987Encoded}`;
}

function hardenUpstreamContentDisposition(value: string | string[] | undefined): string {
  const disposition = Array.isArray(value) ? value[0] : value;
  if (!disposition) {
    return 'attachment';
  }

  const extended = /filename\*\s*=\s*([^'\s;]+)'[^']*'([^;\r\n]*)/i.exec(disposition);
  if (extended) {
    try {
      const charset = extended[1].toLowerCase();
      const encoded = extended[2];
      if (charset === 'utf-8' || charset === 'utf8') {
        return buildAttachmentDisposition(decodeURIComponent(encoded));
      }
      if (charset === 'iso-8859-1' || charset === 'latin1') {
        if (/%(?![0-9a-f]{2})/i.test(encoded)) {
          throw new Error('Malformed extended filename');
        }
        const decoded = encoded.replace(/%([0-9a-f]{2})/gi, (_, hex: string) =>
          String.fromCharCode(Number.parseInt(hex, 16)),
        );
        return buildAttachmentDisposition(decoded);
      }
    } catch {
      // Fall through to the legacy filename or a bare attachment.
    }
  }

  const quoted = /filename\s*=\s*"((?:\\.|[^"\\])*)"/i.exec(disposition)?.[1];
  const token = /filename\s*=\s*([^;\s\r\n]+)/i.exec(disposition)?.[1];
  const filename = quoted?.replace(/\\(["\\])/g, '$1') ?? token;
  return filename ? buildAttachmentDisposition(filename) : 'attachment';
}

// Sanitizes an object key into a safe relative POSIX path for inclusion in a
// tarball. Strips leading slashes and removes "." and ".." segments to
// prevent tar-slip path traversal during extraction. Returns null when the
// result is empty (e.g. for directory-marker objects whose key equals the
// prefix, or paths consisting entirely of unsafe segments).
function sanitizeTarEntryName(name: string): string | null {
  const segments = name
    .split('/')
    .filter((segment) => segment !== '' && segment !== '.' && segment !== '..');
  return segments.length > 0 ? segments.join('/') : null;
}

/**
 * Parses GCS provider info and retrieves credentials from a Kubernetes Secret.
 *
 * Security: The artifact handler only forwards provider info when the
 * requested namespace is the frontend server's own namespace, so this function
 * never reads Secrets from a customer namespace. In multi-user deployments the
 * provider info is dropped for user namespaces and credentials fall back to
 * the server's own environment credentials or the per-namespace artifact
 * proxy. See: https://github.com/kubeflow/pipelines/pull/12860
 */
async function parseGCSProviderInfo(
  providerInfo: GCSProviderInfo,
  namespace: string,
): Promise<CredentialBody> {
  if (!providerInfo.Params.tokenKey || !providerInfo.Params.secretName) {
    throw new Error(
      'Provider info with fromEnv:false supplied with incomplete secret credential info.',
    );
  }
  try {
    const tokenString = await getK8sSecret(
      providerInfo.Params.secretName,
      providerInfo.Params.tokenKey,
      namespace,
    );
    const credentials = parseJSONString<CredentialBody>(tokenString);
    if (!credentials) {
      throw new Error('Provider info token is not valid JSON.');
    }
    return credentials;
  } catch (err) {
    throw new Error('Failed to parse GCS Provider config. Error: ' + err, { cause: err });
  }
}

async function readGCSObject(
  bucket: string,
  objectName: string,
  client: GCSClient,
  credentials?: CredentialBody,
): Promise<Buffer> {
  const stream = await downloadGCSObjectStream({ bucket, objectName, credentials, client });
  const chunks: Buffer[] = [];
  for await (const chunk of stream) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  return Buffer.concat(chunks);
}

function getGCSArtifactHandler(
  options: { key: string; bucket: string },
  peek: number = 0,
  providerInfoString?: string,
  namespace?: string,
  isDownloadRoute: boolean = false,
) {
  const { key, bucket } = options;
  return async (_: Request, res: Response) => {
    try {
      let credentials: CredentialBody | undefined;
      if (providerInfoString) {
        const providerInfo = parseJSONString<GCSProviderInfo>(providerInfoString);
        if (providerInfo && providerInfo.Params.fromEnv === 'false') {
          if (!namespace) {
            sendArtifactError(
              res,
              500,
              'Failed to parse provider info. Reason: No namespace provided',
            );
            return;
          } else {
            credentials = await parseGCSProviderInfo(providerInfo, namespace);
          }
        }
      }
      // Read all files that match the key pattern, which can include wildcards '*'.
      // The way this works is we list all paths whose prefix is the substring
      // of the pattern until the first wildcard, then we create a regular
      // expression out of the pattern, escaping all non-wildcard characters,
      // and we use it to match all enumerated paths.
      const prefix = key.indexOf('*') > -1 ? key.substr(0, key.indexOf('*')) : key;
      const client = await getGCSClient(credentials);
      const matchingFiles = (
        await listGCSObjectNames({
          bucket,
          client,
          credentials,
          prefix,
        })
      ).filter((name) => {
        // Escape regex characters
        const escapeRegexChars = (s: string) => s.replace(/[|\\{}()[\]^$+*?.]/g, '\\$&');
        // Build a RegExp object that only recognizes asterisks ('*'), and
        // escapes everything else.
        const regex = new RegExp('^' + key.split(/\*+/).map(escapeRegexChars).join('.*') + '$');
        return regex.test(name);
      });

      if (!matchingFiles.length) {
        console.log('No matching files found.');
        res.type('text/plain').send();
        return;
      }
      console.log(`Found ${matchingFiles.length} matching files: `, matchingFiles.join(','));
      // TODO: support peek for concatenated matching files
      if (peek) {
        const stream = await downloadGCSObjectStream({
          bucket,
          client,
          credentials,
          objectName: matchingFiles[0],
        });
        res.type('text/plain');
        pipePreviewResponse(stream, res, peek, (err) =>
          sendArtifactError(res, 500, 'Failed to download GCS file(s). Error: ' + err),
        );
        return;
      }

      if (isDownloadRoute) {
        const contents: Buffer[] = [];
        for (const fileName of matchingFiles) {
          contents.push(await readGCSObject(bucket, fileName, client, credentials));
        }
        // Keep path-based downloads untyped and byte-preserving. Artifact
        // bytes are untrusted and may not be text; attachment + nosniff
        // provides the response hardening.
        res.end(Buffer.concat(contents));
        return;
      }

      // Preview wildcard matches are intentionally joined as trimmed text.
      let contents = '';
      for (const fileName of matchingFiles) {
        contents +=
          (await readGCSObject(bucket, fileName, client, credentials)).toString().trim() + '\n';
      }
      res.type('text/plain').send(contents);
    } catch (err) {
      sendArtifactError(res, 500, 'Failed to download GCS file(s). Error: ' + err);
    }
  };
}

function getVolumeArtifactsHandler(options: { bucket: string; key: string }, peek: number = 0) {
  const { key, bucket } = options;
  return async (req: Request, res: Response) => {
    try {
      const [pod, err] = await serverInfo.getHostPod();
      if (err) {
        sendArtifactError(res, 500, String(err));
        return;
      }

      if (!pod) {
        sendArtifactError(res, 500, 'Could not get server pod');
        return;
      }

      // ml-pipeline-ui server container name also be called 'ml-pipeline-ui-artifact' in KFP multi user mode.
      // https://github.com/kubeflow/manifests/blob/master/pipeline/installs/multi-user/pipelines-profile-controller/sync.py#L212
      const [filePath, parseError, volumeMountPath] = findFileOnPodVolume(pod, {
        containerNames: ['ml-pipeline-ui', 'ml-pipeline-ui-artifact'],
        volumeMountName: bucket,
        filePathInVolume: key,
      });
      if (parseError) {
        console.log(`Failed to open volume: ${parseError}`);
        sendArtifactError(res, 404, 'Failed to open volume.');
        return;
      }

      if (!volumeMountPath) {
        sendArtifactError(res, 404, 'Failed to open volume.');
        return;
      }
      const [fileHandle, containmentError] = await openFileWithinRoot(filePath, volumeMountPath);
      if (containmentError || !fileHandle) {
        console.log(`Failed to open volume: ${containmentError?.message || 'unknown error'}`);
        sendArtifactError(res, containmentError?.pathEscaped ? 404 : 500, 'Failed to open volume.');
        return;
      }

      try {
        // TODO: support directory and support filePath include wildcards '*'
        const stat = await fileHandle.stat();
        if (stat.isDirectory()) {
          await fileHandle.close();
          sendArtifactError(
            res,
            400,
            `Failed to open volume file ${filePath} is directory, does not support now`,
          );
          return;
        }

        const stream = fileHandle.createReadStream({ autoClose: true });
        pipePreviewResponse(stream, res, peek, (error) =>
          sendArtifactError(res, 500, `Failed to open volume: ${error}`),
        );
      } catch (error) {
        await fileHandle.close().catch(() => undefined);
        throw error;
      }
    } catch (err) {
      console.log(`Failed to open volume: ${err}`);
      sendArtifactError(res, 500, 'Failed to open volume.');
    }
  };
}

const ARTIFACTS_PROXY_DEFAULTS = {
  serviceName: 'ml-pipeline-ui-artifact',
  servicePort: '80',
};
export type NamespacedServiceGetter = (namespace: string) => string;
export interface ArtifactsProxyConfig {
  serviceName: string;
  servicePort: number;
  enabled: boolean;
}
export function loadArtifactsProxyConfig(env: ProcessEnv): ArtifactsProxyConfig {
  const {
    ARTIFACTS_SERVICE_PROXY_NAME = ARTIFACTS_PROXY_DEFAULTS.serviceName,
    ARTIFACTS_SERVICE_PROXY_PORT = ARTIFACTS_PROXY_DEFAULTS.servicePort,
    ARTIFACTS_SERVICE_PROXY_ENABLED = 'false',
  } = env;
  return {
    serviceName: ARTIFACTS_SERVICE_PROXY_NAME,
    servicePort: parseInt(ARTIFACTS_SERVICE_PROXY_PORT, 10),
    enabled: ARTIFACTS_SERVICE_PROXY_ENABLED.toLowerCase() === 'true',
  };
}

const QUERIES = {
  NAMESPACE: 'namespace',
};

export function getArtifactsProxyHandler({
  enabled,
  allowedDomain,
  namespacedServiceGetter,
}: {
  enabled: boolean;
  allowedDomain: string;
  namespacedServiceGetter: NamespacedServiceGetter;
}): Handler {
  if (!enabled) {
    return (_req, _res, next) => next();
  }
  const proxy = createProxyMiddleware({
    pathFilter: (_pathname, req) => {
      // only proxy requests with namespace query parameter
      return !!getNamespaceFromUrl(req.url || '');
    },
    changeOrigin: true,
    on: {
      proxyReq: (proxyReq) => {
        console.log('Proxied artifact request: ', proxyReq.path);
      },
      // http-proxy-middleware copies upstream headers after this outer handler
      // starts. Rewrite the proxy response itself so a tenant-side artifact
      // service cannot replace the attachment guard with `inline` or remove
      // nosniff while returning active HTML.
      proxyRes: hardenArtifactProxyResponse,
    },
    pathRewrite: (pathStr, _req) => {
      const url = new URL(pathStr || '', DUMMY_BASE_PATH);
      url.searchParams.delete(QUERIES.NAMESPACE);
      const source = url.searchParams.getAll('source');
      const bucket = url.searchParams.getAll('bucket');
      const key = url.searchParams.getAll('key');
      const download = url.searchParams.getAll('download');
      if (
        url.pathname.endsWith('/artifacts/get') &&
        source.length === 1 &&
        bucket.length === 1 &&
        key.length === 1 &&
        download.length === 1 &&
        download[0] === 'true'
      ) {
        // Keep the browser-facing request query-based so URL parsers cannot
        // normalize object-key dot segments. At the final trusted proxy hop,
        // translate it to the legacy download route understood by old tenant
        // artifact services during rolling upgrades. Encoding the complete key
        // as one path segment also keeps embedded slashes and dot segments inert
        // until Express decodes the wildcard parameter in the tenant service.
        url.searchParams.delete('source');
        url.searchParams.delete('bucket');
        url.searchParams.delete('key');
        url.searchParams.delete('download');
        const artifactPath = url.pathname.slice(0, -'get'.length);
        return (
          `${artifactPath}${encodeURIComponent(source[0])}/${encodeURIComponent(bucket[0])}/` +
          `${encodeURIComponent(key[0])}${url.search}`
        );
      }
      return url.pathname + url.search;
    },
    router: (req) => {
      const namespace = getNamespaceFromUrl(req.url || '');
      if (!namespace) {
        console.log(`namespace query param expected in ${req.url}.`);
        throw new Error(`namespace query param expected.`);
      }
      const urlStr = namespacedServiceGetter(namespace!);
      if (!isAllowedDomain(urlStr, allowedDomain)) {
        console.log(`Domain is not allowed.`);
        throw new Error(`Domain is not allowed.`);
      }
      return namespacedServiceGetter(namespace!);
    },
    target: '/artifacts',
    headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
  });
  return (req, res, next) => {
    hardenArtifactResponse(res);
    const namespace = getNamespaceFromUrl(req.url || '');
    if (namespace && !isAllowedResourceName(namespace)) {
      sendArtifactError(res, 400, 'Invalid namespace');
      return;
    }
    proxy(req, res, next);
  };
}

function getNamespaceFromUrl(path: string): string | undefined {
  // Gets namespace from query parameter "namespace"
  const params = new URL(path, DUMMY_BASE_PATH).searchParams;
  const namespaces = params.getAll('namespace');
  if (namespaces.length !== 1) {
    return undefined;
  }
  return namespaces[0] || undefined;
}

// `new URL('/path')` doesn't work, because URL only accepts full URL with scheme and hostname.
// We use the DUMMY_BASE_PATH like `new URL('/path', DUMMY_BASE_PATH)`, so that URL can parse paths
// properly.
const DUMMY_BASE_PATH = 'http://dummy-base-path';

export function getArtifactServiceGetter({ serviceName, servicePort }: ArtifactsProxyConfig) {
  return (namespace: string) => `http://${serviceName}.${namespace}:${servicePort}`;
}

export const TEST_ONLY = {
  getMinioArtifactHandler,
};
