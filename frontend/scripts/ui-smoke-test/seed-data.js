#!/usr/bin/env node
/**
 * Creates deterministic API resources used by seeded UI screenshot routes.
 */

const fs = require('fs');
const http = require('http');
const https = require('https');
const path = require('path');

const API_BASE = process.env.API_BASE || 'http://localhost:3001';
const REPO_ROOT = path.resolve(__dirname, '../../..');
const DEFAULT_MANIFEST_PATH = path.join(REPO_ROOT, '.ui-smoke-test', 'seed-manifest.json');
const SEED_MANIFEST_PATH = process.env.UI_SMOKE_SEED_MANIFEST || DEFAULT_MANIFEST_PATH;
const MULTIPART_BOUNDARY = '----kfp-ui-smoke-pipeline-boundary';
const SEED_IMAGE =
  'docker.io/library/busybox@sha256:73aaf090f3d85aa34ee199857f03fa3a95c8ede2ffd4cc2cdb5b94e566b11662';
const FAILED_RUN_STATES = new Set(['SKIPPED', 'FAILED', 'CANCELED', 'PAUSED']);
const METRICS_EXECUTOR_OUTPUT = {
  artifacts: {
    scalar_metrics: {
      artifacts: [{ metadata: { accuracy: 0.92, loss: 0.08 } }],
    },
    roc_curve: {
      artifacts: [
        {
          metadata: {
            confidenceMetrics: [
              { confidenceThreshold: 1, recall: 0, falsePositiveRate: 0 },
              { confidenceThreshold: 0.8, recall: 0.35, falsePositiveRate: 0.08 },
              { confidenceThreshold: 0.5, recall: 0.72, falsePositiveRate: 0.22 },
              { confidenceThreshold: 0.2, recall: 0.9, falsePositiveRate: 0.55 },
              { confidenceThreshold: 0, recall: 1, falsePositiveRate: 1 },
            ],
          },
        },
      ],
    },
  },
};
const METRICS_EXECUTOR_OUTPUT_JSON = JSON.stringify(METRICS_EXECUTOR_OUTPUT);

const MINIMAL_PIPELINE_YAML = `pipelineInfo:
  name: ui-smoke-pipeline
root:
  dag:
    outputs:
      artifacts:
        roc_curve:
          artifactSelectors:
            - outputArtifactKey: roc_curve
              producerSubtask: write-metrics
        scalar_metrics:
          artifactSelectors:
            - outputArtifactKey: scalar_metrics
              producerSubtask: write-metrics
    tasks:
      write-metrics:
        taskInfo:
          name: write-metrics
        cachingOptions:
          enableCache: false
        componentRef:
          name: comp-write-metrics
  outputDefinitions:
    artifacts:
      roc_curve:
        artifactType:
          schemaTitle: system.ClassificationMetrics
          schemaVersion: 0.0.1
      scalar_metrics:
        artifactType:
          schemaTitle: system.Metrics
          schemaVersion: 0.0.1
schemaVersion: 2.1.0
sdkVersion: kfp-2.14.6
components:
  comp-write-metrics:
    executorLabel: exec-write-metrics
    outputDefinitions:
      artifacts:
        roc_curve:
          artifactType:
            schemaTitle: system.ClassificationMetrics
            schemaVersion: 0.0.1
        scalar_metrics:
          artifactType:
            schemaTitle: system.Metrics
            schemaVersion: 0.0.1
deploymentSpec:
  executors:
    exec-write-metrics:
      container:
        image: ${SEED_IMAGE}
        command:
          - /bin/sh
          - -ec
        args:
          - |
            metadata_path="$(dirname "$1")/output_metadata.json"
            mkdir -p "$(dirname "$1")" "$(dirname "$2")"
            : > "$1"
            : > "$2"
            printf '%s' '${METRICS_EXECUTOR_OUTPUT_JSON}' > "$metadata_path"
          - ui-smoke-metrics
          - "{{$.outputs.artifacts['scalar_metrics'].path}}"
          - "{{$.outputs.artifacts['roc_curve'].path}}"
`;

