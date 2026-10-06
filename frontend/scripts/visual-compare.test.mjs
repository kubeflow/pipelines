// @vitest-environment jsdom

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
import { spawnSync } from 'node:child_process';
import { URL as NodeURL, fileURLToPath, pathToFileURL } from 'node:url';
import { PNG } from 'pngjs';
import path from 'node:path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  captureScreenshots,
  captureInventory,
  loadRoutes,
  parseViewports,
  parseCliOptions,
  assertNodeVersion,
} from './visual-compare.mjs';
import baselineRoutes from './visual-compare.routes.json';

let root;
let routesPath;

beforeEach(() => {
  root = fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-visual-compare-'));
  routesPath = path.join(root, 'routes.json');
  vi.spyOn(console, 'log').mockImplementation(() => {});
  vi.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  vi.restoreAllMocks();
  fs.rmSync(root, { recursive: true, force: true });
});

function saveRoutes(routes) {
  fs.writeFileSync(routesPath, JSON.stringify(routes));
}

function fakeBrowser({ events = [], closeEvents = [], missingSelectors = [] } = {}) {
  const pages = [];
  const browser = {
    close: vi.fn(),
    version: () => 'test-chromium',
    newPage: vi.fn(async () => {
      const handlers = new Map();
      const fields = {
        'Run name': `random-${pages.length}`,
        'Recurring run config name': `random-${pages.length}`,
      };
      const fitButton = { click: vi.fn() };
      const page = {
        fitButton,
        locator: vi.fn(() => fitButton),
        waitForFunction: vi.fn(),
        fields,
        on: vi.fn((name, handler) => handlers.set(name, handler)),
        getByRole: vi.fn((_role, { name }) => ({
          fill: vi.fn(async (value) => {
            fields[name] = value;
          }),
        })),
        clock: { setFixedTime: vi.fn() },
        goto: vi.fn(async () => {
          for (const { name, value } of events) {
            handlers.get(name)?.(value);
          }
        }),
        addStyleTag: vi.fn(),
        waitForSelector: vi.fn(async (selector) => {
          if (selector === '#broken' || missingSelectors.includes(selector)) {
            throw new Error('Ready selector was not found');
          }
        }),
        evaluate: vi.fn(),
        waitForTimeout: vi.fn(),
        screenshot: vi.fn(async ({ path: filePath }) =>
          fs.writeFileSync(filePath, JSON.stringify(fields)),
        ),
        close: vi.fn(async () => {
          for (const { name, value } of closeEvents) handlers.get(name)?.(value);
        }),
      };
      pages.push(page);
      return page;
    }),
  };
  return { browser, browserType: { launch: vi.fn(async () => browser) }, pages };
}

function captureOptions(browserType) {
  return {
    baseUrl: 'http://localhost:3000',
    outDir: path.join(root, 'captures'),
    routesPath,
    viewports: [{ width: 1280, height: 720 }],
    defaultWaitFor: '#root',
    defaultWaitMs: 0,
    fullPage: true,
    fixedTime: '2026-09-26T12:00:00.000Z',
    browserType,
  };
}

