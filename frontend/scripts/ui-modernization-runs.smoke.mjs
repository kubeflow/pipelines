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

// Run after npm run build: node --test scripts/ui-modernization-runs.smoke.mjs
// Fixed HTTP fixtures exercise the production bundle; no cluster or external network is used.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { after, before, test } from 'node:test';
import { chromium } from 'playwright';

const origin = 'http://kfp.test';
const build = new URL('../build/', import.meta.url);
const pipelineSpec = {
  pipelineInfo: { name: 'runs-browser-fixture' },
  root: { dag: { tasks: {} } },
  schemaVersion: '2.1.0',
  sdkVersion: 'kfp-2.0.0',
};
const pipelineVersion = {
  pipeline_id: 'pipeline-a',
  pipeline_version_id: 'version-a',
  display_name: 'Version one',
  pipeline_spec: pipelineSpec,
};
const states = ['SUCCEEDED', 'FAILED', 'RUNNING', 'PENDING', 'CANCELED', 'SKIPPED', 'PAUSED'];
const runId = (index) => `run-${String(index).padStart(2, '0')}`;
const runName = (index) => `Training ${String(index).padStart(2, '0')}`;
let browser;
before(async () => {
  browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
});
after(async () => browser?.close());

function makeRuns() {
  return Array.from({ length: 18 }, (_, offset) => ({
    run_id: runId(offset + 1),
    display_name: runName(offset + 1),
    experiment_id: 'experiment-a',
    storage_state: offset < 15 ? 'AVAILABLE' : 'ARCHIVED',
    state: offset < 15 ? states[offset % states.length] : 'SUCCEEDED',
    created_at: new Date(Date.UTC(2026, 0, 20 - offset)).toISOString(),
    finished_at: new Date(Date.UTC(2026, 0, 20 - offset, 0, 1)).toISOString(),
    pipeline_version_reference: {
      pipeline_id: 'pipeline-a',
      pipeline_version_id: offset === 7 ? 'deleted-version' : 'version-a',
    },
    pipeline_spec: pipelineSpec,
    run_details: { task_details: [] },
  }));
}

function decodeFilter(value) {
  // Existing clients encode the filter JSON before the generated query-string encoder.
  return JSON.parse(value.startsWith('%') ? decodeURIComponent(value) : value);
}

