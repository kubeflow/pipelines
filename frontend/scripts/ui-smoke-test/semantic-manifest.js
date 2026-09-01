'use strict';

const SEMANTIC_SCHEMA_VERSION = 'ui-smoke-semantic/v3';
const SEMANTIC_FIXTURE_SET = 'ui-smoke-deterministic-v3';
const DEFAULT_RUN_PROFILE = 'metrics';
const COMPARISON_RUN_FIXTURES = Object.freeze([
  'run.training-1',
  'run.training-2',
  'run.evaluation',
]);
const RUN_RESOURCE_DEFINITIONS = Object.freeze([
  Object.freeze({
    semanticKey: 'run.training-1',
    displayName: 'UI Smoke Training Run 1',
    fixtureProfile: 'rich-topology',
    pipelineSemanticKey: 'pipeline.training',
  }),
  Object.freeze({
    semanticKey: 'run.training-2',
    displayName: 'UI Smoke Training Run 2',
    fixtureProfile: 'metrics',
    pipelineSemanticKey: 'pipeline.data-ingestion',
  }),
  Object.freeze({
    semanticKey: 'run.evaluation',
    displayName: 'UI Smoke Evaluation Run',
    fixtureProfile: 'metrics',
    pipelineSemanticKey: 'pipeline.model-evaluation',
  }),
  Object.freeze({
    semanticKey: 'run.inference',
    displayName: 'UI Smoke Inference Run',
    fixtureProfile: 'metrics',
    pipelineSemanticKey: 'pipeline.data-ingestion',
  }),
  Object.freeze({
    semanticKey: 'run.data-processing',
    displayName: 'UI Smoke Data Processing Run',
    fixtureProfile: 'metrics',
    pipelineSemanticKey: 'pipeline.model-evaluation',
  }),
]);
const SEMANTIC_RESOURCE_DEFINITIONS = Object.freeze({
  experiments: Object.freeze([
    Object.freeze({
      semanticKey: 'experiment.image-classification',
      displayName: 'UI Smoke - Image Classification',
      description: 'Deterministic UI smoke-test experiment',
    }),
    Object.freeze({
      semanticKey: 'experiment.natural-language-processing',
      displayName: 'UI Smoke - Natural Language Processing',
      description: 'Second deterministic UI smoke-test experiment',
    }),
  ]),
  pipelines: Object.freeze([
    Object.freeze({
      semanticKey: 'pipeline.training',
      name: 'ui-smoke-training-pipeline',
      displayName: 'UI Smoke Training Pipeline',
      description: 'Deterministic training pipeline for UI screenshots',
      fixtureProfile: 'rich-topology',
    }),
    Object.freeze({
      semanticKey: 'pipeline.data-ingestion',
      name: 'ui-smoke-data-ingestion',
      displayName: 'UI Smoke Data Ingestion',
      description: 'Deterministic data-ingestion pipeline for UI screenshots',
      fixtureProfile: 'metrics',
    }),
    Object.freeze({
      semanticKey: 'pipeline.model-evaluation',
      name: 'ui-smoke-model-evaluation',
      displayName: 'UI Smoke Model Evaluation',
      description: 'Deterministic evaluation pipeline for UI screenshots',
      fixtureProfile: 'metrics',
    }),
  ]),
  recurringRuns: Object.freeze([
    Object.freeze({
      semanticKey: 'recurring-run.daily-training',
      displayName: 'UI Smoke Daily Training',
    }),
    Object.freeze({
      semanticKey: 'recurring-run.hourly-data-sync',
      displayName: 'UI Smoke Hourly Data Sync',
    }),
  ]),
  runs: RUN_RESOURCE_DEFINITIONS,
});
const REVISION_FLAVORS = Object.freeze({
  LEGACY: 'legacy-mlmd',
  NATIVE: 'native-task-artifact',
  UNKNOWN: 'unknown',
});

const TASK_FIXTURES = Object.freeze({
  'task.consume-metrics': Object.freeze({
    kind: 'runtime',
    names: Object.freeze(['consume-metrics', 'consume metrics']),
  }),
  'task.loop-worker': Object.freeze({
    kind: 'runtime',
    names: Object.freeze(['loop-worker', 'loop worker']),
  }),
  'task.nested-dag': Object.freeze({
    kind: 'dag',
    names: Object.freeze(['nested-dag', 'nested dag']),
  }),
  'task.nested-worker': Object.freeze({
    kind: 'runtime',
    names: Object.freeze(['nested-worker', 'nested worker']),
  }),
  'task.parallel-loop': Object.freeze({
    kind: 'loop',
    names: Object.freeze(['parallel-loop', 'parallel loop']),
  }),
  'task.retry-once': Object.freeze({
    kind: 'runtime',
    names: Object.freeze(['retry-once', 'retry once']),
  }),
  'task.write-metrics': Object.freeze({
    kind: 'runtime',
    names: Object.freeze(['write-metrics', 'write metrics']),
  }),
});

const ARTIFACT_FIXTURES = Object.freeze({
  'artifact.html-report': Object.freeze({
    contentMarker: 'UI Smoke HTML Report',
    kind: 'html',
    portKey: 'html_report',
    producerTask: 'task.write-metrics',
  }),
  'artifact.markdown-report': Object.freeze({
    contentMarker: 'UI Smoke Markdown Report',
    kind: 'markdown',
    portKey: 'markdown_report',
    producerTask: 'task.write-metrics',
  }),
  'artifact.scalar-metrics': Object.freeze({
    kind: 'metrics',
    members: Object.freeze({
      'metric.accuracy': Object.freeze({ name: 'accuracy', value: 0.92 }),
      'metric.loss': Object.freeze({ name: 'loss', value: 0.08 }),
    }),
    consumerPortKey: 'metrics',
    consumerTask: 'task.consume-metrics',
    portKey: 'scalar_metrics',
    producerTask: 'task.write-metrics',
  }),
  'artifact.roc-curve': Object.freeze({
    kind: 'classification-metrics',
    points: Object.freeze([
      Object.freeze({ confidenceThreshold: 1, falsePositiveRate: 0, recall: 0 }),
      Object.freeze({ confidenceThreshold: 0.8, falsePositiveRate: 0.08, recall: 0.35 }),
      Object.freeze({ confidenceThreshold: 0.5, falsePositiveRate: 0.22, recall: 0.72 }),
      Object.freeze({ confidenceThreshold: 0.2, falsePositiveRate: 0.55, recall: 0.9 }),
      Object.freeze({ confidenceThreshold: 0, falsePositiveRate: 1, recall: 1 }),
    ]),
    portKey: 'roc_curve',
    producerTask: 'task.write-metrics',
  }),
});

const RUN_PROFILES = Object.freeze({
  metrics: Object.freeze({
    artifacts: Object.freeze([
      'artifact.html-report',
      'artifact.markdown-report',
      'artifact.roc-curve',
      'artifact.scalar-metrics',
    ]),
    tasks: Object.freeze({ 'task.write-metrics': 1 }),
  }),
  'rich-topology': Object.freeze({
    artifacts: Object.freeze([
      'artifact.html-report',
      'artifact.markdown-report',
      'artifact.roc-curve',
      'artifact.scalar-metrics',
    ]),
    loop: Object.freeze({
      iterationIndexes: Object.freeze([0, 1]),
      iterations: 2,
      task: 'task.parallel-loop',
      worker: 'task.loop-worker',
    }),
    relationships: Object.freeze([
      Object.freeze({
        kind: 'artifact-consumer',
        source: 'artifact.scalar-metrics',
        target: 'task.consume-metrics',
      }),
      Object.freeze({
        kind: 'contains',
        source: 'task.nested-dag',
        target: 'task.nested-worker',
      }),
      Object.freeze({
        kind: 'contains',
        source: 'task.parallel-loop',
        target: 'task.loop-worker',
      }),
      Object.freeze({
        kind: 'depends-on',
        source: 'task.consume-metrics',
        target: 'task.nested-dag',
      }),
      Object.freeze({
        kind: 'depends-on',
        source: 'task.parallel-loop',
        target: 'task.nested-dag',
      }),
      Object.freeze({
        kind: 'depends-on',
        source: 'task.retry-once',
        target: 'task.nested-dag',
      }),
      Object.freeze({
        kind: 'depends-on',
        source: 'task.write-metrics',
        target: 'task.consume-metrics',
      }),
    ]),
    retry: Object.freeze({ attempts: 2, task: 'task.retry-once' }),
    tasks: Object.freeze({
      'task.consume-metrics': 1,
      'task.loop-worker': 2,
      'task.nested-dag': 1,
      'task.nested-worker': 1,
      'task.parallel-loop': 1,
      'task.retry-once': 1,
      'task.write-metrics': 1,
    }),
  }),
});

function hasOwn(object, key) {
  return object !== null && typeof object === 'object' && Object.hasOwn(object, key);
}

function field(object, ...names) {
  for (const name of names.flat()) {
    if (hasOwn(object, name)) return object[name];
  }
  return undefined;
}

function arrayField(object, ...names) {
  const value = field(object, names);
  return Array.isArray(value) ? value : [];
}

function stringValue(value) {
  if (value === undefined || value === null || value === '') return null;
  return String(value);
}

function sortedUnique(values) {
  return [...new Set(values.map(stringValue).filter(Boolean))].sort();
}

function normalizedName(value) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[_\s]+/g, '-')
    .replace(/[^a-z0-9-]+/g, '')
    .replace(/-+/g, '-');
}

function unwrapRun(response) {
  let run = response;
  for (let depth = 0; depth < 2; depth++) {
    const nested = field(run, 'run');
    if (!nested || typeof nested !== 'object') break;
    run = nested;
  }
  return run && typeof run === 'object' ? run : {};
}

function detectRevisionFlavor(response) {
  const run = unwrapRun(response);
  if (hasOwn(run, 'tasks') || hasOwn(run, 'task_count') || hasOwn(run, 'taskCount')) {
    return REVISION_FLAVORS.NATIVE;
  }

  const runDetails = field(run, 'run_details', 'runDetails');
  if (
    runDetails &&
    (hasOwn(runDetails, 'task_details') ||
      hasOwn(runDetails, 'taskDetails') ||
      hasOwn(runDetails, 'pipeline_context_id') ||
      hasOwn(runDetails, 'pipelineContextId'))
  ) {
    return REVISION_FLAVORS.LEGACY;
  }
  return REVISION_FLAVORS.UNKNOWN;
}

function tasksForRun(response, flavor = detectRevisionFlavor(response)) {
  const run = unwrapRun(response);
  if (flavor === REVISION_FLAVORS.NATIVE) return arrayField(run, 'tasks');
  if (flavor === REVISION_FLAVORS.LEGACY) {
    return arrayField(field(run, 'run_details', 'runDetails'), 'task_details', 'taskDetails');
  }
  return [];
}

function taskCandidateSemanticKey(task) {
  const candidates = [field(task, 'name'), field(task, 'display_name', 'displayName')].map(
    normalizedName,
  );
  for (const [semanticKey, fixture] of Object.entries(TASK_FIXTURES)) {
    const fixtureNames = fixture.names.map(normalizedName);
    if (candidates.some((candidate) => fixtureNames.includes(candidate))) return semanticKey;
  }
  return null;
}

function taskSemanticKey(task) {
  const semanticKey = taskCandidateSemanticKey(task);
  if (!semanticKey) return null;

  const type = field(task, 'type');
  if (
    type !== undefined &&
    type !== null &&
    !nativeTaskTypeMatches(type, TASK_FIXTURES[semanticKey]?.kind)
  ) {
    return null;
  }
  return semanticKey;
}

function iterationIndex(task) {
  const attributes = field(task, 'type_attributes', 'typeAttributes') || {};
  const value =
    field(attributes, 'iteration_index', 'iterationIndex') ??
    field(task, 'iteration_index', 'iterationIndex');
  return value === undefined || value === null ? null : Number(value);
}

function iterationCount(task) {
  const attributes = field(task, 'type_attributes', 'typeAttributes') || {};
  const value = field(attributes, 'iteration_count', 'iterationCount');
  return value === undefined || value === null ? null : Number(value);
}

function nativePodRole(value) {
  const normalized = String(value || '').toUpperCase();
  if (normalized === '1' || normalized === 'DRIVER') return 'DRIVER';
  if (normalized === '2' || normalized === 'EXECUTOR') return 'EXECUTOR';
  return normalized || null;
}

function taskPods(task, type) {
  return arrayField(task, 'pods')
    .filter((pod) => nativePodRole(field(pod, 'type')) === type)
    .map((pod) => stringValue(field(pod, 'name')))
    .filter(Boolean)
    .sort();
}

