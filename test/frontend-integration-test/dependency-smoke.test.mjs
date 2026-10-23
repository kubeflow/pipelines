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
import { mkdtemp, readFile, readdir, rm, writeFile } from "node:fs/promises";
import { EventEmitter } from "node:events";
import { MochaAdapter } from "@wdio/mocha-framework";
import { createRequire } from "node:module";
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
  assert.deepEqual(options.mochaOpts, {
    ...expectedConfig.mochaOpts,
    retries: 1,
  });
  assert.equal(
    capabilities[0].browserName,
    expectedConfig.capabilities[0].browserName,
  );
  assert.ok(options.specs.length > 0);
});

test("the pinned browser installer exposes the API WebdriverIO calls", () => {
  // WebdriverIO 10 supports the pinned 3.x browser installer directly.
  const utilsRequire = createRequire(import.meta.resolve("@wdio/utils"));
  assert.equal(
    utilsRequire.resolve("@puppeteer/browsers"),
    createRequire(import.meta.url).resolve("@puppeteer/browsers"),
  );
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

test("the integration lockfile excludes braces and vulnerable brace-expansion copies", async () => {
  const lock = JSON.parse(
    await readFile(new URL("./package-lock.json", import.meta.url), "utf8"),
  );
  for (const [name, dependency] of Object.entries(lock.packages)) {
    assert.ok(
      !name.endsWith("/braces"),
      `${name} must be removed through the runner upgrade`,
    );
    if (name.endsWith("/brace-expansion")) {
      const [major, minor, patch] = dependency.version.split(".").map(Number);
      assert.ok(
        major !== 1 || minor > 1 || (minor === 1 && patch >= 21),
        `${name}@${dependency.version}`,
      );
      assert.ok(
        major !== 2 || minor > 1 || (minor === 1 && patch >= 7),
        `${name}@${dependency.version}`,
      );
    }
  }
});

test("the WebdriverIO adapter executes Mocha tests and hooks", async () => {
  const directory = await mkdtemp(path.join(tmpdir(), "kfp-mocha-adapter-"));
  const spec = path.join(directory, "adapter.cjs");
  try {
    await writeFile(
      spec,
      `
      const assert = require('node:assert/strict');
      describe('adapter compatibility', function () {
        beforeEach(function () { this.value = 42; });
        it('runs the hook before the test', function () { assert.equal(this.value, 42); });
      });
    `,
    );
    const { options, capabilities } = await loadConfig();
    const reporter = new EventEmitter();
    const passed = [];
    reporter.on("test:pass", (event) => passed.push(event.title));
    const adapter = await new MochaAdapter(
      "0-0",
      options,
      [spec],
      capabilities[0],
      reporter,
    ).init();
    assert.equal(adapter.hasTests(), true);
    assert.equal(await adapter.run(), 0);
    assert.deepEqual(passed, ["runs the hook before the test"]);
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});

test("pipeline uploads attach browser-visible files without the removed upload API", async () => {
  const { default: vm } = await import("node:vm");
  for (const [spec, testName, filename] of [
    ["helloworld.spec.js", "uploads the sample pipeline", "helloworld.yaml"],
    [
      "literal-input.spec.js",
      "uploads the literal-input pipeline",
      "literal-input.yaml",
    ],
    [
      "tensorboard-example.spec.js",
      "deploys the tensorboard example and opens tensorboard",
      "tensorboard-example.yaml",
    ],
  ]) {
    const cases = new Map();
    const uploaded = [];
    const finished = new Error("file attached");
    const element = {
      click: async () => {},
      waitForDisplayed: async () => {},
      setFiles: async (file) => {
        uploaded.push(file);
        throw finished;
      },
    };
    const helpers = new Proxy(
      {},
      {
        get: (_, key) =>
          key === "runPhase" ? async (_, fn) => fn() : async () => {},
      },
    );
    const nativeRequire = createRequire(import.meta.url);
    const context = {
      require: (name) =>
        name === "./test-helpers" ? helpers : nativeRequire(name),
      describe: (_, fn) => fn(),
      it: (name, fn) => cases.set(name, fn),
      before: () => {},
      after: () => {},
      beforeEach: () => {},
      afterEach: () => {},
      browser: {},
      $: () => element,
      process,
      console,
      URL,
    };
    vm.runInNewContext(
      await readFile(new URL(spec, import.meta.url), "utf8"),
      context,
    );
    await assert.rejects(cases.get(testName), (error) => error === finished);
    assert.deepEqual(uploaded, [path.resolve(filename)]);
  }
});
