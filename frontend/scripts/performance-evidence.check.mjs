/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import assert from 'node:assert/strict';
import { mkdtemp, writeFile, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';
import { runInNewContext } from 'node:vm';
import {
  cls,
  compareTiming,
  inventory,
  verifyBuild,
  validateSample,
  cleanupWithEvidence,
  recordScalingTrial,
  hostedReadinessProtocol,
  isExpectedLegacyWorkerError,
} from './performance-evidence.mjs';
const budgets = { samples: 7, relativeMedianAllowance: 0.1, absoluteMedianAllowanceMs: 50 };
const samples = (value) => Array(7).fill(value);
test('CLS excludes recent input, honors capture start and both session boundaries', () => {
  const shift = (time, value, recentInput = false) => ({ time, value, recentInput });
  assert.equal(
    cls([shift(0, 0.1), shift(900, 0.2), shift(950, 1, true), shift(1900, 0.2)]),
    0.30000000000000004,
  );
  assert.equal(cls([shift(0, 0.1), shift(900, 0.2)], 800), 0.2);
  assert.equal(cls(Array.from({ length: 8 }, (_, i) => shift(i * 900, 0.1))), 0.6);
  assert.throws(() => cls([shift(0, NaN)]));
});
test('timing gate retains boundary equality and rejects missing/nonfinite data', () => {
  assert.equal(compareTiming(samples(1000), samples(1100), budgets).passesProposedBudget, true);
  assert.equal(compareTiming(samples(1000), samples(1101), budgets).passesProposedBudget, false);
  assert.equal(compareTiming(samples(100), samples(150), budgets).passesProposedBudget, true);
  assert.throws(() => compareTiming(samples(1), [1], budgets));
  assert.throws(() => compareTiming(samples(1), samples(NaN), budgets));
});
test('build evidence rejects wrong sources, changed assets and injected entry paths', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'kfp-performance-evidence-'));
  try {
    await writeFile(
      join(directory, 'index.html'),
      '<script src="/app.js"></script><link href="/app.css">',
    );
    await writeFile(join(directory, 'app.js'), 'test');
    await writeFile(join(directory, 'app.css'), 'body {}');
    const sourceSha = 'a'.repeat(40);
    const provenance = { sourceSha, assets: await inventory(directory) };
    assert.ok((await verifyBuild(directory, provenance, sourceSha)).entryGzipBytes > 0);
    await assert.rejects(verifyBuild(directory, provenance, 'b'.repeat(40)), /source/);
    await writeFile(join(directory, 'app.js'), 'changed');
    await assert.rejects(verifyBuild(directory, provenance, sourceSha), /assets/);
    await writeFile(
      join(directory, 'index.html'),
      '<script src="/../app.js"></script><link href="/app.css">',
    );
    await assert.rejects(
      verifyBuild(directory, { sourceSha, assets: await inventory(directory) }, sourceSha),
      /Unknown entry/,
    );
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});

test('readiness rejects missing timestamps, failed APIs and incomplete interactions', () => {
  const base = {
    kind: 'runs',
    readiness: { readiness: { apis: [{ status: 200 }] } },
    observations: {
      observedAtMs: 200,
      paint: { shifts: [] },
      readiness: { contentReadyMs: 100, fontsReadyMs: 110 },
    },
  };
  validateSample(base);
  for (const change of [
    (sample) => {
      sample.observations.readiness.contentReadyMs = null;
    },
    (sample) => {
      sample.observations.readiness.fontsReadyMs = 201;
    },
    (sample) => {
      sample.readiness.readiness.apis = [];
    },
    (sample) => {
      sample.readiness.readiness.apis[0].status = 500;
    },
    (sample) => {
      sample.observations.paint = {};
    },
    (sample) => {
      sample.kind = 'filter';
    },
    (sample) => {
      sample.kind = 'navigation';
    },
  ]) {
    const sample = structuredClone(base);
    change(sample);
    assert.throws(() => validateSample(sample));
  }
  const navigation = {
    ...base,
    kind: 'navigation',
    runOpen: { taskNodeVisible: true, measure: [{ startTime: 120, duration: 20 }] },
    taskOpen: { taskDetailsVisible: true, measure: [{ startTime: 150, duration: 20 }] },
  };
  validateSample(navigation);
  navigation.taskOpen.measure[0].duration = 100;
  assert.throws(() => validateSample(navigation), /beyond observation/);
});

