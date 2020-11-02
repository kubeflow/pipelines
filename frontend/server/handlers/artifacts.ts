// Copyright 2019-2020 Google LLC
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
import fetch from 'node-fetch';
import { AWSConfigs, HttpConfigs, MinioConfigs, ProcessEnv } from '../configs';
import { Client as MinioClient } from 'minio';
import { PreviewStream, findFileOnPodVolume } from '../utils';
import { createMinioClient, getObjectStream } from '../minio-helper';
import * as serverInfo from '../helpers/server-info';
import { Handler, Request, Response } from 'express';
import { Storage } from '@google-cloud/storage';
import proxy from 'http-proxy-middleware';
import { HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS } from '../consts';

import * as fs from 'fs';

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
}: {
  artifactsConfigs: {
    aws: AWSConfigs;
    http: HttpConfigs;
    minio: MinioConfigs;
  };
  tryExtract: boolean;
  useParameter: boolean;
}): Handler {
  const { aws, http, minio } = artifactsConfigs;
  return async (req, res) => {
    const source = useParameter ? req.params.source : req.query.source;
    const bucket = useParameter ? req.params.bucket : req.query.bucket;
    const key = useParameter ? req.params[0] : req.query.key;
    const { peek = 0 } = req.query as Partial<ArtifactsQueryStrings>;
    if (!source) {
      res.status(500).send('Storage source is missing from artifact request');
      return;
    }
    if (!bucket) {
      res.status(500).send('Storage bucket is missing from artifact request');
      return;
    }
    if (!key) {
      res.status(500).send('Storage key is missing from artifact request');
      return;
    }
    console.log(`Getting storage artifact at: ${source}: ${bucket}/${key}`);
    switch (source) {
      case 'gcs':
        getGCSArtifactHandler({ bucket, key }, peek)(req, res);
        break;

      case 'minio':
        getMinioArtifactHandler(
          {
            bucket,
            client: new MinioClient(minio),
            key,
            tryExtract,
          },
          peek,
        )(req, res);
        break;

      case 's3':
        getMinioArtifactHandler(
          {
            bucket,
            client: await createMinioClient(aws),
            key,
          },
          peek,
        )(req, res);
        break;

      case 'http':
      case 'https':
        getHttpArtifactsHandler(
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
        res.status(500).send('Unknown storage source: ' + source);
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
  url: string,
  auth: {
    key: string;
    defaultValue: string;
  } = { key: '', defaultValue: '' },
  peek: number = 0,
) {
  return async (req: Request, res: Response) => {
    const headers = {};

    // add authorization header to fetch request if key is non-empty
    if (auth.key.length > 0) {
      // inject original request's value if exists, otherwise default to provided default value
      headers[auth.key] =
        req.headers[auth.key] || req.headers[auth.key.toLowerCase()] || auth.defaultValue;
    }
    const response = await fetch(url, { headers });
    response.body
      .on('error', err => res.status(500).send(`Unable to retrieve artifact at ${url}: ${err}`))
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
        .on('error', err =>
          res
            .status(500)
            .send(
              `Failed to get object in bucket ${options.bucket} at path ${options.key}: ${err}`,
            ),
        )
        .pipe(new PreviewStream({ peek }))
        .pipe(res);
    } catch (err) {
      console.error(err);
      res
        .status(500)
        .send(`Failed to get object in bucket ${options.bucket} at path ${options.key}: ${err}`);
    }
  };
}

function getGCSArtifactHandler(options: { key: string; bucket: string }, peek: number = 0) {
  const { key, bucket } = options;
  return async (_: Request, res: Response) => {
    try {
      // Read all files that match the key pattern, which can include wildcards '*'.
      // The way this works is we list all paths whose prefix is the substring
      // of the pattern until the first wildcard, then we create a regular
      // expression out of the pattern, escaping all non-wildcard characters,
      // and we use it to match all enumerated paths.
      const storage = new Storage();
      const prefix = key.indexOf('*') > -1 ? key.substr(0, key.indexOf('*')) : key;
      const files = await storage.bucket(bucket).getFiles({ prefix });
      const matchingFiles = files[0].filter(f => {
        // Escape regex characters
        const escapeRegexChars = (s: string) => s.replace(/[|\\{}()[\]^$+*?.]/g, '\\$&');
        // Build a RegExp object that only recognizes asterisks ('*'), and
        // escapes everything else.
        const regex = new RegExp(
          '^' +
            key
              .split(/\*+/)
              .map(escapeRegexChars)
              .join('.*') +
            '$',
        );
        return regex.test(f.name);
      });

      if (!matchingFiles.length) {
        console.log('No matching files found.');
        res.send();
        return;
      }
      console.log(
        `Found ${matchingFiles.length} matching files: `,
        matchingFiles.map(file => file.name).join(','),
      );
      let contents = '';
      // TODO: support peek for concatenated matching files
      if (peek) {
        matchingFiles[0]
          .createReadStream()
          .pipe(new PreviewStream({ peek }))
          .pipe(res);
        return;
      }

      // if not peeking, iterate and append all the files
      matchingFiles.forEach((f, i) => {
        const buffer: Buffer[] = [];
        f.createReadStream()
          .on('data', data => buffer.push(Buffer.from(data)))
          .on('end', () => {
            contents +=
              Buffer.concat(buffer)
                .toString()
                .trim() + '\n';
            if (i === matchingFiles.length - 1) {
              res.send(contents);
            }
          })
          .on('error', () => res.status(500).send('Failed to read file: ' + f.name));
      });
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
        res.status(404).send(`Failed to open volume://${bucket}/${key}, ${parseError}`);
        return;
      }

      // TODO: support directory and support filePath include wildcards '*'
      const stat = await fs.promises.stat(filePath);
      if (stat.isDirectory()) {
        res
          .status(400)
          .send(
            `Failed to open volume://${bucket}/${key}, file ${filePath} is directory, does not support now`,
          );
        return;
      }

      fs.createReadStream(filePath)
        .pipe(new PreviewStream({ peek }))
        .pipe(res);
    } catch (err) {
      res.status(500).send(`Failed to open volume://${bucket}/${key}: ${err}`);
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
  namespacedServiceGetter,
}: {
  enabled: boolean;
  namespacedServiceGetter: NamespacedServiceGetter;
}): Handler {
  if (!enabled) {
    return (req, res, next) => next();
  }
  return proxy(
    (_pathname, req) => {
      // only proxy requests with namespace query parameter
      return !!getNamespaceFromUrl(req.url || '');
    },
    {
      changeOrigin: true,
      onProxyReq: proxyReq => {
        console.log('Proxied artifact request: ', proxyReq.path);
      },
      pathRewrite: (pathStr, req) => {
        const url = new URL(pathStr || '', DUMMY_BASE_PATH);
        url.searchParams.delete(QUERIES.NAMESPACE);
        return url.pathname + url.search;
      },
      router: req => {
        const namespace = getNamespaceFromUrl(req.url || '');
        if (!namespace) {
          throw new Error(`namespace query param expected in ${req.url}.`);
        }
        return namespacedServiceGetter(namespace);
      },
      target: '/artifacts',
      headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
    },
  );
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
