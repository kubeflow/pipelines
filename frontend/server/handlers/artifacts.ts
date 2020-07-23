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
import { PreviewStream } from '../utils';
import { createMinioClient, getObjectStream } from '../minio-helper';
import * as k8sHelper from '../k8s-helper';
import * as serverInfo from '../helpers/server-info';
import { Handler, Request, Response } from 'express';
import { Storage } from '@google-cloud/storage';
import proxy from 'http-proxy-middleware';

import * as fs from 'fs';
import * as path from 'path';

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
 */
export function getArtifactsHandler(artifactsConfigs: {
  aws: AWSConfigs;
  http: HttpConfigs;
  minio: MinioConfigs;
}): Handler {
  const { aws, http, minio } = artifactsConfigs;
  return async (req, res) => {
    const { source, bucket, key: encodedKey, peek = 0 } = req.query as Partial<
      ArtifactsQueryStrings
    >;
    if (!source) {
      res.status(500).send('Storage source is missing from artifact request');
      return;
    }
    if (!bucket) {
      res.status(500).send('Storage bucket is missing from artifact request');
      return;
    }
    if (!encodedKey) {
      res.status(500).send('Storage key is missing from artifact request');
      return;
    }
    const key = decodeURIComponent(encodedKey);
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
  options: { bucket: string; key: string; client: MinioClient },
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
    let filePath;
    let volumeMount;
    try {
      // get ml-pipeline-ui pod namespace
      const SERVER_NAMESPACE = serverInfo.getServerNamespace();
      // get ml-pipeline-ui pod name
      const POD_NAME = serverInfo.getServerPodName();

      // only check when POD_NAME and serverNamespace can be obtained
      if (POD_NAME && SERVER_NAMESPACE) {
        const [pod, err] = await k8sHelper.getPod(POD_NAME, SERVER_NAMESPACE);
        if (err) {
          const { message, additionalInfo } = err;
          console.error(message, additionalInfo);
          res.status(500).send(message);
          return;
        }

        if (!pod) {
          const message = `Could not get pod ${POD_NAME} in namespace ${SERVER_NAMESPACE}`;
          res.status(500).send(message);
          return;
        }

        // volumes not specified or volume named ${bucket} not specified
        if (!Array.isArray(pod?.spec?.volumes) || !pod.spec.volumes.find(v => v?.name === bucket)) {
          const message = `Failed to open volume://${bucket}/${key}, volume ${bucket} not exist`;
          res.status(400).send(message);
          return;
        }

        // find volumes mount
        if (
          Array.isArray(pod?.spec?.containers) &&
          Array.isArray(pod.spec.containers[0]?.volumeMounts)
        ) {
          volumeMount = pod.spec.containers[0].volumeMounts.find(v => {
            // volume name must be same
            if (v?.name === bucket) {
              if (v?.subPath) {
                return key.startsWith(v.subPath);
              } else {
                return true;
              }
            }
            return false;
          });
        }
      } else {
        console.error(
          `pod name or server namespace can't be obtained, pod name: ${POD_NAME}, server namespace: ${SERVER_NAMESPACE}`,
        );
        res
          .status(500)
          .send(
            `Failed to open volume://${bucket}/${key}, can't obtain server pod name or server pod namespace`,
          );
        return;
      }

      // volumes mount not exist
      if (!volumeMount) {
        const message = `Failed to open volume://${bucket}/${key}, volume mount not exist`;
        res.status(400).send(message);
        return;
      }

      // relative file path
      const relativeFilePath = volumeMount.subPath
        ? key.substring(volumeMount.subPath.length)
        : key;
      filePath = path.join(volumeMount.mountPath, relativeFilePath);

      if (!fs.existsSync(filePath)) {
        res
          .status(400)
          .send(
            `Failed to open volume://${bucket}/${key}, file ${relativeFilePath} in volume ${bucket}${
              volumeMount.subPath ? ':' + volumeMount.subPath : ''
            } not exist`,
          );
        return;
      }

      // TODO: support directory and support filePath include wildcards '*'
      const stat = fs.statSync(filePath);
      if (stat.isDirectory()) {
        res
          .status(400)
          .send(`Failed to open volume local file: ${filePath}, directory does not support`);
        return;
      }

      fs.createReadStream(filePath)
        .pipe(new PreviewStream({ peek }))
        .pipe(res);
    } catch (err) {
      res.status(500).send(`Failed to get volume local file ${filePath}: ${err}`);
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
      target: '/artifacts/get',
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
