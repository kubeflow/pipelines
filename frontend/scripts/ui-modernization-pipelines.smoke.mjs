/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Run after npm run build: node --test scripts/ui-modernization-pipelines.smoke.mjs
// Local bounded fixtures verify UI/request parity; live cluster authorization is separate.
import assert from 'node:assert/strict';
import { mkdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { createServer } from 'node:http';
import { before, after, test } from 'node:test';
import { launchBrowser } from './browser-launch.mjs';

const build = new URL('../build/', import.meta.url);
const specialPipelineId = 'pipeline/01%fixture';
const specialVersionId = 'version/01%fixture';
const pipelineSpec = JSON.parse(
  await readFile(
    new URL(
      '../mock-backend/data/v2/pipeline/lightweight_python_functions_v2_pipeline.json',
      import.meta.url,
    ),
    'utf8',
  ),
);
const pipelinePath = (id, version) =>
  `/pipelines/details/${encodeURIComponent(id)}${version ? `/version/${encodeURIComponent(version)}` : ''}`;
let browser;
before(async () => {
  browser = await launchBrowser();
  if (process.env.KFP_EXPECTED_BROWSER_VERSION) {
    assert.equal(
      browser.version(),
      process.env.KFP_EXPECTED_BROWSER_VERSION,
      'Browser version must match KFP_EXPECTED_BROWSER_VERSION; select the intended browser binary',
    );
  }
});
after(async () => browser?.close());

function makePipelines() {
  return Array.from({ length: 15 }, (_, index) => ({
    pipeline_id: index ? `pipeline-${index + 1}` : specialPipelineId,
    name: `pipeline-${String(index + 1).padStart(2, '0')}`,
    display_name: `Pipeline ${String(index + 1).padStart(2, '0')}`,
    description: `Description ${index + 1}`,
    created_at: new Date(Date.UTC(2026, 8, 26 - index)).toISOString(),
  }));
}
function versions(pipelineId) {
  return Array.from({ length: 12 }, (_, index) => ({
    pipeline_id: pipelineId,
    pipeline_version_id: index ? `version-${index + 1}` : specialVersionId,
    name: `version-${String(index + 1).padStart(2, '0')}`,
    display_name: `Version ${String(index + 1).padStart(2, '0')}`,
    description: `Version description ${index + 1}`,
    created_at: new Date(Date.UTC(2026, 8, 26 - index)).toISOString(),
    pipeline_spec: pipelineSpec,
  }));
}
function listPage(resources, query, resource) {
  const predicates = query.filter
    ? JSON.parse(query.filter.startsWith('%') ? decodeURIComponent(query.filter) : query.filter)
        .predicates
    : [];
  assert.ok(
    predicates.every(({ key, operation }) => key === 'name' && operation === 'IS_SUBSTRING'),
  );
  let rows = resources.filter((row) =>
    predicates.every(({ string_value }) =>
      row.display_name.toLowerCase().includes(string_value.toLowerCase()),
    ),
  );
  const [sort, direction = 'asc'] = (query.sort_by || 'created_at desc').split(' ');
  assert.ok(['created_at', 'display_name', 'name'].includes(sort));
  assert.ok(['asc', 'desc'].includes(direction));
  rows.sort(
    (a, b) => String(a[sort]).localeCompare(String(b[sort])) * (direction === 'desc' ? -1 : 1),
  );
  const size = Number(query.page_size || 10);
  assert.ok([1, 10, 20, 50, 100].includes(size));
  const key = JSON.stringify({
    resource,
    filter: query.filter || '',
    sort: query.sort_by || '',
    namespace: query.namespace || '',
    size,
  });
  const token = query.page_token
    ? JSON.parse(Buffer.from(query.page_token, 'base64url').toString())
    : { key, offset: 0 };
  assert.equal(
    token.key,
    key,
    'filter/sort/namespace/page-size changes must reset the paging token',
  );
  const end = token.offset + size;
  return {
    rows: rows.slice(token.offset, end),
    next_page_token:
      end < rows.length
        ? Buffer.from(JSON.stringify({ key, offset: end })).toString('base64url')
        : '',
  };
}

async function withFixture(options, exercise) {
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    locale: 'en-US',
    timezoneId: 'UTC',
    reducedMotion: 'reduce',
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  const fixture = {
    pipelines: makePipelines(),
    versionMap: new Map(),
    lists: [],
    mutations: [],
    reads: [],
    errors: [],
    requests: 0,
    workers: [],
    origin: undefined,
  };
  for (const pipeline of fixture.pipelines)
    fixture.versionMap.set(pipeline.pipeline_id, versions(pipeline.pipeline_id));

  // WebKit's request.postData() omits file bytes. Let native uploads reach this
  // same-origin loopback server, then validate their bytes before mutating the fixture.
  const uploadResponses = new Map();
  let receivedUploads = 0;
  const uploadServer = createServer(async (request, response) => {
    const json = (value, status = 200) => {
      response.writeHead(status, { 'content-type': 'application/json' });
      response.end(JSON.stringify(value));
    };
    try {
      assert.ok(++receivedUploads <= 2, 'fixture uploads must remain bounded');
      assert.equal(request.method, 'POST');
      const createResource = uploadResponses.get(request.url);
      assert.ok(createResource, 'only the exact routed upload path and query may reach the server');
      uploadResponses.delete(request.url);
      const contentType = request.headers['content-type'] || '';
      const boundary = contentType.match(/multipart\/form-data; boundary=(?:"([^"]+)"|([^;]+))/);
      assert.ok(boundary, 'native upload must retain its multipart boundary');
      const chunks = [];
      let size = 0;
      for await (const chunk of request) {
        size += chunk.length;
        assert.ok(size <= 1024 * 1024, 'fixture upload body must remain below 1 MiB');
        chunks.push(chunk);
      }
      const body = Buffer.concat(chunks);
      const marker = boundary[1] || boundary[2];
      const start = Buffer.from(`--${marker}\r\n`);
      const end = Buffer.from(`\r\n--${marker}--\r\n`);
      assert.ok(body.subarray(0, start.length).equals(start));
      assert.ok(body.subarray(-end.length).equals(end));
      const headerEnd = body.indexOf('\r\n\r\n');
      assert.ok(headerEnd > start.length);
      const headers = body.subarray(start.length, headerEnd).toString();
      assert.match(
        headers,
        /Content-Disposition: form-data; name="uploadfile"; filename="fixture\.yaml"/i,
      );
      assert.match(headers, /Content-Type: application\/yaml/i);
      assert.deepEqual(
        body.subarray(headerEnd + 4, -end.length),
        Buffer.from(JSON.stringify(pipelineSpec)),
        'the actual upload must contain the exact selected package bytes',
      );
      json(createResource());
    } catch (error) {
      fixture.errors.push(error.message);
      json({ message: error.message }, 500);
    }
  });
  await new Promise((resolve, reject) => {
    uploadServer.once('error', reject);
    uploadServer.listen(0, '127.0.0.1', resolve);
  });
  fixture.origin = `http://127.0.0.1:${uploadServer.address().port}`;

  async function uploadResponse(route, createResource) {
    const original = new URL(route.request().url());
    assert.ok(
      ['/apis/v2beta1/pipelines/upload', '/apis/v2beta1/pipelines/upload_version'].includes(
        original.pathname,
      ),
    );
    const key = `${original.pathname}${original.search}`;
    assert.equal(uploadResponses.has(key), false, 'each upload may be routed only once');
    uploadResponses.set(key, createResource);
    await route.continue();
  }
  page.on('pageerror', (error) => fixture.errors.push(error.message));
  await context.addInitScript(({ namespace }) => {
    localStorage.setItem('kfp.theme', 'light');
    if (namespace)
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
  }, options);
  await context.route('**/*', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const query = Object.fromEntries(url.searchParams);
    const method = request.method();
    const json = (value, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(value) });
    try {
      assert.equal(url.origin, fixture.origin, 'fixture must not contact an external service');
      assert.ok(++fixture.requests <= 250, 'requests must remain bounded');
      // WebKit exposes native Blob workers to routing; their imported HTTP assets remain mocked.
      if (url.protocol === 'blob:') return route.continue();
      if (path === '/' || path.startsWith('/static/')) {
        const name = path === '/' ? 'index.html' : path.slice(1);
        if (/worker-(yaml|json).*\.js$/.test(name)) fixture.workers.push(name);
        let body = await readFile(new URL(name, build));
        if (path === '/')
          body = body
            .toString()
            .replace(
              /window\.KFP_FLAGS\.DEPLOYMENT\s*=\s*null;?/,
              `window.KFP_FLAGS.DEPLOYMENT=${JSON.stringify(options.namespace ? 'KUBEFLOW' : null)};`,
            );
        return route.fulfill({
          body,
          contentType: name.endsWith('.js')
            ? 'text/javascript'
            : name.endsWith('.css')
              ? 'text/css'
              : name.endsWith('.html')
                ? 'text/html'
                : 'application/octet-stream',
        });
      }
      if (path === '/apis/v2beta1/healthz')
        return json({
          apiServerTagName: 'fixture',
          apiServerMultiUser: !!options.namespace,
          pipelineStore: options.pipelineStore || 'database',
        });
      if (path === '/system/cluster-name' || path === '/system/project-id')
        return route.fulfill({ body: '' });
      if (path === '/apis/v2beta1/pipelines' && method === 'GET') {
        fixture.lists.push({ resource: 'pipelines', ...query });
        let rows = fixture.pipelines;
        if (query.namespace === 'team-b') rows = rows.slice(0, 2);
        if (options.namespace && !query.namespace) rows = rows.slice(0, 3);
        const result = listPage(rows, query, 'pipelines');
        return json({ pipelines: result.rows, next_page_token: result.next_page_token });
      }
      if (
        (path === '/apis/v2beta1/pipelines' || path === '/apis/v2beta1/pipelines/upload') &&
        method === 'POST'
      ) {
        const upload = path.endsWith('/upload');
        const data = upload ? query : request.postDataJSON();
        const createPipeline = () => {
          fixture.mutations.push({ kind: upload ? 'upload-pipeline' : 'create-pipeline', data });
          const pipeline = {
            ...data,
            pipeline_id: 'created-pipeline',
            created_at: '2026-09-27T12:00:00Z',
          };
          fixture.pipelines.push(pipeline);
          fixture.versionMap.set(
            pipeline.pipeline_id,
            upload ? versions(pipeline.pipeline_id).slice(0, 1) : [],
          );
          return pipeline;
        };
        return upload ? await uploadResponse(route, createPipeline) : json(createPipeline());
      }
      if (path === '/apis/v2beta1/pipelines/upload_version' && method === 'POST') {
        return await uploadResponse(route, () => {
          const version = {
            ...versions(query.pipelineid)[0],
            name: query.name,
            display_name: query.display_name || query.name,
            pipeline_version_id: 'created-version',
          };
          fixture.mutations.push({ kind: 'upload-version', data: query });
          fixture.versionMap.get(query.pipelineid).unshift(version);
          return version;
        });
      }
      const versionList = path.match(/^\/apis\/v2beta1\/pipelines\/([^/]+)\/versions$/);
      if (versionList) {
        const pipelineId = decodeURIComponent(versionList[1]);
        assert.ok(
          fixture.versionMap.has(pipelineId),
          'version list must reference its actual pipeline',
        );
        if (method === 'POST') {
          const data = request.postDataJSON();
          assert.equal(data.pipeline_id, pipelineId);
          const version = {
            ...versions(pipelineId)[0],
            ...data,
            pipeline_version_id: 'created-version',
          };
          fixture.mutations.push({ kind: 'create-version', data });
          fixture.versionMap.get(pipelineId).unshift(version);
          return json(version);
        }
        assert.equal(method, 'GET');
        fixture.lists.push({ resource: 'versions', pipelineId, ...query });
        const result = listPage(fixture.versionMap.get(pipelineId), query, pipelineId);
        return json({ pipeline_versions: result.rows, next_page_token: result.next_page_token });
      }
      const detail = path.match(/^\/apis\/v2beta1\/pipelines\/([^/]+)(?:\/versions\/([^/]+))?$/);
      if (detail) {
        const pipelineId = decodeURIComponent(detail[1]);
        const versionId = detail[2] && decodeURIComponent(detail[2]);
        fixture.reads.push({ pipelineId, versionId, method });
        assert.equal(method, 'GET');
        const value = versionId
          ? fixture.versionMap
              .get(pipelineId)
              ?.find((version) => version.pipeline_version_id === versionId)
          : fixture.pipelines.find((pipeline) => pipeline.pipeline_id === pipelineId);
        assert.ok(value, 'detail request must preserve encoded resource identity');
        return json(value);
      }
      throw new Error(`Unexpected request: ${method} ${path}`);
    } catch (error) {
      fixture.errors.push(error.message);
      return json({ message: error.message }, 500);
    }
  });
  try {
    await exercise(page, fixture);
    assert.equal(uploadResponses.size, 0, 'every routed upload must reach the native collector');
    assert.deepEqual(
      fixture.errors,
      [],
      'production bundle and fixture request contracts must have no unexpected errors',
    );
  } catch (error) {
    console.error(
      'Pipeline fixture diagnostic:',
      JSON.stringify({
        url: page.url(),
        errors: fixture.errors,
        lists: fixture.lists.slice(-6),
        reads: fixture.reads,
        mutations: fixture.mutations,
        dialog: await page.getByRole('dialog').allTextContents(),
      }),
    );
    throw error;
  } finally {
    try {
      await context.close();
    } finally {
      await new Promise((resolve) => uploadServer.close(resolve));
    }
  }
}