test('editor endpoints retain common model readiness and require candidate worker readiness', () => {
  const hash = 'a'.repeat(64);
  const sample = {
    kind: 'editor',
    variant: 'candidate',
    observations: { observedAtMs: 200, paint: { shifts: [] } },
    editor: {
      duration: 100,
      endpoint: 'complete-read-only-model-fonts-two-frames',
      workerStatus: 'ready',
      workerReadyMs: 110,
      readOnly: true,
      workerResponseStatus: 200,
      expectedModelSha256: hash,
      modelSha256: hash,
      workerModelSha256: hash,
    },
  };
  validateSample(sample);
  for (const field of [
    'expectedModelSha256',
    'modelSha256',
    'workerModelSha256',
    'workerResponseStatus',
    'readOnly',
    'duration',
    'endpoint',
    'workerReadyMs',
    'workerStatus',
  ]) {
    const corrupted = structuredClone(sample);
    delete corrupted.editor[field];
    assert.throws(() => validateSample(corrupted));
  }
  const legacy = structuredClone(sample);
  legacy.variant = 'legacy';
  legacy.editor.workerStatus = 'unavailable-missing-baseline-asset';
  legacy.editor.workerResponseStatus = 404;
  legacy.editor.workerPath = '/worker-yaml.js';
  delete legacy.editor.workerReadyMs;
  delete legacy.editor.workerModelSha256;
  validateSample(legacy);
  for (const change of [
    (item) => {
      item.variant = 'candidate';
    },
    (item) => {
      item.editor.workerResponseStatus = 500;
    },
    (item) => {
      item.editor.workerPath = '/other.js';
    },
    (item) => {
      item.editor.workerReadyMs = 100;
    },
    (item) => {
      item.editor.modelSha256 = 'b'.repeat(64);
    },
  ]) {
    const invalid = structuredClone(legacy);
    change(invalid);
    assert.throws(() => validateSample(invalid));
  }
});

test('cleanup rejection and timeout cannot skip later cleanup or final evidence', async () => {
  const events = [];
  const failures = [];
  await cleanupWithEvidence(
    [
      [
        'browser',
        async () => {
          events.push('browser');
          throw new Error('close rejected');
        },
      ],
      ['context', () => new Promise(() => {})],
      [
        'fixture',
        async () => {
          events.push('fixture');
        },
      ],
    ],
    (failure) => failures.push(failure),
    async () => {
      events.push('saved');
    },
    10,
  );
  assert.deepEqual(events, ['browser', 'fixture', 'saved']);
  assert.deepEqual(
    failures.map((failure) => failure.stage),
    ['browser', 'context'],
  );
  assert.match(failures[1].error, /timed out/);
});

test('later scaling failure preserves prior samples and the failing attempt on disk', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'kfp-scaling-progress-'));
  const path = join(directory, 'progress.json');
  const progress = { expectedSamples: 7, status: 'running', samples: [], attempts: [] };
  try {
    await recordScalingTrial(progress, path, 1, async () => {
      progress.samples.push({ sample: 1, readyMs: 20 });
    });
    await assert.rejects(
      recordScalingTrial(progress, path, 2, async () => {
        throw new Error('second readiness failed');
      }),
      /second readiness/,
    );
    const retained = JSON.parse(await readFile(path));
    assert.equal(retained.status, 'failed');
    assert.deepEqual(retained.samples, [{ sample: 1, readyMs: 20 }]);
    assert.deepEqual(
      retained.attempts.map((trial) => trial.status),
      ['measured', 'failed'],
    );
    assert.match(retained.attempts[1].error, /second readiness failed/);
    assert.ok(retained.attempts.every((trial) => trial.startedAt && trial.finishedAt));
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});

