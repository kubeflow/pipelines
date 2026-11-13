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
import { Handler } from 'express';
import {
  createAuthorizedPodLogsReader,
  PodLogAccessError,
} from '../helpers/authorized-pod-logs.js';
import {
  createPodLogsMinioRequestConfig,
  composePodLogsStreamHandler,
  getPodLogsStreamFromK8s,
  createPodLogsStreamFromWorkflow,
  toGetPodLogsStream,
} from '../workflow-helper.js';
import { ArgoConfigs, MinioConfigs, AWSConfigs } from '../configs.js';
import {
  AuthorizeRequestResources,
  AuthorizeRequestVerb,
} from '../src/generated/apis/auth/index.js';
import { AuthorizeFn } from '../helpers/auth.js';
import { getArtifactStoreOrigin } from '../minio-helper.js';

/**
 * Multi-user reads authorize native run identity through the API and constrain archive keys.
 * Standalone reads retain the existing fallback order for the specific pod,
 * in the following order:
 * - retrieve with k8s api
 * - retrieve log archive location from argo workflow status, and retrieve artifact directly
 * - retrieve log archive with the provided argo archive settings
 * @param argoOptions fallback options to retrieve log archive
 * @param artifactsOptions configs and credentials for the different artifact backend
 * @param authorizeFn function to authorize namespace access
 * @param authEnabled whether namespace authorization checks are enabled
 */
export function getPodLogsHandler(
  argoOptions: ArgoConfigs,
  artifactsOptions: {
    minio: MinioConfigs;
    aws: AWSConfigs;
    allowedEndpoints?: string[];
  },
  podLogContainerName: string,
  authorizeFn: AuthorizeFn,
  authEnabled: boolean,
  apiServerAddress?: string,
  apiPrefix?: string,
): Handler {
  const {
    archiveLogs,
    archiveArtifactory,
    archiveBucketName,
    keyFormat,
    artifactRepositoriesLookup,
  } = argoOptions;

  const trustedEndpoints = [
    getArtifactStoreOrigin(artifactsOptions.minio),
    getArtifactStoreOrigin(artifactsOptions.aws),
    ...(argoOptions.artifactRepositoryEndpoints || []),
    ...(artifactsOptions.allowedEndpoints || []),
  ].filter((endpoint): endpoint is string => endpoint !== undefined);
  const getPodLogsStreamFromWorkflow = createPodLogsStreamFromWorkflow(trustedEndpoints);

  // get pod log from the provided bucket and keyFormat.
  const getPodLogsStreamFromArchive = toGetPodLogsStream(
    createPodLogsMinioRequestConfig(
      archiveArtifactory === 'minio' ? artifactsOptions.minio : artifactsOptions.aws,
      archiveBucketName,
      keyFormat,
      artifactRepositoriesLookup,
    ),
  );

  // get the pod log stream (with fallbacks).
  const getPodLogsStream = composePodLogsStreamHandler(
    (podName: string, createdAt: string, namespace?: string) => {
      return getPodLogsStreamFromK8s(podName, createdAt, namespace, podLogContainerName);
    },
    // if archive logs flag is set, then final attempt will try to retrieve the artifacts
    // from the bucket and prefix provided in the config. Otherwise, only attempts
    // to read from worflow status if the default handler fails.
    archiveLogs && archiveBucketName
      ? composePodLogsStreamHandler(getPodLogsStreamFromWorkflow, getPodLogsStreamFromArchive)
      : getPodLogsStreamFromWorkflow,
  );

  const authorizedReader =
    authEnabled && apiServerAddress && apiPrefix
      ? createAuthorizedPodLogsReader(
          apiServerAddress,
          apiPrefix,
          argoOptions,
          artifactsOptions,
          podLogContainerName,
        )
      : undefined;

  return async (req, res) => {
    res.setHeader('X-Content-Type-Options', 'nosniff');
    res.setHeader('Content-Disposition', 'attachment');
    res.type('text/plain');

    if (!req.query.podname) {
      res.status(400).send('podname argument is required');
      return;
    }
    const podName = authEnabled
      ? String(req.query.podname)
      : decodeURIComponent(req.query.podname as string);
    const createdAt = authEnabled ? '' : decodeURIComponent((req.query.createdat as string) || '');

    // This is optional.
    // Note decodeURIComponent(undefined) === 'undefined', so I cannot pass the argument directly.
    const podNamespace = authEnabled
      ? undefined
      : decodeURIComponent((req.query.podnamespace as string) || '') || undefined;

    // Check access to namespace if podNamespace is provided
    if (!authEnabled && podNamespace) {
      try {
        const authError = await authorizeFn(
          {
            verb: AuthorizeRequestVerb.GET,
            resources: AuthorizeRequestResources.VIEWERS,
            namespace: podNamespace,
          },
          req,
        );
        if (authError) {
          res.status(403).send('Access denied to namespace');
          return;
        }
      } catch (error) {
        console.error('Authorization error:', error);
        res.status(500).send('Authorization check failed');
        return;
      }
    }

    try {
      if (authEnabled && !authorizedReader) {
        throw new PodLogAccessError(503, 'Run log authorization is not configured');
      }
      const stream = authorizedReader
        ? await authorizedReader(req)
        : await getPodLogsStream(podName, createdAt, podNamespace);
      if (res.destroyed || res.writableEnded) {
        stream.destroy();
        return;
      }
      let settled = false;
      const cleanup = () => {
        stream.off('error', failOnce);
        stream.off('end', finishOnce);
        res.off('close', stopOnPrematureClose);
      };
      const failOnce = (err: Error) => {
        if (settled) {
          return;
        }
        settled = true;
        cleanup();
        stream.destroy();
        if (res.headersSent) {
          console.error('[pod-logs] aborting committed response:', err);
          res.destroy();
          return;
        }
        if (
          err?.message &&
          err.message?.indexOf('Unable to find pod log archive information') > -1
        ) {
          res.status(404).send('pod not found');
        } else {
          res
            .status(err instanceof PodLogAccessError ? err.statusCode : 500)
            .send('Could not get main container logs: ' + err);
        }
      };
      const finishOnce = () => {
        if (settled) {
          return;
        }
        settled = true;
        cleanup();
        res.end();
      };
      const stopOnPrematureClose = () => {
        if (settled || res.writableFinished) {
          return;
        }
        settled = true;
        cleanup();
        stream.destroy();
      };
      stream.once('error', failOnce);
      stream.once('end', finishOnce);
      res.once('close', stopOnPrematureClose);

      // getObjectStream starts its pipeline before returning. A fast storage
      // failure can therefore precede this request-specific listener; inspect
      // the terminal state after attaching it so no event can fall in between.
      if (stream.errored) {
        failOnce(stream.errored);
        return;
      }
      if (stream.destroyed) {
        failOnce(new Error('Pod log stream closed before response streaming started'));
        return;
      }
      stream.pipe(res, { end: false });
    } catch (err) {
      res
        .status(err instanceof PodLogAccessError ? err.statusCode : 500)
        .send('Could not get main container logs: ' + err);
    }
  };
}
