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
  createArtifactStoreClient,
  createMinioClient,
  getObjectStream,
  isNoSuchKeyError,
  listObjectsUnderPrefix,
  ArtifactStoreConfigurationError,
  getArtifactStoreOriginAfterRewrite,
  summarizeDirectoryUnderPrefix,
} from '../minio-helper.js';
import type { ArtifactStoreEndpointPolicy, MinioRequestConfig } from '../minio-helper.js';
import { isAWSS3Endpoint, isOfficialAWSS3ServiceEndpoint } from '../aws-helper.js';
import { parseGoBoolean } from '../helpers/provider-options.js';
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
import {
  DEFAULT_GCS_UNIVERSE_DOMAIN,
  getGCSClient,
  listGCSObjectNames,
  downloadGCSObjectStream,
} from '../gcs-helper.js';
import type { GCSClient } from '../gcs-helper.js';

import { isAllowedDomain, isTrustedArtifactEndpoint } from './domain-checker.js';
import { getK8sSecret } from '../k8s-helper.js';
import { CredentialBody } from 'google-auth-library';
import { AuthorizeFn } from '../helpers/auth.js';
import { validateArtifactNamespace } from '../helpers/artifact-validator.js';
import {
  ArtifactCoordinates,
  buildArtifactCoordinateUri,
  normalizeArtifactStorageCoordinates,
  resolveArtifactCoordinates,
} from '../helpers/artifact-coordinates.js';
import { applyArtifactPathPolicy, ARTIFACT_PATH_POLICIES } from '../helpers/artifact-path.js';
import {
  ArtifactSource,
  isArtifactSource,
  isLauncherArtifactSource,
  LauncherArtifactSource,
  requiresArtifactOwnershipValidation,
} from '../helpers/artifact-sources.js';
import {
  AuthorizeRequestResources,
  AuthorizeRequestVerb,
} from '../src/generated/apis/auth/index.js';
import {
  getLauncherProviderInfo,
  LauncherConfigError,
  LauncherConfigValidationError,
} from '../helpers/launcher-config.js';

const ARTIFACT_QUERY_PARAMETER_NAMES = [
  'source',
  'bucket',
  'key',
  'keyEncoding',
  'uriKey',
  'artifactUriQuery',
  'providerInfo',
  'namespace',
  'peek',
  'download',
] as const;
const MALFORMED_ARTIFACT_KEY_MESSAGE =
  'Artifact storage key contains malformed or noncanonical URI path encoding. Use the canonical artifact URI and retry.';
const INVALID_ARTIFACT_PATH_ENCODING_MESSAGE =
  'Artifact path has malformed or noncanonical URI encoding. Use the canonical artifact URI and retry.';

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
    disable_https?: string;
    anonymous?: string;
    forcePathStyle?: string;
    s3ForcePathStyle?: string;
    use_path_style?: string;
    nativeQuery?: string;
    maxRetries?: string;
  };
}

