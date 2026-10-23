// Copyright 2026 The Kubeflow Authors. Licensed under the Apache License, Version 2.0.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { test } from 'node:test';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium } from 'playwright';

const repo = fileURLToPath(new URL('../../', import.meta.url));
const python = process.env.KFP_PYTHON || path.join(repo, '.venv/bin/python');

async function until(predicate, message) {
  for (let attempt = 0; attempt < 150; attempt++) {
    if (await predicate()) return;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(message);
}

async function stop(child) {
  if (child.exitCode !== null || child.signalCode !== null) return;
  const exited = once(child, 'exit');
  child.kill('SIGINT');
  const timer = setTimeout(() => child.kill('SIGKILL'), 5000);
  await exited;
  clearTimeout(timer);
}

test(
  'exports work offline and both watch formats recover from invalid Python',
  { timeout: 60000 },
  async () => {
    const directory = await mkdtemp(path.join(tmpdir(), 'kfp-viewer-smoke-'));
    const source = path.join(directory, 'pipeline.py');
    const helper = path.join(directory, 'helper.py');
    const html = path.join(directory, 'pipeline.html');
    const markdown = path.join(directory, 'pipeline.md');
    const children = [];
    let browser;
    try {
      await writeFile(helper, 'NAME = "first"\n');
      const validSource = `from kfp import dsl
from helper import NAME
@dsl.container_component
def step(message: str) -> dsl.ContainerSpec:
    return dsl.ContainerSpec(image="alpine", command=["echo", message])
@dsl.container_component
def train(model: dsl.Output[dsl.Model]) -> dsl.ContainerSpec:
    return dsl.ContainerSpec(image="alpine", command=["echo", model.path])
@dsl.container_component
def evaluate(model: dsl.Input[dsl.Model]) -> dsl.ContainerSpec:
    return dsl.ContainerSpec(image="alpine", command=["echo", model.path])
@dsl.pipeline(name="training")
def training():
    model = train().set_display_name("Train model")
    evaluate(model=model.outputs["model"]).set_display_name("Evaluate model")
@dsl.pipeline(name=NAME)
def pipeline():
    first = step(message="hello").set_display_name("Load data")
    group = training().after(first).set_display_name("Training pipeline")
    step(message="world").after(group).set_display_name("Deploy model")
`;
      await writeFile(source, validSource);
      const logs = [];
      const child = spawn(
        python,
        [
          '-c',
          'from kfp.cli.__main__ import main; main()',
          'visualize',
          source,
          '--function',
          'pipeline',
          '--format',
          'both',
          '-w',
          '-s',
          '-n',
        ],
        { cwd: repo },
      );
      children.push(child);
      child.stdout.on('data', (data) => logs.push(data.toString()));
      child.stderr.on('data', (data) => logs.push(data.toString()));
      let url;
      await until(
        () => {
          url = logs.join('').match(/Live viewer: (http:\/\/[^\s]+)/)?.[1];
          return url;
        },
        'No live viewer URL: ' + logs.join(''),
      );
      await until(
        async () => (await readFile(markdown, 'utf8').catch(() => '')).includes('# first'),
        'No Markdown export',
      );

      browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
      const offline = await browser.newPage({ viewport: { width: 1280, height: 720 } });
      const errors = [];
      offline.on('pageerror', (error) => errors.push(error.message));
      await offline.context().setOffline(true);
      await offline.goto(pathToFileURL(html).href);
      await offline.locator('.react-flow__node').first().waitFor();
      await offline.locator('.react-flow__minimap-node').first().waitFor();
      assert.equal(await offline.locator('.react-flow__node-EXECUTION').count(), 4);
      assert.equal(await offline.locator('.react-flow__node-ARTIFACT').count(), 1);
      assert.equal(await offline.locator('.react-flow__node-subDagGroup').count(), 1);
      await offline.getByRole('group', { name: 'Graph controls' }).waitFor();
      await offline.getByRole('button', { name: 'Collapse Training pipeline' }).click();
      assert.equal(
        await offline.getByRole('button', { name: 'Train model', exact: true }).count(),
        0,
      );
      await offline.getByRole('button', { name: 'Expand Training pipeline' }).click();
      await offline.getByRole('button', { name: 'Train model', exact: true }).waitFor();
      for (const name of ['Load data', 'model']) {
        const node = offline.getByRole('button', { name, exact: true });
        const shadow = () => node.evaluate((element) => getComputedStyle(element).boxShadow);
        const unselected = await shadow();
        await node.click();
        // Allow the 150ms ring transition and 225ms inspector slide to finish.
        await offline.waitForTimeout(250);
        const focused = await shadow();
        assert.notEqual(focused, unselected, `${name} should highlight when selected`);
        await offline.getByRole('region', { name: 'Pipeline graph' }).focus();
        await offline.waitForTimeout(250);
        assert.equal(
          await node.evaluate((element) =>
            element.closest('.react-flow__node').classList.contains('selected'),
          ),
          true,
        );
        assert.equal(await shadow(), focused, `${name} must stay highlighted after focus moves`);
        await offline.getByRole('button', { name: 'close', exact: true }).click();
        await offline.waitForTimeout(250);
        assert.equal(await shadow(), unselected, `${name} must lose its highlight when deselected`);
      }
      if (process.env.KFP_VIEWER_SCREENSHOTS) {
        await offline.screenshot({
          path: path.join(process.env.KFP_VIEWER_SCREENSHOTS, 'kfp-viewer-desktop.png'),
        });
        await offline.setViewportSize({ width: 390, height: 844 });
        await offline.reload();
        await offline.locator('.react-flow__minimap-node').first().waitFor();
        await offline.screenshot({
          path: path.join(process.env.KFP_VIEWER_SCREENSHOTS, 'kfp-viewer-mobile.png'),
        });
      }
      assert.deepEqual(errors, []);

      // Exercise the same opaque-origin srcdoc sandbox used by HTML preview hosts.
      const sandboxPage = await browser.newPage();
      const sandboxRequests = [];
      sandboxPage.on('request', (request) => sandboxRequests.push(request.url()));
      const artifact = (await readFile(html, 'utf8')).replace(
        /(<script id="kfp-data" type="application\/json">)(.*?)(<\/script>)/s,
        (_match, open, data, close) => {
          const payload = JSON.parse(data);
          // JSON artifact schemas exercise the inspector's read-only code editor.
          for (const component of Object.values(payload.state.spec.components)) {
            for (const artifact of Object.values(component.outputDefinitions?.artifacts ?? {})) {
              artifact.artifactType = { instanceSchema: '{"type":"object"}' };
            }
          }
          return open + JSON.stringify(payload).replaceAll('<', '\\u003c') + close;
        },
      );
      await sandboxPage.setContent(
        '<iframe title="Artifact" sandbox="allow-scripts" style="width:100%;height:600px"></iframe>',
      );
      await sandboxPage.locator('iframe').evaluate((frame, html) => {
        frame.srcdoc = html;
      }, artifact);
      const frame = sandboxPage.frameLocator('iframe');
      await frame.getByRole('button', { name: 'Train model', exact: true }).click();
      await frame.locator('.ace_editor').waitFor();
      await sandboxPage.waitForTimeout(500);
      assert.deepEqual(sandboxRequests, [], 'Sandboxed inspector must not load external assets');

      const live = await browser.newPage({ viewport: { width: 1280, height: 720 } });
      live.on('pageerror', (error) => errors.push(error.message));
      await live.goto(url);
      await live.locator('.react-flow__node').first().waitFor();
      await live.getByRole('button', { name: 'Collapse Training pipeline' }).click();
      await live.getByRole('button', { name: 'Load data', exact: true }).click();
      await live.getByText('Image', { exact: true }).waitFor();
      await live.waitForTimeout(250);
      const selectedNode = live.getByRole('button', { name: 'Load data', exact: true });
      const selectedShadow = await selectedNode.evaluate(
        (element) => getComputedStyle(element).boxShadow,
      );
      await live.getByRole('button', { name: /zoom out/i }).click();
      await live.waitForTimeout(400); // Wait for the zoom animation before comparing transforms.
      const transform = await live.locator('.react-flow__viewport').getAttribute('style');
      await writeFile(helper, 'NAME = "other"\n');
      await live.getByRole('heading', { name: 'other', exact: true }).waitFor();
      assert.equal(await live.locator('.react-flow__viewport').getAttribute('style'), transform);
      assert.equal(
        await selectedNode.evaluate((element) => getComputedStyle(element).boxShadow),
        selectedShadow,
      );
      assert.equal(await live.getByRole('button', { name: 'Expand Training pipeline' }).count(), 1);
      await live.getByText('Image', { exact: true }).waitFor();
      await live.getByText('alpine', { exact: true }).waitFor();
      await until(
        async () => (await readFile(markdown, 'utf8')).includes('# other'),
        'Markdown did not refresh',
      );

      const lastHtml = await readFile(html, 'utf8');
      assert.match(lastHtml, /"name": "other"/);
      const lastMarkdown = await readFile(markdown, 'utf8');
      await writeFile(source, 'invalid Python !!!');
      await live.getByRole('alert').waitFor();
      assert.equal(await live.locator('.react-flow__node-EXECUTION').count(), 2);
      assert.equal(await readFile(markdown, 'utf8'), lastMarkdown);
      assert.equal(await readFile(html, 'utf8'), lastHtml);
      await writeFile(source, validSource);
      await live.getByRole('alert').waitFor({ state: 'hidden' });
      assert.deepEqual(errors, []);
    } finally {
      await browser?.close();
      await Promise.all(children.map(stop));
      await rm(directory, { recursive: true, force: true });
    }
  },
);
