const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const sharp = require('sharp');

const capture = require('../capture-screenshots.js');
const comparison = require('../generate-comparison.js');
const { strictSemanticFixtureManifest } = require('./semantic-fixture.js');

function fixtureDirectory(t) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-hardening-'));
  t.after(() => fs.rmSync(directory, { force: true, recursive: true }));
  return directory;
}

async function writePng(filePath, width = 10, height = 10, color = '#336699') {
  await sharp({
    create: {
      width,
      height,
      channels: 4,
      background: color,
    },
  })
    .png()
    .toFile(filePath);
}

function captureResult(filename, overrides = {}) {
  const match = /^(.*)-([1-9]\d*)x([1-9]\d*)\.png$/.exec(filename);
  assert.ok(match, `invalid fixture filename: ${filename}`);
  return {
    filename,
    page: match[1],
    required: true,
    status: 'success',
    viewport: { width: Number(match[2]), height: Number(match[3]) },
    ...overrides,
  };
}

function sha256File(filePath) {
  return crypto.createHash('sha256').update(fs.readFileSync(filePath)).digest('hex');
}

function writeCaptureManifest(directory, results, label, overrides = {}) {
  const now = Date.now();
  const startedAt = new Date(now - 2000).toISOString();
  const completedAt = new Date(now + 2000).toISOString();
  const normalizedResults = results.map((result) => {
    if (result.status !== 'success' && result.status !== 'degraded') {
      return result;
    }
    return {
      ...result,
      capturedAt: result.capturedAt || new Date(now).toISOString(),
      semanticIdNormalization: result.semanticIdNormalization || {
        complete: true,
        derivedColorScopes: [],
        schemaVersion: 'ui-smoke-id-normalization/v1',
        scopes: [],
        totalReplacementCount: 0,
      },
      sha256: result.sha256 || sha256File(path.join(directory, result.filename)),
    };
  });
  const requiredIncomplete = normalizedResults.some(
    (result) => result.required && result.status !== 'success',
  );
  const hasCapturedScreenshot = normalizedResults.some(
    (result) => result.status === 'success' || result.status === 'degraded',
  );
  const complete = !requiredIncomplete && hasCapturedScreenshot;
  const directoryName = path.basename(directory);
  const revisionRole =
    overrides.revisionRole ||
    (directoryName === 'main' || directoryName === 'base' ? 'base' : 'head');
  const inputAttestation = (name, value) => {
    const inputPath = path.join(directory, `${name}.json`);
    const contents = `${JSON.stringify(value)}\n`;
    fs.writeFileSync(inputPath, contents);
    return {
      path: inputPath,
      schemaVersion: value.schemaVersion ?? null,
      sha256: crypto.createHash('sha256').update(contents).digest('hex'),
      sizeBytes: Buffer.byteLength(contents),
    };
  };
  fs.writeFileSync(
    path.join(directory, 'manifest.json'),
    `${JSON.stringify(
      {
        schemaVersion: 3,
        captureId: overrides.captureId || `${label}-capture`,
        label,
        browser: overrides.browser || {
          engine: 'chromium',
          playwrightVersion: '1.55.0',
          version: 'test-chromium',
        },
        deterministicRendering: overrides.deterministicRendering || {
          colorScheme: 'light',
          locale: 'en-US',
          semanticIdNormalization: {
            derivedColorPalette: ['#4285f4', '#2b9c1e', '#e00000', '#8026c0', '#9dafff', '#82c57a'],
            failOnReplacementCountMismatch: true,
            mode: 'disabled-browser-compatibility',
            rawIdentifierPolicy: 'SHA-256 attestation only',
            schemaVersion: 'ui-smoke-id-normalization/v1',
            tokenFormat: '[ui-id:<kind>:<semantic-path>]',
          },
          timezone: 'UTC',
        },
        startedAt,
        completedAt,
        fatalErrors: [],
        inputs: overrides.inputs || {
          revisionRole,
          seedManifest: inputAttestation('seed', {
            revisionRole,
            schemaVersion: 'ui-smoke-seed/v1',
          }),
          semanticManifest: null,
          sourceProvenance: null,
        },
        scenarioContractSchemaVersion: overrides.scenarioContractSchemaVersion ?? false,
        viewports: [
          ...new Map(
            normalizedResults.map((result) => [
              `${result.viewport.width}x${result.viewport.height}`,
              result.viewport,
            ]),
          ).values(),
        ],
        results: normalizedResults,
        complete,
        summary: { complete },
      },
      null,
      2,
    )}\n`,
  );
}

function comparisonOptions(mainDir, prDir, outputDir, overrides = {}) {
  return {
    diffThreshold: 0,
    failThreshold: null,
    failThresholdRaw: '',
    looksSameClusterSize: 8,
    looksSameTolerance: 2.3,
    mainDir,
    outputDir,
    prDir,
    ...overrides,
  };
}

const equalAnalysis = async () => ({
  equal: true,
  differentPixels: 0,
  totalPixels: 100,
  diffBounds: null,
  diffClusters: [],
});

test('parseViewports accepts only positive integer WIDTHxHEIGHT values', () => {
  assert.deepEqual(capture.parseViewports('1280x800, 390x844'), [
    { width: 1280, height: 800 },
    { width: 390, height: 844 },
  ]);

  for (const invalid of [
    '',
    '0x800',
    '-1x800',
    '1.5x800',
    '1280x0',
    '1e3x800',
    '1X1',
    '1280x800,1280x800',
  ]) {
    assert.throws(() => capture.parseViewports(invalid), /viewport|required/i);
  }
});

test('base URL handling preserves protocol, hostname, port, and path', () => {
  const options = capture.parseCaptureOptions(
    [
      '--base-url',
      'https://example.test:8443/kfp/?tenant=example',
      '--output',
      '/tmp/output',
      '--normalization-mode',
      'disabled-browser-compatibility',
    ],
    {},
  );
  assert.equal(options.baseUrl, 'https://example.test:8443/kfp?tenant=example');
  assert.equal(
    capture.resolveCaptureUrl(options.baseUrl, '/#/pipelines'),
    'https://example.test:8443/kfp/?tenant=example#/pipelines',
  );

  const portFallback = capture.parseCaptureOptions(
    ['--port', '4567', '--normalization-mode', 'disabled-browser-compatibility'],
    {},
  );
  assert.equal(portFallback.baseUrl, 'http://localhost:4567');

  const provenance = capture.parseCaptureOptions(
    [
      '--revision-role',
      'head',
      '--normalization-mode',
      'semantic-full-stack',
      '--semantic-manifest',
      '/tmp/semantic.json',
      '--source-provenance',
      '/tmp/source.json',
    ],
    {},
  );
  assert.equal(provenance.revisionRole, 'head');
  assert.equal(provenance.semanticManifestPath, '/tmp/semantic.json');
  assert.equal(provenance.sourceProvenancePath, '/tmp/source.json');
});