async function screenshot(page, name) {
  if (!process.env.KFP_PIPELINES_SCREENSHOT_DIR) return;
  await mkdir(process.env.KFP_PIPELINES_SCREENSHOT_DIR, { recursive: true });
  await page.screenshot({
    path: join(process.env.KFP_PIPELINES_SCREENSHOT_DIR, `${name}.png`),
    animations: 'disabled',
  });
}

async function darkTheme(page) {
  await page.getByRole('button', { name: /^Theme: / }).click();
  await page.getByRole('menuitemradio', { name: 'Dark', exact: true }).click();
  await page.waitForFunction(() => {
    const shell = document.querySelector('.kfp-shell');
    const brand = document.querySelector('.kfp-shell-brand-title');
    return (
      getComputedStyle(shell).backgroundColor === 'rgb(14, 16, 21)' &&
      getComputedStyle(brand).color === 'rgb(236, 238, 244)' &&
      [...document.querySelectorAll('.kfp-pipeline-card')].every(
        (element) =>
          getComputedStyle(element).backgroundColor ===
          (element.dataset.selected === 'true' ? 'rgb(26, 39, 64)' : 'rgb(22, 25, 33)'),
      )
    );
  });
}

const cards = (page) => page.getByRole('list', { name: 'Pipelines' });
const card = (page, name) =>
  cards(page)
    .getByRole('listitem')
    .filter({ has: page.getByRole('link', { name, exact: true }) });
