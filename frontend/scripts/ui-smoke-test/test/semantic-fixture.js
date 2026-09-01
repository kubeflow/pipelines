'use strict';

const {
  ARTIFACT_FIXTURES,
  REVISION_FLAVORS,
  RUN_RESOURCE_DEFINITIONS,
  RUN_PROFILES,
  SEMANTIC_RESOURCE_DEFINITIONS,
  TASK_FIXTURES,
  buildLogicalFixtures,
} = require('../semantic-manifest');

function generatedValue(role, runKey, suffix) {
  return `${role}-${runKey.replaceAll('.', '-')}-${suffix}`;
}

function artifactBinding(role, runKey, artifactKey, rich) {
  const scalarMetricKeys = Object.keys(ARTIFACT_FIXTURES['artifact.scalar-metrics'].members);
  const artifactIds =
    artifactKey === 'artifact.scalar-metrics' && role === 'head'
      ? scalarMetricKeys.map((metricKey) =>
          generatedValue(role, runKey, metricKey.replace('metric.', 'artifact-metric-')),
        )
      : [generatedValue(role, runKey, artifactKey.replace('artifact.', 'artifact-'))];
  const uri = `s3://ui-smoke/${role}/${runKey}/${artifactKey}`;
  const binding = {
    artifactIds,
    records: artifactIds.map((artifactId, index) => ({
      artifactId,
      uri: artifactIds.length === 1 ? uri : `${uri}/${scalarMetricKeys[index]}`,
    })),
  };
  if (artifactKey === 'artifact.html-report' || artifactKey === 'artifact.markdown-report') {
    const definition = ARTIFACT_FIXTURES[artifactKey];
    binding.files = [
      {
        artifactId: artifactIds[0],
        name: definition.portKey,
        type: artifactKey === 'artifact.html-report' ? 'system.HTML' : 'system.Markdown',
        uri,
      },
    ];
  }
  if (artifactKey === 'artifact.scalar-metrics') {
    binding.members = Object.fromEntries(
      Object.entries(ARTIFACT_FIXTURES[artifactKey].members).map(
        ([metricKey, definition], index) => [
          metricKey,
          {
            artifactIds: [role === 'head' ? artifactIds[index] : artifactIds[0]],
            name: definition.name,
            numberValue: definition.value,
          },
        ],
      ),
    );
    if (rich) {
      binding.consumers = {
        'task.consume-metrics': { artifactIds: [...artifactIds] },
      };
    }
  }
  if (artifactKey === 'artifact.roc-curve') {
    binding.points = structuredClone(ARTIFACT_FIXTURES[artifactKey].points);
  }
  return binding;
}

function taskInstance(role, runKey, taskKey, index = 0) {
  const kind = TASK_FIXTURES[taskKey].kind;
  const instance = {
    displayName: taskKey.replace('task.', ''),
    taskId: generatedValue(role, runKey, `${taskKey.replace('task.', 'task-')}-${index}`),
  };
  if (role === 'base' && taskKey !== 'task.loop-worker') {
    instance.mlmdExecutionId = generatedValue(
      role,
      runKey,
      `${taskKey.replace('task.', 'execution-')}-${index}`,
    );
  } else {
    instance.type = { dag: '8', loop: '5', runtime: '2' }[kind];
  }
  return instance;
}

function executorLogGroup(role, runKey, taskKey, taskIndex, attemptCount) {
  const taskName = taskKey.replace('task.', '');
  return {
    artifacts: Array.from({ length: attemptCount }, (_, attemptIndex) => ({
      artifactId: generatedValue(
        role,
        runKey,
        `${taskName}-${taskIndex}-executor-log-${attemptIndex}`,
      ),
      name: 'executor-logs',
      type: 'Artifact',
      uri: `s3://ui-smoke/${role}/${runKey}/${taskName}-${taskIndex}/executor-logs-${attemptIndex}`,
    })),
    key: 'executor-logs',
  };
}