test('direct-tool parsers reject unknown, duplicate, and ambiguous arguments', () => {
  assert.throws(() => capture.parseCaptureOptions(['--unknown', 'value'], {}), /Unknown/);
  assert.throws(
    () => capture.parseCaptureOptions(['--port', '4001', '--port', '4002'], {}),
    /Duplicate/,
  );
  assert.throws(
    () =>
      capture.parseCaptureOptions(['--base-url', 'http://localhost:4001', '--port', '4001'], {}),
    /mutually exclusive/,
  );
  assert.throws(
    () =>
      capture.parseCaptureOptions(
        [
          '--normalization-mode',
          'semantic-full-stack',
          '--semantic-manifest',
          '/tmp/semantic.json',
        ],
        {},
      ),
    /revision-role is required/,
  );
  assert.throws(
    () =>
      capture.parseCaptureOptions(
        ['--normalization-mode', 'disabled-browser-compatibility', '--revision-role', 'other'],
        {},
      ),
    /base, head, or current/,
  );
  assert.throws(() => capture.parseCaptureOptions([], {}), /normalization-mode is required/);
  assert.throws(
    () => comparison.parseComparisonOptions(['--main', 'one', '--main', 'two'], {}),
    /Duplicate/,
  );
  assert.throws(
    () => comparison.parseComparisonOptions(['--not-a-real-option', 'value'], {}),
    /Unknown/,
  );
});

test('comparison refuses aliased input/output directories before deleting captures', async (t) => {
  const root = fixtureDirectory(t);
  const mainDir = path.join(root, 'main');
  const prDir = path.join(root, 'pr');
  fs.mkdirSync(mainDir);
  fs.mkdirSync(prDir);
  const capturePath = path.join(mainDir, 'pipelines-10x10.png');
  fs.writeFileSync(capturePath, 'keep-me');

  await assert.rejects(
    comparison.runComparison(comparisonOptions(mainDir, prDir, mainDir)),
    /--main and --output must resolve to distinct directories/,
  );
  assert.equal(fs.readFileSync(capturePath, 'utf8'), 'keep-me');

  const alias = path.join(root, 'main-alias');
  fs.symlinkSync(mainDir, alias);
  await assert.rejects(
    comparison.runComparison(comparisonOptions(mainDir, alias, path.join(root, 'output'))),
    /--main and --pr must resolve to distinct directories/,
  );
});

test('waitForFunction actions invoke the supplied predicate', async () => {
  let invoked = false;
  const page = {
    waitForFunction: async (runner, argument, options) => {
      assert.equal(typeof runner, 'function');
      assert.equal(argument, undefined);
      assert.deepEqual(options, { timeout: 10000 });
      invoked = await runner();
    },
  };

  await capture.executeActions(page, [{ type: 'waitForFunction', predicate: () => 2 + 2 === 4 }]);
  assert.equal(invoked, true);
});

test('waitForSelectedTab accepts ARIA tabs and styled MD2Tabs buttons', async (t) => {
  const originalDocument = global.document;
  const originalGetComputedStyle = global.getComputedStyle;
  t.after(() => {
    global.document = originalDocument;
    global.getComputedStyle = originalGetComputedStyle;
  });

  const candidates = [
    { getAttribute: () => null, textContent: 'Input/Output' },
    { getAttribute: () => 'true', textContent: 'Logs' },
  ];
  global.document = {
    querySelectorAll: (selector) => {
      assert.equal(selector, '[role="tab"], button');
      return candidates;
    },
  };
  global.getComputedStyle = (candidate) => ({
    fontWeight: candidate === candidates[0] ? '700' : '400',
  });

  const observed = [];
  const page = {
    locator: () => ({ first: () => ({}) }),
    waitForFunction: async (runner, argument, options) => {
      observed.push(runner(argument));
      assert.deepEqual(options, { timeout: 10000 });
    },
  };

  await capture.executeActions(page, [
    {
      type: 'waitForSelectedTab',
      selector: '[role="tab"], button',
      text: 'Input/Output',
    },
    { type: 'waitForSelectedTab', selector: '[role="tab"], button', text: 'Logs' },
  ]);
  assert.deepEqual(observed, [true, true]);
});

test('capture scroll normalization resets and verifies the document viewport', async () => {
  const calls = [];
  let x = 17;
  let y = 29;
  const page = {
    evaluate: async (runner) => {
      const source = String(runner);
      if (source.includes('window.scrollTo')) {
        calls.push('reset');
        x = 0;
        y = 0;
        return undefined;
      }
      calls.push('read');
      return { x, y };
    },
    waitForFunction: async (runner, argument, options) => {
      calls.push('assert');
      assert.equal(argument, undefined);
      assert.deepEqual(options, { timeout: 10000 });
      assert.match(String(runner), /scrollingElement\.scrollTop === 0/);
    },
  };

  assert.deepEqual(await capture.normalizeDocumentScroll(page), { x: 0, y: 0 });
  assert.deepEqual(calls, ['reset', 'assert', 'read']);
});

test('capture resets nested overflow containers even when the document never scrolled', async (t) => {
  const originalDocument = global.document;
  const originalWindow = global.window;
  t.after(() => {
    global.document = originalDocument;
    global.window = originalWindow;
  });
  const nested = [
    { scrollTop: 430, scrollLeft: 0 },
    { scrollTop: 82, scrollLeft: 12 },
  ];
  global.document = {
    scrollingElement: { scrollTop: 0, scrollLeft: 0 },
    querySelectorAll: () => nested,
  };
  global.window = { scrollX: 0, scrollY: 0, scrollTo() {} };
  const page = {
    evaluate: async (runner) => runner(),
    waitForFunction: async (runner) => assert.equal(runner(), true),
  };
  assert.deepEqual(await capture.normalizeDocumentScroll(page), { x: 0, y: 0 });
  assert.deepEqual(nested, [
    { scrollTop: 0, scrollLeft: 0 },
    { scrollTop: 0, scrollLeft: 0 },
  ]);
});

test('capture preserves the auto-follow log viewport while resetting app scrolling', async (t) => {
  const originalDocument = global.document;
  const originalWindow = global.window;
  t.after(() => {
    global.document = originalDocument;
    global.window = originalWindow;
  });
  const app = { scrollTop: 82, scrollLeft: 12 };
  const logs = {
    get scrollTop() {
      return 430;
    },
    set scrollTop(value) {
      assert.fail('must not fight log auto-follow');
    },
    scrollLeft: 0,
  };
  global.document = {
    scrollingElement: { scrollTop: 0, scrollLeft: 0 },
    querySelectorAll: (selector) =>
      selector === '*:not(#logViewer, #logViewer *)' ? [app] : [app, logs],
  };
  global.window = { scrollX: 0, scrollY: 0, scrollTo() {} };
  const page = {
    evaluate: async (runner) => runner(),
    waitForFunction: async (runner) => assert.equal(runner(), true),
  };
  assert.deepEqual(await capture.normalizeDocumentScroll(page), { x: 0, y: 0 });
  assert.deepEqual(app, { scrollTop: 0, scrollLeft: 0 });
  assert.equal(logs.scrollTop, 430);
});

test('capture clears pointer hover before settling and resetting the viewport', async () => {
  const events = [];
  const page = {
    mouse: { move: async (x, y) => events.push(['pointer', x, y]) },
    waitForTimeout: async (ms) => events.push(['settle', ms]),
    frames: () => [],
    evaluate: async (runner) => {
      events.push(['scroll']);
      return { x: 0, y: 0 };
    },
    waitForFunction: async () => {},
  };
  await capture.prepareCaptureViewport(page);
  assert.deepEqual(events.slice(0, 3), [['pointer', -1, -1], ['settle', 350], ['scroll']]);
});