describe('visual capture completeness', () => {
  it('fails on an unready route, removes its old image, and records all routes', async () => {
    saveRoutes([
      { name: 'broken', path: '/broken', waitForSelector: '#broken' },
      { name: 'ready', path: '/ready', waitForSelector: '#ready' },
    ]);
    const { browserType, browser, pages } = fakeBrowser();
    const options = captureOptions(browserType);
    fs.mkdirSync(options.outDir);
    const staleImage = path.join(options.outDir, 'broken-1280x720.png');
    fs.writeFileSync(staleImage, 'stale image');

    await expect(captureScreenshots(options)).rejects.toThrow('1 screenshot capture(s) failed');

    expect(fs.existsSync(staleImage)).toBe(false);
    expect(fs.existsSync(path.join(options.outDir, 'ready-1280x720.png'))).toBe(true);
    const manifest = JSON.parse(fs.readFileSync(path.join(options.outDir, 'capture-results.json')));
    expect(manifest.results.map(({ status }) => status)).toEqual(['error', 'ok']);
    expect(manifest.results[0].error).toContain('Ready selector was not found');
    expect(pages).toHaveLength(2);
    expect(pages.every((page) => page.close.mock.calls.length === 1)).toBe(true);
    expect(browser.close).toHaveBeenCalledOnce();
  });

  it.each([true, false])(
    'records fullPage=%s and reproducible browser settings',
    async (fullPage) => {
      saveRoutes([
        { name: 'ready', path: '/ready', waitForSelector: '#ready', waitForTimeoutMs: 7 },
      ]);
      const { browserType, browser, pages } = fakeBrowser();
      const options = captureOptions(browserType);
      options.viewports.push({ width: 390, height: 844 });
      options.fullPage = fullPage;

      const results = await captureScreenshots(options);

      expect(results).toHaveLength(2);
      expect(new Set(results.map(({ filePath }) => filePath)).size).toBe(2);
      expect(browser.newPage).toHaveBeenCalledWith({
        viewport: { width: 390, height: 844 },
        locale: 'en-US',
        timezoneId: 'UTC',
        colorScheme: 'light',
        reducedMotion: 'reduce',
      });
      for (const page of pages) {
        expect(page.clock.setFixedTime).toHaveBeenCalledWith(new Date(options.fixedTime));
        expect(page.waitForSelector).toHaveBeenCalledWith('#ready', { timeout: 60000 });
        expect(page.evaluate).toHaveBeenCalledOnce();
        expect(page.waitForTimeout).toHaveBeenCalledWith(7);
        expect(page.screenshot).toHaveBeenCalledWith(
          expect.objectContaining({ animations: 'disabled', fullPage }),
        );
      }
      const manifest = JSON.parse(
        fs.readFileSync(path.join(options.outDir, 'capture-results.json')),
      );
      expect(manifest).toMatchObject({
        browser: 'test-chromium',
        locale: 'en-US',
        timezoneId: 'UTC',
        fixedTime: options.fixedTime,
        fullPage,
      });
      expect(manifest.results[0]).toMatchObject({
        route: '/ready',
        waitForSelector: '#ready',
        waitForTimeoutMs: 7,
      });
    },
  );

  it('rejects an invalid fixed time before launching a browser', async () => {
    saveRoutes([{ name: 'ready', path: '/ready' }]);
    const { browserType } = fakeBrowser();
    await expect(
      captureScreenshots({ ...captureOptions(browserType), fixedTime: 'yesterday' }),
    ).rejects.toThrow('Invalid fixed time');
    expect(browserType.launch).not.toHaveBeenCalled();
  });
});

