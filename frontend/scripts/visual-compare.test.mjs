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
import path from 'node:path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { captureScreenshots, loadRoutes } from './visual-compare.mjs';
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

function fakeBrowser({ events = [], missingSelectors = [] } = {}) {
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
        close: vi.fn(),
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
      expect(JSON.parse(fs.readFileSync(result.filePath))[label]).toBe(value);
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
    { failOnRequestErrors: [false] },
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
    expect(routes.length).toBeGreaterThan(11);
    expect(
      routes.every(
        ({ waitForSelector }) => typeof waitForSelector === 'string' && waitForSelector !== '#root',
      ),
    ).toBe(true);
    expect(routes.find(({ name }) => name === 'compare')?.path).toContain('?runlist=');
    expect(routes.some(({ path }) => path === '/shared/pipelines')).toBe(false);
    expect(
      routes
        .filter(({ fitGraph }) => fitGraph)
        .map(({ name }) => name)
        .sort(),
    ).toEqual(['pipeline-details', 'pipeline-loops', 'run-details']);
  });
});
