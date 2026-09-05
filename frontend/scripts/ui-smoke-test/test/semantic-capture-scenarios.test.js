'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const vm = require('node:vm');

const capture = require('../capture-screenshots.js');
const comparison = require('../generate-comparison.js');
const { validateCombinedSemanticManifest } = require('../semantic-manifest.js');
const { strictSemanticFixtureManifest } = require('./semantic-fixture.js');
const {
  SCENARIO_CONTRACT_SCHEMA_VERSION,
  SEMANTIC_SCENARIOS,
  getGlobalVisualNormalizationContract,
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

function semanticIdentifierManifest(role, suffix, overrides = {}) {
  const runId = overrides.runId || `${suffix}-run-00000000`;
  const writeTaskId = overrides.writeTaskId || `${suffix}-write-task-0000`;
  const consumeTaskId = overrides.consumeTaskId || `${suffix}-consume-task-00`;
  const htmlArtifactId = overrides.htmlArtifactId || `${suffix}-html-artifact-0`;
  return {
    deployments: {
      [role]: {
        bindings: {
          resources: {
            'run.training-1': {
              displayName: 'UI Smoke Training Run 1',
              id: overrides.resourceRunId || runId,
              kind: 'run',
            },
          },
          runs: {
            'run.training-1': {
              artifacts: {
                'artifact.html-report': { artifactIds: [htmlArtifactId] },
              },
              displayName: 'UI Smoke Training Run 1',
              runId,
              taskInstances: {
                'task.consume-metrics': [{ taskId: consumeTaskId }],
                'task.write-metrics': [
                  { mlmdExecutionId: overrides.executionId || '73', taskId: writeTaskId },
                ],
              },
            },
          },
        },
        revisionFlavor: role === 'base' ? 'legacy-mlmd' : 'native-task-artifact',
        validation: { errors: [], valid: true },
      },
    },
    fixtureSet: 'ui-smoke-deterministic-v3',
    logical: {
      resources: {
        'run.training-1': { displayName: 'UI Smoke Training Run 1' },
      },
    },
    schemaVersion: 'ui-smoke-semantic/v3',
  };
}

function fakeTextPage(textValues, selector = '#root') {
  const nodes = textValues.map((nodeValue) => ({
    nodeValue,
    parentElement: { tagName: 'SPAN' },
  }));
  const root = { nodes };
  return {
    nodes,
    page: {
      evaluate: async (pageFunction, argument) => {
        const previousDocument = global.document;
        const previousNodeFilter = global.NodeFilter;
        global.NodeFilter = { SHOW_TEXT: 4 };
        global.document = {
          createTreeWalker: (selectedRoot) => {
            let index = 0;
            return { nextNode: () => selectedRoot.nodes[index++] || null };
          },
          querySelectorAll: (candidate) => (candidate === selector ? [root] : []),
        };
        try {
          return await pageFunction(argument);
        } finally {
          global.document = previousDocument;
          global.NodeFilter = previousNodeFilter;
        }
      },
    },
  };
}

function fakeDerivedColorPage(series, { labelDecorationCount = 1, orderedLabels = false } = {}) {
  const styledElement = (sourceColor, parentText = '') => {
    const properties = new Map();
    return {
      parentElement: { textContent: parentText },
      properties,
      querySelectorAll: () => [],
      sourceColor,
      style: {
        setProperty: (name, value) => properties.set(name, value),
      },
    };
  };
  const curves = series.map(({ color, label }) => {
    const curve = styledElement(color);
    curve.label = label;
    curve.stroke = color;
    curve.getAttribute = (name) => (name === 'stroke' ? curve.stroke : null);
    curve.setAttribute = (name, value) => {
      if (name === 'stroke') curve.stroke = value;
    };
    return curve;
  });
  const labelSwatchesByItem = series.map(({ color, labelColor }) =>
    Array.from({ length: labelDecorationCount }, () => styledElement(labelColor || color)),
  );
  const labelSwatches = labelSwatchesByItem.flat();
  const labelItems = series.map(({ label }, index) => ({
    matches: () => false,
    querySelector: (selector) =>
      orderedLabels && selector === '[title]'
        ? { getAttribute: (name) => (name === 'title' ? label : null) }
        : null,
    querySelectorAll: (selector) => (selector === '[style]' ? labelSwatchesByItem[index] : []),
    textContent: orderedLabels ? '' : label,
  }));
  const internalSwatches = series.map(({ color }, index) =>
    styledElement(color, `Series #${index + 1}`),
  );
  const container = {
    querySelectorAll: (selector) => (selector === 'span[style]' ? internalSwatches : []),
  };
  return {
    curves,
    internalSwatches,
    labelSwatches,
    page: {
      evaluate: async (pageFunction, argument) => {
        const previousDocument = global.document;
        const previousGetComputedStyle = global.getComputedStyle;
        global.document = {
          querySelectorAll: (selector) => {
            if (selector === '.fixture-curve') return curves;
            if (selector === '.fixture-label') return labelItems;
            if (selector === '#fixture-root') return [container];
            return [];
          },
        };
        global.getComputedStyle = (element) => ({
          backgroundColor: element.properties?.get('background-color') || element.sourceColor || '',
          stroke: element.properties?.get('stroke') || element.stroke || '',
        });
        try {
          return await pageFunction(argument);
        } finally {
          global.document = previousDocument;
          global.getComputedStyle = previousGetComputedStyle;
        }
      },
    },
  };
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
  assert.equal(
    catalog.every(
      (entry) =>
        entry.expectedChange.global[0].key === 'executions-navigation-removal' &&
        entry.expectedChange.global[0].expectedChange.includes('sidebar entry'),
    ),
    true,
  );

  const baseNavigation = getGlobalVisualNormalizationContract('base').rules[0];
  const headNavigation = getGlobalVisualNormalizationContract('head').rules[0];
  assert.deepEqual(
    {
      expectedMatches: baseNavigation.expectedMatches,
      operation: baseNavigation.operation,
      selector: baseNavigation.selector,
    },
    { expectedMatches: 1, operation: 'hide', selector: '#executionsBtn' },
  );
  assert.deepEqual(
    {
      expectedMatches: headNavigation.expectedMatches,
      operation: headNavigation.operation,
      selector: headNavigation.selector,
    },
    { expectedMatches: 0, operation: 'assert-absent', selector: '#executionsBtn' },
  );
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
  assert.equal(headTask.path, '/#/runs/details/run-1');
  for (const task of [baseTask, headTask]) {
    assert.match(task.actions.find((action) => action.type === 'click').selector, /write-metrics/);
    assert.ok(
      task.actions.some(
        (action) =>
          action.type === 'waitForSelector' &&
          action.selector.includes('aria-selected="true"') &&
          action.selector.includes('Input/Output'),
      ),
    );
    assert.ok(
      task.actions.some(
        (action) => action.type === 'click' && action.selector === '.react-flow__controls-fitview',
      ),
    );
  }

  for (const graph of [
    byKey(base, 'run-details-rich-graph'),
    byKey(head, 'run-details-rich-graph'),
  ]) {
    assert.ok(graph.actions.some((action) => action.selector?.includes('execution-icon-active')));
    assert.ok(graph.actions.some((action) => action.selector?.includes('artifact-icon-live')));
  }
});

