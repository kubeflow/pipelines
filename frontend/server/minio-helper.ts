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
import { Transform, PassThrough } from 'stream';
import * as tar from 'tar-stream';
import peek from 'peek-stream';
import gunzip from 'gunzip-maybe';
import { URL } from 'url';
import { Client as MinioClient, ClientOptions as MinioClientOptions } from 'minio';
import type { S3ProviderInfo } from './handlers/artifacts.js';
import { getK8sSecret } from './k8s-helper.js';
import { parseJSONString } from './utils.js';
import { fromNodeProviderChain } from '@aws-sdk/credential-providers';
import { parseGoBoolean } from './helpers/provider-options.js';
/** MinioRequestConfig describes the info required to retrieve an artifact. */
export interface MinioRequestConfig {
  bucket: string;
  key: string;
  client: MinioClient;
  signal?: AbortSignal;
  tryExtract?: boolean;
}

/** MinioClientOptionsWithOptionalSecrets wraps around MinioClientOptions where only endPoint is required (accesskey and secretkey are optional). */
export interface MinioClientOptionsWithOptionalSecrets extends Partial<MinioClientOptions> {
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
}

const MAX_S3_ATTEMPTS = 10;
const MAX_S3_RETRY_ATTEMPTS_PER_REQUEST = 10;

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
 * Create minio client for s3 compatible storage
 *
 * If providerInfoString is available, use these over defaultConfigs.
 *
 * If providerInfo is not provided or, if credentials are sourced fromEnv,
 * then, if using aws s3 (via provider chain or instance profile), create a
 * minio client backed by aws s3 client.
 *
 * Otherwise, assume s3 compatible credentials have been provided via configs
 * (defaultConfigs or ProviderInfo), and return a minio client configured
 * respectively.
 *
 * Security: By default, credentials are injected via environment variables
 * (MINIO_ACCESS_KEY, MINIO_SECRET_KEY) from the deployment spec. When
 * providerInfo indicates that credentials should not come from the environment
 * (fromEnv === 'false'), this helper may read namespace-scoped Kubernetes
 * secrets via getK8sSecret. See: https://github.com/kubeflow/pipelines/issues/12373
 *
 * @param config minio client options where `accessKey` and `secretKey` are optional.
 * @param providerType provider type ('s3' or 'minio')
 * @param providerInfoString
 * @param namespace
 * @param customCredentialProvider An optional function which can be added to resolve credentials from a non-standard source. Useful
 * for enterprises who may have bespoke credential retrieval processes or for refreshing short-lived tokens.
 */
export async function createMinioClient(
  config: MinioClientOptionsWithOptionalSecrets,
  providerType: string,
  providerInfoString?: string,
  namespace?: string,
  customCredentialProvider?: () => Promise<Credentials> | Credentials,
  retrySignal?: AbortSignal,
) {
  // Handler configuration is shared by every request. Provider resolution below adds credentials
  // and endpoint overrides, so always work on a request-local copy.
  config = { ...config, retrySignal };

  let providerInfo: S3ProviderInfo | undefined;
  let anonymous = false;
  if (providerInfoString) {
    providerInfo = parseJSONString<S3ProviderInfo>(providerInfoString);
    if (!providerInfo) {
      throw new Error('Failed to parse provider info.');
    }
    rejectUnsupportedS3ReadOptions(providerInfo);
    if (providerInfo.Params.anonymous !== undefined) {
      anonymous = parseGoBoolean(providerInfo.Params.anonymous, 'anonymous');
    }
  }

  if (customCredentialProvider && !anonymous) {
    try {
      const creds = await customCredentialProvider();

      if (creds && creds.accessKeyId && creds.secretAccessKey) {
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
    config = await applyS3ProviderInfo(config, providerInfo, namespace);
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
        const { accessKeyId: accessKey, secretAccessKey: secretKey, sessionToken } = awsCredentials;
        return createConfiguredMinioClient(
          applyEndpointRewrite({
            ...config,
            accessKey,
            secretKey,
            sessionToken,
          }) as MinioClientOptions,
        );
      }
    } catch (error) {
      throw new Error('Unable to resolve AWS credentials for the S3 artifact store.', {
        cause: error,
      });
    }
  }

  // If using any AWS or S3 compatible store (e.g. minio, aws s3 when using manual creds, ceph, etc.)
  let mc: MinioClient;
  try {
    mc = createConfiguredMinioClient(applyEndpointRewrite(config));
  } catch (err) {
    throw new Error(`Failed to create MinioClient: ${err}`, { cause: err });
  }
  return mc;
}