const RESOURCE_DEFINITIONS = {
  experiments: [
    {
      displayName: 'UI Smoke - Image Classification',
      description: 'Deterministic UI smoke-test experiment',
    },
    {
      displayName: 'UI Smoke - Natural Language Processing',
      description: 'Second deterministic UI smoke-test experiment',
    },
  ],
  pipelines: [
    {
      name: 'ui-smoke-training-pipeline',
      displayName: 'UI Smoke Training Pipeline',
      description: 'Deterministic training pipeline for UI screenshots',
    },
    {
      name: 'ui-smoke-data-ingestion',
      displayName: 'UI Smoke Data Ingestion',
      description: 'Deterministic data-ingestion pipeline for UI screenshots',
    },
    {
      name: 'ui-smoke-model-evaluation',
      displayName: 'UI Smoke Model Evaluation',
      description: 'Deterministic evaluation pipeline for UI screenshots',
    },
  ],
  runs: [
    'UI Smoke Training Run 1',
    'UI Smoke Training Run 2',
    'UI Smoke Evaluation Run',
    'UI Smoke Inference Run',
    'UI Smoke Data Processing Run',
  ],
  recurringRuns: ['UI Smoke Daily Training', 'UI Smoke Hourly Data Sync'],
};

function log(message, type = 'info') {
  const colors = {
    info: '\x1b[32m',
    warn: '\x1b[33m',
    error: '\x1b[31m',
    debug: '\x1b[36m',
  };
  console.log(`${colors[type] || ''}[SEED]\x1b[0m ${message}`);
}

function unique(values) {
  return [...new Set(values.filter(Boolean).map((value) => String(value)))];
}

function pickList(response, listKeys) {
  for (const key of listKeys) {
    if (Array.isArray(response?.[key])) return response[key];
  }
  return [];
}

function resourceId(resource, candidateKeys) {
  for (const key of candidateKeys) {
    const value = resource?.[key];
    if (value !== undefined && value !== null && String(value).length > 0) return String(value);
  }
  return null;
}

function requireResourceId(resource, candidateKeys, description) {
  const id = resourceId(resource, candidateKeys);
  if (!id) throw new Error(`${description} response did not contain a resource ID.`);
  return id;
}

function resolveApiUrl(apiBase, endpoint) {
  const base = new URL(apiBase);
  base.search = '';
  base.hash = '';
  if (!base.pathname.endsWith('/')) base.pathname += '/';
  return new URL(String(endpoint).replace(/^\/+/, ''), base);
}

function apiRequest(method, endpoint, body = null, options = {}) {
  const { apiBase = API_BASE, headers = {}, rawBody = null, timeout = 10000 } = options;
  return new Promise((resolve, reject) => {
    const url = resolveApiUrl(apiBase, endpoint);
    const payload =
      rawBody === null ? (body === null ? null : Buffer.from(JSON.stringify(body))) : rawBody;
    const requestHeaders = { ...headers };
    if (rawBody === null && payload !== null && !requestHeaders['Content-Type']) {
      requestHeaders['Content-Type'] = 'application/json';
    }
    if (payload !== null) requestHeaders['Content-Length'] = String(payload.length);

    const protocol = url.protocol === 'https:' ? https : http;
    const request = protocol.request(
      {
        method,
        hostname: url.hostname,
        port: url.port,
        path: `${url.pathname}${url.search}`,
        headers: requestHeaders,
      },
      (response) => {
        let data = '';
        response.setEncoding('utf8');
        response.on('data', (chunk) => {
          data += chunk;
        });
        response.on('end', () => {
          let parsed = data;
          if (data) {
            try {
              parsed = JSON.parse(data);
            } catch (error) {
              // Preserve non-JSON bodies for actionable error messages.
            }
          } else {
            parsed = {};
          }
          if (response.statusCode < 200 || response.statusCode >= 300) {
            reject(new Error(`API error ${response.statusCode}: ${JSON.stringify(parsed)}`));
            return;
          }
          resolve(parsed);
        });
      },
    );
    request.on('error', reject);
    request.setTimeout(timeout, () => {
      request.destroy(new Error(`Request timeout: ${method} ${endpoint}`));
    });
    if (payload !== null) request.write(payload);
    request.end();
  });
}

