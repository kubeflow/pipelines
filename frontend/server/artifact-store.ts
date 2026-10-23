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
import { Readable, Transform, PassThrough, pipeline } from 'stream';
import type { Agent as HttpAgent } from 'http';
import type { Agent as HttpsAgent } from 'https';
import { isIP } from 'node:net';
import * as tar from 'tar-stream';
import peek from 'peek-stream';
import gunzip from 'gunzip-maybe';
import { URL } from 'url';
import { AsyncLocalStorage } from 'async_hooks';
import { S3Client, GetObjectCommand, ListObjectsV2Command } from '@aws-sdk/client-s3';
import { NodeHttpHandler } from '@smithy/node-http-handler';
import { isAWSS3Endpoint } from './aws-helper.js';
import type { S3ProviderInfo } from './handlers/artifacts.js';
import { getK8sSecret } from './k8s-helper.js';
import { parseJSONString } from './utils.js';
import { fromNodeProviderChain } from '@aws-sdk/credential-providers';
import { parseGoBoolean } from './helpers/provider-options.js';
/** ArtifactReadRequest describes the info required to retrieve an artifact. */
export interface ArtifactReadRequest {
  key: string;
  store: ArtifactStore;
  signal?: AbortSignal;
  tryExtract?: boolean;
  /** Receives asynchronous storage or transformation failures from the returned stream. */
  onError?: (error: Error) => void;
  onTransformationDetermined?: (transformed: boolean) => void;
}

/** Bucket-bound object reads and recursive listings, independent of the storage SDK. */
export interface ArtifactStore {
  openRead(key: string, options?: { signal?: AbortSignal }): Promise<Readable>;
  list(
    prefix: string,
    options?: { signal?: AbortSignal; pageSize?: number },
  ): AsyncIterable<{ key: string; size: number }>;
}

export interface ArtifactStoreOptions {
  accessKey?: string;
  secretKey?: string;
  sessionToken?: string;
  region?: string;
  port?: number;
  useSSL?: boolean;
  pathStyle?: boolean;
  transportAgent?: HttpAgent | HttpsAgent;
  endPoint: string;
  endpointBasePath?: string;
  endpointAuthority?: string;
  endpointRewrite?: string;
  endpointTruncatesAtDelimiter?: boolean;
  maxAttempts?: number;
  retrySignal?: AbortSignal;
}

export interface Credentials {
  accessKeyId: string;
  secretAccessKey: string;
  sessionToken?: string;
  expiration?: Date;
}

export interface ArtifactStoreEndpoint {
  endPoint: string;
  port?: number;
  useSSL: boolean;
  origin: string;
}

export class ArtifactStoreConfigurationError extends Error {}

export type ArtifactStoreEndpointPolicy = (
  endpoint: ArtifactStoreEndpoint & { providerType: string; hasEndpointOverride: boolean },
) => void;

/** Captures the operator's effective destination using the same rewrite as client creation. */
export function getArtifactStoreOriginAfterRewrite(
  config: ArtifactStoreOptions,
): string | undefined {
  return getArtifactStoreOrigin(applyEndpointRewrite({ ...config }));
}

export function parseArtifactStoreEndpoint(
  endpoint: string,
  insecure: boolean,
): ArtifactStoreEndpoint | undefined {
  try {
    const hasExplicitProtocol = /^[a-z][a-z0-9+.-]*:\/\//i.test(endpoint);
    const expectedProtocol = insecure ? 'http:' : 'https:';
    const parsed = new URL(hasExplicitProtocol ? endpoint : `${expectedProtocol}//${endpoint}`);
    if (
      parsed.protocol !== expectedProtocol ||
      parsed.username ||
      parsed.password ||
      parsed.pathname !== '/' ||
      parsed.search ||
      parsed.hash
    ) {
      return undefined;
    }
    return {
      endPoint: parsed.hostname,
      port: parsed.port ? Number(parsed.port) : undefined,
      useSSL: !insecure,
      origin: parsed.origin,
    };
  } catch {
    return undefined;
  }
}

/** Returns the effective origin of an operator-configured object-store client. */
export function getArtifactStoreOrigin({
  endPoint,
  port,
  useSSL,
}: Pick<ArtifactStoreOptions, 'endPoint' | 'port' | 'useSSL'>): string | undefined {
  if (port !== undefined && (!Number.isInteger(port) || port < 1 || port > 65535)) {
    return undefined;
  }
  const insecure = useSSL === false;
  const endpoint = parseArtifactStoreEndpoint(
    isIP(endPoint) === 6 ? `[${endPoint}]` : endPoint,
    insecure,
  );
  if (!endpoint) {
    return undefined;
  }
  const origin = new URL(endpoint.origin);
  if (port !== undefined && !origin.port) {
    origin.port = String(port);
  }
  return origin.origin;
}

const MAX_S3_ATTEMPTS = 10;
const MAX_S3_RETRY_ATTEMPTS_PER_REQUEST = 10;
const S3_TRANSPORT_ERROR = Symbol('S3 transport error');

const UNSUPPORTED_S3_READ_OPTIONS = new Set(['role']);
const UNSUPPORTED_S3_BOOLEAN_READ_OPTIONS = [
  'accelerate',
  'dualstack',
  'fips',
  'hostname_immutable',
] as const;

