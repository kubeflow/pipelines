/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */
import assert from 'node:assert/strict';
import { test } from 'node:test';
import { NativeTransactions } from './ui-modernization-native-transactions.ts';
const url = (path) => new URL(path, 'http://127.0.0.1');
test('native transactional fixture is opt-in and rejects undeclared methods/actions', () => {
  const f = new NativeTransactions();
  assert.equal(f.handle('POST', url('/apis/v2beta1/runs'), {}), null);
  assert.throws(() => f.start('arbitrary'));
  f.start('run');
  for (const [method, path] of [
    ['POST', '/apis/v2beta1/experiments'],
    ['DELETE', '/apis/v2beta1/runs/existing'],
    ['POST', '/arbitrary'],
  ])
    assert.equal(f.handle(method, url(path), {}).status, 405);
  assert.equal(f.unexpected.length, 3);
  assert.equal(f.mutations.length, 0);
});
test('experiment retry fails once, preserves request bodies and creates only one scoped resource', () => {
  const f = new NativeTransactions();
  f.start('experiment');
  const body = { display_name: 'Typed experiment', namespace: 'team-a' };
  assert.equal(f.handle('POST', url('/apis/v2beta1/experiments'), body).status, 503);
  assert.equal(f.experiments.length, 1);
  const created = f.handle('POST', url('/apis/v2beta1/experiments'), body);
  assert.equal(created.status, 200);
  assert.equal(f.experiments.length, 2);
  assert.deepEqual(
    f.mutations.map((r) => r.status),
    [503, 200],
  );
  assert.deepEqual(
    f.handle('GET', url('/apis/v2beta1/experiments/native-created-experiment'), null).body,
    created.body,
  );
  assert.throws(() => f.handle('POST', url('/apis/v2beta1/experiments'), { namespace: 'another' }));
});
test('typed run fixture retains parameters and reads its newly created resource', () => {
  const f = new NativeTransactions();
  f.start('run');
  const body = {
    experiment_id: 'native-experiment',
    runtime_config: {
      parameters: { count: 0, enabled: false, message: '', config: { nested: false } },
    },
  };
  const result = f.handle('POST', url('/apis/v2beta1/runs'), body);
  assert.deepEqual(result.body.runtime_config, body.runtime_config);
  assert.equal(
    f.handle('GET', url('/apis/v2beta1/runs/native-created-run'), null).body.run_id,
    'native-created-run',
  );
  assert.deepEqual(f.handle('GET', url('/apis/v2beta1/runs/native-created-run/tasks'), null).body, {
    tasks: [],
  });
  assert.equal(f.handle('GET', url('/apis/v2beta1/runs/missing'), null).status, 404);
});
test('recurring toggle failure preserves enabled state and retry updates the same schedule', () => {
  const f = new NativeTransactions();
  f.start('recurring');
  f.handle('POST', url('/apis/v2beta1/recurringruns'), { experiment_id: 'native-experiment' });
  const action = url('/apis/v2beta1/recurringruns/native-schedule:disable');
  assert.equal(f.handle('POST', action, null).status, 503);
  assert.equal(f.schedules[0].status, 'ENABLED');
  assert.equal(f.handle('POST', action, null).status, 200);
  assert.equal(f.schedules[0].status, 'DISABLED');
  assert.equal(f.schedules.length, 1);
  assert.deepEqual(
    f.mutations.map((r) => r.status),
    [200, 503, 200],
  );
});

test('invalid payloads record rejection without consuming transient failure or mutating state', () => {
  const f = new NativeTransactions();
  f.start('experiment');
  const endpoint = url('/apis/v2beta1/experiments');
  assert.throws(() => f.handle('POST', endpoint, { namespace: 'another' }));
  assert.equal(f.experiments.length, 1);
  assert.equal(f.mutations[0].status, 400);
  assert.equal(f.handle('POST', endpoint, { namespace: 'team-a' }).status, 503);
  assert.equal(f.experiments.length, 1);
  assert.equal(f.handle('POST', endpoint, { namespace: 'team-a' }).status, 200);
  assert.deepEqual(
    f.mutations.map((entry) => entry.status),
    [400, 503, 200],
  );
  f.start('recurring');
  assert.throws(() => f.handle('POST', url('/apis/v2beta1/recurringruns'), null));
  assert.throws(() =>
    f.handle('POST', url('/apis/v2beta1/recurringruns/native-schedule:disable'), null),
  );
  assert.deepEqual(
    f.mutations.slice(-2).map((entry) => entry.status),
    [400, 400],
  );
  assert.equal(f.schedules.length, 0);
});