function createMultipartUpload(contents, filename = 'ui-smoke-pipeline.yaml') {
  const body = Buffer.from(
    `--${MULTIPART_BOUNDARY}\r\n` +
      `Content-Disposition: form-data; name="uploadfile"; filename="${filename}"\r\n` +
      'Content-Type: application/yaml\r\n\r\n' +
      contents +
      `\r\n--${MULTIPART_BOUNDARY}--\r\n`,
  );
  return {
    body,
    headers: { 'Content-Type': `multipart/form-data; boundary=${MULTIPART_BOUNDARY}` },
  };
}

async function checkHealth(request = apiRequest) {
  try {
    await request('GET', '/apis/v2beta1/healthz');
    return true;
  } catch (error) {
    return false;
  }
}

async function listAll(endpoint, listKeys, request = apiRequest) {
  const items = [];
  let pageToken = '';
  for (let page = 0; page < 50; page++) {
    const separator = endpoint.includes('?') ? '&' : '?';
    const query = new URLSearchParams({ page_size: '100' });
    if (pageToken) query.set('page_token', pageToken);
    const response = await request('GET', `${endpoint}${separator}${query}`);
    items.push(...pickList(response, listKeys));
    pageToken = response?.next_page_token || response?.nextPageToken || '';
    if (!pageToken) return items;
  }
  throw new Error(`Pagination did not terminate for ${endpoint}.`);
}

async function fetchInventory(request = apiRequest) {
  const [pipelines, experiments, runs, recurringRuns] = await Promise.all([
    listAll('/apis/v2beta1/pipelines', ['pipelines'], request),
    listAll('/apis/v2beta1/experiments', ['experiments'], request),
    listAll('/apis/v2beta1/runs', ['runs'], request),
    listAll('/apis/v2beta1/recurringruns', ['recurring_runs', 'recurringRuns', 'jobs'], request),
  ]);
  return { pipelines, experiments, runs, recurringRuns };
}

async function fetchResourceIds(request = apiRequest) {
  const inventory = await fetchInventory(request);
  return {
    experimentIds: unique(
      inventory.experiments.map((resource) =>
        resourceId(resource, ['experiment_id', 'experimentId', 'id']),
      ),
    ),
    pipelineIds: unique(
      inventory.pipelines.map((resource) =>
        resourceId(resource, ['pipeline_id', 'pipelineId', 'id']),
      ),
    ),
    recurringRunIds: unique(
      inventory.recurringRuns.map((resource) =>
        resourceId(resource, ['recurring_run_id', 'recurringRunId', 'job_id', 'id']),
      ),
    ),
    runIds: unique(
      inventory.runs.map((resource) => resourceId(resource, ['run_id', 'runId', 'id'])),
    ),
  };
}

function buildSeedManifest(resourceIds, options = {}) {
  const { apiBase = API_BASE } = options;
  return {
    apiBase,
    defaults: {
      compareRunlist: resourceIds.runIds.slice(0, 3).join(','),
      experimentId: resourceIds.experimentIds[0] || null,
      pipelineId: resourceIds.pipelineIds[0] || null,
      recurringRunId: resourceIds.recurringRunIds[0] || null,
      runId: resourceIds.runIds[0] || null,
    },
    generatedAt: new Date().toISOString(),
    resources: resourceIds,
  };
}

function writeSeedManifest(manifest, manifestPath = SEED_MANIFEST_PATH) {
  fs.mkdirSync(path.dirname(manifestPath), { recursive: true });
  fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));
  log(`Wrote seed manifest: ${manifestPath}`);
}

