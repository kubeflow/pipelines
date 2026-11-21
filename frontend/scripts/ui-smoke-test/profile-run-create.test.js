// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

const assert = require('node:assert/strict');
const { randomUUID } = require('node:crypto');
const { writeFile } = require('node:fs/promises');
const { isAbsolute } = require('node:path');
const { test } = require('node:test');
const { setTimeout: delay } = require('node:timers/promises');
const { chromium } = require('playwright');
const { startFixtureGateway } = require('./profile-auth-gateway');

async function submit(page, path, button) {
  const [response] = await Promise.all([
    page.waitForResponse(res => res.request().method() === 'POST' && new URL(res.url()).pathname === path),
    button.click(),
  ]);
  assert.ok(response.ok(), path + ' returned HTTP ' + response.status() + ': ' + (response.ok() ? '' : await response.text()));
  return response.json();
}

async function uploadPipeline(page, namespace, name, file) {
  await page.goto('/#/pipeline_versions/new');
  // The selector renders before the page's asynchronous build info settles.
  // Make the choice explicitly so the form receives its namespace-scoped value.
  await page.locator('#createNewSharedPipelineBtn input').check();
  await page.locator('#createNewPrivatePipelineBtn input').check();
  await page.getByRole('radio', { name: 'Upload a file', exact: true }).check();
  await page.locator('input[type="file"]').setInputFiles(file);
  await page.locator('#newPipelineName').fill(name);
  const pipeline = await submit(page, '/apis/v2beta1/pipelines/upload', page.locator('#createNewPipelineOrVersionBtn'));
  assert.equal(pipeline.namespace, namespace, 'Pipeline upload was not scoped to the selected profile');
  assert.ok(pipeline.pipeline_id, 'Upload returned no pipeline ID');
  await page.waitForURL(url => url.hash.startsWith('#/pipelines/details/' + pipeline.pipeline_id + '/version/'));
  const versionId = new URL(page.url()).hash.split('/version/')[1].split('?')[0];
  assert.ok(versionId, 'Uploaded pipeline has no version');
  return { pipelineId: pipeline.pipeline_id, versionId };
}

async function createExperiment(page, namespace, name) {
  await page.goto('/#/experiments/new');
  await page.locator('#experimentName').fill(name);
  const experiment = await submit(page, '/apis/v2beta1/experiments', page.locator('#createExperimentBtn'));
  assert.equal(experiment.namespace, namespace);
  assert.ok(experiment.experiment_id, 'Create experiment returned no ID');
  await page.waitForURL(url => url.hash.startsWith('#/runs/new'));
  return experiment.experiment_id;
}

async function createRun(page, pipeline, experimentId, name, score) {
  const query = new URLSearchParams({
    experimentId, pipelineId: pipeline.pipelineId, pipelineVersionId: pipeline.versionId,
  });
  await page.goto('/#/runs/new?' + query);
  await page.getByLabel('Run name', { exact: false }).fill(name);
  await page.getByLabel('Service Account', { exact: false }).fill('default-editor');
  await page.locator('input#text').fill(name);
  await page.locator('input#score').fill(String(score));
  const run = await submit(page, '/apis/v2beta1/runs', page.locator('#startNewRunBtn'));
  assert.ok(run.run_id, 'Start returned no run ID');
  await page.waitForURL(url => url.hash === '#/runs/details/' + run.run_id);
  return run.run_id;
}

async function waitForRun(page, namespace, id, name, pipelineId, experimentId, score) {
  const deadline = Date.now() + 600000;
  let state;
  do {
    const response = await page.request.get('/apis/v2beta1/runs/' + encodeURIComponent(id));
    assert.equal(response.status(), 200, 'Cannot read created run ' + id);
    const run = await response.json();
    assert.equal(run.namespace, namespace);
    assert.equal(run.display_name, name);
    assert.equal(run.experiment_id, experimentId);
    assert.equal(run.pipeline_version_reference?.pipeline_id, pipelineId);
    assert.equal(run.runtime_config?.parameters?.text, name);
    assert.equal(run.runtime_config?.parameters?.score, score);
    state = run.state;
    if (state === 'SUCCEEDED') return;
    assert.ok(!['FAILED', 'CANCELED', 'CANCELING', 'PAUSED', 'SKIPPED'].includes(state),
      'Run ' + id + ' entered ' + state);
    await delay(5000);
  } while (Date.now() < deadline);
  assert.fail('Run ' + id + ' timed out in ' + state);
}