test('fixture list sorting uses the application control and waits for its response', async () => {
  let ascending = false;
  let clicks = 0;
  let waits = 0;
  const page = {
    locator: () => ({
      filter: () => ({
        evaluate: async () => ascending,
        click: async () => {
          clicks += 1;
          ascending = true;
        },
      }),
    }),
    waitForLoadState: async () => {
      waits += 1;
    },
  };
  await capture.sortFixtureList(page, 'Pipeline name');
  assert.equal(clicks, 1);
  assert.equal(waits, 1);
  await capture.sortFixtureList(page, 'Pipeline name');
  assert.equal(clicks, 1);
});

test('timestamp normalization accepts Intl nonbreaking spaces without masking fixture labels', async (t) => {
  const originalDocument = global.document;
  const originalNodeFilter = global.NodeFilter;
  t.after(() => {
    global.document = originalDocument;
    global.NodeFilter = originalNodeFilter;
  });
  const nodes = [
    { nodeValue: '9/5/2026, 11:01:00\u202fAM' },
    { nodeValue: 'UI Smoke Training Run 1 — accuracy 0.95' },
    { nodeValue: 'Sat Sep 05 2026 11:01:00 GMT+0000 (Coordinated Universal Time)' },
    { nodeValue: '2026-09-05T11:01:00Z duration 00:01:22' },
  ];
  let index = 0;
  global.NodeFilter = { SHOW_TEXT: 4 };
  global.document = {
    body: {},
    createTreeWalker: () => ({ nextNode: () => nodes[index++] }),
  };
  await capture.normalizeDynamicText({ evaluate: async (runner, arg) => runner(arg) });
  assert.equal(nodes[0].nodeValue, '1/2/2030, 3:04:05 AM');
  assert.equal(nodes[1].nodeValue, 'UI Smoke Training Run 1 — accuracy 0.95');
  assert.equal(nodes[2].nodeValue, new Date('2030-01-02T03:04:05.000Z').toString());
  assert.equal(nodes[3].nodeValue, '2030-01-02T03:04:05.000Z duration 00:00:42');
});

test('comparison fails closed when paired document scroll positions differ', async (t) => {
  const root = fixtureDirectory(t);
  const mainDir = path.join(root, 'main');
  const prDir = path.join(root, 'pr');
  const outputDir = path.join(root, 'comparison');
  fs.mkdirSync(mainDir);
  fs.mkdirSync(prDir);
  const filename = 'pipelines-10x10.png';
  await Promise.all([writePng(path.join(mainDir, filename)), writePng(path.join(prDir, filename))]);
  writeCaptureManifest(
    mainDir,
    [captureResult(filename, { documentScroll: { x: 0, y: 0 } })],
    'base',
  );
  writeCaptureManifest(
    prDir,
    [captureResult(filename, { documentScroll: { x: 0, y: 40 } })],
    'head',
  );

  const result = await comparison.runComparison(comparisonOptions(mainDir, prDir, outputDir));
  assert.equal(result.exitCode, 1);
  assert.equal(result.summary.stats.failed, 1);
  assert.match(result.summary.results[0].error, /document scroll position differs/);
  assert.equal(result.summary.results[0].thresholdsEvaluated, false);
});

test('compare capture readiness rejects loaders and error states', (t) => {
  const originalDocument = global.document;
  t.after(() => {
    global.document = originalDocument;
  });

  const installDocument = ({ error = false, loading = false, rows = 2 } = {}) => {
    global.document = {
      body: { innerText: 'Run Comparison Parameters Scalar Metrics' },
      querySelector(selector) {
        return selector === '[role="alert"]' && error ? {} : null;
      },
      querySelectorAll(selector) {
        if (selector.includes('circularprogress')) return { length: loading ? 1 : 0 };
        if (selector === '[data-testid="table-row"]') return { length: rows };
        return { length: 0 };
      },
    };
  };

  installDocument();
  assert.equal(capture.comparePageReadyPredicate(), true);
  installDocument({ loading: true });
  assert.equal(capture.comparePageReadyPredicate(), false);
  installDocument({ error: true });
  assert.equal(capture.comparePageReadyPredicate(), false);
  installDocument({ rows: 0 });
  assert.equal(capture.comparePageReadyPredicate(), false);
});

test('seeded scalar-metrics readiness requires deterministic populated metrics', (t) => {
  const originalDocument = global.document;
  t.after(() => {
    global.document = originalDocument;
  });

  const installDocument = ({ empty = false, labels = [] } = {}) => {
    global.document = {
      body: {
        innerText: empty
          ? 'There are no Scalar Metrics artifacts available on the selected runs.'
          : 'Run Comparison Scalar Metrics',
      },
      querySelectorAll(selector) {
        if (selector === 'table tbody tr > td:first-child[title]') {
          return labels.map((label) => ({ getAttribute: () => label }));
        }
        return [];
      },
    };
  };

  installDocument({ empty: true, labels: ['accuracy', 'loss'] });
  assert.equal(capture.scalarMetricsReadyPredicate(), false);
  installDocument({ labels: ['accuracy'] });
  assert.equal(capture.scalarMetricsReadyPredicate(), false);
  installDocument({ labels: ['accuracy', 'loss'] });
  assert.equal(capture.scalarMetricsReadyPredicate(), true);
});

test('seeded ROC readiness requires a non-empty heading and rendered data line', (t) => {
  const originalDocument = global.document;
  t.after(() => {
    global.document = originalDocument;
  });

  const installDocument = ({ empty = false, heading = null, line = false } = {}) => {
    global.document = {
      body: {
        innerText: empty
          ? 'There are no ROC Curve artifacts available on the selected runs.'
          : 'ROC Curve',
      },
      querySelector(selector) {
        return selector === '.recharts-wrapper .recharts-line-curve' && line ? {} : null;
      },
      querySelectorAll(selector) {
        return selector === 'h3' && heading ? [{ textContent: heading }] : [];
      },
    };
  };

  installDocument({ empty: true, heading: 'ROC Curve: metrics', line: true });
  assert.equal(capture.rocCurveReadyPredicate(), false);
  installDocument({ heading: 'ROC Curve: metrics' });
  assert.equal(capture.rocCurveReadyPredicate(), false);
  installDocument({ heading: 'ROC Curve: no artifacts', line: true });
  assert.equal(capture.rocCurveReadyPredicate(), false);
  installDocument({ heading: 'ROC Curve: multiple artifacts', line: true });
  assert.equal(capture.rocCurveReadyPredicate(), true);
});

test('capture cleanup removes only files listed in its ownership marker', (t) => {
  const directory = fixtureDirectory(t);
  for (const filename of [
    'pipelines-1280x800.png',
    'pipelines-390x844.png',
    'other-1280x800.png',
    'notes.png',
    'manifest.json',
  ]) {
    fs.writeFileSync(path.join(directory, filename), filename);
  }
  fs.writeFileSync(
    path.join(directory, capture.CAPTURE_OWNER_FILENAME),
    JSON.stringify({
      schemaVersion: 1,
      files: ['pipelines-1280x800.png', 'pipelines-390x844.png'],
    }),
  );

  capture.cleanCaptureOutputs(directory, ['pipelines-1440x900.png']);
  assert.deepEqual(fs.readdirSync(directory).sort(), [
    capture.CAPTURE_OWNER_FILENAME,
    'notes.png',
    'other-1280x800.png',
  ]);
  assert.deepEqual(
    JSON.parse(fs.readFileSync(path.join(directory, capture.CAPTURE_OWNER_FILENAME), 'utf8')),
    { schemaVersion: 1, files: ['pipelines-1440x900.png'] },
  );
});

