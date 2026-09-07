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

import assert from "node:assert/strict";
import { once } from "node:events";
import { mkdtemp, readFile, readdir, rm } from "node:fs/promises";
import { createServer } from "node:http";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import { ConfigParser } from "@wdio/config/node";
import { setupBrowser, setupDriver } from "@wdio/utils/node";

// Resolve the overridden package through its actual consumer, even when nested.
const utilsRequire = createRequire(import.meta.resolve("@wdio/utils"));
const browsers = await import(
  pathToFileURL(utilsRequire.resolve("@puppeteer/browsers")).href
);
const webdriverRequire = createRequire(import.meta.resolve("webdriverio"));
const JSZip = webdriverRequire("jszip");
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
  assert.equal(options.framework, "mocha");
  assert.equal(options.mochaOpts.ui, "bdd");
  assert.equal(options.mochaOpts.timeout, 1200000);
  assert.equal(options.mochaOpts.retries, 1);
  assert.equal(capabilities[0].browserName, "chrome");
  assert.ok(options.specs.length > 0);
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

test(
  "overridden browser installer extracts an ordinary local ZIP archive",
  { timeout: 30000 },
  async () => {
    const cacheDir = await mkdtemp(path.join(tmpdir(), "kfp-browser-install-"));
    const contents = "ordinary browser installation fixture\n";
    const zip = new JSZip();
    zip.file("chrome-linux64/chrome", contents);
    const archive = await zip.generateAsync({ type: "nodebuffer" });
    const requests = [];
    const server = createServer((request, response) => {
      requests.push(request.url);
      response.writeHead(200, {
        "Content-Type": "application/zip",
        "Content-Length": archive.length,
      });
      response.end(archive);
    });
    try {
      server.listen(0, "127.0.0.1");
      await once(server, "listening");
      const options = {
        browser: browsers.Browser.CHROME,
        platform: browsers.BrowserPlatform.LINUX,
        buildId: "123.0.0.0",
        cacheDir,
        baseUrl: `http://127.0.0.1:${server.address().port}`,
      };
      const installed = await browsers.install(options);
      assert.deepEqual(requests, ["/123.0.0.0/linux64/chrome-linux64.zip"]);
      assert.equal(
        installed.executablePath,
        browsers.computeExecutablePath(options),
      );
      assert.equal(await readFile(installed.executablePath, "utf8"), contents);
    } finally {
      server.closeAllConnections();
      await new Promise((resolve) => server.close(resolve));
      await rm(cacheDir, { recursive: true, force: true });
    }
  },
);
