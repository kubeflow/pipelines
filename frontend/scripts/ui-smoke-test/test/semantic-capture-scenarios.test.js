'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const capture = require('../capture-screenshots.js');
const {
  SCENARIO_CONTRACT_SCHEMA_VERSION,
  SEMANTIC_SCENARIOS,
  getSemanticScenarioCatalog,
  resolveSemanticScenarios,
} = require('../semantic-capture-scenarios.js');

const SEED_VALUES = Object.freeze({
  artifactId: 'artifact-1',
  compareRunlist: 'run-1,run-2,run-3',
  consumeMetricsTaskId: 'consume-1',
  htmlArtifactId: 'html-1',
  markdownArtifactId: 'markdown-1',
  nestedDagTaskId: 'nested-1',
  parallelTaskId: 'parallel-1',
  relatedArtifactId: 'accuracy-1',
  retryTaskId: 'retry-1',
  richRunId: 'run-1',
  rocArtifactId: 'roc-1',
  scalarArtifactId: 'scalar-1',
  writeMetricsTaskId: 'write-1',
});

function byKey(scenarios, key) {
  const scenario = scenarios.find((candidate) => candidate.semanticScenario === key);
  assert.ok(scenario, `missing scenario ${key}`);
  return scenario;
}

test('semantic scenario contract has unique paired base and head definitions', () => {
  const keys = SEMANTIC_SCENARIOS.map((scenario) => scenario.key);
  assert.equal(new Set(keys).size, keys.length);
  assert.ok(keys.length >= 15);

  for (const scenario of SEMANTIC_SCENARIOS) {
    assert.match(scenario.key, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
    assert.ok(scenario.title);
    assert.ok(scenario.revisions.base?.path);
    assert.ok(scenario.revisions.head?.path);
    assert.ok(scenario.revisions.base?.routeExpectation);
    assert.ok(scenario.revisions.head?.routeExpectation);
  }
  const catalog = getSemanticScenarioCatalog();
  assert.deepEqual(
    catalog.map((entry) => entry.semanticScenario),
    keys,
  );
  assert.doesNotThrow(() => JSON.stringify(catalog));
});

test('scenario resolution binds canonical pair keys to revision-specific journeys', () => {
  const base = resolveSemanticScenarios('base', SEED_VALUES);
  const head = resolveSemanticScenarios('head', SEED_VALUES);

  assert.deepEqual(
    base.map((scenario) => scenario.semanticScenario),
    head.map((scenario) => scenario.semanticScenario),
  );
  assert.equal(
    base.every(
      (scenario) => scenario.scenarioContractSchemaVersion === SCENARIO_CONTRACT_SCHEMA_VERSION,
    ),
    true,
  );

  const baseExecutions = byKey(base, 'executions-to-runs');
  const headExecutions = byKey(head, 'executions-to-runs');
  assert.equal(baseExecutions.routeExpectation.kind, 'direct');
  assert.equal(baseExecutions.routeExpectation.path, '/executions');
  assert.equal(headExecutions.path, '/#/executions');
  assert.deepEqual(headExecutions.routeExpectation, {
    kind: 'expected-removal',
    path: '/runs',
  });
  assert.ok(headExecutions.actions.some((action) => action.type === 'assertAbsent'));

  const baseTask = byKey(base, 'run-details-task-panel');
  const headTask = byKey(head, 'run-details-task-panel');
  assert.equal(baseTask.path, '/#/runs/details/run-1');
  assert.equal(headTask.path, '/#/runs/details/run-1?task=write-1');
  assert.match(
    baseTask.actions.find((action) => action.type === 'click').selector,
    /write-metrics/,
  );
  assert.ok(headTask.actions.some((action) => action.selector?.includes('Input/Output')));

  for (const graph of [
    byKey(base, 'run-details-rich-graph'),
    byKey(head, 'run-details-rich-graph'),
  ]) {
    assert.ok(graph.actions.some((action) => action.selector?.includes('execution-icon-active')));
    assert.ok(graph.actions.some((action) => action.selector?.includes('artifact-icon-live')));
  }
});

test('base and head use graph Artifact visualization journeys for V2 seeded runs', () => {
  for (const role of ['base', 'head']) {
    const resolved = resolveSemanticScenarios(role, SEED_VALUES);
    const expectations = new Map([
      ['run-details-scalar-metrics', 'scalar_metrics'],
      ['run-details-html', 'html_report'],
      ['run-details-markdown', 'markdown_report'],
      ['run-details-roc', 'roc_curve'],
    ]);
    for (const [scenarioKey, artifactKey] of expectations) {
      const scenario = byKey(resolved, scenarioKey);
      assert.equal(scenario.path, '/#/runs/details/run-1');
      assert.ok(
        scenario.actions.some(
          (action) => action.type === 'click' && action.selector.includes(artifactKey),
        ),
        `${role} ${scenarioKey} must select ${artifactKey}`,
      );
      assert.ok(
        scenario.actions.some(
          (action) =>
            action.type === 'waitForSelector' && action.selector.includes('artifact-icon-live'),
        ),
        `${role} ${scenarioKey} must wait for hydrated Artifact data`,
      );
      assert.ok(
        scenario.actions.some(
          (action) => action.type === 'click' && action.selector.includes('Visualization'),
        ),
        `${role} ${scenarioKey} must open Visualization`,
      );
    }
  }
});

test('HTML and Markdown scenarios positively wait for deterministic rendered fixture text', () => {
  for (const role of ['base', 'head']) {
    const resolved = resolveSemanticScenarios(role, SEED_VALUES);
    for (const key of ['run-details-html', 'compare-html']) {
      const readiness = byKey(resolved, key).actions.find(
        (action) => action.type === 'waitForFrameText' && action.text === 'UI Smoke HTML Report',
      );
      assert.ok(readiness);
      if (key === 'compare-html') assert.equal(readiness.minCount, 2);
    }
    for (const key of ['run-details-markdown', 'compare-markdown']) {
      const readiness = byKey(resolved, key).actions.find(
        (action) => action.type === 'waitForText' && action.text === 'UI Smoke Markdown Report',
      );
      assert.ok(readiness);
      if (key === 'compare-markdown') assert.equal(readiness.minCount, 2);
    }
  }

  const baseHtml = byKey(resolveSemanticScenarios('base', SEED_VALUES), 'compare-html');
  assert.ok(baseHtml.actions.some((action) => action.type === 'hover'));
  assert.deepEqual(
    baseHtml.actions.filter((action) => action.type === 'hover').map((action) => action.index),
    [0, 1],
  );
  assert.ok(
    baseHtml.actions.some(
      (action) => action.type === 'click' && action.selector.includes('second HTML artifact'),
    ),
  );
  assert.ok(
    baseHtml.actions.some(
      (action) => action.type === 'click' && action.selector.includes('dropdownSubmenu'),
    ),
  );
  const headHtml = byKey(resolveSemanticScenarios('head', SEED_VALUES), 'compare-html');
  assert.ok(
    headHtml.actions.some(
      (action) => action.type === 'click' && action.selector.includes('First comparison artifact'),
    ),
  );
  assert.ok(
    headHtml.actions.some(
      (action) => action.type === 'click' && action.selector.includes('Second comparison artifact'),
    ),
  );
  assert.deepEqual(
    headHtml.actions
      .filter((action) => action.selector?.includes('[role="option"]'))
      .map((action) => action.index),
    [0, 1],
  );
});

test('Artifact Details binds HTML while relationships bind a produced and consumed metric', () => {
  for (const role of ['base', 'head']) {
    const resolved = resolveSemanticScenarios(role, SEED_VALUES);
    const details = byKey(resolved, 'artifact-details');
    assert.match(details.path, /\/artifacts\/html-1/);
    assert.deepEqual(details.missingFixtures, []);
    assert.ok(
      details.actions.some(
        (action) => action.type === 'waitForText' && action.text === 'html_report',
      ),
    );

    const relationships = byKey(resolved, 'artifact-related-tasks');
    assert.match(relationships.path, /\/artifacts\/accuracy-1/);
    assert.deepEqual(relationships.missingFixtures, []);
    const expectedText =
      role === 'base'
        ? ['write-metrics', 'consume-metrics']
        : [
            'Produced as scalar_metrics',
            'Consumed as metrics',
            'Run run-1',
            'Task write-1',
            'Task consume-1',
          ];
    for (const text of expectedText) {
      assert.ok(
        relationships.actions.some(
          (action) => action.type === 'waitForText' && action.text === text,
        ),
        `${role} relationships must identify ${text}`,
      );
    }
  }

  const missingHtml = resolveSemanticScenarios('base', {
    ...SEED_VALUES,
    artifactId: 'scalar-default',
    htmlArtifactId: null,
  });
  assert.deepEqual(byKey(missingHtml, 'artifact-details').missingFixtures, ['htmlArtifactId']);
});

test('log scenario requires deterministic successful-attempt output on both revisions', () => {
  for (const role of ['base', 'head']) {
    const logs = byKey(resolveSemanticScenarios(role, SEED_VALUES), 'run-details-task-logs');
    assert.ok(
      logs.actions.some(
        (action) => action.type === 'waitForText' && action.text === 'retry completed',
      ),
    );
    assert.equal(
      logs.actions.some((action) => action.selector?.includes('[role="alert"]')),
      false,
    );
  }
});

test('revision-aware capture retains non-overlapping generic coverage', () => {
  const pages = capture.buildRevisionAwarePages('base', SEED_VALUES);
  const names = new Set(pages.map((page) => page.name));
  for (const retained of [
    'pipelines',
    'pipeline-details-seeded',
    'pipeline-details-seeded-sidepanel',
    'experiments',
    'runs',
    'runs-new',
    'runs-new-pipeline-dialog',
    'runs-new-upload-dialog',
    'recurring-runs',
    'pipeline-create',
    'experiment-create',
  ]) {
    assert.equal(names.has(retained), true, `${retained} should be retained`);
  }
  for (const superseded of [
    'executions',
    'artifacts',
    'artifact-lineage-from-list',
    'run-details-seeded',
    'run-details-seeded-sidepanel',
    'compare-seeded',
    'compare-seeded-roc',
  ]) {
    assert.equal(names.has(superseded), false, `${superseded} should be superseded`);
  }
  assert.equal(names.size, pages.length);
  assert.deepEqual(
    capture.revisionAwarePageNames(['executions', 'run-details-seeded-sidepanel', 'pipelines']),
    ['executions-to-runs', 'run-details-task-panel', 'pipelines'],
  );
});

test('legacy lineage readiness is data-specific and clean mode skips upgrade-only history', () => {
  const base = resolveSemanticScenarios('base', SEED_VALUES);
  const related = byKey(base, 'artifact-related-tasks');
  assert.ok(related.actions.some((action) => action.type === 'waitForFunction'));
  assert.equal(
    related.actions.some((action) => action.selector === 'svg'),
    false,
  );

  const historical = byKey(base, 'historical-artifact-evolution');
  assert.equal(historical.required, false);
  assert.deepEqual(historical.missingFixtures, ['historicalArtifactId']);
  assert.match(historical.expectedChange, /upgrade mode/);
});

test('seed binding extraction supplies revision-specific run, task, and Artifact IDs', (t) => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-scenarios-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const manifestPath = path.join(directory, 'seed.json');
  fs.writeFileSync(
    manifestPath,
    JSON.stringify({
      defaults: {
        artifactId: 'scalar-legacy-default',
        compareRunlist: 'rich,metrics',
        historicalArtifactId: 'history-1',
      },
      resources: { runIds: ['fallback'] },
      semantic: {
        bindings: {
          resources: {
            'run.evaluation': { id: 'evaluation' },
            'run.training-1': { id: 'rich' },
            'run.training-2': { id: 'second' },
          },
          runs: {
            'run.training-1': {
              artifacts: {
                'artifact.html-report': { artifactIds: ['html-1'] },
                'artifact.markdown-report': { artifactIds: ['markdown-1'] },
                'artifact.roc-curve': { artifactIds: ['roc-1'] },
                'artifact.scalar-metrics': {
                  artifactIds: ['scalar-1'],
                  members: { 'metric.accuracy': { artifactIds: ['accuracy-1'] } },
                },
              },
              runId: 'rich',
              taskInstances: {
                'task.consume-metrics': [{ taskId: 'consume-1' }],
                'task.nested-dag': [{ taskId: 'nested-1' }],
                'task.parallel-loop': [{ taskId: 'parallel-1' }],
                'task.retry-once': [{ taskId: 'retry-1' }],
                'task.write-metrics': [{ mlmdExecutionId: 'execution-1', taskId: 'write-1' }],
              },
            },
          },
        },
      },
    }),
  );

  assert.deepEqual(capture.loadSeedValues(manifestPath), {
    artifactId: 'scalar-legacy-default',
    compareRunlist: 'rich,second,evaluation',
    consumeMetricsTaskId: 'consume-1',
    executionId: 'execution-1',
    experimentId: undefined,
    historicalArtifactId: 'history-1',
    htmlArtifactId: 'html-1',
    markdownArtifactId: 'markdown-1',
    nestedDagTaskId: 'nested-1',
    parallelTaskId: 'parallel-1',
    pipelineId: undefined,
    recurringRunId: undefined,
    relatedArtifactId: 'accuracy-1',
    retryTaskId: 'retry-1',
    richRunId: 'rich',
    rocArtifactId: 'roc-1',
    runId: 'fallback',
    scalarArtifactId: 'scalar-1',
    taskId: 'write-1',
    writeMetricsTaskId: 'write-1',
  });
});

