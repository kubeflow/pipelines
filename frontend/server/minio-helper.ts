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
import { Transform, PassThrough, pipeline } from 'stream';
import * as tar from 'tar-stream';
import peek from 'peek-stream';
import gunzip from 'gunzip-maybe';
import { URL } from 'url';
import { Client as MinioClient, ClientOptions as MinioClientOptions } from 'minio';
import { isAWSS3Endpoint } from './aws-helper.js';
import type { S3ProviderInfo } from './handlers/artifacts.js';
import { getK8sSecret } from './k8s-helper.js';
import { parseJSONString } from './utils.js';
import { fromNodeProviderChain } from '@aws-sdk/credential-providers';
/** MinioRequestConfig describes the info required to retrieve an artifact. */
export interface MinioRequestConfig {
  bucket: string;
  key: string;
  client: MinioClient;
  tryExtract?: boolean;
  /** Receives asynchronous storage or transformation failures from the returned stream. */
  onError?: (error: Error) => void;
  onTransformationDetermined?: (transformed: boolean) => void;
}

/** MinioClientOptionsWithOptionalSecrets wraps around MinioClientOptions where only endPoint is required (accesskey and secretkey are optional). */
export interface MinioClientOptionsWithOptionalSecrets extends Partial<MinioClientOptions> {
  endPoint: string;
  endpointRewrite?: string;
}

export interface Credentials {
  accessKeyId: string;
  secretAccessKey: string;
  sessionToken?: string;
}