test('legacy Executions readiness waits for a rendered numeric execution identity', (t) => {
  const originalDocument = global.document;
  t.after(() => {
    global.document = originalDocument;
  });
  const predicate = byKey(resolveSemanticScenarios('base', SEED_VALUES), 'executions-to-runs')
    .actions[0].predicate;
  const row = (id) => ({
    getAttribute: (name) => (name === 'data-row-id' ? id : null),
  });
  const installDocument = ({ error = false, loading = false, rows = [] } = {}) => {
    global.document = {
      querySelector: (selector) => (selector === '[role="alert"]' && error ? {} : null),
      querySelectorAll: (selector) => {
        if (selector.includes('circularprogress')) return loading ? [{}] : [];
        if (selector === '[data-testid="table-row"]') return rows;
        return [];
      },
    };
  };

  installDocument({ rows: [row('run-1')] });
  assert.equal(predicate(), false);
  installDocument({ rows: [row('73')] });
  assert.equal(predicate(), true);
  installDocument({ loading: true, rows: [row('73')] });
  assert.equal(predicate(), false);
  installDocument({ error: true, rows: [row('73')] });
  assert.equal(predicate(), false);
});

test('serialized base V2 ROC readiness requires three curves and three provenance rows', () => {
  const predicate = byKey(
    resolveSemanticScenarios('base', SEED_VALUES),
    'compare-roc-selection',
  ).actions.at(-1).predicate;
  const evaluateSerializedPredicate = (curveCount, rowCount) =>
    vm.runInNewContext(`(${predicate.toString()})()`, {
      document: {
        querySelectorAll: (selector) => {
          const count = selector.includes('recharts-line') ? curveCount : rowCount;
          return Array.from({ length: count }, () => ({}));
        },
      },
    });

  assert.equal(evaluateSerializedPredicate(2, 3), false);
  assert.equal(evaluateSerializedPredicate(3, 2), false);
  assert.equal(evaluateSerializedPredicate(3, 3), true);
});

test('serialized head ROC selection waits for each committed provenance state', () => {
  const actions = byKey(
    resolveSemanticScenarios('head', SEED_VALUES),
    'compare-roc-selection',
  ).actions;
  const optionActions = actions.filter((action) => action.selector?.startsWith('[role="option"]'));
  const transitionPredicates = actions
    .filter(
      (action) =>
        action.type === 'waitForFunction' &&
        ['twoSelectedRocCurvesReady', 'threeSelectedRocCurvesReady'].includes(
          action.predicate?.name,
        ),
    )
    .map((action) => action.predicate);
  const evaluateSerializedPredicate = (predicate, itemCount) =>
    vm.runInNewContext(`(${predicate.toString()})()`, {
      document: {
        querySelectorAll: () => Array.from({ length: itemCount }, () => ({})),
      },
    });

  assert.equal(optionActions.length, 2);
  assert.deepEqual(
    optionActions.map((action) => action.selector),
    [
      '[role="option"]:has-text("UI Smoke Training Run 1")',
      '[role="option"]:has-text("UI Smoke Evaluation Run")',
    ],
  );
  assert.equal(transitionPredicates.length, 2);
  assert.equal(evaluateSerializedPredicate(transitionPredicates[0], 2), true);
  assert.equal(evaluateSerializedPredicate(transitionPredicates[0], 3), false);
  assert.equal(evaluateSerializedPredicate(transitionPredicates[1], 2), false);
  assert.equal(evaluateSerializedPredicate(transitionPredicates[1], 3), true);
});