async function openList(page, fixture) {
  await page.goto(`${fixture.origin}/#/pipelines`);
  await page.getByRole('link', { name: 'Pipeline 01', exact: true }).waitFor();
  await waitForCards(page, 10);
}
async function waitForCards(page, count) {
  await page.waitForFunction(
    (expected) => document.querySelectorAll('ul[aria-label="Pipelines"] > li').length === expected,
    count,
  );
}

// The card controls intentionally keep all server-backed capabilities from CustomTable.
test('pipeline cards retain filtering, sorting, paging, selection and lazy version expansion', async () => {
  await withFixture({}, async (page, fixture) => {
    await openList(page, fixture);
    await waitForCards(page, 10);
    await screenshot(page, 'pipelines-light');
    assert.equal(
      fixture.lists.filter(({ resource }) => resource === 'versions').length,
      0,
      'cards must not fetch versions or histories eagerly',
    );
    assert.equal(
      await card(page, 'Pipeline 01')
        .getByRole('link', { name: 'Pipeline 01', exact: true })
        .getAttribute('href'),
      `#${pipelinePath(specialPipelineId)}`,
    );
    await page.getByRole('button', { name: 'Next page', exact: true }).click();
    await page.getByRole('link', { name: 'Pipeline 11', exact: true }).waitFor();
    await page.getByRole('searchbox', { name: 'Filter pipelines' }).fill('Pipeline 01');
    await page.getByRole('link', { name: 'Pipeline 01', exact: true }).waitFor();
    assert.equal(fixture.lists.at(-1).page_token || '', '');
    await waitForCards(page, 1);
    await page.getByRole('button', { name: 'Clear filter', exact: true }).click();
    await page.getByRole('link', { name: 'Pipeline 10', exact: true }).waitFor();
    await Promise.all([
      page.waitForResponse(
        (response) => new URL(response.url()).searchParams.get('sort_by') === 'display_name',
      ),
      page.getByRole('combobox', { name: 'Sort pipelines' }).selectOption('display_name'),
    ]);
    assert.equal(fixture.lists.at(-1).sort_by, 'display_name');
    await page.getByRole('button', { name: 'Sort descending', exact: true }).click();
    await page.getByRole('link', { name: 'Pipeline 15', exact: true }).waitFor();
    assert.equal(fixture.lists.at(-1).sort_by, 'display_name desc');
    await card(page, 'Pipeline 15')
      .getByRole('checkbox', { name: 'Select pipeline Pipeline 15' })
      .check();
    assert.equal(await page.getByRole('button', { name: 'Delete', exact: true }).isEnabled(), true);
    await Promise.all([
      page.waitForResponse(
        (response) => new URL(response.url()).searchParams.get('sort_by') === 'created_at',
      ),
      page.getByRole('combobox', { name: 'Sort pipelines' }).selectOption('created_at'),
    ]);
    await page.getByRole('button', { name: 'Sort descending', exact: true }).click();
    await page.getByRole('button', { name: 'Expand pipeline Pipeline 01' }).click();
    const versionsTable = card(page, 'Pipeline 01').getByRole('table', {
      name: 'Pipeline versions',
    });
    await versionsTable.getByRole('link', { name: 'Version 01', exact: true }).waitFor();
    assert.equal(fixture.lists.filter(({ resource }) => resource === 'versions').length, 1);
    assert.equal(
      await versionsTable
        .getByRole('link', { name: 'Version 01', exact: true })
        .getAttribute('href'),
      `#${pipelinePath(specialPipelineId, specialVersionId)}`,
    );
    await versionsTable.getByRole('checkbox', { name: 'Select version Version 01' }).check();
    await card(page, 'Pipeline 01').getByRole('button', { name: 'Next page', exact: true }).click();
    await versionsTable.getByRole('link', { name: 'Version 11', exact: true }).waitFor();
    assert.equal(fixture.lists.at(-1).pipelineId, specialPipelineId);
    await screenshot(page, 'pipelines-expanded-light');
    await darkTheme(page);
    await screenshot(page, 'pipelines-expanded-dark');
  });
});