describe('route-specific baseline assertions', () => {
  it('fits only opted-in graphs after fonts and visible node measurement', async () => {
    saveRoutes([
      { name: 'fit', path: '/graph', waitForSelector: '.react-flow__node', fitGraph: true },
      { name: 'focused', path: '/focused', waitForSelector: '.react-flow__node' },
    ]);
    const { browserType, pages } = fakeBrowser();
    const results = await captureScreenshots(captureOptions(browserType));

    expect(pages[0].locator).toHaveBeenCalledWith(
      '[data-testid="DagCanvas"] .react-flow__controls-fitview',
    );
    expect(pages[0].evaluate.mock.invocationCallOrder[0]).toBeLessThan(
      pages[0].waitForFunction.mock.invocationCallOrder[0],
    );
    expect(pages[0].waitForFunction.mock.invocationCallOrder[0]).toBeLessThan(
      pages[0].fitButton.click.mock.invocationCallOrder[0],
    );
    expect(pages[0].fitButton.click.mock.invocationCallOrder[0]).toBeLessThan(
      pages[0].screenshot.mock.invocationCallOrder[0],
    );
    expect(results[0].fitGraph).toBe(true);
    expect(pages[1].fitButton.click).not.toHaveBeenCalled();

    const isMeasured = pages[0].waitForFunction.mock.calls[0][0];
    const canvas = document.createElement('div');
    canvas.dataset.testid = 'DagCanvas';
    const node = document.createElement('div');
    node.className = 'react-flow__node';
    canvas.append(node);
    document.body.append(canvas);
    try {
      expect(isMeasured()).toBe(false);
      Object.defineProperties(node, { offsetWidth: { value: 224 }, offsetHeight: { value: 48 } });
      node.style.visibility = 'hidden';
      expect(isMeasured()).toBe(false);
      node.style.visibility = 'visible';
      expect(isMeasured()).toBe(true);
    } finally {
      canvas.remove();
    }
  });

  it('fills configured form names consistently without changing global randomness', async () => {
    const forms = baselineRoutes.filter(({ name }) =>
      ['new-run-configured', 'new-recurring-run'].includes(name),
    );
    saveRoutes(forms);
    const { browserType, pages } = fakeBrowser();
    const options = captureOptions(browserType);
    options.viewports.push({ width: 900, height: 900 });

    const results = await captureScreenshots(options);

    expect(pages).toHaveLength(4);
    for (const result of results) {
      const form = forms.find(({ name }) => name === result.name);
      expect(form.fillFields).toHaveLength(1);
      const { label, value } = form.fillFields[0];
      expect(JSON.parse(fs.readFileSync(path.join(options.outDir, result.filePath)))[label]).toBe(
        value,
      );
      expect(result.fillFields).toEqual(form.fillFields);
    }
    expect(pages[0].fields['Run name']).toBe(pages[2].fields['Run name']);
    expect(pages[1].fields['Recurring run config name']).toBe(
      pages[3].fields['Recurring run config name'],
    );
  });

  it('rejects an empty comparison message when a named fixture row did not load', async () => {
    const compare = baselineRoutes.find(({ name }) => name === 'compare');
    saveRoutes([compare]);
    const { browserType, pages } = fakeBrowser({ missingSelectors: [compare.waitForSelectors[0]] });

    await expect(captureScreenshots(captureOptions(browserType))).rejects.toThrow(
      '1 screenshot capture(s) failed',
    );

    expect(pages[0].waitForSelector).toHaveBeenCalledWith(compare.waitForSelector, {
      timeout: 60000,
    });
    expect(pages[0].screenshot).not.toHaveBeenCalled();
  });

  it.each([
    [
      'response',
      {
        url: () => 'http://localhost:3000/apis/v2beta1/runs/mock-run-0/tasks',
        status: () => 500,
      },
      'HTTP 500',
    ],
    [
      'requestfailed',
      {
        url: () => 'http://localhost:3000/apis/v2beta1/runs/mock-run-0/tasks',
        failure: () => ({ errorText: 'net::ERR_CONNECTION_RESET' }),
      },
      'net::ERR_CONNECTION_RESET',
    ],
    ['pageerror', new Error('Comparison crashed'), 'Comparison crashed'],
  ])(
    'fails comparison after a %s error even when all ready selectors appear',
    async (name, value, message) => {
      saveRoutes([baselineRoutes.find(({ name }) => name === 'compare')]);
      const { browserType, pages } = fakeBrowser({ events: [{ name, value }] });
      const options = captureOptions(browserType);

      await expect(captureScreenshots(options)).rejects.toThrow('1 screenshot capture(s) failed');

      expect(pages[0].screenshot).not.toHaveBeenCalled();
      const manifest = JSON.parse(
        fs.readFileSync(path.join(options.outDir, 'capture-results.json')),
      );
      expect(manifest.results[0].error).toContain(message);
    },
  );

  it.each(['http://localhost:3000/pipeline', 'http://localhost:3000/pipeline/'])(
    'rejects failed task requests below deployment base %s',
    async (baseUrl) => {
      saveRoutes([baselineRoutes.find(({ name }) => name === 'compare')]);
      const { browserType, pages } = fakeBrowser({
        events: [
          {
            name: 'response',
            value: {
              url: () => 'http://localhost:3000/pipeline/apis/v2beta1/runs/mock-run-0/tasks',
              status: () => 500,
            },
          },
        ],
      });
      const options = { ...captureOptions(browserType), baseUrl };

      await expect(captureScreenshots(options)).rejects.toThrow('1 screenshot capture(s) failed');

      expect(pages[0].screenshot).not.toHaveBeenCalled();
      const manifest = JSON.parse(
        fs.readFileSync(path.join(options.outDir, 'capture-results.json')),
      );
      expect(manifest.results[0].error).toContain(
        'HTTP 500: /pipeline/apis/v2beta1/runs/mock-run-0/tasks',
      );
    },
  );

  it('does not match sibling deployment paths or a different origin with the same base path', async () => {
    saveRoutes([baselineRoutes.find(({ name }) => name === 'compare')]);
    const { browserType } = fakeBrowser({
      events: [
        {
          name: 'response',
          value: {
            url: () => 'http://localhost:3000/pipeline-other/apis/v2beta1/runs/mock-run-0/tasks',
            status: () => 500,
          },
        },
        {
          name: 'response',
          value: {
            url: () => 'https://external.example/pipeline/apis/v2beta1/runs/mock-run-0/tasks',
            status: () => 500,
          },
        },
      ],
    });

    await expect(
      captureScreenshots({
        ...captureOptions(browserType),
        baseUrl: 'http://localhost:3000/pipeline',
      }),
    ).resolves.toEqual([expect.objectContaining({ status: 'ok' })]);
  });

  it('does not fail on deployment probes outside the selected request paths', async () => {
    saveRoutes([baselineRoutes.find(({ name }) => name === 'compare')]);
    const { browserType } = fakeBrowser({
      events: [
        {
          name: 'response',
          value: {
            url: () => 'http://localhost:3000/optional-deployment-probe',
            status: () => 404,
          },
        },
        {
          name: 'response',
          value: {
            url: () => 'https://external.example/apis/v2beta1/runs/mock-run-0',
            status: () => 404,
          },
        },
      ],
    });

    await expect(captureScreenshots(captureOptions(browserType))).resolves.toEqual([
      expect.objectContaining({ status: 'ok' }),
    ]);
  });

  it('keeps request and page error checks opt-in for custom capture routes', async () => {
    saveRoutes([{ name: 'expected-error', path: '/expected-error', waitForSelector: '#ready' }]);
    const { browserType } = fakeBrowser({
      events: [
        {
          name: 'response',
          value: {
            url: () => 'http://localhost:3000/apis/v2beta1/runs/missing',
            status: () => 404,
          },
        },
        { name: 'pageerror', value: new Error('Expected error fixture') },
      ],
    });

    await expect(captureScreenshots(captureOptions(browserType))).resolves.toEqual([
      expect.objectContaining({ status: 'ok' }),
    ]);
  });
});

