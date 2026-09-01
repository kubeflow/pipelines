#!/usr/bin/env node
/**
 * Screenshot Capture Script for Kubeflow Pipelines UI
 *
 * Captures screenshots of key UI pages for visual comparison testing.
 * Uses Playwright for browser automation.
 *
 * Usage: node capture-screenshots.js --base-url http://localhost:4001 --output ./screenshots --label main
 */

const crypto = require('crypto');
const path = require('path');
const fs = require('fs');
const {
  SCENARIO_CONTRACT_SCHEMA_VERSION,
  resolveSemanticScenarios,
} = require('./semantic-capture-scenarios.js');
const { COMPARISON_RUN_FIXTURES } = require('./semantic-manifest.js');

const REPO_ROOT = path.resolve(__dirname, '../../..');
const DEFAULT_SEED_MANIFEST = path.join(REPO_ROOT, '.ui-smoke-test', 'seed-manifest.json');
const CAPTURE_MANIFEST_FILENAME = 'manifest.json';
const CAPTURE_MANIFEST_SCHEMA_VERSION = 2;
const CAPTURE_OWNER_FILENAME = '.ui-smoke-capture-managed.json';
const CAPTURE_OWNER_SCHEMA_VERSION = 1;
const CAPTURE_STATUSES = new Set(['success', 'degraded', 'skipped', 'failed']);
const CAPTURE_VALIDITIES = new Set([
  'valid',
  'ui_rendering_failure',
  'api_incompatibility',
  'seed_failure',
  'missing_fixture',
  'selector_drift',
  'expected_product_removal',
  'infrastructure_failure',
]);
const DETERMINISTIC_STYLE_ID = 'ui-smoke-test-deterministic-rendering';
const DETERMINISTIC_FONT_FAMILY = 'UI Smoke Roboto';
const DETERMINISTIC_FONT_PACKAGE = '@fontsource/roboto@5.3.0';
const DETERMINISTIC_FONT_ASSETS = Object.freeze(
  [400, 500, 700].map((weight) => {
    const filename = `roboto-latin-${weight}-normal.woff2`;
    const bytes = fs.readFileSync(require.resolve(`@fontsource/roboto/files/${filename}`));
    return Object.freeze({
      dataUrl: `data:font/woff2;base64,${bytes.toString('base64')}`,
      filename,
      sha256: crypto.createHash('sha256').update(bytes).digest('hex'),
      weight,
    });
  }),
);
const DETERMINISTIC_TIME_ISO = '2030-01-02T03:04:05.000Z';
const DETERMINISTIC_TIME_MS = Date.parse(DETERMINISTIC_TIME_ISO);
const DIAGNOSTIC_LIMIT = 20;
const DIAGNOSTIC_TEXT_LIMIT = 500;
const CAPTURE_ARGUMENT_NAMES = new Set([
  'base-url',
  'label',
  'output',
  'port',
  'revision-role',
  'seed-manifest',
  'semantic-manifest',
  'source-provenance',
]);
const DETERMINISTIC_CSS = `
  ${DETERMINISTIC_FONT_ASSETS.map(
    (asset) => `
      @font-face {
        font-family: "${DETERMINISTIC_FONT_FAMILY}";
        font-style: normal;
        font-weight: ${asset.weight};
        font-display: block;
        src: url("${asset.dataUrl}") format("woff2");
      }
    `,
  ).join('\n')}
  :root {
    color-scheme: light !important;
  }
  *, *::before, *::after {
    animation-delay: 0s !important;
    animation-duration: 0s !important;
    animation-iteration-count: 1 !important;
    caret-color: transparent !important;
    scroll-behavior: auto !important;
    transition-delay: 0s !important;
    transition-duration: 0s !important;
  }
  :where(
    html, body, button, input, select, textarea, div, p, a, dt, dd, h1, h2, h3, h4, h5, h6,
    label, li, table, text, span:not(.material-icons):not(.material-icons-outlined)
  ) {
    font-family: "${DETERMINISTIC_FONT_FAMILY}", sans-serif !important;
    font-synthesis: none !important;
    font-variant-ligatures: none !important;
  }
`;

function getArg(args, name, defaultValue) {
  const index = args.indexOf(`--${name}`);
  if (index === -1) {
    return defaultValue;
  }
  const value = args[index + 1];
  if (value === undefined || value.startsWith('--')) {
    throw new Error(`Missing value for --${name}`);
  }
  return value;
}

function validateCliArguments(args, allowedNames) {
  const seen = new Set();
  for (let index = 0; index < args.length; index += 2) {
    const token = args[index];
    if (typeof token !== 'string' || !token.startsWith('--')) {
      throw new Error(`Unexpected positional argument: ${token}`);
    }
    const name = token.slice(2);
    if (!allowedNames.has(name)) {
      throw new Error(`Unknown argument: ${token}`);
    }
    if (seen.has(name)) {
      throw new Error(`Duplicate argument: ${token}`);
    }
    const value = args[index + 1];
    if (value === undefined || value.startsWith('--')) {
      throw new Error(`Missing value for ${token}`);
    }
    seen.add(name);
  }
  return seen;
}

function parseViewports(value) {
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error('At least one viewport is required.');
  }

  const seen = new Set();
  return value.split(',').map((raw) => {
    const trimmed = raw.trim();
    const match = /^([1-9]\d*)x([1-9]\d*)$/.exec(trimmed);
    if (!match) {
      throw new Error(`Invalid viewport "${trimmed}". Use WIDTHxHEIGHT (for example 1280x800).`);
    }
    const width = Number(match[1]);
    const height = Number(match[2]);
    if (!Number.isSafeInteger(width) || !Number.isSafeInteger(height)) {
      throw new Error(`Invalid viewport "${trimmed}": dimensions must be safe integers.`);
    }
    if (seen.has(trimmed)) {
      throw new Error(`Duplicate viewport "${trimmed}".`);
    }
    seen.add(trimmed);
    return { width, height };
  });
}

function normalizeBaseUrl(value) {
  let parsed;
  try {
    parsed = new URL(value);
  } catch (error) {
    throw new Error(`Invalid base URL "${value}": ${error.message}`);
  }

  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error(`Invalid base URL "${value}": protocol must be http or https.`);
  }

  parsed.hash = '';
  parsed.pathname = parsed.pathname.replace(/\/+$/, '') || '/';
  const normalized = parsed.toString();
  if (parsed.pathname !== '/') {
    return normalized;
  }
  const rootSuffix = `/${parsed.search}`;
  return `${normalized.slice(0, -rootSuffix.length)}${parsed.search}`;
}

function resolveCaptureUrl(baseUrl, routePath) {
  const resolved = new URL(normalizeBaseUrl(baseUrl));
  const hashIndex = routePath.indexOf('#');
  const routeBeforeHash = hashIndex === -1 ? routePath : routePath.slice(0, hashIndex);
  const normalizedRoutePath = routeBeforeHash.startsWith('/')
    ? routeBeforeHash
    : `/${routeBeforeHash}`;
  const basePath = resolved.pathname.replace(/\/+$/, '');
  resolved.pathname = `${basePath}${normalizedRoutePath}` || '/';
  if (hashIndex !== -1) {
    resolved.hash = routePath.slice(hashIndex + 1);
  }
  return resolved.toString();
}