function taskPodBindings(task) {
  return arrayField(task, 'pods')
    .map((pod) =>
      Object.fromEntries(
        Object.entries({
          name: stringValue(field(pod, 'name')),
          type: nativePodRole(field(pod, 'type')),
          uid: stringValue(field(pod, 'uid')),
        }).filter(([, value]) => value !== null),
      ),
    )
    .filter((pod) => pod.name || pod.uid)
    .sort((left, right) => {
      const roleOrder = { DRIVER: 0, EXECUTOR: 1 };
      return (
        (roleOrder[left.type] ?? 2) - (roleOrder[right.type] ?? 2) ||
        String(left.name || '').localeCompare(String(right.name || '')) ||
        String(left.uid || '').localeCompare(String(right.uid || ''))
      );
    });
}

function normalizedArtifactReferenceGroups(task, flavor, hydratedArtifacts, direction) {
  const normalizeArtifacts = (artifacts, groupKey) => {
    if (
      flavor === REVISION_FLAVORS.NATIVE &&
      direction === 'outputs' &&
      groupKey === 'executor-logs'
    ) {
      const entries = artifacts.map((artifact) => ({
        artifact,
        attemptIndex: executorLogAttemptIndex(field(artifact, 'uri')),
      }));
      if (
        entries.some((entry) => entry.attemptIndex === null) ||
        new Set(entries.map((entry) => entry.attemptIndex)).size !== entries.length
      ) {
        throw new Error(
          'executor-logs artifact references must have distinct executor-logs-N URI leaves.',
        );
      }
      return entries
        .sort((left, right) => left.attemptIndex - right.attemptIndex)
        .map(({ artifact, attemptIndex }, index) => {
          if (attemptIndex !== index) {
            throw new Error(
              'executor-logs artifact references must use contiguous attempt indexes starting at 0.',
            );
          }
          return artifactFileBinding(artifact);
        });
    }
    const entries = artifacts.map((artifact) => ({
      artifact,
      semanticOrderKey: stableStringify({
        metadata: artifactMetadata(artifact),
        name: stringValue(field(artifact, 'name')),
        numberValue: field(artifact, 'number_value', 'numberValue'),
        type: stringValue(field(artifact, 'type')),
      }),
    }));
    const orderKeys = entries.map((entry) => entry.semanticOrderKey);
    if (new Set(orderKeys).size !== orderKeys.length) {
      throw new Error(
        `Artifact reference group ${groupKey} contains records without distinct semantic ordering keys.`,
      );
    }
    return entries
      .sort((left, right) => left.semanticOrderKey.localeCompare(right.semanticOrderKey))
      .map(({ artifact }) => artifactFileBinding(artifact));
  };
  const normalizeGroups = (groups) => {
    const normalized = groups
      .filter((group) => group.artifacts.length > 0)
      .map((group) => ({ ...group, normalizedKey: normalizedName(group.key) || 'artifact' }))
      .sort(
        (left, right) =>
          left.normalizedKey.localeCompare(right.normalizedKey) ||
          left.key.localeCompare(right.key),
      );
    if (new Set(normalized.map((group) => group.normalizedKey)).size !== normalized.length) {
      throw new Error('Artifact reference groups contain duplicate normalized semantic keys.');
    }
    return normalized.map(({ artifacts, key }) => ({
      artifacts: normalizeArtifacts(artifacts, key),
      key,
    }));
  };

  if (flavor === REVISION_FLAVORS.NATIVE) {
    return normalizeGroups(
      nativeArtifactGroups(task, direction).map((group) => ({
        artifacts: arrayField(group, 'artifacts'),
        key: stringValue(field(group, 'artifact_key', 'artifactKey', 'key')) || 'artifact',
      })),
    );
  }
  if (flavor === REVISION_FLAVORS.LEGACY) {
    const io = field(task, direction) || {};
    return normalizeGroups(
      Object.entries(io).map(([key, group]) => ({
        artifacts: artifactsByIds(
          hydratedArtifacts,
          sortedUnique(arrayField(group, 'artifact_ids', 'artifactIds')),
        ),
        key,
      })),
    );
  }
  return [];
}

function taskArtifactReferences(task, flavor, hydratedArtifacts) {
  const inputs = normalizedArtifactReferenceGroups(task, flavor, hydratedArtifacts, 'inputs');
  const outputs = normalizedArtifactReferenceGroups(task, flavor, hydratedArtifacts, 'outputs');
  return inputs.length || outputs.length ? { inputs, outputs } : null;
}

function legacyFailedMainJobs(task) {
  const executor = field(task, 'executor_detail', 'executorDetail') || {};
  return sortedUnique(arrayField(executor, 'failed_main_jobs', 'failedMainJobs'));
}

function taskChildReferences(task) {
  return arrayField(task, 'child_tasks', 'childTasks').map((child) => ({
    name: stringValue(field(child, 'name')),
    podName: stringValue(field(child, 'pod_name', 'podName')),
    taskId: stringValue(field(child, 'task_id', 'taskId', 'id')),
  }));
}

function normalizeTaskBinding(task, flavor, hydratedArtifacts = []) {
  const binding = {
    childTaskReferences: taskChildReferences(task),
    displayName: stringValue(field(task, 'display_name', 'displayName')),
    name: stringValue(field(task, 'name')),
    parentTaskId: stringValue(field(task, 'parent_task_id', 'parentTaskId')),
    state: stringValue(field(task, 'state')),
    taskId: stringValue(field(task, 'task_id', 'taskId', 'id')),
  };

  if (flavor === REVISION_FLAVORS.LEGACY) {
    binding.failedMainJobs = legacyFailedMainJobs(task);
    binding.iterationIndex = iterationIndex(task);
    const executionId = stringValue(field(task, 'execution_id', 'executionId'));
    binding.mlmdExecutionId = /^[1-9]\d*$/.test(executionId || '') ? executionId : null;
    binding.podName = stringValue(field(task, 'pod_name', 'podName'));
  } else if (flavor === REVISION_FLAVORS.NATIVE) {
    binding.executorPods = taskPods(task, 'EXECUTOR');
    binding.iterationCount = iterationCount(task);
    binding.iterationIndex = iterationIndex(task);
    binding.scopePath = stringValue(field(task, 'scope_path', 'scopePath'));
    binding.type = stringValue(field(task, 'type'));
    binding.podBindings = taskPodBindings(task);
  }
  binding.artifactReferences = taskArtifactReferences(task, flavor, hydratedArtifacts);
  return Object.fromEntries(Object.entries(binding).filter(([, value]) => value !== null));
}

function legacyArtifactIds(task, portKey, direction = 'outputs') {
  const io = field(task, direction) || {};
  const artifactList = field(
    io,
    portKey,
    portKey.replace(/_([a-z])/g, (_, c) => c.toUpperCase()),
  );
  if (!artifactList || typeof artifactList !== 'object') return [];
  return sortedUnique(arrayField(artifactList, 'artifact_ids', 'artifactIds'));
}

function nativeArtifactGroups(task, direction = 'outputs') {
  const io = field(task, direction) || {};
  const groups = arrayField(io, 'artifacts');
  if (groups.length > 0) return groups;

  // Some generated clients expose output artifact groups as a keyed object.
  return Object.entries(io)
    .filter(([, value]) => value && typeof value === 'object')
    .map(([key, value]) => ({ artifact_key: key, ...value }));
}

function nativeArtifacts(task, portKey, direction = 'outputs') {
  const camelPortKey = portKey.replace(/_([a-z])/g, (_, c) => c.toUpperCase());
  return nativeArtifactGroups(task, direction).flatMap((group) => {
    const key = field(group, 'artifact_key', 'artifactKey', 'key');
    return key === portKey || key === camelPortKey ? arrayField(group, 'artifacts') : [];
  });
}

function nativeArtifactId(artifact) {
  return stringValue(field(artifact, 'artifact_id', 'artifactId', 'id'));
}

function artifactFileBinding(artifact) {
  const metadata = artifactMetadata(artifact);
  return {
    artifactId: nativeArtifactId(artifact),
    name:
      stringValue(field(artifact, 'name')) ||
      stringValue(field(metadata, 'display_name', 'displayName')),
    type: stringValue(field(artifact, 'type')),
    uri: stringValue(field(artifact, 'uri')),
  };
}

function artifactMetadata(artifact) {
  const metadata = field(artifact, 'metadata', 'custom_properties', 'customProperties');
  return metadata && typeof metadata === 'object' ? metadata : {};
}

function artifactsByIds(artifacts, artifactIds) {
  const expectedIds = new Set(artifactIds);
  return artifacts.filter((artifact) => expectedIds.has(nativeArtifactId(artifact)));
}

function metricMemberKey(name) {
  const normalized = normalizedName(name);
  return normalized ? `metric.${normalized}` : null;
}

function buildMetricMembers(artifacts) {
  const members = {};
  for (const artifact of artifacts) {
    const artifactId = nativeArtifactId(artifact);
    const metricName = stringValue(field(artifact, 'name'));
    const metricKey = metricMemberKey(metricName);
    if (metricKey) {
      members[metricKey] = {
        artifactIds: artifactId ? [artifactId] : [],
        name: metricName,
        numberValue: field(artifact, 'number_value', 'numberValue'),
      };
    }

    const metadata = artifactMetadata(artifact);
    for (const [semanticKey, definition] of Object.entries(
      ARTIFACT_FIXTURES['artifact.scalar-metrics'].members,
    )) {
      if (!hasOwn(metadata, definition.name)) continue;
      members[semanticKey] = {
        artifactIds: artifactId ? [artifactId] : [],
        name: definition.name,
        numberValue: metadata[definition.name],
      };
    }
  }
  return sortObject(members);
}

function normalizeRocPoint(point) {
  return {
    confidenceThreshold: field(point, 'confidence_threshold', 'confidenceThreshold'),
    falsePositiveRate: field(point, 'false_positive_rate', 'falsePositiveRate'),
    recall: field(point, 'recall', 'true_positive_rate', 'truePositiveRate', 'tpr'),
  };
}

function buildRocPoints(artifacts) {
  return artifacts.flatMap((artifact) => {
    const metadata = artifactMetadata(artifact);
    const points = field(metadata, 'confidence_metrics', 'confidenceMetrics');
    return Array.isArray(points) ? points.map(normalizeRocPoint) : [];
  });
}

function buildArtifactBinding(tasks, flavor, definition, hydratedArtifacts = []) {
  const preferredTasks = tasks.filter((task) => taskSemanticKey(task) === definition.producerTask);
  const sourceTasks = preferredTasks.length > 0 ? preferredTasks : tasks;
  const consumerTasks = definition.consumerTask
    ? tasks.filter((task) => taskSemanticKey(task) === definition.consumerTask)
    : [];

  if (flavor === REVISION_FLAVORS.LEGACY) {
    const artifactIds = sortedUnique(
      sourceTasks.flatMap((task) => legacyArtifactIds(task, definition.portKey)),
    );
    const binding = {
      artifactIds,
      portKey: definition.portKey,
      storage: REVISION_FLAVORS.LEGACY,
    };
    const artifacts = artifactsByIds(hydratedArtifacts, artifactIds);
    const records = artifacts
      .map(artifactFileBinding)
      .sort((left, right) => String(left.artifactId).localeCompare(String(right.artifactId)));
    if (records.length > 0) binding.records = records;
    if (definition.kind === 'html' || definition.kind === 'markdown') {
      binding.files = records;
    }
    if (definition.consumerTask) {
      binding.consumers = {
        [definition.consumerTask]: {
          artifactIds: sortedUnique(
            consumerTasks.flatMap((task) =>
              legacyArtifactIds(task, definition.consumerPortKey, 'inputs'),
            ),
          ),
          portKey: definition.consumerPortKey,
        },
      };
    }
    if (definition.kind === 'metrics') binding.members = buildMetricMembers(artifacts);
    if (definition.kind === 'classification-metrics') binding.points = buildRocPoints(artifacts);
    return binding;
  }

  if (flavor === REVISION_FLAVORS.NATIVE) {
    const artifacts = sourceTasks.flatMap((task) => nativeArtifacts(task, definition.portKey));
    const artifactIds = sortedUnique(artifacts.map(nativeArtifactId));
    const binding = {
      artifactIds,
      portKey: definition.portKey,
      storage: REVISION_FLAVORS.NATIVE,
    };
    const records = artifacts
      .map(artifactFileBinding)
      .sort((left, right) => String(left.artifactId).localeCompare(String(right.artifactId)));
    if (records.length > 0) binding.records = records;
    if (definition.kind === 'html' || definition.kind === 'markdown') {
      binding.files = records;
    }
    if (definition.consumerTask) {
      binding.consumers = {
        [definition.consumerTask]: {
          artifactIds: sortedUnique(
            consumerTasks.flatMap((task) =>
              nativeArtifacts(task, definition.consumerPortKey, 'inputs').map(nativeArtifactId),
            ),
          ),
          portKey: definition.consumerPortKey,
        },
      };
    }
    if (definition.kind === 'metrics') binding.members = buildMetricMembers(artifacts);
    if (definition.kind === 'classification-metrics') binding.points = buildRocPoints(artifacts);
    return binding;
  }

  return { artifactIds: [], portKey: definition.portKey, storage: REVISION_FLAVORS.UNKNOWN };
}

