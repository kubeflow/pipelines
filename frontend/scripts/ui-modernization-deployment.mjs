/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
 */
import assert from 'node:assert/strict';
import { execFile } from 'node:child_process';
import { createHash } from 'node:crypto';
import { createServer, request } from 'node:http';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { promisify } from 'node:util';

const exec = promisify(execFile);
const sha = (value) => createHash('sha256').update(value).digest('hex');
const sleep = (ms) => new Promise((done) => setTimeout(done, ms));
const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const helper = resolve(root, '.github/resources/scripts/qualify_frontend_deployment.py');

// Dex v2.45.1 labels its password submit button 'Login'.
export const loginButtonName = /^(?:log\s?in|sign\s?in)$/i;

export const selectedNamespaceSelector = 'namespace-selector #SelectedNamespace';

export function namespaceOptionPattern(namespace) {
  const escaped = namespace.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return new RegExp(`^\\s*${escaped}\\s*$`);
}

export async function selectNamespace(target, selected) {
  const pattern = namespaceOptionPattern(selected);
  const isSelected = async () =>
    pattern.test(await target.locator(selectedNamespaceSelector).innerText());
  // A single-namespace user is selected automatically by the dashboard. Opening
  // that dropdown races its initialization and can hide the option before click.
  if (await isSelected()) return;
  await target.locator('namespace-selector #dropdown-trigger').click();
  // Polymer binds name as a property without reflecting an HTML attribute.
  await target.locator('namespace-selector paper-item').filter({ hasText: pattern }).click();
  await poll(isSelected, 'Dashboard namespace selection did not settle');
}

export function experimentRunDependency(url, experimentIds) {
  const parsed = new URL(url);
  const id = parsed.searchParams.get('experiment_id');
  return parsed.pathname.endsWith('/apis/v2beta1/runs') && experimentIds.has(id) ? id : null;
}

export function safePageUrl(url, secrets = []) {
  try {
    const parsed = new URL(url);
    return sanitize(`${parsed.origin}${parsed.pathname}`, secrets);
  } catch {
    return '[unavailable URL]';
  }
}

export async function openRunGraph(target, url) {
  await target.goto(url, { waitUntil: 'domcontentloaded' });
  // Run details remember the selected tab when navigating to the same run.
  await target
    .locator('button')
    .filter({ hasText: /^Graph$/ })
    .click();
}

export function requireHosted(env = process.env) {
  assert.equal(env.CI, 'true');
  assert.equal(env.GITHUB_ACTIONS, 'true');
  assert.equal(env.RUNNER_ENVIRONMENT, 'github-hosted');
  assert.ok(
    env.RUNNER_TEMP?.startsWith('/'),
    'An absolute disposable runner directory is required',
  );
}