function parseCaptureOptions(args = process.argv.slice(2), env = process.env) {
  const providedArguments = validateCliArguments(args, CAPTURE_ARGUMENT_NAMES);
  if (providedArguments.has('base-url') && providedArguments.has('port')) {
    throw new Error('--base-url and --port are mutually exclusive.');
  }
  const port = getArg(args, 'port', '4001');
  const baseUrl = normalizeBaseUrl(getArg(args, 'base-url', null) || `http://localhost:${port}`);
  const rawViewportEnv = env.UI_SMOKE_VIEWPORTS ?? env.UI_SMOKE_VIEWPORT;
  const viewportValue = rawViewportEnv && rawViewportEnv.trim() ? rawViewportEnv : '1280x800';
  const rawPageNames = env.UI_SMOKE_PAGES;

  const revisionRole = getArg(args, 'revision-role', null);
  if (revisionRole !== null && !['base', 'head', 'current'].includes(revisionRole)) {
    throw new Error('--revision-role must be base, head, or current.');
  }
  const semanticManifestPath = getArg(args, 'semantic-manifest', null);
  const sourceProvenancePath = getArg(args, 'source-provenance', null);
  if ((semanticManifestPath || sourceProvenancePath) && !revisionRole) {
    throw new Error('--revision-role is required with capture provenance inputs.');
  }

  return {
    baseUrl,
    label: getArg(args, 'label', 'screenshot'),
    outputDir: getArg(args, 'output', './screenshots'),
    pageNames:
      rawPageNames && rawPageNames.trim()
        ? rawPageNames
            .split(',')
            .map((name) => name.trim())
            .filter(Boolean)
        : null,
    seedManifestPath: getArg(
      args,
      'seed-manifest',
      env.UI_SMOKE_SEED_MANIFEST || DEFAULT_SEED_MANIFEST,
    ),
    revisionRole,
    semanticManifestPath,
    sourceProvenancePath,
    viewports: parseViewports(viewportValue),
  };
}