test('each browser user uploads a private pipeline and completes two runs', async t => {
  const { PROFILE_UI_URL, PROFILE_PIPELINE_FILE, PROFILE_RUN_OUTPUT } = process.env;
  assert.ok(PROFILE_PIPELINE_FILE && isAbsolute(PROFILE_PIPELINE_FILE), 'Expected an absolute PROFILE_PIPELINE_FILE');
  assert.ok(PROFILE_RUN_OUTPUT && isAbsolute(PROFILE_RUN_OUTPUT), 'Expected an absolute PROFILE_RUN_OUTPUT');
  const profiles = [
    { identity: process.env.PROFILE_OWNER, namespace: process.env.PROFILE_OWNER_NAMESPACE, prefix: '', scores: [1717, 1212] },
    { identity: process.env.PROFILE_FOREIGN, namespace: process.env.PROFILE_FOREIGN_NAMESPACE, prefix: 'foreign_', scores: [2727, 2323] },
  ];
  for (const profile of profiles) assert.ok(profile.identity && profile.namespace);
  assert.notEqual(profiles[0].identity, profiles[1].identity);
  assert.notEqual(profiles[0].namespace, profiles[1].namespace);
  const gateway = await startFixtureGateway(PROFILE_UI_URL || 'http://127.0.0.1:3000', profiles.map(p => p.identity));
  t.after(() => gateway.close());
  const browser = await chromium.launch();
  t.after(() => browser.close());

  const runs = [];
  for (const profile of profiles) {
    const context = await browser.newContext({ baseURL: gateway.url });
    // The fixture has no enclosing Central Dashboard; provide its selected-namespace event.
    await context.addInitScript(namespace => {
      window.centraldashboard = { CentralDashboardEventHandler: { init(register) {
        const handler = {};
        register(handler);
        handler.onNamespaceSelected(namespace);
      } } };
    }, profile.namespace);
    await context.route('**/dashboard_lib.bundle.js', route =>
      route.fulfill({ status: 200, contentType: 'application/javascript', body: '' }),
    );
    const page = await context.newPage();
    page.on('requestfailed', request => {
      const url = new URL(request.url());
      if (url.origin === gateway.url && url.pathname.startsWith('/apis/')) {
        t.diagnostic(request.method() + ' ' + url.pathname + ': ' + request.failure()?.errorText);
      }
    });
    await gateway.signIn(page, profile.identity);
    const name = 'profile-upgrade-' + randomUUID();
    const pipeline = await uploadPipeline(page, profile.namespace, name, PROFILE_PIPELINE_FILE);
    const experiment = await createExperiment(page, profile.namespace, name);
    for (const [i, score] of profile.scores.entries()) {
      const runName = name + '-' + score;
      const id = await createRun(page, pipeline, experiment, runName, score);
      runs.push({ page, namespace: profile.namespace, id, name: runName, pipelineId: pipeline.pipelineId,
        experiment, score, key: profile.prefix + (i === 0 ? '' : 'comparison_') + 'run' });
    }
  }
  for (const run of runs) {
    await waitForRun(run.page, run.namespace, run.id, run.name, run.pipelineId, run.experiment, run.score);
  }
  assert.equal(new Set(runs.map(run => run.id)).size, 4);
  const output = runs.flatMap(run => [run.key + '_id=' + run.id, run.key + '_name=' + run.name]);
  await writeFile(PROFILE_RUN_OUTPUT, output.join('\n') + '\n', { mode: 0o600 });
});