export function assetName(url, base) {
  const parsed = new URL(url);
  if (parsed.origin !== new URL(base).origin) return null;
  const pathname = parsed.pathname.replace(/^\/pipeline\//, '/').slice(1);
  // Dashboard's separately deployed bundle is intentionally outside the UI image.
  return /^(assets|static)\/.*\.(js|css)$/.test(pathname) ? pathname : null;
}

export function sanitize(message, secrets = []) {
  let value = String(message);
  for (const secret of secrets.filter(Boolean)) value = value.split(secret).join('[redacted]');
  return value
    .replace(/(\/apps\/tensorboard\/proxy\/)[^/\s"'?]+/g, '$1[redacted]')
    .replace(/https?:\/\/[^\s"'<>]+/g, (url) => {
      try {
        const parsed = new URL(url);
        return `${parsed.origin}${parsed.pathname}${parsed.search ? '?[redacted]' : ''}`;
      } catch {
        return '[redacted URL]';
      }
    })
    .replace(/(code|token|state|session|password|cookie|secret)=([^\s&]+)/gi, '$1=[redacted]');
}

export function stableRun(run) {
  assert.ok(run.run_id && run.experiment_id && run.display_name);
  return Object.fromEntries(
    [
      'run_id',
      'experiment_id',
      'display_name',
      'description',
      'pipeline_version_reference',
      'runtime_config',
      'service_account',
      'state',
    ].map((key) => [key, run[key] ?? null]),
  );
}

export function stableSchedule(schedule) {
  assert.ok(schedule.recurring_run_id && schedule.experiment_id && schedule.display_name);
  return Object.fromEntries(
    [
      'recurring_run_id',
      'experiment_id',
      'display_name',
      'description',
      'pipeline_version_reference',
      'runtime_config',
      'service_account',
      'trigger',
      'max_concurrency',
      'no_catchup',
      'status',
    ].map((key) => [key, schedule[key] ?? null]),
  );
}

export function verifyAsset(name, bytes, manifest) {
  assert.ok(manifest[name], `Unexpected asset generation: ${name}`);
  assert.equal(sha(bytes), manifest[name], `Wrong asset bytes: ${name}`);
}

export function assertLoginRequired(status, location, base) {
  if ([401, 403].includes(status)) return;
  assert.ok([302, 303].includes(status), 'Ingress accepted an unauthenticated API request');
  const target = new URL(location, base);
  assert.equal(target.origin, new URL(base).origin);
  assert.ok(
    /^\/(dex|oauth2)\//.test(target.pathname),
    'Redirect did not target the real login flow',
  );
}

export function summarizeAvailability(samples) {
  assert.ok(samples.length > 0, 'No availability samples');
  const failures = samples.filter((sample) => !sample.ready);
  const firstFailure = failures[0];
  const lastFailure = failures.at(-1);
  const recovery =
    lastFailure && samples.find((sample) => sample.ready && sample.offsetMs > lastFailure.offsetMs);
  return {
    samples,
    unavailableSamples: failures.length,
    observedUnavailable: failures.length > 0,
    observedUnavailableSpanMs:
      failures.length > 1 ? lastFailure.offsetMs - firstFailure.offsetMs : null,
    recoveryUpperBoundMs: recovery
      ? recovery.offsetMs - firstFailure.offsetMs + firstFailure.durationMs
      : null,
    intervalMs: 200,
    requestDeadlineMs: 2000,
    strategy: 'Recreate',
  };
}

async function poll(check, message, timeout = 60000, interval = 500) {
  const deadline = Date.now() + timeout;
  do {
    if (await check()) return;
    await sleep(interval);
  } while (Date.now() < deadline);
  throw new Error(message);
}

function standaloneProxy() {
  // Keep browser origin and requests unchanged while Kubernetes resolves the
  // current Service endpoint after Recreate. This is a transport proxy only.
  const prefix = '/api/v1/namespaces/kubeflow/services/ml-pipeline-ui:80/proxy';
  const server = createServer((incoming, outgoing) => {
    const upstream = request(
      {
        hostname: '127.0.0.1',
        port: 3001,
        method: incoming.method,
        path: prefix + incoming.url,
        headers: { ...incoming.headers, host: '127.0.0.1:3001' },
      },
      (response) => {
        outgoing.writeHead(response.statusCode, response.headers);
        response.pipe(outgoing);
      },
    );
    upstream.setTimeout(15000, () => upstream.destroy());
    upstream.on('error', () => {
      if (!outgoing.headersSent) outgoing.writeHead(503);
      outgoing.end();
    });
    incoming.pipe(upstream);
  });
  return new Promise((done, reject) => {
    server.once('error', reject);
    server.listen(3000, '127.0.0.1', () => done(server));
  });
}

export async function main() {
  requireHosted();
  const { chromium } = await import('playwright');
  const output = resolve(process.env.KFP_DEPLOYMENT_REPORT_DIR);
  const mode = process.env.KFP_DEPLOYMENT_MODE;
  assert.ok(['standalone', 'multiuser'].includes(mode));
  const base = process.env.KFP_DEPLOYMENT_BASE_URL;
  assert.equal(new URL(base).origin, 'http://127.0.0.1:3000');
  await mkdir(output, { recursive: true });
  const uiBase = base + (mode === 'multiuser' ? '/pipeline/' : '/');
  const report = {
    sourceSha: process.env.GITHUB_SHA,
    runId: process.env.GITHUB_RUN_ID,
    runAttempt: process.env.GITHUB_RUN_ATTEMPT,
    mode,
    startedAt: new Date().toISOString(),
    checks: [],
    phases: {},
    availability: {},
    errors: [],
  };
  const credentials =
    mode === 'multiuser'
      ? JSON.parse(
          await readFile(
            resolve(process.env.RUNNER_TEMP, 'kfp-qualification-credentials.json'),
            'utf8',
          ),
        )
      : {};
  const secretValues = [credentials.password];
  let phase = 'baseline';
  let loggedIn = false;
  let browser;
  let proxy;
  let page;
  const assets = {
    baseline: JSON.parse(await readFile(resolve(output, 'legacy-assets.json'), 'utf8')),
    candidate: JSON.parse(await readFile(resolve(output, 'candidate-assets.json'), 'utf8')),
  };
  assets.rollback = assets.baseline;
  const pendingAssets = new Set();
  const assetErrors = [];
  const pageErrors = [];
  const resources = { runs: [], schedules: [] };
  const namespace = mode === 'multiuser' ? 'kubeflow-user-example-com' : 'kubeflow';
  const prefix = `ui-rollback-${process.env.GITHUB_RUN_ID}-${process.env.GITHUB_RUN_ATTEMPT}`;
  let app;
  let otherContext;
  let otherPage;
  let signedTensorboardUrl;
  let artifactUrl;
  let artifactHash;
  let tensorboardRun;
  let logsUrl;
  let otherExperiment;

  const save = async () =>
    writeFile(resolve(output, 'result.json'), JSON.stringify(report, null, 2) + '\n');
  const python = (operation, extra = []) =>
    exec(
      'python',
      [helper, operation, '--output', output, '--mode', mode, '--phase', phase, ...extra],
      { cwd: root, timeout: 360000, maxBuffer: 1024 * 1024 },
    );
  const check = async (name, action) => {
    const started = performance.now();
    try {
      await action();
      report.checks.push({ phase, name, passed: true, elapsedMs: performance.now() - started });
    } catch (error) {
      report.checks.push({
        phase,
        name,
        passed: false,
        error: sanitize(error.message, secretValues),
      });
      throw error;
    } finally {
      await save();
    }
  };
  const track = (target) => {
    target.on('pageerror', (error) =>
      pageErrors.push({ phase, message: sanitize(error.message, secretValues) }),
    );
    target.on('response', (response) => {
      if (new URL(response.url()).pathname.endsWith('/k8s/pod/logs') && response.ok())
        logsUrl = response.url();
      const name = assetName(response.url(), base);
      if (!name) return;
      const capturedPhase = phase;
      const operation = (async () => {
        assert.equal(response.status(), 200, `Failed asset request: ${name}`);
        verifyAsset(name, await response.body(), assets[capturedPhase]);
        const list = report.phases[capturedPhase].assets;
        if (!list.includes(name)) list.push(name);
      })()
        .catch((error) => assetErrors.push(sanitize(error.message, secretValues)))
        .finally(() => pendingAssets.delete(operation));
      pendingAssets.add(operation);
    });
  };
  const frame = async (target = page) => {
    if (mode === 'standalone') return target;
    await poll(
      () =>
        target
          .frames()
          .some((item) => new URL(item.url() || 'about:blank').pathname === '/pipeline/'),
      'Dashboard did not embed the real pipeline application',
    );
    return target
      .frames()
      .find((item) => new URL(item.url() || 'about:blank').pathname === '/pipeline/');
  };
  const go = async (route) => {
    await app.goto(`${uiBase}#${route}`, { waitUntil: 'domcontentloaded' });
  };
  const api = async (path, target = app) =>
    target.evaluate(async (relative) => {
      const response = await fetch(new URL(relative, window.location.href.split('#')[0]));
      return { status: response.status, body: await response.json() };
    }, `apis/v2beta1/${path}`);
  const read = async (path) => {
    const result = await api(path);
    assert.equal(result.status, 200, `Live API request failed: ${path.split('?')[0]}`);
    return result.body;
  };
  const login = async (target, email) => {
    const documents = [];
    const recordDocument = (response) => {
      if (response.request().resourceType() !== 'document') return;
      documents.push({ status: response.status(), url: safePageUrl(response.url(), secretValues) });
      if (documents.length > 20) documents.shift();
    };
    target.on('response', recordDocument);
    try {
      await target.goto(base, { waitUntil: 'domcontentloaded' });
      const signIn = target.getByRole('button', { name: /Sign in with Dex/i });
      await poll(
        async () =>
          (await signIn.isVisible()) || (await target.locator('input[name="login"]').isVisible()),
        'OIDC login page did not become ready',
      );
      if (await signIn.isVisible()) await signIn.click();
      await target.locator('input[name="login"]').fill(email);
      await target.locator('input[name="password"]').fill(credentials.password);
      await target.getByRole('button', { name: loginButtonName }).click();
      await target.locator(selectedNamespaceSelector).waitFor({ timeout: 120000 });
      await selectNamespace(
        target,
        email === 'user@example.com' ? namespace : 'kfp-qualification-other',
      );
      const pipelineLink = target
        .locator('iframe-link paper-item')
        .filter({ hasText: /^\s*Pipelines\s*$/ });
      if (!(await pipelineLink.isVisible())) {
        await target
          .locator('paper-item.section-item')
          .filter({ hasText: /^\s*Pipelines\s*$/ })
          .click();
      }
      await pipelineLink.click();
      const embedded = await frame(target);
      await embedded.locator('#createPipelineVersionBtn').waitFor({ timeout: 60000 });
      return embedded;
    } catch (error) {
      // Preserve the original failure while collecting only bounded, credential-safe labels.
      const diagnostic = { url: safePageUrl(target.url(), secretValues), documents };
      let timer;
      try {
        const labels = await Promise.race([
          Promise.all([
            target.title(),
            target
              .locator('h1,h2,[role="alert"],button')
              .filter({ visible: true })
              .allTextContents(),
          ]),
          new Promise((_, reject) => {
            timer = setTimeout(() => reject(new Error('diagnostic timeout')), 3000);
          }),
        ]);
        diagnostic.title = sanitize(labels[0], secretValues).slice(0, 200);
        diagnostic.labels = labels[1]
          .slice(0, 20)
          .map((label) => sanitize(label, secretValues).replace(/\s+/g, ' ').trim().slice(0, 200));
      } catch {
        diagnostic.labelsUnavailable = true;
      } finally {
        clearTimeout(timer);
      }
      report.loginDiagnostics = diagnostic;
      throw error;
    } finally {
      target.off('response', recordDocument);
    }
  };
  const submit = async (button, endpoint) => {
    const responsePromise = page.waitForResponse(
      (response) =>
        response.request().method() === 'POST' &&
        new URL(response.url()).pathname.endsWith(endpoint),
    );
    await button.click();
    const response = await responsePromise;
    assert.ok(response.ok(), `UI creation failed: ${endpoint} (${response.status()})`);
    return response.json();
  };
  const upload = async (name, fixture) => {
    await go('/pipeline_versions/new');
    await app.locator('#localPackageBtn').click();
    await app
      .locator('#dropZone input[type="file"]')
      .setInputFiles(resolve(root, 'test/frontend-integration-test', fixture));
    await app.locator('#newPipelineName').fill(name);
    await app.locator('#createNewPipelineOrVersionBtn').click();
    await poll(
      () => /#\/pipelines\/details\/[^/]+\/version\/[^/?]+/.test(app.url()),
      'Uploaded pipeline did not open its version details',
    );
    const match = /#\/pipelines\/details\/([^/]+)\/version\/([^/?]+)/.exec(app.url());
    return { name, pipelineId: match[1], versionId: match[2] };
  };
  const experiment = async (name) => {
    await go('/experiments/new');
    await app.locator('#experimentName').fill(name);
    return submit(app.locator('#createExperimentBtn'), '/apis/v2beta1/experiments');
  };
  const createRun = async (pipeline, experimentId, name, recurring = false) => {
    const parameters = new URLSearchParams({
      pipelineId: pipeline.pipelineId,
      pipelineVersionId: pipeline.versionId,
      experimentId,
    });
    if (recurring) parameters.set('recurring', '1');
    await go(`/runs/new?${parameters}`);
    await app.getByLabel(recurring ? /^Recurring run config name/ : /^Run name/).fill(name);
    if (pipeline.name.endsWith('-hello'))
      await app.locator('#message').fill(`qualification-${name}`);
    if (recurring) {
      await app.getByRole('checkbox', { name: 'Has start date', exact: true }).check();
      const future = new Date(Date.now() + 7 * 86400000).toISOString().slice(0, 10);
      await app.getByLabel('Start date', { exact: true }).fill(future);
      await app.getByLabel('Start time', { exact: true }).fill('12:00');
      const schedule = await submit(app.locator('#startNewRunBtn'), '/apis/v2beta1/recurringruns');
      assert.ok(
        Date.parse(schedule.trigger?.periodic_schedule?.start_time) > Date.now() + 86400000,
      );
      let ready;
      await poll(async () => {
        ready = await read(`recurringruns/${schedule.recurring_run_id}`);
        return ready.status === 'ENABLED';
      }, 'Future schedule did not reconcile to ENABLED');
      assert.ok(Date.parse(ready.trigger?.periodic_schedule?.start_time) > Date.now() + 86400000);
      resources.schedules.push(stableSchedule(ready));
      return ready;
    }
    const run = await submit(app.locator('#startNewRunBtn'), '/apis/v2beta1/runs');
    let completed;
    await poll(
      async () => {
        completed = await read(`runs/${run.run_id}`);
        assert.ok(
          !['FAILED', 'CANCELED', 'SKIPPED'].includes(completed.state),
          'Real pipeline execution failed',
        );
        return completed.state === 'SUCCEEDED';
      },
      'Real pipeline run did not succeed',
      600000,
      2000,
    );
    resources.runs.push(stableRun(completed));
    await go(`/runs/details/${run.run_id}`);
    await app
      .locator('button')
      .filter({ hasText: /^Detail$/ })
      .click();
    await app.getByText('Succeeded', { exact: true }).first().waitFor({ timeout: 60000 });
    return completed;
  };
  const inspectLogs = async (run) => {
    await openRunGraph(app, `${uiBase}#/runs/details/${run.run_id}`);
    await app.locator('.react-flow__node-EXECUTION').filter({ hasText: /^A/ }).first().click();
    await app
      .locator('button')
      .filter({ hasText: /^Logs$/ })
      .click();
    await app
      .getByText(`${run.runtime_config.parameters.message} from node:`, { exact: false })
      .first()
      .waitFor({ timeout: 120000 });
  };
  const openArtifact = async () => {
    await openRunGraph(app, `${uiBase}#/runs/details/${tensorboardRun.run_id}`);
    await app.locator('.react-flow__node-ARTIFACT').first().click();
    const visualization = app.locator('button').filter({ hasText: /^Visualization$/ });
    await visualization.click();
  };
  const tensorboard = async (initial = false) => {
    await openArtifact();
    if (initial) {
      await app.getByRole('button', { name: 'Start Tensorboard', exact: true }).click();
    }
    // Legacy puts its startup warning inside this anchor, changing its accessible name.
    const open = app
      .locator('a[href*="apps/tensorboard/proxy/"]')
      .filter({ hasText: 'Open Tensorboard' });
    await open.waitFor({ timeout: 180000 });
    if (initial) {
      signedTensorboardUrl = new URL(await open.getAttribute('href'), uiBase).href;
      secretValues.push(signedTensorboardUrl);
      const token = /\/apps\/tensorboard\/proxy\/([^/]+)/.exec(
        new URL(signedTensorboardUrl).pathname,
      )?.[1];
      assert.ok(token, 'TensorBoard did not provide a signed application proxy URL');
      secretValues.push(token, decodeURIComponent(token));
      report.tensorboardUrlSha256 = sha(signedTensorboardUrl);
    }
    const viewer = await page.context().newPage();
    try {
      await poll(
        async () => {
          await viewer.goto(signedTensorboardUrl, { waitUntil: 'domcontentloaded' });
          return viewer.locator('#topBar').isVisible();
        },
        'Previously signed TensorBoard URL did not serve the real application',
        240000,
        3000,
      );
      await viewer.screenshot({ path: resolve(output, `${phase}-tensorboard.png`) });
    } finally {
      await viewer.close();
    }
    // Download through the actual visible link in every generation, preserving
    // the original URI and bytes without posting directly to an API.
    await app
      .locator('button')
      .filter({ hasText: /^Artifact Info$/ })
      .click();
    const artifactLink = app.locator('a[href*="artifacts/get"]').first();
    await artifactLink.waitFor();
    const currentUrl = new URL(await artifactLink.getAttribute('href'), uiBase).href;
    if (initial) artifactUrl = currentUrl;
    assert.equal(currentUrl, artifactUrl, 'Artifact storage location changed across the UI swap');
    const downloadPromise = page.waitForEvent('download');
    await artifactLink.click();
    const download = await downloadPromise;
    assert.equal(await download.failure(), null);
    const saved = resolve(process.env.RUNNER_TEMP, `${phase}-qualification-artifact`);
    await download.saveAs(saved);
    const bytes = await readFile(saved);
    assert.ok(bytes.toString().includes('tensorboard'));
    if (initial) artifactHash = sha(bytes);
    assert.equal(sha(bytes), artifactHash);
    report.artifactSha256 = artifactHash;
  };
  const health = async (context) => {
    const started = performance.now();
    try {
      const response = await context.request.get(uiBase + 'apis/v2beta1/healthz', {
        timeout: 2000,
      });
      return {
        offsetMs: performance.now(),
        durationMs: performance.now() - started,
        status: response.status(),
        ready: response.ok() && (await response.json()).apiServerReady === true,
      };
    } catch {
      return {
        offsetMs: performance.now(),
        durationMs: performance.now() - started,
        status: null,
        ready: false,
      };
    }
  };
  const swap = async (context) => {
    const samples = [];
    let collecting = true;
    const sampler = (async () => {
      while (collecting) {
        samples.push(await health(context));
        await sleep(200);
      }
    })();
    try {
      await python('swap');
      await poll(async () => (await health(context)).ready, 'UI health did not recover', 60000);
      await sleep(1000);
    } finally {
      collecting = false;
      await sampler;
      report.availability[phase] = summarizeAvailability(samples);
    }
  };

  try {
    if (mode === 'standalone') proxy = await standaloneProxy();
    browser = await chromium.launch({ headless: true });
    report.browserVersion = browser.version();
    const context = await browser.newContext({
      viewport: { width: 1440, height: 1080 },
      timezoneId: 'UTC',
    });
    context.on('page', track);
    report.phases.baseline = { assets: [] };
    await python('swap');
    // A new unauthenticated context establishes the ingress really requires login.
    if (mode === 'multiuser') {
      const response = await context.request.get(
        uiBase + 'apis/v2beta1/runs?namespace=' + namespace,
        { maxRedirects: 0 },
      );
      assertLoginRequired(response.status(), response.headers().location, base);
    }
    page = await context.newPage();
    page.setDefaultTimeout(60000);
    app = mode === 'multiuser' ? await login(page, 'user@example.com') : page;
    if (mode === 'standalone') await go('/pipelines');
    loggedIn = true;
    await check('legacy session and stored preferences', async () => {
      await app.evaluate(() => {
        localStorage.setItem('navbarCollapsed', 'true');
        localStorage.setItem('tablePageSize_runs', '50');
      });
      await app.goto(app.url());
    });
    const pipeline = await upload(`${prefix}-hello`, 'helloworld.yaml');
    const tbPipeline = await upload(`${prefix}-tensorboard`, 'tensorboard-example.yaml');
    const ownerExperiment = await experiment(`${prefix}-experiment`);
    await check('legacy run creation and logs', async () => {
      const run = await createRun(pipeline, ownerExperiment.experiment_id, `${prefix}-legacy`);
      await inspectLogs(run);
    });
    await check('legacy future schedule creation', () =>
      createRun(pipeline, ownerExperiment.experiment_id, `${prefix}-legacy-schedule`, true),
    );
    await check('legacy artifact and signed TensorBoard access', async () => {
      tensorboardRun = await createRun(
        tbPipeline,
        ownerExperiment.experiment_id,
        `${prefix}-tensorboard`,
      );
      await tensorboard(true);
    });
    if (mode === 'multiuser') {
      otherContext = await browser.newContext({ viewport: { width: 1440, height: 1080 } });
      otherPage = await otherContext.newPage();
      const otherApp = await login(otherPage, 'other@example.com');
      await otherApp.goto(`${uiBase}#/experiments/new`);
      await otherApp.locator('#experimentName').fill(`${prefix}-other-owner`);
      const created = otherPage.waitForResponse(
        (response) =>
          response.request().method() === 'POST' &&
          new URL(response.url()).pathname.endsWith('/apis/v2beta1/experiments'),
      );
      await otherApp.locator('#createExperimentBtn').click();
      const response = await created;
      assert.ok(response.ok());
      otherExperiment = await response.json();
      assert.equal(otherExperiment.namespace, 'kfp-qualification-other');
    }
    const verifyAuthorization = async () =>
      check('authenticated namespace switching and normal access denial', async () => {
        let release;
        let complete;
        let held = false;
        const heldErrors = [];
        const experimentIds = new Set();
        const completedDependencies = new Set();
        let acceptingDependencies = false;
        const recordDependency = (response) => {
          if (!acceptingDependencies) return;
          const id = experimentRunDependency(response.url(), experimentIds);
          if (!id) return;
          (async () => {
            assert.equal(response.status(), 200, `Old-namespace run dependency failed: ${id}`);
            assert.equal(await response.finished(), null);
            await response.body();
            completedDependencies.add(id);
          })().catch((error) => heldErrors.push(sanitize(error.message, secretValues)));
        };
        const drainDependencies = async () => {
          if (!acceptingDependencies) return;
          await poll(
            () => {
              assert.deepEqual(heldErrors, []);
              return [...experimentIds].every((id) => completedDependencies.has(id));
            },
            'Old-namespace run response bodies did not finish before the stale-result assertion',
            60000,
          );
        };
        const gate = new Promise((done) => {
          release = done;
        });
        const delivered = new Promise((done) => {
          complete = done;
        });
        const routePattern = '**/apis/v2beta1/experiments?**';
        const holdPreviousNamespace = async (route) => {
          if (!held && new URL(route.request().url()).searchParams.get('namespace') === namespace) {
            held = true;
            try {
              const response = await route.fetch();
              assert.equal(response.status(), 200);
              const body = await response.json();
              for (const experiment of body.experiments || [])
                experimentIds.add(experiment.experiment_id);
              assert.ok(
                experimentIds.has(ownerExperiment.experiment_id),
                'Held namespace response omitted the seeded experiment',
              );
              await gate;
              acceptingDependencies = true;
              await route.fulfill({ response });
            } catch (error) {
              heldErrors.push(sanitize(error.message, secretValues));
              await route.abort().catch(() => {});
            } finally {
              complete();
            }
          } else await route.continue();
        };
        const drainHeldResponse = async () => {
          if (!held) return;
          let timeout;
          try {
            await Promise.race([
              delivered,
              new Promise((_, reject) => {
                timeout = setTimeout(
                  () => reject(new Error('Held namespace response did not complete')),
                  60000,
                );
              }),
            ]);
          } finally {
            clearTimeout(timeout);
          }
        };
        page.on('response', recordDependency);
        await page.route(routePattern, holdPreviousNamespace);
        try {
          await go('/experiments');
          await poll(() => held, 'No real owner-namespace request was available to delay');
          const nextResponse = page.waitForResponse((response) => {
            const url = new URL(response.url());
            return (
              url.pathname.endsWith('/apis/v2beta1/experiments') &&
              url.searchParams.get('namespace') === 'kfp-qualification-second'
            );
          });
          await selectNamespace(page, 'kfp-qualification-second');
          assert.equal((await nextResponse).status(), 200);
          release();
          await drainHeldResponse();
          await drainDependencies();
          assert.deepEqual(heldErrors, []);
          report.namespaceDependencies ||= [];
          report.namespaceDependencies.push({
            phase,
            experimentIds: [...experimentIds].sort(),
            completedRunResponseIds: [...completedDependencies].sort(),
            responseBodiesComplete: true,
          });
          await app.evaluate(
            () => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done))),
          );
          assert.ok(
            (await page.locator(selectedNamespaceSelector).innerText()).includes(
              'kfp-qualification-second',
            ),
          );
          assert.ok(
            !(await app.locator('body').innerText()).includes(ownerExperiment.display_name),
            'Previous namespace results leaked into the newly selected namespace',
          );
          await selectNamespace(page, namespace);
        } finally {
          release();
          try {
            await drainHeldResponse();
            await drainDependencies();
          } finally {
            page.off('response', recordDependency);
            await page.unroute(routePattern, holdPreviousNamespace);
          }
        }
        await go(`/runs/details/${resources.runs[0].run_id}`);
        assert.equal((await api(`runs/${resources.runs[0].run_id}`)).status, 200);
        await otherPage.reload({ waitUntil: 'domcontentloaded' });
        const otherApp = await frame(otherPage);
        const deniedResponse = otherPage.waitForResponse((response) =>
          new URL(response.url()).pathname.endsWith(
            `/apis/v2beta1/runs/${resources.runs[0].run_id}`,
          ),
        );
        await otherApp.goto(`${uiBase}#/runs/details/${resources.runs[0].run_id}`);
        assert.equal((await deniedResponse).status(), 403);
        assert.ok(
          !(await otherApp.locator('body').innerText()).includes(resources.runs[0].display_name),
        );
        await otherApp.goto(`${uiBase}#/experiments`);
        const own = await api('experiments?namespace=kfp-qualification-other', otherApp);
        assert.equal(own.status, 200);
        assert.ok(
          own.body.experiments.some((item) => item.experiment_id === otherExperiment.experiment_id),
        );
        assert.ok(
          !own.body.experiments.some(
            (item) => item.experiment_id === ownerExperiment.experiment_id,
          ),
        );
        await otherApp.getByText(otherExperiment.display_name, { exact: true }).first().waitFor();
        assert.ok(logsUrl, 'The owner did not read real pod logs through the UI');
        // These are the actual owner-visible links. A normal second login must
        // not acquire access by following a shared log/artifact/TensorBoard URL.
        for (const [kind, url] of [
          ['logs', logsUrl],
          ['artifact', artifactUrl],
          ['tensorboard', signedTensorboardUrl],
        ]) {
          const denied = await otherContext.request.get(url, { maxRedirects: 0 });
          assert.equal(denied.status(), 403, `Other user's session could access owner ${kind}`);
        }
      });
    if (mode === 'multiuser') await verifyAuthorization();
    // Seed resources and TensorBoard before recording stable workload/config identities.
    await python('snapshot');
    for (const next of ['candidate', 'rollback']) {
      phase = next;
      report.phases[phase] = { assets: [] };
      await check('immutable UI-only image replacement and health recovery', () => swap(context));
      await page.reload({ waitUntil: 'domcontentloaded' });
      app = await frame();
      await go('/runs');
      await check('existing preferences survive without resetting storage', async () => {
        const values = await app.evaluate(() => ({
          navbarCollapsed: localStorage.getItem('navbarCollapsed'),
          tablePageSize: localStorage.getItem('tablePageSize_runs'),
          theme: localStorage.getItem('kfp.theme'),
        }));
        assert.equal(values.navbarCollapsed, 'true');
        assert.equal(values.tablePageSize, '50');
        if (phase === 'candidate') {
          const size = app.getByRole('combobox', { name: 'Rows per page', exact: true });
          await size.waitFor();
          assert.equal(await size.inputValue(), '50');
          if (mode === 'standalone') {
            assert.equal(
              await app
                .getByRole('button', { name: 'Expand navigation', exact: true })
                .getAttribute('aria-expanded'),
              'false',
            );
          } else
            assert.equal(
              await app.getByRole('complementary', { name: 'Pipelines sidebar' }).count(),
              0,
            );
          if (mode === 'standalone') {
            await app.getByRole('button', { name: /^Theme: / }).click();
            await app.getByRole('menuitemradio', { name: 'Dark', exact: true }).click();
          } else {
            await app.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
          }
          await app.locator('.kfp-theme[data-theme="dark"]').first().waitFor();
        } else {
          assert.equal(values.theme, 'dark');
          const footer = app.getByText('Rows per page:', { exact: true }).locator('..');
          await footer.locator('[role="combobox"]').waitFor();
          assert.equal((await footer.locator('[role="combobox"]').innerText()).trim(), '50');
          if (mode === 'standalone')
            assert.ok((await app.locator('#sideNav').boundingBox()).width < 100);
          else assert.equal(await app.locator('#sideNav').count(), 0);
        }
      });
      await check('preserved runs and schedules are inspectable', async () => {
        for (const run of resources.runs) {
          assert.deepEqual(stableRun(await read(`runs/${run.run_id}`)), run);
          await go(`/runs/details/${run.run_id}`);
          await app.getByText(run.display_name, { exact: true }).first().waitFor();
          await app.goto(app.url());
          await app
            .locator('button')
            .filter({ hasText: /^Detail$/ })
            .click();
          await app.getByText('Succeeded', { exact: true }).first().waitFor();
        }
        for (const schedule of resources.schedules) {
          assert.deepEqual(
            stableSchedule(await read(`recurringruns/${schedule.recurring_run_id}`)),
            schedule,
          );
          await go(`/recurringrun/details/${schedule.recurring_run_id}`);
          await app.getByText(schedule.display_name, { exact: true }).first().waitFor();
        }
      });
      await check('new run creation and logs against unchanged backend', async () => {
        const run = await createRun(pipeline, ownerExperiment.experiment_id, `${prefix}-${phase}`);
        await inspectLogs(run);
      });
      if (phase === 'candidate')
        await check('candidate future schedule creation', () =>
          createRun(pipeline, ownerExperiment.experiment_id, `${prefix}-candidate-schedule`, true),
        );
      await check('existing artifact and old signed TensorBoard access', () => tensorboard());
      if (mode === 'multiuser') await verifyAuthorization();
      await check('backend, configuration, RBAC and signing state preserved', () =>
        python('snapshot'),
      );
      await page.screenshot({ path: resolve(output, `${phase}-application.png`) });
    }
    await Promise.all(pendingAssets);
    await check('each reload uses one verified asset generation', async () => {
      assert.deepEqual(assetErrors, []);
      for (const [name, evidence] of Object.entries(report.phases)) {
        assert.ok(
          evidence.assets.some((file) => file.endsWith('.js')),
          `${name}: no verified JS`,
        );
        assert.ok(
          evidence.assets.some((file) => file.endsWith('.css')),
          `${name}: no verified CSS`,
        );
      }
      assert.deepEqual(pageErrors, []);
    });
    report.resources = resources;
    report.passed = true;
  } catch (error) {
    report.passed = false;
    report.errors.push(sanitize(error.message, secretValues));
    // Authentication pages and redirect URLs can contain credentials. Never capture them.
    if (
      loggedIn &&
      page &&
      page
        .frames()
        .every((frame) => !/\/(dex|oauth2)\//.test(new URL(frame.url() || 'about:blank').pathname))
    ) {
      await page.screenshot({ path: resolve(output, 'failure.png') }).catch(() => {});
    }
    process.exitCode = 1;
  } finally {
    // Cleanup failures must not suppress the qualification report or JUnit result.
    report.cleanup = [];
    for (const [name, action] of [
      ['other browser context', () => otherContext?.close()],
      ['browser', () => browser?.close()],
      [
        'owned standalone proxy',
        () =>
          new Promise((done) => {
            if (!proxy) return done();
            proxy.closeAllConnections();
            proxy.close(done);
          }),
      ],
    ]) {
      let timer;
      try {
        await Promise.race([
          Promise.resolve().then(action),
          new Promise((_, reject) => {
            timer = setTimeout(() => reject(new Error('cleanup exceeded ten seconds')), 10000);
          }),
        ]);
        report.cleanup.push({ name, passed: true });
      } catch (error) {
        report.cleanup.push({ name, passed: false, error: sanitize(error.message, secretValues) });
        report.passed = false;
        process.exitCode = 1;
      } finally {
        clearTimeout(timer);
      }
    }
    report.finishedAt = new Date().toISOString();
    await save();
    const xml = (text) =>
      String(text).replace(
        /[&<>"']/g,
        (char) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&apos;' })[char],
      );
    const cases = report.checks.map(
      (item) =>
        `<testcase name="${xml(item.phase + ': ' + item.name)}">${item.passed ? '' : `<failure message="${xml(item.error)}"/>`}</testcase>`,
    );
    if (!report.passed && !report.checks.some((item) => !item.passed))
      cases.push(
        `<testcase name="setup"><failure message="${xml(report.errors.join('; '))}"/></testcase>`,
      );
    await writeFile(
      resolve(output, 'junit.xml'),
      `<testsuite name="frontend-deployment-${mode}" tests="${cases.length}" failures="${report.passed ? 0 : Math.max(1, report.checks.filter((item) => !item.passed).length)}">${cases.join('')}</testsuite>\n`,
    );
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) await main();
