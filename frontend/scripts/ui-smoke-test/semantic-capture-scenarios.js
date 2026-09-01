'use strict';

const SCENARIO_CONTRACT_SCHEMA_VERSION = 'ui-smoke-scenarios/v1';

const EXPECTED_CHANGES = Object.freeze({
  artifactList:
    'The grouped MLMD artifact view is replaced by the native Artifact API-backed list.',
  artifactRelationships:
    'The MLMD Lineage Explorer is replaced by native producing and consuming task relationships.',
  executions: 'The removed Executions surface redirects to the replacement Runs experience.',
  nativeRuntime:
    'MLMD-backed runtime presentation is replaced by native Task and Artifact API presentation.',
});

function seededListReady() {
  const hasError = !!document.querySelector('[role="alert"]');
  const isLoading =
    document.querySelectorAll('[role="circularprogress"], .MuiCircularProgress-root').length > 0;
  const hasRows =
    document.querySelectorAll('[data-testid="table-row"]').length > 0 ||
    document.querySelectorAll('table tbody tr').length > 0 ||
    document.querySelectorAll('[class*="tableRow"]').length > 0;
  return !hasError && !isLoading && hasRows;
}

function graphReady() {
  const legacyNodes = document.querySelectorAll('.graphNode');
  if (legacyNodes.length > 0) return true;
  const flowNodes = Array.from(document.querySelectorAll('.react-flow__node'));
  return (
    flowNodes.length > 0 &&
    flowNodes.every((node) => getComputedStyle(node).visibility !== 'hidden')
  );
}

function scalarMetricsReady() {
  const text = document.body.innerText;
  return (
    !text.includes('no Scalar Metrics') &&
    !text.includes('no scalar metrics') &&
    text.includes('accuracy') &&
    text.includes('loss')
  );
}

function rocReady() {
  const text = document.body.innerText;
  return (
    !text.includes('no ROC') &&
    !text.includes('No ROC') &&
    !!document.querySelector('.recharts-wrapper .recharts-line-curve, .rv-xy-plot')
  );
}

function legacyLineageReady() {
  const explorer = document.querySelector('.LineageExplorer');
  return (
    !!explorer &&
    !explorer.querySelector('[role="circularprogress"], .MuiCircularProgress-root') &&
    !!explorer.querySelector('[data-testid="card-row"]')
  );
}

const waitForList = Object.freeze([{ type: 'waitForFunction', predicate: seededListReady }]);
const waitForGraph = Object.freeze([{ type: 'waitForFunction', predicate: graphReady }]);

const nodeSelector = (name) =>
  `.react-flow__node:has-text("${name}"), .graphNode:has-text("${name}")`;
const nodeDataSelector = (name, testId) =>
  `.react-flow__node:has-text("${name}") [data-testid="${testId}"], ` +
  `.graphNode:has-text("${name}") [data-testid="${testId}"]`;
const tabSelector = (name) => `[role="tab"]:has-text("${name}"), button:has-text("${name}")`;
const waitForHydratedGraph = Object.freeze([
  ...waitForGraph,
  {
    type: 'waitForSelector',
    selector: nodeDataSelector('write-metrics', 'execution-icon-active'),
  },
  {
    type: 'waitForSelector',
    selector: nodeDataSelector('scalar_metrics', 'artifact-icon-live'),
  },
]);

const taskPanelActions = (taskName, tabName, extraActions = []) => [
  ...waitForGraph,
  { type: 'waitForSelector', selector: nodeDataSelector(taskName, 'execution-icon-active') },
  { type: 'click', selector: nodeSelector(taskName) },
  { type: 'waitForSelector', selector: '[aria-label="close"]' },
  ...(tabName ? [{ type: 'click', selector: tabSelector(tabName) }] : []),
  ...extraActions,
];

const headTaskPanelActions = (taskName, tabName, extraActions = []) => [
  ...waitForGraph,
  { type: 'waitForSelector', selector: nodeDataSelector(taskName, 'execution-icon-active') },
  { type: 'waitForSelector', selector: '[aria-label="close"]' },
  ...(tabName ? [{ type: 'click', selector: tabSelector(tabName) }] : []),
  ...extraActions,
];

const artifactVisualizationActions = (artifactKey, readyAction) => [
  ...waitForGraph,
  { type: 'waitForSelector', selector: nodeDataSelector(artifactKey, 'artifact-icon-live') },
  { type: 'click', selector: nodeSelector(artifactKey) },
  { type: 'waitForSelector', selector: '[aria-label="close"]' },
  { type: 'click', selector: tabSelector('Visualization') },
  readyAction,
];

