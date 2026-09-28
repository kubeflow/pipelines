/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Run after npm run build: node --test scripts/ui-modernization-workflows.smoke.mjs
// Local HTTP fixtures exercise the production bundle. They do not qualify cluster authorization.
import assert from 'node:assert/strict';
import { mkdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { after, before, test } from 'node:test';
import { chromium, firefox, webkit } from 'playwright';

const origin = 'http://kfp.test';
const build = new URL('../build/', import.meta.url);
const createdAt = '2026-09-26T12:00:00.000Z';
const parameters = {
  count: { parameterType: 'NUMBER_INTEGER', defaultValue: 0 },
  enabled: { parameterType: 'BOOLEAN', defaultValue: false },
  message: { parameterType: 'STRING', defaultValue: '' },
  choice: { parameterType: 'STRING', defaultValue: '', literals: ['', 'production'] },
  config: { parameterType: 'STRUCT', defaultValue: { nested: false } },
};
const pipelineSpec = {
  pipelineInfo: { name: 'workflows-browser-fixture' },
  root: { inputDefinitions: { parameters }, dag: { tasks: {} } },
  deploymentSpec: { executors: {} },
  schemaVersion: '2.1.0',
  sdkVersion: 'kfp-2.0.0',
};
const pipeline = {
  pipeline_id: 'pipeline-a',
  display_name: 'Workflow pipeline',
  created_at: createdAt,
};
const version = {
  pipeline_id: pipeline.pipeline_id,
  pipeline_version_id: 'version-a',
  display_name: 'Version one',
  created_at: createdAt,
  pipeline_spec: pipelineSpec,
};
const latestVersion = {
  ...version,
  pipeline_version_id: 'version-latest',
  display_name: 'Latest version',
};
const expectedParameters = {
  count: 0,
  enabled: false,
  message: '',
  choice: '',
  config: { nested: false },
};
const runCreationPath =
  '/runs/new?pipelineId=pipeline-a&pipelineVersionId=version-a&experimentId=experiment-a';
let browser;
before(async () => {
  const engineName = process.env.KFP_BROWSER || 'chromium';
  const engine = { chromium, firefox, webkit }[engineName];
  assert.ok(engine, `Unsupported KFP_BROWSER: ${engineName}`);
  browser = await engine.launch({
    channel: engineName === 'chromium' ? process.env.PLAYWRIGHT_CHANNEL || undefined : undefined,
  });
});
after(async () => browser?.close());

function gate() {
  let release;
  const promise = new Promise((resolve) => {
    release = resolve;
  });
  return { promise, release };
}
function decodeFilter(value) {
  return value
    ? JSON.parse(value.startsWith('%') ? decodeURIComponent(value) : value)
    : { predicates: [] };
}
function makeRun(index, experiment = 'experiment-a') {
  return {
    run_id: `run-${index}`,
    display_name: `Training ${index}`,
    experiment_id: experiment,
    created_at: new Date(Date.UTC(2026, 8, 26 - index)).toISOString(),
    finished_at: new Date(Date.UTC(2026, 8, 26 - index, 0, 1)).toISOString(),
    state: index % 2 ? 'SUCCEEDED' : 'FAILED',
    storage_state: 'AVAILABLE',
    pipeline_spec: pipelineSpec,
    pipeline_version_reference: { pipeline_id: 'pipeline-a', pipeline_version_id: 'version-a' },
    run_details: { task_details: [] },
  };
}
async function withFixture(options, exercise) {
  const context = await browser.newContext({
    viewport: { width: 1440, height: 960 },
    locale: 'en-US',
    timezoneId: 'UTC',
    reducedMotion: 'reduce',
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  await page.clock.install({ time: new Date('2026-09-26T12:02:00.000Z') });
  const fixture = {
    experiments: [
      {
        experiment_id: 'experiment-a',
        display_name: 'Training experiment',
        description: 'First line\nSecond line\nFull experiment description',
        namespace: 'team-a',
        storage_state: 'AVAILABLE',
        created_at: createdAt,
      },
      {
        experiment_id: 'experiment-b',
        display_name: 'Small experiment',
        description: 'Two recent runs',
        namespace: 'team-b',
        storage_state: 'AVAILABLE',
        created_at: createdAt,
      },
      {
        experiment_id: 'experiment-archived',
        display_name: 'Archived experiment',
        namespace: 'team-a',
        storage_state: 'ARCHIVED',
        created_at: createdAt,
      },
    ],
    runs: [
      ...Array.from({ length: 7 }, (_, i) => makeRun(i + 1)),
      makeRun(8, 'experiment-b'),
      makeRun(9, 'experiment-b'),
    ],
    schedules: [
      {
        recurring_run_id: 'schedule-a',
        display_name: 'Nightly training',
        experiment_id: 'experiment-a',
        namespace: 'team-a',
        status: 'ENABLED',
        created_at: createdAt,
        max_concurrency: '2',
        no_catchup: false,
        trigger: { periodic_schedule: { interval_second: '3600' } },
        pipeline_version_reference: { pipeline_id: 'pipeline-a', pipeline_version_id: 'version-a' },
        pipeline_spec: pipelineSpec,
      },
    ],
    requests: [],
    mutations: [],
    errors: [],
    failNext: new Set(),
    held: new Map(),
  };
  page.on('pageerror', (error) => fixture.errors.push(error.message));
  await context.addInitScript(({ namespace }) => {
    if (!localStorage.getItem('kfp.theme')) localStorage.setItem('kfp.theme', 'light');
    if (namespace) {
      window.centraldashboard = {
        CentralDashboardEventHandler: {
          init(callback) {
            const handler = {};
            callback(handler);
            handler.onNamespaceSelected(namespace);
            window.selectFixtureNamespace = (value) => handler.onNamespaceSelected(value);
          },
        },
      };
    }
  }, options);
  await context.route('**/*', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();
    const query = Object.fromEntries(url.searchParams);
    const json = (body, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    try {
      assert.equal(url.origin, origin, 'requests must stay within the fixture origin');
      assert.ok(fixture.requests.length < 250, 'request count must remain bounded');
      fixture.requests.push({ method, path, query });
      if (path === '/' || path.startsWith('/static/')) {
        const name = path === '/' ? 'index.html' : path.slice(1);
        let body = await readFile(new URL(name, build));
        if (path === '/')
          body = body
            .toString()
            .replace(
              /window\.KFP_FLAGS\.DEPLOYMENT\s*=\s*null;?/,
              `window.KFP_FLAGS.DEPLOYMENT=${JSON.stringify(options.namespace ? 'KUBEFLOW' : null)};`,
            );
        const contentType = name.endsWith('.js')
          ? 'text/javascript'
          : name.endsWith('.css')
            ? 'text/css'
            : name.endsWith('.html')
              ? 'text/html'
              : 'application/octet-stream';
        return route.fulfill({ body, contentType });
      }
      if (path === '/apis/v2beta1/healthz') return json({ apiServerTagName: 'fixture' });
      if (path === '/system/cluster-name' || path === '/system/project-id')
        return route.fulfill({ body: '' });
      if (method === 'POST') {
        const action = `${method} ${path}`;
        const body = request.postData() ? request.postDataJSON() : null;
        fixture.mutations.push({ path, body });
        if (fixture.held.has(action)) await fixture.held.get(action).promise;
        if (fixture.failNext.delete(action))
          return json({ code: 14, message: 'Fixture mutation temporarily unavailable' }, 503);
        if (path === '/apis/v2beta1/experiments') {
          const experiment = {
            ...body,
            experiment_id: 'experiment-created',
            storage_state: 'AVAILABLE',
            created_at: createdAt,
          };
          fixture.experiments.push(experiment);
          return json(experiment);
        }
        if (path === '/apis/v2beta1/runs') {
          const run = { ...makeRun(20), ...body, run_id: 'run-created', state: 'SUCCEEDED' };
          fixture.runs.push(run);
          return json(run);
        }
        if (path === '/apis/v2beta1/recurringruns') {
          const schedule = {
            ...body,
            recurring_run_id: 'schedule-created',
            status: 'ENABLED',
            created_at: createdAt,
          };
          fixture.schedules.push(schedule);
          return json(schedule);
        }
        const scheduleAction = /^\/apis\/v2beta1\/recurringruns\/([^/:]+):(enable|disable)$/.exec(
          path,
        );
        if (scheduleAction) {
          assert.equal(body, null, 'schedule toggles have no request body');
          const schedule = fixture.schedules.find(
            (item) => item.recurring_run_id === scheduleAction[1],
          );
          assert.ok(schedule);
          schedule.status = scheduleAction[2] === 'enable' ? 'ENABLED' : 'DISABLED';
          return json({});
        }
        const runAction = /^\/apis\/v2beta1\/runs\/([^/:]+):(archive|unarchive)$/.exec(path);
        if (runAction) {
          assert.equal(body, null, 'run archive actions have no body');
          const run = fixture.runs.find((item) => item.run_id === runAction[1]);
          assert.ok(run);
          run.storage_state = runAction[2] === 'archive' ? 'ARCHIVED' : 'AVAILABLE';
          return json({});
        }
      }
      assert.equal(method, 'GET', `Unexpected method for ${path}`);
      if (path === '/apis/v2beta1/experiments') {
        const predicates = decodeFilter(query.filter).predicates || [];
        let experiments = fixture.experiments.filter(
          (item) => !query.namespace || item.namespace === query.namespace,
        );
        for (const predicate of predicates) {
          if (predicate.key === 'storage_state') {
            assert.equal(predicate.string_value, 'ARCHIVED');
            assert.ok(['EQUALS', 'NOT_EQUALS'].includes(predicate.operation));
            experiments = experiments.filter(
              (item) => (item.storage_state === 'ARCHIVED') === (predicate.operation === 'EQUALS'),
            );
          } else {
            assert.equal(predicate.key, 'name');
            assert.equal(predicate.operation, 'IS_SUBSTRING');
            experiments = experiments.filter((item) =>
              item.display_name.toLowerCase().includes(predicate.string_value.toLowerCase()),
            );
          }
        }
        return json({ experiments, total_size: experiments.length });
      }
      const experimentMatch = /^\/apis\/v2beta1\/experiments\/([^/]+)$/.exec(path);
      if (experimentMatch) {
        const experiment = fixture.experiments.find(
          (item) => item.experiment_id === decodeURIComponent(experimentMatch[1]),
        );
        assert.ok(experiment);
        return json(experiment);
      }
      if (path === '/apis/v2beta1/runs') {
        assert.equal(query.skip_count, 'true');
        const predicates = decodeFilter(query.filter).predicates || [];
        let runs = fixture.runs.filter(
          (item) => !query.experiment_id || item.experiment_id === query.experiment_id,
        );
        if (query.namespace)
          runs = runs.filter((item) =>
            fixture.experiments.some(
              (experiment) =>
                experiment.experiment_id === item.experiment_id &&
                experiment.namespace === query.namespace,
            ),
          );
        if (query.page_size === '5') {
          assert.ok(query.experiment_id, 'recent-run samples are scoped to one experiment');
          assert.equal(query.sort_by, 'created_at desc');
          assert.ok(!query.page_token, 'recent-run samples must not fetch further pages');
        }
        for (const predicate of predicates) {
          if (predicate.key === 'storage_state') {
            assert.equal(predicate.string_value, 'ARCHIVED');
            runs = runs.filter(
              (item) => (item.storage_state === 'ARCHIVED') === (predicate.operation === 'EQUALS'),
            );
          } else {
            assert.equal(predicate.key, 'name');
            assert.equal(predicate.operation, 'IS_SUBSTRING');
            runs = runs.filter((item) => item.display_name.includes(predicate.string_value));
          }
        }
        runs.sort((a, b) => b.created_at.localeCompare(a.created_at));
        return json({ runs: runs.slice(0, Number(query.page_size || 100)) });
      }
      const runMatch = /^\/apis\/v2beta1\/runs\/([^/]+)(\/tasks)?$/.exec(path);
      if (runMatch) {
        const run = fixture.runs.find((item) => item.run_id === runMatch[1]);
        assert.ok(run);
        return json(runMatch[2] ? { tasks: [] } : run);
      }
      if (path === '/apis/v2beta1/recurringruns') {
        const schedules = fixture.schedules.filter(
          (item) =>
            (!query.namespace || item.namespace === query.namespace) &&
            (!query.experiment_id || item.experiment_id === query.experiment_id),
        );
        return json({ recurringRuns: schedules, total_size: schedules.length });
      }
      const scheduleMatch = /^\/apis\/v2beta1\/recurringruns\/([^/]+)$/.exec(path);
      if (scheduleMatch) {
        const schedule = fixture.schedules.find(
          (item) => item.recurring_run_id === scheduleMatch[1],
        );
        assert.ok(schedule);
        return json(schedule);
      }
      if (path === '/apis/v2beta1/pipelines') {
        if (options.secondaryRoutes && query.filter) {
          const predicates = decodeFilter(query.filter).predicates;
          assert.equal(query.page_size, '10');
          assert.equal(predicates.length, 1);
          assert.equal(predicates[0].key, 'name');
          assert.equal(predicates[0].operation, 'EQUALS');
          assert.ok(!query.page_token, 'tutorial lookup must not scan pages');
          if (predicates[0].string_value === '[Tutorial] Data passing in python components')
            return json({ pipelines: [{ ...pipeline, display_name: predicates[0].string_value }] });
          assert.equal(predicates[0].string_value, '[Tutorial] DSL - Control structures');
          return json({ pipelines: [] });
        }
        return json({ pipelines: [pipeline] });
      }
      if (path === '/apis/v2beta1/pipelines/pipeline-a') return json(pipeline);
      if (path === '/apis/v2beta1/pipelines/pipeline-a/versions') {
        if (query.page_size === '1') assert.equal(query.sort_by, 'created_at desc');
        return json({
          pipeline_versions: [latestVersion, version].slice(0, Number(query.page_size || 2)),
        });
      }
      if (path === '/apis/v2beta1/pipelines/pipeline-a/versions/version-a') return json(version);
      if (path === '/apis/v2beta1/pipelines/pipeline-a/versions/version-latest')
        return json(latestVersion);
      if (path === '/apis/v2beta1/artifacts' || path === '/apis/v2beta1/artifact_tasks')
        return json({});
      throw new Error(`Unexpected fixture request: ${method} ${path}`);
    } catch (error) {
      fixture.errors.push(error.message);
      await json({ message: error.message }, 500);
    }
  });
  try {
    await exercise(page, fixture);
    assert.deepEqual(
      fixture.errors,
      [],
      'bundle and HTTP contracts must have no unexpected errors',
    );
  } catch (error) {
    if (fixture.errors.length) console.error('Fixture errors:', fixture.errors);
    throw error;
  } finally {
    for (const held of fixture.held.values()) held.release();
    await context.close();
  }
}

async function tableReady(page, name) {
  await page.getByRole('table', { name, exact: true }).waitFor();
  await page.waitForFunction(
    (label) =>
      document.querySelector(`table[aria-label="${label}"]`)?.getAttribute('aria-busy') === 'false',
    name,
  );
}
async function focused(page, locator) {
  const handle = await locator.elementHandle();
  assert.ok(handle);
  try {
    await page.waitForFunction((node) => node === document.activeElement, handle);
  } finally {
    await handle.dispose();
  }
}
async function screenshot(page, name) {
  const directory = process.env.KFP_WORKFLOWS_SCREENSHOT_DIR;
  if (!directory) return;
  await mkdir(directory, { recursive: true });
  await page.evaluate(() => {
    window.scrollTo(0, 0);
    document.querySelectorAll('.kfp-modern-page').forEach((element) => {
      element.scrollTop = 0;
    });
  });
  await page.screenshot({ path: join(directory, `${name}.png`) });
}
async function theme(page, value) {
  await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption(value);
  await page.locator(`.kfp-theme[data-theme="${value}"]`).waitFor();
  // The theme attribute changes before inherited descendant colors settle.
  await page.waitForFunction((theme) => {
    const dark = theme === 'dark';
    const foreground = dark ? 'rgb(236, 238, 244)' : 'rgb(21, 23, 30)';
    const background = dark ? 'rgb(14, 16, 21)' : 'rgb(245, 246, 249)';
    const card = dark ? 'rgb(22, 25, 33)' : 'rgb(255, 255, 255)';
    const shell = document.querySelector('.kfp-shell');
    const brand = document.querySelector('.kfp-shell-brand-title');
    return (
      getComputedStyle(shell).backgroundColor === background &&
      getComputedStyle(brand).color === foreground &&
      [...document.querySelectorAll('.kfp-new-run-fields,.kfp-run-form-summary')].every(
        (element) => getComputedStyle(element).backgroundColor === card,
      )
    );
  }, value);
}
async function newRun(page, suffix = '') {
  await page.goto(`${origin}/#${runCreationPath}${suffix}`);
  await page.getByLabel('count - integer', { exact: true }).waitFor();
  assert.equal(await page.getByLabel('count - integer', { exact: true }).inputValue(), '0');
  assert.equal(await page.getByLabel('enabled - boolean', { exact: true }).inputValue(), 'false');
  assert.equal(await page.getByLabel('message - string', { exact: true }).inputValue(), '');
}
function mutation(fixture, path) {
  return fixture.mutations.filter((entry) => entry.path === path);
}
async function waitMutation(page, action) {
  const received = page.waitForRequest(
    (request) => request.method() === 'POST' && new URL(request.url()).pathname === action,
  );
  return received;
}
async function submit(page, path) {
  const received = waitMutation(page, path);
  await page.locator('#startNewRunBtn').click();
  await received;
}
function assertRunBody(body, name, versionId = 'version-a', experimentId = 'experiment-a') {
  assert.equal(body.display_name, name);
  assert.equal(body.experiment_id, experimentId);
  assert.deepEqual(
    body.pipeline_version_reference,
    versionId
      ? { pipeline_id: 'pipeline-a', pipeline_version_id: versionId }
      : { pipeline_id: 'pipeline-a' },
  );
  assert.equal(
    body.pipeline_spec,
    undefined,
    'version-reference creation must not embed a duplicate pipeline spec',
  );
  assert.deepEqual(body.runtime_config.parameters, expectedParameters);
}

// The sample is deliberately seven runs for one experiment and two for another: labels must
// describe the fetched sample, while expanding still loads the complete nested page.
test('Experiments: bounded recent-run samples, expansion, archive navigation, namespace and themes', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    await page.goto(`${origin}/#/experiments`);
    await tableReady(page, 'Experiments');
    await page.getByRole('img', { name: /^Last 5 runs:/ }).waitFor();
    assert.equal(
      fixture.requests.filter((r) => r.path === '/apis/v2beta1/runs' && r.query.page_size === '5')
        .length,
      1,
    );
    await screenshot(page, 'experiments-light');
    const expand = page.getByRole('button', {
      name: 'Expand experiment Training experiment',
      exact: true,
    });
    await expand.focus();
    await page.keyboard.press('Enter');
    await tableReady(page, 'Runs');
    assert.equal(await page.getByTestId('run-name-link').count(), 7);
    await page.getByRole('checkbox', { name: 'Select run Training 1', exact: true }).check();
    assert.equal(
      await page.getByRole('button', { name: 'Archive', exact: true }).isEnabled(),
      true,
    );
    assert.match(page.url(), /#\/experiments$/);
    await theme(page, 'dark');
    await screenshot(page, 'experiments-dark-expanded');
    await page.getByRole('tab', { name: 'Archived', exact: true }).click();
    await tableReady(page, 'Archived experiments');
    await page.getByRole('link', { name: 'Archived experiment', exact: true }).waitFor();
    assert.match(page.url(), /#\/archive\/experiments$/);
    await page.getByRole('tab', { name: 'Active', exact: true }).click();
    await tableReady(page, 'Experiments');
    await page.evaluate(() => window.selectFixtureNamespace('team-b'));
    await page.getByRole('link', { name: 'Small experiment', exact: true }).waitFor();
    await tableReady(page, 'Experiments');
    await page.getByRole('img', { name: /^Last 2 runs:/ }).waitFor();
    assert.equal(
      await page.getByRole('link', { name: 'Training experiment', exact: true }).count(),
      0,
    );
    assert.ok(
      fixture.requests.some(
        (r) => r.path === '/apis/v2beta1/experiments' && r.query.namespace === 'team-b',
      ),
    );
    assert.equal(fixture.mutations.length, 0, 'selecting a nested run must not mutate it');
  });
});

test('Recurring workflows: keyboard toggle failure/recovery, detail navigation and manager mutation', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    await page.goto(`${origin}/#/recurringruns`);
    await tableReady(page, 'Recurring runs');
    const action = '/apis/v2beta1/recurringruns/schedule-a:disable';
    const held = gate();
    fixture.held.set(`POST ${action}`, held);
    fixture.failNext.add(`POST ${action}`);
    const control = page.getByRole('switch', { name: 'Enable schedule Nightly training' });
    const firstToggle = waitMutation(page, action);
    await control.focus();
    await page.keyboard.press('Space');
    await firstToggle;
    assert.equal(await control.isDisabled(), true);
    await page.keyboard.press('Space');
    assert.equal(mutation(fixture, action).length, 1);
    held.release();
    fixture.held.delete(`POST ${action}`);
    await page.getByRole('alert').filter({ hasText: 'Unable to update schedule' }).waitFor();
    assert.equal(await control.isChecked(), true);
    await control.focus();
    await page.keyboard.press('Space');
    await page.waitForFunction(
      () => document.querySelector('[role="switch"]')?.getAttribute('aria-checked') === 'false',
    );
    await tableReady(page, 'Recurring runs');
    assert.equal(mutation(fixture, action).length, 2);
    assert.equal(
      await page.getByRole('alert').filter({ hasText: 'Unable to update schedule' }).count(),
      0,
    );
    await theme(page, 'dark');
    await screenshot(page, 'recurring-dark');
    await page.getByRole('link', { name: 'Nightly training', exact: true }).click();
    await page.waitForURL(/#\/recurringrun\/details\/schedule-a$/);
    await page.getByRole('heading', { name: 'Nightly training', exact: true }).waitFor();
    await page.goto(`${origin}/#/experiments/details/experiment-a`);
    const manage = page.locator('#manageExperimentRecurringRunsBtn');
    await manage.click();
    const dialog = page.getByRole('dialog', { name: 'Recurring run configs', exact: true });
    await dialog.waitFor();
    await dialog.getByRole('switch', { name: 'Enable schedule Nightly training' }).click();
    await page.waitForFunction(
      () =>
        document.querySelector('[role="dialog"] [role="switch"]')?.getAttribute('aria-checked') ===
        'true',
    );
    assert.equal(mutation(fixture, '/apis/v2beta1/recurringruns/schedule-a:enable').length, 1);
    await dialog.locator('#closeExperimentRecurringRunManagerBtn').click();
    await dialog.waitFor({ state: 'hidden' });
    await focused(page, manage);
    await page
      .getByRole('region', { name: 'Recurring run configs' })
      .getByText('1 active', { exact: true })
      .waitFor();
    await screenshot(page, 'experiment-detail-dark');
  });
});

