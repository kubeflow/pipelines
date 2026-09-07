// Copyright 2026 The Kubeflow Authors
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

// Compatibility checks for the WebdriverIO dependency tree. They need neither
// Selenium nor a deployed cluster and run in a plain Node.js job in CI.
// See README.md, "Dependency compatibility checks".

import assert from "node:assert/strict";
import { mkdtemp, readdir, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import * as browsers from "@puppeteer/browsers";
import { ConfigParser } from "@wdio/config/node";
import { setupBrowser, setupDriver } from "@wdio/utils/node";
import { config as expectedConfig } from "./wdio.conf.js";

const configPath = fileURLToPath(new URL("./wdio.conf.js", import.meta.url));

async function loadConfig() {
  const parser = new ConfigParser(configPath);
  await parser.initialize({ mochaOpts: { retries: 1 } });
  return {
    options: parser.getConfig(),
    capabilities: parser.getCapabilities(),
  };
}

test("WebdriverIO loads and merges the integration configuration", async () => {
  const { options, capabilities } = await loadConfig();
  assert.equal(options.framework, expectedConfig.framework);
  assert.equal(options.hostname, expectedConfig.hostname);
  assert.equal(options.port, expectedConfig.port);
  assert.deepEqual(options.mochaOpts, { ...expectedConfig.mochaOpts, retries: 1 });
  assert.equal(capabilities[0].browserName, expectedConfig.capabilities[0].browserName);
  assert.ok(options.specs.length > 0);
});

test("the pinned browser installer exposes the API WebdriverIO calls", () => {
  for (const name of [
    "install",
    "canDownload",
    "resolveBuildId",
    "detectBrowserPlatform",
    "computeExecutablePath",
  ]) {
    assert.equal(typeof browsers[name], "function", name);
  }
  assert.equal(browsers.Browser.CHROME, "chrome");
  assert.equal(browsers.ChromeReleaseChannel.STABLE, "stable");
});

test("explicit Selenium configuration skips automatic browser and driver provisioning", async () => {
  const { options, capabilities } = await loadConfig();
  const cacheDir = await mkdtemp(path.join(tmpdir(), "kfp-wdio-remote-"));
  try {
    assert.ok(options.port);
    await setupBrowser({ ...options, cacheDir }, capabilities);
    await setupDriver({ ...options, cacheDir }, capabilities);
    assert.deepEqual(await readdir(cacheDir), []);
  } finally {
    await rm(cacheDir, { recursive: true, force: true });
  }
});
