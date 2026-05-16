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
} from '../utils.js';
import {
  createMinioClient,
  getObjectStream,
  isNoSuchKeyError,
  listObjectsUnderPrefix,
  summarizeDirectoryUnderPrefix,
} from '../minio-helper.js';
import * as tar from 'tar-stream';
import * as zlib from 'zlib';
import * as serverInfo from '../helpers/server-info.js';
import { Handler, Request, Response, NextFunction } from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import { HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS } from '../consts.js';
import { URL } from 'url';
import { getGCSClient, listGCSObjectNames, downloadGCSObjectStream } from '../gcs-helper.js';
import type { GCSClient } from '../gcs-helper.js';

import * as fs from 'fs';
import { isAllowedDomain } from './domain-checker.js';
import { getK8sSecret } from '../k8s-helper.js';
import { CredentialBody } from 'google-auth-library';
import { AuthorizeFn } from '../helpers/auth.js';
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
}

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
 * Note: Secret-backed provider mode (fromEnv === 'false') is unsupported
 * in multi-user deployments. The ml-pipeline-ui ClusterRole no longer
 * grants secrets:get/list permissions, so getK8sSecret() calls will be
 * denied by RBAC at the cluster level. This mode may still work in
 * standalone (single-tenant) deployments where the service account has
 * direct secret access. See: https://github.com/kubeflow/pipelines/pull/12860
 *
 * Security: This addresses the vulnerability where the namespace parameter
 * could be manipulated to access artifacts from other namespaces.
 * See https://github.com/kubeflow/pipelines/issues/9889
 *
 * @param authorizeFn The authorization function to validate permissions
 * @param authEnabled Whether authorization is enabled
 * @param kubeflowUserIdHeader The header name containing the user identity
 */