test('New experiment: validation, held failure/retry, namespace and latest-version handoff', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    await page.goto(`${origin}/#/experiments/new?pipelineId=pipeline-a`);
    const next = page.locator('#createExperimentBtn');
    await page.getByRole('textbox', { name: 'Experiment name', exact: true }).waitFor();
    assert.equal(await next.isDisabled(), true);
    await page
      .getByRole('textbox', { name: 'Experiment name', exact: true })
      .fill('Created experiment');
    await page.getByLabel('Description', { exact: true }).fill('Browser-created description');
    const action = '/apis/v2beta1/experiments';
    const held = gate();
    fixture.held.set(`POST ${action}`, held);
    fixture.failNext.add(`POST ${action}`);
    const request = waitMutation(page, action);
    await next.click();
    await request;
    assert.equal(await next.isDisabled(), true);
    await next.press('Enter');
    assert.equal(mutation(fixture, action).length, 1);
    held.release();
    fixture.held.delete(`POST ${action}`);
    // The shared generated client's ResponseError currently exposes this generic message.
    const dialog = page.getByRole('dialog', { name: 'Experiment creation failed', exact: true });
    await dialog.getByText('Response returned an error code', { exact: true }).waitFor();
    await dialog.getByRole('button', { name: 'Dismiss', exact: true }).click();
    await next.click();
    await page.waitForURL(/#\/runs\/new\?/);
    await page.getByLabel('count - integer', { exact: true }).waitFor();
    const query = new URL(page.url().split('#')[1], origin).searchParams;
    assert.equal(query.get('experimentId'), 'experiment-created');
    assert.equal(query.get('pipelineId'), 'pipeline-a');
    assert.equal(query.get('pipelineVersionId'), 'version-latest');
    assert.equal(query.get('firstRunInExperiment'), '1');
    assert.equal(
      await page.getByRole('textbox', { name: 'Experiment', exact: true }).inputValue(),
      'Created experiment',
    );
    assert.equal(mutation(fixture, action).length, 2);
    assert.deepEqual(mutation(fixture, action)[1].body, {
      display_name: 'Created experiment',
      description: 'Browser-created description',
      namespace: 'team-a',
    });
    assert.equal(fixture.mutations.filter((entry) => entry.path !== action).length, 0);
  });
});