function compareTaskInstances(left, right) {
  const leftIndex = left.iterationIndex ?? Number.MAX_SAFE_INTEGER;
  const rightIndex = right.iterationIndex ?? Number.MAX_SAFE_INTEGER;
  if (leftIndex !== rightIndex) return leftIndex - rightIndex;
  const leftIdentity = [left.scopePath, left.parentTaskId, left.displayName, left.name, left.taskId]
    .filter(Boolean)
    .join('|');
  const rightIdentity = [
    right.scopePath,
    right.parentTaskId,
    right.displayName,
    right.name,
    right.taskId,
  ]
    .filter(Boolean)
    .join('|');
  return leftIdentity.localeCompare(rightIdentity);
}

function legacyExecutionSemanticKey(execution) {
  const metadata = field(execution, 'metadata') || {};
  return taskSemanticKey({
    component_id: field(metadata, 'component_id', 'componentId'),
    display_name: field(metadata, 'display_name', 'displayName'),
    name: field(execution, 'name'),
    task_id: field(metadata, 'task_id', 'taskId'),
    task_name: field(metadata, 'task_name', 'taskName'),
  });
}

function legacyExecutionRole(semanticKey, iterationIndex, profile) {
  if (semanticKey === 'execution.unclassified') return 'run-root';
  if (semanticKey === profile?.loop?.task) {
    return Number.isSafeInteger(iterationIndex) ? 'loop-iteration' : 'loop-controller';
  }
  return 'task';
}

function normalizeLegacyExecutionBinding(execution, semanticKey, profile) {
  const metadata = field(execution, 'metadata') || {};
  const iterationValue = field(metadata, 'iteration_index', 'iterationIndex');
  const iterationNumber = Number(iterationValue);
  const iterationIndex =
    Number.isSafeInteger(iterationNumber) && iterationNumber >= 0 ? iterationNumber : null;
  const executorLogs = arrayField(execution, 'executor_logs', 'executorLogs').map((record) => ({
    artifactId: stringValue(field(record, 'artifact_id', 'artifactId', 'id')),
    name: stringValue(field(record, 'name')),
    type: stringValue(field(record, 'type')),
    uri: stringValue(field(record, 'uri')),
  }));
  return Object.fromEntries(
    Object.entries({
      displayName: stringValue(field(metadata, 'display_name', 'displayName')),
      executionId: stringValue(field(execution, 'execution_id', 'executionId', 'id')),
      executionRole: legacyExecutionRole(semanticKey, iterationIndex, profile),
      executorLogs,
      iterationIndex,
      name: stringValue(field(execution, 'name')),
      parentDagId: stringValue(field(metadata, 'parent_dag_id', 'parentDagId')),
      podName: stringValue(field(metadata, 'pod_name', 'podName', 'kfp_pod_name', 'kfpPodName')),
      podUid: stringValue(field(metadata, 'pod_uid', 'podUid')),
      state: stringValue(field(execution, 'state')),
      taskName: stringValue(field(metadata, 'task_name', 'taskName')),
      type: stringValue(field(execution, 'type')),
    }).filter(([, value]) => value !== null),
  );
}

function compareLegacyExecutionInstances(semanticKey, left, right) {
  if (semanticKey === 'task.parallel-loop') {
    const roleOrder = { 'loop-controller': 0, 'loop-iteration': 1 };
    const leftRole = roleOrder[left.executionRole] ?? Number.MAX_SAFE_INTEGER;
    const rightRole = roleOrder[right.executionRole] ?? Number.MAX_SAFE_INTEGER;
    if (leftRole !== rightRole) return leftRole - rightRole;
  }
  const leftIteration = left.iterationIndex;
  const rightIteration = right.iterationIndex;
  if (Number.isSafeInteger(leftIteration) || Number.isSafeInteger(rightIteration)) {
    return (leftIteration ?? Number.MAX_SAFE_INTEGER) - (rightIteration ?? Number.MAX_SAFE_INTEGER);
  }
  if (semanticKey === 'task.retry-once') {
    const retryStateOrder = { CANCELED: 0, FAILED: 0, CACHED: 1, COMPLETE: 1 };
    const leftOrder = retryStateOrder[left.state];
    const rightOrder = retryStateOrder[right.state];
    if (leftOrder !== undefined && rightOrder !== undefined && leftOrder !== rightOrder) {
      return leftOrder - rightOrder;
    }
  }
  const leftIdentity = [left.taskName, left.displayName, left.name, left.type]
    .filter(Boolean)
    .join('|');
  const rightIdentity = [right.taskName, right.displayName, right.name, right.type]
    .filter(Boolean)
    .join('|');
  return leftIdentity.localeCompare(rightIdentity);
}

function buildLegacyExecutionBindings(executions, profile) {
  const normalizedExecutions = executions.map((execution) => {
    const semanticKey = legacyExecutionSemanticKey(execution) || 'execution.unclassified';
    return {
      binding: normalizeLegacyExecutionBinding(execution, semanticKey, profile),
      semanticKey,
    };
  });
  const loopIterationsByExecutionId = new Map(
    normalizedExecutions
      .filter(
        ({ binding, semanticKey }) =>
          semanticKey === profile?.loop?.task &&
          binding.executionRole === 'loop-iteration' &&
          binding.executionId,
      )
      .map(({ binding }) => [binding.executionId, binding.iterationIndex]),
  );
  const bindings = {};
  for (const normalized of normalizedExecutions) {
    const { semanticKey } = normalized;
    let { binding } = normalized;
    if (
      semanticKey === profile?.loop?.worker &&
      !Number.isSafeInteger(binding.iterationIndex) &&
      loopIterationsByExecutionId.has(binding.parentDagId)
    ) {
      binding = {
        ...binding,
        iterationIndex: loopIterationsByExecutionId.get(binding.parentDagId),
        iterationIndexEvidence: 'mlmd-parent-dag',
      };
    }
    if (!bindings[semanticKey]) bindings[semanticKey] = [];
    bindings[semanticKey].push(binding);
  }
  for (const [semanticKey, instances] of Object.entries(bindings)) {
    instances.sort((left, right) => compareLegacyExecutionInstances(semanticKey, left, right));
    for (let index = 1; index < instances.length; index++) {
      if (
        compareLegacyExecutionInstances(semanticKey, instances[index - 1], instances[index]) === 0
      ) {
        throw new Error(
          `Legacy MLMD executions for ${semanticKey} lack a stable iteration or retry discriminator.`,
        );
      }
    }
  }
  return sortObject(bindings);
}

function addLegacyExecutionRelationships(relationships, executionInstances, profile) {
  const semanticByExecutionId = new Map();
  for (const [semanticKey, instances] of Object.entries(executionInstances)) {
    for (const instance of instances) {
      if (instance?.executionId) semanticByExecutionId.set(instance.executionId, semanticKey);
    }
  }
  const expectedContains = new Set(
    (profile?.relationships || [])
      .filter((relationship) => relationship.kind === 'contains')
      .map((relationship) => `${relationship.source}|${relationship.target}`),
  );
  for (const [childKey, instances] of Object.entries(executionInstances)) {
    for (const instance of instances) {
      const parentKey = semanticByExecutionId.get(instance?.parentDagId);
      const relationshipKey = `${parentKey}|${childKey}`;
      const mapKey = `contains|${relationshipKey}`;
      if (parentKey && expectedContains.has(relationshipKey) && !relationships.has(mapKey)) {
        relationships.delete(`depends-on|${relationshipKey}`);
        addRelationship(relationships, 'contains', parentKey, childKey, 'mlmd-parent-dag');
      }
    }
  }
}

function addLegacyPipelineSpecDependencies(relationships, pipelineSpec, profile) {
  const expectedDependencyCount = (profile?.relationships || []).filter(
    (relationship) => relationship.kind === 'depends-on',
  ).length;
  if (expectedDependencyCount === 0) return;
  if (!pipelineSpec || typeof pipelineSpec !== 'object' || Array.isArray(pipelineSpec)) {
    throw new Error('Legacy dependency evidence requires the selected pipeline version spec.');
  }

  const dagTasks = [];
  const rootTasks = field(field(pipelineSpec, 'root'), 'dag')?.tasks;
  if (rootTasks && typeof rootTasks === 'object' && !Array.isArray(rootTasks)) {
    dagTasks.push(rootTasks);
  }
  for (const component of Object.values(field(pipelineSpec, 'components') || {})) {
    const tasks = field(field(component, 'dag'), 'tasks');
    if (tasks && typeof tasks === 'object' && !Array.isArray(tasks)) dagTasks.push(tasks);
  }
  if (dagTasks.length === 0) {
    throw new Error('Selected pipeline version spec contains no DAG task definitions.');
  }

  for (const tasks of dagTasks) {
    for (const [taskName, task] of Object.entries(tasks)) {
      const declaredName = field(field(task, 'taskInfo', 'task_info'), 'name') || taskName;
      const targetKey = taskCandidateSemanticKey({ name: declaredName });
      if (!targetKey) {
        throw new Error(`Pipeline version spec contains unknown task ${declaredName}.`);
      }
      const rawDependencies = field(task, 'dependentTasks', 'dependent_tasks');
      if (rawDependencies !== undefined && !Array.isArray(rawDependencies)) {
        throw new Error(`Pipeline version task ${declaredName} has invalid dependentTasks.`);
      }
      for (const dependencyName of rawDependencies || []) {
        const sourceKey = taskCandidateSemanticKey({ name: dependencyName });
        if (!sourceKey) {
          throw new Error(
            `Pipeline version task ${declaredName} depends on unknown task ${dependencyName}.`,
          );
        }
        addRelationship(relationships, 'depends-on', sourceKey, targetKey, 'pipeline-version-spec');
      }
    }
  }
}

function addRelationship(relationships, kind, source, target, evidence = null) {
  if (!kind || !source || !target) return;
  const key = `${kind}|${source}|${target}`;
  const existing = relationships.get(key);
  if (existing) {
    existing.occurrences++;
    return;
  }
  relationships.set(key, {
    ...(evidence ? { evidence } : {}),
    kind,
    occurrences: 1,
    source,
    target,
  });
}

function argoNodeIdentitySuffix(value) {
  const match = String(value || '').match(/-([a-z0-9]{5,})$/i);
  return match ? match[1] : null;
}

function nativeLoopScopeSemanticKey(task, flavor) {
  if (flavor !== REVISION_FLAVORS.NATIVE) return null;
  const semanticKey = taskCandidateSemanticKey(task);
  if (
    semanticKey &&
    TASK_FIXTURES[semanticKey]?.kind === 'loop' &&
    nativeTaskTypeMatches(field(task, 'type'), 'dag')
  ) {
    return semanticKey;
  }
  return null;
}