function attestJsonInput(filePath, description) {
  if (!filePath) return null;
  const resolvedPath = path.resolve(filePath);
  const stat = fs.lstatSync(resolvedPath);
  if (!stat.isFile() || stat.isSymbolicLink()) {
    throw new Error(`${description} must be a non-symlink regular file.`);
  }
  const contents = fs.readFileSync(resolvedPath);
  let value;
  try {
    value = JSON.parse(contents.toString('utf8'));
  } catch (error) {
    throw new Error(`${description} must contain valid JSON: ${error.message}`);
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${description} must contain a JSON object.`);
  }
  return {
    path: resolvedPath,
    schemaVersion:
      typeof value.schemaVersion === 'string' || typeof value.schemaVersion === 'number'
        ? value.schemaVersion
        : null,
    sha256: crypto.createHash('sha256').update(contents).digest('hex'),
    sizeBytes: contents.length,
  };
}

function loadSeedValues(manifestPath, options = {}) {
  const required = options.required === true;
  if (!manifestPath || !fs.existsSync(manifestPath)) {
    if (required) {
      throw new Error(
        `Seed manifest is required and was not found: ${manifestPath || '(missing path)'}`,
      );
    }
    return null;
  }

  try {
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    const defaults = manifest.defaults || {};
    const resources = manifest.resources || {};
    const runIds = Array.isArray(resources.runIds) ? resources.runIds : [];
    const semanticBindings = manifest.semantic?.bindings || {};
    const semanticResources = semanticBindings.resources || {};
    const semanticRuns = semanticBindings.runs || {};
    const richRun = semanticRuns['run.training-1'] || {};
    const taskId = (semanticKey) => richRun.taskInstances?.[semanticKey]?.[0]?.taskId || null;
    const artifactId = (semanticKey) => richRun.artifacts?.[semanticKey]?.artifactIds?.[0] || null;
    const artifactMemberId = (semanticKey, memberKey) =>
      richRun.artifacts?.[semanticKey]?.members?.[memberKey]?.artifactIds?.[0] || null;
    const semanticComparisonRunIds = COMPARISON_RUN_FIXTURES.map(
      (semanticKey) => semanticResources[semanticKey]?.id,
    ).filter(Boolean);
    const semanticCompareRunlist =
      semanticComparisonRunIds.length === COMPARISON_RUN_FIXTURES.length
        ? semanticComparisonRunIds.join(',')
        : null;

    return {
      artifactId:
        defaults.artifactId ||
        artifactId('artifact.html-report') ||
        artifactId('artifact.scalar-metrics'),
      compareRunlist:
        semanticCompareRunlist || defaults.compareRunlist || runIds.slice(0, 3).join(','),
      consumeMetricsTaskId: taskId('task.consume-metrics'),
      executionId:
        defaults.executionId ||
        richRun.taskInstances?.['task.write-metrics']?.[0]?.mlmdExecutionId ||
        null,
      experimentId: defaults.experimentId || (resources.experimentIds || [])[0],
      htmlArtifactId: artifactId('artifact.html-report'),
      historicalArtifactId: defaults.historicalArtifactId || null,
      markdownArtifactId: artifactId('artifact.markdown-report'),
      nestedDagTaskId: taskId('task.nested-dag'),
      parallelTaskId: taskId('task.parallel-loop'),
      pipelineId: defaults.pipelineId || (resources.pipelineIds || [])[0],
      recurringRunId: defaults.recurringRunId || (resources.recurringRunIds || [])[0],
      relatedArtifactId:
        artifactMemberId('artifact.scalar-metrics', 'metric.accuracy') ||
        artifactId('artifact.scalar-metrics'),
      runId: defaults.runId || runIds[0],
      richRunId:
        semanticResources['run.training-1']?.id || richRun.runId || defaults.runId || runIds[0],
      retryTaskId: taskId('task.retry-once'),
      rocArtifactId: artifactId('artifact.roc-curve'),
      scalarArtifactId: artifactId('artifact.scalar-metrics'),
      taskId: defaults.taskId || taskId('task.write-metrics'),
      writeMetricsTaskId: taskId('task.write-metrics'),
    };
  } catch (error) {
    if (required) {
      throw new Error(`Failed to load seed manifest ${manifestPath}: ${error.message}`, {
        cause: error,
      });
    }
    console.log(`Warning: failed to parse seed manifest ${manifestPath}: ${error.message}`);
    return null;
  }
}

function resolvePathTemplate(routePath, seedValues) {
  const missing = [];
  const resolved = routePath.replace(/\{seed\.([a-zA-Z0-9_]+)\}/g, (_match, key) => {
    const value = seedValues && seedValues[key];
    if (value === undefined || value === null || value === '') {
      missing.push(key);
      return '';
    }
    return String(value);
  });

  if (missing.length > 0) {
    return { missing, resolvedPath: null };
  }

  return { missing: [], resolvedPath: resolved };
}

class SkipCaptureError extends Error {
  constructor(message, captureValidity = 'missing_fixture') {
    super(message);
    this.name = 'SkipCaptureError';
    this.captureValidity = captureValidity;
  }
}

async function executeActions(page, actions) {
  if (!Array.isArray(actions) || actions.length === 0) {
    return;
  }

  for (const action of actions) {
    const timeout = action.timeoutMs || 10000;
    const descriptor = action.selector ? `${action.type}(${action.selector})` : action.type;
    const locator = action.selector ? page.locator(action.selector) : null;
    const target =
      locator && Number.isSafeInteger(action.index) ? locator.nth(action.index) : locator?.first();

    try {
      switch (action.type) {
        case 'click':
          await target.click({ timeout });
          break;
        case 'dispatchClick':
          await page
            .locator(action.selector)
            .first()
            .evaluate((element) => element.click());
          break;
        case 'waitForSelector':
          await page.waitForSelector(action.selector, { timeout });
          break;
        case 'waitForFunction':
          if (typeof action.predicate !== 'function') {
            throw new Error('waitForFunction requires a predicate function');
          }
          await page.waitForFunction(action.predicate, undefined, { timeout });
          break;
        case 'waitForText':
          await page
            .getByText(action.text, { exact: false })
            .nth(Math.max(0, (action.minCount || 1) - 1))
            .waitFor({ timeout });
          break;
        case 'waitForFrameText':
          {
            const deadline = Date.now() + timeout;
            let foundCount = 0;
            const minCount = action.minCount || 1;
            do {
              foundCount = 0;
              for (const frame of page.frames()) {
                foundCount += await frame.getByText(action.text, { exact: false }).count();
              }
              if (foundCount < minCount) await page.waitForTimeout(100);
            } while (foundCount < minCount && Date.now() < deadline);
            if (foundCount < minCount) {
              throw new Error(
                `expected ${minCount} occurrence(s) in frames, found ${foundCount}: ${action.text}`,
              );
            }
          }
          break;
        case 'assertAbsent': {
          const count = await page.locator(action.selector).count();
          if (count !== 0) {
            throw new Error(`expected selector to be absent, found ${count} match(es)`);
          }
          break;
        }
        case 'skipIf': {
          if (typeof action.predicate !== 'function') {
            throw new Error('skipIf requires a predicate function');
          }
          const shouldSkip = await page.evaluate(action.predicate);
          if (shouldSkip) {
            throw new SkipCaptureError(
              action.reason || `Skip condition met: ${descriptor}`,
              action.captureValidity,
            );
          }
          break;
        }
        case 'scrollIntoView':
          await page.locator(action.selector).first().scrollIntoViewIfNeeded({ timeout });
          break;
        case 'hover':
          await target.hover({ timeout });
          break;
        case 'moveMouse':
          await page.mouse.move(action.x || 0, action.y || 0);
          break;
        case 'press':
          await page.keyboard.press(action.key);
          break;
        case 'waitForTimeout':
          await page.waitForTimeout(action.ms || 500);
          break;
        default:
          throw new Error(`Unsupported action type "${action.type}"`);
      }
    } catch (error) {
      if (error instanceof SkipCaptureError) {
        throw error;
      }
      if (action.optional) {
        console.log(`  Warning: optional action failed: ${descriptor}: ${error.message}`);
        continue;
      }
      const actionError = new Error(`Action failed: ${descriptor}: ${error.message}`);
      if (action.failureValidity) actionError.captureValidity = action.failureValidity;
      throw actionError;
    }
  }
}

function comparePageReadyPredicate() {
  const bodyText = document.body.innerText;
  const hasError =
    !!document.querySelector('[role="alert"]') ||
    bodyText.includes('Error: failed loading') ||
    bodyText.includes('An error is preventing');
  const isLoading =
    document.querySelectorAll('[role="circularprogress"], .MuiCircularProgress-root').length > 0;
  const hasRunResults =
    document.querySelectorAll('[data-testid="table-row"]').length >= 2 ||
    document.querySelectorAll('table tbody tr').length > 0;
  return (
    !hasError &&
    !isLoading &&
    hasRunResults &&
    bodyText.includes('Parameters') &&
    bodyText.includes('Scalar Metrics')
  );
}

function scalarMetricsReadyPredicate() {
  const pageText = document.body.innerText;
  if (
    pageText.includes('There are no Scalar Metrics artifacts available on the selected runs.') ||
    pageText.includes('An error is preventing the Scalar Metrics from being displayed.')
  ) {
    return false;
  }
  const metricLabels = new Set(
    Array.from(document.querySelectorAll('table tbody tr > td:first-child[title]')).map((cell) =>
      cell.getAttribute('title'),
    ),
  );
  return metricLabels.has('accuracy') && metricLabels.has('loss');
}

function rocCurveReadyPredicate() {
  const pageText = document.body.innerText;
  if (
    pageText.includes('There are no ROC Curve artifacts available on the selected runs.') ||
    pageText.includes('An error is preventing the ROC Curve from being displayed.')
  ) {
    return false;
  }
  const heading = Array.from(document.querySelectorAll('h3')).find((candidate) =>
    candidate.textContent?.trim().startsWith('ROC Curve:'),
  );
  return (
    !!heading &&
    !heading.textContent.includes('no artifacts') &&
    !!document.querySelector('.recharts-wrapper .recharts-line-curve')
  );
}

// Pages to capture - these are the main UI routes (using hash-based routing)
// waitFor: selector to wait for before capturing (indicates page is loaded)
// waitForData: additional selector that indicates data has loaded (optional)
const PAGES = [
  {
    name: 'pipelines',
    path: '/#/pipelines',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes(
              'No pipelines found. Click "Upload pipeline" to start.',
            )),
      },
    ],
  },
  {
    name: 'pipeline-details-seeded',
    path: '/#/pipelines/details/{seed.pipelineId}',
    waitFor: '#root',
    waitForData: '[role="tab"], .ace_editor',
  },
  {
    name: 'pipeline-details-seeded-sidepanel',
    path: '/#/pipelines/details/{seed.pipelineId}',
    waitFor: '#root',
    waitForData: '[role="tab"], .ace_editor',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () => document.querySelectorAll('.react-flow__node, .graphNode').length > 0,
      },
      { type: 'click', selector: '.react-flow__node:visible, .graphNode:visible' },
      { type: 'waitForSelector', selector: '[aria-label="close"]' },
    ],
  },
  {
    name: 'experiments',
    path: '/#/experiments',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes(
              'No experiments found. Click "Create experiment" to start.',
            )),
      },
    ],
  },
  {
    name: 'runs',
    path: '/#/runs',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes('No available runs found.')),
      },
    ],
  },
  {
    name: 'run-details-seeded',
    path: '/#/runs/details/{seed.runId}',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () => {
          const legacyNodes = document.querySelectorAll('.graphNode');
          if (legacyNodes.length > 0) {
            return true;
          }
          const flowNodes = Array.from(document.querySelectorAll('.react-flow__node'));
          return (
            flowNodes.length > 0 &&
            flowNodes.every((node) => getComputedStyle(node).visibility !== 'hidden')
          );
        },
      },
      { type: 'waitForTimeout', ms: 1000, optional: true },
    ],
  },
  {
    name: 'run-details-seeded-sidepanel',
    path: '/#/runs/details/{seed.runId}',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () => {
          const legacyNodes = document.querySelectorAll('.graphNode');
          if (legacyNodes.length > 0) {
            return true;
          }
          const flowNodes = Array.from(document.querySelectorAll('.react-flow__node'));
          return (
            flowNodes.length > 0 &&
            flowNodes.every((node) => getComputedStyle(node).visibility !== 'hidden')
          );
        },
      },
      {
        type: 'click',
        selector: '[role="tab"]:has-text("Graph"), button:has-text("Graph")',
        optional: true,
      },
      { type: 'click', selector: '.react-flow__node:visible, .graphNode:visible' },
      { type: 'waitForSelector', selector: '[aria-label="close"]' },
      { type: 'waitForTimeout', ms: 750, optional: true },
    ],
  },
  {
    name: 'compare-seeded',
    path: '/#/compare?runlist={seed.compareRunlist}',
    waitFor: '#root',
    actions: [
      { type: 'waitForFunction', predicate: comparePageReadyPredicate },
      { type: 'waitForFunction', predicate: scalarMetricsReadyPredicate },
    ],
  },
  {
    name: 'compare-seeded-roc',
    path: '/#/compare?runlist={seed.compareRunlist}',
    waitFor: '#root',
    actions: [
      { type: 'waitForFunction', predicate: comparePageReadyPredicate },
      {
        type: 'waitForSelector',
        selector: '[role="tab"]:has-text("ROC Curve"), button:has-text("ROC Curve")',
      },
      {
        type: 'click',
        selector: '[role="tab"]:has-text("ROC Curve"), button:has-text("ROC Curve")',
      },
      {
        type: 'waitForFunction',
        predicate: rocCurveReadyPredicate,
      },
      { type: 'scrollIntoView', selector: '.recharts-wrapper, .rv-xy-plot', optional: true },
      { type: 'moveMouse', x: 8, y: 8, optional: true },
      { type: 'waitForTimeout', ms: 250 },
      { type: 'waitForTimeout', ms: 1000, optional: true },
    ],
  },
  { name: 'runs-new', path: '/#/runs/new', waitFor: '#choosePipelineBtn' },
  {
    name: 'runs-new-pipeline-dialog',
    path: '/#/runs/new',
    waitFor: '#choosePipelineBtn',
    actions: [
      { type: 'click', selector: '#choosePipelineBtn' },
      { type: 'waitForSelector', selector: '#pipelineSelectorDialog' },
    ],
  },
  {
    name: 'runs-new-upload-dialog',
    path: '/#/runs/new',
    required: false,
    waitFor: '#choosePipelineBtn',
    actions: [
      { type: 'dispatchClick', selector: '#choosePipelineBtn' },
      { type: 'waitForSelector', selector: '#pipelineSelectorDialog' },
      {
        type: 'skipIf',
        captureValidity: 'expected_product_removal',
        predicate: () => !document.body.innerText.includes('Upload pipeline'),
        reason: 'Current new-run selector does not expose an upload option from this dialog.',
      },
      { type: 'dispatchClick', selector: 'text=Upload pipeline' },
      { type: 'waitForSelector', selector: '#dropZone' },
    ],
  },
  {
    name: 'recurring-runs',
    path: '/#/recurringruns',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes('No available recurring runs found.')),
      },
    ],
  },
  {
    name: 'artifacts',
    path: '/#/artifacts',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          document.querySelectorAll('[class*="tableRow"]').length > 0 ||
          document.body.innerText.includes('No artifacts found.'),
      },
    ],
  },
  {
    name: 'artifact-lineage-from-list',
    path: '/#/artifacts',
    required: false,
    waitFor: '[class*="tableRow"]',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !!document.querySelector('a[href*="#/artifacts/"], a[href*="/artifacts/"]') ||
          document.body.innerText.includes('No artifacts found.'),
      },
      {
        type: 'skipIf',
        predicate: () => document.body.innerText.includes('No artifacts found.'),
        reason: 'Artifact list is empty; cannot open a lineage view from the list page.',
      },
      { type: 'click', selector: 'a[href*="#/artifacts/"], a[href*="/artifacts/"]' },
      {
        type: 'waitForSelector',
        selector: '[role="tab"]:has-text("Lineage Explorer"), button:has-text("Lineage Explorer")',
      },
      {
        type: 'click',
        selector: '[role="tab"]:has-text("Lineage Explorer"), button:has-text("Lineage Explorer")',
      },
      { type: 'waitForTimeout', ms: 1000, optional: true },
    ],
  },
  {
    name: 'executions',
    path: '/#/executions',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes('No executions found.')),
      },
    ],
  },
  {
    name: 'pipeline-create',
    path: '/#/pipeline_versions/new',
    waitFor: '#dropZone',
    waitForData: '#pipelinePackageUrl, [data-testid="uploadFileInput"]',
  },
  { name: 'experiment-create', path: '/#/experiments/new', waitFor: 'input' },
];

const SEMANTICALLY_REPLACED_PAGE_NAMES = new Set([
  'artifact-lineage-from-list',
  'artifacts',
  'compare-seeded',
  'compare-seeded-roc',
  'executions',
  'run-details-seeded',
  'run-details-seeded-sidepanel',
]);
const REVISION_AWARE_PAGE_ALIASES = Object.freeze({
  'artifact-lineage-from-list': 'artifact-related-tasks',
  artifacts: 'artifact-list-evolution',
  'compare-seeded': 'compare-runs',
  'compare-seeded-roc': 'compare-roc-selection',
  executions: 'executions-to-runs',
  'run-details-seeded': 'run-details-rich-graph',
  'run-details-seeded-sidepanel': 'run-details-task-panel',
});

function buildRevisionAwarePages(revisionRole, seedValues, legacyPages = PAGES) {
  const semanticPages = resolveSemanticScenarios(revisionRole, seedValues);
  const retainedPages = legacyPages.filter(
    (page) => !SEMANTICALLY_REPLACED_PAGE_NAMES.has(page.name),
  );
  const names = new Set(semanticPages.map((page) => page.name));
  for (const page of retainedPages) {
    if (names.has(page.name)) {
      throw new Error(`Duplicate capture page name after scenario merge: ${page.name}`);
    }
    names.add(page.name);
  }
  return [...semanticPages, ...retainedPages];
}

function revisionAwarePageNames(pageNames) {
  if (!pageNames) return pageNames;
  return pageNames.map((name) => REVISION_AWARE_PAGE_ALIASES[name] || name);
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function captureFilename(pageName, viewport) {
  return `${pageName}-${viewport.width}x${viewport.height}.png`;
}

function isManagedCaptureFilename(filename) {
  return /^[a-z0-9][a-z0-9-]*-[1-9]\d*x[1-9]\d*\.png$/.test(filename);
}

function validateManagedFilenames(filenames, label) {
  if (!Array.isArray(filenames) || filenames.length > 1000) {
    throw new Error(`${label} must contain an array of at most 1000 capture filenames.`);
  }
  const unique = new Set();
  for (const filename of filenames) {
    if (
      typeof filename !== 'string' ||
      path.basename(filename) !== filename ||
      !isManagedCaptureFilename(filename) ||
      unique.has(filename)
    ) {
      throw new Error(`${label} contains an invalid or duplicate filename: ${filename}`);
    }
    unique.add(filename);
  }
  return [...unique];
}

function cleanCaptureOutputs(outputDir, managedFilenames) {
  fs.mkdirSync(outputDir, { recursive: true });
  const expectedFilenames = validateManagedFilenames(managedFilenames, 'Capture output list');
  const ownerPath = path.join(outputDir, CAPTURE_OWNER_FILENAME);
  const existingEntries = fs.readdirSync(outputDir, { withFileTypes: true });
  const removed = [];

  if (existingEntries.length > 0) {
    let ownerStat;
    try {
      ownerStat = fs.lstatSync(ownerPath);
    } catch (error) {
      if (error.code === 'ENOENT') {
        throw new Error(`Refusing to clean non-empty unowned capture directory: ${outputDir}`);
      }
      throw error;
    }
    if (!ownerStat.isFile() || ownerStat.isSymbolicLink()) {
      throw new Error(`Capture ownership marker must be a regular file: ${ownerPath}`);
    }

    let owner;
    try {
      owner = JSON.parse(fs.readFileSync(ownerPath, 'utf8'));
    } catch (error) {
      throw new Error(`Capture ownership marker is invalid: ${error.message}`);
    }
    if (owner?.schemaVersion !== CAPTURE_OWNER_SCHEMA_VERSION) {
      throw new Error(`Unsupported capture ownership marker in ${ownerPath}.`);
    }
    const previousFilenames = validateManagedFilenames(owner.files, 'Capture ownership marker');
    for (const filename of [...previousFilenames, CAPTURE_MANIFEST_FILENAME]) {
      const target = path.join(outputDir, filename);
      try {
        const stat = fs.lstatSync(target);
        if (!stat.isFile() && !stat.isSymbolicLink()) {
          throw new Error(`Managed capture output is not a file: ${target}`);
        }
        fs.unlinkSync(target);
        removed.push(filename);
      } catch (error) {
        if (error.code !== 'ENOENT') throw error;
      }
    }
  }

  for (const filename of expectedFilenames) {
    if (fs.existsSync(path.join(outputDir, filename))) {
      throw new Error(`Refusing to overwrite an unmanaged capture output: ${filename}`);
    }
  }

  fs.writeFileSync(
    ownerPath,
    `${JSON.stringify(
      {
        schemaVersion: CAPTURE_OWNER_SCHEMA_VERSION,
        files: expectedFilenames,
      },
      null,
      2,
    )}\n`,
    { flag: 'w' },
  );

  return removed;
}

function assertNavigationResponse(response, url) {
  if (!response) {
    throw new Error(`Navigation to ${url} did not return an HTTP response.`);
  }

  const status = typeof response.status === 'function' ? response.status() : Number.NaN;
  const ok = typeof response.ok === 'function' ? response.ok() : status >= 200 && status < 300;
  if (!ok) {
    const statusText =
      typeof response.statusText === 'function' && response.statusText()
        ? ` ${response.statusText()}`
        : '';
    throw new Error(`Navigation to ${url} failed with HTTP ${status}${statusText}.`);
  }
}

async function installDeterministicRendering(page) {
  await page.emulateMedia({ colorScheme: 'light', reducedMotion: 'reduce' });
  await page.addInitScript(
    ({ css, fixedTimeMs, pollingDelayMs, styleId }) => {
      const NativeDate = Date;
      function FrozenDate(...args) {
        if (new.target) {
          return new NativeDate(...(args.length > 0 ? args : [fixedTimeMs]));
        }
        return new NativeDate(fixedTimeMs).toString();
      }
      FrozenDate.prototype = NativeDate.prototype;
      Object.setPrototypeOf(FrozenDate, NativeDate);
      FrozenDate.now = () => fixedTimeMs;
      FrozenDate.parse = NativeDate.parse;
      FrozenDate.UTC = NativeDate.UTC;
      globalThis.Date = FrozenDate;

      const nativeSetTimeout = globalThis.setTimeout.bind(globalThis);
      globalThis.setInterval = () => 0;
      globalThis.setTimeout = (callback, delay = 0, ...args) =>
        Number(delay) >= pollingDelayMs ? 0 : nativeSetTimeout(callback, delay, ...args);

      if (document.getElementById(styleId)) {
        return;
      }
      const install = () => {
        if (document.getElementById(styleId)) {
          return;
        }
        const style = document.createElement('style');
        style.id = styleId;
        style.textContent = css;
        (document.head || document.documentElement).appendChild(style);
      };
      if (document.head || document.documentElement) {
        install();
      } else {
        document.addEventListener('DOMContentLoaded', install, { once: true });
      }
    },
    {
      css: DETERMINISTIC_CSS,
      fixedTimeMs: DETERMINISTIC_TIME_MS,
      pollingDelayMs: 5000,
      styleId: DETERMINISTIC_STYLE_ID,
    },
  );
}

async function ensureDeterministicRendering(page) {
  await page.evaluate(
    ({ css, styleId }) => {
      if (!document.getElementById(styleId)) {
        const style = document.createElement('style');
        style.id = styleId;
        style.textContent = css;
        (document.head || document.documentElement).appendChild(style);
      }
    },
    { css: DETERMINISTIC_CSS, styleId: DETERMINISTIC_STYLE_ID },
  );
}

async function waitForFonts(page) {
  await page.evaluate(async () => {
    if (document.fonts && document.fonts.ready) {
      await document.fonts.ready;
    }
  });
}

async function assertDeterministicFont(page) {
  const status = await page.evaluate((fontFamily) => {
    const available = document.fonts.check(`16px "${fontFamily}"`);
    return {
      available,
      computedBodyFont: getComputedStyle(document.body).fontFamily,
      reason: available ? null : `${fontFamily} did not load from the pinned capture asset`,
    };
  }, DETERMINISTIC_FONT_FAMILY);
  if (status && !status.available) {
    const error = new Error(`Deterministic capture font check failed: ${status.reason}`);
    error.captureValidity = 'infrastructure_failure';
    throw error;
  }
  return status || null;
}

async function stabilizeChildFrames(page) {
  if (typeof page.frames !== 'function') return [];
  const mainFrame = typeof page.mainFrame === 'function' ? page.mainFrame() : null;
  const statuses = [];
  for (const frame of page.frames()) {
    if (frame === mainFrame || typeof frame.addStyleTag !== 'function') continue;
    await frame.addStyleTag({ content: DETERMINISTIC_CSS });
    await waitForFonts(frame);
    statuses.push(await assertDeterministicFont(frame));
    await normalizeDynamicText(frame);
  }
  return statuses;
}

async function normalizeDynamicText(page) {
  await page.evaluate(
    ({ fixedDate, fixedDateTime, fixedDuration }) => {
      const dateTimePatterns = [
        /\b\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z\b/g,
        /\b\d{1,2}\/\d{1,2}\/\d{4}, \d{1,2}:\d{2}:\d{2} [AP]M\b/g,
      ];
      const durationPattern = /(?<![\d:])-?\d+:\d{2}:\d{2}(?![\d:])/g;
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      for (let node = walker.nextNode(); node; node = walker.nextNode()) {
        const parentName = node.parentElement?.tagName;
        if (parentName === 'SCRIPT' || parentName === 'STYLE') continue;
        let value = node.nodeValue || '';
        value = value.replace(dateTimePatterns[0], fixedDateTime);
        value = value.replace(dateTimePatterns[1], fixedDate);
        value = value.replace(durationPattern, fixedDuration);
        node.nodeValue = value;
      }
    },
    {
      fixedDate: '1/2/2030, 3:04:05 AM',
      fixedDateTime: DETERMINISTIC_TIME_ISO,
      fixedDuration: '00:00:42',
    },
  );
}

function summarizeCaptureResults(results, fatalErrors = []) {
  const countStatus = (status) => results.filter((result) => result.status === status).length;
  const requiredResults = results.filter((result) => result.required);
  const optionalResults = results.filter((result) => !result.required);
  const requiredIncomplete = requiredResults.filter((result) => result.status !== 'success').length;
  const captured = countStatus('success') + countStatus('degraded');

  return {
    total: results.length,
    required: requiredResults.length,
    optional: optionalResults.length,
    success: countStatus('success'),
    degraded: countStatus('degraded'),
    skipped: countStatus('skipped'),
    failed: countStatus('failed'),
    requiredIncomplete,
    complete:
      results.length > 0 && captured > 0 && requiredIncomplete === 0 && fatalErrors.length === 0,
  };
}

function selectPages(pageNames, pages = PAGES) {
  if (!pageNames) {
    return { pages, unknownPageNames: [] };
  }

  const requested = new Set(pageNames);
  const selected = pages.filter((page) => requested.has(page.name));
  const known = new Set(pages.map((page) => page.name));
  const unknownPageNames = [...requested].filter((name) => !known.has(name));
  return { pages: selected, unknownPageNames };
}

function isAllowedCaptureNetworkUrl(urlValue, baseUrl) {
  let candidate;
  let base;
  try {
    candidate = new URL(urlValue);
    base = new URL(baseUrl);
  } catch (error) {
    return false;
  }
  if (candidate.protocol === 'data:' || candidate.protocol === 'blob:') return true;
  if (candidate.protocol === 'http:' || candidate.protocol === 'https:') {
    return candidate.origin === base.origin;
  }
  if (candidate.protocol === 'ws:' || candidate.protocol === 'wss:') {
    const expectedProtocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
    return candidate.protocol === expectedProtocol && candidate.host === base.host;
  }
  return false;
}

async function installNetworkIsolation(context, baseUrl) {
  await context.route('**/*', async (route) => {
    if (isAllowedCaptureNetworkUrl(route.request().url(), baseUrl)) {
      await route.continue();
    } else {
      await route.abort('blockedbyclient');
    }
  });
  if (typeof context.routeWebSocket === 'function') {
    await context.routeWebSocket(/^(?:ws|wss):\/\//, (webSocket) => {
      if (isAllowedCaptureNetworkUrl(webSocket.url(), baseUrl)) {
        webSocket.connectToServer();
      }
      // Leaving a cross-origin socket unconnected creates an inert mocked socket in the page.
    });
  }
}

function sanitizeDiagnosticText(value) {
  return String(value || '')
    .replace(/[\u0000-\u001f\u007f]+/g, ' ')
    .replace(/:\/\/([^\s/:@]+):([^\s/@]+)@/g, '://<redacted>:<redacted>@')
    .replace(/\bBearer\s+\S+/gi, 'Bearer <redacted>')
    .replace(/([?&][a-zA-Z0-9_.-]+)=([^&\s]+)/g, '$1=<redacted>')
    .replace(
      /(["']?(?:access_token|api[_-]?key|auth|authorization|cookie|credential|password|secret|set-cookie|token|x-api-key)["']?\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,;}]+)/gi,
      '$1<redacted>',
    )
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, DIAGNOSTIC_TEXT_LIMIT);
}

function sanitizeDiagnosticUrl(value, baseUrl) {
  try {
    const url = new URL(value, baseUrl);
    const base = new URL(baseUrl);
    const queryKeys = [...new Set(url.searchParams.keys())].sort();
    const query =
      queryKeys.length > 0 ? `?${queryKeys.map((key) => `${key}=<redacted>`).join('&')}` : '';
    let hash = url.hash;
    if (hash.startsWith('#/')) {
      const hashUrl = new URL(hash.slice(1), base.origin);
      const hashQueryKeys = [...new Set(hashUrl.searchParams.keys())].sort();
      const hashQuery =
        hashQueryKeys.length > 0
          ? `?${hashQueryKeys.map((key) => `${key}=<redacted>`).join('&')}`
          : '';
      hash = `#${hashUrl.pathname}${hashQuery}`;
    }
    const prefix = url.origin === base.origin ? '' : url.origin;
    return sanitizeDiagnosticText(`${prefix}${url.pathname}${query}${hash}`);
  } catch (_error) {
    return '[invalid URL]';
  }
}

