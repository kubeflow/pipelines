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

export interface IHealthzStats {
  apiServerCommitHash: string;
  apiServerReady: boolean;
  buildDate: string;
  frontendCommitHash: string;
}

export function getHealthzEndpoint(apiServerAddress: string, apiVersion: string) {
  return `${apiServerAddress}/${apiVersion}/healthz`;
}

export function getBuildMetadata(currentDir: string = path.resolve(__dirname)) {
  const buildDatePath = path.join(currentDir, 'BUILD_DATE');
  const commitHashPath = path.join(currentDir, 'COMMIT_HASH');
  const buildDate = fs.existsSync(buildDatePath)
    ? fs.readFileSync(buildDatePath, 'utf-8').trim()
    : '';
  const frontendCommitHash = fs.existsSync(commitHashPath)
    ? fs.readFileSync(commitHashPath, 'utf-8').trim()
    : '';
  return {
    buildDate,
    frontendCommitHash,
  };
}

export function getHealthzHandler(options: {
  healthzStats: Partial<IHealthzStats>;
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
    } catch (e) {
      healthzStats.apiServerReady = false;
    }
    res.json(healthzStats);
  };
}
