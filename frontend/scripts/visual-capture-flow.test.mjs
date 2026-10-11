// @vitest-environment node

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

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { PNG } from 'pngjs';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { captureScreenshots, runDiff } from './visual-compare.mjs';
let root;

beforeEach(() => {
  root = fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-visual-flow-'));
  vi.spyOn(console, 'log').mockImplementation(() => {});
  vi.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  vi.restoreAllMocks();
  fs.rmSync(root, { recursive: true, force: true });
});

function writePng(filename, width = 8, color = 0) {
  const png = new PNG({ width, height: 8 });
  for (let index = 0; index < png.data.length; index += 4) {
    png.data[index] = color;
    png.data[index + 1] = color;
    png.data[index + 2] = color;
    png.data[index + 3] = 255;
  }
  fs.writeFileSync(filename, PNG.sync.write(png));
}

async function capture() {
  const routesPath = path.join(root, 'external-routes.json');
  const routesBytes = Buffer.from('[\n  {"name":"Runs", "path":"/runs"}\n]\n');
  fs.writeFileSync(routesPath, routesBytes);
  const rawDir = path.join(root, 'raw');
  // Exercise the real capture writer; replace only the browser transport.
  await captureScreenshots({
    baseUrl: 'http://localhost:3000',
    outDir: rawDir,
    routesPath,
    viewports: '1280x720',
    defaultWaitFor: '#root',
    defaultWaitMs: 0,
    fullPage: true,
    browserType: {
      name: () => 'chromium',
      launch: async () => ({
        version: () => 'fixture-browser',
        close: async () => {},
        newPage: async () => ({
          on: () => {},
          goto: async () => {},
          addStyleTag: async () => {},
          waitForSelector: async () => {},
          evaluate: async () => {},
          screenshot: async ({ path: filename }) => writePng(filename),
          close: async () => {},
        }),
      }),
    },
  });
  return { rawDir };
}

function diff(baselineDir, currentDir, extra = []) {
  const report = path.join(root, 'report.html');
  const start = console.log.mock.calls.length;
  const status = runDiff({
    baselineDir,
    currentDir,
    diffDir: path.join(root, 'diff'),
    sideBySideDir: path.join(root, 'side-by-side'),
    reportPath: report,
    failOnDiff: extra.includes('--fail-on-diff'),
    includeDiff: extra.includes('--include-diff'),
  });
  return {
    status,
    stderr: '',
    stdout: console.log.mock.calls
      .slice(start)
      .map((args) => args.join(' '))
      .join('\n'),
    report: fs.readFileSync(report, 'utf8'),
  };
}

async function comparisonFixture() {
  const { rawDir } = await capture();
  const baselineDir = path.join(root, 'baseline');
  fs.cpSync(rawDir, baselineDir, { recursive: true });
  const manifest = JSON.parse(
    fs.readFileSync(path.join(baselineDir, 'capture-results.json'), 'utf8'),
  );
  return {
    rawDir,
    baselineDir,
    screenshot: path.resolve(baselineDir, manifest.results[0].filePath),
  };
}

it('captures, relocates the capture directory, and compares manifest-relative screenshots', async () => {
  const { rawDir, baselineDir } = await comparisonFixture();
  const relocated = path.join(root, 'relocated-current');
  fs.renameSync(rawDir, relocated);
  const manifest = JSON.parse(
    fs.readFileSync(path.join(relocated, 'capture-results.json'), 'utf8'),
  );
  expect(path.isAbsolute(manifest.results[0].filePath)).toBe(false);
  expect(fs.existsSync(path.resolve(relocated, manifest.results[0].filePath))).toBe(true);
  const result = diff(baselineDir, relocated);
  expect(result.status).toBe(0);
  expect(result.stdout).toContain('0 pixels differ');
  expect(result.report).toContain('class="match"');
  expect(result.report).toContain('relocated-current/');
  expect(result.report).not.toContain('class="error"');
});

it('writes a report and exits nonzero for a missing screenshot without --fail-on-diff', async () => {
  const { rawDir, baselineDir, screenshot } = await comparisonFixture();
  fs.rmSync(screenshot);
  const result = diff(baselineDir, rawDir);
  expect(result.status).toBe(1);
  expect(result.report).toContain('class="error"');
  expect(result.report).toContain('ENOENT');
});

it('writes a report and exits nonzero for incompatible image sizes without --fail-on-diff', async () => {
  const { rawDir, baselineDir, screenshot } = await comparisonFixture();
  writePng(screenshot, 9);
  const result = diff(baselineDir, rawDir);
  expect(result.status).toBe(1);
  expect(result.report).toContain('class="error"');
  expect(result.report).toMatch(/size mismatch/i);
});

it('reports pixel changes successfully unless --fail-on-diff is requested', async () => {
  const { rawDir, baselineDir, screenshot } = await comparisonFixture();
  writePng(screenshot, 8, 255);
  const result = diff(baselineDir, rawDir);
  expect(result.status, result.stderr).toBe(0);
  expect(result.stdout).toContain('64 pixels differ');
  expect(result.report).toContain('class="diff"');
  expect(diff(baselineDir, rawDir, ['--fail-on-diff']).status).toBe(1);
});