function rejectUnsupportedS3ReadOptions(providerInfo: S3ProviderInfo): void {
  const params = providerInfo.Params as Record<string, string | undefined>;
  if (
    params.ssetype !== undefined &&
    !['aes256', 'aws:kms', 'aws:kms:dsse'].includes(params.ssetype.toLowerCase())
  ) {
    throw new Error(
      `Invalid value for provider option ssetype: ${params.ssetype}. ` +
        'Use AES256, aws:kms, or aws:kms:dsse.',
    );
  }
  if (params.kmskeyid !== undefined && params.kmskeyid === '') {
    throw new Error(
      'Invalid empty value for provider option kmskeyid. Remove it or provide a key ID.',
    );
  }
  const unsupported = Object.keys(params)
    .filter((key) => UNSUPPORTED_S3_READ_OPTIONS.has(key))
    .sort();
  if (params.profile !== undefined && params.profile !== '') {
    unsupported.push('profile');
  }
  validateNeutralS3Option(
    params.request_checksum_calculation,
    'request_checksum_calculation',
    unsupported,
  );
  validateNeutralS3Option(
    params.response_checksum_validation,
    'response_checksum_validation',
    unsupported,
  );
  for (const option of UNSUPPORTED_S3_BOOLEAN_READ_OPTIONS) {
    const value = params[option];
    if (value !== undefined && parseGoBoolean(value, option)) {
      unsupported.push(option);
    }
  }
  const rateLimitCapacity = params.rate_limiter_capacity;
  if (rateLimitCapacity !== undefined) {
    if (!/^[+-]?\d+$/.test(rateLimitCapacity)) {
      throw new Error(
        `Invalid integer value for provider option rate_limiter_capacity: ${rateLimitCapacity}`,
      );
    }
    const parsedCapacity = Number(rateLimitCapacity);
    if (
      !Number.isSafeInteger(parsedCapacity) ||
      parsedCapacity < -2147483648 ||
      parsedCapacity > 2147483647
    ) {
      throw new Error(
        `Invalid integer value for provider option rate_limiter_capacity: ${rateLimitCapacity}`,
      );
    }
    if (parsedCapacity > 0) {
      unsupported.push('rate_limiter_capacity');
    }
  }
  unsupported.sort();
  if (unsupported.length) {
    throw new Error(
      `Unsupported S3 artifact read option${unsupported.length === 1 ? '' : 's'}: ${unsupported.join(
        ', ',
      )}. Remove the option or configure an artifact store supported by the frontend.`,
    );
  }
  if (params.endpoint) {
    parseProviderEndpoint(params.endpoint, params.nativeQuery === 'true');
  }
}

function validateNeutralS3Option(
  value: string | undefined,
  option: string,
  unsupported: string[],
): void {
  if (value === undefined) {
    return;
  }
  switch (value.toLowerCase()) {
    case 'when_supported':
      return;
    case 'when_required':
      unsupported.push(option);
      return;
    default:
      throw new Error(
        `Invalid value for provider option ${option}: ${value}. ` +
          'Use when_supported or when_required.',
      );
  }
}

/**
 * Resolves deployment/provider configuration before constructing an S3-compatible store.
 * Endpoint authorization precedes credential resolution and namespace-scoped Secret reads.
 * Provider Secret credentials take precedence over custom and default credential providers.
 */
export async function createS3ArtifactStore(
  config: ArtifactStoreOptions,
  providerType: string,
  bucket: string,
  providerInfoString?: string,
  namespace?: string,
  customCredentialProvider?: () => Promise<Credentials> | Credentials,
  retrySignal?: AbortSignal,
  authorizeEndpoint?: ArtifactStoreEndpointPolicy,
) {
  const providerInfo = parseS3ProviderInfo(providerInfoString);
  return createS3ArtifactStoreWithProviderInfo(
    config,
    providerType,
    bucket,
    providerInfo,
    namespace,
    customCredentialProvider,
    retrySignal,
    authorizeEndpoint,
  );
}

export async function createArtifactStoreClient(
  configs: {
    minio: ArtifactStoreOptions;
    s3: ArtifactStoreOptions;
  },
  source: 'minio' | 's3',
  bucket: string,
  providerInfoString?: string,
  namespace?: string,
  retrySignal?: AbortSignal,
  authorizeEndpoint?: ArtifactStoreEndpointPolicy,
): Promise<ArtifactStore> {
  const providerInfo = parseS3ProviderInfo(providerInfoString);
  const providerType = providerInfo?.Provider === 's3' ? 's3' : source;
  return createS3ArtifactStoreWithProviderInfo(
    configs[providerType],
    providerType,
    bucket,
    providerInfo,
    namespace,
    undefined,
    retrySignal,
    authorizeEndpoint,
  );
}

function parseS3ProviderInfo(providerInfoString?: string): S3ProviderInfo | undefined {
  if (!providerInfoString) return undefined;
  try {
    const parsed = parseJSONString<unknown>(providerInfoString);
    if (
      !parsed ||
      typeof parsed !== 'object' ||
      Array.isArray(parsed) ||
      !('Params' in parsed) ||
      !parsed.Params ||
      typeof parsed.Params !== 'object' ||
      Array.isArray(parsed.Params)
    ) {
      throw new Error('Invalid provider info.');
    }
    const providerInfo = parsed as S3ProviderInfo;
    if (providerInfo.Params.fromEnv !== 'true' && providerInfo.Params.fromEnv !== 'false') {
      throw new Error('Provider info fromEnv must be true or false.');
    }
    for (const [name, value] of Object.entries(providerInfo.Params)) {
      if (value !== undefined && typeof value !== 'string') {
        throw new Error(`Provider info option ${name} must be a string.`);
      }
    }
    rejectUnsupportedS3ReadOptions(providerInfo);
    return providerInfo;
  } catch (error) {
    throw new ArtifactStoreConfigurationError(
      error instanceof Error ? error.message : String(error),
      { cause: error },
    );
  }
}