export interface GCSProviderInfo {
  Provider: string;
  Params: {
    fromEnv: string;
    access_id?: string;
    // Go Cloud URI-query compatibility only. The launcher GCS provider configuration does not
    // emit anonymous; it reaches this boundary through gs://...?anonymous=true when no provider
    // credential policy is configured.
    anonymous?: string;
    universe_domain?: string;
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

const GCS_PROVIDER_INFO_PARAMS = new Set([
  'access_id',
  'anonymous',
  'fromEnv',
  'secretName',
  'tokenKey',
  'universe_domain',
]);

class NamespaceIsolatedProviderRequiredError extends Error {}

function retainDestinationSafeProviderInfo(providerInfoString: string): string {
  const providerInfo = parseJSONString<S3ProviderInfo | GCSProviderInfo>(providerInfoString);
  if (!providerInfo?.Params) {
    return '';
  }

  if (providerInfo.Provider === 'gs') {
    const params = (providerInfo as GCSProviderInfo).Params;
    if (params.anonymous !== undefined) {
      parseGoBoolean(params.anonymous, 'anonymous');
    }
    if (
      params.fromEnv === 'false' ||
      params.secretName !== undefined ||
      params.tokenKey !== undefined
    ) {
      throw new NamespaceIsolatedProviderRequiredError(
        'Secret-backed GCS provider settings require the namespace-isolated artifact proxy.',
      );
    }
    const safe = { ...params };
    delete safe.secretName;
    delete safe.tokenKey;
    return JSON.stringify({ Provider: 'gs', Params: { ...safe, fromEnv: 'true' } });
  }

  const params = (providerInfo as S3ProviderInfo).Params;
  if (params.anonymous !== undefined) {
    parseGoBoolean(params.anonymous, 'anonymous');
  }
  // Without a destination allowlist, a customer-selected S3 endpoint would make the shared UI an
  // object-store proxy. Endpoint-free settings retain the shared service's trusted destination.
  if (params.endpoint) {
    throw new NamespaceIsolatedProviderRequiredError(
      'Custom S3-compatible endpoints require the namespace-isolated artifact proxy.',
    );
  }
  if (
    params.fromEnv === 'false' ||
    params.secretName !== undefined ||
    params.accessKeyKey !== undefined ||
    params.secretKeyKey !== undefined
  ) {
    throw new NamespaceIsolatedProviderRequiredError(
      'Secret-backed S3 provider settings require the namespace-isolated artifact proxy.',
    );
  }
  const safe = { ...params };
  delete safe.accessKeyKey;
  delete safe.secretKeyKey;
  delete safe.secretName;
  return JSON.stringify({
    Provider: providerInfo.Provider,
    Params: { ...safe, fromEnv: 'true' },
  });
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
 * never reads Secrets from a customer namespace. In shared direct mode an
 * explicit Secret or custom destination is rejected rather than substituted
 * with central credentials; those settings require the namespace-isolated
 * artifact proxy.
 * See: https://github.com/kubeflow/pipelines/pull/12860
 *
 * Security: This addresses the vulnerability where the namespace parameter
 * could be manipulated to access artifacts from other namespaces.
 * See https://github.com/kubeflow/pipelines/issues/9889
 *
 * @param authorizeFn The authorization function to validate permissions
 * @param authEnabled Whether authorization is enabled
 * @param kubeflowUserIdHeader The header name containing the user identity
 * @param apiServerAddress KFP API server address used for namespace-ownership validation (#9889).
 */
export function getArtifactsAuthMiddleware(
  authorizeFn: AuthorizeFn,
  authEnabled: boolean,
  kubeflowUserIdHeader: string,
  apiServerAddress?: string,
  allowNamespaceIsolatedCustomRoots = false,
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

    const userIdHeader = request.headers[kubeflowUserIdHeader.toLowerCase()];
    const userId = Array.isArray(userIdHeader) ? userIdHeader[0] : userIdHeader;
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

    const coordinates = resolveArtifactCoordinates(request);
    if (coordinates === null) {
      console.warn(
        `[SECURITY] Malformed or noncanonical percent-encoding in artifact path. ` +
          `User: ${userId}, Path: ${request.path}`,
      );
      sendArtifactError(response, 400, INVALID_ARTIFACT_PATH_ENCODING_MESSAGE);
      return;
    }

    if (
      !coordinates ||
      !isArtifactSource(coordinates.source) ||
      !coordinates.bucket ||
      !coordinates.key
    ) {
      console.warn(
        `[SECURITY] Rejected artifact request with coordinates that cannot be authorized. ` +
          `User: ${userId}, Namespace: ${namespace}, Path: ${request.path}`,
      );
      sendArtifactError(
        response,
        403,
        'Artifact source, bucket, and key are required and must use a supported storage source',
      );
      return;
    }

    if (coordinates.source === 'volume' && !allowNamespaceIsolatedCustomRoots) {
      console.warn(
        `[SECURITY] Rejected direct volume artifact access through the shared UI server. ` +
          `User: ${userId}, Namespace: ${namespace}, Path: ${request.path}`,
      );
      sendArtifactError(
        response,
        403,
        'Volume artifacts require a namespace-isolated artifact service in multi-user mode',
      );
      return;
    }

    if (apiServerAddress) {
      if (requiresArtifactOwnershipValidation(coordinates.source)) {
        const artifactUri = buildArtifactCoordinateUri(coordinates);
        const validationHeaders = { [kubeflowUserIdHeader]: userId };
        const validation = await validateArtifactNamespace(
          apiServerAddress,
          artifactUri,
          namespace,
          validationHeaders,
          allowNamespaceIsolatedCustomRoots,
        );

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

    response.locals.authorizedArtifactUri = buildArtifactCoordinateUri(coordinates);

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
    allowedEndpoints?: string[];
    allowOfficialAwsEndpoints?: boolean;
    allowedGcsUniverseDomains?: string[];
  };
  tryExtract: boolean;
  useParameter: boolean;
  options: UIConfigs;
}): Handler {
  const {
    aws,
    http,
    minio,
    allowedDomain,
    allowedEndpoints = [],
    allowOfficialAwsEndpoints,
    allowedGcsUniverseDomains,
  } = artifactsConfigs;
  // Capture operator-owned trust anchors when the handler is built. Provider
  // parsing must never be able to mutate the configuration used to authorize a
  // later request.
  const configuredEndpoints = {
    minio: getArtifactStoreOriginAfterRewrite(minio),
    s3: getArtifactStoreOriginAfterRewrite(aws),
  };
  const configuredTlsEndpoints = {
    minio: getArtifactStoreOriginAfterRewrite({ ...minio, useSSL: true }),
    s3: getArtifactStoreOriginAfterRewrite({ ...aws, useSSL: true }),
  };
  const configuredAdditionalEndpoints = [...allowedEndpoints];
  const authorizeEndpoint: ArtifactStoreEndpointPolicy = (endpoint) => {
    if (endpoint.providerType !== 's3' && endpoint.providerType !== 'minio') {
      throw new ArtifactStoreConfigurationError('Invalid artifact store provider type');
    }
    const configuredEndpoint = configuredEndpoints[endpoint.providerType];
    if (
      !endpoint.hasEndpointOverride &&
      !endpoint.useSSL &&
      configuredEndpoint?.startsWith('https:')
    ) {
      throw new ArtifactStoreConfigurationError(
        'Artifact store TLS override conflicts with server configuration',
      );
    }
    if (!endpoint.useSSL && isAWSS3Endpoint(endpoint.endPoint)) {
      throw new ArtifactStoreConfigurationError('AWS S3 provider endpoints must use HTTPS');
    }
    const trustsAwsRegionalEndpoint =
      endpoint.providerType === 's3' &&
      allowOfficialAwsEndpoints &&
      isOfficialAwsS3Origin(configuredEndpoint) &&
      isOfficialAwsS3Origin(endpoint.origin);
    const trustedEndpoints = [
      ...(configuredEndpoint ? [configuredEndpoint] : []),
      ...configuredAdditionalEndpoints,
    ];
    if (
      !trustsAwsRegionalEndpoint &&
      !(
        !endpoint.hasEndpointOverride &&
        endpoint.useSSL &&
        endpoint.origin === configuredTlsEndpoints[endpoint.providerType]
      ) &&
      !isTrustedArtifactEndpoint(endpoint.origin, trustedEndpoints)
    ) {
      throw new ArtifactStoreConfigurationError(
        'Artifact store endpoint is not allowed; add its exact origin to ALLOWED_ARTIFACT_ENDPOINTS',
      );
    }
  };
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
    const {
      source,
      bucket,
      key,
      keyEncoding,
      artifactUriQuery,
      peek,
      providerInfo,
      namespace,
      download,
    } = artifactRequest;
    const routeCoordinates =
      useParameter ||
      req.path.endsWith('/artifacts/get') ||
      req.path.endsWith('/pipeline/artifacts/get') ||
      isLauncherArtifactSource(source)
        ? resolveArtifactCoordinates(req)
        : undefined;
    if (routeCoordinates === null) {
      sendArtifactError(res, 400, INVALID_ARTIFACT_PATH_ENCODING_MESSAGE);
      return;
    }
    const trustedRouteCoordinates: ArtifactCoordinates<ArtifactSource> | undefined =
      routeCoordinates &&
      isArtifactSource(routeCoordinates.source) &&
      routeCoordinates.bucket &&
      routeCoordinates.key
        ? { ...routeCoordinates, source: routeCoordinates.source }
        : undefined;
    const coordinates: ArtifactCoordinates<ArtifactSource> = trustedRouteCoordinates ?? {
      source,
      bucket,
      key,
      keyEncoding,
      artifactUriQuery,
    };
    const requestedArtifactUri = buildArtifactCoordinateUri(coordinates);
    // The authorization middleware and storage handler parse independently. Pin artifact identity
    // to the exact URI that was authorized; storage-key decoding below is then determined only by
    // this route's trusted keyEncoding classification.
    if (options.auth.enabled && res.locals.authorizedArtifactUri !== requestedArtifactUri) {
      console.warn(
        '[SECURITY] Rejected artifact request whose coordinates changed after authorization',
      );
      sendArtifactError(res, 403, 'Artifact request coordinates changed after authorization');
      return;
    }
    const setArtifactFilename = (transformed: boolean) => {
      const keyBaseName = storageKey.replace(/\/+$/, '').split('/').pop() || 'artifact';
      res.setHeader(
        'Content-Disposition',
        buildAttachmentDisposition(transformed ? 'artifact' : keyBaseName),
      );
    };
    if (!isAllowedResourceName(bucket)) {
      sendArtifactError(res, 500, 'Invalid bucket name');
      return;
    }
    if (key.length > 1024) {
      sendArtifactError(res, 500, 'Object key too long');
      return;
    }
    let storageKey = key;
    if (isLauncherArtifactSource(source)) {
      try {
        storageKey = normalizeArtifactStorageCoordinates({ ...coordinates, source }).key;
      } catch {
        sendArtifactError(res, 400, MALFORMED_ARTIFACT_KEY_MESSAGE);
        return;
      }
    }
    console.log(`Getting storage artifact at: ${source}: ${bucket}/${storageKey}`);
    if (source !== 'minio' && source !== 's3') {
      setArtifactFilename(false);
    }

    // Security: The ml-pipeline-ui service account is only permitted to read Secrets from its own
    // (server) namespace. For customer namespaces, retain only provider settings that keep the
    // shared service's trusted destination. Secret-backed credentials and customer-selected S3
    // endpoints require the namespace-isolated artifact proxy; otherwise the shared UI could read
    // customer Secrets or send ambient credentials to an untrusted destination. See:
    // https://github.com/kubeflow/pipelines/pull/12860
    // A missing namespace only occurs when auth is disabled (single-tenant): the
    // auth middleware rejects namespace-less requests whenever auth is enabled, so
    // treating it as server-local cannot be triggered by a multi-user caller.
    const allowProviderSecrets = !namespace || namespace === options.server.serverNamespace;
    let resolvedProviderInfo = '';
    if (isLauncherArtifactSource(source)) {
      try {
        resolvedProviderInfo =
          (await getLauncherProviderInfo(
            { ...coordinates, key: storageKey, keyEncoding: 'storage', source },
            namespace,
          )) || '';
      } catch (error) {
        // Direct mode must not substitute central credentials when native provider validation or
        // trusted launcher configuration fails. The namespace-isolated proxy has its own explicit
        // delegation path for ConfigMap availability failures.
        const status = error instanceof LauncherConfigValidationError ? 400 : 500;
        sendArtifactError(
          res,
          status,
          `Failed to resolve artifact storage configuration. Check the kfp-launcher providers configuration: ${error}`,
        );
        return;
      }
    }

    // Preserve legacy single-user store_session_info links only when trusted launcher
    // configuration does not select a provider. Authenticated and proxied requests never
    // accept browser-supplied provider authority.
    if (!options.auth.enabled && !resolvedProviderInfo) {
      resolvedProviderInfo = providerInfo;
    }

    let effectiveProviderInfo: string;
    try {
      effectiveProviderInfo = allowProviderSecrets
        ? resolvedProviderInfo
        : retainDestinationSafeProviderInfo(resolvedProviderInfo);
    } catch (error) {
      if (error instanceof NamespaceIsolatedProviderRequiredError) {
        sendArtifactError(
          res,
          400,
          `${error.message} Enable the namespace-isolated artifact proxy and retry the request.`,
        );
        return;
      }
      sendArtifactError(
        res,
        400,
        `Invalid artifact provider configuration. Correct it and retry: ${error}`,
      );
      return;
    }
    // The client resolves provider transport options once and invokes authorizeEndpoint before
    // reading credentials. Validate that same request-local configuration in every provider mode.

    const retryAbortController = new AbortController();
    const abortRetry = () => retryAbortController.abort();
    const cleanupRetryAbort = () => req.removeListener('aborted', abortRetry);
    if (req.aborted) {
      retryAbortController.abort();
    } else {
      req.once('aborted', abortRetry);
      res.once('finish', cleanupRetryAbort);
      res.once('close', () => {
        if (!res.writableFinished) {
          retryAbortController.abort();
        }
        cleanupRetryAbort();
      });
    }
    let client: MinioClient;
    switch (source) {
      case 'gcs':
        await getGCSArtifactHandler(
          { bucket, key: storageKey },
          peek,
          effectiveProviderInfo,
          namespace,
          allowedGcsUniverseDomains,
          useParameter || download,
        )(req, res);
        break;
      case 'minio':
        try {
          client = await createArtifactStoreClient(
            { minio, s3: aws },
            'minio',
            effectiveProviderInfo,
            namespace,
            retryAbortController.signal,
            authorizeEndpoint,
          );
        } catch (e) {
          sendArtifactError(
            res,
            e instanceof ArtifactStoreConfigurationError ? 400 : 500,
            `Failed to initialize Minio Client for Minio Provider: ${e}`,
          );
          return;
        }
        await getMinioArtifactHandler(
          {
            bucket,
            client,
            key: storageKey,
            signal: retryAbortController.signal,
            tryExtract: tryExtract && !download,
            onTransformationDetermined: setArtifactFilename,
          },
          peek,
        )(req, res);
        break;
      case 's3':
        try {
          client = await createMinioClient(
            aws,
            's3',
            effectiveProviderInfo,
            namespace,
            undefined,
            retryAbortController.signal,
            authorizeEndpoint,
          );
        } catch (e) {
          sendArtifactError(
            res,
            e instanceof ArtifactStoreConfigurationError ? 400 : 500,
            `Failed to initialize Minio Client for S3 Provider: ${e}`,
          );
          return;
        }
        await getMinioArtifactHandler(
          {
            bucket,
            client,
            key: storageKey,
            signal: retryAbortController.signal,
            tryExtract: tryExtract && !download,
            onTransformationDetermined: setArtifactFilename,
          },
          peek,
        )(req, res);
        break;
      case 'http':
      case 'https': {
        const httpUrl = getHttpUrl(
          source,
          http.baseUrl || '',
          bucket,
          coordinates.uriKey ?? key,
          coordinates.uriKey ? 'uri' : 'storage',
        );
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

function isOfficialAwsS3Origin(origin: string | undefined): boolean {
  if (!origin) {
    return false;
  }
  try {
    const parsed = new URL(origin);
    if (parsed.protocol !== 'https:' || (parsed.port && parsed.port !== '443')) {
      return false;
    }
    return isOfficialAWSS3ServiceEndpoint(parsed.hostname);
  } catch {
    return false;
  }
}

type ArtifactRequest =
  | {
      source: ArtifactSource;
      bucket: string;
      key: string;
      keyEncoding: 'storage' | 'uri';
      artifactUriQuery: string;
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

  const artifactUriQuery = getOptionalRequestString(req.query.artifactUriQuery, 'artifactUriQuery');
  if ('error' in artifactUriQuery) {
    return artifactUriQuery;
  }

  const keyEncoding = getOptionalRequestString(req.query.keyEncoding, 'keyEncoding');
  if ('error' in keyEncoding) {
    return keyEncoding;
  }
  if (keyEncoding.value && keyEncoding.value !== 'storage' && keyEncoding.value !== 'uri') {
    return {
      error: {
        status: 400,
        message: 'Artifact key encoding must be storage or uri. Use a supported artifact link.',
      },
    };
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
    keyEncoding: useParameter ? 'storage' : keyEncoding.value === 'uri' ? 'uri' : 'storage',
    artifactUriQuery: artifactUriQuery.value ?? '',
    peek: parsePeekValue(peek.value),
    providerInfo: providerInfo.value ?? '',
    namespace: namespace.value || defaultNamespace,
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

/**
 * Returns the http/https url to retrieve a kfp artifact (of the form: `${source}://${baseUrl}${bucket}/${key}`)
 * @param source "http" or "https".
 * @param baseUrl string to prefix the url.
 * @param bucket name of the bucket.
 * @param key path to the artifact.
 */
function getHttpUrl(
  source: 'http' | 'https',
  baseUrl: string,
  bucket: string,
  key: string,
  keyEncoding: 'storage' | 'uri' = 'storage',
) {
  const configuredBaseUrl = baseUrl.trim().replace(/^\/+/, '');
  if (!configuredBaseUrl) {
    return undefined;
  }
  try {
    const artifactUrl = new URL(`${source}://${configuredBaseUrl}`);
    const storageKey = keyEncoding === 'uri' ? decodeURIComponent(key) : key;
    const safeKey = applyArtifactPathPolicy(storageKey, ARTIFACT_PATH_POLICIES.http);
    if (safeKey === undefined) {
      return undefined;
    }
    const escapedKey = keyEncoding === 'uri' ? key : safeKey.replace(/%/g, '%25');
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
  return async (req: Request, res: Response) => {
    let handlingError = false;
    const handleObjectFailure = async (err: unknown) => {
      if (handlingError || isArtifactRequestCancelled(req, res, err)) {
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
            if (isArtifactRequestCancelled(req, res, summaryErr)) return;
            console.error(summaryErr);
            sendArtifactError(res, 500, `Failed to summarize directory: ${summaryErr}`);
          }
          return;
        }
        try {
          await streamDirectoryAsTarGz(options, res);
        } catch (tarErr) {
          if (
            tarErr instanceof ArtifactResponseClosedError ||
            isArtifactRequestCancelled(req, res, tarErr)
          ) {
            return;
          }
          console.error(tarErr);
          sendArtifactError(res, 500, `Failed to get object in bucket: ${tarErr}`);
        }
        return;
      }
      if (isArtifactRequestCancelled(req, res, err)) return;
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

function isArtifactRequestCancelled(req: Request, res: Response, error: unknown): boolean {
  return req.aborted || res.destroyed || (error instanceof Error && error.name === 'AbortError');
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
  return applyArtifactPathPolicy(name, ARTIFACT_PATH_POLICIES.tarEntry) || null;
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
  options: {
    anonymous?: boolean;
    client?: GCSClient;
    credentials?: CredentialBody;
    universeDomain?: string;
  },
): Promise<Buffer> {
  const stream = await downloadGCSObjectStream({ bucket, objectName, ...options });
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
  allowedUniverseDomains: string[] = ['googleapis.com'],
  isDownloadRoute: boolean = false,
) {
  const { key, bucket } = options;
  return async (_: Request, res: Response) => {
    try {
      let anonymous = false;
      let credentials: CredentialBody | undefined;
      let universeDomain = DEFAULT_GCS_UNIVERSE_DOMAIN;
      if (providerInfoString) {
        const providerInfo = parseJSONString<GCSProviderInfo>(providerInfoString);
        if (!providerInfo) {
          throw new Error('Failed to parse GCS provider info. Correct it and retry.');
        }
        const unsupportedParams = Object.keys(providerInfo.Params).filter(
          (key) => !GCS_PROVIDER_INFO_PARAMS.has(key),
        );
        if (unsupportedParams.length) {
          throw new Error(
            `Unsupported GCS artifact read option${unsupportedParams.length === 1 ? '' : 's'}: ${unsupportedParams
              .sort()
              .join(', ')}. Remove unsupported options and retry.`,
          );
        }
        const anonymousParam = providerInfo?.Params.anonymous;
        anonymous =
          (anonymousParam ? parseGoBoolean(anonymousParam, 'anonymous') : false) ||
          providerInfo?.Params.access_id === '-';
        universeDomain =
          providerInfo.Params.universe_domain?.toLowerCase() || DEFAULT_GCS_UNIVERSE_DOMAIN;
        if (providerInfo && !anonymous && providerInfo.Params.fromEnv === 'false') {
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
      if (!allowedUniverseDomains.includes(universeDomain)) {
        sendArtifactError(
          res,
          400,
          `GCS universe_domain "${universeDomain}" is not allowed. Add it to ALLOWED_GCS_UNIVERSE_DOMAINS and retry.`,
        );
        return;
      }
      // The operator allowlist is the destination trust grant for both anonymous and authenticated
      // reads. This preserves GDC/air-gapped support without allowing artifact URIs to choose an
      // arbitrary host for the shared UI's ADC bearer token.
      // Read all files that match the key pattern, which can include wildcards '*'.
      // The way this works is we list all paths whose prefix is the substring
      // of the pattern until the first wildcard, then we create a regular
      // expression out of the pattern, escaping all non-wildcard characters,
      // and we use it to match all enumerated paths.
      const prefix = key.indexOf('*') > -1 ? key.substr(0, key.indexOf('*')) : key;
      const client = anonymous ? undefined : await getGCSClient(credentials, universeDomain);
      const universeOptions = { universeDomain };
      const accessOptions = anonymous
        ? { anonymous: true, ...universeOptions }
        : { client, credentials, ...universeOptions };
      const matchingFiles = (
        await listGCSObjectNames({
          ...accessOptions,
          bucket,
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
          ...accessOptions,
          bucket,
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
          contents.push(await readGCSObject(bucket, fileName, accessOptions));
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
        contents += (await readGCSObject(bucket, fileName, accessOptions)).toString().trim() + '\n';
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
  return async (req, res, next) => {
    hardenArtifactResponse(res);
    const namespace = getNamespaceFromUrl(req.url || '');
    if (namespace && !isAllowedResourceName(namespace)) {
      sendArtifactError(res, 400, 'Invalid namespace');
      return;
    }
    if (namespace) {
      const url = new URL(req.url || '', DUMMY_BASE_PATH);
      url.searchParams.delete('providerInfo');
      const resolvedCoordinates = resolveArtifactCoordinates({
        path: url.pathname,
        query: {
          source: url.searchParams.get('source') || undefined,
          bucket: url.searchParams.get('bucket') || undefined,
          key: url.searchParams.get('key') || undefined,
          keyEncoding: url.searchParams.get('keyEncoding') || undefined,
          uriKey: url.searchParams.get('uriKey') || undefined,
          artifactUriQuery: url.searchParams.get('artifactUriQuery') || undefined,
        },
      });
      if (resolvedCoordinates === null) {
        sendArtifactError(res, 400, INVALID_ARTIFACT_PATH_ENCODING_MESSAGE);
        return;
      }
      const coordinates: ArtifactCoordinates<LauncherArtifactSource> | undefined =
        resolvedCoordinates &&
        isLauncherArtifactSource(resolvedCoordinates.source) &&
        resolvedCoordinates.bucket &&
        resolvedCoordinates.key
          ? {
              source: resolvedCoordinates.source,
              bucket: resolvedCoordinates.bucket,
              key: resolvedCoordinates.key,
              keyEncoding: resolvedCoordinates.keyEncoding,
              uriKey: resolvedCoordinates.uriKey,
              artifactUriQuery: resolvedCoordinates.artifactUriQuery,
            }
          : undefined;
      if (coordinates) {
        let storageCoordinates: ArtifactCoordinates<LauncherArtifactSource>;
        try {
          storageCoordinates = normalizeArtifactStorageCoordinates(coordinates);
        } catch {
          sendArtifactError(res, 400, MALFORMED_ARTIFACT_KEY_MESSAGE);
          return;
        }
        try {
          const providerInfo = await getLauncherProviderInfo(storageCoordinates, namespace);
          if (providerInfo) {
            url.searchParams.set('providerInfo', providerInfo);
          }
        } catch (error) {
          if (error instanceof LauncherConfigError) {
            // The namespace-isolated service owns credential resolution, so omitting
            // providerInfo delegates to credentials inside the same namespace boundary.
            console.warn(
              `Unable to resolve the ${namespace} kfp-launcher providers configuration; ` +
                `forwarding the request without providerInfo so the namespaced artifact ` +
                `service can use its environment credentials. ${error.message}`,
            );
          } else {
            sendArtifactError(
              res,
              500,
              `Failed to resolve artifact storage configuration. Check the kfp-launcher providers configuration: ${error}`,
            );
            return;
          }
        }
      }
      if (url.pathname.endsWith('/artifacts/get') && url.searchParams.get('download') === 'true') {
        if (!resolvedCoordinates || !isArtifactSource(resolvedCoordinates.source)) {
          sendArtifactError(res, 400, INVALID_ARTIFACT_PATH_ENCODING_MESSAGE);
          return;
        }
        // Old tenant services only support raw downloads on the path route. Keep a canonical
        // path that new tenant services accept too, after validating the source's path policy.
        // Whole-key encoding would introduce encoded separators rejected by native coordinates.
        let storageKey: string;
        try {
          storageKey = normalizeArtifactStorageCoordinates(resolvedCoordinates).key;
        } catch {
          sendArtifactError(res, 400, MALFORMED_ARTIFACT_KEY_MESSAGE);
          return;
        }
        const policy = isLauncherArtifactSource(resolvedCoordinates.source)
          ? ARTIFACT_PATH_POLICIES.ownership
          : resolvedCoordinates.source === 'volume'
            ? ARTIFACT_PATH_POLICIES.volume
            : ARTIFACT_PATH_POLICIES.http;
        if (applyArtifactPathPolicy(storageKey, policy) === undefined) {
          sendArtifactError(res, 400, MALFORMED_ARTIFACT_KEY_MESSAGE);
          return;
        }
        if (resolvedCoordinates.source === 'volume') {
          const normalizedKey = storageKey
            .split('/')
            .filter((segment) => segment !== '.')
            .join('/');
          if (normalizedKey !== storageKey && !url.searchParams.has('uriKey')) {
            url.searchParams.set('uriKey', encodeURI(storageKey));
          }
          storageKey = normalizedKey;
        }
        const pathKey = encodeURI(storageKey).replace(/\?/g, '%3F').replace(/#/g, '%23');
        url.pathname =
          url.pathname.slice(0, -'get'.length) +
          `${encodeURIComponent(resolvedCoordinates.source)}/${encodeURIComponent(resolvedCoordinates.bucket)}/${pathKey}`;
        for (const parameter of ['source', 'bucket', 'key', 'keyEncoding', 'download']) {
          url.searchParams.delete(parameter);
        }
      }
      updateProxyRequestUrl(req, url);
    }
    proxy(req, res, next);
  };
}

function updateProxyRequestUrl(request: Request, url: URL): void {
  const rewrittenUrl = url.pathname + url.search;
  request.url = rewrittenUrl;
  request.originalUrl = rewrittenUrl;
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
