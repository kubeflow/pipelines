import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseArgs } from 'node:util';
import { chromium } from 'playwright';
import pixelmatch from 'pixelmatch';
import { PNG } from 'pngjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const defaultRoutesPath = path.join(__dirname, 'visual-compare.routes.json');

export function assertNodeVersion(version = process.versions.node) {
  const [major, minor] = version.split('.').map(Number);
  if (
    !Number.isInteger(major) ||
    !Number.isInteger(minor) ||
    major < 24 ||
    (major === 24 && minor < 2)
  ) {
    throw new Error(
      `visual-compare requires Node >=24.2.0; found ${version}. Use frontend/.nvmrc.`,
    );
  }
}
// Run even when older Node versions do not provide import.meta.main.
assertNodeVersion();

const resolvePath = (filename) => path.resolve(filename);

function validateCaptureResults(results, label) {
  if (!Array.isArray(results) || !results.length) {
    throw new Error(`${label} must contain results; rerun capture to regenerate the manifest.`);
  }
  const names = new Set();
  for (const [index, result] of results.entries()) {
    const field = `results[${index}]`;
    let reason;
    if (!result || typeof result !== 'object' || Array.isArray(result)) {
      reason = `${field} must be a capture object`;
    } else if (typeof result.filePath !== 'string' || !result.filePath.trim()) {
      reason = `${field}.filePath must be a nonempty string; supply an absolute screenshot path or a path relative to the manifest directory`;
    } else {
      const filename = path.basename(result.filePath);
      if (!filename.endsWith('.png')) {
        reason = `filePath must name a PNG file (${field}.filePath)`;
      } else if (names.has(filename.toLowerCase())) {
        reason = `duplicate filename ${filename} (${field}.filePath); use unique PNG filenames`;
      } else if (!['ok', 'error'].includes(result.status)) {
        reason = `status must be ok or error (${field}.status)`;
      }
      names.add(filename.toLowerCase());
    }
    if (reason) {
      throw new Error(
        `${label}, entry ${index + 1}: ${reason}. Fix the manifest or rerun capture.`,
      );
    }
  }
  return results;
}

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function toSlug(input) {
  return input.replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-+|-+$/g, '');
}

export function parseViewports(value) {
  const seen = new Set();
  const entries =
    typeof value === 'string'
      ? value.split(',').map((raw) => {
          const entry = raw.trim();
          if (!/^\d+x\d+$/.test(entry))
            throw new Error(`Invalid viewport "${entry}". Use WIDTHxHEIGHT, e.g. 1280x720.`);
          const [width, height] = entry.split('x').map(Number);
          return { width, height };
        })
      : value;
  if (!Array.isArray(entries) || !entries.length) throw new Error('Specify at least one viewport.');
  return entries.map(({ width, height }) => {
    if (
      !Number.isSafeInteger(width) ||
      !Number.isSafeInteger(height) ||
      width <= 0 ||
      height <= 0
    ) {
      throw new Error('Viewport width and height must be positive integers.');
    }
    const key = `${width}x${height}`;
    if (seen.has(key)) throw new Error(`Duplicate viewport "${key}". Specify each viewport once.`);
    seen.add(key);
    return { width, height };
  });
}

function buildUrl(baseUrl, routePath) {
  const base = baseUrl.replace(/\/$/, '');
  return `${base}/#${routePath}`;
}

