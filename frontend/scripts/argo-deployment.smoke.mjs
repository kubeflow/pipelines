// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createHash, randomUUID } from 'node:crypto';
import { readFile, writeFile } from 'node:fs/promises';
import { test } from 'node:test';
import { chromium } from 'playwright';

const context = process.env.ARGO_CONTEXT || 'orbstack';
const namespace = process.env.ARGO_NAMESPACE || 'kfp-argo';
const origin = process.env.ARGO_UI_URL || 'http://127.0.0.1:3003';
const editor = process.env.ARGO_EDITOR_TOKEN;
const reader = process.env.ARGO_READER_TOKEN;
let versionName = process.env.ARGO_VERSION_NAME;
const encode = encodeURIComponent;
const workflowUrl = (name) => `/argo/api/v1/workflows/${namespace}/${encode(name)}`;
const kubectl = (args) =>
  execFileSync('kubectl', ['--context', context, '-n', namespace, ...args], {
    encoding: 'utf8',
    timeout: 180000,
  });

async function api(path, options = {}, token = editor) {
  const response = await fetch(origin + path, {
    ...options,
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });
  assert.ok(
    response.ok,
    `${options.method || 'GET'} ${path}: HTTP ${response.status}: ${await (response.ok ? Promise.resolve('') : response.text())}`,
  );
  return response;
}
async function json(path, options, token) {
  return (await api(path, options, token)).json();
}
async function waitFor(check, description, timeout = 180000) {
  const end = Date.now() + timeout;
  let last;
  while (Date.now() < end) {
    try {
      const result = await check();
      if (result) return result;
    } catch (error) {
      last = error;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`Timed out: ${description}${last ? `: ${last.message}` : ''}`);
}
async function completed(name) {
  return waitFor(async () => {
    const run = await json(workflowUrl(name));
    assert.ok(
      !['Failed', 'Error'].includes(run.status?.phase),
      `Unexpected ${run.status?.phase}: ${name}`,
    );
    return run.status?.phase === 'Succeeded' && run;
  }, `workflow ${name}`);
}
const metadataPath = (run, node, archived = false) =>
  archived
    ? `/argo/artifact-files/${namespace}/archived-workflows/${run.metadata.uid}/${node.id}/outputs/kfp-metadata`
    : `/argo/artifacts/${namespace}/${run.metadata.name}/${node.id}/kfp-metadata`;
async function connect(page, token) {
  await page.getByLabel('Kubernetes bearer token').fill(token);
  await page.getByRole('button', { name: 'Connect', exact: true }).click();
  await page.getByRole('tab', { name: 'Pipelines and versions', exact: true }).waitFor();
}
async function saveResponse(page, path, action) {
  const response = page.waitForResponse(
    (response) => response.url().includes(path) && response.request().method() === 'POST',
  );
  await action();
  const result = await response;
  assert.ok(result.ok(), `${path}: ${result.status()} ${await result.text()}`);
  return result.json();
}

test('deployed native lifecycle and RBAC', { timeout: 720000 }, async () => {
  assert.ok(
    editor && reader && versionName,
    'Set ARGO_EDITOR_TOKEN, ARGO_READER_TOKEN and ARGO_VERSION_NAME.',
  );
  const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || 'chrome' });
  const page = await browser.newPage({ viewport: { width: 1500, height: 1100 } });
  const errors = [];
  const paths = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('request', (request) => paths.push(new URL(request.url()).pathname));
  const report = { context, namespace, versionName };
  try {
    await page.goto(`${origin}/argo.html?namespace=${namespace}`);
    await connect(page, editor);
    // Give this verification pair its own cache keys without clearing other runs.
    const original = await json(`/argo/api/v1/workflow-templates/${namespace}/${versionName}`);
    const freshSpec = structuredClone(original.spec);
    const salt = randomUUID();
    for (const template of freshSpec.templates)
      if (template.memoize) template.memoize.key = salt + '-' + template.memoize.key;
    delete freshSpec.workflowMetadata;
    const hash = createHash('sha256').update(JSON.stringify(freshSpec)).digest('hex').slice(0, 12);
    const labels = { ...original.metadata.labels, 'pipelines.kubeflow.org/version': hash };
    freshSpec.workflowMetadata = { labels };
    const fresh = {
      apiVersion: original.apiVersion,
      kind: 'WorkflowTemplate',
      metadata: {
        name: `verification-${hash}`,
        namespace,
        labels,
        annotations: original.metadata.annotations,
      },
      spec: freshSpec,
    };
    await page.getByRole('tab', { name: 'Pipelines and versions', exact: true }).click();
    await page.getByLabel('WorkflowTemplate YAML', { exact: true }).fill(JSON.stringify(fresh));
    const uploaded = await saveResponse(page, '/workflow-templates/', () =>
      page.getByRole('button', { name: 'Upload version', exact: true }).click(),
    );
    versionName = uploaded.metadata.name;
    report.versionName = versionName;
    await page.getByRole('tab', { name: 'Experiments', exact: true }).click();
    const displayName = `Native training ${Date.now()}`;
    await page.getByRole('textbox', { name: /^Experiment name/ }).fill(displayName);
    await page
      .getByLabel('Description', { exact: true })
      .fill('Native Argo end-to-end verification');
    const experiment = await saveResponse(page, '/configmaps', () =>
      page.getByRole('button', { name: 'Create experiment', exact: true }).click(),
    );
    report.experiment = experiment.metadata.name;
    await page
      .getByRole('heading', { name: displayName, exact: true })
      .locator('..')
      .getByRole('button', { name: 'View runs', exact: true })
      .click();
    await page.getByText('No labeled workflows found.', { exact: true }).waitFor();
    await page.getByRole('tab', { name: 'Pipelines and versions', exact: true }).click();
    await page.getByRole('button', { name: `Use ${versionName}`, exact: true }).click();
    await page.getByRole('combobox', { name: /^Experiment(?:\s|$)/ }).click();
    await page.getByRole('option', { name: displayName, exact: true }).click();
    const submitted = [];
    for (const factor of ['1.0', '1.2']) {
      await page.getByLabel('factor', { exact: true }).fill(factor);
      const run = await saveResponse(page, `/workflows/${namespace}`, () =>
        page.getByRole('button', { name: 'Start run', exact: true }).click(),
      );
      submitted.push(await completed(run.metadata.name));
    }
    report.runs = submitted.map((run) => ({ name: run.metadata.name, uid: run.metadata.uid }));
    const taskMetadata = [];
    for (const [index, run] of submitted.entries()) {
      const raw = JSON.parse(kubectl(['get', 'workflow', run.metadata.name, '-o', 'json']));
      assert.ok(raw.status.offloadNodeStatusVersion, 'Node status was not offloaded');
      assert.ok(!raw.status.nodes || Object.keys(raw.status.nodes).length === 0);
      const nodes = Object.values(run.status.nodes);
      const prepare = nodes.find((node) => node.displayName === 'prepare');
      assert.equal(prepare.memoizationStatus.hit, Boolean(index));
      const metrics = await json(
        metadataPath(
          run,
          nodes.find((node) => node.displayName === 'evaluate'),
        ),
      );
      assert.equal(metrics.metrics.score, index ? 1 : 2 / 3);
      taskMetadata.push(await json(metadataPath(run, prepare)));
      const pods = JSON.parse(
        kubectl([
          'get',
          'pods',
          '-l',
          `workflows.argoproj.io/workflow=${run.metadata.name}`,
          '-o',
          'json',
        ]),
      );
      assert.equal(pods.items.length, index ? 2 : 3);
      for (const pod of pods.items) {
        assert.deepEqual(pod.spec.containers.map((container) => container.name).sort(), [
          'main',
          'wait',
        ]);
        const image = pod.spec.containers.find((container) => container.name === 'main').image;
        const digest = image.match(/:kfp-([0-9a-f]{64})$/)?.[1];
        assert.ok(digest, 'Component image was not a content-named snapshot');
        assert.ok(
          pod.status.containerStatuses
            .find((container) => container.name === 'main')
            .imageID.endsWith(digest),
          'Running image differs from its declared build identity',
        );
      }
    }
    assert.deepEqual(taskMetadata[0], taskMetadata[1], 'Cached producer metadata changed');
    await page.getByRole('tab', { name: 'Runs', exact: true }).click();
    const runList = page.getByRole('complementary', { name: 'Runs' });
    await runList.getByRole('button', { name: submitted[1].metadata.name, exact: true }).click();
    await page.getByRole('button', { name: 'prepare', exact: true }).click();
    await page.getByRole('link', { name: 'dataset', exact: true }).waitFor();
    const downloading = page.waitForEvent('download');
    await page.getByRole('link', { name: 'dataset', exact: true }).click();
    const downloaded = await downloading;
    assert.deepEqual(JSON.parse(await readFile(await downloaded.path(), 'utf8')), [1, 2, 3, 4]);
    for (const run of submitted)
      await runList
        .getByRole('button', { name: run.metadata.name, exact: true })
        .locator('..')
        .getByRole('checkbox', { name: 'Compare' })
        .check();
    await page.getByRole('cell', { name: 'evaluate.score: 1', exact: true }).waitFor();
    await page
      .getByRole('cell', { name: 'evaluate.score: 0.6666666666666666', exact: true })
      .waitFor();
    await page.getByRole('button', { name: 'Show run logs', exact: true }).click();
    await page
      .getByText(/Looking for component `train`/)
      .first()
      .waitFor();
    await page.screenshot({ path: '/tmp/kfp-argo-deployed-runs.png', fullPage: true });

    // Recurring runs are ordinary, persistent CronWorkflows, not UI state.
    await page.getByRole('tab', { name: 'Pipelines and versions', exact: true }).click();
    const cron = await saveResponse(page, '/cron-workflows/', () =>
      page.getByRole('button', { name: 'Create suspended schedule', exact: true }).click(),
    );
    assert.equal(cron.spec.suspend, true);
    assert.deepEqual(cron.spec.schedules, ['0 9 * * *']);
    report.schedule = cron.metadata.name;
    await page.reload();
    await connect(page, editor);
    await page.getByRole('tab', { name: 'Recurring runs', exact: true }).click();
    await page.getByRole('heading', { name: cron.metadata.name, exact: true }).waitFor();
    kubectl([
      'patch',
      'cronworkflow',
      cron.metadata.name,
      '--type=merge',
      '-p',
      JSON.stringify({ spec: { schedules: ['0 10 * * *'] } }),
    ]);
    await page
      .getByRole('heading', { name: cron.metadata.name, exact: true })
      .locator('..')
      .getByText('0 10 * * * · Suspended', { exact: true })
      .waitFor();

    // Add a native suspend gate without changing KFP's execution model.
    const version = await json(`/argo/api/v1/workflow-templates/${namespace}/${versionName}`);
    const spec = structuredClone(version.spec);
    delete spec.workflowMetadata;
    const root = spec.templates.find((template) => template.name === spec.entrypoint);
    root.dag.tasks.push({ name: 'approval', template: 'native-approval', dependencies: ['train'] });
    root.dag.tasks.find((task) => task.name === 'evaluate').dependencies.push('approval');
    spec.templates.push({ name: 'native-approval', suspend: {} });
    const gate = await json(`/argo/api/v1/workflows/${namespace}`, {
      method: 'POST',
      body: JSON.stringify({
        namespace,
        workflow: {
          apiVersion: 'argoproj.io/v1alpha1',
          kind: 'Workflow',
          metadata: {
            generateName: 'native-approval-',
            labels: { 'pipelines.kubeflow.org/pipeline': 'native-approval' },
          },
          spec,
        },
      }),
    });
    report.approval = gate.metadata.name;
    await waitFor(
      async () =>
        Object.values((await json(workflowUrl(gate.metadata.name))).status?.nodes || {}).some(
          (node) => node.type === 'Suspend' && node.phase === 'Running',
        ),
      'approval gate',
    );
    await page.getByRole('button', { name: 'Disconnect', exact: true }).click();
    await connect(page, reader);
    await page.getByRole('tab', { name: 'Runs', exact: true }).click();
    await page
      .getByRole('complementary', { name: 'Runs' })
      .getByRole('button', { name: gate.metadata.name, exact: true })
      .click();
    await page.getByRole('button', { name: 'Resume suspended tasks', exact: true }).click();
    await page
      .getByText(/Argo request failed \(403\)/)
      .first()
      .waitFor();
    assert.equal((await json(workflowUrl(gate.metadata.name))).status.phase, 'Running');
    await page.getByRole('tab', { name: 'Experiments', exact: true }).click();
    await page.getByRole('textbox', { name: /^Experiment name/ }).fill('Denied experiment');
    await page.getByRole('button', { name: 'Create experiment', exact: true }).click();
    await page
      .getByText(/Argo request failed \(403\)/)
      .first()
      .waitFor();
    await page.screenshot({ path: '/tmp/kfp-argo-denied.png', fullPage: true });
    await page.getByRole('button', { name: 'Disconnect', exact: true }).click();
    await connect(page, editor);
    await page.getByRole('tab', { name: 'Runs', exact: true }).click();
    await page
      .getByRole('complementary', { name: 'Runs' })
      .getByRole('button', { name: gate.metadata.name, exact: true })
      .click();
    await page.getByRole('button', { name: 'Resume suspended tasks', exact: true }).click();
    await completed(gate.metadata.name);

    // Archive survives deletion of the live CR; no second run identity is invented.
    const first = submitted[0];
    const archivePath = `/argo/api/v1/archived-workflows/${first.metadata.uid}?namespace=${namespace}`;
    await waitFor(
      async () => (await json(archivePath)).metadata.uid === first.metadata.uid,
      'workflow archive',
    );
    kubectl(['delete', 'workflow', first.metadata.name]);
    const archived = await json(archivePath);
    const prepared = Object.values(archived.status.nodes).find(
      (node) => node.displayName === 'prepare',
    );
    assert.deepEqual(await json(metadataPath(archived, prepared, true)), taskMetadata[0]);
    await page
      .getByRole('complementary', { name: 'Runs' })
      .getByRole('button', { name: first.metadata.name, exact: true })
      .click();
    await page.getByRole('button', { name: 'prepare', exact: true }).click();
    await page.getByRole('link', { name: 'dataset', exact: true }).waitFor();
    await page.getByRole('button', { name: 'Show run logs', exact: true }).click();
    await page
      .getByText(/Looking for component `prepare`/)
      .first()
      .waitFor();
    await page.screenshot({ path: '/tmp/kfp-argo-archived.png', fullPage: true });
    const listed = (await json(`/argo/api/v1/archived-workflows?namespace=${namespace}`)).items;
    assert.equal(listed.filter((run) => run.metadata.uid === first.metadata.uid).length, 1);

    await page.getByRole('tab', { name: 'Experiments', exact: true }).click();
    const experimentCard = page
      .getByRole('heading', { name: displayName, exact: true })
      .locator('..');
    page.once('dialog', (dialog) => dialog.accept());
    await experimentCard.getByRole('button', { name: 'Delete experiment', exact: true }).click();
    await waitFor(
      async () =>
        !(
          await json(
            `/kubernetes/api/v1/namespaces/${namespace}/configmaps?labelSelector=pipelines.kubeflow.org%2Fresource%3Dexperiment`,
          )
        ).items.some((item) => item.metadata.uid === experiment.metadata.uid),
      'experiment deletion',
    );
    assert.equal(
      (await json(workflowUrl(submitted[1].metadata.name))).metadata.uid,
      submitted[1].metadata.uid,
    );
    assert.equal((await json(archivePath)).metadata.uid, first.metadata.uid);
    assert.equal(
      (await json(`/argo/api/v1/cron-workflows/${namespace}/${cron.metadata.name}`)).metadata.uid,
      cron.metadata.uid,
    );

    // The archive and cached payloads are on PVCs, independent of application restarts.
    kubectl(['rollout', 'restart', 'deployment/minio', 'deployment/postgres']);
    kubectl(['rollout', 'status', 'deployment/minio', '--timeout=180s']);
    kubectl(['rollout', 'status', 'deployment/postgres', '--timeout=180s']);
    await waitFor(
      async () => (await json(archivePath)).metadata.uid === first.metadata.uid,
      'persistent SQL archive after restart',
    );
    assert.deepEqual(await json(metadataPath(archived, prepared, true)), taskMetadata[0]);
    assert.deepEqual(
      await json(
        `/argo/artifact-files/${namespace}/archived-workflows/${first.metadata.uid}/${prepared.id}/outputs/dataset`,
      ),
      [1, 2, 3, 4],
    );
    assert.deepEqual(errors, []);
    assert.ok(
      !paths.some((path) => /^\/(apis|api|system|k8s)\//.test(path)),
      'Unexpected KFP backend request',
    );
    const deployments = JSON.parse(kubectl(['get', 'deployments', '-o', 'json']))
      .items.map((item) => item.metadata.name)
      .sort();
    assert.deepEqual(deployments, [
      'argo-server',
      'kfp-argo-ui',
      'minio',
      'postgres',
      'workflow-controller',
    ]);
    report.result = 'passed';
    await writeFile('/tmp/kfp-argo-deployment-evidence.json', JSON.stringify(report, null, 2));
  } catch (error) {
    await page
      .screenshot({ path: '/tmp/kfp-argo-e2e-failure.png', fullPage: true })
      .catch(() => {});
    await writeFile('/tmp/kfp-argo-e2e-failure.txt', await page.locator('body').innerText()).catch(
      () => {},
    );
    await writeFile('/tmp/kfp-argo-partial-evidence.json', JSON.stringify(report, null, 2));
    throw error;
  } finally {
    await browser.close();
  }
});