test('One-off creation: typed falsy parameters, validation, responsive themes and one POST per attempt', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    await newRun(page);
    const start = page.locator('#startNewRunBtn');
    await page.getByRole('textbox', { name: 'Run name', exact: true }).fill('');
    assert.equal(await start.isDisabled(), true);
    await page.getByRole('textbox', { name: 'Run name', exact: true }).fill('One-off browser run');
    await page.getByRole('radio', { name: 'Recurring', exact: true }).check();
    await page.getByRole('radio', { name: 'One-off', exact: true }).check();
    await page.getByLabel('Description', { exact: true }).fill('Retained after failure');
    await page.getByLabel('Service Account', { exact: true }).fill('pipeline-runner');
    await page.getByLabel('count - integer', { exact: true }).fill('1.5');
    assert.equal(await start.isDisabled(), true);
    assert.equal(fixture.mutations.length, 0);
    await page.getByLabel('count - integer', { exact: true }).fill('0');
    await page.getByLabel('enabled - boolean', { exact: true }).fill('false');
    await page.getByLabel('message - string', { exact: true }).fill('temporary');
    await page.getByLabel('message - string', { exact: true }).fill('');
    await page.getByLabel('choice - string', { exact: true }).selectOption({ label: 'production' });
    await page
      .getByLabel('choice - string', { exact: true })
      .selectOption({ label: 'Empty string' });
    await page.getByLabel('config - dict', { exact: true }).fill('invalid');
    assert.equal(await start.isDisabled(), true);
    await page.getByLabel('config - dict', { exact: true }).fill('{"nested":false}');
    await page.getByRole('button', { name: 'Open Json Editor', exact: true }).click();
    await page.locator('.ace_editor').waitFor();
    assert.equal(await page.getByLabel('config - dict', { exact: true }).isDisabled(), true);
    await page.getByRole('button', { name: 'Close Json Editor', exact: true }).click();
    const customRoot = page.getByRole('checkbox', {
      name: /Custom Pipeline Root$/,
    });
    await customRoot.focus();
    await page.keyboard.press('Space');
    await page.getByLabel('pipeline-root', { exact: true }).fill('s3://fixture-artifacts/browser');
    await screenshot(page, 'new-run-light');
    await theme(page, 'dark');
    await screenshot(page, 'new-run-dark');
    await page.setViewportSize({ width: 390, height: 844 });
    await screenshot(page, 'new-run-narrow-dark');
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1),
      true,
      'narrow form must not overflow the viewport',
    );
    await page.setViewportSize({ width: 1440, height: 960 });
    const action = '/apis/v2beta1/runs';
    const held = gate();
    fixture.held.set(`POST ${action}`, held);
    fixture.failNext.add(`POST ${action}`);
    await submit(page, action);
    assert.equal(await start.isDisabled(), true);
    await start.press('Enter');
    assert.equal(mutation(fixture, action).length, 1);
    held.release();
    fixture.held.delete(`POST ${action}`);
    const dialog = page.getByRole('dialog', { name: 'Run creation failed', exact: true });
    await dialog.getByText('Response returned an error code', { exact: true }).waitFor();
    await dialog.getByRole('button', { name: 'Dismiss', exact: true }).click();
    assert.equal(
      await page.getByRole('textbox', { name: 'Run name', exact: true }).inputValue(),
      'One-off browser run',
    );
    await submit(page, action);
    await page.waitForURL(/#\/runs\/details\/run-created$/);
    assert.equal(mutation(fixture, action).length, 2);
    const body = mutation(fixture, action)[1].body;
    assertRunBody(body, 'One-off browser run');
    assert.equal(body.runtime_config.pipeline_root, 's3://fixture-artifacts/browser');
    assert.equal(body.description, 'Retained after failure');
    assert.equal(body.service_account, 'pipeline-runner');
    for (const key of ['mode', 'max_concurrency', 'no_catchup', 'trigger'])
      assert.equal(body[key], undefined, `one-off payload excludes ${key}`);
    assert.equal(mutation(fixture, '/apis/v2beta1/recurringruns').length, 0);
  });
});