function createPageDiagnostics(page, baseUrl, limit = DIAGNOSTIC_LIMIT) {
  const consoleErrors = [];
  const failedRequests = [];
  const dropped = { consoleErrors: 0, failedRequests: 0 };
  const append = (collection, key, record) => {
    if (collection.length >= limit) {
      dropped[key]++;
      return;
    }
    collection.push(record);
  };
  const isSameOrigin = (urlValue) => {
    try {
      return new URL(urlValue, baseUrl).origin === new URL(baseUrl).origin;
    } catch (_error) {
      return false;
    }
  };

  if (typeof page.on === 'function') {
    page.on('console', (message) => {
      if (typeof message.type === 'function' && message.type() !== 'error') return;
      const location = typeof message.location === 'function' ? message.location() || {} : {};
      append(consoleErrors, 'consoleErrors', {
        kind: 'console',
        column: Number.isSafeInteger(location.columnNumber) ? location.columnNumber : null,
        line: Number.isSafeInteger(location.lineNumber) ? location.lineNumber : null,
        message: sanitizeDiagnosticText(
          typeof message.text === 'function' ? message.text() : String(message),
        ),
        url: location.url ? sanitizeDiagnosticUrl(location.url, baseUrl) : null,
      });
    });
    page.on('pageerror', (error) => {
      append(consoleErrors, 'consoleErrors', {
        kind: 'pageerror',
        column: null,
        line: null,
        message: sanitizeDiagnosticText(error?.message || error),
        url: null,
      });
    });
    page.on('requestfailed', (request) => {
      const requestUrl = request.url?.() || '';
      append(failedRequests, 'failedRequests', {
        error: sanitizeDiagnosticText(request.failure?.()?.errorText || 'request failed'),
        method: sanitizeDiagnosticText(request.method?.() || 'GET'),
        resourceType: sanitizeDiagnosticText(request.resourceType?.() || 'unknown'),
        sameOrigin: isSameOrigin(requestUrl),
        status: null,
        url: sanitizeDiagnosticUrl(requestUrl, baseUrl),
      });
    });
    page.on('response', (response) => {
      const status = Number(response.status?.());
      if (!Number.isFinite(status) || status < 400) return;
      const request = response.request?.();
      const responseUrl = response.url?.() || '';
      append(failedRequests, 'failedRequests', {
        error: sanitizeDiagnosticText(response.statusText?.() || `HTTP ${status}`),
        method: sanitizeDiagnosticText(request?.method?.() || 'GET'),
        resourceType: sanitizeDiagnosticText(request?.resourceType?.() || 'unknown'),
        sameOrigin: isSameOrigin(responseUrl),
        status,
        url: sanitizeDiagnosticUrl(responseUrl, baseUrl),
      });
    });
  }

  return { consoleErrors, failedRequests, dropped };
}