test('semantic ID normalization is revision-aware and scoped to declared fixture kinds', () => {
  const base = resolveSemanticScenarios('base', SEED_VALUES);
  const head = resolveSemanticScenarios('head', SEED_VALUES);

  assert.equal(SCENARIO_CONTRACT_SCHEMA_VERSION, 'ui-smoke-scenarios/v2');
  const executionListScopes = byKey(base, 'executions-to-runs').semanticIdNormalization.scopes;
  assert.deepEqual(executionListScopes[0].kinds, ['execution']);
  assert.equal(executionListScopes[0].minReplacements, 1);
  assert.deepEqual(executionListScopes[1].kinds, ['run']);
  assert.equal(executionListScopes[1].minReplacements, 0);
  const baseArtifactList = byKey(base, 'artifact-list-evolution');
  const headArtifactList = byKey(head, 'artifact-list-evolution');
  assert.deepEqual(
    baseArtifactList.semanticIdNormalization.scopes.map((scope) => scope.kinds),
    [['artifact'], ['artifact-uri']],
  );
  assert.deepEqual(
    headArtifactList.semanticIdNormalization.scopes.map((scope) => scope.kinds),
    [['artifact'], ['artifact-uri']],
  );
  for (const artifactList of [baseArtifactList, headArtifactList]) {
    assert.equal(
      artifactList.semanticIdNormalization.scopes.every(
        (scope) => scope.match === 'exact' && scope.selector === '#root [data-testid="table-row"]',
      ),
      true,
    );
  }

  const baseArtifactDetails = byKey(base, 'artifact-details').semanticIdNormalization.scopes;
  assert.equal(baseArtifactDetails.length, 2);
  assert.deepEqual(baseArtifactDetails[0].semanticIds, ['run.training-1/artifact.html-report[0]']);
  assert.equal(baseArtifactDetails[0].maxReplacements, 0);
  assert.deepEqual(baseArtifactDetails[1].semanticIds, [
    'run.training-1/artifact.html-report[0]/uri',
  ]);

  const baseRelationships = byKey(base, 'artifact-related-tasks').semanticIdNormalization.scopes;
  const headRelationships = byKey(head, 'artifact-related-tasks').semanticIdNormalization.scopes[0];
  assert.deepEqual(baseRelationships, []);
  assert.deepEqual(headRelationships.semanticIds, [
    'run.training-1',
    'run.training-1/task.write-metrics[0]',
    'run.training-1/task.consume-metrics[0]',
  ]);
  assert.equal(headRelationships.minReplacements, 4);
  assert.equal(headRelationships.maxReplacements, 8);
  assert.equal(headRelationships.maxReplacementsPerIdentifier, 4);
  assert.equal(headRelationships.match, 'substring');
  assert.match(headRelationships.selector, /table-row/);

  const baseRoc = byKey(base, 'compare-roc-selection');
  const headRoc = byKey(head, 'compare-roc-selection');
  assert.equal(
    baseRoc.semanticIdNormalization.derivedColorScopes[0].mappingStrategy,
    'color-backed-labels',
  );
  assert.match(baseRoc.semanticIdNormalization.derivedColorScopes[0].selector, /recharts-line/);
  assert.match(
    baseRoc.semanticIdNormalization.derivedColorScopes[0].labelItemSelector,
    /table-row/,
  );
  assert.equal(
    headRoc.semanticIdNormalization.derivedColorScopes[0].mappingStrategy,
    'ordered-label-cards',
  );
  assert.deepEqual(
    headRoc.actions.filter((action) => action.type === 'click').map((action) => action.selector),
    [
      '[role="tab"]:has-text("Classification Metrics"), button:has-text("Classification Metrics")',
      '[aria-label="ROC curves"]',
      '[role="option"]:has-text("UI Smoke Training Run 1")',
      '[role="option"]:has-text("UI Smoke Evaluation Run")',
    ],
  );

  const headParallelFor = byKey(head, 'topology-parallel-for');
  assert.equal(
    headParallelFor.actions.some(
      (action) =>
        action.type === 'waitForSelector' &&
        action.selector.includes(':has-text("Loop")') &&
        action.selector.includes('execution-icon-active'),
    ),
    true,
  );

  for (const roleScenarios of [base, head]) {
    for (const scenario of roleScenarios) {
      for (const scope of scenario.semanticIdNormalization?.scopes || []) {
        assert.equal(typeof scope.selector, 'string');
        assert.equal(scope.selector.length > 0, true);
        assert.equal(Object.hasOwn(scope, 'regex'), false);
      }
    }
  }
});

test('different generated run, task, and Artifact IDs normalize to identical semantic tokens', async () => {
  const baseCatalog = capture.buildSemanticIdentifierCatalog(
    semanticIdentifierManifest('base', 'base'),
    'base',
  );
  const headCatalog = capture.buildSemanticIdentifierCatalog(
    semanticIdentifierManifest('head', 'head'),
    'head',
  );
  const config = {
    scopes: [
      {
        match: 'substring',
        maxReplacements: 2,
        minReplacements: 2,
        selector: '#root',
        semanticIds: ['run.training-1', 'run.training-1/task.write-metrics[0]'],
      },
      {
        match: 'exact',
        maxReplacements: 1,
        minReplacements: 1,
        selector: '#root',
        semanticIds: ['run.training-1/artifact.html-report[0]'],
      },
    ],
  };
  const basePage = fakeTextPage([
    'Run base-run-00000000 · Task base-write-task-0000',
    'base-html-artifact-0',
    'unrelated-uuid-99999999',
  ]);
  const headPage = fakeTextPage([
    'Run head-run-00000000 · Task head-write-task-0000',
    'head-html-artifact-0',
    'unrelated-uuid-99999999',
  ]);

  const base = await capture.normalizeSemanticIds(basePage.page, config, baseCatalog);
  const head = await capture.normalizeSemanticIds(headPage.page, config, headCatalog);

  assert.deepEqual(
    comparison.normalizeSemanticIdNormalizationAttestation(base, 'base normalization'),
    base,
  );

  assert.deepEqual(
    basePage.nodes.map((node) => node.nodeValue),
    headPage.nodes.map((node) => node.nodeValue),
  );
  assert.equal(basePage.nodes[2].nodeValue, 'unrelated-uuid-99999999');
  assert.equal(base.totalReplacementCount, 3);
  assert.equal(head.totalReplacementCount, 3);
  assert.deepEqual(
    base.scopes.flatMap((scope) =>
      scope.entries.map(({ semanticId, token }) => ({ semanticId, token })),
    ),
    head.scopes.flatMap((scope) =>
      scope.entries.map(({ semanticId, token }) => ({ semanticId, token })),
    ),
  );
  assert.notEqual(
    base.scopes[0].entries[0].sourceIdSha256,
    head.scopes[0].entries[0].sourceIdSha256,
  );
  assert.equal(JSON.stringify(base).includes('base-run-00000000'), false);
  assert.equal(JSON.stringify(base).includes('base-write-task-0000'), false);
  assert.equal(JSON.stringify(base).includes('base-html-artifact-0'), false);
});

