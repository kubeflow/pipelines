#!/usr/bin/env node
/**
 * Screenshot Capture Script for Kubeflow Pipelines UI
 *
 * Captures screenshots of key UI pages for visual comparison testing.
 * Uses Playwright for browser automation.
 *
 * Usage: node capture-screenshots.js --port 4001 --output ./screenshots --label main
 */

const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');

// Parse command line arguments
const args = process.argv.slice(2);
const getArg = (name, defaultValue) => {
  const index = args.indexOf(`--${name}`);
  return index !== -1 && args[index + 1] ? args[index + 1] : defaultValue;
};

const PORT = getArg('port', '4001');
const OUTPUT_DIR = getArg('output', './screenshots');
const LABEL = getArg('label', 'screenshot');
const REPO_ROOT = path.resolve(__dirname, '../../..');
const DEFAULT_SEED_MANIFEST = path.join(REPO_ROOT, '.ui-smoke-test', 'seed-manifest.json');
const SEED_MANIFEST_PATH = getArg(
  'seed-manifest',
  process.env.UI_SMOKE_SEED_MANIFEST || DEFAULT_SEED_MANIFEST,
);
const BASE_URL = `http://localhost:${PORT}`;

function parseViewports(value) {
  return value.split(',').map(raw => {
    const trimmed = raw.trim();
    const [width, height] = trimmed.split('x').map(Number);
    if (!width || !height) {
      throw new Error(`Invalid viewport "${trimmed}". Use WIDTHxHEIGHT (for example 1280x800).`);
    }
    return { width, height };
  });
}

const rawViewportEnv = process.env.UI_SMOKE_VIEWPORTS ?? process.env.UI_SMOKE_VIEWPORT;
const VIEWPORTS = parseViewports(
  rawViewportEnv && rawViewportEnv.trim() ? rawViewportEnv : '1280x800',
);

function loadSeedValues(manifestPath) {
  if (!manifestPath || !fs.existsSync(manifestPath)) {
    return null;
  }

  try {
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    const defaults = manifest.defaults || {};
    const resources = manifest.resources || {};
    const runIds = Array.isArray(resources.runIds) ? resources.runIds : [];

    return {
      compareRunlist: defaults.compareRunlist || runIds.slice(0, 3).join(','),
      experimentId: defaults.experimentId || (resources.experimentIds || [])[0],
      pipelineId: defaults.pipelineId || (resources.pipelineIds || [])[0],
      recurringRunId: defaults.recurringRunId || (resources.recurringRunIds || [])[0],
      runId: defaults.runId || runIds[0],
    };
  } catch (error) {
    console.log(`Warning: failed to parse seed manifest ${manifestPath}: ${error.message}`);
    return null;
  }
}

const seedValues = loadSeedValues(SEED_MANIFEST_PATH);

