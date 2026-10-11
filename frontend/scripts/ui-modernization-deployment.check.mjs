/* Copyright 2026 The Kubeflow Authors. Licensed under the Apache License, Version 2.0. */
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import test from 'node:test';
import {
  assetName,
  experimentRunDependency,
  namespaceOptionPattern,
  selectNamespace,
  selectedNamespaceSelector,
  safePageUrl,
  loginButtonName,
  openRunGraph,
  assertLoginRequired,
  summarizeAvailability,
  requireHosted,
  sanitize,
  stableRun,
  stableSchedule,
  verifyAsset,
} from './ui-modernization-deployment.mjs';

test('hosted guard rejects workstations and self-hosted runners', () => {
  assert.throws(() => requireHosted({}));
  assert.throws(() =>
    requireHosted({
      CI: 'true',
      GITHUB_ACTIONS: 'true',
      RUNNER_ENVIRONMENT: 'self-hosted',
      RUNNER_TEMP: '/tmp',
    }),
  );
  requireHosted({
    CI: 'true',
    GITHUB_ACTIONS: 'true',
    RUNNER_ENVIRONMENT: 'github-hosted',
    RUNNER_TEMP: '/tmp',
  });
});

test('asset scope covers old and new production bundles without conflating Dashboard', () => {
  const base = 'http://127.0.0.1:3000';
  assert.equal(assetName(base + '/pipeline/assets/index-a.js', base), 'assets/index-a.js');
  assert.equal(assetName(base + '/static/js/main-a.js', base), 'static/js/main-a.js');
  assert.equal(assetName(base + '/dashboard_lib.bundle.js', base), null);
  assert.equal(assetName('https://example.org/assets/index.js', base), null);
});

test('asset verification rejects mixed generations and wrong bytes', () => {
  const manifest = { 'assets/index.js': createHash('sha256').update('baseline').digest('hex') };
  verifyAsset('assets/index.js', Buffer.from('baseline'), manifest);
  assert.throws(() => verifyAsset('assets/index.js', Buffer.from('candidate'), manifest));
  assert.throws(() => verifyAsset('assets/later.js', Buffer.from('baseline'), manifest));
});

test('credential and OAuth redirect diagnostics are sanitized before artifact writes', () => {
  const diagnostic =
    'fill("ephemeral-password") failed at http://127.0.0.1:3000/oauth2/callback?code=abc&state=xyz token=private-token';
  const result = sanitize(diagnostic, ['ephemeral-password']);
  for (const secret of ['ephemeral-password', 'abc', 'xyz', 'private-token'])
    assert.ok(!result.includes(secret));
  assert.ok(result.includes('/oauth2/callback'));
});

test('run preservation compares actual identity, configuration and terminal state', () => {
  const run = {
    run_id: 'a',
    experiment_id: 'b',
    display_name: 'run',
    state: 'SUCCEEDED',
    runtime_config: { parameters: { value: 'original' } },
  };
  assert.deepEqual(stableRun({ ...run, last_refreshed: 'later' }), stableRun(run));
  assert.notDeepEqual(stableRun({ ...run, state: 'FAILED' }), stableRun(run));
  assert.notDeepEqual(
    stableRun({ ...run, runtime_config: { parameters: { value: 'changed' } } }),
    stableRun(run),
  );
  assert.throws(() => stableRun({ display_name: 'missing identity' }));
});

test('schedule preservation includes trigger and enabled status', () => {
  const schedule = {
    recurring_run_id: 'a',
    experiment_id: 'b',
    display_name: 'schedule',
    status: 'ENABLED',
    trigger: { periodic_schedule: { interval_second: '3600' } },
  };
  assert.notDeepEqual(
    stableSchedule({ ...schedule, status: 'DISABLED' }),
    stableSchedule(schedule),
  );
  assert.notDeepEqual(stableSchedule({ ...schedule, trigger: {} }), stableSchedule(schedule));
  assert.throws(() => stableSchedule({ display_name: 'missing identity' }));
});

test('unauthenticated access accepts only denial or a same-origin real login redirect', () => {
  const base = 'http://127.0.0.1:3000';
  assertLoginRequired(401, undefined, base);
  assertLoginRequired(403, undefined, base);
  assertLoginRequired(302, '/oauth2/start', base);
  assert.throws(() => assertLoginRequired(200, undefined, base));
  assert.throws(() => assertLoginRequired(302, '/pipeline/', base));
  assert.throws(() => assertLoginRequired(302, 'https://example.org/oauth2/start', base));
});

test('availability evidence distinguishes one failure from no observed interruption', () => {
  const good = { offsetMs: 100, durationMs: 10, ready: true };
  const noFailure = summarizeAvailability([good]);
  assert.equal(noFailure.observedUnavailable, false);
  assert.equal(noFailure.recoveryUpperBoundMs, null);
  const failed = summarizeAvailability([
    good,
    { offsetMs: 300, durationMs: 20, ready: false },
    { offsetMs: 600, durationMs: 10, ready: true },
  ]);
  assert.equal(failed.observedUnavailable, true);
  assert.equal(failed.unavailableSamples, 1);
  assert.equal(failed.recoveryUpperBoundMs, 320);
  assert.equal(failed.observedUnavailableSpanMs, null);
  assert.throws(() => summarizeAvailability([]));
});

