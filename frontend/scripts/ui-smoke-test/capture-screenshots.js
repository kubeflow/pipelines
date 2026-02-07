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
const BASE_URL = `http://localhost:${PORT}`;

// Viewport configuration
const VIEWPORT_ENV = process.env.UI_SMOKE_VIEWPORT || '1280x800';
const [viewportWidth, viewportHeight] = VIEWPORT_ENV.split('x').map(Number);
const VIEWPORT = { width: viewportWidth || 1280, height: viewportHeight || 800 };

// Pages to capture - these are the main UI routes (using hash-based routing)
// waitFor: selector to wait for before capturing (indicates page is loaded)
// waitForData: additional selector that indicates data has loaded (optional)
const PAGES = [
  { name: 'pipelines', path: '/#/pipelines', waitFor: '[class*="tableRow"]', waitForData: 'a[href*="pipeline"]' },
  { name: 'experiments', path: '/#/experiments', waitFor: '[class*="tableRow"]', waitForData: 'a[href*="experiment"]' },
  { name: 'runs', path: '/#/runs', waitFor: '[class*="tableRow"]', waitForData: 'a[href*="run"]' },
  { name: 'recurring-runs', path: '/#/recurringruns', waitFor: '[class*="tableRow"]' },
  { name: 'artifacts', path: '/#/artifacts', waitFor: '[class*="tableRow"]' },
  { name: 'executions', path: '/#/executions', waitFor: '[class*="tableRow"]', waitForData: 'a[href*="execution"]' },
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
  console.log(`Viewport: ${VIEWPORT.width}x${VIEWPORT.height}`);
  console.log(`Pages to capture: ${filteredPages.map(p => p.name).join(', ')}`);

  // Ensure output directory exists
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  // Launch browser
  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const context = await browser.newContext({
    viewport: VIEWPORT,
    deviceScaleFactor: 2, // Retina quality
    ignoreHTTPSErrors: true,
  });

  const page = await context.newPage();

  // Capture each page
  const results = [];

  for (const pageConfig of filteredPages) {
    const url = `${BASE_URL}${pageConfig.path}`;
    const filename = `${pageConfig.name}.png`;
    const filepath = path.join(OUTPUT_DIR, filename);

    console.log(`Capturing ${pageConfig.name}: ${url}`);

    try {
      // Navigate to page
      await page.goto(url, {
        waitUntil: 'networkidle',
        timeout: 30000,
      });

      // Wait for specific element if specified
      if (pageConfig.waitFor) {
        try {
          await page.waitForSelector(pageConfig.waitFor, { timeout: 10000 });
        } catch (e) {
          console.log(`  Warning: waitFor selector '${pageConfig.waitFor}' not found, continuing...`);
        }
      }

      // Wait for data to load if specified (indicates actual content, not just skeleton)
      if (pageConfig.waitForData) {
        try {
          await page.waitForSelector(pageConfig.waitForData, { timeout: 10000 });
          console.log(`  Data loaded: found '${pageConfig.waitForData}'`);
        } catch (e) {
          console.log(`  Warning: data selector '${pageConfig.waitForData}' not found, continuing...`);
        }
      }

      // Additional wait for any animations/loading to settle
      await page.waitForTimeout(2000);

      // Take screenshot
      await page.screenshot({
        path: filepath,
        fullPage: false, // Viewport only for consistent comparisons
      });

      console.log(`  ✓ Saved: ${filename}`);
      results.push({ page: pageConfig.name, status: 'success', path: filepath });
    } catch (error) {
      console.log(`  ✗ Failed: ${error.message}`);
      results.push({ page: pageConfig.name, status: 'failed', error: error.message });
    }
  }

  await browser.close();

  // Write results manifest
  const manifestPath = path.join(OUTPUT_DIR, 'manifest.json');
  fs.writeFileSync(manifestPath, JSON.stringify({
    label: LABEL,
    timestamp: new Date().toISOString(),
    baseUrl: BASE_URL,
    viewport: VIEWPORT,
    results,
  }, null, 2));

  console.log(`\nCapture complete. Results saved to ${manifestPath}`);

  // Return exit code based on success rate
  const failedCount = results.filter(r => r.status === 'failed').length;
  if (failedCount === results.length) {
    console.error('All screenshots failed!');
    process.exit(1);
  } else if (failedCount > 0) {
    console.warn(`${failedCount}/${results.length} screenshots failed`);
  }
}

captureScreenshots().catch(err => {
  console.error('Screenshot capture failed:', err);
  process.exit(1);
});