function buildTaskBindings(tasks, flavor, hydratedArtifacts = []) {
  const allObservations = tasks.map((task) => ({
    binding: normalizeTaskBinding(task, flavor, hydratedArtifacts),
    candidateSemanticKey: taskCandidateSemanticKey(task),
    scopeSemanticKey: nativeLoopScopeSemanticKey(task, flavor),
    semanticKey: taskSemanticKey(task),
  }));
  const observations = allObservations.filter((observation) => observation.semanticKey);
  const scopeObservations = allObservations.filter((observation) => observation.scopeSemanticKey);
  const scopeInstances = {};
  const taskInstances = {};
  const semanticByIdentity = new Map();
  const semanticByArgoSuffix = new Map();

  for (const observation of allObservations) {
    const { binding, candidateSemanticKey, scopeSemanticKey, semanticKey } = observation;
    const identitySemanticKey = semanticKey || scopeSemanticKey || candidateSemanticKey;
    for (const identity of [binding.taskId, binding.podName]) {
      if (identity && identitySemanticKey) {
        semanticByIdentity.set(identity, identitySemanticKey);
        const suffix = argoNodeIdentitySuffix(identity);
        if (suffix) {
          const existing = semanticByArgoSuffix.get(suffix);
          semanticByArgoSuffix.set(
            suffix,
            existing === undefined || existing === identitySemanticKey ? identitySemanticKey : null,
          );
        }
      }
    }
  }
  for (const { binding, semanticKey } of observations) {
    if (!taskInstances[semanticKey]) taskInstances[semanticKey] = [];
    taskInstances[semanticKey].push(binding);
  }
  for (const { binding, scopeSemanticKey } of scopeObservations) {
    if (!scopeInstances[scopeSemanticKey]) scopeInstances[scopeSemanticKey] = [];
    scopeInstances[scopeSemanticKey].push(binding);
  }
  for (const instances of Object.values(taskInstances)) instances.sort(compareTaskInstances);
  for (const instances of Object.values(scopeInstances)) instances.sort(compareTaskInstances);

  const relationships = new Map();
  for (const observation of observations) {
    const { binding, semanticKey } = observation;
    const parentKey = semanticByIdentity.get(binding.parentTaskId);
    if (parentKey) {
      addRelationship(
        relationships,
        'contains',
        parentKey,
        semanticKey,
        flavor === REVISION_FLAVORS.LEGACY ? 'legacy-task-api' : 'native-task-api',
      );
    }
    if (flavor === REVISION_FLAVORS.LEGACY) continue;
    for (const child of binding.childTaskReferences || []) {
      const childSuffix = argoNodeIdentitySuffix(child.taskId || child.podName);
      const childKey =
        semanticByIdentity.get(child.taskId) ||
        semanticByIdentity.get(child.podName) ||
        (childSuffix ? semanticByArgoSuffix.get(childSuffix) : null) ||
        taskSemanticKey(child);
      if (childKey) {
        addRelationship(relationships, 'depends-on', semanticKey, childKey, 'native-task-api');
      }
    }
  }

  const compatibilityTasks = {};
  for (const [semanticKey, instances] of Object.entries(taskInstances)) {
    compatibilityTasks[semanticKey] = instances[0];
  }
  return {
    relationships,
    scopeInstances: sortObject(scopeInstances),
    taskInstances: sortObject(taskInstances),
    tasks: sortObject(compatibilityTasks),
  };
}

function extractRunBinding(response, semanticKey, options = {}) {
  const run = unwrapRun(response);
  const flavor = detectRevisionFlavor(run);
  const rawTasks = tasksForRun(run, flavor);
  const hydratedArtifacts = [
    ...arrayField(response, 'semantic_artifacts', 'semanticArtifacts'),
    ...(run === response ? [] : arrayField(run, 'semantic_artifacts', 'semanticArtifacts')),
  ];
  const hydratedExecutions = [
    ...arrayField(response, 'semantic_executions', 'semanticExecutions'),
    ...(run === response ? [] : arrayField(run, 'semantic_executions', 'semanticExecutions')),
  ];
  const taskBindings = buildTaskBindings(rawTasks, flavor, hydratedArtifacts);
  const profile = RUN_PROFILES[options.fixtureProfile || DEFAULT_RUN_PROFILE];
  const executionInstances =
    flavor === REVISION_FLAVORS.LEGACY
      ? buildLegacyExecutionBindings(hydratedExecutions, profile)
      : {};
  if (flavor === REVISION_FLAVORS.LEGACY) {
    addLegacyExecutionRelationships(taskBindings.relationships, executionInstances, profile);
    addLegacyPipelineSpecDependencies(taskBindings.relationships, options.pipelineSpec, profile);
  }
  const artifacts = {};
  for (const [artifactKey, definition] of Object.entries(ARTIFACT_FIXTURES)) {
    artifacts[artifactKey] = buildArtifactBinding(rawTasks, flavor, definition, hydratedArtifacts);
    for (const [consumerTask, consumer] of Object.entries(artifacts[artifactKey].consumers || {})) {
      if (consumer.artifactIds.length > 0) {
        addRelationship(
          taskBindings.relationships,
          'artifact-consumer',
          artifactKey,
          consumerTask,
          flavor === REVISION_FLAVORS.LEGACY ? 'mlmd-event' : 'native-task-api',
        );
      }
    }
  }

  return {
    artifacts: sortObject(artifacts),
    displayName: stringValue(field(run, 'display_name', 'displayName', 'name')),
    executionInstances,
    fixtureProfile: options.fixtureProfile || DEFAULT_RUN_PROFILE,
    lineageComplete:
      field(response, 'semantic_lineage_complete', 'semanticLineageComplete') === true ||
      (run !== response &&
        field(run, 'semantic_lineage_complete', 'semanticLineageComplete') === true),
    relationships: [...taskBindings.relationships.values()].sort((left, right) =>
      `${left.kind}|${left.source}|${left.target}`.localeCompare(
        `${right.kind}|${right.source}|${right.target}`,
      ),
    ),
    revisionFlavor: flavor,
    runId: stringValue(field(run, 'run_id', 'runId', 'id')),
    scopeInstances: taskBindings.scopeInstances,
    semanticKey,
    taskInstances: taskBindings.taskInstances,
    // Compatibility projection for existing scenario consumers that expect one task per key.
    tasks: taskBindings.tasks,
  };
}

function semanticResourceDefinitions(resourceDefinitions) {
  const resources = {};
  for (const [kind, definitions] of Object.entries(resourceDefinitions || {})) {
    for (const definition of definitions || []) {
      if (!definition || typeof definition !== 'object' || !definition.semanticKey) continue;
      resources[definition.semanticKey] = {
        displayName: definition.displayName || definition.name,
        kind,
      };
      if (definition.fixtureProfile) {
        resources[definition.semanticKey].fixtureProfile = definition.fixtureProfile;
      }
      if (definition.pipelineSemanticKey) {
        resources[definition.semanticKey].pipeline = definition.pipelineSemanticKey;
      }
    }
  }
  return sortObject(resources);
}

function buildLogicalFixtures(resourceDefinitions) {
  return {
    artifacts: cloneSorted(ARTIFACT_FIXTURES),
    runProfiles: cloneSorted(RUN_PROFILES),
    resources: semanticResourceDefinitions(resourceDefinitions),
    tasks: cloneSorted(TASK_FIXTURES),
  };
}

function expectedRunProfile(logical, runKey) {
  const profileKey = logical?.resources?.[runKey]?.fixtureProfile || DEFAULT_RUN_PROFILE;
  const profile = logical?.runProfiles?.[profileKey] || RUN_PROFILES[profileKey];
  return { profile, profileKey };
}

function sameValues(left, right) {
  return stableStringify(sortedUnique(left || [])) === stableStringify(sortedUnique(right || []));
}

function hasRelationship(binding, expected) {
  return (binding.relationships || []).some(
    (relationship) =>
      relationship.kind === expected.kind &&
      relationship.source === expected.source &&
      relationship.target === expected.target,
  );
}

function nativeTaskTypeMatches(actual, expectedKind) {
  const expected = {
    dag: new Set(['8', 'DAG']),
    loop: new Set(['5', 'LOOP']),
    runtime: new Set(['2', 'RUNTIME']),
  }[expectedKind];
  return !expected || expected.has(String(actual || '').toUpperCase());
}

function nativeArtifactTypeMatches(actual, expectedKind) {
  const expected = {
    html: new Set(['4', 'HTML', 'SYSTEM.HTML']),
    markdown: new Set(['5', 'MARKDOWN', 'SYSTEM.MARKDOWN']),
  }[expectedKind];
  return !expected || expected.has(String(actual || '').toUpperCase());
}

function exactKeySetErrors(label, value, expectedKeys) {
  const actualKeys =
    value && typeof value === 'object' && !Array.isArray(value) ? Object.keys(value).sort() : [];
  const expected = [...expectedKeys].sort();
  const expectedSet = new Set(expected);
  const actualSet = new Set(actualKeys);
  const missing = expected.filter((key) => !actualSet.has(key));
  const unexpected = actualKeys.filter((key) => !expectedSet.has(key));
  if (missing.length === 0 && unexpected.length === 0) return [];
  return [
    `${label} has invalid semantic keys` +
      `${missing.length > 0 ? `; missing ${missing.join(', ')}` : ''}` +
      `${unexpected.length > 0 ? `; unexpected ${unexpected.join(', ')}` : ''}`,
  ];
}

function isRecordValue(value) {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value));
}

function canonicalIdentifier(value, allowLegacyNumeric = false) {
  if (typeof value === 'string' && value.length > 0 && value.trim() === value) return value;
  if (allowLegacyNumeric && typeof value === 'number' && Number.isSafeInteger(value) && value > 0) {
    return String(value);
  }
  return null;
}

function validateIdentifierArray(label, value, allowLegacyNumeric, errors, expectedCount = null) {
  if (!Array.isArray(value)) {
    errors.push(`${label} must be an array of nonempty scalar IDs`);
    return [];
  }
  const identifiers = value.map((identifier, index) => {
    const canonical = canonicalIdentifier(identifier, allowLegacyNumeric);
    if (canonical === null) errors.push(`${label}[${index}] is not a valid nonempty scalar ID`);
    return canonical;
  });
  const valid = identifiers.filter((identifier) => identifier !== null);
  if (expectedCount !== null && value.length !== expectedCount) {
    errors.push(`${label} must contain exactly ${expectedCount} ID(s), found ${value.length}`);
  }
  if (new Set(valid).size !== valid.length) errors.push(`${label} contains duplicate IDs`);
  return valid;
}

function validateArtifactRecordArray(
  label,
  value,
  allowLegacyNumeric,
  errors,
  expectedIds,
  expectedCount,
) {
  if (!Array.isArray(value)) {
    errors.push(`${label} must be an array`);
    return [];
  }
  if (value.length !== expectedCount) {
    errors.push(`${label} must contain exactly ${expectedCount} record(s), found ${value.length}`);
  }
  const recordIds = [];
  for (const [index, record] of value.entries()) {
    if (!isRecordValue(record)) {
      errors.push(`${label}[${index}] must be an object`);
      continue;
    }
    const artifactId = canonicalIdentifier(record.artifactId, allowLegacyNumeric);
    if (artifactId === null) {
      errors.push(`${label}[${index}].artifactId is not a valid nonempty scalar ID`);
    } else {
      recordIds.push(artifactId);
    }
    if (
      typeof record.uri !== 'string' ||
      record.uri.length === 0 ||
      record.uri.trim() !== record.uri
    ) {
      errors.push(`${label}[${index}].uri must be a nonempty string`);
    }
  }
  if (stableStringify([...recordIds].sort()) !== stableStringify([...expectedIds].sort())) {
    errors.push(`${label} artifact IDs do not match the parent artifact binding`);
  }
  return recordIds;
}

function executorLogAttemptIndex(uri) {
  if (typeof uri !== 'string' || uri.length === 0 || uri.trim() !== uri) return null;
  const match = uri.match(/(?:^|\/)executor-logs-(0|[1-9]\d*)$/);
  return match ? Number(match[1]) : null;
}