test('capture cleanup rejects non-empty unowned and malformed managed directories', (t) => {
  const unowned = fixtureDirectory(t);
  fs.writeFileSync(path.join(unowned, 'pipelines-1280x800.png'), 'keep-me');

  assert.throws(
    () => capture.cleanCaptureOutputs(unowned, ['pipelines-1440x900.png']),
    /non-empty unowned capture directory/,
  );
  assert.equal(fs.readFileSync(path.join(unowned, 'pipelines-1280x800.png'), 'utf8'), 'keep-me');

  const malformed = fixtureDirectory(t);
  fs.writeFileSync(
    path.join(malformed, capture.CAPTURE_OWNER_FILENAME),
    JSON.stringify({ schemaVersion: 1, files: ['../notes.png'] }),
  );
  assert.throws(
    () => capture.cleanCaptureOutputs(malformed, ['pipelines-1440x900.png']),
    /invalid or duplicate filename/,
  );

  const collision = fixtureDirectory(t);
  fs.writeFileSync(
    path.join(collision, capture.CAPTURE_OWNER_FILENAME),
    JSON.stringify({ schemaVersion: 1, files: [] }),
  );
  fs.writeFileSync(path.join(collision, 'pipelines-1440x900.png'), 'keep-me');
  assert.throws(
    () => capture.cleanCaptureOutputs(collision, ['pipelines-1440x900.png']),
    /Refusing to overwrite an unmanaged capture output/,
  );
});

test('required incomplete captures fail while optional skips do not', () => {
  const complete = capture.summarizeCaptureResults([
    { required: true, status: 'success' },
    { required: false, status: 'skipped' },
  ]);
  assert.equal(complete.complete, true);
  assert.equal(complete.requiredIncomplete, 0);

  for (const status of ['skipped', 'degraded', 'failed']) {
    const summary = capture.summarizeCaptureResults([
      { required: true, status },
      { required: false, status: 'success' },
    ]);
    assert.equal(summary.complete, false, `required ${status} should fail`);
    assert.equal(summary.requiredIncomplete, 1);
  }
});

test('pipeline capture uses the current route and page-specific readiness', () => {
  const pipelinePage = capture.PAGES.find((page) => page.name === 'pipeline-create');
  assert.equal(pipelinePage.path, '/#/pipeline_versions/new');
  assert.equal(pipelinePage.waitFor, '#dropZone');
});

test('pipeline details use revision-stable graph and semantic node selectors', (t) => {
  const originalDocument = global.document;
  const originalGetComputedStyle = global.getComputedStyle;
  t.after(() => {
    global.document = originalDocument;
    global.getComputedStyle = originalGetComputedStyle;
  });
  for (const pageName of ['pipeline-details-seeded', 'pipeline-details-seeded-sidepanel']) {
    const page = capture.PAGES.find((candidate) => candidate.name === pageName);
    assert.equal(page.waitFor, capture.PIPELINE_DETAILS_ROOT_SELECTOR);
    assert.equal(page.waitForData, undefined);
    assert.equal(
      page.actions.some(
        (action) =>
          action.type === 'waitForFunction' &&
          action.predicate === capture.pipelineDetailsGraphReadyPredicate,
      ),
      true,
    );
  }
  const sidePanel = capture.PAGES.find(
    (candidate) => candidate.name === 'pipeline-details-seeded-sidepanel',
  );
  assert.equal(
    sidePanel.actions.find((action) => action.type === 'click').selector,
    capture.PIPELINE_DETAILS_WRITE_METRICS_SELECTOR,
  );
  assert.match(capture.PIPELINE_DETAILS_GRAPH_SELECTOR, /pipeline-detail-v1/);
  assert.match(capture.PIPELINE_DETAILS_GRAPH_SELECTOR, /pipeline-detail-v2/);

  const node = {};
  const root = { querySelectorAll: () => [node] };
  global.document = {
    querySelector: (selector) => (selector === '[role="alert"]' ? null : root),
  };
  global.getComputedStyle = () => ({ display: 'block', visibility: 'visible' });
  assert.equal(capture.pipelineDetailsGraphReadyPredicate(), true);
  global.getComputedStyle = () => ({ display: 'none', visibility: 'visible' });
  assert.equal(capture.pipelineDetailsGraphReadyPredicate(), false);
  root.querySelectorAll = () => [];
  assert.equal(capture.pipelineDetailsGraphReadyPredicate(), false);
  global.document.querySelector = (selector) =>
    selector === '[role="alert"]' ? { textContent: 'failed' } : root;
  assert.equal(capture.pipelineDetailsGraphReadyPredicate(), false);
});

test('semantic full-stack capture precisely normalizes the removed Executions nav row', async () => {
  let baseCss = null;
  const basePage = {
    addStyleTag: async ({ content }) => {
      baseCss = content;
    },
    locator: (selector) => {
      assert.equal(selector, '#executionsBtn');
      return {
        count: async () => 1,
        evaluateAll: async () => (baseCss?.includes('display: none !important') ? 1 : 0),
      };
    },
  };
  const baseEvidence = await capture.applyGlobalVisualNormalizations(
    basePage,
    'semantic-full-stack',
    'base',
  );
  assert.equal(baseEvidence.complete, true);
  assert.deepEqual(baseEvidence.rules[0], {
    actualMatches: 1,
    applied: true,
    expectedChange:
      'The Executions sidebar entry is intentionally removed; semantic full-stack base captures hide only that entry so the remaining navigation stays visually comparable.',
    expectedMatches: 1,
    hiddenMatches: 1,
    key: 'executions-navigation-removal',
    operation: 'hide',
    selector: '#executionsBtn',
  });
  assert.equal(baseCss, '#executionsBtn { display: none !important; }');

  let headStyleCalls = 0;
  const headEvidence = await capture.applyGlobalVisualNormalizations(
    {
      addStyleTag: async () => {
        headStyleCalls += 1;
      },
      locator: () => ({ count: async () => 0 }),
    },
    'semantic-full-stack',
    'head',
  );
  assert.equal(headEvidence.complete, true);
  assert.equal(headEvidence.rules[0].applied, false);
  assert.equal(headStyleCalls, 0);
  assert.equal(
    await capture.applyGlobalVisualNormalizations(
      basePage,
      'disabled-browser-compatibility',
      'base',
    ),
    null,
  );

  for (const [role, count] of [
    ['base', 0],
    ['head', 1],
  ]) {
    await assert.rejects(
      capture.applyGlobalVisualNormalizations(
        { locator: () => ({ count: async () => count }) },
        'semantic-full-stack',
        role,
      ),
      (error) => {
        assert.equal(error.captureValidity, 'selector_drift');
        assert.equal(error.globalVisualNormalization.complete, false);
        assert.equal(error.globalVisualNormalization.rules[0].actualMatches, count);
        return true;
      },
    );
  }
});

test('list-page readiness accepts exact empty states without accepting loading or errors', (t) => {
  const originalDocument = global.document;
  t.after(() => {
    global.document = originalDocument;
  });
  const emptyStates = new Map([
    ['pipelines', 'No pipelines found. Click "Upload pipeline" to start.'],
    ['experiments', 'No experiments found. Click "Create experiment" to start.'],
    ['runs', 'No available runs found.'],
    ['recurring-runs', 'No available recurring runs found.'],
    ['executions', 'No executions found.'],
  ]);

  for (const [pageName, emptyState] of emptyStates) {
    const page = capture.PAGES.find((candidate) => candidate.name === pageName);
    const predicate = page.actions.find((action) => action.type === 'waitForFunction').predicate;
    global.document = {
      body: { innerText: emptyState },
      querySelector: () => null,
      querySelectorAll: () => [],
    };
    assert.equal(predicate(), true, `${pageName} should accept its exact empty state`);
    global.document.body.innerText = 'Loading...';
    assert.equal(predicate(), false, `${pageName} should not accept loading text`);
    global.document.body.innerText = `Error: ${emptyState}`;
    global.document.querySelector = () => ({ role: 'alert' });
    assert.equal(predicate(), false, `${pageName} should reject an error alongside empty text`);
  }
});