async function withFixture(options, exercise) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  const fixture = {
    runs: makeRuns(),
    lists: [],
    searches: [],
    mutations: [],
    versionReads: [],
    errors: [],
    forbidden: new Set(),
    failNextList: options.failNextList || false,
  };
  let requests = 0;
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
    const path = url.pathname.replace(/^\/pipelines(?=\/)/, '');
    const json = (body, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    try {
      assert.equal(url.origin, origin, 'requests must stay inside the fixture origin');
      assert.ok(++requests <= 250, 'fixture request count must remain bounded');
      if (path === '/embed') {
        return route.fulfill({
          contentType: 'text/html',
          body: '<!doctype html><title>Dashboard fixture</title><iframe title="Pipelines" src="/pipelines/#/runs" style="width:100%;height:850px;border:0"></iframe>',
        });
      }
      if (path === '/' || path.startsWith('/static/')) {
        const name = path === '/' ? 'index.html' : path.slice(1);
        let body = await readFile(new URL(name, build));
        if (path === '/') {
          body = body
            .toString()
            .replace(
              /window\.KFP_FLAGS\.DEPLOYMENT\s*=\s*null;?/,
              `window.KFP_FLAGS.DEPLOYMENT=${JSON.stringify(options.namespace ? 'KUBEFLOW' : null)};`,
            )
            .replace(
              /window\.KFP_FLAGS\.HIDE_SIDENAV\s*=\s*null;?/,
              `window.KFP_FLAGS.HIDE_SIDENAV=${!!options.embedded};`,
            );
        }
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
      if (
        ['/apis/v2beta1/runs', '/apis/v2beta1/pipelines', '/apis/v2beta1/experiments'].includes(
          path,
        ) &&
        url.searchParams.get('page_size') === '5'
      ) {
        assert.equal(request.method(), 'GET');
        const query = Object.fromEntries(url.searchParams);
        assert.equal(query.sort_by, 'created_at desc');
        assert.equal(
          query.page_token,
          undefined,
          'palette searches must not scan subsequent pages',
        );
        if (path.endsWith('/runs')) assert.equal(query.skip_count, 'true');
        const predicates = decodeFilter(query.filter).predicates;
        assert.equal(predicates.length, 1);
        assert.equal(predicates[0].key, 'name');
        assert.equal(predicates[0].operation, 'IS_SUBSTRING');
        assert.ok(predicates[0].string_value.length >= 2);
        fixture.searches.push({ path, ...query });
        const matches = (item) =>
          item.display_name.toLowerCase().includes(predicates[0].string_value.toLowerCase());
        if (path.endsWith('/runs')) {
          const runs = query.namespace === 'team-b' ? fixture.runs.slice(0, 2) : fixture.runs;
          return json({ runs: runs.filter(matches).slice(0, 5), next_page_token: 'do-not-follow' });
        }
        if (path.endsWith('/pipelines'))
          return json({
            pipelines: [{ pipeline_id: 'pipeline-a', display_name: 'Training pipeline' }].filter(
              matches,
            ),
          });
        return json({
          experiments: [
            { experiment_id: 'experiment-a', display_name: 'Training experiment' },
          ].filter(matches),
        });
      }
      if (path === '/apis/v2beta1/runs' && request.method() === 'GET') {
        assert.equal(url.pathname, `${options.embedded ? '/pipelines' : ''}/apis/v2beta1/runs`);
        const query = Object.fromEntries(url.searchParams);
        assert.equal(query.skip_count, 'true');
        assert.ok([10, 20, 50, 100].includes(Number(query.page_size)));
        assert.match(query.sort_by, /^(created_at|name)( (asc|desc))?$/);
        const predicates = decodeFilter(query.filter).predicates;
        const storage = predicates.find((predicate) => predicate.key === 'storage_state');
        assert.equal(storage.string_value, 'ARCHIVED');
        assert.ok(['EQUALS', 'NOT_EQUALS'].includes(storage.operation));
        const archived = storage.operation === 'EQUALS';
        const names = predicates.filter((predicate) => predicate.key === 'name');
        assert.ok(names.every((predicate) => predicate.operation === 'IS_SUBSTRING'));
        assert.equal(
          predicates.length,
          names.length + 1,
          'only name and storage filters are expected',
        );
        fixture.lists.push({ ...query, predicates });
        if (fixture.failNextList) {
          fixture.failNextList = false;
          return json({ code: 14, message: 'Fixture list temporarily unavailable' }, 503);
        }
        let runs = fixture.runs.filter((run) => (run.storage_state === 'ARCHIVED') === archived);
        if (query.namespace === 'team-b') runs = runs.slice(0, 2);
        runs = runs.filter((run) =>
          names.every((predicate) =>
            run.display_name.toLowerCase().includes(predicate.string_value.toLowerCase()),
          ),
        );
        const [sort, direction = 'asc'] = query.sort_by.split(' ');
        runs.sort(
          (a, b) =>
            (sort === 'name'
              ? a.display_name.localeCompare(b.display_name)
              : a.created_at.localeCompare(b.created_at)) * (direction === 'desc' ? -1 : 1),
        );
        const offset = query.page_token
          ? Number(query.page_token.replace(/^fixture-offset-/, ''))
          : 0;
        assert.ok(!query.page_token || /^fixture-offset-\d+$/.test(query.page_token));
        const end = offset + Number(query.page_size);
        return json({
          runs: runs.slice(offset, end),
          next_page_token: end < runs.length ? `fixture-offset-${end}` : '',
        });
      }
      const tasks = /^\/apis\/v2beta1\/runs\/([^/]+)\/tasks$/.exec(path);
      if (tasks) {
        assert.equal(request.method(), 'GET');
        assert.ok(fixture.runs.some((run) => run.run_id === tasks[1]));
        return json({ tasks: [] });
      }
      const run = /^\/apis\/v2beta1\/runs\/([^/:]+)(?::(archive|unarchive))?$/.exec(path);
      if (run) {
        const id = decodeURIComponent(run[1]);
        const item = fixture.runs.find((candidate) => candidate.run_id === id);
        assert.ok(item, `unknown fixture run ${id}`);
        if (request.method() === 'GET' && !run[2]) return json(item);
        const action = run[2] || 'delete';
        assert.equal(request.method(), action === 'delete' ? 'DELETE' : 'POST');
        assert.equal(request.postData(), null, 'run mutations have no body');
        fixture.mutations.push({ id, action });
        if (fixture.forbidden.has(id))
          return json({ code: 7, message: 'Fixture permission denied' }, 403);
        if (action === 'delete')
          fixture.runs = fixture.runs.filter((candidate) => candidate !== item);
        else item.storage_state = action === 'archive' ? 'ARCHIVED' : 'AVAILABLE';
        return json({});
      }
      if (path === '/apis/v2beta1/experiments')
        return json({
          experiments: [{ experiment_id: 'experiment-a', display_name: 'Browser experiment' }],
        });
      if (path === '/apis/v2beta1/experiments/experiment-a')
        return json({ experiment_id: 'experiment-a', display_name: 'Browser experiment' });
      if (path === '/apis/v2beta1/pipelines/pipeline-a')
        return json({ pipeline_id: 'pipeline-a', display_name: 'Browser pipeline' });
      if (path === '/apis/v2beta1/pipelines/pipeline-a/versions/deleted-version') {
        fixture.versionReads.push(path);
        return json({ code: 5, message: 'Fixture deleted version' }, 404);
      }
      if (path === '/apis/v2beta1/pipelines/pipeline-a/versions')
        return json({ pipeline_versions: [pipelineVersion] });
      if (path === '/apis/v2beta1/pipelines/pipeline-a/versions/version-a') {
        fixture.versionReads.push(path);
        return json(pipelineVersion);
      }
      if (path === '/apis/v2beta1/artifacts' || path === '/apis/v2beta1/artifact_tasks')
        return json({});
      throw new Error(`Unexpected fixture request: ${request.method()} ${path}`);
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
      'production bundle and HTTP contracts must have no unexpected errors',
    );
  } catch (error) {
    if (fixture.errors.length) console.error('Fixture errors:', fixture.errors);
    throw error;
  } finally {
    await context.close();
  }
}