function validateLegacyExecutionLogs(
  label,
  executionKey,
  instances,
  retryProfile,
  declaredIds,
  declaredUris,
  observedIds,
  observedUris,
  errors,
) {
  const isRuntime = TASK_FIXTURES[executionKey]?.kind === 'runtime';
  const retryAttempts = [];
  for (const [instanceIndex, instance] of instances.entries()) {
    if (!isRecordValue(instance)) continue;
    const logs = instance.executorLogs;
    const instanceLabel = `${label}[${instanceIndex}].executorLogs`;
    if (!Array.isArray(logs)) {
      errors.push(`${instanceLabel} must be an array`);
      continue;
    }
    if (!isRuntime && logs.length !== 0) {
      errors.push(`${instanceLabel} is only allowed for runtime executions`);
    }
    if (isRuntime && executionKey !== retryProfile?.task && logs.length !== 1) {
      errors.push(`${instanceLabel} must contain exactly one executor-log artifact`);
    }
    for (const [recordIndex, record] of logs.entries()) {
      const recordLabel = `${instanceLabel}[${recordIndex}]`;
      if (!isRecordValue(record)) {
        errors.push(`${recordLabel} must be an object`);
        continue;
      }
      errors.push(...exactKeySetErrors(recordLabel, record, ['artifactId', 'name', 'type', 'uri']));
      const artifactId = canonicalIdentifier(record.artifactId, true);
      if (artifactId === null) {
        errors.push(`${recordLabel}.artifactId must be a nonempty legacy artifact ID`);
      } else if (declaredIds.has(artifactId)) {
        errors.push(`${recordLabel}.artifactId collides with a declared semantic artifact`);
      } else if (observedIds.has(artifactId)) {
        errors.push(`${recordLabel}.artifactId duplicates executor-log artifact ${artifactId}`);
      } else {
        observedIds.add(artifactId);
      }
      if (record.name !== 'executor-logs' || record.type !== 'Artifact') {
        errors.push(`${recordLabel} must describe an executor-logs Artifact`);
      }
      const attemptIndex = executorLogAttemptIndex(record.uri);
      if (attemptIndex === null) {
        errors.push(`${recordLabel}.uri must end in an executor-logs attempt suffix`);
      }
      if (typeof record.uri !== 'string' || record.uri.length === 0) {
        errors.push(`${recordLabel}.uri must be a nonempty string`);
      } else if (declaredUris.has(record.uri)) {
        errors.push(`${recordLabel}.uri collides with a declared semantic artifact`);
      } else if (observedUris.has(record.uri)) {
        errors.push(`${recordLabel}.uri duplicates another executor-log URI`);
      } else {
        observedUris.add(record.uri);
      }
      if (executionKey === retryProfile?.task && attemptIndex !== null) {
        retryAttempts.push(attemptIndex);
      } else if (isRuntime && attemptIndex !== 0) {
        errors.push(`${recordLabel}.uri must identify attempt 0`);
      }
    }
  }
  if (executionKey === retryProfile?.task) {
    const expectedAttempts = Array.from({ length: retryProfile.attempts }, (_, index) => index);
    if (
      !sameValues(
        [...retryAttempts].sort((left, right) => left - right),
        expectedAttempts,
      )
    ) {
      errors.push(
        `${label} executor-log attempts ${JSON.stringify(retryAttempts)} did not match ${JSON.stringify(expectedAttempts)}`,
      );
    }
  }
}

function expectedTaskArtifactGroups(taskKey, direction, profileArtifactKeys, allowLegacyNumeric) {
  const groups = [];
  for (const artifactKey of profileArtifactKeys || []) {
    const definition = ARTIFACT_FIXTURES[artifactKey];
    if (direction === 'inputs' && definition?.consumerTask === taskKey) {
      groups.push({ artifactKey, key: normalizedName(definition.consumerPortKey) });
    }
    if (direction === 'outputs' && definition?.producerTask === taskKey) {
      groups.push({ artifactKey, key: normalizedName(definition.portKey) });
    }
  }
  if (
    direction === 'outputs' &&
    !allowLegacyNumeric &&
    TASK_FIXTURES[taskKey]?.kind === 'runtime'
  ) {
    groups.push({ artifactKey: null, key: 'executor-logs' });
  }
  return groups.sort((left, right) => left.key.localeCompare(right.key));
}

function artifactGroupKeySetErrors(label, groups, expectedKeys) {
  const actualKeys = groups
    .filter(isRecordValue)
    .map((group) => normalizedName(group.key) || 'artifact')
    .sort();
  const actualSet = new Set(actualKeys);
  const expectedSet = new Set(expectedKeys);
  const missing = expectedKeys.filter((key) => !actualSet.has(key));
  const unexpected = [...actualSet].filter((key) => !expectedSet.has(key));
  if (missing.length === 0 && unexpected.length === 0) return [];
  return [
    `${label} has invalid artifact group keys` +
      `${missing.length > 0 ? `; missing ${missing.join(', ')}` : ''}` +
      `${unexpected.length > 0 ? `; unexpected ${unexpected.join(', ')}` : ''}`,
  ];
}

function validateTaskArtifactReferences(
  label,
  value,
  allowLegacyNumeric,
  errors,
  declaredIds,
  declaredUris,
  observedExecutorLogIds,
  observedExecutorLogUris,
  taskKey,
  retryProfile,
  profileArtifactKeys,
  artifactBindings,
) {
  const requiresExecutorLogs = !allowLegacyNumeric && TASK_FIXTURES[taskKey]?.kind === 'runtime';
  const expectedGroupsByDirection = Object.fromEntries(
    ['inputs', 'outputs'].map((direction) => [
      direction,
      expectedTaskArtifactGroups(taskKey, direction, profileArtifactKeys, allowLegacyNumeric),
    ]),
  );
  if (value === undefined) {
    if (requiresExecutorLogs) {
      errors.push(`${label} must contain exactly one native runtime executor-logs output group`);
    } else if (Object.values(expectedGroupsByDirection).some((groups) => groups.length > 0)) {
      errors.push(`${label} must contain the declared semantic artifact groups`);
    }
    return;
  }
  if (!isRecordValue(value)) {
    errors.push(`${label} must be an object`);
    return;
  }
  let allowedExecutorLogGroupCount = 0;
  for (const direction of ['inputs', 'outputs']) {
    const groups = value[direction];
    if (!Array.isArray(groups)) {
      errors.push(`${label}.${direction} must be an array`);
      continue;
    }
    errors.push(
      ...artifactGroupKeySetErrors(
        `${label}.${direction}`,
        groups,
        expectedGroupsByDirection[direction].map((group) => group.key),
      ),
    );
    const normalizedGroupKeys = new Set();
    for (const [groupIndex, group] of groups.entries()) {
      if (!isRecordValue(group) || !Array.isArray(group.artifacts)) {
        errors.push(`${label}.${direction}[${groupIndex}] must contain an artifact array`);
        continue;
      }
      const normalizedGroupKey = normalizedName(group.key) || 'artifact';
      if (normalizedGroupKeys.has(normalizedGroupKey)) {
        errors.push(
          `${label}.${direction} contains duplicate normalized group key ${normalizedGroupKey}`,
        );
      }
      normalizedGroupKeys.add(normalizedGroupKey);
      const isExecutorLogs = group.key === 'executor-logs';
      const expectedExecutorLogCount = taskKey === retryProfile?.task ? retryProfile.attempts : 1;
      const expectedGroup = expectedGroupsByDirection[direction].find(
        (candidate) => candidate.key === normalizedGroupKey,
      );
      const expectedBinding = expectedGroup?.artifactKey
        ? artifactBindings?.[expectedGroup.artifactKey]
        : null;
      const expectedRawIds = expectedBinding
        ? direction === 'inputs'
          ? expectedBinding.consumers?.[taskKey]?.artifactIds
          : expectedBinding.artifactIds
        : [];
      const expectedIds = Array.isArray(expectedRawIds)
        ? expectedRawIds
            .map((identifier) => canonicalIdentifier(identifier, allowLegacyNumeric))
            .filter((identifier) => identifier !== null)
        : [];
      const expectedUrisById = new Map(
        (Array.isArray(expectedBinding?.records) ? expectedBinding.records : [])
          .filter(isRecordValue)
          .map((record) => [canonicalIdentifier(record.artifactId, allowLegacyNumeric), record.uri])
          .filter(([identifier, uri]) =>
            Boolean(identifier && typeof uri === 'string' && uri.length > 0),
          ),
      );
      const expectedCount = isExecutorLogs
        ? expectedExecutorLogCount
        : expectedGroup?.artifactKey
          ? expectedIds.length
          : 1;
      const executorLogsAllowed =
        isExecutorLogs &&
        !allowLegacyNumeric &&
        direction === 'outputs' &&
        TASK_FIXTURES[taskKey]?.kind === 'runtime';
      if (executorLogsAllowed) allowedExecutorLogGroupCount += 1;
      if (isExecutorLogs && !executorLogsAllowed) {
        errors.push(`${label}.${direction}[${groupIndex}] contains forbidden executor-logs`);
      }
      if (group.artifacts.length !== expectedCount) {
        errors.push(
          `${label}.${direction}[${groupIndex}].artifacts must contain exactly ${expectedCount} record(s), found ${group.artifacts.length}`,
        );
      }
      const observedGroupIds = [];
      for (const [recordIndex, record] of group.artifacts.entries()) {
        const recordLabel = `${label}.${direction}[${groupIndex}].artifacts[${recordIndex}]`;
        if (!isRecordValue(record)) {
          errors.push(`${recordLabel} must be an object`);
          continue;
        }
        if (isExecutorLogs) {
          errors.push(
            ...exactKeySetErrors(recordLabel, record, ['artifactId', 'name', 'type', 'uri']),
          );
          if (record.name !== 'executor-logs') {
            errors.push(`${recordLabel}.name must be executor-logs`);
          }
          if (record.type !== 'Artifact') {
            errors.push(`${recordLabel}.type must be Artifact`);
          }
        }
        const artifactId = canonicalIdentifier(record.artifactId, allowLegacyNumeric);
        if (artifactId === null) {
          errors.push(`${recordLabel}.artifactId is not a valid nonempty scalar ID`);
        } else if (isExecutorLogs) {
          if (declaredIds.has(artifactId)) {
            errors.push(`${recordLabel}.artifactId collides with a declared semantic artifact`);
          }
          if (observedExecutorLogIds.has(artifactId)) {
            errors.push(`${label}.${direction}[${groupIndex}] contains duplicate executor-log IDs`);
          }
          observedExecutorLogIds.add(artifactId);
        } else if (!declaredIds.has(artifactId)) {
          errors.push(`${recordLabel}.artifactId is not declared by a semantic artifact binding`);
        } else {
          observedGroupIds.push(artifactId);
        }
        if (
          typeof record.uri !== 'string' ||
          record.uri.length === 0 ||
          record.uri.trim() !== record.uri
        ) {
          errors.push(`${recordLabel}.uri must be a nonempty string`);
        } else if (isExecutorLogs) {
          if (declaredUris.has(record.uri)) {
            errors.push(`${recordLabel}.uri collides with a declared semantic artifact`);
          }
          if (observedExecutorLogUris.has(record.uri)) {
            errors.push(
              `${label}.${direction}[${groupIndex}] contains duplicate executor-log URIs`,
            );
          }
          observedExecutorLogUris.add(record.uri);
          const attemptIndex = executorLogAttemptIndex(record.uri);
          if (attemptIndex !== recordIndex) {
            errors.push(
              `${recordLabel}.uri must end exactly in deterministic executor-log leaf executor-logs-${recordIndex}`,
            );
          }
        } else if (
          artifactId !== null &&
          expectedGroup?.artifactKey &&
          expectedUrisById.get(artifactId) !== record.uri
        ) {
          errors.push(
            `${recordLabel}.uri does not match ${expectedGroup.artifactKey} record for artifact ID ${artifactId}`,
          );
        }
      }
      if (
        expectedGroup?.artifactKey &&
        stableStringify([...observedGroupIds].sort()) !== stableStringify([...expectedIds].sort())
      ) {
        errors.push(
          `${label}.${direction}[${groupIndex}] artifact IDs do not match ${expectedGroup.artifactKey}`,
        );
      }
    }
  }
  const expectedGroupCount = requiresExecutorLogs ? 1 : 0;
  if (allowedExecutorLogGroupCount !== expectedGroupCount) {
    errors.push(
      `${label} must contain exactly ${expectedGroupCount} native runtime executor-logs output group(s), found ${allowedExecutorLogGroupCount}`,
    );
  }
}

function relationshipTriple(relationship) {
  if (!relationship || typeof relationship !== 'object' || Array.isArray(relationship)) return null;
  const { kind, source, target } = relationship;
  return kind && source && target ? `${kind}|${source}|${target}` : null;
}

function expectedLegacyExecutionCounts(profile) {
  const counts = { ...(profile.tasks || {}), 'execution.unclassified': 1 };
  if (profile.loop?.task && Number.isSafeInteger(profile.loop.iterations)) {
    counts[profile.loop.task] = (counts[profile.loop.task] || 0) + profile.loop.iterations;
  }
  return counts;
}