const baseComparisonSelection = (kind, ordinal, runIndex) => [
  { type: 'click', selector: `button:has-text("Choose a ${ordinal} ${kind} artifact")` },
  { type: 'hover', selector: 'ul[class*="dropdownMenu"] > li', index: runIndex },
  { type: 'waitForSelector', selector: 'ul[class*="dropdownSubmenu"] > li' },
  { type: 'click', selector: 'ul[class*="dropdownSubmenu"] > li' },
];

const baseFileComparisonActions = (kind, readyAction) => [
  { type: 'waitForFunction', predicate: seededListReady },
  { type: 'click', selector: tabSelector(kind) },
  ...baseComparisonSelection(kind, 'first', 0),
  ...baseComparisonSelection(kind, 'second', 1),
  { ...readyAction, minCount: 2 },
];

const headComparisonSelection = (label, optionIndex) => [
  { type: 'click', selector: `[aria-label="${label} comparison artifact"]` },
  {
    type: 'click',
    selector: '[role="option"]:not(:has-text("Choose an artifact"))',
    index: optionIndex,
  },
];

const headFileComparisonActions = (kind, readyAction) => [
  { type: 'waitForFunction', predicate: seededListReady },
  { type: 'click', selector: tabSelector(kind) },
  ...headComparisonSelection('First', 0),
  ...headComparisonSelection('Second', 1),
  { ...readyAction, minCount: 2 },
];