test('navigation validation rejects missing and unsuccessful HTTP responses', () => {
  assert.throws(
    () => capture.assertNavigationResponse(null, 'http://example.test'),
    /did not return/,
  );
  assert.throws(
    () =>
      capture.assertNavigationResponse(
        { ok: () => false, status: () => 500, statusText: () => 'Internal Server Error' },
        'http://example.test',
      ),
    /HTTP 500 Internal Server Error/,
  );
  assert.doesNotThrow(() =>
    capture.assertNavigationResponse(
      { ok: () => true, status: () => 200, statusText: () => 'OK' },
      'http://example.test',
    ),
  );
});

test('capture network isolation blocks another loopback port and cross-origin WebSockets', async () => {
  let httpHandler;
  let webSocketHandler;
  await capture.installNetworkIsolation(
    {
      route: async (_pattern, handler) => {
        httpHandler = handler;
      },
      routeWebSocket: async (_pattern, handler) => {
        webSocketHandler = handler;
      },
    },
    'http://127.0.0.1:4002/kfp',
  );

  const decisionFor = async (url) => {
    let decision;
    await httpHandler({
      abort: async () => {
        decision = 'abort';
      },
      continue: async () => {
        decision = 'continue';
      },
      request: () => ({ url: () => url }),
    });
    return decision;
  };
  assert.equal(await decisionFor('http://127.0.0.1:4002/apis/v2beta1/runs'), 'continue');
  assert.equal(await decisionFor('http://127.0.0.1:3001/apis/v2beta1/runs'), 'abort');
  assert.equal(await decisionFor('https://example.com/tracker'), 'abort');

  let connected = false;
  webSocketHandler({
    connectToServer: () => {
      connected = true;
    },
    url: () => 'ws://127.0.0.1:3001/private',
  });
  assert.equal(connected, false);
  webSocketHandler({
    connectToServer: () => {
      connected = true;
    },
    url: () => 'ws://127.0.0.1:4002/events',
  });
  assert.equal(connected, true);
});

test('capture flow uses browser sandbox defaults, stabilizes rendering, and enforces navigation', async (t) => {
  const root = fixtureDirectory(t);
  const makeChromium = (status, events) => ({
    launch: async (launchOptions) => {
      assert.deepEqual(launchOptions, { headless: true });
      return {
        newContext: async () => ({
          route: async () => events.push('network-route'),
          routeWebSocket: async () => events.push('websocket-route'),
          newPage: async () => {
            events.push('new-page');
            let navigationCount = 0;
            return {
              mouse: { move: async () => events.push('neutral-pointer') },
              frames: () => [],
              emulateMedia: async () => events.push('reduced-motion'),
              addInitScript: async () => events.push('init-css'),
              goto: async (url) => {
                events.push(`goto:${url}`);
                navigationCount += 1;
                if (navigationCount > 1) {
                  return null;
                }
                return {
                  ok: () => status >= 200 && status < 300,
                  status: () => status,
                  statusText: () => (status === 200 ? 'OK' : 'Server Error'),
                };
              },
              evaluate: async (pageFunction) => {
                const source = String(pageFunction);
                if (source.includes('querySelectorAll(scope.selector)')) {
                  events.push('semantic-id-normalization');
                  return [{ counts: { 'run.training-1': 1 }, rootCount: 1 }];
                }
                events.push(source.includes('document.fonts') ? 'fonts' : 'css');
              },
              locator: () => ({ count: async () => 0 }),
              waitForFunction: async () => events.push('action'),
              waitForSelector: async () => events.push('selector'),
              waitForTimeout: async () => events.push('settled'),
              screenshot: async (screenshotOptions) => {
                assert.equal(screenshotOptions.animations, 'disabled');
                events.push('screenshot');
                fs.writeFileSync(screenshotOptions.path, 'synthetic screenshot');
              },
              close: async () => events.push('page-close'),
            };
          },
          close: async () => events.push('context-close'),
        }),
        close: async () => events.push('browser-close'),
      };
    },
  });
  const seedManifestPath = path.join(root, 'seed.json');
  const semanticManifestPath = path.join(root, 'semantic.json');
  const sourceProvenancePath = path.join(root, 'source.json');
  fs.writeFileSync(seedManifestPath, JSON.stringify({ defaults: {}, resources: {} }));
  fs.writeFileSync(semanticManifestPath, JSON.stringify(strictSemanticFixtureManifest()));
  fs.writeFileSync(sourceProvenancePath, JSON.stringify({ schemaVersion: 'ui-smoke-source/v1' }));
  const options = {
    baseUrl: 'https://example.test/kfp',
    label: 'fixture',
    outputDir: path.join(root, 'success'),
    pageNames: null,
    pages: [
      {
        actions: [{ type: 'waitForFunction', predicate: () => true }],
        name: 'fixture-page',
        path: '/#/fixture',
        semanticIdNormalization: {
          scopes: [
            {
              match: 'exact',
              minReplacements: 1,
              selector: '#root',
              semanticIds: ['run.training-1'],
            },
          ],
        },
        waitFor: '#ready',
      },
      { name: 'second-page', path: '/#/second', waitFor: '#ready' },
    ],
    revisionRole: 'head',
    semanticIdNormalizationMode: 'semantic-full-stack',
    seedManifestPath,
    semanticManifestPath,
    sourceProvenancePath,
    viewports: [{ width: 10, height: 10 }],
  };

  const successEvents = [];
  const success = await capture.captureScreenshots(options, {
    chromium: makeChromium(200, successEvents),
  });
  assert.equal(success.exitCode, 0);
  assert.equal(success.manifest.results.length, 2);
  assert.equal(success.manifest.results[0].sha256.length, 64);
  assert.equal(success.manifest.inputs.revisionRole, 'head');
  assert.equal(success.manifest.inputs.seedManifest.sha256.length, 64);
  assert.equal(success.manifest.inputs.semanticManifest.schemaVersion, 'ui-smoke-semantic/v3');
  assert.equal(success.manifest.inputs.sourceProvenance.schemaVersion, 'ui-smoke-source/v1');
  assert.equal(
    success.manifest.deterministicRendering.semanticIdNormalization.schemaVersion,
    'ui-smoke-id-normalization/v1',
  );
  assert.equal(
    success.manifest.deterministicRendering.semanticIdNormalization.mode,
    'semantic-full-stack',
  );
  assert.equal(success.manifest.results[0].semanticIdNormalization.complete, true);
  assert.equal(success.manifest.results[0].semanticIdNormalization.scopes.length, 1);
  assert.equal(success.manifest.results[0].semanticIdNormalization.totalReplacementCount, 1);
  assert.ok(successEvents.includes('goto:https://example.test/kfp/#/fixture'));
  assert.ok(successEvents.includes('goto:https://example.test/kfp/#/second'));
  assert.equal(successEvents.filter((event) => event === 'new-page').length, 2);
  assert.equal(successEvents.filter((event) => event === 'page-close').length, 2);
  assert.ok(successEvents.indexOf('init-css') < successEvents.indexOf('screenshot'));
  assert.ok(successEvents.indexOf('action') < successEvents.indexOf('semantic-id-normalization'));
  assert.ok(
    successEvents.indexOf('semantic-id-normalization') < successEvents.indexOf('screenshot'),
  );
  assert.ok(successEvents.indexOf('fonts') < successEvents.indexOf('screenshot'));

  const browserOnly = await capture.captureScreenshots(
    {
      ...options,
      outputDir: path.join(root, 'browser-only'),
      semanticIdNormalizationMode: 'disabled-browser-compatibility',
      semanticManifestPath: null,
      sourceProvenancePath: null,
    },
    { chromium: makeChromium(200, []) },
  );
  assert.equal(browserOnly.exitCode, 0);
  assert.equal(
    browserOnly.manifest.deterministicRendering.semanticIdNormalization.mode,
    'disabled-browser-compatibility',
  );
  assert.deepEqual(browserOnly.manifest.results[0].semanticIdNormalization.scopes, []);

  const failureEvents = [];
  const failure = await capture.captureScreenshots(
    { ...options, outputDir: path.join(root, 'failure') },
    { chromium: makeChromium(500, failureEvents) },
  );
  assert.equal(failure.exitCode, 1);
  assert.equal(failure.manifest.results[0].status, 'failed');
  assert.match(failure.manifest.results[0].error, /HTTP 500/);
  assert.equal(failureEvents.includes('screenshot'), false);
});