test('legacy catalogs cover every MLMD execution and pair executor-log attempts with native tokens', async () => {
  const manifest = strictSemanticFixtureManifest();
  const baseRun = manifest.deployments.base.bindings.runs['run.training-1'];
  baseRun.executionInstances['execution.unclassified'] = [
    {
      executionId: '991',
      executionRole: 'run-root',
      executorLogs: [],
      name: `run/${baseRun.runId}`,
      state: 'COMPLETE',
    },
  ];
  const baseCatalog = capture.buildSemanticIdentifierCatalog(manifest, 'base');
  const headCatalog = capture.buildSemanticIdentifierCatalog(manifest, 'head');
  const expectedExecutionIds = Object.values(manifest.deployments.base.bindings.runs)
    .flatMap((run) => Object.values(run.executionInstances).flat())
    .map((execution) => execution.executionId)
    .sort();
  assert.deepEqual(
    baseCatalog
      .filter((identifier) => identifier.kind === 'execution')
      .map((identifier) => identifier.value)
      .sort(),
    expectedExecutionIds,
  );

  const workerTasks = (catalog) =>
    catalog.filter(
      (identifier) =>
        identifier.kind === 'task' &&
        identifier.semanticId.startsWith('run.training-1/task.loop-worker['),
    );
  const baseWorkers = workerTasks(baseCatalog);
  const headWorkers = workerTasks(headCatalog);
  assert.equal(baseWorkers.length, 2);
  assert.equal(headWorkers.length, 2);
  assert.equal(new Set(baseWorkers.map((identifier) => identifier.token)).size, 1);
  assert.equal(new Set(headWorkers.map((identifier) => identifier.token)).size, 1);
  assert.deepEqual(
    baseWorkers.map((identifier) => identifier.equivalenceClass),
    ['run.training-1/task.loop-worker/equivalent', 'run.training-1/task.loop-worker/equivalent'],
  );
  assert.deepEqual(
    headWorkers.map((identifier) => identifier.equivalenceClass),
    baseWorkers.map((identifier) => identifier.equivalenceClass),
  );
  assert.deepEqual(
    new Set(baseWorkers.map((identifier) => identifier.token)),
    new Set(headWorkers.map((identifier) => identifier.token)),
  );

  const baseWorkerExecutions = baseCatalog.filter(
    (identifier) =>
      identifier.kind === 'execution' &&
      identifier.semanticId.startsWith('run.training-1/task.loop-worker['),
  );
  assert.equal(baseWorkerExecutions.length, 2);
  assert.equal(new Set(baseWorkerExecutions.map((identifier) => identifier.token)).size, 2);

  const baseLoopIterations = baseCatalog.filter(
    (identifier) =>
      identifier.kind === 'execution' &&
      identifier.semanticId.startsWith('run.training-1/task.parallel-loop/iteration['),
  );
  const headLoopIterations = headCatalog.filter(
    (identifier) =>
      identifier.kind === 'task' &&
      identifier.semanticId.startsWith('run.training-1/task.parallel-loop/iteration['),
  );
  assert.equal(baseLoopIterations.length, 2);
  assert.equal(headLoopIterations.length, 2);
  assert.deepEqual(
    baseLoopIterations.map((identifier) => identifier.token).sort(),
    headLoopIterations.map((identifier) => identifier.token).sort(),
  );

  const workerNormalization = (semanticIds) => ({
    scopes: [
      {
        match: 'exact',
        maxReplacements: 2,
        maxReplacementsPerIdentifier: 1,
        minReplacements: 2,
        minReplacementsPerIdentifier: 1,
        selector: '#root',
        semanticIds,
      },
    ],
  });
  const baseWorkerPage = fakeTextPage(baseWorkers.map((identifier) => identifier.value));
  const headWorkerPage = fakeTextPage(headWorkers.map((identifier) => identifier.value));
  await capture.normalizeSemanticIds(
    baseWorkerPage.page,
    workerNormalization(baseWorkers.map((identifier) => identifier.semanticId)),
    baseCatalog,
  );
  await capture.normalizeSemanticIds(
    headWorkerPage.page,
    workerNormalization(headWorkers.map((identifier) => identifier.semanticId)),
    headCatalog,
  );
  assert.deepEqual(
    new Set(baseWorkerPage.nodes.map((node) => node.nodeValue)),
    new Set(headWorkerPage.nodes.map((node) => node.nodeValue)),
  );

  const reorderedManifest = strictSemanticFixtureManifest();
  reorderedManifest.deployments.base.bindings.runs['run.training-1'].taskInstances[
    'task.loop-worker'
  ].reverse();
  const reorderedWorkers = workerTasks(
    capture.buildSemanticIdentifierCatalog(reorderedManifest, 'base'),
  );
  assert.deepEqual(
    new Map(baseWorkers.map((identifier) => [identifier.value, identifier.token])),
    new Map(reorderedWorkers.map((identifier) => [identifier.value, identifier.token])),
  );

  const joinableManifest = strictSemanticFixtureManifest();
  const joinableRun = joinableManifest.deployments.base.bindings.runs['run.training-1'];
  for (const [index, task] of joinableRun.taskInstances['task.loop-worker'].entries()) {
    task.mlmdExecutionId = joinableRun.executionInstances['task.loop-worker'][index].executionId;
  }
  assert.equal(validateCombinedSemanticManifest(joinableManifest), joinableManifest);
  for (const role of ['base', 'head']) {
    const joinableWorkers = workerTasks(
      capture.buildSemanticIdentifierCatalog(joinableManifest, role),
    );
    assert.equal(new Set(joinableWorkers.map((identifier) => identifier.token)).size, 2);
    assert.equal(
      joinableWorkers.every((identifier) => !identifier.equivalenceClass),
      true,
    );
  }

  const logProjection = (catalog) =>
    new Map(
      catalog
        .filter(
          (identifier) =>
            identifier.semanticId.startsWith(
              'run.training-1/task.retry-once[0]/artifact.executor-logs[',
            ) && ['artifact', 'artifact-uri'].includes(identifier.kind),
        )
        .map((identifier) => [`${identifier.kind}|${identifier.semanticId}`, identifier]),
    );
  const baseLogs = logProjection(baseCatalog);
  const headLogs = logProjection(headCatalog);
  assert.equal(baseLogs.size, 4);
  assert.deepEqual([...baseLogs.keys()].sort(), [...headLogs.keys()].sort());
  for (const [semanticKey, baseIdentifier] of baseLogs) {
    assert.equal(baseIdentifier.token, headLogs.get(semanticKey).token);
    assert.notEqual(baseIdentifier.value, headLogs.get(semanticKey).value);
  }

  const executionScenario = byKey(
    resolveSemanticScenarios('base', SEED_VALUES),
    'executions-to-runs',
  );
  const rootName = fakeTextPage(['991', `run/${baseRun.runId}`], '#root [data-testid="table-row"]');
  const evidence = await capture.normalizeSemanticIds(
    rootName.page,
    executionScenario.semanticIdNormalization,
    baseCatalog,
  );
  assert.match(rootName.nodes[1].nodeValue, /^run\/\[ui-id:run:/);
  assert.equal(evidence.totalReplacementCount, 2);
});

test('semantic identifier catalog rejects task references outside declared artifact bindings', () => {
  const manifest = semanticIdentifierManifest('head', 'head');
  manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
    'task.write-metrics'
  ][0].artifactReferences = {
    outputs: [
      {
        artifacts: [{ artifactId: 'undeclared-artifact', uri: 's3://fixtures/undeclared' }],
        key: 'undeclared',
      },
    ],
  };

  assert.throws(
    () => capture.buildSemanticIdentifierCatalog(manifest, 'head'),
    /is not bound to a declared semantic artifact/,
  );
});