function createConfiguredMinioClient(config: MinioClientOptionsWithOptionalSecrets): MinioClient {
  const {
    endpointAuthority,
    endpointBasePath,
    endpointTruncatesAtDelimiter,
    maxAttempts,
    retrySignal,
    ...clientOptions
  } = config;
  const client = new MinioClient(clientOptions as MinioClientOptions);
  const retryContext = createS3RetryContext(maxAttempts ?? 3, retrySignal);
  const retryClient = client as unknown as {
    retryOptions: { maximumRetryCount?: number };
  };
  retryClient.retryOptions = {
    ...retryClient.retryOptions,
    // One outer controller owns both the per-operation compatibility budget and the independent
    // ten-attempt request ceiling. Leaving MinIO's narrower loop enabled would hide transport
    // attempts from that aggregate ceiling and multiply mixed failures.
    maximumRetryCount: 0,
  };
  exposeParsedS3Errors(client, retryContext);
  if (endpointBasePath || endpointAuthority) {
    // MinIO JS does not expose endpoint base paths as a client option. Prefix the request path
    // before MinIO signs it so custom Go Cloud endpoints retain the same origin-relative root.
    const requestOptionsClient = client as unknown as {
      getRequestOptions: (options: { bucketName?: string }) => {
        headers: Record<string, string>;
        host: string;
        path: string;
        [key: string]: unknown;
      };
    };
    const getRequestOptions = requestOptionsClient.getRequestOptions.bind(client);
    requestOptionsClient.getRequestOptions = (options) => {
      const requestOptions = getRequestOptions(options);
      const storagePath = requestOptions.path;
      if (endpointAuthority) {
        // Preserve MinIO's addressing decision (not merely the configured preference). In
        // particular, MinIO deliberately uses path style for dotted buckets over HTTPS.
        const bucketPath = options.bucketName ? `/${options.bucketName}` : undefined;
        const usesPathStyle =
          bucketPath !== undefined &&
          (storagePath === bucketPath || storagePath.startsWith(`${bucketPath}/`));
        const usesVirtualHostStyle =
          !!options.bucketName &&
          requestOptions.host.startsWith(`${options.bucketName}.`) &&
          !usesPathStyle;
        const host = usesVirtualHostStyle
          ? `${options.bucketName}.${endpointAuthority}`
          : endpointAuthority;
        requestOptions.host = host;
        const defaultPort = clientOptions.useSSL === false ? 80 : 443;
        requestOptions.headers.host =
          clientOptions.port && clientOptions.port !== defaultPort
            ? `${host}:${clientOptions.port}`
            : host;
      }
      if (endpointBasePath) {
        const bucketPath = options.bucketName ? `/${options.bucketName}` : undefined;
        const pathAfterDelimiter =
          endpointTruncatesAtDelimiter &&
          bucketPath &&
          (storagePath === bucketPath || storagePath.startsWith(`${bucketPath}/`))
            ? storagePath.slice(bucketPath.length) || '/'
            : storagePath;
        requestOptions.path = `${endpointBasePath}${pathAfterDelimiter}`;
      }
      return requestOptions;
    };
  }
  wrapRetryableS3Methods(client, retryContext);
  return client;
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
  };
  const hasParsedCode =
    Object.prototype.hasOwnProperty.call(candidate, 'code') ||
    Object.prototype.hasOwnProperty.call(candidate, 'Code');
  const code = candidate.code ?? candidate.Code;
  if (code !== undefined && RETRYABLE_S3_ERROR_CODES.has(code)) {
    return true;
  }
  if (
    code !== undefined &&
    RETRYABLE_NETWORK_ERROR_CODES.has(code) &&
    (transportFailure || (candidate.errno !== undefined && candidate.syscall !== undefined))
  ) {
    // Node transport failures include errno/syscall evidence. A same-named value parsed from an
    // object store's XML <Code> is untrusted API data and must not impersonate a network error.
    return true;
  }
  if (candidate.errno !== undefined && candidate.syscall === 'connect') {
    // Node system errors preserve the operation that failed. Treat connection establishment as
    // Go's retryable dial class without also retrying permanent DNS failures such as ENOTFOUND.
    return true;
  }
  const status = candidate.statusCode ?? candidate.status;
  if (status !== undefined) {
    return RETRYABLE_S3_HTTP_STATUSES.has(status);
  }
  if (hasParsedCode) {
    // A parsed non-retryable S3 code is authoritative when status metadata is unavailable. Do not
    // let service-controlled error text impersonate the exact MinIO wrapper after AccessDenied.
    return false;
  }
  // The transport records the original status before MinIO rewrites or wraps the response. Never
  // infer retryability from service-controlled error text.
  return transportStatus !== undefined && RETRYABLE_S3_HTTP_STATUSES.has(transportStatus);
}