export function loadRoutes(routesPath) {
  const routes = JSON.parse(fs.readFileSync(routesPath, 'utf8'));
  if (!Array.isArray(routes) || routes.length === 0) {
    throw new Error(`Routes file must be a nonempty array: ${routesPath}`);
  }
  const names = new Set();
  return routes.map((route) => {
    if (!route || typeof route.path !== 'string' || !route.path.startsWith('/')) {
      throw new Error('Each capture route must have a path beginning with /.');
    }
    const allowedKeys = [
      'name',
      'path',
      'waitForSelector',
      'waitForTimeoutMs',
      'waitForSelectors',
      'fillFields',
      'fitGraph',
      'failOnRequestErrors',
      'failOnPageErrors',
    ];
    const unknown = Object.keys(route).filter((key) => !allowedKeys.includes(key));
    if (unknown.length)
      throw new Error(`Unknown route keys: ${unknown.join(', ')}. Check the capture route schema.`);
    const name = route.name || route.path;
    const slug = typeof name === 'string' ? toSlug(name) : '';
    if (!slug || names.has(slug.toLowerCase())) {
      throw new Error(`Capture route names must produce unique filenames: ${name}`);
    }
    names.add(slug.toLowerCase());
    if (
      route.waitForSelector !== undefined &&
      (typeof route.waitForSelector !== 'string' || !route.waitForSelector.trim())
    ) {
      throw new Error(`waitForSelector must be a nonempty string: ${name}`);
    }
    if (
      route.waitForTimeoutMs !== undefined &&
      (typeof route.waitForTimeoutMs !== 'number' ||
        !Number.isFinite(route.waitForTimeoutMs) ||
        route.waitForTimeoutMs < 0)
    ) {
      throw new Error(`waitForTimeoutMs must be a finite nonnegative number: ${name}`);
    }
    for (const key of ['waitForSelectors', 'failOnRequestErrors']) {
      if (
        route[key] !== undefined &&
        (!Array.isArray(route[key]) ||
          route[key].some((value) => typeof value !== 'string' || !value))
      ) {
        throw new Error(`${key} must be an array of nonempty strings: ${name}`);
      }
    }
    if (route.failOnRequestErrors?.some((prefix) => !prefix.startsWith('/'))) {
      throw new Error(`failOnRequestErrors prefixes must begin with /: ${name}`);
    }
    if (
      route.fillFields !== undefined &&
      (!Array.isArray(route.fillFields) ||
        route.fillFields.some(
          (field) =>
            !field ||
            typeof field.label !== 'string' ||
            !field.label ||
            typeof field.value !== 'string',
        ))
    ) {
      throw new Error(`fillFields must contain textbox labels and string values: ${name}`);
    }
    if (route.failOnPageErrors !== undefined && typeof route.failOnPageErrors !== 'boolean') {
      throw new Error(`failOnPageErrors must be a boolean: ${name}`);
    }
    if (route.fitGraph !== undefined && typeof route.fitGraph !== 'boolean') {
      throw new Error(`fitGraph must be a boolean: ${name}`);
    }
    return {
      name,
      path: route.path,
      waitForSelector: route.waitForSelector,
      waitForTimeoutMs: route.waitForTimeoutMs,
      waitForSelectors: route.waitForSelectors,
      fillFields: route.fillFields,
      fitGraph: route.fitGraph,
      failOnRequestErrors: route.failOnRequestErrors,
      failOnPageErrors: route.failOnPageErrors,
    };
  });
}

// Match the cancellation signals used by the pinned Playwright engine adapters.
function isCancelledRequest(errorText, engine) {
  if (engine === 'firefox') return errorText === 'NS_BINDING_ABORTED';
  if (engine === 'webkit') return errorText?.includes('cancelled') === true;
  return errorText === 'net::ERR_ABORTED';
}