test('hosted readiness distinguishes initial font settling from loss after confirmation', async () => {
  const original = JSON.parse(
    await readFile(
      new URL('../docs/ui-modernization/layout-stability/protocol.json', import.meta.url),
    ),
  );
  const retained = JSON.stringify(original);
  const protocol = hostedReadinessProtocol(original);
  assert.equal(JSON.stringify(original), retained, 'Historical protocol must remain unchanged');
  assert.throws(
    () => hostedReadinessProtocol({ scripts: { preNavigation: '' } }),
    /Historical readiness/,
  );

  const runId = 'e0115ac1-0479-4194-a22d-01e65e09a32b';
  let visible = true;
  let now = 0;
  let mutation;
  let disconnects = 0;
  const frames = [];
  const timers = [];
  const window = {};
  const context = {
    window,
    URL,
    Intl,
    Date,
    setTimeout: (callback) => timers.push(callback),
    location: { hash: '#/runs/details/' + runId },
    navigator: { userAgent: 'test', language: 'en-US' },
    innerWidth: 1280,
    innerHeight: 720,
    devicePixelRatio: 1,
    matchMedia: () => ({ matches: false }),
    requestAnimationFrame: (callback) => frames.push(callback),
    getComputedStyle: () => ({ visibility: visible ? 'visible' : 'hidden' }),
    document: {
      body: { innerText: '' },
      fonts: { ready: Promise.resolve() },
      querySelectorAll: () => [],
      querySelector: () => ({
        getAttribute: () => 'task.chicago-taxi-trips-dataset',
        offsetWidth: 100,
        offsetHeight: 30,
      }),
    },
    performance: {
      now: () => now,
      getEntriesByType: (type) =>
        type === 'resource'
          ? ['', '/tasks'].map((suffix) => ({
              name: 'http://fixture/apis/v2beta1/runs/' + runId + suffix,
              responseStatus: 200,
            }))
          : [],
    },
    MutationObserver: class {
      constructor(callback) {
        mutation = callback;
      }
      observe() {}
      disconnect() {
        disconnects++;
      }
    },
    PerformanceObserver: class {
      observe() {}
      disconnect() {
        disconnects++;
      }
    },
  };
  const frame = async () => {
    now += 16;
    assert.ok(frames.length, 'Expected a scheduled confirming frame');
    frames.shift()();
    await Promise.resolve();
  };
  runInNewContext(protocol.scripts.preNavigation, context);
  mutation();
  await frame();
  await frame();
  const first = window.__kfpContentReady.firstContentReadyMs;
  visible = false;
  await frame();
  await frame();
  assert.equal(window.__kfpContentReady.contentReadyMs, null);
  assert.equal(window.__kfpContentReady.fontsReadyMs, null);
  assert.equal(window.__kfpContentReady.transientReadiness.length, 1);
  assert.equal(window.__kfpContentReady.transientReadiness[0].state.node.visible, 'hidden');
  visible = true;
  mutation();
  for (let index = 0; index < 4; index++) await frame();
  assert.ok(window.__kfpContentReady.contentReadyMs > first);
  assert.equal(window.__kfpContentReady.fontsReadyMs, window.__kfpContentReady.contentReadyMs);
  assert.equal(disconnects, 0, 'Observers remain active until capture');

  const capture = runInNewContext('(' + protocol.scripts.perfReadinessFunction + ')', context);
  visible = false;
  mutation();
  visible = true;
  mutation();
  await assert.rejects(capture('run-details'), /Readiness lost after confirmation/);
  assert.equal(window.__kfpContentReady.postConfirmationLoss.state.node.visible, 'hidden');

  // A clean capture freezes the readiness endpoint before later interaction changes.
  delete window.__kfpContentReady.postConfirmationLoss;
  assert.equal((await capture('run-details')).readiness.node.visible, 'visible');
  assert.equal(disconnects, 2);

  // Neither missing marks nor absent observations can be mistaken for confirmation.
  delete window.__kfpContentReady;
  const missing = capture('run-details');
  assert.equal(timers.length, 1);
  now += 30001;
  timers.shift()();
  await assert.rejects(missing, /Confirmed readiness not reached/);
});

test('legacy worker exception cannot hide other assets, modes or candidate errors', () => {
  const origin = 'http://127.0.0.1:4181';
  const record = {
    variant: 'legacy',
    kind: 'editor',
    editor: {
      workerStatus: 'unavailable-missing-baseline-asset',
      workerResponseStatus: 404,
      workerPath: '/worker-yaml.js',
    },
  };
  const error =
    "Failed to execute 'importScripts' on 'WorkerGlobalScope': The script at '" +
    origin +
    "/worker-yaml.js' failed to load.";
  assert.equal(isExpectedLegacyWorkerError(record, error, origin), true);
  for (const change of [
    (item) => {
      item.variant = 'candidate';
    },
    (item) => {
      item.kind = 'runs';
    },
    (item) => {
      item.editor.workerResponseStatus = 500;
    },
    (item) => {
      item.editor.workerPath = '/other.js';
    },
    (item) => {
      item.editor.workerStatus = 'ready';
    },
  ]) {
    const unexpected = structuredClone(record);
    change(unexpected);
    assert.equal(isExpectedLegacyWorkerError(unexpected, error, origin), false);
  }
  assert.equal(
    isExpectedLegacyWorkerError(record, error.replace('worker-yaml.js', 'other.js'), origin),
    false,
  );
  assert.equal(isExpectedLegacyWorkerError(record, 'Unexpected application error', origin), false);
});