async function createS3ArtifactStoreWithProviderInfo(
  config: ArtifactStoreOptions,
  providerType: string,
  bucket: string,
  providerInfo: S3ProviderInfo | undefined,
  namespace?: string,
  customCredentialProvider?: () => Promise<Credentials> | Credentials,
  retrySignal?: AbortSignal,
  authorizeEndpoint?: ArtifactStoreEndpointPolicy,
): Promise<ArtifactStore> {
  // Handler configuration is shared by every request. Provider resolution below adds credentials
  // and endpoint overrides, so always work on a request-local copy.
  config = { ...config, retrySignal };
  let anonymous = false;
  let resolvedCredentialProvider: (() => Promise<Credentials>) | undefined;

  try {
    if (providerInfo?.Params.anonymous !== undefined) {
      anonymous = parseGoBoolean(providerInfo.Params.anonymous, 'anonymous');
    }
    if (providerInfo) config = applyS3ProviderOptions(config, providerInfo);
    config = applyEndpointRewrite(config);
    if (authorizeEndpoint) {
      const origin = getArtifactStoreOrigin(config);
      if (!origin)
        throw new Error('Artifact store endpoint must resolve to a valid HTTP(S) origin.');
      authorizeEndpoint({
        providerType,
        hasEndpointOverride: !!providerInfo?.Params.endpoint,
        endPoint: config.endPoint,
        port: config.port,
        useSSL: config.useSSL !== false,
        origin,
      });
    }
  } catch (error) {
    throw new ArtifactStoreConfigurationError(
      error instanceof Error ? error.message : String(error),
      { cause: error },
    );
  }

  if (customCredentialProvider && !anonymous) {
    try {
      const creds = await customCredentialProvider();

      if (creds && creds.accessKeyId && creds.secretAccessKey) {
        resolvedCredentialProvider = async () => customCredentialProvider();
        config = {
          ...config,
          accessKey: creds.accessKeyId,
          secretKey: creds.secretAccessKey,
          sessionToken: creds.sessionToken,
        };
      } else {
        console.warn(
          'Custom credential resolver returned incomplete credentials, falling back to default chain',
        );
      }
    } catch (error) {
      console.error('Custom credential resolver failed:', error);
      console.warn('Falling back to default credential resolution chain');
    }
  }

  if (providerInfo) {
    config = await applyS3ProviderCredentials(config, providerInfo, namespace);
    if (anonymous) {
      delete config.accessKey;
      delete config.secretKey;
      delete config.sessionToken;
    }
  }

  // If using s3 and sourcing credentials from environment (currently only aws is supported)
  if (providerType === 's3' && !anonymous && !(config.accessKey && config.secretKey)) {
    // Go Cloud resolves the AWS default chain independently of endpoint selection, so IRSA and
    // instance-profile credentials also apply to S3-compatible custom endpoints.
    try {
      const credentials = fromNodeProviderChain({ ignoreCache: true });
      const awsCredentials = await credentials();
      if (awsCredentials) {
        return createConfiguredS3Store(config, bucket, credentials);
      }
    } catch (error) {
      throw new Error('Unable to resolve AWS credentials for the S3 artifact store.', {
        cause: error,
      });
    }
  }

  return createConfiguredS3Store(
    config,
    bucket,
    providerInfo?.Params.fromEnv === 'false' ? undefined : resolvedCredentialProvider,
  );
}

const sharedS3Transport = new NodeHttpHandler();

function createConfiguredS3Store(
  config: ArtifactStoreOptions,
  bucket: string,
  credentialProvider?: () => Promise<Credentials>,
): ArtifactStore {
  const origin = getArtifactStoreOrigin(config);
  if (!origin)
    throw new ArtifactStoreConfigurationError(
      'Artifact store endpoint must be a valid HTTP(S) origin.',
    );
  const retryContext = createS3RetryContext(config.maxAttempts ?? 3, config.retrySignal);
  const transport = config.transportAgent
    ? new NodeHttpHandler(
        config.useSSL === false
          ? { httpAgent: config.transportAgent as HttpAgent }
          : { httpsAgent: config.transportAgent as HttpsAgent },
      )
    : sharedS3Transport;
  const anonymous = !credentialProvider && !(config.accessKey && config.secretKey);
  const client = new S3Client({
    endpoint:
      !config.endpointAuthority && config.endPoint === 's3.amazonaws.com' && !config.port
        ? undefined
        : origin,
    region: config.region || 'us-east-1',
    followRegionRedirects: true,
    forcePathStyle: config.pathStyle ?? !isAwsS3Endpoint(config.endPoint),
    maxAttempts: 1,
    requestChecksumCalculation: 'WHEN_REQUIRED',
    responseChecksumValidation: 'WHEN_REQUIRED',
    credentials: credentialProvider ?? {
      accessKeyId: config.accessKey || '',
      secretAccessKey: config.secretKey || '',
      sessionToken: config.sessionToken,
    },
    ...(anonymous
      ? { signer: { sign: async (request: Parameters<NodeHttpHandler['handle']>[0]) => request } }
      : {}),
    requestHandler: {
      handle: async (
        request: Parameters<NodeHttpHandler['handle']>[0],
        options: Parameters<NodeHttpHandler['handle']>[1],
      ) => {
        const attempt = retryContext.attemptStorage.getStore();
        try {
          const result = await transport.handle(request, options);
          if (attempt) {
            attempt.transportStatus = result.response.statusCode;
            if (result.response.body instanceof Readable) {
              result.response.body.once('error', () => {
                attempt.transportFailure = true;
              });
            }
          }
          return result;
        } catch (error) {
          if (attempt) attempt.transportFailure = true;
          throw error;
        }
      },
    },
  });
  if (config.useSSL === false) {
    // S3 endpoint rules choose HTTPS even when the generic SDK tls option is false.
    // Apply the already-authorized transport scheme before signing, retaining regional routing.
    client.middlewareStack.add(
      (next) => async (args) => {
        (args.request as { protocol: string }).protocol = 'http:';
        return next(args);
      },
      { step: 'build', name: 'artifactEndpointProtocol' },
    );
  }
  if (config.endpointBasePath) {
    client.middlewareStack.add(
      (next) => async (args) => {
        const request = args.request as { path: string; hostname: string };
        const bucketPath = `/${bucket}`;
        const virtualHost = request.hostname.startsWith(`${bucket}.`);
        const path =
          config.endpointTruncatesAtDelimiter &&
          !virtualHost &&
          (request.path === bucketPath || request.path.startsWith(`${bucketPath}/`))
            ? request.path.slice(bucketPath.length) || '/'
            : request.path;
        request.path = `${config.endpointBasePath}${path}`;
        return next(args);
      },
      { step: 'build', name: 'artifactEndpointBasePath' },
    );
  }
  const operationSignal = (signal?: AbortSignal) =>
    signal && config.retrySignal
      ? AbortSignal.any([signal, config.retrySignal])
      : (signal ?? config.retrySignal);
  const send = async <T>(operation: () => Promise<T>, signal?: AbortSignal): Promise<T> => {
    const context =
      signal === retryContext.signal
        ? retryContext
        : {
            ...retryContext,
            signal,
            get remainingRetryAttempts() {
              return retryContext.remainingRetryAttempts;
            },
            set remainingRetryAttempts(value) {
              retryContext.remainingRetryAttempts = value;
            },
          };
    return retryS3Operation(async () => {
      try {
        return await operation();
      } catch (error) {
        if (error instanceof Error) {
          const failure = error as Error & {
            code?: string;
            $metadata?: { httpStatusCode?: number };
          };
          if (!failure.code) failure.code = failure.name;
        }
        throw error;
      }
    }, context);
  };
  return {
    async openRead(key, options) {
      const signal = operationSignal(options?.signal);
      const response = await send(
        () =>
          client.send(new GetObjectCommand({ Bucket: bucket, Key: key }), { abortSignal: signal }),
        signal,
      );
      if (!(response.Body instanceof Readable))
        throw new Error('Artifact store returned no readable object body.');
      bindS3Abort(response.Body, signal, true);
      return response.Body;
    },
    async *list(prefix, options) {
      const signal = operationSignal(options?.signal);
      let token: string | undefined;
      do {
        const page = await send(
          () =>
            client.send(
              new ListObjectsV2Command({
                Bucket: bucket,
                Prefix: prefix,
                ContinuationToken: token,
                MaxKeys: options?.pageSize ?? 300,
              }),
              { abortSignal: signal },
            ),
          signal,
        );
        for (const object of page.Contents ?? []) {
          throwIfS3RequestAborted(signal);
          if (object.Key) yield { key: object.Key, size: object.Size ?? 0 };
        }
        if (!page.IsTruncated) return;
        if (!page.NextContinuationToken || page.NextContinuationToken === token)
          throw new Error(
            'Artifact store returned a truncated listing without a new continuation token.',
          );
        token = page.NextContinuationToken;
      } while (token);
    },
  };
}