it('escapes successful report rows, screenshot attributes, and directory headers', () => {
  const directoryName = 'baseline "quoted" & <tag>';
  const filename = 'run "quoted" & <tag>.png';
  const baselineDir = path.join(root, directoryName);
  const currentDir = path.join(root, 'current');
  fs.mkdirSync(baselineDir);
  fs.mkdirSync(currentDir);
  writePng(path.join(baselineDir, filename));
  writePng(path.join(currentDir, filename));
  for (const directory of [baselineDir, currentDir]) {
    fs.writeFileSync(
      path.join(directory, 'capture-results.json'),
      JSON.stringify({ results: [{ filePath: filename, status: 'ok' }] }),
    );
  }

  const result = diff(baselineDir, currentDir);
  expect(result.status, result.stderr).toBe(0);
  const escapedDirectory = 'baseline &quot;quoted&quot; &amp; &lt;tag&gt;';
  const escapedFilename = 'run &quot;quoted&quot; &amp; &lt;tag&gt;.png';
  expect(result.report).toContain(`<p>Baseline: ${root}/${escapedDirectory}</p>`);
  expect(result.report).toContain(`<td>${escapedFilename}</td>`);
  expect(result.report).toContain(`src="${escapedDirectory}/${escapedFilename}"`);
  expect(result.report).toContain(`alt="baseline ${escapedFilename}"`);
  expect(result.report).toContain(`alt="current ${escapedFilename}"`);
  expect(result.report).toContain(`alt="diff ${escapedFilename}"`);
  expect(result.report).not.toContain('<tag>');
});

it('reports empty capture directories as errors instead of a successful comparison', () => {
  const baselineDir = path.join(root, 'empty-baseline');
  const currentDir = path.join(root, 'empty-current');
  fs.mkdirSync(baselineDir);
  fs.mkdirSync(currentDir);

  const result = diff(baselineDir, currentDir);
  expect(result.status).toBe(1);
  expect(result.report).toContain('class="error"');
  expect(result.report).toContain('Cannot read baseline captures');
  expect(result.report).toContain('Cannot read current captures');
  expect(result.report).not.toContain('class="match"');
});

it('reports a global capture failure while retaining successful route comparisons', async () => {
  const { rawDir, baselineDir } = await comparisonFixture();
  const manifest = path.join(rawDir, 'capture-results.json');
  const raw = JSON.parse(fs.readFileSync(manifest, 'utf8'));
  raw.error = 'Browser close failed after screenshots completed';
  fs.writeFileSync(manifest, JSON.stringify(raw));

  const result = diff(rawDir, baselineDir);
  expect(result.status).toBe(1);
  expect(result.report).toContain('class="error"');
  expect(result.report).toContain(raw.error);
  expect(result.report).toContain('class="match"');
  expect(result.stdout).toContain(`${path.basename(raw.results[0].filePath)}: 0 pixels differ`);
});

it('reports successful additions informationally while missing or failed captures fail', async () => {
  const { rawDir, baselineDir } = await comparisonFixture();
  const manifestPath = path.join(rawDir, 'capture-results.json');
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  writePng(path.join(rawDir, 'new-route.png'));
  manifest.results.push({ filePath: 'new-route.png', status: 'ok' });
  fs.writeFileSync(manifestPath, JSON.stringify(manifest));
  const normal = diff(baselineDir, rawDir, ['--fail-on-diff']);
  expect(normal.status, normal.stderr).toBe(0);
  expect(normal.report).toContain('class="added"');
  expect(normal.report).toContain('class="match"');
  // Missing expected baseline coverage remains an error.
  expect(diff(rawDir, baselineDir).status).toBe(1);
  manifest.results.at(-1).status = 'error';
  fs.writeFileSync(manifestPath, JSON.stringify(manifest));
  const failed = diff(baselineDir, rawDir);
  expect(failed.status).toBe(1);
  expect(failed.report).toContain('Missing or failed current capture for new-route.png');
});

it('decodes each source image only once when also producing side-by-side output', async () => {
  const { rawDir, baselineDir } = await comparisonFixture();
  const read = vi.spyOn(PNG.sync, 'read');
  expect(diff(baselineDir, rawDir, ['--include-diff']).status).toBe(0);
  expect(read).toHaveBeenCalledTimes(2);
});

it.each([
  ['error', 'ok'],
  ['ok', 'error'],
  ['error', 'error'],
])(
  'reports each failed side independently: baseline=%s current=%s',
  async (baselineStatus, currentStatus) => {
    const { rawDir, baselineDir } = await comparisonFixture();
    for (const [directory, status] of [
      [baselineDir, baselineStatus],
      [rawDir, currentStatus],
    ]) {
      const filename = path.join(directory, 'capture-results.json');
      const manifest = JSON.parse(fs.readFileSync(filename, 'utf8'));
      manifest.results[0].status = status;
      fs.writeFileSync(filename, JSON.stringify(manifest));
    }
    const result = diff(baselineDir, rawDir);
    expect(result.status).toBe(1);
    expect(result.report.includes('Missing or failed baseline capture')).toBe(
      baselineStatus === 'error',
    );
    expect(result.report.includes('Missing or failed current capture')).toBe(
      currentStatus === 'error',
    );
  },
);