test('revision-aware capture fails malformed or missing seed manifests before browser launch', async (t) => {
  const root = fixtureDirectory(t);
  const semanticManifestPath = path.join(root, 'semantic.json');
  fs.writeFileSync(semanticManifestPath, JSON.stringify({ schemaVersion: 'ui-smoke-semantic/v3' }));
  const sourceProvenancePath = path.join(root, 'source.json');
  fs.writeFileSync(sourceProvenancePath, JSON.stringify({ schemaVersion: 'ui-smoke-source/v1' }));
  const malformedSeedPath = path.join(root, 'malformed-seed.json');
  fs.writeFileSync(malformedSeedPath, '{not-json');
  let launchCalls = 0;
  const chromium = {
    async launch() {
      launchCalls += 1;
      throw new Error('browser must not launch for an invalid seed manifest');
    },
  };

  for (const [name, seedManifestPath] of [
    ['missing', path.join(root, 'missing-seed.json')],
    ['malformed', malformedSeedPath],
  ]) {
    const run = await capture.captureScreenshots(
      {
        baseUrl: 'https://example.test/kfp',
        label: name,
        outputDir: path.join(root, name),
        pageNames: ['executions-to-runs'],
        revisionRole: 'head',
        seedManifestPath,
        semanticIdNormalizationMode: 'semantic-full-stack',
        semanticManifestPath,
        sourceProvenancePath,
        viewports: [{ width: 10, height: 10 }],
      },
      { chromium },
    );

    assert.equal(run.exitCode, 1);
    assert.equal(run.manifest.results.length, 1);
    assert.equal(run.manifest.results[0].semanticScenario, 'executions-to-runs');
    assert.equal(run.manifest.results[0].status, 'failed');
    assert.equal(run.manifest.results[0].captureValidity, 'seed_failure');
    assert.match(run.manifest.fatalErrors.join(' '), /Seed manifest is invalid/);
    assert.equal(run.manifest.inputs.seedManifest, null);
  }
  assert.equal(launchCalls, 0);
});

test('comparison rejects matching PNG directories without capture manifests', async (t) => {
  const root = fixtureDirectory(t);
  const mainDir = path.join(root, 'main');
  const prDir = path.join(root, 'pr');
  const outputDir = path.join(root, 'comparison');
  fs.mkdirSync(mainDir);
  fs.mkdirSync(prDir);
  await Promise.all([
    writePng(path.join(mainDir, 'pipelines-10x10.png')),
    writePng(path.join(prDir, 'pipelines-10x10.png')),
  ]);

  const run = await comparison.runComparison(comparisonOptions(mainDir, prDir, outputDir));
  assert.equal(run.exitCode, 1);
  assert.match(run.summary.fatalErrors[0], /Both capture manifests are required/);
  assert.equal(fs.existsSync(path.join(outputDir, 'summary.json')), true);
});