describe('visual route validation', () => {
  it.each([[], {}, [{ path: 'runs' }], [{ path: '/' }], [null]])(
    'rejects an empty or invalid route set: %j',
    (routes) => {
      saveRoutes(routes);
      expect(() => loadRoutes(routesPath)).toThrow();
    },
  );

  it.each([
    { waitForSelectors: 'not an array' },
    { waitForSelector: 42 },
    { waitForSelector: '' },
    { waitForTimeoutMs: '1000' },
    { waitForTimeoutMs: -1 },
    { waitForTimeoutMs: null },
    { failOnRequestErrors: [false] },
    { failOnRequestErrors: ['apis/v2beta1'] },
    { failOnPageErrors: 'true' },
    { fitGraph: 'true' },
    { fillFields: [{ label: 'Run name' }] },
  ])('rejects malformed route setup or assertions: %j', (extra) => {
    saveRoutes([{ name: 'ready', path: '/ready', ...extra }]);
    expect(() => loadRoutes(routesPath)).toThrow();
  });

  it('rejects distinct route names that would overwrite the same screenshot', () => {
    saveRoutes([
      { name: 'run details', path: '/runs/1' },
      { name: 'run-details', path: '/runs/2' },
    ]);
    expect(() => loadRoutes(routesPath)).toThrow('unique filenames');
  });

  it('requires a named, ready state for every checked-in baseline route', () => {
    saveRoutes(baselineRoutes);
    const routes = loadRoutes(routesPath);
    expect(routes.length).toBeGreaterThan(0);
    expect(
      routes.every(
        ({ waitForSelector }) => typeof waitForSelector === 'string' && waitForSelector !== '#root',
      ),
    ).toBe(true);
    expect(routes.find(({ name }) => name === 'compare')?.path).toContain('?runlist=');
  });
});

describe('capture completion and CLI entrypoint', () => {
  it('ignores Chromium cancellation while still requiring ready data', async () => {
    saveRoutes([baselineRoutes.find(({ name }) => name === 'compare')]);
    const events = [
      {
        name: 'requestfailed',
        value: {
          url: () => 'http://localhost:3000/apis/v2beta1/runs/mock-run-0/tasks',
          failure: () => ({ errorText: 'net::ERR_ABORTED' }),
        },
      },
    ];
    const { browserType } = fakeBrowser({ events });
    await expect(captureScreenshots(captureOptions(browserType))).resolves.toHaveLength(1);
    const missing = fakeBrowser({
      events,
      missingSelectors: [baselineRoutes.find(({ name }) => name === 'compare').waitForSelectors[0]],
    });
    await expect(captureScreenshots(captureOptions(missing.browserType))).rejects.toThrow(
      '1 screenshot capture(s) failed',
    );
  });

  it.each([
    { name: 'pageerror', value: new Error('late failure') },
    {
      name: 'response',
      value: {
        url: () => 'http://localhost:3000/apis/v2beta1/runs/mock-run-0/tasks',
        status: () => 500,
      },
    },
    {
      name: 'requestfailed',
      value: {
        url: () => 'http://localhost:3000/apis/v2beta1/runs/mock-run-0/tasks',
        failure: () => ({ errorText: 'net::ERR_CONNECTION_RESET' }),
      },
    },
  ])('includes errors delivered during page closure: $name', async (event) => {
    saveRoutes([baselineRoutes.find(({ name }) => name === 'compare')]);
    const { browserType } = fakeBrowser({ closeEvents: [event] });
    const options = captureOptions(browserType);
    await expect(captureScreenshots(options)).rejects.toThrow('1 screenshot capture(s) failed');
    const manifest = JSON.parse(fs.readFileSync(path.join(options.outDir, 'capture-results.json')));
    expect(manifest.results[0].status).toBe('error');
    expect(fs.existsSync(path.join(root, 'captures', manifest.results[0].filePath))).toBe(false);
  });

  it('runs the CLI through a symlink, and importing it does not run the CLI', () => {
    const script = fileURLToPath(new NodeURL('./visual-compare.mjs', import.meta.url));
    const link = path.join(root, 'capture.mjs');
    fs.symlinkSync(script, link);
    for (const entry of [script, link]) {
      const result = spawnSync(process.execPath, [entry], { encoding: 'utf8' });
      expect(result.status).toBe(1);
      expect(result.stdout).toContain('Usage:');
      const invalid = spawnSync(process.execPath, [entry, '--unknown-option'], {
        encoding: 'utf8',
      });
      expect(invalid.status).toBe(1);
    }
    const imported = spawnSync(
      process.execPath,
      ['--input-type=module', '-e', `import ${JSON.stringify(pathToFileURL(script).href)}`],
      { encoding: 'utf8' },
    );
    expect(imported.status).toBe(0);
    expect(imported.stdout).toBe('');
  });
});