test('route matching asserts redirects without requiring generated query parameters to match order', () => {
  assert.equal(
    capture.routeFromUrl('https://example.test/kfp/#/runs/details/run-1?task=task-1'),
    '/runs/details/run-1?task=task-1',
  );
  assert.equal(
    capture.routeMatches('/compare?other=1&runlist=a%2Cb', '/compare?runlist=a%2Cb'),
    true,
  );
  assert.equal(capture.routeMatches('/executions', '/runs'), false);
});

test('capture requires its declared deterministic font instead of silently accepting fallback', async () => {
  let requestedFont;
  const available = await capture.assertDeterministicFont({
    evaluate: async (_predicate, fontFamily) => {
      requestedFont = fontFamily;
      return { available: true, computedBodyFont: '"UI Smoke Roboto", sans-serif', reason: null };
    },
  });
  assert.equal(requestedFont, 'UI Smoke Roboto');
  assert.equal(available.computedBodyFont, '"UI Smoke Roboto", sans-serif');
  assert.deepEqual(
    capture.DETERMINISTIC_FONT_ASSETS.map(({ sha256, weight }) => ({ sha256, weight })),
    [
      {
        sha256: '425c0713a8176f92273d378599c7eac57de7fafabd4bd0ed457b70eb8f80d371',
        weight: 400,
      },
      {
        sha256: '5bcc3aa180e7f26f643cd5b2621cd7c2de193d0661d913a94afd3d4881a7a34b',
        weight: 500,
      },
      {
        sha256: 'b9d66d1708156f765ada51939bc24ed259dafa69eb631b36e443680fe9e15879',
        weight: 700,
      },
    ],
  );

  await assert.rejects(
    capture.assertDeterministicFont({
      evaluate: async () => ({
        available: false,
        reason: 'UI Smoke Roboto did not load from the pinned capture asset',
      }),
    }),
    /Deterministic capture font check failed.*UI Smoke Roboto did not load/,
  );
});