function observeCaptureErrors(page, route, baseUrl, pageErrors, engine) {
  // Error checks are opt-in per route: some deployment pages intentionally probe
  // endpoints that can return 404, and error-state captures need those responses.
  if (route.failOnPageErrors) {
    page.on('pageerror', (error) => pageErrors.push(`Page error: ${error.message}`));
  }
  if (route.failOnRequestErrors?.length) {
    const deploymentUrl = new URL(baseUrl);
    const deploymentPath = deploymentUrl.pathname.replace(/\/+$/, '');
    const checkedPath = (requestUrl) => {
      let parsed;
      try {
        parsed = new URL(requestUrl);
      } catch {
        pageErrors.push(`Invalid request URL: ${String(requestUrl)}`);
        return undefined;
      }
      if (
        parsed.origin !== deploymentUrl.origin ||
        !parsed.pathname.startsWith(`${deploymentPath}/`)
      ) {
        return undefined;
      }
      const relativePath = parsed.pathname.slice(deploymentPath.length);
      return route.failOnRequestErrors.some((prefix) => relativePath.startsWith(prefix))
        ? parsed.pathname
        : undefined;
    };
    page.on('response', (response) => {
      const pathname = checkedPath(response.url());
      if (pathname && response.status() >= 400) {
        pageErrors.push(`HTTP ${response.status()}: ${pathname}`);
      }
    });
    page.on('requestfailed', (request) => {
      const pathname = checkedPath(request.url());
      // Ignore engine-specific cancellation signals, not arbitrary transport errors.
      // Loaded-data selectors still have to pass; transport and HTTP errors remain fatal.
      if (pathname && !isCancelledRequest(request.failure()?.errorText, engine)) {
        pageErrors.push(
          `Request failed: ${pathname}: ${request.failure()?.errorText || 'unknown error'}`,
        );
      }
    });
  }
}

// A manifest is required so stray PNG files never become successful captures.
export function captureInventory(directory, onCaptureError) {
  const manifest = path.join(directory, 'capture-results.json');
  if (!fs.existsSync(manifest))
    throw new Error(`Missing capture manifest: ${manifest}. Run capture before comparison.`);
  const { results, error } = JSON.parse(fs.readFileSync(manifest, 'utf8'));
  try {
    validateCaptureResults(results, manifest);
  } catch (validationError) {
    throw new Error([error, validationError.message].filter(Boolean).join('; '));
  }
  if (error) {
    const message = `Capture failed: ${error}. Rerun capture successfully.`;
    if (!onCaptureError) throw new Error(message);
    onCaptureError(message);
  }
  const inventory = new Map();
  for (const result of results) {
    const filename = path.basename(result.filePath);
    inventory.set(filename, {
      status: result.status,
      filePath: path.resolve(directory, result.filePath),
    });
  }
  return inventory;
}

