import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseArgs } from 'node:util';
import { chromium } from 'playwright';
import pixelmatch from 'pixelmatch';
import { PNG } from 'pngjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const frontendRoot = path.resolve(__dirname, '..');
const defaultRoutesPath = path.join(__dirname, 'visual-compare.routes.json');

function resolvePath(targetPath) {
  return path.isAbsolute(targetPath) ? targetPath : path.resolve(frontendRoot, targetPath);
}

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function toSlug(input) {
  return input.replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-+|-+$/g, '');
}

function parseViewports(value) {
  return value.split(',').map((entry) => {
    const [width, height] = entry.split('x').map(Number);
    if (!width || !height) {
      throw new Error(`Invalid viewport "${entry}". Use WIDTHxHEIGHT, e.g. 1280x720.`);
    }
    return { width, height };
  });
}

function buildUrl(baseUrl, routePath) {
  const base = baseUrl.replace(/\/$/, '');
  const hashPath = routePath.startsWith('/') ? `#${routePath}` : `#/${routePath}`;
  return `${base}/${hashPath}`;
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
    const name = route.name || route.path;
    if (typeof name !== 'string' || !toSlug(name) || names.has(toSlug(name))) {
      throw new Error(`Capture route names must produce unique filenames: ${name}`);
    }
    names.add(toSlug(name));
    for (const key of ['waitForSelectors', 'failOnRequestErrors']) {
      if (
        route[key] !== undefined &&
        (!Array.isArray(route[key]) ||
          route[key].some((value) => typeof value !== 'string' || !value))
      ) {
        throw new Error(`${key} must be an array of nonempty strings: ${name}`);
      }
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
  const routes = loadRoutes(routesPath);
  if (
    fixedTime &&
    (Number.isNaN(Date.parse(fixedTime)) ||
      !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/.test(fixedTime))
  ) {
    throw new Error(`Invalid fixed time: ${fixedTime}`);
  }
  ensureDir(outDir);
  const manifestPath = path.join(outDir, 'capture-results.json');
  fs.rmSync(manifestPath, { force: true });
  const browser = await browserType.launch();
  const results = [];
  const contextOptions = {
    locale: 'en-US',
    timezoneId: 'UTC',
    colorScheme: 'light',
    reducedMotion: 'reduce',
  };

  try {
    for (const viewport of viewports) {
      for (const route of routes) {
        const url = buildUrl(baseUrl, route.path);
        const fileName = `${toSlug(route.name)}-${viewport.width}x${viewport.height}.png`;
        const filePath = path.join(outDir, fileName);
        const waitForSelector = route.waitForSelector || defaultWaitFor;
        const waitForTimeoutMs = route.waitForTimeoutMs ?? defaultWaitMs;
        const setup = {
          waitForSelectors: route.waitForSelectors,
          fillFields: route.fillFields,
          fitGraph: route.fitGraph,
          failOnRequestErrors: route.failOnRequestErrors,
          failOnPageErrors: route.failOnPageErrors,
        };
        const pageErrors = [];
        let page;

        // A failed recapture must never leave a previous screenshot looking current.
        fs.rmSync(filePath, { force: true });
        try {
          // Isolate storage, query caches, and route state between captures.
          page = await browser.newPage({ viewport, ...contextOptions });
          // Error checks are opt-in per route: some deployment pages intentionally probe
          // endpoints that can return 404, and error-state captures need those responses.
          if (route.failOnPageErrors) {
            page.on('pageerror', (error) => pageErrors.push(`Page error: ${error.message}`));
          }
          if (route.failOnRequestErrors?.length) {
            const deploymentUrl = new URL(baseUrl);
            const deploymentPath = deploymentUrl.pathname.replace(/\/+$/, '');
            const checkedPath = (requestUrl) => {
              const parsed = new URL(requestUrl);
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
              if (pathname) {
                pageErrors.push(
                  `Request failed: ${pathname}: ${request.failure()?.errorText || 'unknown error'}`,
                );
              }
            });
          }
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
            await page.getByRole('textbox', { name: field.label }).fill(field.value);
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
          if (pageErrors.length) {
            throw new Error(pageErrors.join('; '));
          }
          await page.screenshot({ path: filePath, fullPage, animations: 'disabled' });
          if (pageErrors.length) {
            throw new Error(pageErrors.join('; '));
          }
          results.push({
            name: route.name,
            route: route.path,
            viewport,
            waitForSelector,
            waitForTimeoutMs,
            ...setup,
            filePath,
            status: 'ok',
          });
          // eslint-disable-next-line no-console
          console.log(`Captured ${url} -> ${filePath}`);
        } catch (error) {
          fs.rmSync(filePath, { force: true });
          results.push({
            name: route.name,
            route: route.path,
            viewport,
            waitForSelector,
            waitForTimeoutMs,
            ...setup,
            filePath,
            status: 'error',
            error: String(error),
          });
          // eslint-disable-next-line no-console
          console.error(`Failed to capture ${url}: ${error}`);
        } finally {
          await page?.close();
        }
      }
    }
  } finally {
    fs.writeFileSync(
      manifestPath,
      JSON.stringify(
        {
          baseUrl,
          routesPath,
          browser: browser.version(),
          ...contextOptions,
          fixedTime,
          fullPage,
          results,
        },
        null,
        2,
      ) + '\n',
    );
    await browser.close();
  }

  const failures = results.filter((result) => result.status === 'error');
  if (failures.length) {
    throw new Error(`${failures.length} screenshot capture(s) failed. See ${manifestPath}`);
  }
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
  return { mismatchedPixels, totalPixels: baseline.width * baseline.height };
}

