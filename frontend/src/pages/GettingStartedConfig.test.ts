// @vitest-environment node
/*
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import fs from 'fs';
import path from 'path';

const backendConfigPath = path.resolve(
  process.cwd(),
  '../backend/src/apiserver/config/sample_config.json',
);
const frontendConfigPath = path.resolve(
  process.cwd(),
  'src/config/sample_config_from_backend.json',
);

describe('src/config/sample_config_from_backend.json', () => {
  it(`should be in sync with ${backendConfigPath}, if not please run "npm run sync-backend-sample-config" to update.`, () => {
    const configBackend = JSON.parse(fs.readFileSync(backendConfigPath, 'utf8'));
    const configFrontend = JSON.parse(fs.readFileSync(frontendConfigPath, 'utf8'));
    expect(configFrontend).toEqual(
      configBackend.pipelines.map((sample: { displayName: string }) => sample.displayName),
    );
  });
});