export async function captureScreenshots({
  baseUrl,
  outDir,
  routesPath,
  viewports,
  defaultWaitFor,
  defaultWaitMs,
  fullPage,
  fixedTime,
  browserType = chromium,
}) {
  ensureDir(outDir);
  const manifestPath = path.join(outDir, 'capture-results.json');
  const results = [];
  const manifest = { error: 'Capture did not complete', results };
  const writeManifest = () =>
    fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + '\n');
  // Invalidate previous success before validation or launch can fail.
  writeManifest();
  let parsedBaseUrl;
  try {
    parsedBaseUrl = new URL(baseUrl);
  } catch {
    throw new Error(
      'base-url must be an absolute HTTP or HTTPS URL. Supply the running frontend URL.',
    );
  }
  if (
    !['http:', 'https:'].includes(parsedBaseUrl.protocol) ||
    parsedBaseUrl.search ||
    parsedBaseUrl.hash
  ) {
    throw new Error(
      'base-url must be an HTTP or HTTPS URL without a query or fragment. Supply the frontend base path.',
    );
  }
  if (typeof defaultWaitMs !== 'number' || !Number.isFinite(defaultWaitMs) || defaultWaitMs < 0) {
    throw new Error(
      'wait-ms must be a finite nonnegative number. Use 0 to disable the settle wait.',
    );
  }
  viewports = parseViewports(viewports);
  const routes = loadRoutes(routesPath);
  if (fixedTime) {
    const validShape = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/.test(fixedTime);
    const timestamp = Date.parse(fixedTime);
    const normalized = fixedTime.length === 20 ? fixedTime.slice(0, -1) + '.000Z' : fixedTime;
    if (
      !validShape ||
      Number.isNaN(timestamp) ||
      new Date(timestamp).toISOString() !== normalized
    ) {
      throw new Error(
        `Invalid fixed time: ${fixedTime}. Expected YYYY-MM-DDTHH:MM:SS[.mmm]Z in UTC.`,
      );
    }
  }
  const contextOptions = {
    locale: 'en-US',
    timezoneId: 'UTC',
    colorScheme: 'light',
    reducedMotion: 'reduce',
  };

  Object.assign(manifest, {
    baseUrl,
    ...contextOptions,
    fixedTime,
    fullPage,
  });
  const browser = await browserType.launch();

  let captureFailure;
  try {
    for (const viewport of viewports) {
      for (const route of routes) {
        const {
          name,
          path: routePath,
          waitForSelector: routeSelector,
          waitForTimeoutMs: routeWait,
          ...setup
        } = route;
        const url = buildUrl(baseUrl, routePath);
        const fileName = `${toSlug(name)}-${viewport.width}x${viewport.height}.png`;
        const filePath = path.join(outDir, fileName);
        const waitForSelector = routeSelector || defaultWaitFor;
        const waitForTimeoutMs = routeWait ?? defaultWaitMs;
        const pageErrors = [];
        let page;
        let captureError;
        const assertNoPageErrors = () => {
          if (pageErrors.length) throw new Error(pageErrors.join('; '));
        };

        // A failed recapture must never leave a previous screenshot looking current.
        fs.rmSync(filePath, { force: true });
        try {
          // Isolate storage, query caches, and route state between captures.
          page = await browser.newPage({ viewport, ...contextOptions });
          observeCaptureErrors(
            page,
            route,
            baseUrl,
            pageErrors,
            browserType.name?.() || 'chromium',
          );
          if (fixedTime) {
            await page.clock.setFixedTime(new Date(fixedTime));
          }
          await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 60000 });
          await page.addStyleTag({
            content: '*{animation:none !important; transition:none !important;}',
          });
          if (waitForSelector) {
            await page.waitForSelector(waitForSelector, { timeout: 60000 });
          }
          for (const selector of route.waitForSelectors || []) {
            await page.waitForSelector(selector, { timeout: 60000 });
          }
          for (const field of route.fillFields || []) {
            await page.getByRole('textbox', { name: field.label, exact: true }).fill(field.value);
          }
          await page.evaluate(() => document.fonts.ready);
          if (route.fitGraph) {
            // React Flow hides nodes until its ResizeObserver has measured them. Fit
            // through the existing control after that measurement and font loading,
            // rather than preserving an onInit viewport based on transient dimensions.
            await page.waitForFunction(
              () => {
                const nodes = [
                  ...document.querySelectorAll('[data-testid="DagCanvas"] .react-flow__node'),
                ];
                return (
                  nodes.length > 0 &&
                  nodes.every(
                    (node) =>
                      node.offsetWidth > 0 &&
                      node.offsetHeight > 0 &&
                      getComputedStyle(node).visibility !== 'hidden',
                  )
                );
              },
              undefined,
              { timeout: 60000 },
            );
            await page.locator('[data-testid="DagCanvas"] .react-flow__controls-fitview').click();
          }
          if (waitForTimeoutMs) {
            await page.waitForTimeout(waitForTimeoutMs);
          }
          assertNoPageErrors();
          await page.screenshot({ path: filePath, fullPage, animations: 'disabled' });
        } catch (error) {
          captureError = error;
        } finally {
          try {
            // Finalize only after closure: protocol events arriving during screenshot
            // completion or page shutdown still belong to this capture.
            await page?.close();
            assertNoPageErrors();
          } catch (error) {
            captureError ??= error;
          }
        }
        const result = {
          name,
          route: routePath,
          viewport,
          waitForSelector,
          waitForTimeoutMs,
          ...setup,
          filePath: fileName,
          status: captureError ? 'error' : 'ok',
        };
        if (captureError) {
          fs.rmSync(filePath, { force: true });
          result.error = String(captureError);
          // eslint-disable-next-line no-console
          console.error(`Failed to capture ${url}: ${captureError}`);
        } else {
          // eslint-disable-next-line no-console
          console.log(`Captured ${url} -> ${filePath}`);
        }
        results.push(result);
      }
    }
    const failures = results.filter((result) => result.status === 'error');
    if (failures.length) {
      throw new Error(`${failures.length} screenshot capture(s) failed`, {
        cause: new AggregateError(failures.map((result) => new Error(result.error))),
      });
    }
  } catch (error) {
    captureFailure = error;
  }
  // Always attempt both cleanup and diagnostics without losing the original failure.
  const errors = captureFailure ? [captureFailure] : [];
  try {
    await browser.close();
  } catch (error) {
    errors.push(error);
  }
  try {
    manifest.browser = browser.version();
    if (errors.length) manifest.error = errors.map((error) => error.message).join('; ');
    else delete manifest.error;
    writeManifest();
  } catch (error) {
    errors.push(error);
  }
  if (errors.length === 1) throw errors[0];
  if (errors.length > 1)
    throw new AggregateError(errors, errors.map((error) => error.message).join('; '));
  return results;
}

