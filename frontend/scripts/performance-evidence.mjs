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
import { createHash } from 'node:crypto';
import { readFile, readdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { gzipSync } from 'node:zlib';

export const sha256 = (bytes) => createHash('sha256').update(bytes).digest('hex');

export function median(values) {
  assert.ok(values.length && values.every((value) => Number.isFinite(value) && value >= 0));
  const ordered = values.toSorted((a, b) => a - b);
  const middle = Math.floor(ordered.length / 2);
  return ordered.length % 2 ? ordered[middle] : (ordered[middle - 1] + ordered[middle]) / 2;
}

export function cls(shifts, since = 0) {
  let largest = 0;
  let sum = 0;
  let start = -Infinity;
  let previous = -Infinity;
  for (const entry of shifts.toSorted((a, b) => a.time - b.time)) {
    assert.ok(Number.isFinite(entry.time) && Number.isFinite(entry.value) && entry.value >= 0);
    if (entry.recentInput || entry.time < since) continue;
    if (entry.time - previous >= 1000 || entry.time - start >= 5000) {
      start = entry.time;
      sum = 0;
    }
    sum += entry.value;
    previous = entry.time;
    largest = Math.max(largest, sum);
  }
  return largest;
}

export function compareTiming(baseline, candidate, budgets) {
  assert.equal(baseline.length, budgets.samples, 'Missing baseline samples');
  assert.equal(candidate.length, budgets.samples, 'Missing candidate samples');
  const reference = median(baseline);
  const actual = median(candidate);
  const limit =
    reference +
    Math.max(reference * budgets.relativeMedianAllowance, budgets.absoluteMedianAllowanceMs);
  return {
    baselineMedianMs: reference,
    candidateMedianMs: actual,
    limitMs: limit,
    passesProposedBudget: actual <= limit,
  };
}

export async function inventory(directory) {
  const files = (await readdir(directory, { recursive: true, withFileTypes: true }))
    .filter((item) => item.isFile())
    .map((item) => resolve(item.parentPath, item.name))
    .sort();
  assert.ok(files.length, 'An empty build cannot be measured');
  return Promise.all(
    files.map(async (file) => {
      const bytes = await readFile(file);
      return {
        path: file.slice(resolve(directory).length + 1).replaceAll('\\', '/'),
        bytes: bytes.length,
        sha256: sha256(bytes),
      };
    }),
  );
}

export async function verifyBuild(directory, provenance, expectedSource) {
  assert.match(expectedSource, /^[a-f0-9]{40}$/);
  assert.equal(
    provenance.sourceSha,
    expectedSource,
    'Build source does not match requested source',
  );
  const assets = await inventory(directory);
  assert.deepEqual(assets, provenance.assets, 'Production assets do not match build-job manifest');
  const html = await readFile(resolve(directory, 'index.html'), 'utf8');
  const paths = [
    ...html.matchAll(/<(?:script|link)\b[^>]*(?:src|href)=["']([^"']+\.(?:js|css))["']/g),
  ].map((match) => match[1].replace(/^\/?(?:\.\/)?/, ''));
  assert.ok(paths.length >= 2, 'Expected emitted entry JS and CSS');
  const entries = await Promise.all(
    [...new Set(paths)].map(async (path) => {
      assert.ok(
        assets.some((asset) => asset.path === path),
        `Unknown entry asset ${path}`,
      );
      const bytes = await readFile(resolve(directory, path));
      return { path, gzipBytes: gzipSync(bytes, { level: 9 }).length };
    }),
  );
  return {
    sourceSha: expectedSource,
    assets,
    entries,
    entryGzipBytes: entries.reduce((sum, entry) => sum + entry.gzipBytes, 0),
  };
}

export function validateSample(record) {
  const finite = (value, label) =>
    assert.ok(Number.isFinite(value) && value >= 0, `Missing or invalid ${label}`);
  const observation = record.observations;
  finite(observation?.observedAtMs, 'observation endpoint');
  assert.ok(Array.isArray(observation?.paint?.shifts), 'Missing layout observations');
  const measurement = (entries, name) => {
    assert.equal(entries?.length, 1, `Expected one ${name} measure`);
    finite(entries[0].startTime, `${name} start`);
    finite(entries[0].duration, `${name} duration`);
    assert.ok(
      entries[0].startTime + entries[0].duration <= observation.observedAtMs,
      `${name} ends beyond observation`,
    );
  };
  if (record.kind === 'editor') {
    finite(record.editor?.duration, 'editor duration');
    assert.equal(record.editor.readOnly, true);
    assert.equal(record.editor.workerResponseStatus, 200);
    assert.match(record.editor.expectedModelSha256, /^[a-f0-9]{64}$/);
    assert.equal(record.editor.modelSha256, record.editor.expectedModelSha256);
    assert.equal(record.editor.workerModelSha256, record.editor.expectedModelSha256);
    return;
  }
  const ready = observation.readiness;
  finite(ready?.contentReadyMs, 'content readiness');
  finite(ready?.fontsReadyMs, 'font readiness');
  assert.ok(
    ready.contentReadyMs <= ready.fontsReadyMs && ready.fontsReadyMs <= observation.observedAtMs,
    'Invalid readiness order',
  );
  assert.ok(record.readiness?.readiness?.apis?.length, 'Missing API readiness');
  assert.ok(
    record.readiness.readiness.apis.every((api) => api.status === 200),
    'Failed API readiness',
  );
  if (record.kind === 'filter') {
    finite(record.filterStart, 'filter input event');
    measurement(
      record.filter?.measures?.filter((entry) => entry.name === 'filter-results'),
      'filter',
    );
    assert.equal(record.filter?.after?.rows?.length, 1, 'Missing filtered result');
    assert.equal(record.filter.after.value, 'xgboost');
  }
  if (record.kind === 'navigation') {
    measurement(record.runOpen?.measure, 'run opening');
    measurement(record.taskOpen?.measure, 'task opening');
    assert.equal(record.runOpen?.taskNodeVisible, true);
    assert.equal(record.taskOpen?.taskDetailsVisible, true);
  }
}

// Cleanup must preserve earlier evidence even when browser/process teardown stalls.
export async function cleanupWithEvidence(steps, onError, save, timeoutMs = 10000) {
  const errors = [];
  for (const [stage, operation] of steps) {
    let timer;
    try {
      await Promise.race([
        Promise.resolve().then(operation),
        new Promise((_, reject) => {
          timer = setTimeout(() => reject(new Error(`${stage} cleanup timed out`)), timeoutMs);
        }),
      ]);
    } catch (error) {
      const failure = { stage, error: error.stack || String(error) };
      errors.push(failure);
      onError(failure);
    } finally {
      clearTimeout(timer);
    }
  }
  await save();
  return errors;
}

export async function recordScalingTrial(progress, path, trial, operation) {
  const attempt = { trial, status: 'running', startedAt: new Date().toISOString() };
  progress.attempts.push(attempt);
  const save = () => writeFile(path, `${JSON.stringify(progress, null, 2)}\n`);
  await save();
  try {
    await operation();
    attempt.status = 'measured';
    if (trial === progress.expectedSamples) progress.status = 'measured';
  } catch (error) {
    attempt.status = 'failed';
    attempt.error = error.stack || String(error);
    progress.status = 'failed';
    throw error;
  } finally {
    attempt.finishedAt = new Date().toISOString();
    await save();
  }
}

// Preserve the historical protocol; adapt only its readiness capture for hosted sampling.
export function hostedReadinessProtocol(original) {
  const protocol = structuredClone(original);
  {
    const previous =
      '  window.__kfpContentReady.contentReadyMs=performance.now();\n  window.__kfpContentReady.readiness=read();\n  mutations.disconnect();resources.disconnect();\n  document.fonts.ready.then(()=>requestAnimationFrame(()=>requestAnimationFrame(()=>{\n   window.__kfpContentReady.fontsReadyMs=performance.now();\n  })));';
    const next =
      '  window.__kfpContentReady.firstContentReadyMs ??= performance.now();\n  document.fonts.ready.then(()=>requestAnimationFrame(()=>requestAnimationFrame(()=>{\n   const state=read();\n   if(!isReady(state)){\n    (window.__kfpContentReady.transientReadiness ??= []).push({atMs:performance.now(),state});\n    pending=false;check();return;\n   }\n   window.__kfpContentReady.contentReadyMs=performance.now();\n   window.__kfpContentReady.fontsReadyMs=window.__kfpContentReady.contentReadyMs;\n   window.__kfpContentReady.readiness=state;\n  })));';
    assert.ok(
      protocol.scripts.preNavigation.includes(previous),
      'Historical readiness protocol changed; review the hosted adapter',
    );
    protocol.scripts.preNavigation = protocol.scripts.preNavigation.replace(previous, next);
    const checkStart = 'function check(){\n if(pending||!document.body||!isReady(read()))return;';
    assert.ok(protocol.scripts.preNavigation.includes(checkStart));
    protocol.scripts.preNavigation = protocol.scripts.preNavigation.replace(
      checkStart,
      'window.__kfpStopReadiness=()=>{mutations.disconnect();resources.disconnect();};\nfunction check(){\n if(Number.isFinite(window.__kfpContentReady.contentReadyMs)){\n  const state=read();if(!isReady(state))window.__kfpContentReady.postConfirmationLoss ??= {atMs:performance.now(),state};\n  return;\n }\n if(pending||!document.body||!isReady(read()))return;',
    );
  }
  {
    const previous =
      " await new Promise((resolve,reject)=>{const begin=performance.now();const poll=()=>{const state=read();if(isReady(state))return resolve();if(performance.now()-begin>30000)return reject(new Error('Readiness not reached: '+JSON.stringify(state)));setTimeout(poll,50);};poll();});\n await document.fonts.ready;await new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)));\n const result=read();if(!isReady(result))throw new Error('Readiness changed');";
    const next =
      " const result=await new Promise((resolve,reject)=>{const begin=performance.now();const poll=()=>{const state=read();const marks=window.__kfpContentReady;if(marks?.postConfirmationLoss||(Number.isFinite(marks?.contentReadyMs)&&!isReady(state)))return reject(new Error('Readiness lost after confirmation: '+JSON.stringify({state,marks})));if(Number.isFinite(marks?.contentReadyMs)&&Number.isFinite(marks?.fontsReadyMs)&&isReady(state)){window.__kfpStopReadiness();return resolve(state);}if(performance.now()-begin>30000)return reject(new Error('Confirmed readiness not reached: '+JSON.stringify({state,marks:window.__kfpContentReady})));setTimeout(poll,50);};poll();});";
    assert.ok(
      protocol.scripts.perfReadinessFunction.includes(previous),
      'Historical readiness protocol changed; review the hosted adapter',
    );
    protocol.scripts.perfReadinessFunction = protocol.scripts.perfReadinessFunction.replace(
      previous,
      next,
    );
  }
  protocol.hostedReadiness =
    'Content readiness requires the unchanged route/API predicate, fonts and two confirming frames. Transient first readiness is retained separately; post-confirmation loss fails the trial even after recovery. Final capture has a 30-second deadline.';
  return protocol;
}