test('manifest comparison ignores stale unlisted PNGs and cleans managed outputs', async (t) => {
  const root = fixtureDirectory(t);
  const mainDir = path.join(root, 'main');
  const prDir = path.join(root, 'pr');
  const outputDir = path.join(root, 'comparison');
  fs.mkdirSync(mainDir);
  fs.mkdirSync(prDir);
  fs.mkdirSync(outputDir);

  const filename = 'pipelines-10x10.png';
  await Promise.all([
    writePng(path.join(mainDir, filename)),
    writePng(path.join(prDir, filename)),
    writePng(path.join(mainDir, 'stale-10x10.png')),
  ]);
  writeCaptureManifest(mainDir, [captureResult(filename)], '<script>alert("base")</script>', {
    captureId: 'base-capture',
  });
  writeCaptureManifest(prDir, [captureResult(filename)], 'head & "revision"', {
    captureId: 'head-capture',
  });
  fs.writeFileSync(path.join(outputDir, 'old-10x10.png'), 'old comparison');
  fs.writeFileSync(path.join(outputDir, 'notes-10x10.png'), 'unmanaged');
  fs.writeFileSync(
    path.join(outputDir, 'summary.json'),
    JSON.stringify({ results: [{ filename: 'old-10x10.png' }] }),
  );
  fs.writeFileSync(
    path.join(outputDir, '.managed-outputs.json'),
    JSON.stringify({ schemaVersion: 1, filenames: ['old-10x10.png'] }),
  );

  const run = await comparison.runComparison(comparisonOptions(mainDir, prDir, outputDir), {
    looksSame: equalAnalysis,
  });

  assert.equal(run.exitCode, 0);
  assert.equal(run.summary.sourceMode, 'manifest');
  assert.equal(run.summary.results.length, 1);
  assert.equal(run.summary.results[0].filename, filename);
  assert.equal(run.summary.results[0].diffPercent, 0);
  const mainManifest = fs.readFileSync(path.join(mainDir, 'manifest.json'));
  const prManifest = fs.readFileSync(path.join(prDir, 'manifest.json'));
  assert.equal(run.summary.captures.base.captureId, 'base-capture');
  assert.equal(run.summary.captures.head.captureId, 'head-capture');
  assert.equal(
    run.summary.captures.base.manifestSha256,
    crypto.createHash('sha256').update(mainManifest).digest('hex'),
  );
  assert.equal(
    run.summary.captures.head.manifestSha256,
    crypto.createHash('sha256').update(prManifest).digest('hex'),
  );
  assert.equal(run.summary.captures.base.manifestSizeBytes, mainManifest.length);
  assert.equal(run.summary.captures.head.manifestSizeBytes, prManifest.length);
  assert.deepEqual(run.summary.captures.base.requiredFilenames, [filename]);
  assert.deepEqual(run.summary.captures.head.requiredFilenames, [filename]);
  assert.equal(run.summary.captures.base.inputs.revisionRole, 'base');
  assert.equal(run.summary.captures.head.inputs.revisionRole, 'head');
  assert.deepEqual(
    run.summary.captures.base.inputs.semanticManifest,
    run.summary.captures.head.inputs.semanticManifest,
  );
  assert.equal(fs.existsSync(path.join(outputDir, filename)), true);
  assert.equal(fs.existsSync(path.join(outputDir, 'old-10x10.png')), false);
  assert.equal(fs.existsSync(path.join(outputDir, 'notes-10x10.png')), true);
  assert.equal(run.reportPath, path.join(outputDir, 'report.html'));
  const report = fs.readFileSync(run.reportPath, 'utf8');
  assert.match(report, /^<!doctype html>/);
  assert.equal(report.includes('<script>alert("base")</script>'), false);
  assert.match(report, /&lt;script&gt;alert\(&quot;base&quot;\)&lt;\/script&gt;/);
  assert.match(report, /head &amp; &quot;revision&quot;/);
  assert.match(report, /Comparison validity<\/strong>valid/);
  assert.match(report, /Status: success/);
  assert.match(report, /0\.0000% visual difference/);
  assert.match(report, /Highlighted comparison/);
  assert.equal((report.match(/<img src="data:image\/png;base64,/g) || []).length, 5);
  assert.equal((report.match(/<a href="data:image\/png;base64,/g) || []).length, 5);
  assert.doesNotMatch(report, /(?:href|src)="https?:/);

  fs.writeFileSync(run.reportPath, '<!doctype html><p>stale managed report</p>');
  const rerun = await comparison.runComparison(comparisonOptions(mainDir, prDir, outputDir), {
    looksSame: equalAnalysis,
  });
  const regeneratedReport = fs.readFileSync(rerun.reportPath, 'utf8');
  assert.equal(regeneratedReport.includes('stale managed report'), false);
  assert.equal(regeneratedReport, report);
  assert.deepEqual(
    JSON.parse(fs.readFileSync(path.join(outputDir, '.managed-outputs.json'), 'utf8')),
    {
      schemaVersion: 2,
      artifacts: ['report.html', 'summary.json'],
      filenames: [
        'pipelines-10x10--base.png',
        'pipelines-10x10--head.png',
        'pipelines-10x10--overlay.png',
        'pipelines-10x10--raw-diff.png',
        filename,
      ],
    },
  );
});

test('comparison cleanup rejects non-empty unowned output directories', (t) => {
  const outputDir = fixtureDirectory(t);
  const unrelatedPath = path.join(outputDir, 'pipelines-10x10.png');
  fs.writeFileSync(unrelatedPath, 'keep-me');

  assert.throws(
    () => comparison.cleanComparisonOutputs(outputDir, ['pipelines-10x10.png']),
    /non-empty unowned comparison directory/,
  );
  assert.equal(fs.readFileSync(unrelatedPath, 'utf8'), 'keep-me');

  const legacyOutputDir = path.join(fixtureDirectory(t), 'legacy-comparison');
  fs.mkdirSync(legacyOutputDir);
  fs.writeFileSync(path.join(legacyOutputDir, 'old-10x10.png'), 'managed image');
  fs.writeFileSync(path.join(legacyOutputDir, 'summary.json'), 'managed summary');
  fs.writeFileSync(path.join(legacyOutputDir, 'report.html'), 'unmanaged report');
  fs.writeFileSync(
    path.join(legacyOutputDir, '.managed-outputs.json'),
    JSON.stringify({ schemaVersion: 1, filenames: ['old-10x10.png'] }),
  );

  assert.throws(
    () => comparison.cleanComparisonOutputs(legacyOutputDir, ['next-10x10.png']),
    /unmanaged comparison output: report\.html/,
  );
  assert.equal(
    fs.readFileSync(path.join(legacyOutputDir, 'old-10x10.png'), 'utf8'),
    'managed image',
  );
  assert.equal(
    fs.readFileSync(path.join(legacyOutputDir, 'summary.json'), 'utf8'),
    'managed summary',
  );
  assert.equal(
    fs.readFileSync(path.join(legacyOutputDir, 'report.html'), 'utf8'),
    'unmanaged report',
  );
});

test('optional unavailable captures are recorded as skipped without failing comparison', async (t) => {
  const root = fixtureDirectory(t);
  const mainDir = path.join(root, 'main');
  const prDir = path.join(root, 'pr');
  const outputDir = path.join(root, 'comparison');
  fs.mkdirSync(mainDir);
  fs.mkdirSync(prDir);

  const requiredFilename = 'pipelines-10x10.png';
  const optionalFilename = 'artifact-lineage-10x10.png';
  await Promise.all([
    writePng(path.join(mainDir, requiredFilename)),
    writePng(path.join(prDir, requiredFilename)),
  ]);
  writeCaptureManifest(
    mainDir,
    [
      captureResult(requiredFilename),
      captureResult(optionalFilename, { required: false, status: 'skipped', reason: 'no data' }),
    ],
    'base',
  );
  writeCaptureManifest(
    prDir,
    [
      captureResult(requiredFilename),
      captureResult(optionalFilename, { required: false, status: 'skipped', reason: 'no data' }),
    ],
    'head',
  );

  const run = await comparison.runComparison(comparisonOptions(mainDir, prDir, outputDir), {
    looksSame: equalAnalysis,
  });
  assert.equal(run.exitCode, 0);
  assert.equal(run.summary.stats.success, 1);
  assert.equal(run.summary.stats.skipped, 1);
  assert.equal(run.summary.valid, true);
});

test('required skipped captures fail and still produce summary.json', async (t) => {
  const root = fixtureDirectory(t);
  const mainDir = path.join(root, 'main');
  const prDir = path.join(root, 'pr');
  const outputDir = path.join(root, 'comparison');
  fs.mkdirSync(mainDir);
  fs.mkdirSync(prDir);
  const filename = 'pipelines-10x10.png';
  await writePng(path.join(mainDir, filename));
  writeCaptureManifest(mainDir, [captureResult(filename)], 'base');
  writeCaptureManifest(
    prDir,
    [captureResult(filename, { status: 'skipped', reason: 'selector missing' })],
    'head',
  );

  const run = await comparison.runComparison(comparisonOptions(mainDir, prDir, outputDir));
  assert.equal(run.exitCode, 1);
  assert.equal(run.summary.stats.failed, 1);
  assert.equal(run.summary.results[0].failureType, 'capture');
  assert.equal(fs.existsSync(path.join(outputDir, 'summary.json')), true);
});

test('comparison rejects identical capture IDs and inconsistent completion metadata', async (t) => {
  const root = fixtureDirectory(t);
  const mainDir = path.join(root, 'main');
  const prDir = path.join(root, 'pr');
  const outputDir = path.join(root, 'comparison');
  fs.mkdirSync(mainDir);
  fs.mkdirSync(prDir);
  const filename = 'pipelines-10x10.png';
  await Promise.all([writePng(path.join(mainDir, filename)), writePng(path.join(prDir, filename))]);
  writeCaptureManifest(mainDir, [captureResult(filename)], 'base', { captureId: 'same' });
  writeCaptureManifest(prDir, [captureResult(filename)], 'head', { captureId: 'same' });

  const sameCapture = await comparison.runComparison(comparisonOptions(mainDir, prDir, outputDir));
  assert.equal(sameCapture.exitCode, 1);
  assert.match(sameCapture.summary.fatalErrors[0], /same captureId/);

  writeCaptureManifest(prDir, [captureResult(filename)], 'head');
  const manifestPath = path.join(prDir, 'manifest.json');
  const malformed = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  malformed.complete = false;
  fs.writeFileSync(manifestPath, JSON.stringify(malformed));
  const inconsistent = await comparison.runComparison(comparisonOptions(mainDir, prDir, outputDir));
  assert.equal(inconsistent.exitCode, 1);
  assert.match(inconsistent.summary.fatalErrors[0], /inconsistent completion metadata/);
});

test('real image analysis compares every pixel because capture CSS hides carets', async (t) => {
  const root = fixtureDirectory(t);
  const mainPath = path.join(root, 'main.png');
  const prPath = path.join(root, 'pr.png');
  await Promise.all([writePng(mainPath, 10, 10, '#000000'), writePng(prPath, 10, 10, '#ffffff')]);

  const analysis = await comparison.analyzeDiff(mainPath, prPath, comparisonOptions('', '', ''));
  assert.equal(analysis.diffPercent, 100);
});

test('missing, stale, corrupt, dimension-mismatched, and null-analysis images fail closed', async (t) => {
  const scenarios = [
    {
      name: 'missing',
      expectedType: 'missing',
      mutate: ({ prPath }) => fs.unlinkSync(prPath),
      analyze: equalAnalysis,
    },
    {
      name: 'stale',
      expectedType: 'stale',
      mutate: ({ prPath }) => {
        const old = new Date(Date.now() - 60_000);
        fs.utimesSync(prPath, old, old);
      },
      analyze: equalAnalysis,
    },
    {
      name: 'post-window timestamp',
      expectedType: 'stale',
      mutate: ({ prPath }) => {
        const future = new Date(Date.now() + 60_000);
        fs.utimesSync(prPath, future, future);
      },
      analyze: equalAnalysis,
    },
    {
      name: 'content replaced',
      expectedType: 'integrity',
      mutate: async ({ prPath }) => writePng(prPath, 10, 10, '#ff0000'),
      analyze: equalAnalysis,
    },
    {
      name: 'corrupt',
      expectedType: 'corrupt',
      mutate: ({ prPath }) => fs.writeFileSync(prPath, 'not a PNG'),
      analyze: equalAnalysis,
      refreshManifest: true,
    },
    {
      name: 'dimension mismatch',
      expectedType: 'dimension-mismatch',
      mutate: async ({ prPath }) => writePng(prPath, 12, 10),
      analyze: equalAnalysis,
      refreshManifest: true,
    },
    {
      name: 'null analysis',
      expectedType: 'analysis',
      mutate: () => {},
      analyze: async () => ({ equal: false, diffBounds: null, diffClusters: [] }),
    },
  ];

  for (const scenario of scenarios) {
    await t.test(scenario.name, async () => {
      const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-failure-'));
      try {
        const mainDir = path.join(root, 'main');
        const prDir = path.join(root, 'pr');
        const outputDir = path.join(root, 'comparison');
        fs.mkdirSync(mainDir);
        fs.mkdirSync(prDir);
        const filename = 'pipelines-10x10.png';
        const mainPath = path.join(mainDir, filename);
        const prPath = path.join(prDir, filename);
        await Promise.all([writePng(mainPath), writePng(prPath)]);
        writeCaptureManifest(mainDir, [captureResult(filename)], 'base');
        writeCaptureManifest(prDir, [captureResult(filename)], 'head');
        await scenario.mutate({ mainPath, prPath });
        if (scenario.refreshManifest) {
          writeCaptureManifest(prDir, [captureResult(filename)], 'head');
        }

        const run = await comparison.runComparison(comparisonOptions(mainDir, prDir, outputDir), {
          looksSame: scenario.analyze,
        });
        assert.equal(run.exitCode, 1);
        assert.equal(run.summary.stats.failed, 1);
        assert.equal(run.summary.results[0].failureType, scenario.expectedType);
        if (scenario.expectedType === 'missing') {
          assert.equal(run.summary.results[0].captureValidity, 'infrastructure_failure');
        }
        assert.equal(fs.existsSync(path.join(outputDir, 'summary.json')), true);
      } finally {
        fs.rmSync(root, { force: true, recursive: true });
      }
    });
  }
});

test('diff and fail thresholds use the same strict boundary and summary is written before failure', async (t) => {
  const root = fixtureDirectory(t);
  const mainDir = path.join(root, 'main');
  const prDir = path.join(root, 'pr');
  const outputAtBoundary = path.join(root, 'at-boundary');
  const outputAboveBoundary = path.join(root, 'above-boundary');
  fs.mkdirSync(mainDir);
  fs.mkdirSync(prDir);
  const filename = 'pipelines-10x10.png';
  await Promise.all([writePng(path.join(mainDir, filename)), writePng(path.join(prDir, filename))]);
  writeCaptureManifest(mainDir, [captureResult(filename)], 'base');
  writeCaptureManifest(prDir, [captureResult(filename)], 'head');
  const fivePercentAnalysis = async () => ({
    equal: false,
    differentPixels: 5,
    totalPixels: 100,
    diffBounds: { left: 0, top: 0, right: 1, bottom: 1 },
    diffClusters: [{ left: 0, top: 0, right: 1, bottom: 1 }],
  });

  const atBoundary = await comparison.runComparison(
    comparisonOptions(mainDir, prDir, outputAtBoundary, {
      diffThreshold: 5,
      failThreshold: 5,
      failThresholdRaw: '5',
    }),
    { looksSame: fivePercentAnalysis },
  );
  assert.equal(atBoundary.exitCode, 0);
  assert.equal(atBoundary.summary.results[0].hasVisualDiff, false);
  assert.equal(atBoundary.summary.results[0].diffRegionCount, 0);
  assert.equal(atBoundary.summary.results[0].exceedsFailThreshold, false);

  const aboveBoundary = await comparison.runComparison(
    comparisonOptions(mainDir, prDir, outputAboveBoundary, {
      diffThreshold: 4,
      failThreshold: 4,
      failThresholdRaw: '4',
    }),
    { looksSame: fivePercentAnalysis },
  );
  assert.equal(aboveBoundary.exitCode, 1);
  assert.equal(aboveBoundary.summary.results[0].hasVisualDiff, true);
  assert.equal(aboveBoundary.summary.results[0].diffRegionCount, 1);
  assert.equal(aboveBoundary.summary.results[0].exceedsFailThreshold, true);
  assert.equal(fs.existsSync(path.join(outputAboveBoundary, filename)), true);
  assert.equal(fs.existsSync(path.join(outputAboveBoundary, 'summary.json')), true);
});

test('invalid comparison options still write a failing summary', async (t) => {
  const root = fixtureDirectory(t);
  const outputDir = path.join(root, 'comparison');
  const run = await comparison.runComparison(
    comparisonOptions(path.join(root, 'main'), path.join(root, 'pr'), outputDir, {
      diffThreshold: Number.NaN,
    }),
  );
  assert.equal(run.exitCode, 1);
  assert.equal(run.summary.valid, false);
  assert.match(run.summary.fatalErrors[0], /Invalid diff threshold/);
  assert.equal(fs.existsSync(path.join(outputDir, 'summary.json')), true);

  const tooHigh = await comparison.runComparison(
    comparisonOptions(path.join(root, 'main'), path.join(root, 'pr'), path.join(root, 'too-high'), {
      diffThreshold: 101,
      failThreshold: 101,
      failThresholdRaw: '101',
    }),
  );
  assert.equal(tooHigh.exitCode, 1);
  assert.equal(tooHigh.summary.fatalErrors.length, 2);
  assert.ok(tooHigh.summary.fatalErrors.every((error) => error.startsWith('Invalid')));
});