const SEMANTIC_SCENARIOS = Object.freeze([
  {
    key: 'executions-to-runs',
    title: 'Legacy Executions to Runs',
    expectedChange: EXPECTED_CHANGES.executions,
    revisions: {
      base: {
        path: '/#/executions',
        routeExpectation: { kind: 'direct', path: '/executions' },
        waitFor: '#root',
        actions: waitForList,
      },
      head: {
        path: '/#/executions',
        routeExpectation: { kind: 'expected-removal', path: '/runs' },
        waitFor: '#root',
        actions: [
          ...waitForList,
          {
            type: 'assertAbsent',
            selector: '#executionsBtn, a[href="/executions"]',
            failureValidity: 'ui_rendering_failure',
          },
        ],
      },
    },
  },
  {
    key: 'artifact-list-evolution',
    title: 'Grouped legacy Artifacts to native Artifact list',
    expectedChange: EXPECTED_CHANGES.artifactList,
    revisions: {
      base: {
        path: '/#/artifacts',
        routeExpectation: { kind: 'direct', path: '/artifacts' },
        waitFor: '#root',
        actions: waitForList,
      },
      head: {
        path: '/#/artifacts',
        routeExpectation: { kind: 'direct', path: '/artifacts' },
        waitFor: '#root',
        actions: waitForList,
      },
    },
  },
  {
    key: 'run-details-rich-graph',
    title: 'Run Details rich topology graph',
    requires: ['richRunId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: waitForHydratedGraph,
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: waitForHydratedGraph,
      },
    },
  },
  {
    key: 'run-details-task-panel',
    title: 'Run Details task panel',
    requires: ['richRunId', 'writeMetricsTaskId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: taskPanelActions('write-metrics', 'Input/Output'),
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}?task={seed.writeMetricsTaskId}',
        routeExpectation: {
          kind: 'direct',
          path: '/runs/details/{seed.richRunId}?task={seed.writeMetricsTaskId}',
        },
        waitFor: '#root',
        actions: headTaskPanelActions('write-metrics', 'Input/Output'),
      },
    },
  },
  {
    key: 'run-details-task-logs',
    title: 'Run Details task logs',
    requires: ['richRunId', 'retryTaskId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: taskPanelActions('retry-once', 'Logs', [
          { type: 'waitForText', text: 'retry completed' },
        ]),
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}?task={seed.retryTaskId}',
        routeExpectation: {
          kind: 'direct',
          path: '/runs/details/{seed.richRunId}?task={seed.retryTaskId}',
        },
        waitFor: '#root',
        actions: headTaskPanelActions('retry-once', 'Logs', [
          { type: 'waitForText', text: 'retry completed' },
        ]),
      },
    },
  },
  {
    key: 'run-details-scalar-metrics',
    title: 'Run Details scalar metrics',
    requires: ['richRunId', 'scalarArtifactId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: artifactVisualizationActions('scalar_metrics', {
          type: 'waitForFunction',
          predicate: scalarMetricsReady,
        }),
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: artifactVisualizationActions('scalar_metrics', {
          type: 'waitForFunction',
          predicate: scalarMetricsReady,
        }),
      },
    },
  },
  {
    key: 'run-details-html',
    title: 'Run Details HTML visualization',
    requires: ['richRunId', 'htmlArtifactId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: artifactVisualizationActions('html_report', {
          type: 'waitForFrameText',
          text: 'UI Smoke HTML Report',
        }),
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: artifactVisualizationActions('html_report', {
          type: 'waitForFrameText',
          text: 'UI Smoke HTML Report',
        }),
      },
    },
  },
  {
    key: 'run-details-markdown',
    title: 'Run Details Markdown visualization',
    requires: ['richRunId', 'markdownArtifactId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: artifactVisualizationActions('markdown_report', {
          type: 'waitForText',
          text: 'UI Smoke Markdown Report',
        }),
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: artifactVisualizationActions('markdown_report', {
          type: 'waitForText',
          text: 'UI Smoke Markdown Report',
        }),
      },
    },
  },
  {
    key: 'run-details-roc',
    title: 'Run Details ROC visualization',
    requires: ['richRunId', 'rocArtifactId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: artifactVisualizationActions('roc_curve', {
          type: 'waitForFunction',
          predicate: rocReady,
        }),
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: artifactVisualizationActions('roc_curve', {
          type: 'waitForFunction',
          predicate: rocReady,
        }),
      },
    },
  },
  {
    key: 'compare-runs',
    title: 'Compare runs and scalar metrics',
    requires: ['compareRunlist'],
    revisions: {
      base: {
        path: '/#/compare?runlist={seed.compareRunlist}',
        routeExpectation: { kind: 'direct', path: '/compare?runlist={seed.compareRunlist}' },
        waitFor: '#root',
        actions: [
          { type: 'waitForFunction', predicate: seededListReady },
          { type: 'waitForFunction', predicate: scalarMetricsReady },
        ],
      },
      head: {
        path: '/#/compare?runlist={seed.compareRunlist}',
        routeExpectation: { kind: 'direct', path: '/compare?runlist={seed.compareRunlist}' },
        waitFor: '#root',
        actions: [
          { type: 'waitForFunction', predicate: seededListReady },
          { type: 'waitForFunction', predicate: scalarMetricsReady },
        ],
      },
    },
  },
  {
    key: 'compare-roc-selection',
    title: 'Compare ROC curve selections',
    requires: ['compareRunlist'],
    expectedChange: 'ROC Curve is consolidated into the native Classification Metrics tab.',
    revisions: {
      base: {
        path: '/#/compare?runlist={seed.compareRunlist}',
        routeExpectation: { kind: 'direct', path: '/compare?runlist={seed.compareRunlist}' },
        waitFor: '#root',
        actions: [
          { type: 'waitForFunction', predicate: seededListReady },
          { type: 'click', selector: tabSelector('ROC Curve') },
          { type: 'waitForFunction', predicate: rocReady },
        ],
      },
      head: {
        path: '/#/compare?runlist={seed.compareRunlist}',
        routeExpectation: { kind: 'direct', path: '/compare?runlist={seed.compareRunlist}' },
        waitFor: '#root',
        actions: [
          { type: 'waitForFunction', predicate: seededListReady },
          { type: 'click', selector: tabSelector('Classification Metrics') },
          { type: 'waitForSelector', selector: '[aria-label="ROC curves"]' },
          { type: 'click', selector: '[aria-label="ROC curves"]' },
          { type: 'click', selector: '[role="option"]' },
          { type: 'press', key: 'Escape' },
          { type: 'waitForFunction', predicate: rocReady },
        ],
      },
    },
  },
  {
    key: 'compare-html',
    title: 'Compare HTML reports',
    requires: ['compareRunlist', 'htmlArtifactId'],
    revisions: {
      base: {
        path: '/#/compare?runlist={seed.compareRunlist}',
        routeExpectation: { kind: 'direct', path: '/compare?runlist={seed.compareRunlist}' },
        waitFor: '#root',
        actions: baseFileComparisonActions('HTML', {
          type: 'waitForFrameText',
          text: 'UI Smoke HTML Report',
        }),
      },
      head: {
        path: '/#/compare?runlist={seed.compareRunlist}',
        routeExpectation: { kind: 'direct', path: '/compare?runlist={seed.compareRunlist}' },
        waitFor: '#root',
        actions: headFileComparisonActions('HTML', {
          type: 'waitForFrameText',
          text: 'UI Smoke HTML Report',
        }),
      },
    },
  },
  {
    key: 'compare-markdown',
    title: 'Compare Markdown reports',
    requires: ['compareRunlist', 'markdownArtifactId'],
    revisions: {
      base: {
        path: '/#/compare?runlist={seed.compareRunlist}',
        routeExpectation: { kind: 'direct', path: '/compare?runlist={seed.compareRunlist}' },
        waitFor: '#root',
        actions: baseFileComparisonActions('Markdown', {
          type: 'waitForText',
          text: 'UI Smoke Markdown Report',
        }),
      },
      head: {
        path: '/#/compare?runlist={seed.compareRunlist}',
        routeExpectation: { kind: 'direct', path: '/compare?runlist={seed.compareRunlist}' },
        waitFor: '#root',
        actions: headFileComparisonActions('Markdown', {
          type: 'waitForText',
          text: 'UI Smoke Markdown Report',
        }),
      },
    },
  },
  {
    key: 'artifact-details',
    title: 'Artifact Details',
    requires: ['htmlArtifactId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/artifacts/{seed.htmlArtifactId}',
        routeExpectation: { kind: 'direct', path: '/artifacts/{seed.htmlArtifactId}' },
        waitFor: '#root',
        actions: [
          { type: 'waitForText', text: 'Overview' },
          { type: 'waitForText', text: 'URI' },
          { type: 'waitForText', text: 'html_report' },
        ],
      },
      head: {
        path: '/#/artifacts/{seed.htmlArtifactId}',
        routeExpectation: { kind: 'direct', path: '/artifacts/{seed.htmlArtifactId}' },
        waitFor: '#root',
        actions: [
          { type: 'waitForText', text: 'Artifact details' },
          { type: 'waitForText', text: 'html_report' },
        ],
      },
    },
  },
  {
    key: 'artifact-related-tasks',
    title: 'Artifact relationships',
    requires: ['consumeMetricsTaskId', 'relatedArtifactId', 'richRunId', 'writeMetricsTaskId'],
    expectedChange: EXPECTED_CHANGES.artifactRelationships,
    revisions: {
      base: {
        path: '/#/artifacts/{seed.relatedArtifactId}',
        routeExpectation: { kind: 'direct', path: '/artifacts/{seed.relatedArtifactId}' },
        waitFor: '#root',
        actions: [
          { type: 'click', selector: tabSelector('Lineage Explorer') },
          { type: 'waitForFunction', predicate: legacyLineageReady },
          { type: 'waitForText', text: 'write-metrics' },
          { type: 'waitForText', text: 'consume-metrics' },
        ],
      },
      head: {
        path: '/#/artifacts/{seed.relatedArtifactId}/lineage',
        routeExpectation: { kind: 'direct', path: '/artifacts/{seed.relatedArtifactId}/lineage' },
        waitFor: '#root',
        actions: [
          { type: 'waitForText', text: 'Related tasks' },
          { type: 'waitForText', text: 'Producing and consuming tasks' },
          { type: 'waitForText', text: 'Produced as scalar_metrics' },
          { type: 'waitForText', text: 'Consumed as metrics' },
          { type: 'waitForText', text: 'Run {seed.richRunId}' },
          { type: 'waitForText', text: 'Task {seed.writeMetricsTaskId}' },
          { type: 'waitForText', text: 'Task {seed.consumeMetricsTaskId}' },
        ],
      },
    },
  },
  {
    key: 'topology-retried-task',
    title: 'Retried task topology',
    requires: ['richRunId', 'retryTaskId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: taskPanelActions('retry-once', 'Input/Output'),
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}?task={seed.retryTaskId}',
        routeExpectation: {
          kind: 'direct',
          path: '/runs/details/{seed.richRunId}?task={seed.retryTaskId}',
        },
        waitFor: '#root',
        actions: headTaskPanelActions('retry-once', 'Task Details'),
      },
    },
  },
  {
    key: 'topology-parallel-for',
    title: 'ParallelFor topology',
    requires: ['richRunId', 'parallelTaskId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: taskPanelActions('parallel-loop', null),
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}?task={seed.parallelTaskId}',
        routeExpectation: {
          kind: 'direct',
          path: '/runs/details/{seed.richRunId}?task={seed.parallelTaskId}',
        },
        waitFor: '#root',
        actions: headTaskPanelActions('parallel-loop', 'Task Details'),
      },
    },
  },
  {
    key: 'topology-nested-dag',
    title: 'Nested DAG topology',
    requires: ['richRunId', 'nestedDagTaskId'],
    expectedChange: EXPECTED_CHANGES.nativeRuntime,
    revisions: {
      base: {
        path: '/#/runs/details/{seed.richRunId}',
        routeExpectation: { kind: 'direct', path: '/runs/details/{seed.richRunId}' },
        waitFor: '#root',
        actions: taskPanelActions('nested-dag', null),
      },
      head: {
        path: '/#/runs/details/{seed.richRunId}?task={seed.nestedDagTaskId}',
        routeExpectation: {
          kind: 'direct',
          path: '/runs/details/{seed.richRunId}?task={seed.nestedDagTaskId}',
        },
        waitFor: '#root',
        actions: headTaskPanelActions('nested-dag', 'Task Details'),
      },
    },
  },
  {
    key: 'historical-artifact-evolution',
    title: 'Historical legacy Artifact to native Artifact',
    required: false,
    requires: ['historicalArtifactId'],
    expectedChange:
      'Clean-stack mode compares equivalent legacy and native artifacts; preserved historical legacy artifacts require upgrade mode.',
    revisions: {
      base: {
        path: '/#/artifacts/{seed.historicalArtifactId}',
        routeExpectation: { kind: 'direct', path: '/artifacts/{seed.historicalArtifactId}' },
        waitFor: '#root',
        actions: [
          { type: 'waitForText', text: 'Overview' },
          { type: 'waitForText', text: 'URI' },
        ],
      },
      head: {
        path: '/#/artifacts/{seed.historicalArtifactId}',
        routeExpectation: { kind: 'direct', path: '/artifacts/{seed.historicalArtifactId}' },
        waitFor: '#root',
        actions: [{ type: 'waitForText', text: 'Artifact details' }],
      },
    },
  },
]);

