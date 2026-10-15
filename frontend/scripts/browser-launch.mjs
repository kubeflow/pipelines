/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import assert from 'node:assert/strict';
import { chromium, firefox, webkit } from 'playwright';

export function browserLaunchConfiguration(env = process.env) {
  const name = env.KFP_BROWSER || 'chromium';
  assert.ok(['chromium', 'firefox', 'webkit'].includes(name), `Unsupported KFP_BROWSER: ${name}`);
  const options = env.KFP_BROWSER_EXECUTABLE_PATH
    ? { executablePath: env.KFP_BROWSER_EXECUTABLE_PATH }
    : name === 'chromium' && env.PLAYWRIGHT_CHANNEL
      ? { channel: env.PLAYWRIGHT_CHANNEL }
      : {};
  return { name, options };
}

export function launchBrowser() {
  const { name, options } = browserLaunchConfiguration();
  return { chromium, firefox, webkit }[name].launch(options);
}