async function createExperiment(name, description, request = apiRequest) {
  const result = await request('POST', '/apis/v2beta1/experiments', {
    display_name: name,
    description,
  });
  requireResourceId(result, ['experiment_id', 'experimentId', 'id'], `Experiment ${name}`);
  return result;
}

async function uploadPipeline(name, description, request = apiRequest, options = {}) {
  const { displayName = name } = options;
  const query = new URLSearchParams({
    name,
    display_name: displayName,
    description,
  });
  const multipart = createMultipartUpload(MINIMAL_PIPELINE_YAML);
  const result = await request('POST', `/apis/v2beta1/pipelines/upload?${query}`, null, {
    headers: multipart.headers,
    rawBody: multipart.body,
  });
  requireResourceId(result, ['pipeline_id', 'pipelineId', 'id'], `Pipeline ${name}`);
  return result;
}

async function uploadPipelineVersion(pipelineId, request = apiRequest) {
  if (!pipelineId) throw new Error('A pipeline ID is required to upload a pipeline version.');
  const query = new URLSearchParams({
    name: 'ui-smoke-version',
    display_name: 'UI Smoke Version',
    pipelineid: pipelineId,
    description: 'Deterministic pipeline version for UI screenshots',
  });
  const multipart = createMultipartUpload(MINIMAL_PIPELINE_YAML);
  const result = await request('POST', `/apis/v2beta1/pipelines/upload_version?${query}`, null, {
    headers: multipart.headers,
    rawBody: multipart.body,
  });
  requireResourceId(
    result,
    ['pipeline_version_id', 'pipelineVersionId', 'id'],
    `Pipeline version for ${pipelineId}`,
  );
  return result;
}

async function createRun(
  name,
  pipelineId,
  experimentId,
  request = apiRequest,
  pipelineVersionId = null,
) {
  if (!pipelineId || !experimentId) {
    throw new Error(`Run ${name} requires both a pipeline ID and an experiment ID.`);
  }
  const reference = { pipeline_id: pipelineId };
  if (pipelineVersionId) reference.pipeline_version_id = pipelineVersionId;
  const result = await request('POST', '/apis/v2beta1/runs', {
    display_name: name,
    description: `Deterministic UI smoke-test run: ${name}`,
    experiment_id: experimentId,
    pipeline_version_reference: reference,
    runtime_config: { parameters: {} },
  });
  requireResourceId(result, ['run_id', 'runId', 'id'], `Run ${name}`);
  return result;
}

async function createRecurringRun(
  name,
  pipelineId,
  experimentId,
  request = apiRequest,
  pipelineVersionId = null,
) {
  if (!pipelineId || !experimentId) {
    throw new Error(`Recurring run ${name} requires both a pipeline ID and an experiment ID.`);
  }
  const reference = { pipeline_id: pipelineId };
  if (pipelineVersionId) reference.pipeline_version_id = pipelineVersionId;
  const result = await request('POST', '/apis/v2beta1/recurringruns', {
    display_name: name,
    description: `Deterministic UI smoke-test schedule: ${name}`,
    experiment_id: experimentId,
    max_concurrency: '1',
    mode: 'DISABLE',
    no_catchup: true,
    pipeline_version_reference: reference,
    runtime_config: { parameters: {} },
    trigger: { periodic_schedule: { interval_second: '3600' } },
  });
  requireResourceId(
    result,
    ['recurring_run_id', 'recurringRunId', 'job_id', 'id'],
    `Recurring run ${name}`,
  );
  return result;
}

async function getExistingCounts(request = apiRequest) {
  const inventory = await fetchInventory(request);
  return {
    experiments: inventory.experiments.length,
    pipelines: inventory.pipelines.length,
    recurringRuns: inventory.recurringRuns.length,
    runs: inventory.runs.length,
  };
}

function targetCount(value, definitions) {
  const number = Number.isFinite(Number(value)) ? Math.floor(Number(value)) : definitions.length;
  return Math.min(definitions.length, Math.max(1, number));
}