function exposeParsedS3Errors(client: MinioClient, retryContext: S3RetryContext): void {
  const statusOnlyResponses = new Set([408, 429, 499, 520]);
  type Destroyable = {
    destroy: (error?: Error) => void;
    once: (event: string, listener: (...args: unknown[]) => void) => unknown;
  };
  type RetryResponse = Destroyable & { statusCode?: number };
  type RetryTransport = {
    request: (options: unknown, callback: (response: RetryResponse) => void) => Destroyable;
  };
  const transportClient = client as unknown as { transport?: RetryTransport };
  const originalTransport = transportClient.transport;
  if (!originalTransport?.request) {
    // Unit-test doubles do not expose MinIO's protected transport. Production clients always do.
    return;
  }
  const originalRequest = originalTransport.request.bind(originalTransport);
  transportClient.transport = {
    ...originalTransport,
    request: (options: unknown, callback: (response: RetryResponse) => void) => {
      const request = originalRequest(options, (response) => {
        retryContext.transportStatus = response.statusCode;
        bindS3Abort(response, retryContext.signal, true);
        if (response.statusCode !== undefined && statusOnlyResponses.has(response.statusCode)) {
          // MinIO retries these statuses before reading their S3 XML body. Present them to its
          // parser as a generic client error so the outer AWS-compatible controller can decide
          // from SlowDown/RequestTimeout/Throttling rather than status alone.
          response.statusCode = 400;
        }
        callback(response);
      });
      request.once('error', () => {
        retryContext.transportFailure = true;
      });
      bindS3Abort(request, retryContext.signal);
      return request;
    },
  };
}

interface S3RetryContext {
  maxAttempts: number;
  remainingRetryAttempts: number;
  signal?: AbortSignal;
  transportFailure?: boolean;
  transportStatus?: number;
}