test('frame-text readiness inspects sandboxed frames through Playwright', async () => {
  let frameChecks = 0;
  await capture.executeActions(
    {
      frames: () => [
        {
          getByText: () => ({
            count: async () => {
              frameChecks++;
              return 0;
            },
          }),
        },
        {
          getByText: (text) => ({
            count: async () => (text === 'UI Smoke HTML Report' ? 1 : 0),
          }),
        },
      ],
      waitForTimeout: async () => {},
    },
    [{ type: 'waitForFrameText', text: 'UI Smoke HTML Report', timeoutMs: 50 }],
  );
  assert.equal(frameChecks, 1);
});

test('capture actions select deterministic indexes and wait for repeated text', async () => {
  const events = [];
  const indexedTarget = (selector, index) => ({
    click: async () => events.push(`click:${selector}:${index}`),
    hover: async () => events.push(`hover:${selector}:${index}`),
  });
  await capture.executeActions(
    {
      getByText: (text) => ({
        nth: (index) => ({ waitFor: async () => events.push(`text:${text}:${index}`) }),
      }),
      locator: (selector) => ({
        first: () => indexedTarget(selector, 0),
        nth: (index) => indexedTarget(selector, index),
      }),
    },
    [
      { type: 'click', selector: '.option', index: 1 },
      { type: 'hover', selector: '.menu', index: 2 },
      { type: 'waitForText', text: 'Report', minCount: 2 },
    ],
  );

  assert.deepEqual(events, ['click:.option:1', 'hover:.menu:2', 'text:Report:1']);
});