test('private/shared tabs and namespace changes preserve scoped pipeline lists', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    await openList(page, fixture);
    assert.equal(fixture.lists.at(-1).namespace, 'team-a');
    await page.getByRole('tab', { name: 'Shared', exact: true }).click();
    await page.waitForURL('**/#/shared/pipelines');
    await page.getByRole('link', { name: 'Pipeline 03', exact: true }).waitFor();
    await waitForCards(page, 3);
    assert.equal(fixture.lists.at(-1).namespace, undefined);
    await page.getByRole('tab', { name: 'Private', exact: true }).click();
    await page.getByRole('link', { name: 'Pipeline 10', exact: true }).waitFor();
    await Promise.all([
      page.waitForResponse(
        (response) => new URL(response.url()).searchParams.get('namespace') === 'team-b',
      ),
      page.evaluate(() => window.selectFixtureNamespace('team-b')),
    ]);
    await page.getByRole('link', { name: 'Pipeline 02', exact: true }).waitFor();
    await waitForCards(page, 2);
    assert.equal(fixture.lists.at(-1).namespace, 'team-b');
  });
});

test('pipeline details preserve encoded IDs, rendered IR, version switching and browser reload', async () => {
  await withFixture({}, async (page, fixture) => {
    await page.goto(`${fixture.origin}/#${pipelinePath(specialPipelineId, specialVersionId)}`);
    await page.getByTestId('DagCanvas').waitFor();
    await page.locator('.react-flow__node').first().waitFor();
    await page.getByRole('button', { name: 'Show Summary', exact: true }).click();
    const version = page.getByRole('combobox', { name: 'Version', exact: true });
    assert.equal(await version.inputValue(), specialVersionId);
    await version.selectOption('version-2');
    await page.waitForURL(`**/#${pipelinePath(specialPipelineId, 'version-2')}`);
    const workerResponse = page.waitForResponse((response) =>
      /\/worker-yaml[^/]*\.js$/.test(new URL(response.url()).pathname),
    );
    await page.getByRole('tab', { name: 'Pipeline Spec', exact: true }).click();
    assert.equal((await workerResponse).status(), 200);
    await page.getByTestId('spec-ir').waitFor();
    assert.ok((await page.getByTestId('spec-ir').textContent()).includes('comp-preprocess'));
    await page.waitForFunction(() => document.querySelector('.ace_editor') !== null);
    assert.ok(
      fixture.workers.some((name) => /worker-yaml.*\.js$/.test(name)),
      'read-only YAML must load its bundled worker without a 404',
    );
    await darkTheme(page);
    const editor = page.locator('.ace_editor');
    await page.waitForFunction(() => {
      const element = document.querySelector('.ace_editor');
      return element && getComputedStyle(element).backgroundColor === 'rgb(22, 25, 33)';
    });
    assert.equal(
      await editor.evaluate((element) => getComputedStyle(element).backgroundColor),
      'rgb(22, 25, 33)',
    );
    assert.match(
      await editor.evaluate((element) => getComputedStyle(element).fontFamily),
      /JetBrains Mono/,
    );
    // Focus can scroll Ace's virtualized lines; compare its complete model.
    const readEditor = (element) => {
      const instance = element.env?.editor;
      if (!instance) throw new Error('The rendered pipeline spec has no initialized Ace editor');
      return { value: instance.getValue(), readOnly: instance.getReadOnly() };
    };
    const beforeTyping = await editor.evaluate(readEditor);
    assert.equal(beforeTyping.readOnly, true);
    assert.ok(beforeTyping.value.includes('comp-preprocess'));
    await editor.locator('.ace_text-input').focus();
    assert.equal(
      await editor.locator('.ace_text-input').evaluate((input) => document.activeElement === input),
      true,
    );
    await page.keyboard.type('read-only-check');
    assert.deepEqual(
      await editor.evaluate(readEditor),
      beforeTyping,
      'pipeline specs remain read-only',
    );
    await screenshot(page, 'pipeline-spec-dark');
    await page.reload();
    await page.getByTestId('DagCanvas').waitFor();
    await page.getByRole('button', { name: 'Show Summary', exact: true }).click();
    assert.equal(
      await page.getByRole('combobox', { name: 'Version', exact: true }).inputValue(),
      'version-2',
    );
    assert.ok(
      fixture.reads.some(
        ({ pipelineId, versionId }) =>
          pipelineId === specialPipelineId && versionId === specialVersionId,
      ),
    );
  });
});