function validateRunBindingClosure(runKey, binding, profile, errors) {
  const expectedScopeKeys =
    binding.revisionFlavor === REVISION_FLAVORS.NATIVE && profile.loop ? [profile.loop.task] : [];
  errors.push(
    ...exactKeySetErrors(
      `${runKey}: taskInstances`,
      binding.taskInstances,
      Object.keys(profile.tasks || {}),
    ),
    ...exactKeySetErrors(`${runKey}: artifacts`, binding.artifacts, profile.artifacts || []),
    ...exactKeySetErrors(`${runKey}: scopeInstances`, binding.scopeInstances, expectedScopeKeys),
  );

  const allowLegacyNumeric = binding.revisionFlavor === REVISION_FLAVORS.LEGACY;
  const declaredArtifactIds = new Set(
    (profile.artifacts || []).flatMap((artifactKey) => {
      const values = binding.artifacts?.[artifactKey]?.artifactIds;
      return Array.isArray(values)
        ? values
            .map((value) => canonicalIdentifier(value, allowLegacyNumeric))
            .filter((value) => value !== null)
        : [];
    }),
  );
  const declaredArtifactUris = new Set(
    (profile.artifacts || []).flatMap((artifactKey) => {
      const records = binding.artifacts?.[artifactKey]?.records;
      return Array.isArray(records)
        ? records
            .map((record) => (isRecordValue(record) ? record.uri : null))
            .filter((uri) => typeof uri === 'string' && uri.length > 0)
        : [];
    }),
  );
  const observedExecutorLogIds = new Set();
  const observedExecutorLogUris = new Set();
  if (binding.revisionFlavor === REVISION_FLAVORS.LEGACY && binding.lineageComplete === true) {
    const expectedExecutionKeys = [...Object.keys(profile.tasks || {}), 'execution.unclassified'];
    errors.push(
      ...exactKeySetErrors(
        `${runKey}: executionInstances`,
        binding.executionInstances,
        expectedExecutionKeys,
      ),
    );
    for (const [executionKey, expectedCount] of Object.entries(
      expectedLegacyExecutionCounts(profile),
    )) {
      const instances = binding.executionInstances?.[executionKey];
      if (!Array.isArray(instances) || instances.length !== expectedCount) {
        errors.push(
          `${runKey}: executionInstances.${executionKey} must contain exactly ${expectedCount} execution(s)`,
        );
      }
    }
  }
  for (const taskKey of Object.keys(profile.tasks || {})) {
    const instances = binding.taskInstances?.[taskKey];
    if (!Array.isArray(instances)) {
      errors.push(`${runKey}: taskInstances.${taskKey} must be an array`);
      continue;
    }
    for (const [index, instance] of instances.entries()) {
      const label = `${runKey}: taskInstances.${taskKey}[${index}]`;
      if (!isRecordValue(instance)) {
        errors.push(`${label} must be an object`);
        continue;
      }
      if (canonicalIdentifier(instance.taskId) === null) {
        errors.push(`${label}.taskId must be a nonempty string ID`);
      }
      const isUnjoinableRepeatedLegacyTask =
        binding.lineageComplete === true &&
        taskKey === profile.loop?.worker &&
        (profile.tasks?.[taskKey] || 0) > 1;
      const requiresLegacyExecutionId =
        (binding.lineageComplete === true && !isUnjoinableRepeatedLegacyTask) ||
        ['inputs', 'outputs'].some(
          (direction) =>
            expectedTaskArtifactGroups(taskKey, direction, profile.artifacts, allowLegacyNumeric)
              .length > 0,
        );
      if (
        binding.revisionFlavor === REVISION_FLAVORS.LEGACY &&
        requiresLegacyExecutionId &&
        canonicalIdentifier(instance.mlmdExecutionId, true) === null
      ) {
        errors.push(`${label}.mlmdExecutionId must be a nonempty legacy execution ID`);
      } else if (
        binding.revisionFlavor === REVISION_FLAVORS.LEGACY &&
        Object.hasOwn(instance, 'mlmdExecutionId') &&
        canonicalIdentifier(instance.mlmdExecutionId, true) === null
      ) {
        errors.push(`${label}.mlmdExecutionId must be omitted or a nonempty legacy execution ID`);
      }
      validateTaskArtifactReferences(
        `${label}.artifactReferences`,
        instance.artifactReferences,
        allowLegacyNumeric,
        errors,
        declaredArtifactIds,
        declaredArtifactUris,
        observedExecutorLogIds,
        observedExecutorLogUris,
        taskKey,
        profile.retry,
        profile.artifacts,
        binding.artifacts,
      );
    }
  }

  const executionIds = new Set();
  if (!isRecordValue(binding.executionInstances)) {
    errors.push(`${runKey}: executionInstances must be an object`);
  } else {
    for (const [executionKey, instances] of Object.entries(binding.executionInstances)) {
      if (executionKey !== 'execution.unclassified' && !TASK_FIXTURES[executionKey]) {
        errors.push(`${runKey}: executionInstances has invalid semantic key ${executionKey}`);
      }
      if (!Array.isArray(instances)) {
        errors.push(`${runKey}: executionInstances.${executionKey} must be an array`);
        continue;
      }
      for (const [index, instance] of instances.entries()) {
        const label = `${runKey}: executionInstances.${executionKey}[${index}]`;
        if (!isRecordValue(instance)) {
          errors.push(`${label} must be an object`);
          continue;
        }
        const executionId = canonicalIdentifier(instance.executionId, true);
        if (executionId === null) {
          errors.push(`${label}.executionId must be a nonempty legacy execution ID`);
        } else if (executionIds.has(executionId)) {
          errors.push(`${label}.executionId duplicates legacy execution ${executionId}`);
        } else {
          executionIds.add(executionId);
        }
      }
      if (binding.revisionFlavor === REVISION_FLAVORS.LEGACY) {
        validateLegacyExecutionLogs(
          `${runKey}: executionInstances.${executionKey}`,
          executionKey,
          instances,
          profile.retry,
          declaredArtifactIds,
          declaredArtifactUris,
          observedExecutorLogIds,
          observedExecutorLogUris,
          errors,
        );
      }
    }
  }
  if (executionIds.size > 0) {
    for (const [taskKey, instances] of Object.entries(binding.taskInstances || {})) {
      for (const [index, instance] of (Array.isArray(instances) ? instances : []).entries()) {
        const executionId = canonicalIdentifier(instance?.mlmdExecutionId, true);
        if (executionId !== null && !executionIds.has(executionId)) {
          errors.push(
            `${runKey}: taskInstances.${taskKey}[${index}].mlmdExecutionId is absent from executionInstances`,
          );
        }
      }
    }
  }
  if (binding.revisionFlavor === REVISION_FLAVORS.LEGACY && binding.lineageComplete === true) {
    for (const taskKey of Object.keys(profile.tasks || {})) {
      const taskInstances = binding.taskInstances?.[taskKey];
      const taskExecutionIds = (Array.isArray(taskInstances) ? taskInstances : [])
        .map((instance) => canonicalIdentifier(instance?.mlmdExecutionId, true))
        .filter((executionId) => executionId !== null)
        .sort((left, right) => left.localeCompare(right, 'en', { numeric: true }));
      const executionInstances = binding.executionInstances?.[taskKey];
      const taskFacingExecutions = (
        Array.isArray(executionInstances) ? executionInstances : []
      ).filter(
        (execution) =>
          taskKey !== profile.loop?.task || execution?.executionRole === 'loop-controller',
      );
      const lineageExecutionIds = taskFacingExecutions
        .map((execution) => canonicalIdentifier(execution?.executionId, true))
        .filter((executionId) => executionId !== null)
        .sort((left, right) => left.localeCompare(right, 'en', { numeric: true }));
      if (taskKey === profile.loop?.worker && taskExecutionIds.length === 0) continue;
      if (stableStringify(taskExecutionIds) !== stableStringify(lineageExecutionIds)) {
        errors.push(
          `${runKey}: taskInstances.${taskKey} MLMD execution IDs do not exactly match task-facing executionInstances`,
        );
      }
    }

    const rawRootExecutions = binding.executionInstances?.['execution.unclassified'];
    const rootExecutions = Array.isArray(rawRootExecutions) ? rawRootExecutions : [];
    if (rootExecutions.some((execution) => execution?.executionRole !== 'run-root')) {
      errors.push(`${runKey}: unclassified execution must use the run-root role`);
    }
    for (const [taskKey, instances] of Object.entries(binding.executionInstances || {})) {
      if (taskKey === 'execution.unclassified' || taskKey === profile.loop?.task) continue;
      if ((instances || []).some((execution) => execution?.executionRole !== 'task')) {
        errors.push(`${runKey}: executionInstances.${taskKey} must use the task role`);
      }
    }

    if (profile.loop) {
      const rawLoopExecutions = binding.executionInstances?.[profile.loop.task];
      const loopExecutions = Array.isArray(rawLoopExecutions) ? rawLoopExecutions : [];
      const controllers = loopExecutions.filter(
        (execution) => execution?.executionRole === 'loop-controller',
      );
      const iterations = loopExecutions.filter(
        (execution) => execution?.executionRole === 'loop-iteration',
      );
      if (controllers.length !== 1) {
        errors.push(
          `${runKey}: ${profile.loop.task} must contain exactly one legacy loop-controller execution`,
        );
      }
      const expectedIndexes = [...(profile.loop.iterationIndexes || [])];
      const actualIndexes = iterations.map((execution) => execution?.iterationIndex);
      if (stableStringify(actualIndexes) !== stableStringify(expectedIndexes)) {
        errors.push(
          `${runKey}: ${profile.loop.task} loop-iteration execution indexes must exactly match ${expectedIndexes.join(', ')}`,
        );
      }
      const controllerId = canonicalIdentifier(controllers[0]?.executionId, true);
      if (
        controllerId !== null &&
        iterations.some(
          (execution) => canonicalIdentifier(execution?.parentDagId, true) !== controllerId,
        )
      ) {
        errors.push(
          `${runKey}: ${profile.loop.task} loop-iteration executions must be children of the loop controller`,
        );
      }
      const iterationIdsByIndex = new Map(
        iterations.map((execution) => [
          execution.iterationIndex,
          canonicalIdentifier(execution.executionId, true),
        ]),
      );
      const rawWorkerExecutions = binding.executionInstances?.[profile.loop.worker];
      const workerExecutions = Array.isArray(rawWorkerExecutions) ? rawWorkerExecutions : [];
      const workerIndexes = workerExecutions.map((execution) => execution?.iterationIndex);
      if (stableStringify(workerIndexes) !== stableStringify(expectedIndexes)) {
        errors.push(
          `${runKey}: ${profile.loop.worker} execution indexes must exactly match ${expectedIndexes.join(', ')}`,
        );
      }
      if (
        workerExecutions.some(
          (execution) =>
            canonicalIdentifier(execution?.parentDagId, true) !==
            iterationIdsByIndex.get(execution?.iterationIndex),
        )
      ) {
        errors.push(
          `${runKey}: ${profile.loop.worker} executions must be children of their matching loop iterations`,
        );
      }
    }
  }

  for (const artifactKey of profile.artifacts || []) {
    const expectedMembers = Object.keys(ARTIFACT_FIXTURES[artifactKey]?.members || {});
    errors.push(
      ...exactKeySetErrors(
        `${runKey}: ${artifactKey}.members`,
        binding.artifacts?.[artifactKey]?.members,
        expectedMembers,
      ),
    );
  }

  if (!Array.isArray(binding.relationships)) {
    errors.push(`${runKey}: relationships must be an array`);
  }
  const expectedRelationships = (profile.relationships || []).map(relationshipTriple).sort();
  const actualRelationships = Array.isArray(binding.relationships)
    ? binding.relationships.map(relationshipTriple).sort()
    : [];
  if (
    binding.revisionFlavor === REVISION_FLAVORS.LEGACY &&
    binding.lineageComplete === true &&
    (Array.isArray(binding.relationships) ? binding.relationships : []).some(
      (relationship) =>
        relationship?.kind === 'depends-on' && relationship?.evidence !== 'pipeline-version-spec',
    )
  ) {
    errors.push(
      `${runKey}: legacy depends-on relationships must declare pipeline-version-spec evidence`,
    );
  }
  const malformed = actualRelationships.filter((triple) => !triple).length;
  const expectedSet = new Set(expectedRelationships);
  const actualSet = new Set(actualRelationships.filter(Boolean));
  const missing = expectedRelationships.filter((triple) => !actualSet.has(triple));
  const unexpected = [...actualSet].filter((triple) => !expectedSet.has(triple));
  if (
    malformed > 0 ||
    missing.length > 0 ||
    unexpected.length > 0 ||
    actualRelationships.length !== expectedRelationships.length
  ) {
    errors.push(
      `${runKey}: relationships do not match the semantic profile` +
        `${missing.length > 0 ? `; missing ${missing.join(', ')}` : ''}` +
        `${unexpected.length > 0 ? `; unexpected ${unexpected.join(', ')}` : ''}` +
        `${malformed > 0 ? `; malformed ${malformed}` : ''}`,
    );
  }
}

