// Copyright 2019 Google LLC
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
import * as k8sHelper from '../k8s-helper';
import {
  createPodLogsMinioRequestConfig,
  composePodLogsStreamHandler,
  getPodLogsStreamFromK8s,
  getPodLogsStreamFromWorkflow,
  toGetPodLogsStream,
} from '../workflow-helper';
import { ArgoConfigs, MinioConfigs, AWSConfigs } from '../configs';

/**
 * Returns a handler which attempts to retrieve the logs for the specific pod,
 * in the following order:
 * - retrieve with k8s api
 * - retrieve log archive location from argo workflow status, and retrieve artifact directly
 * - retrieve log archive with the provided argo archive settings
 * @param argoOptions fallback options to retrieve log archive
 * @param artifactsOptions configs and credentials for the different artifact backend
 */
export function getPodLogsHandler(
  argoOptions: ArgoConfigs,
  artifactsOptions: {
    minio: MinioConfigs;
    aws: AWSConfigs;
  },
): Handler {
  const { archiveLogs, archiveArtifactory, archiveBucketName, archivePrefix = '' } = argoOptions;

  // get pod log from the provided bucket and prefix.
  const getPodLogsStreamFromArchive = toGetPodLogsStream(
    createPodLogsMinioRequestConfig(
      archiveArtifactory === 'minio' ? artifactsOptions.minio : artifactsOptions.aws,
      archiveBucketName,
      archivePrefix,
    ),
  );

  // get the pod log stream (with fallbacks).
  const getPodLogsStream = composePodLogsStreamHandler(
    getPodLogsStreamFromK8s,
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
    const podName = decodeURIComponent(req.query.podname);

    // This is optional.
    // Note decodeURIComponent(undefined) === 'undefined', so I cannot pass the argument directly.
    const podNamespace = decodeURIComponent(req.query.podnamespace || '') || undefined;

    try {
      const stream = await getPodLogsStream(podName, podNamespace);
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