function routeFromUrl(value) {
  const parsed = new URL(value);
  if (parsed.hash.startsWith('#/')) return parsed.hash.slice(1);
  return `${parsed.pathname}${parsed.search}`;
}

function routeMatches(actualRoute, expectedRoute) {
  const actual = new URL(actualRoute, 'http://ui-smoke.invalid');
  const expected = new URL(expectedRoute, 'http://ui-smoke.invalid');
  if (actual.pathname !== expected.pathname) return false;
  for (const [key, value] of expected.searchParams) {
    if (actual.searchParams.get(key) !== value) return false;
  }
  return true;
}

async function assertRouteExpectation(page, routeExpectation) {
  if (!routeExpectation?.path) return null;
  try {
    await page.waitForFunction(
      (expectedRoute) => {
        const actualRoute = location.hash.startsWith('#/')
          ? location.hash.slice(1)
          : `${location.pathname}${location.search}`;
        const actual = new URL(actualRoute, location.origin);
        const expected = new URL(expectedRoute, location.origin);
        if (actual.pathname !== expected.pathname) return false;
        for (const [key, value] of expected.searchParams) {
          if (actual.searchParams.get(key) !== value) return false;
        }
        return true;
      },
      routeExpectation.path,
      { timeout: routeExpectation.timeoutMs || 10000 },
    );
  } catch (cause) {
    const error = new Error(
      `Route expectation failed: expected ${routeExpectation.path}: ${cause.message}`,
    );
    error.captureValidity = 'ui_rendering_failure';
    throw error;
  }
  const resolvedRoute = typeof page.url === 'function' ? routeFromUrl(page.url()) : null;
  if (resolvedRoute && !routeMatches(resolvedRoute, routeExpectation.path)) {
    throw new Error(
      `Route expectation failed: expected ${routeExpectation.path}, resolved ${resolvedRoute}`,
    );
  }
  return resolvedRoute;
}