test('Recurring creation: periodic bounds, concurrency validation, catchup and latest-version payload', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    await newRun(page, '&recurring=1');
    await page
      .getByRole('textbox', { name: 'Recurring run config name', exact: true })
      .fill('Periodic browser schedule');
    await page
      .getByRole('checkbox', { name: 'Always use the latest pipeline version', exact: true })
      .check();
    await page.getByRole('combobox', { name: 'Interval unit', exact: true }).selectOption('Hour');
    await page.getByRole('spinbutton', { name: 'Interval', exact: true }).fill('2');
    await page.getByRole('textbox', { name: 'Maximum concurrent runs', exact: true }).fill('0');
    assert.equal(await page.locator('#startNewRunBtn').isDisabled(), true);
    await page.getByRole('textbox', { name: 'Maximum concurrent runs', exact: true }).fill('3');
    await page.getByRole('checkbox', { name: 'Catchup', exact: true }).uncheck();
    await page.getByRole('checkbox', { name: 'Has start date', exact: true }).check();
    await page.getByLabel('Start date', { exact: true }).fill('2026-10-01');
    await page.getByLabel('Start time', { exact: true }).fill('08:30');
    await page.getByRole('checkbox', { name: 'Has end date', exact: true }).check();
    await page.getByLabel('End date', { exact: true }).fill('2026-10-31');
    await page.getByLabel('End time', { exact: true }).fill('18:45');
    await theme(page, 'dark');
    await screenshot(page, 'new-recurring-dark');
    await submit(page, '/apis/v2beta1/recurringruns');
    await page.waitForURL(/#\/recurringrun\/details\/schedule-created$/);
    assert.equal(mutation(fixture, '/apis/v2beta1/recurringruns').length, 1);
    assert.equal(mutation(fixture, '/apis/v2beta1/runs').length, 0);
    const body = mutation(fixture, '/apis/v2beta1/recurringruns')[0].body;
    assertRunBody(body, 'Periodic browser schedule', null);
    assert.equal(body.mode, 'ENABLE');
    assert.equal(body.max_concurrency, '3');
    assert.equal(body.no_catchup, true);
    assert.deepEqual(body.trigger, {
      periodic_schedule: {
        start_time: '2026-10-01T08:30:00.000Z',
        end_time: '2026-10-31T18:45:00.000Z',
        interval_second: '7200',
      },
    });
  });
});