test('semantic identifier catalog binds native executor-log attempts without weakening artifact closure', () => {
  const manifest = semanticIdentifierManifest('head', 'head');
  manifest.deployments.head.bindings.runs['run.training-1'].taskInstances['task.retry-once'] = [
    {
      artifactReferences: {
        inputs: [],
        outputs: [
          {
            artifacts: [
              {
                artifactId: 'head-retry-executor-log-1',
                name: 'executor-logs',
                type: 'Artifact',
                uri: 's3://ui-smoke/head/retry/executor-logs-1',
              },
              {
                artifactId: 'head-retry-executor-log-0',
                name: 'executor-logs',
                type: 'Artifact',
                uri: 's3://ui-smoke/head/retry/executor-logs-0',
              },
            ],
            key: 'executor-logs',
          },
        ],
      },
      taskId: 'head-retry-task-generated',
    },
  ];

  const catalog = capture.buildSemanticIdentifierCatalog(manifest, 'head');
  const executorLogs = catalog.filter((entry) => entry.semanticId.includes('executor-logs'));
  assert.deepEqual(
    executorLogs.map(({ kind, semanticId, value }) => ({ kind, semanticId, value })),
    [
      {
        kind: 'artifact',
        semanticId: 'run.training-1/task.retry-once[0]/artifact.executor-logs[0]',
        value: 'head-retry-executor-log-0',
      },
      {
        kind: 'artifact',
        semanticId: 'run.training-1/task.retry-once[0]/artifact.executor-logs[1]',
        value: 'head-retry-executor-log-1',
      },
      {
        kind: 'artifact-uri',
        semanticId: 'run.training-1/task.retry-once[0]/artifact.executor-logs[0]/uri',
        value: 's3://ui-smoke/head/retry/executor-logs-0',
      },
      {
        kind: 'artifact-uri',
        semanticId: 'run.training-1/task.retry-once[0]/artifact.executor-logs[1]/uri',
        value: 's3://ui-smoke/head/retry/executor-logs-1',
      },
    ],
  );

  manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
    'task.retry-once'
  ][0].artifactReferences.outputs[0].artifacts[0].uri = 's3://ui-smoke/head/retry/executor-logs';
  assert.throws(
    () => capture.buildSemanticIdentifierCatalog(manifest, 'head'),
    /contiguous executor-log attempt URIs/,
  );
});