function classifyCaptureFailure(error, diagnostics) {
  if (CAPTURE_VALIDITIES.has(error?.captureValidity)) return error.captureValidity;
  const message = String(error?.message || error || '');
  if (/missing fixture|missing seed key/i.test(message)) return 'missing_fixture';
  if (/seed|provenance/i.test(message)) return 'seed_failure';
  if (/Navigation .*HTTP|browser|context|page crashed/i.test(message)) {
    return 'infrastructure_failure';
  }
  if (
    diagnostics?.failedRequests?.some(
      (request) =>
        request.sameOrigin &&
        (['fetch', 'xhr', 'websocket'].includes(request.resourceType) ||
          /(?:^|\/)apis?\/|(?:^|\/)ml_metadata\//.test(request.url)) &&
        (request.status === null || request.status >= 400),
    )
  ) {
    return 'api_incompatibility';
  }
  if (/selector|Action failed|waitFor|locator|text/i.test(message)) return 'selector_drift';
  return 'ui_rendering_failure';
}

async function captureScreenshots(options, dependencies = {}) {
  const chromium = dependencies.chromium || require('playwright').chromium;
  const revisionAware = options.revisionRole === 'base' || options.revisionRole === 'head';
  let seedLoadError = null;
  let seedValues = null;
  try {
    seedValues = loadSeedValues(options.seedManifestPath, { required: revisionAware });
  } catch (error) {
    seedLoadError = error;
  }
  const pages =
    options.pages ||
    (revisionAware ? buildRevisionAwarePages(options.revisionRole, seedValues || {}) : PAGES);
  const selectedPageNames =
    options.pages || !revisionAware ? options.pageNames : revisionAwarePageNames(options.pageNames);
  const { pages: filteredPages, unknownPageNames } = selectPages(selectedPageNames, pages);
  const startedAt = new Date().toISOString();
  const captureId = crypto.randomUUID();
  const fatalErrors = [];
  const results = [];
  const completedFilenames = new Set();
  const inputs = {
    revisionRole: options.revisionRole || null,
    seedManifest: null,
    semanticManifest: null,
    sourceProvenance: null,
  };
  let browser;
  let browserVersion = null;

  if (seedLoadError) {
    fatalErrors.push(`Seed manifest is invalid: ${seedLoadError.message}`);
  }

  try {
    if (seedValues && !seedLoadError) {
      inputs.seedManifest = attestJsonInput(options.seedManifestPath, 'Seed manifest');
    }
    inputs.semanticManifest = attestJsonInput(
      options.semanticManifestPath,
      'Semantic fixture manifest',
    );
    inputs.sourceProvenance = attestJsonInput(options.sourceProvenancePath, 'Source provenance');
  } catch (error) {
    fatalErrors.push(`Capture provenance is invalid: ${error.message}`);
  }

  console.log(`Starting screenshot capture for ${options.label}`);
  console.log(`Base URL: ${options.baseUrl}`);
  console.log(`Output directory: ${options.outputDir}`);
  console.log(
    `Viewports: ${options.viewports.map((viewport) => `${viewport.width}x${viewport.height}`).join(', ')}`,
  );
  console.log(
    seedValues
      ? `Seed manifest: ${options.seedManifestPath}`
      : revisionAware
        ? `Seed manifest: invalid (${seedLoadError?.message || 'required input unavailable'})`
        : 'Seed manifest: not found (seeded routes will be skipped)',
  );
  console.log(`Pages to capture: ${filteredPages.map((page) => page.name).join(', ') || '(none)'}`);

  const managedFilenames = filteredPages.flatMap((pageDefinition) =>
    options.viewports.map((viewport) => captureFilename(pageDefinition.name, viewport)),
  );
  cleanCaptureOutputs(options.outputDir, managedFilenames);

  if (unknownPageNames.length > 0) {
    fatalErrors.push(`Unknown page name(s): ${unknownPageNames.join(', ')}`);
  }
  if (filteredPages.length === 0) {
    fatalErrors.push('No pages selected for capture.');
  }

  const addResult = (result) => {
    if (!CAPTURE_STATUSES.has(result.status)) {
      throw new Error(`Invalid capture status: ${result.status}`);
    }
    if (!CAPTURE_VALIDITIES.has(result.captureValidity)) {
      throw new Error(`Invalid capture validity: ${result.captureValidity}`);
    }
    completedFilenames.add(result.filename);
    results.push(result);
  };

  try {
    if (filteredPages.length > 0 && fatalErrors.length === 0) {
      browser = await chromium.launch({
        headless: true,
      });
      browserVersion = typeof browser.version === 'function' ? browser.version() : null;
    }

    for (const viewport of options.viewports) {
      if (!browser) {
        break;
      }
      let context;
      try {
        context = await browser.newContext({
          viewport,
          colorScheme: 'light',
          deviceScaleFactor: 2,
          ignoreHTTPSErrors: true,
          locale: 'en-US',
          reducedMotion: 'reduce',
          serviceWorkers: 'block',
          timezoneId: 'UTC',
        });
        await installNetworkIsolation(context, options.baseUrl);

        for (const pageConfig of filteredPages) {
          const required = pageConfig.required !== false;
          const filename = captureFilename(pageConfig.name, viewport);
          const filepath = path.join(options.outputDir, filename);
          const { resolvedPath, missing } = resolvePathTemplate(pageConfig.path, seedValues);
          const missingFixtures = [...new Set([...(pageConfig.missingFixtures || []), ...missing])];

          if (!resolvedPath || missingFixtures.length > 0) {
            const reason = `missing fixture key(s): ${missingFixtures.join(', ')}`;
            console.log(
              `Skipping ${pageConfig.name} (${viewport.width}x${viewport.height}): ${reason}`,
            );
            addResult({
              captureValidity: 'missing_fixture',
              expectedChange: pageConfig.expectedChange || null,
              filename,
              page: pageConfig.name,
              reason,
              requestedRoute: pageConfig.path,
              required,
              revisionRole: options.revisionRole || null,
              routeExpectation: pageConfig.routeExpectation || null,
              scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
              semanticScenario: pageConfig.semanticScenario || pageConfig.name,
              status: 'skipped',
              viewport,
            });
            continue;
          }

          const url = resolveCaptureUrl(options.baseUrl, resolvedPath);
          console.log(
            `Capturing ${pageConfig.name} (${viewport.width}x${viewport.height}): ${url}`,
          );

          let page;
          let diagnostics = { consoleErrors: [], failedRequests: [], dropped: {} };
          let fontStatus = null;
          let resolvedRoute = null;
          try {
            // Hash-only navigation on a reused page is a same-document navigation and returns no
            // HTTP response. A fresh page guarantees that every route performs a network request
            // whose status can be validated before capture.
            page = await context.newPage();
            diagnostics = createPageDiagnostics(page, options.baseUrl);
            await installDeterministicRendering(page);
            const response = await page.goto(url, {
              waitUntil: 'networkidle',
              timeout: 30000,
            });
            assertNavigationResponse(response, url);
            await ensureDeterministicRendering(page);
            resolvedRoute = await assertRouteExpectation(page, pageConfig.routeExpectation);

            let selectorFailed = false;
            if (pageConfig.waitFor) {
              try {
                await page.waitForSelector(pageConfig.waitFor, { timeout: 10000 });
              } catch (error) {
                console.log(`  Warning: waitFor selector '${pageConfig.waitFor}' not found.`);
                selectorFailed = true;
              }
            }

            await executeActions(page, pageConfig.actions);

            if (pageConfig.waitForData) {
              try {
                await page.waitForSelector(pageConfig.waitForData, { timeout: 10000 });
                console.log(`  Data loaded: found '${pageConfig.waitForData}'`);
              } catch (error) {
                console.log(`  Warning: data selector '${pageConfig.waitForData}' not found.`);
                selectorFailed = true;
              }
            }

            await waitForFonts(page);
            fontStatus = {
              childFrames: await stabilizeChildFrames(page),
              main: await assertDeterministicFont(page),
            };
            await page.waitForTimeout(pageConfig.waitForTimeoutMs || 2000);
            await normalizeDynamicText(page);
            await page.screenshot({
              animations: 'disabled',
              fullPage: false,
              path: filepath,
            });

            const status = selectorFailed ? 'degraded' : 'success';
            const captureValidity = selectorFailed
              ? classifyCaptureFailure(new Error('selector readiness failed'), diagnostics)
              : pageConfig.routeExpectation?.kind === 'expected-removal'
                ? 'expected_product_removal'
                : 'valid';
            const statusIcon = selectorFailed ? '⚠' : '✓';
            const capturedAt = fs.statSync(filepath).mtime.toISOString();
            const sha256 = crypto
              .createHash('sha256')
              .update(fs.readFileSync(filepath))
              .digest('hex');
            console.log(`  ${statusIcon} Saved: ${filename}${selectorFailed ? ' (degraded)' : ''}`);
            addResult({
              captureValidity,
              capturedAt,
              diagnostics,
              expectedChange: pageConfig.expectedChange || null,
              filename,
              font: fontStatus,
              page: pageConfig.name,
              path: filepath,
              requestedRoute: resolvedPath,
              required,
              resolvedRoute:
                resolvedRoute ||
                (typeof page.url === 'function' ? routeFromUrl(page.url()) : undefined),
              revisionRole: options.revisionRole || null,
              routeExpectation: pageConfig.routeExpectation || null,
              scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
              semanticScenario: pageConfig.semanticScenario || pageConfig.name,
              sha256,
              status,
              viewport,
            });
          } catch (error) {
            if (error instanceof SkipCaptureError) {
              console.log(`  ↷ Skipped: ${error.message}`);
              addResult({
                captureValidity: error.captureValidity || 'missing_fixture',
                diagnostics,
                expectedChange: pageConfig.expectedChange || null,
                filename,
                page: pageConfig.name,
                reason: error.message,
                requestedRoute: resolvedPath,
                required,
                resolvedRoute:
                  resolvedRoute ||
                  (typeof page?.url === 'function' ? routeFromUrl(page.url()) : undefined),
                revisionRole: options.revisionRole || null,
                routeExpectation: pageConfig.routeExpectation || null,
                scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
                semanticScenario: pageConfig.semanticScenario || pageConfig.name,
                status: 'skipped',
                viewport,
              });
              continue;
            }
            console.log(`  ✗ Failed: ${error.message}`);
            addResult({
              captureValidity: classifyCaptureFailure(error, diagnostics),
              diagnostics,
              error: error.message,
              expectedChange: pageConfig.expectedChange || null,
              filename,
              page: pageConfig.name,
              requestedRoute: resolvedPath,
              required,
              resolvedRoute:
                resolvedRoute ||
                (typeof page?.url === 'function' ? routeFromUrl(page.url()) : undefined),
              revisionRole: options.revisionRole || null,
              routeExpectation: pageConfig.routeExpectation || null,
              scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
              semanticScenario: pageConfig.semanticScenario || pageConfig.name,
              status: 'failed',
              viewport,
            });
          } finally {
            if (page) {
              await page
                .close()
                .catch((error) => fatalErrors.push(`Failed to close page: ${error.message}`));
            }
          }
        }
      } catch (error) {
        fatalErrors.push(
          `Capture aborted for viewport ${viewport.width}x${viewport.height}: ${error.message}`,
        );
      } finally {
        if (context) {
          await context
            .close()
            .catch((error) =>
              fatalErrors.push(`Failed to close browser context: ${error.message}`),
            );
        }
      }
    }
  } catch (error) {
    fatalErrors.push(`Capture aborted: ${error.message}`);
  } finally {
    if (browser) {
      await browser
        .close()
        .catch((error) => fatalErrors.push(`Failed to close browser: ${error.message}`));
    }
  }

  for (const viewport of options.viewports) {
    for (const pageConfig of filteredPages) {
      const filename = captureFilename(pageConfig.name, viewport);
      if (!completedFilenames.has(filename)) {
        addResult({
          captureValidity: fatalErrors.some((error) => /seed|provenance/i.test(error))
            ? 'seed_failure'
            : 'infrastructure_failure',
          error: fatalErrors.at(-1) || 'Capture did not complete.',
          expectedChange: pageConfig.expectedChange || null,
          filename,
          page: pageConfig.name,
          requestedRoute: pageConfig.path,
          required: pageConfig.required !== false,
          revisionRole: options.revisionRole || null,
          routeExpectation: pageConfig.routeExpectation || null,
          scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
          semanticScenario: pageConfig.semanticScenario || pageConfig.name,
          status: 'failed',
          viewport,
        });
      }
    }
  }

  const completedAt = new Date().toISOString();
  const summary = summarizeCaptureResults(results, fatalErrors);
  const manifest = {
    schemaVersion: CAPTURE_MANIFEST_SCHEMA_VERSION,
    captureId,
    label: options.label,
    startedAt,
    completedAt,
    timestamp: completedAt,
    baseUrl: options.baseUrl,
    browser: {
      engine: 'chromium',
      playwrightVersion: require('playwright/package.json').version,
      version: browserVersion,
    },
    deterministicRendering: {
      animations: 'disabled',
      colorScheme: 'light',
      fixedTime: DETERMINISTIC_TIME_ISO,
      fontFamily: DETERMINISTIC_FONT_FAMILY,
      fontPackage: DETERMINISTIC_FONT_PACKAGE,
      fontPolicy: 'embedded WOFF2 assets required; synthesis and ligatures disabled',
      fonts: DETERMINISTIC_FONT_ASSETS.map(({ filename, sha256, weight }) => ({
        filename,
        sha256,
        weight,
      })),
      locale: 'en-US',
      polling: 'timers at or above 5000ms disabled',
      reducedMotion: 'reduce',
      timezone: 'UTC',
    },
    inputs,
    scenarioContractSchemaVersion:
      pages.some((page) => page.scenarioContractSchemaVersion) && SCENARIO_CONTRACT_SCHEMA_VERSION,
    seedManifestPath: seedValues ? options.seedManifestPath : null,
    viewports: options.viewports,
    results,
    fatalErrors,
    summary,
    complete: summary.complete,
  };
  const manifestPath = path.join(options.outputDir, CAPTURE_MANIFEST_FILENAME);
  fs.writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);

  console.log(`\nCapture complete. Results saved to ${manifestPath}`);
  console.log(
    `Required captures: ${summary.required - summary.requiredIncomplete}/${summary.required} successful`,
  );
  if (summary.degraded > 0 || summary.skipped > 0 || summary.failed > 0) {
    console.warn(
      `Incomplete captures: ${summary.degraded} degraded, ${summary.skipped} skipped, ${summary.failed} failed`,
    );
  }
  for (const error of fatalErrors) {
    console.error(`  ✗ ${error}`);
  }

  return { exitCode: summary.complete ? 0 : 1, manifest, manifestPath };
}

