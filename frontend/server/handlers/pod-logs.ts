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
import * as k8sHelper from '../k8s-helper.js';
import {
  createPodLogsMinioRequestConfig,
  composePodLogsStreamHandler,
  getPodLogsStreamFromK8s,
  getPodLogsStreamFromWorkflow,
  toGetPodLogsStream,
} from '../workflow-helper.js';
import { ArgoConfigs, MinioConfigs, AWSConfigs } from '../configs.js';
import {
  AuthorizeRequestResources,
  AuthorizeRequestVerb,
} from '../src/generated/apis/auth/index.js';
import { AuthorizeFn } from '../helpers/auth.js';

/**
 * Returns a handler which attempts to retrieve the logs for the specific pod,
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
  },
  podLogContainerName: string,
  authorizeFn: AuthorizeFn,
  authEnabled: boolean,
): Handler {
  const {
    archiveLogs,
    archiveArtifactory,
    archiveBucketName,
    keyFormat,
    artifactRepositoriesLookup,
  } = argoOptions;

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

  return async (req, res) => {
    if (!req.query.podname) {
      res.status(400).send('podname argument is required');
      return;
    }
    const podName = decodeURIComponent(req.query.podname as string);
    const createdAt = decodeURIComponent((req.query.createdat as string) || '');

    // This is optional.
    // Note decodeURIComponent(undefined) === 'undefined', so I cannot pass the argument directly.
    const podNamespace = decodeURIComponent((req.query.podnamespace as string) || '') || undefined;

    // In multi-user mode, namespace must be explicit so authz cannot be bypassed.
    if (authEnabled && !podNamespace) {
      res.status(422).send('podnamespace argument is required');
      return;
    }

    // Check access to namespace if podNamespace is provided
    if (podNamespace) {
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
      const stream = await getPodLogsStream(podName, createdAt, podNamespace);
      stream.on('error', err => {
        if (
          err?.message &&
          err.message?.indexOf('Unable to find pod log archive information') > -1
        ) {
          res.status(404).send('pod not found');
        } else {
          res.status(500).send('Could not get main container logs: ' + err);
        }
      });
      stream.on('end', () => res.end());
      stream.pipe(res);
    } catch (err) {
      res.status(500).send('Could not get main container logs: ' + err);
    }
  };
}