test('revision-specific Artifact URIs and pod identities normalize to stable semantic tokens', async () => {
  const uriSemanticId = 'run.training-1/artifact.html-report[0]/uri';
  const manifests = {
    base: semanticIdentifierManifest('base', 'base'),
    head: semanticIdentifierManifest('head', 'head'),
  };
  for (const [role, manifest] of Object.entries(manifests)) {
    const artifact =
      manifest.deployments[role].bindings.runs['run.training-1'].artifacts['artifact.html-report'];
    artifact.records = [
      {
        artifactId: artifact.artifactIds[0],
        uri: `s3://ui-smoke/${role}/generated-object-key/report.html`,
      },
    ];
  }
  const baseUri =
    manifests.base.deployments.base.bindings.runs['run.training-1'].artifacts[
      'artifact.html-report'
    ].records[0].uri;
  const headUri =
    manifests.head.deployments.head.bindings.runs['run.training-1'].artifacts[
      'artifact.html-report'
    ].records[0].uri;
  const config = {
    scopes: [
      {
        match: 'exact',
        maxReplacements: 1,
        maxReplacementsPerIdentifier: 1,
        minReplacements: 1,
        minReplacementsPerIdentifier: 1,
        selector: '#root',
        semanticIds: [uriSemanticId],
      },
    ],
  };
  const basePage = fakeTextPage([baseUri]);
  const headPage = fakeTextPage([headUri]);
  const baseEvidence = await capture.normalizeSemanticIds(
    basePage.page,
    config,
    capture.buildSemanticIdentifierCatalog(manifests.base, 'base'),
  );
  const headEvidence = await capture.normalizeSemanticIds(
    headPage.page,
    config,
    capture.buildSemanticIdentifierCatalog(manifests.head, 'head'),
  );
  assert.equal(basePage.nodes[0].nodeValue, headPage.nodes[0].nodeValue);
  assert.equal(basePage.nodes[0].nodeValue, `[ui-id:artifact-uri:training-1:html-report:0:uri]`);
  assert.equal(JSON.stringify(baseEvidence).includes(baseUri), false);
  assert.equal(JSON.stringify(headEvidence).includes(headUri), false);

  const podManifest = semanticIdentifierManifest('head', 'head');
  podManifest.deployments.head.bindings.runs['run.training-1'].taskInstances['task.retry-once'] = [
    {
      podBindings: [
        {
          name: 'retry-attempt-0-generated',
          type: 'EXECUTOR',
          uid: '11111111-aaaa-bbbb-cccc-111111111111',
        },
        {
          name: 'retry-attempt-1-generated',
          type: 'EXECUTOR',
          uid: '22222222-aaaa-bbbb-cccc-222222222222',
        },
      ],
      taskId: 'head-retry-task-generated',
    },
  ];
  const podText = podManifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
    'task.retry-once'
  ][0].podBindings
    .flatMap((pod) => [pod.name, pod.uid])
    .join(' | ');
  const podPage = fakeTextPage([podText]);
  const podEvidence = await capture.normalizeSemanticIds(
    podPage.page,
    {
      scopes: [
        {
          match: 'substring',
          maxReplacements: 4,
          minReplacements: 4,
          selector: '#root',
          semanticIdPrefixes: ['run.training-1/task.retry-once[0]/pod.executor['],
        },
      ],
    },
    capture.buildSemanticIdentifierCatalog(podManifest, 'head'),
  );
  assert.equal(podEvidence.totalReplacementCount, 4);
  assert.equal(podPage.nodes[0].nodeValue.includes('generated'), false);
  assert.equal(podPage.nodes[0].nodeValue.includes('11111111-aaaa'), false);
  assert.equal(
    podEvidence.scopes[0].entries.every((entry) => entry.kind === 'pod'),
    true,
  );

  const legacyRetryManifest = strictSemanticFixtureManifest();
  const legacyRetryRun = legacyRetryManifest.deployments.base.bindings.runs['run.training-1'];
  const legacyRetryTask = legacyRetryRun.taskInstances['task.retry-once'][0];
  const legacyRetryExecution = legacyRetryRun.executionInstances['task.retry-once'][0];
  legacyRetryTask.failedMainJobs = [];
  legacyRetryTask.podName = 'retry-attempt-1-generated';
  legacyRetryExecution.podName = 'retry-attempt-1-generated';
  const legacyRetryCatalog = capture.buildSemanticIdentifierCatalog(legacyRetryManifest, 'base');
  assert.equal(
    legacyRetryCatalog.find(
      (entry) =>
        entry.kind === 'pod' &&
        entry.semanticId === 'run.training-1/task.retry-once[0]/pod.executor[1]/name',
    )?.value,
    'retry-attempt-1-generated',
  );
  assert.equal(
    legacyRetryCatalog.some(
      (entry) =>
        entry.kind === 'pod' &&
        entry.semanticId === 'run.training-1/task.retry-once[0]/pod.executor[0]/name' &&
        entry.value === 'retry-attempt-1-generated',
    ),
    false,
  );
});