function createS3RetryContext(maxAttempts: number, signal?: AbortSignal): S3RetryContext {
  return { maxAttempts, remainingRetryAttempts: MAX_S3_RETRY_ATTEMPTS_PER_REQUEST, signal };
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
    retryContext.transportFailure = false;
    retryContext.transportStatus = undefined;
    try {
      return await operation();
    } catch (error) {
      throwIfS3RequestAborted(retryContext.signal);
      if (
        operationAttempt >= retryContext.maxAttempts ||
        retryContext.remainingRetryAttempts <= 0 ||
        !isRetryableS3Error(error, retryContext.transportStatus, retryContext.transportFailure)
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

function wrapRetryableS3Methods(client: MinioClient, retryContext: S3RetryContext): void {
  const retryableMethods = ['getObject', 'listObjectsV2Query'] as const;
  const mutableClient = client as unknown as Record<string, unknown>;
  retryableMethods.forEach((method) => {
    const candidate = mutableClient[method];
    if (typeof candidate !== 'function') {
      throw new Error(
        `Minio client does not expose ${method}; the bundled minio version may be incompatible ` +
          'with configured S3 retries.',
      );
    }
    const operation = candidate.bind(client) as (...args: unknown[]) => Promise<unknown>;
    // getObject retries only failures reported before MinIO returns the response stream. Once body
    // bytes have been consumed, replaying the request here would duplicate or corrupt the caller's
    // stream; downstream failures therefore remain visible to the caller.
    mutableClient[method] = (...args: unknown[]) =>
      retryS3Operation(() => operation(...args), retryContext);
  });
}

function applyEndpointRewrite(
  config: MinioClientOptionsWithOptionalSecrets,
): MinioClientOptionsWithOptionalSecrets {
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
      // WHATWG URL retains brackets around IPv6 hostnames; MinIO expects the bare address.
      host: url.hostname.replace(/^\[(.*)\]$/, '$1'),
      port: url.port ? Number(url.port) : undefined,
      useSSL: hasHttpScheme ? url.protocol.toLowerCase() === 'https:' : undefined,
    };
  } catch (error) {
    const reason = error instanceof Error ? error.message : String(error);
    console.warn(`Ignoring invalid MinIO endpoint rewrite endpoint "${endpoint}": ${reason}`);
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
  for (let index = 0; index < rawPath.length; ) {
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
async function applyS3ProviderInfo(
  config: MinioClientOptionsWithOptionalSecrets,
  providerInfo: S3ProviderInfo,
  namespace?: string,
): Promise<MinioClientOptionsWithOptionalSecrets> {
  const disableSSLValue =
    providerInfo.Params.disableSSL === undefined
      ? undefined
      : parseGoBoolean(providerInfo.Params.disableSSL, 'disableSSL');
  const disableHttpsValue =
    providerInfo.Params.disable_https === undefined
      ? undefined
      : parseGoBoolean(providerInfo.Params.disable_https, 'disable_https');
  const nativeQuery = providerInfo.Params.nativeQuery === 'true';
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

  const structuredStandardAwsEndpoint =
    !nativeQuery &&
    providerInfo.Params.endpoint !== undefined &&
    /^(https:\/\/)?s3\.amazonaws\.com(?::\d+)?(?:\/|$)/i.test(providerInfo.Params.endpoint);
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
    // MinIO rewrites explicit AWS endpoints according to its own region table. Go Cloud keeps an
    // explicit endpoint authoritative, so pin AWS-partition hosts after MinIO builds the request.
    config.endpointAuthority = isAwsS3Endpoint(endpoint.host) ? endpoint.host : undefined;
    config.port = endpoint.port;
    // AWS DisableHTTPS is asymmetric: true downgrades an explicit HTTPS endpoint, while false does
    // not upgrade an explicit HTTP endpoint. The legacy and Go Cloud spellings share this rule.
    config.useSSL =
      disableTransport === true
        ? false
        : (endpoint.useSSL ?? (disableTransport === false ? true : undefined));
  } else if (
    providerInfo.Provider === 's3' &&
    providerInfo.Params.endpoint !== undefined &&
    !nativeQuery
  ) {
    // An explicit empty endpoint in structured launcher configuration means standard AWS S3,
    // not the UI server's separately configured MinIO-compatible endpoint.
    config.endPoint = 's3.amazonaws.com';
    config.endpointBasePath = undefined;
    config.endpointAuthority = undefined;
    config.port = undefined;
    config.useSSL = disableSSLValue === true ? false : true;
  } else if (disableHttpsValue !== undefined) {
    config.useSSL = !disableHttpsValue;
  } else if (disableSSLValue) {
    config.useSSL = false;
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
  return config;
}

function isAwsS3Endpoint(host: string): boolean {
  const normalized = host.toLowerCase();
  return (
    normalized === 's3.amazonaws.com' ||
    /^s3[.-][a-z0-9-]+\.amazonaws\.com(?:\.cn)?$/.test(normalized)
  );
}

export const TEST_ONLY = {
  createS3RetryContext,
  getS3RetryDelayMs,
  isRetryableS3Error,
  parseProviderEndpoint,
  retryS3Operation,
};

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
export function maybeTarball(): Transform {
  return peek(
    { newline: false, maxBuffer: 264 },
    (data: Buffer, swap: (error?: Error, parser?: Transform) => void) => {
      if (isTarball(data)) swap(undefined, extractFirstTarRecordAsStream());
      else swap(undefined, new PassThrough());
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
 * @param param.bucket Bucket name to retrieve the object from.
 * @param param.key Key of the object to retrieve.
 * @param param.client Minio client.
 * @param param.tryExtract Whether we try to extract *.tar.gz, default to true.
 *
 */
export async function getObjectStream({
  bucket,
  key,
  client,
  signal,
  tryExtract = true,
}: MinioRequestConfig): Promise<Transform> {
  const stream = await client.getObject(bucket, key);
  const output = tryExtract
    ? stream.pipe(gunzip()).pipe(maybeTarball())
    : stream.pipe(new PassThrough());
  // Destroy the upstream body quietly. The request-level transport already rejects the active
  // MinIO operation with AbortError; emitting another source-stream error would be unhandled when
  // downstream transforms have already detached after a browser disconnect.
  const abort = () => stream.destroy();
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
  const e = err as { code?: string; Code?: string; message?: string };
  const code = e.code || e.Code;
  if (code === 'NoSuchKey' || code === 'NotFound') {
    return true;
  }
  return typeof e.message === 'string' && e.message.includes('NoSuchKey');
}

type ListObjectsV2QueryResult = {
  objects: Array<{ name?: string; size?: number }>;
  isTruncated: boolean;
  nextContinuationToken: string;
};

type ListObjectsV2Query = (
  bucket: string,
  prefix: string,
  continuationToken: string,
  delimiter: string,
  maxKeys: number,
  startAfter: string,
) => Promise<ListObjectsV2QueryResult>;

// `listObjectsV2Query` is an internal helper on the minio client and is not
// declared in its public type definitions. We narrow to it via a runtime
// check so a future minio upgrade that removes the method fails fast with a
// clear message instead of throwing `undefined is not a function` deep inside
// the listing loop.
function getListObjectsV2Query(client: MinioClient): ListObjectsV2Query {
  const candidate = (client as unknown as { listObjectsV2Query?: unknown }).listObjectsV2Query;
  if (typeof candidate !== 'function') {
    throw new Error(
      'Minio client does not expose listObjectsV2Query; the bundled minio version may be incompatible with listObjectsUnderPrefix',
    );
  }
  return (candidate as ListObjectsV2Query).bind(client);
}

/**
 * Yields all objects under a given prefix in an s3-compatible bucket,
 * recursively, along with their sizes. Implemented as an async generator so
 * callers can begin streaming the first object before the full listing
 * completes — important for large directory artifacts where buffering all
 * keys would delay the first byte and inflate memory use.
 *
 * Pages via the lower-level `listObjectsV2Query` instead of the public
 * `listObjectsV2` streaming API. The public API hard-codes maxKeys=1000 per
 * page, and minio's bundled fast-xml-parser caps entity expansions at 1000;
 * each Contents entry has ~2 `&quot;` entities in its ETag, so a full page
 * trips "Entity expansion limit exceeded" once a directory holds more than
 * ~500 objects. A smaller page size keeps each XML parse under the cap.
 */
export async function* listObjectsUnderPrefix(
  client: MinioClient,
  bucket: string,
  prefix: string,
): AsyncGenerator<{ name: string; size: number }> {
  const PAGE_SIZE = 300;
  const listObjectsV2Query = getListObjectsV2Query(client);
  let continuationToken = '';
  let isTruncated = true;

  while (isTruncated) {
    const page = await listObjectsV2Query(bucket, prefix, continuationToken, '', PAGE_SIZE, '');

    for (const item of page.objects) {
      if (item.name) {
        yield { name: item.name, size: item.size ?? 0 };
      }
    }

    isTruncated = page.isTruncated;
    continuationToken = page.nextContinuationToken;
  }
}

/**
 * Returns a bounded summary of a prefix using a single capped
 * `listObjectsV2Query` call — does not paginate. Designed for preview-style
 * requests where the caller just needs to know "is there anything here, and
 * roughly how many files?" without paying for a full listing of a
 * potentially huge directory.
 *
 * Resolves to `null` for an empty prefix so callers can answer with a 404.
 * `truncated: true` means the directory has more than `maxKeys` files; the
 * caller should treat `count` as a lower bound.
 */
export async function summarizeDirectoryUnderPrefix(
  client: MinioClient,
  bucket: string,
  prefix: string,
  maxKeys: number = 50,
): Promise<{ count: number; truncated: boolean } | null> {
  const listObjectsV2Query = getListObjectsV2Query(client);
  const page = await listObjectsV2Query(bucket, prefix, '', '', maxKeys, '');
  if (page.objects.length === 0) {
    return null;
  }
  return { count: page.objects.length, truncated: page.isTruncated };
}