function validateRunBinding(runKey, binding, profile, errors) {
  if (!profile) {
    errors.push(`${runKey}: unknown fixture profile ${binding.fixtureProfile}`);
    return;
  }

  validateRunBindingClosure(runKey, binding, profile, errors);

  for (const [taskKey, expectedCount] of Object.entries(profile.tasks || {})) {
    const value = binding.taskInstances?.[taskKey];
    const instances = Array.isArray(value) ? value : [];
    if (instances.length !== expectedCount) {
      errors.push(
        `${runKey}: expected ${expectedCount} ${taskKey} instance(s), found ${instances.length}`,
      );
      continue;
    }
    if (binding.revisionFlavor === REVISION_FLAVORS.NATIVE) {
      const expectedKind = TASK_FIXTURES[taskKey]?.kind;
      for (const instance of instances) {
        if (!isRecordValue(instance)) continue;
        if (canonicalIdentifier(instance.taskId) === null) {
          errors.push(`${runKey}: ${taskKey} is missing a native task ID`);
        }
        if (!nativeTaskTypeMatches(instance.type, expectedKind)) {
          errors.push(`${runKey}: ${taskKey} has native type ${instance.type || 'unknown'}`);
        }
        const podBindings = instance.podBindings || [];
        if (!Array.isArray(podBindings)) {
          errors.push(`${runKey}: ${taskKey} podBindings must be an array`);
          continue;
        }
        for (const pod of podBindings) {
          if (
            !isRecordValue(pod) ||
            canonicalIdentifier(pod.name) === null ||
            canonicalIdentifier(pod.uid) === null ||
            !['DRIVER', 'EXECUTOR'].includes(pod.type)
          ) {
            errors.push(
              `${runKey}: ${taskKey} has a pod without a valid DRIVER/EXECUTOR role, name, and UID`,
            );
          }
        }
      }
    }
  }

  for (const artifactKey of profile.artifacts || []) {
    const artifactBinding = binding.artifacts?.[artifactKey];
    if (!isRecordValue(artifactBinding)) {
      errors.push(`${runKey}: missing ${artifactKey}`);
      continue;
    }
    const definition = ARTIFACT_FIXTURES[artifactKey];
    const sourceLabel =
      binding.revisionFlavor === REVISION_FLAVORS.NATIVE ? 'native' : 'legacy MLMD';
    const allowLegacyNumeric = binding.revisionFlavor === REVISION_FLAVORS.LEGACY;
    const expectedArtifactCount =
      artifactKey === 'artifact.scalar-metrics' && !allowLegacyNumeric ? 2 : 1;
    const artifactIds = validateIdentifierArray(
      `${runKey}: ${artifactKey}.artifactIds`,
      artifactBinding.artifactIds,
      allowLegacyNumeric,
      errors,
      expectedArtifactCount,
    );
    const records = Array.isArray(artifactBinding.records) ? artifactBinding.records : [];
    validateArtifactRecordArray(
      `${runKey}: ${artifactKey}.records`,
      artifactBinding.records,
      allowLegacyNumeric,
      errors,
      artifactIds,
      expectedArtifactCount,
    );
    for (const record of records.filter(isRecordValue)) {
      if (canonicalIdentifier(record.artifactId, allowLegacyNumeric) === null) {
        errors.push(`${runKey}: ${artifactKey} is missing a ${sourceLabel} artifact ID`);
      }
      if (
        typeof record.uri !== 'string' ||
        record.uri.length === 0 ||
        record.uri.trim() !== record.uri
      ) {
        errors.push(`${runKey}: ${artifactKey} is missing a ${sourceLabel} artifact URI`);
      }
    }
    if (definition?.kind === 'html' || definition?.kind === 'markdown') {
      const files = Array.isArray(artifactBinding.files) ? artifactBinding.files : [];
      validateArtifactRecordArray(
        `${runKey}: ${artifactKey}.files`,
        artifactBinding.files,
        allowLegacyNumeric,
        errors,
        artifactIds,
        1,
      );
      for (const file of files.filter(isRecordValue)) {
        if (canonicalIdentifier(file.artifactId, allowLegacyNumeric) === null) {
          errors.push(`${runKey}: ${artifactKey} is missing a ${sourceLabel} artifact ID`);
        }
        if (normalizedName(file.name) !== normalizedName(definition.portKey)) {
          errors.push(
            `${runKey}: ${artifactKey} has ${sourceLabel} name ${file.name || 'unknown'}, expected ${definition.portKey}`,
          );
        }
        if (!nativeArtifactTypeMatches(file.type, definition.kind)) {
          errors.push(
            `${runKey}: ${artifactKey} has ${sourceLabel} type ${file.type || 'unknown'}, expected ${definition.kind}`,
          );
        }
      }
    }
  }

  const allowLegacyNumeric = binding.revisionFlavor === REVISION_FLAVORS.LEGACY;
  const scalarBinding = binding.artifacts['artifact.scalar-metrics'];
  const scalarArtifactIds = Array.isArray(scalarBinding?.artifactIds)
    ? scalarBinding.artifactIds
        .map((identifier) => canonicalIdentifier(identifier, allowLegacyNumeric))
        .filter((identifier) => identifier !== null)
    : [];
  const metricMembers = scalarBinding?.members || {};
  const scalarMemberIds = [];
  for (const [metricKey, definition] of Object.entries(
    ARTIFACT_FIXTURES['artifact.scalar-metrics'].members,
  )) {
    const member = metricMembers[metricKey];
    if (!isRecordValue(member)) {
      errors.push(`${runKey}: missing ${metricKey}`);
      continue;
    }
    const memberIds = validateIdentifierArray(
      `${runKey}: artifact.scalar-metrics.members.${metricKey}.artifactIds`,
      member.artifactIds,
      allowLegacyNumeric,
      errors,
      1,
    );
    scalarMemberIds.push(...memberIds);
    if (member.numberValue !== definition.value) {
      errors.push(
        `${runKey}: ${metricKey} value ${String(member.numberValue)} did not match ${definition.value}`,
      );
    }
  }

  const scalarParentSet = new Set(scalarArtifactIds);
  const scalarMemberSet = new Set(scalarMemberIds);
  if (
    scalarParentSet.size !== scalarMemberSet.size ||
    [...scalarParentSet].some((identifier) => !scalarMemberSet.has(identifier))
  ) {
    errors.push(`${runKey}: scalar metric member IDs do not exactly cover the parent artifact IDs`);
  }
  if (!allowLegacyNumeric && scalarMemberSet.size !== scalarMemberIds.length) {
    errors.push(`${runKey}: native scalar metric member IDs overlap`);
  }

  const rocPoints = binding.artifacts['artifact.roc-curve']?.points || [];
  const expectedRocPoints = ARTIFACT_FIXTURES['artifact.roc-curve'].points;
  if (stableStringify(rocPoints) !== stableStringify(expectedRocPoints)) {
    errors.push(
      `${runKey}: artifact.roc-curve points ${stableStringify(rocPoints)} did not match ${stableStringify(expectedRocPoints)}`,
    );
  }

  if (profile.retry) {
    const retryTask = binding.taskInstances[profile.retry.task]?.[0];
    if (binding.revisionFlavor === REVISION_FLAVORS.NATIVE && isRecordValue(retryTask)) {
      const executorPods = retryTask.executorPods;
      if (
        !Array.isArray(executorPods) ||
        executorPods.length !== profile.retry.attempts ||
        executorPods.some((podName) => canonicalIdentifier(podName) === null)
      ) {
        errors.push(
          `${runKey}: ${profile.retry.task} executorPods must contain exactly ${profile.retry.attempts} nonempty pod names`,
        );
      }
      const podBindings = retryTask.podBindings;
      const executorBindings = Array.isArray(podBindings)
        ? podBindings.filter((pod) => pod?.type === 'EXECUTOR')
        : [];
      const driverBindings = Array.isArray(podBindings)
        ? podBindings.filter((pod) => pod?.type === 'DRIVER')
        : [];
      if (!Array.isArray(podBindings) || executorBindings.length !== profile.retry.attempts) {
        errors.push(
          `${runKey}: ${profile.retry.task} podBindings must contain exactly ${profile.retry.attempts} EXECUTOR records`,
        );
      } else {
        const boundPodNames = executorBindings
          .map((pod) => (isRecordValue(pod) ? canonicalIdentifier(pod.name) : null))
          .filter((podName) => podName !== null)
          .sort();
        const executorPodNames = Array.isArray(executorPods)
          ? executorPods
              .map((podName) => canonicalIdentifier(podName))
              .filter((podName) => podName !== null)
              .sort()
          : [];
        if (stableStringify(boundPodNames) !== stableStringify(executorPodNames)) {
          errors.push(
            `${runKey}: ${profile.retry.task} pod binding names differ from executorPods`,
          );
        }
      }
      if (driverBindings.length > 1) {
        errors.push(`${runKey}: ${profile.retry.task} may contain at most one DRIVER pod binding`);
      }
    }
    const legacyRetryExecutions = binding.executionInstances?.[profile.retry.task];
    let attempts;
    if (
      binding.revisionFlavor === REVISION_FLAVORS.LEGACY &&
      Array.isArray(legacyRetryExecutions) &&
      legacyRetryExecutions.length > 0
    ) {
      if (
        legacyRetryExecutions.length !== 1 ||
        !['CACHED', 'COMPLETE'].includes(legacyRetryExecutions[0]?.state)
      ) {
        errors.push(
          `${runKey}: ${profile.retry.task} must have exactly one completed legacy MLMD execution`,
        );
      }
      attempts = Array.isArray(legacyRetryExecutions[0]?.executorLogs)
        ? legacyRetryExecutions[0].executorLogs.length
        : 0;
      const reportedAttempts = (retryTask?.failedMainJobs?.length || 0) + 1;
      if (retryTask?.failedMainJobs?.length > 0 && reportedAttempts !== attempts) {
        errors.push(`${runKey}: ${profile.retry.task} GetRun and MLMD retry evidence disagree`);
      }
    } else {
      attempts =
        binding.revisionFlavor === REVISION_FLAVORS.LEGACY
          ? (retryTask?.failedMainJobs?.length || 0) + 1
          : retryTask?.executorPods?.length || 0;
    }
    if (attempts !== profile.retry.attempts) {
      errors.push(
        `${runKey}: ${profile.retry.task} recorded ${attempts} attempt(s), expected exactly ${profile.retry.attempts}`,
      );
    }
  }

  if (profile.loop && binding.revisionFlavor === REVISION_FLAVORS.NATIVE) {
    const loopTasks = binding.taskInstances[profile.loop.task] || [];
    const loopTaskId = canonicalIdentifier(loopTasks[0]?.taskId);
    const rawScopes = binding.scopeInstances?.[profile.loop.task];
    const scopes = Array.isArray(rawScopes) ? rawScopes : [];
    if (scopes.length !== profile.loop.iterations) {
      errors.push(
        `${runKey}: ${profile.loop.task} must contain exactly ${profile.loop.iterations} native iteration scope(s)`,
      );
    }
    const scopeIndexes = scopes
      .map((instance) => (isRecordValue(instance) ? instance.iterationIndex : null))
      .filter((value) => value !== null && value !== undefined)
      .sort((left, right) => left - right);
    if (!sameValues(scopeIndexes, profile.loop.iterationIndexes)) {
      errors.push(
        `${runKey}: ${profile.loop.task} iteration scope indexes ${JSON.stringify(scopeIndexes)} did not match ${JSON.stringify(profile.loop.iterationIndexes)}`,
      );
    }
    const scopeIdsByIndex = new Map();
    for (const [scopeIndex, scope] of scopes.entries()) {
      const scopeId = canonicalIdentifier(scope?.taskId);
      if (!isRecordValue(scope) || scopeId === null || !nativeTaskTypeMatches(scope.type, 'dag')) {
        errors.push(
          `${runKey}: ${profile.loop.task} iteration scope ${scopeIndex} must be a native DAG with a task ID`,
        );
        continue;
      }
      if (canonicalIdentifier(scope.parentTaskId) !== loopTaskId) {
        errors.push(
          `${runKey}: ${profile.loop.task} iteration scopes must be children of the outer loop task`,
        );
      }
      if (scopeIdsByIndex.has(scope.iterationIndex)) {
        errors.push(
          `${runKey}: ${profile.loop.task} has duplicate native iteration scope ${scope.iterationIndex}`,
        );
      }
      scopeIdsByIndex.set(scope.iterationIndex, scopeId);
    }
    const workers = binding.taskInstances[profile.loop.worker] || [];
    const indexes = workers
      .map((instance) => (isRecordValue(instance) ? instance.iterationIndex : null))
      .filter((value) => value !== null && value !== undefined)
      .sort((left, right) => left - right);
    if (!sameValues(indexes, profile.loop.iterationIndexes)) {
      errors.push(
        `${runKey}: ${profile.loop.worker} iteration indexes ${JSON.stringify(indexes)} did not match ${JSON.stringify(profile.loop.iterationIndexes)}`,
      );
    }
    if (
      workers.some(
        (worker) =>
          canonicalIdentifier(worker?.parentTaskId) !== scopeIdsByIndex.get(worker?.iterationIndex),
      )
    ) {
      errors.push(
        `${runKey}: ${profile.loop.worker} tasks must be children of their matching native iteration scopes`,
      );
    }
  }

  for (const relationship of profile.relationships || []) {
    if (!hasRelationship(binding, relationship)) {
      errors.push(
        `${runKey}: missing ${relationship.kind} relationship ${relationship.source} -> ${relationship.target}`,
      );
    }
  }

  const scalarConsumer = scalarBinding?.consumers?.['task.consume-metrics'];
  if (
    profile.relationships?.some(
      (relationship) =>
        relationship.kind === 'artifact-consumer' &&
        relationship.source === 'artifact.scalar-metrics',
    )
  ) {
    const consumerIds = validateIdentifierArray(
      `${runKey}: artifact.scalar-metrics consumer artifactIds`,
      scalarConsumer?.artifactIds,
      allowLegacyNumeric,
      errors,
      scalarArtifactIds.length,
    );
    if (
      stableStringify([...consumerIds].sort()) !== stableStringify([...scalarArtifactIds].sort())
    ) {
      errors.push(`${runKey}: scalar metric producer and consumer artifact bindings differ`);
    }
  }
}