test('Inline experiment and cron creation preserve selected pipeline, typed parameters and single mutation', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    await newRun(page);
    await page.locator('#chooseExperimentBtn').click();
    const chooser = page.getByRole('dialog', { name: 'Choose an experiment', exact: true });
    await chooser.getByRole('button', { name: 'Create new experiment', exact: true }).click();
    const editor = page.getByRole('dialog', { name: 'New experiment', exact: true });
    await editor
      .getByRole('textbox', { name: 'Experiment name', exact: true })
      .fill('Inline experiment');
    await editor.locator('#createExperimentBtn').click();
    await editor.waitFor({ state: 'hidden' });
    await page.getByLabel('count - integer', { exact: true }).waitFor();
    assert.equal(
      await page.getByRole('textbox', { name: 'Experiment', exact: true }).inputValue(),
      'Inline experiment',
    );
    assert.equal(
      await page.getByRole('textbox', { name: 'Pipeline', exact: true }).inputValue(),
      'Workflow pipeline',
    );
    assert.equal(mutation(fixture, '/apis/v2beta1/experiments').length, 1);
    await page.getByRole('radio', { name: 'Recurring', exact: true }).check();
    await page
      .getByRole('textbox', { name: 'Recurring run config name', exact: true })
      .fill('Cron browser schedule');
    await page
      .getByRole('combobox', { name: 'Trigger type', exact: true })
      .selectOption({ label: 'Cron' });
    await page.getByRole('combobox', { name: 'Interval unit', exact: true }).selectOption('Week');
    const monday = page.getByRole('button', { name: 'Monday', exact: true });
    await monday.focus();
    await page.keyboard.press('Space');
    assert.equal(await monday.getAttribute('aria-pressed'), 'false');
    await page
      .getByRole('checkbox', { name: 'Allow editing cron expression.', exact: true })
      .check();
    await page.getByLabel('cron expression', { exact: true }).fill('0 15 8 ? * 1-5');
    await submit(page, '/apis/v2beta1/recurringruns');
    await page.waitForURL(/#\/recurringrun\/details\/schedule-created$/);
    const body = mutation(fixture, '/apis/v2beta1/recurringruns')[0].body;
    assertRunBody(body, 'Cron browser schedule', 'version-latest', 'experiment-created');
    assert.equal(body.mode, 'ENABLE');
    assert.equal(body.max_concurrency, '10');
    assert.equal(body.no_catchup, false);
    assert.deepEqual(body.trigger, { cron_schedule: { cron: '0 15 8 ? * 1-5' } });
    assert.equal(mutation(fixture, '/apis/v2beta1/recurringruns').length, 1);
    assert.equal(mutation(fixture, '/apis/v2beta1/runs').length, 0);
  });
});