export interface ArtifactStoreEndpoint {
  endPoint: string;
  port?: number;
  useSSL: boolean;
  origin: string;
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
}: Pick<MinioClientOptions, 'endPoint' | 'port' | 'useSSL'>): string | undefined {
  const insecure = useSSL === false;
  const endpoint = parseArtifactStoreEndpoint(endPoint, insecure);
  if (!endpoint) {
    return undefined;
  }
  const origin = new URL(endpoint.origin);
  if (port !== undefined && !origin.port) {
    origin.port = String(port);
  }
  return origin.origin;
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
) {
  // Provider parsing applies request-specific endpoint and credential fields.
  // Never let those mutations escape into the shared server configuration.
  config = { ...config };

  if (customCredentialProvider) {
    try {
      const creds = await customCredentialProvider();

      if (creds && creds.accessKeyId && creds.secretAccessKey) {
        return new MinioClient(
          applyEndpointRewrite({
            ...config,
            accessKey: creds.accessKeyId,
            secretKey: creds.secretAccessKey,
            sessionToken: creds.sessionToken,
          }) as MinioClientOptions,
        );
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

  if (providerInfoString) {
    const providerInfo = parseJSONString<unknown>(providerInfoString);
    if (
      !providerInfo ||
      typeof providerInfo !== 'object' ||
      Array.isArray(providerInfo) ||
      !('Params' in providerInfo) ||
      !providerInfo.Params ||
      typeof providerInfo.Params !== 'object' ||
      Array.isArray(providerInfo.Params)
    ) {
      throw new Error('Invalid provider info.');
    }
    const typedProviderInfo = providerInfo as S3ProviderInfo;
    if (
      typedProviderInfo.Params.fromEnv !== 'true' &&
      typedProviderInfo.Params.fromEnv !== 'false'
    ) {
      throw new Error('Provider info fromEnv must be true or false.');
    }
    // If fromEnv == false, we rely on the default credentials or env to provide credentials (e.g. IRSA)
    if (typedProviderInfo.Params.fromEnv === 'false') {
      if (!namespace) {
        throw new Error('Artifact Store provider given, but no namespace provided.');
      } else {
        config = await parseS3ProviderInfo(config, typedProviderInfo, namespace);
      }
    }
  }

  // If using s3 and sourcing credentials from environment (currently only aws is supported)
  if (providerType === 's3' && !(config.accessKey && config.secretKey)) {
    // AWS S3 with credentials from provider chain
    if (isAWSS3Endpoint(config.endPoint)) {
      try {
        const credentials = fromNodeProviderChain({ ignoreCache: true });
        const awsCredentials = await credentials();
        if (awsCredentials) {
          const {
            accessKeyId: accessKey,
            secretAccessKey: secretKey,
            sessionToken,
          } = awsCredentials;
          return new MinioClient(
            applyEndpointRewrite({
              ...config,
              accessKey,
              secretKey,
              sessionToken,
            }) as MinioClientOptions,
          );
        }
      } catch (e) {
        console.error('Unable to get aws instance profile credentials: ', e);
      }
    } else {
      console.error(
        'Encountered S3-compatible provider type with no provided credentials, and unsupported environment based credential support.',
      );
    }
  }

  // If using any AWS or S3 compatible store (e.g. minio, aws s3 when using manual creds, ceph, etc.)
  let mc: MinioClient;
  try {
    mc = await new MinioClient(applyEndpointRewrite(config) as MinioClientOptions);
  } catch (err) {
    throw new Error(`Failed to create MinioClient: ${err}`, { cause: err });
  }
  return mc;
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
    const url = new URL(endpoint.match(/^https?:\/\//) ? endpoint : `http://${endpoint}`);
    return {
      host: url.hostname,
      port: url.port ? Number(url.port) : undefined,
      useSSL: endpoint.startsWith('https://')
        ? true
        : endpoint.startsWith('http://')
          ? false
          : undefined,
    };
  } catch (error) {
    const reason = error instanceof Error ? error.message : String(error);
    console.warn(`Ignoring invalid MinIO endpoint rewrite endpoint "${endpoint}": ${reason}`);
    return undefined;
  }
}

/**
 * Parse provider info for any S3-compatible store that is not AWS S3.
 *
 * Security: This reads a Kubernetes Secret named by the provider info. The
 * artifact handler only forwards provider info when the requested namespace is
 * the frontend server's own namespace, so this function never reads Secrets
 * from a customer namespace. In multi-user deployments the provider info is
 * dropped for user namespaces and credentials fall back to the server's own
 * environment credentials or the per-namespace artifact proxy.
 * See: https://github.com/kubeflow/pipelines/pull/12860
 */
async function parseS3ProviderInfo(
  config: MinioClientOptionsWithOptionalSecrets,
  providerInfo: S3ProviderInfo,
  namespace: string,
): Promise<MinioClientOptionsWithOptionalSecrets> {
  if (
    !providerInfo.Params.accessKeyKey ||
    !providerInfo.Params.secretKeyKey ||
    !providerInfo.Params.secretName
  ) {
    throw new Error(
      'Provider info with fromEnv:false supplied with incomplete secret credential info.',
    );
  }

  let parsedProviderEndpoint: ArtifactStoreEndpoint | undefined;
  if (providerInfo.Params.endpoint) {
    const insecure = providerInfo.Params.disableSSL?.toLowerCase() === 'true';
    parsedProviderEndpoint = parseArtifactStoreEndpoint(providerInfo.Params.endpoint, insecure);
    if (!parsedProviderEndpoint) {
      throw new Error('Provider info endpoint is not a valid HTTP(S) origin.');
    }
    if (!parsedProviderEndpoint.useSSL && isAWSS3Endpoint(parsedProviderEndpoint.endPoint)) {
      throw new Error('AWS S3 provider endpoints must use HTTPS.');
    }
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

  if (parsedProviderEndpoint) {
    config.endPoint = parsedProviderEndpoint.endPoint;
    config.port = parsedProviderEndpoint.port;
    config.useSSL = parsedProviderEndpoint.useSSL;
  }

  if (providerInfo.Params.region) {
    config.region = providerInfo.Params.region;
  } else if (!isAWSS3Endpoint(config.endPoint)) {
    config.region = undefined;
  }
  if (!providerInfo.Params.endpoint && providerInfo.Params.disableSSL) {
    config.useSSL = !(providerInfo.Params.disableSSL.toLowerCase() === 'true');
  }
  return config;
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
 * @param param.bucket Bucket name to retrieve the object from.
 * @param param.key Key of the object to retrieve.
 * @param param.client Minio client.
 * @param param.tryExtract Whether we try to extract *.tar.gz, default to true.
 * @param param.onError Optional asynchronous error callback. The returned
 * stream also emits the same error; callers that omit this callback must
 * attach their own listener if they need request-specific recovery.
 *
 */
export async function getObjectStream({
  bucket,
  key,
  client,
  tryExtract = true,
  onError,
  onTransformationDetermined,
}: MinioRequestConfig): Promise<Transform> {
  const source = await client.getObject(bucket, key);
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
  signal?: AbortSignal,
): AsyncGenerator<{ name: string; size: number }> {
  const PAGE_SIZE = 300;
  const listObjectsV2Query = getListObjectsV2Query(client);
  let continuationToken = '';
  let isTruncated = true;

  while (isTruncated) {
    if (signal?.aborted) {
      throw getMinioAbortReason(signal);
    }
    const pageRequest = listObjectsV2Query(bucket, prefix, continuationToken, '', PAGE_SIZE, '');
    const page = signal ? await waitForMinioOperation(pageRequest, signal) : await pageRequest;

    for (const item of page.objects) {
      if (item.name) {
        yield { name: item.name, size: item.size ?? 0 };
      }
    }

    isTruncated = page.isTruncated;
    continuationToken = page.nextContinuationToken;
  }
}

function getMinioAbortReason(signal: AbortSignal): Error {
  return signal.reason instanceof Error ? signal.reason : new Error('MinIO operation was aborted');
}

function waitForMinioOperation<T>(operation: Promise<T>, signal: AbortSignal): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    let settled = false;
    const cleanup = () => signal.removeEventListener('abort', rejectOnAbort);
    const rejectOnAbort = () => {
      if (settled) {
        return;
      }
      settled = true;
      cleanup();
      reject(getMinioAbortReason(signal));
    };

    if (signal.aborted) {
      rejectOnAbort();
    } else {
      signal.addEventListener('abort', rejectOnAbort, { once: true });
    }

    void operation.then(
      (value) => {
        if (settled) {
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