describe('capture failure boundaries and inventory', () => {
  it('leaves an explicit failed inventory when browser launch fails', async () => {
    saveRoutes([{ name: 'ready', path: '/ready' }]);
    const { browserType } = fakeBrowser();
    const options = captureOptions(browserType);
    fs.mkdirSync(options.outDir);
    fs.writeFileSync(path.join(options.outDir, 'ready-1280x720.png'), 'stale image');
    browserType.launch.mockRejectedValueOnce(new Error('launch failed'));
    await expect(captureScreenshots(options)).rejects.toThrow('launch failed');
    const manifest = JSON.parse(
      fs.readFileSync(path.join(options.outDir, 'capture-results.json'), 'utf8'),
    );
    expect(manifest.error).toBe('Capture did not complete');
    expect(() => captureInventory(options.outDir)).toThrow('Capture did not complete');
  });

  it('closes the browser when writing its manifest fails', async () => {
    saveRoutes([{ name: 'ready', path: '/ready' }]);
    const { browserType, browser } = fakeBrowser();
    const options = captureOptions(browserType);
    const write = fs.writeFileSync;
    vi.spyOn(fs, 'writeFileSync').mockImplementation((filename, ...args) => {
      if (
        filename === path.join(options.outDir, 'capture-results.json') &&
        browserType.launch.mock.calls.length
      )
        throw new Error('disk full');
      return write(filename, ...args);
    });
    await expect(captureScreenshots(options)).rejects.toThrow('disk full');
    expect(browser.close).toHaveBeenCalledOnce();
  });

  it.each([
    ['chromium', 'net::ERR_ABORTED'],
    ['firefox', 'NS_BINDING_ABORTED'],
    ['webkit', 'Load request cancelled'],
  ])('allows %s cancellation but not genuine failures', async (engine, errorText) => {
    saveRoutes([baselineRoutes.find(({ name }) => name === 'compare')]);
    const event = {
      name: 'requestfailed',
      value: {
        url: () => 'http://localhost:3000/apis/v2beta1/runs/mock-run-0/tasks',
        failure: () => ({ errorText }),
      },
    };
    const { browserType } = fakeBrowser({ events: [event] });
    browserType.name = () => engine;
    await expect(captureScreenshots(captureOptions(browserType))).resolves.toHaveLength(1);
    event.value.failure = () => ({ errorText: 'connection refused' });
    await expect(captureScreenshots(captureOptions(browserType))).rejects.toThrow(
      '1 screenshot capture(s) failed',
    );
  });

  it.each(['2026-09-26T12:00:00Z', '2026-09-26T12:00:00.123Z'])(
    'accepts the documented UTC clock form: %s',
    async (fixedTime) => {
      saveRoutes([{ name: 'ready', path: '/ready' }]);
      const { browserType } = fakeBrowser();
      await expect(
        captureScreenshots({ ...captureOptions(browserType), fixedTime }),
      ).resolves.toHaveLength(1);
    },
  );

  it.each(['2026-02-30T12:00:00Z', '2026-13-01T12:00:00Z', '2026-09-26T12:00:00+02:00'])(
    'rejects invalid calendar dates or unsupported clock syntax before launch: %s',
    async (fixedTime) => {
      saveRoutes([{ name: 'ready', path: '/ready' }]);
      const { browserType } = fakeBrowser();
      await expect(
        captureScreenshots({ ...captureOptions(browserType), fixedTime }),
      ).rejects.toThrow('Invalid fixed time');
      expect(browserType.launch).not.toHaveBeenCalled();
    },
  );

  it('imports the capture module even when argv[1] names a nonexistent entrypoint', () => {
    const script = pathToFileURL(
      fileURLToPath(new NodeURL('./visual-compare.mjs', import.meta.url)),
    ).href;
    const result = spawnSync(
      process.execPath,
      [
        '--input-type=module',
        '-e',
        `process.argv[1] = ${JSON.stringify(path.join(root, 'missing.mjs'))}; await import(${JSON.stringify(script)});`,
      ],
      { encoding: 'utf8' },
    );
    expect(result.status, result.stderr).toBe(0);
    expect(result.stdout).toBe('');
  });

  it('diffs only manifest captures and rejects failed captures even if a stale PNG exists', () => {
    const baseline = path.join(root, 'baseline');
    const current = path.join(root, 'current');
    const png = PNG.sync.write(new PNG({ width: 1, height: 1 }));
    for (const directory of [baseline, current]) {
      fs.mkdirSync(directory);
      fs.writeFileSync(path.join(directory, 'kept.png'), png);
      fs.writeFileSync(path.join(directory, 'removed.png'), 'stale invalid PNG');
      fs.writeFileSync(
        path.join(directory, 'capture-results.json'),
        JSON.stringify({ results: [{ filePath: 'kept.png', status: 'ok' }] }),
      );
    }
    const report = path.join(root, 'report.html');
    const args = [
      fileURLToPath(new NodeURL('./visual-compare.mjs', import.meta.url)),
      'diff',
      '--baseline-dir',
      baseline,
      '--current-dir',
      current,
      '--diff-dir',
      path.join(root, 'diff'),
      '--side-by-side-dir',
      path.join(root, 'side'),
      '--report',
      report,
      '--fail-on-diff',
    ];
    let result = spawnSync(process.execPath, args, { encoding: 'utf8' });
    expect(result.status, result.stderr).toBe(0);
    expect(fs.readFileSync(report, 'utf8')).not.toContain('removed.png');
    fs.writeFileSync(
      path.join(current, 'capture-results.json'),
      JSON.stringify({ results: [{ filePath: 'kept.png', status: 'error' }] }),
    );
    result = spawnSync(process.execPath, args, { encoding: 'utf8' });
    expect(result.status).toBe(1);
    expect(fs.readFileSync(report, 'utf8')).toContain('Missing or failed current capture');
  });
});