function runBinding(role, definition) {
  const rich = definition.fixtureProfile === 'rich-topology';
  const runKey = definition.semanticKey;
  const taskInstances = {
    'task.write-metrics': [taskInstance(role, runKey, 'task.write-metrics')],
  };
  const relationships = [];
  if (rich) {
    Object.assign(taskInstances, {
      'task.consume-metrics': [taskInstance(role, runKey, 'task.consume-metrics')],
      'task.loop-worker': [
        {
          ...taskInstance(role, runKey, 'task.loop-worker', 0),
          ...(role === 'head' ? { iterationIndex: 0 } : {}),
        },
        {
          ...taskInstance(role, runKey, 'task.loop-worker', 1),
          ...(role === 'head' ? { iterationIndex: 1 } : {}),
        },
      ],
      'task.nested-dag': [taskInstance(role, runKey, 'task.nested-dag')],
      'task.nested-worker': [taskInstance(role, runKey, 'task.nested-worker')],
      'task.parallel-loop': [taskInstance(role, runKey, 'task.parallel-loop')],
      'task.retry-once': [taskInstance(role, runKey, 'task.retry-once')],
    });
    const retry = taskInstances['task.retry-once'][0];
    if (role === 'base') {
      retry.failedMainJobs = ['retry-attempt-0'];
    } else {
      retry.podBindings = [
        {
          name: 'retry-attempt-0',
          type: 'EXECUTOR',
          uid: '11111111-aaaa-bbbb-cccc-111111111111',
        },
        {
          name: 'retry-attempt-1',
          type: 'EXECUTOR',
          uid: '22222222-aaaa-bbbb-cccc-222222222222',
        },
      ];
      retry.executorPods = retry.podBindings.map((pod) => pod.name);
    }
    relationships.push(
      ...structuredClone(RUN_PROFILES['rich-topology'].relationships).map((relationship) => ({
        ...relationship,
        evidence:
          role === 'head'
            ? 'native-task-api'
            : relationship.kind === 'depends-on'
              ? 'pipeline-version-spec'
              : relationship.kind === 'contains'
                ? 'mlmd-parent-dag'
                : 'mlmd-event',
      })),
    );
  }
  const artifacts = Object.fromEntries(
    Object.keys(ARTIFACT_FIXTURES).map((artifactKey) => [
      artifactKey,
      artifactBinding(role, runKey, artifactKey, rich),
    ]),
  );
  for (const [taskKey, instances] of Object.entries(taskInstances)) {
    for (const [taskIndex, instance] of instances.entries()) {
      const inputs = [];
      const outputs = [];
      for (const artifactKey of RUN_PROFILES[definition.fixtureProfile].artifacts) {
        const artifactDefinition = ARTIFACT_FIXTURES[artifactKey];
        if (artifactDefinition.consumerTask === taskKey) {
          inputs.push({
            artifacts: structuredClone(artifacts[artifactKey].records),
            key: artifactDefinition.consumerPortKey,
          });
        }
        if (artifactDefinition.producerTask === taskKey) {
          outputs.push({
            artifacts: structuredClone(artifacts[artifactKey].records),
            key: artifactDefinition.portKey,
          });
        }
      }
      if (role === 'head' && TASK_FIXTURES[taskKey].kind === 'runtime') {
        outputs.push(
          executorLogGroup(role, runKey, taskKey, taskIndex, taskKey === 'task.retry-once' ? 2 : 1),
        );
      }
      if (inputs.length > 0 || outputs.length > 0) {
        instance.artifactReferences = { inputs, outputs };
      }
    }
  }
  const executionInstances =
    role === 'base'
      ? Object.fromEntries(
          Object.entries(taskInstances).map(([taskKey, instances]) => [
            taskKey,
            taskKey === 'task.retry-once'
              ? [
                  {
                    executionId: instances[0].mlmdExecutionId,
                    executionRole: 'task',
                    executorLogs: executorLogGroup(role, runKey, taskKey, 0, 2).artifacts,
                    state: 'COMPLETE',
                  },
                ]
              : taskKey === 'task.parallel-loop'
                ? [
                    {
                      executionId: instances[0].mlmdExecutionId,
                      executionRole: 'loop-controller',
                      executorLogs: [],
                      state: 'COMPLETE',
                    },
                    ...RUN_PROFILES['rich-topology'].loop.iterationIndexes.map(
                      (iterationIndex) => ({
                        executionId: generatedValue(
                          role,
                          runKey,
                          `execution-parallel-loop-iteration-${iterationIndex}`,
                        ),
                        executionRole: 'loop-iteration',
                        executorLogs: [],
                        iterationIndex,
                        parentDagId: instances[0].mlmdExecutionId,
                        state: 'COMPLETE',
                      }),
                    ),
                  ]
                : instances.map((instance, index) => ({
                    executionId:
                      instance.mlmdExecutionId ||
                      generatedValue(
                        role,
                        runKey,
                        `${taskKey.replace('task.', 'execution-')}-${index}`,
                      ),
                    executionRole: 'task',
                    executorLogs:
                      TASK_FIXTURES[taskKey].kind === 'runtime'
                        ? executorLogGroup(role, runKey, taskKey, index, 1).artifacts
                        : [],
                    ...(taskKey !== 'task.loop-worker'
                      ? {}
                      : {
                          iterationIndex: index,
                          parentDagId: generatedValue(
                            role,
                            runKey,
                            `execution-parallel-loop-iteration-${index}`,
                          ),
                        }),
                    state: 'COMPLETE',
                  })),
          ]),
        )
      : {};
  if (role === 'base') {
    executionInstances['execution.unclassified'] = [
      {
        executionId: generatedValue(role, runKey, 'execution-root'),
        executionRole: 'run-root',
        executorLogs: [],
        name: `run/${generatedValue(role, runKey, 'run-id')}`,
        state: 'COMPLETE',
      },
    ];
  }
  const scopeInstances =
    role === 'head' && rich
      ? {
          'task.parallel-loop': RUN_PROFILES['rich-topology'].loop.iterationIndexes.map(
            (iterationIndex) => ({
              iterationIndex,
              parentTaskId: taskInstances['task.parallel-loop'][0].taskId,
              taskId: generatedValue(
                role,
                runKey,
                `task-parallel-loop-iteration-${iterationIndex}`,
              ),
              type: '8',
            }),
          ),
        }
      : {};
  if (role === 'head' && rich) {
    for (const [iterationIndex, worker] of taskInstances['task.loop-worker'].entries()) {
      worker.parentTaskId = scopeInstances['task.parallel-loop'][iterationIndex].taskId;
    }
  }
  return {
    artifacts,
    displayName: definition.displayName,
    executionInstances,
    fixtureProfile: definition.fixtureProfile,
    lineageComplete: role === 'base',
    relationships,
    revisionFlavor: role === 'base' ? REVISION_FLAVORS.LEGACY : REVISION_FLAVORS.NATIVE,
    runId: generatedValue(role, runKey, 'run-id'),
    scopeInstances,
    taskInstances,
  };
}