for (const visibility of ['standalone', 'private', 'shared']) {
  test(`URL import creates a ${visibility} pipeline and its version once`, async () => {
    await withFixture(
      { namespace: visibility === 'standalone' ? undefined : 'team-a' },
      async (page, fixture) => {
        await page.goto(`${fixture.origin}/#/pipeline_versions/new`);
        await page
          .getByRole('textbox', { name: 'Pipeline Name', exact: true })
          .fill('browser-created');
        if (visibility !== 'standalone')
          await page
            .getByRole('radio', {
              name: visibility === 'private' ? 'Private' : 'Shared',
              exact: true,
            })
            .check();
        await page
          .getByRole('textbox', { name: 'Package Url', exact: true })
          .fill('https://example.test/fixture.yaml');
        await page
          .getByRole('textbox', { name: 'Code Source', exact: true })
          .fill('https://example.test/source');
        await page.getByRole('button', { name: 'Create', exact: true }).click();
        await page.waitForURL(`**/#${pipelinePath('created-pipeline', 'created-version')}`);
        await page.getByTestId('DagCanvas').waitFor();
        assert.deepEqual(
          fixture.mutations.map(({ kind }) => kind),
          ['create-pipeline', 'create-version'],
        );
        assert.equal(
          fixture.mutations[0].data.namespace,
          visibility === 'private' ? 'team-a' : undefined,
        );
        assert.equal(
          fixture.mutations[1].data.package_url.pipeline_url,
          'https://example.test/fixture.yaml',
        );
        assert.equal(fixture.mutations[1].data.code_source_url, 'https://example.test/source');
      },
    );
  });
}