const link = (page, index) =>
  page.locator(`[data-testid="run-name-link"][data-run-id="${runId(index)}"]`);
const row = (page, index) => page.getByRole('row').filter({ has: link(page, index) });
const checkbox = (page, index) => row(page, index).getByRole('checkbox');
async function ready(page, count = 10) {
  await page.locator('table[aria-label="Runs"]').waitFor();
  await page.waitForFunction((expected) => {
    const table = document.querySelector('table[aria-label="Runs"]');
    return (
      table?.getAttribute('aria-busy') === 'false' &&
      table.querySelectorAll('[data-testid="run-name-link"]').length === expected
    );
  }, count);
}
async function openRuns(page, archived = false) {
  await page.goto(`${origin}/#/${archived ? 'archive/runs' : 'runs'}`);
  await ready(page, archived ? 3 : 10);
}
async function confirm(page, action) {
  await page.getByRole('button', { name: action, exact: true }).click();
  const dialog = page.getByRole('dialog');
  await dialog.waitFor();
  await dialog.getByRole('button', { name: action, exact: true }).click();
}

async function reloadRows(page, action, count = 10) {
  const host = typeof page.page === 'function' ? page.page() : page;
  const response = host.waitForResponse(
    (response) =>
      new URL(response.url()).pathname.endsWith('/apis/v2beta1/runs') &&
      response.request().method() === 'GET',
  );
  await action();
  await response;
  await ready(page, count);
}

async function focused(page, locator) {
  const element = await locator.elementHandle();
  assert.ok(element);
  try {
    await page.waitForFunction((node) => node === document.activeElement, element);
  } finally {
    await element.dispose();
  }
}

async function screenshot(page, name) {
  if (!process.env.KFP_RUNS_SCREENSHOT_DIR) return;
  const { mkdir } = await import('node:fs/promises');
  const { join } = await import('node:path');
  await mkdir(process.env.KFP_RUNS_SCREENSHOT_DIR, { recursive: true });
  await page.screenshot({
    path: join(process.env.KFP_RUNS_SCREENSHOT_DIR, `${name}.png`),
    animations: 'disabled',
  });
}