test('ROC series colors normalize by visible semantic label instead of generated ID order', async () => {
  const config = {
    derivedColorScopes: [
      {
        containerSelector: '#fixture-root',
        key: 'fixture-roc',
        labelItemSelector: '.fixture-label',
        mappingStrategy: 'color-backed-labels',
        maxElements: 2,
        minElements: 2,
        selector: '.fixture-curve',
        semanticIds: ['run.training-a', 'run.training-z'],
      },
    ],
    scopes: [],
  };
  const basePage = fakeDerivedColorPage([
    { color: 'rgb(220,0,0)', label: 'Training Run Z' },
    { color: 'rgb(0,0,220)', label: 'Training Run A' },
  ]);
  const headPage = fakeDerivedColorPage([
    { color: 'rgb(160,0,160)', label: 'Training Run A' },
    { color: 'rgb(255,140,0)', label: 'Training Run Z' },
  ]);
  const catalog = [
    { displayLabel: 'Training Run A', kind: 'run', semanticId: 'run.training-a' },
    { displayLabel: 'Training Run Z', kind: 'run', semanticId: 'run.training-z' },
  ];
  const base = await capture.normalizeSemanticDerivedColors(basePage.page, config, catalog);
  const head = await capture.normalizeSemanticDerivedColors(headPage.page, config, catalog);
  const projection = (evidence) =>
    evidence[0].mappings.map(({ semanticId, paletteColor }) => ({
      paletteColor,
      semanticId,
    }));
  assert.deepEqual(projection(base), projection(head));
  assert.deepEqual(projection(base), [
    { paletteColor: '#4285f4', semanticId: 'run.training-a' },
    { paletteColor: '#2b9c1e', semanticId: 'run.training-z' },
  ]);
  for (const fixture of [basePage, headPage]) {
    const colorByLabel = Object.fromEntries(
      fixture.curves.map((curve) => [curve.label, curve.properties.get('stroke')]),
    );
    assert.deepEqual(colorByLabel, {
      'Training Run A': '#4285f4',
      'Training Run Z': '#2b9c1e',
    });
    assert.equal(
      fixture.labelSwatches.every((swatch) => swatch.properties.has('background-color')),
      true,
    );
    assert.equal(
      fixture.internalSwatches.every((swatch) => swatch.properties.has('background-color')),
      true,
    );
  }
  assert.notDeepEqual(
    base[0].mappings.map((mapping) => mapping.sourceColorSha256),
    head[0].mappings.map((mapping) => mapping.sourceColorSha256),
  );

  const orderedConfig = structuredClone(config);
  orderedConfig.derivedColorScopes[0].mappingStrategy = 'ordered-label-cards';
  const legacyPage = fakeDerivedColorPage(
    [
      {
        color: 'rgb(220,0,0)',
        label: 'Training Run Z',
        labelColor: 'rgb(219,0,0)',
      },
      {
        color: 'rgb(0,0,220)',
        label: 'Training Run A',
        labelColor: 'rgb(0,0,219)',
      },
    ],
    { orderedLabels: true },
  );
  const legacy = await capture.normalizeSemanticDerivedColors(
    legacyPage.page,
    orderedConfig,
    catalog,
  );
  assert.deepEqual(projection(legacy), projection(base));
  assert.equal(
    legacyPage.internalSwatches.every((swatch) => swatch.properties.has('background-color')),
    true,
  );

  for (const labelDecorationCount of [0, 2]) {
    const orderedPage = fakeDerivedColorPage(
      [
        { color: 'rgb(220,0,0)', label: 'Training Run Z' },
        { color: 'rgb(0,0,220)', label: 'Training Run A' },
      ],
      { labelDecorationCount, orderedLabels: true },
    );
    const normalized = await capture.normalizeSemanticDerivedColors(
      orderedPage.page,
      orderedConfig,
      catalog,
    );
    assert.deepEqual(projection(normalized), projection(base));
    assert.equal(
      orderedPage.labelSwatches.every((swatch) => swatch.properties.has('background-color')),
      true,
    );
  }

  await assert.rejects(
    capture.normalizeSemanticDerivedColors(
      fakeDerivedColorPage([
        { color: 'rgb(220,0,0)', label: 'Training Run A' },
        { color: 'rgb(0,0,220)', label: '' },
      ]).page,
      config,
      catalog,
    ),
    (error) => error.captureValidity === 'selector_drift',
  );
});

test('exact normalization does not corrupt short numeric MLMD IDs or unrelated text', async () => {
  const catalog = capture.buildSemanticIdentifierCatalog(
    semanticIdentifierManifest('base', 'base', { executionId: '73' }),
    'base',
  );
  const fixture = fakeTextPage(['73', '173', '73.0', '00:00:73', 'unrelated-uuid']);
  const result = await capture.normalizeSemanticIds(
    fixture.page,
    {
      scopes: [
        {
          match: 'exact',
          maxReplacements: 1,
          minReplacements: 1,
          selector: '#root',
          semanticIds: ['run.training-1/task.write-metrics[0]/execution'],
        },
      ],
    },
    catalog,
  );

  assert.match(fixture.nodes[0].nodeValue, /^\[ui-id:task:/);
  assert.deepEqual(
    fixture.nodes.slice(1).map((node) => node.nodeValue),
    ['173', '73.0', '00:00:73', 'unrelated-uuid'],
  );
  assert.equal(result.totalReplacementCount, 1);
});

test('semantic ID normalization fails closed on missing, excess, and ambiguous bindings', async (t) => {
  const catalog = capture.buildSemanticIdentifierCatalog(
    semanticIdentifierManifest('head', 'head'),
    'head',
  );
  const config = {
    scopes: [
      {
        match: 'exact',
        maxReplacements: 1,
        minReplacements: 1,
        selector: '#root',
        semanticIds: ['run.training-1/artifact.html-report[0]'],
      },
    ],
  };
  for (const [name, values] of [
    ['missing', ['not-the-artifact']],
    ['excess', ['head-html-artifact-0', 'head-html-artifact-0']],
  ]) {
    await t.test(name, async () => {
      await assert.rejects(
        capture.normalizeSemanticIds(fakeTextPage(values).page, config, catalog),
        (error) => error.captureValidity === 'selector_drift',
      );
    });
  }

  assert.throws(
    () =>
      capture.buildSemanticIdentifierCatalog(
        semanticIdentifierManifest('head', 'head', {
          consumeTaskId: 'shared-generated-task',
          writeTaskId: 'shared-generated-task',
        }),
        'head',
      ),
    /ambiguously bound/,
  );

  assert.throws(
    () =>
      capture.buildSemanticIdentifierCatalog(
        semanticIdentifierManifest('head', 'head', {
          resourceRunId: 'resource-run-binding-id',
          runId: 'detail-run-binding-id',
        }),
        'head',
      ),
    /bound to multiple generated values/,
  );

  const tokenCollisionManifest = semanticIdentifierManifest('head', 'head');
  tokenCollisionManifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
    'write-metrics'
  ] = [{ taskId: 'head-token-collision-task' }];
  assert.throws(
    () => capture.buildSemanticIdentifierCatalog(tokenCollisionManifest, 'head'),
    /produce the same visual token/,
  );

  for (const [name, mutate, expected] of [
    [
      'schema',
      (manifest) => {
        manifest.schemaVersion = 'ui-smoke-semantic/v2';
      },
      /must use schema ui-smoke-semantic\/v3/,
    ],
    [
      'fixture set',
      (manifest) => {
        manifest.fixtureSet = 'partial-fixtures';
      },
      /must use fixture set ui-smoke-deterministic-v3/,
    ],
    [
      'validation',
      (manifest) => {
        manifest.deployments.head.validation = { errors: ['missing fixture'], valid: false };
      },
      /has not passed fixture validation/,
    ],
    [
      'revision flavor',
      (manifest) => {
        manifest.deployments.head.revisionFlavor = 'legacy-mlmd';
      },
      /must use native-task-artifact/,
    ],
  ]) {
    const manifest = semanticIdentifierManifest('head', `invalid-${name}`);
    mutate(manifest);
    assert.throws(() => capture.buildSemanticIdentifierCatalog(manifest, 'head'), expected);
  }
});

