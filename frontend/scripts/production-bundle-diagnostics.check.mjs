/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */
import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';
import { startupDiagnostics } from './production-bundle-diagnostics.mjs';
test('startup cancellation preserves the failing stage through cleanup and late completion', () => {
  const dir = mkdtempSync(join(tmpdir(), 'kfp-startup-'));
  try {
    const recorder = startupDiagnostics(dir, { sourceSha: 'source', timeoutMs: 30000 });
    recorder.stage('navigation');
    recorder.pageError('load failed');
    recorder.fail('test deadline');
    recorder.stage('browser-close');
    recorder.fail('secondary cleanup failure');
    recorder.pass();
    const saved = JSON.parse(readFileSync(join(dir, 'production-startup.json')));
    assert.equal(saved.status, 'failed');
    assert.equal(saved.failure.stage, 'navigation');
    assert.equal(saved.failure.error, 'test deadline');
    assert.deepEqual(saved.pageErrors, ['load failed']);
    assert.deepEqual(
      saved.events.map((e) => e.stage),
      ['navigation', 'browser-close'],
    );
  } finally {
    rmSync(dir, { recursive: true });
  }
});