function writeSideBySide({ baselinePath, currentPath, diffPath, outPath, includeDiff }) {
  const baseline = PNG.sync.read(fs.readFileSync(baselinePath));
  const current = PNG.sync.read(fs.readFileSync(currentPath));
  const diff = includeDiff ? PNG.sync.read(fs.readFileSync(diffPath)) : null;
  const width =
    baseline.width + current.width + (includeDiff && diff ? diff.width : 0);
  const height = Math.max(
    baseline.height,
    current.height,
    includeDiff && diff ? diff.height : 0,
  );
  const combined = new PNG({ width, height });
  combined.data.fill(255);
  PNG.bitblt(baseline, combined, 0, 0, baseline.width, baseline.height, 0, 0);
  PNG.bitblt(
    current,
    combined,
    0,
    0,
    current.width,
    current.height,
    baseline.width,
    0,
  );
  if (includeDiff && diff) {
    PNG.bitblt(
      diff,
      combined,
      0,
      0,
      diff.width,
      diff.height,
      baseline.width + current.width,
      0,
    );
  }
  fs.writeFileSync(outPath, PNG.sync.write(combined));
}

function writeReport({ reportPath, baselineDir, currentDir, diffDir, results }) {
  const rows = results
    .map(result => {
      const baselineRel = path.relative(path.dirname(reportPath), result.baselinePath);
      const currentRel = path.relative(path.dirname(reportPath), result.currentPath);
      const diffRel = path.relative(path.dirname(reportPath), result.diffPath);
      const status = result.error ? 'error' : result.mismatchedPixels > 0 ? 'diff' : 'match';
      return `
        <tr class="${status}">
          <td>${result.name}</td>
          <td>${result.mismatchedPixels}</td>
          <td><img src="${baselineRel}" alt="baseline ${result.name}"></td>
          <td><img src="${currentRel}" alt="current ${result.name}"></td>
          <td><img src="${diffRel}" alt="diff ${result.name}"></td>
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
        <p>Baseline: ${baselineDir}</p>
        <p>Current: ${currentDir}</p>
        <p>Diffs: ${diffDir}</p>
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

async function run() {
  const { positionals, values } = parseArgs({
    options: {
      'base-url': { type: 'string', default: 'http://localhost:3000' },
      routes: { type: 'string', default: defaultRoutesPath },
      'out-dir': { type: 'string', default: '.visual/current' },
      'baseline-dir': { type: 'string', default: '.visual/baseline' },
      'current-dir': { type: 'string', default: '.visual/current' },
      'diff-dir': { type: 'string', default: '.visual/diff' },
      'side-by-side-dir': { type: 'string', default: '.visual/side-by-side' },
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

  const command = positionals[0];
  if (!command || (command !== 'capture' && command !== 'diff')) {
    // eslint-disable-next-line no-console
    console.log(`Usage:
  node scripts/visual-compare.mjs capture [--base-url http://localhost:3000] [--out-dir .visual/current]
  node scripts/visual-compare.mjs diff [--baseline-dir .visual/baseline] [--current-dir .visual/current]

Optional flags:
  --routes path/to/routes.json   (default: scripts/visual-compare.routes.json)
  --viewports 1280x720,375x812   (default: 1280x720)
  --wait-for "#root"             (default: #root)
  --wait-ms 1000                 (default: 1000)
  --fixed-time ISO_TIMESTAMP     (freeze Date while allowing timers to run)
  --full-page                    (default: true)
  --side-by-side-dir .visual/side-by-side
  --include-diff                 (include diff image as third panel)
  --fail-on-diff                 (exit 1 if diffs found)
`);
    process.exit(1);
  }

  if (command === 'capture') {
    const outDir = resolvePath(values['out-dir']);
    const routesPath = resolvePath(values.routes);
    const viewports = parseViewports(values.viewports);
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

  const baselineDir = resolvePath(values['baseline-dir']);
  const currentDir = resolvePath(values['current-dir']);
  const diffDir = resolvePath(values['diff-dir']);
  const sideBySideDir = resolvePath(values['side-by-side-dir']);
  const reportPath = resolvePath(values.report);
  ensureDir(diffDir);
  ensureDir(sideBySideDir);
  ensureDir(path.dirname(reportPath));

  const baselineFiles = fs.readdirSync(baselineDir).filter(file => file.endsWith('.png'));
  const results = [];
  let mismatchFound = false;

  for (const fileName of baselineFiles) {
    const baselinePath = path.join(baselineDir, fileName);
    const currentPath = path.join(currentDir, fileName);
    const diffPath = path.join(diffDir, fileName);
    if (!fs.existsSync(currentPath)) {
      results.push({
        name: fileName,
        baselinePath,
        currentPath,
        diffPath,
        mismatchedPixels: 0,
        error: `Missing current screenshot for ${fileName}`,
      });
      mismatchFound = true;
      continue;
    }
    try {
      const { mismatchedPixels } = compareImages(baselinePath, currentPath, diffPath);
      const sideBySidePath = path.join(sideBySideDir, fileName);
      writeSideBySide({
        baselinePath,
        currentPath,
        diffPath,
        outPath: sideBySidePath,
        includeDiff: values['include-diff'],
      });
      if (mismatchedPixels > 0) {
        mismatchFound = true;
      }
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
      mismatchFound = true;
      // eslint-disable-next-line no-console
      console.error(`Failed to diff ${fileName}: ${error}`);
    }
  }

  writeReport({ reportPath, baselineDir, currentDir, diffDir, results });
  // eslint-disable-next-line no-console
  console.log(`Report written to ${reportPath}`);

  if (values['fail-on-diff'] && mismatchFound) {
    process.exit(1);
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === __filename) {
  run().catch((error) => {
    // eslint-disable-next-line no-console
    console.error(error);
    process.exitCode = 1;
  });
}