test('file upload validates the package and preserves the existing pipeline/version relationship', async () => {
  await withFixture({}, async (page, fixture) => {
    await page.goto(
      `${fixture.origin}/#/pipeline_versions/new?pipelineId=${encodeURIComponent(specialPipelineId)}`,
    );
    await page.getByRole('radio', { name: 'Upload a file', exact: true }).check();
    const input = page.locator('input[type="file"]');
    await input.setInputFiles({
      name: 'rejected.gz',
      mimeType: 'application/gzip',
      buffer: Buffer.from('rejected'),
    });
    await page
      .getByText('Invalid file type. Supported formats: .yaml, .yml, .zip, .tar.gz', {
        exact: true,
      })
      .waitFor();
    assert.equal(
      await page.getByRole('button', { name: 'Create', exact: true }).isDisabled(),
      true,
    );
    assert.equal(fixture.mutations.length, 0);
    await input.setInputFiles({
      name: 'fixture.yaml',
      mimeType: 'application/yaml',
      buffer: Buffer.from(JSON.stringify(pipelineSpec)),
    });
    await page
      .getByRole('textbox', { name: 'Pipeline Version Name', exact: true })
      .fill('browser-version');
    await page.getByRole('button', { name: 'Create', exact: true }).click();
    await page.waitForURL(`**/#${pipelinePath(specialPipelineId, 'created-version')}`);
    await page.getByTestId('DagCanvas').waitFor();
    assert.deepEqual(
      fixture.mutations.map(({ kind }) => kind),
      ['upload-version'],
    );
    assert.equal(fixture.mutations[0].data.pipelineid, specialPipelineId);
    assert.equal(fixture.mutations[0].data.name, 'browser-version');
  });
});

test('local file upload creates a private pipeline using its namespace and package body', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    await page.goto(`${fixture.origin}/#/pipeline_versions/new`);
    await page.getByRole('radio', { name: 'Private', exact: true }).check();
    await page.getByRole('radio', { name: 'Upload a file', exact: true }).check();
    await page.locator('input[type="file"]').setInputFiles({
      name: 'fixture.yaml',
      mimeType: 'application/yaml',
      buffer: Buffer.from(JSON.stringify(pipelineSpec)),
    });
    await page.getByRole('textbox', { name: 'Pipeline Name', exact: true }).fill('browser-upload');
    await page.getByRole('button', { name: 'Create', exact: true }).click();
    await page.waitForURL(`**/#${pipelinePath('created-pipeline', specialVersionId)}`);
    await page.getByTestId('DagCanvas').waitFor();
    assert.deepEqual(
      fixture.mutations.map(({ kind }) => kind),
      ['upload-pipeline'],
    );
    assert.equal(fixture.mutations[0].data.namespace, 'team-a');
    assert.equal(fixture.mutations[0].data.name, 'browser-upload');
  });
});