test('Runs preserves token paging, filtering, sorting, and page selection contracts', async () => {
  await withFixture({}, async (page, fixture) => {
    await openRuns(page);
    assert.equal(fixture.lists[0].sort_by, 'created_at desc');
    assert.equal(fixture.lists[0].page_size, '10');
    assert.deepEqual(
      fixture.versionReads.toSorted(),
      [
        '/apis/v2beta1/pipelines/pipeline-a/versions/deleted-version',
        '/apis/v2beta1/pipelines/pipeline-a/versions/version-a',
      ],
      'each distinct referenced version is fetched once per page load',
    );
    assert.match(await row(page, 8).innerText(), /Unavailable/);
    for (const state of ['Succeeded', 'Failed', 'Running', 'Pending', 'Canceled']) {
      assert.ok((await page.getByRole('table', { name: 'Runs' }).innerText()).includes(state));
    }
    await screenshot(page, 'runs-light');
    await reloadRows(page, () => page.getByRole('button', { name: 'Next page' }).click(), 5);
    assert.equal(fixture.lists.at(-1).page_token, 'fixture-offset-10');
    assert.equal(await link(page, 11).count(), 1);
    await reloadRows(page, () => page.getByRole('button', { name: 'Previous page' }).click());
    assert.equal(fixture.lists.at(-1).page_token || '', '');
    await reloadRows(page, () => page.getByRole('button', { name: 'Next page' }).click(), 5);
    await reloadRows(
      page,
      () => page.getByRole('searchbox', { name: 'Filter runs by name' }).fill('Training 15'),
      1,
    );
    assert.deepEqual(
      fixture.lists.at(-1).predicates.find((item) => item.key === 'name'),
      {
        key: 'name',
        operation: 'IS_SUBSTRING',
        string_value: 'Training 15',
      },
    );
    assert.equal(fixture.lists.at(-1).page_token || '', '');
    await reloadRows(page, () =>
      page.getByRole('button', { name: 'Clear filter', exact: true }).click(),
    );
    await reloadRows(page, () => page.getByRole('button', { name: 'Run', exact: true }).click());
    assert.equal(fixture.lists.at(-1).sort_by, 'name');
    assert.equal(
      await page.getByRole('columnheader', { name: 'Run', exact: true }).getAttribute('aria-sort'),
      'ascending',
    );
    await reloadRows(page, () => page.getByRole('button', { name: 'Run', exact: true }).click());
    assert.equal(fixture.lists.at(-1).sort_by, 'name desc');
    await reloadRows(
      page,
      () => page.getByRole('combobox', { name: 'Rows per page' }).selectOption('20'),
      15,
    );
    assert.equal(fixture.lists.at(-1).page_size, '20');
    assert.equal(fixture.lists.at(-1).page_token || '', '');
    await page.getByRole('checkbox', { name: 'Select all runs on this page' }).check();
    assert.equal(await page.getByRole('checkbox', { checked: true }).count(), 16);
    assert.equal(new URL(page.url()).hash, '#/runs');
    await page.getByRole('checkbox', { name: 'Select all runs on this page' }).uncheck();
    assert.equal(await page.getByRole('checkbox', { checked: true }).count(), 0);
    await checkbox(page, 1).check();
    await reloadRows(
      page,
      () => page.getByRole('button', { name: 'Refresh', exact: true }).click(),
      15,
    );
    assert.equal(await checkbox(page, 1).isChecked(), true);
  });
});

test('Runs separates selection from row, link, clone, and comparison navigation', async () => {
  await withFixture({}, async (page) => {
    await openRuns(page);
    await checkbox(page, 1).focus();
    await page.keyboard.press('Space');
    assert.equal(await checkbox(page, 1).isChecked(), true);
    assert.equal(new URL(page.url()).hash, '#/runs');
    await page.getByRole('button', { name: 'Clone run', exact: true }).click();
    await page.waitForURL((url) => url.hash === '#/runs/new?cloneFromRun=run-01');
    await openRuns(page);
    await checkbox(page, 1).check();
    await checkbox(page, 2).check();
    await page.getByRole('button', { name: 'Compare runs', exact: true }).click();
    await page.waitForURL((url) => url.hash.startsWith('#/compare?'));
    assert.deepEqual(
      new URLSearchParams(new URL(page.url()).hash.split('?')[1]).get('runlist').split(',').sort(),
      ['run-01', 'run-02'],
    );
    await openRuns(page);
    await row(page, 1).getByRole('link', { name: 'Version one', exact: true }).click();
    await page.waitForURL((url) => url.hash === '#/pipelines/details/pipeline-a/version/version-a');
    await openRuns(page);
    await row(page, 1).getByRole('cell').last().click();
    await page.waitForURL((url) => url.hash === '#/runs/details/run-01');
    await openRuns(page);
    await link(page, 2).click();
    await page.waitForURL((url) => url.hash === '#/runs/details/run-02');
  });
});