function compareImages(baselinePath, currentPath, diffPath) {
  const baseline = PNG.sync.read(fs.readFileSync(baselinePath));
  const current = PNG.sync.read(fs.readFileSync(currentPath));
  if (baseline.width !== current.width || baseline.height !== current.height) {
    throw new Error(
      `Size mismatch for ${path.basename(baselinePath)} (${baseline.width}x${baseline.height} vs ${current.width}x${current.height})`,
    );
  }
  const diff = new PNG({ width: baseline.width, height: baseline.height });
  const mismatchedPixels = pixelmatch(
    baseline.data,
    current.data,
    diff.data,
    baseline.width,
    baseline.height,
    { threshold: 0.1 },
  );
  fs.writeFileSync(diffPath, PNG.sync.write(diff));
  return { mismatchedPixels, baseline, current, diff };
}

function writeSideBySide({ baseline, current, diff, outPath, includeDiff }) {
  const width = baseline.width + current.width + (includeDiff && diff ? diff.width : 0);
  const height = Math.max(baseline.height, current.height, includeDiff && diff ? diff.height : 0);
  const combined = new PNG({ width, height });
  combined.data.fill(255);
  PNG.bitblt(baseline, combined, 0, 0, baseline.width, baseline.height, 0, 0);
  PNG.bitblt(current, combined, 0, 0, current.width, current.height, baseline.width, 0);
  if (includeDiff && diff) {
    PNG.bitblt(diff, combined, 0, 0, diff.width, diff.height, baseline.width + current.width, 0);
  }
  fs.writeFileSync(outPath, PNG.sync.write(combined));
}

function escapeHtml(value) {
  return String(value).replace(
    /[&<>"']/g,
    (character) =>
      ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;',
      })[character],
  );
}

function writeReport({ reportPath, baselineDir, currentDir, diffDir, results }) {
  const rows = results
    .map((result) => {
      if (result.error) {
        return `<tr class="error"><td>${escapeHtml(result.name)}</td><td colspan="4">${escapeHtml(result.error)}</td></tr>`;
      }
      if (result.added)
        return `<tr class="added"><td>${escapeHtml(result.name)}</td><td colspan="4">Added capture; no baseline comparison.</td></tr>`;
      const baselineRel = path.relative(path.dirname(reportPath), result.baselinePath);
      const currentRel = path.relative(path.dirname(reportPath), result.currentPath);
      const diffRel = path.relative(path.dirname(reportPath), result.diffPath);
      const status = result.mismatchedPixels > 0 ? 'diff' : 'match';
      return `
        <tr class="${status}">
          <td>${escapeHtml(result.name)}</td>
          <td>${result.mismatchedPixels}</td>
          <td><img src="${escapeHtml(baselineRel)}" alt="baseline ${escapeHtml(result.name)}"></td>
          <td><img src="${escapeHtml(currentRel)}" alt="current ${escapeHtml(result.name)}"></td>
          <td><img src="${escapeHtml(diffRel)}" alt="diff ${escapeHtml(result.name)}"></td>
        </tr>
      `;
    })
    .join('');

  const html = `
    <!doctype html>
    <html lang="en">
      <head>
        <meta charset="utf-8" />
        <title>Visual Diff Report</title>
        <style>
          body { font-family: Arial, sans-serif; padding: 16px; }
          table { border-collapse: collapse; width: 100%; }
          th, td { border: 1px solid #ddd; padding: 8px; vertical-align: top; }
          th { background: #f5f5f5; }
          img { max-width: 320px; border: 1px solid #eee; }
          tr.match { background: #f7fff7; }
          tr.diff { background: #fff7f0; }
          tr.error { background: #fff0f0; }
        </style>
      </head>
      <body>
        <h1>Visual Diff Report</h1>
        <p>Baseline: ${escapeHtml(baselineDir)}</p>
        <p>Current: ${escapeHtml(currentDir)}</p>
        <p>Diffs: ${escapeHtml(diffDir)}</p>
        <table>
          <thead>
            <tr>
              <th>Route</th>
              <th>Mismatched Pixels</th>
              <th>Baseline</th>
              <th>Current</th>
              <th>Diff</th>
            </tr>
          </thead>
          <tbody>
            ${rows}
          </tbody>
        </table>
      </body>
    </html>
  `;

  fs.writeFileSync(reportPath, html);
}