function resolvePathTemplate(routePath) {
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

async function executeActions(page, actions) {
  if (!Array.isArray(actions) || actions.length === 0) {
    return;
  }

  for (const action of actions) {
    const timeout = action.timeoutMs || 10000;
    const descriptor = action.selector ? `${action.type}(${action.selector})` : action.type;

    try {
      switch (action.type) {
        case 'click':
          await page.locator(action.selector).first().click({ timeout });
          break;
        case 'waitForSelector':
          await page.waitForSelector(action.selector, { timeout });
          break;
        case 'scrollIntoView':
          await page.locator(action.selector).first().scrollIntoViewIfNeeded({ timeout });
          break;
        case 'moveMouse':
          await page.mouse.move(action.x || 0, action.y || 0);
          break;
        case 'waitForTimeout':
          await page.waitForTimeout(action.ms || 500);
          break;
        default:
          throw new Error(`Unsupported action type "${action.type}"`);
      }
    } catch (error) {
      if (action.optional) {
        console.log(`  Warning: optional action failed: ${descriptor}: ${error.message}`);
        continue;
      }
      throw new Error(`Action failed: ${descriptor}: ${error.message}`);
    }
  }
}

// Pages to capture - these are the main UI routes (using hash-based routing)
// waitFor: selector to wait for before capturing (indicates page is loaded)
// waitForData: additional selector that indicates data has loaded (optional)
const PAGES = [
  {
    name: 'pipelines',
    path: '/#/pipelines',
    waitFor: '[class*="tableRow"]',
    waitForData: 'a[href*="pipeline"]',
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
      { type: 'click', selector: 'text=flip-coin-op', optional: true },
      { type: 'click', selector: 'text=print-op', optional: true },
      { type: 'click', selector: 'text=exit-handler-1', optional: true },
      { type: 'waitForSelector', selector: '[aria-label="close"]' },
    ],
  },
  {
    name: 'experiments',
    path: '/#/experiments',
    waitFor: '[class*="tableRow"]',
    waitForData: 'a[href*="experiment"]',
  },
  {
    name: 'runs',
    path: '/#/runs',
    waitFor: '[class*="tableRow"]',
    waitForData: 'a[href*="run"]',
  },
  {
    name: 'run-details-seeded',
    path: '/#/runs/details/{seed.runId}',
    waitFor: '#root',
    actions: [
      {
        type: 'click',
        selector: '[role="tab"]:has-text("Visualizations"), button:has-text("Visualizations")',
        optional: true,
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
        type: 'click',
        selector: '[role="tab"]:has-text("Graph"), button:has-text("Graph")',
        optional: true,
      },
      { type: 'click', selector: 'text=flip-coin-op', optional: true },
      { type: 'click', selector: 'text=print-op', optional: true },
      { type: 'click', selector: 'text=exit-handler-1', optional: true },
      { type: 'waitForSelector', selector: '[aria-label="close"]' },
      { type: 'waitForTimeout', ms: 750, optional: true },
    ],
  },
  { name: 'compare-seeded', path: '/#/compare?runlist={seed.compareRunlist}', waitFor: '#root' },
  {
    name: 'compare-seeded-roc',
    path: '/#/compare?runlist={seed.compareRunlist}',
    waitFor: '#root',
    actions: [
      {
        type: 'click',
        selector: '[role="tab"]:has-text("ROC Curve"), button:has-text("ROC Curve")',
        optional: true,
      },
      { type: 'waitForSelector', selector: '.recharts-wrapper, .rv-xy-plot' },
      { type: 'scrollIntoView', selector: '.recharts-wrapper, .rv-xy-plot' },
      { type: 'moveMouse', x: 8, y: 8 },
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
    waitFor: '#choosePipelineBtn',
    actions: [
      { type: 'click', selector: '#choosePipelineBtn' },
      { type: 'waitForSelector', selector: '#pipelineSelectorDialog' },
      { type: 'click', selector: 'button:has-text("Upload pipeline")' },
      { type: 'waitForSelector', selector: '#dropZone' },
    ],
  },
  { name: 'recurring-runs', path: '/#/recurringruns', waitFor: '[class*="tableRow"]' },
  { name: 'artifacts', path: '/#/artifacts', waitFor: '[class*="tableRow"]' },
  {
    name: 'artifact-lineage-from-list',
    path: '/#/artifacts',
    waitFor: '[class*="tableRow"]',
    waitForData: 'a[href*="#/artifacts/"], a[href*="/artifacts/"]',
    actions: [
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
    waitFor: '[class*="tableRow"]',
    waitForData: 'a[href*="execution"]',
  },
  { name: 'pipeline-create', path: '/#/pipeline/create', waitFor: 'input' },
  { name: 'experiment-create', path: '/#/experiments/new', waitFor: 'input' },
];

// Filter pages if UI_SMOKE_PAGES env var is set
const envPages = process.env.UI_SMOKE_PAGES;
const filteredPages = envPages
  ? PAGES.filter(p => envPages.split(',').includes(p.name))
  : PAGES;

async function captureScreenshots() {
  console.log(`Starting screenshot capture for ${LABEL}`);
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Output directory: ${OUTPUT_DIR}`);
  console.log(`Viewports: ${VIEWPORTS.map(v => `${v.width}x${v.height}`).join(', ')}`);
  if (seedValues) {
    console.log(`Seed manifest: ${SEED_MANIFEST_PATH}`);
  } else {
    console.log('Seed manifest: not found (seeded routes will be skipped)');
  }
  console.log(`Pages to capture: ${filteredPages.map(p => p.name).join(', ')}`);

  // Ensure output directory exists
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  // Capture each page
  const results = [];

  for (const viewport of VIEWPORTS) {
    const context = await browser.newContext({
      viewport,
      deviceScaleFactor: 2, // Retina quality
      ignoreHTTPSErrors: true,
    });
    const page = await context.newPage();

    for (const pageConfig of filteredPages) {
      const { resolvedPath, missing } = resolvePathTemplate(pageConfig.path);
      if (!resolvedPath) {
        console.log(
          `Skipping ${pageConfig.name} (${viewport.width}x${viewport.height}): missing seed key(s): ${missing.join(', ')}`,
        );
        results.push({
          page: pageConfig.name,
          reason: `missing seed key(s): ${missing.join(', ')}`,
          status: 'skipped',
          viewport,
        });
        continue;
      }

      const url = `${BASE_URL}${resolvedPath}`;
      const filename = `${pageConfig.name}-${viewport.width}x${viewport.height}.png`;
      const filepath = path.join(OUTPUT_DIR, filename);

      console.log(`Capturing ${pageConfig.name} (${viewport.width}x${viewport.height}): ${url}`);

      try {
        // Navigate to page
        await page.goto(url, {
          waitUntil: 'networkidle',
          timeout: 30000,
        });

        // Track whether selectors failed so we can mark as degraded
        let selectorFailed = false;

        // Wait for specific element if specified
        if (pageConfig.waitFor) {
          try {
            await page.waitForSelector(pageConfig.waitFor, { timeout: 10000 });
          } catch (e) {
            console.log(`  Warning: waitFor selector '${pageConfig.waitFor}' not found, continuing...`);
            selectorFailed = true;
          }
        }

        await executeActions(page, pageConfig.actions);

        // Wait for data to load if specified (indicates actual content, not just skeleton)
        if (pageConfig.waitForData) {
          try {
            await page.waitForSelector(pageConfig.waitForData, { timeout: 10000 });
            console.log(`  Data loaded: found '${pageConfig.waitForData}'`);
          } catch (e) {
            console.log(`  Warning: data selector '${pageConfig.waitForData}' not found, continuing...`);
            selectorFailed = true;
          }
        }

        // Additional wait for any animations/loading to settle
        await page.waitForTimeout(pageConfig.waitForTimeoutMs || 2000);

        // Take screenshot
        await page.screenshot({
          path: filepath,
          fullPage: false, // Viewport only for consistent comparisons
        });

        const status = selectorFailed ? 'degraded' : 'success';
        const statusIcon = selectorFailed ? '⚠' : '✓';
        console.log(`  ${statusIcon} Saved: ${filename}${selectorFailed ? ' (degraded)' : ''}`);
        results.push({ page: pageConfig.name, status, path: filepath, viewport });
      } catch (error) {
        console.log(`  ✗ Failed: ${error.message}`);
        results.push({
          page: pageConfig.name,
          status: 'failed',
          error: error.message,
          viewport,
        });
      }
    }

    await page.close();
    await context.close();
  }

  await browser.close();

  // Write results manifest
  const manifestPath = path.join(OUTPUT_DIR, 'manifest.json');
  fs.writeFileSync(manifestPath, JSON.stringify({
    label: LABEL,
    timestamp: new Date().toISOString(),
    baseUrl: BASE_URL,
    seedManifestPath: seedValues ? SEED_MANIFEST_PATH : null,
    viewports: VIEWPORTS,
    results,
  }, null, 2));

  console.log(`\nCapture complete. Results saved to ${manifestPath}`);

  // Return exit code based on success rate
  const failedCount = results.filter(r => r.status === 'failed').length;
  const degradedCount = results.filter(r => r.status === 'degraded').length;
  const capturedCount = results.filter(r => r.status === 'success' || r.status === 'degraded').length;
  if (capturedCount === 0) {
    console.error('All screenshots failed!');
    process.exit(1);
  }
  if (degradedCount > 0) {
    console.warn(`${degradedCount}/${results.length} screenshots degraded (selectors not found)`);
  }
  if (failedCount > 0) {
    console.warn(`${failedCount}/${results.length} screenshots failed`);
  }
}

captureScreenshots().catch(err => {
  console.error('Screenshot capture failed:', err);
  process.exit(1);
});