test('Runs confirms archive once, retains forbidden selections, and restores or deletes archived runs', async () => {
  await withFixture({}, async (page, fixture) => {
    await openRuns(page);
    await checkbox(page, 1).check();
    await page.getByRole('button', { name: 'Archive', exact: true }).click();
    await page.getByRole('dialog').getByRole('button', { name: 'Cancel', exact: true }).click();
    assert.deepEqual(fixture.mutations, []);
    assert.equal(await checkbox(page, 1).isChecked(), true);
    await reloadRows(page, () => confirm(page, 'Archive'));
    assert.deepEqual(fixture.mutations, [{ id: 'run-01', action: 'archive' }]);
    assert.equal(await link(page, 1).count(), 0);
    fixture.forbidden.add('run-05');
    await checkbox(page, 2).check();
    await checkbox(page, 5).check();
    await reloadRows(page, () => confirm(page, 'Archive'));
    const failure = page.getByRole('dialog', { name: 'Failed to archive 1 run' });
    await failure.waitFor();
    assert.match(await failure.innerText(), /Failed to archive run: run-05/);
    await failure.getByRole('button', { name: 'Dismiss', exact: true }).click();
    assert.equal(await checkbox(page, 5).isChecked(), true);
    assert.equal(await link(page, 2).count(), 0);
    assert.deepEqual(
      fixture.mutations.toSorted((a, b) => a.id.localeCompare(b.id)),
      [
        { id: 'run-01', action: 'archive' },
        { id: 'run-02', action: 'archive' },
        { id: 'run-05', action: 'archive' },
      ],
    );
    await reloadRows(
      page,
      () =>
        page
          .getByRole('navigation', { name: 'Run views' })
          .getByRole('link', { name: 'Archived', exact: true })
          .click(),
      5,
    );
    assert.equal(new URL(page.url()).hash, '#/archive/runs');
    assert.equal(fixture.lists.at(-1).predicates[0].operation, 'EQUALS');
    await checkbox(page, 1).check();
    await reloadRows(page, () => confirm(page, 'Restore'), 4);
    await checkbox(page, 16).check();
    await reloadRows(page, () => confirm(page, 'Delete'), 3);
    assert.deepEqual(fixture.mutations.slice(-2), [
      { id: 'run-01', action: 'unarchive' },
      { id: 'run-16', action: 'delete' },
    ]);
    assert.equal(await link(page, 16).count(), 0);
    await reloadRows(page, () =>
      page
        .getByRole('navigation', { name: 'Run views' })
        .getByRole('link', { name: 'Active', exact: true })
        .click(),
    );
    assert.equal(await link(page, 1).count(), 1);
  });
});

test('Runs recovers after an API error and keeps dialogs themed, keyboard accessible, and contained on mobile', async () => {
  await withFixture({ failNextList: true }, async (page, fixture) => {
    await page.goto(`${origin}/#/runs`);
    const alert = page.getByRole('alert');
    await alert.waitFor();
    await reloadRows(page, () =>
      alert.getByRole('button', { name: 'Refresh', exact: true }).click(),
    );
    assert.equal(await alert.count(), 0);
    assert.equal(fixture.lists.length, 2);
    await checkbox(page, 1).check();
    await page.getByRole('button', { name: 'Archive', exact: true }).click();
    const dialog = page.getByRole('dialog');
    await dialog.waitFor();
    await focused(page, dialog.getByRole('button', { name: 'Cancel', exact: true }));
    const lightBackground = await dialog.evaluate((node) => getComputedStyle(node).backgroundColor);
    await page.keyboard.press('Escape');
    await dialog.waitFor({ state: 'hidden' });
    await focused(page, page.getByRole('button', { name: 'Archive', exact: true }));
    assert.deepEqual(fixture.mutations, []);
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    await screenshot(page, 'runs-dark');
    await page.getByRole('button', { name: 'Archive', exact: true }).click();
    await dialog.waitFor();
    assert.notEqual(
      await dialog.evaluate((node) => getComputedStyle(node).backgroundColor),
      lightBackground,
    );
    assert.equal(await dialog.evaluate((node) => !!node.closest('.kfp-theme.dark')), true);
    await focused(page, dialog.getByRole('button', { name: 'Cancel', exact: true }));
    await page.keyboard.press('Shift+Tab');
    await focused(page, dialog.getByRole('button', { name: 'Archive', exact: true }));
    await page.keyboard.press('Tab');
    await focused(page, dialog.getByRole('button', { name: 'Cancel', exact: true }));
    await screenshot(page, 'runs-dark-archive-dialog');
    await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
    await page.reload();
    await ready(page);
    assert.equal(
      await page.getByRole('combobox', { name: 'Theme', exact: true }).inputValue(),
      'dark',
    );
    await page.setViewportSize({ width: 375, height: 812 });
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
      true,
    );
    const region = page.getByRole('region', { name: 'Runs table', exact: true });
    assert.equal(await region.evaluate((node) => node.scrollWidth > node.clientWidth), true);
    await page.getByRole('searchbox', { name: 'Filter runs by name' }).focus();
    assert.equal(
      await page
        .getByRole('searchbox', { name: 'Filter runs by name' })
        .evaluate((node) => node === document.activeElement),
      true,
    );
  });
});