test('substring normalization is one-pass and never rewrites generated semantic tokens', async () => {
  const catalog = capture.buildSemanticIdentifierCatalog(
    semanticIdentifierManifest('head', 'head', {
      runId: 'head-run-00000000',
      writeTaskId: 'training-1',
    }),
    'head',
  );
  const fixture = fakeTextPage(['head-run-00000000']);
  const result = await capture.normalizeSemanticIds(
    fixture.page,
    {
      scopes: [
        {
          match: 'substring',
          maxReplacements: 1,
          minReplacements: 1,
          selector: '#root',
          semanticIds: ['run.training-1', 'run.training-1/task.write-metrics[0]'],
        },
      ],
    },
    catalog,
  );

  assert.equal(fixture.nodes[0].nodeValue, '[ui-id:run:training-1]');
  assert.equal(result.totalReplacementCount, 1);
  assert.equal(
    result.scopes[0].entries.find(
      (entry) => entry.semanticId === 'run.training-1/task.write-metrics[0]',
    ).replacementCount,
    0,
  );
});

test('per-identifier bounds reject the wrong replacement distribution', async () => {
  const catalog = capture.buildSemanticIdentifierCatalog(
    semanticIdentifierManifest('head', 'head'),
    'head',
  );
  const fixture = fakeTextPage([
    'head-run-00000000 head-run-00000000',
    'head-write-task-0000 head-write-task-0000',
  ]);

  await assert.rejects(
    capture.normalizeSemanticIds(
      fixture.page,
      {
        scopes: [
          {
            match: 'substring',
            maxReplacements: 4,
            maxReplacementsPerIdentifier: 2,
            minReplacements: 4,
            minReplacementsPerIdentifier: 1,
            selector: '#root',
            semanticIds: [
              'run.training-1',
              'run.training-1/task.write-metrics[0]',
              'run.training-1/task.consume-metrics[0]',
            ],
          },
        ],
      },
      catalog,
    ),
    (error) =>
      error.captureValidity === 'selector_drift' &&
      /task\.consume-metrics\[0\] 0 time/.test(error.message),
  );
});

test('substring normalization rejects numeric and short generated identifiers', () => {
  const catalog = capture.buildSemanticIdentifierCatalog(
    semanticIdentifierManifest('base', 'base', {
      executionId: '73',
      writeTaskId: 'shortid',
    }),
    'base',
  );
  for (const semanticId of [
    'run.training-1/task.write-metrics[0]/execution',
    'run.training-1/task.write-metrics[0]',
  ]) {
    assert.throws(
      () =>
        capture.prepareSemanticIdNormalization(
          {
            scopes: [
              {
                match: 'substring',
                selector: '#root',
                semanticIds: [semanticId],
              },
            ],
          },
          catalog,
        ),
      /cannot substring-match short or numeric identifiers/,
    );
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
    for (const [key, tab] of [
      ['compare-html', 'HTML'],
      ['compare-markdown', 'Markdown'],
    ]) {
      assert.ok(
        byKey(resolved, key).actions.some(
          (action) =>
            action.type === 'waitForSelector' &&
            action.selector.includes('aria-selected="true"') &&
            action.selector.includes(tab),
        ),
      );
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

test('task comparisons use equivalent node-click, tab, and fit-view state', () => {
  const base = resolveSemanticScenarios('base', SEED_VALUES);
  const head = resolveSemanticScenarios('head', SEED_VALUES);
  const expectedTabs = new Map([
    ['run-details-task-panel', 'Input/Output'],
    ['run-details-task-logs', 'Logs'],
    ['topology-retried-task', 'Task Details'],
    ['topology-parallel-for', 'Task Details'],
    ['topology-nested-dag', 'Task Details'],
  ]);

  for (const [key, expectedTab] of expectedTabs) {
    const variants = [byKey(base, key), byKey(head, key)];
    assert.equal(variants[0].path, variants[1].path, `${key} must use the same entry route`);
    for (const variant of variants) {
      assert.equal(variant.path.includes('?task='), false, `${key} must select from the graph`);
      assert.ok(
        variant.actions.some(
          (action) => action.type === 'click' && action.selector?.includes('react-flow__node'),
        ),
        `${key} must click its graph node`,
      );
      assert.ok(
        variant.actions.some(
          (action) =>
            action.type === 'waitForSelector' &&
            action.selector.includes('aria-selected="true"') &&
            action.selector.includes(expectedTab),
        ),
        `${key} must confirm ${expectedTab}`,
      );
      assert.ok(
        variant.actions.some(
          (action) =>
            action.type === 'click' && action.selector === '.react-flow__controls-fitview',
        ),
        `${key} must fit the graph after opening the panel`,
      );
    }
  }
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
    assert.equal(
      pages.find((page) => page.name === retained).expectedChange.global[0].key,
      'executions-navigation-removal',
    );
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

test('legacy lineage readiness is data-specific', () => {
  const base = resolveSemanticScenarios('base', SEED_VALUES);
  const related = byKey(base, 'artifact-related-tasks');
  assert.ok(related.actions.some((action) => action.type === 'waitForFunction'));
  assert.equal(
    related.actions.some((action) => action.selector === 'svg'),
    false,
  );
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