export function getArtifactsAuthMiddleware(
  authorizeFn: AuthorizeFn,
  authEnabled: boolean,
  kubeflowUserIdHeader: string,
): Handler {
  return async (request: Request, response: Response, next: NextFunction) => {
    if (!authEnabled) {
      return next();
    }

    const userId = request.headers[kubeflowUserIdHeader.toLowerCase()];
    if (!userId) {
      console.warn(
        `[SECURITY] Unauthenticated artifact access attempt. Path: ${request.originalUrl}`,
      );
      response.status(401).send('Authentication required for artifact access');
      return;
    }

    const rawNamespace = request.query.namespace;
    const namespace: string | undefined = Array.isArray(rawNamespace)
      ? String(rawNamespace[0])
      : typeof rawNamespace === 'string'
        ? rawNamespace
        : undefined;

    if (!namespace) {
      console.warn(
        `[SECURITY] Missing namespace parameter. ` +
          `User: ${userId}, Path: ${request.originalUrl}`,
      );
      response.status(400).send('Namespace parameter is required when authentication is enabled');
      return;
    }

    if (!isAllowedResourceName(namespace)) {
      console.warn(
        `[SECURITY] Invalid namespace format. ` +
          `User: ${userId}, ` +
          `Namespace: ${namespace}, Path: ${request.originalUrl}`,
      );
      response.status(400).send('Invalid namespace format');
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
      response.status(403).send(authError.message);
      return;
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
 * @param tryExtract whether the handler try to extract content from *.tar.gz files.
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
    const source = (useParameter ? req.params.source : req.query.source) as string | undefined;
    const bucket = (useParameter ? req.params.bucket : req.query.bucket) as string | undefined;
    const key = (useParameter ? req.params[0] : req.query.key) as string | undefined;
    const {
      peek = 0,
      providerInfo = '',
      // When auth is enabled, the authorization middleware has already validated
      // and required the namespace parameter before this handler runs.
      // When auth is disabled (standalone mode), fallback to serverNamespace
      // for provider-based credential lookups via getK8sSecret (Issue #9889).
      namespace = options.server.serverNamespace,
    } = req.query as Partial<ArtifactsQueryStrings>;
    if (!source) {
      res.status(500).send('Storage source is missing from artifact request');
      return;
    }
    if (!bucket) {
      res.status(500).send('Storage bucket is missing from artifact request');
      return;
    }
    if (!isAllowedResourceName(bucket)) {
      res.status(500).send('Invalid bucket name');
      return;
    }
    if (!key) {
      res.status(500).send('Storage key is missing from artifact request');
      return;
    }
    if (key.length > 1024) {
      res.status(500).send('Object key too long');
      return;
    }
    console.log(`Getting storage artifact at: ${source}: ${bucket}/${key}`);

    let client: MinioClient;
    switch (source) {
      case 'gcs':
        await getGCSArtifactHandler({ bucket, key }, peek, providerInfo, namespace)(req, res);
        break;
      case 'minio':
        try {
          client = await createMinioClient(minio, 'minio', providerInfo, namespace);
        } catch (e) {
          res.status(500).send(`Failed to initialize Minio Client for Minio Provider: ${e}`);
          return;
        }
        await getMinioArtifactHandler(
          {
            bucket,
            client,
            key,
            tryExtract,
          },
          peek,
        )(req, res);
        break;
      case 's3':
        try {
          client = await createMinioClient(aws, 's3', providerInfo, namespace);
        } catch (e) {
          res.status(500).send(`Failed to initialize Minio Client for S3 Provider: ${e}`);
          return;
        }
        await getMinioArtifactHandler(
          {
            bucket,
            client,
            key,
          },
          peek,
        )(req, res);
        break;
      case 'http':
      case 'https':
        await getHttpArtifactsHandler(
          allowedDomain,
          getHttpUrl(source, http.baseUrl || '', bucket, key),
          http.auth,
          peek,
        )(req, res);
        break;
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
        res.status(500).send('Unknown storage source');
        return;
    }
  };
}

/**
 * Returns the http/https url to retrieve a kfp artifact (of the form: `${source}://${baseUrl}${bucket}/${key}`)
 * @param source "http" or "https".
 * @param baseUrl string to prefix the url.
 * @param bucket name of the bucket.
 * @param key path to the artifact.
 */
function getHttpUrl(source: 'http' | 'https', baseUrl: string, bucket: string, key: string) {
  // trim `/` from both ends of the base URL, then append with a single `/` to the end (empty string remains empty)
  baseUrl = baseUrl.replace(/^\/*(.+?)\/*$/, '$1/');
  return `${source}://${baseUrl}${bucket}/${key}`;
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
    if (!isAllowedDomain(url, allowedDomain)) {
      res.status(500).send(`Domain not allowed.`);
      return;
    }
    const response = await fetch(url, { headers });
    if (!response.body) {
      res.status(500).send('Unable to retrieve artifact: empty response body');
      return;
    }
    const { Readable } = await import('stream');
    const nodeStream = Readable.fromWeb(response.body as any);
    nodeStream
      .on('error', (err: Error) => res.status(500).send(`Unable to retrieve artifact: ${err}`))
      .pipe(new PreviewStream({ peek }))
      .pipe(res);
  };
}

function getMinioArtifactHandler(
  options: { bucket: string; key: string; client: MinioClient; tryExtract?: boolean },
  peek: number = 0,
) {
  return async (_: Request, res: Response) => {
    try {
      const stream = await getObjectStream(options);
      stream
        .on('error', (err) => res.status(500).send(`Failed to get object in bucket: ${err}`))
        .pipe(new PreviewStream({ peek }))
        .pipe(res);
    } catch (err) {
      // In KFP v2, output artifacts may be directories (prefixes) rather than
      // single objects. Fall back to packaging the contents of the prefix as
      // a .tar.gz so users can still download them. See
      // https://github.com/kubeflow/pipelines/issues/7809
      if (isNoSuchKeyError(err)) {
        if (peek > 0) {
          // Preview request (e.g. the run details panel calls
          // Apis.readFile with a small peek size). We must not stream a
          // full directory archive here — that would list every object
          // under the prefix and gzip the whole tree just to render a few
          // KB of inline text. Instead, answer with a small text summary
          // backed by one capped ListObjectsV2 call: cost stays bounded
          // (one round trip, <1KB body) and the user sees that the
          // artifact is a directory with N files.
          try {
            await previewDirectorySummary(options, res);
            return;
          } catch (summaryErr) {
            console.error(summaryErr);
            res.status(500).send(`Failed to summarize directory: ${summaryErr}`);
            return;
          }
        }
        try {
          await streamDirectoryAsTarGz(options, res);
          return;
        } catch (tarErr) {
          console.error(tarErr);
          if (!res.headersSent) {
            res.status(500).send(`Failed to get object in bucket: ${tarErr}`);
          } else {
            res.end();
          }
          return;
        }
      }
      console.error(err);
      res.status(500).send(`Failed to get object in bucket: ${err}`);
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
    res.status(404).send(`No objects found at ${bucket}/${key}`);
    return;
  }
  const baseName = key.replace(/\/+$/, '').split('/').pop() || 'artifact';
  const countLabel = `${summary.count}${summary.truncated ? '+' : ''}`;
  res
    .type('text/plain')
    .send(`Directory artifact "${baseName}" — ${countLabel} file(s). Download to view contents.\n`);
}

async function streamDirectoryAsTarGz(
  options: { bucket: string; key: string; client: MinioClient },
  res: Response,
) {
  const { bucket, key, client } = options;
  // Trailing slash so prefix "foo" doesn't also match sibling key "foobar".
  const prefix = key.endsWith('/') ? key : `${key}/`;

  // Peek the first object before sending headers so an empty prefix can still
  // produce a 404 instead of an empty 200 tarball.
  const iterator = listObjectsUnderPrefix(client, bucket, prefix);
  const first = await iterator.next();
  if (first.done) {
    res.status(404).send(`No objects found at ${bucket}/${key}`);
    return;
  }

  const baseName = key.replace(/\/+$/, '').split('/').pop() || 'artifact';
  res.setHeader('Content-Type', 'application/gzip');
  res.setHeader('Content-Disposition', buildAttachmentDisposition(`${baseName}.tar.gz`));

  const pack = tar.pack();
  const gzip = zlib.createGzip();
  pack.pipe(gzip).pipe(res);

  const writeEntry = async ({ name, size }: { name: string; size: number }) => {
    const relativeName = name.startsWith(prefix) ? name.slice(prefix.length) : name;
    const safeName = sanitizeTarEntryName(relativeName);
    if (!safeName) {
      // Skip directory-marker objects (key === prefix) and any keys that
      // sanitize to an empty path.
      return;
    }
    const objStream = await client.getObject(bucket, name);
    await new Promise<void>((resolve, reject) => {
      const entry = pack.entry({ name: safeName, size }, (err) => (err ? reject(err) : resolve()));
      objStream.on('error', reject);
      objStream.pipe(entry);
    });
  };

  try {
    await writeEntry(first.value);
    for await (const item of iterator) {
      await writeEntry(item);
    }
  } finally {
    pack.finalize();
  }
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
 * Parses GCS provider info and retrieves credentials from a Kubernetes secret.
 *
 * WARNING: This function is unsupported in multi-user deployments.
 * The ml-pipeline-ui ClusterRole no longer grants secrets:get/list
 * permissions, so getK8sSecret() calls will be denied by RBAC.
 * See: https://github.com/kubeflow/pipelines/pull/12860
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
    throw new Error('Failed to parse GCS Provider config. Error: ' + err);
  }
}

async function readGCSObjectText(
  bucket: string,
  objectName: string,
  client: GCSClient,
  credentials?: CredentialBody,
): Promise<string> {
  const stream = await downloadGCSObjectStream({ bucket, objectName, credentials, client });
  const chunks: Buffer[] = [];
  for await (const chunk of stream) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  return Buffer.concat(chunks).toString();
}

function getGCSArtifactHandler(
  options: { key: string; bucket: string },
  peek: number = 0,
  providerInfoString?: string,
  namespace?: string,
) {
  const { key, bucket } = options;
  return async (_: Request, res: Response) => {
    try {
      let credentials: CredentialBody | undefined;
      if (providerInfoString) {
        const providerInfo = parseJSONString<GCSProviderInfo>(providerInfoString);
        if (providerInfo && providerInfo.Params.fromEnv === 'false') {
          if (!namespace) {
            res.status(500).send('Failed to parse provider info. Reason: No namespace provided');
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
        res.send();
        return;
      }
      console.log(`Found ${matchingFiles.length} matching files: `, matchingFiles.join(','));
      let contents = '';
      // TODO: support peek for concatenated matching files
      if (peek) {
        const stream = await downloadGCSObjectStream({
          bucket,
          client,
          credentials,
          objectName: matchingFiles[0],
        });
        stream.pipe(new PreviewStream({ peek })).pipe(res);
        return;
      }

      // if not peeking, iterate and append all the files
      for (const fileName of matchingFiles) {
        contents += (await readGCSObjectText(bucket, fileName, client, credentials)).trim() + '\n';
      }
      res.send(contents);
    } catch (err) {
      res.status(500).send('Failed to download GCS file(s). Error: ' + err);
    }
  };
}

function getVolumeArtifactsHandler(options: { bucket: string; key: string }, peek: number = 0) {
  const { key, bucket } = options;
  return async (req: Request, res: Response) => {
    try {
      const [pod, err] = await serverInfo.getHostPod();
      if (err) {
        res.status(500).send(err);
        return;
      }

      if (!pod) {
        res.status(500).send('Could not get server pod');
        return;
      }

      // ml-pipeline-ui server container name also be called 'ml-pipeline-ui-artifact' in KFP multi user mode.
      // https://github.com/kubeflow/manifests/blob/master/pipeline/installs/multi-user/pipelines-profile-controller/sync.py#L212
      const [filePath, parseError] = findFileOnPodVolume(pod, {
        containerNames: ['ml-pipeline-ui', 'ml-pipeline-ui-artifact'],
        volumeMountName: bucket,
        filePathInVolume: key,
      });
      if (parseError) {
        console.log(`Failed to open volume: ${parseError}`);
        res.status(404).send(`Failed to open volume.`);
        return;
      }

      // TODO: support directory and support filePath include wildcards '*'
      const stat = await fs.promises.stat(filePath);
      if (stat.isDirectory()) {
        res
          .status(400)
          .send(`Failed to open volume file ${filePath} is directory, does not support now`);
        return;
      }

      fs.createReadStream(filePath).pipe(new PreviewStream({ peek })).pipe(res);
    } catch (err) {
      console.log(`Failed to open volume: ${err}`);
      res.status(500).send(`Failed to open volume.`);
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
  const proxy = createProxyMiddleware(
    (_pathname, req) => {
      // only proxy requests with namespace query parameter
      return !!getNamespaceFromUrl(req.url || '');
    },
    {
      changeOrigin: true,
      onProxyReq: (proxyReq) => {
        console.log('Proxied artifact request: ', proxyReq.path);
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
    },
  );
  return (req, res, next) => {
    const namespace = getNamespaceFromUrl(req.url || '');
    if (namespace && !isAllowedResourceName(namespace)) {
      res.status(400).send('Invalid namespace');
      return;
    }
    proxy(req, res, next);
  };
}

function getNamespaceFromUrl(path: string): string | undefined {
  // Gets namespace from query parameter "namespace"
  const params = new URL(path, DUMMY_BASE_PATH).searchParams;
  return params.get('namespace') || undefined;
}

// `new URL('/path')` doesn't work, because URL only accepts full URL with scheme and hostname.
// We use the DUMMY_BASE_PATH like `new URL('/path', DUMMY_BASE_PATH)`, so that URL can parse paths
// properly.
const DUMMY_BASE_PATH = 'http://dummy-base-path';

export function getArtifactServiceGetter({ serviceName, servicePort }: ArtifactsProxyConfig) {
  return (namespace: string) => `http://${serviceName}.${namespace}:${servicePort}`;
}