test('signed TensorBoard path tokens are removed even from nested asset errors', () => {
  const value = sanitize(
    'failed http://127.0.0.1:3000/pipeline/apps/tensorboard/proxy/signed-token/plugin.js',
  );
  assert.ok(!value.includes('signed-token'));
});

test('relative proxy paths and extracted bare tokens are redacted', () => {
  assert.ok(
    !sanitize('failed /apps/tensorboard/proxy/relative-token/data').includes('relative-token'),
  );
  assert.ok(!sanitize('token bare-token was rejected', ['bare-token']).includes('bare-token'));
});

test('normal Dex password submission recognizes its shipped Login label', () => {
  for (const label of ['Login', 'Log in', 'Sign in']) assert.match(label, loginButtonName);
  assert.doesNotMatch('Sign in with Dex', loginButtonName);
});

test('run graph inspection leaves a remembered Detail tab before inspecting nodes', async () => {
  let selected = 'Detail';
  let navigated = false;
  const target = {
    goto: async (url) => {
      assert.equal(url, 'http://127.0.0.1:3000/#/runs/details/existing');
      navigated = true;
      // Same-route navigation intentionally preserves the current tab.
    },
    locator: (selector) => {
      assert.equal(selector, 'button');
      return {
        filter: ({ hasText }) => ({
          click: async () => {
            assert.ok(navigated);
            assert.match('Graph', hasText);
            assert.doesNotMatch('Detail', hasText);
            selected = 'Graph';
          },
        }),
      };
    },
  };
  await openRunGraph(target, 'http://127.0.0.1:3000/#/runs/details/existing');
  assert.equal(selected, 'Graph');
});

test('login diagnostic URLs retain only a sanitized origin and path', () => {
  assert.equal(
    safePageUrl('http://127.0.0.1:3000/oauth2/callback?code=private#state=secret'),
    'http://127.0.0.1:3000/oauth2/callback',
  );
  assert.equal(
    safePageUrl('http://127.0.0.1:3000/apps/tensorboard/proxy/private-token/foo?secret=other'),
    'http://127.0.0.1:3000/apps/tensorboard/proxy/[redacted]/foo',
  );
  assert.equal(safePageUrl('not a URL'), '[unavailable URL]');
});

test('Polymer namespace options match exact visible text without a name attribute', () => {
  const pattern = namespaceOptionPattern('kfp-qualification-second');
  assert.match('  kfp-qualification-second\n', pattern);
  assert.doesNotMatch('kfp-qualification-second-extra', pattern);
  assert.doesNotMatch('other-kfp-qualification-second', pattern);
  assert.doesNotMatch('kfp-qualification-other', pattern);
  assert.match('team.example', namespaceOptionPattern('team.example'));
  assert.doesNotMatch('team-example', namespaceOptionPattern('team.example'));
});

test('namespace race waits only for real old-experiment run response dependencies', () => {
  const ids = new Set(['old-experiment']);
  const base = 'http://127.0.0.1:3000/pipeline/apis/v2beta1/';
  assert.equal(
    experimentRunDependency(base + 'runs?experiment_id=old-experiment&page_size=5', ids),
    'old-experiment',
  );
  assert.equal(experimentRunDependency(base + 'runs?experiment_id=new-experiment', ids), null);
  assert.equal(experimentRunDependency(base + 'runs?namespace=old-experiment', ids), null);
  assert.equal(
    experimentRunDependency(base + 'runs/run-id?experiment_id=old-experiment', ids),
    null,
  );
  assert.equal(
    experimentRunDependency(base + 'experiments?experiment_id=old-experiment', ids),
    null,
  );
});

test('namespace selection leaves the automatically selected sole namespace alone', async () => {
  const target = {
    locator(selector) {
      assert.equal(selector, selectedNamespaceSelector);
      return { innerText: async () => '  kfp-qualification-other\n' };
    },
  };
  await selectNamespace(target, 'kfp-qualification-other');
});

test('namespace selection switches a prefix match and verifies the exact destination', async () => {
  let selected = 'team-extra';
  const actions = [];
  const target = {
    locator(selector) {
      if (selector === selectedNamespaceSelector) return { innerText: async () => selected };
      if (selector.endsWith('#dropdown-trigger'))
        return { click: async () => actions.push('open') };
      assert.equal(selector, 'namespace-selector paper-item');
      return {
        filter({ hasText }) {
          assert.match('team', hasText);
          assert.doesNotMatch(selected, hasText);
          return {
            click: async () => {
              actions.push('select');
              selected = 'team';
            },
          };
        },
      };
    },
  };
  await selectNamespace(target, 'team');
  assert.deepEqual(actions, ['open', 'select']);
  assert.equal(selected, 'team');
});