function expectedDeploymentResourceBindings() {
  const kinds = {
    experiments: 'experiment',
    pipelines: 'pipeline',
    recurringRuns: 'recurring-run',
    runs: 'run',
  };
  const expected = {};
  for (const [definitionKind, definitions] of Object.entries(SEMANTIC_RESOURCE_DEFINITIONS)) {
    for (const definition of definitions) {
      const binding = {
        displayName: definition.displayName || definition.name,
        kind: kinds[definitionKind],
      };
      if (definition.fixtureProfile) binding.fixtureProfile = definition.fixtureProfile;
      if (definition.pipelineSemanticKey) binding.pipeline = definition.pipelineSemanticKey;
      expected[definition.semanticKey] = binding;
      if (definitionKind === 'pipelines') {
        expected[`${definition.semanticKey}.version`] = {
          displayName: definition.displayName || definition.name,
          fixtureProfile: definition.fixtureProfile,
          kind: 'pipeline-version',
          pipeline: definition.semanticKey,
        };
      }
    }
  }
  return sortObject(expected);
}

function validateCombinedSemanticManifest(manifest) {
  const errors = [];
  const isRecord = (value) => Boolean(value && typeof value === 'object' && !Array.isArray(value));
  if (!isRecord(manifest)) {
    throw new Error('Semantic fixture manifest must contain an object.');
  }
  if (manifest.schemaVersion !== SEMANTIC_SCHEMA_VERSION) {
    errors.push(`schemaVersion must be ${SEMANTIC_SCHEMA_VERSION}`);
  }
  if (manifest.fixtureSet !== SEMANTIC_FIXTURE_SET) {
    errors.push(`fixtureSet must be ${SEMANTIC_FIXTURE_SET}`);
  }
  const logical = manifest.logical;
  if (!isRecord(logical)) {
    errors.push('logical fixture contract is missing');
  } else {
    const expectedLogical = buildLogicalFixtures(SEMANTIC_RESOURCE_DEFINITIONS);
    for (const [fieldName, expected] of [
      ['artifacts', ARTIFACT_FIXTURES],
      ['runProfiles', RUN_PROFILES],
      ['resources', expectedLogical.resources],
      ['tasks', TASK_FIXTURES],
    ]) {
      if (stableStringify(logical[fieldName]) !== stableStringify(expected)) {
        errors.push(`logical.${fieldName} does not match the deterministic fixture contract`);
      }
    }
  }

  for (const role of ['base', 'head']) {
    const deployment = manifest.deployments?.[role];
    const expectedFlavor = role === 'base' ? REVISION_FLAVORS.LEGACY : REVISION_FLAVORS.NATIVE;
    if (!isRecord(deployment)) {
      errors.push(`${role} deployment is missing`);
      continue;
    }
    if (deployment.revisionFlavor !== expectedFlavor) {
      errors.push(`${role} deployment revisionFlavor must be ${expectedFlavor}`);
    }
    if (
      deployment.validation?.valid !== true ||
      !Array.isArray(deployment.validation?.errors) ||
      deployment.validation.errors.length !== 0
    ) {
      errors.push(`${role} deployment does not attest successful fixture validation`);
    }
    if (!isRecord(deployment.bindings?.resources) || !isRecord(deployment.bindings?.runs)) {
      errors.push(`${role} deployment bindings are missing`);
      continue;
    }

    const expectedResources = expectedDeploymentResourceBindings();
    errors.push(
      ...exactKeySetErrors(
        `${role} deployment resource bindings`,
        deployment.bindings.resources,
        Object.keys(expectedResources),
      ),
      ...exactKeySetErrors(
        `${role} deployment run bindings`,
        deployment.bindings.runs,
        RUN_RESOURCE_DEFINITIONS.map((definition) => definition.semanticKey),
      ),
    );
    const resourceIds = new Map();
    for (const [semanticKey, expected] of Object.entries(expectedResources)) {
      const resource = deployment.bindings.resources[semanticKey];
      if (!isRecord(resource)) {
        errors.push(`${role} deployment is missing resource binding ${semanticKey}`);
        continue;
      }
      const resourceId = canonicalIdentifier(resource.id);
      if (resourceId === null) {
        errors.push(`${role} resource binding ${semanticKey} has an invalid generated ID`);
      } else {
        resourceIds.set(semanticKey, resourceId);
      }
      if (Object.entries(expected).some(([fieldName, value]) => resource[fieldName] !== value)) {
        errors.push(`${role} resource binding ${semanticKey} does not match its logical resource`);
      }
    }

    const rawRunIds = new Set();
    for (const definition of RUN_RESOURCE_DEFINITIONS) {
      const { semanticKey } = definition;
      const resource = deployment.bindings.resources[semanticKey];
      const run = deployment.bindings.runs[semanticKey];
      const rawRunId = resourceIds.get(semanticKey);
      if (!isRecord(resource) || !rawRunId) {
        continue;
      }
      if (resource.kind !== 'run' || resource.displayName !== definition.displayName) {
        errors.push(`${role} resource binding ${semanticKey} has invalid kind or display name`);
      }
      if (rawRunIds.has(rawRunId)) {
        errors.push(`${role} deployment reuses generated run ID ${rawRunId}`);
      }
      rawRunIds.add(rawRunId);
      if (!isRecord(run)) {
        errors.push(`${role} deployment is missing run binding ${semanticKey}`);
        continue;
      }
      const boundRunId = canonicalIdentifier(run.runId);
      if (
        boundRunId === null ||
        boundRunId !== rawRunId ||
        run.displayName !== definition.displayName ||
        run.fixtureProfile !== definition.fixtureProfile ||
        run.revisionFlavor !== expectedFlavor ||
        (expectedFlavor === REVISION_FLAVORS.LEGACY && run.lineageComplete !== true) ||
        !isRecord(run.taskInstances) ||
        !isRecord(run.artifacts)
      ) {
        errors.push(`${role} run binding ${semanticKey} does not match its logical resource`);
        continue;
      }
      const runErrors = [];
      validateRunBinding(semanticKey, run, RUN_PROFILES[definition.fixtureProfile], runErrors);
      errors.push(...runErrors.map((error) => `${role} ${error}`));
    }
  }

  if (errors.length > 0) {
    throw new Error(`Semantic fixture manifest failed strict validation: ${errors.join('; ')}`);
  }
  return manifest;
}

function buildSemanticDeployment({ logical, resourceBindings = {}, runResponses = [] }) {
  const runs = {};
  const flavors = new Set();
  for (const observation of runResponses) {
    const { profileKey } = expectedRunProfile(logical, observation.semanticKey);
    const binding = extractRunBinding(observation.response, observation.semanticKey, {
      fixtureProfile: profileKey,
      pipelineSpec: observation.pipelineSpec,
    });
    runs[observation.semanticKey] = binding;
    if (binding.revisionFlavor !== REVISION_FLAVORS.UNKNOWN) flavors.add(binding.revisionFlavor);
  }

  const errors = [];
  let revisionFlavor = REVISION_FLAVORS.UNKNOWN;
  if (flavors.size === 1) revisionFlavor = [...flavors][0];
  if (flavors.size > 1) errors.push(`mixed revision flavors: ${[...flavors].sort().join(', ')}`);
  if (flavors.size === 0) errors.push('revision flavor could not be discovered from run details');

  for (const [runKey, binding] of Object.entries(runs)) {
    if (binding.revisionFlavor === REVISION_FLAVORS.UNKNOWN) {
      errors.push(`${runKey}: unsupported run detail shape`);
      continue;
    }
    const { profile } = expectedRunProfile(logical, runKey);
    validateRunBinding(runKey, binding, profile, errors);
  }

  return {
    bindings: {
      resources: cloneSorted(resourceBindings),
      runs: sortObject(runs),
    },
    fixtureSet: SEMANTIC_FIXTURE_SET,
    logical: cloneSorted(logical),
    revisionFlavor,
    validation: {
      errors,
      valid: errors.length === 0,
    },
  };
}

function sortObject(object) {
  return Object.fromEntries(
    Object.entries(object || {}).sort(([left], [right]) => left.localeCompare(right)),
  );
}

function cloneSorted(value) {
  if (Array.isArray(value)) return value.map(cloneSorted);
  if (!value || typeof value !== 'object') return value;
  return Object.fromEntries(
    Object.keys(value)
      .sort()
      .map((key) => [key, cloneSorted(value[key])]),
  );
}

function stableStringify(value) {
  return JSON.stringify(cloneSorted(value));
}

function semanticSection(manifest) {
  return manifest?.semantic || manifest;
}

function combineSemanticManifests(manifestsOrBase, optionalHeadOrOptions, optionalOptions = {}) {
  const pairedObject = Boolean(manifestsOrBase?.base && manifestsOrBase?.head);
  const manifests = pairedObject
    ? manifestsOrBase
    : { base: manifestsOrBase, head: optionalHeadOrOptions };
  const options = pairedObject ? optionalHeadOrOptions || {} : optionalOptions;
  if (!manifests?.base || !manifests?.head) {
    throw new Error('Both base and head seed manifests are required.');
  }

  const baseSemantic = semanticSection(manifests.base);
  const headSemantic = semanticSection(manifests.head);
  if (!baseSemantic?.logical || !headSemantic?.logical) {
    throw new Error('Both seed manifests must contain semantic logical fixtures.');
  }
  if (stableStringify(baseSemantic.logical) !== stableStringify(headSemantic.logical)) {
    throw new Error('Base and head semantic logical fixtures do not match.');
  }

  const deployment = (role, manifest, semantic) => ({
    apiBase: manifest.apiBase || null,
    bindings: cloneSorted(semantic.bindings || {}),
    defaults: cloneSorted(manifest.defaults || {}),
    resources: cloneSorted(manifest.resources || {}),
    revision: cloneSorted({
      ...(manifest.revision || {}),
      ...(semantic.revision || {}),
      ...(options.revisions?.[role] || {}),
      role,
    }),
    revisionFlavor: semantic.revisionFlavor || REVISION_FLAVORS.UNKNOWN,
    validation: cloneSorted(semantic.validation || { errors: [], valid: false }),
  });

  return {
    deployments: {
      base: deployment('base', manifests.base, baseSemantic),
      head: deployment('head', manifests.head, headSemantic),
    },
    fixtureSet: baseSemantic.fixtureSet || headSemantic.fixtureSet || SEMANTIC_FIXTURE_SET,
    logical: cloneSorted(baseSemantic.logical),
    schemaVersion: SEMANTIC_SCHEMA_VERSION,
  };
}

module.exports = {
  ARTIFACT_FIXTURES,
  COMPARISON_RUN_FIXTURES,
  DEFAULT_RUN_PROFILE,
  REVISION_FLAVORS,
  RUN_RESOURCE_DEFINITIONS,
  RUN_PROFILES,
  SEMANTIC_FIXTURE_SET,
  SEMANTIC_RESOURCE_DEFINITIONS,
  SEMANTIC_SCHEMA_VERSION,
  TASK_FIXTURES,
  buildLogicalFixtures,
  buildSemanticDeployment,
  cloneSorted,
  combineSemanticManifests,
  detectRevisionFlavor,
  extractRunBinding,
  field,
  validateCombinedSemanticManifest,
};