async function main(args = process.argv.slice(2), env = process.env) {
  try {
    const options = parseCaptureOptions(args, env);
    const result = await captureScreenshots(options);
    process.exitCode = result.exitCode;
    return result;
  } catch (error) {
    console.error('Screenshot capture failed:', error.message);
    process.exitCode = 1;
    return { exitCode: 1, error };
  }
}

module.exports = {
  CAPTURE_MANIFEST_SCHEMA_VERSION,
  CAPTURE_OWNER_FILENAME,
  CAPTURE_VALIDITIES,
  DETERMINISTIC_TIME_ISO,
  DETERMINISTIC_FONT_ASSETS,
  PAGES,
  assertDeterministicFont,
  assertRouteExpectation,
  assertNavigationResponse,
  buildRevisionAwarePages,
  captureFilename,
  captureScreenshots,
  classifyCaptureFailure,
  cleanCaptureOutputs,
  comparePageReadyPredicate,
  createPageDiagnostics,
  executeActions,
  installNetworkIsolation,
  isAllowedCaptureNetworkUrl,
  loadSeedValues,
  normalizeDynamicText,
  normalizeBaseUrl,
  parseCaptureOptions,
  parseViewports,
  rocCurveReadyPredicate,
  resolveCaptureUrl,
  resolvePathTemplate,
  revisionAwarePageNames,
  routeFromUrl,
  routeMatches,
  sanitizeDiagnosticText,
  sanitizeDiagnosticUrl,
  scalarMetricsReadyPredicate,
  selectPages,
  stabilizeChildFrames,
  summarizeCaptureResults,
};

if (require.main === module) {
  void main();
}