test('Embedded Runs preserves its URL prefix and clears selection when the dashboard namespace changes', async () => {
  await withFixture({ namespace: 'team-a', embedded: true }, async (page, fixture) => {
    await page.goto(`${origin}/embed`);
    const app = page.frames().find((frame) => frame.url().includes('/pipelines/'));
    assert.ok(app, 'the dashboard fixture must load the production app in its iframe');
    await ready(app);
    assert.equal(fixture.lists.at(-1).namespace, 'team-a');
    assert.equal(await app.getByRole('combobox', { name: 'Theme', exact: true }).count(), 1);
    await checkbox(app, 1).check();
    await reloadRows(app, () => app.evaluate(() => window.selectFixtureNamespace('team-b')), 2);
    assert.equal(fixture.lists.at(-1).namespace, 'team-b');
    assert.equal(fixture.lists.at(-1).page_token || '', '');
    assert.equal(await checkbox(app, 1).isChecked(), false);
    assert.equal(new URL(app.url()).pathname, '/pipelines/');
  });
});

test('Command palette searches the active namespace and preserves keyboard navigation and focus', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    await openRuns(page);
    const trigger = page.getByRole('button', { name: 'Search', exact: true });
    await trigger.focus();
    await page.keyboard.press('Control+k');
    const dialog = page.getByRole('dialog', { name: 'Search and navigate' });
    const input = dialog.getByRole('searchbox');
    await focused(page, input);
    await input.fill('Training');
    await dialog.getByRole('link', { name: 'Training experiment', exact: true }).waitFor();
    await screenshot(page, 'command-palette-light');
    assert.equal(fixture.searches.length, 3);
    assert.ok(fixture.searches.every((request) => request.namespace === 'team-a'));
    assert.equal(
      await dialog.getByRole('region', { name: 'Runs', exact: true }).getByRole('link').count(),
      5,
    );
    await page.keyboard.press('ArrowDown');
    await focused(page, dialog.getByRole('link', { name: 'Training 01', exact: true }));
    await page.keyboard.press('Tab');
    await focused(page, dialog.getByRole('link', { name: 'Training 02', exact: true }));
    await input.focus();
    await page.keyboard.press('Shift+Tab');
    await focused(page, dialog.getByRole('button', { name: 'Close', exact: true }));
    await page.keyboard.press('Tab');
    await focused(page, input);
    await page.keyboard.press('Escape');
    await dialog.waitFor({ state: 'hidden' });
    await focused(page, trigger);
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    await trigger.click();
    await focused(page, input);
    await input.fill('Training');
    await dialog.getByRole('link', { name: 'Training experiment', exact: true }).waitFor();
    await page.evaluate(() => window.selectFixtureNamespace('team-b'));
    await dialog.getByText('Namespace: team-b', { exact: true }).waitFor();
    await dialog.getByRole('link', { name: 'Training experiment', exact: true }).waitFor();
    assert.equal(
      await dialog.evaluate((element) => element.closest('.kfp-theme')?.classList.contains('dark')),
      true,
    );
    await screenshot(page, 'command-palette-dark');
    assert.equal(fixture.searches.length, 9);
    assert.ok(fixture.searches.slice(-3).every((request) => request.namespace === 'team-b'));
    assert.equal(
      await dialog.getByRole('region', { name: 'Runs', exact: true }).getByRole('link').count(),
      2,
    );
    await dialog.getByRole('link', { name: 'Training 01', exact: true }).click();
    await page.waitForURL((url) => url.hash === '#/runs/details/run-01');
    await dialog.waitFor({ state: 'hidden' });
    await trigger.click();
    await dialog.getByRole('link', { name: 'Runs', exact: true }).click();
    await page.waitForURL((url) => url.hash === '#/runs');
    await ready(page, 2);
  });
});