describe('capture input failures', () => {
  it.each(['routes', 'clock', 'viewports'])(
    'invalidates previous success before invalid %s input',
    async (input) => {
      saveRoutes([{ name: 'ready', path: '/ready' }]);
      const { browserType } = fakeBrowser();
      const options = captureOptions(browserType);
      fs.mkdirSync(options.outDir);
      fs.writeFileSync(
        path.join(options.outDir, 'capture-results.json'),
        JSON.stringify({ results: [{ filePath: 'ready.png', status: 'ok' }] }),
      );
      if (input === 'routes') saveRoutes([]);
      if (input === 'clock') options.fixedTime = 'invalid';
      if (input === 'viewports') options.viewports = '1280x720,1280x720';
      await expect(captureScreenshots(options)).rejects.toThrow();
      expect(browserType.launch).not.toHaveBeenCalled();
      expect(() => captureInventory(options.outDir)).toThrow('Capture did not complete');
    },
  );

  it.each(['1280x720,1280x720', '1280x720,01280x720'])(
    'rejects duplicate viewport filenames: %s',
    (value) => {
      expect(() => parseViewports(value)).toThrow('Duplicate viewport');
    },
  );

  it.each([
    [[{ filePath: 'image.jpg', status: 'ok' }], 'entry 1: filePath must name a PNG'],
    [[{ filePath: 'image.png', status: 'pending' }], 'entry 1: status must be ok or error'],
    [
      [
        { filePath: 'image.png', status: 'ok' },
        { filePath: 'image.png', status: 'ok' },
      ],
      'entry 2: duplicate filename',
    ],
  ])('identifies malformed inventory entries', (results, reason) => {
    fs.writeFileSync(path.join(root, 'capture-results.json'), JSON.stringify({ results }));
    expect(() => captureInventory(root)).toThrow(reason);
  });

  it('writes an error report when both capture directories are missing', () => {
    const report = path.join(root, 'report.html');
    const result = spawnSync(
      process.execPath,
      [
        fileURLToPath(new NodeURL('./visual-compare.mjs', import.meta.url)),
        'diff',
        '--baseline-dir',
        path.join(root, 'missing-baseline'),
        '--current-dir',
        path.join(root, 'missing-current'),
        '--diff-dir',
        path.join(root, 'diff'),
        '--side-by-side-dir',
        path.join(root, 'side'),
        '--report',
        report,
        '--fail-on-diff',
      ],
      { encoding: 'utf8' },
    );
    expect(result.status, result.stderr).toBe(1);
    const html = fs.readFileSync(report, 'utf8');
    expect(html).toContain('Cannot read baseline captures');
    expect(html).toContain('Cannot read current captures');
  });
});