const RETRYABLE_NETWORK_ERROR_CODES = new Set([
  'EAI_AGAIN',
  'EAI_FAIL',
  'ECONNREFUSED',
  'ECONNRESET',
  'ECONNABORTED',
  'EHOSTUNREACH',
  'EHOSTDOWN',
  'EINTR',
  'ENOBUFS',
  'ENETUNREACH',
  'ENETDOWN',
  'ENETRESET',
  'EADDRNOTAVAIL',
  'EADDRINUSE',
  'EPIPE',
  'ESHUTDOWN',
  'ETIMEDOUT',
]);

const RETRYABLE_S3_ERROR_CODES = new Set([
  'NetworkingError',
  'RequestTimeout',
  'RequestTimeoutException',
  'Throttling',
  'ThrottlingException',
  'ThrottledException',
  'RequestThrottledException',
  'TooManyRequestsException',
  'ProvisionedThroughputExceededException',
  'TransactionInProgressException',
  'RequestLimitExceeded',
  'BandwidthLimitExceeded',
  'LimitExceededException',
  'RequestThrottled',
  'SlowDown',
  'PriorRequestNotComplete',
  'EC2ThrottledException',
]);

const RETRYABLE_S3_HTTP_STATUSES = new Set([500, 502, 503, 504]);

function isRetryableS3Error(
  error: unknown,
  transportStatus?: number,
  transportFailure: boolean = false,
): boolean {
  if (!error || typeof error !== 'object') {
    return false;
  }
  const candidate = error as {
    code?: string;
    Code?: string;
    message?: string;
    status?: number;
    statusCode?: number;
    errno?: number | string;
    syscall?: string;
    [S3_TRANSPORT_ERROR]?: boolean;
  };
  const hasParsedCode =
    Object.prototype.hasOwnProperty.call(candidate, 'code') ||
    Object.prototype.hasOwnProperty.call(candidate, 'Code');
  const code = candidate.code ?? candidate.Code;
  const hasTransportProvenance = transportFailure || candidate[S3_TRANSPORT_ERROR] === true;
  if (code !== undefined && RETRYABLE_S3_ERROR_CODES.has(code)) {
    return true;
  }
  if (code !== undefined && RETRYABLE_NETWORK_ERROR_CODES.has(code) && hasTransportProvenance) {
    // Only the HTTP handler establishes transport provenance; service error fields are untrusted.
    return true;
  }
  if (hasTransportProvenance && candidate.errno !== undefined && candidate.syscall === 'connect') {
    // Node system errors preserve the operation that failed. Treat connection establishment as
    // Go's retryable dial class without also retrying permanent DNS failures such as ENOTFOUND.
    return true;
  }
  // The wire status takes precedence over service-controlled error fields.
  const status = transportStatus ?? candidate.statusCode ?? candidate.status;
  if (status !== undefined) {
    return RETRYABLE_S3_HTTP_STATUSES.has(status);
  }
  if (hasParsedCode) {
    // A parsed non-retryable S3 code is authoritative when status metadata is unavailable. Do not
    // let service-controlled error text override AccessDenied.
    return false;
  }
  // The transport records the original status before SDK deserialization. Never
  // infer retryability from service-controlled error text.
  return transportStatus !== undefined && RETRYABLE_S3_HTTP_STATUSES.has(transportStatus);
}

interface S3RetryContext {
  attemptStorage: AsyncLocalStorage<S3AttemptContext>;
  maxAttempts: number;
  remainingRetryAttempts: number;
  signal?: AbortSignal;
}

interface S3AttemptContext {
  transportFailure: boolean;
  transportStatus?: number;
}

function createS3RetryContext(maxAttempts: number, signal?: AbortSignal): S3RetryContext {
  return {
    attemptStorage: new AsyncLocalStorage<S3AttemptContext>(),
    maxAttempts,
    remainingRetryAttempts: MAX_S3_RETRY_ATTEMPTS_PER_REQUEST,
    signal,
  };
}

