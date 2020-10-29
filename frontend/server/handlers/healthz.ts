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
import * as path from 'path';
import * as fs from 'fs';
import fetch from 'node-fetch';
import { Handler } from 'express';

/** HealthzStats describes the ml-pipeline ui server state. */
export interface HealthzStats {
  apiServerCommitHash: string;
  apiServerTagName: string;
  apiServerMultiUser: boolean;
  multi_user: boolean;
  apiServerReady: boolean;
  buildDate: string;
  frontendCommitHash: string;
  frontendTagName: string;
}

/**
 * Returns the url to the ml-pipeline api server healthz endpoint
 * (of form: `${apiServerAddress}/${apiVersionPrefix}/healthz`).
 * @param apiServerAddress address for the ml-pipeline api server.
 * @param apiVersionPrefix prefix to append to the route.
 */
export function getHealthzEndpoint(apiServerAddress: string, apiVersionPrefix: string) {
  return `${apiServerAddress}/${apiVersionPrefix}/healthz`;
}

/**
 * Returns the build date and frontend commit hash.
 * @param currentDir path to the metadata files (BUILD_DATE, COMMIT_HASH).
 */
export function getBuildMetadata(currentDir: string = path.resolve(__dirname)) {
  const buildDatePath = path.join(currentDir, 'BUILD_DATE');
  const commitHashPath = path.join(currentDir, 'COMMIT_HASH');
  const tagNamePath = path.join(currentDir, 'TAG_NAME');
  const buildDate = fs.existsSync(buildDatePath)
    ? fs.readFileSync(buildDatePath, 'utf-8').trim()
    : '';
  const frontendCommitHash = fs.existsSync(commitHashPath)
    ? fs.readFileSync(commitHashPath, 'utf-8').trim()
    : '';
  const frontendTagName = fs.existsSync(tagNamePath)
    ? fs.readFileSync(tagNamePath, 'utf-8').trim()
    : '';
  return {
    buildDate,
    frontendCommitHash,
    frontendTagName,
  };
}

/**
 * Returns a handler which return the current state of the server.
 * @param options.healthzStats  partial health stats to be enriched with ml-pipeline metadata.
 * @param options.healthzEndpoint healthz endpoint for the ml-pipeline api server.
 */
export function getHealthzHandler(options: {
  healthzStats: Partial<HealthzStats>;
  healthzEndpoint: string;
}): Handler {
  const { healthzStats = {}, healthzEndpoint } = options;
  return async (_, res) => {
    try {
      const response = await fetch(healthzEndpoint, {
        timeout: 1000,
      });
      healthzStats.apiServerReady = true;
      const serverStatus = await response.json();
      healthzStats.apiServerCommitHash = serverStatus.commit_sha;
      healthzStats.apiServerTagName = serverStatus.tag_name;
      healthzStats.apiServerMultiUser = serverStatus.multi_user;
      healthzStats.multi_user = serverStatus.multi_user;
    } catch (e) {
      healthzStats.apiServerReady = false;
    }
    res.json(healthzStats);
  };
}