export function parseCliOptions(args) {
  const { positionals, values } = parseArgs({
    args,
    options: {
      'base-url': { type: 'string', default: 'http://localhost:3000' },
      routes: { type: 'string', default: defaultRoutesPath },
      'out-dir': { type: 'string', default: '.visual/current' },
      'baseline-dir': { type: 'string', default: '.visual/baseline' },
      'current-dir': { type: 'string', default: '.visual/current' },
      'diff-dir': { type: 'string', default: '.visual/diff' },
      'side-by-side-dir': {
        type: 'string',
        default: '.visual/side-by-side',
      },
      report: { type: 'string', default: '.visual/report.html' },
      viewports: { type: 'string', default: '1280x720' },
      'wait-for': { type: 'string', default: '#root' },
      'wait-ms': { type: 'string', default: '1000' },
      'fixed-time': { type: 'string' },
      'full-page': { type: 'boolean', default: true },
      'include-diff': { type: 'boolean', default: false },
      'fail-on-diff': { type: 'boolean', default: false },
    },
    allowPositionals: true,
  });

  for (const key of [
    'routes',
    'out-dir',
    'baseline-dir',
    'current-dir',
    'diff-dir',
    'side-by-side-dir',
    'report',
  ])
    values[key] = resolvePath(values[key]);
  return { positionals, values };
}

async function run() {
  const { positionals, values } = parseCliOptions(process.argv.slice(2));
  const command = positionals[0];
  if (!command || (command !== 'capture' && command !== 'diff')) {
    // eslint-disable-next-line no-console
    console.log(`Usage:
  node scripts/visual-compare.mjs capture [--base-url http://localhost:3000]
  node scripts/visual-compare.mjs diff

Relative path flags and defaults use process.cwd(). npm visual:* runs in frontend/.
The wrapper passes absolute paths. Node >=24.2.0 is required.
  --out-dir PATH                (explicit capture output override)
  --baseline-dir PATH --current-dir PATH (explicit comparison overrides)

Optional flags:
  --routes PATH                 (default: frontend/scripts/visual-compare.routes.json in this checkout)
  --viewports 1280x720,375x812   (default: 1280x720)
  --wait-for "#root"             (default: #root)
  --wait-ms 1000                 (default: 1000)
  --fixed-time UTC_TIMESTAMP     (YYYY-MM-DDTHH:MM:SS[.mmm]Z; timers still run)
  --full-page                    (default: true)
  --side-by-side-dir PATH
  --include-diff                 (include diff image as third panel)
  --fail-on-diff                 (exit 1 if diffs found)
`);
    process.exit(1);
  }

  if (command === 'capture') {
    const outDir = values['out-dir'];
    const routesPath = values.routes;
    const viewports = values.viewports;
    await captureScreenshots({
      baseUrl: values['base-url'],
      outDir,
      routesPath,
      viewports,
      defaultWaitFor: values['wait-for'],
      defaultWaitMs: Number(values['wait-ms']),
      fullPage: values['full-page'],
      fixedTime: values['fixed-time'],
    });
    return;
  }

  const baselineDir = values['baseline-dir'];
  const currentDir = values['current-dir'];
  const diffDir = values['diff-dir'];
  const sideBySideDir = values['side-by-side-dir'];
  const reportPath = values.report;
  return runDiff({
    baselineDir,
    currentDir,
    diffDir,
    sideBySideDir,
    reportPath,
    includeDiff: values['include-diff'],
    failOnDiff: values['fail-on-diff'],
  });
}