function consumeS3RetryAttempt(retryContext: S3RetryContext): void {
  if (retryContext.remainingRetryAttempts <= 0) {
    throw new Error('S3 retry attempt limit exhausted for this artifact request.');
  }
  retryContext.remainingRetryAttempts -= 1;
}

function createS3AbortError(): Error {
  return Object.assign(new Error('Artifact request was aborted.'), { name: 'AbortError' });
}

function markS3TransportError<T extends Error>(error: T): T {
  Object.defineProperty(error, S3_TRANSPORT_ERROR, { value: true });
  return error;
}

function bindS3Abort(
  target: {
    destroy: (error?: Error) => void;
    once: (event: string, listener: () => void) => unknown;
  },
  signal?: AbortSignal,
  absorbAbortError: boolean = false,
) {
  if (!signal) return;
  const abort = () => {
    if (absorbAbortError) {
      // IncomingMessage emits the supplied destroy error on the raw response. Consumers usually
      // observe only a derived transform, so install a one-shot listener before the deliberate
      // abort to prevent Node from treating the expected error as uncaught. Existing listeners
      // still receive the same AbortError.
      target.once('error', () => undefined);
    }
    target.destroy(createS3AbortError());
  };
  if (signal.aborted) {
    abort();
    return;
  }
  signal.addEventListener('abort', abort, { once: true });
  target.once('close', () => signal.removeEventListener('abort', abort));
}

function throwIfS3RequestAborted(signal?: AbortSignal): void {
  if (signal?.aborted) {
    throw createS3AbortError();
  }
}

async function waitForS3Retry(delayMs: number, signal?: AbortSignal): Promise<void> {
  throwIfS3RequestAborted(signal);
  await new Promise<void>((resolve, reject) => {
    const timeout = setTimeout(() => {
      signal?.removeEventListener('abort', abort);
      resolve();
    }, delayMs);
    const abort = () => {
      clearTimeout(timeout);
      reject(createS3AbortError());
    };
    signal?.addEventListener('abort', abort, { once: true });
  });
}

async function retryS3Operation<T>(
  operation: () => Promise<T>,
  retryContextOrMaxAttempts: S3RetryContext | number,
  delay: (attempt: number, signal?: AbortSignal) => Promise<void> = (attempt, signal) =>
    waitForS3Retry(getS3RetryDelayMs(attempt), signal),
): Promise<T> {
  const retryContext =
    typeof retryContextOrMaxAttempts === 'number'
      ? createS3RetryContext(retryContextOrMaxAttempts)
      : retryContextOrMaxAttempts;
  let operationAttempt = 1;
  while (true) {
    throwIfS3RequestAborted(retryContext.signal);
    // Distinct first attempts are legitimate work: directory downloads perform one object probe,
    // one listing, and one GET per child. Meter only amplification beyond each operation's first
    // attempt so large successful archives are not truncated by the retry safety guard.
    if (operationAttempt > 1) consumeS3RetryAttempt(retryContext);
    const attemptContext: S3AttemptContext = { transportFailure: false };
    try {
      return await retryContext.attemptStorage.run(attemptContext, operation);
    } catch (error) {
      throwIfS3RequestAborted(retryContext.signal);
      if (
        operationAttempt >= retryContext.maxAttempts ||
        retryContext.remainingRetryAttempts <= 0 ||
        !isRetryableS3Error(error, attemptContext.transportStatus, attemptContext.transportFailure)
      ) {
        throw error;
      }
      await delay(operationAttempt, retryContext.signal);
      operationAttempt += 1;
    }
  }
}

function getS3RetryDelayMs(attempt: number, random: () => number = Math.random): number {
  const exponentialDelay = 1_000 * 2 ** attempt;
  return exponentialDelay >= 20_000 ? 20_000 : random() * exponentialDelay;
}

function applyEndpointRewrite(config: ArtifactStoreOptions): ArtifactStoreOptions {
  const { endpointRewrite, ...clientConfig } = config;
  const rewriteConfig = endpointRewrite || process.env.MINIO_ENDPOINT_REWRITE || '';
  if (!rewriteConfig) {
    return clientConfig;
  }

  for (const rule of rewriteConfig.split(',')) {
    const [rawFrom, rawTo] = rule.split('=').map((part) => part.trim());
    if (!rawFrom || !rawTo) {
      continue;
    }

    const from = parseEndpoint(rawFrom);
    if (!from) {
      continue;
    }
    if (
      from.host !== clientConfig.endPoint ||
      (from.port !== undefined && from.port !== clientConfig.port)
    ) {
      continue;
    }

    const to = parseEndpoint(rawTo);
    if (!to) {
      continue;
    }
    clientConfig.endPoint = to.host;
    if (clientConfig.endpointAuthority) {
      clientConfig.endpointAuthority = to.host;
    }
    if (to.port !== undefined) {
      clientConfig.port = to.port;
    }
    if (to.useSSL !== undefined) {
      clientConfig.useSSL = to.useSSL;
    }
    break;
  }

  return clientConfig;
}

function parseEndpoint(
  endpoint: string,
): { host: string; port?: number; useSSL?: boolean } | undefined {
  try {
    const hasHttpScheme = /^https?:\/\//i.test(endpoint);
    const url = new URL(hasHttpScheme ? endpoint : `http://${endpoint}`);
    return {
      // Store the bare IPv6 address; origin construction restores its brackets.
      host: url.hostname.replace(/^\[(.*)\]$/, '$1'),
      port: url.port ? Number(url.port) : undefined,
      useSSL: hasHttpScheme ? url.protocol.toLowerCase() === 'https:' : undefined,
    };
  } catch (error) {
    const reason = error instanceof Error ? error.message : String(error);
    console.warn(
      `Ignoring invalid artifact store endpoint rewrite endpoint "${endpoint}": ${reason}`,
    );
    return undefined;
  }
}