it('terminates the CLI if browser cleanup rejects with live handles', () => {
  saveRoutes([{ name: 'ready', path: '/ready' }]);
  const script = fileURLToPath(new NodeURL('./visual-compare.mjs', import.meta.url));
  const code = `
    import { createRequire } from 'node:module';
    const { chromium } = createRequire(${JSON.stringify(script)})('playwright');
    chromium.launch = async () => {
      setInterval(() => {}, 1000);
      return {
        version: () => 'fake',
        newPage: async () => { throw new Error('fake page failure'); },
        close: async () => { throw new Error('cleanup failed'); },
      };
    };
  `;
  const bootstrap = path.join(root, 'cleanup-failure.mjs');
  fs.writeFileSync(bootstrap, code);
  const result = spawnSync(
    process.execPath,
    [
      '--import',
      bootstrap,
      script,
      'capture',
      '--routes',
      routesPath,
      '--out-dir',
      path.join(root, 'capture'),
    ],
    { encoding: 'utf8', timeout: 10000 },
  );
  expect(result.error).toBeUndefined();
  expect(result.status).toBe(1);
  expect(result.stderr).toContain('cleanup failed');
  expect(() => captureInventory(path.join(root, 'capture'))).toThrow('cleanup failed');
}, 15000);

describe('capture option validation', () => {
  it.each([
    'not-a-url',
    'file:///tmp/ui',
    'http://localhost:3000/#/runs',
    'http://localhost:3000/?query=x',
  ])('rejects invalid base URL before browser launch: %s', async (baseUrl) => {
    saveRoutes([{ name: 'ready', path: '/ready' }]);
    const { browserType } = fakeBrowser();
    const options = { ...captureOptions(browserType), baseUrl };
    await expect(captureScreenshots(options)).rejects.toThrow('base-url');
    expect(browserType.launch).not.toHaveBeenCalled();
    expect(() => captureInventory(options.outDir)).toThrow('Capture did not complete');
  });
  it.each([NaN, Infinity, -1, '100'])(
    'rejects invalid settle wait before browser launch: %s',
    async (defaultWaitMs) => {
      saveRoutes([{ name: 'ready', path: '/ready' }]);
      const { browserType } = fakeBrowser();
      await expect(
        captureScreenshots({ ...captureOptions(browserType), defaultWaitMs }),
      ).rejects.toThrow('wait-ms');
      expect(browserType.launch).not.toHaveBeenCalled();
    },
  );
  it('fills only the exact textbox label', async () => {
    saveRoutes([
      { name: 'form', path: '/form', fillFields: [{ label: 'Run name', value: 'Fixed name' }] },
    ]);
    const { browserType, pages } = fakeBrowser();
    await captureScreenshots(captureOptions(browserType));
    expect(pages[0].getByRole).toHaveBeenCalledWith('textbox', { name: 'Run name', exact: true });
  });
});

it('rejects unknown route keys rather than silently dropping a misspelled setting', () => {
  saveRoutes([{ path: '/runs', waitForTimoutMs: 1 }]);
  expect(() => loadRoutes(routesPath)).toThrow('Unknown route keys: waitForTimoutMs');
});

it('retains the original capture failure when cleanup and manifest writing also fail', async () => {
  saveRoutes([{ name: 'broken', path: '/broken', waitForSelector: '#broken' }]);
  const { browserType, browser } = fakeBrowser();
  browser.close.mockRejectedValue(new Error('cleanup failed'));
  const options = captureOptions(browserType);
  const write = fs.writeFileSync;
  vi.spyOn(fs, 'writeFileSync').mockImplementation((filename, ...args) => {
    if (
      filename === path.join(options.outDir, 'capture-results.json') &&
      browserType.launch.mock.calls.length
    )
      throw new Error('manifest write failed');
    return write(filename, ...args);
  });
  const failure = await captureScreenshots(options).catch((error) => error);
  expect(failure).toBeInstanceOf(AggregateError);
  expect(failure.errors[0].message).toContain('1 screenshot capture(s) failed');
  expect(failure.errors[0].cause.errors[0].message).toContain('Ready selector was not found');
  expect(failure.message).toContain('cleanup failed');
  expect(failure.message).toContain('manifest write failed');
});

