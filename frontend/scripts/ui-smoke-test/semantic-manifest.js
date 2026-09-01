'use strict';

const SEMANTIC_SCHEMA_VERSION = 'ui-smoke-semantic/v2';
const SEMANTIC_FIXTURE_SET = 'ui-smoke-deterministic-v2';
const DEFAULT_RUN_PROFILE = 'metrics';
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
    artifacts: Object.freeze(['artifact.roc-curve', 'artifact.scalar-metrics']),
    tasks: Object.freeze({ 'task.write-metrics': 1 }),
  }),
  'rich-topology': Object.freeze({
    artifacts: Object.freeze(['artifact.roc-curve', 'artifact.scalar-metrics']),
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
  const value = field(attributes, 'iteration_index', 'iterationIndex');
  return value === undefined || value === null ? null : Number(value);
}

function iterationCount(task) {
  const attributes = field(task, 'type_attributes', 'typeAttributes') || {};
  const value = field(attributes, 'iteration_count', 'iterationCount');
  return value === undefined || value === null ? null : Number(value);
}

function taskPods(task, type) {
  return arrayField(task, 'pods')
    .filter((pod) => {
      const podType = String(field(pod, 'type') || '').toUpperCase();
      return podType === type || (type === 'EXECUTOR' && podType === '2');
    })
    .map((pod) => stringValue(field(pod, 'name')))
    .filter(Boolean);
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

function normalizeTaskBinding(task, flavor) {
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
    binding.mlmdExecutionId = stringValue(field(task, 'execution_id', 'executionId'));
    binding.podName = stringValue(field(task, 'pod_name', 'podName'));
  } else if (flavor === REVISION_FLAVORS.NATIVE) {
    binding.executorPods = taskPods(task, 'EXECUTOR');
    binding.iterationCount = iterationCount(task);
    binding.iterationIndex = iterationIndex(task);
    binding.scopePath = stringValue(field(task, 'scope_path', 'scopePath'));
    binding.type = stringValue(field(task, 'type'));
  }
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
    const artifacts = artifactsByIds(hydratedArtifacts, artifactIds);
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

function addRelationship(relationships, kind, source, target) {
  if (!kind || !source || !target) return;
  const key = `${kind}|${source}|${target}`;
  const existing = relationships.get(key);
  if (existing) {
    existing.occurrences++;
    return;
  }
  relationships.set(key, { kind, occurrences: 1, source, target });
}

function buildTaskBindings(tasks, flavor) {
  const allObservations = tasks.map((task) => ({
    binding: normalizeTaskBinding(task, flavor),
    candidateSemanticKey: taskCandidateSemanticKey(task),
    semanticKey: taskSemanticKey(task),
  }));
  const observations = allObservations.filter((observation) => observation.semanticKey);
  const taskInstances = {};
  const semanticByIdentity = new Map();

  for (const observation of allObservations) {
    const { binding, candidateSemanticKey, semanticKey } = observation;
    const identitySemanticKey = semanticKey || candidateSemanticKey;
    for (const identity of [binding.taskId, binding.podName]) {
      if (identity && identitySemanticKey) semanticByIdentity.set(identity, identitySemanticKey);
    }
  }
  for (const { binding, semanticKey } of observations) {
    if (!taskInstances[semanticKey]) taskInstances[semanticKey] = [];
    taskInstances[semanticKey].push(binding);
  }
  for (const instances of Object.values(taskInstances)) instances.sort(compareTaskInstances);

  const relationships = new Map();
  for (const observation of observations) {
    const { binding, semanticKey } = observation;
    const parentKey = semanticByIdentity.get(binding.parentTaskId);
    if (parentKey) addRelationship(relationships, 'contains', parentKey, semanticKey);
    for (const child of binding.childTaskReferences || []) {
      const childKey =
        semanticByIdentity.get(child.taskId) ||
        semanticByIdentity.get(child.podName) ||
        taskSemanticKey(child);
      if (childKey) addRelationship(relationships, 'depends-on', semanticKey, childKey);
    }
  }

  const compatibilityTasks = {};
  for (const [semanticKey, instances] of Object.entries(taskInstances)) {
    compatibilityTasks[semanticKey] = instances[0];
  }
  return {
    relationships,
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
  const taskBindings = buildTaskBindings(rawTasks, flavor);
  const artifacts = {};
  for (const [artifactKey, definition] of Object.entries(ARTIFACT_FIXTURES)) {
    artifacts[artifactKey] = buildArtifactBinding(rawTasks, flavor, definition, hydratedArtifacts);
    for (const [consumerTask, consumer] of Object.entries(artifacts[artifactKey].consumers || {})) {
      if (consumer.artifactIds.length > 0) {
        addRelationship(taskBindings.relationships, 'artifact-consumer', artifactKey, consumerTask);
      }
    }
  }

  return {
    artifacts: sortObject(artifacts),
    displayName: stringValue(field(run, 'display_name', 'displayName', 'name')),
    fixtureProfile: options.fixtureProfile || DEFAULT_RUN_PROFILE,
    relationships: [...taskBindings.relationships.values()].sort((left, right) =>
      `${left.kind}|${left.source}|${left.target}`.localeCompare(
        `${right.kind}|${right.source}|${right.target}`,
      ),
    ),
    revisionFlavor: flavor,
    runId: stringValue(field(run, 'run_id', 'runId', 'id')),
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
  return binding.relationships.some(
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

function validateRunBinding(runKey, binding, profile, errors) {
  if (!profile) {
    errors.push(`${runKey}: unknown fixture profile ${binding.fixtureProfile}`);
    return;
  }

  for (const [taskKey, expectedCount] of Object.entries(profile.tasks || {})) {
    const instances = binding.taskInstances[taskKey] || [];
    if (instances.length !== expectedCount) {
      errors.push(
        `${runKey}: expected ${expectedCount} ${taskKey} instance(s), found ${instances.length}`,
      );
      continue;
    }
    if (binding.revisionFlavor === REVISION_FLAVORS.NATIVE) {
      const expectedKind = TASK_FIXTURES[taskKey]?.kind;
      for (const instance of instances) {
        if (!nativeTaskTypeMatches(instance.type, expectedKind)) {
          errors.push(`${runKey}: ${taskKey} has native type ${instance.type || 'unknown'}`);
        }
      }
    }
  }

  for (const artifactKey of profile.artifacts || []) {
    if (!binding.artifacts[artifactKey]?.artifactIds?.length) {
      errors.push(`${runKey}: missing ${artifactKey}`);
    }
  }

  const metricMembers = binding.artifacts['artifact.scalar-metrics']?.members || {};
  for (const [metricKey, definition] of Object.entries(
    ARTIFACT_FIXTURES['artifact.scalar-metrics'].members,
  )) {
    const member = metricMembers[metricKey];
    if (!member?.artifactIds?.length) {
      errors.push(`${runKey}: missing ${metricKey}`);
    } else if (member.numberValue !== definition.value) {
      errors.push(
        `${runKey}: ${metricKey} value ${String(member.numberValue)} did not match ${definition.value}`,
      );
    }
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
    const attempts =
      binding.revisionFlavor === REVISION_FLAVORS.LEGACY
        ? (retryTask?.failedMainJobs?.length || 0) + 1
        : retryTask?.executorPods?.length || 0;
    if (attempts < profile.retry.attempts) {
      errors.push(
        `${runKey}: ${profile.retry.task} recorded ${attempts} attempt(s), expected at least ${profile.retry.attempts}`,
      );
    }
  }

  if (profile.loop && binding.revisionFlavor === REVISION_FLAVORS.NATIVE) {
    const indexes = (binding.taskInstances[profile.loop.worker] || [])
      .map((instance) => instance.iterationIndex)
      .filter((value) => value !== null && value !== undefined)
      .sort((left, right) => left - right);
    if (!sameValues(indexes, profile.loop.iterationIndexes)) {
      errors.push(
        `${runKey}: ${profile.loop.worker} iteration indexes ${JSON.stringify(indexes)} did not match ${JSON.stringify(profile.loop.iterationIndexes)}`,
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

  const scalarBinding = binding.artifacts['artifact.scalar-metrics'];
  const scalarConsumer = scalarBinding?.consumers?.['task.consume-metrics'];
  if (
    profile.relationships?.some(
      (relationship) =>
        relationship.kind === 'artifact-consumer' &&
        relationship.source === 'artifact.scalar-metrics',
    ) &&
    !sameValues(scalarBinding?.artifactIds, scalarConsumer?.artifactIds)
  ) {
    errors.push(`${runKey}: scalar metric producer and consumer artifact bindings differ`);
  }
}

function buildSemanticDeployment({ logical, resourceBindings = {}, runResponses = [] }) {
  const runs = {};
  const flavors = new Set();
  for (const observation of runResponses) {
    const { profileKey } = expectedRunProfile(logical, observation.semanticKey);
    const binding = extractRunBinding(observation.response, observation.semanticKey, {
      fixtureProfile: profileKey,
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
  DEFAULT_RUN_PROFILE,
  REVISION_FLAVORS,
  RUN_PROFILES,
  SEMANTIC_FIXTURE_SET,
  SEMANTIC_SCHEMA_VERSION,
  TASK_FIXTURES,
  buildLogicalFixtures,
  buildSemanticDeployment,
  cloneSorted,
  combineSemanticManifests,
  detectRevisionFlavor,
  extractRunBinding,
  field,
};