function parseProviderEndpoint(
  endpoint: string,
  requireScheme: boolean,
): {
  host: string;
  port?: number;
  basePath?: string;
  truncatesAtDelimiter?: boolean;
  useSSL?: boolean;
} {
  if (
    [...endpoint].some((character) => {
      const codePoint = character.codePointAt(0) || 0;
      return codePoint <= 0x1f || codePoint === 0x7f;
    })
  ) {
    throw new Error('Provider info endpoint contains an invalid control character.');
  }
  const schemeMatch = /^(https?):\/\//i.exec(endpoint);
  if (requireScheme && !schemeMatch) {
    throw new Error(`Provider info endpoint must be an absolute HTTP(S) URL: ${endpoint}`);
  }
  const remainder = schemeMatch ? endpoint.slice(schemeMatch[0].length) : endpoint;
  const authorityEnd = remainder.search(/[/?#]/);
  const authority = authorityEnd === -1 ? remainder : remainder.slice(0, authorityEnd);
  const rawSuffix = authorityEnd === -1 ? '' : remainder.slice(authorityEnd);
  if (!authority) {
    throw new Error(`Provider info endpoint must contain a valid authority: ${endpoint}`);
  }
  if (authority.includes('\\') || /\s/.test(authority)) {
    throw new Error(`Provider info endpoint must contain a valid authority: ${endpoint}`);
  }
  if (rawSuffix.startsWith('?') || rawSuffix.startsWith('#') || /[?#]/.test(rawSuffix)) {
    throw new Error(
      `Provider endpoint "${endpoint}" contains a query or fragment that the frontend artifact ` +
        'reader cannot preserve. Remove it and retry.',
    );
  }
  if (/%(?![0-9a-f]{2})/i.test(rawSuffix)) {
    throw new Error(`Provider info endpoint contains an invalid URL escape: ${endpoint}`);
  }
  let authorityUrl: URL;
  try {
    authorityUrl = new URL(`${schemeMatch?.[1] || 'http'}://${authority}`);
  } catch {
    throw new Error(`Provider info has invalid endpoint: ${endpoint}`);
  }
  if (authorityUrl.username || authorityUrl.password) {
    throw new Error('Provider info endpoint must not contain credentials.');
  }
  // Go parses endpoint escapes into URL.Path before the AWS signer serializes the request. Decode
  // exactly one byte layer, then emit the same path spelling without WHATWG normalization. A proxy
  // must preserve this signed target; normalizing dot segments or separators also breaks Go
  // launcher requests using the same endpoint contract.
  // net/url exposes a once-decoded '?' or '#' as a delimiter when the Go Cloud endpoint is
  // composed with an object key. Match the launcher's wire target by dropping the delimiter and
  // the remainder instead of sending the escaped delimiter as object-path data.
  const encodedDelimiter = rawSuffix.search(/%(?:3f|23)/i);
  if (/%(?:00|7f)/i.test(rawSuffix) || containsInvalidSecondLayerEscape(rawSuffix)) {
    throw new Error(`Provider info endpoint contains an unsupported URL escape: ${endpoint}`);
  }
  const pathBeforeDelimiter =
    encodedDelimiter === -1 ? rawSuffix : rawSuffix.slice(0, encodedDelimiter);
  const basePath = normalizeGoEndpointPath(pathBeforeDelimiter).replace(/\/$/, '');
  return {
    basePath: basePath || undefined,
    host: authorityUrl.hostname.replace(/^\[(.*)\]$/, '$1'),
    port: authorityUrl.port ? Number(authorityUrl.port) : undefined,
    useSSL: schemeMatch ? schemeMatch[1].toLowerCase() === 'https' : undefined,
    truncatesAtDelimiter: encodedDelimiter !== -1,
  };
}

function containsInvalidSecondLayerEscape(rawPath: string): boolean {
  for (let index = 0; index < rawPath.length; index++) {
    if (rawPath.slice(index, index + 3).toLowerCase() !== '%25') continue;
    if (!/^[0-9a-f]{2}$/i.test(rawPath.slice(index + 3, index + 5))) return true;
    index += 2;
  }
  return false;
}

function normalizeGoEndpointPath(rawPath: string): string {
  const bytes: number[] = [];
  for (let index = 0; index < rawPath.length;) {
    if (rawPath[index] === '%' && /^[0-9a-f]{2}$/i.test(rawPath.slice(index + 1, index + 3))) {
      bytes.push(Number.parseInt(rawPath.slice(index + 1, index + 3), 16));
      index += 3;
      continue;
    }
    const codePoint = rawPath.codePointAt(index)!;
    bytes.push(...new TextEncoder().encode(String.fromCodePoint(codePoint)));
    index += codePoint > 0xffff ? 2 : 1;
  }

  const isHexByte = (value: number | undefined) =>
    value !== undefined && /[0-9a-f]/i.test(String.fromCharCode(value));
  const allowedAscii = new Set(
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~/:@!$&'()*+,;=[]",
  );
  let normalized = '';
  for (let index = 0; index < bytes.length; index++) {
    const byte = bytes[index];
    const character = String.fromCharCode(byte);
    // Go's parsed Path can contain a literal second-layer %HH escape. The AWS serializer preserves
    // that spelling instead of escaping the percent again.
    if (character === '%' && isHexByte(bytes[index + 1]) && isHexByte(bytes[index + 2])) {
      normalized += `%${String.fromCharCode(bytes[index + 1], bytes[index + 2])}`;
      index += 2;
    } else if (byte < 0x80 && allowedAscii.has(character)) {
      normalized += character;
    } else {
      normalized += `%${byte.toString(16).toUpperCase().padStart(2, '0')}`;
    }
  }
  return normalized;
}

/**
 * Parse provider info for any S3-compatible store that is not AWS S3.
 *
 * Security: This reads a Kubernetes Secret named by the provider info. The
 * artifact handler only forwards provider info when the requested namespace is
 * the frontend server's own namespace, so this function never reads Secrets
 * from a customer namespace. Shared direct mode rejects customer Secret policy
 * before reaching this function; those settings require the namespace-isolated
 * artifact proxy.
 * See: https://github.com/kubeflow/pipelines/pull/12860
 */
async function applyS3ProviderCredentials(
  config: ArtifactStoreOptions,
  providerInfo: S3ProviderInfo,
  namespace?: string,
): Promise<ArtifactStoreOptions> {
  if (providerInfo.Params.fromEnv === 'false') {
    if (!namespace) {
      throw new Error('Artifact Store provider given, but no namespace provided.');
    }
    if (
      !providerInfo.Params.accessKeyKey ||
      !providerInfo.Params.secretKeyKey ||
      !providerInfo.Params.secretName
    ) {
      throw new Error(
        'Provider info with fromEnv:false supplied with incomplete secret credential info.',
      );
    }

    try {
      config.accessKey = await getK8sSecret(
        providerInfo.Params.secretName,
        providerInfo.Params.accessKeyKey,
        namespace,
      );
      config.secretKey = await getK8sSecret(
        providerInfo.Params.secretName,
        providerInfo.Params.secretKeyKey,
        namespace,
      );
    } catch (e) {
      throw new Error(
        `Encountered error when trying to fetch provider secret ${providerInfo.Params.secretName}.`,
        { cause: e },
      );
    }
    if (!config.accessKey || !config.secretKey) {
      throw new Error('Provider Secret contains an empty access key or secret key.');
    }
  }

  return config;
}

function applyS3ProviderOptions(
  config: ArtifactStoreOptions,
  providerInfo: S3ProviderInfo,
): ArtifactStoreOptions {
  const disableSSLValue =
    providerInfo.Params.disableSSL === undefined
      ? undefined
      : parseGoBoolean(providerInfo.Params.disableSSL, 'disableSSL');
  const disableHttpsValue =
    providerInfo.Params.disable_https === undefined
      ? undefined
      : parseGoBoolean(providerInfo.Params.disable_https, 'disable_https');
  const nativeQuery = providerInfo.Params.nativeQuery === 'true';

  const structuredStandardAwsEndpoint =
    !nativeQuery &&
    providerInfo.Params.endpoint !== undefined &&
    ((providerInfo.Provider === 's3' && providerInfo.Params.endpoint === '') ||
      /^(https:\/\/)?s3\.amazonaws\.com(?::\d+)?(?:\/|$)/i.test(providerInfo.Params.endpoint));
  if (structuredStandardAwsEndpoint) {
    // The runtime intentionally discards this structured endpoint and lets the AWS resolver choose
    // the regional authority. Its path, port, and explicit scheme are therefore not authoritative.
    config.endPoint = 's3.amazonaws.com';
    config.endpointBasePath = undefined;
    config.endpointAuthority = undefined;
    config.port = undefined;
    config.useSSL = disableSSLValue === true ? false : true;
  } else if (providerInfo.Params.endpoint) {
    const endpoint = parseProviderEndpoint(providerInfo.Params.endpoint, nativeQuery);
    const disableTransport = disableHttpsValue ?? disableSSLValue;
    config.endPoint = endpoint.host;
    config.endpointBasePath = endpoint.basePath;
    config.endpointTruncatesAtDelimiter = endpoint.truncatesAtDelimiter;
    // Explicit AWS endpoints remain authoritative, even when a different signing region is set.
    config.endpointAuthority = isAwsS3Endpoint(endpoint.host) ? endpoint.host : undefined;
    config.port = endpoint.port;
    // AWS DisableHTTPS is asymmetric: true downgrades an explicit HTTPS endpoint, while false does
    // not upgrade an explicit HTTP endpoint. The legacy and Go Cloud spellings share this rule.
    config.useSSL =
      disableTransport === true
        ? false
        : (endpoint.useSSL ?? (disableTransport === false ? true : undefined));
  } else if (disableHttpsValue !== undefined) {
    config.useSSL = !disableHttpsValue;
  } else if (disableSSLValue !== undefined) {
    config.useSSL = !disableSSLValue;
  }

  if (providerInfo.Params.region) {
    config.region = providerInfo.Params.region;
  }
  {
    const configuredMaxRetries = providerInfo.Params.maxRetries ?? '0';
    const maxRetries = Number(configuredMaxRetries);
    if (!/^\d+$/.test(configuredMaxRetries) || !Number.isSafeInteger(maxRetries)) {
      throw new Error(
        `Invalid non-negative integer value for provider option maxRetries: ${configuredMaxRetries}`,
      );
    }
    // Go's zero value retains the AWS standard retryer's default of three total attempts. One outer
    // controller owns that budget so alternating HTTP, S3-code, and transport failures cannot
    // multiply retries across nested loops.
    const maxAttempts = Math.min(maxRetries > 0 ? maxRetries : 3, MAX_S3_ATTEMPTS);
    config.maxAttempts = maxAttempts;
  }
  const pathStyle =
    providerInfo.Params.forcePathStyle ??
    providerInfo.Params.s3ForcePathStyle ??
    providerInfo.Params.use_path_style;
  if (pathStyle !== undefined) {
    config.pathStyle = parseGoBoolean(pathStyle, 'use_path_style');
  } else if (nativeQuery) {
    config.pathStyle = false;
  }
  if (config.useSSL === false && isAWSS3Endpoint(config.endPoint)) {
    throw new Error('AWS S3 provider endpoints must use HTTPS.');
  }
  return config;
}

function isAwsS3Endpoint(host: string): boolean {
  const normalized = host.toLowerCase();
  return (
    normalized === 's3.amazonaws.com' ||
    /^s3[.-][a-z0-9-]+\.amazonaws\.com(?:\.cn)?$/.test(normalized)
  );
}

/**
 * Checks the magic number of a buffer to see if the mime type is a uncompressed
 * tarball. The buffer must be of length 264 bytes or more.
 *
 * See also: https://www.gnu.org/software/tar/manual/html_node/Standard.html
 *
 * @param buf Buffer
 */
export function isTarball(buf: Buffer) {
  if (!buf || buf.length < 264) {
    return false;
  }
  const offset = 257;
  const v1 = [0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30];
  const v0 = [0x75, 0x73, 0x74, 0x61, 0x72, 0x20, 0x20, 0x00];

  return (
    v1.reduce((res, curr, i) => res && curr === buf[offset + i], true) ||
    v0.reduce((res, curr, i) => res && curr === buf[offset + i], true as boolean)
  );
}

/**
 * Returns a stream that extracts the first record of a tarball if the source
 * stream is a tarball, otherwise just pipe the content as is.
 */
export function maybeTarball(onExtractionDetermined?: (extracted: boolean) => void): Transform {
  return peek(
    { newline: false, maxBuffer: 264 },
    (data: Buffer, swap: (error?: Error, parser?: Transform) => void) => {
      const extracted = isTarball(data);
      onExtractionDetermined?.(extracted);
      if (extracted) swap(undefined, extractFirstTarRecordAsStream());
      else swap(undefined, new PassThrough());
    },
  );
}

function detectCompression(onCompressionDetermined: (compressed: boolean) => void): Transform {
  return peek(
    { newline: false, maxBuffer: 3 },
    (data: Buffer, swap: (error?: Error, parser?: Transform) => void) => {
      // Keep these signatures aligned with gunzip-maybe's is-gzip and
      // is-deflate dependencies. The callback controls the response filename,
      // so its decision must match whether gunzip-maybe transforms the bytes.
      const gzip = data.length >= 3 && data[0] === 0x1f && data[1] === 0x8b && data[2] === 0x08;
      const deflate = data.length >= 2 && data[0] === 0x78 && [0x01, 0x9c, 0xda].includes(data[1]);
      onCompressionDetermined(gzip || deflate);
      swap(undefined, new PassThrough());
    },
  );
}

/**
 * Returns a transform stream where the first record inside a tarball will be
 * pushed - i.e. all other contents will be dropped.
 */
function extractFirstTarRecordAsStream() {
  const extract = tar.extract();
  const transformStream = new Transform({
    write: (chunk: any, _encoding: string, callback: (error?: Error | null) => void) => {
      extract.write(chunk, callback);
    },
  });
  extract.once('entry', function (_header, stream, next) {
    stream.on('data', (buffer: any) => transformStream.push(buffer));
    stream.on('end', () => {
      transformStream.emit('end');
      next();
    });
    stream.resume(); // just auto drain the stream
  });
  extract.on('error', (error) => transformStream.emit('error', error));
  return transformStream;
}

/**
 * Returns a stream from an object in a s3 compatible object store (e.g. minio).
 * The actual content of the stream depends on the object.
 *
 * Any gzipped or deflated objects will be ungzipped or inflated. If the object
 * is a tarball, only the content of the first record in the tarball will be
 * returned. For any other objects, the raw content will be returned.
 *
 * @param param.key Key of the object to retrieve.
 * @param param.store Bucket-bound artifact store.
 * @param param.tryExtract Whether we try to extract *.tar.gz, default to true.
 * @param param.onError Optional asynchronous error callback. The returned
 * stream also emits the same error; callers that omit this callback must
 * attach their own listener if they need request-specific recovery.
 *
 */
export async function getObjectStream({
  key,
  store,
  signal,
  tryExtract = true,
  onError,
  onTransformationDetermined,
}: ArtifactReadRequest): Promise<Transform> {
  const source = await store.openRead(key, { signal });
  let compressed = false;
  const output = tryExtract
    ? maybeTarball((extracted) => onTransformationDetermined?.(compressed || extracted))
    : new PassThrough();
  const streams = tryExtract
    ? [source, detectCompression((value) => (compressed = value)), gunzip(), output]
    : [source, output];
  if (!tryExtract) {
    onTransformationDetermined?.(false);
  }
  output.once(
    'error',
    onError ?? ((error) => console.error('Artifact object stream failed', error)),
  );
  // Readable.pipe() does not forward a source error to its destination. Use
  // pipeline so storage and decompression failures destroy the returned stream
  // and reach the artifact handler's abort-on-error listener.
  pipeline(streams, (error) => {
    if (error && !output.destroyed) {
      output.destroy(error);
    }
  });
  // Closing the source also lets pipeline tear down every downstream transform.
  const abort = () => source.destroy();
  if (signal?.aborted) {
    abort();
  } else {
    signal?.addEventListener('abort', abort, { once: true });
  }
  output.once('close', () => signal?.removeEventListener('abort', abort));
  return output;
}

/**
 * Returns a minio/s3 error as a NoSuchKey error if applicable. Different
 * providers surface the "object not found" condition slightly differently
 * (code, Code, or message). This normalizes the check.
 */
export function isNoSuchKeyError(err: unknown): boolean {
  if (!err || typeof err !== 'object') {
    return false;
  }
  const e = err as { code?: string; Code?: string; name?: string; message?: string };
  const code = e.code || e.Code || e.name;
  if (code === 'NoSuchKey' || code === 'NotFound') {
    return true;
  }
  return typeof e.message === 'string' && e.message.includes('NoSuchKey');
}

/** Streams a recursive directory listing without buffering its keys. */
export async function* listObjectsUnderPrefix(
  store: ArtifactStore,
  prefix: string,
  signal?: AbortSignal,
): AsyncGenerator<{ name: string; size: number }> {
  for await (const item of store.list(prefix, { signal })) {
    yield { name: item.key, size: item.size };
  }
}

/** Reads at most one extra entry to distinguish an exact count from a lower bound. */
export async function summarizeDirectoryUnderPrefix(
  store: ArtifactStore,
  prefix: string,
  maxKeys: number = 50,
): Promise<{ count: number; truncated: boolean } | null> {
  let count = 0;
  for await (const _item of store.list(prefix, { pageSize: maxKeys + 1 })) {
    if (count === maxKeys) return { count, truncated: true };
    count++;
  }
  return count ? { count, truncated: false } : null;
}

export const TEST_ONLY = {
  createS3RetryContext,
  markS3TransportError,
  getS3RetryDelayMs,
  isRetryableS3Error,
  retryS3Operation,
  parseProviderEndpoint,
};