it.each(['frontend', 'repository'])(
  'uses npm package cwd regardless of the invoking %s shell',
  (location) => {
    const frontend = fileURLToPath(new NodeURL('../', import.meta.url)).replace(/\/$/, '');
    const caller = location === 'frontend' ? frontend : path.dirname(frontend);
    vi.stubEnv('INIT_CWD', caller);
    vi.stubEnv('npm_lifecycle_event', 'visual:current');
    vi.spyOn(process, 'cwd').mockReturnValue(frontend); // npm --prefix executes scripts in the package.
    try {
      const scripts = JSON.parse(
        fs.readFileSync(path.join(frontend, 'package.json'), 'utf8'),
      ).scripts;
      for (const [script, expected] of [
        ['visual:baseline', 'baseline'],
        ['visual:current', 'current'],
      ]) {
        expect(parseCliOptions(scripts[script].split(' ').slice(2)).values['out-dir']).toBe(
          path.join(frontend, '.visual', expected),
        );
      }
      expect(parseCliOptions(scripts['visual:diff'].split(' ').slice(2)).values.report).toBe(
        path.join(frontend, '.visual/report.html'),
      );
      expect(parseCliOptions(['capture', '--out-dir', 'custom']).values['out-dir']).toBe(
        path.join(frontend, 'custom'),
      );
    } finally {
      vi.unstubAllEnvs();
    }
  },
);

it.each(['response', 'requestfailed'])(
  'records malformed URLs in %s listeners instead of throwing out of the handler',
  async (name) => {
    saveRoutes([{ name: 'runs', path: '/runs', failOnRequestErrors: ['/apis/'] }]);
    const { browserType } = fakeBrowser({
      events: [
        {
          name,
          value: {
            url: () => 'not a URL',
            status: () => 500,
            failure: () => ({ errorText: 'failed' }),
          },
        },
      ],
    });
    const options = captureOptions(browserType);
    await expect(captureScreenshots(options)).rejects.toThrow('1 screenshot capture(s) failed');
    const manifest = JSON.parse(
      fs.readFileSync(path.join(options.outDir, 'capture-results.json'), 'utf8'),
    );
    expect(manifest.results[0].error).toContain('Invalid request URL: not a URL');
  },
);

it('validates capture viewports', () => {
  expect(parseViewports(' 1280x720, 900x900 ')).toEqual([
    { width: 1280, height: 720 },
    { width: 900, height: 900 },
  ]);
  for (const value of ['0x720', '1.5x720', '1280x720x3'])
    expect(() => parseViewports(value)).toThrow();
});

it('records the actual browser close error after successful screenshots', async () => {
  saveRoutes([{ name: 'ready', path: '/ready' }]);
  const { browserType, browser } = fakeBrowser();
  browser.close.mockRejectedValue(new Error('browser transport disconnected'));
  const options = captureOptions(browserType);
  await expect(captureScreenshots(options)).rejects.toThrow('browser transport disconnected');
  const manifest = JSON.parse(
    fs.readFileSync(path.join(options.outDir, 'capture-results.json'), 'utf8'),
  );
  expect(manifest.results[0].status).toBe('ok');
  expect(manifest.error).toBe('browser transport disconnected');
});

it('uses process.cwd for explicit paths regardless of npm environment', () => {
  vi.stubEnv('INIT_CWD', '/unrelated/outer/npm');
  vi.stubEnv('npm_lifecycle_event', 'outer-task');
  try {
    expect(parseCliOptions(['capture', '--out-dir', 'custom']).values['out-dir']).toBe(
      path.resolve('custom'),
    );
  } finally {
    vi.unstubAllEnvs();
  }
});

it('writes one initial failure marker and one final successful manifest', async () => {
  saveRoutes([{ name: 'ready', path: '/ready' }]);
  const { browserType } = fakeBrowser();
  const options = captureOptions(browserType);
  const write = vi.spyOn(fs, 'writeFileSync');
  await captureScreenshots(options);
  const manifests = write.mock.calls
    .filter(([filename]) => filename === path.join(options.outDir, 'capture-results.json'))
    .map(([, text]) => JSON.parse(text));
  expect(manifests).toHaveLength(2);
  expect(manifests[0].error).toBe('Capture did not complete');
  expect(manifests[1].error).toBeUndefined();
  expect(manifests[1].results[0].status).toBe('ok');
});

it.each(['22.18.0', '24.1.0', 'invalid'])(
  'rejects unsupported Node %s before CLI entrypoint selection',
  (version) => {
    expect(() => assertNodeVersion(version)).toThrow('requires Node >=24.2.0');
  },
);
it.each(['24.2.0', '24.14.0', '25.0.0'])('accepts supported Node %s', (version) => {
  expect(() => assertNodeVersion(version)).not.toThrow();
});
it('requires a manifest even when a directory contains PNG files', () => {
  fs.writeFileSync(path.join(root, 'stray.png'), 'not a capture');
  expect(() => captureInventory(root)).toThrow('Missing capture manifest');
});