function resolveTemplates(value, seedValues) {
  if (typeof value === 'string') {
    return value.replace(/\{seed\.([a-zA-Z0-9_]+)\}/g, (match, key) => {
      const replacement = seedValues?.[key];
      return replacement === undefined || replacement === null || replacement === ''
        ? match
        : String(replacement);
    });
  }
  if (Array.isArray(value)) return value.map((entry) => resolveTemplates(entry, seedValues));
  if (!value || typeof value !== 'object') return value;
  return Object.fromEntries(
    Object.entries(value).map(([key, entry]) => [key, resolveTemplates(entry, seedValues)]),
  );
}

function resolveSemanticScenarios(revisionRole, seedValues, scenarios = SEMANTIC_SCENARIOS) {
  if (revisionRole !== 'base' && revisionRole !== 'head') {
    throw new Error('Semantic scenarios require revisionRole base or head.');
  }
  return scenarios.map((scenario) => {
    const variant = scenario.revisions?.[revisionRole];
    if (!variant) throw new Error(`Scenario ${scenario.key} has no ${revisionRole} definition.`);
    const missingFixtures = (scenario.requires || []).filter((key) => !seedValues?.[key]);
    return resolveTemplates(
      {
        ...variant,
        expectedChange: scenario.expectedChange || null,
        missingFixtures,
        name: scenario.key,
        required: scenario.required !== false,
        scenarioContractSchemaVersion: SCENARIO_CONTRACT_SCHEMA_VERSION,
        scenarioTitle: scenario.title,
        semanticScenario: scenario.key,
      },
      seedValues,
    );
  });
}

function getSemanticScenarioCatalog(scenarios = SEMANTIC_SCENARIOS) {
  return scenarios.map((scenario) => ({
    expectedChange: scenario.expectedChange || null,
    required: scenario.required !== false,
    scenarioTitle: scenario.title,
    semanticScenario: scenario.key,
  }));
}

module.exports = {
  EXPECTED_CHANGES,
  SCENARIO_CONTRACT_SCHEMA_VERSION,
  SEMANTIC_SCENARIOS,
  getSemanticScenarioCatalog,
  resolveSemanticScenarios,
};