test('Secondary routes preserve tutorial links, feature draft save/reset/persistence, theme and keyboard navigation', async () => {
  await withFixture({ secondaryRoutes: true }, async (page, fixture) => {
    await page.goto(`${origin}/#/start`);
    await page
      .getByRole('heading', { name: 'Build your own pipeline with', exact: true })
      .waitFor();
    const dataTutorial = page.getByRole('link', {
      name: 'Data passing in Python components',
      exact: true,
    });
    const controlTutorial = page.getByRole('link', {
      name: 'DSL - Control structures',
      exact: true,
    });
    await page.waitForFunction(() =>
      [...document.querySelectorAll('a')].some(
        (link) =>
          link.textContent === 'Data passing in Python components' &&
          link.getAttribute('href')?.split('?')[0] === '#/pipelines/details/pipeline-a',
      ),
    );
    assert.equal(
      await dataTutorial.getAttribute('target'),
      null,
      'tutorial navigation stays in the app',
    );
    assert.equal(
      await controlTutorial.getAttribute('href'),
      '#/pipelines',
      'missing tutorials retain the pipeline-list fallback',
    );
    const sdk = page.getByRole('link', { name: 'SDK', exact: true });
    assert.equal(await sdk.getAttribute('target'), '_blank');
    assert.match(await sdk.getAttribute('rel'), /noopener/);
    assert.equal(
      fixture.requests.filter(
        (request) => request.path === '/apis/v2beta1/pipelines' && request.query.filter,
      ).length,
      2,
    );
    const runtimeFlags = await page.evaluate(() => JSON.stringify(window.KFP_FLAGS));
    await theme(page, 'dark');
    await screenshot(page, 'getting-started-dark');
    await dataTutorial.focus();
    await page.keyboard.press('Enter');
    await page.waitForURL((url) => url.hash.split('?')[0] === '#/pipelines/details/pipeline-a');
    await page.getByTestId('DagCanvas').waitFor();

    await page.goto(`${origin}/#/frontend_features`);
    const feature = page.getByRole('switch', { name: 'Enable functional_component', exact: true });
    await feature.waitFor();
    assert.equal(await feature.isChecked(), false);
    const originalFlags = await page.evaluate(() => ({
      stored: localStorage.getItem('flags'),
      runtime: window.__FEATURE_FLAGS__,
    }));
    await feature.focus();
    await page.keyboard.press('Space');
    assert.equal(await feature.isChecked(), true);
    assert.deepEqual(
      await page.evaluate(() => ({
        stored: localStorage.getItem('flags'),
        runtime: window.__FEATURE_FLAGS__,
      })),
      originalFlags,
      'editing a draft must not persist or activate it',
    );
    await page.getByRole('button', { name: 'Reset', exact: true }).click();
    assert.equal(await feature.isChecked(), false);
    await feature.focus();
    await page.keyboard.press('Space');
    const save = page.getByRole('button', { name: 'Save changes', exact: true });
    await save.focus();
    await page.keyboard.press('Enter');
    await page.waitForFunction(
      () =>
        JSON.parse(localStorage.getItem('flags')).find(
          (flag) => flag.name === 'functional_component',
        ).active === true,
    );
    assert.equal(
      await page.evaluate(
        () =>
          JSON.parse(window.__FEATURE_FLAGS__).find((flag) => flag.name === 'functional_component')
            .active,
      ),
      true,
    );
    await page.reload();
    await feature.waitFor();
    assert.equal(await feature.isChecked(), true, 'saved preferences survive a full reload');
    assert.equal(
      await page.getByRole('combobox', { name: 'Theme', exact: true }).inputValue(),
      'dark',
    );
    await feature.focus();
    await page.keyboard.press('Space');
    assert.equal(await feature.isChecked(), false);
    await page.getByRole('button', { name: 'Reset', exact: true }).click();
    assert.equal(
      await feature.isChecked(),
      true,
      'reset restores the saved preference rather than the default',
    );
    await screenshot(page, 'frontend-features-dark');
    await feature.focus();
    await page.keyboard.press('Space');
    await save.click();
    assert.equal(
      await page.evaluate(() => JSON.stringify(window.KFP_FLAGS)),
      runtimeFlags,
      'frontend preferences must not rewrite deployment flags',
    );

    await page.goto(`${origin}/#/missing-browser-fixture`);
    await page.getByRole('heading', { name: 'Page not found', exact: true }).waitFor();
    await page.getByText('/missing-browser-fixture', { exact: true }).waitFor();
    await page.setViewportSize({ width: 600, height: 900 });
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
      true,
    );
    await screenshot(page, 'not-found-dark-narrow');
    const returnToPipelines = page.getByRole('button', { name: 'Go to pipelines', exact: true });
    await returnToPipelines.focus();
    await page.keyboard.press('Enter');
    await page.waitForURL(`**/#/pipelines`);
    await page.getByRole('link', { name: 'Workflow pipeline', exact: true }).waitFor();
    assert.equal(fixture.mutations.length, 0, 'secondary routes only change browser preferences');
  });
});