export function runDiff({
  baselineDir,
  currentDir,
  diffDir,
  sideBySideDir,
  reportPath,
  includeDiff = false,
  failOnDiff = false,
}) {
  ensureDir(diffDir);
  ensureDir(sideBySideDir);
  ensureDir(path.dirname(reportPath));

  const results = [];
  let mismatchFound = false;
  const readInventory = (directory, name) => {
    try {
      return captureInventory(directory, (error) => {
        results.push({ name, error });
      });
    } catch (error) {
      results.push({
        name,
        error: `Cannot read ${name} captures at ${directory}: ${error.message}`,
      });

      return new Map();
    }
  };
  const baselineFiles = readInventory(baselineDir, 'baseline');
  const currentFiles = readInventory(currentDir, 'current');

  for (const fileName of baselineFiles.keys()) {
    const baselinePath = baselineFiles.get(fileName)?.filePath;
    const currentPath = currentFiles.get(fileName)?.filePath;
    const diffPath = path.join(diffDir, fileName);
    try {
      const failures = [];
      if (baselineFiles.get(fileName)?.status !== 'ok')
        failures.push(`Missing or failed baseline capture for ${fileName}`);
      if (currentFiles.get(fileName)?.status !== 'ok')
        failures.push(`Missing or failed current capture for ${fileName}`);
      if (failures.length) throw new Error(failures.join('; '));

      const comparison = compareImages(baselinePath, currentPath, diffPath);
      const { mismatchedPixels } = comparison;
      const sideBySidePath = path.join(sideBySideDir, fileName);
      writeSideBySide({
        ...comparison,
        outPath: sideBySidePath,
        includeDiff,
      });
      if (mismatchedPixels > 0) mismatchFound = true;
      results.push({
        name: fileName,
        baselinePath,
        currentPath,
        diffPath,
        mismatchedPixels,
        error: null,
      });
      // eslint-disable-next-line no-console
      console.log(`${fileName}: ${mismatchedPixels} pixels differ`);
    } catch (error) {
      results.push({
        name: fileName,
        baselinePath,
        currentPath,
        diffPath,
        mismatchedPixels: 0,
        error: String(error),
      });

      // eslint-disable-next-line no-console
      console.error(`Failed to diff ${fileName}: ${error}`);
    }
  }

  for (const [name, capture] of currentFiles) {
    if (baselineFiles.has(name)) continue;
    try {
      if (capture.status !== 'ok') throw new Error(`Missing or failed current capture for ${name}`);
      fs.accessSync(capture.filePath, fs.constants.R_OK);
      results.push({ name, added: true });
    } catch (error) {
      results.push({ name, error: String(error) });
    }
  }

  writeReport({ reportPath, baselineDir, currentDir, diffDir, results });
  // eslint-disable-next-line no-console
  console.log(`Report written to ${reportPath}`);

  if (results.some((result) => result.error) || (failOnDiff && mismatchFound)) {
    return 1;
  }
  return 0;
}

if (import.meta.main) {
  run()
    .then((code) => {
      if (code) process.exit(code);
    })
    .catch((error) => {
      // eslint-disable-next-line no-console
      console.error(error);
      process.exit(1);
    });
}