function resourceBindings(role, runs) {
  const kinds = {
    experiments: 'experiment',
    pipelines: 'pipeline',
    recurringRuns: 'recurring-run',
    runs: 'run',
  };
  const resources = {};
  for (const [definitionKind, definitions] of Object.entries(SEMANTIC_RESOURCE_DEFINITIONS)) {
    for (const definition of definitions) {
      const semanticKey = definition.semanticKey;
      const resource = {
        displayName: definition.displayName || definition.name,
        id:
          definitionKind === 'runs'
            ? runs[semanticKey].runId
            : generatedValue(role, semanticKey, `${kinds[definitionKind]}-id`),
        kind: kinds[definitionKind],
      };
      if (definition.fixtureProfile) resource.fixtureProfile = definition.fixtureProfile;
      if (definition.pipelineSemanticKey) resource.pipeline = definition.pipelineSemanticKey;
      resources[semanticKey] = resource;
      if (definitionKind === 'pipelines') {
        resources[`${semanticKey}.version`] = {
          displayName: definition.displayName || definition.name,
          fixtureProfile: definition.fixtureProfile,
          id: generatedValue(role, semanticKey, 'pipeline-version-id'),
          kind: 'pipeline-version',
          pipeline: semanticKey,
        };
      }
    }
  }
  return resources;
}

function strictSemanticFixtureManifest() {
  const logical = buildLogicalFixtures(SEMANTIC_RESOURCE_DEFINITIONS);
  const deployments = {};
  for (const role of ['base', 'head']) {
    const runs = {};
    for (const definition of RUN_RESOURCE_DEFINITIONS) {
      const run = runBinding(role, definition);
      runs[definition.semanticKey] = run;
    }
    deployments[role] = {
      bindings: { resources: resourceBindings(role, runs), runs },
      revisionFlavor: role === 'base' ? REVISION_FLAVORS.LEGACY : REVISION_FLAVORS.NATIVE,
      validation: { errors: [], valid: true },
    };
  }
  return {
    deployments,
    fixtureSet: 'ui-smoke-deterministic-v3',
    logical,
    schemaVersion: 'ui-smoke-semantic/v3',
  };
}

module.exports = { strictSemanticFixtureManifest };