function byDisplayName(resources, displayName) {
  return resources.find(
    (resource) => (resource.display_name || resource.displayName || resource.name) === displayName,
  );
}

function byPipelineDefinition(resources, definition) {
  return resources.find((resource) => {
    const name = resource.name;
    const displayName = resource.display_name || resource.displayName;
    return name === definition.name || displayName === definition.displayName;
  });
}

async function validateDetailRoutes(resources, request = apiRequest) {
  const checks = [
    request('GET', `/apis/v2beta1/pipelines/${encodeURIComponent(resources.pipelineIds[0])}`),
    request('GET', `/apis/v2beta1/experiments/${encodeURIComponent(resources.experimentIds[0])}`),
    request(
      'GET',
      `/apis/v2beta1/recurringruns/${encodeURIComponent(resources.recurringRunIds[0])}`,
    ),
    request(
      'GET',
      `/apis/v2beta1/pipelines/${encodeURIComponent(resources.pipelineIds[0])}/versions/${encodeURIComponent(resources.pipelineVersionIds[0])}`,
    ),
    ...resources.runIds.map((runId) =>
      request('GET', `/apis/v2beta1/runs/${encodeURIComponent(runId)}`),
    ),
  ];
  await Promise.all(checks);
}

function runState(run) {
  return String(run?.state || run?.run?.state || '').toUpperCase();
}

function runFailureDetail(run) {
  const detail = run?.error || run?.message || run?.status?.message || run?.run?.error;
  return detail ? `: ${String(detail).slice(0, 500)}` : '';
}

async function waitForRunsStable(runIds, request = apiRequest, options = {}) {
  const {
    interval = 1000,
    now = Date.now,
    sleep = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds)),
    timeout = 180000,
  } = options;
  const pending = new Map(runIds.map((runId) => [String(runId), 'UNKNOWN']));
  const deadline = now() + timeout;

  while (pending.size > 0 && now() < deadline) {
    const snapshots = await Promise.all(
      [...pending.keys()].map(async (runId) => {
        const run = await request('GET', `/apis/v2beta1/runs/${encodeURIComponent(runId)}`);
        return { detail: runFailureDetail(run), runId, state: runState(run) };
      }),
    );
    for (const snapshot of snapshots) {
      if (snapshot.state === 'SUCCEEDED') {
        pending.delete(snapshot.runId);
      } else if (FAILED_RUN_STATES.has(snapshot.state)) {
        throw new Error(`Seeded run ${snapshot.runId} reached ${snapshot.state}${snapshot.detail}`);
      } else {
        pending.set(snapshot.runId, snapshot.state || 'UNKNOWN');
      }
    }
    if (pending.size > 0) await sleep(interval);
  }

  if (pending.size > 0) {
    const states = [...pending.entries()].map(([runId, state]) => `${runId}=${state}`).join(', ');
    throw new Error(`Timed out waiting for terminal run state: ${states}`);
  }
  return true;
}

function failureRecord(type, name, error) {
  return { type, name, error: error.message };
}

