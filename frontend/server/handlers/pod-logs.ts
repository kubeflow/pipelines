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
import podLogsHandler from '../workflow-helper';
import { IArgoConfigs, IMinioConfigs, IAWSConfigs } from '../configs';

export function getPodLogsHandler(
  argoOptions: IArgoConfigs,
  artifactsOptions: {
    minio: IMinioConfigs;
    aws: IAWSConfigs;
  },
): Handler {
  if (argoOptions.archiveLogs) {
    const minioClientOptions =
      argoOptions.archiveArtifactory === 'minio' ? artifactsOptions.minio : artifactsOptions.aws;
    podLogsHandler.setFallbackHandler(
      minioClientOptions,
      argoOptions.archiveBucketName,
      argoOptions.archivePrefix,
    );
  }
  return async (req, res) => {
    if (!k8sHelper.isInCluster) {
      res.status(500).send('Cannot talk to Kubernetes master');
      return;
    }

    if (!req.query.podname) {
      res.status(404).send('podname argument is required');
      return;
    }
    const podName = decodeURIComponent(req.query.podname);

    // This is optional.
    // Note decodeURIComponent(undefined) === 'undefined', so I cannot pass the argument directly.
    const podNamespace = decodeURIComponent(req.query.podnamespace || '') || undefined;

    try {
      const stream = await podLogsHandler.getPodLogs(podName, podNamespace);
      stream.on('error', err => res.status(500).send('Could not get main container logs: ' + err));
      stream.on('end', () => res.end());
      stream.pipe(res);
    } catch (err) {
      res.status(500).send('Could not get main container logs: ' + err);
    }
  };
}