test('child frame stabilization pins iframe typography and normalizes its text', async () => {
  const events = [];
  const mainFrame = {};
  const childFrame = {
    addStyleTag: async ({ content }) =>
      events.push(content.includes('font-family: "UI Smoke Roboto"')),
    evaluate: async (predicate) => {
      if (String(predicate).includes('document.fonts.check')) {
        return { available: true, computedBodyFont: 'UI Smoke Roboto', reason: null };
      }
      if (String(predicate).includes('document.fonts')) return undefined;
      events.push('normalized');
      return undefined;
    },
  };
  const statuses = await capture.stabilizeChildFrames({
    frames: () => [mainFrame, childFrame],
    mainFrame: () => mainFrame,
  });
  assert.deepEqual(events, [true, 'normalized']);
  assert.equal(statuses.length, 1);
  assert.equal(statuses[0].available, true);
});

test('diagnostics are bounded and redact URL query values', () => {
  const handlers = new Map();
  const diagnostics = capture.createPageDiagnostics(
    { on: (event, handler) => handlers.set(event, handler) },
    'https://example.test/kfp',
    1,
  );
  const consoleMessage = {
    location: () => ({ columnNumber: 2, lineNumber: 3, url: 'https://example.test/app.js' }),
    text: () => 'first\nerror',
    type: () => 'error',
  };
  handlers.get('console')(consoleMessage);
  handlers.get('console')(consoleMessage);
  handlers.get('requestfailed')({
    failure: () => ({ errorText: 'net::ERR_FAILED' }),
    method: () => 'GET',
    url: () => 'https://example.test/apis/runs?token=secret&run=123',
  });

  assert.equal(diagnostics.consoleErrors.length, 1);
  assert.equal(diagnostics.dropped.consoleErrors, 1);
  assert.equal(diagnostics.failedRequests.length, 1);
  assert.equal(diagnostics.failedRequests[0].url, '/apis/runs?run=<redacted>&token=<redacted>');
  assert.equal(JSON.stringify(diagnostics).includes('secret'), false);
  assert.equal(diagnostics.failedRequests[0].sameOrigin, true);
  assert.equal(
    capture.sanitizeDiagnosticUrl(
      'https://example.test/kfp/#/compare?runlist=secret-runs',
      'https://example.test/kfp',
    ),
    '/kfp/#/compare?runlist=<redacted>',
  );
  assert.equal(
    capture.sanitizeDiagnosticText('Authorization failed: Bearer secret-token'),
    'Authorization failed: Bearer <redacted>',
  );
  assert.equal(
    capture.sanitizeDiagnosticText(
      'request failed: authorization=raw-secret password: "hunter2" token=abc123',
    ),
    'request failed: authorization=<redacted> password: <redacted> token=<redacted>',
  );
  assert.equal(
    capture.sanitizeDiagnosticText('connect https://operator:password@example.test/apis'),
    'connect https://<redacted>:<redacted>@example.test/apis',
  );
  assert.equal(
    capture.classifyCaptureFailure(new Error('Action failed: selector'), {
      failedRequests: [{ sameOrigin: false, status: null, url: 'https://cdn.example/font.woff' }],
    }),
    'selector_drift',
  );
  assert.equal(
    capture.classifyCaptureFailure(new Error('Action failed: selector'), {
      failedRequests: [{ sameOrigin: true, status: 500, url: '/apis/v2beta1/runs' }],
    }),
    'api_incompatibility',
  );
});