async function seedData(options = {}) {
  const {
    pipelines = 3,
    experiments = 2,
    runs = 5,
    recurringRuns = 2,
    manifestPath = SEED_MANIFEST_PATH,
    apiBase = API_BASE,
  } = options;
  const waitForRunsFn = options.waitForRunsFn || waitForRunsStable;
  const request =
    options.request ||
    ((method, endpoint, body = null, requestOptions = {}) =>
      apiRequest(method, endpoint, body, { apiBase, ...requestOptions }));
  const targets = {
    pipelines: targetCount(pipelines, RESOURCE_DEFINITIONS.pipelines),
    experiments: targetCount(experiments, RESOURCE_DEFINITIONS.experiments),
    runs: targetCount(runs, RESOURCE_DEFINITIONS.runs),
    recurringRuns: targetCount(recurringRuns, RESOURCE_DEFINITIONS.recurringRuns),
  };
  const created = {
    experiments: [],
    pipelines: [],
    pipelineVersions: [],
    runs: [],
    recurringRuns: [],
  };
  const selected = {
    experiments: [],
    pipelines: [],
    pipelineVersions: [],
    runs: [],
    recurringRuns: [],
  };
  const failures = [];

  if (!(await checkHealth(request))) {
    return { success: false, error: 'API not healthy', failures, created };
  }

  let inventory;
  try {
    inventory = await fetchInventory(request);
  } catch (error) {
    return {
      success: false,
      error: `Failed to inventory API resources: ${error.message}`,
      failures,
      created,
    };
  }

  for (const definition of RESOURCE_DEFINITIONS.experiments.slice(0, targets.experiments)) {
    let resource = byDisplayName(inventory.experiments, definition.displayName);
    if (!resource) {
      try {
        resource = await createExperiment(definition.displayName, definition.description, request);
        created.experiments.push(resource);
      } catch (error) {
        failures.push(failureRecord('experiment', definition.displayName, error));
      }
    }
    if (resource) selected.experiments.push(resource);
  }

  for (const definition of RESOURCE_DEFINITIONS.pipelines.slice(0, targets.pipelines)) {
    let resource = byPipelineDefinition(inventory.pipelines, definition);
    if (!resource) {
      try {
        resource = await uploadPipeline(definition.name, definition.description, request, {
          displayName: definition.displayName,
        });
        created.pipelines.push(resource);
      } catch (error) {
        failures.push(failureRecord('pipeline', definition.name, error));
      }
    }
    if (resource) selected.pipelines.push(resource);
  }

  for (const pipeline of selected.pipelines) {
    const pipelineId = resourceId(pipeline, ['pipeline_id', 'pipelineId', 'id']);
    if (!pipelineId) {
      failures.push(
        failureRecord('pipeline-version', 'unknown pipeline', new Error('Missing pipeline ID')),
      );
      continue;
    }
    try {
      let versions = await listAll(
        `/apis/v2beta1/pipelines/${encodeURIComponent(pipelineId)}/versions`,
        ['pipeline_versions', 'pipelineVersions'],
        request,
      );
      if (versions.length === 0) {
        const version = await uploadPipelineVersion(pipelineId, request);
        created.pipelineVersions.push(version);
        versions = [version];
      }
      requireResourceId(
        versions[0],
        ['pipeline_version_id', 'pipelineVersionId', 'id'],
        `Pipeline version for ${pipelineId}`,
      );
      selected.pipelineVersions.push(versions[0]);
    } catch (error) {
      failures.push(failureRecord('pipeline-version', pipelineId, error));
    }
  }

  const primaryPipelineId = resourceId(selected.pipelines[0], ['pipeline_id', 'pipelineId', 'id']);
  const primaryPipelineVersionId = resourceId(selected.pipelineVersions[0], [
    'pipeline_version_id',
    'pipelineVersionId',
    'id',
  ]);
  const primaryExperimentId = resourceId(selected.experiments[0], [
    'experiment_id',
    'experimentId',
    'id',
  ]);

  for (const name of RESOURCE_DEFINITIONS.runs.slice(0, targets.runs)) {
    let resource = byDisplayName(inventory.runs, name);
    if (!resource) {
      try {
        resource = await createRun(
          name,
          primaryPipelineId,
          primaryExperimentId,
          request,
          primaryPipelineVersionId,
        );
        created.runs.push(resource);
      } catch (error) {
        failures.push(failureRecord('run', name, error));
      }
    }
    if (resource) selected.runs.push(resource);
  }

  for (const name of RESOURCE_DEFINITIONS.recurringRuns.slice(0, targets.recurringRuns)) {
    let resource = byDisplayName(inventory.recurringRuns, name);
    if (!resource) {
      try {
        resource = await createRecurringRun(
          name,
          primaryPipelineId,
          primaryExperimentId,
          request,
          primaryPipelineVersionId,
        );
        created.recurringRuns.push(resource);
      } catch (error) {
        failures.push(failureRecord('recurring-run', name, error));
      }
    }
    if (resource) selected.recurringRuns.push(resource);
  }

  const resources = {
    experimentIds: unique(
      selected.experiments.map((resource) =>
        resourceId(resource, ['experiment_id', 'experimentId', 'id']),
      ),
    ),
    pipelineIds: unique(
      selected.pipelines.map((resource) =>
        resourceId(resource, ['pipeline_id', 'pipelineId', 'id']),
      ),
    ),
    pipelineVersionIds: unique(
      selected.pipelineVersions.map((resource) =>
        resourceId(resource, ['pipeline_version_id', 'pipelineVersionId', 'id']),
      ),
    ),
    recurringRunIds: unique(
      selected.recurringRuns.map((resource) =>
        resourceId(resource, ['recurring_run_id', 'recurringRunId', 'job_id', 'id']),
      ),
    ),
    runIds: unique(
      selected.runs.map((resource) => resourceId(resource, ['run_id', 'runId', 'id'])),
    ),
  };

  const missing = [];
  if (resources.experimentIds.length < targets.experiments) missing.push('experiments');
  if (resources.pipelineIds.length < targets.pipelines) missing.push('pipelines');
  if (resources.pipelineVersionIds.length < targets.pipelines) missing.push('pipeline versions');
  if (resources.runIds.length < targets.runs) missing.push('runs');
  if (resources.recurringRunIds.length < targets.recurringRuns) missing.push('recurring runs');
  if (failures.length > 0 || missing.length > 0) {
    const error = [
      failures.length > 0 ? `${failures.length} resource operation(s) failed` : '',
      missing.length > 0 ? `missing required ${missing.join(', ')}` : '',
    ]
      .filter(Boolean)
      .join('; ');
    return { success: false, error, failures, created, resources };
  }

  try {
    await waitForRunsFn(resources.runIds, request, {
      interval: options.runPollInterval,
      timeout: options.runTimeout,
    });
  } catch (error) {
    return {
      success: false,
      error: `Seeded runs did not reach a stable state: ${error.message}`,
      failures,
      created,
      resources,
    };
  }

  try {
    await validateDetailRoutes(resources, request);
  } catch (error) {
    return {
      success: false,
      error: `Required detail route validation failed: ${error.message}`,
      failures,
      created,
      resources,
    };
  }

  const manifest = buildSeedManifest(resources, { apiBase });
  writeSeedManifest(manifest, manifestPath);
  return {
    success: true,
    skipped: Object.values(created).every((items) => items.length === 0),
    created,
    resources,
    seedManifestPath: manifestPath,
  };
}

async function clearData() {
  log('Data clearing is intentionally disabled. Delete the test cluster to reset it.', 'warn');
  return { success: false, error: 'Not implemented' };
}

if (require.main === module) {
  const clear = process.argv.slice(2).includes('--clear');
  const operation = clear ? clearData() : seedData();
  operation
    .then((result) => {
      if (!result.success) console.error(result.error);
      process.exitCode = result.success ? 0 : 1;
    })
    .catch((error) => {
      console.error(error.message);
      process.exitCode = 1;
    });
}

module.exports = {
  API_BASE,
  METRICS_EXECUTOR_OUTPUT,
  MINIMAL_PIPELINE_YAML,
  RESOURCE_DEFINITIONS,
  SEED_IMAGE,
  SEED_MANIFEST_PATH,
  apiRequest,
  buildSeedManifest,
  checkHealth,
  clearData,
  createExperiment,
  createMultipartUpload,
  createRecurringRun,
  createRun,
  fetchInventory,
  fetchResourceIds,
  getExistingCounts,
  listAll,
  resourceId,
  resolveApiUrl,
  seedData,
  uploadPipeline,
  uploadPipelineVersion,
  validateDetailRoutes,
  waitForRunsStable,
  writeSeedManifest,
};
